import { chromium } from 'playwright'
import { mkdir, readFile } from 'node:fs/promises'
import path from 'node:path'

const baseURL = process.env.QA_BASE_URL || 'http://127.0.0.1:4174'
const outputDir = path.resolve(process.env.QA_OUTPUT_DIR || '../../tmp/design-qa')
const referenceDir = path.resolve(process.env.QA_REFERENCE_DIR || outputDir)

const admin = { id: 1, email: 'admin@example.com', name: '系统管理员', role: 'admin', can_author: true }
const problems = [
  { id: 1, owner_id: 1, display_code: 'T101', title: '数列分块入门', statement: '# 题目描述\n\n给定一个整数序列，完成区间修改与区间查询。', tags: { labels: ['数据结构', '分块'] }, difficulty: '普及-', time_limit_ms: 1000, memory_limit_mb: 256, output_limit_kb: 1024, progress_status: 'accepted', pass_rate: 73.4, accepted_count: 184, evaluated_count: 251 },
  { id: 2, owner_id: 1, display_code: 'T102', title: '城市道路规划', statement: '# 题目描述\n\n计算连接所有城市的最低建设成本。', tags: { labels: ['图论', '最小生成树'] }, difficulty: '普及/提高-', time_limit_ms: 1500, memory_limit_mb: 256, output_limit_kb: 1024, progress_status: 'attempted', pass_rate: 48.8, accepted_count: 82, evaluated_count: 168 },
  { id: 3, owner_id: 1, display_code: 'T103', title: '区间动态规划', statement: '# 题目描述\n\n求最优合并方案。', tags: { labels: ['动态规划', '区间 DP'] }, difficulty: '提高+/省选-', time_limit_ms: 2000, memory_limit_mb: 512, output_limit_kb: 1024, progress_status: 'unattempted', pass_rate: 31.2, accepted_count: 54, evaluated_count: 173 },
  { id: 4, owner_id: 1, display_code: 'T104', title: '字符串哈希查询', statement: '# 题目描述\n\n回答两个子串是否相同。', tags: { labels: ['字符串', '哈希'] }, difficulty: '普及/提高-', time_limit_ms: 1000, memory_limit_mb: 256, output_limit_kb: 1024, progress_status: 'unattempted', pass_rate: 61.7, accepted_count: 121, evaluated_count: 196 },
  { id: 5, owner_id: 1, display_code: 'T105', title: '网络流基础', statement: '# 题目描述\n\n求有向网络的最大流。', tags: { labels: ['图论', '网络流'] }, difficulty: '提高+/省选-', time_limit_ms: 2500, memory_limit_mb: 512, output_limit_kb: 1024, progress_status: 'attempted', pass_rate: 27.6, accepted_count: 40, evaluated_count: 145 },
  { id: 6, owner_id: 1, display_code: 'T106', title: '矩阵快速幂', statement: '# 题目描述\n\n计算递推数列第 n 项。', tags: { labels: ['数学', '矩阵'] }, difficulty: '普及/提高-', time_limit_ms: 1000, memory_limit_mb: 256, output_limit_kb: 1024, progress_status: 'accepted', pass_rate: 68.5, accepted_count: 137, evaluated_count: 200 },
]

const ticket = {
  id: 42, requester_id: 1, requester: admin, problem_id: 2, problem: problems[1], action: 'replace', status: 'pending', target_scope: 'public', description: '修正道路重边条件并补充边界测试点说明。', attachment_name: 'reference-notes.pdf', resolution_note: '', created_at: '2026-08-31T08:15:00Z', updated_at: '2026-08-31T08:15:00Z',
  impact_summary: { future_exams: 2, pinned_exams: 1, future_contests: 1, pinned_contests: 2, historical_submissions: 386 },
}

const contestProblems = problems.slice(0, 4).map((problem, index) => ({
  id: index + 1, contest_id: 101, problem_id: problem.id, label: String.fromCharCode(65 + index), sort_order: index, problem, submission_status: index === 0 ? 'accepted' : index === 1 ? 'wrong_answer' : '',
}))

function contestDetail(id) {
  const rule = id === 102 ? 'acm' : id === 103 ? 'oi' : 'ioi'
  return {
    contest: { id, team_id: 9, title: rule === 'oi' ? '秋季 OI 模拟赛' : rule === 'acm' ? '程序设计新生赛' : '算法能力挑战赛', description: '请独立完成比赛。比赛期间允许多次提交，具体计分方式以当前赛制为准。\n\n提交前请仔细检查输入输出格式。', starts_at: '2026-08-31T08:00:00Z', ends_at: '2026-08-31T11:00:00Z', duration_minutes: 180, scoring_rule: rule, freeze_enabled: rule !== 'oi', freeze_duration_minutes: 60, state: 'running', status: 'running' },
    team: { id: 9, name: '算法竞赛实验室', slug: 'algorithm-lab' },
    problems: contestProblems.map((item) => ({ ...item, contest_id: id })), can_organize: true, can_submit: true, is_participant: true, can_edit: false, can_delete: false, can_publish: false,
  }
}

function ranking(id) {
  const acm = id === 102
  const rows = [
    { user_id: 11, name: '林清越', solved: acm ? 3 : 2, total_score: acm ? 300 : 356, penalty_minutes: 142, problems: [
      { problem_id: 1, attempts: 1, wrong_attempts: 0, best_score: 100, status: 'accepted', solved_at: '2026-08-31T08:22:00Z', elapsed_minutes: 22 },
      { problem_id: 2, attempts: 3, wrong_attempts: 2, best_score: acm ? 100 : 92, status: 'accepted', solved_at: '2026-08-31T09:06:00Z', elapsed_minutes: 66 },
      { problem_id: 3, attempts: 2, wrong_attempts: acm ? 1 : 2, best_score: acm ? 100 : 84, status: acm ? 'accepted' : 'wrong_answer', solved_at: acm ? '2026-08-31T09:44:00Z' : null, elapsed_minutes: 104 },
      { problem_id: 4, attempts: 1, wrong_attempts: 1, best_score: acm ? 0 : 80, status: 'wrong_answer', elapsed_minutes: 118 },
    ]},
    { user_id: 12, name: '陈知行', solved: 2, total_score: acm ? 200 : 330, penalty_minutes: 96, problems: [
      { problem_id: 1, attempts: 2, wrong_attempts: 1, best_score: 100, status: 'accepted', solved_at: '2026-08-31T08:30:00Z', elapsed_minutes: 30 },
      { problem_id: 2, attempts: 1, wrong_attempts: 0, best_score: 100, status: 'accepted', solved_at: '2026-08-31T08:46:00Z', elapsed_minutes: 46 },
      { problem_id: 3, attempts: 1, wrong_attempts: 1, best_score: acm ? 0 : 72, status: 'wrong_answer', elapsed_minutes: 88 },
      { problem_id: 4, attempts: 2, wrong_attempts: 2, best_score: acm ? 0 : 58, status: 'wrong_answer', elapsed_minutes: 111 },
    ]},
    { user_id: 13, name: '周予安', solved: 1, total_score: acm ? 100 : 274, penalty_minutes: 65, problems: [
      { problem_id: 1, attempts: 1, wrong_attempts: 0, best_score: 100, status: 'accepted', solved_at: '2026-08-31T09:05:00Z', elapsed_minutes: 65 },
      { problem_id: 2, attempts: 2, wrong_attempts: 2, best_score: acm ? 0 : 74, status: 'wrong_answer', elapsed_minutes: 93 },
      { problem_id: 3, attempts: 1, wrong_attempts: 1, best_score: acm ? 0 : 60, status: 'wrong_answer', elapsed_minutes: 103 },
      { problem_id: 4, attempts: 1, wrong_attempts: 1, best_score: acm ? 0 : 40, status: 'wrong_answer', elapsed_minutes: 116 },
    ]},
  ]
  return { contest: contestDetail(id).contest, scoring_rule: acm ? 'acm' : 'ioi', participant_count: rows.length, frozen: true, freeze_duration_minutes: 60, problems: contestProblems.map((item) => ({ problem_id: item.problem_id, label: item.label, title: item.problem.title })), rows }
}

async function mockAPI(page) {
  await page.addInitScript(({ user }) => {
    localStorage.setItem('school-oj-token', 'qa-token')
    localStorage.setItem('school-oj-user', JSON.stringify(user))
  }, { user: admin })
  await page.route('**/api/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (!url.pathname.startsWith('/api/')) {
      await route.continue()
      return
    }
    const pathname = url.pathname.replace(/^\/api/, '')
    let payload = null
    if (pathname === '/me') payload = admin
    else if (pathname === '/problems/catalog') payload = { items: problems, total: problems.length, page: 1, page_size: 20, available_tags: ['动态规划', '图论', '字符串', '数据结构', '数学', '网络流'] }
    else if (pathname === '/problem-change-tickets/eligible-problems') payload = problems
    else if (pathname === '/problem-change-tickets/mine' || pathname === '/problem-change-tickets') payload = [ticket]
    else if (pathname === '/problem-change-tickets/42') payload = ticket
    else if (/^\/problem-change-tickets\/42\/(cancel|reject|apply)$/.test(pathname)) payload = { ...ticket, status: 'completed' }
    else if (/^\/contests\/\d+\/submissions$/.test(pathname)) payload = []
    else if (/^\/contests\/\d+\/ranking$/.test(pathname)) payload = ranking(Number(pathname.split('/')[2]))
    else if (/^\/contests\/\d+$/.test(pathname)) payload = contestDetail(Number(pathname.split('/')[2]))
    else if (pathname === '/submissions/latest') payload = []
    else if (pathname === '/me/active-exam') payload = { exam: null }
    else payload = request.method() === 'GET' ? [] : { ok: true }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(payload) })
  })
}

async function capture(browser, { name, url, viewport, check }) {
  const page = await browser.newPage({ viewport, deviceScaleFactor: 1 })
  const errors = []
  page.on('console', (message) => { if (message.type() === 'error') errors.push(message.text()) })
  page.on('pageerror', (error) => errors.push(error.message))
  await mockAPI(page)
  await page.goto(`${baseURL}${url}`, { waitUntil: 'networkidle' })
  await check?.(page)
  await page.waitForTimeout(400)
  await page.screenshot({ path: path.join(outputDir, `${name}.png`), fullPage: true })
  if (errors.length) throw new Error(`${name} console errors:\n${errors.join('\n')}`)
  await page.close()
}

async function comparison(browser, sourceName, implementationName, outputName) {
  const page = await browser.newPage({ viewport: { width: 1440, height: 2200 }, deviceScaleFactor: 1 })
  const source = await readFile(path.join(referenceDir, sourceName), 'base64')
  const implementation = await readFile(path.join(outputDir, implementationName), 'base64')
  await page.setContent(`<style>html,body{margin:0;background:#111827;color:#fff;font:600 14px system-ui}.board{display:grid;grid-template-columns:1fr 1fr;gap:12px;padding:12px}.item{display:grid;gap:8px}.label{padding:8px 12px;background:#1f2937;border-radius:6px}img{display:block;width:100%;height:auto;background:white}</style><div class="board"><div class="item"><div class="label">参考界面</div><img src="data:image/png;base64,${source}"></div><div class="item"><div class="label">本地实现</div><img src="data:image/png;base64,${implementation}"></div></div>`)
  await page.screenshot({ path: path.join(outputDir, outputName), fullPage: true })
  await page.close()
}

const browser = await chromium.launch({ headless: true })
try {
  await mkdir(outputDir, { recursive: true })
  await capture(browser, { name: 'qa-problem-catalog-desktop', url: '/problems', viewport: { width: 1440, height: 1100 }, check: async (page) => { await page.getByPlaceholder('搜索题号或题目名称').fill('图论'); await page.waitForTimeout(450); if (!(await page.getByText('题库').first().isVisible())) throw new Error('catalog heading missing') } })
  await capture(browser, { name: 'qa-problem-catalog-mobile', url: '/problems', viewport: { width: 390, height: 844 }, check: async (page) => { if (!(await page.getByRole('heading', { name: '城市道路规划' }).isVisible())) throw new Error('mobile catalog item missing') } })
  await capture(browser, { name: 'qa-ticket-author-desktop', url: '/problem-changes/new?action=replace&problem_id=2&target_scope=public', viewport: { width: 1440, height: 1100 }, check: async (page) => { if (!(await page.getByText('描述修改需求').isVisible())) throw new Error('ticket form missing') } })
  await capture(browser, { name: 'qa-ticket-admin-desktop', url: '/admin/problem-authors', viewport: { width: 1440, height: 1100 }, check: async (page) => { await page.getByRole('button', { name: '查看' }).click(); if (!(await page.getByRole('heading', { name: '影响范围' }).isVisible())) throw new Error('ticket impact missing') } })
  await capture(browser, { name: 'qa-contest-problems-desktop', url: '/contest/101/problems', viewport: { width: 1440, height: 1100 }, check: async (page) => { if (!(await page.getByText('城市道路规划').first().isVisible())) throw new Error('contest problem missing') } })
  await capture(browser, { name: 'qa-contest-problem-desktop', url: '/contest/101/problems/B', viewport: { width: 1440, height: 1100 }, check: async (page) => { if (!(await page.getByRole('button', { name: '提交代码' }).isVisible())) throw new Error('contest submit missing') } })
  await capture(browser, { name: 'qa-ioi-scoreboard-desktop', url: '/contest/101/scoreboard', viewport: { width: 1440, height: 1100 }, check: async (page) => { if (!(await page.getByText('排行榜已封榜').isVisible())) throw new Error('freeze banner missing') } })
  await capture(browser, { name: 'qa-acm-scoreboard-desktop', url: '/contest/102/scoreboard', viewport: { width: 1440, height: 1100 }, check: async (page) => { if (!(await page.getByText('ACM 排行榜').isVisible())) throw new Error('ACM ranking missing') } })
  await capture(browser, { name: 'qa-scoreboard-mobile', url: '/contest/102/scoreboard', viewport: { width: 390, height: 844 }, check: async (page) => { if ((await page.locator('.scoreboard-scroll').evaluate((node) => node.scrollWidth <= node.clientWidth))) throw new Error('mobile matrix should scroll horizontally') } })
  await capture(browser, { name: 'qa-oi-description-desktop', url: '/contest/103/description', viewport: { width: 1440, height: 1100 }, check: async (page) => { if (await page.getByRole('link', { name: '排行榜' }).count()) throw new Error('OI must not expose ranking navigation') } })
  await capture(browser, { name: 'qa-legacy-route-desktop', url: '/contest/101#problems', viewport: { width: 1440, height: 1100 }, check: async (page) => { if (new URL(page.url()).pathname !== '/contest/101/problems') throw new Error(`legacy route resolved to ${page.url()}`) } })
  await comparison(browser, 'luogu-problem-list.png', 'qa-problem-catalog-desktop.png', 'qa-compare-problem-catalog.png')
  await comparison(browser, 'luogu-contest-problems.png', 'qa-contest-problems-desktop.png', 'qa-compare-contest-problems.png')
  await comparison(browser, 'luogu-contest-problem.png', 'qa-contest-problem-desktop.png', 'qa-compare-contest-problem.png')
  await comparison(browser, 'luogu-ioi-scoreboard.png', 'qa-ioi-scoreboard-desktop.png', 'qa-compare-ioi-scoreboard.png')
  await comparison(browser, 'luogu-acm-scoreboard.png', 'qa-acm-scoreboard-desktop.png', 'qa-compare-acm-scoreboard.png')
  console.log(JSON.stringify({ result: 'passed', outputDir }))
} finally {
  await browser.close()
}
