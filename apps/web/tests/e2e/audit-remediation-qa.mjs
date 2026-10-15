// Destructive fixture checks are confined to the loopback-only seeded candidate.
import assert from 'node:assert/strict'
import { chromium } from 'playwright'
import { mkdir, writeFile } from 'node:fs/promises'

const base = process.env.QA_BASE_URL || 'http://127.0.0.1:4177'
assert.equal(base, 'http://127.0.0.1:4177', 'Use only the isolated seeded candidate')
const output = '../../tmp/audit-functional-qa'
await mkdir(output, { recursive: true })
let token = ''
async function api(path, method = 'GET', body, status = 200) {
  const response = await fetch(`${base}/api/v1${path}`, { method, headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) })
  const result = await response.json()
  assert.equal(response.status, status, `${path}: ${JSON.stringify(result)}`)
  return result
}
const auth = await api('/auth/login', 'POST', { email: 'admin@school.local', password: 'password' })
token = auth.token
const problems = await api('/problems')
const problem = problems.find(p => p.title === 'A + B Problem')
assert.ok(problem, 'seeded sum problem is required')
const checks = []
for (const [language, source_code] of Object.entries({
  c: '#include <stdio.h>\nint main(){int a,b;scanf("%d%d",&a,&b);printf("%d\\n",a+b);}',
  cpp: '#include <iostream>\nint main(){int a,b;std::cin>>a>>b;std::cout<<a+b<<std::endl;}',
  python: 'a,b=map(int,input().split()); print(a+b)',
  java: 'import java.util.*; public class Main { public static void main(String[] a){ Scanner s=new Scanner(System.in); System.out.println(s.nextInt()+s.nextInt()); }}',
})) {
  const submission = await api('/submissions', 'POST', { problem_id: problem.id, language, source_code, is_public: false }, 201)
  const response = await fetch(`${base}/api/v1/submissions/${submission.id}/events`, { headers: { Authorization: `Bearer ${token}` }, signal: AbortSignal.timeout(60000) })
  assert.equal(response.status, 200)
  const reader = response.body.getReader(), decoder = new TextDecoder()
  let buffer = '', terminal
  try {
    while (!terminal) {
      const { value, done } = await reader.read()
      buffer += decoder.decode(value || new Uint8Array(), { stream: !done })
      let boundary
      while ((boundary = buffer.indexOf('\n\n')) >= 0) {
        const frame = buffer.slice(0, boundary); buffer = buffer.slice(boundary + 2)
        const data = frame.split('\n').find(line => line.startsWith('data:'))
        if (!data || !frame.includes('event: status')) continue
        const state = JSON.parse(data.slice(5).trim())
        if (!['queued', 'running'].includes(state.status)) terminal = state
      }
      if (done) break
    }
  } finally { await reader.cancel() }
  assert.equal(terminal?.status, 'accepted', `${language} did not pass`)
  assert.equal(terminal?.score, 100)
  checks.push(`API → Redis → ${language} sandbox → database → SSE`)
}
const team = await api('/teams', 'POST', { name: '审查验收团队', slug: `audit-${Date.now()}`, visibility: 'public', join_mode: 'open' }, 201)
const listed = (await api('/teams?scope=mine')).find(item => item.id === team.id)
assert.equal(listed.member_count, 1)
assert.equal(listed.my_role, 'owner')
assert.equal(listed.owner_name, auth.user.name)
assert.equal(listed.can_manage, true)
checks.push('batched team directory preserves ownership, roles and counts')
const browser = await chromium.launch({ headless: true })
try {
  for (const width of [1440, 390]) {
    const page = await browser.newPage({ viewport: { width, height: width === 390 ? 844 : 1100 } })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    page.on('response', response => { if (response.status() >= 500) errors.push(`${response.status()} ${response.url()}`) })
    await page.addInitScript(auth => {
      localStorage.setItem('school-oj-token', auth.token)
      localStorage.setItem('school-oj-user', JSON.stringify(auth.user))
    }, auth)
    for (const route of [`/problems/${problem.id}`, '/teams', '/submissions']) {
      await page.goto(base + route, { waitUntil: 'networkidle' })
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1), false)
      assert.deepEqual(errors, [])
      await page.screenshot({ path: `${output}/${width}-${route.replaceAll('/', '-')}.png`, fullPage: true })
      checks.push(`${width}px ${route}`)
    }
    await page.close()
  }
} finally { await browser.close() }
await writeFile(`${output}/results.json`, JSON.stringify({ checks }, null, 2))
console.log(checks.join('\n'))
