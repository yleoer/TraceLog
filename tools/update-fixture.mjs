import { createServer } from 'node:http'
import { createReadStream, readFileSync, writeFileSync } from 'node:fs'
import { resolve, basename } from 'node:path'

const root = resolve(process.argv[2])
const server = createServer((request, response) => {
  const manifest = JSON.parse(readFileSync(resolve(root, 'latest.json'), 'utf8'))
  if (request.url === '/latest.json') {
    response.setHeader('Content-Type', 'application/json')
    response.end(JSON.stringify(manifest))
  } else if (request.url === `/${basename(manifest.platforms['windows-x86_64'].url)}`) {
    response.setHeader('Content-Type', 'application/octet-stream')
    createReadStream(resolve(root, basename(manifest.platforms['windows-x86_64'].url))).pipe(response)
  } else {
    response.writeHead(404).end()
  }
})
server.listen(0, '127.0.0.1', () => {
  writeFileSync(resolve(root, 'endpoint.txt'), `http://127.0.0.1:${server.address().port}`)
})
