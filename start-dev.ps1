[CmdletBinding()]
param(
    [string]$ProxyUrl = "http://127.0.0.1:10808",
    [switch]$Restart
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$backendRoot = Join-Path $repoRoot "backend"
$frontendRoot = Join-Path $repoRoot "frontend"
$runtimeRoot = Join-Path $backendRoot "tmp\dev"
$serverExe = Join-Path $runtimeRoot "fatelumen-server.exe"
$logRoot = Join-Path $runtimeRoot "logs"

function Get-ListeningProcessId([int]$Port) {
    $listener = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if ($listener) { return [int]$listener.OwningProcess }
    return 0
}

function Stop-DevPort([int]$Port, [string]$Name) {
    $processId = Get-ListeningProcessId $Port
    if ($processId -eq 0) { return }
    if (-not $Restart) {
        throw "$Name is already listening on port $Port (PID $processId). Re-run with -Restart to replace it."
    }
    Write-Host "Stopping $Name (PID $processId)..."
    Stop-Process -Id $processId -ErrorAction Stop
}

function Wait-Http([string]$Url, [string]$Name) {
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 3
            if ($response.StatusCode -eq 200) {
                Write-Host "$Name ready: $Url"
                return
            }
        } catch {
            Start-Sleep -Milliseconds 500
        }
    }
    throw "$Name did not become ready. Check logs in $logRoot."
}

Set-Location $repoRoot
New-Item -ItemType Directory -Force -Path $runtimeRoot, $logRoot | Out-Null

$proxyUri = [Uri]$ProxyUrl
if (-not (Test-NetConnection -ComputerName $proxyUri.Host -Port $proxyUri.Port -InformationLevel Quiet)) {
    throw "Development proxy is not listening at $ProxyUrl. Start v2rayN or pass -ProxyUrl with the active HTTP proxy."
}

Write-Host "Starting MySQL and Redis..."
docker compose up -d mysql redis

Stop-DevPort 8080 "backend"
Stop-DevPort 3000 "frontend"

$env:APP_ENV = "development"
$env:APP_PORT = "8080"
$env:APP_BASE_URL = "http://localhost:8080"
$env:WEB_BASE_URL = "http://localhost:3000"
$env:DB_HOST = "127.0.0.1"
$env:DB_PORT = "3307"
$env:DB_USER = "fatelumen"
$env:DB_PASSWORD = "fatelumen123"
$env:DB_NAME = "fatelumen"
$env:REDIS_ADDR = "127.0.0.1:6380"
$localEnvPath = Join-Path $backendRoot ".env.local"
$hasDeepSeekKey = (Test-Path $localEnvPath) -and
    [bool](Get-Content $localEnvPath | Where-Object { $_ -match '^DEEPSEEK_API_KEY=.+$' } | Select-Object -First 1)
if (-not $hasDeepSeekKey) {
    $env:DEEPSEEK_API_KEY = "local-development-placeholder"
} else {
    Remove-Item Env:DEEPSEEK_API_KEY -ErrorAction SilentlyContinue
}
$env:HTTP_PROXY = $ProxyUrl
$env:HTTPS_PROXY = $ProxyUrl
$env:NO_PROXY = "localhost,127.0.0.1,::1,mysql,redis"
$env:GOCACHE = Join-Path $runtimeRoot "gocache"

Write-Host "Building backend from current source..."
Push-Location $backendRoot
try {
    go build -o $serverExe .\cmd\server
} finally {
    Pop-Location
}

$backend = Start-Process -FilePath $serverExe `
    -WorkingDirectory $backendRoot `
    -WindowStyle Hidden `
    -RedirectStandardOutput (Join-Path $logRoot "backend.stdout.log") `
    -RedirectStandardError (Join-Path $logRoot "backend.stderr.log") `
    -PassThru

$frontend = Start-Process -FilePath "npm.cmd" `
    -ArgumentList "run", "dev:turbo" `
    -WorkingDirectory $frontendRoot `
    -WindowStyle Hidden `
    -RedirectStandardOutput (Join-Path $logRoot "frontend.stdout.log") `
    -RedirectStandardError (Join-Path $logRoot "frontend.stderr.log") `
    -PassThru

Wait-Http "http://127.0.0.1:8080/health" "Backend"
Wait-Http "http://127.0.0.1:3000/zh" "Frontend"

Write-Host ""
Write-Host "FateLumen development services are running."
Write-Host "Frontend:    http://localhost:3000/zh"
Write-Host "Admin:       http://localhost:3000/admin/login"
Write-Host "Backend PID: $($backend.Id)"
Write-Host "Frontend PID: $($frontend.Id)"
Write-Host "Google proxy: $ProxyUrl"
Write-Host "Logs:        $logRoot"
