package repositories

import "cutrix-backend/internal/models"

type OrdersRepository interface {
    Create(order *models.ProductionOrder) error
    GetByID(id int) (*models.ProductionOrder, error)
    Delete(id int) error
}