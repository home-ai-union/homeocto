# copy_images.ps1
# Copy all images from web directory to target folder

# Set UTF-8 encoding for Chinese character support
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

# Set error handling
$ErrorActionPreference = "Stop"

# Set source and target directories (default: from imgbak to homeocto)
$SourceRoot = "G:\code\imgbak"
$TargetDir = "G:\code\homeocto"

# Check if reverse parameter is passed (from homeocto to imgbak)
$ReverseMode = $false
if ($args -contains "--reverse" -or $args -contains "-r") {
    $ReverseMode = $true
    $SourceRoot = "G:\code\homeocto"
    $TargetDir = "G:\code\imgbak"
}

Write-Host ("=" * 60)
Write-Host "HomeOcto Image Copy Tool" -ForegroundColor Cyan
Write-Host ("=" * 60)
Write-Host ""
if ($ReverseMode) {
    Write-Host "[Mode] Reverse Copy (homeocto -> imgbak)" -ForegroundColor Magenta
    Write-Host ("[Info] Source: " + $SourceRoot) -ForegroundColor Cyan
    Write-Host ("[Info] Target: " + $TargetDir) -ForegroundColor Cyan
} else {
    Write-Host "[Mode] Default Copy (imgbak -> homeocto)" -ForegroundColor Green
    Write-Host ("[Info] Source: " + $SourceRoot) -ForegroundColor Cyan
    Write-Host ("[Info] Target: " + $TargetDir) -ForegroundColor Cyan
}
Write-Host ""

# Save current directory
$OriginalLocation = Get-Location

# Check if Go is installed
$goVersion = go version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "[Error] Go environment not detected, please install Go first" -ForegroundColor Red
    exit 1
}
Write-Host ("[Info] Go Version: " + $goVersion) -ForegroundColor Green

# Change to script directory for compilation
Set-Location $PSScriptRoot

# Compile Go program
Write-Host ""
Write-Host "[Info] Compiling copy program..." -ForegroundColor Yellow
go build -o copy_images.exe copy_images.go

if ($LASTEXITCODE -ne 0) {
    Write-Host "[Error] Compilation failed" -ForegroundColor Red
    Set-Location $OriginalLocation
    exit 1
}

Write-Host "[Success] Compilation complete" -ForegroundColor Green
Write-Host ""

# Run copy program
Write-Host "[Info] Starting image copy..." -ForegroundColor Yellow
Write-Host ""
if ($ReverseMode) {
    .\copy_images.exe --reverse
} else {
    .\copy_images.exe
}

$exitCode = $LASTEXITCODE

# Clean up compiled file
if (Test-Path "copy_images.exe") {
    Remove-Item "copy_images.exe" -Force
}

# Restore original directory
Set-Location $OriginalLocation

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host ("=" * 60)
    Write-Host "Copy Complete!" -ForegroundColor Green
    Write-Host ("Target: " + $TargetDir) -ForegroundColor Green
    Write-Host ("=" * 60)
} else {
    Write-Host ("=" * 60)
    Write-Host "Errors occurred during copy" -ForegroundColor Red
    Write-Host ("=" * 60)
}

exit $exitCode
