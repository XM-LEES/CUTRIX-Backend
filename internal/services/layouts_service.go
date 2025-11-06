package services

import "cutrix-backend/internal/models"

type LayoutsService interface {
    Create(layout *models.CuttingLayout) error
    Delete(id int) error
}