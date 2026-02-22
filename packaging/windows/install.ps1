# M365Kit Windows Installer (no admin required)
# Usage: iwr https://raw.githubusercontent.com/monykiss/m365kit/main/packaging/windows/install.ps1 | iex

$ErrorActionPreference = "Stop"
$version = "v0.4.0"
$installDir = "$env:LOCALAPPDATA\M365Kit"
$downloadUrl = "https://github.com/monykiss/m365kit/releases/download/$version/m365kit_Windows_amd64.zip"

Write-Host ""
Write-Host "  M365Kit Installer" -ForegroundColor Cyan
Write-Host "  Version: $version" -ForegroundColor Gray
Write-Host ""

# Create install directory
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    Write-Host "  Created $installDir" -ForegroundColor Gray
}

# Download
Write-Host "  Downloading M365Kit $version..." -ForegroundColor Yellow
$zipPath = "$env:TEMP\m365kit.zip"
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
} catch {
    Write-Host "  Download failed: $_" -ForegroundColor Red
    Write-Host "  Please download manually from:" -ForegroundColor Yellow
    Write-Host "  https://github.com/monykiss/m365kit/releases/latest" -ForegroundColor White
    exit 1
}

# Extract
Write-Host "  Extracting..." -ForegroundColor Yellow
try {
    Expand-Archive -Path $zipPath -DestinationPath $installDir -Force
} catch {
    Write-Host "  Extraction failed: $_" -ForegroundColor Red
    exit 1
}

# Clean up zip
Remove-Item $zipPath -ErrorAction SilentlyContinue

# Ensure kit.exe exists
$kitExe = Join-Path $installDir "kit.exe"
if (-not (Test-Path $kitExe)) {
    # Check if it was extracted into a subdirectory
    $found = Get-ChildItem -Path $installDir -Recurse -Filter "kit.exe" | Select-Object -First 1
    if ($found) {
        Move-Item $found.FullName $kitExe -Force
    } else {
        Write-Host "  kit.exe not found in archive" -ForegroundColor Red
        exit 1
    }
}

# Add to user PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
    Write-Host "  Added $installDir to PATH" -ForegroundColor Green
}

# Verify
Write-Host ""
Write-Host "  M365Kit installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "  Location: $installDir" -ForegroundColor Gray
Write-Host ""
Write-Host "  Restart your terminal, then run:" -ForegroundColor Yellow
Write-Host "    kit --help" -ForegroundColor White
Write-Host "    kit config init" -ForegroundColor White
Write-Host ""
