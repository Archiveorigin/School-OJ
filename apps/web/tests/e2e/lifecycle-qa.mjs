import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir } from 'node:fs/promises'
const base='http://127.0.0.1:4176'
const browser=await chromium.launch()
await mkdir('../../tmp/lifecycle-qa',{recursive:true})
try {
 const admin=await browser.newContext({viewport:{width:1440,height:1100}})
 const page=await admin.newPage()
 async function login(page,email) {
  await page.goto(base+'/login')
  await page.getByPlaceholder('邮箱',{exact:true}).fill(email)
  await page.getByPlaceholder('密码',{exact:true}).fill('ExamFixture-2026!')
  await page.getByRole('button',{name:'登录',exact:true}).click()
  await page.waitForURL(u=>u.pathname!='/login')
 }
 await login(page,'exam-admin-3@example.test')
 await page.goto(base+'/admin/users')
 const search=page.getByRole('textbox',{name:'搜索用户'})
 await search.fill('exam-student-0@')
 await page.getByRole('cell',{name:'exam-student-0@example.test',exact:true}).waitFor()
 assert.equal(await page.locator('.el-table__body tbody tr').count(),1)
 await search.fill('does-not-exist')
 await page.getByText('No Data',{exact:true}).or(page.getByText('暂无数据',{exact:true})).waitFor()
 await search.fill('')
 await page.screenshot({path:'../../tmp/lifecycle-qa/users.png',fullPage:true})
 await page.goto(base+'/admin/audit-logs')
 await page.getByRole('cell',{name:'登录',exact:true}).first().waitFor()
 const teacher=await browser.newContext({viewport:{width:1440,height:1100}})
 const tp=await teacher.newPage(); await login(tp,'exam-teacher-2@example.test')
 await tp.goto(base+'/exams/1/description')
 await tp.getByRole('button',{name:'发布公告',exact:true}).click()
 const panel=tp.getByRole('dialog',{name:'发布赛时公告'})
 await panel.getByLabel('公告正文').fill('赛时更正：输入范围为 100，请以本公告为准。')
 await panel.getByLabel('关联题目 ID（选填）').fill('1')
 await panel.getByRole('button',{name:'发布公告',exact:true}).click()
 await panel.waitFor({state:'hidden'})
 await tp.getByText('赛时更正：输入范围为 100，请以本公告为准。',{exact:true}).waitFor()
 for(const width of [1440,390]) {
  await tp.setViewportSize({width,height:width===390?844:1100})
  for(const theme of ['light','dark']) {
   await tp.evaluate(theme=>{document.documentElement.dataset.theme=theme;document.documentElement.classList.toggle('dark',theme==='dark')},theme)
   assert(await tp.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1))
   await tp.screenshot({path:`../../tmp/lifecycle-qa/announcements-${width}-${theme}.png`,fullPage:true})
  }
 }
 console.log('PASS user filters, audit labels, announcement publication, desktop/mobile light/dark layout')
} finally {await browser.close()}
