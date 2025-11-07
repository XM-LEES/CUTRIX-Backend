package services

import (
    "context"
    "errors"
    "strings"
    "time"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

// ordersService implements OrdersService using OrdersRepository.
type ordersService struct { repo repositories.OrdersRepository }

// NewOrdersService constructs an OrdersService.
func NewOrdersService(repo repositories.OrdersRepository) OrdersService { return &ordersService{repo: repo} }

// CreateWithItems creates an order with items atomically.
func (s *ordersService) CreateWithItems(ctx context.Context, order *models.ProductionOrder, items []models.OrderItem) (*models.ProductionOrder, error) {
    if order == nil { return nil, errors.New("nil order") }
    if strings.TrimSpace(order.OrderNumber) == "" { return nil, errors.New("order_number required") }
    if strings.TrimSpace(order.StyleNumber) == "" { return nil, errors.New("style_number required") }
    if len(items) == 0 { return nil, errors.New("order must include at least one item") }
    if err := s.repo.CreateWithItems(order, items); err != nil { return nil, err }
    return order, nil
}

// UpdateNote updates the order note.
func (s *ordersService) UpdateNote(ctx context.Context, id int, note *string) error {
    if id <= 0 { return errors.New("invalid order_id") }
    return s.repo.UpdateNote(id, note)
}

// UpdateFinishDate updates the order finish date.
func (s *ordersService) UpdateFinishDate(ctx context.Context, id int, finishDate *time.Time) error {
    if id <= 0 { return errors.New("invalid order_id") }
    return s.repo.UpdateFinishDate(id, finishDate)
}

// GetByID returns an order by ID.
func (s *ordersService) GetByID(ctx context.Context, id int) (*models.ProductionOrder, error) {
    if id <= 0 { return nil, errors.New("invalid order_id") }
    return s.repo.GetByID(id)
}

// GetAll returns all orders.
func (s *ordersService) GetAll(ctx context.Context) ([]models.ProductionOrder, error) {
    return s.repo.GetAll(ctx)
}

// GetByOrderNumber returns order by unique order_number.
func (s *ordersService) GetByOrderNumber(ctx context.Context, number string) (*models.ProductionOrder, error) {
    if strings.TrimSpace(number) == "" { return nil, errors.New("order_number required") }
    return s.repo.GetByOrderNumber(ctx, number)
}

// GetWithItems returns order and its items by ID.
func (s *ordersService) GetWithItems(ctx context.Context, id int) (*models.ProductionOrder, []models.OrderItem, error) {
    if id <= 0 { return nil, nil, errors.New("invalid order_id") }
    return s.repo.GetWithItems(ctx, id)
}

// Delete removes an order by ID.
func (s *ordersService) Delete(ctx context.Context, id int) error {
    if id <= 0 { return errors.New("invalid order_id") }
    return s.repo.Delete(id)
}