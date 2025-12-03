-- Teardown schema for CUTRIX production domain
-- Drop triggers first, then functions, then tables, then schema

BEGIN;

-- =====================
-- Drop Triggers
-- =====================
-- Orders
DROP TRIGGER IF EXISTS trg_guard_orders_update ON production.orders;

-- Order items
DROP TRIGGER IF EXISTS trg_prevent_update_order_items ON production.order_items;

-- Plans
DROP TRIGGER IF EXISTS trg_guard_plan_update ON production.plans;
DROP TRIGGER IF EXISTS trg_guard_plan_publish ON production.plans;
DROP TRIGGER IF EXISTS trg_after_plan_publish_mark_tasks ON production.plans;

-- Cutting layouts
DROP TRIGGER IF EXISTS trg_guard_layouts_by_plan_status ON production.cutting_layouts;

-- Layout size ratios
DROP TRIGGER IF EXISTS trg_guard_layout_size ON production.layout_size_ratios;
DROP TRIGGER IF EXISTS trg_after_ratio_change_guard_positive ON production.layout_size_ratios;
DROP TRIGGER IF EXISTS trg_guard_ratios_by_plan_status ON production.layout_size_ratios;

-- Tasks
DROP TRIGGER IF EXISTS trg_guard_task_color ON production.tasks;
DROP TRIGGER IF EXISTS trg_after_task_change_update_plan ON production.tasks;
DROP TRIGGER IF EXISTS trg_guard_tasks_by_plan_status ON production.tasks;

-- Logs
DROP TRIGGER IF EXISTS trg_before_log_insert_set_name ON production.logs;
DROP TRIGGER IF EXISTS trg_guard_log_insert_task_status ON production.logs;
DROP TRIGGER IF EXISTS trg_after_log_insert ON production.logs;
DROP TRIGGER IF EXISTS trg_before_log_update_set_voided_by_name ON production.logs;
DROP TRIGGER IF EXISTS trg_after_log_update_void ON production.logs;
DROP TRIGGER IF EXISTS trg_guard_logs_update ON production.logs;
DROP TRIGGER IF EXISTS trg_prevent_logs_delete ON production.logs;

-- =====================
-- Drop Functions
-- =====================
-- Orders
DROP FUNCTION IF EXISTS production.guard_orders_update();

-- Order items
DROP FUNCTION IF EXISTS production.prevent_order_items_update();

-- Plans
DROP FUNCTION IF EXISTS production.update_plan_progress(INT);
DROP FUNCTION IF EXISTS production.guard_plan_update();
DROP FUNCTION IF EXISTS production.guard_plan_publish();
DROP FUNCTION IF EXISTS production.publish_plan_mark_tasks();

-- Cutting layouts
DROP FUNCTION IF EXISTS production.guard_layouts_by_plan_status();

-- Layout size ratios
DROP FUNCTION IF EXISTS production.ensure_layout_size_in_order();
DROP FUNCTION IF EXISTS production.guard_layout_ratios_positive();
DROP FUNCTION IF EXISTS production.guard_ratios_by_plan_status();

-- Tasks
DROP FUNCTION IF EXISTS production.ensure_task_color_in_order();
DROP FUNCTION IF EXISTS production.update_plan_on_task_change();
DROP FUNCTION IF EXISTS production.guard_tasks_by_plan_status();

-- Logs
DROP FUNCTION IF EXISTS production.set_log_worker_name();
DROP FUNCTION IF EXISTS production.guard_log_insert_task_status();
DROP FUNCTION IF EXISTS production.update_completed_layers();
DROP FUNCTION IF EXISTS production.set_voided_by_name();
DROP FUNCTION IF EXISTS production.apply_log_void_delta();
DROP FUNCTION IF EXISTS production.guard_logs_update();
DROP FUNCTION IF EXISTS production.prevent_logs_delete();

-- =====================
-- Drop Indexes
-- =====================
DROP INDEX IF EXISTS public.users_single_active_manager_idx;
DROP INDEX IF EXISTS public.users_single_active_admin_idx;

-- =====================
-- Drop Tables
-- =====================
DROP TABLE IF EXISTS production.logs;
DROP TABLE IF EXISTS production.tasks;
DROP TABLE IF EXISTS production.layout_size_ratios;
DROP TABLE IF EXISTS production.cutting_layouts;
DROP TABLE IF EXISTS production.plans;
DROP TABLE IF EXISTS production.order_items;
DROP TABLE IF EXISTS production.orders;
DROP TABLE IF EXISTS public.users;

-- =====================
-- Drop Schema
-- =====================
DROP SCHEMA IF EXISTS production;

COMMIT;