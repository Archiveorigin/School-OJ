// Real local database fixture only. No production accounts or mutations.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import http from 'node:http'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { mkdir, readFile, writeFile } from 'node:fs/promises'

const origin = 'http://127.0.0.1:4179', api = 'http://127.0.0.1:18089'
const output = path.resolve('../../tmp/ranking-browser'), dist = path.resolve('dist')
await mkdir(output, { recursive: true })
const server = http.createServer(async (req, res) => {
  if (req.url.startsWith('/api/')) {
    const headers = { ...req.headers, host: '127.0.0.1:18089' }
    delete headers.origin
    const upstream = http.request(api + req.url, { method: req.method, headers }, result => { res.writeHead(result.statusCode, result.headers); result.pipe(res) })
    upstream.on('error', () => { res.writeHead(502); res.end() })
    req.pipe(upstream)
    return
  }
  const file = path.resolve(dist, '.' + new URL(req.url, origin).pathname)
  if (!file.startsWith(dist + path.sep)) { res.writeHead(403); res.end(); return }
  try {
    const bytes = await readFile(file)
    res.setHeader('Content-Type', ({ '.js': 'application/javascript', '.css': 'text/css', '.png': 'image/png', '.svg': 'image/svg+xml', '.woff2': 'font/woff2' })[path.extname(file)] || 'application/octet-stream')
    res.end(bytes)
  } catch { res.setHeader('Content-Type', 'text/html'); res.end(await readFile(path.join(dist, 'index.html'))) }
})
await new Promise(resolve => server.listen(4179, '127.0.0.1', resolve))
const browser = await chromium.launch({ headless: true }), errors = [], checks = []
try {
  const fixture = await (await fetch(api + '/__qa/info')).json()
  assert.equal(fixture.rankings.length, 6)
  const page = await browser.newPage({ viewport: { width: 1440, height: 1050 } })
  page.setDefaultTimeout(20000)
  page.on('pageerror', error => errors.push(error.message))
  page.on('response', response => { if (response.status() >= 500) errors.push(`${response.status()} ${response.url()}`) })
  await page.goto(origin + '/login')
  await page.getByPlaceholder('邮箱', { exact: true }).fill('exam-admin-3@example.test')
  await page.getByPlaceholder('密码', { exact: true }).fill('ExamFixture-2026!')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.waitForURL(url => url.pathname !== '/login')
  for (const item of fixture.rankings) {
    const name = `${item.kind}-${item.rule}`
    await page.goto(origin + `/admin/competitions/${item.kind}/${item.id}`)
    await page.getByRole('tab', { name: '成绩与排名', exact: true }).click()
    const excel = page.getByRole('button', { name: '导出 Excel', exact: true })
    await page.getByText(/待评测 1 份.*全部处理完成后可导出/).waitFor()
    assert.ok(await excel.isDisabled())
    assert.ok(await page.getByRole('button', { name: '确认完结', exact: true }).isDisabled())
    // No browser reload or manual refresh: the next automatic fetch must see
    // this judge result even though the event deadline has already passed.
    const resolved = await fetch(api + `/__qa/rankings/${item.kind}/${item.id}/resolve`, { method: 'POST' })
    assert.equal(resolved.status, 200)
    await page.getByText('已截止：评测与人工评分均已完成，可导出供管理员审核；确认完结后为最终成绩。').waitFor()
    assert.ok(await excel.isEnabled())
    assert.ok(await page.getByRole('button', { name: '确认完结', exact: true }).isEnabled())
    for (const [button, format] of [['导出 Excel', 'xlsx'], ['导出打印榜单', 'html']]) {
      const pending = page.waitForEvent('download')
      await page.getByRole('button', { name: button, exact: true }).click()
      const download = await pending
      assert.equal(download.suggestedFilename(), `排名表-${item.id}.${format}`)
      await download.saveAs(path.join(output, `${name}.${format}`))
    }
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 })
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 2), `${name}/${width}: overflow`)
      await page.screenshot({ path: path.join(output, `${name}-${width}-review.png`), fullPage: true })
    }
    const exported = await browser.newPage({ viewport: { width: 1440, height: 900 } })
    await exported.goto(pathToFileURL(path.join(output, `${name}.html`)).href)
    assert.ok((await exported.locator('table tbody tr').count()) >= 2)
    await exported.getByText('待审核', { exact: true }).waitFor()
    assert.equal(await exported.locator('script').count(), 0)
    await exported.screenshot({ path: path.join(output, `${name}-print.png`), fullPage: true })
    await exported.emulateMedia({ media: 'print' })
    await exported.pdf({ path: path.join(output, `${name}-print.pdf`), printBackground: true, preferCSSPageSize: true })
    await exported.close()
    await page.setViewportSize({ width: 1440, height: 1050 })
    checks.push({ ...item, autoRefreshAfterDeadline: true, export: ['xlsx', 'html', 'print-pdf'], desktopMobile: true })
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ passed: true, checks, errors }, null, 2))
  console.log(JSON.stringify({ passed: true, flows: checks.length, errors }))
} catch (error) {
  for (const context of browser.contexts()) for (const page of context.pages()) {
    console.error(page.url(), (await page.locator('body').innerText()).slice(-1500))
    await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true }).catch(() => {})
  }
  throw error
} finally {
  await browser.close()
  await new Promise(resolve => server.close(resolve))
}
