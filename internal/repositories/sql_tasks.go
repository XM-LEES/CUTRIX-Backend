package repositories

import (
    "context"
    "database/sql"
    "fmt"

    "cutrix-backend/internal/models"
)

type SqlTasksRepository struct{ db *sql.DB }

var _ TasksRepository = (*SqlTasksRepository)(nil)

func NewSqlTasksRepository(db *sql.DB) *SqlTasksRepository { return &SqlTasksRepository{db: db} }

// planStatusByLayout returns the parent plan status for a given layout.
func (r *SqlTasksRepository) planStatusByLayout(ctx context.Context, layoutID int) (string, error) {
    const q = `
        SELECT p.status
        FROM production.cutting_layouts l
        JOIN production.plans p ON p.plan_id = l.plan_id
        WHERE l.layout_id = $1`
    var status string
    if err := r.db.QueryRowContext(ctx, q, layoutID).Scan(&status); err != nil { return "", err }
    return status, nil
}

// layoutIDByTask looks up the layout_id for a given task.
func (r *SqlTasksRepository) layoutIDByTask(ctx context.Context, taskID int) (int, error) {
    const q = `SELECT layout_id FROM production.tasks WHERE task_id = $1`
    var layoutID int
    if err := r.db.QueryRowContext(ctx, q, taskID).Scan(&layoutID); err != nil { return 0, err }
    return layoutID, nil
}

func (r *SqlTasksRepository) Create(ctx context.Context, task *models.ProductionTask) (int, error) {
    // Pre-check: only allow creating tasks when plan is pending
    status, err := r.planStatusByLayout(ctx, task.LayoutID)
    if err != nil { return 0, err }
    if status != "pending" {
        return 0, fmt.Errorf("计划发布后不允许新增任务 (layout_id=%d, status=%s)", task.LayoutID, status)
    }

    const q = `
        INSERT INTO production.tasks (layout_id, color, planned_layers)
        VALUES ($1, $2, $3)
        RETURNING task_id, completed_layers, status`
    var id int
    err = r.db.QueryRowContext(ctx, q, task.LayoutID, task.Color, task.PlannedLayers).
        Scan(&id, &task.CompletedLayers, &task.Status)
    if err == nil { task.TaskID = id }
    return id, err
}

func (r *SqlTasksRepository) Delete(ctx context.Context, id int) error {
    // Pre-check: only allow deleting tasks when plan is pending
    layoutID, err := r.layoutIDByTask(ctx, id)
    if err != nil { return err }
    status, err := r.planStatusByLayout(ctx, layoutID)
    if err != nil { return err }
    if status != "pending" {
        return fmt.Errorf("计划发布后不允许删除任务 (task_id=%d, status=%s)", id, status)
    }

    res, err := r.db.ExecContext(ctx, `DELETE FROM production.tasks WHERE task_id = $1`, id)
    if err != nil { return err }
    if n, _ := res.RowsAffected(); n == 0 { return sql.ErrNoRows }
    return nil
}

func (r *SqlTasksRepository) UpdateStatus(ctx context.Context, id int, status string) error {
    // Pre-check: direct status updates are only allowed before publish (pending)
    layoutID, err := r.layoutIDByTask(ctx, id)
    if err != nil { return err }
    planStatus, err := r.planStatusByLayout(ctx, layoutID)
    if err != nil { return err }
    if planStatus != "pending" {
        return fmt.Errorf("发布后禁止直接修改任务状态，请通过日志累计完成层数 (task_id=%d, plan_status=%s)", id, planStatus)
    }
    _, err = r.db.ExecContext(ctx, `UPDATE production.tasks SET status = $1 WHERE task_id = $2`, status, id)
    return err
}

func (r *SqlTasksRepository) GetByID(ctx context.Context, id int) (*models.ProductionTask, error) {
    const q = `
        SELECT task_id, layout_id, color, planned_layers, completed_layers, status
        FROM production.tasks WHERE task_id = $1`
    row := r.db.QueryRowContext(ctx, q, id)
    var t models.ProductionTask
    if err := row.Scan(&t.TaskID, &t.LayoutID, &t.Color, &t.PlannedLayers, &t.CompletedLayers, &t.Status); err != nil { return nil, err }
    return &t, nil
}

func (r *SqlTasksRepository) ListByLayout(ctx context.Context, layoutID int) ([]models.ProductionTask, error) {
    const q = `
        SELECT task_id, layout_id, color, planned_layers, completed_layers, status
        FROM production.tasks WHERE layout_id = $1 ORDER BY task_id ASC`
    rows, err := r.db.QueryContext(ctx, q, layoutID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.ProductionTask
    for rows.Next() {
        var t models.ProductionTask
        if err := rows.Scan(&t.TaskID, &t.LayoutID, &t.Color, &t.PlannedLayers, &t.CompletedLayers, &t.Status); err != nil { return nil, err }
        res = append(res, t)
    }
    return res, rows.Err()
}