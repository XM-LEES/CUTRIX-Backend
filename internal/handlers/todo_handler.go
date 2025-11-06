package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "example-demo/internal/services"
)

type TodoHandler struct{
    svc *services.TodoService
}

func NewTodoHandler(svc *services.TodoService) *TodoHandler { return &TodoHandler{svc: svc} }

func (h *TodoHandler) Register(r *gin.RouterGroup) {
    g := r.Group("/todos")
    g.POST("/", h.create)
    g.GET("/", h.list)
    g.GET(":id", h.get)
    g.PUT(":id", h.update)
    g.DELETE(":id", h.delete)
}

type createReq struct {
    Title string `json:"title" binding:"required,min=1"`
    UserID *int64 `json:"user_id"`
}

type updateReq struct {
    Title     *string `json:"title"`
    Completed *bool   `json:"completed"`
    UserID    *int64  `json:"user_id"`
}

func (h *TodoHandler) create(c *gin.Context) {
    var req createReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "detail": err.Error()})
        return
    }
    todo, err := h.svc.Create(req.Title, req.UserID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed", "detail": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, todo)
}

func (h *TodoHandler) list(c *gin.Context) {
    var userID *int64
    if q := c.Query("user_id"); q != "" {
        if v, err := strconv.ParseInt(q, 10, 64); err == nil {
            userID = &v
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_user_id"})
            return
        }
    }
    items, err := h.svc.List(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "list_failed", "detail": err.Error()})
        return
    }
    c.JSON(http.StatusOK, items)
}

func (h *TodoHandler) get(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
        return
    }
    todo, err := h.svc.Get(id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "get_failed", "detail": err.Error()})
        return
    }
    if todo == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
        return
    }
    c.JSON(http.StatusOK, todo)
}

func (h *TodoHandler) update(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
        return
    }
    var req updateReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "detail": err.Error()})
        return
    }
    todo, err := h.svc.Update(id, req.Title, req.Completed, req.UserID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "update_failed", "detail": err.Error()})
        return
    }
    if todo == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
        return
    }
    c.JSON(http.StatusOK, todo)
}

func (h *TodoHandler) delete(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
        return
    }
    ok, err := h.svc.Delete(id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "delete_failed", "detail": err.Error()})
        return
    }
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
        return
    }
    c.JSON(http.StatusNoContent, nil)
}