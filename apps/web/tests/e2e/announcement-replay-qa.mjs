import {chromium} from 'playwright'
import assert from 'node:assert/strict'
import {mkdir} from 'node:fs/promises'
const base='http://127.0.0.1:4179'
await mkdir('../../tmp/replay-qa',{recursive:true})
const browser=await chromium.launch()
try {
 const context=await browser.newContext({viewport:{width:1440,height:1100},acceptDownloads:true})
 const p=await context.newPage();let announcementReads=0;p.on('request',r=>{if(r.method()==='GET'&&r.url().endsWith('/announcements'))announcementReads++});const errors=[];p.on('pageerror',e=>errors.push(e.message))
 const info=await (await context.request.get(base+'/__qa/info')).json()
 const item=info.rankings.find(x=>x.kind==='contest'&&x.rule==='acm')
 await p.goto(base+'/login');await p.getByPlaceholder('邮箱',{exact:true}).fill('exam-teacher-2@example.test');await p.getByPlaceholder('密码',{exact:true}).fill('ExamFixture-2026!');await p.getByRole('button',{name:'登录',exact:true}).click();await p.waitForURL(u=>u.pathname!='/login')
 await context.request.post(`${base}/__qa/clock/contest/${item.id}/start`)
 await p.goto(`${base}/contest/${item.id}/description`)
 await p.getByRole('button',{name:'发布公告',exact:true}).click()
 const dialog=p.getByRole('dialog',{name:'发布赛时公告'})
 await dialog.getByLabel('公告正文').fill('输入范围更正，请查看解释文档。')
 await dialog.locator('input[type=file]').setInputFiles({name:'解释.txt',mimeType:'text/plain',buffer:Buffer.from('边界条件解释')})
 await dialog.getByRole('button',{name:'发布公告',exact:true}).click()
 await dialog.waitFor({state:'hidden'})
 await p.getByRole('button',{name:'解释.txt',exact:true}).waitFor()
 const reads=announcementReads;await p.waitForTimeout(16000);assert.equal(announcementReads,reads,'announcement must not poll')
 for(const width of [1440,390]){await p.setViewportSize({width,height:width===390?844:1100});assert(await p.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));for(const dark of [false,true]) {await p.evaluate(dark=>{document.documentElement.dataset.theme=dark?'dark':'light';document.documentElement.classList.toggle('dark',dark)},dark);await p.screenshot({path:`../../tmp/replay-qa/announcement-${width}-${dark?'dark':'light'}.png`,fullPage:true})}}
 await context.request.post(`${base}/__qa/clock/contest/${item.id}/end`)
 await context.request.post(`${base}/__qa/rankings/contest/${item.id}/resolve`)
 await p.goto(`${base}/contest/${item.id}/scoreboard`)
 await p.getByRole('button',{name:'赛后滚榜',exact:true}).click()
 const replay=p.getByRole('dialog',{name:'赛后滚榜'})
 const download=p.waitForEvent('download');await replay.getByRole('button',{name:'导出滚榜数据包',exact:true}).click();const file=await download;await file.saveAs('../../tmp/replay-qa/contest-replay.json')
 await replay.locator('.scoreboard-core').waitFor()
 await replay.getByRole('button',{name:'自动滚榜',exact:true}).click()
 await p.waitForTimeout(500)
 await replay.getByRole('button',{name:'暂停',exact:true}).click()
 await p.screenshot({path:'../../tmp/replay-qa/replay-mobile.png',fullPage:true})
 assert.deepEqual(errors,[])
 console.log('PASS announcement modal, attachment upload, responsive layouts, real export, replay playback/pause')
} finally {await browser.close()}
