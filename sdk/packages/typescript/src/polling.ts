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

import { PlatformError } from './errors.js';

/** Options for waiting on a run to reach a terminal state. */
export interface WaitOptions {
  /** Give up after this many milliseconds (default 60000). */
  timeoutMs?: number;
  /** Delay between polls in milliseconds (default 2000). */
  pollMs?: number;
}

export const DEFAULT_WAIT_TIMEOUT_MS = 60_000;
export const DEFAULT_WAIT_POLL_MS = 2_000;

/** Run statuses after which a function or transform run will not change. */
export function isTerminalRunStatus(status: string | null | undefined): boolean {
  switch ((status ?? '').trim().toLowerCase()) {
    case 'success':
    case 'succeeded':
    case 'completed':
    case 'complete':
    case 'failed':
    case 'failure':
    case 'cancelled':
    case 'canceled':
    case 'error':
      return true;
    default:
      return false;
  }
}

/**
 * Polls `read` until `done` reports a terminal value or the timeout elapses.
 * The timeout error is a PlatformError with code RUN_TIMEOUT so callers can
 * keep polling with the run id they already hold.
 */
export async function pollUntil<T>(
  read: () => Promise<T>,
  done: (value: T) => boolean,
  options: WaitOptions,
  describe: string,
): Promise<T> {
  const timeoutMs = options.timeoutMs ?? DEFAULT_WAIT_TIMEOUT_MS;
  const pollMs = options.pollMs ?? DEFAULT_WAIT_POLL_MS;
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const value = await read();
    if (done(value)) return value;
    if (Date.now() >= deadline) {
      throw new PlatformError(`${describe} did not finish within ${timeoutMs}ms`, 'RUN_TIMEOUT');
    }
    if (pollMs > 0) await new Promise(resolve => setTimeout(resolve, pollMs));
  }
}
