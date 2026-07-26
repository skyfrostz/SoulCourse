# SoulCourse Design System

## Direction

SoulCourse uses a quiet, information-first interface inspired by Apple system UI and focused productivity products. Color identifies subjects, content types, and state; it does not decorate empty space. Forum content remains the dominant visual element.

## Color

| Token | Value | Use |
| --- | --- | --- |
| `--color-canvas` | `#f5f5f7` | Page background |
| `--color-surface` | `#ffffff` | Cards, dialogs, inputs |
| `--color-ink` | `#1d1d1f` | Headings and primary actions |
| `--color-body` | `#424245` | Body copy |
| `--color-muted` | `#6e6e73` | Secondary labels |
| `--color-subtle` | `#86868b` | Metadata |
| `--color-line` | `#d2d2d7` | Strong borders |
| `--color-line-soft` | `#e5e5ea` | Dividers and card borders |
| `--color-blue` | `#0071e3` | Links and focus |
| `--color-green` | `#1f8a70` | Active learning state |

The hello spectrum uses cyan, green, yellow, orange, red, and indigo as small solid signals. Never use the full spectrum as a page background or large gradient.

## Typography

- UI stack: Apple system fonts, then PingFang SC and Helvetica/Arial fallbacks.
- Page title: 28px desktop, 24px mobile, weight 800.
- Card title: 20px desktop, 18px mobile, weight 850.
- Body: 14px with 1.7-1.75 line height.
- Metadata: 12px; control labels: 13-14px.
- Letter spacing stays at `0`.

## Shape And Depth

- Repeated cards and controls: maximum 8px radius.
- Compact chips: 6px radius or a pill only for status/count semantics.
- Sidebars and page sections remain unframed.
- Shadows are reserved for hover feedback, dialogs, and popovers.

## Interaction

- All keyboard-focusable controls use a visible blue focus ring.
- Stateful buttons expose `aria-pressed`; expandable filters expose `aria-expanded`.
- Mobile touch targets are at least 44px where actions sit in dense toolbars.
- Motion uses the shared 160ms/260ms timings and respects `prefers-reduced-motion`.

## Responsive Layout

- Desktop: filter rail, 760px reading column, context rail.
- Tablet: filter rail plus reading column; context rail is hidden.
- Mobile: single reading column with a collapsed filter control before the feed.
