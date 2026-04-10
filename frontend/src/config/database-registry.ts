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

import { IDatabaseDropdownItem } from './database-types';

/**
 * Database type registry for dynamically registered database types.
 *
 * Extensions register additional database types at boot.
 *
 * 
 */
let registeredExtensionDatabases: IDatabaseDropdownItem[] = [];

/** Register additional database types (called by extension modules at boot). */
export const registerDatabaseTypes = (types: IDatabaseDropdownItem[]) => {
    registeredExtensionDatabases = types;
};

/** Get all registered extension database types. */
export const getRegisteredDatabaseTypes = (): IDatabaseDropdownItem[] => {
    return registeredExtensionDatabases;
};
