// Run after exam-security-qa.mjs against the same isolated browser fixture.
// Uses real authenticated API responses; never intercepts or mocks network data.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'

const base = process.env.QA_BASE_URL || 'http://127.0.0.1:4176'
assert.equal(base, 'http://127.0.0.1:4176', 'Only the isolated fixture may be mutated')
const output = path.resolve('../../tmp/governance-qa')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true })
const checks = [], errors = []
async function shot(page, name) {
  await page.locator('.el-loading-mask').first().waitFor({ state: 'hidden' })
  await page.screenshot({ path: path.join(output, name + '.png'), fullPage: true, animations: 'disabled', style: '.el-message { visibility: hidden !important; }' })
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${name}: document overflow`)
  checks.push(name); console.log('PASS ' + name)
}
async function login(context, email) {
  const page = await context.newPage()
  page.on('pageerror', error => errors.push(error.message))
  await page.goto(base + '/login')
  await page.getByPlaceholder('邮箱', { exact: true }).fill(email)
  await page.getByPlaceholder('密码', { exact: true }).fill('ExamFixture-2026!')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.waitForURL(url => url.pathname !== '/login')
  return page
}
async function api(page, resource, method = 'GET', data) {
  return page.evaluate(async ({ resource, method, data }) => {
    const response = await fetch('/api/v1' + resource, { method, headers: { Authorization: `Bearer ${localStorage.getItem('school-oj-token')}`, 'Content-Type': 'application/json' }, body: data === undefined ? undefined : JSON.stringify(data) })
    return { status: response.status, data: await response.json(), version: response.headers.get('X-API-Version') }
  }, { resource, method, data })
}
async function verifyCode(page, name, title = '提交详情') {
  const dialog = page.getByRole('dialog', { name: title, exact: true })
  await dialog.locator('.source-code').waitFor()
  assert.equal(await dialog.locator('.source-viewer').evaluate(el => getComputedStyle(el).backgroundColor), 'rgb(255, 255, 255)')
  assert(await dialog.locator('.submission-detail-summary').innerText())
  const button = dialog.getByRole('button', { name: '复制代码', exact: true })
  const viewer = dialog.locator('.source-viewer')
  await page.mouse.move(0, 0)
  await page.waitForFunction(() => getComputedStyle(document.querySelector('.el-dialog .source-action')).opacity === '0')
  await viewer.hover()
  await page.waitForFunction(() => getComputedStyle(document.querySelector('.el-dialog .source-action')).opacity === '1')
  const b = await button.boundingBox(), area = await dialog.locator('.source-viewer').boundingBox()
  assert(b.x > area.x + area.width / 2 && area.x + area.width - b.x - b.width < 20 && b.y - area.y < 20, name + ': copy button placement')
  await button.click()
  assert.equal((await page.evaluate(() => navigator.clipboard.readText())).replace(/\r\n/g, '\n'), (await dialog.locator('.source-code').textContent()).replace(/\r\n/g, '\n'))
  await page.mouse.move(0, 0)
  await page.waitForFunction(() => getComputedStyle(document.querySelector('.el-dialog .source-action')).opacity === '0')
  await page.keyboard.press('Tab')
  await button.focus()
  assert.equal(await button.evaluate(el => el.matches(':focus-visible')), true)
  await shot(page, name)
  const bounds = await dialog.boundingBox(), viewport = page.viewportSize()
  assert(bounds.x >= 0 && bounds.x + bounds.width <= viewport.width + 1, name + ': dialog outside viewport')
  assert(await dialog.evaluate(el => el.scrollWidth <= el.clientWidth + 1), name + ': dialog content overflow')
  await dialog.getByRole('button', { name: /close/i }).click()
}
try {
  const studentContext = await browser.newContext({ permissions: ['clipboard-read', 'clipboard-write'] })
  const teacherContext = await browser.newContext({ permissions: ['clipboard-read', 'clipboard-write'] })
  const student = await login(studentContext, 'exam-student-0@example.test')
  const teacher = await login(teacherContext, 'exam-teacher-2@example.test')
  assert.equal((await api(student, '/me')).version, 'v1')
  for (const viewport of [{ width: 1440, height: 1100 }, { width: 390, height: 844 }]) {
    const size = viewport.width
    await student.setViewportSize(viewport); await teacher.setViewportSize(viewport)
    await student.goto(base + '/submissions')
    await student.locator('.el-table__body tbody tr').first().click()
    await verifyCode(student, `${size}-all-submissions`)
    await student.goto(base + '/problems/1/submissions')
    await student.getByRole('button', { name: '查看代码', exact: true }).first().click()
    await verifyCode(student, `${size}-practice-submission`)
    await student.goto(base + '/assignments/1')
    await student.getByRole('button', { name: '查看详情', exact: true }).first().click()
    await verifyCode(student, `${size}-assignment-submission`)
    await student.goto(base + '/problem-set/1#submissions')
    await student.getByRole('button', { name: '查看详情', exact: true }).first().click()
    await verifyCode(student, `${size}-team-set-submission`)
    await student.goto(base + '/contest/1/problems/B')
    await student.getByRole('button', { name: /^查看提交 / }).first().click()
    await verifyCode(student, `${size}-contest-submission`)

    await teacher.goto(base + '/admin/exams')
    await teacher.getByRole('row').filter({ hasText: '共享人工阅卷' }).getByRole('button', { name: '完成情况', exact: true }).click()
    await teacher.locator('.el-drawer .el-table__expand-icon').first().click()
    await teacher.getByRole('button', { name: '阅卷', exact: true }).first().click()
    await verifyCode(teacher, `${size}-manual-review`, '人工阅卷')
    await teacher.goto(base + '/teams/qa-invitation/contests')
    await teacher.getByRole('button', { name: '复制邀请码', exact: true }).waitFor()
    assert.equal(await teacher.getByRole('button', { name: /复制.*链接/ }).count(), 0)
    await shot(teacher, `${size}-invite-owner`)
    await teacher.goto(base + '/teams/qa-application/contests')
    await teacher.getByRole('button', { name: '复制申请链接', exact: true }).waitFor()
    assert.equal(await teacher.getByRole('button', { name: '复制邀请码', exact: true }).count(), 0)
    await shot(teacher, `${size}-application-owner`)
    assert.equal((await api(teacher, '/teams/3/members/1', 'PUT', { role: 'admin' })).status, 200)
    await student.goto(base + '/teams/qa-open/members')
    await student.locator('.member-panel .el-table__body tbody tr').first().waitFor()
    assert(await student.locator('.member-panel').getByText('创建者', { exact: true }).count())
    assert(await student.locator('.member-panel').getByText('管理员', { exact: true }).count())
    assert(await student.locator('.member-panel').getByText('团员', { exact: true }).count())
    assert.equal(await student.locator('.member-panel .el-select').count(), 0, 'team admin cannot adjust ranks')
    assert.equal((await api(student, '/teams/3/members/3', 'DELETE')).status, 403, 'admin cannot remove creator')
    await shot(student, `${size}-three-team-ranks`)
    assert.equal((await api(teacher, '/teams/3/members/1', 'PUT', { role: 'member' })).status, 200)
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ result: 'passed', source: 'real local API + PostgreSQL + Redis; seeded judge results', checks, errors }, null, 2))
  console.log(`PASSED ${checks.length} governance checkpoints`)
} catch (error) {
  for (const context of browser.contexts()) for (const [index, page] of context.pages().entries()) await page.screenshot({ path: path.join(output, `failure-${index}-${Date.now()}.png`), fullPage: true }).catch(() => {})
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ result: 'failed', checks, errors, error: String(error) }, null, 2))
  throw error
} finally { await browser.close() }
