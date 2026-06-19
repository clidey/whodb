---
name: WhoDB
description: Lightweight, fast database management with quiet confidence
colors:
  brand-blue: "#11518E"
  brand-blue-light: "#8EBFFB"
  muted-light: "#f8f8f8"
  muted-dark: "#1a1a1a"
  surface-light: "#ffffff"
  surface-dark: "#0f0f0f"
  code-bg-light: "#f5f5f5"
  code-bg-dark: "#252526"
  foreground-light: "#0f0f0f"
  foreground-dark: "#fafafa"
  muted-foreground-light: "#6b7280"
  muted-foreground-dark: "#9ca3af"
typography:
  body:
    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: 1.4
  title:
    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 600
    lineHeight: 1.3
  headline:
    fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.25
rounded:
  sm: "0.375rem"
  md: "0.5rem"
  lg: "0.625rem"
  xl: "0.75rem"
spacing:
  xs: "0.5rem"
  sm: "0.75rem"
  md: "1rem"
  lg: "1.5rem"
  xl: "2rem"
components:
  button-primary:
    backgroundColor: "{colors.brand-blue}"
    textColor: "#ffffff"
    rounded: "{rounded.md}"
    padding: "0.5rem 1rem"
  button-primary-hover:
    backgroundColor: "#0d4070"
    textColor: "#ffffff"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.foreground-light}"
    rounded: "{rounded.md}"
    padding: "0.5rem 1rem"
  card:
    backgroundColor: "{colors.surface-light}"
    rounded: "{rounded.lg}"
    padding: "1rem"
  input:
    backgroundColor: "{colors.surface-light}"
    textColor: "{colors.foreground-light}"
    rounded: "{rounded.md}"
    padding: "0.5rem 0.75rem"
---

# Design System: WhoDB

## 1. Overview

**Creative North Star: "The Clean Terminal"**

Terminal efficiency meets modern interface. WhoDB's visual system channels the clarity of a well-configured terminal — functional density, monochrome confidence, and zero decorative noise — while maintaining the approachability of a modern product UI. Every pixel serves comprehension or interaction; nothing exists for ornament.

The system rejects the cluttered chaos of legacy database tools (phpMyAdmin, pgAdmin) and the bloated menu hierarchies of enterprise software (Oracle, IBM tooling). It equally rejects playful SaaS illustration-fests that undermine credibility with infrastructure professionals.

Information is the interface. The design removes itself so data can speak. Controls are discoverable but unobtrusive. State changes are immediate. The system assumes expertise and rewards it.

**Key Characteristics:**
- Monochrome-dominant with a single blue accent used sparingly
- Flat surfaces differentiated by tonal layering, not shadows
- Inter variable font at multiple weights for clear hierarchy without font pairing
- Configurable density (compact / comfortable / spacious spacing)
- Dark and light modes as equal citizens — neither is an afterthought

## 2. Colors

A restrained palette: tinted neutrals plus one accent at ≤10% surface coverage.

### Primary
- **Steady Blue** (#11518E / light mode): The brand voice. Used for primary actions, active states, and brand identity marks. Appears in the logo, selected navigation items, and CTAs. Never used as a background fill on large surfaces.
- **Lifted Blue** (#8EBFFB / dark mode): The dark-mode complement. Same role, lighter value for contrast on dark surfaces.

### Neutral
- **Ink** (#0f0f0f / light mode foreground): Primary text and high-emphasis elements.
- **Ink Inverted** (#fafafa / dark mode foreground): Primary text on dark surfaces.
- **Muted Foreground** (#6b7280 light / #9ca3af dark): Secondary text, labels, placeholders.
- **Surface** (#ffffff light / #0f0f0f dark): Primary background.
- **Muted Surface** (#f8f8f8 light / #1a1a1a dark): Elevated cards, sidebar, secondary surfaces.
- **Code Surface** (#f5f5f5 light / #252526 dark): Editor backgrounds, code blocks.
- **Backdrop** (oklch(0 0 0 / 50%) light / oklch(0 0 0 / 60%) dark): Modal overlays.

### Named Rules
**The 10% Rule.** The primary blue is used on ≤10% of any given screen. Its rarity gives it authority. When everything is blue, nothing is.

## 3. Typography

**Body Font:** Inter (variable, 100–900 weight, with optical sizing)
**Fallback Stack:** ui-sans-serif, system-ui, sans-serif

**Character:** One family, many weights. Inter's optical sizing and weight range provide the entire hierarchy without font-pairing risk. The variable font loads a single file covering all weights, keeping resource overhead minimal.

### Hierarchy
- **Headline** (600, 1.5rem / 24px, line-height 1.25): Page titles, section headers.
- **Title** (600, 1.25rem / 20px, line-height 1.3): Card titles, dialog headers, sidebar section labels.
- **Body** (400, 1rem / 16px, line-height 1.5): Default text. Max line length 65–75ch where prose appears.
- **Label** (500, 0.875rem / 14px, line-height 1.4): Form labels, table headers, metadata.
- **Caption** (400, 0.75rem / 12px, line-height 1.4): Timestamps, badge text, secondary metadata.

### Named Rules
**The Weight-Not-Size Rule.** Hierarchy within a section is conveyed through weight changes (400 → 500 → 600) before reaching for size changes. Size jumps are reserved for structural boundaries (page title vs card content), not inline emphasis.

## 4. Elevation

Flat by default. Depth is conveyed through tonal layering (surface → muted surface → code surface) and 1px borders, not shadows. Shadows appear only as a response to state.

### Shadow Vocabulary
- **Highlight** (`shadow-2xl`): Transient attention signal on newly-created or freshly-navigated-to cards. Fades after 3 seconds.
- **Dropdown / Popover** (system-provided by @clidey/ux): Only on floating elements that escape document flow.
- **Modal backdrop** (`oklch(0 0 0 / 50-60%)`): Overlay dim, not a shadow.

### Named Rules
**The Flat-By-Default Rule.** Surfaces are flat at rest. The only shadows in the system are transient state indicators (highlight pulse) or floating-layer signals (popovers, modals). No ambient shadows on cards, no resting-state elevation.

## 5. Components

All interactive components are sourced from `@clidey/ux` (a shadcn/ui-derived library). The system extends them through Tailwind classes and CSS variables — never by forking the library.

### Buttons
- **Shape:** Medium radius (0.5rem / 8px)
- **Primary:** Steady Blue background, white text, 0.5rem vertical / 1rem horizontal padding
- **Hover:** Darkened blue (#0d4070), no transform, no shadow
- **Ghost:** Transparent background, foreground text, same radius. Hover reveals muted-surface tint.
- **Character:** Restrained and precise. No gradients, no elevation on hover. State changes are color shifts only.

### Cards
- **Corner Style:** Gently curved (0.625rem / 10px radius)
- **Background:** Surface color (white/dark)
- **Border:** 1px border in muted tone (provided by @clidey/ux defaults)
- **Shadow:** None at rest. `shadow-2xl` on transient highlight only (3s fade).
- **Internal Padding:** 1rem (py-4 px-4)
- **Expandable Cards:** Click-to-expand via Sheet (right drawer). Fixed 240px min-width, 200px min-height.

### Inputs / Fields
- **Style:** Surface background, 1px border, medium radius
- **Focus:** Brand-blue ring (provided by @clidey/ux focus-visible system)
- **Code Editor:** Dedicated code-surface background (#f5f5f5 light / #252526 dark) using CodeMirror

### Navigation (Sidebar)
- **Style:** Collapsible sidebar via `@clidey/ux` SidebarProvider
- **Active State:** Brand-blue text/icon, muted-surface background tint
- **Hover:** Muted-surface background tint
- **Sections:** Separated by `SidebarSeparator` (1px muted line)
- **Trigger:** Hamburger toggle, collapses to icon-only rail

### Data Table
- **The signature component.** Spreadsheet-like data grid with virtualization for large datasets.
- **Header:** Label weight (500), background tint for visual separation
- **Rows:** Full-width, minimal vertical padding for density. Hover highlights row.
- **Search Highlight:** Background transition (0.3s ease-in-out) with 4px radius

## 6. Do's and Don'ts

### Do:
- **Do** use Inter at multiple weights (400, 500, 600) to create hierarchy without introducing additional typefaces.
- **Do** differentiate surfaces with tonal steps (surface → muted → code) rather than shadows.
- **Do** keep the brand blue to ≤10% of any screen. Active nav items, primary CTAs, brand marks only.
- **Do** provide full keyboard navigation for all interactive elements (via @clidey/ux defaults and custom shortcut system).
- **Do** respect the configurable density system (compact/comfortable/spacious) — spacing variables, not hardcoded values.
- **Do** test both light and dark mode as equal citizens. Neither is derived from the other.

### Don't:
- **Don't** add ambient shadows to resting cards or surfaces. Flat-by-default is the system.
- **Don't** use gradient fills on buttons, backgrounds, or text. Solid, single-tone fills only.
- **Don't** add illustrations, mascots, or decorative graphics. The data IS the content.
- **Don't** introduce additional typefaces. Inter handles the entire hierarchy.
- **Don't** create cluttered toolbars with dozens of visible buttons (phpMyAdmin pattern). Progressive disclosure via command palette and keyboard shortcuts.
- **Don't** gate features behind wizard flows or multi-step modals (enterprise bloat pattern). Surface power directly.
- **Don't** use playful rounded corners (≥1rem on small elements), pastel accent colors, or illustration-heavy empty states. This is infrastructure tooling.
- **Don't** add loading skeletons or spinners where instant rendering is achievable. Speed is the feature.
