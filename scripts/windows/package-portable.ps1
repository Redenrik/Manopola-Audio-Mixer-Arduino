param(
    [string]$OutputDir = "dist\mama-portable"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$mamaDir = Join-Path $repoRoot "mama"
$outDir = Join-Path $repoRoot $OutputDir

New-Item -ItemType Directory -Path $outDir -Force | Out-Null
Remove-Item -Path (Join-Path $outDir "mama-ui.exe") -ErrorAction SilentlyContinue
Get-ChildItem -Path $outDir -Filter "ui*.log" -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue

Push-Location $mamaDir
try {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"

    Write-Host "Building Windows portable binaries for GOARCH=$env:GOARCH..."
    Write-Host "Building mama.exe..."
    go build -o (Join-Path $outDir "mama.exe") ./cmd/mama
}
finally {
    Pop-Location
}

Copy-Item (Join-Path $mamaDir "internal\config\default.yaml") (Join-Path $outDir "config.yaml") -Force

$setupLauncher = @'
@echo off
cd /d "%~dp0"
start "" "mama.exe"
'@

$runtimeLauncher = @'
@echo off
cd /d "%~dp0"
start "" "mama.exe" -open=false
'@

$notes = @'
MAMA Portable Package

1) Run "Open Setup UI.cmd" (or "Start MAMA.cmd") to launch MAMA.
2) The mixer engine and setup UI run together in one process.
3) Save config at any time; changes apply immediately without restart.
4) "Start Mixer.cmd" keeps running in background mode (no auto-open browser).

All settings stay in this folder (`config.yaml`).
'@

Set-Content -Path (Join-Path $outDir "Open Setup UI.cmd") -Value $setupLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "Start Mixer.cmd") -Value $runtimeLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "Start MAMA.cmd") -Value $setupLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "README-PORTABLE.txt") -Value $notes -Encoding ASCII

Write-Host ""
Write-Host "Portable package ready at: $outDir"
