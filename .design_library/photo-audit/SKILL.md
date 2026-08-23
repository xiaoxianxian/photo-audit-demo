---
name: photo-audit-design
description: Use this skill to generate well-branded interfaces for Photo Audit. Contains colors, type, fonts, assets, and UI kit for prototyping dashboard UIs.
user-invocable: true
---
# Photo Audit Design Skill

Read the `README.md` file within this skill, and explore the other available files.

If creating visual artifacts, copy assets out and create static HTML files. If working on production code, read the rules here to become an expert in designing with this brand.

## Quick map

- `README.md` — brand context, content fundamentals, visual foundations (read first)
- `colors_and_type.css` — drop-in CSS variables for colors, type, radius, shadow, spacing
- `css.json` — structured token understanding source
- `components/index.json` — component index + cross-component patterns
- `preview/` — small HTML cards illustrating foundations and components
- `uikit-plan.json` — component whitelist and UIKit planner output
- `library-consumption.json` — recommended downstream read order

## Essentials at a glance

- Primary color `#2563eb` conveys trust and professionalism for a moderation dashboard
- Border radius spans xs(2px) through 2xl(24px); default to md(8px) for cards, sm(4px) for controls
- Control height is 40px (md); compact actions use 32px (sm); large CTAs use 48px (lg)
- Font faces: Inter (Latin/display), Noto Sans SC (CJK body), JetBrains Mono (code)
- Voice is direct and neutral; risk labels use Low/Medium/High without euphemism
- Shadow philosophy: 5 elevation levels from Card (shadow-1) to Overlay (shadow-5); prefer subtle depth over heavy borders
- Dispute items get a 2px solid #fa8c16 border on review cards to signal AI裁判 divergence

## Components

| Slug | Name | Key Insight |
|------|------|-------------|
| button | 按钮 | Primary for approve, danger for reject actions |
| card | 卡片 | Review cards with risk score badges and AI labels |
| table | 表格 | Status badges, risk columns, batch action toolbar |
| form | 表单 | Vertical layout with inline validation |
| navigation | 导航 | Collapsible sidebar with route sync |
| badge | 徽章 | Risk-level color coding for moderation states |
