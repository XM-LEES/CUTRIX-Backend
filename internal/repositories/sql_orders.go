package repositories

import (
    "context"
    "database/sql"

    "cutrix-backend/internal/models"
)

type SqlOrdersRepository struct { db *sql.DB }

func NewSqlOrdersRepository(db *sql.DB) *SqlOrdersRepository { return &SqlOrdersRepository{db: db} }

func (r *SqlOrdersRepository) Create(order *models.ProductionOrder) error {
    const q = `
        INSERT INTO production.orders (order_number, style_number, customer_name, notes, status)
        VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'pending'))
        RETURNING order_id, status
    `
    ctx := context.Background()
    return r.db.QueryRowContext(ctx, q,
        order.OrderNumber,
        order.StyleNumber,
        order.CustomerName,
        order.Notes,
        order.Status,
    ).Scan(&order.OrderID, &order.Status)
}

func (r *SqlOrdersRepository) GetByID(id int) (*models.ProductionOrder, error) {
    const q = `
        SELECT order_id, order_number, style_number, customer_name, notes, status
        FROM production.orders WHERE order_id = $1
    `
    ctx := context.Background()
    var o models.ProductionOrder
    err := r.db.QueryRowContext(ctx, q, id).
        Scan(&o.OrderID, &o.OrderNumber, &o.StyleNumber, &o.CustomerName, &o.Notes, &o.Status)
    if err != nil { return nil, err }
    return &o, nil
}

func (r *SqlOrdersRepository) Delete(id int) error {
    const q = `DELETE FROM production.orders WHERE order_id = $1`
    ctx := context.Background()
    _, err := r.db.ExecContext(ctx, q, id)
    return err
}