# CUTRIX Backend Logging Guide

This document explains how logging is implemented, what events and fields are recorded, and how to configure and use logs in CUTRIX Backend.

## Overview

- Library: Go `slog` with JSON output.
- Global logger: `cutrix-backend/internal/logger` exposes `logger.L`.
- Level control: via env `LOG_LEVEL` (`debug|info|warn|error`), defaults to `info`.
- Goals: request tracing, error diagnostics, and domain event auditing.

## Where Logs Are Emitted

- Request middleware: `internal/middleware/request_logger.go`
  - Event: `http_request` emitted once per request after completion.
  - Level: `<400: info`, `4xx: warn`, `>=500: error`.
  - Fields: `method`, `path`, `status_code`, `duration_ms`, `client_ip`, `request_id`, `user_id`.
- Error mapping: `internal/handlers/http_helpers.go`
  - Event: `http_error` emitted when service errors are written to HTTP.
  - Level: `404: info`, `4xx: warn`, `>=500: error`.
  - Fields: `method`, `path`, `status_code`, `error`, `request_id`, `user_id`.
- Startup & DB:
  - `cmd/api/main.go`: `api_listen`, `startup`, `db_connect_failed`, `migrations_failed`.
  - `internal/db/db.go`: `migrations_applied` (script name).
- Service domain events:
  - Plans: `plan_created`, `plan_deleted`, `plan_note_updated`, `plan_published`, `plan_frozen`.
  - Tasks: `task_created`, `task_deleted`.
  - Logs: `log_created`, `log_voided`.

## Field Conventions

- Correlation: `request_id` (from RequestID middleware), `user_id` (from Auth middleware).
- HTTP: `method`, `path`, `status_code`, `duration_ms`, `client_ip`.
- Domain: prefer concise identifiers: `plan_id`, `order_id`, `task_id`, `layout_id`, `log_id`.
- Text fields: avoid logging large free-text by default; use `info` level and consider truncation if necessary.

## Level Policy

- Informational: normal operations (e.g., `http_request` <400, `plan_created`).
- Warning: client-side or action issues (e.g., validations, 4xx `http_error`).
- Error: server-side failures (e.g., 5xx `http_request` / `http_error`, `db_connect_failed`).

## Configuration

- `LOG_LEVEL`: controls minimum log level.
- `HTTP_HOST` and `HTTP_PORT`: server bind address; logged as `api_listen`.
- `DATABASE_URL`: database connection string; if empty, API starts with services disabled for DB-bound operations.

## Usage Examples

- Request log (info):
  ```json
  {"time":"...","level":"INFO","msg":"http_request","method":"POST","path":"/api/v1/plans/:id/publish","status_code":204,"duration_ms":5,"client_ip":"127.0.0.1","request_id":"...","user_id":1}
  ```
- Error log (warn):
  ```json
  {"time":"...","level":"WARN","msg":"http_error","method":"POST","path":"/api/v1/plans/:id/freeze","status_code":500,"error":"仅允许在 completed 状态下冻结计划","request_id":"...","user_id":1}
  ```
- Domain event (info):
  ```json
  {"time":"...","level":"INFO","msg":"plan_published","plan_id":6}
  ```

## Security & Privacy

- Do not log secrets or raw tokens. Auth middleware should set `user_id` only.
- Redact or omit personally identifiable information unless necessary.
- Avoid logging entire payloads; log identifiers and summary counts.

## Operational Tips

- Aggregation: ship stdout to your log collector (e.g., `docker logs`, Fluent Bit, Vector).
- Filtering: use `msg` (event name) and IDs to slice logs for a single flow.
- Troubleshooting: correlate `http_error` with preceding `http_request` via `request_id`.

## Maintenance

- When adding new features, emit domain events at service layer after successful repository operations.
- Keep event names concise and stable.
- Prefer `slog.String/Int/Any` for consistent structured fields.

## Quick Start

1. Set `LOG_LEVEL=debug` during development for verbose output.
2. Ensure RequestID/Auth middlewares are active to populate `request_id` and `user_id`.
3. Start the API and observe JSON logs:
   - `go run cmd/api/main.go`
   - Call `GET http://127.0.0.1:8080/api/v1/health` and check `http_request` logs.