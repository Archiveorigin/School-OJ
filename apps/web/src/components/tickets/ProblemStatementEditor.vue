<template>
  <div class="statement-fields">
    <p class="template-hint">
      按固定栏目填写，系统统一生成标题。正文支持
      Markdown、公式及三级以下子标题；样例保留原始文本。
    </p>
    <el-alert
      v-if="legacy"
      title="原题面已完整保留在题目描述中，请整理到对应栏目后保存。"
      type="info"
      :closable="false"
    />
    <template v-for="section in statementSections" :key="section.key">
      <div v-if="section.key === 'hints'" class="sample-fields">
        <h4>样例</h4>
        <div
          v-for="(sample, index) in draft.samples"
          :key="index"
          class="sample-card"
        >
          <div class="sample-heading">
            <strong>样例 {{ index + 1 }}</strong
            ><el-button
              v-if="draft.samples.length > 1"
              link
              type="danger"
              @click="draft.samples.splice(index, 1)"
              >移除样例 {{ index + 1 }}</el-button
            >
          </div>
          <el-form-item :label="`样例 ${index + 1} 输入`"
            ><el-input
              v-model="sample.input"
              type="textarea"
              :rows="3"
              :aria-label="`样例 ${index + 1} 输入`"
              placeholder="原样填写样例输入；无输入时可留空"
          /></el-form-item>
          <el-form-item :label="`样例 ${index + 1} 输出`"
            ><el-input
              v-model="sample.output"
              type="textarea"
              :rows="3"
              :aria-label="`样例 ${index + 1} 输出`"
              placeholder="原样填写样例输出"
          /></el-form-item>
          <el-form-item :label="`样例 ${index + 1} 说明（可选）`"
            ><el-input
              v-model="sample.explanation"
              type="textarea"
              :rows="2"
              :aria-label="`样例 ${index + 1} 说明`"
              @focus="activeSection = index"
          /></el-form-item>
        </div>
        <el-button @click="draft.samples.push(emptySample())"
          >添加样例</el-button
        >
      </div>
      <el-form-item :label="section.title" :required="section.required"
        ><el-input
          v-model="draft[section.key]"
          type="textarea"
          :rows="section.key === 'description' ? 6 : 3"
          :aria-label="section.title"
          :maxlength="200000"
          :placeholder="`填写${section.title}${section.required ? '' : '（可选）'}`"
          @focus="activeSection = section.key"
      /></el-form-item>
    </template>
  </div>
</template>
<script setup lang="ts">
import { reactive, ref, watch } from "vue";
import {
  emptySample,
  parseStatement,
  serializeStatement,
  statementSections,
  validateStatement,
  type StatementSection,
} from "../../features/problems/statementTemplate";
const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ "update:modelValue": [string] }>();
const parsed = parseStatement(props.modelValue),
  draft = reactive(parsed.draft),
  legacy = ref(parsed.legacy);
const activeSection = ref<StatementSection | number>("description");
watch(
  () => serializeStatement(draft),
  (value) => emit("update:modelValue", value),
  { immediate: true },
);
watch(
  () => props.modelValue,
  (value) => {
    if (value === serializeStatement(draft)) return;
    const next = parseStatement(value);
    Object.assign(draft, next.draft);
    legacy.value = next.legacy;
  },
);
function insertImage(markdown: string) {
  const target = activeSection.value;
  if (typeof target === "number" && draft.samples[target])
    draft.samples[target].explanation += markdown;
  else draft[typeof target === "string" ? target : "description"] += markdown;
}
defineExpose({ validate: () => validateStatement(draft), insertImage });
</script>
<style scoped>
.statement-fields {
  min-width: 0;
}
.template-hint {
  margin: 0 0 14px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.7;
}
.sample-fields {
  margin-bottom: 22px;
}
.sample-fields h4 {
  margin: 0 0 12px;
}
.sample-card {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 14px;
  margin-bottom: 12px;
}
.sample-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 14px;
}
.el-alert {
  margin-bottom: 16px;
}
</style>
