/** Per-operation versioning-policy state embedded at generation time. */
export declare const embeddedManifest: Record<string, {
    kind: string;
    deprecated?: boolean;
    sunsetAt?: string;
    behaviorChanged?: boolean;
    note?: string;
}>;
/** SHA-256 of the platform-manifest.json this SDK was generated from. */
export declare const manifestHash = "4b84678f5fe7d777fbd4c4b2a93dff64f4c8951978ce68006ddc3a0e25c6942f";
/** Manifest protocol version this SDK understands. */
export declare const manifestProtocolVersion = "1";
