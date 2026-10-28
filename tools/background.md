# Background — black with a light from the top

The default ground of every app on this template: flat black with a soft light falling from
the top edge, the same as the `menu` app and the README banners. There is no background image —
nothing to download, nothing to preload, nothing to route.

## The pieces (`build/web/style.css`)

The light is a theme variable, so a project can retint or replace it in `:root`:

```css
:root {
  --bg: #000000;
  --glow:
    radial-gradient(30rem 20rem at 50% -2%, rgba(255, 255, 255, .13), transparent 70%),
    radial-gradient(64rem 40rem at 50% -8%, rgba(255, 255, 255, .09), transparent 66%);
}
```

Two radial gradients centered just above the top edge: a small bright core and a wide faint
halo. Both fade to transparent, so the rest of the page is pure `--bg`.

It is painted by a fixed pseudo-element, so it stays put while the page scrolls:

```css
html { background: var(--bg); }
body { background: var(--bg); }

body::before {
  content: '';
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background: var(--glow);
}

.screen,
.login-screen { position: relative; z-index: 1; }
```

⚠️ **`z-index: 0`, not `-1`.** A negative layer is painted *under* `body`'s own background, so
the glow silently disappears. The content blocks sit at `z-index: 1` above it instead.

`pointer-events: none` keeps the layer from swallowing clicks.

## Retinting

- Warmer or colored light: change the `rgba(...)` colors in `--glow` (keep the alphas low,
  ≈ .08–.15 — it should read as light, not as a shape).
- No light at all: `--glow: none;`.
- A different ground: change `--bg`. The frosted chrome (tab pill, avatar, user menu) is tinted
  from the theme variables and follows.
