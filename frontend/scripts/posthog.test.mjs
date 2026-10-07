import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

async function harness({ consent = 'granted', host = 'app.whodb.com', enabled = true, e2e = false } = {}) {
    const captured = [];
    const storage = new Map([['whodb.analytics.consent', consent]]);
    let initCount = 0;
    const client = {
        init: () => { initCount++; },
        register() {}, identify() {}, group() {},
        captureException: (error, props) => captured.push({ error, props }),
        opt_in_capturing() {}, opt_out_capturing() {}, reset() {},
        get_distinct_id: () => 'test-id',
    };
    const browserStorage = { getItem: key => storage.get(key) ?? null, setItem: (key, value) => storage.set(key, value), removeItem: key => storage.delete(key) };
    const context = vm.createContext({
        console, URL, setTimeout: () => 0, clearTimeout() {},
        window: { location: { hostname: host }, localStorage: browserStorage, sessionStorage: browserStorage, addEventListener() {} },
        navigator: {},
    });
    const modules = new Map();
    async function synthetic(id, exports) {
        if (!modules.has(id)) {
            const module = new vm.SyntheticModule(Object.keys(exports), function () {
                for (const [key, value] of Object.entries(exports)) this.setExport(key, value);
            }, { context, identifier: id });
            modules.set(id, module);
            await module.link(() => {});
            await module.evaluate();
        }
        return modules.get(id);
    }
    async function load(name) {
        if (modules.has(name)) return modules.get(name);
        const extension = name === 'posthog' ? 'tsx' : 'ts';
        const source = readFileSync(new URL(`../src/config/${name}.${extension}`, import.meta.url), 'utf8');
        const { outputText } = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } });
        const module = new vm.SourceTextModule(outputText, {
            context,
            identifier: name,
            initializeImportMeta(meta) { meta.env = { MODE: 'production', VITE_POSTHOG_ENABLED: String(enabled), VITE_E2E_TEST: String(e2e), VITE_POSTHOG_DEPLOYMENT: 'whodb-platform-prod' }; },
            importModuleDynamically: () => synthetic('posthog-js', { default: client }),
        });
        modules.set(name, module);
        await module.link(specifier => {
            if (specifier === './features') return synthetic(specifier, { featureFlags: { sampleDatabaseTour: false } });
            if (specifier === './edition') return synthetic(specifier, { getEdition: () => 'ee' });
            return load(specifier.replace('./', ''));
        });
        return module;
    }
    const module = await load('posthog');
    await module.evaluate();
    return { api: module.namespace, captured, context, get initCount() { return initCount; } };
}

test('hosted initialization and crash reporting work with the sample tour disabled', async () => {
    const h = await harness();
    assert.ok(await h.api.initPosthog());
    assert.equal(h.initCount, 1);
    const error = vm.runInContext('new TypeError("private SQL and credentials")', h.context);
    await h.api.captureException(error, { reference: 'ERR-1234ABCD', page: '/private/project', componentStack: 'secret', occurredAt: 'private' });
    assert.equal(h.captured.length, 1);
    assert.equal(h.captured[0].error.name, 'TypeError');
    assert.equal(h.captured[0].error.message, 'Unexpected frontend error');
    assert.equal(h.captured[0].error.stack, undefined);
    assert.equal(h.captured[0].props.reference, 'ERR-1234ABCD');
    const serialized = JSON.stringify(h.captured);
    assert.ok(!serialized.includes('private') && !serialized.includes('secret'));
});

for (const [name, options] of Object.entries({ 'denied consent': { consent: 'denied' }, 'disabled build': { enabled: false }, 'E2E': { e2e: true }, 'local host': { host: 'localhost' } })) {
    test(`${name} still prevents remote initialization and crash capture`, async () => {
        const h = await harness(options);
        assert.equal(await h.api.initPosthog(), null);
        await h.api.captureException(new Error('secret'));
        assert.equal(h.initCount, 0);
        assert.equal(h.captured.length, 0);
    });
}

test('the hosted allowlist continues to exclude UAT', async () => {
    const h = await harness({ host: 'uat.whodb.com' });
    h.api.setRemoteAnalyticsHostAllowlist(['app.whodb.com']);
    assert.equal(await h.api.initPosthog(), null);
    assert.equal(h.initCount, 0);
});
