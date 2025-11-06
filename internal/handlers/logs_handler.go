package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/services"
)

type LogsHandler struct{ svc services.LogsService }

func NewLogsHandler(svc services.LogsService) *LogsHandler { return &LogsHandler{svc: svc} }

func (h *LogsHandler) Register(r *gin.RouterGroup) {
    r.POST("/logs", h.create)
    r.PATCH("/logs/:id", h.void)
    r.GET("/tasks/:id/participants", h.listParticipants)
}

func (h *LogsHandler) create(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var in models.ProductionLog
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.Create(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusCreated, in)
}

func (h *LogsHandler) void(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{
        Voided    bool    `json:"voided"`
        Reason    *string `json:"void_reason"`
        VoidedBy  *int    `json:"voided_by"`
    }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.SetVoided(id, body.Voided, body.Reason, body.VoidedBy); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
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