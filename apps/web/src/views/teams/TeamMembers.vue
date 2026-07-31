<template>
  <section class="workspace-view">
    <div class="view-heading">
      <div><h3>团队成员</h3><p>职级按创建者、管理员、团员细分，管理权限清晰可控。</p></div>
      <el-tag type="info">{{ members.length }} 名成员</el-tag>
    </div>

    <div v-if="canManage && applications.length" class="panel application-panel">
      <h4>待审核申请</h4>
      <div v-for="item in applications" :key="item.id" class="application-row">
        <div>
          <strong>{{ item.user_name }}</strong>
          <span>{{ item.email }} · {{ userRoleLabel(item.user_role) }}</span>
          <p>{{ item.message || '未填写申请说明' }}</p>
        </div>
        <div>
          <el-button size="small" @click="review(item, 'reject')">拒绝</el-button>
          <el-button size="small" type="primary" @click="review(item, 'approve')">通过</el-button>
        </div>
      </div>
    </div>

    <div v-loading="loading" class="panel member-panel">
      <el-table :data="members">
        <el-table-column label="成员" min-width="220">
          <template #default="{ row }">
            <div class="member-person">
              <el-avatar :size="34" :src="row.avatar_url">{{ row.name.slice(0, 1) }}</el-avatar>
              <div><strong>{{ row.name }}</strong><span>{{ row.email }}</span></div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="账号身份" width="120">
          <template #default="{ row }">{{ userRoleLabel(row.user_role) }}</template>
        </el-table-column>
        <el-table-column label="团队职级" width="150">
          <template #default="{ row }">
            <el-select
              v-if="canChangeRole(row)"
              :model-value="row.team_role"
              size="small"
              @change="(role: string) => changeRole(row, role)"
            >
              <el-option label="管理员" value="admin" />
              <el-option label="团员" value="member" />
            </el-select>
            <el-tag v-else :type="row.team_role === 'owner' ? 'warning' : row.team_role === 'admin' ? 'success' : 'info'">
              {{ teamRoleLabel(row.team_role) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canManage" label="操作" width="100" align="right">
          <template #default="{ row }">
            <el-button v-if="canRemove(row)" type="danger" text @click="removeMember(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { client, type Role, type Team, type TeamRole } from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const props = defineProps<{ team: Team }>()
const auth = useAuthStore()
const members = ref<any[]>([])
const applications = ref<any[]>([])
const loading = ref(false)
const canManage = computed(() => props.team.my_role === 'owner' || props.team.my_role === 'admin')

async function load() {
  loading.value = true
  try {
    members.value = (await client.get(`/teams/${props.team.id}/members`)).data || []
    applications.value = canManage.value ? (await client.get(`/teams/${props.team.id}/applications`)).data || [] : []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

async function review(item: any, action: 'approve' | 'reject') {
  try {
    await client.put(`/teams/${props.team.id}/applications/${item.id}`, { action })
    ElMessage.success(action === 'approve' ? '已同意加入申请' : '已拒绝加入申请')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

function canChangeRole(row: any) {
  return props.team.my_role === 'owner' && row.team_role !== 'owner' && row.user_id !== auth.user?.id
}

function canRemove(row: any) {
  if (row.team_role === 'owner' || row.user_id === auth.user?.id) return false
  if (props.team.my_role === 'owner') return true
  return props.team.my_role === 'admin' && row.team_role === 'member'
}

async function changeRole(row: any, role: string) {
  try {
    await client.put(`/teams/${props.team.id}/members/${row.user_id}`, { role })
    ElMessage.success('成员职级已更新')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function removeMember(row: any) {
  try {
    await ElMessageBox.confirm(`确认将 ${row.name} 移出团队？`, '移除成员', { type: 'warning' })
    await client.delete(`/teams/${props.team.id}/members/${row.user_id}`)
    ElMessage.success('成员已移除')
    await load()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  }
}

function teamRoleLabel(role: TeamRole) {
  return role === 'owner' ? '创建者' : role === 'admin' ? '管理员' : '团员'
}

function userRoleLabel(role: Role) {
  return role === 'admin' ? '管理员' : role === 'teacher' ? '教师' : '学生'
}

onMounted(load)
</script>

<style scoped>
.workspace-view { padding: 28px 34px 54px; }
.view-heading { display: flex; align-items: end; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.view-heading h3 { margin: 0 0 5px; font-size: 22px; }
.view-heading p { margin: 0; color: var(--muted); }
.application-panel { margin-bottom: 14px; }
.application-panel h4 { margin: 0 0 12px; }
.application-row { display: flex; align-items: center; justify-content: space-between; gap: 18px; padding: 13px 0; border-top: 1px solid var(--border); }
.application-row > div:first-child { display: grid; gap: 4px; }
.application-row span, .application-row p { margin: 0; color: var(--muted); font-size: 13px; }
.member-panel { padding: 18px; }
.member-person { display: flex; align-items: center; gap: 10px; }
.member-person > div { display: grid; gap: 2px; }
.member-person span { color: var(--muted); font-size: 12px; }
@media (max-width: 680px) { .workspace-view { padding: 22px 14px 44px; } .view-heading, .application-row { align-items: stretch; flex-direction: column; } }
</style>
