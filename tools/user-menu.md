# User Menu — reusable spec

The avatar → dropdown "user menu" in the top-right corner of every sh-development app. Focus is
on **structure, the buttons and what they do**. Colors come from the project's theme variables,
so they change from project to project; two things do not: **the frosted blur**, and **log out
is red**.

This matches the live implementation in `auth-client` (server-rendered Go template,
`build/web/index.html` + `style.css` + `script.js`). If your app is a JS SPA, see
"Client-rendered variant" at the end.

## What it is

A circular avatar button in the header (to the right of the tab pill). Clicking it toggles a
popover anchored to the button. The popover has two parts:

1. **Identity block** (read-only) — provider label, display name, permanent user id.
2. **Action list** — one rounded box, rows divided by hairlines: **apps**, **readme**,
   **log out**. These three are the minimum set and are always present.

```
┌──────────────────────────────┐
│ YANDEX                        │  ← provider (uppercase, muted)
│ Сергей Шумилов                │  ← display name (falls back to the id)
│ 1130000012345678              │  ← permanent id — only when there is a name
│ ┌──────────────────────────┐  │
│ │ apps                     │  │  ← delegates to the menu app
│ ├──────────────────────────┤  │  ← hairline divider between rows
│ │ readme                   │  │  ← external link, new tab
│ ├──────────────────────────┤  │
│ │ log out                  │  │  ← RED, clears the session
│ └──────────────────────────┘  │
└──────────────────────────────┘
```

No "member since" / "last login" rows — identity + actions only.

## Structure (HTML)

`{{$name}}` is the user's name, or the id when there is none (set once at the top of the
logged-in branch). Labels come from the region files (`{{.Text.Apps}}` …), the readme URL from
`readmeURL` in `main.go`.

```html
{{$name := .User.Name}}{{if not $name}}{{$name = .User.AuthID}}{{end}}

<div class="profile-area">
  <!-- avatar trigger: JS sets the initial from data-name -->
  <button class="profile-btn" id="profileBtn" data-name="{{$name}}" aria-label="Account menu">
    <span class="profile-initial">?</span>
  </button>

  <div class="profile-popover" id="profilePopover">
    <div class="pi-info">
      <span class="pi-label">{{.User.Method}}</span>
      <span class="pi-name">{{$name}}</span>
      {{if .User.Name}}<span class="pi-sub">{{.User.AuthID}}</span>{{end}}
    </div>
    <div class="pi-list">
      <a class="pi-action" href="/apps">{{.Text.Apps}}</a>
      <a class="pi-action" href="{{.ReadmeURL}}" target="_blank" rel="noopener">{{.Text.Readme}}</a>
      <a class="pi-action pi-logout" href="/logout">{{.Text.Logout}}</a>
    </div>
  </div>
</div>
```

- Trigger and popover share a positioned wrapper (`.profile-area`) so the popover anchors to
  the button.
- `.pi-list` is what makes the grouped-list look.
- Rows can be `<a>` or `<button>`; they share `.pi-action`.

## The buttons — what each one does

| Button | Element | Action |
|---|---|---|
| **apps** | `<a href="/apps">` | The server delegates the logged-in user to the central **menu app** and redirects there (see "Delegation"). |
| **readme** | `<a target="_blank">` | Opens the project's repo/README in a new tab. Every app has it; the URL is `readmeURL` in `main.go`. |
| **log out** | `<a href="/logout">` | Clears the session cookie and returns to the login screen. **Red**: the destructive action. |

## Behavior (JS, `script.js`)

```js
const profileBtn = document.getElementById('profileBtn');
if (profileBtn) {
  const name    = profileBtn.dataset.name || '';
  const initial = profileBtn.querySelector('.profile-initial');
  if (initial) initial.textContent = (name[0] || '?').toUpperCase();

  const popover = document.getElementById('profilePopover');
  profileBtn.addEventListener('click', () => {
    const open = popover.classList.toggle('open');
    profileBtn.classList.toggle('open', open);
  });
  document.addEventListener('click', e => {
    if (!profileBtn.contains(e.target) && !popover.contains(e.target)) {
      popover.classList.remove('open');
      profileBtn.classList.remove('open');
    }
  });
}
```

Open/close is an `.open` class on both the popover and the trigger.

## Styling

Everything is built from the theme variables in `:root` — a new palette retints the menu with
no edits here. The blur is shared with the tab pill through `--glass-blur`.

```css
:root {
  --bg: …; --surface-alt: …; --line: …; --text: …; --text-muted: …;
  --bad: #f87171;                          /* log out */
  --glass-blur: blur(24px) saturate(1.4);  /* the frosted chrome */
}

.profile-area { position: relative; flex: 0 0 auto; }

.profile-btn {
  width: 48px; height: 48px; border-radius: 50%;
  background: color-mix(in srgb, var(--text) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--text) 45%, transparent);
  color: var(--text); font-size: 18px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  transition: border-color .15s, color .15s;
}
.profile-btn:hover,
.profile-btn.open { border-color: var(--text); }

.profile-popover {
  position: absolute; top: calc(100% + 10px); right: 0;
  width: 256px; max-width: calc(100vw - 2 * var(--pad));
  padding: 14px; z-index: 101;
  display: none; flex-direction: column; gap: 10px;
  background: color-mix(in srgb, var(--bg) 85%, transparent);
  border: 1px solid var(--line); border-radius: 16px;
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}
.profile-popover.open { display: flex; }

.pi-info  { display: flex; flex-direction: column; gap: 3px; }
.pi-label { font-size: 10px; color: var(--text-muted); letter-spacing: .1em; text-transform: uppercase; }
.pi-name  { color: var(--text); word-break: break-all; }
.pi-sub   { font-size: 11px; color: var(--text-muted); word-break: break-all; }

/* one rounded box, rows divided by hairlines */
.pi-list {
  display: flex; flex-direction: column;
  border: 1px solid var(--line); border-radius: 12px; overflow: hidden;
}
.pi-action {
  display: block; width: 100%; padding: 12px 14px; text-align: left;
  color: var(--text); text-decoration: none; font-size: 14px;
  transition: background .15s, color .15s;
}
.pi-action + .pi-action { border-top: 1px solid var(--line); }
.pi-action:hover { background: var(--surface-alt); }

.pi-logout       { color: var(--bad); }
.pi-logout:hover { color: color-mix(in srgb, var(--bad) 80%, white);
                   background: color-mix(in srgb, var(--bad) 16%, transparent); }
```

The grouped-list recipe in three rules:
1. Wrap rows in a container with a border + `border-radius` + `overflow: hidden`.
2. Give each row no border of its own, only padding.
3. Add a top border to every row **except the first** (`.pi-action + .pi-action`).

## Delegation (the "apps" button)

`GET /apps` (`handleOpenApps` in `auth-server.go`):

1. Loads the user's permanent id, provider and stored name.
2. `POST {AUTH_INTERNAL}/delegate`:
   ```json
   {
     "user_id":    "1130000012345678",
     "app_token":  "<this app's APP_TOKEN>",
     "method":     "yandex",
     "name":       "Сергей Шумилов",
     "first_name": "Сергей Шумилов"
   }
   ```
3. Redirects the browser to `https://menu.<domain>/?code=<code>` — `domain` comes from the
   region, because SSO works only within one zone.

auth-center is **stateless**: the receiver gets only what was sent. So:

- Always send the real `method`, otherwise the receiver shows `delegate`.
- Send the stored name in **both** `name` and `first_name`. This template reads `name` for
  every provider; older apps read `name` only for Google and `first_name` otherwise.
- **Never send a placeholder** when the name is empty (`"no name"`, the id, …). The receiver
  stores it as the user's real name, never overwrites a non-empty name, and forwards it further.
  Send nothing instead — the receiver then shows the id.

## Checklist to rebuild in a new app

- [ ] Avatar + popover in a `position: relative` wrapper (`.profile-area`), right side of the header.
- [ ] Avatar shows the first letter of the name (JS reads `data-name`).
- [ ] `.open` toggled on popover and trigger; outside click closes.
- [ ] Identity block: provider, name (or id), id under the name only when there is a name.
- [ ] Rows: **apps**, **readme**, **log out**, in that order, always all three.
- [ ] Colors only from theme variables; frosted with `--glass-blur`; log out uses `--bad`.
- [ ] `/apps` forwards id + token + method + name to `/delegate` and redirects to `menu.<domain>`.

## Client-rendered variant (JS SPA)

Populate the identity block from a session endpoint instead of template fields. Everything else
(classes, CSS, toggle JS) is identical.

```js
const user = await (await fetch('/me', { credentials: 'include' })).json();
profileBtn.dataset.name                         = user.name || user.id;
document.querySelector('.pi-label').textContent = user.method || '';
document.querySelector('.pi-name').textContent  = user.name || user.id;
document.querySelector('.pi-sub').textContent   = user.name ? user.id : '';
```
