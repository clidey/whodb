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

const BACKUP_HINT_STORAGE_KEY = '@clidey/whodb/platform-backup-hint-dismissed';

/** Whether the "back up your connections" hint was permanently dismissed. */
export const hasDismissedBackupHint = (): boolean => {
    return localStorage.getItem(BACKUP_HINT_STORAGE_KEY) === 'true';
};

/** Permanently dismisses the "back up your connections" hint. */
export const dismissBackupHint = (): void => {
    try {
        localStorage.setItem(BACKUP_HINT_STORAGE_KEY, 'true');
    } catch (error) {
        console.error('Failed to save backup hint state:', error);
    }
};
