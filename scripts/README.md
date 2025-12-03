# CUTRIX Backend 脚本工具

本目录提供一些辅助脚本，用于开发和测试。

## 脚本说明

### `migrate.ps1` - 手动数据库迁移

在 Docker 容器中手动执行数据库迁移（通常不需要，因为 API 启动时会自动迁移）。

```powershell
# 使用默认容器名
.\scripts\migrate.ps1

# 指定容器名
.\scripts\migrate.ps1 -Container "cutrix-postgres"
```

### `demo-flow.ps1` - 端到端演示流程

测试完整的业务流程：订单 → 计划 → 布局 → 任务 → 发布 → 日志。

```powershell
# 使用默认 API 地址
.\scripts\demo-flow.ps1

# 指定 API 地址
.\scripts\demo-flow.ps1 -BaseUrl "http://localhost:8080/api/v1"
```

**注意**：需要先启动 Docker 服务（使用项目根目录的 `docker-compose up`）。

## 推荐工作流

现在推荐使用项目根目录的统一 Docker 管理：

```powershell
# 在项目根目录
.\start.ps1 dev -Detached    # 启动所有服务
.\start.ps1 logs             # 查看日志
.\start.ps1 stop             # 停止服务
```

详细说明请查看项目根目录的 `README.md`。
