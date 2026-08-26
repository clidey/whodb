# WhoDB Platform SDKs

Customer-facing SDKs for the WhoDB hosted platform: the ontology, datasets,
and sources as in-code function APIs.

| Package | Registry | Path |
|---|---|---|
| `@clidey/whodb-sdk` | npm | `packages/typescript/` |
| `whodb-sdk` | PyPI | `packages/python/` |

Design: `SDK_DESIGN.md` in the EE repository. Architecture in one line:
a **generated wire core** (rendered from the committed spec snapshot in
`spec/`) under a **handwritten facade** per language, kept behaviorally
identical across languages by shared conformance fixtures.

## Layout

```
spec/                    # committed sync boundary with the platform backend
  platform-schema.graphql   # merged SDL (generated — ee/dev/export-sdk-spec.sh)
  platform-manifest.json    # curated public operation list (generated)
  surface.yaml              # hand-curated operation → facade mapping
  fixtures/                 # cross-language behavior fixtures
tools/
  generate-core.mjs      # spec → per-language wire cores (--check = drift gate)
  render/                # per-language renderers
  conformance-runner.mjs # runs fixtures against a language (--lang ts|python)
  sync-versions.mjs      # lockstep version stamping (--set X.Y.Z / --check)
  smoke.mjs              # pre-release staging round-trip (real API key)
packages/                # the SDKs
```

## Workflows

Backend API changed? (schema or platform manifest)

```bash
bash ee/dev/export-sdk-spec.sh        # re-export the spec snapshot
node sdk/tools/generate-core.mjs      # regenerate wire cores
# commit both — EE and CE CI both gate on drift
```

Verify everything:

```bash
cd sdk
pnpm install
pnpm run check                        # generate --check + version lockstep
pnpm -r --if-present run build
pnpm -r test
node tools/conformance-runner.mjs --lang ts
node tools/conformance-runner.mjs --lang python
```

Releases ride `release-ce.yml` (the `deploy-sdk` toggle) and share the repo
release version — see `_deploy-sdk-npm.yml` and the inlined
`deploy-sdk-pypi` job in `release-ce.yml` (inlined because PyPI attestations
don't work from reusable workflows).

Adding a language: `docs/porting.md`.
