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
    Write-Host "Embedding app icon into executable..."
    $iconPath = Join-Path $mamaDir "assets\\icons\\mama-app.ico"
    $sysoPath = Join-Path $mamaDir "cmd\\mama\\mama_windows.syso"
    go run github.com/akavel/rsrc@latest -ico $iconPath -o $sysoPath

    Write-Host "Building mama.exe..."
    go build -ldflags "-H=windowsgui" -o (Join-Path $outDir "mama.exe") ./cmd/mama
    Remove-Item -Path $sysoPath -ErrorAction SilentlyContinue
}
finally {
    Pop-Location
}

Copy-Item (Join-Path $mamaDir "internal\config\default.yaml") (Join-Path $outDir "config.yaml") -Force
Copy-Item (Join-Path $mamaDir "assets\icons\mama-app.ico") (Join-Path $outDir "mama-app.ico") -Force
Copy-Item (Join-Path $mamaDir "assets\icons\mama-tray.ico") (Join-Path $outDir "mama-tray.ico") -Force

$setupLauncher = @'
@echo off
cd /d "%~dp0"
start "" "mama.exe"
'@

$runtimeLauncher = @'
@echo off
cd /d "%~dp0"
start "" "mama.exe" -open=false -start-hidden=true
'@

$notes = @'
MAMA Portable Package

1) Run "Open Setup UI.cmd" (or "Start MAMA.cmd") to launch MAMA.
2) The mixer engine and setup UI run together in one process.
3) Save config at any time; changes apply immediately without restart.
4) "Start Mixer.cmd" runs hidden in tray (no auto-open browser).

All settings stay in this folder (`config.yaml`).
'@

Set-Content -Path (Join-Path $outDir "Open Setup UI.cmd") -Value $setupLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "Start Mixer.cmd") -Value $runtimeLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "Start MAMA.cmd") -Value $setupLauncher -Encoding ASCII
Set-Content -Path (Join-Path $outDir "README-PORTABLE.txt") -Value $notes -Encoding ASCII

Write-Host ""
Write-Host "Portable package ready at: $outDir"
