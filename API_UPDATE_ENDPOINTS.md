# 后端修改接口清单

## 📋 概述

后端设计遵循**最小权限原则**，只允许修改必要的字段，其他字段通过数据库触发器保护，防止误操作。

---

## 🔧 可修改的接口

### 1. **订单 (Orders)** - 仅限 2 个字段

#### ✅ PATCH `/api/v1/orders/:id/note`
- **权限要求：** `admin` 或 `manager`
- **可修改字段：** `note`（备注）
- **请求格式：**
  ```json
  {
    "note": "新的备注信息"  // 可为 null
  }
  ```
- **说明：** 订单创建后，只能修改备注。其他字段（订单号、款号、客户名称、开始日期等）不可修改。

#### ✅ PATCH `/api/v1/orders/:id/finish-date`
- **权限要求：** `admin` 或 `manager`
- **可修改字段：** `order_finish_date`（完成日期）
- **请求格式：**
  ```json
  {
    "order_finish_date": "2024-12-31T00:00:00Z"  // RFC3339 格式，可为 null
  }
  ```
- **说明：** 只能修改完成日期，用于记录订单实际完成时间。

#### ❌ **不可修改的字段：**
- `order_number`（订单号）- 创建后不可修改
- `style_number`（款号）- 创建后不可修改
- `customer_name`（客户名称）- 创建后不可修改
- `order_start_date`（开始日期）- 创建后不可修改
- `order_items`（订单明细）- **完全不可修改**（数据库触发器保护）

---

### 2. **计划 (Plans)** - 仅限 1 个字段

#### ✅ PATCH `/api/v1/plans/:id/note`
- **权限要求：** `plan:update` 权限
- **可修改字段：** `note`（备注）
- **请求格式：**
  ```json
  {
    "note": "新的备注信息"  // 可为 null
  }
  ```
- **说明：** 
  - 计划发布后，**只能修改备注**
  - 发布前可以修改其他字段（通过删除重建实现）

#### ✅ POST `/api/v1/plans/:id/publish`
- **权限要求：** `plan:publish` 权限
- **功能：** 发布计划（状态：`pending` → `in_progress`）
- **说明：** 发布后，计划的结构性字段（名称、版型、任务等）被锁定

#### ✅ POST `/api/v1/plans/:id/freeze`
- **权限要求：** `plan:freeze` 权限
- **功能：** 冻结计划（状态：`completed` → `frozen`）
- **说明：** 冻结后计划完全锁定

#### ❌ **不可修改的字段：**
- `plan_name`（计划名称）- 发布后不可修改
- `order_id`（关联订单）- 创建后不可修改
- 版型结构 - 发布后不可修改
- 任务结构 - 发布后不可修改

---

### 3. **版型 (Layouts)** - 2 个字段，但有限制

#### ✅ PATCH `/api/v1/layouts/:id/name`
- **权限要求：** `layout:update` 权限
- **可修改字段：** `layout_name`（版型名称）
- **限制条件：** **仅当计划状态为 `pending` 时允许修改**
- **请求格式：**
  ```json
  {
    "name": "新版型名称"
  }
  ```
- **说明：** 计划发布后，版型名称不可修改。

#### ✅ PATCH `/api/v1/layouts/:id/note`
- **权限要求：** `layout:update` 权限
- **可修改字段：** `note`（备注）
- **限制条件：** 计划发布后仍可修改备注
- **请求格式：**
  ```json
  {
    "note": "新的备注信息"  // 可为 null
  }
  ```

#### ❌ **不可修改的字段：**
- `plan_id`（关联计划）- 创建后不可修改
- 尺码比例 - 发布后不可修改

---

### 4. **任务 (Tasks)** - 无直接修改接口

#### ❌ **任务没有修改接口**
- **说明：** 任务的状态和完成层数通过**日志 (Logs)** 自动更新
- **任务状态更新：** 通过提交日志，数据库触发器自动更新 `completed_layers` 和 `status`
- **任务字段：** `layout_id`、`color`、`planned_layers` 等创建后不可修改

---

### 5. **日志 (Logs)** - 仅作废功能

#### ✅ PATCH `/api/v1/logs/:id`
- **权限要求：** `log:update` 权限（worker 可操作）
- **功能：** 作废日志（软删除）
- **请求格式：**
  ```json
  {
    "voided": true,
    "void_reason": "作废原因"
  }
  ```
- **说明：** 
  - 只能作废，**不能取消作废**
  - 作废后，任务的 `completed_layers` 会自动减少
  - 日志的其他字段（`task_id`、`layers_completed`、`log_time` 等）**完全不可修改**

#### ❌ **不可修改的字段：**
- `task_id` - 不可修改
- `layers_completed` - 不可修改
- `log_time` - 不可修改
- `worker_id` - 不可修改
- `worker_name` - 不可修改
- `note` - 不可修改

---

### 6. **用户 (Users)** - 多个字段，但有权限限制

#### ✅ PATCH `/api/v1/users/:id/profile`
- **权限要求：** 本人或 `admin`/`manager`
- **可修改字段：** `name`（姓名）、`group`（分组）、`note`（备注）
- **请求格式：**
  ```json
  {
    "name": "新姓名",      // 可选
    "group": "新分组",     // 可选
    "note": "新备注"       // 可选
  }
  ```
- **说明：** 只能修改非敏感字段

#### ✅ PUT `/api/v1/users/:id/role`
- **权限要求：** `admin` 或 `manager`
- **可修改字段：** `role`（角色）
- **请求格式：**
  ```json
  {
    "role": "worker"  // admin, manager, pattern_maker, worker
  }
  ```
- **业务规则限制：**
  - Admin/Manager 不能修改自己的角色
  - Manager 不能修改 Admin 的角色
  - 系统只能有一个 Admin 和一个 Manager

#### ✅ PUT `/api/v1/users/:id/active`
- **权限要求：** `admin` 或 `manager`
- **可修改字段：** `is_active`（启用状态）
- **请求格式：**
  ```json
  {
    "active": true  // 或 false
  }
  ```
- **业务规则限制：**
  - Admin/Manager 不能停用自己

#### ✅ PUT `/api/v1/users/:id/password`
- **权限要求：** `admin` 或 `manager`
- **功能：** 重置用户密码（无需旧密码）
- **请求格式：**
  ```json
  {
    "new_password": "新密码"
  }
  ```
- **业务规则限制：**
  - Manager 不能重置 Admin 的密码

---

## 🛡️ 数据库保护机制

### 触发器保护

1. **订单更新保护** (`production.guard_orders_update`)
   - 只允许修改 `note` 和 `order_finish_date`
   - 其他字段修改会抛出异常

2. **订单项保护** (`production.prevent_order_items_update`)
   - **完全禁止修改订单项**
   - 任何修改尝试都会抛出异常

3. **计划更新保护** (`production.guard_plan_update`)
   - 发布后只允许修改 `note`
   - 其他字段修改会抛出异常

4. **版型保护** (`production.guard_layouts_by_plan_status`)
   - 计划发布后，版型结构不可修改

5. **任务保护** (`production.guard_tasks_by_plan_status`)
   - 计划发布后，任务结构不可修改

6. **日志保护** (`production.guard_logs_update`)
   - 只允许修改作废相关字段
   - 其他字段修改会抛出异常

---

## 📊 修改权限总结表

| 实体 | 可修改字段 | 限制条件 | 权限要求 |
|------|-----------|---------|---------|
| **订单** | `note`, `order_finish_date` | 无 | admin/manager |
| **订单项** | ❌ 无 | 完全不可修改 | - |
| **计划** | `note` | 发布后只能改备注 | plan:update |
| **版型** | `name`（发布前）, `note`（始终） | 发布后名称不可改 | layout:update |
| **任务** | ❌ 无直接接口 | 通过日志自动更新 | - |
| **日志** | `voided`, `void_reason` | 只能作废，不能取消 | log:update |
| **用户** | `name`, `group`, `note`, `role`, `is_active`, `password` | 有业务规则限制 | 见上表 |

---

## 🔍 检查方法

### 查看数据库触发器
```sql
-- 查看所有触发器
SELECT trigger_name, event_object_table, action_statement 
FROM information_schema.triggers 
WHERE trigger_schema = 'production';
```

### 测试修改限制
尝试修改被保护的字段会返回数据库错误，例如：
```
仅允许更新 note 和 order_finish_date
```

---

## 💡 设计理念

1. **数据完整性优先**：核心业务数据（订单号、款号、订单项）创建后不可修改
2. **审计追踪**：通过限制修改，确保历史记录的可追溯性
3. **状态机控制**：计划发布后锁定结构性字段，防止误操作
4. **最小权限**：只开放必要的修改接口，减少安全风险

---

**总结：后端只允许修改备注、完成日期等非核心字段，核心业务数据（订单号、订单项、任务结构等）创建后完全不可修改。**

