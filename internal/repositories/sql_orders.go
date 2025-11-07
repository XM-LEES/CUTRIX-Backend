package repositories

import (
    "context"
    "database/sql"
    "errors"
    "time"

    "cutrix-backend/internal/models"
)

// SqlOrdersRepository implements OrdersRepository against PostgreSQL.
type SqlOrdersRepository struct { db *sql.DB }

// NewSqlOrdersRepository creates a new SQL-based orders repository.
func NewSqlOrdersRepository(db *sql.DB) *SqlOrdersRepository { return &SqlOrdersRepository{db: db} }

// Compile-time check that SqlOrdersRepository satisfies OrdersRepository.
var _ OrdersRepository = (*SqlOrdersRepository)(nil)

// Create is disabled: orders must be created with items.
func (r *SqlOrdersRepository) Create(order *models.ProductionOrder) error {
    return errors.New("禁止创建无订单项的订单，请使用 CreateWithItems 并提供至少一个订单项")
}

// CreateWithItems starts a transaction to insert an order and its items atomically.
func (r *SqlOrdersRepository) CreateWithItems(order *models.ProductionOrder, items []models.OrderItem) error {
    if len(items) == 0 {
        return errors.New("订单必须至少包含一个订单项")
    }
    ctx := context.Background()
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil { return err }

    const insertOrder = `
        INSERT INTO production.orders (order_number, style_number, customer_name, order_start_date, order_finish_date, note)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING order_id, created_at, updated_at
    `
    if err := tx.QueryRowContext(ctx, insertOrder,
        order.OrderNumber,
        order.StyleNumber,
        order.CustomerName,
        order.OrderStartDate,
        order.OrderFinishDate,
        order.Note,
    ).Scan(&order.OrderID, &order.CreatedAt, &order.UpdatedAt); err != nil {
        tx.Rollback()
        return err
    }

    const insertItem = `
        INSERT INTO production.order_items (order_id, color, size, quantity)
        VALUES ($1, $2, $3, $4)
    `
    for i := range items {
        items[i].OrderID = order.OrderID
        if _, err := tx.ExecContext(ctx, insertItem,
            items[i].OrderID, items[i].Color, items[i].Size, items[i].Quantity,
        ); err != nil {
            tx.Rollback()
            return err
        }
    }

    return tx.Commit()
}

// UpdateNote updates order note; DB trigger sets updated_at.
func (r *SqlOrdersRepository) UpdateNote(id int, note *string) error {
    const q = `UPDATE production.orders SET note = $1 WHERE order_id = $2`
    ctx := context.Background()
    _, err := r.db.ExecContext(ctx, q, note, id)
    return err
}

// UpdateFinishDate updates order finish date (nullable); DB trigger sets updated_at.
func (r *SqlOrdersRepository) UpdateFinishDate(id int, finishDate *time.Time) error {
    const q = `UPDATE production.orders SET order_finish_date = $1 WHERE order_id = $2`
    ctx := context.Background()
    _, err := r.db.ExecContext(ctx, q, finishDate, id)
    return err
}

// GetByID loads an order by ID.
func (r *SqlOrdersRepository) GetByID(id int) (*models.ProductionOrder, error) {
    const q = `
        SELECT order_id, order_number, style_number, customer_name, order_start_date, order_finish_date, note, created_at, updated_at
        FROM production.orders WHERE order_id = $1
    `
    ctx := context.Background()
    var o models.ProductionOrder
    err := r.db.QueryRowContext(ctx, q, id).
        Scan(
            &o.OrderID, &o.OrderNumber, &o.StyleNumber, &o.CustomerName,
            &o.OrderStartDate, &o.OrderFinishDate, &o.Note,
            &o.CreatedAt, &o.UpdatedAt,
        )
    if err != nil { return nil, err }
    return &o, nil
}

// GetAll returns all orders ordered by created_at desc.
func (r *SqlOrdersRepository) GetAll(ctx context.Context) ([]models.ProductionOrder, error) {
    const q = `
        SELECT order_id, order_number, style_number, customer_name, order_start_date, order_finish_date, note, created_at, updated_at
        FROM production.orders
        ORDER BY created_at DESC
    `
    rows, err := r.db.QueryContext(ctx, q)
    if err != nil { return nil, err }
    defer rows.Close()

    var list []models.ProductionOrder
    for rows.Next() {
        var o models.ProductionOrder
        if err := rows.Scan(
            &o.OrderID, &o.OrderNumber, &o.StyleNumber, &o.CustomerName,
            &o.OrderStartDate, &o.OrderFinishDate, &o.Note,
            &o.CreatedAt, &o.UpdatedAt,
        ); err != nil { return nil, err }
        list = append(list, o)
    }
    return list, rows.Err()
}

// GetByOrderNumber returns an order by unique order_number.
func (r *SqlOrdersRepository) GetByOrderNumber(ctx context.Context, number string) (*models.ProductionOrder, error) {
    const q = `
        SELECT order_id, order_number, style_number, customer_name, order_start_date, order_finish_date, note, created_at, updated_at
        FROM production.orders WHERE order_number = $1
    `
    var o models.ProductionOrder
    err := r.db.QueryRowContext(ctx, q, number).
        Scan(
            &o.OrderID, &o.OrderNumber, &o.StyleNumber, &o.CustomerName,
            &o.OrderStartDate, &o.OrderFinishDate, &o.Note,
            &o.CreatedAt, &o.UpdatedAt,
        )
    if err != nil { return nil, err }
    return &o, nil
}

// GetWithItems loads an order and its items by order ID.
func (r *SqlOrdersRepository) GetWithItems(ctx context.Context, id int) (*models.ProductionOrder, []models.OrderItem, error) {
    order, err := r.GetByID(id)
    if err != nil { return nil, nil, err }

    const qi = `
        SELECT item_id, order_id, color, size, quantity
        FROM production.order_items WHERE order_id = $1 ORDER BY item_id
    `
    rows, err := r.db.QueryContext(ctx, qi, id)
    if err != nil { return nil, nil, err }
    defer rows.Close()

    var items []models.OrderItem
    for rows.Next() {
        var it models.OrderItem
        if err := rows.Scan(&it.ItemID, &it.OrderID, &it.Color, &it.Size, &it.Quantity); err != nil { return nil, nil, err }
        items = append(items, it)
    }
    return order, items, rows.Err()
}

// Delete removes an order by ID (items are removed via cascade).
func (r *SqlOrdersRepository) Delete(id int) error {
    const q = `DELETE FROM production.orders WHERE order_id = $1`
    ctx := context.Background()
    res, err := r.db.ExecContext(ctx, q, id)
    if err != nil { return err }
    if n, _ := res.RowsAffected(); n == 0 { return sql.ErrNoRows }
    return nil
}