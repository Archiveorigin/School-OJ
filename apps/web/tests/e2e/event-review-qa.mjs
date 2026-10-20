// Real local API/SQL/Redis fixture only; this script never targets production.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
const base = 'http://127.0.0.1:4179', api = 'http://127.0.0.1:18089', output = path.resolve('../../tmp/lifecycle-browser-qa')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true }), checks = [], errors = []
const info = await (await fetch(api + '/__qa/info')).json()
const imageURL = 'https://cdn.luogu.com.cn/upload/image_hosting/lcdtefnp.png'
const testFile = (name, text) => ({ name, mimeType: 'text/plain', buffer: Buffer.from(text) })
async function login(email, viewport) {
  const page = await browser.newPage({ viewport }); page.setDefaultTimeout(20000)
  page.on('pageerror', e => errors.push(e.message)); page.on('response', r => { if (r.status() >= 500) errors.push(`${r.status()} ${r.url()}`) })
  await page.goto(base + '/login'); await page.getByPlaceholder('邮箱', { exact: true }).fill(email); await page.getByPlaceholder('密码', { exact: true }).fill('ExamFixture-2026!'); await page.getByRole('button', { name: '登录', exact: true }).click(); await page.waitForURL(url => url.pathname !== '/login'); await page.waitForLoadState('networkidle'); return page
}
async function call(page, method, url, body) {
  const token = await page.evaluate(() => localStorage.getItem('school-oj-token'))
  return page.request.fetch(api + '/api/v1' + url, { method, headers: { Authorization: `Bearer ${token}` }, data: body })
}
async function response(page, method, ending, action, expected = 200) {
  const pending = page.waitForResponse(r => r.request().method() === method && new URL(r.url()).pathname.endsWith(ending))
  const [r] = await Promise.all([pending, action()]); assert.equal(r.status(), expected, await r.text()); return r.json()
}
async function confirm(page) { await page.locator('.el-message-box__btns .el-button--primary').click() }
async function review(page, kind, id, label) { return response(page, 'POST', `/event-papers/${kind}/${id}/review`, async () => { await page.getByRole('button', { name: label, exact: true }).click(); await confirm(page) }) }
async function shot(page, name) { await page.screenshot({ path: path.join(output, name + '.png'), fullPage: true }); assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 2), `${name}: horizontal overflow`) }
function localDate(milliseconds) { const d = new Date(milliseconds), pad = v => String(v).padStart(2, '0'); return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}` }
try {
  for (const [size, viewport] of [['desktop', { width: 1440, height: 1100 }], ['mobile', { width: 390, height: 844 }]]) {
    const author = await login('exam-teacher-2@example.test', viewport), admin = await login('exam-admin-3@example.test', viewport), student = await login('exam-student-0@example.test', viewport)
    for (const kind of ['exam', 'contest']) {
      const title = `整场审核验收 ${kind} ${size}`; let id
      if (kind === 'exam') {
        await author.goto(base + `/admin/exams/new?course_id=${info.course_id}`)
        await author.getByPlaceholder('算法设计期中考试').fill(title)
        await author.getByPlaceholder('请选择未来开始时间').fill(localDate(Date.now()+3600000)); await author.getByPlaceholder('请选择未来开始时间').press('Enter')
        await author.getByPlaceholder('请选择结束时间').fill(localDate(Date.now()+7200000)); await author.getByPlaceholder('请选择结束时间').press('Enter')
        const created = await response(author, 'POST', '/exams', () => author.getByRole('button', { name: '创建草稿并组题', exact: true }).click(), 201); id = created.id
      } else {
        const created = await call(author, 'POST', '/teams/1/contests', { title, description: '比赛附件与新题审核验收', starts_at: new Date(Date.now()+3600000).toISOString(), duration_minutes: 60, scoring_rule: 'ioi' }); assert.equal(created.status(), 201, await created.text()); id = (await created.json()).id
        await author.goto(base + `/contest/${id}/description`)
        await author.getByRole('button', {name: '组题', exact: true}).click()
      }
      const paperURL = `/event-papers/${kind}/${id}`, liveURL = kind === 'exam' ? `/exams/${id}/description` : `/contest/${id}/description`
      await author.getByRole('button', { name: '新增题目', exact: true }).first().click()
      const editor = author.locator('.problem-editor')
      await editor.getByLabel('题目标题', { exact: true }).fill('本场新题')
      await editor.getByLabel('题目描述', { exact: true }).fill(`输出输入的两倍。\n\n![示意图](${imageURL})`)
      await editor.getByLabel('输入格式', { exact: true }).fill('一个整数。')
      await editor.getByLabel('输出格式', { exact: true }).fill('输出这个整数的两倍。')
      await editor.getByLabel('样例 1 输入', { exact: true }).fill('1')
      await editor.getByLabel('样例 1 输出', { exact: true }).fill('2')
      await editor.locator('input[type=file]').last().setInputFiles([testFile('first.in', '1\n'), testFile('first.out', '2\n'), testFile('second.in', '3\n' + ' '.repeat(100*1024)), testFile('second.out', '6\n'), testFile('large.in', '4\n' + ' '.repeat(1024*1024+12)), testFile('large.out', '8\n')])
      await editor.locator('.files li').filter({hasText: 'second.in'}).getByRole('button', {name: '预览', exact: true}).click()
      await author.getByRole('dialog').getByText('仅显示前 64 KiB，完整文件将原样上传。').waitFor()
      assert.equal((await author.locator('.file-preview').innerText()).length, 65536)
      await author.getByRole('dialog').getByRole('button', {name: 'Close this dialog'}).click()
      await author.getByRole('dialog').waitFor({state:'hidden'})
      await shot(author, `${size}-${kind}-editor`)
      const problem = await response(author, 'POST', paperURL + '/problems', () => editor.getByRole('button', { name: '保存题目', exact: true }).click())
      await editor.waitFor({ state: 'hidden' })
      await author.waitForFunction(() => [...document.querySelectorAll('.paper-problem img')].some(i => i.complete && i.naturalWidth > 0))
      await author.getByRole('tab', {name: '本场附件', exact: true}).click()
      const file = await response(author, 'POST', paperURL + '/attachments', () => author.locator('.event-attachments input[type=file]').setInputFiles(testFile('本场说明.txt', '本场授权附件\n')), 201)
      await author.locator('.event-attachments').getByText('本场说明.txt', { exact: false }).waitFor()
      await author.getByRole('tab', {name: '题目预览', exact: true}).click()
      await author.getByRole('button', { name: '编辑题目', exact: true }).click()
      assert.equal(await editor.getByLabel('输入格式', {exact:true}).inputValue(), '一个整数。')
      await editor.locator('.el-form-item').filter({ hasText: '测试点处理' }).locator('.el-select').click()
      await author.getByRole('option', { name: '按名称合并，保留未上传的测试点及原权重', exact: true }).click()
      await editor.locator('input[type=file]').last().setInputFiles([testFile('first.in', '2\n'), testFile('first.out', '4\n')])
      await response(author, 'PUT', paperURL + `/problems/${problem.id}`, () => editor.getByRole('button', { name: '保存题目', exact: true }).click())
      await editor.waitFor({ state: 'hidden' })
      const paper = await (await call(author, 'GET', paperURL)).json(); assert.equal(paper.problems[0].case_count, 3); assert.notEqual(paper.problems[0].problem_version_id, problem.current_version_id)
      assert.ok(paper.problems[0].statement_sections)
      assert.ok(!paper.problems[0].problem.statement.includes('## 说明'))
      await author.getByRole('combobox', {name:'输入题号或标题'}).fill('T001')
      await author.getByRole('option', {name: 'T001 · 特殊扫雷', exact:true}).click()
      await response(author, 'PUT', paperURL + '/problems', () => author.getByRole('button', {name:'添加已有题目', exact:true}).click())
      await shot(author, `${size}-${kind}-authoring`)
      await review(author, kind, id, '提交整场审核')
      assert.equal((await call(student, 'GET', paperURL + `/attachments/${file.id}`)).status(), 403)
      await admin.goto(base + `/admin/competitions/${kind}/${id}`)
      await admin.waitForFunction(() => [...document.querySelectorAll('.paper-problem img')].some(i => i.complete && i.naturalWidth > 0))
      const zipDownload = admin.waitForEvent('download'); await admin.getByRole('button', { name: '下载审核题包', exact: true }).click(); assert.ok((await zipDownload).suggestedFilename().endsWith('.zip'))
      await shot(admin, `${size}-${kind}-admin-review`)
      await review(admin, kind, id, '批准整场')
      assert.equal((await call(student, 'GET', paperURL + `/attachments/${file.id}`)).status(), 403)
      await student.goto(base + liveURL)
      await student.locator('.event-nav').waitFor()
      assert.equal(await student.locator('.event-nav a').count(), 1)
      assert.equal(await student.getByRole('button', {name:'附件',exact:true}).count(), 0)
      assert.equal(await student.getByRole('button', {name:'组题',exact:true}).count(), 0)
      await shot(student, `${size}-${kind}-upcoming-description`)
      await fetch(api + `/__qa/clock/${kind}/${id}/start`, { method: 'POST' })
      await student.goto(base + liveURL)
      if (kind === 'exam') await student.getByRole('button', { name: '确认进入', exact: true }).click()
      await student.getByRole('button', {name:'附件',exact:true}).click()
      await student.locator('.event-attachments').getByText('本场说明.txt', { exact: false }).waitFor()
      const download = student.waitForEvent('download'); await student.locator('.event-attachments').getByRole('button', { name: '下载', exact: true }).click(); assert.equal((await download).suggestedFilename(), '本场说明.txt')
      await shot(student, `${size}-${kind}-participant-attachment`)
      await fetch(api + `/__qa/clock/${kind}/${id}/end`, { method: 'POST' })
      await student.goto(base + liveURL)
      await student.getByText('本场已截止，现为只读状态。题目与附件仍可查看，等待管理员确认完结。', {exact:true}).waitFor()
      assert.equal(await student.locator('.event-attachments').count(), 0)
      assert.equal((await call(student, 'GET', paperURL + `/attachments/${file.id}`)).status(),200)
      const beforeCompletion = await (await call(student,'GET','/event-archives')).json()
      assert.ok(!beforeCompletion.items.some(e=>e.kind===kind && e.id===id))
      await student.goto(base + (kind==='exam'?`/exams/${id}/records`:`/contest/${id}/records`))
      await student.getByRole('heading', {name: '本人提交', exact: true}).waitFor()
      await shot(student, `${size}-${kind}-closed-records`)
      await admin.reload()
      await review(admin, kind, id, '确认完结')
      await student.goto(base + liveURL)
      await student.waitForURL('**/competitions/' + kind + '/' + id)
      await student.getByRole('heading', { name: title, exact: true }).waitFor()
      await student.getByRole('tab', { name: '题目', exact: true }).click()
      await student.waitForFunction(() => [...document.querySelectorAll('.description img')].some(i => i.complete && i.naturalWidth > 0))
      assert.equal(await student.getByRole('button', { name: '提交代码', exact: true }).count(), 0)
      await shot(student, `${size}-${kind}-archive-problem`)
      await student.getByRole('tab', { name: '排名', exact: true }).click(); await shot(student, `${size}-${kind}-archive-ranking`)
      await student.goto(base + '/competitions'); await student.locator(size==='mobile'?'.mobile-list h2':'.desktop-table .event-link').filter({hasText:title}).first().waitFor(); assert.equal(await student.getByText('不应归档的过期草稿', { exact: true }).count(), 0)
      await shot(student, `${size}-${kind}-archive-list`)
      const deletion = await call(author, 'DELETE', kind === 'exam' ? `/exams/${id}` : `/contests/${id}`); assert.equal(deletion.status(), 200)
      assert.equal((await call(student, 'GET', paperURL + `/attachments/${file.id}`)).status(), 404)
      const archives = await (await call(student, 'GET', '/event-archives')).json(); assert.ok(!archives.items.some(e => e.kind === kind && e.id === id))
      checks.push(`${size}/${kind}: create, new question, image, partial merge, attachment, whole-paper approval, prestart denial, participant download, archive, deletion`)
    }
    await author.close(); await admin.close(); await student.close()
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ checks, errors }, null, 2))
  console.log(JSON.stringify({ checks, errors }, null, 2))
} catch (error) { console.error('Browser errors:', errors); for (const [i, context] of browser.contexts().entries()) { for (const page of context.pages()) { console.error('Page:', page.url(), (await page.locator('body').innerText()).slice(0, 1500)); await page.screenshot({ path: path.join(output, `failure-${i}.png`), fullPage: true }).catch(() => {}) } }; throw error } finally { await browser.close() }
