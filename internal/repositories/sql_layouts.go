package repositories

import (
    "context"
    "database/sql"

    "cutrix-backend/internal/models"
)

type SqlLayoutsRepository struct { db *sql.DB }

func NewSqlLayoutsRepository(db *sql.DB) *SqlLayoutsRepository { return &SqlLayoutsRepository{db: db} }

func (r *SqlLayoutsRepository) Create(layout *models.CuttingLayout) error {
    const q = `
        INSERT INTO production.cutting_layouts (plan_id, layout_name, description)
        VALUES ($1, $2, $3)
        RETURNING layout_id
    `
    ctx := context.Background()
    return r.db.QueryRowContext(ctx, q, layout.PlanID, layout.LayoutName, layout.Description).
        Scan(&layout.LayoutID)
}

func (r *SqlLayoutsRepository) Delete(id int) error {
    const q = `DELETE FROM production.cutting_layouts WHERE layout_id = $1`
    ctx := context.Background()
    _, err := r.db.ExecContext(ctx, q, id)
    return err
}