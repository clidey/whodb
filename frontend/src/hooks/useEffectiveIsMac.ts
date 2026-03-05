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

import { useAppSelector } from '../store/hooks';
import { isMacPlatform } from '../utils/platform';

/**
 * Returns whether the effective platform is Mac, respecting any OS override
 * stored in settings (set via the ?os= URL parameter).
 * Falls back to system detection when no override is set.
 */
export function useEffectiveIsMac(): boolean {
    const os = useAppSelector(state => state.settings.os);
    if (os === undefined) return isMacPlatform;
    return os === 'macos';
}
