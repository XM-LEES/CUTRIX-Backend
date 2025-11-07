package services

import (
    "context"
    "time"
    "cutrix-backend/internal/models"
)

// OrdersService handles production order lifecycle: create with items, query, update, delete.
type OrdersService interface {
    // CreateWithItems creates an order with its items atomically.
    CreateWithItems(ctx context.Context, order *models.ProductionOrder, items []models.OrderItem) (*models.ProductionOrder, error)

    // Updates allowed by policy: note and finish date.
    UpdateNote(ctx context.Context, id int, note *string) error
    UpdateFinishDate(ctx context.Context, id int, finishDate *time.Time) error

    // Queries
    GetByID(ctx context.Context, id int) (*models.ProductionOrder, error)
    GetAll(ctx context.Context) ([]models.ProductionOrder, error)
    GetByOrderNumber(ctx context.Context, number string) (*models.ProductionOrder, error)
    GetWithItems(ctx context.Context, id int) (*models.ProductionOrder, []models.OrderItem, error)

    // Delete removes an order by ID (cascades to items).
    Delete(ctx context.Context, id int) error
}