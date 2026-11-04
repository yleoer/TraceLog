[CmdletBinding()]
param(
  [string]$FromVersion = '0.2.0-rc.1',
  [string]$ToVersion = '0.2.0-rc.2',
  [string]$FinalVersion = '0.2.0-rc.3',
  [string]$RunRoot = '',
  [string]$BuiltInstallers = '',
  [switch]$KeepArtifacts
)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $RunRoot) { $RunRoot = Join-Path ([IO.Path]::GetTempPath()) ('TraceLog-tauri-upgrade-' + [guid]::NewGuid().ToString('N')) }
$testRoot = [IO.Path]::GetFullPath($RunRoot)
if (Test-Path -LiteralPath $testRoot) { throw '测试目录必须不存在。' }
New-Item -ItemType Directory -Path $testRoot | Out-Null
$installRoot = Join-Path $testRoot 'install'
$installedExe = Join-Path $installRoot 'TraceLogUpgradeTest.exe'
$envNames = @('TAURI_SIGNING_PRIVATE_KEY','TAURI_SIGNING_PRIVATE_KEY_PASSWORD','TRACELOG_UPDATE_URL','TRACELOG_E2E_ROOT','APP_DATA_DIR','DATABASE_PATH')
$savedEnv = @{}
foreach ($name in $envNames) { $savedEnv[$name] = [Environment]::GetEnvironmentVariable($name) }
$fixture = $null
$success = $false
function Write-JSON([string]$Path,$Value) { [IO.File]::WriteAllText($Path,($Value | ConvertTo-Json -Depth 50),[Text.UTF8Encoding]::new($false)) }
function Invoke-Checked([string]$Executable,[string[]]$Arguments) { & $Executable @Arguments; if ($LASTEXITCODE -ne 0) { throw "命令失败：$Executable (exit $LASTEXITCODE)" } }
function Stop-TestApp {
  Get-Process -Name 'TraceLogUpgradeTest' -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $installedExe } | Stop-Process -Force
}
function Build-Installer([string]$Version) {
  $config = Join-Path $testRoot "$Version.config.json"
  Write-JSON $config @{
    productName='TraceLog Upgrade Test';identifier='app.tracelog.upgrade-test';version=$Version;mainBinaryName='TraceLogUpgradeTest'
    build=@{frontendDist=(Join-Path $repoRoot 'tools/update-probe');beforeBuildCommand=''}
    app=@{windows=@(@{label='main';title='Upgrade Probe';visible=$false})}
    plugins=@{updater=@{pubkey=(Get-Content -LiteralPath "$testRoot\test.key.pub" -Raw).Trim();dangerousInsecureTransportProtocol=$true}}
  }
  Invoke-Checked 'npm.cmd' @('run','tauri','--','build','--ci','--features','update-e2e','--config',$config)
  $installer = @(Get-ChildItem -LiteralPath "$repoRoot\target\release\bundle\nsis" -Filter '*-setup.exe' | Where-Object { $_.Name -like "*_$($Version)_x64*" })
  if ($installer.Count -ne 1) { throw "缺少唯一的 $Version 安装器" }
  Copy-Item -LiteralPath $installer[0].FullName -Destination "$testRoot\$Version-setup.exe"
  Copy-Item -LiteralPath ($installer[0].FullName+'.sig') -Destination "$testRoot\$Version-setup.exe.sig"
}
function Set-Manifest([string]$Version,[string]$SignatureVersion=$Version) {
  Write-JSON "$testRoot\latest.json" @{version=$Version;notes='本地签名升级验收';platforms=@{'windows-x86_64'=@{url="$script:endpoint/$Version-setup.exe";signature=(Get-Content -LiteralPath "$testRoot\$SignatureVersion-setup.exe.sig" -Raw).Trim()}}}
}
function Wait-Marker([string]$Path) {
  $deadline=(Get-Date).AddMinutes(3)
  while (-not (Test-Path -LiteralPath $Path)) {
    if ((Get-Date) -gt $deadline) { throw "等待超时：$Path" }
    Start-Sleep -Milliseconds 250
  }
}
function Start-Scenario([string]$Name) {
  Stop-TestApp
  $root=Join-Path $testRoot $Name
  New-Item -ItemType Directory -Path $root | Out-Null
  $env:TRACELOG_E2E_ROOT=$root
  $process=Start-Process -FilePath $installedExe -WindowStyle Hidden -PassThru
  return @{root=$root;process=$process}
}
function Assert-Failure([string]$Name,[string]$Hash) {
  $scenario=Start-Scenario $Name
  Wait-Marker "$($scenario.root)\failure.txt"
  $scenario.process.Refresh()
  if ($scenario.process.HasExited) { throw "$Name 失败时关闭了原应用" }
  if ((Get-FileHash -LiteralPath $installedExe -Algorithm SHA256).Hash -ne $Hash) { throw "$Name 修改了原安装" }
  $script:results.Add(@{case=$Name;original_running=$true;binary_unchanged=$true;error=(Get-Content -LiteralPath "$($scenario.root)\failure.txt" -Raw)})
}
Push-Location $repoRoot
try {
  foreach ($version in @($FromVersion,$ToVersion,$FinalVersion)) {
    if ($version -notmatch '^\d+\.\d+\.\d+(?:-[a-zA-Z0-9.-]+)?$') { throw '版本格式无效' }
  }
  if ($BuiltInstallers) {
    foreach ($version in @($FromVersion,$ToVersion,$FinalVersion)) {
      Copy-Item -LiteralPath (Join-Path $BuiltInstallers "$version-setup.exe") -Destination $testRoot
      Copy-Item -LiteralPath (Join-Path $BuiltInstallers "$version-setup.exe.sig") -Destination $testRoot
    }
  } else {
    & .\node_modules\.bin\tauri.cmd signer generate --ci --password= --write-keys "$testRoot\test.key" *> $null
    if ($LASTEXITCODE -ne 0) { throw '生成测试密钥失败' }
    $env:TAURI_SIGNING_PRIVATE_KEY=Join-Path $testRoot 'test.key'
    $env:TAURI_SIGNING_PRIVATE_KEY_PASSWORD=''
    foreach ($version in @($FromVersion,$ToVersion,$FinalVersion)) { Build-Installer $version }
  }
  $expected=@{}
  foreach ($version in @($ToVersion,$FinalVersion)) {
    $referenceRoot=Join-Path $testRoot "reference-$version"
    $reference=Start-Process -FilePath "$testRoot\$version-setup.exe" -ArgumentList @('/S',"/D=$referenceRoot") -WindowStyle Hidden -Wait -PassThru
    if ($reference.ExitCode -ne 0) { throw '参考安装失败' }
    $expected[$version]=(Get-FileHash -LiteralPath "$referenceRoot\TraceLogUpgradeTest.exe" -Algorithm SHA256).Hash
    Start-Process -FilePath "$referenceRoot\uninstall.exe" -ArgumentList @('/S',"_?=$referenceRoot") -WindowStyle Hidden -Wait | Out-Null
  }
  Write-JSON "$testRoot\latest.json" @{}
  $fixture=Start-Process -FilePath 'node.exe' -ArgumentList @("`"$repoRoot\tools\update-fixture.mjs`"","`"$testRoot`"") -WindowStyle Hidden -PassThru
  Wait-Marker "$testRoot\endpoint.txt"
  $script:endpoint=(Get-Content -LiteralPath "$testRoot\endpoint.txt" -Raw).Trim()
  $env:TRACELOG_UPDATE_URL="$endpoint/latest.json"
  $env:APP_DATA_DIR=Join-Path $testRoot 'data'
  $env:DATABASE_PATH=''
  Invoke-Checked 'cargo' @('run','--bin','seed-demo','--','--data-dir',$env:APP_DATA_DIR)
  [IO.File]::WriteAllText("$env:APP_DATA_DIR\preserve.txt",'升级必须保留的数据')
  Write-JSON "$env:APP_DATA_DIR\tracelog-settings.json" @{ai=@{provider='deepseek'};prompts=@{issue_summary='保留自定义提示词'}}
  $imageHash=(Get-FileHash -LiteralPath "$env:APP_DATA_DIR\uploads\demo.png").Hash
  $settingsHash=(Get-FileHash -LiteralPath "$env:APP_DATA_DIR\tracelog-settings.json").Hash
  $install=Start-Process -FilePath "$testRoot\$FromVersion-setup.exe" -ArgumentList @('/S',"/D=$installRoot") -WindowStyle Hidden -Wait -PassThru
  if ($install.ExitCode -ne 0) { throw '旧版本安装失败' }
  $originalHash=(Get-FileHash -LiteralPath $installedExe -Algorithm SHA256).Hash
  $script:results=[Collections.Generic.List[object]]::new()
  Set-Manifest $FromVersion
  $baseline=Start-Scenario 'no-update'
  Wait-Marker "$($baseline.root)\checked-$FromVersion.json"
  $snapshot=(Get-Content -LiteralPath "$($baseline.root)\started-$FromVersion.json" -Raw | ConvertFrom-Json).snapshot | ConvertTo-Json -Depth 50 -Compress
  if (-not $snapshot -or $snapshot -eq 'null') { throw '缺少业务数据快照' }
  $baseline.process.Refresh()
  if ($baseline.process.HasExited) { throw '无更新时退出了原应用' }
  $results.Add(@{case='no-update';original_running=$true})
  Set-Manifest $ToVersion $FromVersion
  Assert-Failure 'wrong-signature' $originalHash
  Set-Manifest $ToVersion
  Write-JSON "$testRoot\mode.json" @{asset_status=503}
  Assert-Failure 'download-failure' $originalHash
  Write-JSON "$testRoot\mode.json" @{manifest_status=503}
  Assert-Failure 'manifest-failure' $originalHash
  Write-JSON "$testRoot\mode.json" @{}
  foreach ($pair in @(@($FromVersion,$ToVersion),@($ToVersion,$FinalVersion))) {
    $from=$pair[0];$to=$pair[1]
    Set-Manifest $to
    $scenario=Start-Scenario "upgrade-$to"
    Wait-Marker "$($scenario.root)\started-$to.json"
    $scenario.process.Refresh()
    if (-not $scenario.process.HasExited) { throw '旧进程未退出' }
    $new=Get-Content -LiteralPath "$($scenario.root)\started-$to.json" -Raw | ConvertFrom-Json
    if ([IO.Path]::GetFullPath($new.exe) -ne $installedExe) { throw '新版没有从原安装目录启动' }
    $hash=(Get-FileHash -LiteralPath $installedExe -Algorithm SHA256).Hash
    if ($hash -ne $expected[$to]) { throw '新版与参考安装哈希不一致' }
    if (($new.snapshot | ConvertTo-Json -Depth 50 -Compress) -ne $snapshot) { throw '业务数据发生变化' }
    if ((Get-FileHash -LiteralPath "$env:APP_DATA_DIR\uploads\demo.png").Hash -ne $imageHash -or (Get-FileHash -LiteralPath "$env:APP_DATA_DIR\tracelog-settings.json").Hash -ne $settingsHash) { throw '图片或设置发生变化' }
    if ((Get-Content -LiteralPath "$env:APP_DATA_DIR\preserve.txt" -Raw) -ne '升级必须保留的数据') { throw '测试文件丢失' }
    $results.Add(@{case='upgrade';from=$from;to=$to;sha256=$hash;old_process_exited=$true;restarted=$true;business_data_preserved=$true})
  }
  Write-JSON "$testRoot\result.json" $results
  $success=$true
  Write-Host "签名升级、连续升级和失败恢复验收通过。证据：$testRoot"
} finally {
  if ($fixture -and -not $fixture.HasExited) { Stop-Process -Id $fixture.Id -Force }
  Stop-TestApp
  if (Test-Path -LiteralPath "$installRoot\uninstall.exe") { Start-Process -FilePath "$installRoot\uninstall.exe" -ArgumentList @('/S',"_?=$installRoot") -WindowStyle Hidden -Wait | Out-Null }
  foreach ($name in $envNames) { [Environment]::SetEnvironmentVariable($name,$savedEnv[$name],'Process') }
  Pop-Location
  [IO.File]::Delete("$testRoot\test.key")
  if ($success -and -not $KeepArtifacts) {
    # 目标是本次新建的绝对测试目录；仅逐项删除其内部文件和空目录。
    $checked=[IO.Path]::GetFullPath($testRoot)
    if ($checked -ne $testRoot -or $checked -eq $repoRoot) { throw '拒绝清理非测试目录' }
    Get-ChildItem -LiteralPath $checked -File -Recurse | ForEach-Object { [IO.File]::Delete($_.FullName) }
    Get-ChildItem -LiteralPath $checked -Directory -Recurse | Sort-Object { $_.FullName.Length } -Descending | ForEach-Object { [IO.Directory]::Delete($_.FullName,$false) }
    [IO.Directory]::Delete($checked,$false)
  } elseif (-not $success) { Write-Host "验收失败，证据保留：$testRoot" }
}
