# Photo Audit Design System

A design system for **Photo Audit** — a multi-tenant AI content moderation platform supporting photos, short videos, and live streams. This library defines the visual language, tokens, and component patterns used across the review workstation, appeal management, quality auditing, and tenant configuration modules. It is built on Ant Design Pro conventions and extends them with semantic layering for risk-aware interfaces.

---

## 1. Overview

The Photo Audit Design System serves a professional, data-heavy workflow:审核员需要快速、准确地判断内容风险。设计语言以冷静、精确、可信赖为核心，避免任何分散注意力的视觉噪音。系统采用三层 token 架构（品牌色阶 → 语义别名 → 可移植别名），确保颜色在不同上下文中的含义一致。

---

## 2. Content Fundamentals

### Voice & Tone

Professional, neutral, data-focused Chinese UI copy. Prioritize clarity over friendliness. Every label, status indicator, and error message should communicate information density efficiently.

**Concrete copy examples:**
- 审核工作台 — functional, not "Review Hub"
- 内容管理 — straightforward noun phrase
- 申诉处理 — action-oriented
- 质量抽检 — domain terminology
- 租户配置 — administrative context

**When generating copy:** Use clear action verbs (通过 / 打回 / 提交), precise status indicators (已通过 / 审核中 / 已驳回), and standardized risk levels (低风险 / 中等风险 / 高风险). Avoid adjectives that introduce ambiguity.

---

## 3. Visual Foundations

### Color

The primary color is a confident blue (`#2563eb`) drawn from the Tailwind palette. The full scale spans ten steps from `#eff6ff` (50) to `#172554` (900), with `#3b82f6` (500) as the mid-tone and `#2563eb` (600) as the interactive default. Five semantic palettes — success (`#059669`), warning (`#d97706`), error (`#dc2626`), info (`#0284c7`), and accent (`#a21caf`) — each follow the same ten-step structure. Neutrals use a slate-toned text/surface scale (`#0f172a` through `#f8fafc`) with distinct but harmonious undertones.

Surface layers employ a seven-tier container hierarchy (`surface-container-lowest` through `highest`), enabling clear depth signaling without relying exclusively on shadow. Interactive states use a consistent opacity overlay pattern: hover at 8%, focus at 12%, press/drag at 16%, all tinted to the primary blue. Dark mode inverts the palette with a GitHub-inspired dark background (`#0d1117`) and lighter accents (`#58a6ff`).

### Typography

Three font families define the typographic system:

- **Display & Headings:** `Inter` — used for all heading levels (h1–display), numbers, and the eyebrow class. Weights span 300–700.
- **Body:** `Inter` with `'Noto Sans SC'` fallback — handles Chinese CJK text alongside Latin. Same weight range.
- **Monospace:** `JetBrains Mono` with `'Courier New'` fallback — reserved for risk scores, timestamps, technical identifiers, and code-like content.

The scale has nine stops: display (56px, 700, 1.1), h1 (40px), h2 (32px), h3 (24px), h4 (20px), body (16px, 400, 1.6), lead (18px, 1.7), caption (12px), eyebrow (11px, 600, uppercase, 0.08em tracking). Line heights tighten for headings and loosen for body text to optimize readability in dense review tables.

### Spacing

A 4px base grid governs all spacing. Eight tokens (`--space-1` through `--space-8`) cover 4px to 64px in 4px increments. Component dimensions follow suit: button heights are 32px (sm), 40px (md), 48px (lg); input height is fixed at 40px; sidebar width is 240px; navbar height is 56px. Gutter sizes are 16px (sm), 24px (md), 32px (lg). Max content width is 1440px.

### Border Radius

Seven radius tokens support a progressive softening pattern: `xs` (2px) for tight tags and badges, `sm` (4px) for inputs and small cards, `md` (8px) as the default for content cards and buttons, `lg` (12px) for elevated surfaces and modals, `xl` (16px) for large containers, `2xl` (24px) for prominent feature areas, and `full` (9999px) for pills and avatars.

### Shadow & Elevation

Five elevation levels provide depth without heaviness:

1. **Card** (`--shadow-1`): Subtle lift for content cards and table rows — `0 1px 2px` at 6% opacity.
2. **Card Hover** (`--shadow-2`): Interactive feedback on hover — `0 4px 8px` at 10%.
3. **Float** (`--shadow-3`): Dropdown menus, tooltips, floating panels — `0 8px 24px` at 18%.
4. **Modal** (`--shadow-4`): Dialog overlays — `0 16px 40px` at 24%.
5. **Overlay** (`--shadow-5`): Full-screen overlays and backdrops — `0 24px 60px` at 30%.

Dark mode shadows increase in intensity to compensate for lower contrast environments.

### Motion

Three durations (fast: 150ms, normal: 250ms, slow: 400ms) paired with three easing curves (`ease`, `out`, `in`) govern transitions. Hover states and micro-interactions use fast; page transitions use normal; modal open/close use slow.

### Borders & Backgrounds

Default border uses `--rule` (`--audit-surface-200`, `#e2e8f0`) at 1px thickness. Outline variants use `--audit-text-100`. Ring color for focus states is `--audit-primary-600` (`#2563eb`). Background layers distinguish between `surface-container-lowest` (pure white `#ffffff`), `surface-container-low` (`#f8fafc`), and progressively darker containers for nested content.

---

## 4. Component Patterns

| Component | File | Key Insight |
|-----------|------|-------------|
| Button | `component-button.html` | Primary/ghost/danger variants mapped to semantic tokens; 3 height tiers (32/40/48px) |
| Card | `component-card.html` | 5-level shadow hierarchy; hover elevation change; risk badge integration |
| Table | `component-table.html` | Row hover states; sticky header; alternating row backgrounds via surface tokens |
| Form | `component-form.html` | 40px input height; focus ring uses primary token; validation error borders use error token |
| Navigation | `component-navigation.html` | 56px navbar; 240px sidebar; active state uses primary container background |
| Badge | `component-badge.html` | 20px icon badge; risk-level color mapping; pill shape via radius-full |

---

## 5. Index

| File | Description |
|------|-------------|
| `colors_and_type.css` | All design tokens: colors, typography, spacing, radius, shadows, motion, layout |
| `components.css` | Shared component-level CSS overrides |
| `preview/component-button.html` | Button variant preview |
| `preview/component-card.html` | Card component preview |
| `preview/component-table.html` | Table component preview |
| `preview/component-form.html` | Form controls preview |
| `preview/component-navigation.html` | Navbar + sidebar navigation preview |
| `preview/component-badge.html` | Badge and tag preview |

---

## 6. Caveats / Known Substitutions

- **Font fallback chain:** `Inter` is loaded via Google Fonts CDN. If the CDN is unreachable, Chinese text falls back to `Noto Sans SC`; Latin text falls back to `sans-serif`. `JetBrains Mono` falls back to `Courier New`.
- **CDN dependency:** The `@import` for Google Fonts is unconditional — offline environments need a self-hosted font copy.
- **Color scale generation:** All intermediate shades (50–900) are AI-generated interpolations, not measured color science. Verify contrast ratios for WCAG compliance on critical risk indicators.
- **Dark mode manual override:** Dark mode tokens are hardcoded overrides rather than pure inversions of light mode — the palette shift is intentional (GitHub-dark inspired) but means `dark` mode is not a 1:1 inverse transformation.
- **No icon set defined:** Iconography is assumed to come from Ant Design's built-in icons or an external library; no icon token system exists in this library.
- **`--size-badge` at 20px:** This dimension is defined but not referenced by any component preview. Treat as reserved/future-use.
