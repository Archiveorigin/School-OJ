<template>
  <section class="page problem-detail-page">
    <div class="detail-container">
      <el-page-header class="back-to-bank" title="返回题库" @back="router.push('/problems')">
        <template #content>
          <span v-if="problem" class="page-header-title">
            {{ problemDisplayCode(problem) }} · {{ problem.title }}
          </span>
        </template>
      </el-page-header>

      <el-skeleton v-if="loading" :rows="8" animated class="loading-state" />
      <el-result
        v-else-if="loadError"
        icon="error"
        title="题目无法加载"
        :sub-title="loadError"
      >
        <template #extra>
          <el-button type="primary" @click="router.push('/problems')">返回题库</el-button>
        </template>
      </el-result>

      <template v-else-if="problem">
        <header class="problem-heading">
          <div class="heading-copy">
            <div class="heading-code">{{ problemDisplayCode(problem) }}</div>
            <h1>{{ problem.title }}</h1>
            <div class="problem-meta">
              <span>{{ problem.slug }}</span>
              <span>{{ problemLimitText(problem) }}</span>
              <el-tag :type="difficultyTagType(difficulty)" effect="light">
                {{ difficulty || '未分级' }}
              </el-tag>
            </div>
            <div v-if="tags.length" class="tag-strip">
              <el-tag v-for="tag in tags" :key="tag" size="small" effect="plain">{{ tag }}</el-tag>
            </div>
          </div>
          <div v-if="canManage" class="toolbar manage-actions">
            <ProblemTestDownloads
              :problem-id="problem.id"
              :problem-code="problem.display_code"
            />
            <el-button type="primary" plain @click="editVisible = true">修改题目</el-button>
            <el-button v-if="canDelete" type="danger" plain @click="removeProblem">删除题目</el-button>
          </div>
        </header>

        <main class="problem-content">
          <section class="statement-section">
            <MarkdownRenderer :source="statementBody" :problem-id="problem.id" />
            <ProblemSamplesView :samples="samples" />
          </section>

          <section class="submission-section">
            <div class="submission-toolbar">
              <el-select v-model="language" aria-label="编程语言" class="language-select">
                <el-option label="C++17" value="cpp" />
                <el-option label="C" value="c" />
                <el-option label="Python" value="python" />
                <el-option label="Java" value="java" />
              </el-select>
              <div class="toolbar">
                <el-button @click="formatSource">自动格式化</el-button>
                <el-button type="primary" :loading="submitting" @click="submit">提交</el-button>
              </div>
            </div>
            <CodeEditor ref="editorRef" v-model="source" :language="language" />
            <div v-if="live" class="submission-result">
              <StatusBadge :status="live.status" />
              <span>分数 {{ live.score }}，{{ live.message }}</span>
            </div>
          </section>
        </main>

        <ProblemEditDialog v-model="editVisible" :problem="problem" @saved="handleSaved" />
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, sseUrl, type Problem } from '../../api/client'
import CodeEditor from '../../components/CodeEditor.vue'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'
import ProblemEditDialog from '../../components/ProblemEditDialog.vue'
import ProblemSamplesView from '../../components/ProblemSamplesView.vue'
import ProblemTestDownloads from '../../components/ProblemTestDownloads.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import {
  difficultyFromTags,
  difficultyTagType,
  extractStatementSamples,
  problemDisplayCode,
  problemLimitText,
  stripStatementSamples,
  tagList
} from '../../features/problems/problemMeta'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const problem = ref<Problem | null>(null)
const loading = ref(false)
const loadError = ref('')
const editVisible = ref(false)
const language = ref('cpp')
const submitting = ref(false)
const live = ref<any>(null)
const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
let submissionEvents: EventSource | null = null

const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const canDelete = computed(() => Boolean(problem.value && (auth.role === 'admin' || problem.value.owner_id === auth.user?.id)))
const tags = computed(() => tagList(problem.value?.tags))
const difficulty = computed(() => difficultyFromTags(problem.value?.tags))
const samples = computed(() => extractStatementSamples(problem.value?.statement))
const statementBody = computed(() => stripStatementSamples(problem.value?.statement))
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
    loadError.value = err.response?.status === 404
      ? '该题目不存在或尚未公开'
      : (err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function formatSource() {
  editorRef.value?.format()
}

async function submit() {
  if (!auth.isAuthed) {
    await router.push({ path: '/login', query: { redirect: route.fullPath } })
    return
  }
  if (!problem.value) return
  submitting.value = true
  try {
    const { data } = await client.post('/submissions', {
      problem_id: problem.value.id,
      language: language.value,
      source_code: source.value
    })
    watchSubmission(data.id)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function watchSubmission(id: number) {
  submissionEvents?.close()
  submissionEvents = new EventSource(sseUrl(`/submissions/${id}/events`))
  submissionEvents.addEventListener('status', (event) => {
    live.value = JSON.parse((event as MessageEvent).data)
    if (!['queued', 'running'].includes(live.value.status)) {
      submissionEvents?.close()
      submissionEvents = null
    }
  })
}

function handleSaved(value: Problem) {
  problem.value = value
}

async function removeProblem() {
  if (!problem.value) return
  try {
    await ElMessageBox.confirm(
      '删除后题目将从公共题库隐藏，历史提交与报表会保留。确认删除？',
      '删除题目',
      { type: 'warning' }
    )
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

.back-to-bank :deep(.el-page-header__left) {
  padding: 9px 14px;
  color: #fff;
  border-radius: 10px;
  background: linear-gradient(135deg, #0a5ea6, #0f766e);
  box-shadow: 0 8px 20px rgba(10, 94, 166, 0.22);
}

.back-to-bank :deep(.el-page-header__title),
.back-to-bank :deep(.el-page-header__icon) {
  color: #fff;
  font-weight: 800;
}

.loading-state {
  margin-top: 28px;
}

.problem-heading {
  display: flex;
  align-items: flex-start;
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

.problem-meta,
.tag-strip,
.submission-toolbar,
.submission-result {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.problem-meta {
  margin-top: 12px;
  color: var(--muted);
  font-size: 14px;
}

.tag-strip {
  margin-top: 12px;
}

.manage-actions {
  flex: 0 0 auto;
}

.problem-content {
  display: grid;
  gap: 0;
}

.statement-section,
.submission-section {
  min-width: 0;
  padding: 28px 0;
}

.statement-section {
  border-bottom: 1px solid var(--border);
}

.submission-toolbar {
  justify-content: space-between;
  margin-bottom: 14px;
}

.language-select {
  width: 140px;
}

.submission-result {
  margin-top: 14px;
}

@media (max-width: 760px) {
  .problem-detail-page {
    padding: 16px 14px 28px;
  }

  .problem-heading {
    flex-direction: column;
    padding-top: 24px;
  }

  .problem-heading h1 {
    font-size: 24px;
  }

  .submission-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .language-select {
    width: 100%;
  }
}
</style>
