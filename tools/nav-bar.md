# Navbar — segmented pill tabs

A frosted "pill" holding 2–4 tabs, on the left of the header (the user menu is on the right,
see `user-menu.md`). One tab is active at a time; the active tab is a lighter capsule inside
the pill.

Two interchangeable variants, **same markup, same JS** — only a few CSS values differ:

1. **Text** — pills with a label (the template default).
2. **Icons** — circles with a single glyph each. Saves horizontal space.

---

## What it looks like

**Text variant**

```
╭─────────────────────────────────────╮
│  ╭──────────╮                        │
│  │  text 1  │   text 2    text 3     │
│  ╰──────────╯                        │
╰─────────────────────────────────────╯
   ↑ active = lighter fill, full text color
                ↑ inactive = no fill, muted text (full color on hover)
```

**Icon variant**

```
╭───────────────────────╮
│  ╭───╮                 │
│  │ ◉ │   ○      ○      │
│  ╰───╯                 │
╰───────────────────────╯
```

### Design rules
- **Pill:** the surface color at 72% opacity, 1px `--line` border, fully rounded ends, frosted
  with `--glass-blur` (shared with the user menu).
- **Tabs:** transparent, `--text-muted`; hover → `--text`; active → `--text` on a 10% `--text`
  fill.
- All colors come from the theme variables in `:root`, so a new palette retints the pill.
- **One active at a time.** Transitions are quick (`.15s`) on background + color only.

---

## Markup

Tab labels come from the region files (`Tab1`…`Tab3` in `region-*.go`).

```html
<header class="header">
  <nav class="nav-tabs">
    <button class="tab active">{{.Text.Tab1}}</button>
    <button class="tab">{{.Text.Tab2}}</button>
    <button class="tab">{{.Text.Tab3}}</button>
  </nav>
  <div class="profile-area">…user menu…</div>
</header>
```

Icon variant — a glyph instead of a label, plus `aria-label` (from the region files too):

```html
<button class="tab active" aria-label="{{.Text.Tab1}}">⌂</button>
```

---

## CSS

```css
.header {
  position: relative; z-index: 10;
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
}

.nav-tabs {
  display: flex; align-items: center; min-width: 0;
  background: color-mix(in srgb, var(--surface) 72%, transparent);
  border: 1px solid var(--line);
  border-radius: 28px; padding: 5px; gap: 2px;
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}

.tab {
  height: 40px; min-width: 44px; padding: 0 18px;
  border-radius: 20px;
  color: var(--text-muted); font-size: 14px; font-weight: 600;
  white-space: nowrap;
  transition: background .15s, color .15s;
}
.tab:hover  { color: var(--text); }
.tab.active { background: color-mix(in srgb, var(--text) 10%, transparent); color: var(--text); }
```

### Icon variant

```css
.tab {
  padding: 0;
  width: 40px;          /* equal width + height = a circle */
  border-radius: 50%;
  font-size: 18px;
}
```

---

## Behaviour (JS)

Pure class toggling:

```js
document.querySelectorAll('.tab').forEach(tab => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    tab.classList.add('active');
  });
});
```

To switch content per tab, give each tab a `data-target` and show the matching panel:

```html
<button class="tab active" data-target="panel1">…</button>
<section id="panel1">…</section>
```
```js
document.querySelectorAll('.tab').forEach(tab => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    tab.classList.add('active');
    document.querySelectorAll('[id^="panel"]').forEach(p => p.hidden = true);
    document.getElementById(tab.dataset.target).hidden = false;
  });
});
```

If panels differ in height, stack them in one grid cell (`grid-area: 1 / 1`, hidden with
`visibility`) so switching tabs never shifts the page — `menu` does this.

---

## Notes

- Keep **2–4 tabs**; with long labels the pill and the avatar must still fit an iPhone 16e
  (390px wide). More than that — use the icon variant.
- Sizes: 40px tabs keep a comfortable tap target. Scale the pill by changing `height` /
  `min-width` and the radii together.
