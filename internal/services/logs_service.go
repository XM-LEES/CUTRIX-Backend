package services

import "cutrix-backend/internal/models"

type LogsService interface {
    Create(log *models.ProductionLog) error
    ListParticipants(taskID int) ([]string, error)
    SetVoided(logID int, voided bool, reason *string, voidedBy *int) error
}