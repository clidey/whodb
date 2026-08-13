---
name: WhoDB
description: Lightweight, fast database management with quiet confidence
colors:
  background-light: "#FAF5EC"
  background-dark: "#0E2240"
  foreground-light: "#0E2240"
  foreground-dark: "#FAF5EC"
  card-light: "#FFFFFF"
  card-dark: "#162B4A"
  popover-light: "#FFFFFF"
  popover-dark: "#1D3151"
  muted-foreground-light: "#5C6778"
  muted-foreground-dark: "#9BA5B3"
  primary-light: "#2C6BD4"
  primary-dark: "#5091FD"
  accent-light: "#EAF0FB"
  accent-dark: "#233E69"
  border-light: "#D9D4CB"
  border-dark: "#314667"
  brand-orange: "#F4781C"
  highlight-yellow: "#FFC233"
  destructive-light: "#E30117"
  destructive-dark: "#FF6668"
typography:
  body:
    fontFamily: "Hanken Grotesk, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "Hanken Grotesk, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: 1.4
  title:
    fontFamily: "Hanken Grotesk, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 600
    lineHeight: 1.3
  headline:
    fontFamily: "Hanken Grotesk, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.25
  mono:
    fontFamily: "JetBrains Mono, ui-monospace, SF Mono, Menlo, monospace"
    fontSize: "0.875rem"
    fontWeight: 400
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
    backgroundColor: "{colors.primary-light}"
    textColor: "#FFFFFF"
    rounded: "{rounded.md}"
    padding: "0.5rem 1rem"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.foreground-light}"
    rounded: "{rounded.md}"
  card:
    backgroundColor: "{colors.card-light}"
    textColor: "{colors.foreground-light}"
    rounded: "{rounded.lg}"
    padding: "1rem"
---

# Design System: WhoDB

## Overview

**Creative North Star: "Quiet Confidence"**

WhoDB is a database management tool for people who already know what they're doing. The interface doesn't perform competence — it assumes it. Every surface exists to frame data, never to decorate it. All tokens flow from `@clidey/ux`'s Clidey brand kit (imported directly via `@import '@clidey/ux/brand.css'`): Paper/Ink dual-mode, Signal Blue as the product accent, Beacon Yellow as the single sanctioned highlight, Clidey Orange reserved for company-branded moments rather than general product UI.

The system explicitly rejects:
- The cluttered, dated aesthetic of legacy database tools (phpMyAdmin, pgAdmin)
- Corporate-gray enterprise bloat and wizard-driven flows
- Mixed-hue decoration — more than one accent color competing on the same surface

**Key Characteristics:**
- Flat surfaces by default; depth comes from a lightness step between surfaces, not shadow or blur
- Dual theme: Paper (light) and Ink (dark), both first-class
- Hanken Grotesk carries the UI; Bricolage Grotesque is self-hosted and available (`brand-fonts.css`) but not currently applied anywhere in the product UI — treat it as reserved for a future display-heading use, not an active token
- A user-facing density system (Settings) swaps font-size, radius, and spacing scales between small/medium/large and none/small/medium/large tiers respectively — the values below are the medium/default tier, not the only valid values
- Token-driven styling via CSS custom properties from `@clidey/ux` — extend, don't override component internals with `!important`

## Colors

The palette is Clidey's brand kit: Paper and Ink as the two surface worlds, Signal Blue as the product accent (lifts one lightness step in dark mode), Beacon Yellow as the single highlight, Clidey Orange reserved for company-branded emphasis.

### Primary
- **Signal Blue** (`#2C6BD4` light / `#5091FD` dark): Interactive affordances — buttons, links, focus rings, active states.

### Secondary
- **Clidey Orange** (`#F4781C`): Company-branded emphasis only, not general product UI. One accent per surface — don't mix with Signal Blue on the same card or banner.

### Surfaces
- **Background** (`#FAF5EC` Paper / `#0E2240` Ink): Page canvas, flat at rest.
- **Card** (`#FFFFFF` light / `#162B4A` dark): One lightness step up from background.
- **Popover** (`#FFFFFF` light / `#1D3151` dark): Menus, dropdowns, tooltips.

### Borders
- **Light mode:** `#D9D4CB`
- **Dark mode:** `#314667`

### Text
- **Foreground** (`#0E2240` light / `#FAF5EC` dark): Primary body text.
- **Muted Foreground** (`#5C6778` light / `#9BA5B3` dark): Secondary text, labels, descriptions. Always verify contrast against the actual surface the text sits on (`--card`, not just `--background`) — the dark-mode value was raised earlier after measuring under WCAG AA against `--card`.

### Named Rules
**The One-Accent Rule.** A single surface uses exactly one accent hue at a time. Blue leads product UI; orange leads company-branded moments. They do not mix on the same card or banner.

## Typography

**Primary Font:** Hanken Grotesk (with Inter, then system sans-serif fallback)
**Mono Font:** JetBrains Mono, ui-monospace, SF Mono, Menlo, monospace

**Character:** Single active UI typeface. Bricolage Grotesque is self-hosted alongside Hanken Grotesk but not wired into any component today — don't assume it renders anywhere until it's actually applied to an element.

### Hierarchy (medium density tier — the default)
- **3xl** (600, 1.875rem/30px): Rare hero text.
- **2xl** (600, 1.5rem/24px): Headline.
- **xl** (600, 1.25rem/20px): Title/section headers.
- **lg** (500, 1.125rem/18px): Card titles, dialog headers.
- **base** (400, 1rem/16px): Primary body text.
- **sm** (400, 0.875rem/14px): Secondary text, labels.
- **xs** (400, 0.75rem/12px): Metadata, captions.

### Named Rules
**The Density-Tier Rule.** Font sizes, radius, and spacing are all user-adjustable via Settings (small/medium/large tiers). Don't hardcode a pixel value where a `var(--font-size-*)`/`var(--radius-*)`/`var(--spacing-*)` token exists — hardcoding breaks the density switch for that element.

## Layout

Spacing follows a `--spacing-{xs,sm,md,lg,xl,2xl,3xl}` scale, adjustable via the same density system (compact/comfortable/spacious tiers; comfortable is the default: 8/12/16/24/32/40/48px).

## Elevation & Depth

Flat by default — no shadow at rest. Depth is a lightness step between `--background` → `--card` → `--popover`, not an elevation effect.

## Shapes

Radius follows a `--radius-{sm,md,lg,xl}` scale derived from a single `--radius` primitive (`calc()`-based: sm = radius−4px, md = radius−2px, lg = radius, xl = radius+4px), also swappable via the density system. Medium/default: `--radius: 0.625rem` (10px), giving sm 6px / md 8px / lg 10px / xl 14px.

## Components

Interactive components are sourced from `@clidey/ux`. The app extends them through CSS custom properties in `frontend/src/index.css` and the density-mapping variables in `theme-customization.ts` — never by overriding component internals with `!important`.

### Buttons
- **Shape:** `{rounded.md}`
- **Primary:** Signal Blue background, contrast-checked text.
- **Ghost:** Transparent background, foreground text.

### Cards
- **Corner Style:** `{rounded.lg}`
- **Background:** `var(--card)`, flat, no shadow at rest.

### Icon tiles
- **Pairing rule:** an icon tile's background token and its icon/text color token must be from different roles (e.g. `bg-icon` + `text-icon-foreground`, not `bg-icon` + `text-primary`) — a prior bug painted blue icon glyphs on a blue tile background because both referenced the same hue family. Always check the foreground/background pairing when introducing a new tinted tile.

## Do's and Don'ts

### Do:
- **Do** use flat, token-driven backgrounds for all static surfaces.
- **Do** use `var(--font-size-*)`, `var(--radius-*)`, and `var(--spacing-*)` tokens rather than hardcoded values, so the density-customization system keeps working.
- **Do** respect both themes equally — verify contrast against the actual surface text sits on.
- **Do** extend `@clidey/ux` via CSS custom properties, never by overriding component styles with `!important`.

### Don't:
- **Don't** mix Signal Blue and Clidey Orange on the same surface.
- **Don't** add drop shadows to cards or containers at rest.
- **Don't** assume Bricolage Grotesque renders anywhere — it's self-hosted but unused; Hanken Grotesk is the only active UI typeface.
- **Don't** pair a tinted tile's background and foreground from the same color role (see the icon-tile bug above).
- **Don't** use hardcoded color values (`bg-white`, `dark:bg-gray-800`). Use semantic tokens.
