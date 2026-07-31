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
        <button type="button" :class="{ active: activeTab === 'problems' }" @click="activeTab = 'problems'">题目</button>
        <button type="button" :class="{ active: activeTab === 'submissions' }" @click="activeTab = 'submissions'">提交状态</button>
        <button type="button" :class="{ active: activeTab === 'discussions' }" @click="openDiscussions">讨论</button>
      </nav>

      <template v-if="links.length">
        <section class="problem-switcher panel">
          <div class="number-row">
            <button
              v-for="(link, index) in links"
              :key="link.id"
              type="button"
              :class="{ active: selectedIndex === index, accepted: link.submission_status === 'accepted' }"
              @click="selectedIndex = index"
            >
              {{ link.label || String.fromCharCode(65 + index) }}
            </button>
          </div>
          <div v-if="selectedLink" class="problem-quick-actions">
            <strong>{{ selectedLink.problem.display_code }} · {{ selectedLink.problem.title }}</strong>
            <div>
              <span>提交</span>
              <el-button type="primary" size="small" @click="openProblem(selectedLink)">提交本题</el-button>
              <span>提交状态</span>
              <StatusBadge v-if="selectedLink.submission_status" :status="selectedLink.submission_status" />
              <el-tag v-else type="info" effect="plain">未提交</el-tag>
              <el-button v-if="canOrganize" type="danger" text size="small" @click="removeProblem(selectedLink)">移出题单</el-button>
            </div>
          </div>
        </section>

        <ProblemStatementView
          v-if="activeTab === 'problems' && selectedLink"
          :problem="selectedLink.problem"
          :problem-number="selectedLink.label || String.fromCharCode(65 + selectedIndex)"
          :status-text="statusText(selectedLink.submission_status)"
          :status-type="statusType(selectedLink.submission_status)"
        />

        <section v-else-if="activeTab === 'submissions'" class="panel status-panel">
          <el-table :data="links">
            <el-table-column label="题号" width="90">
              <template #default="{ row, $index }">{{ row.label || String.fromCharCode(65 + $index) }}</template>
            </el-table-column>
            <el-table-column label="题目" min-width="230">
              <template #default="{ row }"><a href="#" @click.prevent="selectAndOpen(row)">{{ row.problem.display_code }} · {{ row.problem.title }}</a></template>
            </el-table-column>
            <el-table-column label="状态" width="130">
              <template #default="{ row }"><StatusBadge v-if="row.submission_status" :status="row.submission_status" /><el-tag v-else type="info">未提交</el-tag></template>
            </el-table-column>
            <el-table-column label="最近提交" width="190">
              <template #default="{ row }">{{ row.submitted_at ? formatDateTime(row.submitted_at) : '-' }}</template>
            </el-table-column>
          </el-table>
        </section>

        <section v-else-if="activeTab === 'discussions'" class="discussion-layout">
          <div class="panel discussion-editor">
            <h3>参与讨论</h3>
            <el-select v-model="discussionProblemID" clearable placeholder="整个题单">
              <el-option v-for="(link, index) in links" :key="link.id" :label="`${link.label || String.fromCharCode(65 + index)} · ${link.problem.title}`" :value="link.problem_id" />
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
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../../api/client'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'
import ProblemStatementView from '../../components/ProblemStatementView.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { formatDateTime } from '../../features/time'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const detail = ref<any>(null)
const links = ref<any[]>([])
const selectedIndex = ref(0)
const activeTab = ref<'problems' | 'submissions' | 'discussions'>('problems')
const discussions = ref<any[]>([])
const discussionProblemID = ref<number>()
const discussionContent = ref('')
const posting = ref(false)
const addVisible = ref(false)
const adding = ref(false)
const problemCode = ref('')
const teamID = computed(() => Number(route.params.teamId))
const problemSetID = computed(() => Number(route.params.setId))
const selectedLink = computed(() => links.value[selectedIndex.value])
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
    const { data } = await client.get(`/teams/${teamID.value}/problem-sets/${problemSetID.value}`)
    detail.value = data
    links.value = data.problems || []
    if (selectedIndex.value >= links.value.length) selectedIndex.value = 0
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
    await router.replace('/teams')
  }
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
    await client.post(`/teams/${teamID.value}/problem-sets/${problemSetID.value}/problems`, { problem_code: problemCode.value.trim() })
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
    await client.delete(`/teams/${teamID.value}/problem-sets/${problemSetID.value}/problems/${link.problem_id}`)
    ElMessage.success('题目已移出题单')
    await load()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  }
}

function openProblem(link: any) {
  router.push(`/problems/${encodeURIComponent(link.problem.display_code || String(link.problem.id))}`)
}

function selectAndOpen(link: any) {
  selectedIndex.value = links.value.findIndex((item) => item.id === link.id)
  activeTab.value = 'problems'
}

async function openDiscussions() {
  activeTab.value = 'discussions'
  try {
    discussions.value = (await client.get(`/teams/${teamID.value}/problem-sets/${problemSetID.value}/discussions`)).data || []
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
    await client.post(`/teams/${teamID.value}/problem-sets/${problemSetID.value}/discussions`, {
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
  return links.value[index].label || String.fromCharCode(65 + index)
}

watch(() => [route.params.teamId, route.params.setId], load, { immediate: true })
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
.problem-switcher { margin-bottom: 14px; }
.number-row { display: flex; flex-wrap: wrap; gap: 9px; }
.number-row button { width: 42px; height: 42px; color: var(--text); border: 1px solid var(--border); border-radius: 8px; background: var(--surface-strong); font-weight: 800; cursor: pointer; }
.number-row button.accepted { color: #15803d; border-color: #86efac; background: #f0fdf4; }
.number-row button.active { color: #fff; border-color: var(--accent); background: var(--accent); }
.problem-quick-actions { display: flex; align-items: center; justify-content: space-between; gap: 15px; margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--border); }
.problem-quick-actions > div { display: flex; align-items: center; flex-wrap: wrap; gap: 9px; }
.problem-quick-actions span { color: var(--muted); font-size: 12px; }
.status-panel { padding: 18px; }
.discussion-layout { display: grid; grid-template-columns: minmax(280px, .65fr) minmax(0, 1.35fr); gap: 14px; align-items: start; }
.discussion-editor { display: grid; gap: 12px; }
.discussion-editor h3 { margin: 0; }
.discussion-list { display: grid; gap: 11px; }
.discussion-author { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.discussion-author > div { display: grid; gap: 2px; margin-right: auto; }
.discussion-author span { color: var(--muted); font-size: 12px; }
.dialog-hint { color: var(--muted); }
@media (max-width: 760px) { .problem-set-page { padding: 16px 13px 42px; } .set-heading, .problem-quick-actions { align-items: stretch; flex-direction: column; } .heading-actions { flex-direction: column; } .discussion-layout { grid-template-columns: 1fr; } }
</style>
