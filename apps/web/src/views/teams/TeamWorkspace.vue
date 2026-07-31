<template>
  <section class="team-workspace">
    <aside v-if="team" class="team-sidebar">
      <button type="button" class="back-link" @click="router.push('/teams')">← 返回团队列表</button>
      <div class="workspace-icon">
        <img v-if="team.icon_url" :src="team.icon_url" alt="" />
        <span v-else>{{ team.name.slice(0, 1).toUpperCase() }}</span>
      </div>
      <h1>{{ team.name }}</h1>
      <code>/{{ team.slug }}</code>
      <p class="description">{{ team.description || '这个团队还没有填写简介。' }}</p>

      <section class="side-info">
        <span>我的身份</span>
        <strong>{{ teamRoleLabel(team.my_role) }}</strong>
        <span>创建者</span>
        <strong>{{ team.owner_name || '未知' }}</strong>
        <span>成员</span>
        <strong>{{ team.member_count || 0 }} 人</strong>
        <span>准入方式</span>
        <strong>{{ joinModeLabel(team.join_mode) }}</strong>
      </section>

      <section v-if="team.announcement" class="announcement">
        <span>团队公告</span>
        <MarkdownRenderer :source="team.announcement" />
      </section>

      <el-button v-if="canManage" plain @click="settingsVisible = true">团队设置</el-button>
      <el-button v-if="team.my_role && team.my_role !== 'owner'" type="danger" plain @click="leaveTeam">退出团队</el-button>
    </aside>
    <el-skeleton v-else :rows="10" animated class="team-sidebar" />

    <main class="team-main">
      <header v-if="team" class="team-main-header">
        <div>
          <span>TEAM SPACE</span>
          <h2>{{ team.name }}</h2>
        </div>
        <nav>
          <RouterLink :to="`${basePath}/contests`" active-class="active">比赛</RouterLink>
          <RouterLink :to="`${basePath}/problem-sets`" active-class="active">题单</RouterLink>
          <RouterLink :to="`${basePath}/members`" active-class="active">成员</RouterLink>
        </nav>
      </header>
      <router-view v-if="team" :team="team" @team-updated="loadTeam" />
    </main>

    <el-dialog v-if="team" v-model="settingsVisible" title="团队设置" width="min(720px, calc(100vw - 24px))">
      <el-form label-width="120px">
        <el-form-item label="团队名"><el-input v-model="settings.name" maxlength="120" /></el-form-item>
        <el-form-item label="可见性">
          <el-radio-group v-model="settings.visibility" :disabled="team.my_role !== 'owner'">
            <el-radio-button value="private">私有</el-radio-button>
            <el-radio-button value="public">公开</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="准入制">
          <el-select v-model="settings.join_mode" :disabled="team.my_role !== 'owner'">
            <el-option label="邀请制" value="invitation" />
            <el-option label="申请制" value="application" />
            <el-option label="随便进" value="open" />
          </el-select>
        </el-form-item>
        <el-form-item label="组织权限">
          <el-select v-model="settings.contest_permission" :disabled="team.my_role !== 'owner'">
            <el-option label="所有成员" value="all" />
            <el-option label="创建者和管理员" value="admin" />
            <el-option label="仅创建者" value="owner" />
          </el-select>
        </el-form-item>
        <el-form-item label="简介"><el-input v-model="settings.description" type="textarea" :rows="3" maxlength="140" show-word-limit /></el-form-item>
        <el-form-item label="团队公告"><el-input v-model="settings.announcement" type="textarea" :rows="7" /></el-form-item>
        <el-form-item v-if="team.join_mode === 'invitation'" label="团队邀请码"><el-input :model-value="team.join_code" readonly /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, type Team, type TeamRole } from '../../api/client'
import MarkdownRenderer from '../../components/MarkdownRenderer.vue'

const route = useRoute()
const router = useRouter()
const team = ref<Team | null>(null)
const settingsVisible = ref(false)
const saving = ref(false)
const basePath = computed(() => `/teams/${route.params.teamSlug}`)
const canManage = computed(() => team.value?.my_role === 'owner' || team.value?.my_role === 'admin')
const settings = reactive({
  name: '',
  visibility: 'private',
  join_mode: 'application',
  contest_permission: 'admin',
  description: '',
  announcement: '',
  icon_url: ''
})

async function loadTeam() {
  try {
    const { data } = await client.get<Team>(`/teams/${encodeURIComponent(String(route.params.teamSlug))}`)
    if (!data.joined) {
      ElMessage.warning('加入团队后才能进入团队空间')
      await router.replace('/teams')
      return
    }
    team.value = data
    Object.assign(settings, {
      name: data.name,
      visibility: data.visibility,
      join_mode: data.join_mode,
      contest_permission: data.contest_permission,
      description: data.description || '',
      announcement: data.announcement || '',
      icon_url: data.icon_url || ''
    })
  } catch (err: any) {
    ElMessage.error(err.response?.status === 404 ? '团队不存在或无权访问' : err.response?.data?.error || err.message)
    await router.replace('/teams')
  }
}

async function saveSettings() {
  if (!team.value) return
  saving.value = true
  try {
    await client.put(`/teams/${team.value.id}`, settings)
    settingsVisible.value = false
    ElMessage.success('团队设置已更新')
    await loadTeam()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}

async function leaveTeam() {
  if (!team.value) return
  try {
    await ElMessageBox.confirm('退出后将无法访问团队比赛、私有题单与讨论，确认退出？', '退出团队', { type: 'warning' })
    await client.post(`/teams/${team.value.id}/leave`)
    ElMessage.success('已退出团队')
    await router.push('/teams')
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  }
}

function teamRoleLabel(role?: TeamRole) {
  return role === 'owner' ? '创建者' : role === 'admin' ? '管理员' : '团员'
}

function joinModeLabel(mode: Team['join_mode']) {
  return mode === 'invitation' ? '邀请制' : mode === 'application' ? '申请制' : '自由加入'
}

watch(() => route.params.teamSlug, loadTeam, { immediate: true })
</script>

<style scoped>
.team-workspace { min-height: calc(100vh - 64px); display: grid; grid-template-columns: 300px minmax(0, 1fr); background: var(--app-bg); }
.team-sidebar { min-height: calc(100vh - 64px); display: flex; align-items: stretch; flex-direction: column; gap: 12px; padding: 30px 26px; border-right: 1px solid var(--border); background: var(--surface-strong); }
.back-link { align-self: flex-start; padding: 0 0 14px; color: var(--accent); border: 0; background: transparent; cursor: pointer; }
.workspace-icon { width: 82px; height: 82px; display: grid; place-items: center; overflow: hidden; margin-top: 5px; color: #fff; border-radius: 22px; background: linear-gradient(135deg, #0a5ea6, #14b8a6); font-size: 34px; font-weight: 900; }
.workspace-icon img { width: 100%; height: 100%; object-fit: cover; }
.team-sidebar h1 { margin: 5px 0 -6px; font-size: 24px; }
.team-sidebar code { color: var(--accent); }
.description { margin: 4px 0; color: var(--muted); line-height: 1.65; }
.side-info { display: grid; grid-template-columns: 1fr auto; gap: 10px 14px; margin: 7px 0; padding: 16px 0; border-block: 1px solid var(--border); font-size: 13px; }
.side-info span { color: var(--muted); }
.announcement { margin-bottom: auto; padding: 14px; border-radius: 10px; background: var(--app-bg); }
.announcement > span { color: var(--muted); font-size: 12px; font-weight: 700; }
.announcement :deep(.markdown-body) { font-size: 13px; }
.team-main { min-width: 0; }
.team-main-header { display: flex; align-items: end; justify-content: space-between; gap: 22px; padding: 28px 34px 0; border-bottom: 1px solid var(--border); background: var(--surface-strong); }
.team-main-header span { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .14em; }
.team-main-header h2 { margin: 6px 0 20px; font-size: 25px; }
.team-main-header nav { display: flex; gap: 5px; }
.team-main-header a { padding: 18px 25px 16px; color: var(--muted); border-bottom: 3px solid transparent; }
.team-main-header a.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
@media (max-width: 820px) { .team-workspace { grid-template-columns: 1fr; } .team-sidebar { min-height: auto; border-right: 0; border-bottom: 1px solid var(--border); } .announcement { margin-bottom: 0; } .team-main-header { align-items: stretch; flex-direction: column; padding-inline: 18px; } .team-main-header h2 { margin-bottom: 0; } .team-main-header nav { overflow-x: auto; } .team-main-header a { white-space: nowrap; } }
</style>
