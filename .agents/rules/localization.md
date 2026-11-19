---
paths:
  - "frontend/src/**/*.ts"
  - "frontend/src/**/*.tsx"
  - "frontend/src/locales/**"
  - "dev/translate/**"
---

# Localization Rules

## Keys and placement
- Every user-facing string uses `t('key')` from `useTranslation('<component-path>')`: labels, placeholders, errors, toasts, tooltips, aria-labels, dialog text, table headers. Not localized: CSS classes, IDs, console logs, test fixtures.
- `t()` takes no fallback (`t('key', 'fallback')` does not compile). Keys are plain strings, so a missing key is not a type error; it renders as the key itself. `dev/check-translation-keys.py` (agent Stop hook) reports literal keys with no `en_US` entry.
- Lookup merges `common.yaml`, then the component YAML, then registered extension translations; the last one wins. Common keys are available everywhere.
- Use `common.yaml` when two or more components need the same key and value (actions, statuses, generic labels). Keep context-specific wording in the component file even if the English matches, because other languages may need different words. Never copy a common key and value into a component file.
- camelCase key names.

## Writing strings
- No sentence fragments. Interpolate with `{placeholder}`; never concatenate translated pieces with runtime values.
- To put JSX inside a sentence, pass it as a param (`t('details', { link: <a/> })`); `t()` then returns `ReactNode`.
- Plurals use CLDR suffixes (`_zero`, `_one`, `_two`, `_few`, `_many`, `_other`) selected by a numeric `count` param; the base key is the fallback.

## Languages
- Add or update `en_US` only. Don't edit other languages or run the `dev/translate` tooling unless the user asks. When asked: `dev/translate/run.sh [locale ...]` detects drift and machine-translates.
- Only `en_US` ships in the bundle; `frontend/plugins/vite-plugin-locale-packs.mjs` splits the other languages into packs that load on demand.
- Change the UI language only with `changeLanguage()` from `frontend/src/utils/change-language.ts`, which loads the pack before switching. `pnpm lint` rejects direct `SettingsActions.setLanguage` calls.
