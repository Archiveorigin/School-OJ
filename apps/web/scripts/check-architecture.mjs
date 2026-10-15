import { readdir, readFile } from 'node:fs/promises'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../src')
const errors = []
const colorBaseline = JSON.parse(await readFile(new URL('./architecture-colors.json', import.meta.url), 'utf8'))
async function inspect(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const file = resolve(directory, entry.name)
    if (entry.isDirectory()) { await inspect(file); continue }
    if (!/\.(ts|vue)$/.test(entry.name)) continue
    const name = relative(root, file).replaceAll('\\', '/')
    if (name.startsWith('api/')) continue
    const text = await readFile(file, 'utf8')
    const script = name.endsWith('.vue') ? text.match(/<script\b[^>]*>([\s\S]*?)<\/script>/)?.[1] || '' : text
    const ast = ts.createSourceFile(name, script, ts.ScriptTarget.Latest, true)
    function visit(node) {
      if (ts.isCallExpression(node) && ts.isPropertyAccessExpression(node.expression)
        && ['client', 'http', 'axios'].includes(node.expression.expression.getText(ast))
        && ['get', 'post', 'put', 'patch', 'delete', 'request'].includes(node.expression.name.text)) {
        errors.push(`${name}: 业务请求必须使用 generatedApi 的 operationId`)
      }
      if (ts.isImportDeclaration(node) && /(?:^axios$|\/api\/http$)/.test(node.moduleSpecifier.text)) {
        errors.push(`${name}: 页面不得绕过契约直接依赖 HTTP 传输层`)
      }
      ts.forEachChild(node, visit)
    }
    visit(ast)
    if (/\.response\?\.data\?\.error/.test(script)) errors.push(`${name}: 使用 apiErrorMessage 统一错误提示`)
    // Every Vue surface is scanned. Existing literals are individually budgeted;
    // a new file, new literal, or additional occurrence cannot bypass the guard.
    const styles = [...text.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/g)].map(m => m[1]).join('\n')
    const counts = new Map()
    for (const match of styles.matchAll(/#[0-9a-fA-F]{3,8}\b|rgba?\([^)]*\)/g)) {
      const key = `${name}:${match[0]}`
      counts.set(key, (counts.get(key) || 0) + 1)
    }
    for (const [key, count] of counts) if (count > (colorBaseline[key] || 0)) errors.push(`${key}: 新样式必须使用语义颜色 Token`)
  }
}
await inspect(root)
if (errors.length) { console.error(errors.join('\n')); process.exit(1) }
console.log('前端架构校验通过：业务请求使用 operationId，全量扫描颜色存量预算与错误提示')
