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
  CreateFunctionInput, FunctionRun, FunctionTestResult, OntologyFunction,
  TestFunctionInput, UpdateFunctionInput,
} from './generated/types.js';
import { warnIfFlagged } from './manifest-check.js';
import { VersionsHandle } from './versions.js';
import type { ObjectVersion } from './generated/types.js';
import { NotFoundError, ValidationError } from './errors.js';
import { isTerminalRunStatus, pollUntil, type WaitOptions } from './polling.js';

/** Fields accepted when creating a function; projectId is injected. */
export type FunctionDefinition = Omit<CreateFunctionInput, 'projectId'>;
/** Fields accepted when updating a function; id and projectId are injected. */
export type FunctionUpdate = Omit<UpdateFunctionInput, 'id' | 'projectId'>;

/** Options for a one-off test execution of the draft function. */
export interface FunctionTestOptions {
  files?: TestFunctionInput['files'];
  inputFileIds?: string[];
}

/** Encodes a function input value as the JSON string the platform expects. */
function encodeInput(input: unknown): string {
  return typeof input === 'string' ? input : JSON.stringify(input ?? {});
}

/**
 * FunctionsHandle is the `whodb.functions()` facade: the project's function
 * collection.
 */
export class FunctionsHandle {
  constructor(
    private readonly transport: Transport,
    private readonly projectId: () => Promise<string>,
  ) {}

  /** Lists the project's functions. */
  async list(): Promise<OntologyFunction[]> {
    warnIfFlagged('ProjectFunctions');
    return ops.projectFunctions(this.transport, { projectId: await this.projectId() });
  }

  /** Creates a function from its files and runtime settings. */
  async create(definition: FunctionDefinition): Promise<OntologyFunction> {
    warnIfFlagged('CreateFunction');
    return ops.createFunction(this.transport, {
      input: { ...definition, projectId: await this.projectId() },
    });
  }
}

/**
 * FunctionHandle is the `whodb.function("normalize")` facade, addressed by
 * function name or id.
 */
export class FunctionHandle {
  private functionCache: OntologyFunction | null = null;

  constructor(
    private readonly transport: Transport,
    private readonly projectId: () => Promise<string>,
    private readonly name: string,
  ) {}

  /** Resolves and caches the function backing this handle. */
  async meta(): Promise<OntologyFunction> {
    if (this.functionCache) return this.functionCache;
    warnIfFlagged('ProjectFunctions');
    const functions = await ops.projectFunctions(this.transport, { projectId: await this.projectId() });
    const fn = functions.find(f => f.id === this.name || f.name === this.name);
    if (!fn) throw new NotFoundError(`function "${this.name}" not found in this project`);
    this.functionCache = fn;
    return fn;
  }

  /** Executes the active version synchronously and returns its result. */
  async execute(input: unknown = {}, inputFileIds?: string[]): Promise<FunctionTestResult> {
    const fn = await this.meta();
    warnIfFlagged('ExecuteFunction');
    return ops.executeFunction(this.transport, {
      projectId: await this.projectId(),
      functionId: fn.id,
      input: encodeInput(input),
      inputFileIds: inputFileIds ?? null,
    });
  }

  /** Runs the draft version once with test input; nothing is persisted. */
  async test(input: unknown = {}, options: FunctionTestOptions = {}): Promise<FunctionTestResult> {
    const fn = await this.meta();
    warnIfFlagged('TestFunction');
    return ops.testFunction(this.transport, {
      input: {
        projectId: await this.projectId(),
        functionId: fn.id,
        input: encodeInput(input),
        files: options.files ?? null,
        inputFileIds: options.inputFileIds ?? null,
      },
    });
  }

  /** Starts a persisted run and returns it immediately; follow it with wait(). */
  async startRun(input: unknown = {}, options: { useActiveVersion?: boolean } = {}): Promise<FunctionRun> {
    const fn = await this.meta();
    warnIfFlagged('StartFunctionRun');
    return ops.startFunctionRun(this.transport, {
      projectId: await this.projectId(),
      functionId: fn.id,
      input: encodeInput(input),
      useActiveVersion: options.useActiveVersion ?? null,
    });
  }

  /** Reads one persisted run. */
  async run(runId: string): Promise<FunctionRun> {
    if (!runId) throw new ValidationError('runId is required');
    warnIfFlagged('FunctionRun');
    return ops.functionRun(this.transport, { projectId: await this.projectId(), id: runId });
  }

  /** Lists persisted runs, newest first. */
  async runs(limit = 20): Promise<FunctionRun[]> {
    const fn = await this.meta();
    warnIfFlagged('FunctionRuns');
    return ops.functionRuns(this.transport, { projectId: await this.projectId(), functionId: fn.id, limit });
  }

  /** Polls a run until it reaches a terminal status and returns it. */
  async wait(runId: string, options: WaitOptions = {}): Promise<FunctionRun> {
    if (!runId) throw new ValidationError('runId is required');
    return pollUntil(
      () => this.run(runId),
      run => isTerminalRunStatus(run.status),
      options,
      `function run ${runId}`,
    );
  }

  /** Requests cancellation of a queued or running run. */
  async cancelRun(runId: string): Promise<FunctionRun> {
    if (!runId) throw new ValidationError('runId is required');
    warnIfFlagged('CancelFunctionRun');
    return ops.cancelFunctionRun(this.transport, { projectId: await this.projectId(), id: runId });
  }

  /** Deploys the active version. */
  async deploy(): Promise<void> {
    const fn = await this.meta();
    warnIfFlagged('DeployFunction');
    await ops.deployFunction(this.transport, { projectId: await this.projectId(), id: fn.id });
  }

  /** Redeploys the active version. */
  async redeploy(): Promise<void> {
    const fn = await this.meta();
    warnIfFlagged('RedeployFunction');
    await ops.redeployFunction(this.transport, { projectId: await this.projectId(), id: fn.id });
  }

  /** Updates the draft function; send only changed fields. */
  async update(changes: FunctionUpdate): Promise<OntologyFunction> {
    const fn = await this.meta();
    warnIfFlagged('UpdateFunction');
    const updated = await ops.updateFunction(this.transport, {
      input: { ...changes, id: fn.id, projectId: await this.projectId() },
    });
    this.functionCache = updated;
    return updated;
  }

  /** Promotes the draft function: snapshots it and makes it the active production version. */
  async promote(message?: string): Promise<ObjectVersion> {
    const fn = await this.meta();
    return new VersionsHandle(this.transport, this.projectId).promote('function', fn.id, message);
  }

  /** Lists the function's promoted versions, newest first. */
  async versions(): Promise<ObjectVersion[]> {
    const fn = await this.meta();
    return new VersionsHandle(this.transport, this.projectId).list('function', fn.id);
  }

  /** Soft-deletes the function. */
  async delete(): Promise<void> {
    const fn = await this.meta();
    warnIfFlagged('DeleteFunction');
    await ops.deleteFunction(this.transport, { projectId: await this.projectId(), id: fn.id });
    this.functionCache = null;
  }
}
