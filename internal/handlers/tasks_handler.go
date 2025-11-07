package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/services"
)

type TasksHandler struct{ svc services.TasksService }

func NewTasksHandler(svc services.TasksService) *TasksHandler { return &TasksHandler{svc: svc} }

func (h *TasksHandler) Register(r *gin.RouterGroup) {
    r.POST("/tasks", h.create)
    r.GET("/tasks/:id", h.get)
    r.DELETE("/tasks/:id", h.delete)
    r.GET("/layouts/:id/tasks", h.listByLayout)
}

func (h *TasksHandler) create(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var in models.ProductionTask
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.Create(&in); err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusCreated, in)
}

func (h *TasksHandler) get(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.GetByID(id)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *TasksHandler) delete(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if err := h.svc.Delete(id); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

func (h *TasksHandler) listByLayout(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    layoutID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.ListByLayout(layoutID)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}