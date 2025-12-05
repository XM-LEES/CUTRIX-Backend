package services

import "cutrix-backend/internal/models"

type LogsService interface {
    Create(log *models.ProductionLog) error
    ListParticipants(taskID int) ([]string, error)

    // 不可反作废；允许在作废态下修正原因/作废人。
    Void(logID int, reason *string, voidedBy *int) error

    // 查看某个任务的所有日志（包含作废）。
    ListByTask(taskID int) ([]models.ProductionLog, error)
    // 查看某个布局的所有日志（包含作废）。
    ListByLayout(layoutID int) ([]models.ProductionLog, error)
    // 查看某个计划的所有日志（包含作废）。
    ListByPlan(planID int) ([]models.ProductionLog, error)

    // 查看某个工人的所有日志（包含作废）。
    // workerID 和 workerName 至少提供一个。
    ListByWorker(workerID *int, workerName *string) ([]models.ProductionLog, error)
}