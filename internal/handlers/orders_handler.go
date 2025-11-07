package handlers

import (
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/models"
    "cutrix-backend/internal/services"
)

// OrdersHandler exposes production order endpoints: create with items, query, update, delete.
type OrdersHandler struct{ svc services.OrdersService }

func NewOrdersHandler(svc services.OrdersService) *OrdersHandler { return &OrdersHandler{svc: svc} }

func (h *OrdersHandler) Register(r *gin.RouterGroup) {
    // Create with items
    r.POST("/orders", h.create)
    // Basic queries
    r.GET("/orders", h.list)
    r.GET("/orders/:id", h.get)
    r.GET("/orders/by-number/:number", h.getByNumber)
    r.GET("/orders/:id/full", h.getFull)
    // Updates
    r.PATCH("/orders/:id/note", h.updateNote)
    r.PATCH("/orders/:id/finish-date", h.updateFinishDate)
    // Delete
    r.DELETE("/orders/:id", h.delete)
}

// create creates an order with items atomically.
func (h *OrdersHandler) create(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var body struct {
        models.ProductionOrder `json:",inline"`
        Items []models.OrderItem `json:"items"`
    }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    out, err := h.svc.CreateWithItems(c.Request.Context(), &body.ProductionOrder, body.Items)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusCreated, out)
}

// list returns all orders ordered by created_at desc.
func (h *OrdersHandler) list(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    out, err := h.svc.GetAll(c.Request.Context())
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

// get returns a single order by ID.
func (h *OrdersHandler) get(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.svc.GetByID(c.Request.Context(), id)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

// getByNumber returns order by order_number.
func (h *OrdersHandler) getByNumber(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    number := c.Param("number")
    out, err := h.svc.GetByOrderNumber(c.Request.Context(), number)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

// getFull returns an order and its items by ID.
func (h *OrdersHandler) getFull(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    order, items, err := h.svc.GetWithItems(c.Request.Context(), id)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, gin.H{"order": order, "items": items})
}

// updateNote updates the order note.
func (h *OrdersHandler) updateNote(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ Note *string `json:"note"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.UpdateNote(c.Request.Context(), id, body.Note); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

// updateFinishDate updates the order finish date.
func (h *OrdersHandler) updateFinishDate(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ FinishDate *time.Time `json:"order_finish_date"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if err := h.svc.UpdateFinishDate(c.Request.Context(), id, body.FinishDate); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

// delete removes an order by ID.
func (h *OrdersHandler) delete(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if err := h.svc.Delete(c.Request.Context(), id); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}