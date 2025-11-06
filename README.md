# CUTRIX-Backend

一个可扩展的后端骨架，参考 example 与 example-demo：
- Gin 入口、路由分组、健康检查
- 中间件：CORS、RequestID
- 配置：从环境变量读取端口
- 预留分层：handlers / services / repositories / models / logger

## 运行

```
go run cmd/api/main.go
```

可用环境变量：
- `PORT`（默认 `:8080`）

## 路由
- `GET /api/v1/health` 健康检查

## 结构
- `cmd/api/main.go` 入口
- `db/functions` 数据库函数
- `internal/config` 端口配置
- `internal/db` 数据库连接
- `internal/middleware` CORS 与 RequestID
- `internal/handlers` 健康检查处理器
- `internal/logger` 最小 JSON 日志
- `internal/services` 业务层占位
- `internal/repositories` 数据访问占位
- `internal/models` 模型占位
- `migrations` 数据库迁移
- `test/integration` 集成测试
