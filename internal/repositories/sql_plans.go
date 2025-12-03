package repositories

import (
    "context"
    "database/sql"

    "cutrix-backend/internal/models"
)

type SqlPlansRepository struct{ db *sql.DB }

var _ PlansRepository = (*SqlPlansRepository)(nil)

func NewSqlPlansRepository(db *sql.DB) *SqlPlansRepository { return &SqlPlansRepository{db: db} }

func (r *SqlPlansRepository) Create(ctx context.Context, plan *models.ProductionPlan) (int, error) {
    const q = `
        INSERT INTO production.plans (plan_name, order_id, note)
        VALUES ($1, $2, $3)
        RETURNING plan_id`
    var id int
    var note any
    if plan.Note != nil { note = *plan.Note } else { note = nil }
    err := r.db.QueryRowContext(ctx, q, plan.PlanName, plan.OrderID, note).Scan(&id)
    if err != nil { return 0, err }
    plan.PlanID = id
    plan.Status = "pending"
    return id, nil
}

func (r *SqlPlansRepository) Delete(ctx context.Context, id int) error {
    res, err := r.db.ExecContext(ctx, `DELETE FROM production.plans WHERE plan_id = $1`, id)
    if err != nil { return err }
    if n, _ := res.RowsAffected(); n == 0 { return sql.ErrNoRows }
    return nil
}

func (r *SqlPlansRepository) UpdateNote(ctx context.Context, id int, note *string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE production.plans SET note = $1 WHERE plan_id = $2`, note, id)
    return err
}

func (r *SqlPlansRepository) Publish(ctx context.Context, id int) error {
    _, err := r.db.ExecContext(ctx, `UPDATE production.plans SET status = 'in_progress' WHERE plan_id = $1`, id)
    return err
}

func (r *SqlPlansRepository) Freeze(ctx context.Context, id int) error {
    _, err := r.db.ExecContext(ctx, `UPDATE production.plans SET status = 'frozen' WHERE plan_id = $1`, id)
    return err
}

func (r *SqlPlansRepository) GetByID(ctx context.Context, id int) (*models.ProductionPlan, error) {
    const q = `
        SELECT plan_id, plan_name, order_id, note, planned_publish_date, planned_finish_date, status
        FROM production.plans WHERE plan_id = $1`
    row := r.db.QueryRowContext(ctx, q, id)
    var p models.ProductionPlan
    var note sql.NullString
    var pub sql.NullTime
    var fin sql.NullTime
    if err := row.Scan(&p.PlanID, &p.PlanName, &p.OrderID, &note, &pub, &fin, &p.Status); err != nil {
        return nil, err
    }
    if note.Valid { v := note.String; p.Note = &v }
    if pub.Valid { t := pub.Time; p.PlannedPublishDate = &t }
    if fin.Valid { t := fin.Time; p.PlannedFinishDate = &t }
    return &p, nil
}

func (r *SqlPlansRepository) List(ctx context.Context) ([]models.ProductionPlan, error) {
    const q = `
        SELECT plan_id, plan_name, order_id, note, planned_publish_date, planned_finish_date, status
        FROM production.plans ORDER BY plan_id DESC`
    rows, err := r.db.QueryContext(ctx, q)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.ProductionPlan
    for rows.Next() {
        var p models.ProductionPlan
        var note sql.NullString
        var pub sql.NullTime
        var fin sql.NullTime
        if err := rows.Scan(&p.PlanID, &p.PlanName, &p.OrderID, &note, &pub, &fin, &p.Status); err != nil {
            return nil, err
        }
        if note.Valid { v := note.String; p.Note = &v }
        if pub.Valid { t := pub.Time; p.PlannedPublishDate = &t }
        if fin.Valid { t := fin.Time; p.PlannedFinishDate = &t }
        res = append(res, p)
    }
    return res, rows.Err()
}

func (r *SqlPlansRepository) ListByOrder(ctx context.Context, orderID int) ([]models.ProductionPlan, error) {
    const q = `
        SELECT plan_id, plan_name, order_id, note, planned_publish_date, planned_finish_date, status
        FROM production.plans WHERE order_id = $1 ORDER BY plan_id ASC`
    rows, err := r.db.QueryContext(ctx, q, orderID)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []models.ProductionPlan
    for rows.Next() {
        var p models.ProductionPlan
        var note sql.NullString
        var pub sql.NullTime
        var fin sql.NullTime
        if err := rows.Scan(&p.PlanID, &p.PlanName, &p.OrderID, &note, &pub, &fin, &p.Status); err != nil {
            return nil, err
        }
        if note.Valid { v := note.String; p.Note = &v }
        if pub.Valid { t := pub.Time; p.PlannedPublishDate = &t }
        if fin.Valid { t := fin.Time; p.PlannedFinishDate = &t }
        res = append(res, p)
    }
    return res, rows.Err()
}