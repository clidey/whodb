import type { Transport } from './transport.js';
import { type WhoDBConfig } from './config.js';
import { OntologyHandle } from './ontology.js';
import { DatasetHandle } from './dataset.js';
import { SourceHandle } from './source.js';
import { FilesHandle } from './files.js';
import type { OntologyObjectType, PlatformSource } from './generated/types.js';
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
export declare class WhoDB {
    /** SHA-256 of the platform manifest this SDK release was generated from. */
    static readonly manifestHash = "4b84678f5fe7d777fbd4c4b2a93dff64f4c8951978ce68006ddc3a0e25c6942f";
    /** This SDK package's version. */
    static readonly version = "0.0.0";
    private readonly transport;
    private readonly httpTransport;
    private readonly filesHandle;
    private readonly orgInput?;
    private readonly projectInput?;
    private workspacePromise;
    constructor(config?: WhoDBConfig, transportOverride?: Transport);
    private cliDefaults;
    private skipWorkspaceResolution;
    private usingApiKey;
    /**
     * Resolves org/project slugs or IDs to IDs, once, and stamps the workspace
     * headers onto the transport. IDs pass through without a lookup.
     */
    private workspace;
    private projectId;
    /** Returns a handle for one ontology entity, addressed by apiName. */
    ontology(name: string): OntologyHandle;
    /** Lists all ontology entities in the project. */
    ontologyEntities(): Promise<OntologyObjectType[]>;
    /** Returns a handle for one dataset, addressed by name. */
    dataset(name: string): DatasetHandle;
    /** Returns a handle for one connected source, addressed by ID. */
    source(id: string): SourceHandle;
    /** Lists the project's connected sources. */
    sources(): Promise<PlatformSource[]>;
    /** File upload/download over the platform's bulk HTTP endpoints. */
    get files(): FilesHandle;
}
