param(
    [string]$PortableDir = "dist\mama-portable",
    [string]$OutputDir = "dist\installer",
    [string]$AppVersion = "0.1.0"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$portablePath = Join-Path $repoRoot $PortableDir
$outDir = Join-Path $repoRoot $OutputDir
$issPath = Join-Path $PSScriptRoot "mama-installer.iss"

if (-not (Test-Path $portablePath)) {
    throw "Portable directory not found: $portablePath. Run package-portable.ps1 first."
}

$iscc = Get-Command "iscc" -ErrorAction SilentlyContinue
if (-not $iscc) {
    Write-Warning "Inno Setup compiler (iscc) not found. Skipping installer packaging."
    exit 0
}

New-Item -ItemType Directory -Path $outDir -Force | Out-Null

$installerPath = Join-Path $outDir "MAMA-Setup-$AppVersion.exe"

& $iscc.Source \
    "/DMyAppVersion=$AppVersion" \
    "/DSourceDir=$portablePath" \
    "/DOutputDir=$outDir" \
    "/DOutputBaseFilename=MAMA-Setup-$AppVersion" \
    $issPath

if (-not (Test-Path $installerPath)) {
    throw "Installer generation failed: $installerPath was not created."
}

Write-Host "Installer package ready at: $installerPath"
