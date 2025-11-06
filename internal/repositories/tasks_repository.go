package repositories

import "cutrix-backend/internal/models"

type TasksRepository interface {
    Create(task *models.ProductionTask) error
    GetByID(id int) (*models.ProductionTask, error)
}