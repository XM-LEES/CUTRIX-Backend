package handlers

import (
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/services"
    "cutrix-backend/internal/middleware"
)

type LogsHandler struct{ svc services.LogsService }

func NewLogsHandler(svc services.LogsService) *LogsHandler { return &LogsHandler{svc: svc} }

func (h *LogsHandler) Register(r *gin.RouterGroup) {
    r.POST("/logs", h.create)
    r.PATCH("/logs/:id", h.void)
    r.GET("/logs", h.listAll) // 必须在 /logs/my 之前注册
    r.GET("/logs/my", h.listMyLogs)
    r.GET("/logs/recent-voided", h.listRecentVoided)
    r.GET("/tasks/:id/participants", h.listParticipants)
    r.GET("/tasks/:id/logs", h.listTaskLogs)
    r.GET("/layouts/:id/logs", h.listLayoutLogs)
    r.GET("/plans/:id/logs", h.listPlanLogs)
}

// RegisterProtected registers routes with fine-grained permissions. Use on authenticated groups.
func (h *LogsHandler) RegisterProtected(r *gin.RouterGroup) {
    // Workers can create/update logs; admins/managers bypass permission via super roles.
    r.POST("/logs", middleware.RequirePermissions("log:create"), h.create)
    r.PATCH("/logs/:id", middleware.RequirePermissions("log:update"), h.void)

    // Get my logs - any authenticated user can view their own logs
    r.GET("/logs/my", h.listMyLogs)

    // Get recent voided logs - for manager notification
    r.GET("/logs/recent-voided", middleware.RequireRoles("admin", "manager"), h.listRecentVoided)

    // Get all logs - for admin/manager to view and manage all logs
    r.GET("/logs", middleware.RequireRoles("admin", "manager"), h.listAll)

    // Listing endpoints restricted to admin/manager via role check.
    r.GET("/tasks/:id/participants", middleware.RequireRoles("admin", "manager"), h.listParticipants)
    r.GET("/tasks/:id/logs", middleware.RequireRoles("admin", "manager"), h.listTaskLogs)
    r.GET("/layouts/:id/logs", middleware.RequireRoles("admin", "manager"), h.listLayoutLogs)
    r.GET("/plans/:id/logs", middleware.RequireRoles("admin", "manager"), h.listPlanLogs)
}

func (h *LogsHandler) create(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var in models.ProductionLog
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.Create(&in); err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusCreated, in)
}

func (h *LogsHandler) void(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{
        Reason    *string `json:"void_reason"`
        VoidedBy  *int    `json:"voided_by"`
    }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    
    // 如果是 worker 角色，验证只能作废自己的日志
    v, ok := c.Get("role")
    if ok && v != nil {
        role, _ := v.(string)
        role = strings.ToLower(strings.TrimSpace(role))
        if role == "worker" {
            // 获取当前用户信息
            claims, ok := c.Get("claims")
            if !ok || claims == nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
                return
            }
            userClaims, ok := claims.(*services.Claims)
            if !ok || userClaims == nil {
                c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
                return
            }
            
            // 获取日志详情
            log, err := h.svc.GetByID(id)
            if err != nil {
                writeSvcError(c, err)
                return
            }
            if log == nil {
                c.JSON(http.StatusNotFound, gin.H{"error":"not_found"})
                return
            }
            
            // 验证是否是自己的日志
            isOwnLog := false
            if log.WorkerID != nil && *log.WorkerID == userClaims.UserID {
                isOwnLog = true
            }
            if !isOwnLog && log.WorkerName != nil && *log.WorkerName == userClaims.Name {
                isOwnLog = true
            }
            
            if !isOwnLog {
                c.JSON(http.StatusForbidden, gin.H{"error":"forbidden", "message":"只能作废自己的日志"})
                return
            }
            
            // 时间限制：只能作废24小时内的日志
            logAge := time.Since(log.LogTime)
            if logAge > 24*time.Hour {
                c.JSON(http.StatusForbidden, gin.H{"error":"forbidden", "message":"只能作废24小时内的日志"})
                return
            }
            
            // 数量限制：每天最多作废3条
            voidedCount, err := h.svc.CountVoidedByWorkerIn24Hours(userClaims.UserID)
            if err != nil {
                writeSvcError(c, err)
                return
            }
            if voidedCount >= 3 {
                c.JSON(http.StatusForbidden, gin.H{"error":"forbidden", "message":"24小时内最多只能作废3条日志"})
                return
            }
            
            // 设置 voided_by 为当前用户
            if body.VoidedBy == nil {
                body.VoidedBy = &userClaims.UserID
            }
        }
    }
    
    if err := h.svc.Void(id, body.Reason, body.VoidedBy); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

func (h *LogsHandler) listParticipants(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.ListParticipants(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, out)
}

func (h *LogsHandler) listTaskLogs(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.ListByTask(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, out)
}

func (h *LogsHandler) listLayoutLogs(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.ListByLayout(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, out)
}

func (h *LogsHandler) listPlanLogs(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.ListByPlan(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusOK, out)
}

func (h *LogsHandler) listMyLogs(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    
    // Get current user from context (set by RequireAuth middleware)
    v, ok := c.Get("claims")
    if !ok || v == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
        return
    }
    claims, ok := v.(*services.Claims)
    if !ok || claims == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
        return
    }
    
    // Get logs for current user (by worker_id and/or worker_name)
    workerID := &claims.UserID
    workerName := &claims.Name
    out, err := h.svc.ListByWorker(workerID, workerName)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *LogsHandler) listRecentVoided(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    
    limit := 50
    if limitStr := c.Query("limit"); limitStr != "" {
        if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
            limit = parsed
        }
    }
    
    out, err := h.svc.ListRecentVoided(limit)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *LogsHandler) listAll(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    
    var taskID *int
    var workerID *int
    var voided *bool
    
    if taskIDStr := c.Query("task_id"); taskIDStr != "" {
        if parsed, err := strconv.Atoi(taskIDStr); err == nil && parsed > 0 {
            taskID = &parsed
        }
    }
    
    if workerIDStr := c.Query("worker_id"); workerIDStr != "" {
        if parsed, err := strconv.Atoi(workerIDStr); err == nil && parsed > 0 {
            workerID = &parsed
        }
    }
    
    if voidedStr := c.Query("voided"); voidedStr != "" {
        if voidedStr == "true" || voidedStr == "1" {
            parsed := true
            voided = &parsed
        } else if voidedStr == "false" || voidedStr == "0" {
            parsed := false
            voided = &parsed
        }
    }
    
    limit := 50
    if limitStr := c.Query("limit"); limitStr != "" {
        if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 200 {
            limit = parsed
        }
    }
    
    offset := 0
    if offsetStr := c.Query("offset"); offsetStr != "" {
        if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
            offset = parsed
        }
    }
    
    logs, total, err := h.svc.ListAll(taskID, workerID, voided, limit, offset)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, gin.H{
        "logs": logs,
        "total": total,
        "limit": limit,
        "offset": offset,
    })
}