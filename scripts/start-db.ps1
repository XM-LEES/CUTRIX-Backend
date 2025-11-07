param(
    [string]$Compose = (Join-Path $PSScriptRoot "..\test\docker-compose.yml"),
    [string]$Container = "cutrix-postgres"
)

Write-Host "[start-db] 使用 Compose 启动 PostgreSQL..."
docker compose -f $Compose up -d

Write-Host "[start-db] 等待容器健康检查通过..."
$maxRetries = 60
$delaySeconds = 2
$tries = 0
$status = ""
while ($tries -lt $maxRetries) {
    try {
        $status = docker inspect -f '{{.State.Health.Status}}' $Container
        if ($status -eq "healthy") {
            Write-Host "[start-db] PostgreSQL 已就绪 (healthy)." -ForegroundColor Green
            break
        }
    } catch {
        # 容器或健康状态尚未可用，继续重试
    }
    Start-Sleep -Seconds $delaySeconds
    $tries++
}

if ($status -ne "healthy") {
    Write-Error "[start-db] PostgreSQL 未在预期时间内变为 healthy。请检查日志： docker logs $Container"
    exit 1
}

$dsn = "postgres://cutrix:cutrix@localhost:5432/cutrix?sslmode=disable"
Write-Host "[start-db] 建议的 DATABASE_URL： $dsn"