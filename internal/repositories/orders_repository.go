package repositories

import (
    "context"
    "time"
    "cutrix-backend/internal/models"
)

// OrdersRepository defines the order data access contract.
// Notes:
// - Creating an order requires at least one item; empty orders are not allowed.
// - Orders allow updates to `note` and `order_finish_date` only.
// - `order_start_date` is set at creation and remains immutable.
// - `created_at` and `updated_at` are maintained by DB defaults/triggers.
// - Order items are created at order creation and immutable afterward.
// - Deleting an order cascades to its items via foreign key.
type OrdersRepository interface {
    // Basic operations
    // Create is disabled: use CreateWithItems with at least one item.
    Create(order *models.ProductionOrder) error
    // CreateWithItems atomically creates one order and its items (transaction).
    CreateWithItems(order *models.ProductionOrder, items []models.OrderItem) error

    // Business updates
    // UpdateNote updates note; DB triggers update `updated_at`.
    UpdateNote(id int, note *string) error
    // UpdateFinishDate updates finish date (nullable); DB triggers update `updated_at`.
    UpdateFinishDate(id int, finishDate *time.Time) error

    // Queries
    // GetByID returns an order by ID.
    GetByID(id int) (*models.ProductionOrder, error)
    // GetAll returns all orders ordered by created_at desc.
    GetAll(ctx context.Context) ([]models.ProductionOrder, error)
    // GetByOrderNumber returns an order by unique order_number.
    GetByOrderNumber(ctx context.Context, number string) (*models.ProductionOrder, error)
    // GetWithItems returns the order and its items.
    GetWithItems(ctx context.Context, id int) (*models.ProductionOrder, []models.OrderItem, error)

    // Delete removes an order by ID (cascades to items).
    Delete(id int) error
}