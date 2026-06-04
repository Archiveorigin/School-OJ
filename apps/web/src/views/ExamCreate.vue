<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>新建考试</h2>
        <p class="muted">组卷时可从题库选题，也可以直接创建仅本场考试使用的 Markdown 题目。</p>
      </div>
      <div class="toolbar">
        <el-button @click="router.push('/exams')">返回考试</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">创建考试</el-button>
      </div>
    </div>

    <div class="exam-create-grid">
      <section class="panel">
        <div class="section-title"><h3>考试信息</h3></div>
        <el-form :model="form" label-width="92px">
          <el-form-item label="课程">
            <el-select v-model="form.course_id" style="width: 100%" disabled>
              <el-option v-for="course in courses" :key="course.id" :label="`${course.code} ${course.name}`" :value="course.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="班级">
            <el-select v-model="form.class_id" style="width: 100%" @change="syncCourseFromClass">
              <el-option v-for="item in classroom.classes" :key="item.class_id" :label="`${item.course_code} / ${item.class_name}`" :value="item.class_id" />
            </el-select>
          </el-form-item>
          <el-form-item label="标题">
            <el-input v-model="form.title" placeholder="期中考试" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="form.description" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item label="开始时间">
            <el-date-picker v-model="form.starts_at" type="datetime" style="width: 100%" />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-date-picker v-model="form.ends_at" type="datetime" style="width: 100%" />
          </el-form-item>
          <el-form-item label="阅卷方式">
            <el-checkbox v-model="form.manual_review">提交后判题，教师人工确认分数</el-checkbox>
          </el-form-item>
          <el-form-item label="考试退出">
            <span class="muted">学生进入考试后必须点击“结束考试”才能退出，结束后不能再次进入。</span>
          </el-form-item>
        </el-form>
      </section>

      <section class="panel">
        <div class="section-title">
          <h3>已选题目</h3>
          <strong>总分：{{ selectedTotalScore }}</strong>
        </div>
        <el-table :data="selectedProblems" size="small" class="selected-table">
          <el-table-column label="题号" width="110">
            <template #default="{ row }">
              <el-input v-model="row.label" maxlength="16" />
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="90" />
          <el-table-column prop="title" label="题目" min-width="180" />
          <el-table-column label="分值" width="150">
            <template #default="{ row }">
              <el-input-number v-model="row.score" :min="1" :max="1000" />
            </template>
          </el-table-column>
          <el-table-column label="发布" width="130">
            <template #default="{ row }">
              <el-tag v-if="row.release_after_exam" type="warning" effect="light">结束后同步</el-tag>
              <span v-else class="muted">已在题库</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90">
            <template #default="{ $index }">
              <el-button size="small" text type="danger" @click="selectedProblems.splice($index, 1)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </section>
    </div>

    <section class="panel">
      <div class="section-title"><h3>添加题目</h3></div>
      <el-tabs v-model="problemSource">
        <el-tab-pane label="班级题库" name="class">
          <div class="problem-add">
            <el-select v-model="problemPickID" filterable placeholder="选择题目">
              <el-option v-for="option in classProblemOptions" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
            <el-button @click="addSelectedProblem('class')">添加</el-button>
          </div>
        </el-tab-pane>
        <el-tab-pane label="预备题库" name="prepared">
          <div class="problem-add">
            <el-select v-model="problemPickID" filterable placeholder="选择预备题">
              <el-option v-for="option in preparedProblemOptions" :key="option.value" :label="option.label" :value="option.value" />
            </el-select>
            <el-button @click="addSelectedProblem('prepared')">添加</el-button>
          </div>
          <p class="muted form-note">预备题会在考试结束时间后自动同步到当前班级题库。</p>
        </el-tab-pane>
        <el-tab-pane label="Markdown 出题" name="markdown">
          <el-form label-width="92px" class="problem-form">
            <el-row :gutter="12">
              <el-col :span="6">
                <el-form-item label="题号">
                  <el-input v-model="problemForm.label" maxlength="16" />
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="分值">
                  <el-input-number v-model="problemForm.score" :min="1" :max="1000" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Slug">
                  <el-input v-model="problemForm.slug" placeholder="exam-problem-a" @input="slugManuallyEdited = true" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="标题">
              <el-input v-model="problemForm.title" placeholder="两数之和" @input="syncSlugFromTitle" />
            </el-form-item>
            <el-form-item label="题面">
              <el-input
                v-model="problemForm.statement"
                type="textarea"
                :rows="9"
                placeholder="支持 Markdown、LaTeX 和图片。例如：![示意图](assets/example.png)"
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
                <span class="muted">图片会自动写入题面 Markdown，单张不超过 5 MB。</span>
              </div>
              <div v-if="problemForm.assets.length" class="asset-row">
                <el-tag v-for="asset in problemForm.assets" :key="asset.path" closable @close="removeProblemImage(asset.path)">
                  {{ asset.name }}
                </el-tag>
              </div>
              <div class="statement-preview">
                <div class="muted">题面预览</div>
                <MarkdownRenderer :source="problemForm.statement || '支持 **Markdown** 和 $a+b$。'" :asset-urls="problemAssetPreviewUrls" />
                <div v-if="statementSamples.length" class="preview-samples">
                  <div v-for="sample in statementSamples" :key="sample.index" class="preview-sample-pair">
                    <div class="preview-sample">
                      <div class="sample-head">
                        <strong>输入样例 {{ sample.index }}</strong>
                        <el-button size="small" text @click="copyText(sample.input)">复制</el-button>
                      </div>
                      <pre>{{ sample.input }}</pre>
                    </div>
                    <div class="preview-sample">
                      <div class="sample-head">
                        <strong>输出样例 {{ sample.index }}</strong>
                        <el-button size="small" text @click="copyText(sample.output)">复制</el-button>
                      </div>
                      <pre>{{ sample.output }}</pre>
                    </div>
                  </div>
                </div>
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
            <el-form-item label="隐藏测试点">
              <div class="test-file-panel">
                <el-upload
                  :key="testPointUploadKey"
                  drag
                  action="#"
                  multiple
                  accept=".zip,.in,.out"
                  :auto-upload="false"
                  :file-list="testPointUploadFiles"
                  :on-change="syncTestPointFiles"
                  :on-remove="syncTestPointFiles"
                >
                  <div class="upload-text">选择或拖入 .zip / .in / .out 测试点文件</div>
                  <div class="muted">按文件名中的数字序号配对，例如 data1.in 与 answer1.out；ZIP 会在上传后解析，这些文件不会展示给考生。</div>
                </el-upload>
                <div v-if="testPointErrors.length" class="test-errors">
                  <el-alert
                    v-for="error in testPointErrors"
                    :key="error"
                    type="error"
                    :title="error"
                    show-icon
                    :closable="false"
                  />
                </div>
                <el-table v-if="problemForm.cases.length" :data="problemForm.cases" size="small" class="test-case-table">
                  <el-table-column prop="name" label="测试点" min-width="120" />
                  <el-table-column label="输入文件" min-width="180">
                    <template #default="{ row }">{{ row.inputName }} · {{ formatBytes(row.inputSize) }}</template>
                  </el-table-column>
                  <el-table-column label="输出文件" min-width="180">
                    <template #default="{ row }">{{ row.outputName }} · {{ formatBytes(row.outputSize) }}</template>
                  </el-table-column>
                  <el-table-column prop="weight" label="权重" width="80" />
                </el-table>
                <p v-else-if="testPointUploadFiles.length" class="muted form-note">已选择 ZIP 文件，上传后将由后端解析并按数字序号配对。</p>
                <p v-else class="muted form-note">请至少上传一组完整的 .in / .out 测试点文件，或上传一个测试点 ZIP。</p>
              </div>
            </el-form-item>
            <div class="toolbar form-actions">
              <el-button @click="resetProblemForm">重置出题表单</el-button>
              <el-button type="primary" :loading="creatingProblem" @click="createMarkdownProblem">创建并加入考试</el-button>
            </div>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </section>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type PreparedProblem, type Problem } from '../api/client'
import MarkdownRenderer from '../components/MarkdownRenderer.vue'
import { extractStatementSamples, problemDisplayCode, tagList } from '../features/problems/problemMeta'
import { useClassroomStore } from '../stores/classroom'

type ProblemAssetForm = { name: string; path: string; content_type: string; data: string; preview_url: string }
type ProblemCaseForm = { name: string; inputName: string; inputSize: number; outputName: string; outputSize: number; weight: number }
type SelectedProblem = {
  problem_id: number
  title: string
  source: string
  score: number
  label: string
  release_after_exam?: boolean
}

const router = useRouter()
const classroom = useClassroomStore()
const courses = ref<any[]>([])
const problems = ref<Problem[]>([])
const preparedProblems = ref<PreparedProblem[]>([])
const selectedProblems = ref<SelectedProblem[]>([])
const saving = ref(false)
const creatingProblem = ref(false)
const problemSource = ref<'class' | 'prepared' | 'markdown'>('class')
const problemPickID = ref<number>()
const slugManuallyEdited = ref(false)
const testPointUploadFiles = ref<any[]>([])
const testPointUploadKey = ref(0)
const testPointErrors = ref<string[]>([])
const readingTestPoints = ref(false)

const form = reactive<any>({
  course_id: undefined,
  class_id: undefined,
  title: '',
  description: '',
  starts_at: null,
  ends_at: null,
  manual_review: false
})

const problemForm = reactive({
  label: 'A',
  score: 100,
  slug: `exam-problem-${Date.now()}`,
  title: '',
  statement: '',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  assets: [] as ProblemAssetForm[],
  cases: [] as ProblemCaseForm[]
})

const selectedTotalScore = computed(() => selectedProblems.value.reduce((sum, item) => sum + Number(item.score || 0), 0))
const problemAssetPreviewUrls = computed(() => Object.fromEntries(problemForm.assets.map((asset) => [asset.path, asset.preview_url])))
const statementSamples = computed(() => extractStatementSamples(problemForm.statement))
const classProblemOptions = computed(() => {
  return problems.value.map((problem) => ({ value: problem.id, label: `[题库] ${problemDisplayCode(problem)}. ${problem.title}`, title: problem.title, source: '题库' }))
})
const preparedProblemOptions = computed(() => {
  return preparedProblems.value.map((item) => {
    const tags = tagList(item.problem?.tags)
    const suffix = [item.folder, item.difficulty, tags.join('/')].filter(Boolean).join(' · ')
    const code = item.problem ? problemDisplayCode(item.problem) : item.problem_id
    return { value: item.problem_id, label: `[预备] ${code}. ${item.problem?.title}${suffix ? `（${suffix}）` : ''}`, title: item.problem?.title, source: '预备' }
  })
})

async function load() {
  const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
  const [coursesRes, problemsRes, preparedRes] = await Promise.all([
    client.get('/courses'),
    client.get('/problems', { params }),
    client.get('/prepared-problems')
  ])
  courses.value = coursesRes.data
  problems.value = problemsRes.data
  preparedProblems.value = preparedRes.data
}

function syncCourseFromClass() {
  const item = classroom.classes.find((entry) => entry.class_id === form.class_id)
  form.course_id = item?.course_id
  selectedProblems.value = []
  problemPickID.value = undefined
  loadClassProblems()
}

async function loadClassProblems() {
  if (!form.class_id) return
  problems.value = (await client.get('/problems', { params: { class_id: form.class_id } })).data
}

function addSelectedProblem(source: 'class' | 'prepared') {
  const options = source === 'prepared' ? preparedProblemOptions.value : classProblemOptions.value
  const option = options.find((item) => item.value === problemPickID.value)
  if (!option) return
  if (selectedProblems.value.some((item) => item.problem_id === option.value)) {
    ElMessage.warning('该题已经加入考试')
    return
  }
  selectedProblems.value.push({
    problem_id: option.value,
    title: option.title || option.label,
    source: option.source,
    score: 100,
    label: nextAvailableLabel(),
    release_after_exam: source === 'prepared'
  })
  problemPickID.value = undefined
  problemForm.label = nextAvailableLabel()
}

async function createMarkdownProblem() {
  if (!problemForm.label.trim() || !problemForm.title.trim() || !problemForm.slug.trim()) {
    ElMessage.error('请填写题号、Slug 和标题')
    return
  }
  if (selectedProblems.value.some((item) => item.label.trim().toLowerCase() === problemForm.label.trim().toLowerCase())) {
    ElMessage.error('题号不能重复')
    return
  }
  if (readingTestPoints.value) {
    ElMessage.warning('测试点文件仍在读取中，请稍后再试')
    return
  }
  if (testPointErrors.value.length > 0) {
    ElMessage.error('请先修正测试点文件错误')
    return
  }
  if (testPointUploadFiles.value.length === 0) {
    ElMessage.error('请至少上传一组完整的 .in / .out 测试点文件，或上传一个测试点 ZIP')
    return
  }
  creatingProblem.value = true
  try {
    const fd = new FormData()
    fd.append('draft', JSON.stringify({
      slug: problemForm.slug,
      title: problemForm.title,
      statement: problemForm.statement,
      time_limit_ms: problemForm.time_limit_ms,
      memory_limit_mb: problemForm.memory_limit_mb,
      output_limit_kb: problemForm.output_limit_kb,
      class_ids: [],
      assets: problemForm.assets.map(({ name, path, content_type, data }) => ({ name, path, content_type, data }))
    }))
    for (const item of testPointUploadFiles.value) {
      const file = item.raw as File | undefined
      if (file) fd.append('tests', file, file.name)
    }
    const { data } = await client.post('/problems', fd, { timeout: 120000 })
    selectedProblems.value.push({
      problem_id: data.id,
      title: data.title,
      source: '出题',
      score: problemForm.score,
      label: problemForm.label.trim(),
      release_after_exam: true
    })
    ElMessage.success('题目已加入考试，考试结束后同步到题库')
    resetProblemForm()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    creatingProblem.value = false
  }
}

async function submitCreate() {
  if (!form.class_id || !form.course_id || !form.title || selectedProblems.value.length === 0) {
    ElMessage.error('请选择班级、填写标题并选择题目')
    return
  }
  if (selectedProblems.value.some((item) => !item.label.trim())) {
    ElMessage.error('每道题都需要填写题号')
    return
  }
  if (hasDuplicateLabels()) {
    ElMessage.error('题号不能重复')
    return
  }
  if (selectedProblems.value.some((item) => item.release_after_exam) && !form.ends_at) {
    ElMessage.error('使用预备题或考试内新建题时必须填写结束时间')
    return
  }
  saving.value = true
  try {
    const { data } = await client.post('/exams', {
      course_id: form.course_id,
      class_id: form.class_id,
      title: form.title,
      description: form.description,
      starts_at: form.starts_at,
      ends_at: form.ends_at,
      manual_review: form.manual_review,
      problems: selectedProblems.value.map((item) => ({
        problem_id: item.problem_id,
        score: item.score,
        label: item.label.trim(),
        release_after_exam: Boolean(item.release_after_exam)
      }))
    })
    ElMessage.success('考试已创建')
    router.push(`/exams/${data.id}`)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

function hasDuplicateLabels() {
  const seen = new Set<string>()
  for (const item of selectedProblems.value) {
    const label = item.label.trim().toLowerCase()
    if (seen.has(label)) return true
    seen.add(label)
  }
  return false
}

function nextAvailableLabel() {
  const used = new Set(selectedProblems.value.map((item) => item.label.trim().toLowerCase()))
  for (let index = 0; index < 702; index += 1) {
    const label = defaultProblemLabel(index)
    if (!used.has(label.toLowerCase())) return label
  }
  return defaultProblemLabel(selectedProblems.value.length)
}

function defaultProblemLabel(index: number) {
  index += 1
  let label = ''
  while (index > 0) {
    index -= 1
    label = String.fromCharCode(65 + (index % 26)) + label
    index = Math.floor(index / 26)
  }
  return label
}

function syncSlugFromTitle() {
  if (slugManuallyEdited.value) return
  problemForm.slug = slugifyTitle(problemForm.title) || `exam-problem-${Date.now()}`
}

function slugifyTitle(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9._-]/g, '')
    .replace(/^-+|-+$/g, '')
}

function resetProblemForm() {
  problemForm.assets.forEach((asset) => URL.revokeObjectURL(asset.preview_url))
  problemForm.label = nextAvailableLabel()
  problemForm.score = 100
  problemForm.slug = `exam-problem-${Date.now()}`
  problemForm.title = ''
  problemForm.statement = ''
  problemForm.time_limit_ms = 1000
  problemForm.memory_limit_mb = 256
  problemForm.output_limit_kb = 1024
  problemForm.assets.splice(0, problemForm.assets.length)
  problemForm.cases.splice(0, problemForm.cases.length)
  testPointUploadFiles.value = []
  testPointUploadKey.value += 1
  testPointErrors.value = []
  slugManuallyEdited.value = false
}

async function syncTestPointFiles(_uploadFile: any, uploadFiles: any[]) {
  testPointUploadFiles.value = [...uploadFiles]
  await rebuildTestCasesFromFiles(testPointUploadFiles.value)
}

async function rebuildTestCasesFromFiles(uploadFiles: any[]) {
  readingTestPoints.value = true
  try {
    const errors: string[] = []
    const groups = new Map<number, { input?: File; output?: File }>()
    const seen = new Set<string>()
    for (const item of uploadFiles) {
      const file = item.raw as File | undefined
      if (!file) continue
      if (file.name.toLowerCase().endsWith('.zip')) {
        continue
      }
      const ext = file.name.toLowerCase().endsWith('.in') ? '.in' : file.name.toLowerCase().endsWith('.out') ? '.out' : ''
      if (!ext) {
        errors.push(`${file.name} 不是 .zip、.in 或 .out 文件`)
        continue
      }
      const base = file.name.slice(0, -ext.length).trim()
      if (!base) {
        errors.push(`${file.name} 缺少测试点名称`)
        continue
      }
      const seq = extractLastNumber(base)
      if (!seq) {
        errors.push(`${file.name} 缺少数字测试点序号`)
        continue
      }
      const key = `${seq}${ext}`
      if (seen.has(key)) {
        errors.push(`${file.name} 与已有第 ${seq} 个${ext}文件重复`)
        continue
      }
      seen.add(key)
      const group = groups.get(seq) || {}
      if (ext === '.in') group.input = file
      else group.output = file
      groups.set(seq, group)
    }
    const cases: ProblemCaseForm[] = []
    for (const [seq, group] of [...groups.entries()].sort(([a], [b]) => a - b)) {
      if (!group.input || !group.output) {
        errors.push(`第 ${seq} 个测试点缺少 ${group.input ? '.out' : '.in'} 配对文件`)
        continue
      }
      cases.push({
        name: `case-${String(cases.length + 1).padStart(2, '0')}`,
        inputName: group.input.name,
        inputSize: group.input.size,
        outputName: group.output.name,
        outputSize: group.output.size,
        weight: 1
      })
    }
    problemForm.cases.splice(0, problemForm.cases.length, ...cases)
    testPointErrors.value = errors
  } catch (err: any) {
    problemForm.cases.splice(0, problemForm.cases.length)
    testPointErrors.value = [err.message || '读取测试点文件失败']
  } finally {
    readingTestPoints.value = false
  }
}

function extractLastNumber(value: string) {
  const match = value.match(/(\d+)(?!.*\d)/)
  if (!match) return 0
  return Number(match[1])
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动选择文本')
  }
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
    problemForm.statement = `${problemForm.statement.trimEnd()}\n\n![${file.name}](${path})\n`
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

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

watch(
  () => classroom.activeClassId,
  async () => {
    form.class_id = classroom.activeClassId || classroom.classes[0]?.class_id
    syncCourseFromClass()
    await load()
  }
)

onMounted(async () => {
  await classroom.load()
  form.class_id = classroom.activeClassId || classroom.classes[0]?.class_id
  syncCourseFromClass()
  await load()
  problemForm.label = nextAvailableLabel()
})
</script>

<style scoped>
.exam-create-grid {
  display: grid;
  grid-template-columns: minmax(320px, 0.8fr) minmax(420px, 1.2fr);
  gap: 14px;
  align-items: start;
  margin-bottom: 14px;
}

.selected-table {
  width: 100%;
}

.problem-add {
  display: grid;
  grid-template-columns: minmax(240px, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.problem-form {
  padding-top: 4px;
}

.form-note {
  margin: 10px 0 0;
}

.form-actions {
  justify-content: flex-end;
}

.unit {
  margin-left: 8px;
  color: #6b7280;
}

.statement-preview {
  width: 100%;
  margin-top: 10px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.03);
}

.preview-samples,
.test-file-panel,
.test-errors {
  display: grid;
  gap: 10px;
  width: 100%;
}

.preview-samples {
  margin-top: 12px;
}

.preview-sample-pair {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.preview-sample {
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--surface);
}

.sample-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
}

.preview-sample pre {
  max-height: 220px;
  margin: 0;
  overflow: auto;
  padding: 10px;
  color: #e2e8f0;
  background: #0f172a;
}

.test-case-table {
  width: 100%;
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

@media (max-width: 980px) {
  .exam-create-grid {
    grid-template-columns: 1fr;
  }

  .preview-sample-pair {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .problem-add {
    grid-template-columns: 1fr;
  }
}
</style>
