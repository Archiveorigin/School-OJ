import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const spec = JSON.parse(await readFile(resolve(here, '../../api/openapi/openapi.json'), 'utf8'))
const routes = await readFile(resolve(here, '../../api/internal/handlers/routes.go'), 'utf8')
const normalize = (path) =>
  path.replace(/:([A-Za-z0-9_]+)/g, '{$1}').replace(/\*([A-Za-z0-9_]+)/g, '{$1}')

const codeOperations = new Set()
for (const rawLine of routes.split(/\r?\n/)) {
  const line = rawLine.trim()
  if (line.startsWith('//')) continue
  const match = line.match(/^(?:api|auth)\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)"/)
  if (match) codeOperations.add(`${match[1]} ${normalize(match[2])}`)
}

const specOperations = new Set()
const operationIds = new Set()
for (const [path, pathItem] of Object.entries(spec.paths || {})) {
  for (const method of ['get', 'post', 'put', 'patch', 'delete']) {
    const operation = pathItem[method]
    if (!operation) continue
    const key = `${method.toUpperCase()} ${path}`
    specOperations.add(key)
    if (!operation.operationId) throw new Error(`${key} 缺少 operationId`)
    if (operationIds.has(operation.operationId)) throw new Error(`operationId 重复：${operation.operationId}`)
    operationIds.add(operation.operationId)
  }
}

const missingInSpec = [...codeOperations].filter((item) => !specOperations.has(item))
const missingInCode = [...specOperations].filter((item) => !codeOperations.has(item))
if (missingInSpec.length || missingInCode.length) {
  if (missingInSpec.length) console.error('代码存在但 OpenAPI 缺失：\n' + missingInSpec.join('\n'))
  if (missingInCode.length) console.error('OpenAPI 存在但代码未注册：\n' + missingInCode.join('\n'))
  process.exit(1)
}

console.log(`OpenAPI 路由校验通过：${specOperations.size} 个操作`)
