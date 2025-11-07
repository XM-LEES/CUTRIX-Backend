package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/services"
)

type LayoutsHandler struct{ svc services.LayoutsService }

func NewLayoutsHandler(svc services.LayoutsService) *LayoutsHandler { return &LayoutsHandler{svc: svc} }

func (h *LayoutsHandler) Register(r *gin.RouterGroup) {
    r.POST("/layouts", h.create)
    r.DELETE("/layouts/:id", h.delete)
    r.GET("/layouts/:id", h.get)
    r.GET("/plans/:id/layouts", h.listByPlan)
    r.PATCH("/layouts/:id/name", h.updateName)
    r.PATCH("/layouts/:id/note", h.updateNote)
}

func (h *LayoutsHandler) create(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var in models.CuttingLayout
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.Create(&in); err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusCreated, in)
}

func (h *LayoutsHandler) delete(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if err := h.svc.Delete(id); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

func (h *LayoutsHandler) get(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.GetByID(id)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *LayoutsHandler) listByPlan(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    planID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.ListByPlan(planID)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

func (h *LayoutsHandler) updateName(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ Name string `json:"name"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.UpdateName(id, body.Name); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

func (h *LayoutsHandler) updateNote(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ Note *string `json:"note"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.UpdateNote(id, body.Note); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}