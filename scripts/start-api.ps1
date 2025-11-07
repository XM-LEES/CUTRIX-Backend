param(
    [string]$Port = ":8080",
    [string]$DatabaseUrl = "postgres://cutrix:cutrix@localhost:5432/cutrix?sslmode=disable"
)

$env:PORT = $Port
$env:DATABASE_URL = $DatabaseUrl

Write-Host "[start-api] 启动后端 API，PORT=$env:PORT，DATABASE_URL=$env:DATABASE_URL"

$repoRoot = (Join-Path $PSScriptRoot "..")
Push-Location $repoRoot
try {
    go run ./cmd/api
} finally {
    Pop-Location
}