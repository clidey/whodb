import { createRequire } from "node:module";
import { readFileSync, readdirSync, lstatSync } from "node:fs";
import { join, relative } from "node:path";

const { options, url, session, playwrightPackage } = JSON.parse(
  readFileSync(0, "utf8"),
);
const require = createRequire(playwrightPackage);
const { chromium } = require("@playwright/test");
const origin = new URL(options.Host).origin;

// App files stay on disk. Each AppView request reads the latest workspace, so
// a browser reload picks up edits without writing a hosted draft.
const localWorkspace = () => {
  if (!options.LocalDir) return null;
  const files = [];
  const walk = (dir) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (entry.name.startsWith(".")) continue;
      const path = join(dir, entry.name);
      if (lstatSync(path).isSymbolicLink()) throw new Error(`app file is a symlink: ${path}`);
      if (entry.isDirectory()) walk(path);
      else if (entry.isFile()) files.push({ path: relative(options.LocalDir, path).replaceAll("\\", "/"), content: readFileSync(path, "utf8") });
    }
  };
  for (const name of ["app.json", "pages", "shared"]) {
    const path = join(options.LocalDir, name);
    try {
      const entry = lstatSync(path);
      if (entry.isSymbolicLink()) throw new Error(`app file is a symlink: ${path}`);
      if (entry.isDirectory()) walk(path);
      else if (entry.isFile()) files.push({ path: name, content: readFileSync(path, "utf8") });
    } catch (error) {
      if (error.code !== "ENOENT") throw error;
    }
  }
  const app = JSON.parse(files.find((file) => file.path === "app.json")?.content ?? "{}");
  const pages = files.filter((file) => /^pages\/[^/]+\/tree\.json$/.test(file.path)).map((file) => file.path.split("/")[1]).sort();
  return { files, pages, defaultPage: app.defaultPage || pages[0] };
};

const viewFiles = (workspace, page) => workspace.files.filter((file) =>
  file.path === "app.json" || file.path.startsWith("shared/") || file.path.startsWith(`pages/${page}/`));

// A document is read-only when it starts with a query and defines no mutation or
// subscription anywhere, so a decoy query followed by a mutation selected through
// operationName is rejected. Block strings, strings, and comments are removed in
// one left-to-right pass so a quote inside a comment cannot hide later tokens.
const isReadOnlyDocument = (raw) => {
  const query = String(raw ?? "")
    .replace(/"""[\s\S]*?"""|"(?:[^"\\\n]|\\.)*"|#[^\n]*/g, (token) => (token[0] === "#" ? "" : '""'))
    .trimStart();
  return /^(query\b|\{)/.test(query) && !/\b(mutation|subscription)\b/.test(query);
};

const browser = await chromium.launch({ headless: !options.Interactive });
try {
  const context = await browser.newContext({
    viewport: { width: options.Width, height: options.Height },
    colorScheme: "light",
    isMobile: options.Mobile === true || options.Tablet === true,
    hasTouch: options.Mobile === true || options.Tablet === true,
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

  await context.route("**/*", async (route) => {
    const request = route.request();
    const parsed = new URL(request.url());
	let requestBody = null;
    if (parsed.origin === origin && parsed.pathname === "/api/auth/session") {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(session),
      });
    }
	if (options.KernelDir && parsed.origin === origin && request.method() === "GET") {
		const kernelFiles = { "/sandbox.html": "text/html", "/sandbox-kernel.js": "application/javascript", "/sandbox-kernel.css": "text/css" };
		if (kernelFiles[parsed.pathname]) {
			return route.fulfill({ status: 200, contentType: kernelFiles[parsed.pathname], body: readFileSync(join(options.KernelDir, parsed.pathname.slice(1))) });
		}
	}
    if (request.method() !== "GET" && request.method() !== "HEAD") {
      let query = "";
      try {
        requestBody = request.postDataJSON();
        query = requestBody?.query ?? "";
      } catch {
        // Non-GraphQL writes are blocked below.
      }
      const platformAPI = parsed.origin === origin && (parsed.pathname.startsWith("/api/") || parsed.pathname === "/graphql");
      if (!platformAPI || (!options.AllowWrites && !isReadOnlyDocument(query))) {
        return route.abort();
      }
    }
    if (parsed.origin === origin && (parsed.pathname.startsWith("/api/") || parsed.pathname === "/graphql")) {
      const headers = {
          ...request.headers(),
          authorization: `Bearer ${options.Token}`,
          "x-whodb-org-id": options.OrgID,
          "x-whodb-project-id": options.ProjectID,
      };
      if (options.LocalDir && requestBody && /\b(AppView|AppPageFiles)\s*\(/.test(requestBody.query ?? "")) {
        const response = await route.fetch({ headers });
        const body = await response.json();
        const workspace = localWorkspace();
        const page = requestBody.variables?.page || workspace.defaultPage;
        if (body.data?.AppView) {
          body.data.AppView.files = viewFiles(workspace, page);
          body.data.AppView.pages = workspace.pages;
          body.data.AppView.version = null;
        }
        if (body.data?.AppPageFiles) {
          body.data.AppPageFiles = workspace.files.filter((file) => file.path.startsWith(`pages/${page}/`));
        }
        return route.fulfill({ response, json: body });
      }
      return route.continue({ headers });
    }
    return route.continue();
  });

  const page = await context.newPage();
	await page.addInitScript((trackEarlyBridge) => {
		if (window.top !== window) return;
		window.__whodbAppTrace = [];
		window.__whodbPending = { handlers: {}, bridges: {}, lastEvent: performance.now() };
		window.__whodbTraceAck = null;
		window.addEventListener("message", (message) => {
			const event = message.data;
			if (event?.replyTo === "local-verify-trace") window.__whodbTraceAck = event.ok === true;
			if (!event || event.__ux_sandbox !== true || event.direction !== "sandbox-to-host") return;
			if (!["trace", "stateChange", "navigate", "error", "bridge"].includes(event.event)) return;
			const data = event.data ?? {};
			const resource = data.resource || data.args?.entity || data.args?.functionName || data.args?.app || "";
			const entry = {
				kind: event.event === "trace" ? data.kind : event.event,
				trace_id: data.traceId || "",
				handler: data.handler || "", node_id: data.nodeId || "", event_type: data.eventType || "",
				path: data.path || "", method: data.method || "", resource,
				call_id: data.callId || "", page: data.page || "", message: data.message || "",
				elapsed_ms: Math.round(performance.now()),
			};
			const pending = window.__whodbPending;
			pending.lastEvent = performance.now();
			if (entry.kind === "handler-start") pending.handlers[entry.trace_id] = true;
			if (entry.kind === "handler-end" || entry.kind === "handler-error") delete pending.handlers[entry.trace_id];
			if ((trackEarlyBridge && entry.kind === "bridge") || entry.kind === "bridge-start") pending.bridges[entry.call_id] = true;
			if (entry.kind === "bridge-end" || entry.kind === "bridge-error") delete pending.bridges[entry.call_id];
			if (window.__whodbAppTrace.length < 500) window.__whodbAppTrace.push(entry);
		});
	}, Boolean(options.LocalDir && options.KernelDir));
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
	if (options.Interactive) {
		let fingerprint = JSON.stringify(localWorkspace());
		let tracedFrame = null;
		const timer = setInterval(() => {
			try {
				const currentFrame = page.frames().find((item) => item !== page.mainFrame() && item.url() === "about:srcdoc");
				if (currentFrame && currentFrame !== tracedFrame) {
					tracedFrame = currentFrame;
					currentFrame.evaluate(() => window.postMessage({ __ux_sandbox: true, id: "local-preview-trace", direction: "host-to-sandbox", cmd: "setTracing", enabled: true }, "*")).catch(() => { tracedFrame = null; });
				}
				const next = JSON.stringify(localWorkspace());
				if (next !== fingerprint) {
					fingerprint = next;
					process.stderr.write("Local app files changed; reloading preview\n");
					page.reload({ waitUntil: "domcontentloaded" }).catch((error) => process.stderr.write(`${error.message}\n`));
				}
			} catch (error) { process.stderr.write(`${error.message}\n`); }
		}, 500);
		await new Promise((resolve) => page.on("close", resolve));
		clearInterval(timer);
	} else {
  // The app frame was /sandbox.html before the inlined kernel; it is a srcdoc frame since.
  let frame;
  const frameDeadline = Date.now() + 30000;
  while (!frame && Date.now() < frameDeadline) {
    frame = page.frames().find((item) => item !== page.mainFrame() && (item.url().endsWith("/sandbox.html") || item.url() === "about:srcdoc"));
    if (!frame) await page.waitForTimeout(100);
  }
  if (!frame) throw new Error("App sandbox did not render");
  const waitForContent = async () => {
    const deadline = Date.now() + 60000;
    while (Date.now() < deadline) {
      const text = await frame.locator("body").innerText();
      if (text.includes("App not available")) {
        throw new Error("App is not available in the selected environment");
      }
      if (await frame.locator("[data-node-id]").count() > 0) {
        await page.waitForTimeout(300);
        return frame.locator("body").innerText();
      }
      await page.waitForTimeout(250);
    }
    throw new Error("App content did not finish loading; screenshot was not saved");
  };
	const waitForIdle = async () => {
		const deadline = Date.now() + 60000;
		while (Date.now() < deadline) {
			const idle = await page.evaluate(() => {
				const pending = window.__whodbPending;
				return pending && Object.keys(pending.handlers).length === 0 && Object.keys(pending.bridges).length === 0 && performance.now() - pending.lastEvent >= 250;
			});
			if (idle) return;
			await page.waitForTimeout(100);
		}
		throw new Error("App actions did not become idle within 60 seconds");
	};
  await waitForContent();
	if (options.LocalDir) {
		await frame.evaluate(() => window.postMessage({ __ux_sandbox: true, id: "local-verify-trace", direction: "host-to-sandbox", cmd: "setTracing", enabled: true }, "*"));
		try {
			await page.waitForFunction(() => window.__whodbTraceAck !== null, null, { timeout: 2000 });
			if (!(await page.evaluate(() => window.__whodbTraceAck))) throw new Error("runtime rejected tracing");
		} catch {
			record(diagnostics.script_errors, "App runtime tracing is unavailable; build the sandbox and pass --kernel-dir, or use a server with tracing support");
		}
	}
	await waitForIdle();
  if (options.Tab) {
    await frame.getByText(options.Tab, { exact: true }).first().click({ timeout: 5000 });
    await page.waitForTimeout(300);
  }
  await waitForContent();
	await waitForIdle();
  for (const [index, action] of (options.Actions ?? []).entries()) {
    try {
      const locator = action.selector ? frame.locator(action.selector) : null;
      if (locator) {
        await locator.first().waitFor({ state: action.kind === "wait_for" ? "visible" : "attached", timeout: action.kind === "wait_for" ? 30000 : 10000 });
        const matches = await locator.count();
        if (matches !== 1) throw new Error(`selector ${JSON.stringify(action.selector)} matched ${matches} elements; scope it to a unique node or table row (for example, [data-node-id='save-button'])`);
      }
      switch (action.kind) {
        case "click":
          // Button nodes render as a wrapper around the interactive element.
          // Clicking the wrapper's center can miss a narrow button.
          if (await locator.getAttribute("data-node-type") === "Button" && await locator.locator("button").count() === 1) {
            await locator.locator("button").click({ timeout: 10000 });
          } else {
            await locator.click({ timeout: 10000 });
          }
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
        case "expect_text":
          if (!(await frame.locator("body").innerText()).includes(action.value)) throw new Error(`expected visible text ${JSON.stringify(action.value)}`);
          break;
        case "expect_not_text":
          if ((await frame.locator("body").innerText()).includes(action.value)) throw new Error(`unexpected visible text ${JSON.stringify(action.value)}`);
          break;
        default:
          throw new Error(`Unknown action kind ${JSON.stringify(action.kind)}`);
      }
	  await waitForIdle();
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
		  await waitForIdle();
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
		  trace: await page.evaluate(() => window.__whodbAppTrace ?? []),
    }),
  );
	}
} finally {
  await browser.close();
}
