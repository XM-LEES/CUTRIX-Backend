package handlers

import (
    "net/http"
    "strconv"

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