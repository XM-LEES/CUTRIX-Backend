package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "example-demo/internal/services"
)

type UserHandler struct{
    svc *services.UserService
}

func NewUserHandler(svc *services.UserService) *UserHandler { return &UserHandler{svc: svc} }

func (h *UserHandler) Register(r *gin.RouterGroup) {
    g := r.Group("/users")
    g.POST("/", h.create)
    g.GET("/", h.list)
    g.GET(":id", h.get)
    g.PUT(":id", h.update)
    g.DELETE(":id", h.delete)
}

type createUserReq struct {
    Name  string `json:"name" binding:"required,min=1"`
    Email string `json:"email" binding:"required,email"`
}

type updateUserReq struct {
    Name  *string `json:"name"`
    Email *string `json:"email"`
}

func (h *UserHandler) create(c *gin.Context) {
    var req createUserReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "detail": err.Error()})
        return
    }
    user, err := h.svc.Create(req.Name, req.Email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed", "detail": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) list(c *gin.Context) {
    items, err := h.svc.List()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "list_failed", "detail": err.Error()})
        return
    }
    c.JSON(http.StatusOK, items)
}

func (h *UserHandler) get(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
        return
    }
    user, err := h.svc.Get(id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "get_failed", "detail": err.Error()})
        return
    }
    if user == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
        return
    }
    c.JSON(http.StatusOK, user)
}

func (h *UserHandler) update(c *gin.Context) {
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
        return
    }
    var req updateUserReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "detail": err.Error()})
        return
    }
    user, err := h.svc.Update(id, req.Name, req.Email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "update_failed", "detail": err.Error()})
        return
    }
    if user == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
        return
    }
    c.JSON(http.StatusOK, user)
}

func (h *UserHandler) delete(c *gin.Context) {
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