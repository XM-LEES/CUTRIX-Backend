-- Initial schema for CUTRIX production domain
-- Unified order: tables first, then functions and triggers, finally seed data

BEGIN;

-- =====================
-- Schema
-- =====================
CREATE SCHEMA IF NOT EXISTS production;

-- =====================
-- Tables
-- =====================
-- Users (shared across domains)
CREATE TABLE IF NOT EXISTS public.users (
    user_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255),
    role VARCHAR(20) NOT NULL DEFAULT 'worker' CHECK (role IN ('admin', 'manager', 'worker', 'pattern_maker')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    user_group VARCHAR(50),
    note VARCHAR(255)
);

-- Orders
CREATE TABLE IF NOT EXISTS production.orders (
    order_id SERIAL PRIMARY KEY,
    order_number VARCHAR(100) NOT NULL UNIQUE,
    style_number VARCHAR(50) NOT NULL,
    customer_name VARCHAR(100),
    note TEXT,
    order_start_date TIMESTAMP,
    order_finish_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Order items (color-size-quantity)
CREATE TABLE IF NOT EXISTS production.order_items (
    item_id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES production.orders(order_id) ON DELETE CASCADE,
    color VARCHAR(50) NOT NULL,
    size VARCHAR(30) NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    UNIQUE(order_id, color, size)
);

-- Plans (must be based on an order)
CREATE TABLE IF NOT EXISTS production.plans (
    plan_id SERIAL PRIMARY KEY,
    plan_name VARCHAR(255) NOT NULL,
    order_id INT NOT NULL REFERENCES production.orders(order_id) ON DELETE CASCADE,
    note TEXT,
    planned_publish_date TIMESTAMP,                             -- when plan is published (auto)
    planned_finish_date TIMESTAMP,                              -- when plan is finished (auto)
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_progress','completed','frozen'))
);

-- Cutting layouts
CREATE TABLE IF NOT EXISTS production.cutting_layouts (
    layout_id SERIAL PRIMARY KEY,
    plan_id INT NOT NULL REFERENCES production.plans(plan_id) ON DELETE CASCADE,
    layout_name VARCHAR(100) NOT NULL,
    note TEXT
);

-- Layout size ratios
CREATE TABLE IF NOT EXISTS production.layout_size_ratios (
    ratio_id SERIAL PRIMARY KEY,
    layout_id INT NOT NULL REFERENCES production.cutting_layouts(layout_id) ON DELETE CASCADE,
    size VARCHAR(50) NOT NULL,
    ratio INT NOT NULL CHECK (ratio >= 0),
    UNIQUE(layout_id, size)
);

-- Tasks (拉布)
CREATE TABLE IF NOT EXISTS production.tasks (
    task_id SERIAL PRIMARY KEY,
    layout_id INT NOT NULL REFERENCES production.cutting_layouts(layout_id) ON DELETE CASCADE,
    color VARCHAR(50) NOT NULL,
    planned_layers INT NOT NULL CHECK (planned_layers > 0),
    completed_layers INT DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed'))
);

-- Logs (worker submissions)
CREATE TABLE IF NOT EXISTS production.logs (
    log_id SERIAL PRIMARY KEY,
    task_id INT NOT NULL REFERENCES production.tasks(task_id) ON DELETE CASCADE,
    worker_id INT REFERENCES public.users(user_id) ON DELETE SET NULL,
    worker_name VARCHAR(50),
    layers_completed INT NOT NULL CHECK (layers_completed > 0),
    log_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    note TEXT,
    voided BOOLEAN NOT NULL DEFAULT false,
    void_reason TEXT,
    voided_at TIMESTAMP,
    voided_by INT REFERENCES public.users(user_id) ON DELETE SET NULL,
    voided_by_name VARCHAR(50)
);

-- =====================
-- Indexes
-- =====================
-- Unique constraints to ensure only one active admin and one active manager
CREATE UNIQUE INDEX IF NOT EXISTS users_single_active_admin_idx 
ON public.users (role) 
WHERE role = 'admin' AND is_active = true;

CREATE UNIQUE INDEX IF NOT EXISTS users_single_active_manager_idx 
ON public.users (role) 
WHERE role = 'manager' AND is_active = true;

-- =====================
-- Functions & Triggers (ordered by table)
-- =====================
-- Orders
CREATE OR REPLACE FUNCTION production.guard_orders_update()
RETURNS TRIGGER AS $$
BEGIN
    IF (NEW.order_start_date IS DISTINCT FROM OLD.order_start_date)
        OR (NEW.order_number IS DISTINCT FROM OLD.order_number)
        OR (NEW.style_number IS DISTINCT FROM OLD.style_number)
        OR (NEW.customer_name IS DISTINCT FROM OLD.customer_name) THEN
        RAISE EXCEPTION '仅允许更新 note 和 order_finish_date';
    END IF;
    NEW.updated_at := CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_orders_update ON production.orders;
CREATE TRIGGER trg_guard_orders_update
BEFORE UPDATE ON production.orders
FOR EACH ROW
EXECUTE FUNCTION production.guard_orders_update();

-- Order items
CREATE OR REPLACE FUNCTION production.prevent_order_items_update()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION '订单项不可修改';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_update_order_items ON production.order_items;
CREATE TRIGGER trg_prevent_update_order_items
BEFORE UPDATE ON production.order_items
FOR EACH ROW
EXECUTE FUNCTION production.prevent_order_items_update();

-- Plans

CREATE OR REPLACE FUNCTION production.update_plan_progress(p_plan_id INT)
RETURNS VOID AS $$
DECLARE
    v_total INT;
    v_completed INT;
BEGIN
    -- Count tasks under the plan
    SELECT COUNT(*) INTO v_total
    FROM production.tasks t
    JOIN production.cutting_layouts l ON l.layout_id = t.layout_id
    WHERE l.plan_id = p_plan_id;

    SELECT COUNT(*) INTO v_completed
    FROM production.tasks t
    JOIN production.cutting_layouts l ON l.layout_id = t.layout_id
    WHERE l.plan_id = p_plan_id AND t.status = 'completed';

    -- Only mark completed when all tasks are completed and there is at least one task
    UPDATE production.plans
    SET
        status = CASE
            WHEN v_total > 0 AND v_completed = v_total THEN 'completed'
            WHEN v_total > 0 AND v_completed <> v_total AND status = 'completed' THEN 'in_progress'
            ELSE status
        END,
        planned_finish_date = CASE
            WHEN v_total > 0 AND v_completed = v_total THEN CURRENT_TIMESTAMP
            ELSE planned_finish_date
        END
    WHERE plan_id = p_plan_id AND status <> 'frozen';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION production.guard_plan_update()
RETURNS TRIGGER AS $$
DECLARE
    v_total INT;
    v_completed INT;
BEGIN
    IF production.is_plan_adjustment_context() THEN
        RETURN NEW;
    END IF;

    -- Disallow manual changes of publish/finish date when status unchanged
    IF NEW.status = OLD.status THEN
        IF NEW.planned_publish_date IS DISTINCT FROM OLD.planned_publish_date THEN
            RAISE EXCEPTION '发布日期由发布动作自动记录，禁止手动修改';
        END IF;
        IF NEW.planned_finish_date IS DISTINCT FROM OLD.planned_finish_date THEN
            RAISE EXCEPTION '完成时间由系统自动记录，禁止手动修改';
        END IF;
    END IF;

    -- Disallow pending -> completed direct transition
    IF OLD.status = 'pending' AND NEW.status = 'completed' THEN
        RAISE EXCEPTION '禁止从 pending 直接变更为 completed';
    END IF;
    -- Disallow pending -> frozen direct transition
    IF OLD.status = 'pending' AND NEW.status = 'frozen' THEN
        RAISE EXCEPTION '仅允许在 completed 状态下冻结计划';
    END IF;

    -- Publish: pending -> in_progress, require tasks exist; publish date set by AFTER trigger
    IF NEW.status = 'in_progress' AND (OLD.status IS DISTINCT FROM 'in_progress') THEN
        IF OLD.status <> 'pending' THEN
            RAISE EXCEPTION '仅允许从 pending 发布到 in_progress';
        END IF;
        SELECT COUNT(*) INTO v_total
        FROM production.tasks t
        JOIN production.cutting_layouts l ON l.layout_id = t.layout_id
        WHERE l.plan_id = OLD.plan_id;
        IF v_total <= 0 THEN
            RAISE EXCEPTION '发布失败：该计划必须包含至少一个任务';
        END IF;
        -- publish date will be set by AFTER trigger, do not set NEW.planned_publish_date here
    END IF;

    -- After publish: enforce restrictions and controlled transitions
    IF OLD.status IN ('in_progress','completed','frozen') THEN
        IF NEW.status = OLD.status THEN
            IF NEW.plan_name IS DISTINCT FROM OLD.plan_name OR NEW.order_id IS DISTINCT FROM OLD.order_id THEN
                RAISE EXCEPTION '计划发布后仅允许修改备注';
            END IF;
        ELSE
            IF NEW.status = 'completed' THEN
                SELECT COUNT(*) INTO v_total
                FROM production.tasks t
                JOIN production.cutting_layouts l ON l.layout_id = t.layout_id
                WHERE l.plan_id = OLD.plan_id;
                SELECT COUNT(*) INTO v_completed
                FROM production.tasks t
                JOIN production.cutting_layouts l ON l.layout_id = t.layout_id
                WHERE l.plan_id = OLD.plan_id AND t.status = 'completed';
                IF v_total > 0 AND v_completed = v_total THEN
                    NEW.planned_finish_date := CURRENT_TIMESTAMP;
                ELSE
                    RAISE EXCEPTION '状态变更为已完成失败：仍有未完成任务';
                END IF;
            ELSIF NEW.status = 'frozen' THEN
                IF OLD.status <> 'completed' THEN
                    RAISE EXCEPTION '仅允许在 completed 状态下冻结计划';
                END IF;
                NEW.planned_finish_date := OLD.planned_finish_date;
            ELSE
                RAISE EXCEPTION '计划发布后不允许更改为该状态';
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_plan_publish ON production.plans;

-- BEFORE trigger: when status changes to in_progress, set publish date once
CREATE OR REPLACE FUNCTION production.guard_plan_publish()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'in_progress' AND OLD.status = 'pending' THEN
        IF NEW.planned_publish_date IS NULL THEN
            NEW.planned_publish_date := CURRENT_TIMESTAMP;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_guard_plan_publish
BEFORE UPDATE OF status ON production.plans
FOR EACH ROW
EXECUTE FUNCTION production.guard_plan_publish(); -- remove legacy if any
DROP TRIGGER IF EXISTS trg_guard_plan_update ON production.plans;
CREATE TRIGGER trg_guard_plan_update
BEFORE UPDATE ON production.plans
FOR EACH ROW
EXECUTE FUNCTION production.guard_plan_update();

-- After publish, set publish date and mark tasks in_progress
CREATE OR REPLACE FUNCTION production.publish_plan_mark_tasks()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'in_progress' AND (OLD.status IS DISTINCT FROM 'in_progress') THEN
        -- publish date handled by BEFORE trigger; do not write here

        -- Mark pending tasks as in_progress under the plan
        UPDATE production.tasks t
        SET status = 'in_progress'
        FROM production.cutting_layouts l
        WHERE l.layout_id = t.layout_id
          AND l.plan_id = NEW.plan_id
          AND t.status = 'pending';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_after_plan_publish_mark_tasks ON production.plans;
CREATE TRIGGER trg_after_plan_publish_mark_tasks
AFTER UPDATE OF status ON production.plans
FOR EACH ROW
EXECUTE FUNCTION production.publish_plan_mark_tasks();


--
-- Helper guard for plan deletion context
CREATE OR REPLACE FUNCTION production.is_plan_delete_context()
RETURNS BOOLEAN AS $$
DECLARE
    v_setting TEXT;
BEGIN
    v_setting := current_setting('cutrix.plan_delete_flag', true);
    RETURN COALESCE(v_setting::BOOLEAN, FALSE);
EXCEPTION WHEN others THEN
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;


CREATE OR REPLACE FUNCTION production.is_plan_adjustment_context()
RETURNS BOOLEAN AS $$
DECLARE
    v_setting TEXT;
BEGIN
    v_setting := current_setting('cutrix.plan_adjustment_flag', true);
    RETURN COALESCE(v_setting::BOOLEAN, FALSE);
EXCEPTION WHEN others THEN
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION production.is_ratios_replace_context()
RETURNS BOOLEAN AS $$
DECLARE
    v_setting TEXT;
BEGIN
    v_setting := current_setting('cutrix.ratios_replace_flag', true);
    RETURN COALESCE(v_setting::BOOLEAN, FALSE);
EXCEPTION WHEN others THEN
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION production.is_task_adjustment_context()
RETURNS BOOLEAN AS $$
DECLARE
    v_setting TEXT;
BEGIN
    v_setting := current_setting('cutrix.task_adjustment_flag', true);
    RETURN COALESCE(v_setting::BOOLEAN, FALSE);
EXCEPTION WHEN others THEN
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;


-- Cutting layouts


CREATE OR REPLACE FUNCTION production.guard_layouts_by_plan_status()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(20);
    v_plan_id INT;
BEGIN
    IF production.is_plan_delete_context() THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    v_plan_id := COALESCE(NEW.plan_id, OLD.plan_id);
    SELECT status INTO v_status FROM production.plans WHERE plan_id = v_plan_id;

    IF v_status IN ('in_progress','completed','frozen') THEN
        RAISE EXCEPTION '计划发布后布局不可增删改 (plan=%)', v_plan_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_layouts_by_plan_status ON production.cutting_layouts;
CREATE TRIGGER trg_guard_layouts_by_plan_status
BEFORE INSERT OR UPDATE OR DELETE ON production.cutting_layouts
FOR EACH ROW
EXECUTE FUNCTION production.guard_layouts_by_plan_status();

-- Layout size ratios
CREATE OR REPLACE FUNCTION production.ensure_layout_size_in_order()
RETURNS TRIGGER AS $$
DECLARE
    v_order_id INT;
BEGIN
    SELECT p.order_id INTO v_order_id
    FROM production.cutting_layouts l
    JOIN production.plans p ON p.plan_id = l.plan_id
    WHERE l.layout_id = NEW.layout_id;

    IF v_order_id IS NULL THEN
        RAISE EXCEPTION '布局未关联到计划: %', NEW.layout_id;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM production.order_items oi
        WHERE oi.order_id = v_order_id AND oi.size = NEW.size
    ) THEN
        RAISE EXCEPTION '布局尺码 % 必须属于订单的尺码集合', NEW.size;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_layout_size ON production.layout_size_ratios;
CREATE TRIGGER trg_guard_layout_size
BEFORE INSERT OR UPDATE ON production.layout_size_ratios
FOR EACH ROW
EXECUTE FUNCTION production.ensure_layout_size_in_order();



CREATE OR REPLACE FUNCTION production.guard_layout_ratios_positive()
RETURNS TRIGGER AS $$
DECLARE
    v_sum INT;
BEGIN
    IF production.is_plan_delete_context() OR production.is_ratios_replace_context() THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    IF TG_OP = 'UPDATE' THEN
        SELECT COALESCE(SUM(ratio),0) INTO v_sum FROM production.layout_size_ratios WHERE layout_id = OLD.layout_id;
        IF v_sum <= 0 THEN
            RAISE EXCEPTION '布局 % 的尺码比例总和必须大于0', OLD.layout_id;
        END IF;
        SELECT COALESCE(SUM(ratio),0) INTO v_sum FROM production.layout_size_ratios WHERE layout_id = NEW.layout_id;
        IF v_sum <= 0 THEN
            RAISE EXCEPTION '布局 % 的尺码比例总和必须大于0', NEW.layout_id;
        END IF;
    ELSE
        SELECT COALESCE(SUM(ratio),0) INTO v_sum FROM production.layout_size_ratios WHERE layout_id = COALESCE(NEW.layout_id, OLD.layout_id);
        IF v_sum <= 0 THEN
            RAISE EXCEPTION '布局 % 的尺码比例总和必须大于0', COALESCE(NEW.layout_id, OLD.layout_id);
        END IF;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_after_ratio_change_guard_positive ON production.layout_size_ratios;
CREATE TRIGGER trg_after_ratio_change_guard_positive
AFTER INSERT OR UPDATE OR DELETE ON production.layout_size_ratios
FOR EACH ROW
EXECUTE FUNCTION production.guard_layout_ratios_positive();

CREATE OR REPLACE FUNCTION production.guard_ratios_by_plan_status()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(20);
    v_layout_id INT;
    v_plan_id INT;
BEGIN
    IF production.is_plan_delete_context() THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    v_layout_id := COALESCE(NEW.layout_id, OLD.layout_id);
    SELECT l.plan_id, p.status INTO v_plan_id, v_status
    FROM production.cutting_layouts l
    JOIN production.plans p ON p.plan_id = l.plan_id
    WHERE l.layout_id = v_layout_id;

    IF v_status IN ('in_progress','completed','frozen') THEN
        RAISE EXCEPTION '计划发布后比例不可增删改 (plan=%)', v_plan_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_ratios_by_plan_status ON production.layout_size_ratios;
CREATE TRIGGER trg_guard_ratios_by_plan_status
BEFORE INSERT OR UPDATE OR DELETE ON production.layout_size_ratios
FOR EACH ROW
EXECUTE FUNCTION production.guard_ratios_by_plan_status();

-- Tasks
CREATE OR REPLACE FUNCTION production.ensure_task_color_in_order()
RETURNS TRIGGER AS $$
DECLARE
    v_order_id INT;
BEGIN
    SELECT p.order_id INTO v_order_id
    FROM production.cutting_layouts l
    JOIN production.plans p ON p.plan_id = l.plan_id
    WHERE l.layout_id = NEW.layout_id;

    IF v_order_id IS NULL THEN
        RAISE EXCEPTION '任务未关联到计划: %', NEW.layout_id;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM production.order_items oi
        WHERE oi.order_id = v_order_id AND oi.color = NEW.color
    ) THEN
        RAISE EXCEPTION '任务颜色 % 必须出现在订单项中', NEW.color;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_task_color ON production.tasks;
CREATE TRIGGER trg_guard_task_color
BEFORE INSERT OR UPDATE ON production.tasks
FOR EACH ROW
EXECUTE FUNCTION production.ensure_task_color_in_order();

CREATE OR REPLACE FUNCTION production.update_plan_on_task_change()
RETURNS TRIGGER AS $$
DECLARE
    v_plan_id INT;
    v_layout_id INT;
BEGIN
    v_layout_id := COALESCE(NEW.layout_id, OLD.layout_id);
    SELECT plan_id INTO v_plan_id FROM production.cutting_layouts WHERE layout_id = v_layout_id;
    IF v_plan_id IS NOT NULL THEN
        PERFORM production.update_plan_progress(v_plan_id);
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_after_task_change_update_plan ON production.tasks;
CREATE TRIGGER trg_after_task_change_update_plan
AFTER INSERT OR UPDATE OR DELETE ON production.tasks
FOR EACH ROW
EXECUTE FUNCTION production.update_plan_on_task_change();

CREATE OR REPLACE FUNCTION production.guard_tasks_by_plan_status()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(20);
    v_layout_id INT;
    v_plan_id INT;
BEGIN
    IF production.is_plan_delete_context() OR production.is_task_adjustment_context() THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    v_layout_id := COALESCE(NEW.layout_id, OLD.layout_id);
    SELECT l.plan_id, p.status INTO v_plan_id, v_status
    FROM production.cutting_layouts l
    JOIN production.plans p ON p.plan_id = l.plan_id
    WHERE l.layout_id = v_layout_id;

    IF v_status = 'frozen' THEN
        RAISE EXCEPTION '计划冻结后不能增删改任务 (plan=%)', v_plan_id;
    ELSIF v_status = 'completed' THEN
        RAISE EXCEPTION '计划完成后不能增删改任务 (plan=%)', v_plan_id;
    ELSIF v_status = 'in_progress' THEN
        IF TG_OP = 'INSERT' THEN
            RAISE EXCEPTION '计划发布后不能新增任务 (plan=%)', v_plan_id;
        ELSIF TG_OP = 'DELETE' THEN
            RAISE EXCEPTION '计划发布后不能删除任务 (plan=%)', v_plan_id;
        ELSIF TG_OP = 'UPDATE' THEN
            IF NEW.planned_layers IS DISTINCT FROM OLD.planned_layers
               OR NEW.color IS DISTINCT FROM OLD.color
               OR NEW.layout_id IS DISTINCT FROM OLD.layout_id THEN
                RAISE EXCEPTION '计划发布后任务仅允许更新完成层数';
            END IF;
        END IF;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_tasks_by_plan_status ON production.tasks;
CREATE TRIGGER trg_guard_tasks_by_plan_status
BEFORE INSERT OR UPDATE OR DELETE ON production.tasks
FOR EACH ROW
EXECUTE FUNCTION production.guard_tasks_by_plan_status();

-- Logs
CREATE OR REPLACE FUNCTION production.set_log_worker_name()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.worker_name IS NULL AND NEW.worker_id IS NOT NULL THEN
        SELECT name INTO NEW.worker_name FROM public.users WHERE user_id = NEW.worker_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_before_log_insert_set_name ON production.logs;
CREATE TRIGGER trg_before_log_insert_set_name
BEFORE INSERT ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.set_log_worker_name();

-- Guard: only allow log submissions to in_progress tasks
CREATE OR REPLACE FUNCTION production.guard_log_insert_task_status()
RETURNS TRIGGER AS $$
DECLARE
    v_status VARCHAR(20);
BEGIN
    SELECT status INTO v_status FROM production.tasks WHERE task_id = NEW.task_id;
    IF v_status IS DISTINCT FROM 'in_progress' THEN
        RAISE EXCEPTION '仅允许向 in_progress 任务提交日志 (当前状态: %)', v_status;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_log_insert_task_status ON production.logs;
CREATE TRIGGER trg_guard_log_insert_task_status
BEFORE INSERT ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.guard_log_insert_task_status();

CREATE OR REPLACE FUNCTION production.update_completed_layers()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.layers_completed <= 0 THEN
        RAISE EXCEPTION '完成层数必须大于0';
    END IF;

    UPDATE production.tasks
    SET 
        completed_layers = completed_layers + NEW.layers_completed,
        status = CASE
            WHEN planned_layers IS NOT NULL AND completed_layers + NEW.layers_completed >= planned_layers THEN 'completed'
            WHEN completed_layers + NEW.layers_completed > 0 THEN 'in_progress'
            ELSE status
        END
    WHERE task_id = NEW.task_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_after_log_insert ON production.logs;
CREATE TRIGGER trg_after_log_insert
AFTER INSERT ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.update_completed_layers();

CREATE OR REPLACE FUNCTION production.set_voided_by_name()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.voided = TRUE AND (OLD.voided IS DISTINCT FROM TRUE) THEN
        IF NEW.voided_by_name IS NULL AND NEW.voided_by IS NOT NULL THEN
            SELECT name INTO NEW.voided_by_name FROM public.users WHERE user_id = NEW.voided_by;
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

CREATE OR REPLACE FUNCTION production.apply_log_void_delta()
RETURNS TRIGGER AS $$
DECLARE
    planned INT;
    completed INT;
    task_status VARCHAR(20);
    new_completed INT;
BEGIN
    SELECT planned_layers, completed_layers INTO planned, completed
    FROM production.tasks
    WHERE task_id = NEW.task_id;

    -- Apply delta only when voiding; unvoid is not supported
    IF NEW.voided = TRUE AND (OLD.voided IS DISTINCT FROM TRUE) THEN
        SELECT status INTO task_status FROM production.tasks WHERE task_id = NEW.task_id;
        new_completed := completed - OLD.layers_completed;
        IF new_completed < 0 THEN
            RAISE EXCEPTION '作废后完成层数不能小于0 (当前: %, 作废层数: %, 任务: %)', completed, OLD.layers_completed, NEW.task_id;
        END IF;
        PERFORM set_config('cutrix.task_adjustment_flag','true', true);
        PERFORM set_config('cutrix.plan_adjustment_flag','true', true);
        UPDATE production.tasks
        SET
            completed_layers = new_completed,
            status = CASE
                WHEN planned IS NOT NULL AND new_completed >= planned THEN 'completed'
                WHEN new_completed > 0 THEN 'in_progress'
                WHEN task_status = 'completed' THEN 'in_progress'
                ELSE task_status
            END
        WHERE task_id = NEW.task_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_after_log_update_void ON production.logs;
CREATE TRIGGER trg_after_log_update_void
AFTER UPDATE OF voided ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.apply_log_void_delta();

-- Guard: restrict logs updates to void-related fields only (no unvoid)
CREATE OR REPLACE FUNCTION production.guard_logs_update()
RETURNS TRIGGER AS $$
BEGIN
    -- Restrict immutable fields
    IF (NEW.task_id IS DISTINCT FROM OLD.task_id)
        OR (NEW.worker_id IS DISTINCT FROM OLD.worker_id)
        OR (NEW.worker_name IS DISTINCT FROM OLD.worker_name)
        OR (NEW.layers_completed IS DISTINCT FROM OLD.layers_completed)
        OR (NEW.log_time IS DISTINCT FROM OLD.log_time)
        OR (NEW.note IS DISTINCT FROM OLD.note) THEN
        RAISE EXCEPTION '日志仅允许作废相关字段的变更';
    END IF;

    -- Disallow unvoid: once voided, cannot revert
    IF NEW.voided = FALSE AND OLD.voided = TRUE THEN
        RAISE EXCEPTION '日志作废后不可恢复';
    END IF;

    -- Only allow void info updates when voided is TRUE
    IF (NEW.void_reason IS DISTINCT FROM OLD.void_reason OR NEW.voided_by IS DISTINCT FROM OLD.voided_by)
       AND NEW.voided IS DISTINCT FROM TRUE THEN
        RAISE EXCEPTION '仅在作废状态下允许更新作废信息';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_logs_update ON production.logs;
CREATE TRIGGER trg_guard_logs_update
BEFORE UPDATE ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.guard_logs_update();

-- Prevent deletion of logs to preserve audit trail
CREATE OR REPLACE FUNCTION production.prevent_logs_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF production.is_plan_delete_context() THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION '日志不可删除，请使用作废/反作废';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_logs_delete ON production.logs;
CREATE TRIGGER trg_prevent_logs_delete
BEFORE DELETE ON production.logs
FOR EACH ROW
EXECUTE FUNCTION production.prevent_logs_delete();

-- =====================
-- Seed Data (idempotent)
-- =====================
INSERT INTO public.users (name, password_hash, role) VALUES
('admin', '$2a$12$gwwSt9.uKHrxcCffsmgc0OvsdcRa1qldHE4bR/XrKNlYMK6IRyGty', 'admin'),
('manager', '$2a$12$kFFQ9IF1WV3Ky4VefgLFfOJl.bD1Ef/9bQC/7Ghc.IlFRRCjosya2', 'manager'),
('张三', '$2a$12$gwwSt9.uKHrxcCffsmgc0OvsdcRa1qldHE4bR/XrKNlYMK6IRyGty', 'worker'),
('王五', '$2a$12$gwwSt9.uKHrxcCffsmgc0OvsdcRa1qldHE4bR/XrKNlYMK6IRyGty', 'pattern_maker')
ON CONFLICT (name) DO NOTHING;

INSERT INTO production.orders (order_number, style_number, customer_name, order_start_date, order_finish_date, note) VALUES 
('ORD-2024-001', 'BEE3TS111', '客户A', '2024-01-15', '2024-02-15', NULL),
('ORD-2024-002', 'BEE3TS112', '客户B', '2024-01-16', '2024-02-20', NULL),
('ORD-2024-003', 'BEE3TS113', '客户C', '2024-01-17', '2024-02-25', NULL)
ON CONFLICT (order_number) DO NOTHING;

COMMIT;