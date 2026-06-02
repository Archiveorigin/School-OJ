<template>
  <div class="markdown-body" v-html="rendered"></div>
</template>

<script setup lang="ts">
import katex from 'katex'
import MarkdownIt from 'markdown-it'
import { computed } from 'vue'
import { problemAssetUrl } from '../api/client'
import 'katex/dist/katex.min.css'

const props = defineProps<{ source?: string | null; problemId?: number; assetUrls?: Record<string, string> }>()

const md = new MarkdownIt({
  html: false,
  linkify: false,
  typographer: false,
  breaks: true
})

md.inline.ruler.before('escape', 'math_inline', (state: any, silent: boolean) => {
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

md.block.ruler.before('fence', 'math_block', (state: any, startLine: number, endLine: number, silent: boolean) => {
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

md.renderer.rules.math_inline = (tokens, idx) => renderMath(tokens[idx].content, false)
md.renderer.rules.math_block = (tokens, idx) => `<div class="math-block">${renderMath(tokens[idx].content, true)}</div>`
md.renderer.rules.image = (tokens, idx, options, env, self) => {
  const token = tokens[idx]
  const src = token.attrGet('src') || ''
  const resolved = resolveImage(src)
  if (resolved) token.attrSet('src', resolved)
  token.attrSet('loading', 'lazy')
  token.attrSet('decoding', 'async')
  return self.renderToken(tokens, idx, options)
}

function renderMath(source: string, displayMode: boolean) {
  try {
    return katex.renderToString(source, {
      displayMode,
      throwOnError: false,
      strict: false,
      trust: false
    })
  } catch {
    return md.utils.escapeHtml(source)
  }
}

const rendered = computed(() => md.render(props.source || ''))

function resolveImage(src: string) {
  if (!src || src.includes('://') || src.startsWith('data:') || src.startsWith('blob:') || src.startsWith('/') || src.startsWith('#')) {
    return ''
  }
  if (props.assetUrls?.[src]) return props.assetUrls[src]
  if (props.problemId && src.startsWith('assets/')) return problemAssetUrl(props.problemId, src)
  return ''
}
</script>

<style scoped>
.markdown-body {
  color: var(--text);
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(pre),
.markdown-body :deep(table),
.markdown-body :deep(blockquote) {
  margin: 0 0 12px;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3) {
  margin: 18px 0 10px;
  line-height: 1.3;
}

.markdown-body :deep(pre) {
  overflow: auto;
  padding: 12px;
  border-radius: 8px;
  background: #111827;
  color: #f9fafb;
}

.markdown-body :deep(code) {
  padding: 2px 5px;
  border-radius: 5px;
  background: rgba(15, 23, 42, 0.08);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.markdown-body :deep(pre code) {
  padding: 0;
  background: transparent;
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: 8px;
  border: 1px solid var(--border);
  text-align: left;
}

.markdown-body :deep(blockquote) {
  padding: 8px 12px;
  border-left: 4px solid var(--accent);
  background: rgba(10, 94, 166, 0.06);
}

.markdown-body :deep(img) {
  display: block;
  max-width: 100%;
  max-height: 520px;
  margin: 12px 0;
  border-radius: 8px;
  object-fit: contain;
  border: 1px solid var(--border);
  background: var(--surface-strong);
}

.math-block {
  overflow-x: auto;
  margin: 12px 0;
}
</style>
