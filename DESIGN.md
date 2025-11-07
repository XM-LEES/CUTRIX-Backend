# CUTRIX Backend Database Design (Laying-up Only)

This backend focuses on the laying-up (拉布) process. The production flow links orders, plans, layouts, tasks, and logs.

## Data Model
- `production.orders`: Core order record; cascades remove dependent data.
- `production.order_items`: Color-size-quantity per order.
- `production.plans`: Planned work for an order; status drives publish.
- `production.cutting_layouts`: Layouts under a plan.
- `production.layout_size_ratios`: Size ratio per layout.
- `production.tasks`: Work unit for laying-up: `layout_id`, `color`, `planned_layers`, `completed_layers`, `status` (`pending` | `in_progress` | `completed`).
- `production.logs`: Worker submissions: `task_id`, optional `worker_id` (FK to `public.users(user_id)`; `ON DELETE SET NULL`), auto-filled `worker_name`, `layers_completed`, `note`, `log_time`.
- Corrections (soft-void): `production.logs` adds `voided BOOLEAN DEFAULT false NOT NULL`, `void_reason TEXT`, `voided_at TIMESTAMP`, `voided_by INT REFERENCES public.users(user_id) ON DELETE SET NULL`, `voided_by_name VARCHAR(50)` (auto-filled; preserved even if user record is deleted).
- `public.users`: Shared user/worker directory; logs reference via FK with `ON DELETE SET NULL`.

Schema file:
- Base: `CUTRIX-Backend/migrations/000001_initial_schema.up.sql` (includes corrections and triggers)

## Automation
- `production.set_log_worker_name()` (BEFORE INSERT on `production.logs`): If `worker_name` is null and `worker_id` is given, fills `worker_name` from `public.users.name`.
- `production.guard_log_insert_task_status()` (BEFORE INSERT on `production.logs`): 仅允许向 `in_progress` 任务提交日志；否则抛错。
- `production.update_completed_layers()` (AFTER INSERT on `production.logs`): 校验 `layers_completed > 0`；更新 `completed_layers` 与 `status`（完成判定为 `completed_layers >= planned_layers`；否则 `in_progress` 或 `pending`）。
- `production.set_voided_by_name()` (BEFORE UPDATE OF `voided` on `production.logs`): 当日志从未作废变为作废时，若 `voided_by_name` 为空且提供了 `voided_by`，从 `public.users.name` 自动填充 `voided_by_name`；同时若 `voided_at` 为空则打上当前时间戳。
- `production.apply_log_void_delta()` (AFTER UPDATE OF `voided` on `production.logs`): 仅在“作废”时应用增量（不支持取消作废）；作废时减去该日志层数，保证不为负，并重算任务 `status`。
- `production.guard_logs_update()` (BEFORE UPDATE on `production.logs`): 限制更新范围仅为作废相关字段；禁止取消作废，且仅在 `voided = true` 时允许更新 `void_reason`/`voided_by`。
- `production.prevent_logs_delete()` (BEFORE DELETE on `production.logs`): 禁止删除日志，审计留存；使用作废代替删除。
- Plans publish:
  - `production.guard_plan_publish()` (BEFORE UPDATE on `production.plans`): 当状态变更为 `in_progress` 时写入 `planned_publish_date`（若为空），并进行前置校验。
  - `production.publish_plan_mark_tasks()` (AFTER UPDATE on `production.plans`): 发布后将该计划下的任务标记为 `in_progress`。

## Deletion Policy
- Cascades:
  - Delete `production.orders` → deletes `production.order_items`, `production.plans`, `production.cutting_layouts`, `production.layout_size_ratios`, `production.tasks`, `production.logs`.
  - Delete `production.plans` → deletes `production.cutting_layouts`, `production.layout_size_ratios`, `production.tasks`, `production.logs`.
  - Delete `production.cutting_layouts` → deletes `production.layout_size_ratios`, `production.tasks`, `production.logs`.
  - Delete `production.tasks` → deletes `production.logs`.
- Users: Deleting a user sets `worker_id` to NULL in logs (keeps `worker_name` text for history).
- Logs: 禁止硬删除；使用作废保留审计线索。
- **Deletion Strategy**: The system employs a hard-delete mechanism for orders/plans/layouts/tasks（应用层需保护有依赖的记录不被删除）; logs use soft-void for corrections.

## API Sketch (Moderate)
- Orders
  - `POST /orders`: create.
  - `GET /orders/:id`: read.
  - `DELETE /orders/:id`: delete (cascades).
- Plans
  - `POST /plans`: create for an order.
  - `POST /plans/:id/publish`: 发布计划（`pending → in_progress`），需至少存在一个任务；发布时间由触发器自动记录；返回 204。
  - `DELETE /plans/:id`: delete (cascades).
- Layouts
  - `POST /layouts`: create under a plan.
  - `DELETE /layouts/:id`: delete (cascades).
- Tasks (拉布)
  - `POST /tasks`: create (`layout_id`, `color`, `planned_layers`).
  - `GET /tasks/:id`: read progress/status.
- Logs
  - `POST /logs`: submit (`task_id`, `layers_completed`, optional `worker_id`, optional `worker_name`, optional `log_time`, optional `note`). 仅允许 `in_progress` 任务提交。
  - `PATCH /logs/:id`: corrections（作废）。Body 仅支持 `{ void_reason: string, voided_by: int }`；不支持取消作废；触发器会写入 `voided_at` 与 `voided_by_name`。
  - `GET /tasks/:id/participants`: list distinct worker names for a task (excludes voided logs).

## SQL Examples
- Insert log (worker-led):
  ```sql
  INSERT INTO production.logs(task_id, worker_id, layers_completed, note)
  VALUES ($1, $2, $3, $4);
  ```
- Task progress:
  ```sql
  SELECT planned_layers, completed_layers, status
  FROM production.tasks
  WHERE task_id = $1;
  ```
- Task participants & contribution (exclude voided):
  ```sql
  SELECT COALESCE(pl.worker_name, u.name) AS worker,
         SUM(pl.layers_completed) AS layers
  FROM production.logs pl
  LEFT JOIN public.users u ON pl.worker_id = u.user_id
  WHERE pl.task_id = $1 AND NOT pl.voided
  GROUP BY worker
  ORDER BY layers DESC;
  ```
- Daily output per worker (exclude voided):
  ```sql
  SELECT COALESCE(pl.worker_name, u.name) AS worker,
         SUM(pl.layers_completed) AS layers
  FROM production.logs pl
  LEFT JOIN public.users u ON pl.worker_id = u.user_id
  WHERE pl.log_time::date = CURRENT_DATE AND NOT pl.voided
  GROUP BY worker
  ORDER BY layers DESC;
  ```

## Notes
- Tasks are worker-agnostic; all participation comes from logs.
- Keep `layers_completed` positive; triggers enforce limits and update status.
- Logs insertion requires `in_progress` tasks; corrections are append-only via void；不支持取消作废。
- Use cascades for hard deletes across orders/plans/layouts/tasks; prefer soft-void for audit preservation in logs.