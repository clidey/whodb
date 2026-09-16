/** Per-operation versioning-policy state embedded at generation time. */
export declare const embeddedManifest: Record<string, {
    kind: string;
    deprecated?: boolean;
    sunsetAt?: string;
    behaviorChanged?: boolean;
    note?: string;
}>;
/** SHA-256 of the platform-manifest.json this SDK was generated from. */
export declare const manifestHash = "3cc0e6ed8d6e928e57d0acaec664df395c36307c8425124270d4a4efa7bfb7a7";
/** Manifest protocol version this SDK understands. */
export declare const manifestProtocolVersion = "1";
