# Picoclaw to Homeocto Migration Script
# Uses Go to handle file replacement to avoid Chinese character encoding issues

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Picoclaw -> Homeocto Migration Tool" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Define paths
$PicoclawRoot = "G:\code\picoclaw"
$HomeoctoRoot = "G:\code\homeocto"

# Check if source directory exists
if (-not (Test-Path $PicoclawRoot)) {
    Write-Host "Error: Source directory not found: $PicoclawRoot" -ForegroundColor Red
    exit 1
}

# Check if target directory exists
if (-not (Test-Path $HomeoctoRoot)) {
    Write-Host "Error: Target directory not found: $HomeoctoRoot" -ForegroundColor Red
    exit 1
}

Write-Host "Source (picoclaw): $PicoclawRoot" -ForegroundColor Green
Write-Host "Target (homeocto): $HomeoctoRoot" -ForegroundColor Green
Write-Host ""

# Confirm before proceeding
Write-Host "WARNING: This operation will overwrite files in the target directory!" -ForegroundColor Yellow
$confirm = Read-Host "Continue? (y/N)"

if ($confirm -ne "y" -and $confirm -ne "Y") {
    Write-Host "Operation cancelled" -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "Starting migration..." -ForegroundColor Cyan
Write-Host ""

# Get script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Run Go migration script
$goScript = Join-Path $ScriptDir "migrate-picoclaw.go"

Write-Host "Executing: go run $goScript $PicoclawRoot $HomeoctoRoot" -ForegroundColor Gray
Write-Host ""

go run $goScript $PicoclawRoot $HomeoctoRoot

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "  Migration Completed Successfully!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Cyan
    Write-Host "1. Build Go code: go build ./cmd/homeocto" -ForegroundColor White
    Write-Host "2. Check frontend: cd web/frontend; npm install; npm run dev" -ForegroundColor White
    Write-Host "3. Search for remaining keywords: Select-String -Path '**/*.go' -Pattern 'picoclaw' -CaseSensitive" -ForegroundColor White
    Write-Host "4. Verify Chinese comments display correctly" -ForegroundColor White
} else {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "  Migration Failed!" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    exit $LASTEXITCODE
}
