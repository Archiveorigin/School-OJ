<template>
  <div class="team-icon-control">
    <button
      type="button"
      class="team-icon-button"
      :class="{ editable }"
      :disabled="!editable || loading"
      :aria-label="editable ? '点击上传团队图标' : '团队图标'"
      @click.stop="inputRef?.click()"
    >
      <img :src="modelValue || '/logo1.png'" alt="" />
      <span v-if="editable" class="upload-hint">{{ loading ? '处理中' : '上传图标' }}</span>
    </button>
    <input ref="inputRef" class="hidden-input" type="file" accept="image/png,image/jpeg,image/webp" @change="selectIcon" />
    <small v-if="showHelp">点击图片上传，支持 PNG / JPG / WebP，最大 2MB；未上传时使用学院图标。</small>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { ref } from 'vue'

withDefaults(defineProps<{ modelValue?: string; editable?: boolean; showHelp?: boolean }>(), {
  modelValue: '',
  editable: true,
  showHelp: false
})
const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()
const inputRef = ref<HTMLInputElement>()
const loading = ref(false)

async function selectIcon(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
    ElMessage.error('请选择 PNG、JPG 或 WebP 图片')
    input.value = ''
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.error('团队图标不能超过 2MB')
    input.value = ''
    return
  }
  loading.value = true
  try {
    const value = await readFile(file)
    emit('update:modelValue', value)
    emit('change', value)
  } finally {
    loading.value = false
    input.value = ''
  }
}

function readFile(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}
</script>

<style scoped>
.team-icon-control { display: grid; gap: 8px; justify-items: start; }
.team-icon-button { position: relative; width: 88px; height: 88px; padding: 0; overflow: hidden; border: 1px solid var(--border); border-radius: 16px; background: #fff; box-shadow: 0 10px 24px rgba(10, 94, 166, .12); }
.team-icon-button.editable { cursor: pointer; }
.team-icon-button:disabled { cursor: default; opacity: 1; }
.team-icon-button img { width: 100%; height: 100%; object-fit: cover; }
.upload-hint { position: absolute; inset: auto 0 0; padding: 6px 3px; color: #fff; background: rgba(15, 23, 42, .78); font-size: 11px; }
.hidden-input { display: none; }
small { max-width: 360px; color: var(--muted); line-height: 1.55; }
</style>
