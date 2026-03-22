# Go Project Template

A personal Go template to start new projects quickly.

This repository captures lessons learned from designing, building, and maintaining Go applications over the last several years.

## Why this template exists

I made this template for myself so I can bootstrap new services with:

- practical architecture boundaries
- multiple binaries in one repository
- consistent logging and traceability
- simple, text-based API tests
- optional web UI support without jumping to a monorepo too early

## Design influences

This template is strongly influenced by:

- **Go for Industrial Programming** — Peter Bourgon  
  https://peter.bourgon.org/go-for-industrial-programming/
- **Packages as Layers (avoid cyclic dependencies)**  
  https://github.com/benbjohnson/wtf
- **GoTime / Changelog logger-debugger mindset**  
  https://changelog.com/gotime/309
- **Bootstrap/environment patterns** (`serverenv`, `setup`, `key`)  
  inspired by https://github.com/google/exposure-notifications-server

The core principles from those notes are implemented directly in this project.

---

## Architecture

This template uses layered packages (not package grouping by type-only):

`transport (REST, Inertia UI, etc) -> business (service) -> data (repository)`

### Layer examples in this repo

- **Transport**: `rest/v1`, `web`, `server`
- **Business**: `service`
- **Data**: `repository`, `database`

This keeps dependencies flowing in one direction and helps avoid cyclic imports as the codebase grows.

---

## Multiple binaries (`cmd/`)

Following the "industrial Go" idea, this repo supports multiple binaries in one project:

- `cmd/server` – API server (`:3030` by default)
- `cmd/admin-tools` – admin web UI with Inertia (`:8081` by default)
- `cmd/worker` – worker entrypoint (currently no-op for in-memory GoChannel mode)
- `cmd/elasticcli` – example operational CLI

This demonstrates that one repository can hold multiple executables without introducing monorepo complexity too early.

---

## Logging philosophy (`logger/`)

The logger is intentionally constrained to reduce noise:

- app-level levels are **Info** and **Error** (what matters most)
- request tracing uses `X-Request-ID` -> logged as `correlation_id`
- JSON structured logs with service/build metadata

This aligns with the GoTime/Changelog mindset: rely on logs + mental model first, debugger as fallback.

### Sample trace log (single request across layers)

```json
{"time":"2026-03-22T17:23:41.716159536+07:00","level":"INFO","file":"user.go:55","msg":"handler.user.create","service":"APP-API","version":"dev","correlation_id":"demo-readme-123"}
{"time":"2026-03-22T17:23:41.716225369+07:00","level":"INFO","file":"user.go:44","msg":"service.user.create","service":"APP-API","version":"dev","correlation_id":"demo-readme-123"}
{"time":"2026-03-22T17:23:41.71624178+07:00","level":"INFO","file":"user_postgres.go:70","msg":"repository.user.save","service":"APP-API","version":"dev","user_id":"2b5e5cdd-a79b-4a61-9f72-e057882e5cc9","correlation_id":"demo-readme-123"}
{"time":"2026-03-22T17:23:41.718380303+07:00","level":"INFO","file":"user_index_postgres.go:29","msg":"repository.user_search.index","service":"APP-API","version":"dev","user_id":"2b5e5cdd-a79b-4a61-9f72-e057882e5cc9","correlation_id":"demo-readme-123"}
```

---

## API testing with Hurl (`test/hurl`)

I use **Hurl** for API tests because it is plain text:

- easy for humans to read/review
- easy for AI agents to generate/update
- git-friendly and conflict-friendly compared to JSON dumps

Run all API tests:

```bash
task test:http
```

### Sample test summary

```text
Final Test Summary
==================
Total Tests: 8
Passed: 8
Failed: 0
```

---

## Admin tools with Inertia

`cmd/admin-tools` provides an Inertia-based web UI.

I chose Inertia because of prior Laravel + Inertia experience and good developer productivity. In this template it serves two purposes:

1. practical admin-tools use case for future projects that need web UI
2. demonstration of multiple binaries (API + admin UI) in one Go project

---

## Quick start

## Prerequisites

- Go (project targets Go 1.25+)
- [Task](https://taskfile.dev)
- PostgreSQL
- [Hurl](https://hurl.dev)
- Node.js + `vp` (vite-plus) for admin-tools frontend assets

### 1) Environment

Use `.envrc.example` as baseline:

```bash
cp .envrc.example .envrc
```

Set at least:

- `DATABASE_URL` (example: `postgres://app:app@localhost:5432/app_db?sslmode=disable`)

### 2) Run API server

```bash
task run
```

API base URL: `http://localhost:3030`

### 3) Run Hurl API tests

```bash
task test:http
```

### 4) Run admin tools (Inertia)

```bash
task run:admin-tools
```

Admin tools URL: `http://localhost:8081/users`

### Admin tools screenshots

Users page:

![Admin tools users page](docs/screenshots/admin-tools-users.png)

Search example (`/users?q=Hurl`):

![Admin tools users search page](docs/screenshots/admin-tools-users-search.png)

---

## Build

Build all binaries with injected version metadata (single `VERSION` value):

### Versioning convention

Use one deploy identifier in this format:

`<branch>-<commit>-<timestamp>`

Example:

`main-ffa869b-20260316133451`

This single value is used as the application version in build metadata and container builds.

```bash
task build
```

You can override `VERSION` (example format: `<branch>-<commit>-<timestamp>`):

```bash
VERSION=main-ffa869b-20260316133451 task build
```

Output binaries are written to `./bin/`.

### Build and run container image (Podman)

This project ships a single multi-stage `Dockerfile` that:

- builds frontend assets for `cmd/admin-tools`
- builds Go binaries into one image
- uses a distroless non-root runtime image

Build image (Docker-compatible format, single `VERSION` build arg):

```bash
podman build --format=docker \
  --build-arg VERSION=main-ffa869b-20260316133451 \
  -t localhost/go-project-template:latest \
  -f Dockerfile .
```

#### Run `cmd/server`

```bash
podman run --rm \
  -p 3030:3030 \
  -e DATABASE_URL="postgres://app:app@host.containers.internal:5432/app_db?sslmode=disable" \
  localhost/go-project-template:latest
```

API URL: `http://localhost:3030`

#### Run `cmd/admin-tools`

Use the same image, but override entrypoint to the admin-tools binary:

```bash
podman run --rm \
  --entrypoint /app/bin/admin-tools \
  -p 8081:8081 \
  -e DATABASE_URL="postgres://app:app@host.containers.internal:5432/app_db?sslmode=disable" \
  -e APP_ENV=production \
  localhost/go-project-template:latest
```

Admin tools URL: `http://localhost:8081/users`

---

## Repository structure (high-level)

```text
cmd/            # binaries (server, admin-tools, worker, cli)
server/         # HTTP router + middleware
rest/           # REST handlers
web/            # Inertia handlers/setup
service/        # business logic
repository/     # persistence and search index access
database/       # SQL + embedded migrations
logger/         # structured logger and adapters
requestid/      # request ID context + propagation
test/hurl/      # plain-text API tests
```

---

## References

- Peter Bourgon — Go for Industrial Programming: https://peter.bourgon.org/go-for-industrial-programming/
- Ben Johnson — wtf (package/layering inspiration): https://github.com/benbjohnson/wtf
- GoTime / Changelog ep. 309: https://changelog.com/gotime/309
- Google — exposure-notifications-server (bootstrap/env inspiration for `serverenv`, `setup`, `key`): https://github.com/google/exposure-notifications-server
