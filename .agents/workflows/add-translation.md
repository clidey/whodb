---
name: add-translation
description: Add or update translation keys across all languages with proper tooling
---

# Add Translation Keys

## Quick Path (single key)

### 1. Determine Location
- Shared across components → `frontend/src/locales/common.yaml`
- Component-specific → `frontend/src/locales/components/<name>.yaml` or `pages/<name>.yaml`
- EE-only → `ee/frontend/src/locales/...`

### 2. Add en_US Entry
```yaml
en_US:
  newKey: English text here
  parameterized: "Hello {name}, you have {count} items"
```

### 3. Generate All Translations
```bash
cd dev/translate && bash run.sh
```

### 4. Use in Component
```typescript
const { t } = useTranslation('components/my-component');
// or for common keys, just use t('key') — they're auto-available

t('newKey')
t('parameterized', { name: userName, count: itemCount })
```

## Pluralization
Add suffix variants:
```yaml
en_US:
  items: "{count} items"
  items_one: "{count} item"
```
The `count` param triggers plural resolution automatically.

## JSX in Translations
```yaml
en_US:
  details: "See our {link} for info."
```
```typescript
t('details', { link: <a href="/docs">{t('docsLink')}</a> })
```

## Removing Keys
1. Remove from YAML file
2. Run `cd dev/translate && bash run.sh`

## Verification
```bash
cd frontend && pnpm run build:ce  # TypeScript catches missing keys
```
