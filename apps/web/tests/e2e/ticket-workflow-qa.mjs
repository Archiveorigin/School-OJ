// Uses only the isolated real API fixture; never production.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
const base = process.env.QA_BASE_URL || 'http://127.0.0.1:4177'
assert.equal(base, 'http://127.0.0.1:4177')
const output = path.resolve('../../tmp/ticket-workflow-qa')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true }), checks = [], errors = []
const picture = { name: '图 1.png', mimeType: 'image/png', buffer: await readFile('public/logo1.png') }
const zip = { name: 'cases.zip', mimeType: 'application/zip', buffer: await readFile('tests/fixtures/ticket-cases.zip') }
async function login(email, viewport) {
  const page = await browser.newPage({ viewport })
  page.on('pageerror', error => errors.push(error.message))
  page.on('response', r => { if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`) })
  await page.goto(base + '/login')
  await page.getByPlaceholder('邮箱', { exact: true }).fill(email)
  await page.getByPlaceholder('密码', { exact: true }).fill('ExamFixture-2026!')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.waitForURL(url => url.pathname !== '/login')
  return page
}
async function open(page, id, admin = false) {
  await page.goto(base + (admin ? '/admin/problem-authors' : '/problem-changes'))
  await page.locator('.el-table__row').filter({ has: page.getByText(`#${id}`, { exact: true }) }).getByRole('button', { name: '查看', exact: true }).click()
  await page.getByRole('heading', { name: `工单 #${id}`, exact: true }).waitFor()
}
async function response(page, method, ending, act, expected) {
  const pending = page.waitForResponse(r => r.request().method() === method && new URL(r.url()).pathname.endsWith(ending))
  await act(); const r = await pending
  assert.equal(r.status(), expected, await r.text()); return r.json()
}
async function shot(page, name) {
  for (const image of await page.locator('.problem-draft-preview img').all()) {
    await image.scrollIntoViewIfNeeded()
    await page.waitForFunction(img => img.complete && img.naturalWidth > 0, await image.elementHandle())
  }
  await page.evaluate(() => scrollTo(0, 0))
  await page.waitForFunction(() => !document.querySelector('[class*="enter-active"], [class*="leave-active"]'))
  await page.screenshot({ path: path.join(output, name + '.png'), fullPage: true, animations: 'disabled', style: '.el-message{visibility:hidden!important}' })
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), name + ': page overflow')
  checks.push(name); console.log('PASS ' + name)
}
async function apply(page, id) {
  await page.getByRole('button', { name: '执行并完成', exact: true }).click()
  const result = await response(page, 'POST', `/${id}/apply`, () => page.locator('.el-message-box').getByRole('button', { name: '确认执行', exact: true }).click(), 200)
  await page.locator('.decision-form').waitFor({ state: 'hidden' }); return result
}
try {
  for (const viewport of [{ width: 1440, height: 1100 }, { width: 390, height: 844 }]) {
    const size = viewport.width, author = await login('exam-teacher-2@example.test', viewport), admin = await login('exam-admin-3@example.test', viewport)
    await author.goto(base + '/problem-changes/new')
    await author.getByPlaceholder(/请具体说明问题/).fill('测试材料直接入库以及补充审核闭环')
    await author.getByPlaceholder('填写新题目标题').fill(`材料入库验收 ${size}`)
    await author.getByRole('textbox', { name: '题目描述', exact: true }).fill('输出数字 $1$。\n\n![题图](attachments/%E5%9B%BE%201.png)')
    await author.getByRole('textbox', { name: '输入格式', exact: true }).fill('输入可以为空。')
    await author.getByRole('textbox', { name: '输出格式', exact: true }).fill('输出数字 1。')
    await author.getByRole('textbox', { name: '样例 1 输出', exact: true }).fill('1')
    await author.locator('input[data-ticket-files]').setInputFiles([zip, picture, { name: 'empty.in', mimeType: 'text/plain', buffer: Buffer.alloc(0) }])
    const ticket = await response(author, 'POST', '/problem-change-tickets', () => author.getByRole('button', { name: '提交工单', exact: true }).click(), 201)
    await open(admin, ticket.id, true)
    assert.equal(await admin.getByText('选择完整 ZIP 题包', { exact: true }).count(), 0)
    await response(admin, 'POST', `/${ticket.id}/preview`, () => admin.getByRole('button', { name: '校验材料并预览', exact: true }).click(), 400)
    await admin.getByText(/测试点 empty 缺少配对/).waitFor()
    await shot(admin, `${size}-missing-material-explained`)
    await admin.getByPlaceholder('填写处理结果或驳回原因').fill('请补充 empty.out，其他材料已经收到。')
    await response(admin, 'POST', `/${ticket.id}/return`, () => admin.getByRole('button', { name: '退回补充', exact: true }).click(), 200)
    await open(author, ticket.id)
    await author.getByRole('button', { name: '补充材料并重提', exact: true }).click()
    assert.equal(await author.locator('.retained-file').count(), 0) // Files remain compact until the manager is opened.
    await author.locator('input[data-ticket-files]').setInputFiles({ name: 'empty.out', mimeType: 'text/plain', buffer: Buffer.from('1\n') })
    const resubmitted = await response(author, 'PATCH', `/problem-change-tickets/${ticket.id}`, () => author.getByRole('button', { name: '重新提交工单', exact: true }).click(), 200)
    assert.equal(resubmitted.revision, 3); assert.equal(resubmitted.attachments.length, 4)
    await shot(author, `${size}-resubmitted-history`)
    await open(admin, ticket.id, true)
    const preview = await response(admin, 'POST', `/${ticket.id}/preview`, () => admin.getByRole('button', { name: '校验材料并预览', exact: true }).click(), 200)
    assert.equal(preview.case_count, 101); assert.equal(preview.asset_count, 1)
    await admin.locator('.package-review .problem-draft-preview img').first().waitFor()
    await admin.waitForFunction(() => [...document.querySelectorAll('.package-review .problem-draft-preview img')].every(i => i.complete && i.naturalWidth > 0))
    await admin.getByRole('button', { name: '选择算法标签', exact: true }).click()
    const tagDialog = admin.getByRole('dialog', { name: '选择算法标签', exact: true })
    await tagDialog.getByRole('button', { name: '模拟', exact: true }).click()
    await tagDialog.getByRole('button', { name: '确认', exact: true }).click()
    await tagDialog.waitFor({ state: 'hidden' })
    await shot(admin, `${size}-ready-to-publish-from-attachments`)
    const result = await apply(admin, ticket.id)
    await shot(admin, `${size}-materials-published`)
    await author.goto(base + `/problems/${result.problem_id}`)
    await author.waitForFunction(() => [...document.querySelectorAll('.markdown-body img')].some(i => i.complete && i.naturalWidth > 0))
    await shot(author, `${size}-published-image-and-statement`)
    // Create a normal correction from the problem page, then edit the current statement.
    await author.getByRole('button', { name: '题目反馈', exact: true }).click()
    const dialog = author.getByRole('dialog', { name: '题目反馈', exact: true })
    await dialog.getByRole('button', { name: '提交反馈', exact: true }).click()
    await dialog.getByPlaceholder(/请具体说明问题/).fill('仅补充题面说明，保留原有全部测试点和图片')
    const correction = await response(author, 'POST', '/change-tickets', () => dialog.getByRole('button', { name: '提交工单', exact: true }).click(), 201)
    await open(admin, correction.id, true)
    await admin.getByRole('button', { name: '已选 1 个标签', exact: true }).waitFor()
    const statement = admin.locator('.package-review textarea').first()
    await statement.fill((await statement.inputValue()) + '\n\n补充：此题输出恒为 1。')
    assert.equal(await admin.getByRole('checkbox', { name: '用所选附件替换全部测试点', exact: true }).isChecked(), false)
    const editPreview = await response(admin, 'POST', `/${correction.id}/preview`, () => admin.getByRole('button', { name: '校验材料并预览', exact: true }).click(), 200)
    assert.equal(editPreview.case_count, 101); assert.equal(editPreview.asset_count, 1)
    await shot(admin, `${size}-statement-only-correction`)
    await apply(admin, correction.id)
    await author.reload(); await author.getByText('补充：此题输出恒为 1。', { exact: false }).first().waitFor()
    checks.push(`${size}-correction-keeps-tests-and-images`)
    await author.close(); await admin.close()
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'report.json'), JSON.stringify({ result: 'passed', checks, errors }, null, 2))
  console.log(`PASS ${checks.length} workflow checks`)
} catch (error) {
  await writeFile(path.join(output, 'failure.json'), JSON.stringify({ checks, errors, error: String(error) }, null, 2))
  for (const [i, context] of browser.contexts().entries()) for (const page of context.pages()) await page.screenshot({ path: path.join(output, `failure-${i}.png`), fullPage: true }).catch(() => {})
  throw error
} finally { await browser.close() }
