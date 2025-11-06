package repositories

import (
    "context"
    "database/sql"

    "cutrix-backend/internal/models"
)

type SqlPlansRepository struct { db *sql.DB }

func NewSqlPlansRepository(db *sql.DB) *SqlPlansRepository { return &SqlPlansRepository{db: db} }

func (r *SqlPlansRepository) Create(plan *models.ProductionPlan) error {
    const q = `
        INSERT INTO production.plans (plan_name, order_id)
        VALUES ($1, $2)
        RETURNING plan_id, status
    `
    ctx := context.Background()
    return r.db.QueryRowContext(ctx, q, plan.PlanName, plan.OrderID).
        Scan(&plan.PlanID, &plan.Status)
}

func (r *SqlPlansRepository) Delete(id int) error {
    const q = `DELETE FROM production.plans WHERE plan_id = $1`
    ctx := context.Background()
    _, err := r.db.ExecContext(ctx, q, id)
    return err
}