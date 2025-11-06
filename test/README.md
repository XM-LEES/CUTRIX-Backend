# CUTRIX Test

本目录包含用于本地和 CI 的集成测试设施。

- `docker-compose.yml`：启动 PostgreSQL 16（数据库/用户/密码均为 `cutrix`）。
- `run.ps1`：PowerShell 启动脚本，拉起容器、等待健康检查、设置 `DATABASE_URL` 并运行 `go test ./test/integration -v`。传入 `-Down` 参数可在测试结束后自动销毁容器。
- `integration/`：集成测试代码，覆盖 Handler + Service + Repository + DB 链路。

使用方法：

1. 启动并运行测试（保留容器）：
   ```powershell
   pwsh ./test/run.ps1
   ```

2. 启动、运行并销毁容器：
   ```powershell
   pwsh ./test/run.ps1 -Down
   ```

3. 手动启动容器：
   ```powershell
   docker compose -f test/docker-compose.yml up -d
   $env:DATABASE_URL = "postgres://cutrix:cutrix@localhost:5432/cutrix?sslmode=disable"
   go test ./test/integration -v
   ```