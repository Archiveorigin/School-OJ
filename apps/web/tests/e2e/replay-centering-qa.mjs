import { chromium } from 'playwright';
import assert from 'node:assert/strict';
const browser = await chromium.launch();
try {
 for (const viewport of [{width:1440,height:1100},{width:390,height:844}]) {
  const page = await browser.newPage({viewport,reducedMotion:'reduce'});
  const rows = Array.from({length:24},(_,i)=>({user_id:i+1,name:`选手${String(i+1).padStart(2,'0')}`,rank:1,unranked:i%2===0,solved:0,total_score:0,penalty_minutes:0,submission_count:1,problems:[{problem_id:1,status:'wrong_answer',attempts:1,submission_count:1,best_score:0}]}));
  const packet={schema_version:1,kind:'contest-replay',contest:{scoring_rule:'acm',duration_minutes:120},problems:[{problem_id:1,label:'A'}],initial_rows:rows,final_rows:rows,events:[]};
  await page.route('**/api/v1/**',route=>route.fulfill({json:route.request().url().includes('/ranking/replay')?packet:[]}));
  await page.goto('http://127.0.0.1:4183/tests/e2e/replay-fullscreen.html');
  await page.getByRole('button',{name:'赛后滚榜',exact:true}).click();
  const surface=page.locator('[data-replay-surface]');
  await page.locator('[data-reveal-focus="true"]').waitFor();
  async function check() {
   const geometry=await surface.evaluate(el=>{
    const row=el.querySelector('[data-reveal-focus="true"]');
    const r=row.getBoundingClientRect(),v=el.getBoundingClientRect();
    return {id:Number(row.dataset.rowId),delta:r.top+r.height/2-v.top-el.clientHeight/2,top:el.scrollTop};
   });
   assert(geometry.top<1 || Math.abs(geometry.delta)<4,JSON.stringify({viewport,...geometry}));
   return geometry;
  }
  await page.waitForTimeout(300);
  assert.equal((await check()).id,24);
  for(let i=0;i<23;i++) {
   await surface.press('ArrowRight');
   await page.waitForTimeout(250);
   await check();
  }
  assert.equal((await check()).id,1);
  console.log(`Centered replay and top boundary passed ${viewport.width}x${viewport.height}`);
  await page.close();
 }
} finally { await browser.close(); }
