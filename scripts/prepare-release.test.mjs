import { test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { prepareRelease } from './prepare-release.mjs'

test('release manifest requires matching versions, notes and signed x64 installer', t => {
  const root = mkdtempSync(join(tmpdir(), 'tracelog-release-'))
  t.after(() => {
    assert.ok(root.startsWith(join(tmpdir(), 'tracelog-release-')))
    rmSync(root, { recursive: true, force: true })
  })
  for (const dir of ['src-tauri', 'release-notes', 'bundle']) mkdirSync(join(root, dir))
  writeFileSync(join(root, 'src-tauri/tauri.conf.json'), JSON.stringify({ productName: 'TraceLog', version: '0.2.0' }))
  writeFileSync(join(root, 'src-tauri/Cargo.toml'), 'version = "0.2.0"\n')
  const args = { root, tag: 'v0.2.0', repository: 'yleoer/TraceLog', bundleDir: join(root, 'bundle'), outputDir: join(root, 'out') }
  assert.throws(() => prepareRelease({ ...args, tag: 'v0.3.0' }), /标签/)
  assert.throws(() => prepareRelease(args), /ENOENT/)
  writeFileSync(join(root, 'release-notes/v0.2.0.md'), '迁移版本说明')
  // 升级验收在同一个 bundle 目录生成测试产品，不能作为正式包发布。
  writeFileSync(join(root, 'bundle/TraceLog Upgrade Test_0.2.0_x64-setup.exe'), 'probe installer')
  assert.throws(() => prepareRelease(args), /安装器/)
  const installer = join(root, 'bundle/TraceLog_0.2.0_x64-setup.exe')
  writeFileSync(installer, 'fixture installer')
  writeFileSync(`${installer}.sig`, 'invalid')
  assert.throws(() => prepareRelease(args), /签名/)
  writeFileSync(`${installer}.sig`, Buffer.from('untrusted comment: fixture\nsignature\ntrusted comment: fixture\nsignature').toString('base64'))
  const manifest = prepareRelease(args)
  assert.equal(manifest.platforms['windows-x86_64'].url, 'https://github.com/yleoer/TraceLog/releases/download/v0.2.0/TraceLog-v0.2.0-windows-amd64-setup.exe')
  assert.equal(JSON.parse(readFileSync(join(root, 'out/latest.json'), 'utf8')).version, '0.2.0')
  assert.equal(readFileSync(join(root, 'out/TraceLog-v0.2.0-windows-amd64-setup.exe'), 'utf8'), 'fixture installer')
})
