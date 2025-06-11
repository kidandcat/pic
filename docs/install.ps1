# pic installation script for Windows
# This script installs pic on Windows systems

# Configuration
$RepoOwner = "kidandcat"
$RepoName = "pic"
$BinaryName = "pic"
$InstallDir = "$env:LOCALAPPDATA\Programs\pic"

# Error handling
$ErrorActionPreference = "Stop"

# Functions
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Add-ToPath {
    param([string]$Path)
    
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$Path*") {
        Write-ColorOutput "Adding $Path to user PATH..." "Blue"
        [Environment]::SetEnvironmentVariable(
            "Path",
            "$currentPath;$Path",
            "User"
        )
        $env:Path = "$env:Path;$Path"
        Write-ColorOutput "PATH updated successfully!" "Green"
    } else {
        Write-ColorOutput "$Path is already in PATH" "Yellow"
    }
}

function Test-Chrome {
    Write-ColorOutput "Checking for Chrome installation..." "Blue"
    
    $chromePaths = @(
        "$env:ProgramFiles\Google\Chrome\Application\chrome.exe",
        "$env:ProgramFiles (x86)\Google\Chrome\Application\chrome.exe",
        "$env:LocalAppData\Google\Chrome\Application\chrome.exe"
    )
    
    $chromeFound = $false
    foreach ($path in $chromePaths) {
        if (Test-Path $path) {
            $chromeFound = $true
            break
        }
    }
    
    if ($chromeFound) {
        Write-ColorOutput "Chrome found!" "Green"
    } else {
        Write-ColorOutput "Chrome not found. Please install Chrome to use pic." "Yellow"
        Write-ColorOutput "Visit https://www.google.com/chrome/ to download Chrome." "Blue"
    }
}

# Main installation process
try {
    Write-ColorOutput "`nInstalling pic..." "Blue"
    
    # Check if already installed
    $existingPic = Get-Command pic -ErrorAction SilentlyContinue
    if ($existingPic) {
        Write-ColorOutput "pic is already installed at $($existingPic.Source)" "Yellow"
        $response = Read-Host "Do you want to reinstall? (y/N)"
        if ($response -ne 'y' -and $response -ne 'Y') {
            Write-ColorOutput "Installation cancelled." "Blue"
            exit 0
        }
    }
    
    # Create install directory
    if (!(Test-Path $InstallDir)) {
        Write-ColorOutput "Creating installation directory..." "Blue"
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    # Detect architecture
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    $platform = "windows-$arch"
    Write-ColorOutput "Detected platform: $platform" "Blue"
    
    # Download binary
    $downloadUrl = "https://github.com/$RepoOwner/$RepoName/releases/latest/download/$BinaryName-$platform.exe"
    $outputPath = Join-Path $InstallDir "$BinaryName.exe"
    
    Write-ColorOutput "Downloading pic..." "Blue"
    try {
        # Use TLS 1.2
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        
        # Download with progress
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($downloadUrl, $outputPath)
        
        Write-ColorOutput "Download complete!" "Green"
    } catch {
        Write-ColorOutput "Failed to download pic: $_" "Red"
        exit 1
    }
    
    # Add to PATH
    Add-ToPath $InstallDir
    
    # Check Chrome
    Test-Chrome
    
    # Verify installation
    Write-ColorOutput "`nVerifying installation..." "Blue"
    
    # Refresh environment for current session
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "User") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "Machine")
    
    $picCmd = Get-Command pic -ErrorAction SilentlyContinue
    if ($picCmd) {
        Write-ColorOutput "pic installed successfully!" "Green"
        Write-ColorOutput "Location: $($picCmd.Source)" "Blue"
        
        # Try to get version
        try {
            $version = & pic --version 2>$null
            if ($version) {
                Write-ColorOutput "Version: $version" "Blue"
            }
        } catch {
            # Version command might not be available
        }
    } else {
        Write-ColorOutput "Installation completed, but pic is not yet available in the current session." "Yellow"
        Write-ColorOutput "Please restart your terminal or run the following command:" "Blue"
        Write-ColorOutput '  $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "User")' "White"
    }
    
    Write-ColorOutput "`nInstallation complete!" "Green"
    Write-ColorOutput "`nQuick start:" "Blue"
    Write-ColorOutput "  pic example.com                    # Take a screenshot" "White"
    Write-ColorOutput "  pic --pdf example.com              # Save as PDF" "White"
    Write-ColorOutput "  pic --help                         # Show all options" "White"
    Write-ColorOutput "`nNote: You may need to restart your terminal for the PATH changes to take effect." "Yellow"
    
} catch {
    Write-ColorOutput "Installation failed: $_" "Red"
    exit 1
}