import { afterEach, describe, expect, it, vi } from 'vitest'
import { copyTextToClipboard } from './clipboard'

const originalClipboard = navigator.clipboard
const originalExecCommand = document.execCommand

afterEach(() => {
  Object.defineProperty(navigator, 'clipboard', { configurable: true, value: originalClipboard })
  Object.defineProperty(document, 'execCommand', { configurable: true, value: originalExecCommand })
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('copyTextToClipboard', () => {
  it('uses the modern clipboard API when it is available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })

    await copyTextToClipboard('sample data')

    expect(writeText).toHaveBeenCalledWith('sample data')
  })

  it('falls back to execCommand when the modern API is rejected', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('not allowed'))
    const execCommand = vi.fn().mockReturnValue(true)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    Object.defineProperty(document, 'execCommand', { configurable: true, value: execCommand })

    await copyTextToClipboard('1 2\n')

    expect(execCommand).toHaveBeenCalledWith('copy')
    expect(document.querySelector('textarea')).toBeNull()
  })
})
