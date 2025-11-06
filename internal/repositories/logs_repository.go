package repositories

import "cutrix-backend/internal/models"

type LogsRepository interface {
    Create(log *models.ProductionLog) error
    ListParticipants(taskID int) ([]string, error)
    SetVoided(logID int, voided bool, reason *string, voidedBy *int) error
}