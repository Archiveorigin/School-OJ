import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CodeEditor from '../src/components/CodeEditor.vue'
import { editorLanguage, formatIndentation } from '../src/features/editor/format'

const state = vi.hoisted(() => ({ value: '', changed: () => {}, action: { isSupported: vi.fn(() => false), run: vi.fn() }, model: {} as any, editor: {} as any, create: vi.fn(), language: vi.fn() }))
vi.mock('../src/features/editor/monaco', () => ({ monaco: { editor: {
  createModel: vi.fn((value) => { state.value = value; return state.model }),
  create: (...args: any[]) => { state.create(...args); return state.editor },
  setModelLanguage: state.language, setTheme: vi.fn(),
} } }))
vi.mock('element-plus', () => ({ ElMessage: { info: vi.fn(), warning: vi.fn() } }))

beforeEach(() => {
  vi.clearAllMocks()
  state.action.isSupported.mockReturnValue(false)
  state.model = { getValue: vi.fn(() => state.value), updateOptions: vi.fn(), getVersionId: () => 1, getFullModelRange: () => ({}), dispose: vi.fn(), onDidChangeContent: vi.fn((fn) => { state.changed = fn; return { dispose: vi.fn() } }) }
  state.editor = { getValue: () => state.value, dispose: vi.fn(), setValue: vi.fn((value) => { state.value = value; state.changed() }), getAction: () => state.action, saveViewState: () => ({}), restoreViewState: vi.fn(), pushUndoStop: vi.fn(), executeEdits: vi.fn((_source, edits) => { state.value = edits[0].text; state.changed() }) }
})

describe('editor lifecycle and safe formatting', () => {
  it('uses the latest props after import and disposes its owned model', async () => {
    const wrapper = mount(CodeEditor, { props: { modelValue: 'old', language: 'cpp' } })
    await wrapper.setProps({ modelValue: 'new', language: 'java' })
    await flushPromises()
    expect(state.value).toBe('new')
    state.value = 'typed'; state.changed()
    expect(wrapper.emitted('update:modelValue')).toEqual([['typed']])
    await wrapper.setProps({ modelValue: 'typed' })
    expect(state.editor.setValue).not.toHaveBeenCalled()
    await wrapper.setProps({ modelValue: 'external' })
    expect(wrapper.emitted('update:modelValue')).toHaveLength(1)
    wrapper.unmount()
    expect(state.model.dispose).toHaveBeenCalledOnce()
    expect(state.editor.dispose).toHaveBeenCalledOnce()
  })
  it('does not create an editor if unmounted before the import resolves', async () => {
    const wrapper = mount(CodeEditor, { props: { modelValue: '', language: 'cpp' } })
    wrapper.unmount()
    await flushPromises()
    expect(state.create).not.toHaveBeenCalled()
  })
  it('keeps text editable and releases a partial model if initialization fails', async () => {
    state.create.mockImplementationOnce(() => { throw new Error('initialization failure') })
    const wrapper = mount(CodeEditor, { props: { modelValue: 'saved draft', language: 'cpp' } })
    await flushPromises()
    expect(state.model.dispose).toHaveBeenCalledOnce()
    expect(wrapper.get('textarea').element.value).toBe('saved draft')
    await wrapper.get('textarea').setValue('new draft')
    expect(wrapper.emitted('update:modelValue')).toEqual([['new draft']])
    wrapper.unmount()
  })
  it('does not fall back when an installed formatter makes no changes', async () => {
    const wrapper = mount(CodeEditor, { props: { modelValue: 'int main() {\nreturn 0;\n}', language: 'cpp' } })
    await flushPromises()
    state.action.isSupported.mockReturnValue(true)
    await wrapper.vm.format()
    expect(state.action.run).toHaveBeenCalledOnce()
    expect(state.editor.executeEdits).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('makes fallback indentation undoable and preserves view state', async () => {
    const wrapper = mount(CodeEditor, { props: { modelValue: 'int main() {\nreturn 0;\n}', language: 'cpp' } })
    await flushPromises()
    await wrapper.vm.format()
    expect(state.value).toBe('int main() {\n  return 0;\n}')
    expect(state.editor.setValue).not.toHaveBeenCalled()
    expect(state.editor.pushUndoStop).toHaveBeenCalledTimes(2)
    expect(state.editor.restoreViewState).toHaveBeenCalledOnce()
    wrapper.unmount()
  })
  it('preserves operators, literals, comments and already meaningful indentation', () => {
    const source = 'int main() {\nint a=1; a++; // } ++\nif (a) {\nputs("{x}"); double e=1e-5;\n}\nreturn a;\n}'
    const result = formatIndentation(source, 'cpp')
    expect(result.split('\n').map(x => x.trimStart())).toEqual(source.split('\n').map(x => x.trimStart()))
    expect(result).toContain('\n  return a;')
    expect(formatIndentation(result, 'cpp')).toBe(result)
    for (const [language, code] of [
      ['python', 'if True:\n    x = """text  \n  literal"""\nprint(x)'],
      ['cpp', 'auto text = R"(a\n   b)";'],
      ['java', 'String text = """\n  literal\n""";'],
      ['cpp', '#define F(x) \\\n    x++'],
    ]) expect(formatIndentation(code, language)).toBe(code)
    expect(editorLanguage('rust')).toBe('plaintext')
  })
})
