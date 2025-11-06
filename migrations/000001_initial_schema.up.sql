-- Initial schema for laying-up domain using a dedicated schema
-- Adopt concise names under the `production` schema and keep `Workers` in public.

BEGIN;

-- Create domain schema
CREATE SCHEMA IF NOT EXISTS production;

-- Workers (shared across domains)
CREATE TABLE IF NOT EXISTS public.Workers (
    worker_id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    notes VARCHAR(150),
    password_hash VARCHAR(255),
    role VARCHAR(20) NOT NULL DEFAULT 'worker' CHECK (role IN ('admin', 'manager', 'worker', 'pattern_maker')),
    worker_group VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Orders (core record)
CREATE TABLE IF NOT EXISTS production.orders (
    order_id SERIAL PRIMARY KEY,
    order_number VARCHAR(100) NOT NULL UNIQUE,
    style_number VARCHAR(50) NOT NULL,
    customer_name VARCHAR(100),
    order_date DATE,
    delivery_date DATE,
    notes TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'cancelled')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Order items (color-size-quantity)
CREATE TABLE IF NOT EXISTS production.order_items (
    item_id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES production.orders(order_id) ON DELETE CASCADE,
    color VARCHAR(50) NOT NULL,
    size VARCHAR(50) NOT NULL,
    quantity INT NOT NULL,
    UNIQUE(order_id, color, size)
);

-- Plans (must be based on an order)
CREATE TABLE IF NOT EXISTS production.plans (
    plan_id SERIAL PRIMARY KEY,
    plan_name VARCHAR(255) NOT NULL,
    order_id INT NOT NULL REFERENCES production.orders(order_id) ON DELETE CASCADE,
    planned_start_date DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'in_production', 'completed', 'paused', 'cancelled')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Cutting layouts
CREATE TABLE IF NOT EXISTS production.cutting_layouts (
    layout_id SERIAL PRIMARY KEY,
    plan_id INT NOT NULL REFERENCES production.plans(plan_id) ON DELETE CASCADE,
    layout_name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Layout size ratios
CREATE TABLE IF NOT EXISTS production.layout_size_ratios (
    ratio_id SERIAL PRIMARY KEY,
    layout_id INT NOT NULL REFERENCES production.cutting_layouts(layout_id) ON DELETE CASCADE,
    size VARCHAR(50) NOT NULL,
    ratio INT NOT NULL
);

-- Tasks (拉布)
CREATE TABLE IF NOT EXISTS production.tasks (
    task_id SERIAL PRIMARY KEY,
    layout_id INT NOT NULL REFERENCES production.cutting_layouts(layout_id) ON DELETE CASCADE,
    color VARCHAR(50) NOT NULL,
    planned_layers INT,
    completed_layers INT DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Logs (worker submissions)
CREATE TABLE IF NOT EXISTS production.logs (
    log_id SERIAL PRIMARY KEY,
    task_id INT NOT NULL REFERENCES production.tasks(task_id) ON DELETE CASCADE,
    worker_id INT REFERENCES public.Workers(worker_id) ON DELETE SET NULL,
    worker_name VARCHAR(50), -- redundant to preserve history when worker deleted
    layers_completed INT NOT NULL CHECK (layers_completed > 0),
    log_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes TEXT,
    voided BOOLEAN NOT NULL DEFAULT false,
    void_reason TEXT,
    voided_at TIMESTAMP,
    voided_by INT REFERENCES public.Workers(worker_id) ON DELETE SET NULL,
    voided_by_name VARCHAR(50)
);

-- Trigger functions (schema-qualified)
CREATE OR REPLACE FUNCTION production.update_completed_layers()
RETURNS TRIGGER AS $$
DECLARE
    current_planned_layers INT;
    current_completed_layers INT;
BEGIN
    SELECT planned_layers, completed_layers INTO current_planned_layers, current_completed_layers
    FROM production.tasks
    WHERE task_id = NEW.task_id;

    IF current_planned_layers IS NULL THEN
        RETURN NEW;
    END IF;

    IF NEW.layers_completed <= 0 THEN
        RAISE EXCEPTION '完成层数必须大于0';
    END IF;


    UPDATE production.tasks
    SET
        completed_layers = current_completed_layers + NEW.layers_completed,
        status = CASE
            WHEN current_planned_layers IS NOT NULL
                 AND (current_completed_layers + NEW.layers_completed) >= current_planned_layers THEN 'completed'
            WHEN (current_completed_layers + NEW.layers_completed) > 0 THEN 'in_progress'
            ELSE 'pending'
        END,
        updated_at = CURRENT_TIMESTAMP
    WHERE task_id = NEW.task_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION production.set_log_worker_name()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.worker_name IS NULL THEN
        SELECT name INTO NEW.worker_name FROM public.Workers WHERE worker_id = NEW.worker_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers on logs
DROP TRIGGER IF EXISTS trg_before_log_insert_set_name ON production.logs;
CREATE TRIGGER trg_before_log_insert_set_name
BEFORE INSERT ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.set_log_worker_name();

DROP TRIGGER IF EXISTS trg_after_log_insert ON production.logs;
CREATE TRIGGER trg_after_log_insert
AFTER INSERT ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.update_completed_layers();

-- Seed data (optional, idempotent)
INSERT INTO public.Workers (name, password_hash, role) VALUES
('admin', '$2a$12$gwwSt9.uKHrxcCffsmgc0OvsdcRa1qldHE4bR/XrKNlYMK6IRyGty', 'admin'),
('manager', '$2a$12$kFFQ9IF1WV3Ky4VefgLFfOJl.bD1Ef/9bQC/7Ghc.IlFRRCjosya2', 'manager'),
('张三', '$2a$12$gwwSt9.uKHrxcCffsmgc0OvsdcRa1qldHE4bR/XrKNlYMK6IRyGty', 'worker'),
('王五', '$2a$12$gwwSt9.uKHrxcCffsmgc0OvsdcRa1qldHE4bR/XrKNlYMK6IRyGty', 'pattern_maker')
ON CONFLICT (name) DO NOTHING;

INSERT INTO production.orders (order_number, style_number, customer_name, order_date, delivery_date) VALUES 
('ORD-2024-001', 'BEE3TS111', '客户A', '2024-01-15', '2024-02-15'),
('ORD-2024-002', 'BEE3TS112', '客户B', '2024-01-16', '2024-02-20'),
('ORD-2024-003', 'BEE3TS113', '客户C', '2024-01-17', '2024-02-25')
ON CONFLICT (order_number) DO NOTHING;

-- Corrections: apply delta on void/unvoid of logs
CREATE OR REPLACE FUNCTION production.apply_log_void_delta()
RETURNS TRIGGER AS $$
DECLARE
    planned INT;
    completed INT;
    new_completed INT;
BEGIN
    SELECT planned_layers, completed_layers INTO planned, completed
    FROM production.tasks
    WHERE task_id = NEW.task_id;

    -- 仅在 voided 标志变化时处理
    IF NEW.voided = TRUE AND (OLD.voided IS DISTINCT FROM TRUE) THEN
        new_completed := completed - OLD.layers_completed;
        IF new_completed < 0 THEN
            RAISE EXCEPTION '作废后完成层数不能小于0 (当前: %, 作废层数: %, 任务: %)', completed, OLD.layers_completed, NEW.task_id;
        END IF;
        UPDATE production.tasks
        SET
            completed_layers = new_completed,
            status = CASE
                WHEN planned IS NOT NULL AND new_completed >= planned THEN 'completed'
                WHEN new_completed > 0 THEN 'in_progress'
                ELSE 'pending'
            END,
            updated_at = CURRENT_TIMESTAMP
        WHERE task_id = NEW.task_id;

    ELSIF NEW.voided = FALSE AND (OLD.voided IS DISTINCT FROM FALSE) THEN
        new_completed := completed + OLD.layers_completed;
        UPDATE production.tasks
        SET
            completed_layers = new_completed,
            status = CASE
                WHEN planned IS NOT NULL AND new_completed >= planned THEN 'completed'
                WHEN new_completed > 0 THEN 'in_progress'
                ELSE 'pending'
            END,
            updated_at = CURRENT_TIMESTAMP
        WHERE task_id = NEW.task_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Fill voided_by_name before applying deltas
CREATE OR REPLACE FUNCTION production.set_voided_by_name()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.voided = TRUE AND (OLD.voided IS DISTINCT FROM TRUE) THEN
        IF NEW.voided_by_name IS NULL AND NEW.voided_by IS NOT NULL THEN
            SELECT name INTO NEW.voided_by_name FROM public.Workers WHERE worker_id = NEW.voided_by;
        END IF;
        IF NEW.voided_at IS NULL THEN
            NEW.voided_at := CURRENT_TIMESTAMP;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_before_log_update_set_voided_by_name ON production.logs;
CREATE TRIGGER trg_before_log_update_set_voided_by_name
BEFORE UPDATE OF voided ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.set_voided_by_name();

-- Trigger on logs to apply delta when void status changes
DROP TRIGGER IF EXISTS trg_after_log_update_void ON production.logs;
CREATE TRIGGER trg_after_log_update_void
AFTER UPDATE OF voided ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.apply_log_void_delta();

COMMIT;