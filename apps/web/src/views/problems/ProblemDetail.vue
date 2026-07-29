<template>
  <section class="page problem-detail-page">
    <div class="detail-container">
      <el-page-header title="返回题库" @back="router.push('/problems')">
        <template #content>
          <span v-if="problem" class="page-header-title"> {{ problemDisplayCode(problem) }} · {{ problem.title }} </span>
        </template>
      </el-page-header>

      <el-skeleton v-if="loading" :rows="8" animated class="loading-state" />
      <el-result v-else-if="loadError" icon="error" title="题目无法加载" :sub-title="loadError">
        <template #extra>
          <el-button type="primary" @click="router.push('/problems')">返回题库</el-button>
        </template>
      </el-result>

      <template v-else-if="problem">
        <header class="problem-heading">
          <div class="heading-copy">
            <div class="heading-code">{{ problemDisplayCode(problem) }}</div>
            <h1>{{ problem.title }}</h1>
          </div>
          <div class="heading-actions">
            <div class="toolbar submission-actions">
              <el-button v-if="auth.isAuthed" @click="openSubmissionRecords">提交记录</el-button>
              <el-button type="primary" @click="openSubmitDialog">提交代码</el-button>
            </div>
            <div v-if="canManage" class="toolbar manage-actions">
              <ProblemTestDownloads :problem-id="problem.id" :problem-code="problem.display_code" />
              <el-button type="primary" plain @click="editVisible = true">修改题目</el-button>
              <el-button v-if="canDelete" type="danger" plain @click="removeProblem">删除题目</el-button>
            </div>
          </div>
        </header>

        <main class="problem-content">
          <div class="problem-detail-grid">
            <section class="panel statement-section">
              <MarkdownRenderer :source="problem.statement" :problem-id="problem.id" />
            </section>
            <ProblemMetaCard
              :problem="problem"
              :status-text="progressLabel(problem.progress_status)"
              :status-type="progressTag(problem.progress_status)"
            />
          </div>
          <div v-if="live" class="panel submission-result">
            <StatusBadge :status="live.status" />
            <span>{{ live.message }}</span>
          </div>
        </main>

        <el-dialog
          v-model="submitVisible"
          :title="`提交代码 · ${problem.title}`"
          width="min(980px, calc(100vw - 28px))"
          destroy-on-close
          align-center
        >
          <div class="submit-dialog-body">
            <div class="submit-options">
              <el-form-item label="提交语言">
                <el-select v-model="language" aria-label="编程语言" class="language-select">
                  <el-option label="C++17" value="cpp" />
                  <el-option label="C" value="c" />
                  <el-option label="Python 3" value="python" />
                  <el-option label="Java 21" value="java" />
                </el-select>
              </el-form-item>
              <el-form-item label="公开代码">
                <el-switch v-model="isPublic" active-text="所有登录用户可查看" inactive-text="仅自己可见" />
              </el-form-item>
            </div>
            <CodeEditor ref="editorRef" v-model="source" :language="language" />
            <el-input :model-value="codeFileName" readonly placeholder="可从本地代码文件载入文本框">
              <template #prepend>代码文件</template>
              <template #append>
                <el-button @click="codeFileInput?.click()">选择文件</el-button>
              </template>
            </el-input>
            <input
              ref="codeFileInput"
              class="hidden-file-input"
              type="file"
              accept=".c,.cc,.cpp,.cxx,.h,.hpp,.py,.java,.txt"
              @change="loadCodeFile"
            />
          </div>
          <template #footer>
            <el-button @click="formatSource">自动格式化</el-button>
            <el-button @click="submitVisible = false">取消</el-button>
            <el-button type="primary" :loading="submitting" @click="submit">确认提交</el-button>
          </template>
        </el-dialog>

        <ProblemEditDialog v-model="editVisible" :problem="problem" @saved="handleSaved" />
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AuthenticatedEventSource, client, openEventStream, type Problem } from '../../api/client'
import CodeEditor from '../../components/CodeEditor.vue'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'
import ProblemEditDialog from '../../components/ProblemEditDialog.vue'
import ProblemMetaCard from '../../components/ProblemMetaCard.vue'
import ProblemTestDownloads from '../../components/ProblemTestDownloads.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { problemDisplayCode, progressLabel, progressTag } from '../../features/problems/problemMeta'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const problem = ref<Problem | null>(null)
const loading = ref(false)
const loadError = ref('')
const editVisible = ref(false)
const submitVisible = ref(false)
const language = ref('cpp')
const isPublic = ref(false)
const submitting = ref(false)
const live = ref<any>(null)
const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
const codeFileInput = ref<HTMLInputElement>()
const codeFileName = ref('')
let submissionEvents: AuthenticatedEventSource | null = null

const canManage = computed(() => auth.role === 'admin' || (Boolean(auth.user?.can_author) && problem.value?.owner_id === auth.user?.id))
const canDelete = computed(() => Boolean(problem.value && (auth.role === 'admin' || problem.value.owner_id === auth.user?.id)))
const source = ref(`#include <bits/stdc++.h>
using namespace std;
int main() {
  return 0;
}
`)

async function loadProblem() {
  loading.value = true
  loadError.value = ''
  problem.value = null
  live.value = null
  try {
    const { data } = await client.get(`/problems/${encodeURIComponent(String(route.params.id))}`)
    problem.value = data
  } catch (err: any) {
    loadError.value = err.response?.status === 404 ? '该题目不存在或尚未公开' : err.response?.data?.error || err.message
  } finally {
    loading.value = false
  }
}

function formatSource() {
  editorRef.value?.format()
}

async function requireLogin() {
  if (auth.isAuthed) return true
  await router.push({ path: '/login', query: { redirect: route.fullPath } })
  return false
}

async function openSubmitDialog() {
  if (!(await requireLogin())) return
  submitVisible.value = true
}

async function openSubmissionRecords() {
  if (!(await requireLogin()) || !problem.value) return
  await router.push(`/problems/${encodeURIComponent(problem.value.display_code || String(problem.value.id))}/submissions`)
}

async function submit() {
  if (!(await requireLogin())) return
  if (!problem.value) return
  if (!source.value.trim()) {
    ElMessage.error('请输入或导入代码')
    return
  }
  submitting.value = true
  try {
    const { data } = await client.post('/submissions', {
      problem_id: problem.value.id,
      language: language.value,
      source_code: source.value,
      is_public: isPublic.value
    })
    submitVisible.value = false
    ElMessage.success('提交已进入评测队列')
    watchSubmission(data.id)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function watchSubmission(id: number) {
  submissionEvents?.close()
  submissionEvents = openEventStream(`/submissions/${id}/events`)
  submissionEvents.addEventListener('status', (event) => {
    live.value = JSON.parse((event as MessageEvent).data)
    if (!['queued', 'running'].includes(live.value.status)) {
      submissionEvents?.close()
      submissionEvents = null
    }
  })
}

async function loadCodeFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (file.size > 1024 * 1024) {
    ElMessage.error('代码文件不能超过 1 MB')
    input.value = ''
    return
  }
  source.value = await file.text()
  codeFileName.value = file.name
  const extension = file.name.toLowerCase().split('.').pop()
  if (extension === 'c') language.value = 'c'
  else if (['cc', 'cpp', 'cxx', 'h', 'hpp'].includes(extension || '')) language.value = 'cpp'
  else if (extension === 'py') language.value = 'python'
  else if (extension === 'java') language.value = 'java'
  input.value = ''
}

function handleSaved(value: Problem) {
  problem.value = value
}

async function removeProblem() {
  if (!problem.value) return
  try {
    await ElMessageBox.confirm('删除后题目将从公共题库隐藏，历史提交与报表会保留。确认删除？', '删除题目', { type: 'warning' })
    await client.delete(`/problems/${problem.value.id}`)
    ElMessage.success('题目已下架')
    await router.push('/problems')
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') {
      ElMessage.error(err.response?.data?.error || err.message)
    }
  }
}

watch(() => route.params.id, loadProblem, { immediate: true })
onBeforeUnmount(() => submissionEvents?.close())
</script>

<style scoped>
.problem-detail-page {
  padding: 22px 20px 40px;
}

.detail-container {
  width: min(1180px, 100%);
  margin: 0 auto;
}

.page-header-title {
  color: var(--text);
  font-size: 16px;
  font-weight: 700;
}

.loading-state {
  margin-top: 28px;
}

.problem-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 30px 0 24px;
  border-bottom: 1px solid var(--border);
}

.heading-copy {
  min-width: 0;
}

.heading-code {
  margin-bottom: 6px;
  color: var(--accent);
  font-size: 14px;
  font-weight: 800;
}

.problem-heading h1 {
  margin: 0;
  color: var(--text);
  font-size: 30px;
  line-height: 1.25;
}

.submission-result {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.heading-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  flex: 0 0 auto;
  gap: 12px;
}

.manage-actions {
  padding-left: 12px;
  border-left: 1px solid var(--border);
}

.problem-content {
  padding-top: 24px;
}

.problem-detail-grid {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(260px, 320px);
  align-items: start;
  gap: 14px;
}

.statement-section {
  min-width: 0;
}

.language-select {
  width: 140px;
}

.submission-result {
  margin-top: 14px;
  padding: 14px 18px;
}

.submit-dialog-body {
  display: grid;
  gap: 14px;
}

.submit-options {
  display: grid;
  grid-template-columns: minmax(180px, 240px) minmax(280px, 1fr);
  gap: 16px;
}

.submit-options :deep(.el-form-item) {
  margin-bottom: 0;
}

.hidden-file-input {
  display: none;
}

@media (max-width: 760px) {
  .problem-detail-page {
    padding: 16px 14px 28px;
  }

  .problem-heading {
    align-items: stretch;
    flex-direction: column;
    padding-top: 24px;
  }

  .problem-heading h1 {
    font-size: 24px;
  }

  .problem-detail-grid {
    grid-template-columns: 1fr;
  }

  .submit-options {
    align-items: stretch;
    grid-template-columns: 1fr;
  }

  .heading-actions {
    align-items: stretch;
    justify-content: flex-start;
    flex-direction: column;
  }

  .submission-actions,
  .manage-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }

  .manage-actions {
    padding-top: 12px;
    padding-left: 0;
    border-top: 1px solid var(--border);
    border-left: 0;
  }

  .heading-actions .el-button,
  .heading-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }

  .language-select {
    width: 100%;
  }
}
</style>
