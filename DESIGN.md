---
name: DataFlow
description: Sealos-native database workspace for inspection, editing, querying, and lightweight analysis.
colors:
  background: "#ffffff"
  foreground: "#18181b"
  surface: "#ffffff"
  sidebar: "#fafafa"
  muted: "#f4f4f5"
  muted-foreground: "#71717a"
  border: "#e4e4e7"
  primary: "#27272a"
  primary-foreground: "#fafafa"
  destructive: "#e5484d"
  success: "#28a980"
  warning: "#d99b19"
  highlight: "#3366d8"
typography:
  body:
    fontFamily: "Geist Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "0"
  title:
    fontFamily: "Geist Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: 1.4
    letterSpacing: "0"
  label:
    fontFamily: "Geist Sans, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 500
    lineHeight: 1.33
    letterSpacing: "0"
  mono:
    fontFamily: "Geist Mono, ui-monospace, SFMono-Regular, monospace"
rounded:
  sm: "6px"
  md: "8px"
  lg: "10px"
  xl: "14px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.foreground}"
    rounded: "{rounded.md}"
    padding: "8px"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
---

# Design System: DataFlow

## 1. Overview

**Creative North Star: "Calm Workbench"**

DataFlow is a task-first database workspace. It should feel quiet, trustworthy, and capable: closer to a focused editor than a decorative dashboard. The product carries dense database state, pending edits, query output, and chart composition in one shell, so the visual system favors restrained contrast, clear affordances, and predictable component behavior.

The interface explicitly rejects enterprise-console clutter, casual treatment of destructive database work, and decorative visual treatments that compete with records or query results. The product can be dense, but each dense surface should be scannable through alignment, stable spacing, and consistent controls.

**Key Characteristics:**

- White and near-neutral surfaces with strong text contrast.
- A fixed app shell with activity rail, resizable sidebar, tab strip, and main work area.
- Icons from `lucide-react`, paired with concise labels only where the label improves scan speed.
- State-driven styling for active tabs, pending edits, errors, success, warning, and selected rows.

## 2. Colors

The palette is restrained and neutral, with semantic color reserved for state and database-risk communication.

### Primary

- **Workbench Ink** (`--primary`, `oklch(0.205 0 0)`): primary actions, selected controls, and strong UI emphasis.
- **Focus Blue** (`--highlight`, `oklch(0.623 0.214 259.815)`): highlighted matches, active semantic emphasis, and focused workflow moments.

### Secondary

- **Success Green** (`--success`, `oklch(0.696 0.17 162.48)`): successful export, save, or apply states.
- **Warning Amber** (`--warning`, `oklch(0.769 0.188 70.08)`): caution states before potentially risky changes.
- **Destructive Red** (`--destructive`, `oklch(0.577 0.245 27.325)`): delete, drop, clear, and failed states.

### Neutral

- **Canvas White** (`--background`, `oklch(1 0 0)`): application base.
- **Sidebar Wash** (`--sidebar`, `oklch(0.985 0 0)`): navigation, sidebar, and workbench framing.
- **Muted Panel** (`--muted`, `oklch(0.97 0 0)`): inactive controls, subtle rows, and toolbar backgrounds.
- **Divider Gray** (`--border`, `oklch(0.922 0 0)`): table borders, panel dividers, input strokes.
- **Readable Gray** (`--muted-foreground`, `oklch(0.556 0 0)`): secondary text only, never the main explanation for a destructive or blocking state.

### Named Rules

**The State Color Rule.** Use saturated color for user state, validation, and selected workflow feedback. Do not use it as page decoration.

**The Mutation Clarity Rule.** Destructive or database-writing actions must use semantic color, explicit labels, and confirmation state rather than relying on placement alone.

## 3. Typography

**Display Font:** Geist Sans with system sans fallback
**Body Font:** Geist Sans with system sans fallback
**Label/Mono Font:** Geist Mono for code, SQL, JSON, Redis commands, identifiers, and technical values

**Character:** The type system is product-native and compact. It uses weight and spacing instead of display-scale typography because the app is an operational tool.

### Hierarchy

- **Display**: Rare in the app shell. Avoid large marketing-scale headings inside authenticated surfaces.
- **Headline** (`text-lg` to `text-xl`, semibold): modal titles, empty-state titles, and major dashboard headings.
- **Title** (`text-sm` to `text-base`, medium or semibold): panel headers, tab labels, toolbar titles.
- **Body** (`text-sm`, regular): forms, explanatory copy, table supporting text. Keep prose blocks short and avoid repeating the visible heading.
- **Label** (`text-xs` to `text-sm`, medium): field labels, badges, table metadata, icon button tooltips.
- **Mono** (`font-mono`, `text-xs` to `text-sm`): SQL, JSON, shell fragments, raw values, and IDs.

### Named Rules

**The Compact Type Rule.** Keep product controls on a fixed rem scale. Do not scale app-shell or toolbar type with viewport width.

## 4. Elevation

DataFlow is flat by default. Depth comes from borders, background changes, sticky table separators, and modal overlays rather than large shadows. This preserves the feel of a workbench and keeps attention on data.

### Shadow Vocabulary

- **Modal Layer**: Dialogs may use the shared dialog overlay and surface styling from `src/components/ui/dialog`.
- **Interactive State**: Prefer background and border changes for hover, active, selected, and focus states.

### Named Rules

**The Flat Surface Rule.** Tables, sidebars, tabs, toolbars, and dashboard canvases should not gain decorative drop shadows at rest.

## 5. Components

### Buttons

- **Shape:** restrained rounded rectangles using the shared radius scale, generally 6 to 10px.
- **Primary:** dark neutral background with high-contrast text for the main action in a local context.
- **Ghost:** transparent at rest, muted background on hover, used for toolbars and icon actions.
- **Focus:** visible focus ring via the shared `ring` token and Tailwind focus utilities.
- **Disabled:** preserve layout and icon position while reducing affordance with muted text and disabled cursor.

### Chips

- **Style:** small, compact labels for status, filters, data type, or selected resource metadata.
- **State:** selected chips should show an explicit selected background or border, not color alone.

### Cards / Containers

- **Corner Style:** 8 to 10px for repeated cards and dialogs; avoid oversized rounded panels.
- **Background:** use `--card`, `--sidebar`, and `--muted` for hierarchy.
- **Shadow Strategy:** flat by default; rely on borders and tonal contrast.
- **Internal Padding:** compact by default, usually 8 to 16px in panels and 16 to 24px in dialogs.

### Inputs / Fields

- **Style:** neutral border, white surface, compact height, monospace for query or JSON fields.
- **Focus:** ring or border emphasis that does not resize the field.
- **Error / Disabled:** semantic message plus visual state. Never use color alone for errors.

### Navigation

- **Activity Bar:** fixed 80px rail with icon plus short label for the two workspaces.
- **Sidebar:** resizable context tree for database browsing or dashboard management.
- **Tabs:** database workspace tabs own query and storage-unit surfaces; tabs can show unsaved database edit state.
- **Leave Guard:** only blocks actions that would discard database edits, not ordinary switching between open workspace tabs.

## 6. Do's and Don'ts

**Do**

- Keep data tables, collection grids, and query results visually stable under loading, hover, and edit states.
- Use `lucide-react` icons in toolbars and pair unfamiliar icon-only actions with tooltips.
- Preserve MongoDB terminology: Collection, Document, Collection Table View, JSON View.
- Localize all user-facing strings through the existing i18n message system.
- Add `data-testid` and `data-qa-*` semantics to new interactive surfaces when they matter for automation.

**Don't**

- Do not make database mutation feel playful or casual.
- Do not add decorative cards around already-framed tools.
- Do not use large display headings inside compact panels or modals.
- Do not introduce new color families for one-off components.
- Do not call MongoDB collections tables except for the explicit Collection Table View concept.
