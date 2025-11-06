package services

import "cutrix-backend/internal/models"

type PlansService interface {
    Create(plan *models.ProductionPlan) error
    Delete(id int) error
}