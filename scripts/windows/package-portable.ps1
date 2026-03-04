param(
    [string]$OutputDir = "dist\mama-portable"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$mamaDir = Join-Path $repoRoot "mama"
$outDir = Join-Path $repoRoot $OutputDir

New-Item -ItemType Directory -Path $outDir -Force | Out-Null

Push-Location $mamaDir
try {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"

    Write-Host "Building Windows portable binaries for GOARCH=$env:GOARCH..."
    Write-Host "Building mama.exe..."
    go build -o (Join-Path $outDir "mama.exe") ./cmd/mama

    Write-Host "Building mama-ui.exe..."
    go build -o (Join-Path $outDir "mama-ui.exe") ./cmd/mama-ui
}
finally {
    Pop-Location
}

Copy-Item (Join-Path $mamaDir "internal\config\default.yaml") (Join-Path $outDir "config.yaml") -Force

$setupLauncher = @'
@echo off
cd /d "%~dp0"
start "" "mama-ui.exe"
'@

$runtimeLauncher = @'
@echo off
cd /d "%~dp0"
start "" "mama.exe"
'@

$notes = @'
MAMA Portable Package

1) Run "Open Setup UI.cmd" to configure port and mappings.
2) Save config.
3) Run "Start Mixer.cmd".

All settings stay in this folder (`config.yaml`).
'@

Set-Content -Path (Join-Path $outDir "Open Setup UI.cmd") -Value $setupLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "Start Mixer.cmd") -Value $runtimeLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "README-PORTABLE.txt") -Value $notes -Encoding ASCII

Write-Host ""
Write-Host "Portable package ready at: $outDir"
