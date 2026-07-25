import { describe, expect, it } from 'vitest'
import {
  extractStatementSamples,
  replaceStatementSamples,
  stripStatementSamples
} from './problemMeta'

describe('problem statement samples', () => {
  it('extracts named samples and keeps the statement body separate', () => {
    const source = `# A + B

请计算答案。

### 输入样例 1（基础数据）

\`\`\`text
1 2
\`\`\`

### 输出样例 1（基础数据）

\`\`\`text
3
\`\`\``

    expect(extractStatementSamples(source)).toEqual([
      { index: 1, name: '基础数据', input: '1 2', output: '3' }
    ])
    expect(stripStatementSamples(source)).toBe('# A + B\n\n请计算答案。')
  })

  it('replaces existing samples without duplicating them', () => {
    const source = `题面

### 输入样例 1
\`\`\`
old
\`\`\`

### 输出样例 1
\`\`\`
old result
\`\`\``
    const updated = replaceStatementSamples(source, [
      { name: '新样例', input: '4 5', output: '9' }
    ])

    expect(updated.match(/输入样例/g)).toHaveLength(1)
    expect(extractStatementSamples(updated)).toEqual([
      { index: 1, name: '新样例', input: '4 5', output: '9' }
    ])
  })
})
