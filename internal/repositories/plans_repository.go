package repositories

import "cutrix-backend/internal/models"

type PlansRepository interface {
    Create(plan *models.ProductionPlan) error
    Delete(id int) error
}