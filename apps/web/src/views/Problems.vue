<template>
  <section class="page">
    <div class="page-header">
      <h2>题库</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openProblemDialog">上传题目包</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <el-row :gutter="16">
      <el-col :span="10">
        <div class="panel">
          <el-table :data="problems" highlight-current-row @current-change="selectProblem">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="slug" label="Slug" width="140" />
            <el-table-column prop="title" label="题目" />
            <el-table-column v-if="auth.role === 'student'" label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="progressTag(row.progress_status)" effect="light">
                  {{ progressLabel(row.progress_status) }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="14">
        <div class="panel" v-if="selected">
          <h3>{{ selected.title }}</h3>
          <p class="muted">
            {{ selected.time_limit_ms }} ms / {{ selected.memory_limit_mb }} MB /
            {{ selected.output_limit_kb }} KB
          </p>
          <p>{{ selected.statement }}</p>
          <el-divider />
          <div class="toolbar">
            <el-select v-model="language" style="width: 130px">
              <el-option label="C++17" value="cpp" />
              <el-option label="C" value="c" />
              <el-option label="Python" value="python" />
              <el-option label="Java" value="java" />
            </el-select>
            <el-button type="primary" :loading="submitting" @click="submit">提交</el-button>
          </div>
          <CodeEditor v-model="source" :language="language" />
          <div v-if="live" class="live">
            <StatusBadge :status="live.status" /> 分数 {{ live.score }}，{{ live.message }}
          </div>
        </div>
      </el-col>
    </el-row>

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
            <div class="muted">ZIP 根目录需要包含 problem.yaml 和测试数据文件</div>
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
                :rows="4"
                placeholder="描述输入、输出和样例要求"
              />
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
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { client, sseUrl, type Problem } from '../api/client'
import CodeEditor from '../components/CodeEditor.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

const auth = useAuthStore()
const classroom = useClassroomStore()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const problems = ref<Problem[]>([])
const selected = ref<Problem | null>(null)
const language = ref('cpp')
const submitting = ref(false)
const live = ref<any>(null)
const problemDialogVisible = ref(false)
const problemDialogTab = ref('zip')
const savingProblem = ref(false)
const selectedClassIDs = ref<number[]>([])
const problemForm = reactive({
  slug: '',
  title: '',
  statement: '',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  cases: [{ name: 'case-01', input: '1 2\n', output: '3\n', weight: 100 }]
})
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

function selectProblem(row: Problem) {
  selected.value = row
  live.value = null
}

function openProblemDialog() {
  problemDialogVisible.value = true
  problemDialogTab.value = 'zip'
  selectedClassIDs.value = classroom.activeClassId ? [classroom.activeClassId] : []
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
  problemForm.slug = ''
  problemForm.title = ''
  problemForm.statement = ''
  problemForm.time_limit_ms = 1000
  problemForm.memory_limit_mb = 256
  problemForm.output_limit_kb = 1024
  problemForm.cases.splice(0, problemForm.cases.length, {
    name: 'case-01',
    input: '1 2\n',
    output: '3\n',
    weight: 100
  })
}

function progressLabel(status?: string) {
  if (status === 'accepted') return '通过'
  if (status === 'attempted') return '未通过'
  return '未尝试'
}

function progressTag(status?: string): 'success' | 'warning' | 'info' {
  if (status === 'accepted') return 'success'
  if (status === 'attempted') return 'warning'
  return 'info'
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

onMounted(async () => {
  await classroom.load()
  await load()
})
</script>

<style scoped>
.live {
  margin-top: 12px;
  display: flex;
  gap: 10px;
  align-items: center;
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
</style>
