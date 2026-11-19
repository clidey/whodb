# Security Advisory Triage

- CE configured profiles are usable by anyone who can reach the instance, with the server-stored credentials. That is the documented product model (`docs/installation.mdx`, `docs/best-practices/access-control.mdx`), not a vulnerability. `WHODB_DISABLE_CREDENTIAL_FORM` only hides manual entry; it is not authentication.
- Profile access has several paths: `LoginWithSourceProfile`, saved-profile references resolved by ID in `core/src/auth/auth.go`, and `ExportSourceConnection(includeSecrets)`. A fix that closes one must check the others.
- Private (triage) advisories return 404 as HTML pages. Read them with `gh api repos/clidey/whodb/security-advisories/<GHSA-id>`; a 404 doesn't mean the advisory is invalid.
