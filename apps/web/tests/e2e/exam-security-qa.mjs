// Requires TestExamBrowserFixture and a production preview proxying its real API.
// Never point this mutating acceptance test at production.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'

const base = process.env.QA_BASE_URL || 'http://127.0.0.1:4176'
assert.equal(base, 'http://127.0.0.1:4176', 'Only the isolated local fixture is allowed')
const output = path.resolve('../../tmp/exam-security-qa')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true })
const checks = [], errors = []
async function shot(page, name) {
  await page.locator('.el-loading-mask').first().waitFor({ state: 'hidden' })
  await page.screenshot({ path: path.join(output, `${name}.png`), fullPage: true, animations: 'disabled', style: '.el-message { visibility: hidden !important; }' })
  assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${name}: page overflow`)
  checks.push(name)
  console.log(`PASS ${name}`)
}
async function login(context, email) {
  const page = await context.newPage()
  page.on('pageerror', error => errors.push(error.message))
  await page.goto(`${base}/login`)
  await page.getByPlaceholder('邮箱', { exact: true }).fill(email)
  await page.getByPlaceholder('密码', { exact: true }).fill('ExamFixture-2026!')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.waitForURL(url => url.pathname !== '/login')
  return page
}
async function api(page, resource, method = 'GET', data) {
  return page.evaluate(async ({ resource, method, data }) => {
    const response = await fetch(`/api${resource}`, { method, headers: { Authorization: `Bearer ${localStorage.getItem('school-oj-token')}`, 'Content-Type': 'application/json' }, body: data === undefined ? undefined : JSON.stringify(data) })
    return { status: response.status, data: await response.json() }
  }, { resource, method, data })
}
async function dismissWarning(page) {
  const button = page.getByRole('button', { name: '返回作答', exact: true })
  if (await button.isVisible()) await button.click()
}
try {
  const teacherContext = await browser.newContext({ viewport: { width: 1440, height: 1100 } })
  const teacher = await login(teacherContext, 'exam-teacher-2@example.test')
  for (const [viewport, userIndex] of [[{ width: 1440, height: 1100 }, 0], [{ width: 390, height: 844 }, 1]]) {
    const size = String(viewport.width)
    const context = await browser.newContext({ viewport, permissions: ['clipboard-read', 'clipboard-write'] })
    const page = await login(context, `exam-student-${userIndex}@example.test`)
    for (const [id, rule, hidden] of [[1, 'ioi', false], [2, 'oi', true], [3, 'acm', false], [4, 'ioi', true], [5, 'acm', true]]) {
      await page.goto(`${base}/exams/${id}/description`)
      await page.locator('.el-message-box, .exam-shell nav').first().waitFor()
      if (await page.getByRole('button', { name: '确认进入', exact: true }).isVisible()) await page.getByRole('button', { name: '确认进入', exact: true }).click()
      await page.getByRole('navigation', { name: '考试导航' }).waitFor()
      await dismissWarning(page)
      await shot(page, `${size}-exam-${id}-description`)
      assert.equal(await page.getByRole('link', { name: '排行榜', exact: true }).count(), rule === 'oi' ? 0 : 1)
      await page.getByRole('link', { name: '题目列表', exact: true }).click()
      await page.getByRole('link', { name: '特殊扫雷', exact: true }).waitFor()
      await shot(page, `${size}-exam-${id}-problems`)
      await page.getByRole('link', { name: '特殊扫雷', exact: true }).click()
      await page.getByRole('heading', { name: 'B.特殊扫雷', exact: true, level: 1 }).waitFor()
      await page.getByRole('button', { name: /^查看提交 / }).first().click()
      const dialog = page.getByRole('dialog', { name: '提交详情', exact: true })
      await dialog.waitFor()
      const summary = await dialog.locator('.submission-detail-summary').innerText()
      assert(summary.includes(hidden ? '运行时间：—' : '运行时间：302ms'), summary)
      assert(summary.includes(hidden ? '用例通过情况：—' : '用例通过情况：2/2'), summary)
      assert.equal(await dialog.locator('.source-viewer').evaluate(el => getComputedStyle(el).backgroundColor), 'rgb(255, 255, 255)')
      assert.equal(await dialog.locator('.source-lines').textContent(), '1\n2\n3\n4\n5')
      const codeBounds = await dialog.locator('.source-viewer').boundingBox()
      const copyBounds = await dialog.getByRole('button', { name: '复制代码', exact: true }).boundingBox()
      assert(copyBounds.x > codeBounds.x + codeBounds.width / 2 && codeBounds.x + codeBounds.width - copyBounds.x - copyBounds.width < 20, 'copy button must be at top right')
      assert(copyBounds.y - codeBounds.y < 20, 'copy button must be at the top of the code area')
      await dialog.locator('.source-viewer').hover()
      await dialog.getByRole('button', { name: '复制代码', exact: true }).click()
      assert.equal((await page.evaluate(() => navigator.clipboard.readText())).replace(/\r\n/g, '\n'), await dialog.locator('.source-code').innerText())
      await shot(page, `${size}-exam-${id}-submission`)
      await dialog.getByRole('button', { name: /close/i }).click()
      await shot(page, `${size}-exam-${id}-problem`)
      if (rule !== 'oi') {
        await page.getByRole('link', { name: '排行榜', exact: true }).click()
        await page.locator('.scoreboard-core').waitFor()
        assert.equal(await page.locator('.scoreboard-core').getAttribute('data-scoring-rule'), rule === 'ioi' ? 'score' : 'penalty')
        if (id >= 4) await page.getByText('排行榜已封榜', { exact: false }).waitFor()
        await shot(page, `${size}-exam-${id}-scoreboard`)
      }
      if (id === 1) {
        await page.goto(`${base}/exams/${id}/records?problem_id=1`)
        await page.waitForURL(`**/exams/${id}/problems/B`)
        await page.getByRole('heading', { name: 'B.特殊扫雷', exact: true, level: 1 }).waitFor()
        await shot(page, `${size}-legacy-exam-route`)
        await page.getByRole('button', { name: '提交代码', exact: true }).click()
        const composer = page.getByRole('dialog', { name: /^提交 B\./ })
        await composer.locator('.monaco-editor').waitFor()
        const submitted = page.waitForResponse(response => /\/api(?:\/v1)?\/submissions$/.test(response.url()) && response.request().method() === 'POST')
        await composer.getByRole('button', { name: '提交评测', exact: true }).click()
        const recorded = await submitted
        assert.equal(recorded.status(), 201)
        const newSubmission = await recorded.json()
        assert.equal(newSubmission.exam_id, id)
        assert.equal(newSubmission.problem_version_id, 1)
        await shot(page, `${size}-exam-submit-recorded`)
        await composer.getByRole('button', { name: /close/i }).click()
        // Genuine second page + focus events, with no response interception.
        const tab = await context.newPage()
        await tab.goto(`${base}/problems`)
        await tab.waitForURL(`**/exams/${id}/description`)
        await tab.close()
        await page.bringToFront()
        // Headless Chromium does not model OS focus changes reliably. Dispatch
        // native DOM focus events; requests still reach the real API unchanged.
        await page.evaluate(() => {
          window.dispatchEvent(new Event('blur'))
          document.dispatchEvent(new Event('visibilitychange'))
          window.dispatchEvent(new Event('focus'))
        })
        await page.keyboard.press('F12')
        await page.getByRole('dialog', { name: '考试环境警告', exact: true }).waitFor()
        await shot(page, `${size}-exam-warning`)
        await dismissWarning(page)
        assert.equal((await api(page, '/problems')).status, 403)
        assert.equal((await api(page, '/v1/tags')).status, 403)
        await page.reload()
        await page.getByRole('navigation', { name: '考试导航' }).waitFor()
        await dismissWarning(page)
        assert.equal((await api(page, '/me/active-exam')).data.active, true)
      }
      await page.getByRole('button', { name: '交卷', exact: true }).click()
      await page.locator('.el-message-box').getByRole('button', { name: /^(确认交卷|确定|确认|OK)$/ }).click()
      await page.waitForURL(url => !url.pathname.startsWith('/exams/'))
      assert.equal((await api(page, '/problems')).status, 200)
      assert.equal((await api(page, `/exams/${id}/enter`, 'POST')).status, 403)
    }
    for (const mode of ['invitation', 'application', 'open']) {
      if (mode === 'invitation') {
        await page.goto(`${base}/teams/join`)
        await page.getByPlaceholder('输入管理员提供的邀请码').fill('invalid2026')
        await page.getByRole('button', { name: '查找团队', exact: true }).click()
        await page.getByText('邀请码无效或已失效，请向团队管理员确认', { exact: true }).waitFor()
        await shot(page, `${size}-team-invalid-invitation`)
        await page.getByPlaceholder('输入管理员提供的邀请码').fill(' examjoin2026 ')
        await page.getByRole('button', { name: '查找团队', exact: true }).click()
      } else await page.goto(`${base}/teams/qa-${mode}/join`)
      await page.getByRole('heading', { name: `加入 验收团队 ${mode}` }).waitFor()
      await shot(page, `${size}-team-${mode}-join`)
      if (mode === 'invitation') {
        assert.equal(await page.locator('.join-panel input').count(), 0, 'invitation must not require a link or a second code entry')
      }
      await page.getByRole('button', { name: mode === 'application' ? '提交申请' : '确认加入', exact: true }).click()
      if (mode === 'application') {
        await page.getByText('申请审核中', { exact: true }).waitFor()
        await shot(page, `${size}-team-application-pending`)
        const applications = (await api(teacher, '/teams/2/applications')).data
        const application = applications.find(item => item.user_id === userIndex + 1)
        assert(application)
        assert.equal((await api(teacher, `/teams/2/applications/${application.id}`, 'PUT', { action: 'approve' })).status, 200)
        await page.getByRole('button', { name: '刷新状态', exact: true }).click()
      }
      await page.waitForURL(`**/teams/qa-${mode}/contests`)
      const dto = await api(page, `/teams/qa-${mode}`)
      assert.equal(dto.data.joined, true)
      assert.equal(dto.data.can_manage, false)
      assert.equal(dto.data.join_code, undefined)
    }
    await context.close()
  }
  await teacher.goto(`${base}/admin/exams/1/activity`)
  await teacher.getByRole('heading', { name: /行为日志/ }).waitFor()
  await teacher.getByText('进入考试', { exact: true }).first().waitFor()
  await shot(teacher, '1440-exam-activity')
  await teacher.setViewportSize({ width: 390, height: 844 })
  await shot(teacher, '390-exam-activity')
  await teacher.setViewportSize({ width: 1440, height: 1100 })
  const logs = await api(teacher, '/exams/1/activity?page_size=100')
  assert.equal(logs.status, 200)
  for (const event of ['enter', 'finish', 'access_blocked', 'devtools_shortcut', 'window_leave', 'window_return']) assert(logs.data.items.some(row => row.event === event), event)
  assert(logs.data.items.filter(row => row.event === 'enter').every(row => row.meta.login_at), 'enter must link actual login')
  await teacher.goto(`${base}/contest/1/problems/B`)
  await teacher.getByRole('heading', { name: 'B.特殊扫雷', level: 1, exact: true }).waitFor()
  await shot(teacher, '1440-team-contest-problem')
  await teacher.setViewportSize({ width: 390, height: 844 })
  await shot(teacher, '390-team-contest-problem')
  assert.deepEqual(errors, [], 'browser runtime errors')
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ result: 'passed', source: 'real local PostgreSQL + Redis + API; seeded evaluation results, no worker', checks, errors }, null, 2))
  console.log(`PASSED ${checks.length} screenshots/checkpoints`)
} catch (error) {
  for (const context of browser.contexts()) for (const [index, page] of context.pages().entries()) await page.screenshot({ path: path.join(output, `failure-${index}-${Date.now()}.png`), fullPage: true }).catch(() => {})
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ result: 'failed', checks, errors, error: String(error) }, null, 2))
  throw error
} finally { await browser.close() }
