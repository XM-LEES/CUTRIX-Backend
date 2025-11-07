# CUTRIX Backend API 测试脚本

本目录提供在不启动前端的情况下，通过后端 API 进行功能验证的 PowerShell 脚本。适用于 Windows/PowerShell 环境。

## 预备条件
- 已安装 Docker 与 Docker Compose（使用仓库中的 `test/docker-compose.yml`）。
- 已安装 Go（如需运行 `go run` 启动后端）。

## 脚本一览
- `start-db.ps1`：启动 PostgreSQL 容器并等待健康检查就绪。
- `start-api.ps1`：设置环境变量并启动后端 API。
- `demo-flow.ps1`：端到端调用典型业务流程（订单→计划→布局→任务→发布→日志→作废→参与者）。
- `cleanup.ps1`：停止并（可选）清理数据库容器与卷。
- `migrate.ps1`：在容器内部直接应用 SQL 迁移（通常无需使用，因为 API 启动会自动迁移）。

## 快速开始
1. 启动数据库：
   ```powershell
   .\scripts\start-db.ps1
   ```
2. 启动后端 API（默认端口 `:8080`）：
   ```powershell
   .\scripts\start-api.ps1
   ```
3. 运行端到端演示流程：
   ```powershell
   .\scripts\demo-flow.ps1
   ```
4. 测试完成后清理：
   ```powershell
   .\scripts\cleanup.ps1
   # 如需删除卷（数据会丢失）：
   .\scripts\cleanup.ps1 -Purge
   ```

## 说明
- 以上业务 API（订单/计划/布局/任务/日志）默认无需认证即可调用；用户相关接口需要登录，本演示不涉及。
- API 启动时会自动加载并应用 `migrations/000001_initial_schema.up.sql` 中的迁移脚本。
- 默认数据库连接串：`postgres://cutrix:cutrix@localhost:5432/cutrix?sslmode=disable`，可在 `start-api.ps1` 中修改。