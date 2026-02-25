param(
    [Parameter(Mandatory = $true)]
    [string]$Context,
    [Parameter(Mandatory = $true)]
    [string]$Service
)

$ErrorActionPreference = "Stop"

$modulePath = Join-Path "contexts" (Join-Path $Context $Service)
if (Test-Path $modulePath) {
    throw "Module already exists: $modulePath"
}

$dirs = @(
    "domain",
    "application",
    "ports",
    "adapters",
    "transport"
)

foreach ($dir in $dirs) {
    New-Item -ItemType Directory -Path (Join-Path $modulePath $dir) -Force | Out-Null
}

$title = ($Service.Split("-") | ForEach-Object {
    if ($_.Length -eq 0) { return $_ }
    $_.Substring(0, 1).ToUpper() + $_.Substring(1)
}) -join " "

$readme = @"
# $title

Module scaffold for Solomon monolith.

## Structure
- domain/: entities, value objects, domain services, invariants
- application/: use cases, command/query handlers, orchestration
- ports/: repository, event, and client interfaces
- adapters/: DB, HTTP/gRPC, event bus, cache implementations
- transport/: module-private transport DTOs and event payload mappers
"@

[System.IO.File]::WriteAllText((Join-Path $modulePath "README.md"), $readme)
Write-Host "Created module scaffold at $modulePath"
