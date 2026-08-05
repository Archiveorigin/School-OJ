<template>
  <section class="page team-list-page">
    <header class="team-tabs">
      <button type="button" :class="{ active: activeTab === 'mine' }" @click="activeTab = 'mine'">我的团队</button>
      <button type="button" :class="{ active: activeTab === 'discover' }" @click="activeTab = 'discover'">发现</button>
      <el-button type="primary" plain @click="openCreate">创建团队</el-button>
    </header>

    <div v-loading="loading" class="team-grid">
      <article v-for="team in teams" :key="team.id" class="team-card" @click="openTeam(team)">
        <div class="team-card-main">
          <div class="team-icon">
            <img v-if="team.icon_url" :src="team.icon_url" alt="" />
            <img v-else src="/logo1.png" alt="" />
          </div>
          <div class="team-copy">
            <div class="team-name-row">
              <h2>{{ team.name }}</h2>
              <el-tag size="small" :type="team.visibility === 'private' ? 'danger' : 'success'" effect="plain">{{ team.visibility === 'private' ? '私有' : '公开' }}</el-tag>
            </div>
            <p>{{ team.description || '这个团队还没有填写简介。' }}</p>
          </div>
        </div>
        <footer class="team-card-footer">
          <div class="owner-meta"><span>{{ team.owner_name || '团队创建者' }}</span><code>/{{ team.slug }}</code></div>
          <div class="team-meta">
            <el-tag v-if="team.my_role" size="small" :type="team.my_role === 'owner' ? 'warning' : team.my_role === 'admin' ? 'success' : 'info'">{{ teamRoleLabel(team.my_role) }}</el-tag>
            <span>{{ team.member_count || 0 }} 名成员</span>
            <span>{{ joinModeLabel(team.join_mode) }}</span>
          </div>
          <el-button v-if="team.joined" type="primary" @click.stop="openTeam(team)">进入</el-button>
          <el-button
            v-else-if="team.application_status === 'pending'"
            disabled
            @click.stop
          >
            申请审核中
          </el-button>
          <el-button v-else type="primary" plain @click.stop="requestJoin(team)">加入</el-button>
        </footer>
      </article>
    </div>
    <el-empty v-if="!loading && !teams.length" :description="activeTab === 'mine' ? '还没有加入团队' : '暂未发现公开团队'" />

    <el-dialog v-model="createVisible" title="创建团队" width="min(820px, calc(100vw - 24px))" destroy-on-close>
      <el-form label-width="142px" class="create-team-form">
        <el-form-item label="团队图标">
          <TeamIconUpload v-model="form.icon_url" show-help />
        </el-form-item>
        <el-form-item label="团队名">
          <el-input v-model="form.name" maxlength="120" show-word-limit />
        </el-form-item>
        <el-form-item label="团队链接名">
          <el-input v-model="form.slug" maxlength="30" placeholder="以小写英文字母开头，仅含小写字母、数字和连字符" />
        </el-form-item>
        <el-form-item label="可见性">
          <el-radio-group v-model="form.visibility">
            <el-radio-button value="private">私有</el-radio-button>
            <el-radio-button value="public">公开</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="准入制">
          <el-radio-group v-model="form.join_mode">
            <el-radio-button value="invitation">邀请制</el-radio-button>
            <el-radio-button value="application">申请制</el-radio-button>
            <el-radio-button value="open">随便进</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="谁可以组织比赛">
          <el-radio-group v-model="form.contest_permission">
            <el-radio-button value="all">所有成员</el-radio-button>
            <el-radio-button value="admin">创建者和管理员</el-radio-button>
            <el-radio-button value="owner">仅创建者</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="简介">
          <el-input v-model="form.description" type="textarea" :rows="4" maxlength="140" show-word-limit placeholder="不超过 140 字符" />
        </el-form-item>
        <el-form-item label="团队公告">
          <el-input v-model="form.announcement" type="textarea" :rows="8" placeholder="支持 Markdown 文本" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createTeam">确认</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="joinVisible" :title="`加入 ${joiningTeam?.name || '团队'}`" width="460px">
      <template v-if="joiningTeam?.join_mode === 'invitation'">
        <p class="dialog-hint">该团队采用邀请制，请输入团队管理员提供的邀请码。</p>
        <el-input v-model="joinForm.join_code" placeholder="团队邀请码" @keyup.enter="joinTeam" />
      </template>
      <template v-else>
        <p class="dialog-hint">向团队管理员简单介绍一下自己，申请通过后即可进入。</p>
        <el-input v-model="joinForm.message" type="textarea" :rows="4" maxlength="300" show-word-limit placeholder="申请说明（选填）" />
      </template>
      <template #footer>
        <el-button @click="joinVisible = false">取消</el-button>
        <el-button type="primary" :loading="joining" @click="joinTeam">{{ joiningTeam?.join_mode === 'application' ? '提交申请' : '加入团队' }}</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Team, type TeamRole } from '../../api/client'
import TeamIconUpload from '../../components/TeamIconUpload.vue'

const router = useRouter()
const activeTab = ref<'mine' | 'discover'>('mine')
const teams = ref<Team[]>([])
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const joinVisible = ref(false)
const joining = ref(false)
const joiningTeam = ref<Team | null>(null)
const form = reactive({
  name: '',
  slug: '',
  visibility: 'private',
  join_mode: 'application',
  contest_permission: 'admin',
  description: '',
  announcement: '',
  icon_url: ''
})
const joinForm = reactive({ join_code: '', message: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await client.get<Team[]>('/teams', { params: { scope: activeTab.value } })
    teams.value = data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { name: '', slug: '', visibility: 'private', join_mode: 'application', contest_permission: 'admin', description: '', announcement: '', icon_url: '' })
  createVisible.value = true
}

async function createTeam() {
  if (!form.name.trim() || !form.slug.trim()) {
    ElMessage.warning('请填写团队名和团队链接名')
    return
  }
  creating.value = true
  try {
    const { data } = await client.post<Team>('/teams', form)
    createVisible.value = false
    ElMessage.success('团队创建成功')
    await router.push(`/teams/${data.slug}`)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    creating.value = false
  }
}

function requestJoin(team: Team) {
  joiningTeam.value = team
  Object.assign(joinForm, { join_code: '', message: '' })
  if (team.join_mode === 'open') {
    void joinTeam()
    return
  }
  joinVisible.value = true
}

async function joinTeam() {
  if (!joiningTeam.value) return
  joining.value = true
  try {
    const { data } = await client.post(`/teams/${joiningTeam.value.id}/join`, joinForm)
    joinVisible.value = false
    if (data.joined) {
      ElMessage.success('已加入团队')
      await router.push(`/teams/${joiningTeam.value.slug}`)
    } else {
      ElMessage.success('加入申请已提交')
      await load()
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    joining.value = false
  }
}

function openTeam(team: Team) {
  if (team.joined) router.push(`/teams/${team.slug}`)
}

function teamRoleLabel(role: TeamRole) {
  return role === 'owner' ? '创建者' : role === 'admin' ? '管理员' : '团员'
}

function joinModeLabel(mode: Team['join_mode']) {
  return mode === 'invitation' ? '邀请制' : mode === 'application' ? '申请制' : '自由加入'
}

watch(activeTab, load)
onMounted(load)
</script>

<style scoped>
.team-list-page { max-width: 1180px; margin: 0 auto; padding: 14px 28px 70px; }
.team-tabs { display: flex; align-items: center; gap: 6px; margin: 0 0 40px; border-bottom: 1px solid var(--border); }
.team-tabs button { padding: 14px 22px; color: var(--muted); border: 0; border-bottom: 3px solid transparent; background: transparent; font-size: 15px; cursor: pointer; }
.team-tabs button.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 800; }
.team-tabs > .el-button { margin: 0 0 7px auto; }
.team-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 20px; min-height: 100px; }
.team-card { position: relative; min-width: 0; overflow: hidden; border: 1px solid var(--border); border-top: 4px solid #c5304f; border-radius: 10px; background: var(--surface-strong); box-shadow: var(--shadow); cursor: pointer; transition: transform .18s ease, border-color .18s ease; }
.team-card:hover { border-color: color-mix(in srgb, var(--accent) 50%, var(--border)); transform: translateY(-2px); }
.team-card-main { display: grid; grid-template-columns: 88px minmax(0, 1fr); gap: 17px; min-height: 132px; padding: 20px 20px 16px; }
.team-icon { width: 88px; height: 88px; display: grid; place-items: center; overflow: hidden; color: #fff; border: 1px solid var(--border); border-radius: 9px; background: var(--accent); font-size: 30px; font-weight: 900; }
.team-icon img { width: 100%; height: 100%; object-fit: cover; background: #fff; }
.team-name-row { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.team-name-row h2 { margin: 0; font-size: 20px; }
.team-copy p { display: -webkit-box; margin: 10px 0 0; overflow: hidden; color: var(--muted); line-height: 1.65; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.team-card-footer { display: flex; align-items: center; gap: 12px; padding: 13px 16px; border-top: 1px solid var(--border); background: color-mix(in srgb, var(--surface-strong) 88%, var(--app-bg)); }
.owner-meta { display: grid; gap: 2px; margin-right: auto; color: var(--accent); font-size: 13px; }
.owner-meta code { color: var(--muted); font-size: 11px; }
.team-meta { display: flex; align-items: center; flex-wrap: wrap; justify-content: flex-end; gap: 9px; color: var(--muted); font-size: 12px; }
.create-team-form { padding-right: 18px; }
.dialog-hint { margin: 0 0 16px; color: var(--muted); line-height: 1.7; }
@media (max-width: 860px) { .team-grid { grid-template-columns: 1fr; } }
@media (max-width: 560px) { .team-list-page { padding: 8px 14px 48px; } .team-tabs { margin-bottom: 24px; } .team-tabs button { padding-inline: 13px; } .team-card-main { grid-template-columns: 64px minmax(0, 1fr); padding: 16px; } .team-icon { width: 64px; height: 64px; } .team-card-footer { align-items: stretch; flex-direction: column; } .team-meta { justify-content: flex-start; } .team-card-footer > .el-button { width: 100%; } .create-team-form { padding-right: 0; } }
</style>
