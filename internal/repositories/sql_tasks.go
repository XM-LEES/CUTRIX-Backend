package repositories

import (
    "context"
    "database/sql"

    "cutrix-backend/internal/models"
)

type SqlTasksRepository struct { db *sql.DB }

func NewSqlTasksRepository(db *sql.DB) *SqlTasksRepository { return &SqlTasksRepository{db: db} }

func (r *SqlTasksRepository) Create(task *models.ProductionTask) error {
    const q = `
        INSERT INTO production.tasks (layout_id, color, planned_layers)
        VALUES ($1, $2, $3)
        RETURNING task_id, completed_layers, status
    `
    ctx := context.Background()
    return r.db.QueryRowContext(ctx, q, task.LayoutID, task.Color, task.PlannedLayers).
        Scan(&task.TaskID, &task.CompletedLayers, &task.Status)
}

func (r *SqlTasksRepository) GetByID(id int) (*models.ProductionTask, error) {
    const q = `
        SELECT task_id, layout_id, color, planned_layers, completed_layers, status
        FROM production.tasks WHERE task_id = $1
    `
    ctx := context.Background()
    var t models.ProductionTask
    err := r.db.QueryRowContext(ctx, q, id).
        Scan(&t.TaskID, &t.LayoutID, &t.Color, &t.PlannedLayers, &t.CompletedLayers, &t.Status)
    if err != nil { return nil, err }
    return &t, nil
}