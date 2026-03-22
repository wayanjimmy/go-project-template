# AGENTS.md

## Database assumptions

- Assume the database is already running via `docker-compose`.
- Do **not** start or stop database containers unless explicitly asked.

## Database querying

- Use `pgcli` for interactive database queries.
- Run it via `uv` (e.g. `uv run pgcli <connection-url>`).
- If `pgcli` is not available yet, install it with `uv` before using it.
