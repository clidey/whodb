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
import type { ActiveProdVersion, ObjectVersion, VersionableObjectType } from './generated/types.js';
import { warnIfFlagged } from './manifest-check.js';
import { ValidationError } from './errors.js';

const OBJECT_TYPES: readonly VersionableObjectType[] = ['app', 'function', 'transform', 'dataset', 'ontology'];

/** Validates an object type against the platform's versionable set. */
export function assertObjectType(objectType: string): VersionableObjectType {
  if (!(OBJECT_TYPES as readonly string[]).includes(objectType)) {
    throw new ValidationError(`objectType must be one of ${OBJECT_TYPES.join(', ')}; got "${objectType}"`);
  }
  return objectType as VersionableObjectType;
}

/**
 * VersionsHandle is the `whodb.versions()` facade: version history, the active
 * production version, promotion, rollback, and restore-to-draft for any
 * versionable object (app, function, transform, dataset, ontology).
 */
export class VersionsHandle {
  constructor(
    private readonly transport: Transport,
    private readonly projectId: () => Promise<string>,
  ) {}

  /** Lists an object's promoted versions, newest first. */
  async list(objectType: VersionableObjectType, objectId: string): Promise<ObjectVersion[]> {
    assertObjectType(objectType);
    if (!objectId) throw new ValidationError('objectId is required');
    warnIfFlagged('ObjectVersions');
    return ops.objectVersions(this.transport, { projectId: await this.projectId(), objectId, objectType });
  }

  /** The active production version, or null when nothing has been promoted. */
  async active(objectType: VersionableObjectType, objectId: string): Promise<ActiveProdVersion | null> {
    assertObjectType(objectType);
    if (!objectId) throw new ValidationError('objectId is required');
    warnIfFlagged('ActiveProdVersion');
    return (await ops.activeProdVersion(this.transport, { projectId: await this.projectId(), objectId, objectType })) ?? null;
  }

  /** Every active production version in the project. */
  async activeAll(): Promise<ActiveProdVersion[]> {
    warnIfFlagged('ProjectActiveProdVersions');
    return ops.projectActiveProdVersions(this.transport, { projectId: await this.projectId() });
  }

  /**
   * Promotes the object's draft: snapshots it as a new version and makes that
   * version the active production version in one step.
   */
  async promote(objectType: VersionableObjectType, objectId: string, message?: string): Promise<ObjectVersion> {
    assertObjectType(objectType);
    if (!objectId) throw new ValidationError('objectId is required');
    warnIfFlagged('PromoteObject');
    return ops.promoteObject(this.transport, {
      input: { projectId: await this.projectId(), objectId, objectType, message },
    });
  }

  /** Makes an earlier promoted version the active production version. */
  async rollback(objectType: VersionableObjectType, objectId: string, version: number): Promise<ActiveProdVersion> {
    assertObjectType(objectType);
    if (!objectId) throw new ValidationError('objectId is required');
    warnIfFlagged('RollbackObject');
    return ops.rollbackObject(this.transport, {
      input: { projectId: await this.projectId(), objectId, objectType, version },
    });
  }

  /** Copies a promoted version back into the object's draft. Returns the restored object. */
  async restoreToDraft(objectType: VersionableObjectType, objectId: string, version: number): Promise<unknown> {
    assertObjectType(objectType);
    if (!objectId) throw new ValidationError('objectId is required');
    const projectId = await this.projectId();
    switch (objectType) {
      case 'ontology':
        warnIfFlagged('RestoreOntologyVersionToDraft');
        return ops.restoreOntologyVersionToDraft(this.transport, { projectId, ontologyId: objectId, version });
      case 'dataset':
        warnIfFlagged('RestoreDatasetVersionToDraft');
        return ops.restoreDatasetVersionToDraft(this.transport, { projectId, datasetId: objectId, version });
      case 'transform':
        warnIfFlagged('RestoreTransformVersionToDraft');
        return ops.restoreTransformVersionToDraft(this.transport, { projectId, transformId: objectId, version });
      case 'function':
        warnIfFlagged('RestoreFunctionVersionToDraft');
        return ops.restoreFunctionVersionToDraft(this.transport, { projectId, functionId: objectId, version });
      case 'app':
        warnIfFlagged('RestoreAppVersionToDraft');
        return ops.restoreAppVersionToDraft(this.transport, { projectId, appId: objectId, version });
    }
  }
}
