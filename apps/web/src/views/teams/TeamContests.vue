<template>
  <section class="workspace-view">
    <div class="view-heading">
      <div><h3>团队比赛</h3><p>比赛仅对团队成员开放</p></div>
      <el-button v-if="canOrganize" type="primary" @click="createVisible = true">创建比赛</el-button>
    </div>
    <div v-loading="loading" class="panel list-panel">
      <el-table :data="contests" row-class-name="contest-row" @row-click="openContest">
        <el-table-column prop="title" label="标题" min-width="240" />
        <el-table-column label="开始时间" width="190">
          <template #default="{ row }">{{ row.starts_at ? formatTime(row.starts_at) : '待定' }}</template>
        </el-table-column>
        <el-table-column label="时长" width="130">
          <template #default="{ row }">{{ durationText(row.duration_minutes) }}</template>
        </el-table-column>
        <el-table-column label="题目" width="90">
          <template #default="{ row }">{{ row.problem_count || 0 }} 题</template>
        </el-table-column>
        <el-table-column label="计分方式" width="170">
          <template #default="{ row }">{{ row.scoring_rule === 'score' ? '总分数' : '通过数 + 罚时' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" :width="hasManagementActions ? 224 : 90" align="right" fixed="right">
          <template #default="{ row }">
            <div class="row-actions" @click.stop>
              <el-button v-if="row.can_edit" link @click.stop="editContest(row)">编辑</el-button>
              <el-button
                v-if="row.can_delete"
                type="danger"
                link
                :loading="deletingID === row.id"
                @click.stop="deleteContest(row)"
              >删除</el-button>
              <el-button type="primary" link @click.stop="openContest(row)">进入</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && !contests.length" description="暂无团队比赛" />
    </div>

    <el-dialog v-model="createVisible" title="创建团队比赛" width="min(640px, calc(100vw - 24px))">
      <el-form label-position="top">
        <el-form-item label="比赛标题"><el-input v-model="form.title" maxlength="200" /></el-form-item>
        <el-form-item label="开始时间"><el-date-picker v-model="form.starts_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" placeholder="选择开始时间" /></el-form-item>
        <el-form-item label="比赛时长">
          <el-input-number v-model="form.duration_minutes" :min="15" :step="15" />
          <span class="unit">分钟</span>
        </el-form-item>
        <el-form-item label="计分方式">
          <el-radio-group v-model="form.scoring_rule">
            <el-radio-button value="penalty">通过数 + 罚时</el-radio-button>
            <el-radio-button value="score">总分数</el-radio-button>
          </el-radio-group>
          <p class="rule-hint">罚时按首次通过分钟数并累计通过前的错误提交；总分数按各题最高得分合计排序。</p>
        </el-form-item>
        <el-form-item label="奖项比例">
          <TeamContestAwardFields
            v-model:gold-award-percent="form.gold_award_percent"
            v-model:silver-award-percent="form.silver_award_percent"
            v-model:bronze-award-percent="form.bronze_award_percent"
          />
        </el-form-item>
        <el-form-item label="说明"><el-input v-model="form.description" type="textarea" :rows="4" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createContest">创建</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Team, type TeamContest } from '../../api/client'
import TeamContestAwardFields from '../../components/TeamContestAwardFields.vue'
import { contestAwardValidationError, defaultContestAwardPercents } from '../../features/teams/contestAwards'
import { formatDateTime } from '../../features/time'

const props = defineProps<{ team: Team }>()
const router = useRouter()
const contests = ref<TeamContest[]>([])
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const deletingID = ref<number | null>(null)
const form = reactive({
  title: '',
  description: '',
  starts_at: '',
  duration_minutes: 120,
  scoring_rule: 'penalty',
  ...defaultContestAwardPercents(),
})
const canOrganize = computed(() => {
  if (props.team.my_role === 'owner') return true
  if (props.team.contest_permission === 'all') return Boolean(props.team.my_role)
  return props.team.contest_permission === 'admin' && props.team.my_role === 'admin'
})
const hasManagementActions = computed(() => contests.value.some((row) => Boolean(row.can_edit || row.can_delete)))

async function load() {
  loading.value = true
  try {
    contests.value = (await client.get(`/teams/${props.team.id}/contests`)).data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

async function createContest() {
  if (!form.title.trim()) {
    ElMessage.warning('请输入比赛标题')
    return
  }
  if (!form.starts_at) {
    ElMessage.warning('请选择比赛开始时间')
    return
  }
  const awardError = contestAwardValidationError(form)
  if (awardError) {
    ElMessage.warning(awardError)
    return
  }
  creating.value = true
  try {
    const { data } = await client.post<TeamContest>(`/teams/${props.team.id}/contests`, form)
    Object.assign(form, { title: '', description: '', starts_at: '', duration_minutes: 120, scoring_rule: 'penalty', ...defaultContestAwardPercents() })
    createVisible.value = false
    ElMessage.success('团队比赛已创建')
    await load()
    await router.push(`/contest/${data.id}#overview`)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    creating.value = false
  }
}

function editContest(row: TeamContest) {
  router.push({ path: `/contest/${row.id}`, query: { manage: 'edit' }, hash: '#overview' })
}

async function deleteContest(row: TeamContest) {
  try {
    await ElMessageBox.confirm(
      `确认删除比赛“${row.title}”？比赛题目关联与榜单将不再对团队成员显示，此操作不可撤销。`,
      '删除团队比赛',
      { type: 'error', confirmButtonText: '确认删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' },
    )
    deletingID.value = row.id
    await client.delete(`/contests/${row.id}`)
    ElMessage.success('团队比赛已删除')
    await load()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    deletingID.value = null
  }
}

function openContest(row: TeamContest) {
  if ((row.status === 'draft' || row.status === 'published') && !canOrganize.value && !row.can_edit && !row.can_delete) {
    ElMessage.warning('比赛尚未开始')
    return
  }
  router.push(`/contest/${row.id}#overview`)
}

function statusLabel(status?: TeamContest['status']) {
  if (status === 'draft') return '草稿'
  if (status === 'published') return '已发布'
  if (status === 'closed') return '已结束'
  return '进行中'
}

function statusType(status?: TeamContest['status']): 'success' | 'warning' | 'info' {
  if (status === 'draft' || status === 'published') return 'warning'
  if (status === 'closed') return 'info'
  return 'success'
}

function formatTime(value: string) {
  return formatDateTime(value)
}

function durationText(minutes: number) {
  if (minutes >= 60 && minutes % 60 === 0) return `${minutes / 60} 小时`
  return `${minutes} 分钟`
}

onMounted(load)
</script>

<style scoped>
.workspace-view { padding: 28px 34px 54px; }
.view-heading { display: flex; align-items: end; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.view-heading h3 { margin: 0 0 5px; font-size: 22px; }
.view-heading p { margin: 0; color: var(--muted); }
.list-panel { padding: 18px; }
.list-panel :deep(.contest-row) { cursor: pointer; }
.row-actions { display: inline-flex; align-items: center; justify-content: flex-end; gap: 2px; white-space: nowrap; }
.unit { margin-left: 8px; color: var(--muted); }
.rule-hint { margin: 8px 0 0; color: var(--muted); font-size: 12px; line-height: 1.5; }
@media (max-width: 680px) { .workspace-view { padding: 22px 14px 44px; } .view-heading { align-items: stretch; flex-direction: column; } }
</style>
