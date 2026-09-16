import { chromium } from 'playwright'
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'

const output = path.resolve('../../tmp/components-qa')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true })
const errors = [], checks = []
try {
  for (const mobile of [false, true]) {
    const context = await browser.newContext({ viewport: mobile ? { width: 390, height: 844 } : { width: 1440, height: 1100 }, hasTouch: mobile, isMobile: mobile, permissions: ['clipboard-read', 'clipboard-write'] })
    const page = await context.newPage()
    page.on('pageerror', error => { errors.push(error.message); console.error(error.stack) })
    page.on('console', message => { if (/Could not create web worker|Falling back/.test(message.text())) errors.push(message.text()) })
    await page.goto('http://127.0.0.1:5173/tests/e2e/components-harness.html')
    await page.locator('.monaco-editor').waitFor()
    const viewer = page.locator('.source-viewer'), copy = viewer.getByRole('button', { name: '复制代码' })
    if (!mobile) {
      await page.mouse.move(0, 0)
      await page.waitForFunction(() => getComputedStyle(document.querySelector('.source-action')).opacity === '0')
      await viewer.hover()
    }
    await copy.click()
    assert.equal((await page.evaluate(() => navigator.clipboard.readText())).replace(/\r\n/g, '\n'), (await page.evaluate(() => qa.source.value)).replace(/\r\n/g, '\n'))
    if (!mobile) {
      await page.mouse.move(0, 0)
      await page.waitForFunction(() => getComputedStyle(document.querySelector('.source-action')).opacity === '0')
      await page.keyboard.press('Tab'); await copy.focus()
      await page.waitForFunction(() => getComputedStyle(document.querySelector('.source-action')).opacity === '1')
    }
    await page.getByRole('button', { name: '格式化', exact: true }).click()
    assert.equal(await page.evaluate(() => qa.source.value), 'int main() {\n  return 0;\n}')
    await page.evaluate(() => qa.monaco.editor.getEditors()[0].trigger('test', 'undo', null))
    assert.equal(await page.evaluate(() => qa.source.value), 'int main() {\nreturn 0;\n}')
    await page.evaluate(() => { qa.monaco.editor.getModels()[0].setValue('latest typed source'); qa.language.value = 'python' })
    assert.equal(await page.locator('output').textContent(), 'latest typed source')
    await page.waitForFunction(() => qa.monaco.editor.getModels()[0].getLanguageId() === 'python')
    for (let i = 0; i < 5; i++) {
      await page.evaluate(() => { qa.mounted.value = false })
      await page.waitForFunction(() => qa.monaco.editor.getModels().length === 0)
      await page.evaluate(() => { qa.mounted.value = true })
      await page.locator('.monaco-editor').waitFor()
      assert.equal(await page.evaluate(() => qa.monaco.editor.getModels().length), 1)
    }
    await page.evaluate(() => { document.documentElement.classList.add('dark') })
    await page.waitForFunction(() => document.querySelector('.monaco-editor').classList.contains('vs-dark'))
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1))
    await page.evaluate(() => { qa.language.value = 'cpp'; qa.source.value = 'int main() {\n  // 保留源码、行号和原始空行\n  return 0;\n}' })
    await viewer.hover()
    await page.waitForFunction(() => getComputedStyle(document.querySelector('.source-action')).opacity === '1')
    await page.screenshot({ path: path.join(output, `${mobile ? '390' : '1440'}-components.png`), fullPage: true })
    checks.push(`${mobile ? 'touch' : 'desktop'}: copy visibility/original, format/undo, language, five remounts, dark theme, layout`)
    await context.close()
  }
  assert.deepEqual(errors, [])
  await writeFile(path.join(output, 'result.json'), JSON.stringify({ result: 'passed', checks, errors }, null, 2))
  console.log(checks.join('\n'))
} finally { await browser.close() }
