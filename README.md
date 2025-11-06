# example-demo（Starter 模板）

一个可运行的最小 Starter 模板，基于 `gin + SQLite`，包含两张表（`users` 与 `todos`）、完整 CRUD、可选 `user_id` 过滤、请求ID中间件，以及最小化配置与结构化日志封装。

## 运行

1. 安装 Go（1.20+）
2. 在仓库根目录执行：

```
go run example-demo/cmd/api/main.go
```

可通过环境变量调整：
- `PORT`（默认 `:8080`）
- `DB_PATH`（默认 `example-demo.db`）

首次运行会在当前目录生成 SQLite 文件，自动创建 `users` 与 `todos` 表，并尝试给 `todos` 添加 `user_id` 列（老数据兼容）。

## API

- `GET /api/v1/health` 健康检查

Users（用户CRUD）
- `POST /api/v1/users` 创建 `{ name, email }`
- `GET /api/v1/users` 列表
- `GET /api/v1/users/:id` 详情
- `PUT /api/v1/users/:id` 更新 `{ name?, email? }`
- `DELETE /api/v1/users/:id` 删除

Todos（待办CRUD，支持可选归属）
- `POST /api/v1/todos` 创建 `{ title, user_id? }`
- `GET /api/v1/todos` 列表，支持 `?user_id=` 过滤
- `GET /api/v1/todos/:id` 详情
- `PUT /api/v1/todos/:id` 更新 `{ title?, completed?, user_id? }`
- `DELETE /api/v1/todos/:id` 删除

## 示例请求

```
# 创建用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

# 创建 todo 绑定用户
curl -X POST http://localhost:8080/api/v1/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Learn Go","user_id":1}'

# 按用户过滤列表
curl http://localhost:8080/api/v1/todos?user_id=1

# 更新 todo 归属或完成状态
curl -X PUT http://localhost:8080/api/v1/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"completed":true}'

# 删除用户
curl -X DELETE http://localhost:8080/api/v1/users/1
```

## 代码结构

- `cmd/api/main.go` 入口：加载配置、初始化 SQLite（启用外键）、注册中间件与路由
- `internal/config/config.go` 最小配置（从环境变量读取端口与DB路径）
- `internal/logger/logger.go` 最小结构化日志封装（JSON行）
- `internal/middleware/cors.go` 最小 CORS 中间件
- `internal/middleware/requestid.go` 请求ID注入，便于日志关联
- `internal/models/*.go` 模型定义（`user`, `todo`）
- `internal/repositories/*.go` 数据访问（建表 + CRUD + 可选 `user_id` 支持）
- `internal/services/*.go` 服务层（薄封装）
- `internal/handlers/*.go` 路由与请求处理（users 与 todos）

## 设计与扩展建议

- 迁移：后续改为 `migrations/` 统一管理；当前仅示例建表与兼容列添加
- 校验：引入 `validator/v10` 做更严谨的参数校验
- 日志：需要更强日志时替换为 `zap` 并集成 trace/request_id
- 配置：根据环境引入 `viper` 或 `envconfig` 支持多配置源
- 认证：新增 `auth` 模块，提供 JWT 登录与认证中间件
- 事务：在服务层引入事务包装（尤其多表写入场景）