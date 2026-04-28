# deploy.ps1
# Script to stop running homeocto-launcher, copy new binaries, and restart

$ErrorActionPreference = "Stop"

$TARGET_DIR = "C:\homeocto-0.0.1"
$BUILD_DIR = Join-Path $PSScriptRoot "..\build"
$LAUNCHER_EXE = "homeocto-launcher.exe"
$MAIN_EXE = "homeocto.exe"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  HomeOcto Deployment Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# Step 1: Check and stop running homeocto-launcher.exe
Write-Host "`n[1/4] Checking for running $LAUNCHER_EXE..." -ForegroundColor Yellow

# Try multiple methods to find and stop the process
$stopped = $false
$maxRetries = 3

for ($i = 1; $i -le $maxRetries; $i++) {
    # Try to find the process
    $launcherProcess = Get-Process -Name "homeocto-launcher" -ErrorAction SilentlyContinue
    
    if (-not $launcherProcess) {
        # Try with .exe extension
        $launcherProcess = Get-Process | Where-Object { $_.ProcessName -eq "homeocto-launcher" -or $_.MainWindowTitle -like "*homeocto*" } | Select-Object -First 1
    }
    
    if ($launcherProcess) {
        Write-Host "  Found running $LAUNCHER_EXE (PID: $($launcherProcess.Id), Attempt: $i)" -ForegroundColor Yellow
        Write-Host "  Attempting to stop process..." -ForegroundColor Yellow
        
        # Method 1: Try graceful stop first
        try {
            Stop-Process -Id $launcherProcess.Id -Force -ErrorAction Stop
            Write-Host "  Stop command sent, waiting for process to exit..." -ForegroundColor Yellow
            Start-Sleep -Seconds 2
            
            # Check if process is still running
            $stillRunning = Get-Process -Id $launcherProcess.Id -ErrorAction SilentlyContinue
            if (-not $stillRunning) {
                Write-Host "  Process stopped successfully" -ForegroundColor Green
                $stopped = $true
                break
            }
        } catch {
            Write-Host "  Stop-Process failed: $_" -ForegroundColor Red
        }
        
        # Method 2: Use taskkill as fallback
        if (-not $stopped) {
            Write-Host "  Using taskkill to force stop..." -ForegroundColor Yellow
            $taskkillResult = taskkill /F /PID $launcherProcess.Id 2>&1
            Write-Host "  $taskkillResult" -ForegroundColor Yellow
            Start-Sleep -Seconds 2
            
            $stillRunning = Get-Process -Id $launcherProcess.Id -ErrorAction SilentlyContinue
            if (-not $stillRunning) {
                Write-Host "  Process stopped via taskkill" -ForegroundColor Green
                $stopped = $true
                break
            }
        }
    } else {
        Write-Host "  No running $LAUNCHER_EXE found" -ForegroundColor Green
        $stopped = $true
        break
    }
}

if (-not $stopped) {
    Write-Host "  WARNING: Could not stop $LAUNCHER_EXE after $maxRetries attempts" -ForegroundColor Red
    Write-Host "  Please close it manually and try again" -ForegroundColor Red
    exit 1
}

# Also stop homeocto.exe if running
Write-Host "`n  Checking for $MAIN_EXE..." -ForegroundColor Yellow
$mainStopped = $false

for ($i = 1; $i -le $maxRetries; $i++) {
    $mainProcess = Get-Process -Name "homeocto" -ErrorAction SilentlyContinue
    
    if ($mainProcess) {
        Write-Host "  Found running $MAIN_EXE (PID: $($mainProcess.Id))" -ForegroundColor Yellow
        Write-Host "  Stopping $MAIN_EXE..." -ForegroundColor Yellow
        
        try {
            Stop-Process -Id $mainProcess.Id -Force -ErrorAction Stop
            Start-Sleep -Seconds 1
            
            $stillRunning = Get-Process -Id $mainProcess.Id -ErrorAction SilentlyContinue
            if (-not $stillRunning) {
                Write-Host "  $MAIN_EXE stopped successfully" -ForegroundColor Green
                $mainStopped = $true
                break
            }
        } catch {
            Write-Host "  Failed to stop $MAIN_EXE, using taskkill..." -ForegroundColor Yellow
            taskkill /F /PID $mainProcess.Id 2>&1 | Out-Null
            Start-Sleep -Seconds 1
            
            $stillRunning = Get-Process -Id $mainProcess.Id -ErrorAction SilentlyContinue
            if (-not $stillRunning) {
                Write-Host "  $MAIN_EXE stopped via taskkill" -ForegroundColor Green
                $mainStopped = $true
                break
            }
        }
    } else {
        Write-Host "  No running $MAIN_EXE found" -ForegroundColor Green
        $mainStopped = $true
        break
    }
}

# Step 2: Verify build directory exists
Write-Host "`n[2/4] Verifying build directory..." -ForegroundColor Yellow

$buildPath = Resolve-Path $BUILD_DIR -ErrorAction SilentlyContinue
if (-not $buildPath) {
    Write-Host "  ERROR: Build directory not found: $BUILD_DIR" -ForegroundColor Red
    Write-Host "  Please run build first!" -ForegroundColor Red
    exit 1
}

# Step 3: Copy executables to target directory
Write-Host "`n[3/4] Copying executables to $TARGET_DIR..." -ForegroundColor Yellow

if (-not (Test-Path $TARGET_DIR)) {
    Write-Host "  Target directory does not exist, creating..." -ForegroundColor Yellow
    New-Item -Path $TARGET_DIR -ItemType Directory -Force | Out-Null
}

$launcherSource = Join-Path $BUILD_DIR $LAUNCHER_EXE
$mainSource = Join-Path $BUILD_DIR $MAIN_EXE

if (-not (Test-Path $launcherSource)) {
    Write-Host "  ERROR: $LAUNCHER_EXE not found in build directory" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $mainSource)) {
    Write-Host "  ERROR: $MAIN_EXE not found in build directory" -ForegroundColor Red
    exit 1
}

# Copy files with overwrite
Copy-Item -Path $launcherSource -Destination $TARGET_DIR -Force
Write-Host "  Copied $LAUNCHER_EXE" -ForegroundColor Green

Copy-Item -Path $mainSource -Destination $TARGET_DIR -Force
Write-Host "  Copied $MAIN_EXE" -ForegroundColor Green

# Step 4: Start homeocto-launcher.exe
Write-Host "`n[4/4] Starting $LAUNCHER_EXE..." -ForegroundColor Yellow

$launcherPath = Join-Path $TARGET_DIR $LAUNCHER_EXE
Start-Process -FilePath $launcherPath -WorkingDirectory $TARGET_DIR

Start-Sleep -Seconds 2

# Verify it started
$newProcess = Get-Process -Name "homeocto-launcher" -ErrorAction SilentlyContinue
if ($newProcess) {
    Write-Host "  $LAUNCHER_EXE started successfully (PID: $($newProcess.Id))" -ForegroundColor Green
} else {
    Write-Host "  WARNING: Process may not have started. Check manually." -ForegroundColor Yellow
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  Deployment Complete!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Target: $TARGET_DIR" -ForegroundColor White
Write-Host "" -ForegroundColor White
