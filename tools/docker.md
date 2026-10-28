# Docker setup

How the Docker setup of this template works. Docker does two jobs here — dev hot-reload and the
release build. Production itself runs no Docker: the binary is installed as a systemd unit by
`bin/deploy.sh` (see `tech.md`).

1. **Dev**: hot-reload (Air), source mounted, native arch.
2. **Release**: a static linux/amd64 binary, cross-compiled, committed to `bin/`.
3. **One multi-stage Dockerfile** for both; each compose file picks a stage with `target:`.
4. **`--version`** prints the app name and the build time baked in at link time.

---

## File map

| File | Role |
|------|------|
| `Dockerfile` | Multi-stage: `dev` (Air) + `builder` (static binary) |
| `dev-compose.yml` | Runs the `dev` stage: source mount, DB volume, module cache |
| `prod-compose.yml` | Runs the `builder` stage, copies the binary to `./bin`, prints a banner |
| `build/.air.toml` | What Air watches and how it rebuilds |
| `.env` / `.env.example` | Runtime env for dev (`env_file`) |

---

## 1) Dev

```bash
docker-compose -f dev-compose.yml up --remove-orphans
# http://localhost:8890
```

```dockerfile
FROM golang:1.23-bookworm AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@v1.52.3
ENV GONOSUMDB=* GOFLAGS=-mod=mod
EXPOSE 8890
CMD ["air"]
```

- No `--platform`: the dev stage runs as the host's native arch, no emulation.
- No `COPY` — the source comes in through the bind mount.

```yaml
services:
  client:
    build:
      context: .
      target: dev
    ports:
      - "8890:8890"
    env_file: .env
    environment:
      DB_DIR: /data
    volumes:
      - ./build:/app
      - db-data:/data
      - go-cache:/root/go/pkg/mod

volumes:
  go-cache:
  db-data:
```

- `./build:/app` — source mount: edit on the host, Air rebuilds in the container.
- `db-data:/data` + `DB_DIR=/data` — the SQLite file lives on a **named volume**, not on the
  bind mount: modernc sqlite locking fails on the macOS bind mount (`CANTOPEN`). The file is
  still `<APP>.db` — only the directory is set.
- `go-cache` — module cache between runs.

`build/.air.toml` watches `go, js, css, html, svg`: the web files are `//go:embed`-ed, so a
frontend edit needs a rebuild too. It builds into `/tmp/airbuild`, outside the mounted
directory, so the binary never lands on the host or retriggers the watcher.

---

## 2) Release binary

```bash
docker-compose -f prod-compose.yml build release      # required after any code change
docker-compose -f prod-compose.yml run --rm release   # → bin/auth-client
```

A bare `run` reuses the existing image and silently ships a **stale binary** — always `build`
first and check that the `Built:` time in the banner is now.

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY build/ .
RUN go mod tidy && \
    BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.buildTime=${BUILD_TIME}" \
    -o /auth-client .
```

- `CGO_ENABLED=0` → fully static, no libc dependency. Requires pure-Go deps
  (`modernc.org/sqlite`; `mattn/go-sqlite3` would break it).
- `GOOS=linux GOARCH=amd64` → cross-compiles for the servers from any host.
- `-s -w` → strips debug tables.

`prod-compose.yml` runs the stage once: copies `/auth-client` to the host's `./bin`, prints a
green banner with the build time from `--version`, exits.

---

## 3) `--version`

`-X main.buildTime=…` sets a string variable at link time. `main.go` declares it with a fallback
and prints it before any startup work:

```go
var buildTime = "unknown"

if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "--info") {
    fmt.Printf("%s built: %s\n", appName, buildTime)
    os.Exit(0)
}
```

`deploy.sh` calls `$BIN --version` after a successful deploy. The same `buildTime` is the
cache-bust token of the static files (`?v={{.Version}}`), so every deploy refreshes them.

---

## Forking

The binary name `auth-client` appears in `Dockerfile` (`-o /auth-client`), `prod-compose.yml`
(the copy and the banner) and `build/.air.toml`. Rename it to the new `APP` in all three —
`deploy.sh` expects `bin/$APP`.
