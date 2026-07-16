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

/**
 * Matches credential value keys that carry secret material and must never be
 * persisted to browser storage: the password, and advanced fields such as an
 * SSL client private key or a passphrase/token. Non-secret advanced fields
 * (SSL mode, server name, cert content, paths, search path, port) are kept so
 * the connection can still be displayed and reconnected.
 */
const SECRET_KEY_PATTERN = /(password|passphrase|secret|token|private key|key content)/i;

/** Reports whether a credential value key holds secret material. */
export function isSecretCredentialKey(key: string | undefined): boolean {
  return SECRET_KEY_PATTERN.test(String(key ?? ''));
}

/**
 * Returns a copy of a stored login profile with secret material removed:
 * the top-level Password cleared, AccessToken dropped, and any secret entries
 * removed from Values and Advanced. Non-secret fields are preserved so the
 * profile switcher keeps working.
 */
export function stripProfileSecrets(profile: any): any {
  if (!profile || typeof profile !== 'object') return profile;
  const { Password: _password, AccessToken: _accessToken, ...rest } = profile;
  const filterSecrets = (entries: any) =>
    Array.isArray(entries) ? entries.filter((entry: any) => !isSecretCredentialKey(entry?.Key)) : entries;
  return {
    ...rest,
    Password: '',
    Values: filterSecrets(profile.Values),
    Advanced: filterSecrets(profile.Advanced),
  };
}
