<template>
  <div class="sample-editor">
    <div class="editor-heading">
      <div>
        <strong>前台展示测试点</strong>
        <p class="muted">这里的样例会展示给答题用户，不会替代后台评测数据。</p>
      </div>
      <el-button size="small" @click="addSample">添加样例</el-button>
    </div>

    <el-empty v-if="!modelValue.length" description="暂未设置前台样例" :image-size="54" />
    <div v-for="(sample, index) in modelValue" :key="index" class="sample-editor-card">
      <div class="sample-card-head">
        <el-input
          :model-value="sample.name"
          :placeholder="`样例 ${index + 1} 名称`"
          @update:model-value="updateSample(index, 'name', String($event))"
        />
        <el-button type="danger" text @click="removeSample(index)">删除</el-button>
      </div>
      <div class="sample-fields">
        <div class="sample-field">
          <div class="field-head">
            <strong>输入</strong>
            <div>
              <el-button size="small" text @click="copyText(sample.input)">复制</el-button>
              <el-button size="small" text @click="pasteText(index, 'input')">粘贴</el-button>
            </div>
          </div>
          <el-input
            :model-value="sample.input"
            type="textarea"
            :rows="5"
            placeholder="输入数据"
            @update:model-value="updateSample(index, 'input', String($event))"
          />
        </div>
        <div class="sample-field">
          <div class="field-head">
            <strong>输出</strong>
            <div>
              <el-button size="small" text @click="copyText(sample.output)">复制</el-button>
              <el-button size="small" text @click="pasteText(index, 'output')">粘贴</el-button>
            </div>
          </div>
          <el-input
            :model-value="sample.output"
            type="textarea"
            :rows="5"
            placeholder="期望输出"
            @update:model-value="updateSample(index, 'output', String($event))"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { copyTextToClipboard } from '../features/clipboard'

interface EditableProblemSample {
  name: string
  input: string
  output: string
}

const props = defineProps<{
  modelValue: EditableProblemSample[]
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: EditableProblemSample[]): void
}>()

function addSample() {
  const next = props.modelValue.length + 1
  emit('update:modelValue', [...props.modelValue, { name: `样例 ${next}`, input: '', output: '' }])
}

function removeSample(index: number) {
  emit('update:modelValue', props.modelValue.filter((_, itemIndex) => itemIndex !== index))
}

function updateSample(index: number, field: keyof EditableProblemSample, value: string) {
  emit('update:modelValue', props.modelValue.map((sample, itemIndex) => (
    itemIndex === index ? { ...sample, [field]: value } : sample
  )))
}

async function copyText(value: string) {
  try {
    await copyTextToClipboard(value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败，请手动选择文本')
  }
}

async function pasteText(index: number, field: 'input' | 'output') {
  try {
    updateSample(index, field, await navigator.clipboard.readText())
    ElMessage.success('已粘贴')
  } catch {
    ElMessage.error('无法读取剪贴板，请检查浏览器权限')
  }
}
</script>

<style scoped>
.sample-editor {
  width: 100%;
  display: grid;
  gap: 12px;
}

.editor-heading,
.sample-card-head,
.field-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.editor-heading p {
  margin: 4px 0 0;
  font-size: 12px;
}

.sample-editor-card {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: color-mix(in srgb, var(--surface-strong) 72%, transparent);
}

.sample-card-head .el-input {
  max-width: 360px;
}

.sample-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.sample-field {
  min-width: 0;
}

.field-head {
  margin-bottom: 6px;
}

@media (max-width: 720px) {
  .sample-fields {
    grid-template-columns: 1fr;
  }
}
</style>
