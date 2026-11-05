import { readFileSync, existsSync, copyFileSync, mkdirSync, writeFileSync } from 'node:fs'
import { resolve, basename } from 'node:path'

export function prepareRelease({ root, tag, repository, bundleDir, outputDir }) {
  const config = JSON.parse(readFileSync(resolve(root, 'src-tauri/tauri.conf.json'), 'utf8'))
  const version = config.version
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version) || tag !== `v${version}`) throw new Error('标签必须与 Tauri SemVer 版本完全一致')
  const cargo = readFileSync(resolve(root, 'src-tauri/Cargo.toml'), 'utf8').match(/^version\s*=\s*"([^"]+)"/m)?.[1]
  if (cargo !== version) throw new Error('Cargo 与 Tauri 版本不一致')
  if (!/^[\w.-]+\/[\w.-]+$/.test(repository)) throw new Error('GitHub 仓库名称无效')
  const notes = readFileSync(resolve(root, `release-notes/${tag}.md`), 'utf8').trim()
  if (!notes) throw new Error('必须提供非空的版本发布说明')
  const source = resolve(bundleDir, `${config.productName}_${version}_x64-setup.exe`)
  if (!existsSync(source)) throw new Error('缺少当前产品及版本的 Windows x64 安装器')
  const signature = readFileSync(`${source}.sig`, 'utf8').trim()
  const decoded = Buffer.from(signature, 'base64').toString('utf8')
  if (!decoded.startsWith('untrusted comment:') || !decoded.includes('trusted comment:')) throw new Error('缺少有效格式的 Tauri 签名')
  const name = `TraceLog-${tag}-windows-amd64-setup.exe`
  const manifest = {
    version, notes, pub_date: new Date().toISOString(),
    platforms: { 'windows-x86_64': { signature, url: `https://github.com/${repository}/releases/download/${tag}/${name}` } }
  }
  mkdirSync(outputDir, { recursive: true })
  copyFileSync(source, resolve(outputDir, name))
  writeFileSync(resolve(outputDir, `${name}.sig`), signature + '\n')
  // 仅在安装包和签名均已完整生成后写入清单，发布时整个目录先上传到草稿 Release。
  writeFileSync(resolve(outputDir, 'latest.json'), JSON.stringify(manifest, null, 2) + '\n')
  return manifest
}

if (process.argv[1] && basename(process.argv[1]) === 'prepare-release.mjs') {
  prepareRelease({ root: process.cwd(), tag: process.argv[2], repository: process.argv[3], bundleDir: resolve('target/release/bundle/nsis'), outputDir: resolve('release') })
}
