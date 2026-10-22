import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import http from 'node:http'
import path from 'node:path'
import { mkdir, readFile, writeFile } from 'node:fs/promises'

const origin = 'http://127.0.0.1:4179', api = 'http://127.0.0.1:18089'
const output = path.resolve('../../tmp/contest-ui-browser'), dist = path.resolve('dist')
await mkdir(output, { recursive: true })
const server = http.createServer(async (req, res) => {
  if (req.url.startsWith('/api/')) {
    const headers = { ...req.headers, host: '127.0.0.1:18089' }; delete headers.origin
    const upstream = http.request(api + req.url, { method: req.method, headers }, result => { res.writeHead(result.statusCode, result.headers); result.pipe(res) })
    upstream.on('error', () => { res.writeHead(502); res.end() }); req.pipe(upstream); return
  }
  const file = path.resolve(dist, '.' + new URL(req.url, origin).pathname)
  if (!file.startsWith(dist + path.sep)) { res.writeHead(403); res.end(); return }
  try { const bytes = await readFile(file); res.setHeader('Content-Type', ({ '.js': 'application/javascript', '.css': 'text/css', '.png': 'image/png', '.svg': 'image/svg+xml', '.woff2': 'font/woff2' })[path.extname(file)] || 'application/octet-stream'); res.end(bytes) }
  catch { res.setHeader('Content-Type', 'text/html'); res.end(await readFile(path.join(dist, 'index.html'))) }
})
await new Promise(resolve => server.listen(4179, '127.0.0.1', resolve))
const browser = await chromium.launch({ headless: true }), errors = [], checks = []
async function request(page, resource) {
  return page.evaluate(async resource => {
    const response = await fetch('/api/v1' + resource, { headers: { Authorization: 'Bearer ' + localStorage.getItem('school-oj-token') } })
    if (!response.ok) throw new Error(`${response.status} ${resource}`)
    return response.json()
  }, resource)
}
async function capture(page, name) {
  await page.waitForTimeout(250)
  assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 2), `${name}: page overflow`)
  await page.screenshot({ path: path.join(output, name + '.png'), fullPage: true })
}
try {
  const fixture = await (await fetch(api + '/__qa/info')).json()
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } })
  page.setDefaultTimeout(20000)
  page.on('pageerror', error => errors.push(error.message))
  page.on('response', response => { if (response.status() >= 500) errors.push(`${response.status()} ${response.url()}`) })
  await page.goto(origin + '/login')
  await page.getByPlaceholder('邮箱', { exact: true }).fill('exam-admin-3@example.test')
  await page.getByPlaceholder('密码', { exact: true }).fill('ExamFixture-2026!')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.waitForURL(url => url.pathname !== '/login')
  const contest = fixture.contest_id
  await page.goto(origin + `/contest/${contest}/description?manage=edit`)
  const dialog = page.getByRole('dialog', { name: '比赛设置', exact: true })
  await dialog.waitFor()
  const description = [`比赛说明：保留尚未保存的输入 ${Date.now()}`, ...Array.from({ length: 16 }, (_, i) => `第 ${i + 1} 行：长文本应自动撑开，不被后台刷新覆盖。`)].join('\n')
  await dialog.getByRole('textbox', { name: '比赛说明', exact: true }).fill(description)
  await page.waitForTimeout(16200)
  assert.equal(await dialog.getByRole('textbox', { name: '比赛说明', exact: true }).inputValue(), description)
  assert.ok(await dialog.locator('textarea').evaluate(el => el.scrollHeight <= el.clientHeight + 2))
  await dialog.getByText('人数（人）', { exact: true }).click()
  await dialog.getByRole('spinbutton', { name: '预计参赛人数', exact: true }).fill('6')
  await dialog.getByRole('spinbutton', { name: '金奖人数', exact: true }).fill('1')
  await dialog.getByRole('spinbutton', { name: '银奖人数', exact: true }).fill('0')
  await dialog.getByRole('spinbutton', { name: '铜奖人数', exact: true }).fill('1')
  await dialog.getByText('比例（%）', { exact: true }).click()
  assert.equal(Number(await dialog.getByRole('spinbutton', { name: '金奖百分比', exact: true }).inputValue()), 16.666666)
  await dialog.getByText('人数（人）', { exact: true }).click()
  assert.equal(Number(await dialog.getByRole('spinbutton', { name: '金奖人数', exact: true }).inputValue()), 1)
  await capture(page, 'settings-light-desktop')
  await page.evaluate(() => document.documentElement.classList.add('dark'))
  await page.setViewportSize({ width: 390, height: 844 })
  await capture(page, 'settings-dark-mobile')
  await dialog.getByRole('button', { name: '取消', exact: true }).click()
  await page.getByRole('button', { name: '继续编辑', exact: true }).click()
  assert.equal(await dialog.locator('textarea').inputValue(), description)
  await dialog.getByRole('button', { name: '保存', exact: true }).click()
  await dialog.waitFor({ state: 'hidden' })
  const saved = await request(page, `/contests/${contest}`)
  assert.equal(saved.contest.description, description)
  assert.equal(saved.contest.gold_award_percent, 16.666666)
  assert.equal(saved.contest.silver_award_percent, 0)
  assert.equal(saved.contest.submission_version, 0)
  checks.push('unsaved-description-survives-poll-and-cancel', 'textarea-autogrow', 'fractional-awards-persist-and-units-roundtrip')

  await page.setViewportSize({ width: 1440, height: 1100 })
  await page.goto(origin + `/event-papers/contest/${contest}/edit`)
  await page.getByText('尚未提交审核', { exact: true }).waitFor()
  const actions = page.locator('.problem-actions .el-button')
  assert.deepEqual(await actions.allTextContents(), ['新增题目', '导入 Markdown 单题', '添加现有题目'])
  const boxes = await actions.evaluateAll(buttons => buttons.map(b => { const r = b.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width } }))
  assert.ok(boxes[0].y < boxes[1].y && boxes[1].y < boxes[2].y && boxes.every(b => b.x === boxes[0].x && b.width === boxes[0].width))
  const before = await request(page, `/event-papers/contest/${contest}`)
  await page.getByRole('button', { name: '添加现有题目', exact: true }).click()
  const picker = page.getByRole('dialog', { name: '添加现有题目', exact: true })
  await picker.locator('.picker-row').first().waitFor()
  await picker.locator('.picker-row .el-checkbox:not(.is-disabled)').first().click()
  await picker.locator('.el-pager .number').filter({ hasText: /^2$/ }).click()
  await page.waitForTimeout(400)
  await picker.locator('.picker-row .el-checkbox:not(.is-disabled)').first().click()
  await picker.locator('.el-select').first().click()
  await page.getByRole('option', { name: '团队私有题目／题单', exact: true }).click()
  await picker.locator('.el-select').nth(1).click()
  await page.getByRole('option', { name: '团队训练题单', exact: true }).click()
  await page.waitForTimeout(400)
  assert.equal(await picker.locator('.picker-row').count(), 3)
  await picker.locator('.picker-row .el-checkbox').first().click()
  await capture(page, 'picker-light-desktop')
  await page.evaluate(() => document.documentElement.classList.add('dark'))
  await page.setViewportSize({ width: 390, height: 844 })
  await capture(page, 'picker-dark-mobile')
  await picker.getByRole('button', { name: '添加已选 3 道题目', exact: true }).click()
  await picker.waitFor({ state: 'hidden' })
  const after = await request(page, `/event-papers/contest/${contest}`)
  assert.equal(after.problems.length, before.problems.length + 3)
  assert.equal(after.event.review_revision, before.event.review_revision + 1)
  assert.equal(after.event.submission_version, 0)
  checks.push('vertical-actions', 'multi-page-public-and-team-set-selection', 'atomic-single-revision-batch')
  await page.getByRole('tab', { name: '说明与时间', exact: true }).click()
  await page.getByRole('textbox', { name: '场次说明', exact: true }).fill('保留组题说明编辑')
  await page.getByRole('button', { name: '刷新', exact: true }).click()
  assert.equal(await page.getByRole('textbox', { name: '场次说明', exact: true }).inputValue(), '保留组题说明编辑')
  await page.getByRole('button', { name: '保存说明与时间', exact: true }).click()
  await page.waitForTimeout(400)
  checks.push('authoring-refresh-preserves-settings')

  for (const rule of ['acm', 'ioi']) {
    const item = fixture.rankings.find(x => x.kind === 'contest' && x.rule === rule)
    await page.goto(origin + `/admin/competitions/contest/${item.id}`)
    await page.getByRole('tab', { name: '成绩与排名', exact: true }).click()
    await page.getByText('实时表现', { exact: true }).waitFor()
    assert.equal(await page.getByText('自动刷新', { exact: true }).count(), 0)
    assert.equal(await page.getByRole('combobox', { name: '排名模式' }).count(), 0)
    assert.equal(await page.getByText('Name', { exact: true }).count(), 1)
    await fetch(api + `/__qa/rankings/contest/${item.id}/resolve`, { method: 'POST' })
    await page.setViewportSize({ width: 1440, height: 1100 })
    await page.getByRole('button', { name: '刷新榜单', exact: true }).click()
    await page.getByText('最终排名', { exact: true }).waitFor()
    for (const theme of ['light', 'dark']) {
      await page.evaluate(theme => document.documentElement.classList.toggle('dark', theme === 'dark'), theme)
      for (const width of [1440, 390]) {
        await page.setViewportSize({ width, height: width === 390 ? 844 : 1100 })
        await capture(page, `ranking-${rule}-${theme}-${width}`)
      }
    }
    const surfaces = await page.locator('.scoreboard-core, .summary-dialog').evaluateAll(nodes => nodes.map(el => getComputedStyle(el).backgroundColor))
    assert.ok(surfaces.every(s => !s.includes('255, 255, 255')))
    assert.equal(await page.locator('.native-stripe-row.award-gold, .native-stripe-row.award-silver, .native-stripe-row.award-bronze').count(), 0)
    checks.push(`ranking-${rule}-name-award-dark-mobile-final-readiness`)
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ passed: true, checks, errors }, null, 2))
  console.log(JSON.stringify({ passed: true, checks, errors }))
} catch (error) {
  for (const context of browser.contexts()) for (const page of context.pages()) {
    console.error(page.url(), (await page.locator('body').innerText()).slice(-1800))
    await page.screenshot({ path: path.join(output, 'failure.png'), fullPage: true }).catch(() => {})
  }
  throw error
} finally { await browser.close(); await new Promise(resolve => server.close(resolve)) }
