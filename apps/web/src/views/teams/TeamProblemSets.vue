<template>
  <section class="workspace-view">
    <div class="view-heading">
      <div><h3>团队题单</h3><p>在团队内部组织训练题目</p></div>
      <el-button v-if="canOrganize" type="primary" @click="createVisible = true">创建题单</el-button>
    </div>
    <div v-loading="loading" class="problem-set-grid">
      <article v-for="item in items" :key="item.id" class="panel problem-set-card" @click="openSet(item)">
        <div class="set-symbol"><img :src="item.icon_url || team.icon_url || '/logo1.png'" alt="题单图标" /></div>
        <div>
          <h4>{{ item.title }}</h4>
          <p>{{ item.description || '暂无题单说明' }}</p>
          <span>{{ item.problem_count || 0 }} 道题目</span>
        </div>
        <div class="card-actions" @click.stop>
          <el-button v-if="item.can_edit" @click.stop="openEdit(item)">编辑</el-button>
          <el-button
            v-if="item.can_delete"
            type="danger"
            plain
            :loading="deletingID === item.id"
            @click.stop="deleteSet(item)"
          >删除</el-button>
          <el-button type="primary" plain @click.stop="openSet(item)">打开题单</el-button>
        </div>
      </article>
    </div>
    <el-empty v-if="!loading && !items.length" description="暂无团队题单" />

    <el-dialog v-model="createVisible" title="创建团队题单" width="min(520px, calc(100vw - 24px))">
      <el-form label-position="top">
        <el-form-item label="题单图标"><TeamIconUpload v-model="form.icon_url" :fallback="team.icon_url" label="题单图标" show-help /></el-form-item>
        <el-form-item label="题单标题"><el-input v-model="form.title" maxlength="200" /></el-form-item>
        <el-form-item label="题单说明"><el-input v-model="form.description" type="textarea" :rows="5" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createSet">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" title="编辑题单信息" width="min(520px, calc(100vw - 24px))">
      <el-form label-position="top">
        <el-form-item label="题单图标"><TeamIconUpload v-model="editForm.icon_url" :fallback="team.icon_url" label="题单图标" show-help /></el-form-item>
        <el-form-item label="题单标题"><el-input v-model="editForm.title" maxlength="200" /></el-form-item>
        <el-form-item label="题单说明"><el-input v-model="editForm.description" type="textarea" :rows="5" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="editing" @click="saveSet">保存修改</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { apiErrorMessage } from '../../api/errors'
import TeamIconUpload from '../../components/TeamIconUpload.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { generatedApi, type Team, type TeamProblemSet } from '../../api/client'

const props = defineProps<{ team: Team }>()
const router = useRouter()
const items = ref<TeamProblemSet[]>([])
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const editVisible = ref(false)
const editing = ref(false)
const editingItem = ref<TeamProblemSet | null>(null)
const deletingID = ref<number | null>(null)
const form = reactive({ title: '', description: '', icon_url: '' })
const editForm = reactive({ title: '', description: '', icon_url: '' })
const canOrganize = computed(() => Boolean(props.team.can_organize))

async function load() {
  loading.value = true
  try {
    items.value = (await generatedApi.listTeamProblemSets<any>({ path: { "id": props.team.id } })).data || []
  } catch (err: any) {
    ElMessage.error(apiErrorMessage(err))
  } finally {
    loading.value = false
  }
}

function openEdit(item: TeamProblemSet) {
  editingItem.value = item
  Object.assign(editForm, { title: item.title, description: item.description || '', icon_url: item.icon_url || '' })
  editVisible.value = true
}

async function saveSet() {
  if (!editingItem.value || !editForm.title.trim()) {
    ElMessage.warning('请输入题单标题')
    return
  }
  editing.value = true
  try {
    await generatedApi.updateTeamProblemSetViaProblemSetsSetId<any>({ path: { "set_id": editingItem.value.id }, body: editForm })
    editVisible.value = false
    editingItem.value = null
    ElMessage.success('题单信息已更新')
    await load()
  } catch (err: any) {
    ElMessage.error(apiErrorMessage(err))
  } finally {
    editing.value = false
  }
}

async function deleteSet(item: TeamProblemSet) {
  try {
    await ElMessageBox.confirm(
      `确认删除题单“${item.title}”？题单内的题目本身不会被删除，此操作不可撤销。`,
      '删除团队题单',
      { type: 'error', confirmButtonText: '确认删除', cancelButtonText: '取消', confirmButtonClass: 'el-button--danger' },
    )
    deletingID.value = item.id
    await generatedApi.deleteTeamProblemSetViaProblemSetsSetId<any>({ path: { "set_id": item.id } })
    ElMessage.success('团队题单已删除')
    await load()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(apiErrorMessage(err))
  } finally {
    deletingID.value = null
  }
}

async function createSet() {
  if (!form.title.trim()) {
    ElMessage.warning('请输入题单标题')
    return
  }
  creating.value = true
  try {
    const { data } = await generatedApi.createTeamProblemSet<TeamProblemSet>({ path: { "id": props.team.id }, body: form })
    Object.assign(form, { title: '', description: '', icon_url: '' })
    createVisible.value = false
    ElMessage.success('团队题单已创建')
    await router.push(`/problem-set/${data.id}#problems`)
  } catch (err: any) {
    ElMessage.error(apiErrorMessage(err))
  } finally {
    creating.value = false
  }
}

function openSet(item: TeamProblemSet) {
  router.push(`/problem-set/${item.id}#problems`)
}

onMounted(load)
</script>

<style scoped>
.set-symbol img { display: block; width: 100%; height: 100%; object-fit: contain; object-position: center; border-radius: inherit; background: var(--color-surface-solid); }
.workspace-view { padding: 28px 34px 54px; }
.view-heading { display: flex; align-items: end; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.view-heading h3 { margin: 0 0 5px; font-size: 22px; }
.view-heading p { margin: 0; color: var(--muted); }
.problem-set-grid { display: grid; gap: 12px; }
.problem-set-card { display: grid; grid-template-columns: 48px minmax(0, 1fr) auto; align-items: center; gap: 15px; cursor: pointer; }
.set-symbol { width: 48px; height: 48px; display: grid; place-items: center; color: var(--color-on-emphasis); border-radius: 13px; background: linear-gradient(135deg, var(--color-primary), var(--color-secondary)); font-weight: 900; }
.problem-set-card h4 { margin: 0 0 6px; font-size: 17px; }
.problem-set-card p { margin: 0 0 7px; color: var(--muted); }
.problem-set-card span { color: var(--accent); font-size: 12px; }
.card-actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
@media (max-width: 760px) { .problem-set-card { grid-template-columns: 42px minmax(0, 1fr); } .card-actions { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); } .card-actions .el-button { width: 100%; margin: 0; } }
@media (max-width: 680px) { .workspace-view { padding: 22px 14px 44px; } .view-heading { align-items: stretch; flex-direction: column; } }
@media (max-width: 460px) { .card-actions { grid-template-columns: 1fr; } }
</style>
