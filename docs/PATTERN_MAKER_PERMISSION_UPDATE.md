# Pattern Maker 权限更新文档

## 更新日期
2025-01-XX

## 更新原因
根据业务需求，调整了 `pattern_maker`（制版员）角色的权限配置，确保其只能管理未发布的计划，不能发布计划或修改已发布计划。

## 安全漏洞修复

### 发现的问题
1. **后端权限配置过于宽松**：`pattern_maker` 使用了通配符权限（`plan:*`），拥有所有计划相关权限，包括不应该拥有的 `plan:publish` 权限。
2. **缺少业务逻辑检查**：即使有权限，也应该根据计划状态限制操作（如不能修改已发布计划的备注、不能删除已发布的计划）。

### 修复内容

#### 1. 后端权限配置更新
**文件：** `internal/middleware/permissions.go`

**修改前：**
```go
"pattern_maker": {
    "plan:*",
    "layout:*",
    "layout_ratios:*",
    "task:*",
},
```

**修改后：**
```go
"pattern_maker": {
    "plan:create",
    "plan:read",
    "plan:update",
    "plan:delete",
    "layout:create",
    "layout:read",
    "layout:update",
    "layout:delete",
    "layout_ratios:create",
    "layout_ratios:read",
    "task:create",
    "task:delete",
    "order:read",
},
```

**关键变化：**
- ✅ 移除了 `plan:publish` 权限（不能发布计划）
- ✅ 移除了 `plan:freeze` 权限（不能冻结计划）
- ✅ 移除了 `task:read` 权限（不能查看任务管理页面）
- ✅ 移除了通配符权限，改为明确的权限列表

#### 2. Handler 层业务逻辑检查
**文件：** `internal/handlers/plans_handler.go`

**添加的检查：**

##### `updateNote()` 函数
- 检查：`pattern_maker` 只能修改未发布计划（`status === 'pending'`）的备注
- 已发布计划（`in_progress`, `completed`, `frozen`）的备注不能修改
- 返回：`403 Forbidden` 如果尝试修改已发布计划的备注

##### `delete()` 函数
- 检查：`pattern_maker` 只能删除未发布计划（`status === 'pending'`）
- 已发布计划不能删除
- 返回：`403 Forbidden` 如果尝试删除已发布的计划

**文件：** `internal/handlers/tasks_handler.go`

**权限调整：**

##### `listByLayout()` 接口
- **修改前**：只允许 `task:read` 权限
- **修改后**：允许 `task:read` 或 `layout:read` 权限
- **原因**：`pattern_maker` 没有 `task:read` 权限，但有 `layout:read` 权限。通过版型查看任务是合理的业务需求（在计划详情中需要显示任务信息），因此允许通过 `layout:read` 权限访问此接口。

#### 3. 前端权限配置更新
**文件：** `CUTRIX-Frontend/src/utils/permissions.ts`

**修改内容：**
- 移除了 `pattern_maker` 的 `plan:publish` 权限
- 移除了 `pattern_maker` 的 `task:read` 权限
- 添加了 `plan:delete` 权限（用于删除未发布计划）

#### 4. 前端 UI 更新
**文件：** `CUTRIX-Frontend/src/components/plans/usePlanTableColumns.tsx`

**修改内容：**
- 已发布计划的"修改备注"按钮对 `pattern_maker` 隐藏
- 已发布计划的"删除"按钮对 `pattern_maker` 隐藏

**文件：** `CUTRIX-Frontend/src/components/layout/Sidebar.tsx`

**修改内容：**
- 隐藏仪表板菜单（对 `pattern_maker`）
- 隐藏任务管理菜单（对 `pattern_maker`，无 `task:read` 权限）
- 隐藏日志记录菜单（对 `pattern_maker`，无 `log:create` 权限）

**文件：** `CUTRIX-Frontend/src/pages/Plans.tsx`

**修改内容：**
- 在 `handleViewDetail` 函数中添加权限检查
- 有 `task:read` 权限：使用 `tasksApi.list()` 加载所有任务（效率更高）
- 无 `task:read` 权限：使用 `tasksApi.listByLayout()` 通过版型逐个获取任务（使用 `layout:read` 权限）
- 错误处理：如果加载任务失败，静默处理，不影响计划详情显示

**文件：** `CUTRIX-Frontend/src/components/plans/PlanDetailModal.tsx`

**修改内容：**
- 隐藏"查看任务"按钮（对没有 `task:read` 权限的用户，如 `pattern_maker`）
- 添加 `canViewTasks` 权限检查

## 当前 Pattern Maker 权限

### 可访问的页面
- ✅ 生产订单（查看）
- ✅ 生产计划（创建、查看、修改、删除未发布计划）

### 可执行的操作
- ✅ 创建生产计划
- ✅ 查看生产计划（包括查看计划详情中的任务信息，通过 `layout:read` 权限）
- ✅ 修改未发布计划（包括添加版型和任务）
- ✅ 删除未发布计划
- ✅ 管理版型（创建、查看、修改、删除）
- ✅ 创建和删除任务（在创建/编辑计划时）
- ✅ 在计划详情中查看任务信息（通过版型接口，使用 `layout:read` 权限）

### 不可执行的操作
- ❌ 发布计划（无 `plan:publish` 权限）
- ❌ 冻结计划（无 `plan:freeze` 权限）
- ❌ 修改已发布计划的备注（Handler 层业务规则检查）
- ❌ 删除已发布的计划（Handler 层业务规则检查）
- ❌ 查看任务管理页面（无 `task:read` 权限）
- ❌ 查看日志记录（无 `log:create` 权限）
- ❌ 访问仪表板（UI 限制）

## 安全防护层级

现在系统有三层防护：

1. **前端权限检查**：UI 层面隐藏/禁用按钮
2. **后端权限中间件**：`RequirePermissions` 检查用户角色权限
3. **Handler 层业务逻辑**：根据计划状态和用户角色进行额外检查

即使攻击者绕过前端直接调用 API，也会被后端权限中间件和业务逻辑检查拦截。

## 测试建议

### 应该被拦截的操作（403 Forbidden）

1. **尝试发布计划**
   ```bash
   curl -X POST http://localhost:8080/api/v1/plans/1/publish \
     -H "Authorization: Bearer <pattern_maker_token>"
   # 预期：403 Forbidden（无 plan:publish 权限）
   ```

2. **尝试删除已发布计划**
   ```bash
   curl -X DELETE http://localhost:8080/api/v1/plans/1 \
     -H "Authorization: Bearer <pattern_maker_token>"
   # 预期：403 Forbidden（业务逻辑检查：只能删除 pending 计划）
   ```

3. **尝试修改已发布计划备注**
   ```bash
   curl -X PATCH http://localhost:8080/api/v1/plans/1/note \
     -H "Authorization: Bearer <pattern_maker_token>" \
     -H "Content-Type: application/json" \
     -d '{"note":"新备注"}'
   # 预期：403 Forbidden（业务逻辑检查：只能修改 pending 计划备注）
   ```

4. **尝试查看任务管理页面**
   ```bash
   curl -X GET http://localhost:8080/api/v1/tasks \
     -H "Authorization: Bearer <pattern_maker_token>"
   # 预期：403 Forbidden（无 task:read 权限）
   ```

### 应该成功的操作（200/204）

1. **删除未发布计划**
   ```bash
   curl -X DELETE http://localhost:8080/api/v1/plans/1 \
     -H "Authorization: Bearer <pattern_maker_token>"
   # 预期：204 No Content（如果计划状态为 pending）
   ```

2. **修改未发布计划备注**
   ```bash
   curl -X PATCH http://localhost:8080/api/v1/plans/1/note \
     -H "Authorization: Bearer <pattern_maker_token>" \
     -H "Content-Type: application/json" \
     -d '{"note":"新备注"}'
   # 预期：204 No Content（如果计划状态为 pending）
   ```

## 相关文件

### 后端
- `internal/middleware/permissions.go` - 权限配置
- `internal/handlers/plans_handler.go` - Handler 层业务逻辑检查
- `internal/handlers/tasks_handler.go` - 任务接口权限调整（`listByLayout` 允许 `layout:read`）

### 前端
- `src/utils/permissions.ts` - 前端权限配置
- `src/components/plans/usePlanTableColumns.tsx` - 计划列表操作按钮
- `src/components/layout/Sidebar.tsx` - 侧边栏菜单
- `src/pages/Plans.tsx` - 计划详情加载逻辑（支持通过 `layout:read` 获取任务）
- `src/components/plans/PlanDetailModal.tsx` - 计划详情模态框（隐藏"查看任务"按钮）

## 注意事项

1. **数据库迁移**：无需数据库迁移，这是代码层面的权限调整。
2. **现有数据**：不影响现有数据，只是限制了操作权限。
3. **前端兼容性**：前端代码已同步更新，确保 UI 与后端权限一致。

## 完成状态

✅ **所有安全修复已完成**
- ✅ 后端权限配置更新
- ✅ Handler 层业务逻辑检查
- ✅ 前端权限配置更新
- ✅ 前端 UI 更新
- ✅ 无 Linter 错误

