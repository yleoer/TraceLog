[CmdletBinding()]
param([string]$RunRoot='')
$ErrorActionPreference='Stop'
$repoRoot=Split-Path -Parent $PSScriptRoot
if (-not $RunRoot) { $RunRoot=Join-Path ([IO.Path]::GetTempPath()) ('TraceLog-desktop-smoke-'+[guid]::NewGuid().ToString('N')) }
$root=[IO.Path]::GetFullPath($RunRoot)
if (Test-Path -LiteralPath $root) { throw '验收目录必须不存在' }
New-Item -ItemType Directory -Path $root | Out-Null
$names=@('VITE_TRACELOG_DESKTOP_E2E','APP_DATA_DIR','DATABASE_PATH')
$saved=@{}
foreach ($name in $names) { $saved[$name]=[Environment]::GetEnvironmentVariable($name) }
$process=$null
Push-Location $repoRoot
try {
  $env:VITE_TRACELOG_DESKTOP_E2E='1'
  $env:APP_DATA_DIR=Join-Path $root 'data'
  $env:DATABASE_PATH=''
  $config=Join-Path $root 'config.json'
  @{productName='TraceLog Desktop Test';identifier='app.tracelog.desktop-test';mainBinaryName='TraceLogDesktopTest';bundle=@{createUpdaterArtifacts=$false};app=@{windows=@(@{label='main';title='Desktop Probe';visible=$false})}} | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $config
  npm.cmd run tauri -- build --debug --no-bundle --features desktop-e2e --config $config
  if ($LASTEXITCODE -ne 0) { throw '桌面验收构建失败' }
  $process=Start-Process -FilePath "$repoRoot\target\debug\TraceLogDesktopTest.exe" -WindowStyle Hidden -PassThru
  $deadline=(Get-Date).AddMinutes(3)
  $result=Join-Path $env:APP_DATA_DIR 'desktop-smoke.json'
  while (-not (Test-Path -LiteralPath $result)) {
    $process.Refresh()
    if ($process.HasExited -or (Get-Date) -gt $deadline) { throw "桌面验收未完成，证据：$root" }
    Start-Sleep -Milliseconds 250
  }
  $report=Get-Content -LiteralPath $result -Raw | ConvertFrom-Json
  if (-not $report.passed) { throw ($report | ConvertTo-Json -Depth 10) }
  Write-Host "真实 WebView 与原生 IPC 验收通过，$($report.checks.Count) 项检查。证据：$root"
} finally {
  if ($process -and -not $process.HasExited) { Stop-Process -Id $process.Id -Force }
  foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name,$saved[$name],'Process') }
  Pop-Location
}
