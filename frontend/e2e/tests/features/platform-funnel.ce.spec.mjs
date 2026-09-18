import { test, expect } from '@playwright/test';
import { buildImportConnection } from '../../../src/utils/platform-funnel';
import { defaultFeatures } from '../../../src/config/feature-defaults';

// The funnel UI only renders when the platformFunnel feature flag is on.
test.skip(!defaultFeatures.platformFunnel, 'platformFunnel feature flag is disabled');

const profile = {
    Id: 'test-connection', SourceType: 'Postgres', Type: 'Postgres', Values: [],
    Hostname: 'db.example.test', Username: 'reader', Password: 'fixture-password', Database: 'reports',
    Advanced: [
        { Key: 'Port', Value: '5432' },
        { Key: 'Search Path', Value: 'public' },
        { Key: 'SSL Mode', Value: 'verify-full' },
        { Key: 'SSL Private Key', Value: 'fixture-private-key\nsecond-line' },
        { Key: 'Access Token', Value: 'fixture-token' },
        { Key: 'Passphrase', Value: 'fixture-passphrase' },
    ],
};

test('credential opt-out omits the password and blanks all advanced secrets', () => {
    const connection = buildImportConnection(profile, false);
    expect(connection).not.toHaveProperty('password');
    expect(connection.port).toBe('5432');
    expect(connection.advanced).toEqual([
        { Key: 'Search Path', Value: 'public' },
        { Key: 'SSL Mode', Value: 'verify-full' },
        { Key: 'SSL Private Key', Value: '' },
        { Key: 'Access Token', Value: '' },
        { Key: 'Passphrase', Value: '' },
    ]);
    expect(profile.Advanced[3].Value).toContain('fixture-private-key');
});

test('explicit credential consent includes passwords and advanced secrets', () => {
    const connection = buildImportConnection(profile, true);
    expect(connection.password).toBe(profile.Password);
    expect(connection.advanced).toEqual(profile.Advanced.filter(field => field.Key !== 'Port'));
});

test('sidebar import defaults to no secrets and resets consent when reopened', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('whodb.analytics.consent', 'denied'));
    await page.route('**/api/query', route => route.fulfill({ json: { data: {
        ExportSourceConnection: [{ Key: 'Database', Value: 'reports' }],
        SourceTypes: [], SourceProfiles: [], SourceSession: { id: profile.Id, sourceType: 'Postgres', database: 'reports' },
        SourceSessionMetadata: { sourceType: 'Postgres', queryLanguages: [], typeDefinitions: [], operators: [], aliasMap: [] },
        SettingsConfig: { CloudProvidersEnabled: false, AWSProviderEnabled: false, AzureProviderEnabled: false, GCPProviderEnabled: false, DisableCredentialForm: false, MaxPageSize: 10000 },
        Health: { Server: 'healthy', Database: 'healthy' },
    } } }));
    let submitted;
    await page.route('https://app.whodb.com/api/ce-import', route => {
        submitted = route.request().postDataJSON();
        return route.fulfill({ status: 503, body: 'Test staging failure' });
    });
    await page.goto('/settings');
    await page.getByTestId('sidebar-platform-funnel').click();
    await page.getByRole('button', { name: 'Import connection', exact: true }).click();
    const consent = page.getByTestId('platform-import-send-credentials');
    await expect(consent).not.toBeChecked();
    await page.getByText('Include passwords and other connection secrets', { exact: true }).click();
    await expect(consent).toBeChecked();
    await page.getByRole('button', { name: 'Cancel', exact: true }).click();
    await page.getByRole('button', { name: 'Import connection', exact: true }).click();
    await expect(consent).not.toBeChecked();
    await page.getByRole('button', { name: 'Import in Platform', exact: true }).click();
    await expect.poll(() => submitted).toBeDefined();
    expect(submitted.connections[0]).not.toHaveProperty('password');
});

for (const includeSecrets of [false, true]) {
    test(`reloaded browser session exports connection details (secrets: ${includeSecrets})`, async ({ page }) => {
        await page.addInitScript(() => localStorage.setItem('whodb.analytics.consent', 'denied'));
        const values = [
            { Key: 'Hostname', Value: profile.Hostname },
            { Key: 'Username', Value: profile.Username },
            { Key: 'Password', Value: profile.Password },
            { Key: 'Database', Value: profile.Database },
            ...profile.Advanced,
        ];
        let exportVariables;
        await page.route('**/api/query', route => {
            const request = route.request().postDataJSON();
            if (request.operationName === 'ExportSourceConnection') {
                exportVariables = request.variables;
                return route.fulfill({ json: { data: { ExportSourceConnection: values } } });
            }
            return route.fulfill({ json: { data: {
                SourceTypes: [], SourceProfiles: [],
                SourceSession: { id: profile.Id, sourceType: profile.Type, database: profile.Database },
                SourceSessionMetadata: { sourceType: profile.Type, queryLanguages: [], typeDefinitions: [], operators: [], aliasMap: [] },
                SettingsConfig: { CloudProvidersEnabled: false, AWSProviderEnabled: false, AzureProviderEnabled: false, GCPProviderEnabled: false, DisableCredentialForm: false, MaxPageSize: 10000 },
                Health: { Server: 'healthy', Database: 'healthy' },
            } } });
        });
        let submitted;
        await page.route('https://app.whodb.com/api/ce-import', route => {
            submitted = route.request().postDataJSON();
            return route.fulfill({ status: 503, body: 'Test staging failure' });
        });
        await page.goto('/settings');
        await page.reload();
        await page.getByTestId('sidebar-platform-funnel').click();
        await page.getByRole('button', { name: 'Import connection', exact: true }).click();
        if (includeSecrets) await page.getByTestId('platform-import-send-credentials').click();
        await page.getByRole('button', { name: 'Import in Platform', exact: true }).click();
        await expect.poll(() => submitted).toBeDefined();
        expect(submitted.connections).toEqual([buildImportConnection(profile, includeSecrets)]);
        expect(exportVariables).toEqual({ id: profile.Id, includeSecrets });
        const storage = await page.evaluate(() => JSON.stringify({ ...localStorage, ...sessionStorage }));
        expect(storage).not.toContain(profile.Password);
        expect(storage).not.toContain('fixture-private-key');
    });
}

test('Platform connectors use plain names and open the attributed source flow', async ({ page, context }) => {
    await page.addInitScript(() => localStorage.setItem('whodb.analytics.consent', 'denied'));
    await page.route('**/api/query', route => route.fulfill({ json: { data: {
        SourceTypes: [], SourceProfiles: [], SourceSession: null,
        SettingsConfig: { CloudProvidersEnabled: false, AWSProviderEnabled: false, AzureProviderEnabled: false, GCPProviderEnabled: false, DisableCredentialForm: false, MaxPageSize: 10000 },
        Health: { Server: 'healthy', Database: 'healthy' },
    } } }));
    await context.route('https://app.whodb.com/**', route => route.fulfill({ body: 'Hosted destination intercepted for test' }));
    await page.goto('/login');
    await expect(page.getByTestId('platform-funnel-link-login_panel')).toHaveCount(0);
    await page.getByTestId('database-type-select').click();
    await expect(page.getByRole('option', { name: /— WhoDB Platform/ })).toHaveCount(0);
    await page.getByRole('option', { name: 'Oracle', exact: true }).click();
    await expect(page.getByRole('dialog')).toContainText('Oracle is available on WhoDB Platform');
    const popupPromise = page.waitForEvent('popup');
    await page.getByRole('button', { name: 'Connect Oracle', exact: true }).click();
    const popup = await popupPromise;
    await popup.waitForLoadState();
    const url = new URL(popup.url());
    expect(url.pathname).toBe('/import');
    expect(url.searchParams.get('type')).toBe('Oracle');
    expect(url.searchParams.get('utm_campaign')).toBe('source_picker');
});
