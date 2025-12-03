# 后端权限约束分析

## 当前状态

### ✅ 已实现的约束
1. **Handler 层基础权限检查**
   - 创建用户：需要 `admin` 或 `manager` 角色
   - 分配角色：需要 `admin` 或 `manager` 角色
   - 设置状态：需要 `admin` 或 `manager` 角色
   - 删除用户：需要 `admin` 或 `manager` 角色
   - 重置密码：需要 `admin` 或 `manager` 角色

2. **数据库基础约束**
   - `role` 字段 CHECK 约束：只能为 `admin`, `manager`, `worker`, `pattern_maker`
   - `name` 字段 UNIQUE 约束：用户名唯一

### ❌ 缺失的约束（目前只在前端实现）

#### 1. 业务规则约束（必须在后端实现）

| 约束 | 当前状态 | 风险等级 |
|------|---------|---------|
| Admin 不能修改自己的角色 | ❌ 仅前端 | 🔴 高 |
| Admin 不能停用自己的状态 | ❌ 仅前端 | 🔴 高 |
| Admin 不能删除自己 | ❌ 仅前端 | 🔴 高 |
| Manager 不能对 Admin 做任何操作 | ❌ 仅前端 | 🔴 高 |
| Manager 不能创建 Admin 用户 | ❌ 仅前端 | 🔴 高 |
| Manager 不能创建新的 Manager 用户 | ❌ 仅前端 | 🟡 中 |
| Manager 不能修改自己的角色 | ❌ 仅前端 | 🔴 高 |
| Manager 不能删除自己 | ❌ 仅前端 | 🔴 高 |
| Manager 不能修改自己的状态 | ❌ 仅前端 | 🔴 高 |
| Admin 不能创建新的 Admin（系统只需一个） | ❌ 仅前端 | 🟡 中 |
| Manager 不能创建新的 Manager（系统只需一个） | ❌ 仅前端 | 🟡 中 |

#### 2. 数据库层面约束（推荐但不强制）

| 约束 | 实现方式 | 优先级 |
|------|---------|--------|
| 系统只能有一个 Admin | 唯一索引 + 触发器 | 🟡 中 |
| 系统只能有一个 Manager | 唯一索引 + 触发器 | 🟡 中 |
| 防止删除自己 | 触发器 | 🟢 低（应用层已足够）|

---

## 安全风险

### 🔴 高风险：可直接绕过前端检查

**示例攻击场景：**

1. **Admin 删除自己**
   ```bash
   # 直接调用 API，绕过前端检查
   curl -X DELETE http://localhost:3001/api/v1/users/1 \
     -H "Authorization: Bearer <admin_token>"
   ```

2. **Manager 创建 Admin**
   ```bash
   curl -X POST http://localhost:3001/api/v1/users \
     -H "Authorization: Bearer <manager_token>" \
     -H "Content-Type: application/json" \
     -d '{"name":"hacker","role":"admin"}'
   ```

3. **Manager 修改 Admin 角色**
   ```bash
   curl -X PUT http://localhost:3001/api/v1/users/1/role \
     -H "Authorization: Bearer <manager_token>" \
     -H "Content-Type: application/json" \
     -d '{"role":"worker"}'
   ```

---

## 解决方案

### 方案 A：后端 Service 层添加业务规则（推荐）✅

**优点：**
- 安全可靠，无法绕过
- 逻辑集中，易于维护
- 符合分层架构原则

**实现位置：**
- `internal/services/users_service_impl.go`

**需要修改的方法：**
1. `Create()` - 添加角色创建限制检查
2. `AssignRole()` - 添加自我修改限制检查
3. `SetActive()` - 添加自我停用限制检查
4. `Delete()` - 添加自我删除限制检查

**需要的信息：**
- 当前操作用户的 ID（从 JWT Claims 获取）
- 当前操作用户的角色（从 JWT Claims 获取）
- 目标用户的角色（从数据库查询）

### 方案 B：数据库触发器（可选）

**优点：**
- 数据库层面强制约束
- 即使应用层有 bug 也能保护

**缺点：**
- 维护复杂
- 错误信息不够友好
- 性能影响（虽然很小）

**实现方式：**
```sql
-- 限制 Admin 数量
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS 
  users_single_admin_idx ON public.users (role) 
  WHERE role = 'admin' AND is_active = true;

-- 限制 Manager 数量
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS 
  users_single_manager_idx ON public.users (role) 
  WHERE role = 'manager' AND is_active = true;
```

---

## 实施建议

### 优先级 1：必须实现（高安全风险）

1. ✅ **在 Service 层添加业务规则检查**
   - 修改 `Create()` 方法
   - 修改 `AssignRole()` 方法
   - 修改 `SetActive()` 方法
   - 修改 `Delete()` 方法

2. ✅ **在 Handler 层传递当前用户信息**
   - 从 JWT Claims 获取当前用户 ID 和角色
   - 传递给 Service 层进行权限检查

### 优先级 2：推荐实现（中安全风险）

3. ⚠️ **数据库唯一索引约束**
   - 防止创建多个 Admin/Manager
   - 作为最后一道防线

### 优先级 3：可选实现（低优先级）

4. ⚠️ **数据库触发器**
   - 防止删除自己（应用层已足够）
   - 防止停用自己（应用层已足够）

---

## 实施步骤

### Step 1: 修改 Service 接口

在 `internal/services/users_service.go` 中，为需要当前用户信息的方法添加参数：

```go
// 修改前
AssignRole(ctx context.Context, userID int, role string) error

// 修改后
AssignRole(ctx context.Context, currentUserID int, currentUserRole string, targetUserID int, role string) error
```

### Step 2: 实现业务规则检查

在 `internal/services/users_service_impl.go` 中实现：

```go
func (s *usersService) AssignRole(ctx context.Context, currentUserID int, currentUserRole string, targetUserID int, role string) error {
    // 1. Manager 不能操作 Admin
    if currentUserRole == "manager" {
        targetUser, err := s.repo.GetByID(ctx, targetUserID)
        if err != nil { return err }
        if targetUser.Role == "admin" {
            return errors.New("manager cannot operate on admin")
        }
    }
    
    // 2. Admin/Manager 不能修改自己的角色
    if (currentUserRole == "admin" || currentUserRole == "manager") && currentUserID == targetUserID {
        return errors.New("cannot modify own role")
    }
    
    // 3. 检查角色数量限制
    if role == "admin" {
        // 检查是否已存在 admin
        existing, err := s.repo.List(ctx, &role, nil, nil, nil)
        if err != nil { return err }
        if len(existing) > 0 {
            return errors.New("system already has an admin")
        }
    }
    
    return s.repo.UpdateRole(ctx, targetUserID, role)
}
```

### Step 3: 修改 Handler 传递当前用户信息

在 `internal/handlers/users_handler.go` 中：

```go
func (h *UsersHandler) assignRole(c *gin.Context) {
    claims, ok := h.requireAuth(c)
    if !ok { return }
    
    // 获取当前用户信息
    currentUserID := claims.UserID
    currentUserRole := claims.Role
    
    // 获取目标用户 ID
    targetUserID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_id"}); return }
    
    // 传递当前用户信息给 Service
    var body struct{ Role string `json:"role"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    
    if err := h.users.AssignRole(c.Request.Context(), currentUserID, currentUserRole, targetUserID, body.Role); err != nil {
        writeSvcError(c, err); return
    }
    
    c.Status(http.StatusNoContent)
}
```

---

## 总结

**必须实现：**
- ✅ 后端 Service 层业务规则检查（所有高安全风险的约束）

**推荐实现：**
- ⚠️ 数据库唯一索引（防止多个 Admin/Manager）

**可选实现：**
- ⚠️ 数据库触发器（作为额外保护层）

**当前风险：**
- 🔴 所有前端约束都可以通过直接调用 API 绕过
- 🔴 系统安全性完全依赖前端，这是不安全的

**建议：**
立即实施 **方案 A**（后端 Service 层添加业务规则检查），这是必须的，不能仅依赖前端检查。

