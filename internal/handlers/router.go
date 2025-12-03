package handlers

import (
    "github.com/gin-gonic/gin"
    "cutrix-backend/internal/middleware"
    "cutrix-backend/internal/services"
)

// RegisterRoutes centralizes route registration for finalized auth and users modules.
// - Public: /auth/login, /auth/refresh, /users(list/get)
// - Authenticated: /auth/password, /users/:id/profile
// - Admin: user creation, role assignment, activation, password reset, deletion
func RegisterRoutes(api *gin.RouterGroup, authSvc services.AuthService, usersSvc services.UsersService) {
    // Health remains simple
    NewHealthHandler().Register(api)

    // Auth
    authH := NewAuthHandler(authSvc)
    api.POST("/auth/login", authH.login)
    api.POST("/auth/refresh", authH.refresh)

    protected := api.Group("")
    protected.Use(middleware.RequireAuth(authSvc))
    protected.PUT("/auth/password", authH.changePassword)

    // Users
    usersH := NewUsersHandler(usersSvc, authSvc)
    // Public (no auth)
    api.GET("/users", usersH.list)
    api.GET("/users/:id", usersH.get)

    // Authenticated user (self or admin)
    protected.PATCH("/users/:id/profile", usersH.updateProfile)

    // Admin/Manager routes
    admin := protected.Group("")
    admin.Use(middleware.RequireRoles("admin", "manager"))
    admin.POST("/users", usersH.create)
    admin.PUT("/users/:id/role", usersH.assignRole)
    admin.PUT("/users/:id/active", usersH.setActive)
    admin.PUT("/users/:id/password", usersH.setPassword)
    admin.DELETE("/users/:id", usersH.delete)
}