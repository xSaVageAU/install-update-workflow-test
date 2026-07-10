# One-command installer for Windows:
#   iwr https://raw.githubusercontent.com/xSaVageAU/install-update-workflow-test/main/scripts/install.ps1 -useb | iex
[CmdletBinding()]
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\iuw"
)

$ErrorActionPreference = "Stop"

$Repo = "xSaVageAU/install-update-workflow-test"
$Binary = "iuw"

function Write-Info    { param([string]$Message) Write-Host "==> $Message" -ForegroundColor Cyan }
function Write-Success { param([string]$Message) Write-Host "==> $Message" -ForegroundColor Green }
function Write-Note    { param([string]$Message) Write-Host $Message -ForegroundColor Yellow }

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "unsupported architecture: 32-bit Windows is not supported"
}
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

Write-Info "Fetching latest release info for $Repo..."
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$tag = $release.tag_name
if (-not $tag) {
    throw "could not determine latest release tag (check that $Repo has a published release)"
}

$assetName = "${Binary}_windows_${Arch}.exe"
$downloadUrl = "https://github.com/$Repo/releases/download/$tag/$assetName"
$checksumsUrl = "https://github.com/$Repo/releases/download/$tag/checksums.txt"

Write-Info "Installing $Binary $tag (windows/$Arch)..."

$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmpDir | Out-Null

try {
    $tmpBinary = Join-Path $tmpDir $assetName
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tmpBinary

    $checksumsPath = Join-Path $tmpDir "checksums.txt"
    $haveChecksums = $false
    try {
        Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath
        $haveChecksums = $true
    } catch {
        Write-Warning "Could not fetch checksums.txt: $_"
    }

    if ($haveChecksums) {
        $pattern = [regex]::Escape($assetName) + '$'
        $line = Get-Content $checksumsPath | Where-Object { $_ -match $pattern }
        if ($line) {
            $expected = ($line -split '\s+')[0].ToLower()
            $actual = (Get-FileHash -Algorithm SHA256 -Path $tmpBinary).Hash.ToLower()
            if ($actual -ne $expected) {
                throw "checksum mismatch for $assetName (expected $expected, got $actual)"
            }
            Write-Success "Checksum verified."
        }
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $destPath = Join-Path $InstallDir "$Binary.exe"
    Move-Item -Force $tmpBinary $destPath

    Write-Success "Installed to $destPath"

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @()
    if ($userPath) { $pathEntries = $userPath -split ';' }
    if ($pathEntries -notcontains $InstallDir) {
        $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Success "Added $InstallDir to your user PATH. Open a new terminal for it to take effect."
    }

    Write-Host ""
    Write-Info "Run '$Binary --version' to verify."
}
finally {
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
