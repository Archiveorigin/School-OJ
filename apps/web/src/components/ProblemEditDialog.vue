<template>
  <el-dialog v-model="visible" title="修改题目" width="920px" destroy-on-close>
    <el-form label-width="96px" class="problem-edit-form">
      <el-form-item label="题目">
        <div class="problem-meta">
          <strong>{{ problem?.display_code || `#${problem?.id || ''}` }}</strong>
          <span class="muted">{{ problem?.slug }}</span>
        </div>
      </el-form-item>
      <el-form-item label="标题">
        <el-input v-model="form.title" maxlength="200" />
      </el-form-item>
      <el-form-item label="题面">
        <el-input
          v-model="form.statement"
          type="textarea"
          :rows="12"
          placeholder="支持 Markdown、LaTeX 和已存在的题面图片引用"
        />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="8">
          <el-form-item label="时间限制">
            <el-input-number v-model="form.time_limit_ms" :min="100" :step="100" />
            <span class="unit">ms</span>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="内存限制">
            <el-input-number v-model="form.memory_limit_mb" :min="16" :step="16" />
            <span class="unit">MB</span>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="输出限制">
            <el-input-number v-model="form.output_limit_kb" :min="1" :step="64" />
            <span class="unit">KB</span>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="标签">
        <el-input v-model="form.tags" placeholder="多个标签用逗号、空格或换行分隔" />
      </el-form-item>
      <el-form-item label="隐藏测试点">
        <div class="test-upload-panel">
          <el-upload
            drag
            action="#"
            multiple
            accept=".zip,.in,.out"
            :auto-upload="false"
            :file-list="testFiles"
            :on-change="syncTestFiles"
            :on-remove="syncTestFiles"
          >
            <div class="upload-text">选择或拖入新的 .zip / .in / .out 测试点文件</div>
            <div class="muted">不选择文件时仅更新题面与限制；选择后会整体替换该题隐藏测试点。</div>
          </el-upload>
          <el-alert
            v-if="testFiles.length"
            type="warning"
            show-icon
            :closable="false"
            title="保存后历史提交不会自动重判，需要时可手动重判相关提交。"
          />
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存修改</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { client, type Problem } from '../api/client'
import { tagList } from '../features/problems/problemMeta'

const props = defineProps<{
  modelValue: boolean
  problem: Problem | null
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'saved', problem: Problem): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})
const saving = ref(false)
const testFiles = ref<any[]>([])
const form = reactive({
  title: '',
  statement: '',
  time_limit_ms: 1000,
  memory_limit_mb: 256,
  output_limit_kb: 1024,
  tags: ''
})

watch(
  () => [props.modelValue, props.problem?.id],
  () => {
    if (!props.modelValue || !props.problem) return
    form.title = props.problem.title || ''
    form.statement = props.problem.statement || ''
    form.time_limit_ms = props.problem.time_limit_ms || 1000
    form.memory_limit_mb = props.problem.memory_limit_mb || 256
    form.output_limit_kb = props.problem.output_limit_kb || 1024
    form.tags = tagList(props.problem.tags).join(', ')
    testFiles.value = []
  },
  { immediate: true }
)

function syncTestFiles(_file: any, fileList: any[]) {
  testFiles.value = fileList
}

function parseTags(value: string) {
  return value
    .split(/[\s,，、]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

async function save() {
  if (!props.problem) return
  if (!form.title.trim()) {
    ElMessage.error('请输入题目标题')
    return
  }
  saving.value = true
  try {
    const fd = new FormData()
    fd.append(
      'draft',
      JSON.stringify({
        title: form.title,
        statement: form.statement,
        time_limit_ms: form.time_limit_ms,
        memory_limit_mb: form.memory_limit_mb,
        output_limit_kb: form.output_limit_kb,
        tags: parseTags(form.tags)
      })
    )
    for (const item of testFiles.value) {
      if (item.raw) fd.append('test_files', item.raw)
    }
    const { data } = await client.put(`/problems/${props.problem.id}`, fd)
    emit('saved', data)
    visible.value = false
    ElMessage.success(testFiles.value.length ? '题目与测试点已更新' : '题目已更新')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.problem-edit-form {
  max-height: 72vh;
  overflow-y: auto;
  padding-right: 6px;
}

.problem-meta {
  display: flex;
  gap: 10px;
  align-items: center;
}

.test-upload-panel {
  width: 100%;
  display: grid;
  gap: 10px;
}

.unit {
  margin-left: 8px;
  color: var(--text-muted);
}
</style>
