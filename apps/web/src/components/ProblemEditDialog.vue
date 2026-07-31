<template>
  <el-dialog v-model="visible" title="修改题目" width="min(920px, calc(100vw - 28px))" destroy-on-close>
    <el-form label-width="96px" class="problem-edit-form">
      <el-form-item label="题目">
        <div class="problem-meta">
          <strong>{{ problem?.display_code || '未编号' }}</strong>
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
      <el-form-item label="答案比较">
        <el-select v-model="form.checker_type" style="width: 100%">
          <el-option label="标准比较（忽略行尾空白）" value="exact" />
          <el-option label="令牌比较（忽略空白差异）" value="tokens" />
          <el-option label="浮点比较（允许误差）" value="float" />
        </el-select>
      </el-form-item>
      <el-row v-if="form.checker_type === 'float'" :gutter="12">
        <el-col :span="12">
          <el-form-item label="绝对误差">
            <el-input-number v-model="form.absolute_tolerance" :min="0" :max="1" :step="0.000001" :precision="8" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="相对误差">
            <el-input-number v-model="form.relative_tolerance" :min="0" :max="1" :step="0.000001" :precision="8" />
          </el-form-item>
        </el-col>
      </el-row>
      <p class="checker-note">比较器只使用平台内置算法，不运行题目包中的脚本。</p>
      <el-form-item label="标签">
        <ProblemTagSelector v-model="form.tags" />
      </el-form-item>
      <el-form-item label="难度">
        <el-select v-model="form.difficulty" placeholder="请选择难度" style="width: 100%">
          <el-option v-for="item in problemDifficultyOptions" :key="item" :label="item" :value="item" />
        </el-select>
      </el-form-item>
      <el-form-item label="后台测试点">
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
import { problemDifficulty, problemDifficultyOptions, tagList } from '../features/problems/problemMeta'
import ProblemTagSelector from './ProblemTagSelector.vue'

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
  checker_type: 'exact',
  absolute_tolerance: 0.000001,
  relative_tolerance: 0.000001,
  tags: [] as string[],
  difficulty: '入门'
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
    const checker = problemChecker(props.problem.manifest)
    form.checker_type = checker.type
    form.absolute_tolerance = checker.absolute_tolerance
    form.relative_tolerance = checker.relative_tolerance
    form.tags = tagList(props.problem.tags)
    form.difficulty = problemDifficulty(props.problem) || '入门'
    testFiles.value = []
  },
  { immediate: true }
)

function syncTestFiles(_file: any, fileList: any[]) {
  testFiles.value = fileList
}

function problemChecker(manifest?: Record<string, unknown>) {
  const raw = (manifest?.checker || {}) as Record<string, unknown>
  const type = ['exact', 'tokens', 'float'].includes(String(raw.type)) ? String(raw.type) : 'exact'
  return {
    type,
    absolute_tolerance: checkerTolerance(raw.absolute_tolerance),
    relative_tolerance: checkerTolerance(raw.relative_tolerance)
  }
}

function checkerTolerance(value: unknown) {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0.000001
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
        checker: {
          type: form.checker_type,
          absolute_tolerance: form.checker_type === 'float' ? form.absolute_tolerance : 0,
          relative_tolerance: form.checker_type === 'float' ? form.relative_tolerance : 0
        },
        tags: form.tags,
        difficulty: form.difficulty
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

.checker-note {
  margin: -4px 0 12px 96px;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}
</style>
