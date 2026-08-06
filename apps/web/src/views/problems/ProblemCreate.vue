<template>
  <section class="page problem-create-page">
    <div class="create-container">
      <el-page-header :title="teamScope ? '返回团队题单' : '返回题库'" @back="goBack">
        <template #content><span>创建题目</span></template>
      </el-page-header>

      <header class="create-heading">
        <div>
          <span class="eyebrow">PROBLEM AUTHORING</span>
          <h1>{{ teamScope ? '新建团队私有题目' : reviewStatus === 'rejected' ? '修改退回题目' : reviewStatus === 'withdrawn' ? '修改已撤销题目' : '新建题库题目' }}</h1>
          <p>{{ teamScope ? '题目仅对当前团队成员开放，并会自动加入当前题单。' : '题目内容会自动缓存；非管理员提交后需经后台审核才会进入公共题库。' }}</p>
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

      <el-alert
        v-if="reviewStatus === 'pending'"
        type="warning"
        :closable="false"
        show-icon
        :title="`${cachedProblemCode || '当前题目'} 已提交，正在等待管理员审核`"
      />
      <el-alert
        v-else-if="reviewStatus === 'rejected'"
        class="review-alert"
        type="error"
        :closable="false"
        show-icon
        :title="`题目已退回：${reviewNote || '请按规范修改后重新提交'}`"
      />
      <el-alert
        v-else-if="reviewStatus === 'withdrawn'"
        class="review-alert"
        type="warning"
        :closable="false"
        show-icon
        :title="`题目已撤销：${reviewNote || '修改后可重新提交管理员审核'}`"
      />

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
              <ProblemTagSelector v-model="form.tags" />
            </el-form-item>
          </div>

          <div class="section-title preview-title"><h2>题面预览</h2><span>代码块右上角可复制</span></div>
          <div class="statement-preview">
            <MarkdownRenderer :source="form.statement || '在题面文本框中输入内容后，这里会实时预览。'" />
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
            <el-form-item label="答案比较">
              <el-select v-model="form.checker_type" style="width: 100%">
                <el-option label="标准比较（忽略行尾空白）" value="exact" />
                <el-option label="令牌比较（忽略空白差异）" value="tokens" />
                <el-option label="浮点比较（允许误差）" value="float" />
              </el-select>
            </el-form-item>
            <template v-if="form.checker_type === 'float'">
              <el-form-item label="绝对误差">
                <el-input-number v-model="form.absolute_tolerance" :min="0" :max="1" :step="0.000001" :precision="8" />
              </el-form-item>
              <el-form-item label="相对误差">
                <el-input-number v-model="form.relative_tolerance" :min="0" :max="1" :step="0.000001" :precision="8" />
              </el-form-item>
            </template>
            <p class="checker-note">比较器只使用平台内置算法，不运行题目包中的脚本。</p>
          </section>

          <section class="panel import-panel">
            <div class="section-title"><h2>测试点文件</h2><span>ZIP / IN / OUT</span></div>
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
              <div class="upload-title">拖入测试点压缩包</div>
              <div class="muted">也可选择成对的 .in / .out 文件</div>
            </el-upload>
            <el-alert
              v-if="testFiles.length"
              type="success"
              :closable="false"
              :title="`已选择 ${testFiles.length} 个测试文件，共 ${testFileSizeText}`"
            />
            <el-alert v-if="usesChunkedUpload" type="info" :closable="false" title="大文件将在提交时自动分块并发上传，可实时查看进度。" />
            <div v-if="uploadStage" class="upload-progress">
              <span>{{ uploadStage }}</span>
              <el-progress :percentage="uploadProgress" :stroke-width="10" />
            </div>
            <el-alert
              v-else-if="importedCases.length"
              type="info"
              :closable="false"
              :title="`MD 文档中识别到 ${importedCases.length} 个测试点`"
            />
            <p class="cache-note">文字内容自动保存在当前浏览器。测试文件无法写入浏览器缓存，退回题目沿用服务器中已上传的测试点。</p>
          </section>
        </aside>
      </el-form>

      <footer class="create-footer">
        <span class="draft-state">草稿已自动缓存</span>
        <div>
          <el-button v-if="reviewStatus !== 'pending'" @click="clearDraft">清除草稿</el-button>
          <el-button @click="goBack">取消</el-button>
          <el-button
            type="primary"
            size="large"
            :loading="saving"
            :disabled="reviewStatus === 'pending'"
            @click="submitProblem"
          >
            {{ submitLabel }}
          </el-button>
        </div>
      </footer>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, type ProblemReview } from '../../api/client'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'
import ProblemTagSelector from '../../components/ProblemTagSelector.vue'
import { problemDifficultyOptions } from '../../features/problems/problemMeta'
import {
  MAX_TEST_UPLOAD_SIZE,
  formatFileSize,
  shouldUseChunkedUpload,
  totalTestFileSize,
  uploadTestFilesInChunks
} from '../../features/problems/testFileUpload'
import { useAuthStore } from '../../stores/auth'

type TestCase = { name: string; input: string; output: string; weight: number }
type ReviewStatus = ProblemReview['status'] | ''

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const saving = ref(false)
const importingMarkdown = ref(false)
const testFiles = ref<any[]>([])
const uploadProgress = ref(0)
const uploadStage = ref('')
const importedCases = ref<TestCase[]>([])
const cachedProblemId = ref<number>()
const cachedProblemCode = ref('')
const reviewStatus = ref<ReviewStatus>('')
const reviewNote = ref('')
const form = reactive({
  title: '',
  statement: '',
  difficulty: '入门',
  tags: [] as string[],
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  checker_type: 'exact',
  absolute_tolerance: 0.000001,
  relative_tolerance: 0.000001
})

const teamID = computed(() => Number(route.query.teamId) || 0)
const problemSetID = computed(() => Number(route.query.problemSetId) || 0)
const teamScope = computed(() => Boolean(teamID.value && problemSetID.value))
const draftKey = computed(() => teamScope.value ? `school-oj-team-${teamID.value}-set-${problemSetID.value}-problem-draft-v1` : 'school-oj-problem-author-draft-v2')
const rawTestFiles = computed(() => testFiles.value.map((item) => item.raw).filter((file): file is File => file instanceof File))
const testFileSize = computed(() => totalTestFileSize(rawTestFiles.value))
const testFileSizeText = computed(() => formatFileSize(testFileSize.value))
const usesChunkedUpload = computed(() => shouldUseChunkedUpload(rawTestFiles.value))

const submitLabel = computed(() => {
  if (teamScope.value) return '创建团队私有题目'
  if (auth.role === 'admin') return '创建并发布'
  if (reviewStatus.value === 'pending') return '等待管理员审核'
  if (reviewStatus.value === 'rejected' || reviewStatus.value === 'withdrawn') return '重新提交审核'
  return '提交管理员审核'
})

function normalizeStatement(value: string) {
  return value.replace(/(```[^\r\n]*\r?\n)(?:[ \t]*\r?\n)+/g, '$1')
}

function syncTestFiles(_file: any, fileList: any[]) {
  const size = totalTestFileSize(fileList.map((item) => item.raw).filter((file): file is File => file instanceof File))
  if (size > MAX_TEST_UPLOAD_SIZE) {
    ElMessage.error('测试点文件总大小不能超过 128MB')
    testFiles.value = fileList.slice(0, -1)
    return
  }
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
    form.statement = normalizeStatement(draft.statement || '')
    form.time_limit_ms = draft.time_limit_ms || 1000
    form.memory_limit_mb = draft.memory_limit_mb || 256
    form.output_limit_kb = draft.output_limit_kb || 1024
    importedCases.value = Array.isArray(draft.cases)
      ? draft.cases.map((item: any, index: number) => ({
          name: item.name || `case-${String(index + 1).padStart(2, '0')}`,
          input: item.input || '',
          output: item.output || '',
          weight: item.weight || Math.max(1, Math.floor(100 / Math.max(1, draft.cases.length)))
        }))
      : []
    persistDraft()
    const extra = Math.max(0, (data.problems?.length || 1) - 1)
    ElMessage.success(extra ? `已载入第一道题，文档中另有 ${extra} 道题` : 'MD 文档已载入')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    importingMarkdown.value = false
  }
}

async function submitProblem() {
  if (!form.title.trim()) {
    ElMessage.error('请输入题目名称')
    return
  }
  if (!form.statement.trim()) {
    ElMessage.error('请输入题面')
    return
  }
  const resubmitting = (reviewStatus.value === 'rejected' || reviewStatus.value === 'withdrawn') && Boolean(cachedProblemId.value)
  if (!resubmitting && !testFiles.value.length && !importedCases.value.length) {
    ElMessage.error('请导入测试点压缩包、IN/OUT 文件，或从 MD 文档载入测试点')
    return
  }
  saving.value = true
  try {
    form.statement = normalizeStatement(form.statement)
    const draft = {
      title: form.title.trim(),
      statement: form.statement,
      difficulty: form.difficulty,
      tags: form.tags,
      team_id: teamScope.value ? teamID.value : undefined,
      problem_set_id: teamScope.value ? problemSetID.value : undefined,
      time_limit_ms: form.time_limit_ms,
      memory_limit_mb: form.memory_limit_mb,
      output_limit_kb: form.output_limit_kb,
      checker: {
        type: form.checker_type,
        absolute_tolerance: form.checker_type === 'float' ? form.absolute_tolerance : 0,
        relative_tolerance: form.checker_type === 'float' ? form.relative_tolerance : 0
      },
      cases: importedCases.value
    }
    const endpoint = resubmitting ? `/problems/${cachedProblemId.value}` : '/problems'
    const request = testFiles.value.length
      ? usesChunkedUpload.value && !resubmitting
        ? buildChunkedRequest(endpoint, draft)
        : buildMultipartRequest(endpoint, draft, resubmitting)
      : resubmitting
        ? client.put(endpoint, draft)
        : client.post(endpoint, draft)
    const { data } = await request
    if (teamScope.value) {
      localStorage.removeItem(draftKey.value)
      ElMessage.success('团队私有题目已创建并加入题单')
      await router.push(`/problem-set/${problemSetID.value}#problems`)
      return
    }
    if (auth.role === 'admin') {
      localStorage.removeItem(draftKey.value)
      ElMessage.success('题目已创建并发布')
      await router.push(`/problems/${encodeURIComponent(data.display_code || String(data.id))}`)
      return
    }
    cachedProblemId.value = data.id
    cachedProblemCode.value = data.display_code || `#${data.id}`
    reviewStatus.value = 'pending'
    reviewNote.value = ''
    testFiles.value = []
    persistDraft()
    ElMessage.success('题目已提交管理员审核，草稿会继续保留')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
    uploadStage.value = ''
    uploadProgress.value = 0
  }
}

function buildMultipartRequest(endpoint: string, draft: Record<string, unknown>, updating: boolean) {
  const payload = new FormData()
  payload.append('draft', JSON.stringify(draft))
  testFiles.value.forEach((item) => { if (item.raw) payload.append('test_files', item.raw) })
  uploadStage.value = '正在上传测试点文件'
  uploadProgress.value = 0
  const config = {
    timeout: 300_000,
    onUploadProgress: (event: { loaded: number; total?: number }) => {
      if (event.total) uploadProgress.value = Math.min(100, Math.round((event.loaded / event.total) * 100))
    }
  }
  return updating ? client.put(endpoint, payload, config) : client.post(endpoint, payload, config)
}

async function buildChunkedRequest(endpoint: string, draft: Record<string, unknown>) {
  uploadStage.value = '正在分块上传测试点文件'
  uploadProgress.value = 0
  const testUploads = await uploadTestFilesInChunks(rawTestFiles.value, (percent) => { uploadProgress.value = percent })
  uploadStage.value = '上传完成，正在解析并保存题目'
  return client.post(endpoint, { ...draft, test_uploads: testUploads }, { timeout: 300_000 })
}

function persistDraft() {
  try {
    localStorage.setItem(draftKey.value, JSON.stringify({
      ...form,
      cases: importedCases.value,
      problem_id: cachedProblemId.value,
      problem_code: cachedProblemCode.value,
      review_status: reviewStatus.value,
      review_note: reviewNote.value
    }))
  } catch {
    // A very large imported document may exceed browser storage; server data remains authoritative.
  }
}

function restoreDraft() {
  try {
    const cached = JSON.parse(localStorage.getItem(draftKey.value) || 'null')
    if (!cached) return
    form.title = cached.title || ''
    form.statement = normalizeStatement(cached.statement || '')
    form.difficulty = cached.difficulty || '入门'
    form.tags = Array.isArray(cached.tags) ? cached.tags.map(String) : String(cached.tags || '').split(/[\s,，、]+/).filter(Boolean)
    form.time_limit_ms = Number(cached.time_limit_ms) || 1000
    form.memory_limit_mb = Number(cached.memory_limit_mb) || 256
    form.output_limit_kb = Number(cached.output_limit_kb) || 1024
    form.checker_type = ['exact', 'tokens', 'float'].includes(cached.checker_type) ? cached.checker_type : 'exact'
    form.absolute_tolerance = checkerTolerance(cached.absolute_tolerance)
    form.relative_tolerance = checkerTolerance(cached.relative_tolerance)
    importedCases.value = Array.isArray(cached.cases) ? cached.cases : []
    cachedProblemId.value = cached.problem_id ? Number(cached.problem_id) : undefined
    cachedProblemCode.value = cached.problem_code || ''
    reviewStatus.value = cached.review_status || ''
    reviewNote.value = cached.review_note || ''
  } catch {
    localStorage.removeItem(draftKey.value)
  }
}

async function loadReviewState() {
  if (auth.role === 'admin' || teamScope.value) return
  try {
    const { data } = await client.get<ProblemReview[]>('/problem-reviews/mine')
    const review = cachedProblemId.value
      ? data.find((item) => item.problem_id === cachedProblemId.value)
      : data.find((item) => item.status === 'rejected' || item.status === 'withdrawn' || item.status === 'pending')
    if (!review) return
    if (review.status === 'approved') {
      localStorage.removeItem(draftKey.value)
      form.title = ''
      form.statement = ''
      form.difficulty = '入门'
      form.tags = []
      form.time_limit_ms = 1000
      form.memory_limit_mb = 256
      form.output_limit_kb = 1024
      form.checker_type = 'exact'
      form.absolute_tolerance = 0.000001
      form.relative_tolerance = 0.000001
      importedCases.value = []
      cachedProblemId.value = undefined
      cachedProblemCode.value = ''
      reviewStatus.value = ''
      reviewNote.value = ''
      return
    }
    cachedProblemId.value = review.problem_id
    cachedProblemCode.value = review.problem.display_code || `#${review.problem_id}`
    reviewStatus.value = review.status
    reviewNote.value = review.review_note || ''
    if (!form.title && review.problem) {
      form.title = review.problem.title || ''
      form.statement = normalizeStatement(review.problem.statement || '')
      form.difficulty = review.problem.difficulty || '入门'
      form.tags = problemTags(review.problem.tags)
      form.time_limit_ms = review.problem.time_limit_ms || 1000
      form.memory_limit_mb = review.problem.memory_limit_mb || 256
      form.output_limit_kb = review.problem.output_limit_kb || 1024
      const checker = problemChecker(review.problem.manifest)
      form.checker_type = checker.type
      form.absolute_tolerance = checker.absolute_tolerance
      form.relative_tolerance = checker.relative_tolerance
    }
    persistDraft()
  } catch {
    // The draft can still be edited offline; submission will surface authorization errors.
  }
}

function problemTags(tags?: Record<string, unknown>) {
  const labels = tags?.labels
  return Array.isArray(labels) ? labels.map(String) : []
}

function problemChecker(manifest?: Record<string, unknown>) {
  const raw = (manifest?.checker || {}) as Record<string, unknown>
  const type = ['exact', 'tokens', 'float'].includes(String(raw.type)) ? String(raw.type) : 'exact'
  return {
    type,
    absolute_tolerance: checkerTolerance(raw.absolute_tolerance),
    relative_tolerance: checkerTolerance(raw.relative_tolerance)
  }
}

function checkerTolerance(value: unknown) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0.000001
}

function goBack() {
  if (teamScope.value) {
    router.push(`/problem-set/${problemSetID.value}#problems`)
    return
  }
  router.push('/problems')
}

async function clearDraft() {
  try {
    await ElMessageBox.confirm('确认清除当前浏览器中的出题草稿？服务器中的待审或退回题目不会被删除。', '清除草稿', {
      type: 'warning',
      confirmButtonText: '清除',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }
  localStorage.removeItem(draftKey.value)
  form.title = ''
  form.statement = ''
  form.difficulty = '入门'
  form.tags = []
  form.time_limit_ms = 1000
  form.memory_limit_mb = 256
  form.output_limit_kb = 1024
  form.checker_type = 'exact'
  form.absolute_tolerance = 0.000001
  form.relative_tolerance = 0.000001
  importedCases.value = []
  testFiles.value = []
  cachedProblemId.value = undefined
  cachedProblemCode.value = ''
  reviewStatus.value = ''
  reviewNote.value = ''
}

watch(form, persistDraft, { deep: true })
watch(importedCases, persistDraft, { deep: true })
onMounted(async () => {
  restoreDraft()
  await loadReviewState()
})
</script>

<style scoped>
.problem-create-page { padding: 22px 20px 50px; }
.create-container { width: min(1180px, 100%); margin: 0 auto; }
.create-heading { display: flex; align-items: end; justify-content: space-between; gap: 24px; padding: 34px 0 24px; }
.create-heading h1 { margin: 6px 0; font-size: 30px; }
.create-heading p { margin: 0; color: var(--muted); }
.review-alert { margin-top: 12px; }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .16em; }
.authoring-grid { display: grid; grid-template-columns: minmax(0, 1.55fr) minmax(300px, .65fr); gap: 16px; margin-top: 16px; }
.form-panel { grid-row: span 2; }
.side-column { display: grid; align-content: start; gap: 16px; }
.section-title { display: flex; align-items: baseline; justify-content: space-between; gap: 14px; margin-bottom: 16px; }
.section-title h2 { margin: 0; font-size: 19px; }
.section-title span { color: var(--muted); font-size: 12px; }
.preview-title { margin-top: 22px; padding-top: 20px; border-top: 1px solid var(--border); }
.statement-preview { min-height: 140px; padding: 16px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface-strong); }
.two-columns { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.unit { margin-left: 8px; color: var(--muted); }
.limits-panel :deep(.el-input-number) { width: calc(100% - 38px); }
.checker-note { margin: -4px 0 0; color: var(--muted); font-size: 12px; line-height: 1.6; }
.import-panel { display: grid; gap: 12px; }
.upload-title { margin-bottom: 6px; font-weight: 700; }
.cache-note { margin: 0; color: var(--muted); font-size: 12px; line-height: 1.65; }
.upload-progress { display: grid; gap: 8px; color: var(--muted); font-size: 12px; }
.create-footer { position: sticky; bottom: 0; z-index: 5; display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-top: 18px; padding: 14px; border: 1px solid var(--border); border-radius: 12px; background: var(--glass); backdrop-filter: blur(14px); }
.create-footer > div { display: flex; gap: 10px; }
.draft-state { color: var(--muted); font-size: 12px; }
@media (max-width: 900px) {
  .authoring-grid { grid-template-columns: 1fr; }
  .form-panel { grid-row: auto; }
}
@media (max-width: 640px) {
  .problem-create-page { padding: 16px 12px 36px; }
  .create-heading, .create-footer { align-items: stretch; flex-direction: column; }
  .two-columns { grid-template-columns: 1fr; }
  .create-footer > div { flex-wrap: wrap; }
}
</style>
