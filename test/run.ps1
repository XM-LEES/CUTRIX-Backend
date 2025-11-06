param(
  [switch]$Keep,   # 保留容器，不自动清理
  [switch]$Purge   # 同时清理数据卷（与自动清理一起使用）
)

$exitCode = 0

try {
  Write-Host "[CUTRIX] Starting Postgres via docker compose..."
  docker compose -f test/docker-compose.yml up -d

  Write-Host "[CUTRIX] Waiting for Postgres health..."
  $retries=60
  $status=""
  for ($i=0; $i -lt $retries; $i++) {
    $status = docker inspect --format "{{if .State.Health}}{{.State.Health.Status}}{{end}}" cutrix-postgres 2>$null
    if ($status -eq "healthy") { break }
    Start-Sleep -Seconds 2
  }

  if ($status -ne "healthy") {
    Write-Error "Postgres not healthy (status=$status)"
    $exitCode = 1
  } else {
    $env:DATABASE_URL = "postgres://cutrix:cutrix@localhost:5432/cutrix?sslmode=disable"
    Write-Host "[CUTRIX] DATABASE_URL set for this session: $env:DATABASE_URL"

    Write-Host "[CUTRIX] Running integration tests..."
    # 仅运行 test/integration 下的测试
    go test ./test/integration -v
    $exitCode = $LASTEXITCODE
  }
}
finally {
  if (-not $Keep) {
    Write-Host "[CUTRIX] Tearing down Postgres container..."
    if ($Purge) {
      docker compose -f test/docker-compose.yml down -v
    } else {
      docker compose -f test/docker-compose.yml down
    }
  } else {
    Write-Host "[CUTRIX] Keeping containers as requested (-Keep)."
  }
}

exit $exitCode