<template>
  <section class="workspace-view">
    <div class="view-heading">
      <div><h3>团队比赛</h3><p>比赛仅对团队成员可见，不会出现在公共比赛列表。</p></div>
      <el-button v-if="canOrganize" type="primary" @click="createVisible = true">创建比赛</el-button>
    </div>
    <div v-loading="loading" class="panel list-panel">
      <el-table :data="contests">
        <el-table-column prop="title" label="标题" min-width="240" />
        <el-table-column label="开始时间" width="190">
          <template #default="{ row }">{{ row.starts_at ? formatTime(row.starts_at) : '待定' }}</template>
        </el-table-column>
        <el-table-column label="时长" width="130">
          <template #default="{ row }">{{ durationText(row.duration_minutes) }}</template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && !contests.length" description="暂无团队比赛" />
    </div>

    <el-dialog v-model="createVisible" title="创建团队比赛" width="560px">
      <el-form label-position="top">
        <el-form-item label="比赛标题"><el-input v-model="form.title" maxlength="200" /></el-form-item>
        <el-form-item label="开始时间"><el-date-picker v-model="form.starts_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" placeholder="选择开始时间" /></el-form-item>
        <el-form-item label="比赛时长">
          <el-input-number v-model="form.duration_minutes" :min="15" :step="15" />
          <span class="unit">分钟</span>
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
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { client, type Team } from '../../api/client'
import { formatDateTime } from '../../features/time'

const props = defineProps<{ team: Team }>()
const contests = ref<any[]>([])
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const form = reactive({ title: '', description: '', starts_at: '', duration_minutes: 120 })
const canOrganize = computed(() => {
  if (props.team.my_role === 'owner') return true
  if (props.team.contest_permission === 'all') return Boolean(props.team.my_role)
  return props.team.contest_permission === 'admin' && props.team.my_role === 'admin'
})

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
  creating.value = true
  try {
    await client.post(`/teams/${props.team.id}/contests`, { ...form, starts_at: form.starts_at || null })
    Object.assign(form, { title: '', description: '', starts_at: '', duration_minutes: 120 })
    createVisible.value = false
    ElMessage.success('团队比赛已创建')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    creating.value = false
  }
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
.unit { margin-left: 8px; color: var(--muted); }
@media (max-width: 680px) { .workspace-view { padding: 22px 14px 44px; } .view-heading { align-items: stretch; flex-direction: column; } }
</style>
