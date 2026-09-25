<template>
  <section class="page problem-set-page">
    <div v-if="detail" class="problem-set-container">
      <header class="set-heading">
        <el-button text class="back-to-team" @click="router.push(`/teams/${detail.team.slug}/problem-sets`)">← 返回团队</el-button>
        <div v-if="canOrganize || detail.can_edit || detail.can_delete" class="heading-actions">
          <el-button v-if="detail.can_edit" @click="openEditSet">编辑题单信息</el-button>
          <el-button v-if="canOrganize" @click="addVisible = true">添加题目</el-button>
          <el-button v-if="detail.can_delete" type="danger" plain :loading="deleting" @click="deleteSet">删除题单</el-button>
        </div>
      </header>

      <div class="set-identity"><img :src="detail.problem_set.icon_url || detail.team.icon_url || '/logo1.png'" alt="题单图标" /><div><h1>{{ detail.problem_set.title }}</h1><p>{{ detail.problem_set.description || detail.team.name }}</p></div></div>
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
          <ProblemStatementView
            class="set-problem-view"
            :problem="selectedLink.problem"
            :problem-number="problemSetLabel(selectedIndex)"
            :status-text="statusText(selectedLink.submission_status)"
            :status-type="statusType(selectedLink.submission_status)"
            :show-tags="false"
          >
            <template #statement-header>
              <header class="statement-heading">
                <h2>{{ selectedLink.problem.title }}</h2>
                <el-button @click="ticketVisible = true">题目反馈</el-button>
              </header>
            </template>
            <template #sidebar-header>
              <nav class="problem-navigation" aria-label="题目导航">
                <h3>题目导航</h3>
                <div class="problem-navigation-links">
                  <router-link
                    v-for="(link, index) in links"
                    :key="link.problem_id"
                    :to="{ path: route.path, query: { problem: String(link.problem_id) }, hash: '#problems' }"
                    :class="{ active: index === selectedIndex }"
                    :aria-current="index === selectedIndex ? 'page' : undefined"
                    :title="`${problemSetLabel(index)} · ${link.problem.title} · ${statusText(link.submission_status)}`"
                  >{{ problemSetLabel(index) }}</router-link>
                </div>
              </nav>
            </template>
            <template #sidebar-footer>
              <el-button type="primary" class="submit-current" @click="openSubmit(selectedLink)">提交代码</el-button>
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
            <el-table-column label="操作" width="100"><template #default="{ row }"><el-button v-if="row.user_id === auth.user?.id || detail?.can_edit" text type="primary" @click="openSubmission(row)">查看详情</el-button></template></el-table-column>
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

    <ProblemSelectionDialog v-if="detail" v-model="addVisible" :team-i-d="teamID" :set-i-d="problemSetID" :existing-i-ds="links.map(link => link.problem_id)" @added="load" />

    <el-dialog v-model="editVisible" title="编辑题单信息" width="min(520px, calc(100vw - 24px))">
      <el-form label-position="top">
        <el-form-item label="题单图标"><TeamIconUpload v-model="editForm.icon_url" :fallback="detail?.team.icon_url" label="题单图标" show-help /></el-form-item>
        <el-form-item label="题单标题"><el-input v-model="editForm.title" maxlength="200" /></el-form-item>
        <el-form-item label="题单说明"><el-input v-model="editForm.description" type="textarea" :rows="5" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="editing" @click="saveSet">保存修改</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="submitVisible" :title="`提交 ${selectedLink ? problemSetLabel(selectedIndex) : ''} ${selectedLink?.problem?.title || ''}`" width="min(900px, calc(100vw - 24px))" destroy-on-close>
      <SubmissionComposer
        ref="composerRef"
        v-model:language="language"
        v-model:source="source"
        :status="liveStatus?.status"
        :submitting="submitting"
        :draft-context="draftContext"
        @submit="submitSolution"
      />
      <template #footer>
        <el-button @click="submitVisible = false">关闭</el-button>
      </template>
    </el-dialog>
    <SubmissionDetailDialog v-model="submissionVisible" :detail="submissionDetail" />
    <TicketDialog v-if="selectedLink" v-model="ticketVisible" :context="{ kind: 'problem', id: selectedLink.problem_id }" />
  </section>
</template>

<script setup lang="ts">
import { submissionLanguage } from '../../features/submissions/presentation'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { generatedApi, getLatestSubmissions, openEventStream, type AuthenticatedEventSource, type Submission } from '../../api/client'
import TeamIconUpload from '../../components/TeamIconUpload.vue'
import ProblemSelectionDialog from '../../components/ProblemSelectionDialog.vue'
import ListPagination from '../../components/ListPagination.vue'
import TicketDialog from '../../components/tickets/TicketDialog.vue'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'
import ProblemStatementView from '../../components/ProblemStatementView.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import SubmissionComposer from '../../components/SubmissionComposer.vue'
import SubmissionDetailDialog from '../../components/SubmissionDetailDialog.vue'
import { useSubmissionDetail } from '../../features/submissions/detail'
import { formatDateTime } from '../../features/time'
import { loadSubmissionDraft } from '../../features/submissions/drafts'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const { visible: submissionVisible, detail: submissionDetail, open: openSubmission } = useSubmissionDetail()
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
const ticketVisible = ref(false)
const editVisible = ref(false)
const editing = ref(false)
const deleting = ref(false)
const editForm = reactive({ title: '', description: '', icon_url: '' })
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
const canOrganize = computed(() => {
  const team = detail.value?.team
  if (!team) return false
  if (team.my_role === 'owner') return true
  if (team.contest_permission === 'all') return Boolean(team.my_role)
  return team.contest_permission === 'admin' && team.my_role === 'admin'
})

async function load() {
  try {
    const { data } = await generatedApi.getTeamProblemSetViaProblemSetsSetId<any>({ path: { "set_id": problemSetID.value } })
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
    submissions.value = (await generatedApi.listTeamProblemSetSubmissionsViaProblemSetsSetIdSubmissions<any>({ path: { "set_id": problemSetID.value } })).data || []
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

function openEditSet() {
  const set = detail.value?.problem_set
  if (!set) return
  Object.assign(editForm, { title: set.title, description: set.description || '', icon_url: set.icon_url || '' })
  editVisible.value = true
}

async function saveSet() {
  if (!editForm.title.trim()) {
    ElMessage.warning('请输入题单标题')
    return
  }
  editing.value = true
  try {
    await generatedApi.updateTeamProblemSetViaProblemSetsSetId<any>({ path: { "set_id": problemSetID.value }, body: editForm })
    editVisible.value = false
    ElMessage.success('题单信息已更新')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    editing.value = false
  }
}

async function deleteSet() {
  const set = detail.value?.problem_set
  if (!set) return
  try {
    await ElMessageBox.confirm(
      `确认删除题单“${set.title}”？题单内的题目本身不会被删除，此操作不可撤销。`,
      '删除团队题单',
      { type: 'error', confirmButtonText: '确认删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' },
    )
    deleting.value = true
    await generatedApi.deleteTeamProblemSetViaProblemSetsSetId<any>({ path: { "set_id": problemSetID.value } })
    ElMessage.success('团队题单已删除')
    await router.replace(detail.value?.team?.slug ? `/teams/${detail.value.team.slug}/problem-sets` : '/teams')
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    deleting.value = false
  }
}

async function removeProblem(link: any) {
  try {
    await ElMessageBox.confirm(`确认将 ${link.problem.title} 移出当前题单？题目本身不会被删除。`, '移出题单', { type: 'warning' })
    await generatedApi.removeTeamProblemSetProblemViaProblemSetsSetIdProblemsProblemId<any>({ path: { "set_id": problemSetID.value, "problem_id": link.problem_id } })
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
    const { data } = await generatedApi.createTeamProblemSetSubmissionViaProblemSetsSetIdSubmissions<any>({ path: { "set_id": problemSetID.value }, body: {
      problem_id: selectedLink.value.problem.id,
      language: submissionLanguage(language.value),
      source_code: source.value
    } })
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
    discussions.value = (await generatedApi.listTeamDiscussionsViaProblemSetsSetIdDiscussions<any>({ path: { "set_id": problemSetID.value } })).data || []
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
    await generatedApi.createTeamDiscussionViaProblemSetsSetIdDiscussions<any>({ path: { "set_id": problemSetID.value }, body: {
      problem_id: discussionProblemID.value || null,
      content: discussionContent.value
    } })
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
.set-identity { display: flex; align-items: center; gap: 16px; margin: 18px 0; min-width: 0; }
.set-identity img { width: 60px; height: 60px; flex: 0 0 auto; object-fit: contain; object-position: center; background: var(--color-surface-solid); border: 1px solid var(--border); border-radius: 12px; }
.set-identity h1 { margin: 0 0 6px; font-size: 24px; overflow-wrap: anywhere; }
.set-identity p { margin: 0; color: var(--muted); overflow-wrap: anywhere; }

.statement-heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 20px; }
.statement-heading h2 { min-width: 0; margin: 0; font-size: 22px; overflow-wrap: anywhere; }
.statement-heading .el-button { flex-shrink: 0; }
.problem-set-page { padding: 12px 20px 40px; }
.problem-set-container { width: min(1200px, 100%); margin: 0 auto; }
.set-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px 12px; padding: 0 0 6px; }
.back-to-team { padding-inline: 0; }
.heading-actions { display: flex; justify-content: flex-end; flex-wrap: wrap; gap: 8px; margin-left: auto; }
.heading-actions :deep(.el-button) { margin: 0; }
.set-tabs { display: flex; gap: 5px; margin-bottom: 10px; border-bottom: 1px solid var(--border); }
.set-tabs button { padding: 10px 20px; color: var(--muted); border: 0; border-bottom: 3px solid transparent; background: transparent; cursor: pointer; }
.set-tabs button.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.problem-list-panel { overflow: hidden; }
:deep(.problem-row) { cursor: pointer; }
:deep(.problem-row:hover > td.el-table__cell) { background: color-mix(in srgb, var(--accent) 7%, var(--surface-strong)); }
.submission-stat, .problem-title-button, .problem-code-button, .user-name { color: var(--color-info); }
.problem-title-button, .problem-code-button { padding: 0; border: 0; background: transparent; font: inherit; cursor: pointer; }
.problem-title-button { font-weight: 700; }
.set-problem-view { grid-template-columns: minmax(0, 1fr) 300px; gap: 16px; }
.problem-navigation { margin-bottom: 18px; padding-bottom: 18px; border-bottom: 1px solid var(--border); }
.problem-navigation h3 { margin: 0; font-size: 16px; }
.problem-navigation-links { display: flex; flex-wrap: wrap; gap: 7px; margin-top: 14px; }
.problem-navigation-links a { min-width: 38px; padding: 8px; text-align: center; border: 1px solid var(--border); border-radius: 6px; }
.problem-navigation-links a:hover, .problem-navigation-links a:focus-visible { border-color: var(--accent); }
.problem-navigation-links a.active { color: var(--color-on-emphasis); border-color: var(--accent); background: var(--accent); }
.submit-current { width: 100%; margin-top: 20px; }
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
@media (max-width: 980px) { .set-problem-view { grid-template-columns: minmax(0, 1fr); } }
@media (max-width: 760px) { .problem-set-page { padding: 10px 13px 32px; } .heading-actions { gap: 6px; } .heading-actions :deep(.el-button) { padding-inline: 10px; } .set-tabs button { padding: 10px 16px; } .discussion-layout { grid-template-columns: 1fr; } .submission-toolbar { align-items: stretch; flex-wrap: wrap; } }
</style>
