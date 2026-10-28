# Template Rules

Rules for building apps on top of this template.

---

## Forking checklist

1. `APP` — the repo-level GitHub variable (locally `APP=` in `.env`). It is the app's name in
   the UI, the binary, `/opt/$APP`, the systemd unit and `<APP>.db`.
2. `build/main.go` — `defaultAppName` (fallback when `APP` is unset) and `readmeURL`.
3. `Dockerfile`, `prod-compose.yml`, `build/.air.toml` — the binary name `auth-client`.
4. `build/region-ru.go`, `build/region-com.go` — `About`, the tabs, every string.
5. `build/web/favicon.svg` + `apple-touch-icon.png` — the hazard-stripe icon is a placeholder on
   purpose. The PNG is 180×180, full-bleed, no rounded corners (iOS rounds it itself).
6. `build/web/style.css` `:root` — the project's palette.
7. GitHub environments `production-ru` / `production-com` (see `tech.md`).

---

## File Structure

```
build/
  main.go        — APP / DB path / env, routes, handlers                  ← edit
  region.go      — region types + initRegion                                ← edit (new strings)
  region-ru.go, region-com.go — per-zone domain and texts                   ← edit
  app_db.go      — app-specific migrations                                  ← edit
  auth-human.go  — /login, /logout, handleCallback, JWT, extractName        ← shared infra
  auth-server.go — delegateCode, /apps → menu                               ← shared infra
  db.go          — SQLite init, users table, core queries                   ← shared infra
  web/
    index.html   — Go template: {{if .User}} app / {{else}} login           ← edit
    style.css    — VARIABLES · BASE · LAYOUT · HEADER · USER MENU · LOGIN · APP · ERROR
    script.js    — profile popover, tabs, then the App logic section        ← edit App logic
    favicon.svg, apple-touch-icon.png — placeholder icon                    ← replace per app
bin/
  deploy.sh, backup.sh — identical in every app (reference: the menu repo)
```

Shared chrome (header, user menu, login screen) and your app code live in the same three web
files, separated by labeled sections — build inside the `APP` section of `style.css` and the
`App logic` section of `script.js`. The shared-infra Go files are where fixes to the auth chrome
originate: change them here first, then copy into the apps.

**`app_db.go`** is the entry point for app-specific database work. Add tables in
`appMigrate()`; it runs on startup after the core `users` table is ready.

**`main.go`** is where you add app routes. Auth routes (`/login`, `/logout`, `/apps`) and the
request logger are already wired up — do not duplicate them.

---

## UI rules

- **Name.** The app's name comes from `APP` only: `<title>` (lowercase, as written) and the login
  card (`{{.App}}`). Never type it into the template.
- **`<head>`** holds the title, the icons and the CSS — no Open Graph / preview meta.
- **Colors** live in `:root` of `style.css` and nowhere else. The chrome (tab pill, avatar, user
  menu) is tinted from them, so a new palette retints it. Log out is always `--bad` (red).
- **Ground**: black with a soft light from the top (`--glow`), see `background.md`.
- **No needless scroll.** A page is never taller than the screen unless its content needs it:
  `.screen` is `min-height: 100dvh`, `.app-content` takes what the header leaves.
- **Login page** is exactly one screen (`100dvh`, `overflow: hidden`): the card centered in the
  top zone, the description and the divider anchored at the bottom. It is the same in every app.
- **User menu** always has **apps**, **readme**, **log out** — see `user-menu.md`.
- **Strings** come from the region files: `{{.Text.X}}` in the template, `TEXT.x` in `script.js`.

---

## Auth Flow

```
GET /login
  → redirect to auth-center
  → auth-center authenticates the user
  → redirect back to /?code=

GET /?code=...  (handleCallback in auth-human.go)
  → POST /exchange to auth-center (server-side)
  → upsertUser in DB
  → set JWT cookie
  → redirect to /
```

Use `sessionUserID(r)` anywhere to get the current user's internal DB id (`0` if not logged in).
Use `requireAuth(handler)` to protect routes that need a logged-in user.

**Names.** Google, Yandex and every delegate send `name`; Telegram and Solana send
`first_name`/`last_name`. `extractName` reads `name` for every method and falls back to the
pair, so a provider added to auth-center later does not land with an empty name. `upsertUser`
never overwrites a stored name with an empty one.

---

## Database

The `users` table is created automatically:

| Column | Description |
|---|---|
| `id` | Internal primary key — use this everywhere in your app |
| `auth_id` | Permanent ID from the provider |
| `method` | `google`, `yandex`, `telegram` or `solana` |
| `name` | Display name |
| `created_at` | First login |
| `last_login` | Updated on every login |

Add app tables in `app_db.go`. Always reference `users.id` as the foreign key — never `auth_id`.

The database file is always `<APP>.db`. Only its directory is configurable (`DB_DIR`), so a
config copied from another app cannot make this one write into that app's database. Prod leaves
`DB_DIR` unset: with `WorkingDirectory=/opt/$APP` the DB is `/opt/$APP/$APP.db`, exactly what
`deploy.sh` and `backup.sh` back up. The start line confirms it:

```
start app=qcode db=qcode.db port=8890
```

Renaming `APP` does **not** migrate anything: the app opens a different file and comes up
empty. Stop the service, `mv` the `.db` (plus `-wal` / `-shm`), then change the variable.

New columns on an existing DB need an `ALTER TABLE` fallback:
```go
db.Exec(`ALTER TABLE users ADD COLUMN my_col TEXT`) // nolint:errcheck — ok if column exists
```

---

## Logging

Every meaningful user action produces exactly one log line, `key=value`:

```
login  uid=1 method=yandex name="Ivan" new=true
logout uid=1
action uid=1 item_id=42 result=ok
action uid=1 error="not found"
```

- One line per action — not per code step
- Always include `uid=` when a user is involved
- Always log errors, including unexpected DB errors
- Never log secrets, tokens, or raw codes

The request logger in `main.go` covers HTTP-level logging (`GET / 200 3ms`) — do not add
per-route request logging on top of it.

---

## Environment Variables

See the table in `tech.md`. Copy `.env.example` to `.env` for local dev. Never commit `.env`.

---

## Cross-app login (delegate)

Lets a user open another app without logging in again. The receiving app needs no changes — its
`/?code=` handling already works. The **apps** button (`GET /apps`) is the built-in case: it
delegates to `https://menu.<domain>/`. To hand off to another app, add a handler the same way:

```go
func handleOpenApp2(w http.ResponseWriter, r *http.Request) {
    uid := sessionUserID(r)
    if uid == 0 {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }
    code, err := delegateCode(uid)
    if err != nil {
        log.Printf("open-app2 uid=%d error=%v", uid, err)
        http.Error(w, "could not open app", http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "https://app2."+domain+"/?code="+code, http.StatusFound)
}
```

```go
mux.HandleFunc("GET /open-app2", handleOpenApp2)
```

`delegateCode` forwards the user's id, provider and stored name (in both `name` and
`first_name`), so the receiving app shows the real name. An empty name is not sent and no
placeholder is invented — a fake name would be stored by the receiver as the real one.

Requirements: both apps have a valid `APP_TOKEN` in auth-center's `APP_TOKENS`; App1 does not
need App2's token. SSO works only within one zone, so build the target from `domain`.

---

## Build & Deploy

See `tech.md` (commands, GitHub Actions deploy, nightly backup) and `docker.md` (how the Docker
setup works).

Logs in production go to journalctl:
```bash
journalctl -u <app> -f
```
