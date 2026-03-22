# AGENTS.md

## Database assumptions

- Assume the database is already running via `docker-compose`.
- Do **not** start or stop database containers unless explicitly asked.

## Database querying

- Use `pgcli` for interactive database queries.
- Run it via `uv` (e.g. `uv run pgcli <connection-url>`).
- If `pgcli` is not available yet, install it with `uv` before using it.

## Container tooling

- Use `podman` instead of `docker` for image/container operations.
- When building Docker-compatible images, always pass `--format=docker`.
- Preferred build command:
  - `podman build --format=docker -t <image>:<tag> -f Dockerfile .`

## Container runtime notes

- Runtime image is distroless + non-root. Avoid writing under `/app` at runtime.
- For `cmd/admin-tools` in containers, set a writable secret root:
  - `SECRET_ROOT=/tmp/.secrets`
- If `SECRET_ROOT` is not writable, startup can fail with:
  - `failed to create key root: mkdir .secrets: permission denied`
