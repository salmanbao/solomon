$ErrorActionPreference = "Stop"

go run ./scripts/check_boundaries.go

$lint = Get-Command golangci-lint -ErrorAction SilentlyContinue
if ($null -ne $lint) {
    golangci-lint run
    exit 0
}

$fallback = Join-Path (go env GOPATH) "bin/golangci-lint.exe"
if (-not (Test-Path $fallback)) {
    throw "golangci-lint not found in PATH or at $fallback"
}

& $fallback run
