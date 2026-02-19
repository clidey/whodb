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

import AxeBuilder from "@axe-core/playwright";
import {
  test,
  expect,
  forEachDatabase,
  conditionalTest,
} from "../../support/test-fixture.mjs";
import { clearBrowserState } from "../../support/helpers/animation.mjs";
import { getSqlQuery, hasFeature } from "../../support/database-config.mjs";

async function dismissTelemetryModal(page) {
  for (let attempt = 0; attempt < 5; attempt++) {
    const btn = page.locator("button").filter({ hasText: "Disable Telemetry" });
    if ((await btn.count()) > 0) {
      await btn.click();
      return;
    }
    await page.waitForTimeout(300);
  }
}

async function runAxeScan(page, testInfo, label) {
  const results = await new AxeBuilder({ page }).analyze();
  await testInfo.attach(`${label}.axe.json`, {
    body: JSON.stringify(results, null, 2),
    contentType: "application/json",
  });

  const failing = results.violations.filter((v) => v.impact === "critical");
  expect(
    failing,
    `Critical axe violations detected on ${label}: ${failing
      .map((v) => v.id)
      .join(", ")}`
  ).toEqual([]);
}

test.describe("Accessibility (axe-core)", () => {
  test("login page has no critical violations", async ({ whodb, page }, testInfo) => {
    await clearBrowserState(page);
    await page.goto(whodb.url("/login"));

    await dismissTelemetryModal(page);

    await page.locator('[data-testid="database-type-select"]').waitFor({ timeout: 15_000 });
    await runAxeScan(page, testInfo, "login");
  });

  test("login advanced options have no critical violations", async ({ whodb, page }, testInfo) => {
    await clearBrowserState(page);
    await page.goto(whodb.url("/login"));

    await dismissTelemetryModal(page);

    await page.locator('[data-testid="database-type-select"]').waitFor({ timeout: 15_000 });
    await page.locator('[data-testid="database-type-select"]').click();
    await page.locator('[data-value="Postgres"]').click();

    await page.locator('[data-testid="advanced-button"]').click();
    await page.locator('[data-testid="Port-input"]').waitFor({ timeout: 15_000 });
    await runAxeScan(page, testInfo, "login-advanced");
  });

  test("tour overlay has no critical violations", async ({ whodb, page }, testInfo) => {
    await clearBrowserState(page);
    await page.goto(whodb.url("/login"));

    await dismissTelemetryModal(page);

    await page.locator('[data-testid="get-started-sample-db"]').waitFor({ timeout: 15_000 });
    await page.locator('[data-testid="get-started-sample-db"]').click();

    await page.waitForURL(/\/storage-unit/, { timeout: 15_000 });
    await page.locator('[data-testid="tour-tooltip"]').waitFor({ timeout: 15_000 });
    await runAxeScan(page, testInfo, "tour-tooltip");

    await page.locator('[data-testid="tour-skip-button"]').click();
    await expect(page.locator('[data-testid="tour-tooltip"]')).not.toBeAttached();
  });

  // Run against the target database (controlled by env DATABASE in CI/workflows).
  forEachDatabase("all", (db) => {
    test("storage-unit page has no critical violations", async ({ page }, testInfo) => {
      await page.locator('[data-testid="storage-unit-card"]').first().waitFor({
        timeout: 15_000,
      });
      await runAxeScan(page, testInfo, "storage-unit");
    });

    test("storage-unit list view has no critical violations", async ({ whodb, page }, testInfo) => {
      await page.evaluate(() => {
        const settings = JSON.parse(localStorage.getItem("persist:settings") || "{}");
        settings.storageUnitView = '"list"';
        localStorage.setItem("persist:settings", JSON.stringify(settings));
      });

      await page.goto(whodb.url("/storage-unit"));
      await page.locator("table").filter({ visible: true }).waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "storage-unit-list");
    });

    conditionalTest(
      db.type !== "Redis",
      "create storage-unit view has no critical violations",
      async ({ whodb, page }, testInfo) => {
        await page.goto(whodb.url("/storage-unit"));
        await page.locator('[data-testid="storage-unit-card-list"]').waitFor({ timeout: 15_000 });
        await page.locator('[data-testid="create-storage-unit-card"] button').first().click();
        await page.locator('[data-testid="create-field-card"]').first().waitFor({ timeout: 15_000 });
        await runAxeScan(page, testInfo, "create-storage-unit");
        await page.keyboard.press("Escape");
      }
    );

    test("explore page has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await page
        .locator("table")
        .filter({ visible: true })
        .locator("tbody tr")
        .first()
        .waitFor({ timeout: 30_000 });
      await runAxeScan(page, testInfo, "explore");
    });

    test("search intellisense dropdown has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await page.locator('[data-testid="table-search"]').click();
      await page.locator('[data-testid="table-search"]').fill("id ");
      await page.locator('[data-testid="search-intellisense-dropdown"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "search-intellisense");
      await page.keyboard.press("Escape");
    });

    test("context menu has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await whodb.openContextMenu(0);
      await page.locator('[role="menu"]').filter({ visible: true }).first().waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "context-menu");
      await page.keyboard.press("Escape");
    });

    conditionalTest(
      db.category === "sql",
      "entity search sheet has no critical violations",
      async ({ whodb, page }, testInfo) => {
        await whodb.data("orders");

        const columns = await whodb.getTableColumns();
        const fkIdx = columns.findIndex((c) => c.name === "user_id");
        expect(fkIdx).toBeGreaterThan(-1);

        await page
          .locator("table tbody tr")
          .first()
          .locator(`td[data-col-idx="${fkIdx}"]`)
          .click({ button: "right", force: true });
        await page.locator('[role="menu"]').filter({ visible: true }).first().waitFor({ timeout: 15_000 });
        await page
          .locator('[role="menuitem"]')
          .filter({ hasText: "Search for Entity" })
          .click();

        await page.getByText("Search Around").waitFor({ timeout: 15_000 });
        await runAxeScan(page, testInfo, "entity-search-sheet");
        await page.keyboard.press("Escape");
      }
    );

    conditionalTest(hasFeature(db, "export"), "export dialog has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await page.locator('[data-testid="export-all-button"]').click();
      await page.locator('[data-testid="export-dialog"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "export-dialog");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="export-dialog"]')).not.toBeAttached();
    });

    conditionalTest(hasFeature(db, "mockData"), "mock data sheet has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await whodb.selectMockData();
      await page.locator('[data-testid="mock-data-sheet"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "mock-data-sheet");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="mock-data-sheet"]')).not.toBeAttached();
    });

    conditionalTest(hasFeature(db, "import"), "import dialog has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.openImport(db.testTable.name);
      await page.locator('[data-testid="import-dialog"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "import-dialog");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="import-dialog"]')).not.toBeAttached();
    });

    conditionalTest(hasFeature(db, "whereConditions"), "where conditions UI has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await page.locator('[data-testid="where-button"]').click();

      await Promise.race([
        page.locator('[data-testid="field-key"]').waitFor({ timeout: 10_000 }).catch(() => {}),
        page.locator('[data-testid*="sheet-field"]').first().waitFor({ timeout: 10_000 }).catch(() => {}),
      ]);

      await runAxeScan(page, testInfo, "where-conditions");
      await page.keyboard.press("Escape");
    });

    conditionalTest(hasFeature(db, "scratchpad"), "scratchpad page has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.goto("scratchpad");
      await page.locator('[data-testid="cell-0"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "scratchpad");
    });

    conditionalTest(
      hasFeature(db, "scratchpad") && db.category === "sql",
      "scratchpad error state has no critical violations",
      async ({ whodb, page }, testInfo) => {
        await whodb.goto("scratchpad");
        await page.locator('[data-testid="cell-0"]').waitFor({ timeout: 15_000 });

        const query = getSqlQuery(db, "invalidQuery");
        await whodb.writeCode(0, query);
        await whodb.runCode(0);

        await page.locator('[data-testid="cell-error"]').waitFor({ timeout: 15_000 });
        await runAxeScan(page, testInfo, "scratchpad-error");
      }
    );

    conditionalTest(hasFeature(db, "queryHistory"), "query history dialog has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.goto("scratchpad");
      await page.locator('[data-testid="cell-0"]').waitFor({ timeout: 15_000 });

      const query = getSqlQuery(db, "countUsers");
      await whodb.writeCode(0, query);
      await whodb.runCode(0);

      await whodb.openQueryHistory(0);
      await page.locator('[role="dialog"]').filter({ visible: true }).first().waitFor({
        timeout: 15_000,
      });

      await runAxeScan(page, testInfo, "query-history");
      await page.keyboard.press("Escape");
      await expect(page.locator('[role="dialog"]')).not.toBeAttached();
    });

    conditionalTest(hasFeature(db, "graph"), "graph page has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.goto("graph");
      await page
        .locator('[data-testid="graph-sidebar-content"]')
        .waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "graph");
    });

    conditionalTest(hasFeature(db, "chat"), "chat page has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.goto("chat");
      await page.locator('[data-testid="chat-input"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "chat");
    });

    test("settings page has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.goto("settings");
      await page.waitForURL(/\/settings/, { timeout: 15_000 });
      await page.locator("#font-size").waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "settings");
    });

    test("settings cloud providers section has no critical violations", async ({ whodb, page }, testInfo) => {
      const enableCloudProviders = async (route) => {
        let postData;
        try {
          postData = route.request().postDataJSON();
        } catch {
          return route.fallback();
        }

        const op = postData?.operationName;
        if (op === "SettingsConfig") {
          return route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
              data: {
                SettingsConfig: {
                  MetricsEnabled: "false",
                  CloudProvidersEnabled: true,
                  DisableCredentialForm: false,
                },
              },
            }),
          });
        }
        if (op === "GetCloudProviders") {
          return route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ data: { CloudProviders: [] } }),
          });
        }

        return route.fallback();
      };

      await page.route("**/api/query", enableCloudProviders);
      await whodb.goto("settings");
      await page.waitForURL(/\/settings/, { timeout: 15_000 });
      await page.locator('[data-testid="add-first-aws-provider"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "settings-cloud-providers");
      await page.unroute("**/api/query", enableCloudProviders);
    });

    test("contact-us page has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.goto("contact-us");
      await page.waitForURL(/\/contact-us/, { timeout: 15_000 });
      await page.locator('[data-testid="contact-email"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "contact-us");
    });

    test("command palette has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await whodb.typeCmdShortcut("k");
      await page.locator('[data-testid="command-palette"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "command-palette");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="command-palette"]')).not.toBeAttached();
    });

    test("keyboard shortcuts modal has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await page.keyboard.press("Shift+/");
      await page.locator('[data-testid="shortcuts-modal"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "shortcuts-modal");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="shortcuts-modal"]')).not.toBeAttached();
    });

    conditionalTest(db.category !== "keyvalue", "edit row dialog has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await whodb.openContextMenu(0);
      await page.locator('[data-testid="context-menu-edit-row"]').click();
      await page.locator('[data-testid="edit-row-dialog"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "edit-row-dialog");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="edit-row-dialog"]')).not.toBeAttached();
    });

    conditionalTest(db.category !== "keyvalue", "add row dialog has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await page.locator('[data-testid="add-row-button"]').click();
      await page.locator('[data-testid="submit-add-row-button"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "add-row-dialog");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="submit-add-row-button"]')).not.toBeAttached();
    });

    conditionalTest(hasFeature(db, "scratchpad"), "embedded scratchpad drawer has no critical violations", async ({ whodb, page }, testInfo) => {
      await whodb.data(db.testTable.name);
      await page.locator('[data-testid="embedded-scratchpad-button"]').click();
      await page.locator('[data-testid="scratchpad-drawer"]').waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "embedded-scratchpad-drawer");
      await page.keyboard.press("Escape");
      await expect(page.locator('[data-testid="scratchpad-drawer"]')).not.toBeAttached();
    });

    test("server down overlay has no critical violations", async ({ whodb, page }, testInfo) => {
      const handleHealth = async (route) => {
        let postData;
        try {
          postData = route.request().postDataJSON();
        } catch {
          return route.fallback();
        }

        if (postData?.operationName === "GetHealth") {
          return route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({ errors: [{ message: "forced health error" }] }),
          });
        }
        return route.fallback();
      };

      await page.route("**/api/query", handleHealth);

      await page.goto(whodb.url("/storage-unit"));
      await page.getByRole("heading", { name: "Server Unavailable" }).waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "server-down-overlay");
      await page.unroute("**/api/query", handleHealth);
    });

    test("database down overlay has no critical violations", async ({ whodb, page }, testInfo) => {
      const handleHealth = async (route) => {
        let postData;
        try {
          postData = route.request().postDataJSON();
        } catch {
          return route.fallback();
        }

        if (postData?.operationName === "GetHealth") {
          return route.fulfill({
            contentType: "application/json",
            body: JSON.stringify({
              data: { Health: { Server: "HEALTHY", Database: "ERROR" } },
            }),
          });
        }
        return route.fallback();
      };

      await page.route("**/api/query", handleHealth);

      await page.goto(whodb.url("/storage-unit"));
      await page.getByRole("heading", { name: "Database Connection Lost" }).waitFor({ timeout: 15_000 });
      await runAxeScan(page, testInfo, "database-down-overlay");
      await page.unroute("**/api/query", handleHealth);
    });
  });
});
