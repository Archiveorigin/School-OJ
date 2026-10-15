<template>
  <section class="package-review">
    <el-alert type="info" :closable="false" :title="ticket.action === 'create' ? '从工单题面、测试点和图片生成题包，核对后即可入库。' : '以当前题目为基础修改；未选择替换的测试数据和图片会保留。'" />
    <el-alert v-if="ticket.review_error" type="error" :closable="false" :title="ticket.review_error" />
    <template v-if="form">
      <el-form-item label="题目标题"><el-input v-model="form.title" maxlength="200" /></el-form-item>
      <div class="review-columns">
        <div>
          <ProblemStatementEditor v-if="ticket.action === 'create'" ref="statementEditor" v-model="form.statement" />
          <el-form-item v-else label="修改题面（Markdown）"><el-input v-model="form.statement" type="textarea" :rows="16" /></el-form-item>
        </div>
        <ProblemDraftPreview :title="form.title" :source="preview?.statement || form.statement" :asset-urls="preview?.asset_urls || imageUrls" />
      </div>
      <div class="judge-settings">
        <el-form-item label="时间限制（ms）"><el-input-number v-model="form.time_limit_ms" :min="1" :max="60000" /></el-form-item>
        <el-form-item label="内存限制（MiB）"><el-input-number v-model="form.memory_limit_mb" :min="1" :max="4096" /></el-form-item>
        <el-form-item label="输出限制（KiB）"><el-input-number v-model="form.output_limit_kb" :min="1" :max="65536" /></el-form-item>
        <el-form-item label="答案比较方式"><el-select v-model="form.checker.type"><el-option value="exact" label="精确匹配" /><el-option value="tokens" label="忽略空白差异" /><el-option value="float" label="浮点误差" /></el-select></el-form-item>
        <template v-if="form.checker.type === 'float'">
          <el-form-item label="绝对误差"><el-input-number v-model="form.checker.absolute_tolerance" :min="0" :max="1" :step="0.000001" /></el-form-item>
          <el-form-item label="相对误差"><el-input-number v-model="form.checker.relative_tolerance" :min="0" :max="1" :step="0.000001" /></el-form-item>
        </template>
      </div>
      <el-checkbox v-if="ticket.problem_id" v-model="form.replace_tests">用所选附件替换全部测试点</el-checkbox>
      <p class="muted">勾选用于入库的附件。测试文件按同路径、同名 .in / .out 配对；新测试点按等权计分。图片单个最多 5 MiB、合计 20 MiB，其余文件可保留为参考附件。</p>
      <div class="review-attachments">
        <el-checkbox-group v-model="form.attachment_ids">
          <el-checkbox v-for="file in ticket.attachments" :key="file.id" :value="file.id">{{ file.name }}</el-checkbox>
        </el-checkbox-group>
        <p v-if="!ticket.attachments?.length" class="muted">尚无附件，可退回作者补充测试点和图片。</p>
      </div>
      <div class="image-actions"><el-button v-for="file in selectedImages" :key="file.id" link @click="insertImage(file.name)">插入图片：{{ file.name }}</el-button></div>
      <el-button :loading="checking" :disabled="disabled" @click="check">校验材料并预览</el-button>
      <el-alert v-if="error" type="error" :closable="false" :title="error" />
      <el-alert v-if="preview" type="success" :closable="false" :title="`校验通过：${preview.case_count} 组测试点、${preview.asset_count} 张图片。可以执行入库。`" />
      <el-collapse v-if="preview"><el-collapse-item title="查看测试点清单" name="cases"><div class="case-names">{{ preview.case_names.join('、') }}</div></el-collapse-item></el-collapse>
    </template>
  </section>
</template>
<script setup lang="ts">
import { apiErrorMessage } from '../../api/errors'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { generatedApi, type components, type ProblemChangeTicket } from '../../api/client'
import ProblemStatementEditor from './ProblemStatementEditor.vue'
import ProblemDraftPreview from './ProblemDraftPreview.vue'
type Review = components['schemas']['TicketPackageReview']
type Preview = components['schemas']['TicketPackagePreview']
const props = defineProps<{ ticket: ProblemChangeTicket; imageUrls: Record<string, string>; disabled?: boolean }>()
const form = ref<Review | undefined>(props.ticket.review ? JSON.parse(JSON.stringify(props.ticket.review)) : undefined)
const preview = ref<Preview>(), error = ref(''), checking = ref(false)
const statementEditor = ref<InstanceType<typeof ProblemStatementEditor>>()
let controller: AbortController | undefined, validated = ''
const selectedImages = computed(() => (props.ticket.attachments || []).filter(f => form.value?.attachment_ids.includes(f.id) && f.content_type.startsWith('image/')))
watch(form, () => { preview.value = undefined; validated = ''; error.value = ''; controller?.abort(); checking.value = false }, { deep: true, flush: 'sync' })
function insertImage(name: string) {
  const text = `\n\n![题面图片](attachments/${encodeURIComponent(name)})\n`
  if (statementEditor.value) statementEditor.value.insertImage(text)
  else if (form.value) form.value.statement += text
}
async function check() {
  if (!form.value || checking.value) return
  const validation = statementEditor.value?.validate()
  if (validation) { error.value = validation; return }
  const key = JSON.stringify(form.value), current = new AbortController(); controller?.abort(); controller = current
  checking.value = true; error.value = ''; preview.value = undefined
  try {
    const { data } = await generatedApi.previewProblemChangeTicket({ path: { id: props.ticket.id }, body: JSON.parse(key), signal: current.signal, timeout: 600000 })
    if (!current.signal.aborted && key === JSON.stringify(form.value)) { preview.value = data; validated = key }
  } catch (e: any) { if (!current.signal.aborted) error.value = apiErrorMessage(e, '材料校验失败，请重试') }
  finally { if (controller === current) checking.value = false }
}
function appendTo(data: FormData) {
  if (!form.value || !preview.value || validated !== JSON.stringify(form.value)) { ElMessage.warning('请先校验材料并预览，修改后需重新校验'); return false }
  data.append('review', validated); return true
}
onBeforeUnmount(() => controller?.abort())
defineExpose({ appendTo })
</script>
<style scoped>
.package-review { display: grid; gap: 16px; min-width: 0; }
.review-columns { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 18px; }
.review-columns > div { min-width: 0; }
.judge-settings { display: flex; flex-wrap: wrap; gap: 16px; }
.judge-settings .el-select { width: 180px; }
.review-attachments { max-height: 260px; overflow: auto; padding: 12px; border: 1px solid var(--border); border-radius: 8px; }
.review-attachments .el-checkbox-group { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(240px, 100%), 1fr)); gap: 6px; }
.review-attachments :deep(.el-checkbox__label) { white-space: normal; overflow-wrap: anywhere; }
.review-attachments .el-checkbox { height: auto; min-height: 32px; margin: 0; }
.muted { margin: 0; color: var(--muted); line-height: 1.7; }
.case-names { max-height: 160px; overflow: auto; overflow-wrap: anywhere; }
.image-actions { display: flex; flex-wrap: wrap; gap: 8px; }
@media(max-width: 820px) { .review-columns { grid-template-columns: minmax(0, 1fr); } }
</style>
