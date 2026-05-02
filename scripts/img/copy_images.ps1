# copy_images.ps1
# Copy all images from web directory to target folder

# Set UTF-8 encoding for Chinese character support
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

# Set error handling
$ErrorActionPreference = "Stop"

# Get script directory
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Define default paths
$DefaultSourceRoot = "G:\code\homeocto\doc\images"
$DefaultTargetDir = "G:\code\homeocto"

# Parse command line arguments
$SourceRoot = $null
$TargetDir = $null
$ReverseMode = $false

for ($i = 0; $i -lt $args.Count; $i++) {
    switch ($args[$i]) {
        "--reverse" { $ReverseMode = $true }
        "-r" { $ReverseMode = $true }
        "-source" {
            if ($i + 1 -lt $args.Count) {
                $i++
                $SourceRoot = $args[$i]
            }
        }
        "-target" {
            if ($i + 1 -lt $args.Count) {
                $i++
                $TargetDir = $args[$i]
            }
        }
    }
}

# Use defaults if not specified
if (-not $SourceRoot) { $SourceRoot = $DefaultSourceRoot }
if (-not $TargetDir) { $TargetDir = $DefaultTargetDir }

# Swap source and target if reverse mode
if ($ReverseMode) {
    $temp = $SourceRoot
    $SourceRoot = $TargetDir
    $TargetDir = $temp
}

Write-Host ("=" * 60)
Write-Host "HomeOcto Image Copy Tool" -ForegroundColor Cyan
Write-Host ("=" * 60)
Write-Host ""
if ($ReverseMode) {
    Write-Host "[Mode] Reverse Copy (homeocto -> imgbak)" -ForegroundColor Magenta
} else {
    Write-Host "[Mode] Default Copy (imgbak -> homeocto)" -ForegroundColor Green
}
Write-Host ("[Info] Source: " + $SourceRoot) -ForegroundColor Cyan
Write-Host ("[Info] Target: " + $TargetDir) -ForegroundColor Cyan
Write-Host ""

# Check if source directory exists
if (-not (Test-Path $SourceRoot)) {
    Write-Host "[Error] Source directory not found: $SourceRoot" -ForegroundColor Red
    exit 1
}

# Check if target directory exists
if (-not (Test-Path $TargetDir)) {
    Write-Host "[Error] Target directory not found: $TargetDir" -ForegroundColor Red
    exit 1
}

# Confirm before proceeding
Write-Host "WARNING: This operation will copy images!" -ForegroundColor Yellow
$confirm = Read-Host "Continue? (y/N)"

if ($confirm -ne "y" -and $confirm -ne "Y") {
    Write-Host "Operation cancelled" -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "Starting copy..." -ForegroundColor Cyan
Write-Host ""

# Save current directory
$OriginalDir = Get-Location

# Change to script directory to use the correct go.mod
Set-Location $ScriptDir

# Run Go copy script
$goScript = Join-Path $ScriptDir "copy_images.go"

Write-Host "Working directory: $ScriptDir" -ForegroundColor Gray
if ($ReverseMode) {
    Write-Host "Executing: go run $goScript -source $SourceRoot -target $TargetDir --reverse" -ForegroundColor Gray
    go run $goScript -source $SourceRoot -target $TargetDir --reverse
} else {
    Write-Host "Executing: go run $goScript -source $SourceRoot -target $TargetDir" -ForegroundColor Gray
    go run $goScript -source $SourceRoot -target $TargetDir
}

$exitCode = $LASTEXITCODE

# Restore original directory
Set-Location $OriginalDir

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host ("=" * 60) -ForegroundColor Green
    Write-Host "Copy Complete!" -ForegroundColor Green
    Write-Host ("Target: " + $TargetDir) -ForegroundColor Green
    Write-Host ("=" * 60) -ForegroundColor Green
} else {
    Write-Host ("=" * 60) -ForegroundColor Red
    Write-Host "Errors occurred during copy" -ForegroundColor Red
    Write-Host ("=" * 60) -ForegroundColor Red
}

exit $exitCode
