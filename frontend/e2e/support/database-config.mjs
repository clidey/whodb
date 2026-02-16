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
 * Database configuration loader.
 *
 * Loads fixture JSON files from the CE (and optionally EE) fixtures directories.
 *
 * Environment variables:
 *   FIXTURES_DIR          - Override path to CE fixtures
 *   EE_FIXTURES_DIR       - Override path to EE fixtures
 *   GATEWAY_FIXTURES_DIR  - Override path to gateway fixtures (Docker hostnames)
 */

import {existsSync, readdirSync, readFileSync} from "fs";
import {join, resolve} from "path";

const thisDir = import.meta.dirname;

const FIXTURES_DIR =
  process.env.FIXTURES_DIR ||
  resolve(thisDir, "../fixtures/databases");

// EE fixtures only load when explicitly set via env var (by ee/dev/run-e2e.sh)
// or registered via registerDatabases() (by ee/frontend/e2e/support/test-runner.mjs)
const EE_FIXTURES_DIR = process.env.EE_FIXTURES_DIR || "";

// GATEWAY_FIXTURES_DIR removed - host overrides are now handled via
// DB_HOST_<NAME> env vars (e.g., DB_HOST_POSTGRES=e2e_postgres)

function loadJsonFile(filePath) {
  return JSON.parse(readFileSync(filePath, "utf-8"));
}

function loadFixturesFromDir(dir) {
  if (!dir || !existsSync(dir)) return {};
  const configs = {};
  for (const file of readdirSync(dir)) {
    if (!file.endsWith(".json")) continue;
    const name = file.replace(".json", "");
    configs[name] = loadJsonFile(join(dir, file));
  }
  return configs;
}

// EE database registrations (populated by ee/frontend/e2e/support/test-runner.mjs)
let additionalConfigs = {};

/** Register additional database configurations (used by EE) */
export function registerDatabases(configs) {
  additionalConfigs = { ...additionalConfigs, ...configs };
}

/** All database configs (CE + EE), with env-based host overrides applied */
export function getDatabaseConfigs() {
  const ce = loadFixturesFromDir(FIXTURES_DIR);
  const ee = loadFixturesFromDir(EE_FIXTURES_DIR);
  const all = { ...ce, ...ee, ...additionalConfigs };

  // Apply host overrides from environment (e.g., DB_HOST_POSTGRES=e2e_postgres)
  // This allows Docker/gateway environments to override fixture hostnames
  // without duplicating fixture files.
  for (const [name, config] of Object.entries(all)) {
    const envKey = `DB_HOST_${name.toUpperCase()}`;
    const hostOverride = process.env[envKey];
    if (hostOverride && config.connection) {
      config.connection.host = hostOverride;
    }
  }

  return all;
}

/** Get a single database config by name */
export function getDatabaseConfig(name) {
  const all = getDatabaseConfigs();
  const config = all[name.toLowerCase()];
  if (!config) {
    throw new Error(
      `Unknown database: ${name}. Available: ${Object.keys(all).join(", ")}`
    );
  }
  return config;
}

/** Get all database configurations */
export function getAllDatabaseConfigs() {
  return getDatabaseConfigs();
}

/** Filter databases by category */
export function getDatabasesByCategory(category) {
  const all = Object.values(getDatabaseConfigs());
  if (category === "all") return all;
  return all.filter((db) => db.category === category);
}

/** Check if a database supports a feature */
export function hasFeature(dbConfig, feature) {
  return dbConfig.features?.[feature] === true;
}

/** Get database identifier */
export function getDatabaseId(dbConfig) {
  return dbConfig.id || dbConfig.type.toLowerCase();
}

/** Get SQL query for database */
export function getSqlQuery(dbConfig, queryKey) {
  if (!dbConfig.sql?.[queryKey]) {
    throw new Error(`SQL query '${queryKey}' not found for ${dbConfig.type}`);
  }
  return dbConfig.sql[queryKey];
}

/** Get expected error pattern for database */
export function getErrorPattern(dbConfig, errorKey) {
  return dbConfig.sql?.errorPatterns?.[errorKey] ?? null;
}

/** Get table configuration from database config */
export function getTableConfig(dbConfig, tableName) {
  return dbConfig.tables?.[tableName] ?? null;
}
