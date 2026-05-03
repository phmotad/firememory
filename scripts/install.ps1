# Installs fmem.exe and fquery.exe from the latest FireMemory GitHub Release.
#
#   irm https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install.ps1 | iex
#   $env:INSTALL_DIR = "$env:LOCALAPPDATA\firememory\bin"; .\install.ps1
[CmdletBinding()]
param(
    [string]$InstallDir = $env:INSTALL_DIR,
    [string]$Version    = $env:VERSION
)

$ErrorActionPreference = "Stop"
$Repo = "phmotad/firememory"

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "firememory\bin"
}

# ── Detect platform ──────────────────────────────────────────────────────────

$Arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
$GoArch = switch ($Arch) {
    "X64"   { "amd64" }
    "Arm64" { "arm64" }
    default { Write-Error "Unsupported architecture: $Arch"; exit 1 }
}

# ── Resolve latest version ───────────────────────────────────────────────────

if (-not $Version) {
    $Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name -replace '^v', ''
}

Write-Host "Installing FireMemory v$Version (windows/$GoArch)..."

# ── Download and verify ──────────────────────────────────────────────────────

$Archive  = "firememory_${Version}_windows_${GoArch}.zip"
$BaseUrl  = "https://github.com/$Repo/releases/download/v$Version"
$Tmp      = Join-Path $env:TEMP "firememory-install"

New-Item -ItemType Directory -Force -Path $Tmp | Out-Null

$ZipPath  = Join-Path $Tmp $Archive
$SumsPath = Join-Path $Tmp "SHA256SUMS"

Write-Host "  Downloading $Archive..."
Invoke-WebRequest "$BaseUrl/$Archive"   -OutFile $ZipPath  -UseBasicParsing
Invoke-WebRequest "$BaseUrl/SHA256SUMS" -OutFile $SumsPath -UseBasicParsing

# Verify checksum.
$Expected = (Get-Content $SumsPath | Where-Object { $_ -match $Archive }) -split '\s+' | Select-Object -First 1
$Actual   = (Get-FileHash $ZipPath -Algorithm SHA256).Hash.ToLower()
if ($Actual -ne $Expected) {
    Write-Error "Checksum mismatch for $Archive`n  expected: $Expected`n  actual:   $Actual"
    exit 1
}
Write-Host "  checksum OK"

Expand-Archive -Path $ZipPath -DestinationPath $Tmp -Force

# ── Install binaries ─────────────────────────────────────────────────────────

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$Src = Join-Path $Tmp "firememory_${Version}_windows_${GoArch}"
# GoReleaser archives have the binary at the root of the archive.
Copy-Item (Join-Path $Tmp "fmem.exe")       (Join-Path $InstallDir "fmem.exe")   -Force
Copy-Item (Join-Path $Tmp "fquery.exe")     (Join-Path $InstallDir "fquery.exe") -Force
if (Test-Path (Join-Path $Tmp "onnxruntime.dll")) {
    Copy-Item (Join-Path $Tmp "onnxruntime.dll") (Join-Path $InstallDir "onnxruntime.dll") -Force
}

# ── Add to user PATH if not present ─────────────────────────────────────────

$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$InstallDir;$UserPath", "User")
    Write-Host "  Added $InstallDir to user PATH (restart terminal to apply)"
}

Remove-Item $Tmp -Recurse -Force

# ── Done ─────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "Installed:"
Write-Host "  $InstallDir\fmem.exe"
Write-Host "  $InstallDir\fquery.exe"
Write-Host ""
Write-Host "Next steps:"
Write-Host "  fmem init $env:USERPROFILE\my.fbrain    # create a brainfile"
Write-Host "  fquery mcp                               # start MCP server (downloads models once)"
