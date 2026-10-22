# BlockAds Desktop — Design System Specification

## 1. Design Direction: Modern Neo-Brutalism (Restrained Desktop)

BlockAds Desktop employs a **modern, restrained neo-brutalist aesthetic** engineered specifically for a technical Windows desktop utility. Unlike playful web brutalism, this design direction conveys **structural honesty, security, engineering discipline, and operational transparency**.

### Core Tenets
1. **Structural Truth**: High-contrast, sharp boundaries (`2px solid #121212`) reflect crisp network packet boundaries and deterministic filtering logic.
2. **Honest Telemetry**: No fabricated graphs or deceptive animations. Sample/mock data is visibly demarcated with dedicated badges.
3. **Ergonomic Density**: Desktop-first information density without clutter. Generous click targets (minimum 38px height for primary controls) combined with monospaced data tables for rapid scanning.
4. **Zero AI Slop**: No generic purple/blue gradients, no borderless floating cards, no decorative unclickable pills, and no ambiguous icon buttons without accessible labels.

---

## 2. Color Palette & Tokens

### Base & Backgrounds
| Token Name | Hex Code | Purpose / Usage |
| :--- | :--- | :--- |
| `--surface-bg` | `#F5F1E8` | Warm ivory / canvas background for entire app window |
| `--surface-card` | `#FFFFFF` | Primary card, container, and dialog background |
| `--surface-subtle` | `#EBE7DE` | Secondary container, table headers, inactive tabs |
| `--surface-hover` | `#DFDACF` | Hover state for secondary surfaces |
| `--surface-dark` | `#121212` | Sidebar background, primary button background |

### Text & Borders
| Token Name | Hex Code | Purpose / Usage |
| :--- | :--- | :--- |
| `--text-primary` | `#121212` | Highest contrast text, headings, body text (Contrast > 14:1) |
| `--text-secondary` | `#4B5563` | Subtitles, helper text, table column labels |
| `--text-muted` | `#6B7280` | Metadata, timestamps, disabled indicators |
| `--text-on-dark` | `#F5F1E8` | Primary text when resting on dark surfaces |
| `--border-strong` | `#121212` | Universal element border (`2px solid #121212`) |
| `--border-subtle` | `#D1CBC0` | Secondary divider lines within cards |

### Brand & Accents
| Token Name | Hex Code | Purpose / Usage |
| :--- | :--- | :--- |
| `--accent-primary` | `#B38B6D` | Muted warm ochre/brown — key interactive accents |
| `--accent-hover` | `#9C775A` | Darkened accent on hover/press |
| `--accent-soft` | `#F2ECE4` | Tinted background for accent callouts |

### Semantic Status
| Status | Hex Code | Badge Background | Meaning / Usage |
| :--- | :--- | :--- | :--- |
| **Blocked / Danger** | `#D9381E` | `#FEECEB` | Blocked DNS queries, blocklist rules, critical errors |
| **Allowed / Success** | `#2B8A3E` | `#EBFBEE` | Allowed DNS queries, active protection, passing probes |
| **Warning / Notice** | `#D97706` | `#FEF3C7` | Sample data warnings, port conflicts, planned features |
| **Inactive / Neutral** | `#4B5563` | `#EBE7DE` | Milestone 1 disabled engine state, standby status |

---

## 3. Typography & Hierarchy

- **Primary Font Family**: Clean system-ui sans-serif (`Segoe UI`, `-apple-system`, `Inter`, `Roboto`, `sans-serif`)
- **Code & Network Records**: Monospace (`Consolas`, `Cascadia Code`, `Fira Code`, `monospace`)

### Type Scale
- **H1 / Screen Title**: `20px` (`1.25rem`), font-weight 700, letter-spacing `-0.02em`.
- **H2 / Section Title**: `15px` (`0.9375rem`), font-weight 700, uppercase with slight letter-spacing.
- **H3 / Card Header**: `13px` (`0.8125rem`), font-weight 600.
- **Body / Standard**: `13px` (`0.8125rem`), font-weight 500, line-height 1.5.
- **Code / Monospace**: `12px` (`0.75rem`), font-weight 600, tabular numbers.
- **Caption / Meta**: `11px` (`0.6875rem`), font-weight 500.

---

## 4. Spacing, Borders, and Elevation

### Border Standards
- **Standard Border**: `2px solid #121212`
- **Corner Radius**: `0px` or `2px` (sharp, geometric, intentional neo-brutalism)
- **Dividers**: `1px solid #121212` or `1px solid #D1CBC0`

### Neo-Brutalist Hard Shadows
- **Card Shadow**: `3px 3px 0px #121212`
- **Button Normal**: `2px 2px 0px #121212`
- **Button Active (Pressed)**: `0px 0px 0px #121212` (with `transform: translate(2px, 2px)`)
- **Modal / Floating**: `5px 5px 0px #121212`

---

## 5. Interaction States & Controls

### Buttons
- **Primary**: Background `#121212`, text `#F5F1E8`, border `2px solid #121212`, shadow `2px 2px 0px #121212`.
- **Secondary**: Background `#FFFFFF`, text `#121212`, border `2px solid #121212`, shadow `2px 2px 0px #121212`.
- **Accent**: Background `#B38B6D`, text `#FFFFFF`, border `2px solid #121212`, shadow `2px 2px 0px #121212`.
- **Destructive**: Background `#D9381E`, text `#FFFFFF`, border `2px solid #121212`, shadow `2px 2px 0px #121212`.
- **Disabled**: Background `#E5E7EB`, text `#9CA3AF`, border `2px solid #9CA3AF`, cursor `not-allowed`, no shadow.

### Inputs & Selects
- Background `#FFFFFF`, border `2px solid #121212`, padding `8px 12px`, font-size `13px`.
- Focus state: Outline `none`, shadow `3px 3px 0px #B38B6D`.

---

## 6. Accessibility (a11y) Rules

1. **Contrast Ratio**: Body text and button labels achieve at least `7:1` against their backgrounds (well exceeding WCAG AAA standards).
2. **Keyboard Focus**: Explicit keyboard outline using high-visibility offset indicator on all interactive controls (`:focus-visible`).
3. **ARIA Semantics**:
   - Status indicators have explicit `role="status"` and accessible text.
   - Tables use semantic `<thead>`, `<tbody>`, `<th>`, and `<td>` tags.
   - Dialog modals use `role="dialog"` with `aria-modal="true"` and `aria-labelledby`.
4. **No Color-Only Signaling**: Every status pill includes both an explicit text label and a geometric or distinct icon in addition to its color.
