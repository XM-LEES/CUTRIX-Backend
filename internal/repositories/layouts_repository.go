package repositories

import "cutrix-backend/internal/models"

type LayoutsRepository interface {
    Create(layout *models.CuttingLayout) error
    Delete(id int) error
}