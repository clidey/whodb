package whodb

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

const defaultPageSize = 100

// OntologyHandle is the client.Ontology("User") facade: reads and record
// writes for one ontology entity, addressed by apiName. Entity metadata is
// fetched once per handle and reused for pk lookups and row hydration.
type OntologyHandle struct {
	client  *Client
	apiName string

	mu     sync.Mutex
	entity map[string]any
}

// ListOptions filter and page list-shaped reads. Where is a JSON filter
// object (property → {"eq": ...} etc.) serialized into OntologyQuery.
type ListOptions struct {
	Where    map[string]any
	Sort     []map[string]any
	PageSize int
	Offset   int
}

// EntityMeta resolves and caches the entity metadata backing this handle.
func (h *OntologyHandle) EntityMeta(ctx context.Context) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.entity != nil {
		return h.entity, nil
	}
	warnIfFlagged("OntologyEntities")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.client.execute(ctx, gen.NewOntologyEntitiesRequest(map[string]any{"projectId": projectID}))
	if err != nil {
		return nil, err
	}
	entities, _ := result.([]any)
	for _, entry := range entities {
		entity, _ := entry.(map[string]any)
		if entity["apiName"] == h.apiName {
			h.entity = entity
			return entity, nil
		}
	}
	return nil, fmt.Errorf("%w: ontology entity %q not found in this project", ErrNotFound, h.apiName)
}

func (h *OntologyHandle) entityID(ctx context.Context) (string, error) {
	entity, err := h.EntityMeta(ctx)
	if err != nil {
		return "", err
	}
	id, _ := entity["id"].(string)
	return id, nil
}

func (h *OntologyHandle) primaryKey(ctx context.Context) (string, error) {
	entity, err := h.EntityMeta(ctx)
	if err != nil {
		return "", err
	}
	pk, _ := entity["primaryKey"].(string)
	return pk, nil
}

// Describe describes the entity: schema, properties, links.
func (h *OntologyHandle) Describe(ctx context.Context) (map[string]any, error) {
	if _, err := h.EntityMeta(ctx); err != nil { // ErrNotFound for unknown entities
		return nil, err
	}
	warnIfFlagged("OntologyDescribe")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.client.execute(ctx, gen.NewOntologyDescribeRequest(map[string]any{
		"projectId": projectID,
		"input":     map[string]any{"entities": []string{h.apiName}, "includeInferred": true},
	}))
	if err != nil {
		return nil, err
	}
	description, _ := result.(map[string]any)
	return description, nil
}

func (h *OntologyHandle) query(ctx context.Context, input map[string]any) (map[string]any, error) {
	warnIfFlagged("OntologyQuery")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	input["entity"] = h.apiName
	result, err := h.client.execute(ctx, gen.NewOntologyQueryRequest(map[string]any{
		"projectId": projectID,
		"input":     input,
	}))
	if err != nil {
		return nil, err
	}
	wire, _ := result.(map[string]any)
	return wire, nil
}

// Get fetches a single record by primary key; nil when absent.
func (h *OntologyHandle) Get(ctx context.Context, pk any) (Row, error) {
	primaryKey, err := h.primaryKey(ctx)
	if err != nil {
		return nil, err
	}
	if primaryKey == "" {
		return nil, fmt.Errorf("%w: entity %q has no primary key — use List with a where filter", ErrValidation, h.apiName)
	}
	whereJSON, err := json.Marshal(map[string]any{primaryKey: map[string]any{"eq": fmt.Sprint(pk)}})
	if err != nil {
		return nil, err
	}
	result, err := h.query(ctx, map[string]any{"whereJson": string(whereJSON), "pageSize": 1, "offset": 0})
	if err != nil {
		return nil, err
	}
	entity, _ := h.EntityMeta(ctx)
	rows, _ := hydrateRows(result, propertyTypesOf(entity))
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// List returns one page of records with optional filter/sort.
func (h *OntologyHandle) List(ctx context.Context, options ListOptions) ([]Row, error) {
	entity, err := h.EntityMeta(ctx)
	if err != nil {
		return nil, err
	}
	pageSize := options.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	input := map[string]any{"pageSize": pageSize, "offset": options.Offset}
	if options.Where != nil {
		whereJSON, err := json.Marshal(options.Where)
		if err != nil {
			return nil, err
		}
		input["whereJson"] = string(whereJSON)
	}
	if options.Sort != nil {
		input["sort"] = options.Sort
	}
	result, err := h.query(ctx, input)
	if err != nil {
		return nil, err
	}
	rows, _ := hydrateRows(result, propertyTypesOf(entity))
	return rows, nil
}

// Pages walks every page, invoking fn per page until a short page ends the
// result set or fn returns false.
func (h *OntologyHandle) Pages(ctx context.Context, options ListOptions, fn func(rows []Row) bool) error {
	pageSize := options.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
		options.PageSize = pageSize
	}
	for {
		rows, err := h.List(ctx, options)
		if err != nil {
			return err
		}
		if !fn(rows) || len(rows) < pageSize {
			return nil
		}
		options.Offset += pageSize
	}
}

// toRecordInputs converts a data map to GraphQL RecordInput pairs, JSON-
// encoding objects and lowercasing booleans (cross-language behavior).
func toRecordInputs(values map[string]any) []map[string]string {
	records := make([]map[string]string, 0, len(values))
	for key, value := range values {
		var encoded string
		switch typed := value.(type) {
		case nil:
			encoded = ""
		case bool:
			if typed {
				encoded = "true"
			} else {
				encoded = "false"
			}
		case string:
			encoded = typed
		case map[string]any, []any:
			raw, _ := json.Marshal(typed)
			encoded = string(raw)
		default:
			encoded = fmt.Sprint(typed)
		}
		records = append(records, map[string]string{"Key": key, "Value": encoded})
	}
	return records
}

// Create inserts one record. Values are field name/value pairs.
func (h *OntologyHandle) Create(ctx context.Context, values map[string]any) error {
	entityID, err := h.entityID(ctx)
	if err != nil {
		return err
	}
	warnIfFlagged("OntologyAddRow")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return err
	}
	_, err = h.client.execute(ctx, gen.NewOntologyAddRowRequest(map[string]any{
		"projectId": projectID,
		"entityId":  entityID,
		"values":    toRecordInputs(values),
	}))
	return err
}

// CreateMany inserts many records; idempotencyKey makes retries safe.
func (h *OntologyHandle) CreateMany(ctx context.Context, rows []map[string]any, idempotencyKey string) (map[string]any, error) {
	entityID, err := h.entityID(ctx)
	if err != nil {
		return nil, err
	}
	warnIfFlagged("OntologyAddRows")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	wireRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		wireRows = append(wireRows, map[string]any{"values": toRecordInputs(row)})
	}
	variables := map[string]any{"projectId": projectID, "entityId": entityID, "rows": wireRows}
	if idempotencyKey != "" {
		variables["idempotencyKey"] = idempotencyKey
	}
	result, err := h.client.execute(ctx, gen.NewOntologyAddRowsRequest(variables))
	if err != nil {
		return nil, err
	}
	outcome, _ := result.(map[string]any)
	return outcome, nil
}

// Update updates one record identified by primary key.
func (h *OntologyHandle) Update(ctx context.Context, pk any, values map[string]any) error {
	primaryKey, err := h.primaryKey(ctx)
	if err != nil {
		return err
	}
	if primaryKey == "" {
		return fmt.Errorf("%w: entity %q has no primary key — updates are not supported", ErrValidation, h.apiName)
	}
	entityID, err := h.entityID(ctx)
	if err != nil {
		return err
	}
	warnIfFlagged("OntologyUpdateRow")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return err
	}
	merged := map[string]any{}
	updatedColumns := make([]string, 0, len(values))
	for key, value := range values {
		merged[key] = value
		updatedColumns = append(updatedColumns, key)
	}
	merged[primaryKey] = fmt.Sprint(pk)
	_, err = h.client.execute(ctx, gen.NewOntologyUpdateRowRequest(map[string]any{
		"projectId":      projectID,
		"entityId":       entityID,
		"values":         toRecordInputs(merged),
		"updatedColumns": updatedColumns,
	}))
	return err
}

// Delete deletes one record identified by primary key.
func (h *OntologyHandle) Delete(ctx context.Context, pk any) error {
	primaryKey, err := h.primaryKey(ctx)
	if err != nil {
		return err
	}
	if primaryKey == "" {
		return fmt.Errorf("%w: entity %q has no primary key — deletes are not supported", ErrValidation, h.apiName)
	}
	entityID, err := h.entityID(ctx)
	if err != nil {
		return err
	}
	warnIfFlagged("OntologyDeleteRow")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return err
	}
	_, err = h.client.execute(ctx, gen.NewOntologyDeleteRowRequest(map[string]any{
		"projectId": projectID,
		"entityId":  entityID,
		"values":    toRecordInputs(map[string]any{primaryKey: fmt.Sprint(pk)}),
	}))
	return err
}

// FollowLink follows an outgoing link from one record to its related records.
func (h *OntologyHandle) FollowLink(ctx context.Context, pk any, linkAPIName string, pageSize, offset int) ([]Row, error) {
	entityID, err := h.entityID(ctx)
	if err != nil {
		return nil, err
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	warnIfFlagged("OntologyFollowLink")
	projectID, err := h.client.projectID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := h.client.execute(ctx, gen.NewOntologyFollowLinkRequest(map[string]any{
		"projectId":   projectID,
		"entityId":    entityID,
		"pk":          fmt.Sprint(pk),
		"linkApiName": linkAPIName,
		"pageSize":    pageSize,
		"pageOffset":  offset,
	}))
	if err != nil {
		return nil, err
	}
	wire, _ := result.(map[string]any)
	rows, _ := hydrateRows(wire, nil)
	return rows, nil
}
