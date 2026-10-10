// Read-only browser checks; works against the isolated candidate and production.
import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'

const base = process.env.QA_BASE_URL || 'http://127.0.0.1:4177'
const hardened = process.env.QA_EXPECT_HARDENING === '1'
const output = path.resolve('../../tmp', process.env.QA_OUTPUT || 'performance-security-qa')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true })
const results = []
try {
  for (const width of [1440, 390]) {
    for (const route of ['/', '/login', '/problems']) {
      const page = await browser.newPage({ viewport: { width, height: width === 390 ? 844 : 1100 } })
      const errors = []
      const warnings = []
      page.on('pageerror', error => errors.push(error.message))
      page.on('console', message => {
        if (base.startsWith('http:') && message.text().startsWith('The Cross-Origin-Opener-Policy header has been ignored')) {
          warnings.push(message.text())
          return
        }
        if (message.type() === 'error' || /Failed to resolve (component|directive)/.test(message.text())) errors.push(message.text())
      })
      page.on('response', response => { if (response.status() >= 500) errors.push(`${response.status()} ${response.url()}`) })
      await page.addInitScript(() => {
        window.__longTasks = []
        window.__lcp = 0
        window.__cls = 0
        new PerformanceObserver(list => { window.__lcp = list.getEntries().at(-1).startTime }).observe({ type: 'largest-contentful-paint', buffered: true })
        new PerformanceObserver(list => { for (const x of list.getEntries()) if (!x.hadRecentInput) window.__cls += x.value }).observe({ type: 'layout-shift', buffered: true })
        new PerformanceObserver(list => window.__longTasks.push(...list.getEntries().map(x => x.duration))).observe({ type: 'longtask', buffered: true })
      })
      const response = await page.goto(base + route, { waitUntil: 'networkidle' })
      assert.equal(response.status(), 200)
      assert.equal(await page.locator('el-button, el-input, el-table').count(), 0, 'unresolved component')
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `overflow ${route} ${width}`)
      const metrics = await page.evaluate(() => {
        const resources = performance.getEntriesByType('resource')
        const scripts = resources.filter(x => new URL(x.name).pathname.endsWith('.js'))
        const navigation = performance.getEntriesByType('navigation')[0]
        return { ttfbMs: navigation.responseStart, domContentLoadedMs: navigation.domContentLoadedEventEnd, lcpMs: window.__lcp, cls: window.__cls,
          scriptCount: scripts.length, scriptDecodedBytes: scripts.reduce((n, x) => n + x.decodedBodySize, 0),
          scriptEncodedBytes: scripts.reduce((n, x) => n + x.encodedBodySize, 0), longTasks: window.__longTasks,
          scripts: scripts.map(x => new URL(x.name).pathname) }
      })
      assert.equal(metrics.scripts.some(x => /monaco|editor\.worker/.test(x)), false, 'editor eagerly loaded')
      if (hardened) {
        const headers = response.headers()
        for (const name of ['content-security-policy', 'x-frame-options', 'x-content-type-options']) assert.ok(headers[name], `missing ${name}`)
        const asset = await page.request.get(base + metrics.scripts[0])
        assert.match(asset.headers()['cache-control'], /immutable/)
        assert.equal(asset.headers()['content-encoding'], 'gzip')
        assert.ok(asset.headers()['content-security-policy'], 'asset lost inherited security headers')
        const missing = await page.request.get(base + '/assets/definitely-missing.js')
        assert.equal(missing.status(), 404)
        assert.doesNotMatch(missing.headers()['cache-control'] || '', /immutable/)
        const secret = await page.request.get(base + '/.env')
        assert.equal(secret.status(), 404)
        const api = await page.request.get(base + '/api/problems/catalog?page_size=20')
        assert.equal(api.status(), 200)
        assert.match(api.headers()['cache-control'], /no-store/)
      }
      assert.deepEqual(errors, [])
      await page.screenshot({ path: path.join(output, `${width}-${route.replaceAll('/', '') || 'home'}.png`), fullPage: true })
      results.push({ width, route, warnings, ...metrics })
      await page.close()
    }
  }
  await writeFile(path.join(output, 'results.json'), JSON.stringify({ base, hardened, results }, null, 2))
  console.log(JSON.stringify(results.map(({ scripts, longTasks, ...result }) => ({ ...result, longTaskCount: longTasks.length })), null, 2))
} finally { await browser.close() }
