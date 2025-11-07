param(
    [switch]$Purge,
    [string]$Compose = (Join-Path $PSScriptRoot "..\test\docker-compose.yml")
)

Write-Host "[cleanup] 停止 PostgreSQL 容器..."
if ($Purge) {
    Write-Host "[cleanup] 将移除数据卷 (pgdata)，数据不可恢复。" -ForegroundColor Yellow
    docker compose -f $Compose down -v
} else {
    docker compose -f $Compose down
}

Write-Host "[cleanup] 完成。"