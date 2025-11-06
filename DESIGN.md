# CUTRIX Backend Database Design (Laying-up Only)

This backend focuses on the laying-up (拉布) process. The production flow links orders, plans, layouts, tasks, and logs.

## Data Model
- `production.orders`: Core order record; cascades remove dependent data.
- `production.order_items`: Color-size-quantity per order.
- `production.plans`: Planned work for an order.
- `production.cutting_layouts`: Layouts under a plan.
- `production.layout_size_ratios`: Size ratio per layout.
- `production.tasks`: Work unit for laying-up: `layout_id`, `color`, `planned_layers`, `completed_layers`, `status`.
- `production.logs`: Worker submissions: `task_id`, optional `worker_id`, auto-filled `worker_name`, `layers_completed`, `notes`, optional `log_time`.
- Corrections (soft-void): `production.logs` adds `voided BOOLEAN DEFAULT false NOT NULL`, `void_reason TEXT`, `voided_at TIMESTAMP`, `voided_by INT REFERENCES public.Workers(worker_id) ON DELETE SET NULL`, `voided_by_name VARCHAR(50)` (auto-filled; preserved even if worker record is deleted).
- `public.Workers`: Shared worker directory; logs reference via FK with `ON DELETE SET NULL`.

Schema file:
- Base: `CUTRIX-Backend/migrations/000001_initial_schema.up.sql` (includes corrections)

## Automation
- `production.set_log_worker_name()` (BEFORE INSERT on `production.logs`): If `worker_name` is null and `worker_id` is given, fills `worker_name` from `public.Workers.name`.
- `production.set_voided_by_name()` (BEFORE UPDATE OF `voided` on `production.logs`): 当日志从未作废变为作废时，若 `voided_by_name` 为空且提供了 `voided_by`，从 `public.Workers.name` 自动填充 `voided_by_name`；同时若 `voided_at` 为空则打上当前时间戳。
- `production.update_completed_layers()` (AFTER INSERT on `production.logs`): 校验 `layers_completed > 0`；不限制超过 `planned_layers`；更新 `completed_layers` 与 `status`（完成判定为 `completed_layers >= planned_layers`）。
- Corrections: `production.apply_log_void_delta()` (AFTER UPDATE OF `voided` on `production.logs`): 作废时减去该日志层数，取消作废时加回；仅保证作废后不为负；并重算任务 `status`（完成判定为 `completed_layers >= planned_layers`）。

## Deletion Policy
- Cascades:
  - Delete `production.orders` → deletes `production.order_items`, `production.plans`, `production.cutting_layouts`, `production.layout_size_ratios`, `production.tasks`, `production.logs`.
  - Delete `production.plans` → deletes `production.cutting_layouts`, `production.layout_size_ratios`, `production.tasks`, `production.logs`.
  - Delete `production.cutting_layouts` → deletes `production.layout_size_ratios`, `production.tasks`, `production.logs`.
  - Delete `production.tasks` → deletes `production.logs`.
- Workers: Deleting a worker sets `worker_id` to NULL in logs (keeps `worker_name` text for history).
- Recommendation: Consider soft-delete (`status = 'cancelled'`) for orders/plans/layouts to avoid accidental loss of logs.

## API Sketch (Moderate)
- Orders
  - `POST /orders`: create.
  - `GET /orders/:id`: read.
  - `DELETE /orders/:id`: delete (cascades).
- Plans
  - `POST /plans`: create for an order.
  - `DELETE /plans/:id`: delete (cascades).
- Layouts
  - `POST /layouts`: create under a plan.
  - `DELETE /layouts/:id`: delete (cascades).
- Tasks (拉布)
  - `POST /tasks`: create (`layout_id`, `color`, `planned_layers`).
  - `GET /tasks/:id`: read progress/status.
- Logs
  - `POST /logs`: submit (`task_id`, `layers_completed`, optional `worker_id`, optional `worker_name`, optional `log_time`). Triggers handle name fill and progress.
  - `PATCH /logs/:id`: corrections (soft-void/unvoid). Body supports `{ voided: boolean, void_reason?: string, voided_by?: int }`.
  - `GET /tasks/:id/participants`: list distinct worker names for a task (excludes voided logs).

## SQL Examples
- Insert log (worker-led):
  ```sql
  INSERT INTO production.logs(task_id, worker_id, layers_completed, notes)
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
  SELECT COALESCE(pl.worker_name, w.name) AS worker,
         SUM(pl.layers_completed) AS layers
  FROM production.logs pl
  LEFT JOIN public.Workers w ON pl.worker_id = w.worker_id
  WHERE pl.task_id = $1 AND NOT pl.voided
  GROUP BY worker
  ORDER BY layers DESC;
  ```
- Daily output per worker (exclude voided):
  ```sql
  SELECT COALESCE(pl.worker_name, w.name) AS worker,
         SUM(pl.layers_completed) AS layers
  FROM production.logs pl
  LEFT JOIN public.Workers w ON pl.worker_id = w.worker_id
  WHERE pl.log_time::date = CURRENT_DATE AND NOT pl.voided
  GROUP BY worker
  ORDER BY layers DESC;
  ```

## Notes
- Tasks are worker-agnostic; all participation comes from logs.
- Keep `layers_completed` positive; triggers enforce limits and update status.
- Corrections are append-only through void/unvoid; use `PATCH /logs/:id` to toggle `voided`. Reports and participants exclude voided logs.
- Use cascades for hard deletes; prefer soft-delete for audit preservation when needed.