<template>
  <section class="workspace-view">
    <div class="view-heading">
      <div><h3>团队题单</h3><p>在团队内部组织训练题目，私有题目不会公开到公共题库。</p></div>
      <el-button v-if="canOrganize" type="primary" @click="createVisible = true">创建题单</el-button>
    </div>
    <div v-loading="loading" class="problem-set-grid">
      <article v-for="item in items" :key="item.id" class="panel problem-set-card" @click="openSet(item)">
        <div class="set-symbol">题</div>
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Team, type TeamProblemSet } from '../../api/client'

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
const form = reactive({ title: '', description: '' })
const editForm = reactive({ title: '', description: '' })
const canOrganize = computed(() => {
  if (props.team.my_role === 'owner') return true
  if (props.team.contest_permission === 'all') return Boolean(props.team.my_role)
  return props.team.contest_permission === 'admin' && props.team.my_role === 'admin'
})

async function load() {
  loading.value = true
  try {
    items.value = (await client.get(`/teams/${props.team.id}/problem-sets`)).data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function openEdit(item: TeamProblemSet) {
  editingItem.value = item
  Object.assign(editForm, { title: item.title, description: item.description || '' })
  editVisible.value = true
}

async function saveSet() {
  if (!editingItem.value || !editForm.title.trim()) {
    ElMessage.warning('请输入题单标题')
    return
  }
  editing.value = true
  try {
    await client.put(`/problem-sets/${editingItem.value.id}`, editForm)
    editVisible.value = false
    editingItem.value = null
    ElMessage.success('题单信息已更新')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
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
    await client.delete(`/problem-sets/${item.id}`)
    ElMessage.success('团队题单已删除')
    await load()
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
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
    const { data } = await client.post<TeamProblemSet>(`/teams/${props.team.id}/problem-sets`, form)
    Object.assign(form, { title: '', description: '' })
    createVisible.value = false
    ElMessage.success('团队题单已创建')
    await router.push(`/problem-set/${data.id}#problems`)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
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
.workspace-view { padding: 28px 34px 54px; }
.view-heading { display: flex; align-items: end; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.view-heading h3 { margin: 0 0 5px; font-size: 22px; }
.view-heading p { margin: 0; color: var(--muted); }
.problem-set-grid { display: grid; gap: 12px; }
.problem-set-card { display: grid; grid-template-columns: 48px minmax(0, 1fr) auto; align-items: center; gap: 15px; cursor: pointer; }
.set-symbol { width: 48px; height: 48px; display: grid; place-items: center; color: #fff; border-radius: 13px; background: linear-gradient(135deg, #0a5ea6, #14b8a6); font-weight: 900; }
.problem-set-card h4 { margin: 0 0 6px; font-size: 17px; }
.problem-set-card p { margin: 0 0 7px; color: var(--muted); }
.problem-set-card span { color: var(--accent); font-size: 12px; }
.card-actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
@media (max-width: 760px) { .problem-set-card { grid-template-columns: 42px minmax(0, 1fr); } .card-actions { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); } .card-actions .el-button { width: 100%; margin: 0; } }
@media (max-width: 680px) { .workspace-view { padding: 22px 14px 44px; } .view-heading { align-items: stretch; flex-direction: column; } }
@media (max-width: 460px) { .card-actions { grid-template-columns: 1fr; } }
</style>
