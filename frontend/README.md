# WhoDB Frontend Development Guide

This guide covers frontend development for WhoDB, including theme customization, component extension, and enterprise feature integration.

## Architecture Overview

The frontend uses:
- **React** with TypeScript
- **Vite** for building and development
- **TailwindCSS** for styling
- **Apollo GraphQL** for API communication
- **Dynamic imports** for EE/CE separation

## Theme System

### Current Theme Architecture

WhoDB uses a CSS variable-based theme system that allows runtime theme switching:

```typescript
// src/theme/theme-provider.tsx
export const themes = {
  light: {
    // Background colors
    '--color-background': '#ffffff',
    '--color-surface': '#f9fafb',
    '--color-card': '#ffffff',
    
    // Text colors
    '--color-text': '#111827',
    '--color-text-secondary': '#6b7280',
    
    // Border colors
    '--color-border': '#e5e7eb',
    
    // Primary colors
    '--color-primary': '#3b82f6',
    '--color-primary-hover': '#2563eb',
  },
  dark: {
    // Dark theme variables
  }
}
```

### Adding New Themes

#### Option 1: Add to Existing Theme System (Recommended)

1. **Add theme definition**:
```typescript
// src/theme/themes/ocean.ts
export const oceanTheme = {
  '--color-background': '#0a192f',
  '--color-surface': '#172a45',
  '--color-card': '#172a45',
  '--color-text': '#ccd6f6',
  '--color-text-secondary': '#8892b0',
  '--color-border': '#233554',
  '--color-primary': '#64ffda',
  '--color-primary-hover': '#4dd4b8',
  // ... all other variables
}
```

2. **Register theme**:
```typescript
// src/theme/theme-provider.tsx
import { oceanTheme } from './themes/ocean';

export const themes = {
  light: lightTheme,
  dark: darkTheme,
  ocean: oceanTheme,
  // ... other themes
}
```

3. **Add to theme selector**:
```typescript
// Update theme selector component to include new theme
```

#### Option 2: Theme Branches (For Major UI Changes)

If themes require structural changes:

1. **Create feature branch**:
```bash
git checkout -b theme/ocean-theme
```

2. **Implement theme-specific components**:
```typescript
// Override components as needed
// src/components/table.tsx
export const Table = () => {
  const theme = useTheme();
  
  if (theme === 'ocean') {
    return <OceanTable />;
  }
  
  return <DefaultTable />;
}
```

3. **Maintain as parallel branch**:
- Cherry-pick features from main
- Periodically rebase on main
- Build separate artifacts

### Enterprise Themes

For EE-specific themes:

```typescript
// ee/frontend/src/themes/enterprise.ts
export const enterpriseThemes = {
  corporate: {
    // Corporate theme variables
  },
  highContrast: {
    // Accessibility-focused theme
  }
}

// ee/frontend/src/themes/index.ts
export { enterpriseThemes } from './enterprise';
```

Integration in CE:
```typescript
// src/theme/theme-provider.tsx
const eeThemes = await loadEEThemes();
const allThemes = { ...themes, ...eeThemes };
```

## Component Extension

### Creating Extensible Components

1. **Design with extension in mind**:
```typescript
// src/components/DataTable/DataTable.tsx
export interface DataTableProps {
  data: any[];
  columns: Column[];
  // Extension points
  renderCell?: (value: any, column: Column) => React.ReactNode;
  renderHeader?: (column: Column) => React.ReactNode;
  onRowClick?: (row: any) => void;
  className?: string;
}

export const DataTable: React.FC<DataTableProps> = ({
  renderCell = defaultRenderCell,
  renderHeader = defaultRenderHeader,
  ...props
}) => {
  // Component implementation
}
```

2. **Create EE extensions**:
```typescript
// ee/frontend/src/components/DataTable/EnhancedDataTable.tsx
export const EnhancedDataTable = (props: DataTableProps) => {
  return (
    <DataTable
      {...props}
      renderCell={enhancedRenderCell}
      className="ee-data-table"
    />
  );
}
```

### Dynamic Component Loading

```typescript
// src/utils/ee-loader.ts
export async function loadEEComponent<T>(
  path: string,
  fallback: T
): Promise<T> {
  if (!import.meta.env.VITE_ENABLE_EE) {
    return fallback;
  }
  
  try {
    const module = await import(`../../ee/frontend/src/${path}`);
    return module.default || module;
  } catch {
    return fallback;
  }
}

// Usage
const DataViz = await loadEEComponent(
  'components/DataViz',
  BasicDataViz
);
```

## Adding New Features

### Community Edition Features

1. **Create feature in CE**:
```typescript
// src/features/query-builder/QueryBuilder.tsx
export const QueryBuilder = () => {
  // CE implementation
}
```

2. **Add to routing**:
```typescript
// src/config/routes.tsx
{
  path: '/query-builder',
  element: <QueryBuilder />
}
```

### Enterprise Edition Features

1. **Create in EE module**:
```typescript
// ee/frontend/src/features/advanced-analytics/AdvancedAnalytics.tsx
export const AdvancedAnalytics = () => {
  // EE implementation
}
```

2. **Export from EE**:
```typescript
// ee/frontend/src/pages/index.ts
export { AdvancedAnalytics } from '../features/advanced-analytics';
```

3. **Conditionally load in CE**:
```typescript
// src/config/routes.tsx
const routes = [
  // ... CE routes
];

if (import.meta.env.VITE_ENABLE_EE) {
  const eeRoutes = await import('../../ee/frontend/src/routes');
  routes.push(...eeRoutes.default);
}
```

## State Management

### Global State with Zustand

```typescript
// src/store/theme.ts
interface ThemeStore {
  theme: string;
  setTheme: (theme: string) => void;
  // EE extension point
  customThemes?: Record<string, ThemeConfig>;
}

export const useThemeStore = create<ThemeStore>((set) => ({
  theme: 'light',
  setTheme: (theme) => set({ theme }),
}));
```

### EE State Extensions

```typescript
// ee/frontend/src/store/ee-features.ts
interface EEFeatureStore {
  advancedMode: boolean;
  setAdvancedMode: (enabled: boolean) => void;
}

// Merge with CE stores
```

## Building and Development

### Development Mode

```bash
# CE only
pnpm dev

# With EE features
VITE_ENABLE_EE=true pnpm dev
```

### Production Build

```bash
# CE build
pnpm build

# EE build
VITE_ENABLE_EE=true pnpm build
```

### Theme-Specific Builds

```bash
# Build with specific theme as default
VITE_DEFAULT_THEME=ocean pnpm build

# Build with only specific themes
VITE_AVAILABLE_THEMES=light,dark,ocean pnpm build
```

## Best Practices

### 1. Theme Development

- **Use CSS variables**: All colors and spacing should use variables
- **Test all themes**: Ensure components work with all theme variations
- **Accessibility**: Test with high contrast themes
- **Performance**: Lazy load theme-specific assets

### 2. Component Architecture

- **Composition over inheritance**: Use React composition patterns
- **Props for customization**: Make components configurable
- **Render props**: For complex customization needs
- **Hooks for logic**: Extract reusable logic into hooks

### 3. EE/CE Separation

- **Dynamic imports**: Load EE code only when needed
- **Feature flags**: Use environment variables for feature toggling
- **Graceful fallbacks**: Always provide CE alternatives
- **Type safety**: Ensure TypeScript types work for both editions

### 4. Performance

- **Code splitting**: Use dynamic imports for large features
- **Lazy loading**: Load components as needed
- **Memoization**: Use React.memo and useMemo appropriately
- **Virtual scrolling**: For large data sets

## Testing

### Unit Tests

```bash
# Run all tests
pnpm test

# Run with coverage
pnpm test:coverage

# Test specific theme
THEME=ocean pnpm test
```

### E2E Tests

```bash
# Run Cypress tests
pnpm test:e2e

# Test with specific theme
CYPRESS_THEME=ocean pnpm test:e2e
```

### Theme Testing

```typescript
// src/tests/theme.test.tsx
describe('Theme Tests', () => {
  themes.forEach(theme => {
    it(`renders correctly with ${theme} theme`, () => {
      render(
        <ThemeProvider theme={theme}>
          <App />
        </ThemeProvider>
      );
      // Test theme-specific behavior
    });
  });
});
```

## Deployment Strategies

### Single Build, Multiple Themes

Default approach - themes switchable at runtime:
```nginx
# Serve same build for all users
location / {
  root /var/www/whodb;
}
```

### Theme-Specific Builds

For optimized, theme-specific deployments:
```bash
# Build each theme
for theme in light dark ocean; do
  VITE_DEFAULT_THEME=$theme pnpm build
  mv dist dist-$theme
done

# Deploy to different subdomains
# ocean.whodb.com -> dist-ocean/
# dark.whodb.com -> dist-dark/
```

### A/B Testing Themes

```typescript
// src/app.tsx
const App = () => {
  const theme = useABTest('theme-experiment', ['light', 'ocean']);
  
  return (
    <ThemeProvider theme={theme}>
      {/* App content */}
    </ThemeProvider>
  );
}
```

## Contributing

### Adding a New Theme

1. Create theme file in `src/themes/`
2. Add comprehensive color palette
3. Test with all components
4. Add theme preview/screenshot
5. Update documentation

### Creating Extensible Components

1. Identify extension points
2. Use composition patterns
3. Document props clearly
4. Provide usage examples
5. Consider EE extensions

### Performance Guidelines

1. Measure before optimizing
2. Use React DevTools Profiler
3. Monitor bundle size
4. Lazy load when appropriate
5. Test on slower devices

For questions or support, please file an issue on GitHub.