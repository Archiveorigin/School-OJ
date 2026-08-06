<template>
  <section class="page problem-set-page">
    <div v-if="detail" class="problem-set-container">
      <el-page-header title="返回团队" @back="router.push(`/teams/${detail.team.slug}/problem-sets`)">
        <template #content><strong>{{ detail.problem_set.title }}</strong></template>
      </el-page-header>

      <header class="set-heading">
        <div>
          <span>TEAM PROBLEM SET</span>
          <h1>{{ detail.problem_set.title }}</h1>
          <p>{{ detail.problem_set.description || '团队私有训练题单' }}</p>
        </div>
        <div v-if="canOrganize" class="heading-actions">
          <el-button @click="addVisible = true">添加已有题目</el-button>
          <el-button v-if="canAuthor" type="primary" @click="createPrivateProblem">新建私有题目</el-button>
        </div>
      </header>

      <nav class="set-tabs">
        <button type="button" :class="{ active: activeTab === 'problems' }" @click="openProblemList">题目列表</button>
        <button type="button" :class="{ active: activeTab === 'submissions' }" @click="openSubmissions">提交状态</button>
        <button type="button" :class="{ active: activeTab === 'discussions' }" @click="openDiscussions">讨论</button>
      </nav>

      <template v-if="links.length">
        <section v-if="activeTab === 'problems' && problemView === 'list'" class="panel problem-list-panel">
          <el-table :data="links" row-class-name="problem-row" @row-click="showProblem">
            <el-table-column label="状态" width="130" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.submission_status === 'accepted'" type="success" effect="plain">已解决</el-tag>
                <el-tag v-else-if="row.submission_status" type="warning" effect="plain">已尝试</el-tag>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
            <el-table-column label="提交统计" width="150" align="center">
              <template #default="{ row }"><span class="submission-stat">{{ row.accepted_count || 0 }} / {{ row.submission_count || 0 }}</span></template>
            </el-table-column>
            <el-table-column label="#" width="76" align="center">
              <template #default="{ $index }"><strong>{{ problemSetLabel($index) }}</strong></template>
            </el-table-column>
            <el-table-column label="标题" min-width="260">
              <template #default="{ row }"><button type="button" class="problem-title-button" @click.stop="showProblem(row)">{{ row.problem.title }}</button></template>
            </el-table-column>
            <el-table-column v-if="canOrganize" label="操作" width="110" align="right">
              <template #default="{ row }"><el-button type="danger" text size="small" @click.stop="removeProblem(row)">移出题单</el-button></template>
            </el-table-column>
          </el-table>
        </section>

        <template v-else-if="activeTab === 'problems' && selectedLink">
          <div class="problem-detail-toolbar">
            <el-button text @click="openProblemList">← 返回题目列表</el-button>
            <strong>{{ problemSetLabel(selectedIndex) }} · {{ selectedLink.problem.title }}</strong>
          </div>
          <ProblemStatementView
            :problem="selectedLink.problem"
            :problem-number="problemSetLabel(selectedIndex)"
            :status-text="statusText(selectedLink.submission_status)"
            :status-type="statusType(selectedLink.submission_status)"
            :show-tags="false"
          >
            <template #sidebar-footer>
              <div class="problem-number-selector">
                <span class="muted">题目选择</span>
                <ProblemSwitcher v-model="selectedProblemID" :items="switcherItems" />
                <el-button type="primary" class="submit-current" @click="openSubmit(selectedLink)">提交代码</el-button>
              </div>
            </template>
          </ProblemStatementView>
        </template>

        <section v-else-if="activeTab === 'submissions'" class="panel status-panel">
          <div class="submission-toolbar">
            <strong>提交状态</strong>
            <span class="muted">共 {{ filteredSubmissions.length }} 条记录</span>
            <el-button @click="resetSubmissionFilters">重置</el-button>
            <el-button :loading="loadingSubmissions" @click="loadSubmissions">刷新</el-button>
          </div>
          <el-table :data="pagedSubmissions" v-loading="loadingSubmissions" size="small">
            <el-table-column prop="id" label="ID" width="104" />
            <el-table-column min-width="170" align="center">
              <template #header>
                <div class="filter-heading"><span>用户名</span><el-input v-model="submissionFilters.username" clearable size="small" placeholder="查询用户名" /></div>
              </template>
              <template #default="{ row }"><span class="user-name">{{ row.user_name || `用户 ${row.user_id}` }}</span></template>
            </el-table-column>
            <el-table-column width="128" align="center">
              <template #header>
                <div class="filter-heading"><span>题号</span><el-select v-model="submissionFilters.problemID" clearable size="small" placeholder="全部"><el-option v-for="(link, index) in links" :key="link.problem_id" :label="problemSetLabel(index)" :value="link.problem_id" /></el-select></div>
              </template>
              <template #default="{ row }"><button type="button" class="problem-code-button" @click="openSubmissionProblem(row.problem_id)">{{ problemLabel(row.problem_id) }}</button></template>
            </el-table-column>
            <el-table-column min-width="178" align="center">
              <template #header>
                <div class="filter-heading"><span>评测结果</span><el-select v-model="submissionFilters.status" clearable size="small" placeholder="全部"><el-option v-for="status in submissionStatusOptions" :key="status" :label="submissionStatusLabel(status)" :value="status" /></el-select></div>
              </template>
              <template #default="{ row }"><StatusBadge :status="row.status" /></template>
            </el-table-column>
            <el-table-column label="耗时 (ms)" width="112" align="center"><template #default="{ row }">{{ row.time_ms || '-' }}</template></el-table-column>
            <el-table-column label="内存 (MB)" width="112" align="center"><template #default="{ row }">{{ formatMemory(row.memory_kb) }}</template></el-table-column>
            <el-table-column prop="code_length" label="代码长度" width="112" align="center" />
            <el-table-column width="142" align="center">
              <template #header>
                <div class="filter-heading"><span>语言</span><el-select v-model="submissionFilters.language" clearable size="small" placeholder="全部"><el-option v-for="item in submissionLanguageOptions" :key="item" :label="languageLabel(item)" :value="item" /></el-select></div>
              </template>
              <template #default="{ row }">{{ languageLabel(row.language) }}</template>
            </el-table-column>
            <el-table-column label="提交时间" min-width="176"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column>
          </el-table>
          <ListPagination v-model:page="submissionPage" v-model:page-size="submissionPageSize" :total="filteredSubmissions.length" />
        </section>

        <section v-else-if="activeTab === 'discussions'" class="discussion-layout">
          <div class="panel discussion-editor">
            <h3>参与讨论</h3>
            <el-select v-model="discussionProblemID" clearable placeholder="整个题单">
                <el-option v-for="(link, index) in links" :key="link.id" :label="`${problemSetLabel(index)} · ${link.problem.title}`" :value="link.problem_id" />
            </el-select>
            <el-input v-model="discussionContent" type="textarea" :rows="5" maxlength="5000" show-word-limit placeholder="分享思路、提出疑问或补充题解" />
            <el-button type="primary" :loading="posting" @click="postDiscussion">发布讨论</el-button>
          </div>
          <div class="discussion-list">
            <article v-for="item in discussions" :key="item.id" class="panel discussion-item">
              <div class="discussion-author">
                <el-avatar :size="34" :src="item.author_avatar">{{ item.author_name.slice(0, 1) }}</el-avatar>
                <div><strong>{{ item.author_name }}</strong><span>{{ formatDateTime(item.created_at) }}</span></div>
                <el-tag v-if="item.problem_id" size="small">{{ problemLabel(item.problem_id) }}</el-tag>
              </div>
              <MarkdownRenderer :source="item.content" />
            </article>
            <el-empty v-if="!discussions.length" description="还没有讨论，来发布第一条吧" />
          </div>
        </section>
      </template>
      <el-empty v-else description="题单中还没有题目">
        <el-button v-if="canOrganize" type="primary" @click="addVisible = true">添加第一道题</el-button>
      </el-empty>
    </div>
    <el-skeleton v-else :rows="10" animated class="problem-set-container" />

    <el-dialog v-model="addVisible" title="添加已有题目" width="480px">
      <p class="dialog-hint">输入公共题目或本团队私有题目的题号，例如 T001。</p>
      <el-input v-model="problemCode" placeholder="题目编号" @keyup.enter="addProblem" />
      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button type="primary" :loading="adding" @click="addProblem">添加</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="submitVisible" :title="`提交 ${selectedLink ? problemSetLabel(selectedIndex) : ''} ${selectedLink?.problem?.title || ''}`" width="min(900px, calc(100vw - 24px))" destroy-on-close>
      <SubmissionComposer
        ref="composerRef"
        v-model:language="language"
        v-model:source="source"
        :status="liveStatus?.status"
        :message="liveStatus?.message"
        :submitting="submitting"
        :draft-context="draftContext"
        scope-text="本次代码仅计入当前团队题单"
        @submit="submitSolution"
      />
      <template #footer>
        <el-button @click="submitVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, getLatestSubmissions, openEventStream, type AuthenticatedEventSource, type Submission } from '../../api/client'
import ListPagination from '../../components/ListPagination.vue'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'
import ProblemStatementView from '../../components/ProblemStatementView.vue'
import ProblemSwitcher from '../../components/ProblemSwitcher.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import SubmissionComposer from '../../components/SubmissionComposer.vue'
import { formatDateTime } from '../../features/time'
import { loadSubmissionDraft } from '../../features/submissions/drafts'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const detail = ref<any>(null)
const links = ref<any[]>([])
const selectedIndex = ref(0)
const activeTab = ref<'problems' | 'submissions' | 'discussions'>('problems')
const problemView = ref<'list' | 'detail'>('list')
const submissions = ref<Submission[]>([])
const loadingSubmissions = ref(false)
const submissionPage = ref(1)
const submissionPageSize = ref(20)
const submissionFilters = reactive({ username: '', problemID: undefined as number | undefined, status: '', language: '' })
const discussions = ref<any[]>([])
const discussionProblemID = ref<number>()
const discussionContent = ref('')
const posting = ref(false)
const addVisible = ref(false)
const adding = ref(false)
const problemCode = ref('')
const submitVisible = ref(false)
const submitting = ref(false)
const language = ref('cpp')
const source = ref('')
const liveStatus = ref<any>(null)
const composerRef = ref<{ clearDraft: () => void } | null>(null)
const liveStreams = new Set<AuthenticatedEventSource>()
const teamID = computed(() => Number(detail.value?.team?.id))
const problemSetID = computed(() => Number(route.params.setId))
const selectedLink = computed(() => links.value[selectedIndex.value])
const draftContext = computed(() => ({ userId: auth.user?.id || 0, resourceType: 'problem-set' as const, resourceId: problemSetID.value, problemId: selectedLink.value?.problem_id || 0 }))
const selectedProblemID = computed({
  get: () => selectedLink.value?.problem_id,
  set: (value: number | undefined) => {
    const index = links.value.findIndex((item) => item.problem_id === value)
    if (index >= 0) {
      selectedIndex.value = index
      void router.replace({ path: route.path, query: { ...route.query, problem: String(value) }, hash: '#problems' })
    }
  }
})
const switcherItems = computed(() => links.value.map((link, index) => ({ ...link, label: problemSetLabel(index) })))
const submissionStatusOptions = computed(() => [...new Set(submissions.value.map((item) => item.status).filter(Boolean))])
const submissionLanguageOptions = computed(() => [...new Set(submissions.value.map((item) => item.language).filter(Boolean))])
const filteredSubmissions = computed(() => submissions.value.filter((item) => {
  const username = submissionFilters.username.trim().toLowerCase()
  if (username && !String(item.user_name || '').toLowerCase().includes(username)) return false
  if (submissionFilters.problemID && item.problem_id !== submissionFilters.problemID) return false
  if (submissionFilters.status && item.status !== submissionFilters.status) return false
  if (submissionFilters.language && item.language !== submissionFilters.language) return false
  return true
}))
const pagedSubmissions = computed(() => {
  const start = (submissionPage.value - 1) * submissionPageSize.value
  return filteredSubmissions.value.slice(start, start + submissionPageSize.value)
})
const canAuthor = computed(() => auth.role === 'admin' || Boolean(auth.user?.can_author))
const canOrganize = computed(() => {
  const team = detail.value?.team
  if (!team) return false
  if (team.my_role === 'owner') return true
  if (team.contest_permission === 'all') return Boolean(team.my_role)
  return team.contest_permission === 'admin' && team.my_role === 'admin'
})

async function load() {
  try {
    const { data } = await client.get(`/problem-sets/${problemSetID.value}`)
    detail.value = data
    links.value = data.problems || []
    if (selectedIndex.value >= links.value.length) selectedIndex.value = 0
    const routeProblemIndex = links.value.findIndex((item) => item.problem_id === Number(route.query.problem))
    if (routeProblemIndex >= 0) { selectedIndex.value = routeProblemIndex; problemView.value = 'detail' }
    await loadSubmissions()
    if (activeTab.value === 'discussions') await openDiscussions()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
    await router.replace('/teams')
  }
}

function openProblemList() {
  activeTab.value = 'problems'
  problemView.value = 'list'
  void router.push({ path: route.path, hash: '#problems' })
}

function showProblem(link: any) {
  const index = links.value.findIndex((item) => item.id === link.id)
  if (index >= 0) selectedIndex.value = index
  activeTab.value = 'problems'
  problemView.value = 'detail'
  void router.push({ path: route.path, query: { problem: String(link.problem_id) }, hash: '#problems' })
}

async function openSubmissions() {
  activeTab.value = 'submissions'
  await router.push({ path: route.path, hash: '#submissions' })
  await loadSubmissions()
}

async function loadSubmissions() {
  loadingSubmissions.value = true
  try {
    submissions.value = (await client.get(`/problem-sets/${problemSetID.value}/submissions`)).data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loadingSubmissions.value = false
  }
}

function resetSubmissionFilters() {
  Object.assign(submissionFilters, { username: '', problemID: undefined, status: '', language: '' })
  submissionPage.value = 1
}

function createPrivateProblem() {
  router.push({ path: '/problems/create', query: { teamId: teamID.value, problemSetId: problemSetID.value } })
}

async function addProblem() {
  if (!problemCode.value.trim()) {
    ElMessage.warning('请输入题目编号')
    return
  }
  adding.value = true
  try {
    await client.post(`/problem-sets/${problemSetID.value}/problems`, { problem_code: problemCode.value.trim() })
    problemCode.value = ''
    addVisible.value = false
    ElMessage.success('题目已加入题单')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    adding.value = false
  }
}

async function removeProblem(link: any) {
  try {
    await ElMessageBox.confirm(`确认将 ${link.problem.title} 移出当前题单？题目本身不会被删除。`, '移出题单', { type: 'warning' })
    await client.delete(`/problem-sets/${problemSetID.value}/problems/${link.problem_id}`)
    ElMessage.success('题目已移出题单')
    await load()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function openSubmit(link: any) {
  selectedIndex.value = links.value.findIndex((item) => item.id === link.id)
  const latest = (await getLatestSubmissions({ problem_set_id: problemSetID.value }, link.problem_id))[0]
  language.value = latest?.language || 'cpp'
  source.value = loadSubmissionDraft(draftContext.value, language.value) ?? latest?.source_code ?? ''
  liveStatus.value = latest || null
  submitVisible.value = true
}

async function submitSolution() {
  if (!selectedLink.value || !source.value.trim()) {
    ElMessage.warning('请输入代码')
    return
  }
  submitting.value = true
  try {
    const { data } = await client.post(`/problem-sets/${problemSetID.value}/submissions`, {
      problem_id: selectedLink.value.problem.id,
      language: language.value,
      source_code: source.value
    })
    composerRef.value?.clearDraft()
    liveStatus.value = data
    const stream = openEventStream(`/submissions/${data.id}/events`)
    liveStreams.add(stream)
    stream.addEventListener('status', async (event) => {
      liveStatus.value = JSON.parse((event as MessageEvent).data)
      if (!['queued', 'running'].includes(liveStatus.value.status)) {
        stream.close()
        liveStreams.delete(stream)
        await load()
        if (activeTab.value === 'submissions') await loadSubmissions()
      }
    })
    ElMessage.success('代码已提交评测')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function openSubmissionProblem(problemID: number) {
  const link = links.value.find((item) => item.problem_id === problemID)
  if (link) showProblem(link)
}

async function openDiscussions() {
  activeTab.value = 'discussions'
  await router.push({ path: route.path, hash: '#discussions' })
  try {
    discussions.value = (await client.get(`/problem-sets/${problemSetID.value}/discussions`)).data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function postDiscussion() {
  if (!discussionContent.value.trim()) {
    ElMessage.warning('请输入讨论内容')
    return
  }
  posting.value = true
  try {
    await client.post(`/problem-sets/${problemSetID.value}/discussions`, {
      problem_id: discussionProblemID.value || null,
      content: discussionContent.value
    })
    discussionContent.value = ''
    ElMessage.success('讨论已发布')
    await openDiscussions()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    posting.value = false
  }
}

function statusText(status?: string) {
  if (!status) return '未提交'
  if (status === 'accepted') return '已通过'
  if (status === 'queued' || status === 'running') return '评测中'
  return '未通过'
}

function statusType(status?: string): 'success' | 'warning' | 'info' | 'danger' {
  if (status === 'accepted') return 'success'
  if (status === 'queued' || status === 'running') return 'warning'
  return status ? 'danger' : 'info'
}

function problemLabel(problemID: number) {
  const index = links.value.findIndex((link) => link.problem_id === problemID)
  if (index < 0) return '题单讨论'
  return problemSetLabel(index)
}

function problemSetLabel(index: number) {
  index += 1
  let label = ''
  while (index > 0) {
    index -= 1
    label = String.fromCharCode(65 + (index % 26)) + label
    index = Math.floor(index / 26)
  }
  return label
}

function submissionStatusLabel(status: string) {
  const labels: Record<string, string> = {
    accepted: '通过', wrong_answer: '答案错误', compile_error: '编译错误', runtime_error: '运行错误',
    time_limit: '时间超限', memory_limit: '内存超限', output_limit: '输出超限', queued: '排队中', running: '评测中', system_error: '系统错误'
  }
  return labels[status] || status
}

function languageLabel(language: string) {
  return ({ cpp: 'C++17', c: 'C', python: 'Python', java: 'Java' } as Record<string, string>)[language] || language
}

function formatMemory(memoryKB: number) {
  if (!memoryKB) return '-'
  return (memoryKB / 1024).toFixed(memoryKB >= 10240 ? 0 : 1)
}

watch(() => route.params.setId, load, { immediate: true })
watch(() => route.hash, (hash) => {
  const tab = hash.replace(/^#/, '')
  if (tab === 'submissions' || tab === 'discussions' || tab === 'problems') activeTab.value = tab
}, { immediate: true })
watch(() => route.query.problem, (value) => {
  const index = links.value.findIndex((item) => item.problem_id === Number(value))
  if (index >= 0) { selectedIndex.value = index; problemView.value = 'detail' }
})
watch(
  () => [submissionFilters.username, submissionFilters.problemID, submissionFilters.status, submissionFilters.language, submissionPageSize.value],
  () => { submissionPage.value = 1 }
)
onBeforeUnmount(() => { for (const stream of liveStreams) stream.close() })
</script>

<style scoped>
.problem-set-page { padding: 24px 20px 58px; }
.problem-set-container { width: min(1200px, 100%); margin: 0 auto; }
.set-heading { display: flex; align-items: end; justify-content: space-between; gap: 22px; padding: 32px 0 24px; }
.set-heading > div > span { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .15em; }
.set-heading h1 { margin: 8px 0 5px; font-size: 31px; }
.set-heading p { margin: 0; color: var(--muted); }
.heading-actions { display: flex; gap: 9px; }
.set-tabs { display: flex; gap: 5px; margin-bottom: 16px; border-bottom: 1px solid var(--border); }
.set-tabs button { padding: 14px 24px; color: var(--muted); border: 0; border-bottom: 3px solid transparent; background: transparent; cursor: pointer; }
.set-tabs button.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.problem-list-panel { overflow: hidden; }
:deep(.problem-row) { cursor: pointer; }
:deep(.problem-row:hover > td.el-table__cell) { background: color-mix(in srgb, var(--accent) 7%, var(--surface-strong)); }
.submission-stat, .problem-title-button, .problem-code-button, .user-name { color: #1f9fc2; }
.problem-title-button, .problem-code-button { padding: 0; border: 0; background: transparent; font: inherit; cursor: pointer; }
.problem-title-button { font-weight: 700; }
.problem-detail-toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.problem-number-selector { display: grid; gap: 14px; margin-top: 20px; padding-top: 18px; border-top: 1px solid var(--border); }
.submit-current { width: 100%; }
.status-panel { min-width: 0; padding: 16px; overflow: hidden; }
.submission-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 14px; }
.submission-toolbar strong { margin-right: auto; }
.filter-heading { display: grid; gap: 6px; }
.filter-heading > span { color: var(--muted); font-size: 12px; font-weight: 700; }
.filter-heading :deep(.el-select) { width: 100%; }
.discussion-layout { display: grid; grid-template-columns: minmax(280px, .65fr) minmax(0, 1.35fr); gap: 14px; align-items: start; }
.discussion-editor { display: grid; gap: 12px; }
.discussion-editor h3 { margin: 0; }
.discussion-list { display: grid; gap: 11px; }
.discussion-author { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.discussion-author > div { display: grid; gap: 2px; margin-right: auto; }
.discussion-author span { color: var(--muted); font-size: 12px; }
.dialog-hint { color: var(--muted); }
@media (max-width: 760px) { .problem-set-page { padding: 16px 13px 42px; } .set-heading { align-items: stretch; flex-direction: column; } .heading-actions { flex-direction: column; } .discussion-layout { grid-template-columns: 1fr; } .submission-toolbar { align-items: stretch; flex-wrap: wrap; } }
</style>
