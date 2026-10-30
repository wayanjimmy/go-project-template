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

## Events and jobs: Watermill + Asynq

The template includes two different background-work examples:

| Example | Tool | Meaning |
| --- | --- | --- |
| Maintain the user search index | Watermill | A user changed; subscribers react to the fact |
| Deliver a signed `user.created` webhook | Asynq | Execute an HTTP delivery with bounded retries |

User creates, updates, and deletes save an event in `event_outbox` in the same
PostgreSQL transaction as the user change. The API relays pending events to
Watermill with stable event IDs. In Redis mode the API publishes only; workers
consume through separate groups: `app-worker` for search and `app-webhooks` for
webhook scheduling. Replicas within a group share work, while each group receives
its own copy.

Search events trigger a refresh from the current `users` row under a row lock.
Replayed creation events cannot overwrite newer user fields, and events for
deleted users leave no search entry. Webhooks retain the original event snapshot.

The webhook subscriber persists an immutable delivery body in `webhook_deliveries`.
A dispatcher enqueues its ID in the Asynq `webhooks` queue. Enqueue failures leave
the row recoverable. A two-minute backstop re-enqueues overdue pending deliveries,
including tasks lost from Redis. Database leases serialize attempts across replicas
without holding a database connection during the HTTP request.

Webhooks are optional. Configure these on `cmd/worker`:

| Variable | Meaning |
| --- | --- |
| `REDIS_ADDR` | Required Redis address for distributed events and Asynq |
| `WEBHOOK_URL` | One operator-configured HTTP(S) destination; unset disables webhooks |
| `WEBHOOK_SECRET` | Required HMAC signing secret when a URL is configured |
| `WEBHOOK_RETRY_BASE` | Exponential retry base, default `5s`, minimum `1s` |

Delivery uses four concurrent handlers, a ten-second HTTP timeout, and at most six
attempts. Network errors, `408`, `429`, and `5xx` retry; other non-`2xx` responses
are terminal. Redirects are not followed. Asynq's retry schedule includes one
extra second to accommodate Redis's whole-second timestamps. Terminal failures
are recorded in PostgreSQL, logged for ntfy, and archived in Asynq. Asynqmon may
be connected to the same Redis instance for task inspection.

Requests include `X-Webhook-ID`, `X-Webhook-Timestamp`, `X-Webhook-Signature`, and
`X-Request-ID`. The signature is `sha256=<hex HMAC-SHA256>` over the timestamp,
a period, and the exact request body. Receivers should verify signatures with
constant-time comparison, reject stale timestamps, and deduplicate successful
deliveries by `X-Webhook-ID`. Delivery is **at least once**, not exactly once:
a crash after receiver acceptance can repeat the same ID and body. Payloads
include user ID, name, and email; addresses never enter the outbox or webhook.

Persist Redis data for normal operation. PostgreSQL recovery can reconstruct
pending webhook tasks, but does not reconstruct acknowledged Watermill history
after Redis data loss. The in-process API mode remains a development convenience,
not a durable replacement for Redis Streams. Retention/cleanup of outbox rows,
delivery records, and streams is an application-specific operational policy.

### Run and verify through Tilt on the host

`TILT_MODE=host` runs the real API and worker binaries plus a controlled local
HTTP receiver, without Kubernetes or starting/stopping infrastructure containers.
Provide existing local PostgreSQL and Redis instances. Use a **disposable database**:
startup applies migrations and the end-to-end tests create and delete users.

```bash
TILT_MODE=host \
DATABASE_URL='postgres://app:app@127.0.0.1:25432/webhook_test?sslmode=disable' \
REDIS_ADDR='127.0.0.1:6379' \
tilt up

# After server and worker are running:
tilt trigger webhook-e2e
tilt logs webhook-e2e
tilt logs server worker webhook-receiver
```

Host mode defaults to the receiver at `127.0.0.1:8092/hooks`, the public demo
signing value `local-demo-not-a-secret`, and a one-second retry base. Do not use
that signing value for an external destination. The receiver's `/control` and
`/requests` endpoints are test tools and must never be exposed publicly.

The `webhook-e2e` resource checks signed success, `500 → 500 → 204` retries,
permanent rejection, redirect refusal, exhausted retries, duplicate event
deduplication, and the independent search subscriber. Its output records delivery
and event IDs. Watch `outbox.published`, `webhook.recorded`, `webhook.enqueued`,
`webhook.attempt`, and `webhook.failed` logs to follow the same operation.

For failure recovery, run the following separately from other tests, against an
isolated host-mode stack. It pauses Redis, temporarily denies Redis scripting,
and kills only the worker binary from this checkout before restarting it through
Tilt. It checks recovery after database commit, enqueue failure, and receiver
acceptance before a worker crash. Linux `/proc` is required for the crash case.

```bash
WEBHOOK_TEST_DISPOSABLE=1 \
BASE_URL=http://127.0.0.1:3030 \
WEBHOOK_RECEIVER_URL=http://127.0.0.1:8092 \
DATABASE_URL="$DATABASE_URL" REDIS_ADDR="$REDIS_ADDR" \
go test -tags=e2e ./test/webhook -run '^TestWebhookRecovery$' -v -count=1 -timeout=8m
```

The recovery test changes Redis permissions for the default user. Never run it
against shared Redis or alongside another application.

Default Tilt mode still uses kind/ctlptl through `Tiltfile.kind`; it includes a
receiver Deployment and the webhook configuration for the local example.
`Tiltfile.host` intentionally runs only the API, worker, receiver, and manual
test resource. Run admin-tools separately when needed.

Repository integration tests can use `TEST_DATABASE_URL` to target an existing
disposable database instead of starting testcontainers. Fixture loading resets
tables; never point it at application or production data.

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

## Local Kubernetes with Tilt

Run the multi-binary deployment (server, worker, admin-tools) against a local kind cluster with a local image registry, managed by [ctlptl](https://github.com/tilt-dev/ctlptl):

```bash
# 1) Start infra (postgres, redis, ntfy) on the host
docker-compose up -d

# 2) One-time: create the kind cluster + local registry
ctlptl apply -f cluster.yaml

# 3) Start Tilt (rootless podman needs DOCKER_HOST)
export DOCKER_HOST=unix:///run/user/1000/podman/podman.sock
tilt up
```

On each change Tilt builds the binary on the host (`CGO_ENABLED=0 go build`), packs it into a thin image, pushes it to the local registry, and rolls the Deployment. The pods reach host-side postgres/redis/ntfy via `host.containers.internal`. Open `http://localhost:10350` for the Tilt dashboard:

![Tilt dashboard with server, worker and admin-tools running on kind](docs/screenshots/tilt-dashboard.png)

### Logs

Every binary logs structured JSON (time, level, msg, service, correlation_id) visible live in the Tilt resource panes.

Server — request handling, repository writes, publishing `user.created` to the Redis stream, and error responses (these also fire an ntfy alert):

![Server log pane: JSON logs including redis stream publish and error response](docs/screenshots/tilt-logs-server.png)

Worker — receiving the message from the stream (consumer group `app-worker`), indexing the user, and acking:

![Worker log pane: received, indexed, acked](docs/screenshots/tilt-logs-worker.png)

ntfy subscriber pane — the live alert stream the apps publish on error-level logs:

![ntfy-subscriber pane streaming alerts](docs/screenshots/tilt-logs-ntfy-subscriber.png)

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
