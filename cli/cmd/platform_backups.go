/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/clidey/whodb/cli/internal/platform"
)

// backupSession is the signed-in hosted connection for one organization.
type backupSession struct {
	Host   string
	OrgID  string
	client *platform.Client
}

// Do sends an authenticated request to the host.
func (s *backupSession) Do(request *http.Request) (*http.Response, error) {
	return s.client.DoAuthenticated(request)
}

type backupTarget struct{ host, org string }

func (t *backupTarget) session(ctx context.Context) (*backupSession, error) {
	session, err := loadPlatformSession(ctx, t.host)
	if err != nil {
		return nil, err
	}
	if session.Automation {
		return nil, errors.New("organization backups require an owner or admin user session; service accounts are not supported")
	}
	org, err := resolveOrganization(ctx, session.Client, session.Host, t.org)
	if err != nil {
		return nil, err
	}
	session.Client.SetWorkspaceContext(org.ID, "")
	return &backupSession{Host: session.Host.URL, OrgID: org.ID, client: session.Client}, nil
}

func (t *backupTarget) flags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&t.host, "host", "", "hosted WhoDB URL")
	cmd.PersistentFlags().StringVar(&t.org, "org", "", "organization ID, slug, or name (defaults to selected organization)")
}

func backupQuery(ctx context.Context, s *backupSession, operation, document string, variables map[string]any) (json.RawMessage, error) {
	kind := "Query"
	if strings.HasPrefix(document, "mutation ") {
		kind = "Mutation"
	}
	// Hosts advertise what they support; an older host reports a missing operation here.
	if err := s.client.RequireOperation(kind, operation, "organization backups"); err != nil {
		return nil, err
	}
	var result map[string]json.RawMessage
	if err := s.client.ExecuteGraphQL(ctx, document, variables, &result); err != nil {
		return nil, err
	}
	value, ok := result[operation]
	if !ok {
		return nil, fmt.Errorf("server returned no %s result", operation)
	}
	return value, nil
}

func backupPrint(cmd *cobra.Command, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

func backupRun(t *backupTarget, operation, document string, variables func(*cobra.Command, *backupSession) (map[string]any, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		s, err := t.session(cmd.Context())
		if err != nil {
			return err
		}
		v, err := variables(cmd, s)
		if err != nil {
			return err
		}
		v["orgId"] = s.OrgID
		result, err := backupQuery(cmd.Context(), s, operation, document, v)
		if err != nil {
			return err
		}
		return backupPrint(cmd, result)
	}
}

func backupVars(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
	return map[string]any{}, nil
}

func backupSelection(path string) (map[string]any, error) {
	if path == "" {
		return map[string]any{"projects": []string{}, "resources": []any{}}, nil
	}
	data, err := os.ReadFile(path) // #nosec G304 -- the CLI user selected this input file.
	if err != nil {
		return nil, err
	}
	var selection struct {
		Projects  []string `json:"projects"`
		Resources []struct {
			ProjectID string `json:"projectId"`
			Type      string `json:"type"`
			ID        string `json:"id"`
		} `json:"resources"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&selection); err != nil {
		return nil, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, errors.New("selection file must contain one JSON object")
	}
	if selection.Projects == nil {
		selection.Projects = []string{}
	}
	if selection.Resources == nil {
		selection.Resources = []struct {
			ProjectID string `json:"projectId"`
			Type      string `json:"type"`
			ID        string `json:"id"`
		}{}
	}
	encoded, err := json.Marshal(selection)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	return value, json.Unmarshal(encoded, &value)
}

func backupID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func newBackupsCommand() *cobra.Command {
	t := &backupTarget{}
	root := &cobra.Command{Use: "backups", Short: "Manage encrypted organization backups"}
	t.flags(root)
	status := &cobra.Command{Use: "status", Short: "Show backup jobs, policy and recovery key status", RunE: backupRun(t, "OrganizationBackups", `query OrganizationBackups($orgId: ID!) { OrganizationBackups(orgId: $orgId) { canCreate isOwner storedBytes maxArchiveBytes emailConfigured key { id recipient confirmedAt } policy { id projectId enabled schedule timeZone retentionDays selection notificationUsers nextRunAt } projectPolicies { id projectId enabled schedule timeZone retentionDays selection notificationUsers nextRunAt } jobs { id keyId scheduleId projectId deletionRequestedAt deleteAfter kind state createdAt capturedAt scheduledFor completedAt expiresAt retentionDays sizeBytes sha256 errorCode } } }`, backupVars)}
	notifications := &cobra.Command{Use: "notifications", Short: "List backup failure notifications", RunE: backupRun(t, "OrganizationBackupNotifications", `query OrganizationBackupNotifications($orgId: ID!) { OrganizationBackupNotifications(orgId: $orgId) { id backupId errorCode createdAt readAt } }`, backupVars)}
	var selectionFile string
	scope := &cobra.Command{Use: "scope", Short: "Preview selected content and dependencies (owner only)", RunE: backupRun(t, "OrganizationBackupScope", `query OrganizationBackupScope($orgId: ID!, $selection: OrganizationBackupSelectionInput!) { OrganizationBackupScope(orgId: $orgId, selection: $selection) { full projects projectNames resources { projectId type id name automatic } requirements { kind projectId id version } } }`, func(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
		selection, err := backupSelection(selectionFile)
		return map[string]any{"selection": selection}, err
	})}
	scope.Flags().StringVar(&selectionFile, "selection-file", "", "JSON with projects and resources; omitted selects all")
	var createSelection string
	create := &cobra.Command{Use: "create", Short: "Queue a manual backup", RunE: backupRun(t, "CreateOrganizationBackup", `mutation CreateOrganizationBackup($orgId: ID!, $requestId: ID!, $selection: OrganizationBackupSelectionInput) { CreateOrganizationBackup(orgId: $orgId, requestId: $requestId, selection: $selection) { id state createdAt } }`, func(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
		id, err := backupID()
		if err != nil {
			return nil, err
		}
		v := map[string]any{"requestId": id}
		if createSelection != "" {
			v["selection"], err = backupSelection(createSelection)
		}
		return v, err
	})}
	create.Flags().StringVar(&createSelection, "selection-file", "", "owner-only scope override; admins use the saved policy")
	key := &cobra.Command{Use: "key", Short: "Manage the owner recovery key"}
	var keyOut string
	generate := &cobra.Command{Use: "generate", Short: "Generate a recovery key into a new private file", RunE: func(cmd *cobra.Command, _ []string) error {
		if keyOut == "" {
			return errors.New("--out is required")
		}
		s, err := t.session(cmd.Context())
		if err != nil {
			return err
		}
		file, err := os.OpenFile(keyOut, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600) // #nosec G304 -- the user selected this exclusive output file.
		if err != nil {
			return err
		}
		saved := false
		defer func() {
			if !saved {
				_ = file.Close()
				_ = os.Remove(keyOut)
			}
		}()
		raw, err := backupQuery(cmd.Context(), s, "GenerateOrganizationBackupKey", `mutation GenerateOrganizationBackupKey($orgId: ID!) { GenerateOrganizationBackupKey(orgId: $orgId) { id recipient identity } }`, map[string]any{"orgId": s.OrgID})
		if err != nil {
			return err
		}
		var result struct {
			ID        string `json:"id"`
			Recipient string `json:"recipient"`
			Identity  string `json:"identity"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return err
		}
		if result.Identity == "" {
			return errors.New("server returned no recovery identity")
		}
		if _, err = file.WriteString(result.Identity + "\n"); err != nil {
			return err
		}
		if err = file.Sync(); err != nil {
			return err
		}
		if err = file.Close(); err != nil {
			return err
		}
		saved = true
		return backupPrint(cmd, map[string]string{"id": result.ID, "recipient": result.Recipient, "savedTo": keyOut, "next": "backups key confirm --id " + result.ID})
	}}
	generate.Flags().StringVar(&keyOut, "out", "", "new file for the private recovery identity (mode 0600)")
	var keyID string
	confirm := &cobra.Command{Use: "confirm", Short: "Confirm the owner saved the recovery key", RunE: backupRun(t, "ConfirmOrganizationBackupKey", `mutation ConfirmOrganizationBackupKey($orgId: ID!, $keyId: ID!) { ConfirmOrganizationBackupKey(orgId: $orgId, keyId: $keyId) { Status } }`, func(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
		if keyID == "" {
			return nil, errors.New("--id is required")
		}
		return map[string]any{"keyId": keyID}, nil
	})}
	confirm.Flags().StringVar(&keyID, "id", "", "generated key ID")
	key.AddCommand(generate, confirm)
	policy := &cobra.Command{Use: "policy", Short: "View or set the backup policy"}
	policy.AddCommand(&cobra.Command{Use: "show", Short: "Show the backup policy with status", RunE: status.RunE})
	var policyFile string
	setPolicy := &cobra.Command{Use: "set", Short: "Set the owner's schedule, retention, selection and alert recipients from JSON", RunE: backupRun(t, "SaveOrganizationBackupPolicy", `mutation SaveOrganizationBackupPolicy($input: OrganizationBackupPolicyInput!) { SaveOrganizationBackupPolicy(input: $input) { Status } }`, func(_ *cobra.Command, s *backupSession) (map[string]any, error) {
		if policyFile == "" {
			return nil, errors.New("--file is required")
		}
		data, err := os.ReadFile(policyFile) // #nosec G304 -- the CLI user selected this input file.
		if err != nil {
			return nil, err
		}
		var input map[string]any
		if err := json.Unmarshal(data, &input); err != nil {
			return nil, err
		}
		input["orgId"] = s.OrgID
		return map[string]any{"input": input}, nil
	})}
	setPolicy.Flags().StringVar(&policyFile, "file", "", "JSON policy: enabled, schedule, timeZone, retentionDays, selection, notificationUsers")
	policy.AddCommand(setPolicy)
	var backupIDFlag, out string
	download := &cobra.Command{Use: "download", Short: "Download and verify an encrypted archive (owner only)", RunE: func(cmd *cobra.Command, _ []string) error {
		if backupIDFlag == "" || out == "" {
			return errors.New("--id and --out are required")
		}
		s, err := t.session(cmd.Context())
		if err != nil {
			return err
		}
		raw, err := backupQuery(cmd.Context(), s, "OrganizationBackups", `query OrganizationBackups($orgId: ID!) { OrganizationBackups(orgId: $orgId) { jobs { id state sha256 sizeBytes } } }`, map[string]any{"orgId": s.OrgID})
		if err != nil {
			return err
		}
		var overview struct {
			Jobs []struct {
				ID     string `json:"id"`
				State  string `json:"state"`
				SHA256 string `json:"sha256"`
			} `json:"jobs"`
		}
		if err := json.Unmarshal(raw, &overview); err != nil {
			return err
		}
		var digest string
		for _, job := range overview.Jobs {
			if job.ID == backupIDFlag && job.State == "ready" {
				digest = job.SHA256
				break
			}
		}
		endpoint := s.Host + "/api/organizations/" + url.PathEscape(s.OrgID) + "/backups/" + url.PathEscape(backupIDFlag) + "/download"
		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept-Encoding", "identity")
		resp, err := s.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
		}
		if header := strings.Join(resp.Header.Values("Content-Digest"), ","); header != "" {
			responseDigest, err := backupContentDigest(header)
			if err != nil {
				return err
			}
			if digest != "" && !strings.EqualFold(digest, responseDigest) {
				return errors.New("backup digest changed between status and download")
			}
			digest = responseDigest
		}
		if len(digest) != 64 {
			return errors.New("server provided no valid backup SHA-256 digest")
		}
		if _, err := os.Stat(out); err == nil {
			return errors.New("output file already exists")
		} else if !os.IsNotExist(err) {
			return err
		}
		file, err := os.CreateTemp(filepath.Dir(out), ".whodb-archive-*")
		if err != nil {
			return err
		}
		defer func() { _ = os.Remove(file.Name()) }()
		hash := sha256.New()
		if _, err := io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
			_ = file.Close()
			return err
		}
		if err := file.Sync(); err != nil {
			_ = file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), digest) {
			return errors.New("downloaded archive hash does not match backup")
		}
		if err := os.Link(file.Name(), out); err != nil {
			return err
		}
		return backupPrint(cmd, map[string]string{"id": backupIDFlag, "savedTo": out, "sha256": digest})
	}}
	download.Flags().StringVar(&backupIDFlag, "id", "", "ready backup ID")
	download.Flags().StringVar(&out, "out", "", "new encrypted archive file")
	var expiryID, expiryAt string
	expiry := &cobra.Command{Use: "extend", Short: "Extend a backup's expiry (owner only)", RunE: backupRun(t, "ExtendOrganizationBackupExpiry", `mutation ExtendOrganizationBackupExpiry($orgId: ID!, $backupId: ID!, $expiresAt: Time!) { ExtendOrganizationBackupExpiry(orgId: $orgId, backupId: $backupId, expiresAt: $expiresAt) { Status } }`, func(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
		if expiryID == "" || expiryAt == "" {
			return nil, errors.New("--id and --expires-at are required")
		}
		return map[string]any{"backupId": expiryID, "expiresAt": expiryAt}, nil
	})}
	expiry.Flags().StringVar(&expiryID, "id", "", "backup ID")
	expiry.Flags().StringVar(&expiryAt, "expires-at", "", "new RFC3339 expiry time")
	var deleteID string
	var deleteYes bool
	deleteCmd := &cobra.Command{Use: "delete", Short: "Request backup deletion after 24 hours (owner; recent sign-in required)", RunE: backupRun(t, "DeleteOrganizationBackup", `mutation DeleteOrganizationBackup($orgId: ID!, $backupId: ID!) { DeleteOrganizationBackup(orgId: $orgId, backupId: $backupId) { Status } }`, func(_ *cobra.Command, s *backupSession) (map[string]any, error) {
		if deleteID == "" || !deleteYes {
			return nil, fmt.Errorf("deleting backup %s in organization %s requires --id and --yes", deleteID, s.OrgID)
		}
		return map[string]any{"backupId": deleteID}, nil
	})}
	deleteCmd.Flags().StringVar(&deleteID, "id", "", "backup ID")
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "confirm permanent deletion")
	restores := &cobra.Command{Use: "restores", Short: "List restore previews and progress (owner only)", RunE: backupRun(t, "OrganizationBackupRestores", `query OrganizationBackupRestores($orgId: ID!) { OrganizationBackupRestores(orgId: $orgId) { id backupId mode state previewDigest safetyBackupId errorCode createdAt completedAt } }`, backupVars)}
	restore := &cobra.Command{Use: "restore", Short: "Preview and start owner recovery"}
	var restoreBackup, restoreKey, restoreMode, restoreProject string
	preview := &cobra.Command{Use: "preview", Short: "Preview a restore and get its digest (owner only)", RunE: backupRun(t, "PreviewOrganizationBackupRestore", `mutation PreviewOrganizationBackupRestore($orgId: ID!, $backupId: ID!, $identity: String!, $mode: String!, $targetProjectId: ID) { PreviewOrganizationBackupRestore(orgId: $orgId, backupId: $backupId, identity: $identity, mode: $mode, targetProjectId: $targetProjectId) { id backupId mode state previewDigest preview { sourceOrganizationId capturedAt projects resources { projectId type id name automatic } files rows requirements { kind projectId id version } excluded destinations { sourceProjectId sourceName projectId name } referenceWarnings { resourceId field originalId recoveredId } idMapping { originalId recoveredId } } } }`, func(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
		if restoreBackup == "" || restoreKey == "" {
			return nil, errors.New("--id and --key-file are required")
		}
		if restoreMode != "copy" && restoreMode != "replace" && restoreMode != "current" {
			return nil, errors.New("--mode must be copy, replace, or current")
		}
		if restoreMode == "current" && restoreProject == "" {
			return nil, errors.New("--project is required for current mode")
		}
		data, err := os.ReadFile(restoreKey) // #nosec G304 -- the owner selected this private key file.
		if err != nil {
			return nil, err
		}
		v := map[string]any{"backupId": restoreBackup, "identity": strings.TrimSpace(string(data)), "mode": restoreMode}
		if restoreProject != "" {
			v["targetProjectId"] = restoreProject
		}
		return v, nil
	})}
	preview.Flags().StringVar(&restoreBackup, "id", "", "backup ID")
	preview.Flags().StringVar(&restoreKey, "key-file", "", "private recovery identity file")
	preview.Flags().StringVar(&restoreMode, "mode", "copy", "copy, replace, or current")
	preview.Flags().StringVar(&restoreProject, "project", "", "destination project ID for current mode")
	var cancelID string
	cancelDelete := &cobra.Command{Use: "cancel-delete", Short: "Cancel a pending backup deletion (owner only)", RunE: backupRun(t, "CancelOrganizationBackupDeletion", `mutation CancelOrganizationBackupDeletion($orgId: ID!, $backupId: ID!) { CancelOrganizationBackupDeletion(orgId: $orgId, backupId: $backupId) { Status } }`, func(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
		if cancelID == "" {
			return nil, errors.New("--id is required")
		}
		return map[string]any{"backupId": cancelID}, nil
	})}
	cancelDelete.Flags().StringVar(&cancelID, "id", "", "backup ID")
	var restoreID, digest string
	var replaceYes bool
	start := &cobra.Command{Use: "start", Short: "Start a previewed restore (owner only)", RunE: backupRun(t, "StartOrganizationBackupRestore", `mutation StartOrganizationBackupRestore($orgId: ID!, $restoreId: ID!, $previewDigest: String!, $confirmReplacement: Boolean!) { StartOrganizationBackupRestore(orgId: $orgId, restoreId: $restoreId, previewDigest: $previewDigest, confirmReplacement: $confirmReplacement) { Status } }`, func(_ *cobra.Command, s *backupSession) (map[string]any, error) {
		if restoreID == "" || digest == "" {
			return nil, errors.New("--id and --digest from the preview are required")
		}
		if !replaceYes {
			return nil, fmt.Errorf("starting restore %s in organization %s requires --yes", restoreID, s.OrgID)
		}
		return map[string]any{"restoreId": restoreID, "previewDigest": digest, "confirmReplacement": replaceYes}, nil
	})}
	start.Flags().StringVar(&restoreID, "id", "", "restore preview ID")
	start.Flags().StringVar(&digest, "digest", "", "exact preview digest")
	start.Flags().BoolVar(&replaceYes, "yes", false, "confirm restore; replacement requires this flag")
	restore.AddCommand(preview, start)
	var noticeID string
	readNotice := &cobra.Command{Use: "read-notification", Short: "Mark a failure notification as read", RunE: backupRun(t, "ReadOrganizationBackupNotification", `mutation ReadOrganizationBackupNotification($orgId: ID!, $notificationId: ID!) { ReadOrganizationBackupNotification(orgId: $orgId, notificationId: $notificationId) { Status } }`, func(_ *cobra.Command, _ *backupSession) (map[string]any, error) {
		if noticeID == "" {
			return nil, errors.New("--id is required")
		}
		return map[string]any{"notificationId": noticeID}, nil
	})}
	readNotice.Flags().StringVar(&noticeID, "id", "", "notification ID")
	var uploadPath string
	var retentionDays int
	upload := &cobra.Command{Use: "upload", Short: "Import an encrypted organization backup archive (owner only)", RunE: func(cmd *cobra.Command, _ []string) error {
		if uploadPath == "" || retentionDays < 1 || retentionDays > 36500 {
			return errors.New("--file and --retention-days (1..36500) are required")
		}
		s, err := t.session(cmd.Context())
		if err != nil {
			return err
		}
		return uploadBackup(cmd, s, uploadPath, retentionDays)
	}}
	upload.Flags().StringVar(&uploadPath, "file", "", "encrypted .age archive")
	upload.Flags().IntVar(&retentionDays, "retention-days", 0, "retention for the imported archive")
	root.AddCommand(status, scope, create, key, policy, notifications, readNotice, download, upload, expiry, deleteCmd, cancelDelete, restores, restore)
	return root
}

func backupContentDigest(header string) (string, error) {
	var digest string
	for _, member := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(member), "=")
		if !ok || key != "sha-256" {
			continue
		}
		value = strings.TrimSpace(value)
		if digest != "" || len(value) < 2 || value[0] != ':' || value[len(value)-1] != ':' {
			return "", errors.New("invalid Content-Digest SHA-256")
		}
		raw, err := base64.StdEncoding.Strict().DecodeString(value[1 : len(value)-1])
		if err != nil || len(raw) != sha256.Size {
			return "", errors.New("invalid Content-Digest SHA-256")
		}
		digest = hex.EncodeToString(raw)
	}
	if digest == "" {
		return "", errors.New("Content-Digest has no SHA-256 value")
	}
	return digest, nil
}

func uploadBackup(cmd *cobra.Command, s *backupSession, path string, retentionDays int) error {
	file, err := os.Open(path) // #nosec G304 -- the owner selected this encrypted archive file.
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() || stat.Size() < 1 {
		return errors.New("archive must be a nonempty regular file")
	}
	raw, err := backupQuery(cmd.Context(), s, "BeginOrganizationBackupUpload", `mutation BeginOrganizationBackupUpload($orgId: ID!, $sizeBytes: String!, $retentionDays: Int!) { BeginOrganizationBackupUpload(orgId: $orgId, sizeBytes: $sizeBytes, retentionDays: $retentionDays) { id partBytes } }`, map[string]any{"orgId": s.OrgID, "sizeBytes": strconv.FormatInt(stat.Size(), 10), "retentionDays": retentionDays})
	if err != nil {
		return err
	}
	var reservation struct {
		ID        string `json:"id"`
		PartBytes int64  `json:"partBytes"`
	}
	if err := json.Unmarshal(raw, &reservation); err != nil {
		return err
	}
	if reservation.ID == "" || reservation.PartBytes < 1 || reservation.PartBytes > 32<<20 {
		return errors.New("invalid upload reservation")
	}
	completed := false
	defer func() {
		if !completed {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(cmd.Context()), 10*time.Second)
			defer cancel()
			_, _ = backupQuery(cleanupCtx, s, "CancelOrganizationBackupUpload", `mutation CancelOrganizationBackupUpload($orgId: ID!, $uploadId: ID!) { CancelOrganizationBackupUpload(orgId: $orgId, uploadId: $uploadId) { Status } }`, map[string]any{"orgId": s.OrgID, "uploadId": reservation.ID})
		}
	}()
	for offset, part := int64(0), 0; offset < stat.Size(); offset, part = offset+reservation.PartBytes, part+1 {
		length := min(reservation.PartBytes, stat.Size()-offset)
		if err := uploadBackupPart(cmd.Context(), s, reservation.ID, part, io.NewSectionReader(file, offset, length)); err != nil {
			return err
		}
	}
	raw, err = backupQuery(cmd.Context(), s, "CompleteOrganizationBackupUpload", `mutation CompleteOrganizationBackupUpload($orgId: ID!, $uploadId: ID!) { CompleteOrganizationBackupUpload(orgId: $orgId, uploadId: $uploadId) { id state sizeBytes sha256 } }`, map[string]any{"orgId": s.OrgID, "uploadId": reservation.ID})
	if err != nil {
		return err
	}
	completed = true
	return backupPrint(cmd, raw)
}

func uploadBackupPart(ctx context.Context, s *backupSession, uploadID string, part int, content io.Reader) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	variables := map[string]any{"orgId": s.OrgID, "uploadId": uploadID, "part": part, "file": nil}
	operations, err := json.Marshal(map[string]any{"query": `mutation UploadOrganizationBackupPart($orgId: ID!, $uploadId: ID!, $part: Int!, $file: Upload!) { UploadOrganizationBackupPart(orgId: $orgId, uploadId: $uploadId, part: $part, file: $file) { Status } }`, "variables": variables})
	if err != nil {
		return err
	}
	if err := writer.WriteField("operations", string(operations)); err != nil {
		return err
	}
	if err := writer.WriteField("map", `{"0":["variables.file"]}`); err != nil {
		return err
	}
	field, err := writer.CreateFormFile("0", "archive-part")
	if err != nil {
		return err
	}
	if _, err := io.Copy(field, content); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Host+"/api/query", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := s.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("archive part upload failed: HTTP %d", resp.StatusCode)
	}
	var envelope struct {
		Data struct {
			Uploaded struct {
				Status bool `json:"Status"`
			} `json:"UploadOrganizationBackupPart"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		return errors.New(envelope.Errors[0].Message)
	}
	if !envelope.Data.Uploaded.Status {
		return errors.New("archive part was not accepted")
	}
	return nil
}
