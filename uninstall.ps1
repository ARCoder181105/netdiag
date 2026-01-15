# netdiag Uninstallation Script for Windows
# Run with: irm https://raw.githubusercontent.com/ARCoder181105/netdiag/main/uninstall.ps1 | iex

$ErrorActionPreference = "Stop"
$BinaryName = "netdiag.exe"

function Write-Success { param([string]$Message) Write-Host "✓ $Message" -ForegroundColor Green }
function Write-Info { param([string]$Message) Write-Host "→ $Message" -ForegroundColor Cyan }
function Write-Warning { param([string]$Message) Write-Host "! $Message" -ForegroundColor Yellow }

Write-Host "🗑️  netdiag Uninstaller" -ForegroundColor Green

# 1. Check Administrator Location (System32)
$adminPath = Join-Path "C:\Windows\System32" $BinaryName
if (Test-Path $adminPath) {
    Write-Info "Found in System32. Attempting to remove..."
    try {
        Remove-Item -Path $adminPath -Force -ErrorAction Stop
        Write-Success "Removed from System32"
    } catch {
        Write-Warning "Failed to remove from System32. Please run PowerShell as Administrator."
    }
}

# 2. Check User Location (~/bin)
$userBinDir = Join-Path $env:USERPROFILE "bin"
$userPath = Join-Path $userBinDir $BinaryName

if (Test-Path $userPath) {
    Write-Info "Found in User bin directory. Removing..."
    Remove-Item -Path $userPath -Force
    Write-Success "Removed from $userPath"
}

# 3. Check current PATH (cleanup verification)
if (Get-Command netdiag -ErrorAction SilentlyContinue) {
    $remainingPath = (Get-Command netdiag).Source
    Write-Warning "netdiag is still found at: $remainingPath"
    Write-Warning "You may need to delete this file manually."
} else {
    Write-Success "netdiag successfully uninstalled."
}