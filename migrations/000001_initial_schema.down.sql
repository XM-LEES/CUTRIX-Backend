-- Rollback for initial laying-up schema (production schema)
BEGIN;

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_after_log_insert ON production.logs;
DROP TRIGGER IF EXISTS trg_before_log_insert_set_name ON production.logs;
DROP TRIGGER IF EXISTS trg_before_log_update_set_voided_by_name ON production.logs;
DROP TRIGGER IF EXISTS trg_after_log_update_void ON production.logs;

-- Drop trigger functions
DROP FUNCTION IF EXISTS production.set_voided_by_name();
DROP FUNCTION IF EXISTS production.apply_log_void_delta();
DROP FUNCTION IF EXISTS production.update_completed_layers();
DROP FUNCTION IF EXISTS production.set_log_worker_name();

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS production.logs;
DROP TABLE IF EXISTS production.tasks;
DROP TABLE IF EXISTS production.layout_size_ratios;
DROP TABLE IF EXISTS production.cutting_layouts;
DROP TABLE IF EXISTS production.plans;
DROP TABLE IF EXISTS production.order_items;
DROP TABLE IF EXISTS production.orders;
DROP TABLE IF EXISTS public.Workers;

-- Drop schema
DROP SCHEMA IF EXISTS production;

COMMIT;