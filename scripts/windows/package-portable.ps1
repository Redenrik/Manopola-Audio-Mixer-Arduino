param(
    [string]$OutputDir = "dist\mama-portable",
    [ValidateSet("amd64", "386", "arm64")]
    [string]$Arch = "amd64",
    [string]$BinaryName = "mama.exe"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$mamaDir = Join-Path $repoRoot "mama"
$outDir = Join-Path $repoRoot $OutputDir

New-Item -ItemType Directory -Path $outDir -Force | Out-Null
Get-ChildItem -Path $outDir -Filter "ui*.log" -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue

Push-Location $mamaDir
try {
    $env:GOOS = "windows"
    $env:GOARCH = $Arch

    Write-Host "Building Windows portable binaries for GOARCH=$env:GOARCH..."
    $sysoPath = Join-Path $mamaDir "cmd\\mama\\mama_windows.syso"
    if ($Arch -eq "amd64" -or $Arch -eq "386") {
        Write-Host "Embedding app icon into executable..."
        $iconPath = Join-Path $mamaDir "assets\\icons\\mama-app.ico"
        go run github.com/akavel/rsrc@latest -arch $Arch -ico $iconPath -o $sysoPath
        if ($LASTEXITCODE -ne 0) {
            throw "rsrc icon embedding failed"
        }
    }

    Write-Host "Building $BinaryName..."
    go build -ldflags "-H=windowsgui" -o (Join-Path $outDir $BinaryName) ./cmd/mama
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed for GOARCH=$env:GOARCH"
    }
    Remove-Item -Path $sysoPath -ErrorAction SilentlyContinue
}
finally {
    Pop-Location
}

Copy-Item (Join-Path $mamaDir "internal\config\default.yaml") (Join-Path $outDir "config.yaml") -Force
Copy-Item (Join-Path $mamaDir "assets\icons\mama-app.ico") (Join-Path $outDir "mama-app.ico") -Force
Copy-Item (Join-Path $mamaDir "assets\icons\mama-tray.ico") (Join-Path $outDir "mama-tray.ico") -Force

$launcherBinary = $BinaryName

$setupLauncher = @"
@echo off
cd /d "%~dp0"
start "" "$launcherBinary"
"@

$runtimeLauncher = @"
@echo off
cd /d "%~dp0"
start "" "$launcherBinary" -open=false -start-hidden=true
"@

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
