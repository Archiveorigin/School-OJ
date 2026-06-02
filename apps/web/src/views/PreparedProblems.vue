<template>
  <section class="page">
    <div class="page-header">
      <h2>预备题库</h2>
      <div class="toolbar">
        <el-button type="primary" @click="openCreateDialog">上传预备题</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel filters">
      <el-input v-model="filters.q" clearable placeholder="搜索题目、文件夹、来源" />
      <el-select v-model="filters.folder" clearable filterable placeholder="文件夹">
        <el-option v-for="folder in folderOptions" :key="folder" :label="folder" :value="folder" />
      </el-select>
      <el-input v-model="filters.tag" clearable placeholder="标签" />
      <el-select v-model="filters.difficulty" clearable placeholder="难度">
        <el-option label="入门" value="入门" />
        <el-option label="简单" value="简单" />
        <el-option label="中等" value="中等" />
        <el-option label="困难" value="困难" />
        <el-option label="挑战" value="挑战" />
      </el-select>
      <el-select v-model="filters.archived" placeholder="归档">
        <el-option label="未归档" value="false" />
        <el-option label="已归档" value="true" />
        <el-option label="全部" value="all" />
      </el-select>
      <el-button @click="load">筛选</el-button>
    </div>

    <div class="prepared-layout">
      <section class="panel prepared-list-panel">
          <el-table :data="items" highlight-current-row @current-change="selectItem">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column label="题目" min-width="210">
              <template #default="{ row }">
                <div class="problem-title">{{ row.problem?.title }}</div>
                <div class="muted">{{ row.problem?.slug }}</div>
              </template>
            </el-table-column>
            <el-table-column prop="folder" label="文件夹" width="140" />
            <el-table-column prop="difficulty" label="难度" width="90" />
            <el-table-column label="标签" min-width="160">
              <template #default="{ row }">
                <el-tag v-for="tag in tagList(row.problem?.tags)" :key="tag" size="small" class="tag">
                  {{ tag }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="source" label="来源" width="130" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.archived ? 'info' : 'success'" effect="light">
                  {{ row.archived ? '已归档' : '可用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="220" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click.stop="openPublishDialog(row)">发布</el-button>
                <el-button size="small" @click.stop="openEditDialog(row)">分类</el-button>
                <el-button size="small" @click.stop="toggleArchive(row)">
                  {{ row.archived ? '恢复' : '归档' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
      </section>
      <aside v-if="selected" class="panel detail">
          <div class="detail-head">
            <div>
              <h3>{{ selected.problem.title }}</h3>
              <p class="muted">{{ selected.problem.slug }}</p>
            </div>
            <el-tag :type="selected.archived ? 'info' : 'success'">
              {{ selected.archived ? '已归档' : '可发布' }}
            </el-tag>
          </div>
          <div class="meta-grid">
            <span>文件夹</span><strong>{{ selected.folder || '-' }}</strong>
            <span>难度</span><strong>{{ selected.difficulty || '-' }}</strong>
            <span>来源</span><strong>{{ selected.source || '-' }}</strong>
            <span>限制</span>
            <strong>
              {{ selected.problem.time_limit_ms }} ms /
              {{ selected.problem.memory_limit_mb }} MB
            </strong>
          </div>
          <el-divider />
          <MarkdownRenderer :source="selected.problem.statement" :problem-id="selected.problem.id" />
          <el-divider />
          <div class="tag-row">
            <el-tag v-for="tag in tagList(selected.problem.tags)" :key="tag" size="small">{{ tag }}</el-tag>
          </div>
          <p v-if="selected.notes" class="notes">{{ selected.notes }}</p>
      </aside>
    </div>

    <el-dialog v-model="createVisible" title="上传预备题" width="940px">
      <el-form label-width="96px" class="meta-form">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="文件夹">
              <el-input v-model="metaForm.folder" placeholder="例如：动态规划 / 期末复习" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="难度">
              <el-select v-model="metaForm.difficulty" clearable style="width: 100%">
                <el-option label="入门" value="入门" />
                <el-option label="简单" value="简单" />
                <el-option label="中等" value="中等" />
                <el-option label="困难" value="困难" />
                <el-option label="挑战" value="挑战" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="来源">
              <el-input v-model="metaForm.source" placeholder="自编 / OJ" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="标签">
          <el-input v-model="metaForm.tagsText" placeholder="用逗号分隔，例如：数组, 入门, 模拟" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="metaForm.notes" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>

      <el-tabs v-model="createTab">
        <el-tab-pane label="上传现有 ZIP" name="zip">
          <el-upload
            drag
            :show-file-list="false"
            :http-request="uploadZip"
            accept=".zip"
            class="zip-upload"
          >
            <div class="upload-text">选择或拖入题目包 ZIP</div>
            <div class="muted">problem.yaml 的 statement 可使用 Markdown 和 LaTeX 多行文本</div>
          </el-upload>
        </el-tab-pane>
        <el-tab-pane label="表单创建题目" name="form">
          <el-form label-width="96px" class="problem-form">
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
                placeholder="支持 Markdown 和 LaTeX，例如：**加粗**、`代码`、$a+b$、$$\\sum_i a_i$$"
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
                <span class="muted">支持 PNG、JPG、GIF、WebP，图片会自动写入题面 Markdown。</span>
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
        <el-button @click="createVisible = false">取消</el-button>
        <el-button v-if="createTab === 'form'" type="primary" :loading="saving" @click="createFromForm">
          创建预备题
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" title="分类信息" width="560px">
      <el-form label-width="82px">
        <el-form-item label="文件夹"><el-input v-model="editForm.folder" /></el-form-item>
        <el-form-item label="难度">
          <el-select v-model="editForm.difficulty" clearable style="width: 100%">
            <el-option label="入门" value="入门" />
            <el-option label="简单" value="简单" />
            <el-option label="中等" value="中等" />
            <el-option label="困难" value="困难" />
            <el-option label="挑战" value="挑战" />
          </el-select>
        </el-form-item>
        <el-form-item label="来源"><el-input v-model="editForm.source" /></el-form-item>
        <el-form-item label="标签"><el-input v-model="editForm.tagsText" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="editForm.notes" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="publishVisible" title="发布到班级题库" width="520px">
      <el-form label-width="96px">
        <el-form-item label="题目">
          <strong>{{ publishing?.problem.title }}</strong>
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
      <template #footer>
        <el-button @click="publishVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="publish">立即公开</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { client, type PreparedProblem } from '../api/client'
import MarkdownRenderer from '../components/MarkdownRenderer.vue'
import { useClassroomStore } from '../stores/classroom'

const classroom = useClassroomStore()
type ProblemAssetForm = { name: string; path: string; content_type: string; data: string; preview_url: string }
const items = ref<PreparedProblem[]>([])
const selected = ref<PreparedProblem | null>(null)
const createVisible = ref(false)
const editVisible = ref(false)
const publishVisible = ref(false)
const createTab = ref('zip')
const saving = ref(false)
const publishing = ref<PreparedProblem | null>(null)
const publishClassIDs = ref<number[]>([])

const filters = reactive({
  q: '',
  folder: '',
  tag: '',
  difficulty: '',
  archived: 'false'
})

const metaForm = reactive({
  folder: '',
  difficulty: '',
  source: '',
  tagsText: '',
  notes: ''
})

const editForm = reactive({
  id: 0,
  folder: '',
  difficulty: '',
  source: '',
  tagsText: '',
  notes: ''
})

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

const folderOptions = computed(() => {
  const set = new Set(items.value.map((item) => item.folder).filter(Boolean) as string[])
  return [...set].sort()
})

async function load() {
  const params: Record<string, string> = { archived: filters.archived }
  if (filters.q) params.q = filters.q
  if (filters.folder) params.folder = filters.folder
  if (filters.tag) params.tag = filters.tag
  if (filters.difficulty) params.difficulty = filters.difficulty
  items.value = (await client.get('/prepared-problems', { params })).data
  if (!selected.value || !items.value.some((item) => item.id === selected.value?.id)) {
    selected.value = items.value[0] || null
  }
}

function selectItem(row: PreparedProblem) {
  selected.value = row
}

function openCreateDialog() {
  resetMeta()
  resetProblemForm()
  createTab.value = 'zip'
  createVisible.value = true
}

function openEditDialog(row: PreparedProblem) {
  editForm.id = row.id
  editForm.folder = row.folder || ''
  editForm.difficulty = row.difficulty || ''
  editForm.source = row.source || ''
  editForm.tagsText = tagList(row.problem?.tags).join(', ')
  editForm.notes = row.notes || ''
  editVisible.value = true
}

function openPublishDialog(row: PreparedProblem) {
  publishing.value = row
  publishClassIDs.value = classroom.activeClassId ? [classroom.activeClassId] : []
  publishVisible.value = true
}

async function createFromForm() {
  saving.value = true
  try {
    const { data } = await client.post('/prepared-problems', {
      ...metadataPayload(),
      slug: problemForm.slug,
      title: problemForm.title,
      statement: problemForm.statement,
      time_limit_ms: problemForm.time_limit_ms,
      memory_limit_mb: problemForm.memory_limit_mb,
      output_limit_kb: problemForm.output_limit_kb,
      assets: problemForm.assets.map(({ name, path, content_type, data }) => ({ name, path, content_type, data })),
      cases: problemForm.cases
    })
    ElMessage.success('预备题已创建')
    createVisible.value = false
    await load()
    selected.value = data
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

async function uploadZip(options: any) {
  saving.value = true
  try {
    const fd = new FormData()
    fd.append('package', options.file)
    fd.append('folder', metaForm.folder)
    fd.append('difficulty', metaForm.difficulty)
    fd.append('source', metaForm.source)
    fd.append('notes', metaForm.notes)
    parseTags(metaForm.tagsText).forEach((tag) => fd.append('tags', tag))
    const { data } = await client.post('/prepared-problems/upload', fd)
    ElMessage.success('预备题包已上传')
    createVisible.value = false
    await load()
    selected.value = data
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

async function saveEdit() {
  saving.value = true
  try {
    const { data } = await client.put(`/prepared-problems/${editForm.id}`, {
      folder: editForm.folder,
      difficulty: editForm.difficulty,
      source: editForm.source,
      notes: editForm.notes,
      tags: parseTags(editForm.tagsText)
    })
    ElMessage.success('分类已保存')
    editVisible.value = false
    await load()
    selected.value = data
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

async function toggleArchive(row: PreparedProblem) {
  try {
    await client.put(`/prepared-problems/${row.id}`, {
      folder: row.folder || '',
      difficulty: row.difficulty || '',
      source: row.source || '',
      notes: row.notes || '',
      tags: tagList(row.problem?.tags),
      archived: !row.archived
    })
    ElMessage.success(row.archived ? '已恢复' : '已归档')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function publish() {
  if (!publishing.value || publishClassIDs.value.length === 0) {
    ElMessage.error('请选择班级')
    return
  }
  saving.value = true
  try {
    await client.post(`/prepared-problems/${publishing.value.id}/publish`, {
      class_ids: publishClassIDs.value
    })
    ElMessage.success('已发布到班级题库')
    publishVisible.value = false
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

function metadataPayload() {
  return {
    folder: metaForm.folder,
    difficulty: metaForm.difficulty,
    source: metaForm.source,
    notes: metaForm.notes,
    tags: parseTags(metaForm.tagsText)
  }
}

function addCase() {
  const next = problemForm.cases.length + 1
  problemForm.cases.push({ name: `case-${String(next).padStart(2, '0')}`, input: '', output: '', weight: 1 })
}

function removeCase(index: number) {
  problemForm.cases.splice(index, 1)
}

function resetMeta() {
  metaForm.folder = ''
  metaForm.difficulty = ''
  metaForm.source = ''
  metaForm.tagsText = ''
  metaForm.notes = ''
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

function parseTags(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function tagList(tags: any) {
  if (!tags) return []
  if (Array.isArray(tags)) return tags.map(String)
  if (Array.isArray(tags.labels)) return tags.labels.map(String)
  if (Array.isArray(tags.items)) return tags.items.map(String)
  return []
}

onMounted(async () => {
  await classroom.load()
  await load()
})
</script>

<style scoped>
.filters {
  display: grid;
  grid-template-columns: minmax(220px, 2fr) 160px 150px 130px 120px auto;
  gap: 10px;
  margin-bottom: 16px;
  align-items: center;
}

.prepared-layout {
  display: grid;
  grid-template-columns: minmax(420px, 0.95fr) minmax(420px, 1.05fr);
  gap: 16px;
  align-items: start;
}

.prepared-list-panel,
.detail {
  min-width: 0;
}

.problem-title {
  font-weight: 700;
}

.tag {
  margin: 0 4px 4px 0;
}

.detail {
  min-height: 420px;
}

.detail-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.detail h3 {
  margin: 0 0 4px;
}

.meta-grid {
  display: grid;
  grid-template-columns: 72px 1fr;
  gap: 8px 12px;
  margin-top: 12px;
}

.meta-grid span {
  color: #6b7280;
}

.statement {
  white-space: pre-wrap;
  line-height: 1.7;
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

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.notes {
  margin-top: 12px;
  padding: 10px;
  background: rgba(15, 23, 42, 0.04);
  border-radius: 8px;
  white-space: pre-wrap;
}

.zip-upload {
  width: 100%;
}

.upload-text {
  font-weight: 700;
  margin-bottom: 6px;
}

.problem-form {
  max-height: 54vh;
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

@media (max-width: 1100px) {
  .prepared-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .filters {
    grid-template-columns: 1fr;
  }
}
</style>
