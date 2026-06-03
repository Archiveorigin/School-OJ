<template>
  <section class="test-downloads">
    <div class="section-title">
      <h3>后台测试点</h3>
      <div class="toolbar">
        <el-button size="small" :loading="downloadingAll" @click="downloadAll">下载全部</el-button>
        <el-button size="small" text :loading="loading" @click="loadTests">刷新</el-button>
      </div>
    </div>
    <el-table v-if="tests.length" :data="tests" size="small">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="输入文件" min-width="120">
        <template #default="{ row }">
          <el-button size="small" text :loading="downloadingPath === row.input" @click="downloadFile(row.input, `${row.name}.in`)">
            {{ row.input }}
          </el-button>
        </template>
      </el-table-column>
      <el-table-column label="输出文件" min-width="120">
        <template #default="{ row }">
          <el-button size="small" text :loading="downloadingPath === row.output" @click="downloadFile(row.output, `${row.name}.out`)">
            {{ row.output }}
          </el-button>
        </template>
      </el-table-column>
      <el-table-column prop="weight" label="权重" width="80" />
    </el-table>
    <p v-else class="muted empty-tests">暂无可下载测试点</p>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { ref, watch } from 'vue'
import { client } from '../api/client'

interface ProblemTest {
  name: string
  input: string
  output: string
  weight: number
}

const props = defineProps<{
  problemId: number
  problemCode?: string
}>()

const tests = ref<ProblemTest[]>([])
const loading = ref(false)
const downloadingAll = ref(false)
const downloadingPath = ref('')

async function loadTests() {
  if (!props.problemId) return
  loading.value = true
  try {
    const { data } = await client.get(`/problems/${props.problemId}/tests`)
    tests.value = data.tests || []
  } catch (err: any) {
    tests.value = []
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
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

async function downloadFile(path: string, filename: string) {
  downloadingPath.value = path
  try {
    const encoded = path
      .split('/')
      .map((part) => encodeURIComponent(part))
      .join('/')
    const { data } = await client.get(`/problems/${props.problemId}/tests/${encoded}`, { responseType: 'blob' })
    downloadBlob(data, filename)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    downloadingPath.value = ''
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

watch(() => props.problemId, loadTests, { immediate: true })
</script>

<style scoped>
.test-downloads {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.empty-tests {
  margin: 0;
}
</style>
