[CmdletBinding()]
param(
  [string]$FromVersion = 'v0.1.10-rc.1',
  [string]$ToVersion = 'v0.1.10-rc.2',
  [string]$MakensisPath = '',
  [string]$RunRoot = '',
  [switch]$KeepArtifacts
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $RunRoot) {
  $RunRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("TraceLog-upgrade-e2e\\" + [guid]::NewGuid().ToString('N'))
}
$runRoot = [System.IO.Path]::GetFullPath($RunRoot)
$installRoot = Join-Path $runRoot 'install'
$dataRoot = Join-Path $runRoot 'data'
$releaseRoot = Join-Path $runRoot 'release'
$readyFile = Join-Path $runRoot 'fixture-url.txt'
$fixtureLog = Join-Path $runRoot 'fixture.log'
$originalPath = $env:Path
$originalFixture = $env:VITE_TRACELOG_UPDATE_E2E
$originalUpdateURL = $env:TRACELOG_UPDATE_URL
$originalDataDir = $env:APP_DATA_DIR
$fixtureProcess = $null
$testFailed = $true

function Invoke-Native([string]$FilePath, [string[]]$Arguments, [string]$WorkingDirectory = $repoRoot) {
  Push-Location $WorkingDirectory
  try {
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
      throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
    }
  } finally {
    Pop-Location
  }
}

function Resolve-Makensis {
  if ($MakensisPath) {
    return (Resolve-Path -LiteralPath $MakensisPath).Path
  }
  $command = Get-Command makensis.exe -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }
  throw 'makensis.exe was not found. Install NSIS or pass -MakensisPath C:\path\to\makensis.exe.'
}

function Build-Installer([string]$Version, [bool]$AutoStart, [string]$Destination, [string]$ExpectedExe) {
  if ($AutoStart) {
    $env:VITE_TRACELOG_UPDATE_E2E = '1'
  } else {
    Remove-Item Env:VITE_TRACELOG_UPDATE_E2E -ErrorAction SilentlyContinue
  }

  Invoke-Native 'wails' @(
    'build', '-clean', '-platform', 'windows/amd64', '-nsis', '-webview2', 'embed',
    '-ldflags', "-X tracelog/internal/desktop.AppVersion=$Version"
  )

  $installerDir = Join-Path $repoRoot 'build\windows\installer'
  Invoke-Native $script:makensis @(
    '-DTRACELOG_E2E_USER_INSTALL', '-DARG_WAILS_AMD64_BINARY=..\..\bin\TraceLog.exe', 'project.nsi'
  ) $installerDir

  $installer = Join-Path $repoRoot 'build\bin\TraceLog-amd64-installer.exe'
  if (-not (Test-Path -LiteralPath $installer)) {
    throw "Expected test installer was not produced: $installer"
  }
  Copy-Item -LiteralPath $installer -Destination $Destination -Force
  Copy-Item -LiteralPath (Join-Path $repoRoot 'build\bin\TraceLog.exe') -Destination $ExpectedExe -Force
}

function Wait-FileText([string]$Path, [int]$TimeoutSeconds) {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (Test-Path -LiteralPath $Path) {
      $value = (Get-Content -LiteralPath $Path -Raw).Trim()
      if ($value) { return $value }
    }
    Start-Sleep -Milliseconds 100
  }
  throw "Timed out waiting for $Path"
}

function Wait-LogText([string]$Path, [string]$Needle, [int]$TimeoutSeconds) {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (Test-Path -LiteralPath $Path) {
      $content = Get-Content -LiteralPath $Path -Raw
      if ($content.Contains($Needle)) { return $content }
    }
    Start-Sleep -Milliseconds 250
  }
  throw "Timed out waiting for update log entry: $Needle"
}

function Stop-TestProcess([string]$Path) {
  $resolvedPath = [System.IO.Path]::GetFullPath($Path)
  Get-Process -Name TraceLog -ErrorAction SilentlyContinue | Where-Object {
    $_.Path -and [System.String]::Equals($_.Path, $resolvedPath, [System.StringComparison]::OrdinalIgnoreCase)
  } | Stop-Process -Force
}

New-Item -ItemType Directory -Force -Path $releaseRoot, $installRoot, $dataRoot | Out-Null

try {
  $script:makensis = Resolve-Makensis
  $env:Path = "$(Split-Path -Parent $script:makensis);$env:Path"

  $fromInstaller = Join-Path $releaseRoot 'TraceLog-from-installer.exe'
  # Keep the fixture asset name aligned with the production Windows selector.
  $toInstaller = Join-Path $releaseRoot 'TraceLog-windows-amd64-installer.exe'
  $expectedFromExe = Join-Path $releaseRoot 'TraceLog-from.exe'
  $expectedToExe = Join-Path $releaseRoot 'TraceLog-to.exe'
  Build-Installer $FromVersion $true $fromInstaller $expectedFromExe
  Build-Installer $ToVersion $false $toInstaller $expectedToExe

  $fixtureBinary = Join-Path $runRoot 'TraceLogUpdateFixture.exe'
  Invoke-Native (Get-Command go.exe).Source @('build', '-o', $fixtureBinary, './cmd/update-fixture')
  $fixtureProcess = Start-Process -FilePath $fixtureBinary -ArgumentList @(
    '--installer', $toInstaller, '--version', $ToVersion, '--ready-file', $readyFile
  ) -WorkingDirectory $repoRoot -WindowStyle Hidden -RedirectStandardOutput $fixtureLog -RedirectStandardError "$fixtureLog.err" -PassThru
  $fixtureURL = Wait-FileText $readyFile 30

  $install = Start-Process -FilePath $fromInstaller -ArgumentList @('/S', "/D=$installRoot") -WindowStyle Hidden -Wait -PassThru
  if ($install.ExitCode -ne 0) {
    throw "Initial installer failed with exit code $($install.ExitCode)"
  }

  $appPath = Join-Path $installRoot 'TraceLog.exe'
  $helperPath = Join-Path $installRoot 'TraceLogUpdater.exe'
  if (-not (Test-Path -LiteralPath $appPath) -or -not (Test-Path -LiteralPath $helperPath)) {
    throw 'Initial installer did not place TraceLog.exe and TraceLogUpdater.exe in the isolated install directory.'
  }

  $env:APP_DATA_DIR = $dataRoot
  $env:TRACELOG_UPDATE_URL = "$fixtureURL/releases/latest"
  $initialProcess = Start-Process -FilePath $appPath -WorkingDirectory $installRoot -PassThru
  $logPath = Join-Path $dataRoot 'logs\update.log'
  $logText = Wait-LogText $logPath 'update completed successfully' 180

  if (Get-Process -Id $initialProcess.Id -ErrorAction SilentlyContinue) {
    throw "The original TraceLog process $($initialProcess.Id) did not exit."
  }
  if ($logText -match '(?i)powershell|windows terminal') {
    throw 'Update log unexpectedly references a console-hosting command.'
  }
  foreach ($entry in @('update helper readiness acknowledged', 'helper readiness signal written', 'app process exited', 'installer phase completed', 'restart completed', 'update completed successfully')) {
    if (-not $logText.Contains($entry)) {
      throw "Upgrade log is missing required entry: $entry"
    }
  }
  if ((Get-FileHash -LiteralPath $appPath -Algorithm SHA256).Hash -ne (Get-FileHash -LiteralPath $expectedToExe -Algorithm SHA256).Hash) {
    throw 'Installed TraceLog.exe hash does not match the target release binary.'
  }
  if (-not (Test-Path -LiteralPath (Join-Path $dataRoot 'tracelog.db'))) {
    throw 'The application data directory was not preserved across the upgrade.'
  }
  if (-not (Test-Path -LiteralPath $helperPath)) {
    throw 'The upgraded installation does not contain TraceLogUpdater.exe.'
  }

  $resultPath = Join-Path $runRoot 'result.txt'
  Set-Content -LiteralPath $resultPath -Value "PASS $FromVersion -> $ToVersion`n$logPath" -NoNewline
  Write-Host "PASS: local upgrade $FromVersion -> $ToVersion"
  Write-Host "Evidence: $runRoot"
  $testFailed = $false
} finally {
  if ($fixtureProcess -and -not $fixtureProcess.HasExited) {
    Stop-Process -Id $fixtureProcess.Id -Force -ErrorAction SilentlyContinue
  }
  Stop-TestProcess (Join-Path $installRoot 'TraceLog.exe')
  if (-not $testFailed) {
    $uninstaller = Join-Path $installRoot 'uninstall.exe'
    if (Test-Path -LiteralPath $uninstaller) {
      Start-Process -FilePath $uninstaller -ArgumentList '/S' -WindowStyle Hidden -Wait
    }
  }
  if (-not $KeepArtifacts -and -not $testFailed -and (Test-Path -LiteralPath $runRoot)) {
    Remove-Item -LiteralPath $runRoot -Recurse -Force
  }
  $env:Path = $originalPath
  $env:VITE_TRACELOG_UPDATE_E2E = $originalFixture
  $env:TRACELOG_UPDATE_URL = $originalUpdateURL
  $env:APP_DATA_DIR = $originalDataDir
  $distPath = Join-Path $repoRoot 'frontend\dist\app'
  if (Test-Path -LiteralPath $distPath) {
    Remove-Item -LiteralPath $distPath -Recurse -Force
  }
}
