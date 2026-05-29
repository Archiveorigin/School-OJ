import { describe, expect, it } from 'vitest'

describe('smoke', () => {
  it('has a document', () => {
    expect(document.createElement('div')).toBeTruthy()
  })
})
