<template>
  <div class="test-data-entry">
    <el-button type="primary" plain class="test-data-button" @click="openDialog">
      测试数据
    </el-button>

    <el-dialog v-model="visible" title="后台测试数据" width="920px" class="test-data-dialog" destroy-on-close>
      <div class="dialog-toolbar">
        <p>仅管理人员可查看。每个测试点按名称对应输入和输出。</p>
        <el-button type="primary" :loading="downloadingAll" :disabled="!tests.length" @click="downloadAll">
          下载全部样例
        </el-button>
      </div>

      <el-skeleton v-if="loading" :rows="7" animated />
      <div v-else-if="tests.length" class="test-list">
        <article v-for="(test, index) in tests" :key="`${test.name}-${index}`" class="test-card">
          <header class="test-card-head">
            <div>
              <span>测试点 {{ index + 1 }}</span>
              <h3>{{ test.name }}</h3>
            </div>
            <el-tag effect="plain">权重 {{ test.weight }}</el-tag>
          </header>
          <div class="io-grid">
            <section class="io-block">
              <div class="io-head">
                <div><strong>输入</strong><small>{{ test.inputPath }}</small></div>
                <el-button size="small" text @click="copyText(test.input)">复制</el-button>
              </div>
              <pre>{{ test.input }}</pre>
            </section>
            <section class="io-block">
              <div class="io-head">
                <div><strong>输出</strong><small>{{ test.outputPath }}</small></div>
                <el-button size="small" text @click="copyText(test.output)">复制</el-button>
              </div>
              <pre>{{ test.output }}</pre>
            </section>
          </div>
        </article>
      </div>
      <el-empty v-else description="暂无可查看测试数据" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { ref, watch } from 'vue'
import { client } from '../api/client'

interface ProblemTestMeta {
  name: string
  input: string
  output: string
  weight: number
}

interface ProblemTest {
  name: string
  inputPath: string
  outputPath: string
  input: string
  output: string
  weight: number
}

const props = defineProps<{
  problemId: number
  problemCode?: string
}>()

const visible = ref(false)
const tests = ref<ProblemTest[]>([])
const loading = ref(false)
const downloadingAll = ref(false)

function openDialog() {
  visible.value = true
  void loadTests(true)
}

async function loadTests(includeContents = visible.value) {
  if (!props.problemId) return
  loading.value = true
  try {
    const { data } = await client.get(`/problems/${props.problemId}/tests`)
    const metadata: ProblemTestMeta[] = data.tests || []
    tests.value = await Promise.all(metadata.map(async (test) => {
      if (!includeContents) {
        return {
          name: test.name,
          inputPath: test.input,
          outputPath: test.output,
          input: '',
          output: '',
          weight: test.weight
        }
      }
      const [input, output] = await Promise.all([
        readTestFile(test.input),
        readTestFile(test.output)
      ])
      return {
        name: test.name,
        inputPath: test.input,
        outputPath: test.output,
        input,
        output,
        weight: test.weight
      }
    }))
  } catch (err: any) {
    tests.value = []
    ElMessage.error(err.response?.data?.error || err.message || '测试数据加载失败')
  } finally {
    loading.value = false
  }
}

async function readTestFile(path: string) {
  const encoded = path
    .split('/')
    .map((part) => encodeURIComponent(part))
    .join('/')
  const { data } = await client.get(`/problems/${props.problemId}/tests/file/${encoded}`, { responseType: 'text' })
  return typeof data === 'string' ? data : String(data ?? '')
}

async function downloadAll() {
  downloadingAll.value = true
  try {
    const { data } = await client.get(`/problems/${props.problemId}/tests/download`, { responseType: 'blob' })
    downloadBlob(data, `${props.problemCode || `problem-${props.problemId}`}-tests.zip`)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    downloadingAll.value = false
  }
}

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动选择文本')
  }
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}

watch(() => props.problemId, () => {
  tests.value = []
  void loadTests(visible.value)
}, { immediate: true })
</script>

<style scoped>
.test-data-entry {
  display: inline-flex;
}

.test-data-button {
  min-width: 108px;
  font-weight: 700;
}

.dialog-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 18px;
}

.dialog-toolbar p {
  margin: 0;
  color: var(--muted);
}

.test-list {
  display: grid;
  gap: 16px;
  max-height: 66vh;
  overflow: auto;
  padding-right: 4px;
}

.test-card {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
}

.test-card-head,
.io-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.test-card-head {
  padding: 12px 14px;
  background: color-mix(in srgb, var(--surface-strong) 82%, var(--app-bg));
}

.test-card-head span,
.io-head small {
  display: block;
  color: var(--muted);
  font-size: 12px;
}

.test-card-head h3 {
  margin: 2px 0 0;
  font-size: 16px;
}

.io-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.io-block {
  min-width: 0;
}

.io-block + .io-block {
  border-left: 1px solid var(--border);
}

.io-head {
  padding: 9px 12px;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}

.io-block pre {
  min-height: 112px;
  max-height: 300px;
  overflow: auto;
  margin: 0;
  padding: 13px;
  color: #e2e8f0;
  background: #0f172a;
  white-space: pre;
}

@media (max-width: 760px) {
  .dialog-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .io-grid {
    grid-template-columns: 1fr;
  }

  .io-block + .io-block {
    border-left: 0;
  }
}
</style>
