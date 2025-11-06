package services

import "cutrix-backend/internal/models"

type TasksService interface {
    Create(task *models.ProductionTask) error
    GetByID(id int) (*models.ProductionTask, error)
}