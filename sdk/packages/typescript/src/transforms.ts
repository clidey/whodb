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
import type { SaveTransformInput, Transform, TransformRun } from './generated/types.js';
import { hydrateRows, type Row } from './hydrate.js';
import { warnIfFlagged } from './manifest-check.js';
import { VersionsHandle } from './versions.js';
import type { ObjectVersion } from './generated/types.js';
import { NotFoundError, ValidationError } from './errors.js';
import { isTerminalRunStatus, pollUntil, type WaitOptions } from './polling.js';

/** Fields accepted when creating or updating a transform; projectId is injected. */
export type TransformDefinition = Omit<SaveTransformInput, 'id' | 'projectId'>;

/**
 * TransformsHandle is the `whodb.transforms()` facade: the project's
 * transform collection.
 */
export class TransformsHandle {
  constructor(
    private readonly transport: Transport,
    private readonly projectId: () => Promise<string>,
  ) {}

  /** Lists the project's transforms. */
  async list(): Promise<Transform[]> {
    warnIfFlagged('ProjectTransforms');
    return ops.projectTransforms(this.transport, { projectId: await this.projectId() });
  }

  /** Creates a transform from a pipeline graph. */
  async create(definition: TransformDefinition): Promise<Transform> {
    warnIfFlagged('SaveTransform');
    return ops.saveTransform(this.transport, {
      input: { ...definition, projectId: await this.projectId() },
    });
  }

  /** Runs a pipeline graph without saving it and returns the resulting rows. */
  async preview(graphJson: string): Promise<{ rows: Row[]; totalCount: number | null }> {
    warnIfFlagged('PreviewTransform');
    const result = await ops.previewTransform(this.transport, { projectId: await this.projectId(), graphJson });
    return hydrateRows(result);
  }
}

/**
 * TransformHandle is the `whodb.transform("nightly-rollup")` facade,
 * addressed by transform name or id.
 */
export class TransformHandle {
  private transformCache: Transform | null = null;

  constructor(
    private readonly transport: Transport,
    private readonly projectId: () => Promise<string>,
    private readonly name: string,
  ) {}

  /** Resolves and caches the transform backing this handle. */
  async meta(): Promise<Transform> {
    if (this.transformCache) return this.transformCache;
    warnIfFlagged('ProjectTransforms');
    const transforms = await ops.projectTransforms(this.transport, { projectId: await this.projectId() });
    const transform = transforms.find(t => t.id === this.name || t.name === this.name);
    if (!transform) throw new NotFoundError(`transform "${this.name}" not found in this project`);
    this.transformCache = transform;
    return transform;
  }

  /** Lists recent runs, newest first. */
  async runs(limit = 20): Promise<TransformRun[]> {
    const transform = await this.meta();
    warnIfFlagged('TransformRuns');
    return ops.transformRuns(this.transport, { projectId: await this.projectId(), transformId: transform.id, limit });
  }

  /** Starts a run and returns its record immediately; follow it with wait(). */
  async run(): Promise<TransformRun> {
    const transform = await this.meta();
    warnIfFlagged('RunTransform');
    return ops.runTransform(this.transport, { projectId: await this.projectId(), id: transform.id });
  }

  /** Cancels the active run, or a specific run when runId is given. */
  async cancelRun(runId?: string): Promise<void> {
    const transform = await this.meta();
    warnIfFlagged('CancelTransformRun');
    await ops.cancelTransformRun(this.transport, {
      projectId: await this.projectId(),
      transformId: transform.id,
      runId: runId ?? null,
    });
  }

  /** Polls until the run reaches a terminal status and returns it. */
  async wait(runId: string, options: WaitOptions = {}): Promise<TransformRun> {
    if (!runId) throw new ValidationError('runId is required');
    const transform = await this.meta();
    return pollUntil(
      async () => {
        warnIfFlagged('TransformRuns');
        const runs = await ops.transformRuns(this.transport, { projectId: await this.projectId(), transformId: transform.id, limit: 100 });
        const run = runs.find(r => r.id === runId);
        if (!run) throw new NotFoundError(`run "${runId}" not found for transform "${this.name}"`);
        return run;
      },
      run => isTerminalRunStatus(run.status),
      options,
      `transform run ${runId}`,
    );
  }

  /** Replaces the transform definition. */
  async update(definition: TransformDefinition): Promise<Transform> {
    const transform = await this.meta();
    warnIfFlagged('SaveTransform');
    const saved = await ops.saveTransform(this.transport, {
      input: { ...definition, id: transform.id, projectId: await this.projectId() },
    });
    this.transformCache = saved;
    return saved;
  }

  /** Promotes the draft transform: snapshots it and makes it the active production version. */
  async promote(message?: string): Promise<ObjectVersion> {
    const transform = await this.meta();
    return new VersionsHandle(this.transport, this.projectId).promote('transform', transform.id, message);
  }

  /** Lists the transform's promoted versions, newest first. */
  async versions(): Promise<ObjectVersion[]> {
    const transform = await this.meta();
    return new VersionsHandle(this.transport, this.projectId).list('transform', transform.id);
  }

  /** Soft-deletes the transform. */
  async delete(): Promise<void> {
    const transform = await this.meta();
    warnIfFlagged('DeleteTransform');
    await ops.deleteTransform(this.transport, { projectId: await this.projectId(), id: transform.id });
    this.transformCache = null;
  }
}
