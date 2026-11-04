[CmdletBinding()]
param(
  [string]$FromVersion = '0.2.0-rc.1',
  [string]$ToVersion = '0.2.0-rc.2',
  [string]$RunRoot = '',
  [string]$BuiltInstallers = '',
  [switch]$KeepArtifacts
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $RunRoot) { $RunRoot = Join-Path ([IO.Path]::GetTempPath()) ('TraceLog-tauri-upgrade-' + [guid]::NewGuid().ToString('N')) }
$testRoot = [IO.Path]::GetFullPath($RunRoot)
if (Test-Path -LiteralPath $testRoot) { throw '升级测试目录必须不存在，以避免覆盖已有文件。' }
New-Item -ItemType Directory -Path $testRoot | Out-Null
$installRoot = Join-Path $testRoot 'install'
$envNames = @('TAURI_SIGNING_PRIVATE_KEY', 'TAURI_SIGNING_PRIVATE_KEY_PASSWORD', 'TRACELOG_UPDATE_URL', 'TRACELOG_E2E_ROOT', 'APP_DATA_DIR')
$savedEnv = @{}
foreach ($name in $envNames) { $savedEnv[$name] = [Environment]::GetEnvironmentVariable($name) }
$fixture = $null
$success = $false

function Write-JSON([string]$Path, $Value) {
  [IO.File]::WriteAllText($Path, ($Value | ConvertTo-Json -Depth 20), [Text.UTF8Encoding]::new($false))
}

function Invoke-Checked([string]$Executable, [string[]]$Arguments) {
  & $Executable @Arguments
  if ($LASTEXITCODE -ne 0) { throw "命令失败：$Executable (exit $LASTEXITCODE)" }
}

function Build-Installer([string]$Version) {
  $configPath = Join-Path $testRoot "$Version.config.json"
  Write-JSON $configPath @{
    productName = 'TraceLog Upgrade Test'; identifier = 'app.tracelog.upgrade-test'; version = $Version
    mainBinaryName = 'TraceLogUpgradeTest'
    build = @{ frontendDist = (Join-Path $repoRoot 'tools/update-probe'); beforeBuildCommand = '' }
    plugins = @{ updater = @{ pubkey = (Get-Content -LiteralPath "$testRoot\test.key.pub" -Raw).Trim(); dangerousInsecureTransportProtocol = $true } }
  }
  Invoke-Checked 'npm.cmd' @('run', 'tauri', '--', 'build', '--ci', '--features', 'update-e2e', '--config', $configPath)
  $installer = Get-ChildItem -LiteralPath "$repoRoot\target\release\bundle\nsis" -Filter '*-setup.exe' |
    Where-Object { $_.Name -like "*$Version*" } | Select-Object -First 1
  if (-not $installer) { throw "缺少 $Version 安装器" }
  $destination = Join-Path $testRoot "$Version-setup.exe"
  Copy-Item -LiteralPath $installer.FullName -Destination $destination
  Copy-Item -LiteralPath ($installer.FullName + '.sig') -Destination ($destination + '.sig')
  return $destination
}

Push-Location $repoRoot
try {
  if ($BuiltInstallers) {
    foreach ($version in @($FromVersion, $ToVersion)) {
      Copy-Item -LiteralPath (Join-Path $BuiltInstallers "$version-setup.exe") -Destination $testRoot
      Copy-Item -LiteralPath (Join-Path $BuiltInstallers "$version-setup.exe.sig") -Destination $testRoot
    }
    $fromInstaller = Join-Path $testRoot "$FromVersion-setup.exe"
    $toInstaller = Join-Path $testRoot "$ToVersion-setup.exe"
  } else {
    & .\node_modules\.bin\tauri.cmd signer generate --ci --password= --write-keys "$testRoot\test.key" *> $null
    if ($LASTEXITCODE -ne 0) { throw '生成测试签名密钥失败' }
    $env:TAURI_SIGNING_PRIVATE_KEY = Join-Path $testRoot 'test.key'
    $env:TAURI_SIGNING_PRIVATE_KEY_PASSWORD = ''
    $fromInstaller = Build-Installer $FromVersion | Select-Object -Last 1
    $toInstaller = Build-Installer $ToVersion | Select-Object -Last 1
  }
  # Tauri 会在打包副本中写入 NSIS 类型标记，并恢复 Cargo 原始二进制；
  # 应与同一目标安装器的全新安装结果比较，不能拿未打包的 exe 比较。
  $referenceRoot = Join-Path $testRoot 'reference'
  $reference = Start-Process -FilePath $toInstaller -ArgumentList @('/S', "/D=$referenceRoot") -WindowStyle Hidden -Wait -PassThru
  if ($reference.ExitCode -ne 0) { throw '目标版本参考安装失败' }
  $expectedHash = (Get-FileHash -LiteralPath "$referenceRoot\TraceLogUpgradeTest.exe" -Algorithm SHA256).Hash
  Start-Process -FilePath "$referenceRoot\uninstall.exe" -ArgumentList @('/S', "_?=$referenceRoot") -WindowStyle Hidden -Wait | Out-Null
  Write-JSON "$testRoot\latest.json" @{ version = $ToVersion; notes = '本地签名升级验收'; platforms = @{'windows-x86_64' = @{ url = "http://127.0.0.1/$(Split-Path -Leaf $toInstaller)"; signature = (Get-Content -LiteralPath "$toInstaller.sig" -Raw).Trim() }} }
  $fixture = Start-Process -FilePath 'node.exe' -ArgumentList @("`"$repoRoot\tools\update-fixture.mjs`"", "`"$testRoot`"") -WindowStyle Hidden -PassThru
  $deadline = (Get-Date).AddSeconds(20)
  while (-not (Test-Path -LiteralPath "$testRoot\endpoint.txt")) {
    if ((Get-Date) -gt $deadline) { throw '测试更新源启动超时' }
    Start-Sleep -Milliseconds 250
  }
  $endpoint = (Get-Content -LiteralPath "$testRoot\endpoint.txt" -Raw).Trim()
  Write-JSON "$testRoot\latest.json" @{ version = $ToVersion; notes = '本地签名升级验收'; platforms = @{'windows-x86_64' = @{ url = "$endpoint/$(Split-Path -Leaf $toInstaller)"; signature = (Get-Content -LiteralPath "$toInstaller.sig" -Raw).Trim() }} }
  $env:TRACELOG_UPDATE_URL = "$endpoint/latest.json"
  $env:TRACELOG_E2E_ROOT = $testRoot
  $env:APP_DATA_DIR = Join-Path $testRoot 'data'
  New-Item -ItemType Directory -Path $env:APP_DATA_DIR | Out-Null
  [IO.File]::WriteAllText("$env:APP_DATA_DIR\preserve.txt", '升级必须保留的数据')
  $install = Start-Process -FilePath $fromInstaller -ArgumentList @('/S', "/D=$installRoot") -WindowStyle Hidden -Wait -PassThru
  if ($install.ExitCode -ne 0) { throw "旧版本安装失败：$($install.ExitCode)" }
  $installedExe = Join-Path $installRoot 'TraceLogUpgradeTest.exe'
  $old = Start-Process -FilePath $installedExe -WindowStyle Hidden -PassThru
  $deadline = (Get-Date).AddMinutes(3)
  while (-not (Test-Path -LiteralPath "$testRoot\started-$ToVersion.json")) {
    if (Test-Path -LiteralPath "$testRoot\failure.txt") { throw (Get-Content -LiteralPath "$testRoot\failure.txt" -Raw) }
    if ((Get-Date) -gt $deadline) { throw '升级或新版重启超时' }
    Start-Sleep -Milliseconds 500
  }
  $old.Refresh()
  if (-not $old.HasExited) { throw '旧进程没有退出' }
  $actualHash = (Get-FileHash -LiteralPath $installedExe -Algorithm SHA256).Hash
  if ($actualHash -ne $expectedHash) { throw '安装后的二进制与目标版本哈希不一致' }
  if ((Get-Content -LiteralPath "$env:APP_DATA_DIR\preserve.txt" -Raw) -ne '升级必须保留的数据') { throw '升级破坏了用户数据' }
  $new = Get-Content -LiteralPath "$testRoot\started-$ToVersion.json" -Raw | ConvertFrom-Json
  if ([IO.Path]::GetFullPath($new.exe) -ne [IO.Path]::GetFullPath($installedExe)) { throw '新版没有从原安装目录启动' }
  Write-JSON "$testRoot\result.json" @{ from = $FromVersion; to = $ToVersion; sha256 = $actualHash; old_process_exited = $true; restarted = $true; data_preserved = $true }
  $success = $true
  Write-Host "真实签名升级通过：$FromVersion -> $ToVersion。证据：$testRoot"
} finally {
  if ($fixture -and -not $fixture.HasExited) { Stop-Process -Id $fixture.Id -Force }
  Get-Process -Name 'TraceLogUpgradeTest' -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq "$installRoot\TraceLogUpgradeTest.exe" } | Stop-Process -Force
  $uninstaller = Join-Path $installRoot 'uninstall.exe'
  if (Test-Path -LiteralPath $uninstaller) { Start-Process -FilePath $uninstaller -ArgumentList @('/S', "_?=$installRoot") -WindowStyle Hidden -Wait | Out-Null }
  foreach ($name in $envNames) { [Environment]::SetEnvironmentVariable($name, $savedEnv[$name], 'Process') }
  Pop-Location
  # 私钥不属于测试证据，成功或失败均移除。
  Remove-Item -LiteralPath "$testRoot\test.key" -Force -ErrorAction SilentlyContinue
  if ($success -and -not $KeepArtifacts) { Remove-Item -LiteralPath $testRoot -Recurse -Force }
  elseif (-not $success) { Write-Host "升级失败，证据保留在 $testRoot" }
}
