package repositories

import "cutrix-backend/internal/models"

type LogsRepository interface {
    // Create 记录新的生产日志；仅允许向 in_progress 任务提交，DB 触发器强制校验。
    Create(log *models.ProductionLog) error

    // GetByID 获取单个日志详情。
    GetByID(logID int) (*models.ProductionLog, error)

    // ListParticipants 列出参与该任务的人员快照（过滤作废日志）。
    ListParticipants(taskID int) ([]string, error)

    // Void 将日志标记为作废，并可同时设置/更新作废原因与作废人。
    // 注意：不可反作废；如需修正信息，重复调用本方法即可在作废态下更新 void_reason/voided_by。
    Void(logID int, reason *string, voidedBy *int) error

    // ListByTask 查看某个任务的所有日志（包含作废日志）。
    ListByTask(taskID int) ([]models.ProductionLog, error)

    // ListByLayout 查看某个布局的所有日志（包含作废日志）。
    ListByLayout(layoutID int) ([]models.ProductionLog, error)

    // ListByPlan 查看某个计划的所有日志（包含作废日志）。
    ListByPlan(planID int) ([]models.ProductionLog, error)

    // ListByWorker 查看某个工人的所有日志（包含作废日志）。
    // workerID 和 workerName 至少提供一个，如果都提供则同时匹配。
    ListByWorker(workerID *int, workerName *string) ([]models.ProductionLog, error)

    // CountVoidedByWorkerIn24Hours 统计worker在最近24小时内作废的日志数量。
    CountVoidedByWorkerIn24Hours(workerID int) (int, error)

    // ListRecentVoided 获取最近作废的日志（用于通知manager）。
    // limit 限制返回数量，默认50条。
    ListRecentVoided(limit int) ([]models.ProductionLog, error)

    // ListAll 获取所有日志（管理员用），支持筛选。
    // taskID, workerID, voided 为可选筛选条件，nil表示不过滤。
    ListAll(taskID *int, workerID *int, voided *bool, limit int, offset int) ([]models.ProductionLog, int, error)
}