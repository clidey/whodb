import { createRequire } from "node:module";
import { readFileSync } from "node:fs";

const { options, url, session, playwrightPackage } = JSON.parse(
  readFileSync(0, "utf8"),
);
const require = createRequire(playwrightPackage);
const { chromium } = require("@playwright/test");
const origin = new URL(options.Host).origin;

const browser = await chromium.launch({ headless: true });
try {
  const context = await browser.newContext({
    viewport: { width: options.Width, height: options.Height },
    colorScheme: "light",
  });
  await context.addInitScript(
    ({ orgId, projectId }) => {
      if (window.top !== window) return;
      localStorage.setItem(
        "persist:platform",
        JSON.stringify({
          currentOrgId: JSON.stringify(orgId),
          currentProjectId: JSON.stringify(projectId),
          selectedAIProviderId: "null",
          selectedAIModel: "null",
          _persist: JSON.stringify({ version: -1, rehydrated: true }),
        }),
      );
    },
    { orgId: options.OrgID, projectId: options.ProjectID },
  );

  await context.route("**/*", (route) => {
    const request = route.request();
    const parsed = new URL(request.url());
    if (parsed.origin === origin && parsed.pathname === "/api/auth/session") {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(session),
      });
    }
    if (request.method() !== "GET" && request.method() !== "HEAD") {
      let query = "";
      try {
        query = request.postDataJSON()?.query?.trimStart() ?? "";
      } catch {
        // Non-GraphQL writes are blocked below.
      }
      if (
        parsed.origin !== origin ||
        !["/api/query", "/graphql"].includes(parsed.pathname) ||
        !/^(query\b|\{)/.test(query)
      ) {
        return route.abort();
      }
    }
    if (parsed.origin === origin && parsed.pathname.startsWith("/api/")) {
      return route.continue({
        headers: {
          ...request.headers(),
          authorization: `Bearer ${options.Token}`,
          "x-whodb-org-id": options.OrgID,
          "x-whodb-project-id": options.ProjectID,
        },
      });
    }
    return route.continue();
  });

  const page = await context.newPage();
  const diagnostics = {
    console_errors: [],
    page_errors: [],
    failed_requests: [],
    http_error_responses: [],
    action_errors: [],
    script_errors: [],
  };
  const record = (items, value) => {
    if (items.length < 30) items.push(String(value).slice(0, 500));
  };
  const cleanURL = (raw) => {
    try {
      const parsed = new URL(raw);
      return `${parsed.origin}${parsed.pathname}`;
    } catch {
      return "unknown URL";
    }
  };
  page.on("console", (message) => {
    if (message.type() === "error") {
      record(diagnostics.console_errors, message.text());
    }
  });
  page.on("pageerror", (error) => record(diagnostics.page_errors, error.message));
  page.on("requestfailed", (request) => {
    record(
      diagnostics.failed_requests,
      `${request.method()} ${cleanURL(request.url())}: ${request.failure()?.errorText ?? "failed"}`,
    );
  });
  page.on("response", (response) => {
    if (response.status() >= 400) {
      record(
        diagnostics.http_error_responses,
        `${response.status()} ${cleanURL(response.url())}`,
      );
    }
  });
  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 30000 });
  await page.waitForTimeout(15000);
  const frame = page.frames().find((item) => item.url().endsWith("/sandbox.html"));
  if (!frame) throw new Error("App sandbox did not render");
  const waitForContent = async () => {
    const deadline = Date.now() + 60000;
    while (Date.now() < deadline) {
      const text = await frame.locator("body").innerText();
      if (text.includes("App not available")) {
        throw new Error("App is not available in the selected environment");
      }
      if (text.trim().length >= 80 && !/\bloading\b/i.test(text)) {
        await page.waitForTimeout(1500);
        const settled = await frame.locator("body").innerText();
        if (!/\bloading\b/i.test(settled)) return settled;
      }
      await page.waitForTimeout(1000);
    }
    throw new Error("App content did not finish loading; screenshot was not saved");
  };
  await waitForContent();
  if (options.Tab) {
    await frame.getByText(options.Tab, { exact: true }).first().click({ timeout: 5000 });
    await page.waitForTimeout(8000);
  }
  await waitForContent();
  for (const [index, action] of (options.Actions ?? []).entries()) {
    try {
      const locator = action.selector ? frame.locator(action.selector) : null;
      switch (action.kind) {
        case "click":
          await locator.click({ timeout: 10000 });
          break;
        case "fill":
          await locator.fill(action.value, { timeout: 10000 });
          break;
        case "press":
          await locator.press(action.value, { timeout: 10000 });
          break;
        case "wait_for":
          await locator.waitFor({ state: "visible", timeout: 30000 });
          break;
        case "wait":
          await page.waitForTimeout(action.milliseconds);
          break;
        default:
          throw new Error(`Unknown action kind ${JSON.stringify(action.kind)}`);
      }
      await waitForContent();
    } catch (error) {
      record(diagnostics.action_errors, `Action ${index + 1} (${action.kind}): ${error.message}`);
      break;
    }
  }
  let scriptResultJSON = "";
  if (options.Script) {
    try {
      const result = await frame.evaluate(options.Script);
      if (result !== undefined) scriptResultJSON = JSON.stringify(result).slice(0, 8000);
      await waitForContent();
    } catch (error) {
      record(diagnostics.script_errors, error.message);
    }
  }
  let body;
  try {
    body = await waitForContent();
  } catch (error) {
    record(diagnostics.action_errors, error.message);
    body = await frame.locator("body").innerText();
  }
  const image = await page.screenshot({
    type: "webp",
    quality: 88,
    animations: "disabled",
  });
  process.stdout.write(
    JSON.stringify({
      imageBase64: image.toString("base64"),
      text: body.slice(0, 12000),
      diagnostics,
      scriptResultJSON,
    }),
  );
} finally {
  await browser.close();
}
