import { readdir, readFile } from 'node:fs/promises'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../src')
const errors = []
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
    // Apply the color guard to converged surfaces; legacy modules can migrate
    // incrementally without weakening this boundary for the new shared UI.
    if (name.startsWith('views/teams/') || /^components\/(SourceCodeViewer|SubmissionDetailContent|SubmissionDetailDialog)\.vue$/.test(name)) {
      const styles = [...text.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/g)].map(m => m[1]).join('\n')
      if (/#[0-9a-fA-F]{3,8}\b|rgba?\(/.test(styles)) errors.push(`${name}: 已收敛组件应使用语义颜色 Token`)
    }
  }
}
await inspect(root)
if (errors.length) { console.error(errors.join('\n')); process.exit(1) }
console.log('前端架构校验通过：业务请求使用 operationId，团队/提交详情使用语义颜色 Token')
