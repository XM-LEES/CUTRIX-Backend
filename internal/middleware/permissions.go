package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

// RequirePermissions checks whether current user's role grants at least one of the required permissions.
// - Permissions follow the form "module:action" (e.g., "log:create", "task:read").
// - RolePermissions may include wildcards like "module:*" to allow all actions under a module.
// - "admin" and "manager" are super roles with implicit full access.
func RequirePermissions(perms ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        v, ok := c.Get("role")
        if !ok || v == nil {
            // RequireAuth must set role; otherwise treat as unauthorized.
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
            return
        }
        role, _ := v.(string)
        role = strings.ToLower(strings.TrimSpace(role))

        // Super roles: admin/manager — full access
        if role == "admin" || role == "manager" {
            c.Next()
            return
        }

        // Evaluate permissions against role map
        allowed := getRolePermissions(role)
        if len(perms) == 0 {
            // No specific permission required
            c.Next()
            return
        }
        for _, p := range perms {
            if hasPermission(allowed, p) {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error":"forbidden"})
    }
}

// RolePermissionsMap defines allowed permissions for each role.
// Keys are lower-cased role names.
var RolePermissionsMap = map[string][]string{
    // worker: only submit/modify/void own logs, view tasks, view in-progress plans and layouts
    "worker": {
        "log:create",
        "log:update",
        "log:void", // Allow workers to void their own logs
        "task:read",
        "plan:read", // Allow workers to view plans (needed for WorkerDashboard)
        "layout:read", // Allow workers to view layouts (needed to associate tasks with plans)
    },
    // pattern_maker (打板员): full permissions for plan/layout/layout_ratios/task modules
    "pattern_maker": {
        "plan:*",
        "layout:*",
        "layout_ratios:*",
        "task:*",
    },
    // Other roles can be added here as needed
}

func getRolePermissions(role string) []string {
    if role == "" { return nil }
    if perms, ok := RolePermissionsMap[role]; ok { return perms }
    return nil
}

// hasPermission determines if any of the allowed permissions covers the required permission.
// Supports exact matches and module wildcard ("module:*").
func hasPermission(allowed []string, required string) bool {
    if len(allowed) == 0 { return false }
    req := strings.ToLower(strings.TrimSpace(required))
    if req == "" { return false }

    // Split by module:action
    var reqMod string
    if i := strings.IndexByte(req, ':'); i >= 0 {
        reqMod = req[:i]
    } else {
        reqMod = req
    }

    for _, a := range allowed {
        a = strings.ToLower(strings.TrimSpace(a))
        if a == req { return true }
        // Wildcard module:*
        if j := strings.IndexByte(a, ':'); j >= 0 {
            am := a[:j]
            aa := a[j+1:]
            if aa == "*" && am == reqMod {
                return true
            }
        }
    }
    return false
}