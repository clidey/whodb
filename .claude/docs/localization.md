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

### With Parameters

```typescript
// YAML: greeting: "Hello {name}!"
t('greeting', { name: userName })
```

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
3. If EE needs Spanish, add translation to `ee/frontend/src/locales/`

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

### Dynamic Content with Parameters

```yaml
en:
  itemCount: "{count} items"
  greeting: "Welcome, {username}"
```

```typescript
t('itemCount', { count: items.length })
t('greeting', { username: user.name })
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
