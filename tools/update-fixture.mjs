import { createServer } from 'node:http'
import { createReadStream, readFileSync, writeFileSync, existsSync, statSync } from 'node:fs'
import { resolve, basename } from 'node:path'

const root = resolve(process.argv[2])
const server = createServer((request, response) => {
  const mode = existsSync(resolve(root, 'mode.json')) ? JSON.parse(readFileSync(resolve(root, 'mode.json'), 'utf8')) : {}
  const manifest = JSON.parse(readFileSync(resolve(root, 'latest.json'), 'utf8'))
  if (request.url === '/latest.json') {
    if (mode.manifest_status) { response.writeHead(mode.manifest_status).end(); return }
    response.setHeader('Content-Type', 'application/json')
    response.end(JSON.stringify(manifest))
  } else if (request.url === `/${basename(manifest.platforms['windows-x86_64'].url)}`) {
    if (mode.asset_status) { response.writeHead(mode.asset_status).end(); return }
    response.setHeader('Content-Type', 'application/octet-stream')
    response.setHeader('Content-Length', statSync(resolve(root, basename(manifest.platforms['windows-x86_64'].url))).size)
    createReadStream(resolve(root, basename(manifest.platforms['windows-x86_64'].url))).pipe(response)
  } else {
    response.writeHead(404).end()
  }
})
server.listen(0, '127.0.0.1', () => {
  writeFileSync(resolve(root, 'endpoint.txt'), `http://127.0.0.1:${server.address().port}`)
})
