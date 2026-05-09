/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package audit

import (
	"context"
	"sync"
)

// AuditService defines the interface for recording system audit events.
type AuditService interface {
	Record(event AuditEvent)
}

// ActorProvider enriches audit events with caller information from a request
// context when one is available.
type ActorProvider func(ctx context.Context) Actor

// EventEnricher enriches audit events with implementation-specific metadata
// derived from the current context or the event payload itself.
type EventEnricher func(ctx context.Context, event AuditEvent) AuditEvent

var (
	currentService       AuditService  = &noOpService{}
	currentActorProvider ActorProvider = noOpActorProvider
	currentEventEnricher EventEnricher = noOpEventEnricher
	mu                   sync.RWMutex
)

// Record captures an audit event using the globally configured audit service.
func Record(event AuditEvent) {
	mu.RLock()
	defer mu.RUnlock()
	currentService.Record(prepareEvent(context.Background(), event, currentActorProvider, currentEventEnricher))
}

// RecordWithContext captures an audit event and enriches it from the supplied
// request context when an actor provider has been registered.
func RecordWithContext(ctx context.Context, event AuditEvent) {
	mu.RLock()
	defer mu.RUnlock()
	currentService.Record(prepareEvent(ctx, event, currentActorProvider, currentEventEnricher))
}

// SetAuditService injects a concrete implementation of the AuditService.
// This is used by the EE binary to provide the real auditing engine.
func SetAuditService(svc AuditService) {
	mu.Lock()
	defer mu.Unlock()
	if svc == nil {
		svc = &noOpService{}
	}
	currentService = svc
}

// SetActorProvider injects an actor provider for request-scoped audit
// enrichment. Passing nil resets the provider to the default no-op behavior.
func SetActorProvider(provider ActorProvider) {
	mu.Lock()
	defer mu.Unlock()
	if provider == nil {
		provider = noOpActorProvider
	}
	currentActorProvider = provider
}

// SetEventEnricher injects an event enricher for request- and system-scoped
// audit metadata. Passing nil resets the enricher to the default no-op
// behavior.
func SetEventEnricher(enricher EventEnricher) {
	mu.Lock()
	defer mu.Unlock()
	if enricher == nil {
		enricher = noOpEventEnricher
	}
	currentEventEnricher = enricher
}

// noOpService is the default implementation that drops all events.
type noOpService struct{}

func (s *noOpService) Record(event AuditEvent) {}

func noOpActorProvider(context.Context) Actor {
	return Actor{}
}

func noOpEventEnricher(_ context.Context, event AuditEvent) AuditEvent {
	return event
}
