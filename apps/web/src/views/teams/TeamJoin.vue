<template>
  <section class="page team-join-page">
    <el-button link @click="router.push('/teams')">返回团队列表</el-button>
    <article v-loading="loading" class="panel join-panel">
      <template v-if="team">
        <img v-if="team.icon_url" :src="team.icon_url" alt="" class="team-icon" />
        <h1>加入 {{ team.name }}</h1>
        <p class="muted">/{{ team.slug }} · {{ team.visibility === 'private' ? '私有团队' : '公开团队' }}</p>
        <el-result v-if="awaitingApproval" icon="info" title="申请审核中">
          <template #extra><el-button @click="load">刷新状态</el-button></template>
        </el-result>
        <el-form v-else label-position="top" @submit.prevent="join">
          <el-form-item v-if="team.join_mode === 'invitation' && !codeResolved" label="团队邀请码" required>
            <el-input v-model="joinCode" autocomplete="off" maxlength="24" />
          </el-form-item>
          <el-form-item v-else-if="team.join_mode === 'application'" label="申请说明">
            <el-input v-model="message" type="textarea" :rows="4" maxlength="300" show-word-limit />
          </el-form-item>
          <el-button class="join-submit" native-type="submit" type="primary" :loading="joining" :disabled="loading">
            {{ team.join_mode === 'application' ? '提交申请' : '确认加入' }}
          </el-button>
        </el-form>
        <el-button v-if="!routeSlug" class="change-entry" link :disabled="joining || loading" @click="resetEntry">重新选择团队</el-button>
      </template>
      <template v-else-if="!routeSlug">
        <h1>加入团队</h1>
        <el-radio-group v-model="entryMode" class="join-modes" :disabled="loading" @change="error = ''">
          <el-radio-button value="invitation">邀请码加入</el-radio-button>
          <el-radio-button value="application">链接申请</el-radio-button>
        </el-radio-group>
        <el-form label-position="top" @submit.prevent="resolveEntry">
          <el-form-item v-if="entryMode === 'invitation'" label="团队邀请码" required>
            <el-input v-model="joinCode" autocomplete="off" maxlength="24" placeholder="输入管理员提供的邀请码" />
          </el-form-item>
          <el-form-item v-else label="团队申请链接" required>
            <el-input v-model="applicationLink" autocomplete="off" placeholder="粘贴团队申请链接" />
          </el-form-item>
          <el-button class="join-submit" native-type="submit" type="primary" :loading="loading">{{ entryMode === 'invitation' ? '查找团队' : '打开申请' }}</el-button>
        </el-form>
      </template>
      <el-result v-else-if="!loading" icon="warning" title="无法加载团队">
        <template #extra><el-button @click="load">重试</el-button></template>
      </el-result>
      <el-alert v-if="error" class="join-error" :title="error" type="error" :closable="false" />
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { generatedApi, type components } from '../../api/client'
import { parseTeamJoinSlug } from '../../features/teams/join'

const route = useRoute(), router = useRouter()
const routeSlug = computed(() => String(route.params.teamSlug || ''))
const team = ref<components['schemas']['TeamJoinPreview'] | null>(null)
const loading = ref(false), joining = ref(false), error = ref('')
const joinCode = ref(''), message = ref(''), applicationLink = ref('')
const entryMode = ref<'invitation' | 'application'>('invitation')
const codeResolved = ref(false)
const awaitingApproval = computed(() => team.value?.join_mode === 'application' && team.value.application_status === 'pending')
let generation = 0

async function acceptPreview(preview: components['schemas']['TeamJoinPreview'], ticket: number) {
  if (ticket !== generation) return
  team.value = preview
  if (preview.joined) await router.replace('/teams/' + preview.slug + '/contests')
}

async function load() {
  const id = routeSlug.value || team.value?.id
  if (!id || loading.value || joining.value) return
  const ticket = ++generation
  loading.value = true
  error.value = ''
  try { await acceptPreview((await generatedApi.previewTeamJoin({ path: { id } })).data, ticket) }
  catch (err: any) {
    if (ticket === generation) {
      team.value = null
      error.value = err.response?.status === 404 ? '团队不存在，请核对申请链接' : err.response?.data?.error || '加载失败'
    }
  } finally { if (ticket === generation) loading.value = false }
}

async function resolveEntry() {
  if (loading.value) return
  error.value = ''
  if (entryMode.value === 'application') {
    const slug = parseTeamJoinSlug(applicationLink.value)
    if (!slug) { error.value = '请填写有效的团队申请链接'; return }
    await router.push('/teams/' + slug + '/join')
    return
  }
  if (!joinCode.value.trim()) { error.value = '请输入团队邀请码'; return }
  const ticket = ++generation
  loading.value = true
  try {
    const { data } = await generatedApi.previewTeamInvitation({ body: { join_code: joinCode.value.trim() } })
    if (ticket !== generation) return
    codeResolved.value = true
    await acceptPreview(data, ticket)
  } catch (err: any) {
    if (ticket === generation) error.value = err.response?.data?.error || '邀请码查询失败'
  } finally { if (ticket === generation) loading.value = false }
}

async function join() {
  if (!team.value || joining.value || loading.value) return
  if (team.value.join_mode === 'invitation' && !joinCode.value.trim()) { error.value = '请输入团队邀请码'; return }
  const target = team.value, ticket = generation
  joining.value = true
  error.value = ''
  try {
    const { data } = await generatedApi.joinTeam({
      path: { id: target.id },
      body: { join_code: target.join_mode === 'invitation' ? joinCode.value.trim() : '', message: message.value.trim() }
    })
    if (ticket !== generation) return
    if (data.joined) {
      joinCode.value = ''
      ElMessage.success('已加入团队')
      await router.replace('/teams/' + target.slug + '/contests')
    } else {
      await acceptPreview((await generatedApi.previewTeamJoin({ path: { id: target.id } })).data, ticket)
      ElMessage.success('加入申请已提交')
    }
  } catch (err: any) {
    if (ticket === generation) { codeResolved.value = false; error.value = err.response?.data?.error || '加入失败' }
  } finally { joining.value = false }
}

function resetEntry() {
  generation++
  team.value = null
  joinCode.value = ''
  message.value = ''
  codeResolved.value = false
  error.value = ''
  loading.value = false
}
watch(routeSlug, () => { resetEntry(); void load() }, { immediate: true })
onMounted(() => window.addEventListener('focus', load))
onBeforeUnmount(() => { generation++; joinCode.value = ''; window.removeEventListener('focus', load) })
</script>

<style scoped>
.team-join-page { max-width: 640px; margin: 20px auto; }
.join-panel { margin-top: 16px; padding: 28px; min-height: 240px; }
.join-panel h1 { margin: 0 0 8px; font-size: 25px; overflow-wrap: anywhere; }
.team-icon { width: 60px; height: 60px; object-fit: contain; object-position: center; border-radius: var(--radius-lg); margin-bottom: 16px; }
.join-modes { margin: 20px 0 24px; }
.join-submit { margin-top: 12px; width: 100%; }
.change-entry, .join-error { margin-top: 20px; }
@media (max-width: 600px) { .join-panel { padding: 20px; } }
</style>
