import type { Transport } from './transport.js';
import { HttpTransport } from './transport-http.js';
import { IpcTransport } from './transport-ipc.js';
import { resolveConfig, DEFAULT_HOST, type WhoDBConfig } from './config.js';
import { OntologyHandle } from './ontology.js';
import { DatasetHandle } from './dataset.js';
import { SourceHandle, listSources } from './source.js';
import { FilesHandle } from './files.js';
import * as ops from './generated/operations.js';
import type { OntologyObjectType, PlatformSource } from './generated/types.js';
import { manifestHash } from './generated/manifest.js';
import { ValidationError } from './errors.js';
import { SDK_VERSION } from './version.js';

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * WhoDB is the platform client. Configure with an API key (headless), a raw
 * token, or nothing at all — local development falls back to `whodb login`
 * credentials via the CLI helper.
 *
 * ```ts
 * const whodb = new WhoDB({ apiKey: process.env.WHODB_API_KEY, org: "acme", project: "growth" });
 * const user = await whodb.ontology("User").get("u_123");
 * ```
 */
export class WhoDB {
  /** SHA-256 of the platform manifest this SDK release was generated from. */
  static readonly manifestHash = manifestHash;
  /** This SDK package's version. */
  static readonly version = SDK_VERSION;

  private readonly transport: Transport;
  private readonly httpTransport: HttpTransport | null;
  private readonly filesHandle: FilesHandle | null;
  private readonly orgInput?: string;
  private readonly projectInput?: string;
  private workspacePromise: Promise<{ orgId: string; projectId: string }> | null = null;

  constructor(config: WhoDBConfig = {}, transportOverride?: Transport) {
    // Functions runtime autodetect: no explicit transport/credentials and the
    // IPC env vars are present → route through the in-container IPC server.
    if (
      !transportOverride &&
      !config.credentials && !config.apiKey && !config.token &&
      !process.env.WHODB_API_KEY && process.env.WHODB_IPC_TOKEN
    ) {
      transportOverride = new IpcTransport();
    }
    const resolved = resolveConfig(config);
    this.orgInput = resolved.org;
    this.projectInput = resolved.project;
    this.usingApiKey = resolved.usingApiKey;
    if (transportOverride) {
      // Custom transports (IPC in the functions runtime, mocks in tests) skip
      // slug resolution: org/project inputs are taken as IDs verbatim.
      this.transport = transportOverride;
      this.httpTransport = null;
      this.filesHandle = null;
      this.skipWorkspaceResolution = true;
      return;
    }
    const host = resolved.host ?? DEFAULT_HOST;
    const httpTransport = new HttpTransport({ host, credentials: resolved.credentials });
    this.transport = httpTransport;
    this.httpTransport = httpTransport;
    this.filesHandle = new FilesHandle({
      host,
      credentials: resolved.credentials,
      orgId: async () => (await this.workspace()).orgId,
      projectId: async () => (await this.workspace()).projectId,
    });
    // Workspace headers must be present before the first scoped call; resolve
    // lazily but wire the transport update into the resolution.
    if (resolved.usingCliCredentials) {
      this.cliDefaults = async () => {
        const defaults = await resolved.credentials.defaults?.();
        return { orgId: defaults?.orgId, projectId: defaults?.projectId };
      };
    }
  }

  private cliDefaults: (() => Promise<{ orgId?: string; projectId?: string }>) | null = null;
  private skipWorkspaceResolution = false;
  private usingApiKey = false;

  /**
   * Resolves org/project slugs or IDs to IDs, once, and stamps the workspace
   * headers onto the transport. IDs pass through without a lookup.
   */
  private workspace(): Promise<{ orgId: string; projectId: string }> {
    this.workspacePromise ??= (async () => {
      let orgInput = this.orgInput;
      let projectInput = this.projectInput;
      if ((!orgInput || !projectInput) && this.cliDefaults) {
        const defaults = await this.cliDefaults();
        orgInput ??= defaults.orgId;
        projectInput ??= defaults.projectId;
      }
      if ((!orgInput || !projectInput) && this.usingApiKey) {
        // API keys carry their org, and the platform auto-resolves the
        // project when the key has exactly one grant — discover both.
        const mine = await ops.myWorkspace(this.transport, {});
        orgInput ??= mine.orgId ?? undefined;
        projectInput ??= mine.projectId ?? undefined;
        if (!projectInput) {
          throw new ValidationError(
            'this API key has access to multiple (or zero) projects — pass { project } to the WhoDB constructor or set WHODB_PROJECT',
          );
        }
      }
      if (!orgInput || !projectInput) {
        throw new ValidationError(
          'org and project are required — pass { org, project } to the WhoDB constructor, set WHODB_ORG/WHODB_PROJECT, or run: whodb use',
        );
      }
      if (this.skipWorkspaceResolution) {
        return { orgId: orgInput, projectId: projectInput };
      }
      let orgId = orgInput;
      if (!UUID_PATTERN.test(orgInput)) {
        const orgs = await ops.myOrganizations(this.transport, {});
        const match = orgs.find(o => o.slug === orgInput || o.name === orgInput);
        if (!match) throw new ValidationError(`organization "${orgInput}" not found for this account`);
        orgId = match.id;
      }
      // Project resolution needs the org header in place.
      this.httpTransport?.setWorkspace(orgId, undefined);
      let projectId = projectInput;
      if (!UUID_PATTERN.test(projectInput)) {
        const projects = await ops.projects(this.transport, { orgId });
        const match = projects.find(p => p.slug === projectInput || p.name === projectInput);
        if (!match) throw new ValidationError(`project "${projectInput}" not found in this organization`);
        projectId = match.id;
      }
      this.httpTransport?.setWorkspace(orgId, projectId);
      return { orgId, projectId };
    })();
    return this.workspacePromise;
  }

  private projectId = async (): Promise<string> => (await this.workspace()).projectId;

  /** Returns a handle for one ontology entity, addressed by apiName. */
  ontology(name: string): OntologyHandle {
    return new OntologyHandle(this.transport, this.projectId, name);
  }

  /** Lists all ontology entities in the project. */
  async ontologyEntities(): Promise<OntologyObjectType[]> {
    return ops.ontologyEntities(this.transport, { projectId: await this.projectId() });
  }

  /** Returns a handle for one dataset, addressed by name. */
  dataset(name: string): DatasetHandle {
    return new DatasetHandle(this.transport, this.projectId, name);
  }

  /** Returns a handle for one connected source, addressed by ID. */
  source(id: string): SourceHandle {
    return new SourceHandle(this.transport, this.projectId, id);
  }

  /** Lists the project's connected sources. */
  async sources(): Promise<PlatformSource[]> {
    return listSources(this.transport, await this.projectId());
  }

  /** File upload/download over the platform's bulk HTTP endpoints. */
  get files(): FilesHandle {
    if (!this.filesHandle) {
      throw new ValidationError('files are not available over a custom transport');
    }
    return this.filesHandle;
  }
}
