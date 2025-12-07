# CUTRIX Backend Design (Laying-up Only)

本后端专注拉布（Laying-up）流程，围绕订单 → 计划 → 版型 → 任务 → 日志形成闭环。

## Data Model
- `production.orders`: 订单主记录；删除时级联清理依赖数据。
- `production.order_items`: 订单的颜色/尺码/数量明细。
- `production.plans`: 订单的工作计划；状态用于发布（publish）。
- `production.cutting_layouts`: 计划下的版型（排料）。
- `production.layout_size_ratios`: 版型对应的尺码比例。
- `production.tasks`: 拉布任务，包含 `layout_id`、`color`、`planned_layers`、`completed_layers`、`status`（`pending` | `in_progress` | `completed`）。
- `production.logs`: 工人提交的工作日志：`task_id`、可选 `worker_id`（FK 到 `public.users(user_id)`，`ON DELETE SET NULL`）、自动填充的 `worker_name`、`layers_completed`、`note`、`log_time`。
- 修正（软作废）：日志增加 `voided BOOLEAN NOT NULL DEFAULT false`、`void_reason`、`voided_at TIMESTAMP`、`voided_by INT REFERENCES public.users(user_id) ON DELETE SET NULL`、`voided_by_name VARCHAR(50)`（自动填充；即使用户被删除也保留文本）。
- `public.users`: 用户目录；日志通过 FK 引用，删除用户时将日志中的 `worker_id` 置空并保留 `worker_name`。
  - 唯一索引约束：`users_single_active_admin_idx` 和 `users_single_active_manager_idx` 确保系统只能有一个活跃的 Admin 和一个活跃的 Manager。

Schema 文件：`migrations/000001_initial_schema.up.sql`（含触发器与约束）。

## Automation（数据库触发器）
- `production.set_log_worker_name()`（BEFORE INSERT on `production.logs`）：若提供 `worker_id` 且 `worker_name` 为空，则自动填充 `worker_name`。
- `production.guard_log_insert_task_status()`（BEFORE INSERT on `production.logs`）：仅允许向 `in_progress` 任务提交日志；否则抛错。
- `production.update_completed_layers()`（AFTER INSERT on `production.logs`）：校验 `layers_completed > 0`；更新 `completed_layers` 与 `status`。
- `production.set_voided_by_name()`（BEFORE UPDATE OF `voided` on `production.logs`）：作废时自动填充 `voided_by_name` 与时间戳。
- `production.apply_log_void_delta()`（AFTER UPDATE OF `voided` on `production.logs`）：作废日志对应减层并重算任务状态（不支持取消作废）。
- `production.guard_logs_update()`（BEFORE UPDATE on `production.logs`）：将更新范围限制为作废相关字段；禁止取消作废。
- `production.prevent_logs_delete()`（BEFORE DELETE on `production.logs`）：禁止硬删除日志，采用软作废保留审计线索。
- 发布计划：
  - `production.guard_plan_publish()`（BEFORE UPDATE on `production.plans`）：当状态变更为 `in_progress` 时写入 `planned_publish_date` 并进行前置校验。
  - `production.publish_plan_mark_tasks()`（AFTER UPDATE on `production.plans`）：发布后将该计划下的任务标记为 `in_progress`。

## Deletion Policy
- 级联删除：
  - 删除 `orders` → 级联删除 `order_items`、`plans`、`cutting_layouts`、`layout_size_ratios`、`tasks`、`logs`。
  - 删除 `plans` → 级联删除其下 `cutting_layouts`、`layout_size_ratios`、`tasks`、`logs`。
  - 删除 `cutting_layouts` → 级联删除其下 `layout_size_ratios`、`tasks`、`logs`。
  - 删除 `tasks` → 级联删除其下 `logs`。
- 用户删除：将日志的 `worker_id` 置空但保留 `worker_name` 文本。
- 日志删除：禁止硬删除，用软作废代替。

## 认证与权限设计

认证（Authentication）与权限（Authorization/Permissions）是两件事：
- 认证：证明“你是谁”。本系统使用用户名/密码登录，密码使用安全哈希存储。登录成功后签发 `access_token` 与 `refresh_token`，`Authorization: Bearer <token>` 传递。
- 权限：决定“你可以做什么”。本系统采用 RBAC（角色为主）+ 权限字符串（细粒度），在路由层按模块/动作进行校验。

实现要点：
- 中间件链路：
  - `RequireAuth` 校验令牌、解析用户并注入到请求上下文，失败返回 `401 Unauthorized`。
  - 其后路由根据需要使用 `RequireRoles(...)`（只允许特定角色）或 `RequirePermissions(...)`（校验权限字符串）。
- 权限模型：`RolePermissionsMap` 在 `internal/middleware/permissions.go`，支持通配符（如 `plan:*` 表示该模块所有动作）。
  - `admin` / `manager`：拥有全部权限（代码中直接短路放行）。
  - `pattern_maker`（制版员）：可创建、查看、修改、删除计划；可管理版型和任务；**但不能发布计划**（无 `plan:publish` 权限）；**不能查看任务管理页面**（无 `task:read` 权限）；**不能修改已发布计划的备注**（Handler 层业务规则检查）；**只能删除未发布的计划**（Handler 层业务规则检查）；**可以在计划详情中查看任务信息**（通过 `/layouts/:id/tasks` 接口，使用 `layout:read` 权限）。
  - `worker`：允许拉布日志相关与任务查看（如 `log:create/update`、`task:read`）。
- 业务兜底：除路由权限外，服务/数据库层还包含业务规则，如：
  - 计划发布后限制部分编辑（例如已发布后不可改版型名称）。
  - 列出任务日志与参与者仅限管理层（admin/manager）。
  - 插入日志仅允许针对 `in_progress` 任务。
  - **用户管理约束**（在 `internal/services/users_service_impl.go` 实现）：
    - Admin/Manager 不能修改自己的角色、不能删除自己、不能停用自己。
    - Manager 不能对 Admin 执行任何操作（创建、编辑、删除、重置密码、修改角色、修改状态）。
    - Manager 不能创建 Admin 用户，也不能创建新的 Manager 用户（系统只需一个 Manager）。
    - Admin 不能创建新的 Admin 用户（系统只需一个 Admin）。
    - 数据库唯一索引确保系统只能有一个活跃的 Admin 和一个活跃的 Manager。

权限矩阵（精选）：
- `admin`：全模块全动作；可管理订单、计划、版型、任务、日志、参与者。**限制**：不能修改/删除/停用自己；不能创建新的 Admin（系统只需一个）。
- `manager`：与 `admin` 等价全访问。**限制**：不能操作 Admin；不能修改/删除/停用自己；不能创建 Admin 或新的 Manager（系统只需一个）。
- `pattern_maker`（制版员）：
  - **可操作**：创建/查看/修改/删除计划（仅未发布状态）；管理版型和任务（创建/删除任务，但不查看任务管理页面）；查看订单。
  - **不可操作**：发布计划（无 `plan:publish` 权限）；冻结计划（无 `plan:freeze` 权限）；修改已发布计划的备注（Handler 层业务规则）；删除已发布的计划（Handler 层业务规则）；查看任务管理页面（无 `task:read` 权限）；查看日志记录（无 `log:create` 权限）。
- `worker`：任务可读、日志可提交与作废；不可查看日志列表、不可修改计划/版型/订单。

流程示意：
- 登录 → 获取令牌。
- 发起请求 → `RequireAuth` 通过 → `RequireRoles/RequirePermissions` 校验 → Handler 执行业务 → Service 层业务规则检查 → 触发器维护一致性。

**用户管理流程**：Handler 从 JWT Claims 获取当前用户信息（ID 和角色），传递给 Service 层进行业务规则检查（如自我操作限制、Manager 对 Admin 的限制等），确保即使绕过前端检查，后端也会阻止违规操作。

## RBAC 测试策略

集成测试文件：`test/integration/rbac_roles_integration_test.go`
- Manager 全访问：覆盖创建订单、计划、版型、任务、日志；可查看任务日志列表。
- Worker 日志与任务：可创建/作废日志、阅读任务与按版型列任务；禁止查看任务日志列表（期待 `403`）。
- Pattern Maker 模块访问：可创建计划/版型/任务；**不能发布计划**（无 `plan:publish` 权限）；在发布前可更新版型名称；**不能修改已发布计划的备注**（Handler 层业务规则）；可通过 `layout:read` 权限在计划详情中查看任务信息；禁止查看任务管理页面和任务日志列表（期待 `403`）。
- Admin 订单与日志：可创建订单并查看任务参与者列表。

运行方式：
- 推荐脚本：`./test/run.ps1 -Keep`（启动 Postgres、设置 `DATABASE_URL` 并运行测试）。
- 手动方式：
  1) `docker compose -f ./test/docker-compose.yml up -d`
  2) `PowerShell` 中设置：`$Env:DATABASE_URL = 'postgres://cutrix:cutrix@localhost:5432/cutrix?sslmode=disable'`
  3) 运行：`go test ./test/integration -run RBAC -v`
- 测试数据唯一性：订单号使用 `UnixNano` 生成避免重复主键冲突。

## 设计思路与扩展
- 解耦：认证与权限分离；路由层“先认证再鉴权”，服务/数据库层提供业务兜底。
- 可维护性：权限字符串 + 通配符简化配置；角色映射集中管理，业务端点可组合角色/权限两类保护。
- 一致性：数据库触发器保证层数累计、状态变更、作废一致性；防止应用越权或遗漏校验。
- 扩展建议：
  - 将 `RolePermissionsMap` 配置化（来自 DB/配置文件），支持多租户或项目自定义。
  - 增加审计日志与权限变更历史。
  - 引入“资源级权限”（按订单/计划归属进一步限制）。