import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { gzipSync } from 'node:zlib'

const manifest = JSON.parse(await readFile('dist/.vite/manifest.json', 'utf8'))
const visited = new Set()
let bytes = 0, gzipBytes = 0
async function visit(key) {
  if (visited.has(key)) return
  visited.add(key)
  const chunk = manifest[key]
  assert.ok(chunk, `Missing manifest chunk ${key}`)
  assert.doesNotMatch(chunk.file, /(?:monaco|katex)/, `Heavy lazy module became an initial dependency: ${chunk.file}`)
  const data = await readFile(`dist/${chunk.file}`)
  bytes += data.length
  gzipBytes += gzipSync(data, { level: 9 }).length
  for (const dependency of chunk.imports || []) await visit(dependency)
}
await visit('index.html')
assert.ok(gzipBytes <= 250 * 1024, `Initial JavaScript exceeds 250 KiB gzip: ${gzipBytes}`)
console.log(`Initial JavaScript: ${bytes} bytes, ${gzipBytes} gzip bytes; editor and math remain lazy.`)
