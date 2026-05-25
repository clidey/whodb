---
name: new-frontend-page
description: Add a new page/route to the WhoDB frontend with proper localization and routing
---

# Add a New Frontend Page

## Steps

### 1. Create Page Component
```
frontend/src/pages/<feature-name>/index.tsx
```

```typescript
import { FC } from 'react';
import { useTranslation } from '@/hooks/use-translation';

export const FeatureNamePage: FC = () => {
  const { t } = useTranslation('pages/feature-name');

  return (
    <div>
      <h1>{t('title')}</h1>
    </div>
  );
};
```

### 2. Add Translation File
Create `frontend/src/locales/pages/feature-name.yaml`:
```yaml
en_US:
  title: Feature Name
```

Then run:
```bash
cd dev/translate && python3 detect.py && node translate.mjs
```

### 3. Add Route
In the router configuration, add the new route. Follow existing patterns for layout and navigation guards.

### 4. Add Sidebar Entry (if needed)
Update the sidebar configuration to include the new page with appropriate icon and label.

### 5. Add Keyboard Shortcut (if needed)
In `frontend/src/utils/shortcuts.ts`:
```typescript
export const SHORTCUTS = {
  // ...existing
  featureName: { key: 'X', modifiers: ['meta'], displayKeys: ['⌘', 'X'] },
};
```

### 6. Verification
```bash
cd frontend && pnpm run build:ce
cd frontend && pnpm start  # Visual verification in browser
```

### 7. E2E Test
Add test file `frontend/e2e/tests/features/feature-name.spec.mjs` following existing patterns.
