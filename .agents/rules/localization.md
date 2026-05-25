---
paths:
  - "frontend/src/locales/**"
  - "ee/frontend/src/locales/**"
  - "dev/translate/**"
---

# Localization Rules

## Adding New Translation Keys
1. Check `common.yaml` first — if the key exists there, just use it
2. If shared across 2+ components → add to `common.yaml`
3. If component-specific → add to that component's YAML file
4. Always add `en_US` entry first
5. Run: `cd dev/translate && python3 detect.py && node translate.mjs`

## YAML Format
```yaml
en_US:
  keyName: English text
fr_FR:
  keyName: Texte en français
```

## Key Rules
- camelCase key names
- No sentence fragments — full sentences with `{placeholder}` interpolation
- No duplicate keys across files (component keys override common)
- Pluralization uses `_one`, `_other`, `_few`, `_many`, `_zero` suffixes with `count` param

## Merge Order
common.yaml → component YAML → EE extension (last wins)
