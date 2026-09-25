// Real, isolated API/MinIO only. Do not run against production.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
const base = process.env.QA_BASE_URL || 'http://127.0.0.1:4177'
assert.equal(base, 'http://127.0.0.1:4177')
const output = path.resolve('../../tmp/ticket-import-team-qa')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true }), errors = [], checks = []
const markdown = '# 区间求和与边界\n\n## 题目描述\n\n求区间内的和 $S=\\sum_{i=l}^{r}a_i$。\n\n$$S=r(r+1)/2$$\n\n## 输入格式\n\n输入两个整数 $l,r$。\n\n## 输出格式\n\n输出一个整数。\n\n## 样例输入 1\n\n```text\n1 3\n```\n\n## 样例输出 1\n\n```text\n6\n```'
const pairs = Array.from({ length: 15 }, (_, i) => [
  { name: `${i + 1}.in`, mimeType: 'text/plain', buffer: Buffer.from('1 3\n') },
  { name: `${i + 1}.out`, mimeType: 'text/plain', buffer: Buffer.from('6\n') },
]).flat()
const icon = { name: 'team-icon.png', mimeType: 'image/png', buffer: await readFile('public/logo1.png') }
async function shot(page, name) {
  await page.waitForFunction(() => !document.querySelector('[class*="enter-active"], [class*="leave-active"]'))
  await page.screenshot({ path: path.join(output, `${name}.png`), fullPage: !(await page.locator('.el-overlay-dialog:visible').count()), animations: 'disabled', style: '.el-message{visibility:hidden!important}' })
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${name}: overflow`)
  checks.push(name); console.log('PASS ' + name)
}
async function login(viewport) {
  const context = await browser.newContext({ viewport }), page = await context.newPage()
  page.on('pageerror', error => errors.push(error.message))
  page.on('response', response => { if (response.status() >= 500) errors.push(`${response.status()} ${response.url()}`) })
  await page.goto(base + '/login')
  await page.getByPlaceholder('邮箱', { exact: true }).fill('exam-teacher-2@example.test')
  await page.getByPlaceholder('密码', { exact: true }).fill('ExamFixture-2026!')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.waitForURL(url => url.pathname !== '/login')
  return page
}
try {
  for (const viewport of [{ width: 1440, height: 1100 }, { width: 390, height: 844 }]) {
    const page = await login(viewport), size = viewport.width
    await page.goto(base + '/profile')
    const entry = page.getByRole('button', { name: '进入我的工单', exact: true })
    await entry.waitFor(); const rect = await entry.boundingBox()
    assert(rect.x >= 0 && rect.x + rect.width <= size)
    await shot(page, `${size}-profile-ticket-entry`)
    await entry.click(); await page.waitForURL('**/problem-changes')
    await page.goto(base + '/problem-changes/new?from=catalog')
    await page.locator('input[aria-label="导入题目 Markdown"]').setInputFiles({ name: 'problem.md', mimeType: 'text/markdown', buffer: Buffer.from(markdown) })
    await page.getByPlaceholder('填写新题目标题').fill(`区间求和与边界 ${size}`)
    await page.getByPlaceholder(/请具体说明问题/).fill('增加测试')
    await page.getByRole('textbox', { name: '题目描述', exact: true }).waitFor()
    assert((await page.getByRole('textbox', { name: '题目描述', exact: true }).inputValue()).includes('\\sum'))
    assert.equal(await page.getByRole('textbox', { name: '样例 1 输出', exact: true }).inputValue(), '6')
    await page.locator('input[data-ticket-files]').setInputFiles(pairs)
    await page.getByRole('button', { name: '管理附件（30）', exact: true }).click()
    const files = page.getByRole('dialog', { name: '工单附件', exact: true })
    await files.locator('.file-chip').nth(29).waitFor()
    const grid = await files.locator('.file-grid').boundingBox(); assert(grid.height <= viewport.height * .6)
    await shot(page, `${size}-attachment-manager`)
    await files.getByRole('button', { name: '清空待上传', exact: true }).click()
    await files.getByRole('button', { name: '完成', exact: true }).click(); await files.waitFor({ state: 'hidden' })
    if (size === 390) await page.locator('.mobile-tabs').getByText('预览', { exact: true }).click()
    const preview = page.locator('.problem-draft-preview').first()
    await preview.locator('.katex').first().waitFor()
    assert.deepEqual(await preview.locator('h2').allTextContents(), ['题目描述', '输入格式', '输出格式', '样例'])
    assert.equal(await preview.evaluate(e => getComputedStyle(e).maxHeight), 'none')
    await shot(page, `${size}-imported-live-preview`)
    await preview.getByRole('button', { name: '展开预览', exact: true }).click()
    const expanded = page.getByRole('dialog', { name: '题目预览', exact: true })
    await expanded.locator('.katex').first().waitFor()
    await shot(page, `${size}-expanded-preview`)
    await expanded.locator('.el-dialog__headerbtn').click(); await expanded.waitFor({ state: 'hidden' })
    const zip = { name: 'cases.zip', mimeType: 'application/zip', buffer: await readFile('tests/fixtures/ticket-cases.zip') }
    const upload = size === 1440 ? [zip, { name: 'large.in', mimeType: 'text/plain', buffer: Buffer.alloc(5 * 1024 * 1024, '1') }, { name: 'large.out', mimeType: 'text/plain', buffer: Buffer.from('1') }] : [zip]
    await page.locator('input[data-ticket-files]').setInputFiles(upload)
    const cdp = await page.context().newCDPSession(page)
    if (size === 1440) await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: 1024 * 1024, uploadThroughput: 128 * 1024 })
    const started = Date.now(), response = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/problem-change-tickets'), { timeout: 120000 })
    await page.getByRole('button', { name: '提交工单', exact: true }).click()
    await page.locator('.upload-progress').waitFor()
    await shot(page, `${size}-upload-progress`)
    const result = await response; assert.equal(result.status(), 201, await result.text())
    const ticket = await result.json(); assert.equal(ticket.attachment_count, upload.length)
    if (size === 1440) { assert(Date.now() - started > 30000, 'Slow-network upload did not exercise the old 30-second timeout'); checks.push('slow-network-upload-over-30-seconds') }
    await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 }); await cdp.detach()
    await page.waitForURL('**/problem-changes')
    await page.goto(base + '/problem-set/1#problems')
    await page.getByRole('button', { name: '编辑题单信息', exact: true }).click()
    const editor = page.getByRole('dialog', { name: '编辑题单信息', exact: true })
    await editor.locator('input[type=file]').setInputFiles(icon)
    await editor.locator('.team-icon-button img').evaluate(img => img.decode())
    const imageRect = await editor.locator('.team-icon-button').boundingBox(), editorRect = await editor.boundingBox()
    assert(Math.abs(imageRect.x + imageRect.width / 2 - editorRect.x - editorRect.width / 2) < 3)
    await shot(page, `${size}-problem-set-icon-editor`)
    await editor.getByRole('button', { name: '保存修改', exact: true }).click()
    await editor.waitFor({ state: 'hidden' })
    await page.reload(); assert((await page.locator('.set-identity img').getAttribute('src')).startsWith('data:image/png'))
    await page.getByRole('button', { name: '添加题目', exact: true }).click()
    const picker = page.getByRole('dialog', { name: '添加题目', exact: true })
    await picker.getByPlaceholder('搜索题号或题目名称').fill('QA')
    await picker.locator('.el-select').click()
    await page.getByRole('option', { name: '基础', exact: true }).click()
    await picker.getByRole('button', { name: '选择算法标签', exact: true }).click()
    const tags = page.getByRole('dialog', { name: '选择算法标签', exact: true })
    await tags.getByRole('button', { name: '模拟', exact: true }).click()
    await tags.getByRole('button', { name: '确认', exact: true }).click()
    const filtered = page.waitForResponse(r => r.url().includes('/problems/catalog?') && r.url().includes('tags='))
    await picker.getByRole('button', { name: '检索', exact: true }).click(); await filtered
    await picker.locator('.picker-row').nth(19).waitFor()
    const first = picker.locator('.picker-row .el-checkbox:not(.is-disabled)').first()
    await first.click()
    const nextPage = page.waitForResponse(r => r.url().includes('/problems/catalog?') && new URL(r.url()).searchParams.get('page') === '2')
    await picker.locator('.el-pagination .btn-next').click(); await nextPage
    await page.waitForFunction(() => document.querySelectorAll('.picker-row').length === 5)
    await picker.locator('.picker-row .el-checkbox:not(.is-disabled)').first().click()
    await picker.getByText('已选 2 道题目', { exact: true }).waitFor()
    await shot(page, `${size}-problem-selection`)
    const adding = page.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/problem-sets/1/problems'))
    await picker.getByRole('button', { name: '添加已选 2 道题目', exact: true }).click()
    const added = await adding; assert.equal(added.status(), 201, await added.text()); assert.equal((await added.json()).added, 2)
    await picker.waitFor({ state: 'hidden' })
    await shot(page, `${size}-problem-set-after-batch`)
    await page.context().close()
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'report.json'), JSON.stringify({ result: 'passed', checks, errors }, null, 2))
  console.log(`PASS ${checks.length} import/team acceptance checks`)
} catch (error) {
  await writeFile(path.join(output, 'failure.json'), JSON.stringify({ checks, errors, error: String(error) }, null, 2))
  for (const [i, context] of browser.contexts().entries()) for (const page of context.pages()) await page.screenshot({ path: path.join(output, `failure-${i}.png`), fullPage: true }).catch(() => {})
  throw error
} finally { await browser.close() }
