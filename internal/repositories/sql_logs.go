package repositories

import (
    "context"
    "database/sql"

    "cutrix-backend/internal/models"
)

type SqlLogsRepository struct { db *sql.DB }

func NewSqlLogsRepository(db *sql.DB) *SqlLogsRepository { return &SqlLogsRepository{db: db} }

func (r *SqlLogsRepository) Create(log *models.ProductionLog) error {
    const q = `
        INSERT INTO production.logs (task_id, worker_id, worker_name, layers_completed, notes)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING log_id
    `
    ctx := context.Background()
    return r.db.QueryRowContext(ctx, q,
        log.TaskID,
        log.WorkerID,
        log.WorkerName,
        log.LayersCompleted,
        log.Notes,
    ).Scan(&log.LogID)
}

func (r *SqlLogsRepository) ListParticipants(taskID int) ([]string, error) {
    const q = `
        SELECT DISTINCT COALESCE(pl.worker_name, w.name) AS worker
        FROM production.logs pl
        LEFT JOIN public.Workers w ON pl.worker_id = w.worker_id
        WHERE pl.task_id = $1 AND NOT pl.voided
        ORDER BY worker ASC
    `
    ctx := context.Background()
    rows, err := r.db.QueryContext(ctx, q, taskID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []string
    for rows.Next() {
        var name string
        if err := rows.Scan(&name); err != nil { return nil, err }
        res = append(res, name)
    }
    return res, rows.Err()
}

func (r *SqlLogsRepository) SetVoided(logID int, voided bool, reason *string, voidedBy *int) error {
    const q = `
        UPDATE production.logs
        SET voided = $2,
            void_reason = $3,
            voided_by = $4
        WHERE log_id = $1
    `
    ctx := context.Background()
    _, err := r.db.ExecContext(ctx, q, logID, voided, reason, voidedBy)
    return err
}