package repositories

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    "cutrix-backend/internal/models"
)

type SqlLogsRepository struct { db *sql.DB }

var _ LogsRepository = (*SqlLogsRepository)(nil)

func scanLog(s scanner) (*models.ProductionLog, error) {
    var l models.ProductionLog
    var wID sql.NullInt64
    var wName, note, vReason, vByName sql.NullString
    var vAt sql.NullTime
    var vBy sql.NullInt64
    if err := s.Scan(
        &l.LogID,
        &l.TaskID,
        &wID,
        &wName,
        &l.LayersCompleted,
        &l.LogTime,
        &note,
        &l.Voided,
        &vReason,
        &vAt,
        &vBy,
        &vByName,
    ); err != nil { return nil, err }
    if wID.Valid { tmp := int(wID.Int64); l.WorkerID = &tmp }
    if wName.Valid { tmp := wName.String; l.WorkerName = &tmp }
    if note.Valid { tmp := note.String; l.Note = &tmp }
    if vReason.Valid { tmp := vReason.String; l.VoidReason = &tmp }
    if vAt.Valid { tmp := vAt.Time; l.VoidedAt = &tmp }
    if vBy.Valid { tmp := int(vBy.Int64); l.VoidedBy = &tmp }
    if vByName.Valid { tmp := vByName.String; l.VoidedByName = &tmp }
    return &l, nil
}

func NewSqlLogsRepository(db *sql.DB) *SqlLogsRepository { return &SqlLogsRepository{db: db} }

func (r *SqlLogsRepository) Create(log *models.ProductionLog) error {
    const q = `
        INSERT INTO production.logs (task_id, worker_id, worker_name, layers_completed, note)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING log_id, log_time
    `
    ctx := context.Background()
    return r.db.QueryRowContext(ctx, q,
        log.TaskID,
        log.WorkerID,
        log.WorkerName,
        log.LayersCompleted,
        log.Note,
    ).Scan(&log.LogID, &log.LogTime)
}

func (r *SqlLogsRepository) GetByID(logID int) (*models.ProductionLog, error) {
    const q = `
        SELECT 
            l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
            l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
        FROM production.logs l
        WHERE l.log_id = $1
    `
    ctx := context.Background()
    row := r.db.QueryRowContext(ctx, q, logID)
    log, err := scanLog(row)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return log, nil
}

func (r *SqlLogsRepository) ListParticipants(taskID int) ([]string, error) {
    const q = `
        SELECT DISTINCT COALESCE(pl.worker_name, u.name) AS worker
        FROM production.logs pl
        LEFT JOIN public.users u ON pl.worker_id = u.user_id
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

func (r *SqlLogsRepository) Void(logID int, reason *string, voidedBy *int) error {
    const q = `
        UPDATE production.logs
        SET voided = TRUE,
            void_reason = $2,
            voided_by = $3
        WHERE log_id = $1
    `
    ctx := context.Background()
    _, err := r.db.ExecContext(ctx, q, logID, reason, voidedBy)
    return err
}

func (r *SqlLogsRepository) ListByTask(taskID int) ([]models.ProductionLog, error) {
    const q = `
        SELECT 
            l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
            l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
        FROM production.logs l
        WHERE l.task_id = $1
        ORDER BY l.log_time ASC, l.log_id ASC
    `
    ctx := context.Background()
    rows, err := r.db.QueryContext(ctx, q, taskID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.ProductionLog
    for rows.Next() {
        pl, err := scanLog(rows)
        if err != nil { return nil, err }
        res = append(res, *pl)
    }
    return res, rows.Err()
}

func (r *SqlLogsRepository) ListByLayout(layoutID int) ([]models.ProductionLog, error) {
    const q = `
        SELECT 
            l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
            l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
        FROM production.logs l
        JOIN production.tasks t ON t.task_id = l.task_id
        WHERE t.layout_id = $1
        ORDER BY l.log_time ASC, l.log_id ASC
    `
    ctx := context.Background()
    rows, err := r.db.QueryContext(ctx, q, layoutID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.ProductionLog
    for rows.Next() {
        pl, err := scanLog(rows)
        if err != nil { return nil, err }
        res = append(res, *pl)
    }
    return res, rows.Err()
}

func (r *SqlLogsRepository) ListByPlan(planID int) ([]models.ProductionLog, error) {
    const q = `
        SELECT 
            l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
            l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
        FROM production.logs l
        JOIN production.tasks t ON t.task_id = l.task_id
        JOIN production.cutting_layouts lay ON lay.layout_id = t.layout_id
        WHERE lay.plan_id = $1
        ORDER BY l.log_time ASC, l.log_id ASC
    `
    ctx := context.Background()
    rows, err := r.db.QueryContext(ctx, q, planID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.ProductionLog
    for rows.Next() {
        pl, err := scanLog(rows)
        if err != nil { return nil, err }
        res = append(res, *pl)
    }
    return res, rows.Err()
}

func (r *SqlLogsRepository) ListByWorker(workerID *int, workerName *string) ([]models.ProductionLog, error) {
    var q string
    var args []interface{}
    
    if workerID != nil && workerName != nil {
        // 同时匹配 worker_id 和 worker_name
        q = `
            SELECT 
                l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
                l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
            FROM production.logs l
            WHERE (l.worker_id = $1 OR l.worker_name = $2)
            ORDER BY l.log_time DESC, l.log_id DESC
        `
        args = []interface{}{*workerID, *workerName}
    } else if workerID != nil {
        // 只匹配 worker_id
        q = `
            SELECT 
                l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
                l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
            FROM production.logs l
            WHERE l.worker_id = $1
            ORDER BY l.log_time DESC, l.log_id DESC
        `
        args = []interface{}{*workerID}
    } else if workerName != nil {
        // 只匹配 worker_name
        q = `
            SELECT 
                l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
                l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
            FROM production.logs l
            WHERE l.worker_name = $1
            ORDER BY l.log_time DESC, l.log_id DESC
        `
        args = []interface{}{*workerName}
    } else {
        // 两个参数都为空，返回空结果
        return []models.ProductionLog{}, nil
    }
    
    ctx := context.Background()
    rows, err := r.db.QueryContext(ctx, q, args...)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.ProductionLog
    for rows.Next() {
        pl, err := scanLog(rows)
        if err != nil { return nil, err }
        res = append(res, *pl)
    }
    return res, rows.Err()
}

func (r *SqlLogsRepository) CountVoidedByWorkerIn24Hours(workerID int) (int, error) {
    const q = `
        SELECT COUNT(*)
        FROM production.logs
        WHERE voided = TRUE
          AND voided_by = $1
          AND voided_at >= NOW() - INTERVAL '24 hours'
    `
    ctx := context.Background()
    var count int
    err := r.db.QueryRowContext(ctx, q, workerID).Scan(&count)
    return count, err
}

func (r *SqlLogsRepository) ListRecentVoided(limit int) ([]models.ProductionLog, error) {
    if limit <= 0 {
        limit = 50
    }
    const q = `
        SELECT 
            l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
            l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
        FROM production.logs l
        WHERE l.voided = TRUE
        ORDER BY l.voided_at DESC
        LIMIT $1
    `
    ctx := context.Background()
    rows, err := r.db.QueryContext(ctx, q, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var res []models.ProductionLog
    for rows.Next() {
        pl, err := scanLog(rows)
        if err != nil {
            return nil, err
        }
        res = append(res, *pl)
    }
    return res, rows.Err()
}

func (r *SqlLogsRepository) ListAll(taskID *int, workerID *int, voided *bool, limit int, offset int) ([]models.ProductionLog, int, error) {
    if limit <= 0 {
        limit = 50
    }
    if offset < 0 {
        offset = 0
    }
    
    // 构建WHERE条件
    var conditions []string
    var args []interface{}
    argIndex := 1
    
    if taskID != nil {
        conditions = append(conditions, fmt.Sprintf("l.task_id = $%d", argIndex))
        args = append(args, *taskID)
        argIndex++
    }
    if workerID != nil {
        conditions = append(conditions, fmt.Sprintf("l.worker_id = $%d", argIndex))
        args = append(args, *workerID)
        argIndex++
    }
    if voided != nil {
        conditions = append(conditions, fmt.Sprintf("l.voided = $%d", argIndex))
        args = append(args, *voided)
        argIndex++
    }
    
    whereClause := ""
    if len(conditions) > 0 {
        whereClause = "WHERE " + strings.Join(conditions, " AND ")
    }
    
    // 查询总数
    countQuery := fmt.Sprintf("SELECT COUNT(*) FROM production.logs l %s", whereClause)
    ctx := context.Background()
    var total int
    err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
    if err != nil {
        return nil, 0, err
    }
    
    // 查询数据
    dataQuery := fmt.Sprintf(`
        SELECT 
            l.log_id, l.task_id, l.worker_id, l.worker_name, l.layers_completed, l.log_time, l.note,
            l.voided, l.void_reason, l.voided_at, l.voided_by, l.voided_by_name
        FROM production.logs l
        %s
        ORDER BY l.log_time DESC, l.log_id DESC
        LIMIT $%d OFFSET $%d
    `, whereClause, argIndex, argIndex+1)
    
    args = append(args, limit, offset)
    rows, err := r.db.QueryContext(ctx, dataQuery, args...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()
    
    var res []models.ProductionLog
    for rows.Next() {
        pl, err := scanLog(rows)
        if err != nil {
            return nil, 0, err
        }
        res = append(res, *pl)
    }
    return res, total, rows.Err()
}