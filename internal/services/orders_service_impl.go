package services

import (
    "errors"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

type OrdersServiceImpl struct { repo repositories.OrdersRepository }

func NewOrdersService(repo repositories.OrdersRepository) *OrdersServiceImpl { return &OrdersServiceImpl{repo: repo} }

func (s *OrdersServiceImpl) Create(order *models.ProductionOrder) error {
    if order == nil { return errors.New("nil order") }
    if order.OrderNumber == "" { return errors.New("order_number required") }
    if order.StyleNumber == "" { return errors.New("style_number required") }
    return s.repo.Create(order)
}

func (s *OrdersServiceImpl) GetByID(id int) (*models.ProductionOrder, error) {
    if id <= 0 { return nil, errors.New("invalid order_id") }
    return s.repo.GetByID(id)
}

func (s *OrdersServiceImpl) Delete(id int) error {
    if id <= 0 { return errors.New("invalid order_id") }
    return s.repo.Delete(id)
}