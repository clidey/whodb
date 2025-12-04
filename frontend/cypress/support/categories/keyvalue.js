/*
 * Copyright 2025 Clidey, Inc.
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
 * Key-Value Database Helpers
 * Used for: Redis
 */

/**
 * Verify hash field value
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {string} field - Field name to find
 * @param {string} expectedValue - Expected value
 */
export function verifyHashField(rows, field, expectedValue) {
    const row = rows.find(r => r[1] === field);
    expect(row, `Hash field '${field}' should exist`).to.exist;
    expect(row[2]).to.equal(expectedValue);
}

/**
 * Get hash field value
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {string} field - Field name
 * @returns {string|null} Field value or null
 */
export function getHashFieldValue(rows, field) {
    const row = rows.find(r => r[1] === field);
    return row ? row[2] : null;
}

/**
 * Verify hash fields exist
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {Array<string>} fields - Expected field names
 */
export function verifyHashFields(rows, fields) {
    const actualFields = rows.map(r => r[1]);
    fields.forEach(field => {
        expect(actualFields, `Should contain field: ${field}`).to.include(field);
    });
}

/**
 * Verify list/set members
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {Array<string>} expectedMembers - Expected member values
 */
export function verifyMembers(rows, expectedMembers) {
    // Value is in column 2 for lists/sets (after checkbox and index)
    const actualMembers = rows.map(r => r[2]);
    expectedMembers.forEach(member => {
        expect(actualMembers).to.include(member);
    });
}

/**
 * Verify sorted set entries
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {Array<Object>} expectedEntries - Expected {member, score} pairs
 */
export function verifySortedSetEntries(rows, expectedEntries) {
    expectedEntries.forEach(({member, score}) => {
        const row = rows.find(r => r[2] === member);
        expect(row, `Sorted set should contain member: ${member}`).to.exist;
        if (score !== undefined) {
            expect(row[3]).to.equal(score.toString());
        }
    });
}

/**
 * Verify string value
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {string} expectedValue - Expected string value
 */
export function verifyStringValue(rows, expectedValue) {
    expect(rows.length).to.equal(1);
    expect(rows[0][1]).to.equal(expectedValue);
}

/**
 * Verify key metadata from explore view
 * @param {Array<Array>} fields - Fields from getExploreFields
 * @param {string} expectedType - Expected Redis type
 */
export function verifyKeyMetadata(fields, expectedType) {
    const typeField = fields.find(([k, v]) => k === 'Type' && v === expectedType);
    expect(typeField, `Key should be of type: ${expectedType}`).to.exist;
    expect(fields.some(([k]) => k === 'Size')).to.be.true;
}

/**
 * Verify table columns based on key type
 * @param {Array} columns - Columns from getTableData
 * @param {string} keyType - Redis key type
 */
export function verifyColumnsForType(columns, keyType) {
    const expectedColumns = {
        hash: ['', 'field', 'value'],
        list: ['', 'index', 'value'],
        set: ['', 'index', 'value'],
        zset: ['', 'index', 'member', 'score'],
        string: ['', 'value'],
    };
    expect(columns).to.deep.equal(expectedColumns[keyType] || expectedColumns.string);
}

/**
 * Filter out session keys from key list
 * @param {Array<string>} keys - List of Redis keys
 * @returns {Array<string>} Filtered keys
 */
export function filterSessionKeys(keys) {
    return keys.filter(key => !key.startsWith('session:'));
}

export default {
    verifyHashField,
    getHashFieldValue,
    verifyHashFields,
    verifyMembers,
    verifySortedSetEntries,
    verifyStringValue,
    verifyKeyMetadata,
    verifyColumnsForType,
    filterSessionKeys,
};
