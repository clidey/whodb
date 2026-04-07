# Localization Standards

This document defines the localization requirements for the WhoDB frontend.

## Overview

WhoDB uses a YAML-based localization system with the `useTranslation` hook. All user-facing strings MUST be localized.

## How It Works

1. Translation files are YAML files in `frontend/src/locales/`
2. Components load translations via `useTranslation('component-path')`
3. The `t()` function retrieves strings by key

## File Structure

```
frontend/src/locales/
├── components/
│   ├── table.yaml
│   ├── editor.yaml
│   └── ...
└── pages/
    ├── chat.yaml
    ├── login.yaml
    └── ...
```

## YAML File Format

```yaml
en:
  keyName: English text here
  anotherKey: Another string
  parameterized: "Hello {name}, you have {count} messages"
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
en:
  submit: Submit
```
```typescript
t('submit')  // → "Submit" (string)
```

#### String interpolation

```yaml
en:
  greeting: "Hello {name}!"
  itemCount: "{count} items selected"
```
```typescript
t('greeting', { name: userName })    // → "Hello Alice!" (string)
t('itemCount', { count: 5 })         // → "5 items selected" (string)
```

#### Multiple placeholders

```yaml
en:
  searchMatch: "Match {current} of {total}"
```
```typescript
t('searchMatch', { current: 3, total: 10 })  // → "Match 3 of 10" (string)
```

#### JSX element interpolation

When any param value is a React element, `t()` returns `ReactNode` instead of `string`. This allows translators to reorder the element freely within the sentence.

```yaml
en:
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
en:
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
en:
  createPrefix: Create a
```
```typescript
// Concatenating outside the translation — translators can't reorder
{t('createPrefix')} {storageUnitLabel}
```

**Correct:**
```yaml
en:
  createTitle: "Create a {storageUnit}"
```
```typescript
t('createTitle', { storageUnit: storageUnitLabel })
```

### JSX Inside Translated Strings

When a translated string must contain a React element (link, bold text, etc.), pass the element as a param value. The `t()` function detects JSX automatically and returns `ReactNode` instead of `string`.

```yaml
en:
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
en:
  submitButton: Submit
  cancelAction: Cancel
  loadingMessage: Loading...
  errorNotFound: Item not found
  confirmDelete: Are you sure you want to delete this?
```

## Adding New Strings

1. Add the key to the appropriate YAML file under `en:`
2. Use `t('keyName')` in your component

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
# In YAML
en:
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

## Verifying Translations

Before committing, ensure:
1. All `t()` calls have corresponding YAML keys
2. No fallback strings are used
3. No hardcoded user-facing text remains

## Reference Files

- Hook: `frontend/src/hooks/use-translation.ts`
- Utility: `frontend/src/utils/i18n.ts`
- Example YAML: `frontend/src/locales/components/table.yaml`
