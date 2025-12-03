package handlers

import (
    "database/sql"
    "errors"
    "net/http"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/services"
)

// UsersHandler exposes user management endpoints; excludes password lifecycle except admin reset.
type UsersHandler struct{
    users services.UsersService
    auth  services.AuthService
}

func NewUsersHandler(users services.UsersService, auth services.AuthService) *UsersHandler {
    return &UsersHandler{users: users, auth: auth}
}

func (h *UsersHandler) Register(r *gin.RouterGroup) {
    r.GET("/users", h.list)
    r.GET("/users/:id", h.get)
    r.POST("/users", h.create)
    r.PATCH("/users/:id/profile", h.updateProfile)
    r.PUT("/users/:id/role", h.assignRole)
    r.PUT("/users/:id/active", h.setActive)
    r.PUT("/users/:id/password", h.setPassword)
    r.DELETE("/users/:id", h.delete)
}

// list returns users filtered by query params.
func (h *UsersHandler) list(c *gin.Context) {
    if h.users == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var (
        qPtr, namePtr, rolePtr, groupPtr *string
        activePtr *bool
    )
    if v := strings.TrimSpace(c.Query("query")); v != "" { qPtr = &v }
    if v := strings.TrimSpace(c.Query("name")); v != "" { namePtr = &v }
    if v := strings.TrimSpace(c.Query("role")); v != "" { rolePtr = &v }
    if v := strings.TrimSpace(c.Query("group")); v != "" { groupPtr = &v }
    if v := strings.TrimSpace(c.Query("active")); v != "" {
        b, err := strconv.ParseBool(v)
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"validation_error", "message":"active must be boolean"}); return }
        activePtr = &b
    }
    filter := services.UsersFilter{
        Query: qPtr, Name: namePtr, Role: rolePtr, Active: activePtr, UserGroup: groupPtr,
    }
    out, err := h.users.List(c.Request.Context(), filter)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

// get returns a single user by ID.
func (h *UsersHandler) get(c *gin.Context) {
    if h.users == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    out, err := h.users.GetByID(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, services.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
            c.JSON(http.StatusNotFound, gin.H{"error":"not_found"}); return
        }
        writeSvcError(c, err); return
    }
    if out == nil { c.JSON(http.StatusNotFound, gin.H{"error":"not_found"}); return }
    c.JSON(http.StatusOK, out)
}

// create creates a new user; admin only.
func (h *UsersHandler) create(c *gin.Context) {
    if h.users == nil || h.auth == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    claims, ok := h.requireAuth(c)
    if !ok { return }
    if claims.Role != "admin" && claims.Role != "manager" { c.JSON(http.StatusForbidden, gin.H{"error":"forbidden"}); return }

    var body struct{
        Name string  `json:"name"`
        Role string  `json:"role"`
        Group *string `json:"group"`
        Note  *string `json:"note"`
    }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Role) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error":"validation_error", "message":"name and role required"}); return
    }
    out, err := h.users.Create(c.Request.Context(), claims.UserID, claims.Role, body.Name, body.Role, body.Group, body.Note)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusCreated, out)
}

// updateProfile updates name/group/note; owner or admin.
func (h *UsersHandler) updateProfile(c *gin.Context) {
    if h.users == nil || h.auth == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    claims, ok := h.requireAuth(c)
    if !ok { return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if claims.Role != "admin" && claims.Role != "manager" && claims.UserID != id { c.JSON(http.StatusForbidden, gin.H{"error":"forbidden"}); return }

    var body struct{
        Name      *string `json:"name"`
        UserGroup *string `json:"group"`
        Note      *string `json:"note"`
    }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    out, err := h.users.UpdateProfile(c.Request.Context(), id, services.UpdateUserFields{Name: body.Name, UserGroup: body.UserGroup, Note: body.Note})
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, out)
}

// assignRole sets user's role; admin only.
func (h *UsersHandler) assignRole(c *gin.Context) {
    if h.users == nil || h.auth == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    claims, ok := h.requireAuth(c)
    if !ok { return }
    if claims.Role != "admin" && claims.Role != "manager" { c.JSON(http.StatusForbidden, gin.H{"error":"forbidden"}); return }
    targetUserID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ Role string `json:"role"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if strings.TrimSpace(body.Role) == "" { c.JSON(http.StatusBadRequest, gin.H{"error":"validation_error", "message":"role required"}); return }
    if err := h.users.AssignRole(c.Request.Context(), claims.UserID, claims.Role, targetUserID, body.Role); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

// setActive toggles user's active status; admin only.
func (h *UsersHandler) setActive(c *gin.Context) {
    if h.users == nil || h.auth == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    claims, ok := h.requireAuth(c)
    if !ok { return }
    if claims.Role != "admin" && claims.Role != "manager" { c.JSON(http.StatusForbidden, gin.H{"error":"forbidden"}); return }
    targetUserID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ Active *bool `json:"active"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if body.Active == nil { c.JSON(http.StatusBadRequest, gin.H{"error":"validation_error", "message":"active required"}); return }
    if err := h.users.SetActive(c.Request.Context(), claims.UserID, claims.Role, targetUserID, *body.Active); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

// setPassword sets or resets user's password without old password check; admin only.
func (h *UsersHandler) setPassword(c *gin.Context) {
    if h.auth == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    claims, ok := h.requireAuth(c)
    if !ok { return }
    if claims.Role != "admin" && claims.Role != "manager" { c.JSON(http.StatusForbidden, gin.H{"error":"forbidden"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    var body struct{ NewPassword string `json:"new_password"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if strings.TrimSpace(body.NewPassword) == "" { c.JSON(http.StatusBadRequest, gin.H{"error":"validation_error", "message":"new_password required"}); return }
    if err := h.auth.SetInitialPassword(c.Request.Context(), id, body.NewPassword); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

// delete removes a user; admin only.
func (h *UsersHandler) delete(c *gin.Context) {
    if h.users == nil || h.auth == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    claims, ok := h.requireAuth(c)
    if !ok { return }
    if claims.Role != "admin" && claims.Role != "manager" { c.JSON(http.StatusForbidden, gin.H{"error":"forbidden"}); return }
    targetUserID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    if err := h.users.Delete(c.Request.Context(), claims.UserID, claims.Role, targetUserID); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}

// requireAuth parses bearer token and returns claims; writes 401 if invalid.
func (h *UsersHandler) requireAuth(c *gin.Context) (*services.Claims, bool) {
    token := bearerToken(c)
    if token == "" { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return nil, false }
    claims, err := h.auth.ParseToken(c.Request.Context(), token)
    if err != nil { writeSvcError(c, err); return nil, false }
    return claims, true
}