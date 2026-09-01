param(
    [string]$Remote = "origin",
    [string]$Branch = "",
    [string]$Proxy = "socks5h://127.0.0.1:10808",
    [switch]$Direct
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Branch)) {
    $Branch = (git branch --show-current).Trim()
}

if ([string]::IsNullOrWhiteSpace($Branch)) {
    throw "Cannot detect current Git branch. Pass one with -Branch."
}

if ($Direct) {
    Write-Host "Pushing directly: $Remote $Branch"
    git push $Remote $Branch
} else {
    Write-Host "Pushing through proxy ${Proxy}: $Remote $Branch"
    git -c "http.proxy=$Proxy" -c "https.proxy=$Proxy" push $Remote $Branch
}

if ($LASTEXITCODE -ne 0) {
    throw "Git push failed with exit code $LASTEXITCODE."
}

Write-Host "Push completed."
