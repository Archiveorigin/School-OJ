import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'

const base = 'http://127.0.0.1:4180'
const output = path.resolve('../../tmp/authorization-qa')
await mkdir(output, { recursive: true })
const info = await (await fetch('http://127.0.0.1:18090/__qa/info')).json()
const browser = await chromium.launch({ headless: true })
const checks = [], errors = []
async function api(page, resource, method='GET', data) {
  return page.evaluate(async ({resource, method, data}) => {
    const response=await fetch('/api/v1'+resource,{method,headers:{Authorization:'Bearer '+localStorage.getItem('school-oj-token'),'Content-Type':'application/json'},body:data ? JSON.stringify(data):undefined})
    return {status:response.status,body:await response.json()}
  },{resource,method,data})
}
async function login(context,email) {
  const page=await context.newPage()
  page.on('pageerror',error=>errors.push(error.message))
  await page.goto(base+'/login')
  await page.getByPlaceholder('邮箱',{exact:true}).fill(email)
  await page.getByPlaceholder('密码',{exact:true}).fill('ExamFixture-2026!')
  await page.getByRole('button',{name:'登录',exact:true}).click()
  await page.waitForURL(url=>url.pathname!='/login')
  return page
}
async function shot(page,name) {
  await page.locator('.el-loading-mask').first().waitFor({state:'hidden'})
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),name+': overflow')
  await page.screenshot({path:path.join(output,name+'.png'),fullPage:true,animations:'disabled'})
  checks.push(name);console.log('PASS',name)
}
try {
  for (const viewport of [{width:1440,height:1100},{width:390,height:844}]) {
    const tag=viewport.width===1440?'desktop':'mobile'
    const context=await browser.newContext({viewport})
    const admin=await login(context,'exam-admin-3@example.test')
    await admin.goto(base+'/admin/users/permissions')
    await admin.getByRole('heading',{name:'角色与权限范围'}).waitFor()
    await admin.getByText(/查看能力清单/).click()
    await admin.getByRole('cell',{name:'跨团队管理',exact:true}).waitFor()
    assert.equal((await api(admin,'/authorization/policies')).body.routes.length,220)
    await shot(admin,tag+'-permission-registry')
    await admin.goto(base+'/teams/authorization-qa/contests')
    await admin.getByRole('button',{name:'创建比赛',exact:true}).waitFor()
    await shot(admin,tag+'-admin-team-content')
    await context.close()

    const teaching=await browser.newContext({viewport})
    const assistant=await login(teaching,'exam-student-1@example.test')
    await assistant.goto(base+'/admin/courses')
    await assistant.getByRole('button',{name:'成员',exact:true}).first().waitFor()
    assert.equal(await assistant.getByRole('button',{name:'编辑',exact:true}).count(),0)
    assert.equal(await assistant.getByRole('button',{name:'加班级',exact:true}).count(),0)
    const courses=await api(assistant,'/courses');assert(courses.body[0].can_manage&&!courses.body[0].can_admin)
    assert.equal((await api(assistant,'/authorization/policies')).status,403)
    await shot(assistant,tag+'-course-assistant')
    await assistant.goto(base+'/admin/users/permissions')
    await assistant.waitForURL(url=>url.pathname==='/')
    await assistant.goto(base+'/teams/authorization-qa/contests')
    await assistant.getByText('权限验收团队',{exact:true}).first().waitFor()
    assert.equal(await assistant.getByRole('button',{name:'创建比赛',exact:true}).count(),0)
    await shot(assistant,tag+'-team-member')
    await teaching.close()
  }
  const studentContext=await browser.newContext({viewport:{width:1440,height:1100}})
  const student=await login(studentContext,'exam-student-0@example.test')
  const before=await api(student,'/me');assert(before.body.permissions.includes('problem:author'))
  const operatorContext=await browser.newContext()
  const operator=await login(operatorContext,'exam-admin-3@example.test')
  assert.equal((await api(operator,`/users/${info.student_id}/permissions/problem_author`,'PUT',{enabled:false,reason:'隔离权限撤销验收'})).status,200)
  const after=await api(student,'/me');assert(!after.body.permissions.includes('problem:author'));assert(after.body.perm_version>before.body.perm_version)
  assert.equal((await api(student,'/problems/parse-markdown','POST',{})).status,403)
  await student.reload()
  assert(!(await student.evaluate(()=>JSON.parse(localStorage.getItem('school-oj-user')).permissions)).includes('problem:author'))
  checks.push('old-token-revocation')
  await studentContext.close();await operatorContext.close()
  assert.deepEqual(errors,[])
  await writeFile(path.join(output,'result.json'),JSON.stringify({passed:true,checks,errors},null,2))
} finally { await browser.close() }
