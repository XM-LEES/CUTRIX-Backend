package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/services"
    "cutrix-backend/internal/middleware"
)

type PlansHandler struct{ svc services.PlansService }

func NewPlansHandler(svc services.PlansService) *PlansHandler { return &PlansHandler{svc: svc} }

func (h *PlansHandler) Register(r *gin.RouterGroup) {
    r.GET("/plans", h.list)
    r.POST("/plans", h.create)
    r.DELETE("/plans/:id", h.delete)
    r.GET("/plans/:id", h.get)
    r.GET("/orders/:id/plans", h.listByOrder)
    r.PATCH("/plans/:id/note", h.updateNote)
    r.POST("/plans/:id/publish", h.publish)
    r.POST("/plans/:id/freeze", h.freeze)
}

// RegisterProtected registers routes with permissions applied. Use on authenticated groups.
func (h *PlansHandler) RegisterProtected(r *gin.RouterGroup) {
    r.GET("/plans", middleware.RequirePermissions("plan:read"), h.list)
    r.POST("/plans", middleware.RequirePermissions("plan:create"), h.create)
    r.DELETE("/plans/:id", middleware.RequirePermissions("plan:delete"), h.delete)
    r.GET("/plans/:id", middleware.RequirePermissions("plan:read"), h.get)
    r.GET("/orders/:id/plans", middleware.RequirePermissions("plan:read"), h.listByOrder)
    r.PATCH("/plans/:id/note", middleware.RequirePermissions("plan:update"), h.updateNote)
    r.POST("/plans/:id/publish", middleware.RequirePermissions("plan:publish"), h.publish)
    r.POST("/plans/:id/freeze", middleware.RequirePermissions("plan:freeze"), h.freeze)
}

func (h *PlansHandler) list(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    out, err := h.svc.List()
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *PlansHandler) create(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var in models.ProductionPlan
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.Create(&in); err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusCreated, in)
}

func (h *PlansHandler) delete(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if err := h.svc.Delete(id); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

func (h *PlansHandler) get(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.GetByID(id)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *PlansHandler) listByOrder(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    orderID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.ListByOrder(orderID)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *PlansHandler) updateNote(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ Note *string `json:"note"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.UpdateNote(id, body.Note); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

func (h *PlansHandler) publish(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if err := h.svc.Publish(id); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

func (h *PlansHandler) freeze(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if err := h.svc.Freeze(id); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}