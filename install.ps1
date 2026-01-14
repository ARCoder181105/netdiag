# netdiag Installation Script for Windows
# Run with: irm https://raw.githubusercontent.com/ARCoder181105/netdiag/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

# Configuration
$Repo = "ARCoder181105/netdiag"
$BinaryName = "netdiag.exe"
$Version = $env:NETDIAG_VERSION
if (-not $Version) {
    $Version = "latest"
}

# Color functions
function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Failure {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Info {
    param([string]$Message)
    Write-Host "→ $Message" -ForegroundColor Cyan
}

function Write-Warning {
    param([string]$Message)
    Write-Host "! $Message" -ForegroundColor Yellow
}

# Check if running as Administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Get latest version from GitHub
function Get-LatestVersion {
    Write-Info "Fetching latest version from GitHub..."
    
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -ErrorAction Stop
        $latestVersion = $release.tag_name
        Write-Success "Latest version: $latestVersion"
        return $latestVersion
    } catch {
        Write-Warning "Could not fetch latest release. Will attempt to build from source."
        return $null
    }
}

# Download pre-built binary
function Download-Binary {
    param([string]$Version)
    
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/netdiag-windows-amd64.exe"
    $tempFile = Join-Path $env:TEMP "netdiag.exe"
    
    Write-Info "Downloading netdiag $Version..."
    
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -ErrorAction Stop
        Write-Success "Downloaded binary"
        return $tempFile
    } catch {
        Write-Warning "Failed to download binary: $_"
        return $null
    }
}

# Build from source
function Build-FromSource {
    Write-Info "Building from source..."
    
    # Check if Go is installed
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Failure "Go is not installed. Please install Go 1.25 or higher from https://go.dev/dl/"
        exit 1
    }
    
    $goVersion = (go version) -replace 'go version go', '' -replace ' .*', ''
    Write-Info "Using Go version: $goVersion"
    
    # Check if Git is installed
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Failure "Git is not installed. Please install Git from https://git-scm.com/"
        exit 1
    }
    
    # Clone repository
    $tempDir = Join-Path $env:TEMP ("netdiag-build-" + [guid]::NewGuid().ToString())
    Write-Info "Cloning repository to $tempDir..."
    
    try {
        git clone "https://github.com/$Repo.git" $tempDir 2>&1 | Out-Null
        Push-Location $tempDir
        
        # Build binary
        Write-Info "Building binary..."
        go build -ldflags="-s -w" -o netdiag.exe
        
        $binaryPath = Join-Path $tempDir "netdiag.exe"
        if (Test-Path $binaryPath) {
            Write-Success "Build successful"
            Pop-Location
            return $binaryPath
        } else {
            throw "Build failed - binary not created"
        }
    } catch {
        Write-Failure "Build failed: $_"
        if (Test-Path $tempDir) {
            Remove-Item -Recurse -Force $tempDir
        }
        Pop-Location
        exit 1
    }
}

# Install binary
function Install-Binary {
    param([string]$BinaryPath)
    
    $isAdmin = Test-Administrator
    
    if ($isAdmin) {
        # Install to System32 if running as admin
        $installDir = "C:\Windows\System32"
        $targetPath = Join-Path $installDir $BinaryName
        
        Write-Info "Installing to $installDir (Administrator mode)..."
        Copy-Item -Path $BinaryPath -Destination $targetPath -Force
        Write-Success "Installed to $targetPath"
    } else {
        # Install to user bin directory
        $installDir = Join-Path $env:USERPROFILE "bin"
        
        # Create directory if it doesn't exist
        if (-not (Test-Path $installDir)) {
            New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        }
        
        $targetPath = Join-Path $installDir $BinaryName
        
        Write-Info "Installing to $installDir (User mode)..."
        Copy-Item -Path $BinaryPath -Destination $targetPath -Force
        Write-Success "Installed to $targetPath"
        
        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$installDir*") {
            Write-Info "Adding $installDir to PATH..."
            [Environment]::SetEnvironmentVariable(
                "Path",
                "$userPath;$installDir",
                "User"
            )
            # Update PATH in current session
            $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
            Write-Success "Added to PATH"
        }
    }
    
    return $targetPath
}

# Verify installation
function Test-Installation {
    Write-Info "Verifying installation..."
    
    # Refresh PATH in current session
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    
    try {
        $version = & netdiag --version 2>&1
        Write-Success "netdiag installed successfully!"
        Write-Host ""
        Write-Host "Version: " -NoNewline -ForegroundColor Cyan
        Write-Host $version
        Write-Host "Location: " -NoNewline -ForegroundColor Cyan
        Write-Host (Get-Command netdiag).Source
        Write-Host ""
        Write-Host "🚀 Quick Start:" -ForegroundColor Green
        Write-Host "  netdiag ping google.com"
        Write-Host "  netdiag speedtest"
        Write-Host "  netdiag scan localhost -p 1-1000"
        Write-Host ""
        Write-Host "Run 'netdiag --help' for more information"
        
        if (-not (Test-Administrator)) {
            Write-Host ""
            Write-Warning "For ICMP operations (ping, trace, discover), run PowerShell as Administrator"
        }
    } catch {
        Write-Failure "Installation verification failed"
        Write-Info "Please restart your terminal and try again"
        Write-Info "If the issue persists, make sure the installation directory is in your PATH"
        exit 1
    }
}

# Main installation flow
function Main {
    Write-Host "🚀 netdiag Installation Script" -ForegroundColor Green
    Write-Host ""
    
    $isAdmin = Test-Administrator
    if ($isAdmin) {
        Write-Info "Running as Administrator"
    } else {
        Write-Info "Running as User (not Administrator)"
    }
    
    # Determine version
    if ($Version -eq "latest") {
        $latestVersion = Get-LatestVersion
        if ($latestVersion) {
            $Version = $latestVersion
        }
    }
    
    # Try to download pre-built binary first
    $binaryPath = $null
    if ($Version -ne "latest") {
        $binaryPath = Download-Binary -Version $Version
    }
    
    # Fall back to building from source if download failed
    if (-not $binaryPath) {
        $binaryPath = Build-FromSource
    }
    
    # Install binary
    $installedPath = Install-Binary -BinaryPath $binaryPath
    
    # Verify installation
    Test-Installation
}

# Run main function
Main
