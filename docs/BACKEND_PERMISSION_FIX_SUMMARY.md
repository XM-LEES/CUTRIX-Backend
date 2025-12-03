# 后端权限约束修复总结

## ✅ 修复完成

所有权限约束已成功在后端实现，现在即使绕过前端检查，后端也会阻止违规操作。

---

## 修改的文件

### 1. Service 接口层
**文件：** `internal/services/users_service.go`

**修改内容：**
- `Create()` - 添加 `currentUserID` 和 `currentUserRole` 参数
- `AssignRole()` - 添加 `currentUserID`、`currentUserRole` 和 `targetUserID` 参数
- `SetActive()` - 添加 `currentUserID`、`currentUserRole` 和 `targetUserID` 参数
- `Delete()` - 添加 `currentUserID`、`currentUserRole` 和 `targetUserID` 参数

### 2. Service 实现层
**文件：** `internal/services/users_service_impl.go`

**添加的业务规则检查：**

#### `Create()` 方法：
- ✅ Manager 不能创建 Admin
- ✅ Manager 不能创建 Manager
- ✅ 系统只能有一个 Admin（检查活跃的 Admin）
- ✅ 系统只能有一个 Manager（检查活跃的 Manager）

#### `AssignRole()` 方法：
- ✅ Manager 不能操作 Admin
- ✅ Admin/Manager 不能修改自己的角色
- ✅ 系统只能有一个 Admin（检查活跃的 Admin）
- ✅ 系统只能有一个 Manager（检查活跃的 Manager）

#### `SetActive()` 方法：
- ✅ Manager 不能操作 Admin
- ✅ Admin/Manager 不能停用自己

#### `Delete()` 方法：
- ✅ Manager 不能删除 Admin
- ✅ Admin/Manager 不能删除自己

### 3. Handler 层
**文件：** `internal/handlers/users_handler.go`

**修改内容：**
- `create()` - 传递 `claims.UserID` 和 `claims.Role` 给 Service
- `assignRole()` - 传递 `claims.UserID`、`claims.Role` 和 `targetUserID` 给 Service
- `setActive()` - 传递 `claims.UserID`、`claims.Role` 和 `targetUserID` 给 Service
- `delete()` - 传递 `claims.UserID`、`claims.Role` 和 `targetUserID` 给 Service

### 4. 数据库约束
**文件：** `migrations/000001_initial_schema.up.sql`

**添加的唯一索引：**
- ✅ `users_single_active_admin_idx` - 确保只有一个活跃的 Admin
- ✅ `users_single_active_manager_idx` - 确保只有一个活跃的 Manager

**说明：** 约束已直接添加到初始 schema 文件中（因为数据库为空，无需单独的 migration）

---

## 实现的约束列表

| 约束 | 实现位置 | 状态 |
|------|---------|------|
| Admin 不能修改自己的角色 | Service: `AssignRole()` | ✅ |
| Admin 不能停用自己的状态 | Service: `SetActive()` | ✅ |
| Admin 不能删除自己 | Service: `Delete()` | ✅ |
| Manager 不能对 Admin 做任何操作 | Service: `AssignRole()`, `SetActive()`, `Delete()` | ✅ |
| Manager 不能创建 Admin 用户 | Service: `Create()` | ✅ |
| Manager 不能创建新的 Manager 用户 | Service: `Create()` | ✅ |
| Manager 不能修改自己的角色 | Service: `AssignRole()` | ✅ |
| Manager 不能删除自己 | Service: `Delete()` | ✅ |
| Manager 不能修改自己的状态 | Service: `SetActive()` | ✅ |
| Admin 不能创建新的 Admin（系统只需一个） | Service: `Create()` + DB Index | ✅ |
| Manager 不能创建新的 Manager（系统只需一个） | Service: `Create()` + DB Index | ✅ |

---

## 安全改进

### 修复前：
- ❌ 所有约束只在前端实现
- ❌ 可以直接调用 API 绕过前端检查
- ❌ 数据库层面没有约束

### 修复后：
- ✅ 所有约束在后端 Service 层实现
- ✅ 即使绕过前端，后端也会阻止违规操作
- ✅ 数据库唯一索引作为最后一道防线
- ✅ 错误信息统一使用 `ErrForbidden`

---

## 错误处理

所有权限违规操作现在会返回：
- **HTTP 状态码：** `403 Forbidden`
- **错误类型：** `services.ErrForbidden`
- **错误消息：** `"forbidden"`

Handler 层的 `writeSvcError()` 会自动将 `ErrForbidden` 映射到 HTTP 403。

---

## 数据库迁移

### 运行迁移：
```bash
# 应用初始 schema（包含所有约束）
migrate -path migrations -database "postgres://..." up

# 或使用 Docker
docker exec -it cutrix-db psql -U cutrix -d cutrix -f /migrations/000001_initial_schema.up.sql
```

**说明：** 唯一索引约束已直接包含在初始 schema 文件中，无需单独运行迁移。

---

## 测试建议

### 1. 测试 Manager 不能操作 Admin：
```bash
# 使用 Manager token 尝试修改 Admin 角色
curl -X PUT http://localhost:3001/api/v1/users/1/role \
  -H "Authorization: Bearer <manager_token>" \
  -H "Content-Type: application/json" \
  -d '{"role":"worker"}'
# 预期：403 Forbidden
```

### 2. 测试 Admin 不能删除自己：
```bash
# 使用 Admin token 尝试删除自己
curl -X DELETE http://localhost:3001/api/v1/users/1 \
  -H "Authorization: Bearer <admin_token>"
# 预期：403 Forbidden
```

### 3. 测试系统只能有一个 Admin：
```bash
# 尝试创建第二个 Admin
curl -X POST http://localhost:3001/api/v1/users \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"admin2","role":"admin"}'
# 预期：403 Forbidden（如果已存在活跃的 Admin）
```

---

## 注意事项

1. **数据库迁移必须运行**：新的唯一索引约束需要应用到数据库
2. **现有数据检查**：如果数据库中已存在多个 Admin 或 Manager，迁移可能会失败。需要先清理数据。
3. **前端兼容性**：前端代码无需修改，因为错误处理逻辑已经存在。

---

## 完成状态

✅ **所有任务已完成**
- ✅ Service 接口修改
- ✅ Service 实现业务规则检查
- ✅ Handler 传递当前用户信息
- ✅ 数据库唯一索引约束
- ✅ 无 Linter 错误

**修复完成时间：** 2025-12-03

