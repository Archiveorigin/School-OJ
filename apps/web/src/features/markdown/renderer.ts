import katex from 'katex'
import MarkdownIt from 'markdown-it'
import type Token from 'markdown-it/lib/token.mjs'

export interface MarkdownEnvironment { assetUrls?: Record<string, string> }

export const markdown = new MarkdownIt({
  html: false,
  linkify: false,
  typographer: false,
  breaks: true
})

markdown.inline.ruler.before('escape', 'math_inline', (state: any, silent: boolean) => {
  if (state.src[state.pos] !== '$' || state.src[state.pos + 1] === '$') return false
  let pos = state.pos + 1
  while ((pos = state.src.indexOf('$', pos)) !== -1) {
    if (state.src[pos - 1] === '\\') {
      pos += 1
      continue
    }
    const content = state.src.slice(state.pos + 1, pos)
    if (!content.trim()) return false
    if (!silent) {
      const token = state.push('math_inline', 'math', 0)
      token.content = content
    }
    state.pos = pos + 1
    return true
  }
  return false
})

markdown.block.ruler.before('fence', 'math_block', (state: any, startLine: number, endLine: number, silent: boolean) => {
  const start = state.bMarks[startLine] + state.tShift[startLine]
  const max = state.eMarks[startLine]
  const firstLine = state.src.slice(start, max).trim()
  if (!firstLine.startsWith('$$')) return false
  let content = firstLine.slice(2)
  let nextLine = startLine
  if (content.trim().endsWith('$$') && content.trim().length > 2) {
    content = content.trim().slice(0, -2)
  } else {
    let found = false
    while (++nextLine < endLine) {
      const lineStart = state.bMarks[nextLine] + state.tShift[nextLine]
      const lineMax = state.eMarks[nextLine]
      const line = state.src.slice(lineStart, lineMax)
      if (line.trim().endsWith('$$')) {
        content += `\n${line.slice(0, line.lastIndexOf('$$'))}`
        found = true
        break
      }
      content += `\n${line}`
    }
    if (!found) return false
  }
  if (silent) return true
  const token = state.push('math_block', 'math', 0)
  token.block = true
  token.content = content.trim()
  token.map = [startLine, nextLine + 1]
  state.line = nextLine + 1
  return true
})

markdown.renderer.rules.math_inline = (tokens, idx) => renderMath(tokens[idx].content, false)
markdown.renderer.rules.math_block = (tokens, idx) => `<div class="math-block">${renderMath(tokens[idx].content, true)}</div>`
markdown.renderer.rules.fence = (tokens, idx) => renderCodeBlock(tokens[idx].content, tokens[idx].info)
markdown.renderer.rules.code_block = (tokens, idx) => renderCodeBlock(tokens[idx].content, '')
markdown.renderer.rules.image = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const src = token.attrGet('src') || ''
  const original = token.attrs?.map((attr) => [...attr] as [string, string]) || null
  const resolved = (env as MarkdownEnvironment | undefined)?.assetUrls?.[src]
  if (resolved) token.attrSet('src', resolved)
  else if (src.startsWith('assets/')) token.attrSet('src', '')
  token.attrSet('loading', 'lazy')
  token.attrSet('decoding', 'async')
  const html = self.renderToken(tokens, idx, options)
  token.attrs = original
  return html
}

const mathCache = new Map<string, string>()
function renderMath(source: string, displayMode: boolean) {
  const key = `${displayMode}:${source}`
  const cached = mathCache.get(key)
  if (cached !== undefined) return cached
  try {
    const html = katex.renderToString(source, {
      displayMode,
      throwOnError: false,
      strict: false,
      trust: false
    })
    if (source.length <= 4096 && html.length <= 65536) {
      if (mathCache.size >= 128) mathCache.delete(mathCache.keys().next().value!)
      mathCache.set(key, html)
    }
    return html
  } catch {
    return markdown.utils.escapeHtml(source)
  }
}

function renderCodeBlock(source: string, info: string) {
  const language = info.trim().split(/\s+/)[0].replace(/[^A-Za-z0-9_-]/g, '')
  const languageClass = language ? ` class="language-${markdown.utils.escapeHtml(language)}"` : ''
  const content = source.replace(/^(?:[ \t]*\r?\n)+/, '')
  return [
    '<div class="markdown-code-block">',
    '<div class="markdown-code-toolbar"><button type="button" class="markdown-copy-button" aria-label="复制代码">复制</button></div>',
    `<pre><code${languageClass}>${markdown.utils.escapeHtml(content)}</code></pre>`,
    '</div>'
  ].join('')
}

export function collectAssetPaths(tokens: Token[]) {
  const paths = new Set<string>()
  const visit = (tokens: any[]) => {
    for (const token of tokens) {
      if (token.type === 'image') {
        const src = token.attrGet?.('src') || ''
        if (src.startsWith('assets/')) paths.add(src)
      }
      if (Array.isArray(token.children)) visit(token.children)
    }
  }
  visit(tokens)
  return [...paths]
}
