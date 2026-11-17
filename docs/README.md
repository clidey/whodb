# WhoDB Community documentation

The public CE site is authored in this directory. `dory.json` defines its navigation and appearance; MDX pages keep their existing public routes. Dory **1.13.0** is pinned in `package.json` and the Docker build.

## Global styles

`dory.json` registers `custom.css` once through `styles`. It contains the shared screenshot theme rules, header branding, and inline-code layout. MDX pages do not import the stylesheet individually. This requires Dory 1.11.0 or newer.

## Work locally

From the repository root:

```bash
pnpm --dir docs install --frozen-lockfile
pnpm --dir docs check
pnpm --dir docs dev
```

Build and inspect the static site:

```bash
pnpm --dir docs build
pnpm --dir docs preview
```

The output is `docs/dist/`. Dory checks navigation, internal links, assets, and metadata before building. The build then verifies that every route has prerendered content, so a per-page rendering failure cannot silently produce a partial site. The build generates prerendered pages, search assets, a sitemap, social preview images, and machine-readable documentation. Validate the built output as well as the MDX source.

The repository's existing docs Docker build remains available. It uses the same pinned Dory version and runs the content check before building. Production deployment is a separate step.

## GitHub Pages

The **Documentation (CE)** workflow builds the site on pull requests, pushes to `main` that change the docs, and manual runs. Pull requests produce a downloadable preview artifact. Successful builds from `main` deploy to `https://clidey.github.io/whodb/` through the `github-pages` environment.

Before the first deployment, select **Settings → Pages → Build and deployment → Source → GitHub Actions** in the repository. If the `github-pages` environment has protection rules, allow deployments from `main`.

To build the same artifact locally:

```bash
pnpm --dir docs install --frozen-lockfile
pnpm --dir docs build:pages
```

The output is `dist/docs-pages/` at the repository root. The script stages the documentation in a temporary directory, sets the Pages origin and `/whodb` prefix there, and verifies every prerendered route. It leaves `docs/dory.json` and the normal `docs/dist/` build untouched, so the Docker deployment can continue serving `docs.whodb.com` at its root path.

The workflow checks out only the public repository, uses the pinned Dory package, and gives Pages deployment permissions only to the deployment job. That job runs the official Pages actions without checking out or executing documentation code.

## Refresh screenshots

The screenshot suite exercises the current CE frontend and real PostgreSQL seed data from `dev/sample-data/postgres/data.sql`. It uses an isolated Compose project and CE server ports **8086** and **3106**, leaving ordinary development listeners alone.

```bash
cd frontend
pnpm exec playwright install chromium
pnpm screenshot
pnpm screenshot:copy
```

Set `WHODB_SCREENSHOT_BACKEND_PORT` and `WHODB_SCREENSHOT_FRONTEND_PORT` before capture if those ports are in use. The screenshot database uses port 5432; stop a conflicting development database before running the full managed runner.

If an isolated CE frontend and seeded PostgreSQL database are already running, capture directly:

```bash
cd frontend/e2e
BASE_URL=http://localhost:3106 \
  WHODB_SCREENSHOT_DIR="$PWD/../../docs/images" \
  NODE_ENV=test pnpm exec playwright test \
  --config=playwright.screenshot.config.mjs --project=screenshots-light --project=screenshots-dark \
  tests/postgres-screenshots.spec.mjs
```

Screenshots are captured in both light and dark themes as 1920 × 1080 PNGs under `images/light/` and `images/dark/`. These are the only screenshot copies; the repository README links directly to the light screenshots. The `custom.css` wrappers follow Dory’s theme class, including a reader’s manual theme selection. Keep existing filenames when replacing published images, and check the intended panel or menu is visible before accepting a capture. The suite also covers import preview, table creation, Settings, and the command palette.

Chat screenshots use fixed provider responses. They are interface illustrations, not live-model accuracy or performance evidence. Keep that distinction visible on AI documentation pages.

## Review coverage

For each changed workflow, compare the prose with current source behavior and the rendered CE UI. Check source-specific availability, active connection context, browser-local history, export scope, and confirmation behavior. Use `dory check` to catch broken local references, then inspect desktop and mobile layouts in light and dark themes.
