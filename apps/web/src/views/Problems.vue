<template>
  <section class="page sub-page">
    <div class="sub-hero">
      <div class="sub-hero-inner">
        <div class="sub-hero-text">
          <h1 class="sub-hero-title">题库</h1>
          <p class="sub-hero-sub">{{ canManage ? '管理题目、上传题目包、发布预备题' : '浏览题目并提交代码' }}</p>
        </div>
        <div class="sub-hero-stats">
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ problems.length }}</span>
            <span class="sub-hero-stat-label">题目总数</span>
          </div>
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ tagOptions.length }}</span>
            <span class="sub-hero-stat-label">标签数</span>
          </div>
          <div v-if="!canManage" class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ solvedCount }}</span>
            <span class="sub-hero-stat-label">已解决</span>
          </div>
        </div>
      </div>
    </div>

    <div class="sub-content">
      <div class="panel-header">
        <div class="toolbar">
          <el-button v-if="canManage" type="primary" @click="openProblemDialog">上传题目包</el-button>
          <el-button v-if="canManage" @click="openPreparedPublish">从预备题库发布</el-button>
          <el-button @click="load">刷新</el-button>
        </div>
      </div>

      <div class="panel problem-filters">
        <el-input v-model="filters.keyword" clearable placeholder="搜索编号、标题、Slug、标签" />
        <el-select v-model="filters.tag" clearable filterable placeholder="标签">
          <el-option v-for="tag in tagOptions" :key="tag" :label="tag" :value="tag" />
        </el-select>
        <el-select v-if="auth.role === 'student'" v-model="filters.status" placeholder="状态">
          <el-option
            v-for="option in problemStatusOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
        <el-button @click="resetFilters">重置</el-button>
        <span class="muted filter-count">{{ filteredProblems.length }} / {{ problems.length }}</span>
      </div>

      <div class="problem-layout">
        <aside class="panel problem-list-panel">
          <el-table :data="filteredProblems" highlight-current-row @current-change="selectProblem" height="calc(100vh - 320px)">
            <el-table-column label="编号" width="88">
              <template #default="{ row }">{{ problemDisplayCode(row) }}</template>
            </el-table-column>
            <el-table-column label="题目" min-width="190">
              <template #default="{ row }">
                <div class="problem-title">{{ row.title }}</div>
                <div class="muted">{{ row.slug }}</div>
                <div v-if="tagList(row.tags).length" class="tag-strip">
                  <el-tag v-for="tag in tagList(row.tags)" :key="tag" size="small">
                    {{ tag }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column v-if="auth.role === 'student'" label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="progressTag(row.progress_status)" effect="light">
                  {{ progressLabel(row.progress_status) }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </aside>
        <main class="panel problem-detail-panel" v-if="selected">
          <div class="detail-head">
            <div>
              <h3>{{ selected.title }}</h3>
              <p class="muted">{{ problemDisplayCode(selected) }} · {{ selected.slug }}</p>
            </div>
            <div class="toolbar">
              <el-button v-if="canManage" type="primary" plain @click="openEditProblem">修改题目</el-button>
              <el-button v-if="canDeleteSelected" type="danger" plain @click="removeProblem">删除题目</el-button>
            </div>
          </div>
          <p class="muted">{{ problemLimitText(selected) }}</p>
          <div v-if="tagList(selected.tags).length" class="tag-strip detail-tags">
            <el-tag v-for="tag in tagList(selected.tags)" :key="tag" size="small">
              {{ tag }}
            </el-tag>
          </div>
          <ProblemTestDownloads v-if="canManage" :problem-id="selected.id" :problem-code="selected.display_code" class="detail-tests" />
          <el-divider v-if="canManage" />
          <MarkdownRenderer :source="selected.statement" :problem-id="selected.id" />
          <el-divider />
          <div class="toolbar">
            <el-select v-model="language" style="width: 130px">
              <el-option label="C++17" value="cpp" />
              <el-option label="C" value="c" />
              <el-option label="Python" value="python" />
              <el-option label="Java" value="java" />
            </el-select>
            <el-button @click="formatSource">自动格式化</el-button>
            <el-button type="primary" :loading="submitting" @click="submit">提交</el-button>
          </div>
          <CodeEditor ref="editorRef" v-model="source" :language="language" />
          <div v-if="live" class="live">
            <StatusBadge :status="live.status" /> 分数 {{ live.score }}，{{ live.message }}
          </div>
        </main>
        <main v-else class="panel empty-detail muted">请选择题目</main>
      </div>
    </div>

    <el-dialog v-model="problemDialogVisible" title="上传题目包" width="920px">
      <el-tabs v-model="problemDialogTab">
        <el-tab-pane label="上传现有 ZIP" name="zip">
          <el-form label-width="110px">
            <el-form-item label="发布班级">
              <el-select v-model="selectedClassIDs" multiple clearable style="width: 100%">
                <el-option
                  v-for="item in classroom.classes"
                  :key="item.class_id"
                  :label="`${item.course_code} / ${item.class_name}`"
                  :value="item.class_id"
                />
              </el-select>
            </el-form-item>
          </el-form>
          <el-upload
            drag
            :show-file-list="false"
            :http-request="upload"
            accept=".zip"
            class="zip-upload"
          >
            <div class="upload-text">选择或拖入题目包 ZIP</div>
            <div class="muted">problem.yaml 的 statement 可使用 Markdown 和 LaTeX 多行文本</div>
          </el-upload>
        </el-tab-pane>
        <el-tab-pane label="表单创建题目" name="form">
          <el-form label-width="110px" class="problem-form">
            <el-form-item label="发布班级">
              <el-select v-model="selectedClassIDs" multiple clearable style="width: 100%">
                <el-option
                  v-for="item in classroom.classes"
                  :key="item.class_id"
                  :label="`${item.course_code} / ${item.class_name}`"
                  :value="item.class_id"
                />
              </el-select>
            </el-form-item>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="Slug">
                  <el-input v-model="problemForm.slug" placeholder="two-sum" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="标题">
                  <el-input v-model="problemForm.title" placeholder="两数之和" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="题面">
              <el-input
                v-model="problemForm.statement"
                type="textarea"
                :rows="8"
                placeholder="支持 Markdown 和 LaTeX"
              />
              <div class="statement-tools">
                <el-upload
                  action="#"
                  :auto-upload="false"
                  :show-file-list="false"
                  multiple
                  accept="image/png,image/jpeg,image/gif,image/webp"
                  :on-change="addProblemImage"
                >
                  <el-button>插入图片</el-button>
                </el-upload>
                <span class="muted">支持 PNG、JPG、GIF、WebP</span>
              </div>
              <div v-if="problemForm.assets.length" class="asset-row">
                <el-tag v-for="asset in problemForm.assets" :key="asset.path" closable @close="removeProblemImage(asset.path)">
                  {{ asset.name }}
                </el-tag>
              </div>
              <div class="statement-preview">
                <div class="muted">题面预览</div>
                <MarkdownRenderer :source="problemForm.statement || '支持 **Markdown** 和 $a+b$。'" :asset-urls="problemAssetPreviewUrls" />
              </div>
            </el-form-item>
            <el-row :gutter="12">
              <el-col :span="8">
                <el-form-item label="时间限制">
                  <el-input-number v-model="problemForm.time_limit_ms" :min="100" :step="100" />
                  <span class="unit">ms</span>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="内存限制">
                  <el-input-number v-model="problemForm.memory_limit_mb" :min="16" :step="16" />
                  <span class="unit">MB</span>
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="输出限制">
                  <el-input-number v-model="problemForm.output_limit_kb" :min="1" :step="64" />
                  <span class="unit">KB</span>
                </el-form-item>
              </el-col>
            </el-row>
            <div class="case-toolbar">
              <h4>测试点</h4>
              <el-button size="small" @click="addCase">添加测试点</el-button>
            </div>
            <div v-for="(item, index) in problemForm.cases" :key="index" class="case-editor">
              <div class="case-head">
                <el-input v-model="item.name" placeholder="测试点名称" />
                <el-input-number v-model="item.weight" :min="1" :max="100" />
                <el-button :disabled="problemForm.cases.length === 1" @click="removeCase(index)">
                  删除
                </el-button>
              </div>
              <el-row :gutter="12">
                <el-col :span="12">
                  <el-input v-model="item.input" type="textarea" :rows="5" placeholder="输入数据" />
                </el-col>
                <el-col :span="12">
                  <el-input v-model="item.output" type="textarea" :rows="5" placeholder="期望输出" />
                </el-col>
              </el-row>
            </div>
          </el-form>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="problemDialogVisible = false">取消</el-button>
        <el-button
          v-if="problemDialogTab === 'form'"
          type="primary"
          :loading="savingProblem"
          @click="createFromForm"
        >
          创建题目
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="preparedPublishVisible" title="从预备题库发布" width="760px">
      <el-form label-width="90px">
        <el-form-item label="预备题">
          <el-select v-model="preparedIDs" multiple filterable style="width: 100%">
            <el-option
              v-for="item in preparedItems"
              :key="item.id"
              :label="preparedLabel(item)"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="班级">
          <el-select v-model="publishClassIDs" multiple filterable style="width: 100%">
            <el-option
              v-for="item in classroom.classes"
              :key="item.class_id"
              :label="`${item.course_code} / ${item.class_name}`"
              :value="item.class_id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <el-table :data="preparedItems" height="260">
        <el-table-column prop="problem.slug" label="Slug" width="140" />
        <el-table-column prop="problem.title" label="题目" />
        <el-table-column prop="folder" label="文件夹" width="130" />
        <el-table-column prop="difficulty" label="难度" width="90" />
      </el-table>
      <template #footer>
        <el-button @click="preparedPublishVisible = false">取消</el-button>
        <el-button type="primary" :loading="publishingPrepared" @click="publishPrepared">
          立即公开
        </el-button>
      </template>
    </el-dialog>
    <ProblemEditDialog v-model="editProblemVisible" :problem="selected" @saved="handleProblemSaved" />
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { client, sseUrl, type PreparedProblem, type Problem } from '../api/client'
import CodeEditor from '../components/CodeEditor.vue'
import MarkdownRenderer from '../components/MarkdownRenderer.vue'
import ProblemEditDialog from '../components/ProblemEditDialog.vue'
import ProblemTestDownloads from '../components/ProblemTestDownloads.vue'
import StatusBadge from '../components/StatusBadge.vue'
import {
  problemDisplayCode,
  problemLimitText,
  problemMatchesFilters,
  problemStatusOptions,
  progressLabel,
  progressTag,
  tagList,
  type ProblemFilters
} from '../features/problems/problemMeta'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

const auth = useAuthStore()
const classroom = useClassroomStore()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
type ProblemAssetForm = { name: string; path: string; content_type: string; data: string; preview_url: string }
const problems = ref<Problem[]>([])
const selected = ref<Problem | null>(null)
const filters = reactive<ProblemFilters>({
  keyword: '',
  tag: '',
  status: 'all'
})
const canDeleteSelected = computed(() => Boolean(selected.value && (auth.role === 'admin' || selected.value.owner_id === auth.user?.id)))
const language = ref('cpp')
const submitting = ref(false)
const live = ref<any>(null)
const editorRef = ref<InstanceType<typeof CodeEditor> | null>(null)
const problemDialogVisible = ref(false)
const problemDialogTab = ref('zip')
const savingProblem = ref(false)
const editProblemVisible = ref(false)
const selectedClassIDs = ref<number[]>([])
const preparedPublishVisible = ref(false)
const preparedItems = ref<PreparedProblem[]>([])
const preparedIDs = ref<number[]>([])
const publishClassIDs = ref<number[]>([])
const publishingPrepared = ref(false)
const problemForm = reactive({
  slug: '',
  title: '',
  statement: '',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  assets: [] as ProblemAssetForm[],
  cases: [{ name: 'case-01', input: '1 2\n', output: '3\n', weight: 100 }]
})
const problemAssetPreviewUrls = computed(() => {
  return Object.fromEntries(problemForm.assets.map((asset) => [asset.path, asset.preview_url]))
})
const tagOptions = computed(() => {
  const set = new Set(problems.value.flatMap((problem) => tagList(problem.tags)))
  return [...set].sort((a, b) => a.localeCompare(b, 'zh-CN'))
})
const filteredProblems = computed(() => problems.value.filter((problem) => problemMatchesFilters(problem, filters)))
const solvedCount = computed(() => problems.value.filter((p) => p.progress_status === 'accepted').length)
const source = ref(`#include <bits/stdc++.h>
using namespace std;
int main() {
  long long a, b;
  cin >> a >> b;
  cout << a + b << "\\n";
  return 0;
}
`)

async function load() {
  const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
  problems.value = (await client.get('/problems', { params })).data
  selected.value ||= problems.value[0] || null
  if (selected.value && !problems.value.some((item) => item.id === selected.value?.id)) {
    selected.value = problems.value[0] || null
  }
}

function selectProblem(row: Problem | null) {
  if (!row) return
  selected.value = row
  live.value = null
}

function resetFilters() {
  filters.keyword = ''
  filters.tag = ''
  filters.status = 'all'
}

function openProblemDialog() {
  problemDialogVisible.value = true
  problemDialogTab.value = 'zip'
  selectedClassIDs.value = classroom.activeClassId ? [classroom.activeClassId] : []
}

function openEditProblem() {
  if (!selected.value) return
  editProblemVisible.value = true
}

function handleProblemSaved(problem: Problem) {
  const index = problems.value.findIndex((item) => item.id === problem.id)
  if (index >= 0) problems.value[index] = problem
  selected.value = problem
}

async function openPreparedPublish() {
  preparedPublishVisible.value = true
  publishClassIDs.value = classroom.activeClassId ? [classroom.activeClassId] : []
  preparedIDs.value = []
  preparedItems.value = (await client.get('/prepared-problems')).data
}

function preparedLabel(item: PreparedProblem) {
  const tags = tagList(item.problem?.tags)
  const suffix = [item.folder, item.difficulty, tags.join('/')].filter(Boolean).join(' · ')
  const code = item.problem ? problemDisplayCode(item.problem) : '未编号'
  return `${code}. ${item.problem?.title || '未知题目'}${suffix ? `（${suffix}）` : ''}`
}

async function publishPrepared() {
  if (preparedIDs.value.length === 0 || publishClassIDs.value.length === 0) {
    ElMessage.error('请选择预备题和班级')
    return
  }
  publishingPrepared.value = true
  try {
    await Promise.all(
      preparedIDs.value.map((id) =>
        client.post(`/prepared-problems/${id}/publish`, { class_ids: publishClassIDs.value })
      )
    )
    ElMessage.success('已发布到班级题库')
    preparedPublishVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    publishingPrepared.value = false
  }
}

async function removeProblem() {
  if (!selected.value) return
  try {
    await ElMessageBox.confirm('删除后题目将从题库和学生端隐藏，历史提交与报表会保留。确认删除？', '删除题目', { type: 'warning' })
    await client.delete(`/problems/${selected.value.id}`)
    ElMessage.success('题目已下架')
    selected.value = null
    await load()
  } catch (err: any) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function upload(options: any) {
  try {
    const fd = new FormData()
    fd.append('package', options.file)
    selectedClassIDs.value.forEach((id) => fd.append('class_ids', String(id)))
    await client.post('/problems/upload', fd)
    ElMessage.success('题目包已上传')
    problemDialogVisible.value = false
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

function addCase() {
  const next = problemForm.cases.length + 1
  problemForm.cases.push({ name: `case-${String(next).padStart(2, '0')}`, input: '', output: '', weight: 1 })
}

function removeCase(index: number) {
  problemForm.cases.splice(index, 1)
}

async function createFromForm() {
  savingProblem.value = true
  try {
    const { data } = await client.post('/problems', {
      slug: problemForm.slug,
      title: problemForm.title,
      statement: problemForm.statement,
      time_limit_ms: problemForm.time_limit_ms,
      memory_limit_mb: problemForm.memory_limit_mb,
      output_limit_kb: problemForm.output_limit_kb,
      class_ids: selectedClassIDs.value,
      assets: problemForm.assets.map(({ name, path, content_type, data }) => ({ name, path, content_type, data })),
      cases: problemForm.cases
    })
    ElMessage.success('题目已创建')
    problemDialogVisible.value = false
    await load()
    selected.value = data
    resetProblemForm()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingProblem.value = false
  }
}

function resetProblemForm() {
  problemForm.assets.forEach((asset) => URL.revokeObjectURL(asset.preview_url))
  problemForm.slug = ''
  problemForm.title = ''
  problemForm.statement = ''
  problemForm.time_limit_ms = 1000
  problemForm.memory_limit_mb = 256
  problemForm.output_limit_kb = 1024
  problemForm.assets.splice(0, problemForm.assets.length)
  problemForm.cases.splice(0, problemForm.cases.length, {
    name: 'case-01',
    input: '1 2\n',
    output: '3\n',
    weight: 100
  })
}

function addProblemImage(uploadFile: any) {
  const file = uploadFile.raw as File | undefined
  if (!file) return
  if (!['image/png', 'image/jpeg', 'image/gif', 'image/webp'].includes(file.type)) {
    ElMessage.error('仅支持 PNG、JPG、GIF、WebP 图片')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.error('单张图片不能超过 5 MB')
    return
  }
  const path = uniqueAssetPath(file.name)
  const reader = new FileReader()
  reader.onload = () => {
    problemForm.assets.push({
      name: file.name,
      path,
      content_type: file.type,
      data: String(reader.result),
      preview_url: URL.createObjectURL(file)
    })
    const markdown = `![${file.name}](${path})`
    problemForm.statement = `${problemForm.statement.trimEnd()}\n\n${markdown}\n`
  }
  reader.readAsDataURL(file)
}

function removeProblemImage(path: string) {
  const index = problemForm.assets.findIndex((asset) => asset.path === path)
  if (index < 0) return
  URL.revokeObjectURL(problemForm.assets[index].preview_url)
  problemForm.assets.splice(index, 1)
  problemForm.statement = problemForm.statement.replace(new RegExp(`!\\[[^\\]]*\\]\\(${escapeRegExp(path)}\\)\\n?`, 'g'), '').trimEnd()
}

function uniqueAssetPath(name: string) {
  const safe = name
    .trim()
    .replace(/\s+/g, '-')
    .replace(/[^A-Za-z0-9._-]/g, '')
    .replace(/^\.+/, '')
  const fallback = `image-${Date.now()}.png`
  const base = safe || fallback
  let path = `assets/${base}`
  let index = 1
  while (problemForm.assets.some((asset) => asset.path === path)) {
    const dot = base.lastIndexOf('.')
    path = dot > 0 ? `assets/${base.slice(0, dot)}-${index}${base.slice(dot)}` : `assets/${base}-${index}`
    index += 1
  }
  return path
}

function formatSource() {
  editorRef.value?.format()
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

async function submit() {
  if (!selected.value) return
  submitting.value = true
  try {
    const { data } = await client.post('/submissions', {
      problem_id: selected.value.id,
      language: language.value,
      source_code: source.value
    })
    watchSubmission(data.id)
  } finally {
    submitting.value = false
  }
}

function watchSubmission(id: number) {
  const es = new EventSource(sseUrl(`/submissions/${id}/events`))
  es.addEventListener('status', (event) => {
    live.value = JSON.parse((event as MessageEvent).data)
    if (!['queued', 'running'].includes(live.value.status)) es.close()
  })
}

watch(() => classroom.activeClassId, load)

watch(filteredProblems, (list) => {
  if (selected.value && list.some((item) => item.id === selected.value?.id)) return
  selected.value = list[0] || null
})

onMounted(async () => {
  await classroom.load()
  await load()
})
</script>

<style scoped>
.sub-page {
  padding: 0;
  overflow-x: hidden;
}

.sub-hero {
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0a5ea6 100%);
}

.sub-hero-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 36px 40px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.sub-hero-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sub-hero-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  color: #f8fafc;
}

.sub-hero-sub {
  margin: 0;
  font-size: 14px;
  color: rgba(248, 250, 252, 0.6);
}

.sub-hero-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.sub-hero-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 20px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 10px;
  min-width: 80px;
  text-align: center;
  transition: background 0.2s;
}

.sub-hero-stat:hover {
  background: rgba(255, 255, 255, 0.18);
}

.sub-hero-stat-val {
  font-size: 22px;
  font-weight: 700;
  color: #fff;
}

.sub-hero-stat-label {
  font-size: 12px;
  color: rgba(248, 250, 252, 0.55);
}

.sub-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px 20px 32px;
}

.panel-header {
  margin-bottom: 14px;
}

.live {
  margin-top: 12px;
  display: flex;
  gap: 10px;
  align-items: center;
}

.problem-layout {
  display: grid;
  grid-template-columns: minmax(280px, 340px) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.problem-filters {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) 160px minmax(120px, 140px) auto auto;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
}

.problem-title {
  font-weight: 700;
  color: var(--text);
}

.tag-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 6px;
}

.detail-tags {
  margin-bottom: 12px;
}

.detail-tests {
  margin: 12px 0;
}

.filter-count {
  justify-self: end;
  white-space: nowrap;
}

.problem-list-panel,
.problem-detail-panel,
.empty-detail {
  min-height: calc(100vh - 380px);
}

.problem-list-panel {
  padding: 10px;
}

.problem-detail-panel {
  min-width: 0;
}

.detail-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.detail-head h3 {
  margin: 0 0 4px;
}

.detail-head p {
  margin: 0;
}

.empty-detail {
  display: grid;
  place-items: center;
}

.zip-upload {
  width: 100%;
}

.upload-text {
  font-weight: 600;
  margin-bottom: 6px;
}

.problem-form {
  max-height: 60vh;
  overflow: auto;
  padding-right: 8px;
}

.unit {
  margin-left: 8px;
  color: #6b7280;
}

.case-toolbar,
.case-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.case-toolbar {
  margin: 8px 0 10px;
}

.case-toolbar h4 {
  margin: 0;
}

.case-editor {
  border: 1px solid #d9dee8;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}

.case-head {
  margin-bottom: 10px;
}

.statement-preview {
  width: 100%;
  margin-top: 10px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.03);
}

.statement-tools {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  width: 100%;
  margin-top: 10px;
}

.asset-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  width: 100%;
  margin-top: 8px;
}

@media (max-width: 1100px) {
  .problem-layout {
    grid-template-columns: 1fr;
  }

  .problem-list-panel,
  .problem-detail-panel,
  .empty-detail {
    min-height: auto;
  }
}

@media (max-width: 760px) {
  .sub-hero-inner {
    padding: 24px 20px 32px;
    gap: 16px;
  }

  .problem-filters {
    grid-template-columns: 1fr;
  }

  .filter-count {
    justify-self: start;
  }
}
</style>
