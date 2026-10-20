// Run from apps/web with TestEventBrowserFixture listening on 127.0.0.1:18089.
// Optionally target an isolated seeded Compose project forwarded to 127.0.0.1:18091.
// Chromium resolves oj-http.test to loopback, but treats this ordinary HTTP origin
// as insecure. Do not use localhost, patch crypto, or allow insecure-origin flags.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import http from 'node:http'
import { networkInterfaces } from 'node:os'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'

const candidate = process.env.OJ_HTTP_QA_TARGET
assert.ok(!candidate || candidate === 'http://127.0.0.1:18091', 'Only the isolated candidate tunnel is supported')
const port = 4181
// A local adapter IP is useful on hosts whose network software intercepts .test.
const bind = process.env.OJ_HTTP_QA_BIND || '127.0.0.1'
assert.ok(Object.values(networkInterfaces()).flat().some(address => address.address === bind), 'Bind must be an IP owned by this machine')
const direct = candidate || `http://${bind}:${port}`
const origin = `http://${bind === '127.0.0.1' ? 'oj-http.test' : bind}:${port}`
const output = path.resolve(`../../tmp/uuid-${candidate ? 'candidate' : 'local'}-browser`)
const dist = path.resolve('dist')
await mkdir(output, { recursive: true })
let server
{
  // Serve the exact production build. Proxy API requests as production Nginx does.
  const mime = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.svg': 'image/svg+xml', '.png': 'image/png', '.woff2': 'font/woff2' }
  server = http.createServer(async (request, response) => {
    if (candidate || request.url.startsWith('/api/')) {
      const target = candidate || 'http://127.0.0.1:18089'
      const headers = { ...request.headers, host: new URL(target).host }
      delete headers.origin
      const upstream = http.request(target + request.url, { method: request.method, headers }, result => {
        response.writeHead(result.statusCode, result.headers)
        result.pipe(response)
      })
      upstream.on('error', () => { response.writeHead(502); response.end() })
      request.pipe(upstream)
      return
    }
    try {
      const pathname = decodeURIComponent(new URL(request.url, direct).pathname)
      const file = path.resolve(dist, '.' + pathname)
      if (!file.startsWith(dist + path.sep)) { response.writeHead(403); response.end(); return }
      let content
      try { content = await readFile(file) } catch {
        response.setHeader('Content-Type', 'text/html')
        response.end(await readFile(path.join(dist, 'index.html')))
        return
      }
      response.setHeader('Content-Type', mime[path.extname(file)] || 'application/octet-stream')
      response.end(content)
    } catch { response.writeHead(500); response.end() }
  })
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(port, bind, resolve) })
}
const browser = await chromium.launch({
  headless: true,
  args: ['--host-resolver-rules=MAP oj-http.test 127.0.0.1', '--no-proxy-server'],
})
const checks = [], errors = []
const v4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
const file = (name, text) => ({ name, mimeType: 'text/plain', buffer: Buffer.from(text) })
const markdown = process.env.OJ_IMPORT_MARKDOWN_FIXTURE
  ? await readFile(process.env.OJ_IMPORT_MARKDOWN_FIXTURE, 'utf8')
  : '# HTTP 导入回归\n\n## 题目描述\n输出输入的两倍。\n\n## 输入格式\n一个整数。\n\n## 输出格式\n一个整数。\n\n## 样例输入\n```\n1\n```\n\n## 样例输出\n```\n2\n```'
async function call(page, method, route, body) {
  const token = await page.evaluate(() => localStorage.getItem('school-oj-token'))
  const response = await page.request.fetch(direct + '/api/v1' + route, {
    method, headers: { Authorization: `Bearer ${token}` }, data: body,
  })
  assert.ok(response.ok(), `${method} ${route}: ${response.status()} ${await response.text()}`)
  return response.json()
}
async function save(page, paper, editor) {
  const pending = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/v1' + paper + '/problems')
  await editor.getByRole('button', { name: '保存题目', exact: true }).click()
  const response = await pending
  assert.equal(response.status(), 200, await response.text())
  await editor.waitFor({ state: 'hidden' })
}
try {
  for (const [size, viewport] of [
    ['desktop', { width: 1440, height: 1000 }],
    ['mobile', { width: 390, height: 844 }],
  ]) {
    const page = await browser.newPage({ viewport })
    page.setDefaultTimeout(30000)
    page.on('pageerror', error => errors.push(error.message))
    page.on('console', message => { if (message.type() === 'error') console.error('Browser console:', message.text()) })
    page.on('response', response => { if (response.status() >= 500) errors.push(`${response.status()} ${response.url()}`) })
    await page.goto(origin + '/login')
    const capabilities = await page.evaluate(() => ({
      secure: isSecureContext,
      uuid: typeof crypto.randomUUID,
      randomValues: typeof crypto.getRandomValues,
    }))
    assert.deepEqual(capabilities, { secure: false, uuid: 'undefined', randomValues: 'function' })
    await page.getByPlaceholder('邮箱', { exact: true }).fill(candidate ? 'teacher@school.local' : 'exam-teacher-2@example.test')
    await page.getByPlaceholder('密码', { exact: true }).fill(candidate ? 'password' : 'ExamFixture-2026!')
    await page.getByRole('button', { name: '登录', exact: true }).click()
    await page.waitForURL(url => url.pathname !== '/login')
    const team = candidate
      ? await call(page, 'POST', '/teams', { name: 'HTTP UUID QA', slug: `uuid-${size}-${Date.now()}`, join_mode: 'open', visibility: 'private', contest_permission: 'all' })
      : { id: 1 }
    for (const kind of ['exam', 'contest']) {
      const created = await call(page, 'POST', kind === 'exam' ? '/exams' : `/teams/${team.id}/contests`, {
        course_id: 1, title: `HTTP UUID ${kind} ${size}`, description: 'HTTP compatibility regression',
        starts_at: new Date(Date.now() + 3600000).toISOString(),
        ends_at: new Date(Date.now() + 7200000).toISOString(),
        duration_minutes: 60, scoring_rule: 'ioi',
      })
      const paper = `/event-papers/${kind}/${created.id}`
      await page.goto(origin + paper + '/edit')
      await page.getByRole('button', { name: '导入 Markdown 单题', exact: true }).click()
      const dialog = page.getByRole('dialog', { name: '导入 Markdown 单题', exact: true })
      await dialog.getByLabel('选择 Markdown 文档', { exact: true }).setInputFiles(file('题目.md', markdown))
      await dialog.getByRole('button', { name: '使用此题面', exact: true }).click()
      const editor = page.locator('.problem-editor')
      const importedTitle = await editor.getByLabel('题目标题', { exact: true }).inputValue()
      assert.ok(importedTitle.length > 0)
      await editor.locator('input[type=file]').setInputFiles([file('sample.in', '1\n'), file('sample.out', '2\n')])
      await editor.locator('.el-checkbox').filter({ hasText: '已对照原文核对题面、公式、图片和样例' }).click()
      assert.ok(await editor.getByRole('checkbox', { name: '已对照原文核对题面、公式、图片和样例' }).isChecked())
      await save(page, paper, editor)
      await page.getByRole('button', { name: '新增题目', exact: true }).first().click()
      for (const [label, value] of [
        ['题目标题', 'HTTP 分片上传'], ['题目描述', '输出输入的两倍。'],
        ['输入格式', '一个整数。'], ['输出格式', '一个整数。'],
        ['样例 1 输入', '1'], ['样例 1 输出', '2'],
      ]) await editor.getByLabel(label, { exact: true }).fill(value)
      const chunks = []
      const record = request => {
        const url = new URL(request.url())
        if (request.method() === 'PUT' && url.pathname.includes(paper + '/test-uploads/')) {
          const [id, index] = url.pathname.split('/').slice(-2)
          chunks.push({ id, index: Number(index) })
        }
      }
      page.on('request', record)
      await editor.locator('input[type=file]').setInputFiles([
        file('large.in', '1\n' + ' '.repeat(1024 * 1024 + 17)), file('large.out', '2\n'),
      ])
      await save(page, paper, editor)
      page.off('request', record)
      assert.deepEqual(chunks.map(chunk => chunk.index), [0, 1, 0])
      chunks.forEach(chunk => assert.match(chunk.id, v4))
      assert.equal(chunks[0].id, chunks[1].id)
      assert.notEqual(chunks[0].id, chunks[2].id)
      const saved = await call(page, 'GET', paper)
      assert.equal(saved.problems.length, 2)
      assert.equal(saved.problems[0].problem.title, importedTitle)
      assert.deepEqual(saved.problems.map(problem => problem.case_count), [1, 1])
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 2))
      await page.screenshot({ path: path.join(output, `${size}-${kind}.png`), fullPage: true })
      checks.push({ size, kind, capabilities, importedTitle, chunks, savedProblems: saved.problems.length })
      await call(page, 'DELETE', kind === 'exam' ? `/exams/${created.id}` : `/contests/${created.id}`)
    }
    await page.close()
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ passed: true, checks, errors }, null, 2))
  console.log(JSON.stringify({ passed: true, flows: checks.length, errors }))
} catch (error) {
  console.error('Browser errors:', errors)
  for (const context of browser.contexts()) for (const page of context.pages()) {
    console.error('Page:', page.url(), (await page.locator('body').innerText()).slice(-1800))
    await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true }).catch(() => {})
  }
  throw error
} finally {
  await browser.close()
  if (server) await new Promise(resolve => server.close(resolve))
}
