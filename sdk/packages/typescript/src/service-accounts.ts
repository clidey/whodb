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
import type { CreatedServiceAccountKey, ServiceAccount } from './generated/types.js';
import { warnIfFlagged } from './manifest-check.js';
import { ValidationError } from './errors.js';

/**
 * ServiceAccountsHandle is the `whodb.serviceAccounts()` facade: API access
 * principals for the organization the client is scoped to.
 */
export class ServiceAccountsHandle {
  constructor(
    private readonly transport: Transport,
    private readonly orgId: () => Promise<string>,
  ) {}

  /** Lists service accounts with key metadata; raw keys are never returned. */
  async list(): Promise<ServiceAccount[]> {
    warnIfFlagged('ServiceAccounts');
    return ops.serviceAccounts(this.transport, { orgId: await this.orgId() });
  }

  /** Creates a service account. Issue a key with createKey(). */
  async create(name: string, description?: string): Promise<ServiceAccount> {
    if (!name?.trim()) throw new ValidationError('name is required');
    warnIfFlagged('CreateServiceAccount');
    return ops.createServiceAccount(this.transport, {
      orgId: await this.orgId(),
      name: name.trim(),
      description: description ?? null,
    });
  }

  /**
   * Issues an API key. `rawKey` is returned exactly once; store it securely
   * and never log it.
   */
  async createKey(serviceAccountId: string, expiresAt?: string): Promise<CreatedServiceAccountKey> {
    if (!serviceAccountId) throw new ValidationError('serviceAccountId is required');
    warnIfFlagged('CreateServiceAccountKey');
    return ops.createServiceAccountKey(this.transport, {
      orgId: await this.orgId(),
      serviceAccountId,
      expiresAt: expiresAt ?? null,
    });
  }

  /** Revokes one key. */
  async revokeKey(keyId: string): Promise<void> {
    if (!keyId) throw new ValidationError('keyId is required');
    warnIfFlagged('RevokeServiceAccountKey');
    await ops.revokeServiceAccountKey(this.transport, { orgId: await this.orgId(), keyId });
  }

  /** Deletes a service account and revokes all of its keys. */
  async delete(serviceAccountId: string): Promise<void> {
    if (!serviceAccountId) throw new ValidationError('serviceAccountId is required');
    warnIfFlagged('DeleteServiceAccount');
    await ops.deleteServiceAccount(this.transport, { orgId: await this.orgId(), id: serviceAccountId });
  }
}
