# example 后端骨架（极简DEMO）

目标：快速启动新项目的最小可用架构，不包含任何业务，仅保留分层与路由骨架。

包含：
- 入口：`cmd/api/main.go`（注册中间件、路由、健康检查）
- 分层：`internal/handlers`、`internal/services`、`internal/repositories`、`internal/models`、`internal/config`
- 中间件占位：`pkg/middleware/middleware.go`

使用建议：
- 在 `handlers` 仅做参数绑定与响应返回；业务逻辑写在 `services`；数据访问写在 `repositories`。
- 事务仅在 `services` 开启，`repositories` 提供 `tx` 方法。
- 保持统一响应格式与错误码约定；中间件用于认证/授权/日志/恢复等横切关注。

运行（示意）：
- 可在 `go.mod` 中设定模块名并按需添加依赖；当前骨架不强制可运行，仅用于结构参考。
