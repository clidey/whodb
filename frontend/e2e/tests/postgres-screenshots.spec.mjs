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

import { test, expect } from "../support/test-fixture.mjs";

test.describe("Postgres Screenshot Generation", () => {
  const dbHost = "localhost";
  const dbUser = "user";
  const dbPassword = 'jio53$*(@nfe)';
  const screenshotDir = "postgres";

  function ssPath(name) {
    return `e2e/screenshots/${screenshotDir}/${name}.png`;
  }

  // Tests that need clean login page (01-04)
  test.describe("Login Flow Screenshots", () => {
    test.beforeEach(async ({ whodb }) => {
      await whodb.mockVersion("v1.1.1");
    });

    test("01 - Login Page", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("01-login-page") });
    });

    test("02 - Login Form - Database Type Selection", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.locator('[data-testid="database-type-select"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("02-login-database-type-dropdown") });
      await page.locator('[data-value="Postgres"]').click();
    });

    test("03 - Login Form - Filled", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.locator('[data-testid="database-type-select"]').click();
      await page.locator('[data-value="Postgres"]').click();
      await page.locator('[data-testid="hostname"]').clear();
      await page.locator('[data-testid="hostname"]').fill(dbHost);
      await page.locator('[data-testid="username"]').clear();
      await page.locator('[data-testid="username"]').fill(dbUser);
      await page.locator('[data-testid="password"]').clear();
      await page.locator('[data-testid="password"]').fill(dbPassword);
      await page.locator('[data-testid="database"]').clear();
      await page.locator('[data-testid="database"]').fill("test_db");
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("03-login-form-filled") });
    });

    test("04 - Login Form - Advanced Options", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.locator('[data-testid="database-type-select"]').click();
      await page.locator('[data-value="Postgres"]').click();
      await page.locator('[data-testid="hostname"]').clear();
      await page.locator('[data-testid="hostname"]').fill(dbHost);
      await page.locator('[data-testid="username"]').clear();
      await page.locator('[data-testid="username"]').fill(dbUser);
      await page.locator('[data-testid="password"]').clear();
      await page.locator('[data-testid="password"]').fill(dbPassword);
      await page.locator('[data-testid="database"]').clear();
      await page.locator('[data-testid="database"]').fill("test_db");
      await page.locator('[data-testid="advanced-button"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("04-login-advanced-options") });
    });
  });

  // All other tests that need logged-in state with schema selected (05+)
  test.describe("Main Application Screenshots", () => {
    test.beforeEach(async ({ whodb, page }) => {
      await whodb.mockVersion("v1.1.1");
      await whodb.login("Postgres", dbHost, dbUser, dbPassword, "test_db");
      await whodb.selectSchema("test_schema");
      await page.goto(whodb.url("/storage-unit"));
    });

    test("05 - Storage Unit List Page", async ({ whodb, page }) => {
      await page.goto(whodb.url("/storage-unit"));
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("05-storage-unit-list"), fullPage: true });
    });

    test("06 - Storage Unit List - With Sidebar", async ({ whodb, page }) => {
      await page.goto(whodb.url("/storage-unit"));
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("06-storage-unit-list-with-sidebar") });
    });

    test("07 - Table Explore View - Users Table", async ({ whodb, page }) => {
      await whodb.explore("users");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("07-explore-users-table"), fullPage: true });
    });

    test("08 - Table Explore View - Table Metadata", async ({ whodb, page }) => {
      await whodb.explore("users");
      await page.waitForTimeout(1000);
      await expect(page.locator('[data-testid="explore-fields"]')).toBeVisible();
      await page.screenshot({ path: ssPath("08-explore-table-metadata") });
    });

    test("09 - Data View - Users Table", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("09-data-view-users-table"), fullPage: true });
    });

    test("10 - Data View - Table with Data", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("10-data-view-table-content") });
    });

    test("11 - Data View - Add Row Dialog", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="add-row-button"]').click();
      await expect(page.locator('[role="dialog"]')).toBeVisible();
      await page.keyboard.press("Escape");
      await page.screenshot({ path: ssPath("11-data-view-add-row-dialog") });
    });

    test("12 - Data View - Add Row Dialog Filled", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="add-row-button"]').click();
      await page.locator('[data-testid="add-row-field-id"] input').clear();
      await page.locator('[data-testid="add-row-field-id"] input').fill("5");
      await page.locator('[data-testid="add-row-field-username"] input').clear();
      await page.locator('[data-testid="add-row-field-username"] input').fill("screenshot_user");
      await page.locator('[data-testid="add-row-field-email"] input').clear();
      await page.locator('[data-testid="add-row-field-email"] input').fill("screenshot@example.com");
      await page.locator('[data-testid="add-row-field-password"] input').clear();
      await page.locator('[data-testid="add-row-field-password"] input').fill("testpass123");
      await page.locator('[data-testid="add-row-field-created_at"] input').clear();
      await page.locator('[data-testid="add-row-field-created_at"] input').fill("2025-01-15");
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("12-data-view-add-row-filled") });
      await page.keyboard.press("Escape");
    });

    test("13 - Data View - Context Menu", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator("table tbody tr").first().click({ button: "right", force: true });
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("13-data-view-context-menu") });
      await page.mouse.click(0, 0);
    });

    test("14 - Data View - Edit Row Dialog", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator("table tbody tr").first().click({ button: "right", force: true });
      await page.locator('[data-testid="context-menu-edit-row"]').click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("14-data-view-edit-row-dialog") });
      await page.keyboard.press("Escape");
    });

    test("15 - Data View - Page Size Dropdown", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="table-page-size"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("15-data-view-page-size-dropdown") });
      await page.mouse.click(0, 0);
    });

    test("16 - Data View - Where Conditions Popover", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="where-button"]').click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("16-data-view-where-conditions-popover") });
      await page.keyboard.press("Escape");
    });

    test("17 - Data View - Where Conditions Field Dropdown", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="where-button"]').click();
      await page.waitForTimeout(300);
      const fieldKey = page.locator('[data-testid="field-key"]');
      if ((await fieldKey.count()) > 0) {
        await fieldKey.first().click();
        await page.waitForTimeout(300);
        await page.screenshot({ path: ssPath("17-data-view-where-field-dropdown") });
      }
      await page.keyboard.press("Escape");
    });

    test("18 - Data View - Where Conditions with Badge", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", "=", "1"]]);
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("18-data-view-where-conditions-badge") });
      await whodb.clearWhereConditions();
    });

    test("19 - Data View - Search Functionality", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.searchTable("john");
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("19-data-view-search-highlight") });
    });

    test("20 - Data View - Export Dialog", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.getByRole("button", { name: "Export All" }).click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("20-data-view-export-dialog") });
      await page.keyboard.press("Escape");
    });

    test("21 - Data View - Export Format Dropdown", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.getByRole("button", { name: "Export All" }).click();
      await page.waitForTimeout(300);
      await page.locator('[data-testid="export-format-select"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("21-data-view-export-format-dropdown") });
      await page.keyboard.press("Escape");
    });

    test("22 - Data View - Mock Data Dialog", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.selectMockData();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("22-data-view-mock-data-dialog") });
      await page.keyboard.press("Escape");
    });

    test("23 - Data View - Embedded Scratchpad Drawer", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="embedded-scratchpad-button"]').click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("23-data-view-embedded-scratchpad") });
      await page.keyboard.press("Escape");
    });

    test("24 - Graph View - Schema Topology", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.screenshot({ path: ssPath("24-graph-view-schema-topology"), fullPage: true });
    });

    test("25 - Graph View - With Layout Button", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.screenshot({ path: ssPath("25-graph-view-with-controls") });
    });

    test("26 - Graph View - Node with Details", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await expect(page.locator('[data-testid="rf__node-users"]')).toBeVisible();
      await page.screenshot({ path: ssPath("26-graph-view-node-details") });
    });

    test("27 - Scratchpad - Main View", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("27-scratchpad-main-view"), fullPage: true });
    });

    test("28 - Scratchpad - Code Editor", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await page.waitForTimeout(500);
      await whodb.writeCode(0, "SELECT * FROM test_schema.users ORDER BY id;");
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("28-scratchpad-code-editor") });
    });

    test("29 - Scratchpad - Query Results", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT * FROM test_schema.users ORDER BY id;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("29-scratchpad-query-results") });
    });

    test("30 - Scratchpad - Query Error", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT * FROM test_schema.nonexistent_table;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("30-scratchpad-query-error") });
    });

    test("31 - Scratchpad - Multiple Pages", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.addScratchpadPage();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("31-scratchpad-multiple-pages") });
    });

    test("32 - Scratchpad - Cell Options Menu", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await page.waitForTimeout(1500);
      const cell = page.locator('[role="tabpanel"][data-state="active"] [data-testid="cell-0"]');
      await expect(cell).toBeVisible();
      await cell.hover();
      await page.waitForTimeout(500);
      await cell.locator('[data-testid="icon-button"]').first().click({ force: true });
      await page.waitForTimeout(1000);
      await expect(page.locator('[role="menu"]')).toBeVisible();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("32-scratchpad-cell-options-menu") });
    });

    test("33 - Scratchpad - Query History Dialog", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT * FROM test_schema.users;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await whodb.openQueryHistory(0);
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("33-scratchpad-query-history") });
      await page.keyboard.press("Escape");
    });

    test("34 - Sidebar - Database Selector", async ({ whodb, page }) => {
      await page.goto(whodb.url("/storage-unit"));
      await page.waitForTimeout(1500);
      await expect(page.locator('[data-testid="sidebar-database"]')).toBeVisible();
      await page.waitForTimeout(300);
      await page.locator('[data-testid="sidebar-database"]').click({ force: true });
      await page.waitForTimeout(800);
      await expect(page.locator('[role="listbox"]')).toBeVisible({ timeout: 5000 });
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("34-sidebar-database-selector") });
      await page.keyboard.press("Escape");
      await page.waitForTimeout(300);
    });

    test("35 - Sidebar - Schema Selector", async ({ whodb, page }) => {
      await page.goto(whodb.url("/storage-unit"));
      await page.waitForTimeout(1500);
      await expect(page.locator('[data-testid="sidebar-schema"]')).toBeVisible();
      await page.waitForTimeout(300);
      await page.locator('[data-testid="sidebar-schema"]').click({ force: true });
      await page.waitForTimeout(800);
      await expect(page.locator('[role="listbox"]')).toBeVisible({ timeout: 5000 });
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("35-sidebar-schema-selector") });
      await page.keyboard.press("Escape");
      await page.waitForTimeout(300);
    });

    test("36 - Sidebar - Navigation Menu", async ({ whodb, page }) => {
      await page.goto(whodb.url("/storage-unit"));
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("36-sidebar-navigation-menu") });
    });

    test("37 - Table Header - Context Menu", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator("table thead tr.cursor-context-menu").first().click({ button: "right", force: true });
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("37-table-header-context-menu") });
      await page.mouse.click(0, 0);
    });

    test("38 - Data View - Sorted Column", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.sortBy(1);
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("38-data-view-sorted-column") });
    });

    test("39 - Data View - Multiple Where Conditions", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([
        ["id", ">", "0"],
        ["username", "!=", "admin"],
      ]);
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("39-data-view-multiple-conditions") });
      await whodb.clearWhereConditions();
    });

    test("40 - Scratchpad - Action Query Result", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "UPDATE test_schema.users SET username='temp' WHERE id=999;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("40-scratchpad-action-result") });
    });

    // ============================================================================
    // ADDITIONAL INTERACTIONS - ADD/DELETE OPERATIONS (41-50)
    // ============================================================================

    test("41 - Add Row - Submit Button", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="add-row-button"]').click();
      await page.locator('[data-testid="add-row-field-id"] input').clear();
      await page.locator('[data-testid="add-row-field-id"] input').fill("100");
      await page.locator('[data-testid="add-row-field-username"] input').clear();
      await page.locator('[data-testid="add-row-field-username"] input').fill("test_user");
      await page.locator('[data-testid="add-row-field-email"] input').clear();
      await page.locator('[data-testid="add-row-field-email"] input').fill("test@example.com");
      await page.locator('[data-testid="add-row-field-password"] input').clear();
      await page.locator('[data-testid="add-row-field-password"] input').fill("password");
      await page.locator('[data-testid="add-row-field-created_at"] input').clear();
      await page.locator('[data-testid="add-row-field-created_at"] input').fill("2025-01-01");
      await whodb.screenshotWithHighlight('[data-testid="submit-add-row-button"]', ssPath("41-add-row-submit-button"));
      await page.keyboard.press("Escape");
    });

    test("42 - Edit Row - Update Button Hover", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator("table tbody tr").first().click({ button: "right", force: true });
      await page.locator('[data-testid="context-menu-edit-row"]').click();
      await page.waitForTimeout(300);
      await page.locator('[data-testid="editable-field-2"]').clear();
      await page.locator('[data-testid="editable-field-2"]').fill("updated_name");
      await whodb.screenshotWithHighlight('[data-testid="update-button"]', ssPath("42-edit-row-update-hover"));
      await page.keyboard.press("Escape");
    });

    test("43 - Context Menu - Delete Row Option", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.locator("table tbody tr").first().click({ button: "right", force: true });
      await page.waitForTimeout(200);
      await whodb.screenshotWithHighlight('[data-testid="context-menu-delete-row"]', ssPath("43-context-menu-delete-option"));
      await page.mouse.click(0, 0);
    });

    test("44 - Table - Row Selection Single", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator("table tbody tr").first().click();
      await page.waitForTimeout(200);
      await page.screenshot({ path: ssPath("44-table-row-selection-single") });
    });

    test("45 - Table - Select Row from Context Menu", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.waitForTimeout(200);
      await page.locator("table tbody tr").first().click({ button: "right", force: true });
      await page.waitForTimeout(200);
      await whodb.screenshotWithHighlight('[data-slot="context-menu-item"]:has-text("Select Row")', ssPath("45-context-menu-select-row"));
      await page.mouse.click(0, 0);
    });

    test("46 - Export - Selected Rows Mode", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.waitForTimeout(1000);
      await page.locator("table tbody tr").first().click({ button: "right", force: true });
      await page.waitForTimeout(300);
      await page.getByText("Select Row").click({ force: true });
      await page.waitForTimeout(800);
      await page.locator("button").filter({ hasText: "Export" }).filter({ visible: true }).last().click({ force: true });
      await page.waitForTimeout(1000);
      await expect(page.locator('[role="dialog"]')).toBeVisible();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("46-export-selected-rows-dialog") });
      await page.keyboard.press("Escape");
    });

    test("47 - Mock Data - Overwrite Confirmation", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.waitForTimeout(800);
      await whodb.selectMockData();
      await page.waitForTimeout(500);
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").clear();
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").fill("10");
      await page.waitForTimeout(300);
      await page.locator('[data-testid="mock-data-handling-select"]').click({ force: true });
      await page.waitForTimeout(500);
      await page.locator('[role="option"]').filter({ hasText: "Overwrite Existing Data" }).click({ force: true });
      await page.waitForTimeout(300);
      await page.locator('[data-testid="mock-data-generate-button"]').click({ force: true });
      await page.waitForTimeout(1000);
      await expect(page.locator('[role="alertdialog"],[role="dialog"]')).toBeVisible({ timeout: 5000 });
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("47-mock-data-overwrite-confirm") });
      await page.keyboard.press("Escape");
      await page.waitForTimeout(200);
      await page.keyboard.press("Escape");
    });

    test("48 - Scratchpad - Multiple Cells with Results", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT * FROM test_schema.users ORDER BY id LIMIT 3;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await whodb.addCell(0);
      await whodb.writeCode(1, "SELECT COUNT(*) as total FROM test_schema.users;");
      await whodb.runCode(1);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("48-scratchpad-multiple-cells-results") });
    });

    test("49 - Scratchpad - Query History Clone to Editor", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT * FROM test_schema.users;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await whodb.writeCode(0, "SELECT * FROM test_schema.orders;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await whodb.openQueryHistory(0);
      await page.waitForTimeout(500);
      await whodb.screenshotWithHighlight('[data-testid="clone-to-editor-button"]', ssPath("49-scratchpad-history-clone-button"));
      await page.keyboard.press("Escape");
    });

    test("50 - Graph - Click Node Data Button", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await whodb.screenshotWithHighlight('[data-testid="rf__node-users"] [data-testid="data-button"]', ssPath("50-graph-node-data-button-hover"));
    });
  });

  // ============================================================================
  // SECTION: DETAILED DROPDOWNS & OPTIONS (51-75)
  // ============================================================================

  test.describe("Additional Login Page Screenshots", () => {
    test.beforeEach(async ({ whodb }) => {
      await whodb.mockVersion("v1.1.1");
    });

    test("51 - Login - Database Type - All Options Visible", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.locator('[data-testid="database-type-select"]').click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("51-login-database-types-all-options") });
      await page.keyboard.press("Escape");
    });

    test("52 - Login - Database Type - MySQL Selected", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.locator('[data-testid="database-type-select"]').click();
      await page.locator('[data-value="MySQL"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("52-login-mysql-selected") });
    });

    test("53 - Login - Database Type - MongoDB Selected", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.locator('[data-testid="database-type-select"]').click();
      await page.locator('[data-value="MongoDB"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("53-login-mongodb-selected") });
    });

    test("54 - Login - Database Type - Redis Selected", async ({ whodb, page }) => {
      await page.goto(whodb.url("/login"));
      await page.locator('[data-testid="database-type-select"]').click();
      await page.locator('[data-value="Redis"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("54-login-redis-selected") });
    });
  });

  test.describe("Continued Application Screenshots", () => {
    test.beforeEach(async ({ whodb, page }) => {
      await whodb.mockVersion("v1.1.1");
      await whodb.login("Postgres", dbHost, dbUser, dbPassword, "test_db");
      await whodb.selectSchema("test_schema");
      await page.goto(whodb.url("/storage-unit"));
    });

    test("55 - Page Size - Dropdown All Options", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.locator('[data-testid="table-page-size"]').click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("55-page-size-all-options") });
      await page.keyboard.press("Escape");
    });

    test("56 - Page Size - 10 Rows Selected", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.setTablePageSize(10);
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("56-page-size-10-selected") });
    });

    test("57 - Page Size - 25 Rows Selected", async ({ whodb, page }) => {
      await whodb.data("orders");
      await whodb.setTablePageSize(25);
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("57-page-size-25-selected") });
    });

    test("58 - Page Size - 50 Rows Selected", async ({ whodb, page }) => {
      await whodb.data("products");
      await whodb.setTablePageSize(50);
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("58-page-size-50-selected") });
    });

    test("59 - Where Operator - Equals Selected", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", "=", "1"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("59-where-operator-equals") });
      await whodb.clearWhereConditions();
    });

    test("60 - Where Operator - Greater Than", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", ">", "1"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("60-where-operator-greater-than") });
      await whodb.clearWhereConditions();
    });

    test("61 - Where Operator - Less Than", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", "<", "3"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("61-where-operator-less-than") });
      await whodb.clearWhereConditions();
    });

    test("62 - Where Operator - Not Equals", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["username", "!=", "admin_user"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("62-where-operator-not-equals") });
      await whodb.clearWhereConditions();
    });

    test("63 - Where Operator - Greater Than or Equal", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", ">=", "2"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("63-where-operator-gte") });
      await whodb.clearWhereConditions();
    });

    test("64 - Where Operator - Less Than or Equal", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", "<=", "2"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("64-where-operator-lte") });
      await whodb.clearWhereConditions();
    });

    test("65 - Export Format - CSV Option Highlighted", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.getByRole("button", { name: "Export All" }).click();
      await page.waitForTimeout(300);
      await page.locator('[data-testid="export-format-select"]').click();
      await page.waitForTimeout(300);
      await whodb.screenshotWithHighlight('[role="option"]:has-text("CSV")', ssPath("65-export-format-csv-option"));
      await page.keyboard.press("Escape");
    });

    test("66 - Export Format - Excel Option Highlighted", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.getByRole("button", { name: "Export All" }).click();
      await page.waitForTimeout(300);
      await page.locator('[data-testid="export-format-select"]').click();
      await page.waitForTimeout(300);
      await whodb.screenshotWithHighlight('[role="option"]:has-text("Excel")', ssPath("66-export-format-excel-option"));
      await page.keyboard.press("Escape");
    });

    test("67 - Export Delimiter - Comma Option", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.getByRole("button", { name: "Export All" }).click();
      await page.waitForTimeout(300);
      await page.locator('[data-testid="export-delimiter-select"]').click();
      await page.waitForTimeout(300);
      await whodb.screenshotWithHighlight('[role="option"]:has-text("Comma")', ssPath("67-export-delimiter-comma"));
      await page.keyboard.press("Escape");
    });

    test("68 - Export Delimiter - Semicolon Option", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.waitForTimeout(800);
      await page.getByRole("button", { name: "Export All" }).click({ force: true });
      await page.waitForTimeout(500);
      await page.locator('[data-testid="export-delimiter-select"]').click({ force: true });
      await page.waitForTimeout(500);
      await page.locator('[role="option"][data-value=";"]').click({ force: true });
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("68-export-delimiter-semicolon") });
      await page.keyboard.press("Escape");
      await page.waitForTimeout(200);
      await page.keyboard.press("Escape");
    });

    test("69 - Export Delimiter - Pipe Option", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.waitForTimeout(800);
      await page.getByRole("button", { name: "Export All" }).click({ force: true });
      await page.waitForTimeout(500);
      await page.locator('[data-testid="export-delimiter-select"]').click({ force: true });
      await page.waitForTimeout(500);
      await whodb.screenshotWithHighlight('[role="option"][data-value="|"]', ssPath("69-export-delimiter-pipe"));
      await page.keyboard.press("Escape");
      await page.waitForTimeout(200);
      await page.keyboard.press("Escape");
    });

    test("71 - Mock Data - Append Mode Selected", async ({ whodb, page }) => {
      await whodb.data("products");
      await whodb.selectMockData();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("71-mock-data-append-mode") });
      await page.keyboard.press("Escape");
    });

    test("72 - Mock Data - Overwrite Mode Options", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.waitForTimeout(800);
      await whodb.selectMockData();
      await page.waitForTimeout(500);
      await page.locator('[data-testid="mock-data-handling-select"]').click({ force: true });
      await page.waitForTimeout(500);
      await expect(page.locator('[role="listbox"]')).toBeVisible();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("72-mock-data-handling-options") });
      await page.keyboard.press("Escape");
      await page.waitForTimeout(200);
      await page.keyboard.press("Escape");
    });

    test("73 - Mock Data - Row Count Min Value", async ({ whodb, page }) => {
      await whodb.data("products");
      await whodb.selectMockData();
      await page.waitForTimeout(300);
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").clear();
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").fill("1");
      await page.waitForTimeout(200);
      await page.screenshot({ path: ssPath("73-mock-data-row-count-min") });
      await page.keyboard.press("Escape");
    });

    test("74 - Mock Data - Row Count Medium Value", async ({ whodb, page }) => {
      await whodb.data("products");
      await whodb.selectMockData();
      await page.waitForTimeout(300);
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").clear();
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").fill("100");
      await page.waitForTimeout(200);
      await page.screenshot({ path: ssPath("74-mock-data-row-count-medium") });
      await page.keyboard.press("Escape");
    });

    test("75 - Mock Data - Row Count Max Value", async ({ whodb, page }) => {
      await whodb.data("products");
      await whodb.selectMockData();
      await page.waitForTimeout(300);
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").clear();
      await page.locator('label').filter({ hasText: "Number of Rows" }).locator("..").locator("input").fill("300");
      await page.waitForTimeout(200);
      await page.screenshot({ path: ssPath("75-mock-data-row-count-max-clamped") });
      await page.keyboard.press("Escape");
    });

    // ============================================================================
    // SECTION: TABLE STATES & EDGE CASES (76-90)
    // ============================================================================

    test("76 - Table - Empty State No Data", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", "=", "999999"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("76-table-empty-state-no-results") });
      await whodb.clearWhereConditions();
    });

    test("77 - Table - Single Row Result", async ({ whodb, page }) => {
      await whodb.data("users");
      await whodb.whereTable([["id", "=", "1"]]);
      await whodb.submitTable();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("77-table-single-row-result") });
      await whodb.clearWhereConditions();
    });

    test("78 - Table - Many Columns Wide Table", async ({ whodb, page }) => {
      await whodb.data("orders");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("78-table-many-columns-wide") });
    });

    test("79 - Table - With Null Values", async ({ whodb, page }) => {
      await whodb.data("payments");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("79-table-with-null-values") });
    });

    test("80 - Table - With Long Text Content", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("80-table-long-text-content") });
    });

    test("81 - Table - With Dates and Timestamps", async ({ whodb, page }) => {
      await whodb.data("users");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("81-table-dates-timestamps") });
    });

    test("82 - Table - With Numeric Data Types", async ({ whodb, page }) => {
      await whodb.data("test_casting");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("82-table-numeric-types") });
    });

    test("83 - Pagination - First Page", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.waitForTimeout(800);
      await whodb.setTablePageSize(1);
      await whodb.submitTable();
      await page.waitForTimeout(1000);
      await expect(page.locator('[data-slot="pagination-link"]').first()).toBeVisible();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("83-pagination-first-page") });
    });

    test("84 - Pagination - Middle Page", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.waitForTimeout(800);
      await whodb.setTablePageSize(1);
      await whodb.submitTable();
      await page.waitForTimeout(1000);
      await page.locator('[data-slot="pagination-link"]').filter({ hasText: "2" }).click({ force: true });
      await page.waitForTimeout(800);
      await page.screenshot({ path: ssPath("84-pagination-middle-page") });
    });

    test("85 - Pagination - Last Page", async ({ whodb, page }) => {
      await whodb.data("products");
      await page.waitForTimeout(800);
      await whodb.setTablePageSize(1);
      await whodb.submitTable();
      await page.waitForTimeout(1000);
      await page.locator('[data-slot="pagination-link"]').last().click({ force: true });
      await page.waitForTimeout(800);
      await page.screenshot({ path: ssPath("85-pagination-last-page") });
    });

    test("86 - Scratchpad - SELECT Query Result", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT id, username, email FROM test_schema.users ORDER BY id;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("86-scratchpad-select-query-result") });
    });

    test("87 - Scratchpad - COUNT Query Result", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT COUNT(*) as total_users FROM test_schema.users;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("87-scratchpad-count-query-result") });
    });

    test("88 - Scratchpad - JOIN Query Result", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "SELECT u.username, COUNT(o.id) as order_count FROM test_schema.users u LEFT JOIN test_schema.orders o ON u.id = o.user_id GROUP BY u.username ORDER BY order_count DESC;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("88-scratchpad-join-query-result") });
    });

    test("89 - Scratchpad - UPDATE Statement Result", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "UPDATE test_schema.users SET username=username WHERE id=999;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("89-scratchpad-update-statement") });
    });

    test("90 - Scratchpad - DELETE Statement Result", async ({ whodb, page }) => {
      await whodb.goto("scratchpad");
      await whodb.writeCode(0, "DELETE FROM test_schema.users WHERE id=999;");
      await whodb.runCode(0);
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("90-scratchpad-delete-statement") });
    });

    // ============================================================================
    // SECTION: GRAPH DETAILS & RELATIONSHIPS (91-100)
    // ============================================================================

    test("91 - Graph - Simple Table No Relations", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.locator('[data-testid="rf__node-test_casting"]').scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("91-graph-isolated-table-node") });
    });

    test("92 - Graph - One to Many Relationship", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.locator('[data-testid="rf__node-users"]').scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("92-graph-one-to-many-relationship") });
    });

    test("93 - Graph - Many to One Relationship", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.locator('[data-testid="rf__node-order_items"]').scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("93-graph-many-to-one-relationship") });
    });

    test("94 - Graph - Multiple Foreign Keys", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.locator('[data-testid="rf__node-orders"]').scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("94-graph-multiple-foreign-keys") });
    });

    test("95 - Graph - Zoom In View", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.locator(".react-flow__controls-zoomin").click();
      await page.waitForTimeout(300);
      await page.locator(".react-flow__controls-zoomin").click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("95-graph-zoomed-in-view") });
    });

    test("96 - Graph - Zoom Out View", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await page.locator(".react-flow__controls-zoomout").click();
      await page.waitForTimeout(300);
      await page.locator(".react-flow__controls-zoomout").click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("96-graph-zoomed-out-view") });
    });

    test("97 - Graph - Fit View Control", async ({ whodb, page }) => {
      await whodb.goto("graph");
      await page.waitForTimeout(1500);
      await whodb.screenshotWithHighlight(".react-flow__controls-fitview", ssPath("97-graph-fit-view-control"));
    });

    test("98 - Explore - Primary Key Column", async ({ whodb, page }) => {
      await whodb.explore("users");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("98-explore-primary-key-column") });
    });

    test("99 - Explore - Foreign Key Columns", async ({ whodb, page }) => {
      await whodb.explore("orders");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("99-explore-foreign-key-columns") });
    });

    test("100 - Explore - Table with Indexes", async ({ whodb, page }) => {
      await whodb.explore("order_items");
      await page.waitForTimeout(1000);
      await page.screenshot({ path: ssPath("100-explore-table-with-indexes") });
    });

    // ============================================================================
    // SECTION: CHAT (AI ASSISTANT) FUNCTIONALITY (101-115)
    // ============================================================================

    test("101 - Chat - Initial Page with Model Selection", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("101-chat-initial-page"), fullPage: true });
    });

    test("102 - Chat - AI Provider Dropdown", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await page.goto(whodb.url("/chat"));
      await page.waitForTimeout(1500);
      await page.locator('[data-testid="ai-provider-select"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("102-chat-ai-provider-dropdown") });
      await page.keyboard.press("Escape");
    });

    test("103 - Chat - AI Model Dropdown", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await page.locator('[data-testid="ai-model-select"]').click();
      await page.waitForTimeout(300);
      await page.screenshot({ path: ssPath("103-chat-ai-model-dropdown") });
      await page.keyboard.press("Escape");
    });

    test("104 - Chat - Example Prompts", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("104-chat-example-prompts") });
    });

    test("105 - Chat - Simple Text Response", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "Hello! I can help you query and explore your database. What would you like to know?",
      }]);
      await whodb.sendChatMessage("Hello!");
      await whodb.waitForChatResponse();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("105-chat-simple-text-response"), fullPage: true });
    });

    test("106 - Chat - SQL Query with Results", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "Here are all the users in the database.",
      }, {
        type: "sql:get",
        text: "SELECT * FROM test_schema.users ORDER BY id",
        result: {
          Columns: [
            { Name: "id", Type: "integer", __typename: "Column" },
            { Name: "username", Type: "text", __typename: "Column" },
            { Name: "email", Type: "text", __typename: "Column" },
          ],
          Rows: [
            ["1", "john_doe", "john@example.com"],
            ["2", "jane_smith", "jane@example.com"],
            ["3", "admin_user", "admin@example.com"],
          ],
          __typename: "RowsResult",
        },
      }]);
      await whodb.sendChatMessage("Show me all users");
      await whodb.waitForChatResponse();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("106-chat-sql-query-results"), fullPage: true });
    });

    test("107 - Chat - SQL Query Code View", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "Here are all the users in the database.",
      }, {
        type: "sql:get",
        text: "SELECT * FROM test_schema.users ORDER BY id",
        result: {
          Columns: [
            { Name: "id", Type: "integer", __typename: "Column" },
            { Name: "username", Type: "text", __typename: "Column" },
          ],
          Rows: [["1", "john_doe"], ["2", "jane_smith"]],
          __typename: "RowsResult",
        },
      }]);
      await whodb.sendChatMessage("Show me all users");
      await whodb.waitForChatResponse();
      await page.locator('[data-testid="icon-button"]').first().click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("107-chat-sql-code-view") });
    });

    test("108 - Chat - Error Message", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "error",
        text: 'ERROR: relation "test_schema.nonexistent_table" does not exist (SQLSTATE 42P01)',
      }]);
      await whodb.sendChatMessage("Show me data from nonexistent_table");
      await whodb.waitForChatResponse();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("108-chat-error-message"), fullPage: true });
    });

    test("109 - Chat - Aggregation Query", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "Here is the user count by email domain.",
      }, {
        type: "sql:get",
        text: "SELECT SUBSTRING(email FROM POSITION('@' IN email) + 1) as domain, COUNT(*) as user_count FROM test_schema.users GROUP BY domain",
        result: {
          Columns: [
            { Name: "domain", Type: "text", __typename: "Column" },
            { Name: "user_count", Type: "bigint", __typename: "Column" },
          ],
          Rows: [["example.com", "3"]],
          __typename: "RowsResult",
        },
      }]);
      await whodb.sendChatMessage("Count users by email domain");
      await whodb.waitForChatResponse();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("109-chat-aggregation-query"), fullPage: true });
    });

    test("110 - Chat - Action Query Confirmation", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "I can help you delete that user. Would you like me to proceed?",
      }]);
      await whodb.sendChatMessage("Delete user with id 5");
      await whodb.waitForChatResponse();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("110-chat-action-confirmation"), fullPage: true });
    });

    test("111 - Chat - Action Query Executed", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "I can help you delete that user. Would you like me to proceed?",
      }]);
      await whodb.sendChatMessage("Delete user with id 5");
      await whodb.waitForChatResponse();
      await whodb.mockChatResponse([{
        type: "sql:delete",
        text: "DELETE FROM test_schema.users WHERE id = 5",
        result: { Columns: [], Rows: [], __typename: "RowsResult" },
      }]);
      await whodb.sendChatMessage("Yes, delete it");
      await whodb.waitForChatResponse();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("111-chat-action-executed"), fullPage: true });
    });

    test("112 - Chat - Multiple Messages Conversation", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "The users table contains user account information including usernames, emails, and creation dates.",
      }]);
      await whodb.sendChatMessage("What is in the users table?");
      await whodb.waitForChatResponse();
      await whodb.mockChatResponse([{
        type: "sql:get",
        text: "SELECT COUNT(*) as total FROM test_schema.users",
        result: {
          Columns: [{ Name: "total", Type: "bigint", __typename: "Column" }],
          Rows: [["3"]],
          __typename: "RowsResult",
        },
      }]);
      await whodb.sendChatMessage("How many users are there?");
      await whodb.waitForChatResponse();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("112-chat-multiple-messages"), fullPage: true });
    });

    test("113 - Chat - Move to Scratchpad Dialog", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "Here are all the users.",
      }, {
        type: "sql:get",
        text: "SELECT * FROM test_schema.users",
        result: {
          Columns: [
            { Name: "id", Type: "integer", __typename: "Column" },
            { Name: "username", Type: "text", __typename: "Column" },
          ],
          Rows: [["1", "john_doe"]],
          __typename: "RowsResult",
        },
      }]);
      await whodb.sendChatMessage("Show me all users");
      await whodb.waitForChatResponse();
      // Hover over the table to reveal the actions button, click it, then click Move to Scratchpad
      const tablePreview = page.locator('.group\\/table-preview').last();
      await tablePreview.hover();
      await page.waitForTimeout(300);
      await tablePreview.locator('[data-testid="icon-button"]').click();
      await page.locator('[data-testid="move-to-scratchpad-option"]').click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: ssPath("113-chat-move-to-scratchpad-dialog") });
    });

    test("114 - Chat - New Chat Button", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.mockChatResponse([{
        type: "text",
        text: "Hello! How can I help you?",
      }]);
      await whodb.sendChatMessage("Hello");
      await whodb.waitForChatResponse();
      await whodb.screenshotWithHighlight('[data-testid="chat-new-chat"]', ssPath("114-chat-new-chat-button"));
    });

    test("115 - Chat - Delete Provider Button", async ({ whodb, page }) => {
      await whodb.setupChatMock();
      await whodb.gotoChat();
      await whodb.screenshotWithHighlight('[data-testid="chat-delete-provider"]', ssPath("115-chat-delete-provider-button"));
    });
  });
});
