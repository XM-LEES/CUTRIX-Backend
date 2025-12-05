package services

import "cutrix-backend/internal/models"

type LogsService interface {
    Create(log *models.ProductionLog) error
    GetByID(logID int) (*models.ProductionLog, error)
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

    // CountVoidedByWorkerIn24Hours 统计worker在最近24小时内作废的日志数量。
    CountVoidedByWorkerIn24Hours(workerID int) (int, error)

    // ListRecentVoided 获取最近作废的日志（用于通知manager）。
    ListRecentVoided(limit int) ([]models.ProductionLog, error)

    // ListAll 获取所有日志（管理员用），支持筛选。
    // 返回日志列表和总数。
    ListAll(taskID *int, workerID *int, voided *bool, limit int, offset int) ([]models.ProductionLog, int, error)
}