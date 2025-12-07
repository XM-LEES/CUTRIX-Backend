# Handlers API Endpoints

This document summarizes the REST endpoints exposed by the server, mapped to UsersService and AuthService responsibilities, with error conventions.

## Auth
- POST `/api/v1/auth/login`
  - Request: `{ "name": "...", "password": "..." }`
  - Response: `{ "access_token": "...", "refresh_token": "...", "expires_at": "RFC3339", "user": UserDTO }`
  - Notes: Returns tokens and user view on success.

- POST `/api/v1/auth/refresh`
  - Request: `{ "refresh_token": "..." }`
  - Response: Same as login; returns new access and refresh tokens.
  - Notes: Validates internal token_type=refresh and signature.

- PUT `/api/v1/auth/password`
  - Header: `Authorization: Bearer <access_token>`
  - Request: `{ "old_password": "...", "new_password": "..." }`
  - Response: `204 No Content`
  - Notes: Changes current user's password after old password verification.

## Users
- GET `/api/v1/users`
  - Query: `query`, `name`, `role`, `active`, `group`
  - Response: `[]UserDTO`
  - Notes: If `name` provided, performs exact lookup first then applies other filters; otherwise uses DB ILIKE and orders by `name ASC`.

- GET `/api/v1/users/:id`
  - Response: `UserDTO`

- POST `/api/v1/users`
  - Header: `Authorization: Bearer <access_token>` (admin)
  - Request: `{ "name": "...", "role": "...", "group": "nullable", "note": "nullable" }`
  - Response: `UserDTO`
  - Notes: Creates user with `is_active=true`. Password is managed via Auth endpoints.

- PATCH `/api/v1/users/:id/profile`
  - Header: `Authorization: Bearer <access_token>` (owner or admin)
  - Request: `{ "name": "optional", "group": "optional", "note": "optional" }`
  - Response: `UserDTO`
  - Notes: Updates non-sensitive fields only.

- PUT `/api/v1/users/:id/role`
  - Header: `Authorization: Bearer <access_token>` (admin)
  - Request: `{ "role": "..." }`
  - Response: `204 No Content`

- PUT `/api/v1/users/:id/active`
  - Header: `Authorization: Bearer <access_token>` (admin)
  - Request: `{ "active": true|false }`
  - Response: `204 No Content`

- PUT `/api/v1/users/:id/password`
  - Header: `Authorization: Bearer <access_token>` (admin)
  - Request: `{ "new_password": "..." }`
  - Response: `204 No Content`
  - Notes: Sets or resets user's password without old password check.

- DELETE `/api/v1/users/:id`
  - Header: `Authorization: Bearer <access_token>` (admin)
  - Response: `204 No Content`

## Orders
- POST `/api/v1/orders`
  - Request: `{ order: ProductionOrder fields, items: [OrderItem, ...] }`
  - Response: `ProductionOrder`
  - Notes: Creates order and items atomically; order must include at least one item.

- GET `/api/v1/orders`
  - Response: `[]ProductionOrder`
  - Notes: Lists all orders ordered by `created_at DESC`.

- GET `/api/v1/orders/:id`
  - Response: `ProductionOrder`

- GET `/api/v1/orders/by-number/:number`
  - Response: `ProductionOrder`
  - Notes: Fetches order by unique `order_number`.

- GET `/api/v1/orders/:id/full`
  - Response: `{ order: ProductionOrder, items: []OrderItem }`
  - Notes: Returns order with its immutable items.

- PATCH `/api/v1/orders/:id/note`
  - Request: `{ note: "nullable" }`
  - Response: `204 No Content`
  - Notes: Updates `note`; `updated_at` is auto-managed by DB trigger.

- PATCH `/api/v1/orders/:id/finish-date`
  - Request: `{ order_finish_date: "RFC3339 or null" }`
  - Response: `204 No Content`
  - Notes: Updates `order_finish_date`; `updated_at` is auto-managed by DB trigger.

- DELETE `/api/v1/orders/:id`
  - Response: `204 No Content`
  - Notes: Cascades deletion to order items.

## Plans
- POST `/api/v1/plans`
  - Request: `ProductionPlan` fields
  - Response: `ProductionPlan`
  - Notes: Must reference an order; created with `pending` status.

- DELETE `/api/v1/plans/:id`
  - Response: `204 No Content`
  - Notes: Cascades deletion to related layouts/tasks/ratios.

- GET `/api/v1/plans`
  - Response: `[]ProductionPlan`
  - Notes: Returns all plans ordered by `plan_id DESC`.

- GET `/api/v1/plans/:id`
  - Response: `ProductionPlan`

- GET `/api/v1/orders/:id/plans`
  - Response: `[]ProductionPlan`

- PATCH `/api/v1/plans/:id/note`
  - Request: `{ note: "nullable" }`
  - Response: `204 No Content`
  - Notes: After publish, only `note` is editable.

- POST `/api/v1/plans/:id/publish`
  - Response: `204 No Content`
  - Notes: Transitions `pending → in_progress`; requires at least one task; publish time is set by DB trigger.

- POST `/api/v1/plans/:id/freeze`
  - Response: `204 No Content`
  - Notes: Transitions `completed → frozen`; completion time preserved by DB trigger.

## Layouts
- POST `/api/v1/layouts`
  - Request: `CuttingLayout` fields
  - Response: `CuttingLayout`
  - Notes: Structural changes (create/delete/rename) allowed only when the plan is `pending`.

- DELETE `/api/v1/layouts/:id`
  - Response: `204 No Content`

- GET `/api/v1/layouts`
  - Response: `[]CuttingLayout`
  - Notes: Returns all layouts. Useful for batch operations and performance optimization.

- GET `/api/v1/layouts/:id`
  - Response: `CuttingLayout`

- GET `/api/v1/plans/:id/layouts`
  - Response: `[]CuttingLayout`

- PATCH `/api/v1/layouts/:id/name`
  - Request: `{ name: "..." }`
  - Response: `204 No Content`
  - Notes: Only allowed in `pending` state.

- PATCH `/api/v1/layouts/:id/note`
  - Request: `{ note: "nullable" }`
  - Response: `204 No Content`
  - Notes: Allowed after publish.

- POST `/api/v1/layouts/:id/ratios`
  - Request: `{ ratios: {...} }`
  - Response: `204 No Content`
  - Notes: Sets size ratios for a layout. Only allowed when plan is `pending`.

- GET `/api/v1/layouts/:id/ratios`
  - Response: `[]LayoutSizeRatio`
  - Notes: Returns size ratios for a single layout.

- POST `/api/v1/layouts/ratios/batch`
  - Request: `{ layout_ids: [1, 2, 3, ...] }`
  - Response: `{ layout_id: []LayoutSizeRatio, ... }`
  - Notes: Batch retrieve ratios for multiple layouts. Useful for performance optimization.

## Tasks
- POST `/api/v1/tasks`
  - Request: `ProductionTask` fields
  - Response: `ProductionTask`
  - Notes: Structural changes allowed only when the plan is `pending`.

- DELETE `/api/v1/tasks/:id`
  - Response: `204 No Content`

- GET `/api/v1/tasks`
  - Response: `[]ProductionTask`
  - Notes: Returns all tasks. Useful for batch operations and performance optimization.

- GET `/api/v1/tasks/:id`
  - Response: `ProductionTask`

- GET `/api/v1/layouts/:id/tasks`
  - Response: `[]ProductionTask`
  - Notes: Task status updates are recorded via Logs endpoints; handlers do not expose direct status updates.

## Logs
- POST `/api/v1/logs`
  - Request: `{ "task_id": int, "layers_completed": int, "worker_id": "optional", "worker_name": "optional", "note": "nullable" }`
  - Response: `ProductionLog`
  - Notes: Requires task status `in_progress`; `layers_completed > 0`; if only `worker_id` is provided, `worker_name` is auto-filled by a DB trigger; the request field is named `note` (not `notes`).

- PATCH `/api/v1/logs/:id`
  - Header: `Authorization: Bearer <access_token>` (requires `log:update` permission)
  - Request: `{ "voided_by": int, "void_reason": "nullable" }`
  - Response: `204 No Content` on success
  - Error Responses:
    - `403 forbidden` with `message: "只能作废自己的日志"` - Worker can only void their own logs
    - `403 forbidden` with `message: "只能作废24小时内的日志"` - Log must be submitted within 24 hours
    - `403 forbidden` with `message: "24小时内最多只能作废3条日志"` - Worker has reached the 24-hour limit (3 logs)
  - Notes: 
    - Marks the log as voided; DB triggers set `voided_at` and `voided_by_name` and adjust task `completed_layers`. Unvoid is not allowed.
    - **Worker restrictions**: If the requester is a `worker` role, additional validations apply:
      - Must be the owner of the log (matched by `worker_id` or `worker_name`)
      - Log must have been submitted within the last 24 hours
      - Worker must not have voided more than 3 logs in the past 24 hours
    - Admin and manager roles bypass these restrictions.

- GET `/api/v1/logs/my`
  - Header: `Authorization: Bearer <access_token>`
  - Response: `[]ProductionLog`
  - Notes: Returns all logs for the current authenticated user (matched by worker_id and/or worker_name). Requires authentication. Ordered by log_time DESC.

- GET `/api/v1/logs/recent-voided`
  - Header: `Authorization: Bearer <access_token>` (requires admin/manager role)
  - Query: `limit` (optional, default: 50, max: 100)
  - Response: `[]ProductionLog`
  - Notes: Returns recently voided logs ordered by `voided_at DESC`. Used for manager notifications. Only logs with `voided = true` are returned.

- GET `/api/v1/tasks/:id/participants`
  - Header: `Authorization: Bearer <access_token>` (requires admin/manager role)
  - Response: `[]string`
  - Notes: Distinct worker names from non-void logs; uses `COALESCE(logs.worker_name, users.name)`.

- GET `/api/v1/tasks/:id/logs`
  - Header: `Authorization: Bearer <access_token>` (requires admin/manager role)
  - Response: `[]ProductionLog`
  - Notes: Returns all logs for the specified task (including voided logs).

- GET `/api/v1/layouts/:id/logs`
  - Header: `Authorization: Bearer <access_token>` (requires admin/manager role)
  - Response: `[]ProductionLog`
  - Notes: Returns all logs for tasks under the specified layout (including voided logs).

- GET `/api/v1/plans/:id/logs`
  - Header: `Authorization: Bearer <access_token>` (requires admin/manager role)
  - Response: `[]ProductionLog`
  - Notes: Returns all logs for tasks under the specified plan (including voided logs).

## Error Conventions
- `401 unauthorized`: invalid/expired token, login failed, wrong old password.
- `403 forbidden`: insufficient permissions (non-admin modifying restricted fields).
- `409 conflict`: name uniqueness violations on create/rename.
- `404 not_found`: resource not found.
- `400 validation_error`: missing or invalid parameters.
- `500 internal_error`: unexpected errors.

Response format for errors: `{ "error": "<code>", "message": "..." }` where `<code>` is one of `unauthorized`, `forbidden`, `conflict`, `not_found`, `validation_error`, `internal_error`.