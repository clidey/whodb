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

import { expect } from "@playwright/test";

/**
 * Document Database Helpers
 * Used for: MongoDB, Elasticsearch
 */

/**
 * Parse document from table row
 * @param {Array} row - Row data from getTableData
 * @returns {Object} Parsed JSON document
 */
export function parseDocument(row) {
    // Document is in column 1 (after checkbox at column 0)
    return JSON.parse(row[1]);
}

/**
 * Verify document matches expected data (ignoring _id)
 * @param {Object|string} doc - Document or JSON string
 * @param {Object} expected - Expected document properties
 */
export function verifyDocument(doc, expected) {
    const parsed = typeof doc === 'string' ? JSON.parse(doc) : doc;
    const {_id, ...rest} = parsed;
    Object.entries(expected).forEach(([key, value]) => {
        expect(rest[key], `Document field ${key}`).toEqual(value);
    });
}

/**
 * Verify document row from table data
 * @param {Array} row - Row data from getTableData
 * @param {Object} expected - Expected document properties
 */
export function verifyDocumentRow(row, expected) {
    const doc = parseDocument(row);
    verifyDocument(doc, expected);
}

/**
 * Verify multiple document rows
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {Array<Object>} expectedDocs - Expected document properties
 */
export function verifyDocumentRows(rows, expectedDocs) {
    expect(rows.length).toEqual(expectedDocs.length);
    expectedDocs.forEach((expected, idx) => {
        verifyDocumentRow(rows[idx], expected);
    });
}

/**
 * Get document field value from row
 * @param {Array} row - Row data
 * @param {string} field - Field name
 * @returns {*} Field value
 */
export function getDocumentField(row, field) {
    const doc = parseDocument(row);
    return doc[field];
}

/**
 * Get document _id from row
 * @param {Array} row - Row data
 * @returns {string} Document ID
 */
export function getDocumentId(row) {
    const doc = parseDocument(row);
    return doc._id;
}

/**
 * Create updated document JSON for editing
 * @param {Array} row - Original row data
 * @param {Object} updates - Fields to update
 * @returns {string} JSON string for updateRow
 */
export function createUpdatedDocument(row, updates) {
    const doc = parseDocument(row);
    return JSON.stringify({...doc, ...updates});
}

/**
 * Verify collection metadata from explore view
 * @param {Array<Array>} fields - Fields from getExploreFields
 * @param {Object} metadata - Expected metadata
 */
export function verifyMetadata(fields, metadata) {
    if (metadata.type) {
        const typeField = fields.find(([k]) => k === 'Type');
        expect(typeField, 'Type field should exist').toBeDefined();
        expect(typeField[1]).toEqual(metadata.type);
    }
    if (metadata.hasStorageSize) {
        expect(fields.some(([k]) => k === 'Storage Size')).toBeTruthy();
    }
    if (metadata.hasCount) {
        expect(fields.some(([k]) => k === 'Count')).toBeTruthy();
    }
}

/**
 * Verify graph structure (same as SQL)
 * @param {Object} graph - Graph from getGraph
 * @param {Object} expectedNodes - Expected node connections
 */
export function verifyGraph(graph, expectedNodes) {
    Object.keys(expectedNodes).forEach(node => {
        expect(graph, `Graph should have node: ${node}`).toHaveProperty(node);
        expect(graph[node].sort()).toEqual(expectedNodes[node].sort());
    });
}

export default {
    parseDocument,
    verifyDocument,
    verifyDocumentRow,
    verifyDocumentRows,
    getDocumentField,
    getDocumentId,
    createUpdatedDocument,
    verifyMetadata,
    verifyGraph,
};
