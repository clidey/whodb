# Localization Standards

This document defines the localization requirements for the WhoDB frontend.

## Overview

WhoDB uses a YAML-based localization system with the `useTranslation` hook. All user-facing strings MUST be localized. Translations are loaded at build time via Vite's `import.meta.glob()`.

## How It Works

1. Translation files are YAML files in `frontend/src/locales/` (CE) and `ee/frontend/src/locales/` (EE)
2. Components load translations via `useTranslation('component-path')`
3. The `t()` function retrieves strings by key, with automatic pluralization and interpolation
4. EE translations merge on top of CE translations at load time

## File Structure

```
frontend/src/locales/          # CE translations
├── components/
│   ├── table.yaml
│   ├── editor.yaml
│   └── ...
└── pages/
    ├── chat.yaml
    ├── login.yaml
    └── ...

ee/frontend/src/locales/       # EE-only translations (overrides CE)
├── components/
│   └── dynamodb-login-form.yaml
└── pages/
    ├── login.yaml
    └── sql-agent.yaml
```

## Supported Languages

Languages are defined in `frontend/src/utils/languages.ts`. Each YAML file contains all languages as top-level keys.

## YAML File Format

Each file has one top-level key per language, with `en_US` as the source of truth:

```yaml
en_US:
  keyName: English text here
  anotherKey: Another string
  parameterized: "Hello {name}, you have {count} messages"

fr_FR:
  keyName: Texte en français ici
  anotherKey: Une autre chaîne
  parameterized: "Bonjour {name}, vous avez {count} messages"
```

## Usage in Components

### Basic Usage

```typescript
import { useTranslation } from '@/hooks/use-translation';

export const MyComponent: FC = () => {
  const { t } = useTranslation('components/my-component');

  return <button>{t('buttonLabel')}</button>;
};
```

### Supported `t()` Patterns

All interpolation uses `{placeholder}` syntax in YAML. The `t()` function's return type adapts based on the param values.

#### Plain string (no interpolation)

```yaml
en_US:
  submit: Submit
```
```typescript
t('submit')  // → "Submit" (string)
```

#### String interpolation

```yaml
en_US:
  greeting: "Hello {name}!"
  itemCount: "{count} items selected"
```
```typescript
t('greeting', { name: userName })    // → "Hello Alice!" (string)
t('itemCount', { count: 5 })         // → "5 items selected" (string)
```

#### Multiple placeholders

```yaml
en_US:
  searchMatch: "Match {current} of {total}"
```
```typescript
t('searchMatch', { current: 3, total: 10 })  // → "Match 3 of 10" (string)
```

#### JSX element interpolation

When any param value is a React element, `t()` returns `ReactNode` instead of `string`. This allows translators to reorder the element freely within the sentence.

```yaml
en_US:
  details: "Please refer to our {link} for more info."
```
```typescript
t('details', {
    link: <ExternalLink href="/privacy">{t('privacyPolicy')}</ExternalLink>
})
// → ReactNode: ["Please refer to our ", <ExternalLink/>, " for more info."]
```

#### Mixed string and JSX params

```yaml
en_US:
  welcome: "Hello {name}, see {link} for details."
```
```typescript
t('welcome', {
    name: 'Alice',
    link: <a href="/docs">the docs</a>
})
// → ReactNode (JSX detected, so all params go through the ReactNode path)
```

### 5. No Sentence Fragments

Never split a sentence across multiple translation keys or concatenate translated text with runtime values outside the string. Languages like Japanese, Chinese, and Korean have different word orders, so fragments can't be reordered by translators.

**Wrong:**
```yaml
en_US:
  createPrefix: Create a
```
```typescript
// Concatenating outside the translation — translators can't reorder
{t('createPrefix')} {storageUnitLabel}
```

**Correct:**
```yaml
en_US:
  createTitle: "Create a {storageUnit}"
```
```typescript
t('createTitle', { storageUnit: storageUnitLabel })
```

### JSX Inside Translated Strings

When a translated string must contain a React element (link, bold text, etc.), pass the element as a param value. The `t()` function detects JSX automatically and returns `ReactNode` instead of `string`.

```yaml
en_US:
  details: "For more information, see our {link}."
```

```typescript
// t() detects the JSX element and returns ReactNode
t('details', {
    link: <ExternalLink href="/privacy">{t('privacyPolicy')}</ExternalLink>
})
```

This works because `t()` has overloaded return types:
- `Record<string, string | number>` params → returns `string`
- `Record<string, ReactNode>` params (any JSX element) → returns `ReactNode`

No special function needed — just pass JSX as a param value.

## Critical Rules

### 1. No Fallback Strings

**Wrong:**
```typescript
t('buttonLabel', 'Click me')  // NO - fallback string
```

**Correct:**
```typescript
t('buttonLabel')  // YES - key must exist in YAML
```

Why: Fallback strings create maintenance burden and can hide missing translations.

### 2. No Hardcoded UI Text

**Wrong:**
```typescript
<button>Submit</button>
```

**Correct:**
```typescript
<button>{t('submit')}</button>
```

### 3. All User-Facing Strings Must Be Localized

This includes:
- Button labels
- Form placeholders
- Error messages
- Toast notifications
- Tooltips and aria-labels
- Dialog titles and content
- Table headers
- Status messages

### 4. Key Naming Conventions

Use camelCase keys that describe the content:
```yaml
en_US:
  submitButton: Submit
  cancelAction: Cancel
  loadingMessage: Loading...
  errorNotFound: Item not found
  confirmDelete: Are you sure you want to delete this?
```

## Pluralization

Uses CLDR plural rules via `Intl.PluralRules`. Add suffix keys for plural categories (`_one`, `_other`, `_two`, `_few`, `_many`, `_zero`):

```yaml
en_US:
  rowsSelected: "{count} rows selected"
  rowsSelected_one: "{count} row selected"

  deleteRowConfirmTitle: Delete Rows?
  deleteRowConfirmTitle_one: Delete Row?
```

```typescript
t('rowsSelected', { count: 1 })  // → "1 row selected" (picks _one)
t('rowsSelected', { count: 5 })  // → "5 rows selected" (picks base key)
```

The `count` param triggers plural resolution automatically. The base key (without suffix) is used as fallback when no matching plural form exists.

## Adding New Strings

1. Add the key to the appropriate YAML file under `en_US:`
2. Use `t('keyName')` in your component
3. Run the translation script to generate translations for all languages:
   ```bash
   cd dev/translate && node translate.mjs
   ```
   The script is resumable — it skips languages already present in each file.

## What NOT to Localize

- Technical identifiers (CSS classes, IDs)
- Console log messages (developer-only)
- Test fixtures
- Code comments

## Common Patterns

### Arrays of Strings

**Wrong:**
```typescript
const items = ["Option 1", "Option 2", "Option 3"];
```

**Correct:**
```yaml
en_US:
  option1: Option 1
  option2: Option 2
  option3: Option 3
```

```typescript
const items = [t('option1'), t('option2'), t('option3')];
```

### Conditional Text

```typescript
{isLoading ? t('loading') : t('ready')}
```

## Translation Tooling

### Machine Translation Script (`dev/translate/`)

Translates `en_US` strings to all supported languages using Google Translate (via `google-translate-api-x`). Handles both CE and EE locale files.

```bash
cd dev/translate
pnpm install
node translate.mjs              # all languages
node translate.mjs fr_FR de_DE  # specific languages only
```

- Resumable: skips locale blocks already present in each YAML file
- Protects `{placeholder}` tokens during translation
- Handles `en_GB` via US→UK spelling substitutions
- Groups duplicate Google Translate codes (e.g. `zh_TW`/`zh_HK`) to avoid redundant API calls

### Drift Detection & Import (`ee/dev/translation-tool/`)

Tools for managing translation quality after machine translation:

**`generate.py`** — Finds missing, stale, and orphaned translations. Produces a ZIP package for human translators.

```bash
cd ee/dev/translation-tool
uv run python generate.py                    # all languages
uv run python generate.py -l "fr_FR,de_DE"   # specific languages
```

- Uses SHA-256 checksums (`checksums.json`) to detect when English strings change after translation
- Reports missing keys (untranslated), stale keys (English changed), orphaned keys (no longer in English)

**`import_translations.py`** — Imports human-reviewed translations back into CE and EE locale files.

```bash
uv run python import_translations.py /path/to/translations --dry-run
uv run python import_translations.py /path/to/translations
```

- Routes keys to the correct file: CE keys → CE file, EE-only keys → EE file
- Normalizes indentation, fixes `{{placeholders}}` → `{placeholders}`, quotes values with braces

## Verifying Translations

Before committing, ensure:
1. All `t()` calls have corresponding YAML keys
2. No fallback strings are used
3. No hardcoded user-facing text remains

## Reference Files

- Languages: `frontend/src/utils/languages.ts`
- Hook: `frontend/src/hooks/use-translation.ts`
- i18n engine: `frontend/src/utils/i18n.ts`
- Example YAML: `frontend/src/locales/components/table.yaml`
- Translation script: `dev/translate/translate.mjs`
- Drift detection: `ee/dev/translation-tool/generate.py`
- Translation import: `ee/dev/translation-tool/import_translations.py`
