<template>
  <section class="page problem-create-page">
    <div class="create-container">
      <el-page-header title="返回题库" @back="router.push('/problems')">
        <template #content><span>创建题目</span></template>
      </el-page-header>

      <header class="create-heading">
        <div>
          <span class="eyebrow">PROBLEM AUTHORING</span>
          <h1>新建题库题目</h1>
          <p>系统会自动分配内部唯一编号；题面中的输入输出代码块会直接展示并提供复制按钮。</p>
        </div>
        <el-upload
          action="#"
          accept=".md,text/markdown"
          :auto-upload="false"
          :show-file-list="false"
          :on-change="importMarkdown"
        >
          <el-button :loading="importingMarkdown">从 MD 文档导入</el-button>
        </el-upload>
      </header>

      <el-form label-position="top" class="authoring-grid">
        <section class="panel form-panel">
          <div class="section-title"><h2>题目内容</h2><span>支持 Markdown 与 LaTeX</span></div>
          <el-form-item label="题目名称">
            <el-input v-model="form.title" maxlength="200" show-word-limit placeholder="例如：两数之和" />
          </el-form-item>
          <el-form-item label="题面">
            <el-input
              v-model="form.statement"
              type="textarea"
              :rows="18"
              placeholder="请在题面中直接编写输入、输出与样例。使用 ```text 代码块包裹输入输出数据。"
            />
          </el-form-item>
          <div class="two-columns">
            <el-form-item label="难度">
              <el-select v-model="form.difficulty" style="width: 100%">
                <el-option v-for="item in problemDifficultyOptions" :key="item" :label="item" :value="item" />
              </el-select>
            </el-form-item>
            <el-form-item label="标签">
              <el-input v-model="form.tags" placeholder="数组、动态规划；用逗号或空格分隔" />
            </el-form-item>
          </div>

          <div class="section-title preview-title"><h2>题面预览</h2><span>代码块右上角可复制</span></div>
          <div class="statement-preview">
            <MarkdownRenderer :source="form.statement || '在左侧题面文本框中输入内容后，这里会实时预览。'" />
          </div>
        </section>

        <aside class="side-column">
          <section class="panel limits-panel">
            <div class="section-title"><h2>评测限制</h2><span>必须大于 0</span></div>
            <el-form-item label="时间限制">
              <el-input-number v-model="form.time_limit_ms" :min="100" :step="100" />
              <span class="unit">ms</span>
            </el-form-item>
            <el-form-item label="内存限制">
              <el-input-number v-model="form.memory_limit_mb" :min="16" :step="16" />
              <span class="unit">MB</span>
            </el-form-item>
            <el-form-item label="输出限制">
              <el-input-number v-model="form.output_limit_kb" :min="1" :step="64" />
              <span class="unit">KB</span>
            </el-form-item>
          </section>

          <section class="panel import-panel">
            <div class="section-title"><h2>测试点压缩包</h2><span>ZIP / IN / OUT</span></div>
            <el-upload
              drag
              action="#"
              multiple
              accept=".zip,.in,.out"
              :auto-upload="false"
              :file-list="testFiles"
              :on-change="syncTestFiles"
              :on-remove="syncTestFiles"
            >
              <div class="upload-title">拖入测试样例压缩包</div>
              <div class="muted">也可选择成对的 .in / .out 文件</div>
            </el-upload>
            <el-alert
              v-if="testFiles.length"
              type="success"
              :closable="false"
              :title="`已选择 ${testFiles.length} 个文件，将替代下方手动测试点`"
            />
          </section>
        </aside>

        <section class="panel cases-panel">
          <div class="case-toolbar">
            <div class="section-title">
              <h2>测试样例</h2>
              <span>用于后台评测，不在前台单独展示</span>
            </div>
            <el-button @click="addCase">添加测试样例</el-button>
          </div>
          <div v-for="(item, index) in form.cases" :key="index" class="case-editor">
            <div class="case-head">
              <strong>测试点 {{ index + 1 }}</strong>
              <el-input v-model="item.name" placeholder="测试点名称" />
              <el-input-number v-model="item.weight" :min="1" :max="100" />
              <el-button type="danger" plain :disabled="form.cases.length === 1" @click="removeCase(index)">删除</el-button>
            </div>
            <div class="two-columns">
              <el-form-item label="输入">
                <el-input v-model="item.input" type="textarea" :rows="7" placeholder="输入数据" />
              </el-form-item>
              <el-form-item label="期望输出">
                <el-input v-model="item.output" type="textarea" :rows="7" placeholder="期望输出" />
              </el-form-item>
            </div>
          </div>
        </section>
      </el-form>

      <footer class="create-footer">
        <el-button @click="router.push('/problems')">取消</el-button>
        <el-button type="primary" size="large" :loading="saving" @click="createProblem">创建并打开题目</el-button>
      </footer>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client } from '../../api/client'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'
import { problemDifficultyOptions } from '../../features/problems/problemMeta'

type TestCase = { name: string; input: string; output: string; weight: number }
const router = useRouter()
const saving = ref(false)
const importingMarkdown = ref(false)
const testFiles = ref<any[]>([])
const form = reactive({
  title: '',
  statement: '',
  difficulty: '入门',
  tags: '',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  cases: [{ name: 'case-01', input: '1 2\n', output: '3\n', weight: 100 }] as TestCase[]
})

function parseTags() {
  return form.tags.split(/[\s,，、]+/).map((item) => item.trim()).filter(Boolean)
}

function addCase() {
  const index = form.cases.length + 1
  form.cases.push({ name: `case-${String(index).padStart(2, '0')}`, input: '', output: '', weight: 1 })
}

function removeCase(index: number) {
  form.cases.splice(index, 1)
}

function syncTestFiles(_file: any, fileList: any[]) {
  testFiles.value = fileList
}

async function importMarkdown(uploadFile: any) {
  const file = uploadFile.raw as File | undefined
  if (!file) return
  importingMarkdown.value = true
  try {
    const payload = new FormData()
    payload.append('file', file)
    const { data } = await client.post('/problems/parse-markdown', payload)
    const draft = data.problems?.[0]
    if (!draft) {
      ElMessage.error('MD 文档中没有识别到题目')
      return
    }
    form.title = draft.title || ''
    form.statement = draft.statement || ''
    form.time_limit_ms = draft.time_limit_ms || 1000
    form.memory_limit_mb = draft.memory_limit_mb || 256
    form.output_limit_kb = draft.output_limit_kb || 1024
    if (Array.isArray(draft.cases) && draft.cases.length) {
      form.cases.splice(0, form.cases.length, ...draft.cases.map((item: any, index: number) => ({
        name: item.name || `case-${String(index + 1).padStart(2, '0')}`,
        input: item.input || '',
        output: item.output || '',
        weight: item.weight || Math.max(1, Math.floor(100 / draft.cases.length))
      })))
    }
    const extra = Math.max(0, (data.problems?.length || 1) - 1)
    ElMessage.success(extra ? `已载入第一道题，文档中另有 ${extra} 道题` : 'MD 文档已载入')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    importingMarkdown.value = false
  }
}

async function createProblem() {
  if (!form.title.trim()) {
    ElMessage.error('请输入题目名称')
    return
  }
  if (!form.statement.trim()) {
    ElMessage.error('请输入题面')
    return
  }
  if (!testFiles.value.length && !form.cases.some((item) => item.input || item.output)) {
    ElMessage.error('请添加测试样例或导入测试点压缩包')
    return
  }
  saving.value = true
  try {
    const draft = {
      title: form.title.trim(),
      statement: form.statement,
      difficulty: form.difficulty,
      tags: parseTags(),
      time_limit_ms: form.time_limit_ms,
      memory_limit_mb: form.memory_limit_mb,
      output_limit_kb: form.output_limit_kb,
      cases: form.cases
    }
    let data: any
    if (testFiles.value.length) {
      const payload = new FormData()
      payload.append('draft', JSON.stringify(draft))
      testFiles.value.forEach((item) => { if (item.raw) payload.append('test_files', item.raw) })
      data = (await client.post('/problems', payload)).data
    } else {
      data = (await client.post('/problems', draft)).data
    }
    ElMessage.success('题目已创建')
    await router.push(`/problems/${encodeURIComponent(data.display_code || String(data.id))}`)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.problem-create-page { padding: 22px 20px 50px; }
.create-container { width: min(1180px, 100%); margin: 0 auto; }
.create-heading { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 34px 0 24px; }
.create-heading h1 { margin: 6px 0; font-size: 30px; }
.create-heading p { margin: 0; color: var(--muted); }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .16em; }
.authoring-grid { display: grid; grid-template-columns: minmax(0, 1.55fr) minmax(300px, .65fr); gap: 16px; }
.form-panel { grid-row: span 2; }
.side-column { display: grid; align-content: start; gap: 16px; }
.cases-panel { grid-column: 1 / -1; }
.section-title { display: flex; align-items: baseline; justify-content: space-between; gap: 14px; margin-bottom: 16px; }
.section-title h2 { margin: 0; font-size: 19px; }
.section-title span { color: var(--muted); font-size: 12px; }
.preview-title { margin-top: 22px; padding-top: 20px; border-top: 1px solid var(--border); }
.statement-preview { min-height: 140px; padding: 16px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface-strong); }
.two-columns { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.unit { margin-left: 8px; color: var(--muted); }
.limits-panel :deep(.el-input-number) { width: calc(100% - 38px); }
.import-panel { display: grid; gap: 12px; }
.upload-title { margin-bottom: 6px; font-weight: 700; }
.case-toolbar, .case-head { display: flex; align-items: center; gap: 12px; }
.case-toolbar { justify-content: space-between; }
.case-toolbar .section-title { margin-bottom: 0; }
.case-editor { display: grid; gap: 12px; margin-top: 16px; padding: 16px; border: 1px solid var(--border); border-radius: 10px; }
.case-head strong { flex: 0 0 auto; }
.case-head .el-input { min-width: 180px; }
.create-footer { position: sticky; bottom: 0; z-index: 5; display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; padding: 14px; border: 1px solid var(--border); border-radius: 12px; background: var(--glass); backdrop-filter: blur(14px); }
@media (max-width: 900px) {
  .authoring-grid { grid-template-columns: 1fr; }
  .form-panel { grid-row: auto; }
  .cases-panel { grid-column: auto; }
}
@media (max-width: 640px) {
  .problem-create-page { padding: 16px 12px 36px; }
  .create-heading { align-items: stretch; flex-direction: column; }
  .two-columns { grid-template-columns: 1fr; }
  .case-head { align-items: stretch; flex-direction: column; }
  .case-head > * { width: 100%; }
}
</style>
