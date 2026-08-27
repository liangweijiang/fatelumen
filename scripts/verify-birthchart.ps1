[CmdletBinding()]
param(
    [string]$OutputDirectory = "",
    [switch]$Precision,
    [switch]$Full,
    [ValidateRange(1, 64)][int]$FullWorkers = 8,
    [ValidateRange(0, 1000000)][int]$FullLimit = 0
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$backendRoot = Join-Path $repoRoot "backend"
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $backendRoot "tmp\acceptance"
}

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$env:GOCACHE = Join-Path $backendRoot "tmp\acceptance-gocache"
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$coreLog = Join-Path $OutputDirectory "birthchart-core-$timestamp.log"
$consistencyLog = Join-Path $OutputDirectory "birthchart-consistency-$timestamp.log"
$reportPath = Join-Path $OutputDirectory "birthchart-acceptance-$timestamp.md"
$fullLog = Join-Path $OutputDirectory "birthchart-full-$timestamp.log"
$precisionLog = Join-Path $OutputDirectory "birthchart-precision-$timestamp.log"

Push-Location $backendRoot
try {
    & go test ./internal/birthchart -run 'TestGlobalSameInstantAcceptance|TestGlobalAcceptanceSameInstantInvariants|TestIANATimezoneResolverHistoricalDST|TestIANATimezoneResolverNonHourOffsets|TestIANATimezoneResolverRejectsDSTGapAndOverlap|TestEngineAppliesTrueSolarCrossDayBeforeMidnightRule|TestEngineMidnightBoundaryRegression|TestLunarGoMidnight00Boundary|TestSolarTimeLongitudeCorrectionDirection|TestInputNormalizerLunarAndLeapMonthRegression' -count=1 -v 2>&1 | Tee-Object -FilePath $coreLog
    $coreExit = $LASTEXITCODE

    & go test ./internal/service -run 'TestGlobalFreeAndReportPathConsistency|TestFreeAndReportPathsProduceIdenticalHashAndPillars' -count=1 -v 2>&1 | Tee-Object -FilePath $consistencyLog
    $consistencyExit = $LASTEXITCODE

    $precisionExit = 0
    if ($Precision) {
        & go test ./internal/birthchart -run 'TestSolarTimePrecisionAgainstNOAAJulianCenturyReference' -count=1 -v 2>&1 | Tee-Object -FilePath $precisionLog
        $precisionExit = $LASTEXITCODE
    }

    $fullExit = 0
    if ($Full) {
        if ([string]::IsNullOrWhiteSpace($env:DB_HOST)) { $env:DB_HOST = "127.0.0.1" }
        if ([string]::IsNullOrWhiteSpace($env:DB_PORT)) { $env:DB_PORT = "3307" }
        if ([string]::IsNullOrWhiteSpace($env:DB_USER)) { $env:DB_USER = "fatelumen" }
        if ([string]::IsNullOrWhiteSpace($env:DB_PASSWORD)) { $env:DB_PASSWORD = "fatelumen123" }
        if ([string]::IsNullOrWhiteSpace($env:DB_NAME)) { $env:DB_NAME = "fatelumen" }
        & go run ./cmd/verify-birthchart-full -workers $FullWorkers -limit $FullLimit -output $OutputDirectory 2>&1 | Tee-Object -FilePath $fullLog
        $fullExit = $LASTEXITCODE
    }
} finally {
    Pop-Location
}

$coreStatus = if ($coreExit -eq 0) { "PASS" } else { "FAIL" }
$consistencyStatus = if ($consistencyExit -eq 0) { "PASS" } else { "FAIL" }
$fullStatus = if (-not $Full) { "NOT_RUN" } elseif ($fullExit -eq 0) { "PASS" } else { "FAIL" }
$precisionStatus = if (-not $Precision) { "NOT_RUN" } elseif ($precisionExit -eq 0) { "PASS" } else { "FAIL" }
$overall = if ($coreExit -eq 0 -and $consistencyExit -eq 0 -and (-not $Precision -or $precisionExit -eq 0) -and (-not $Full -or $fullExit -eq 0)) { "PASS" } else { "FAIL" }
$report = @(
    "# Birth Chart Acceptance Report",
    "",
    "- Executed at: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss zzz')",
    "- Overall: $overall",
    "- Global/timezone/solar/boundary suite: $coreStatus",
    "- Free-chart/report consistency suite: $consistencyStatus",
    "- Authoritative solar precision suite: $precisionStatus",
    "- Full geo database suite: $fullStatus",
    "- Core log: $coreLog",
    "- Consistency log: $consistencyLog",
    "- Precision log: $precisionLog",
    "- Full scan log: $fullLog",
    "",
    "Fixtures: backend/testdata/birthchart/global_acceptance_cases.json, backend/testdata/birthchart/precision_reference_cases.json"
) -join [Environment]::NewLine
Set-Content -LiteralPath $reportPath -Value $report -Encoding UTF8

Write-Host ""
Write-Host "Birth chart acceptance: $overall"
Write-Host "Report: $reportPath"
if ($overall -ne "PASS") {
    exit 1
}
