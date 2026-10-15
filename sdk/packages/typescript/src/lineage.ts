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

import type { Transport } from './transport.js';
import * as ops from './generated/operations.js';
import type { LineageProvider, SaveLineageProviderInput } from './generated/types.js';
import { warnIfFlagged } from './manifest-check.js';
import { ValidationError } from './errors.js';

/** Fields accepted when saving a provider; projectId is injected and id selects an existing provider. */
export type LineageProviderDefinition = Omit<SaveLineageProviderInput, 'projectId'>;

/**
 * LineageProvidersHandle is the `whodb.lineageProviders()` facade: external
 * OpenLineage providers feeding the project's lineage graph.
 */
export class LineageProvidersHandle {
  constructor(
    private readonly transport: Transport,
    private readonly projectId: () => Promise<string>,
  ) {}

  /** Lists the project's providers; API keys are never returned. */
  async list(): Promise<LineageProvider[]> {
    warnIfFlagged('ProjectLineageProviders');
    return ops.projectLineageProviders(this.transport, { projectId: await this.projectId() });
  }

  /** Creates a provider, or updates the one whose id is given. */
  async save(definition: LineageProviderDefinition): Promise<LineageProvider> {
    warnIfFlagged('SaveLineageProvider');
    return ops.saveLineageProvider(this.transport, {
      input: { ...definition, projectId: await this.projectId() },
    });
  }

  /** Deletes a provider. */
  async delete(providerId: string): Promise<void> {
    if (!providerId) throw new ValidationError('providerId is required');
    warnIfFlagged('DeleteLineageProvider');
    await ops.deleteLineageProvider(this.transport, { projectId: await this.projectId(), providerId });
  }
}
