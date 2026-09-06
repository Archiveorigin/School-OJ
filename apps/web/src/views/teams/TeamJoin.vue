<template>
  <section class="page team-join-page">
    <el-button link @click="router.push('/teams')">返回团队列表</el-button>
    <article v-loading="loading" class="panel join-panel">
      <template v-if="team">
        <img v-if="team.icon_url" :src="team.icon_url" alt="" class="team-icon" />
        <h1>加入 {{ team.name }}</h1><p class="muted">/{{ team.slug }} · {{ team.visibility === 'private' ? '私有团队' : '公开团队' }}</p>
        <el-result v-if="team.application_status === 'pending'" icon="info" title="申请审核中"><template #extra><el-button @click="load">刷新状态</el-button></template></el-result>
        <el-form v-else label-position="top" @submit.prevent="join">
          <el-form-item v-if="team.join_mode === 'invitation'" label="团队邀请码" required><el-input v-model="joinCode" autocomplete="off" maxlength="24" /></el-form-item>
          <el-form-item v-else-if="team.join_mode === 'application'" label="申请说明"><el-input v-model="message" type="textarea" :rows="4" maxlength="300" show-word-limit /></el-form-item>
          <el-alert v-if="error" :title="error" type="error" :closable="false" />
          <el-button class="join-submit" native-type="submit" type="primary" :loading="joining">{{ team.join_mode === 'application' ? '提交申请' : '确认加入' }}</el-button>
        </el-form>
      </template>
      <el-result v-else-if="error" icon="warning" title="无法加入团队" :sub-title="error"><template #extra><el-button @click="load">重试</el-button></template></el-result>
    </article>
  </section>
</template>
<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { generatedApi, type components } from '../../api/client'
const route = useRoute(), router = useRouter()
const team = ref<components['schemas']['TeamJoinPreview'] | null>(null), loading = ref(false), joining = ref(false), error = ref(''), joinCode = ref(''), message = ref('')
async function load() {
  if (loading.value) return
  loading.value = true; error.value = ''
  try {
    team.value = (await generatedApi.previewTeamJoin({ path: { id: String(route.params.teamSlug) } })).data
    if (team.value.joined) await router.replace(`/teams/${team.value.slug}/contests`)
  } catch (err: any) { team.value = null; error.value = err.response?.status === 404 ? '团队不存在，请核对加入链接' : err.response?.data?.error || '加载失败' }
  finally { loading.value = false }
}
async function join() {
  if (!team.value || joining.value) return
  if (team.value.join_mode === 'invitation' && !joinCode.value.trim()) { error.value = '请输入团队邀请码'; return }
  joining.value = true; error.value = ''
  try {
    const { data } = await generatedApi.joinTeam({ path: { id: team.value.id }, body: { join_code: joinCode.value.trim(), message: message.value.trim() } })
    if (data.joined) { ElMessage.success('已加入团队'); await router.replace(`/teams/${team.value.slug}/contests`) }
    else { await load(); ElMessage.success('加入申请已提交') }
  } catch (err: any) { error.value = err.response?.data?.error || '加入失败' }
  finally { joining.value = false }
}
watch(() => route.params.teamSlug, () => { team.value = null; joinCode.value = ''; message.value = ''; void load() })
onMounted(() => { void load(); window.addEventListener('focus', load) })
onBeforeUnmount(() => window.removeEventListener('focus', load))
</script>
<style scoped>
.team-join-page { max-width: 640px; margin: 20px auto; }
.join-panel { margin-top: 16px; padding: 28px; min-height: 240px; }
.join-panel h1 { margin: 0 0 8px; font-size: 25px; overflow-wrap: anywhere; }
.team-icon { width: 60px; height: 60px; object-fit: cover; border-radius: 12px; margin-bottom: 16px; }
.join-submit { margin-top: 20px; width: 100%; }
</style>
