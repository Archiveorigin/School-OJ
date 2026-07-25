export async function copyTextToClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {
      // HTTP pages and restricted browser permissions can reject the modern API.
    }
  }

  if (!legacyCopyText(value)) {
    throw new Error('clipboard copy is unavailable')
  }
}

function legacyCopyText(value: string) {
  if (!document.body || typeof document.execCommand !== 'function') return false
  const activeElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.setAttribute('aria-hidden', 'true')
  Object.assign(textarea.style, {
    position: 'fixed',
    top: '0',
    left: '-9999px',
    width: '1px',
    height: '1px',
    opacity: '0'
  })
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  textarea.setSelectionRange(0, textarea.value.length)
  let copied = false
  try {
    copied = document.execCommand('copy')
  } finally {
    textarea.remove()
    activeElement?.focus()
  }
  return copied
}
