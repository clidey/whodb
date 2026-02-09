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
 * Fixture Schema Validator for E2E Tests.
 * Validates that database fixtures conform to the expected schema.
 * Ported 1:1 from cypress/support/helpers/fixture-validator.js
 */

const REQUIRED_FIELDS = ["type", "category", "connection", "features"];

const TEST_TABLE_REQUIRED_FIELDS = {
  sql: ["name", "identifierField", "identifierColIndex", "testValues"],
  document: ["name", "identifierField", "testValues"],
  keyvalue: ["name", "identifierField", "testValues"],
};

const TEST_VALUES_REQUIRED_FIELDS = ["original", "modified", "rowIndex"];

const REQUIRED_FEATURES = [
  "graph",
  "export",
  "scratchpad",
  "mockData",
  "chat",
  "whereConditions",
  "queryHistory",
];

export const VALID_FEATURES = [
  ...REQUIRED_FEATURES,
  "crud",
  "scratchpadUpdate",
  "multiConditionFilter",
  "typeCasting",
  "sslConnection",
];

export function validateFixture(fixture, name) {
  const errors = [];

  for (const field of REQUIRED_FIELDS) {
    if (fixture[field] === undefined) {
      errors.push(`Missing required field: ${field}`);
    }
  }

  const validCategories = ["sql", "document", "keyvalue"];
  if (fixture.category && !validCategories.includes(fixture.category)) {
    errors.push(
      `Invalid category: ${fixture.category}. Must be one of: ${validCategories.join(", ")}`
    );
  }

  if (fixture.features) {
    for (const feature of REQUIRED_FEATURES) {
      if (typeof fixture.features[feature] !== "boolean") {
        errors.push(
          `Missing or invalid feature flag: ${feature} (must be boolean)`
        );
      }
    }

    for (const feature of Object.keys(fixture.features)) {
      if (!VALID_FEATURES.includes(feature)) {
        errors.push(
          `Unknown feature: '${feature}'. Valid features: ${VALID_FEATURES.join(", ")}`
        );
      }
    }
  }

  if (fixture.testTable) {
    const category = fixture.category;
    const requiredTestTableFields = TEST_TABLE_REQUIRED_FIELDS[category] || [];

    for (const field of requiredTestTableFields) {
      if (fixture.testTable[field] === undefined) {
        errors.push(`testTable missing required field: ${field}`);
      }
    }

    if (fixture.testTable.testValues) {
      for (const field of TEST_VALUES_REQUIRED_FIELDS) {
        if (fixture.testTable.testValues[field] === undefined) {
          errors.push(
            `testTable.testValues missing required field: ${field}`
          );
        }
      }
    }
  } else {
    errors.push(
      "Missing testTable config - required for feature-focused tests"
    );
  }

  if (fixture.connection) {
    if (typeof fixture.connection !== "object") {
      errors.push("connection must be an object");
    }
  }

  if (fixture.features && fixture.featureNotes) {
    for (const [feature, enabled] of Object.entries(fixture.features)) {
      if (!enabled && !fixture.featureNotes[feature]) {
        errors.push(
          `Warning: Feature '${feature}' is disabled but has no explanation in featureNotes`
        );
      }
    }
  }

  return {
    valid: errors.filter((e) => !e.startsWith("Warning:")).length === 0,
    errors,
    warnings: errors.filter((e) => e.startsWith("Warning:")),
  };
}

export function validateAllFixtures(fixtures) {
  const results = {};
  let allValid = true;

  for (const [name, fixture] of Object.entries(fixtures)) {
    const result = validateFixture(fixture, name);
    results[name] = result;

    if (!result.valid) {
      allValid = false;
      console.error(`Fixture validation failed for ${name}:`);
      result.errors.forEach((err) => console.error(`   - ${err}`));
    } else if (result.warnings.length > 0) {
      console.warn(`Fixture ${name} has warnings:`);
      result.warnings.forEach((warn) => console.warn(`   - ${warn}`));
    }
  }

  return { allValid, results };
}

export function assertFixturesValid(fixtures) {
  const { allValid, results } = validateAllFixtures(fixtures);

  if (!allValid) {
    const failedFixtures = Object.entries(results)
      .filter(([_, r]) => !r.valid)
      .map(
        ([name, r]) =>
          `${name}: ${r.errors.filter((e) => !e.startsWith("Warning:")).join(", ")}`
      )
      .join("\n");

    throw new Error(`Fixture validation failed:\n${failedFixtures}`);
  }
}
