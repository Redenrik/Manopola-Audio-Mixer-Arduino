$ErrorActionPreference = "Stop"

$rootDir = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$mamaDir = Join-Path $rootDir "mama"
$artifactRoot = Join-Path $rootDir "artifacts\readiness"
$runId = (Get-Date).ToUniversalTime().ToString("yyyyMMddTHHmmssZ")
$runDir = Join-Path $artifactRoot $runId
$summaryFile = Join-Path $runDir "summary.txt"

New-Item -ItemType Directory -Path $runDir -Force | Out-Null

@(
  "run_id=$runId"
  "start_utc=$((Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"))"
  "root_dir=$rootDir"
) | Set-Content -Path $summaryFile -Encoding ascii

function Add-SummaryLine {
  param([string]$Line)
  Add-Content -Path $summaryFile -Value $Line -Encoding ascii
}

function Invoke-ReadinessCheck {
  param(
    [string]$Name,
    [scriptblock]$Action
  )

  $logFile = Join-Path $runDir "$Name.log"
  Add-SummaryLine "check=$Name status=RUNNING"

  try {
    & $Action *> $logFile
    Add-SummaryLine "check=$Name status=PASS log=$([System.IO.Path]::GetFileName($logFile))"
  } catch {
    $_ | Out-String | Add-Content -Path $logFile -Encoding ascii
    Add-SummaryLine "check=$Name status=FAIL log=$([System.IO.Path]::GetFileName($logFile))"
    Add-SummaryLine "end_utc=$((Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"))"
    Add-SummaryLine "result=FAIL"
    throw
  }
}

function Invoke-GoInMama {
  param([scriptblock]$Action)
  Push-Location $mamaDir
  try {
    & $Action
  } finally {
    Pop-Location
  }
}

function Invoke-CrossBuild {
  param(
    [string]$Goos,
    [string]$Goarch
  )

  $prevGoos = $env:GOOS
  $prevGoarch = $env:GOARCH
  try {
    $env:GOOS = $Goos
    $env:GOARCH = $Goarch
    go build ./cmd/mama
  } finally {
    if ($null -eq $prevGoos) {
      Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    } else {
      $env:GOOS = $prevGoos
    }
    if ($null -eq $prevGoarch) {
      Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    } else {
      $env:GOARCH = $prevGoarch
    }
  }
}

Invoke-ReadinessCheck "go_test" { Invoke-GoInMama { go test ./... } }
Invoke-ReadinessCheck "go_build_host" { Invoke-GoInMama { go build ./... } }
Invoke-ReadinessCheck "go_vet" { Invoke-GoInMama { go vet ./... } }
Invoke-ReadinessCheck "go_mod_verify" { Invoke-GoInMama { go mod verify } }

if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
  Invoke-ReadinessCheck "govulncheck" { Invoke-GoInMama { govulncheck ./... } }
} else {
  Add-SummaryLine "check=govulncheck status=SKIP reason=tool_not_installed"
}

Invoke-ReadinessCheck "cross_build_linux_amd64" { Invoke-GoInMama { Invoke-CrossBuild -Goos "linux" -Goarch "amd64" } }
Invoke-ReadinessCheck "cross_build_linux_arm64" { Invoke-GoInMama { Invoke-CrossBuild -Goos "linux" -Goarch "arm64" } }
Invoke-ReadinessCheck "cross_build_darwin_amd64" { Invoke-GoInMama { Invoke-CrossBuild -Goos "darwin" -Goarch "amd64" } }
Invoke-ReadinessCheck "cross_build_darwin_arm64" { Invoke-GoInMama { Invoke-CrossBuild -Goos "darwin" -Goarch "arm64" } }
Invoke-ReadinessCheck "cross_build_windows_amd64" { Invoke-GoInMama { Invoke-CrossBuild -Goos "windows" -Goarch "amd64" } }
Invoke-ReadinessCheck "cross_build_windows_386" { Invoke-GoInMama { Invoke-CrossBuild -Goos "windows" -Goarch "386" } }

Add-SummaryLine "end_utc=$((Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"))"
Add-SummaryLine "result=PASS"

Write-Output "Production readiness checks completed successfully."
Write-Output "Artifacts: $runDir"
