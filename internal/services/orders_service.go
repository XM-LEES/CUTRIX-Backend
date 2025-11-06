package services

import "cutrix-backend/internal/models"

type OrdersService interface {
    Create(order *models.ProductionOrder) error
    GetByID(id int) (*models.ProductionOrder, error)
    Delete(id int) error
}