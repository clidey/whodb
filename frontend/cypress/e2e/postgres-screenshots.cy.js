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

describe('Postgres Screenshot Generation', () => {
  const isDocker = Cypress.env('isDocker');
  const dbHost = isDocker ? 'screenshot_postgres' : 'localhost';
  const dbUser = 'user';
  const dbPassword = 'jio53$*(@nfe)';
  const screenshotDir = 'postgres';

  before(() => {
    cy.log('Starting Postgres screenshot generation suite');
  });

  it('01 - Login Page', () => {
    cy.visit('/login');
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/01-login-page`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('02 - Login Form - Database Type Selection', () => {
    cy.visit('/login');
    cy.get('[data-testid="database-type-select"]').click();
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/02-login-database-type-dropdown`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get(`[data-value="Postgres"]`).click();
  });

  it('03 - Login Form - Filled', () => {
    cy.visit('/login');
    cy.get('[data-testid="database-type-select"]').click();
    cy.get(`[data-value="Postgres"]`).click();
    cy.get('[data-testid="hostname"]').clear().type(dbHost);
    cy.get('[data-testid="username"]').clear().type(dbUser);
    cy.get('[data-testid="password"]').clear().type(dbPassword, {log: false});
    cy.get('[data-testid="database"]').clear().type('test_db');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/03-login-form-filled`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('04 - Login Form - Advanced Options', () => {
    cy.visit('/login');
    cy.get('[data-testid="database-type-select"]').click();
    cy.get(`[data-value="Postgres"]`).click();
    cy.get('[data-testid="hostname"]').clear().type(dbHost);
    cy.get('[data-testid="username"]').clear().type(dbUser);
    cy.get('[data-testid="password"]').clear().type(dbPassword, {log: false});
    cy.get('[data-testid="database"]').clear().type('test_db');
    cy.get('[data-testid="advanced-button"]').click();
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/04-login-advanced-options`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('05 - Storage Unit List Page', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.visit('/storage-unit');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/05-storage-unit-list`, {
      capture: 'fullPage',
      overwrite: true
    });
  });

  it('06 - Storage Unit List - With Sidebar', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.visit('/storage-unit');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/06-storage-unit-list-with-sidebar`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('07 - Table Explore View - Users Table', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.explore('users');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/07-explore-users-table`, {
      capture: 'fullPage',
      overwrite: true
    });
  });

  it('08 - Table Explore View - Table Metadata', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.explore('users');
    cy.wait(1000);
    cy.get('[data-testid="explore-fields"]').should('be.visible');
    cy.screenshot(`${screenshotDir}/08-explore-table-metadata`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('09 - Data View - Users Table', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/09-data-view-users-table`, {
      capture: 'fullPage',
      overwrite: true
    });
  });

  it('10 - Data View - Table with Data', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/10-data-view-table-content`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('11 - Data View - Add Row Dialog', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="add-row-button"]').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/11-data-view-add-row-dialog`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('12 - Data View - Add Row Dialog Filled', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="add-row-button"]').click();
    cy.get('[data-testid="add-row-field-id"] input').clear().type('5');
    cy.get('[data-testid="add-row-field-username"] input').clear().type('screenshot_user');
    cy.get('[data-testid="add-row-field-email"] input').clear().type('screenshot@example.com');
    cy.get('[data-testid="add-row-field-password"] input').clear().type('testpass123');
    cy.get('[data-testid="add-row-field-created_at"] input').clear().type('2025-01-15');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/12-data-view-add-row-filled`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('13 - Data View - Context Menu', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('table tbody tr').first().rightclick({ force: true });
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/13-data-view-context-menu`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').click(0, 0);
  });

  it('14 - Data View - Edit Row Dialog', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('table tbody tr').first().rightclick({ force: true });
    cy.get('[data-testid="context-menu-edit-row"]').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/14-data-view-edit-row-dialog`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('15 - Data View - Page Size Dropdown', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="table-page-size"]').click();
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/15-data-view-page-size-dropdown`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').click(0, 0);
  });

  it('16 - Data View - Where Conditions Popover', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="where-button"]').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/16-data-view-where-conditions-popover`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('17 - Data View - Where Conditions Field Dropdown', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="where-button"]').click();
    cy.wait(300);
    cy.get('body').then($body => {
      if ($body.find('[data-testid="field-key"]').length > 0) {
        cy.get('[data-testid="field-key"]').first().click();
        cy.wait(300);
        cy.screenshot(`${screenshotDir}/17-data-view-where-field-dropdown`, {
          capture: 'viewport',
          overwrite: true
        });
      }
    });
    cy.get('body').type('{esc}');
  });

  it('18 - Data View - Where Conditions with Badge', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '=', '1']]);
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/18-data-view-where-conditions-badge`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('19 - Data View - Search Functionality', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.searchTable('john');
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/19-data-view-search-highlight`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('20 - Data View - Export Dialog', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.contains('button', 'Export all').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/20-data-view-export-dialog`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('21 - Data View - Export Format Dropdown', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.contains('button', 'Export all').click();
    cy.wait(300);
    cy.get('[role="dialog"]').within(() => {
      cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
    });
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/21-data-view-export-format-dropdown`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('22 - Data View - Mock Data Dialog', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.selectMockData();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/22-data-view-mock-data-dialog`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('23 - Data View - Embedded Scratchpad Drawer', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="embedded-scratchpad-button"]').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/23-data-view-embedded-scratchpad`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('24 - Graph View - Schema Topology', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.screenshot(`${screenshotDir}/24-graph-view-schema-topology`, {
      capture: 'fullPage',
      overwrite: true
    });
  });

  it('25 - Graph View - With Layout Button', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.screenshot(`${screenshotDir}/25-graph-view-with-controls`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('26 - Graph View - Node with Details', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('[data-testid="rf__node-users"]').should('be.visible');
    cy.screenshot(`${screenshotDir}/26-graph-view-node-details`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('27 - Scratchpad - Main View', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/27-scratchpad-main-view`, {
      capture: 'fullPage',
      overwrite: true
    });
  });

  it('28 - Scratchpad - Code Editor', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.wait(500);
    cy.writeCode(0, 'SELECT * FROM test_schema.users ORDER BY id;');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/28-scratchpad-code-editor`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('29 - Scratchpad - Query Results', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT * FROM test_schema.users ORDER BY id;');
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/29-scratchpad-query-results`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('30 - Scratchpad - Query Error', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT * FROM test_schema.nonexistent_table;');
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/30-scratchpad-query-error`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('31 - Scratchpad - Multiple Pages', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.addScratchpadPage();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/31-scratchpad-multiple-pages`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('32 - Scratchpad - Cell Options Menu', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.wait(1500);
    cy.get('[role="tabpanel"][data-state="active"] [data-testid="cell-0"]').should('be.visible');
    cy.get('[role="tabpanel"][data-state="active"] [data-testid="cell-0"]').trigger('mouseover');
    cy.wait(500);
    cy.get('[role="tabpanel"][data-state="active"] [data-testid="cell-0"] [data-testid="icon-button"]').first().should('be.visible').click({force: true});
    cy.wait(1000);
    cy.get('[role="menu"]').should('be.visible');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/32-scratchpad-cell-options-menu`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').click(0, 0);
  });

  it('33 - Scratchpad - Query History Dialog', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT * FROM test_schema.users;');
    cy.runCode(0);
    cy.wait(1000);
    cy.openQueryHistory(0);
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/33-scratchpad-query-history`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('34 - Sidebar - Database Selector', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.visit('/storage-unit');
    cy.wait(1500);
    cy.get('[data-testid="sidebar-database"]').should('be.visible');
    cy.wait(300);
    cy.get('[data-testid="sidebar-database"]').click({force: true});
    cy.wait(800);
    cy.get('[role="listbox"]', {timeout: 5000}).should('be.visible');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/34-sidebar-database-selector`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
    cy.wait(300);
  });

  it('35 - Sidebar - Schema Selector', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.visit('/storage-unit');
    cy.wait(1500);
    cy.get('[data-testid="sidebar-schema"]').should('be.visible');
    cy.wait(300);
    cy.get('[data-testid="sidebar-schema"]').click({force: true});
    cy.wait(800);
    cy.get('[role="listbox"]', {timeout: 5000}).should('be.visible');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/35-sidebar-schema-selector`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
    cy.wait(300);
  });

  it('36 - Sidebar - Navigation Menu', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.visit('/storage-unit');
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/36-sidebar-navigation-menu`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('37 - Table Header - Context Menu', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('table thead tr.cursor-context-menu').first().rightclick({ force: true });
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/37-table-header-context-menu`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').click(0, 0);
  });

  it('38 - Data View - Sorted Column', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.sortBy(1);
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/38-data-view-sorted-column`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('39 - Data View - Multiple Where Conditions', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([
      ['id', '>', '0'],
      ['username', '!=', 'admin']
    ]);
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/39-data-view-multiple-conditions`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('40 - Scratchpad - Action Query Result', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, "UPDATE test_schema.users SET username='temp' WHERE id=999;");
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/40-scratchpad-action-result`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  // ============================================================================
  // ADDITIONAL INTERACTIONS - ADD/DELETE OPERATIONS (41-50)
  // ============================================================================

  it('41 - Add Row - Submit Button', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="add-row-button"]').click();
    cy.get('[data-testid="add-row-field-id"] input').clear().type('100');
    cy.get('[data-testid="add-row-field-username"] input').clear().type('test_user');
    cy.get('[data-testid="add-row-field-email"] input').clear().type('test@example.com');
    cy.get('[data-testid="add-row-field-password"] input').clear().type('password');
    cy.get('[data-testid="add-row-field-created_at"] input').clear().type('2025-01-01');
    cy.get('[data-testid="submit-add-row-button"]').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/41-add-row-submit-button`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('42 - Edit Row - Update Button Hover', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('table tbody tr').first().rightclick({ force: true });
    cy.get('[data-testid="context-menu-edit-row"]').click();
    cy.wait(300);
    cy.get('[data-testid="editable-field-2"]').clear().type('updated_name');
    cy.get('[data-testid="update-button"]').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/42-edit-row-update-hover`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('43 - Context Menu - Delete Row Option', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.get('table tbody tr').first().rightclick({ force: true });
    cy.get('[data-testid="context-menu-more-actions"]').click();
    cy.wait(200);
    cy.get('[data-testid="context-menu-delete-row"]').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/43-context-menu-delete-option`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').click(0, 0);
  });

  it('44 - Table - Row Selection Single', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('table tbody tr').first().click();
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/44-table-row-selection-single`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('45 - Table - Select Row from Context Menu', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('table tbody tr').first().rightclick({force: true});
    cy.contains('div,button,span', 'Select Row').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/45-context-menu-select-row`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').click(0, 0);
  });

  it('46 - Export - Selected Rows Mode', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.wait(1000);
    cy.get('table tbody tr').first().should('be.visible').rightclick({force: true});
    cy.wait(300);
    cy.contains('div,button,span', 'Select Row').should('be.visible').click({force: true});
    cy.wait(800);
    cy.get('button').contains('Export').filter(':visible').last().should('be.visible').click({force: true});
    cy.wait(1000);
    cy.get('[role="dialog"]').should('be.visible');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/46-export-selected-rows-dialog`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('47 - Mock Data - Overwrite Confirmation', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.wait(800);
    cy.selectMockData();
    cy.wait(500);
    cy.contains('label', 'Number of Rows').parent().find('input').should('be.visible').clear().type('10');
    cy.wait(300);
    cy.contains('label', 'Data Handling').parent().find('[role="combobox"]').should('be.visible').click({force: true});
    cy.wait(500);
    cy.contains('[role="option"]', 'Overwrite existing data').should('be.visible').click({force: true});
    cy.wait(300);
    cy.contains('button', 'Generate').should('be.visible').click({force: true});
    cy.wait(1000);
    cy.get('[role="alertdialog"],[role="dialog"]', {timeout: 5000}).should('be.visible');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/47-mock-data-overwrite-confirm`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
    cy.wait(200);
    cy.get('body').type('{esc}');
  });

  it('48 - Scratchpad - Multiple Cells with Results', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT * FROM test_schema.users ORDER BY id LIMIT 3;');
    cy.runCode(0);
    cy.wait(1000);
    cy.addCell(0);
    cy.writeCode(1, 'SELECT COUNT(*) as total FROM test_schema.users;');
    cy.runCode(1);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/48-scratchpad-multiple-cells-results`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('49 - Scratchpad - Query History Clone to Editor', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT * FROM test_schema.users;');
    cy.runCode(0);
    cy.wait(1000);
    cy.writeCode(0, 'SELECT * FROM test_schema.orders;');
    cy.runCode(0);
    cy.wait(1000);
    cy.openQueryHistory(0);
    cy.wait(500);
    cy.get('[data-testid="clone-to-editor-button"]').first().trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/49-scratchpad-history-clone-button`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('50 - Graph - Click Node Data Button', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('[data-testid="rf__node-users"] [data-testid="data-button"]').trigger('mouseover', {force: true});
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/50-graph-node-data-button-hover`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  // ============================================================================
  // SECTION: DETAILED DROPDOWNS & OPTIONS (51-75)
  // ============================================================================

  it('51 - Login - Database Type - All Options Visible', () => {
    cy.visit('/login');
    cy.get('[data-testid="database-type-select"]').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/51-login-database-types-all-options`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('52 - Login - Database Type - MySQL Selected', () => {
    cy.visit('/login');
    cy.get('[data-testid="database-type-select"]').click();
    cy.get('[data-value="MySQL"]').click();
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/52-login-mysql-selected`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('53 - Login - Database Type - MongoDB Selected', () => {
    cy.visit('/login');
    cy.get('[data-testid="database-type-select"]').click();
    cy.get('[data-value="MongoDB"]').click();
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/53-login-mongodb-selected`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('54 - Login - Database Type - Redis Selected', () => {
    cy.visit('/login');
    cy.get('[data-testid="database-type-select"]').click();
    cy.get('[data-value="Redis"]').click();
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/54-login-redis-selected`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('55 - Page Size - Dropdown All Options', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.get('[data-testid="table-page-size"]').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/55-page-size-all-options`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('56 - Page Size - 10 Rows Selected', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.setTablePageSize(10);
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/56-page-size-10-selected`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('57 - Page Size - 25 Rows Selected', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('orders');
    cy.setTablePageSize(25);
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/57-page-size-25-selected`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('58 - Page Size - 50 Rows Selected', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.setTablePageSize(50);
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/58-page-size-50-selected`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('59 - Where Operator - Equals Selected', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '=', '1']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/59-where-operator-equals`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('60 - Where Operator - Greater Than', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '>', '1']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/60-where-operator-greater-than`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('61 - Where Operator - Less Than', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '<', '3']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/61-where-operator-less-than`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('62 - Where Operator - Not Equals', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['username', '!=', 'admin_user']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/62-where-operator-not-equals`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('63 - Where Operator - Greater Than or Equal', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '>=', '2']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/63-where-operator-gte`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('64 - Where Operator - Less Than or Equal', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '<=', '2']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/64-where-operator-lte`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('65 - Export Format - CSV Option Highlighted', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.contains('button', 'Export all').click();
    cy.wait(300);
    cy.get('[role="dialog"]').within(() => {
      cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
    });
    cy.wait(300);
    cy.get('[role="option"]').contains('CSV').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/65-export-format-csv-option`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('66 - Export Format - Excel Option Highlighted', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.contains('button', 'Export all').click();
    cy.wait(300);
    cy.get('[role="dialog"]').within(() => {
      cy.contains('label', 'Format').parent().find('[role="combobox"]').first().click();
    });
    cy.wait(300);
    cy.get('[role="option"]').contains('Excel').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/66-export-format-excel-option`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('67 - Export Delimiter - Comma Option', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.contains('button', 'Export all').click();
    cy.wait(300);
    cy.get('[role="dialog"]').within(() => {
      cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').first().click();
    });
    cy.wait(300);
    cy.get('[role="option"]').contains('Comma').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/67-export-delimiter-comma`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('68 - Export Delimiter - Semicolon Option', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.wait(800);
    cy.contains('button', 'Export all').should('be.visible').click({force: true});
    cy.wait(500);
    cy.get('[role="dialog"]').should('be.visible').within(() => {
      cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').first().should('be.visible').click({force: true});
    });
    cy.wait(500);
    cy.get('[role="option"]').contains('Semicolon').should('be.visible').trigger('mouseover');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/68-export-delimiter-semicolon`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
    cy.wait(200);
    cy.get('body').type('{esc}');
  });

  it('69 - Export Delimiter - Pipe Option', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.wait(800);
    cy.contains('button', 'Export all').should('be.visible').click({force: true});
    cy.wait(500);
    cy.get('[role="dialog"]').should('be.visible').within(() => {
      cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').first().should('be.visible').click({force: true});
    });
    cy.wait(500);
    cy.get('[role="option"]').contains('Pipe').should('be.visible').trigger('mouseover');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/69-export-delimiter-pipe`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
    cy.wait(200);
    cy.get('body').type('{esc}');
  });

  it('70 - Export Delimiter - Tab Option', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.wait(800);
    cy.contains('button', 'Export all').should('be.visible').click({force: true});
    cy.wait(500);
    cy.get('[role="dialog"]').should('be.visible').within(() => {
      cy.contains('label', 'Delimiter').parent().find('[role="combobox"]').first().should('be.visible').click({force: true});
    });
    cy.wait(500);
    cy.get('[role="option"]').eq(3).should('be.visible').trigger('mouseover');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/70-export-delimiter-tab`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
    cy.wait(200);
    cy.get('body').type('{esc}');
  });

  it('71 - Mock Data - Append Mode Selected', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.selectMockData();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/71-mock-data-append-mode`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('72 - Mock Data - Overwrite Mode Options', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.wait(800);
    cy.selectMockData();
    cy.wait(500);
    cy.contains('label', 'Data Handling').parent().find('[role="combobox"]').should('be.visible').click({force: true});
    cy.wait(500);
    cy.get('[role="listbox"]').should('be.visible');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/72-mock-data-handling-options`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
    cy.wait(200);
    cy.get('body').type('{esc}');
  });

  it('73 - Mock Data - Row Count Min Value', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.selectMockData();
    cy.wait(300);
    cy.contains('label', 'Number of Rows').parent().find('input').clear().type('1');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/73-mock-data-row-count-min`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('74 - Mock Data - Row Count Medium Value', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.selectMockData();
    cy.wait(300);
    cy.contains('label', 'Number of Rows').parent().find('input').clear().type('100');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/74-mock-data-row-count-medium`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  it('75 - Mock Data - Row Count Max Value', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.selectMockData();
    cy.wait(300);
    cy.contains('label', 'Number of Rows').parent().find('input').clear().type('300');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/75-mock-data-row-count-max-clamped`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.get('body').type('{esc}');
  });

  // ============================================================================
  // SECTION: TABLE STATES & EDGE CASES (76-90)
  // ============================================================================

  it('76 - Table - Empty State No Data', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '=', '999999']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/76-table-empty-state-no-results`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('77 - Table - Single Row Result', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.whereTable([['id', '=', '1']]);
    cy.submitTable();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/77-table-single-row-result`, {
      capture: 'viewport',
      overwrite: true
    });
    cy.clearWhereConditions();
  });

  it('78 - Table - Many Columns Wide Table', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('orders');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/78-table-many-columns-wide`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('79 - Table - With Null Values', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('payments');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/79-table-with-null-values`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('80 - Table - With Long Text Content', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/80-table-long-text-content`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('81 - Table - With Dates and Timestamps', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('users');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/81-table-dates-timestamps`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('82 - Table - With Numeric Data Types', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('test_casting');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/82-table-numeric-types`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('83 - Pagination - First Page', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.wait(800);
    cy.setTablePageSize(2);
    cy.submitTable();
    cy.wait(1000);
    cy.get('[data-testid="table-page-number"]').should('be.visible');
    cy.wait(300);
    cy.screenshot(`${screenshotDir}/83-pagination-first-page`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('84 - Pagination - Middle Page', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.wait(800);
    cy.setTablePageSize(2);
    cy.submitTable();
    cy.wait(1000);
    cy.get('[data-testid="table-page-number"]').should('be.visible').contains('2').click({force: true});
    cy.wait(800);
    cy.screenshot(`${screenshotDir}/84-pagination-middle-page`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('85 - Pagination - Last Page', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.data('products');
    cy.wait(800);
    cy.setTablePageSize(2);
    cy.submitTable();
    cy.wait(1000);
    cy.get('[data-testid="table-page-number"]').should('be.visible').last().click({force: true});
    cy.wait(800);
    cy.screenshot(`${screenshotDir}/85-pagination-last-page`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('86 - Scratchpad - SELECT Query Result', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT id, username, email FROM test_schema.users ORDER BY id;');
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/86-scratchpad-select-query-result`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('87 - Scratchpad - COUNT Query Result', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT COUNT(*) as total_users FROM test_schema.users;');
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/87-scratchpad-count-query-result`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('88 - Scratchpad - JOIN Query Result', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'SELECT u.username, COUNT(o.id) as order_count FROM test_schema.users u LEFT JOIN test_schema.orders o ON u.id = o.user_id GROUP BY u.username ORDER BY order_count DESC;');
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/88-scratchpad-join-query-result`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('89 - Scratchpad - UPDATE Statement Result', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'UPDATE test_schema.users SET username=username WHERE id=999;');
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/89-scratchpad-update-statement`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('90 - Scratchpad - DELETE Statement Result', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('scratchpad');
    cy.writeCode(0, 'DELETE FROM test_schema.users WHERE id=999;');
    cy.runCode(0);
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/90-scratchpad-delete-statement`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  // ============================================================================
  // SECTION: GRAPH DETAILS & RELATIONSHIPS (91-100)
  // ============================================================================

  it('91 - Graph - Simple Table No Relations', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('[data-testid="rf__node-test_casting"]').scrollIntoView();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/91-graph-isolated-table-node`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('92 - Graph - One to Many Relationship', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('[data-testid="rf__node-users"]').scrollIntoView();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/92-graph-one-to-many-relationship`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('93 - Graph - Many to One Relationship', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('[data-testid="rf__node-order_items"]').scrollIntoView();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/93-graph-many-to-one-relationship`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('94 - Graph - Multiple Foreign Keys', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('[data-testid="rf__node-orders"]').scrollIntoView();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/94-graph-multiple-foreign-keys`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('95 - Graph - Zoom In View', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('.react-flow__controls-zoomin').click();
    cy.wait(300);
    cy.get('.react-flow__controls-zoomin').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/95-graph-zoomed-in-view`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('96 - Graph - Zoom Out View', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('.react-flow__controls-zoomout').click();
    cy.wait(300);
    cy.get('.react-flow__controls-zoomout').click();
    cy.wait(500);
    cy.screenshot(`${screenshotDir}/96-graph-zoomed-out-view`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('97 - Graph - Fit View Control', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.goto('graph');
    cy.wait(1500);
    cy.get('.react-flow__controls-fitview').trigger('mouseover');
    cy.wait(200);
    cy.screenshot(`${screenshotDir}/97-graph-fit-view-control`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('98 - Explore - Primary Key Column', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.explore('users');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/98-explore-primary-key-column`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('99 - Explore - Foreign Key Columns', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.explore('orders');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/99-explore-foreign-key-columns`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  it('100 - Explore - Table with Indexes', () => {
    cy.login('Postgres', dbHost, dbUser, dbPassword, 'test_db');
    cy.selectSchema('test_schema');
    cy.explore('order_items');
    cy.wait(1000);
    cy.screenshot(`${screenshotDir}/100-explore-table-with-indexes`, {
      capture: 'viewport',
      overwrite: true
    });
  });

  after(() => {
    cy.log('Completed comprehensive Postgres screenshot generation with 100 tests');
  });
});
