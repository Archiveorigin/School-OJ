export function editorLanguage(language: string) {
  return ['c', 'cpp', 'java', 'python'].includes(language) ? language : 'plaintext'
}

/** Conservative fallback: indent existing C-family lines without rewriting tokens.
 * Python indentation is syntax; raw strings, text blocks, escaped newlines and Java
 * Unicode escapes require a real parser. Leave these documents byte-for-byte intact.
 */
export function formatIndentation(source: string, language: string): string {
  if (!['c', 'cpp', 'java'].includes(language) || /R"|"""|\\\r?\n|\\u[0-9a-fA-F]{4}/.test(source)) return source
  let depth = 0
  let parentheses = 0
  let blockComment = false
  return source.split(/(\r\n|\n|\r)/).map((line, index) => {
    if (index % 2 || !line.trim()) return line
    const text = line.trimStart()
    if (!blockComment && text.startsWith('#')) return line
    const preserveIndent = blockComment || parentheses > 0
    const level = Math.max(0, depth - (text.startsWith('}') && !blockComment ? 1 : 0))
    let quote = ''
    for (let i = 0; i < text.length; i++) {
      const char = text[i], next = text[i + 1]
      if (blockComment) {
        if (char === '*' && next === '/') { blockComment = false; i++ }
      } else if (quote) {
        if (char === '\\') i++
        else if (char === quote) quote = ''
      } else if (char === '/' && next === '/') break
      else if (char === '/' && next === '*') { blockComment = true; i++ }
      else if (char === '"' || (char === "'" && !/[\w]/.test(text[i - 1] || ''))) quote = char
      else if (char === '{') depth++
      else if (char === '}') depth = Math.max(0, depth - 1)
      else if (char === '(') parentheses++
      else if (char === ')') parentheses = Math.max(0, parentheses - 1)
    }
    return preserveIndent ? line : '  '.repeat(level) + text
  }).join('')
}
