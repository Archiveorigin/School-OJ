<template>
  <el-dialog :model-value="modelValue" title="添加题目" width="min(1040px, calc(100vw - 24px))" append-to-body destroy-on-close :close-on-click-modal="!saving" :close-on-press-escape="!saving" :show-close="!saving" @update:model-value="$emit('update:modelValue', $event)">
    <div class="problem-picker">
      <div class="picker-filters">
        <el-input v-model="filters.q" clearable placeholder="搜索题号或题目名称" aria-label="搜索题号或题目名称" @keyup.enter="search" />
        <el-select v-model="filters.difficulty" clearable placeholder="全部难度" aria-label="题目难度"><el-option v-for="difficulty in problemDifficultyOptions" :key="difficulty" :value="difficulty" :label="difficulty" /></el-select>
        <ProblemTagSelector v-model="filters.tags" />
        <el-button type="primary" @click="search">检索</el-button><el-button @click="reset">重置</el-button>
      </div>
      <p class="picker-note">可检索公共题库及本团队私有题目。已加入题单的题目不可重复选择，每次最多添加 100 道。</p>
      <div class="picker-tags"><el-tag v-for="tag in filters.tags" :key="tag" closable @close="filters.tags = filters.tags.filter(t => t !== tag)">{{ tag }}</el-tag></div>
      <el-alert v-if="error" :title="error" type="error" :closable="false" />
      <div class="picker-results" v-loading="loading">
        <div v-for="problem in items" :key="problem.id" class="picker-row">
          <el-checkbox :model-value="selected.has(problem.id) || existingIDs.includes(problem.id)" :disabled="saving || existingIDs.includes(problem.id)" :aria-label="`选择 ${problemDisplayCode(problem)} ${problem.title}`" @change="toggle(problem)"><span class="problem-code">{{ problemDisplayCode(problem) }}</span> {{ problem.title }}</el-checkbox>
          <el-tag v-if="existingIDs.includes(problem.id)" type="info">已在题单</el-tag><span v-else class="difficulty">{{ problemDifficulty(problem) }}</span>
        </div>
        <el-empty v-if="!loading && !items.length" description="暂无符合条件的题目" />
      </div>
      <el-pagination v-model:current-page="page" :page-size="20" :total="total" :pager-count="5" layout="prev, pager, next" @current-change="load" />
      <section class="selection-tray" aria-label="已选题目">
        <header><strong>已选 {{ selected.size }} 道题目</strong><el-button link :disabled="saving || !selected.size" @click="selected = new Map()">清空已选</el-button></header>
        <div class="selected-problems"><el-tag v-for="problem in selected.values()" :key="problem.id" :closable="!saving" @close="toggle(problem)">{{ problemDisplayCode(problem) }} · {{ problem.title }}</el-tag><span v-if="!selected.size" class="picker-note">跨页选择会保留在这里</span></div>
      </section>
    </div>
    <template #footer><el-button :disabled="saving" @click="$emit('update:modelValue', false)">取消</el-button><el-button type="primary" :loading="saving" :disabled="!selected.size" @click="add">添加已选 {{ selected.size }} 道题目</el-button></template>
  </el-dialog>
</template>
<script setup lang="ts">
import { apiErrorMessage } from '../api/errors'
import { onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { generatedApi, type ProblemCatalogItem, type ProblemCatalogResponse } from '../api/client'
import { problemDifficulty, problemDifficultyOptions, problemDisplayCode } from '../features/problems/problemMeta'
import ProblemTagSelector from './ProblemTagSelector.vue'
const props = defineProps<{ modelValue: boolean; teamID: number; setID: number; existingIDs: number[] }>()
const emit = defineEmits<{ 'update:modelValue': [boolean]; added: [] }>()
const filters = reactive({ q: '', difficulty: '', tags: [] as string[] }), query = ref({ ...filters })
const selected = ref(new Map<number, ProblemCatalogItem>()), items = ref<ProblemCatalogItem[]>([])
const page = ref(1), total = ref(0), loading = ref(false), saving = ref(false), error = ref('')
let controller: AbortController | undefined, sequence = 0
async function load() {
  controller?.abort(); controller = new AbortController(); const current = ++sequence, signal = controller.signal
  loading.value = true; error.value = ''
  try {
    const { data } = await generatedApi.listProblemCatalog<ProblemCatalogResponse>({ signal, query: { q: query.value.q, difficulty: query.value.difficulty, tags: query.value.tags.join(','), team_id: props.teamID, page: page.value, page_size: 20 } })
    if (current !== sequence) return
    items.value = data.items; total.value = data.total
  } catch (err: any) { if (!signal.aborted) error.value = apiErrorMessage(err, '题目加载失败，请重新检索') }
  finally { if (current === sequence) loading.value = false }
}
function search() { query.value = { ...filters, tags: [...filters.tags] }; page.value = 1; void load() }
function reset() { Object.assign(filters, { q: '', difficulty: '', tags: [] }); search() }
function toggle(problem: ProblemCatalogItem) {
  if (saving.value || props.existingIDs.includes(problem.id)) return
  if (selected.value.has(problem.id)) selected.value.delete(problem.id)
  else if (selected.value.size >= 100) ElMessage.warning('每次最多选择 100 道题目')
  else selected.value.set(problem.id, problem)
}
async function add() {
  if (saving.value || !selected.value.size) return
  saving.value = true
  try {
    const { data } = await generatedApi.addTeamProblemSetProblemViaProblemSetsSetIdProblems<{ added: number; skipped: number }>({ path: { set_id: props.setID }, body: { problem_ids: [...selected.value.keys()] } })
    ElMessage.success(`已添加 ${data.added} 道题目${data.skipped ? `，${data.skipped} 道已在题单` : ''}`)
    selected.value = new Map(); emit('added'); emit('update:modelValue', false)
  } catch (err: any) { ElMessage.error(apiErrorMessage(err, '添加失败，已选题目已保留，请重试')) }
  finally { saving.value = false }
}
watch(() => props.modelValue, value => { if (value) { for (const id of props.existingIDs) selected.value.delete(id); void load() } else { controller?.abort(); sequence++; loading.value = false } })
watch(() => props.setID, () => { selected.value = new Map(); reset() })
onBeforeUnmount(() => controller?.abort())
</script>
<style scoped>
.problem-picker { min-width: 0; }
.picker-filters { display: grid; grid-template-columns: minmax(180px, 1fr) 130px auto auto auto; gap: 10px; align-items: center; }
.picker-filters .el-button { margin: 0; }
.picker-note { color: var(--muted); font-size: 12px; line-height: 1.7; }
.picker-tags { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px; }
.picker-results { min-height: 120px; max-height: 42dvh; overflow: auto; border: 1px solid var(--border); border-radius: 8px; }
.picker-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-height: 46px; padding: 9px 14px; border-bottom: 1px solid var(--border); }
.picker-row .el-checkbox { min-width: 0; height: auto; }
.picker-row :deep(.el-checkbox__label) { white-space: normal; overflow-wrap: anywhere; }
.problem-code { color: var(--accent); font-weight: 700; margin-right: 8px; }
.difficulty { white-space: nowrap; font-size: 12px; color: var(--muted); }
.el-pagination { justify-content: flex-end; margin: 14px 0; }
.selection-tray { padding: 14px; background: var(--app-bg); border-radius: 8px; border: 1px solid var(--border); }
.selection-tray header { display: flex; justify-content: space-between; align-items: center; }
.selected-problems { display: flex; gap: 8px; flex-wrap: wrap; max-height: 120px; overflow-y: auto; margin-top: 10px; }
.selected-problems .el-tag { max-width: 100%; height: auto; white-space: normal; padding: 6px; overflow-wrap: anywhere; }
@media(max-width: 760px) { .picker-filters { grid-template-columns: 1fr 1fr; } .picker-filters > .el-input { grid-column: 1 / -1; } .picker-row { padding: 9px; gap: 6px; } }
</style>
