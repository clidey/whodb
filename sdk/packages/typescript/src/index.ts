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

export { WhoDB } from './client.js';
export type { WhoDBConfig } from './config.js';
export type { Transport } from './transport.js';
export { IpcTransport } from './transport-ipc.js';
export type { CredentialProvider } from './auth.js';
export { apiKeyProvider, tokenProvider, cliProvider } from './auth.js';
export {
  WhoDBError, AuthError, NotFoundError, ValidationError,
  WhoDBVersionError, CliCredentialsError, TransportCapabilityError, PlatformError,
} from './errors.js';
export type { Row } from './hydrate.js';
export type { Page } from './pagination.js';
export { ListCall } from './pagination.js';
export { OntologyHandle } from './ontology.js';
export type { ActionOptions } from './ontology.js';
export { DatasetHandle } from './dataset.js';
export { SourceHandle } from './source.js';
export { FilesHandle } from './files.js';
