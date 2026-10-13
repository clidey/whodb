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
import type {
  OntologyReference, OntologyReferenceImpact, SharedOntology, SharedOntologyVisibility,
} from './generated/types.js';
import { warnIfFlagged } from './manifest-check.js';
import { ValidationError } from './errors.js';

/**
 * SharingHandle is the `whodb.sharing()` facade: share ontologies with other
 * projects in the organization and reference shared ontologies locally.
 */
export class SharingHandle {
  constructor(
    private readonly transport: Transport,
    private readonly projectId: () => Promise<string>,
  ) {}

  /** Ontologies this project shares, with their consumers. */
  async shared(): Promise<SharedOntology[]> {
    warnIfFlagged('SharedOntologies');
    return ops.sharedOntologies(this.transport, { projectId: await this.projectId() });
  }

  /** Ontologies other projects share that this project may reference. */
  async discover(): Promise<SharedOntology[]> {
    warnIfFlagged('DiscoverSharedOntologies');
    return ops.discoverSharedOntologies(this.transport, { consumerProjectId: await this.projectId() });
  }

  /** Shared ontologies this project references. */
  async references(): Promise<OntologyReference[]> {
    warnIfFlagged('OntologyReferences');
    return ops.ontologyReferences(this.transport, { projectId: await this.projectId() });
  }

  /** Counts dependents of a reference before removing it. */
  async referenceImpact(referenceId: string): Promise<OntologyReferenceImpact> {
    if (!referenceId) throw new ValidationError('referenceId is required');
    warnIfFlagged('OntologyReferenceImpact');
    return ops.ontologyReferenceImpact(this.transport, { referenceId });
  }

  /** Shares one of this project's ontologies. */
  async share(ontologyId: string, visibility: SharedOntologyVisibility, allowedProjectIds?: string[]): Promise<SharedOntology> {
    if (!ontologyId) throw new ValidationError('ontologyId is required');
    warnIfFlagged('ShareOntology');
    return ops.shareOntology(this.transport, {
      input: {
        projectId: await this.projectId(),
        ontologyId,
        visibility,
        allowedProjectIds: allowedProjectIds ?? null,
      },
    });
  }

  /** Stops sharing an ontology; existing references become broken. */
  async unshare(ontologyId: string): Promise<void> {
    if (!ontologyId) throw new ValidationError('ontologyId is required');
    warnIfFlagged('UnshareOntology');
    await ops.unshareOntology(this.transport, { projectId: await this.projectId(), ontologyId });
  }

  /** References a shared ontology from this project under a local alias. */
  async createReference(ownerProjectId: string, ontologyId: string, alias: string): Promise<OntologyReference> {
    if (!ownerProjectId || !ontologyId || !alias) {
      throw new ValidationError('ownerProjectId, ontologyId and alias are required');
    }
    warnIfFlagged('CreateOntologyReference');
    return ops.createOntologyReference(this.transport, {
      input: { consumerProjectId: await this.projectId(), ownerProjectId, ontologyId, alias },
    });
  }

  /** Removes a reference. Check referenceImpact() first. */
  async removeReference(referenceId: string): Promise<void> {
    if (!referenceId) throw new ValidationError('referenceId is required');
    warnIfFlagged('RemoveOntologyReference');
    await ops.removeOntologyReference(this.transport, { referenceId });
  }
}
