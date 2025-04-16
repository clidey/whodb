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

describe('SQLite Data Type Handling Tests', () => {
  const testDbName = 'data_types_test.db';
  
  beforeEach(() => {
    // Login to the application
    cy.login('Sqlite3', undefined, undefined, undefined, testDbName);
    cy.selectDatabase(testDbName);
    
    // Set up test database with both STRICT and non-STRICT tables
    cy.goto("scratchpad");
    
    // Drop test tables if they exist
    cy.writeCode(0, `
      DROP TABLE IF EXISTS regular_table;
      DROP TABLE IF EXISTS strict_table;
    `);
    cy.runCode(0);
    
    // Create a regular (non-STRICT) table with various data types
    cy.writeCode(0, `
      CREATE TABLE regular_table (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        description TEXT,
        int_value INTEGER,
        real_value REAL,
        boolean_value BOOLEAN,
        date_value DATE,
        datetime_value DATETIME
      );
    `);
    cy.runCode(0);
    
    // Create a STRICT table with the same columns
    cy.writeCode(0, `
      CREATE TABLE strict_table (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        description TEXT,
        int_value INTEGER,
        real_value REAL,
        boolean_value BOOLEAN,
        date_value DATE,
        datetime_value DATETIME
      ) STRICT;
    `);
    cy.runCode(0);
  });
  
  it('should allow flexible data types in non-STRICT tables', () => {
    // Insert data with type mismatches into the regular table
    cy.writeCode(0, `
      -- All of these should work in a non-STRICT table despite type mismatches
      INSERT INTO regular_table 
        (description, int_value, real_value, boolean_value, date_value, datetime_value) 
      VALUES
        ('Correct types', 42, 3.14, 1, '2023-05-17', '2023-05-17 21:54:13'),
        ('String as int', 'forty-two', 3.14, 1, '2023-05-17', '2023-05-17 21:54:13'),
        ('String as real', 42, 'pi', 1, '2023-05-17', '2023-05-17 21:54:13'),
        ('String as bool', 42, 3.14, 'true', '2023-05-17', '2023-05-17 21:54:13'),
        ('Text as date', 42, 3.14, 1, 'May 17, 2023', '2023-05-17 21:54:13'),
        ('Text as datetime', 42, 3.14, 1, '2023-05-17', '05/17/2023 09:54 PM');
    `);
    cy.runCode(0);
    cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));
    
    // Check that all rows were inserted
    cy.writeCode(0, `SELECT COUNT(*) as count FROM regular_table;`);
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows }) => {
      expect(rows[0]).to.include('6'); // All 6 rows should be inserted
    });
    
    // View the table data to verify values were stored as provided
    cy.goto("data");
    cy.explore("regular_table");
    cy.data("regular_table");
    
    cy.getTableData().then(({ columns, rows }) => {
      // Get column indexes
      const intColumnIndex = columns.findIndex(col => col === "int_value [INTEGER]");
      const realColumnIndex = columns.findIndex(col => col === "real_value [REAL]");
      const boolColumnIndex = columns.findIndex(col => col === "boolean_value [BOOLEAN]");
      const dateColumnIndex = columns.findIndex(col => col === "date_value [DATE]");
      const datetimeColumnIndex = columns.findIndex(col => col === "datetime_value [DATETIME]");
      
      // Get row with string as int
      const stringAsIntRow = rows.find(row => row.includes('String as int'));
      expect(stringAsIntRow[intColumnIndex]).to.equal('forty-two');
      
      // Get row with string as real
      const stringAsRealRow = rows.find(row => row.includes('String as real'));
      expect(stringAsRealRow[realColumnIndex]).to.equal('pi');
      
      // Get row with string as bool
      const stringAsBoolRow = rows.find(row => row.includes('String as bool'));
      expect(stringAsBoolRow[boolColumnIndex]).to.equal('true');
      
      // Get row with text as date
      const textAsDateRow = rows.find(row => row.includes('Text as date'));
      expect(textAsDateRow[dateColumnIndex]).to.equal('May 17, 2023');
      
      // Get row with text as datetime
      const textAsDatetimeRow = rows.find(row => row.includes('Text as datetime'));
      expect(textAsDatetimeRow[datetimeColumnIndex]).to.equal('05/17/2023 09:54 PM');
    });
    
    // Test querying with the non-standard values
    cy.goto("scratchpad");
    cy.writeCode(0, `
      -- Even though 'forty-two' was stored, SQLite lets us search for it as 'forty-two'
      SELECT * FROM regular_table WHERE int_value = 'forty-two';
    `);
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows }) => {
      expect(rows.length).to.equal(1);
      expect(rows[0]).to.include('String as int');
    });
  });
  
  it('should strictly enforce types in STRICT tables', () => {
    // Insert correct types into the STRICT table - should work
    cy.writeCode(0, `
      -- This should work as all types match
      INSERT INTO strict_table 
        (description, int_value, real_value, boolean_value, date_value, datetime_value) 
      VALUES
        ('Correct types', 42, 3.14, 1, '2023-05-17', '2023-05-17 21:54:13');
    `);
    cy.runCode(0);
    cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));
    
    // Attempt to insert mismatched types - should fail
    cy.writeCode(0, `
      -- This should fail in STRICT mode
      INSERT INTO strict_table 
        (description, int_value, real_value, boolean_value, date_value, datetime_value) 
      VALUES
        ('String as int', 'forty-two', 3.14, 1, '2023-05-17', '2023-05-17 21:54:13');
    `);
    cy.runCode(0);
    cy.getCellError(0).then(error => {
      expect(error).to.include('INSERT'); // Should have an error message
      expect(error).to.include('forty-two'); // Error should mention the problematic value
    });
    
    // Attempt to insert mismatched real - should fail
    cy.writeCode(0, `
      -- This should fail in STRICT mode
      INSERT INTO strict_table 
        (description, int_value, real_value, boolean_value, date_value, datetime_value) 
      VALUES
        ('String as real', 42, 'pi', 1, '2023-05-17', '2023-05-17 21:54:13');
    `);
    cy.runCode(0);
    cy.getCellError(0).then(error => {
      expect(error).to.include('INSERT');
      expect(error).to.include('pi');
    });
    
    // Test a custom format date/time in strict table - should fail
    cy.writeCode(0, `
      -- This should fail in STRICT mode due to non-standard datetime format
      INSERT INTO strict_table 
        (description, int_value, real_value, boolean_value, date_value, datetime_value) 
      VALUES
        ('Custom datetime', 42, 3.14, 1, '2023-05-17', '05/17/2023 09:54 PM');
    `);
    cy.runCode(0);
    cy.getCellError(0).then(error => {
      expect(error).to.include('INSERT');
      expect(error).to.include('05/17/2023 09:54 PM');
    });
    
    // Verify only the first row was inserted
    cy.writeCode(0, `SELECT COUNT(*) as count FROM strict_table;`);
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows }) => {
      expect(rows[0]).to.include('1'); // Only the first row should be inserted
    });
    
    // View the STRICT table data
    cy.goto("data");
    cy.explore("strict_table");
    cy.data("strict_table");
    
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(1); // Only one row should exist
      expect(rows[0]).to.include('Correct types'); 
    });
  });
  
  it('should handle WHERE conditions differently between STRICT and non-STRICT tables', () => {
    // Insert test data
    cy.writeCode(0, `
      -- Insert into regular table
      INSERT INTO regular_table 
        (description, int_value, real_value, boolean_value, date_value, datetime_value) 
      VALUES
        ('Regular 1', 42, 3.14, 1, '2023-05-17', '2023-05-17 21:54:13');
        
      -- Insert into strict table
      INSERT INTO strict_table 
        (description, int_value, real_value, boolean_value, date_value, datetime_value) 
      VALUES
        ('Strict 1', 42, 3.14, 1, '2023-05-17', '2023-05-17 21:54:13');
    `);
    cy.runCode(0);
    
    // Test type conversion in WHERE clause for non-STRICT table
    cy.writeCode(0, `
      -- In non-STRICT table, string '42' is converted to integer 42 for comparison
      SELECT * FROM regular_table WHERE int_value = '42';
    `);
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows }) => {
      expect(rows.length).to.equal(1); // Should find the row
      expect(rows[0]).to.include('Regular 1');
    });
    
    // Test boolean values in non-STRICT table
    cy.writeCode(0, `
      -- In non-STRICT table, string 'true' is evaluated as a truthy value
      SELECT * FROM regular_table WHERE boolean_value = 'true';
    `);
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows }) => {
      expect(rows.length).to.equal(1); // Should find the row
    });
    
    // Test type conversion in STRICT table
    cy.writeCode(0, `
      -- In STRICT table, string '42' should also match integer 42
      SELECT * FROM strict_table WHERE int_value = '42';
    `);
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows }) => {
      expect(rows.length).to.equal(1); // SQLite still converts for comparison
      expect(rows[0]).to.include('Strict 1');
    });
  });
  
  afterEach(() => {
    // Clean up - drop the test tables
    cy.goto("scratchpad");
    cy.writeCode(0, `
      DROP TABLE IF EXISTS regular_table;
      DROP TABLE IF EXISTS strict_table;
    `);
    cy.runCode(0);
    
    // Logout
    cy.logout();
  });
}); 