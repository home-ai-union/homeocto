# Fix UTF-8 encoding issues in Go files
# This script re-encodes Go files to UTF-8 without BOM

$files = @(
    "pkg\homeocto\data\types.go",
    "pkg\homeocto\intent\classifier.go",
    "pkg\homeocto\intent\device_control.go",
    "pkg\homeocto\intent\device_mgmt.go",
    "pkg\homeocto\intent\space_mgmt.go",
    "pkg\homeocto\ioc\factory.go",
    "pkg\homeocto\third\client.go",
    "pkg\homeocto\third\ioc\third_factory.go",
    "pkg\homeocto\third\miio\mi_client.go",
    "pkg\homeocto\third\miio\mi_device.go",
    "pkg\homeocto\third\tuya\tuya_client_test.go",
    "pkg\homeocto\tool\cli_tool.go",
    "pkg\homeocto\tool\common_tool.go",
    "pkg\homeocto\tool\llm_tool.go",
    "pkg\homeocto\tool\video_tool.go",
    "pkg\homeocto\tool\workflow_tool.go"
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
