# Fix UTF-8 encoding issues in TypeScript/JSX files
# This script re-encodes frontend files to UTF-8 without BOM

$files = @(
    "web\frontend\src\components\chat\assistant-message.tsx",
    "web\frontend\src\homeocto\api\device-command-executor.ts",
    "web\frontend\src\homeocto\components\device-control-page.tsx",
    "web\frontend\src\homeocto\components\device-control-panel.tsx",
    "web\frontend\src\homeocto\hooks\use-smart-home-websocket.ts"
)

foreach ($file in $files) {
    $fullPath = Join-Path $PSScriptRoot "..\$file"
    if (Test-Path $fullPath) {
        Write-Host "Processing: $file"
        try {
            # Read file content with UTF-8 encoding
            $content = Get-Content -Path $fullPath -Raw -Encoding UTF8
            # Write back with UTF-8 encoding without BOM
            $utf8NoBom = New-Object System.Text.UTF8Encoding $false
            [System.IO.File]::WriteAllText($fullPath, $content, $utf8NoBom)
            Write-Host "  ✓ Fixed: $file" -ForegroundColor Green
        }
        catch {
            Write-Host "  ✗ Error processing $file : $_" -ForegroundColor Red
        }
    }
    else {
        Write-Host "  ! File not found: $file" -ForegroundColor Yellow
    }
}

Write-Host "`nAll files processed!" -ForegroundColor Cyan
