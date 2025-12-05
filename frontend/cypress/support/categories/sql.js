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
 * SQL Database Helpers
 * Used for: Postgres, MySQL, MariaDB, SQLite, ClickHouse
 */

/**
 * Verify table row data matches expected values
 * @param {Array} row - Row data from getTableData
 * @param {Array} expected - Expected values (without checkbox column)
 */
export function verifyRow(row, expected) {
    // Row includes checkbox column at index 0, skip it
    expected.forEach((val, idx) => {
        expect(row[idx + 1]).to.equal(val);
    });
}

/**
 * Verify table rows match expected data
 * @param {Array<Array>} rows - Rows from getTableData
 * @param {Array<Array>} expected - Expected rows (each without leading checkbox)
 */
export function verifyRows(rows, expected) {
    expect(rows.length).to.equal(expected.length);
    expected.forEach((expectedRow, idx) => {
        verifyRow(rows[idx], expectedRow);
    });
}

/**
 * Verify column types from explore view
 * @param {Array<Array>} fields - Fields from getExploreFields [key, value] pairs
 * @param {Object} expectedColumns - Map of column name to type
 */
export function verifyColumnTypes(fields, expectedColumns) {
    Object.entries(expectedColumns).forEach(([col, type]) => {
        const found = fields.some(([k, v]) => k === col && v === type);
        expect(found, `Column ${col} should have type ${type}`).to.be.true;
    });
}

/**
 * Verify table metadata from explore view
 * @param {Array<Array>} fields - Fields from getExploreFields
 * @param {Object} metadata - Expected metadata
 */
export function verifyMetadata(fields, metadata) {
    if (metadata.type) {
        expect(fields.some(([k, v]) => k === 'Type' && v === metadata.type)).to.be.true;
    }
    if (metadata.hasSize) {
        expect(fields.some(([k]) => k === 'Total Size' || k === 'Data Size')).to.be.true;
    }
    if (metadata.hasCount) {
        expect(fields.some(([k]) => k === 'Total Count:' || k === 'Total Count' || k === 'Count')).to.be.true;
    }
}

/**
 * Get row value by column index (accounting for checkbox column)
 * @param {Array} row - Row data
 * @param {number} colIndex - Column index (0-based, not counting checkbox)
 * @returns {string} Cell value
 */
export function getRowValue(row, colIndex) {
    return row[colIndex + 1]; // +1 for checkbox column
}

/**
 * Verify scratchpad query output
 * @param {Object} output - Output from getCellQueryOutput {columns, rows}
 * @param {Array} expectedColumns - Expected column names
 * @param {Array<Array>} expectedRows - Expected row data
 */
export function verifyScratchpadOutput(output, expectedColumns, expectedRows) {
    if (expectedColumns) {
        expect(output.columns).to.deep.equal(expectedColumns);
    }
    if (expectedRows) {
        expect(output.rows.map(row => row.slice(0, -1))).to.deep.equal(expectedRows);
    }
}

/**
 * Verify graph structure
 * @param {Object} graph - Graph from getGraph
 * @param {Object} expectedNodes - Expected node connections
 */
export function verifyGraph(graph, expectedNodes) {
    Object.keys(expectedNodes).forEach(node => {
        expect(graph, `Graph should have node: ${node}`).to.have.property(node);
        expect(graph[node].sort()).to.deep.equal(expectedNodes[node].sort());
    });
}

/**
 * Verify graph node metadata
 * @param {Array<Array>} fields - Fields from getGraphNode
 * @param {Object} expectedColumns - Expected column types
 * @param {Object} metadata - Expected metadata
 */
export function verifyGraphNode(fields, expectedColumns, metadata) {
    verifyColumnTypes(fields, expectedColumns);
    verifyMetadata(fields, metadata);
}

export default {
    verifyRow,
    verifyRows,
    verifyColumnTypes,
    verifyMetadata,
    getRowValue,
    verifyScratchpadOutput,
    verifyGraph,
    verifyGraphNode,
};
