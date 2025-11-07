param(
    [string]$Container = "cutrix-postgres",
    [string]$SqlPath = (Join-Path $PSScriptRoot "..\migrations\000001_initial_schema.up.sql")
)

if (-not (Test-Path $SqlPath)) {
    Write-Error "[migrate] 找不到迁移文件: $SqlPath"
    exit 1
}

Write-Host "[migrate] 拷贝 SQL 到容器..."
docker cp $SqlPath "$Container:/tmp/schema.sql"

Write-Host "[migrate] 在容器内执行 psql 应用迁移..."
docker exec -i $Container psql -U cutrix -d cutrix -v ON_ERROR_STOP=1 -f /tmp/schema.sql

Write-Host "[migrate] 迁移完成。"