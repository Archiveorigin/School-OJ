<template>
  <header class="wizard-header">
    <div class="wizard-context">
      <button type="button" class="back-button" @click="emit('back')">
        <el-icon><ArrowLeft /></el-icon>
        <span>返回考试</span>
      </button>
      <div class="wizard-title">
        <strong>新建考试</strong>
        <span v-if="savedAt">草稿已自动保存 {{ savedAt }}</span>
      </div>
    </div>

    <el-steps
      class="wizard-steps"
      :active="step"
      align-center
      finish-status="success"
      aria-label="创建考试进度"
    >
      <el-step v-for="(item, index) in steps" :key="item">
        <template #title>
          <button
            type="button"
            :disabled="index > step"
            @click="emit('go', index)"
          >
            {{ item }}
          </button>
        </template>
      </el-step>
    </el-steps>

    <div class="wizard-actions">
      <el-button @click="emit('save')">保存草稿</el-button>
      <el-button
        type="primary"
        :loading="saving"
        :disabled="!canContinue"
        @click="emit('continue')"
      >
        {{ step === 2 ? "确认发布" : "下一步" }}
      </el-button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ArrowLeft } from "@element-plus/icons-vue";

defineProps<{
  step: number;
  canContinue: boolean;
  saving: boolean;
  savedAt?: string;
}>();

const emit = defineEmits<{
  back: [];
  save: [];
  continue: [];
  go: [step: number];
}>();

const steps = ["基本信息", "选择题目", "发布确认"];
</script>

<style scoped>
.wizard-header {
  position: sticky;
  top: 0;
  z-index: 3;
  display: grid;
  grid-template-columns: minmax(210px, 0.8fr) minmax(420px, 1.5fr) minmax(
      220px,
      0.8fr
    );
  align-items: center;
  min-height: 82px;
  gap: 22px;
  padding: 12px 28px;
  border-bottom: 1px solid #dce3ee;
  background: rgba(255, 255, 255, 0.96);
}

.wizard-context {
  display: flex;
  align-items: center;
  gap: 18px;
}

.back-button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0;
  border: 0;
  color: #42526b;
  background: transparent;
  cursor: pointer;
}

.wizard-title strong,
.wizard-title span {
  display: block;
}

.wizard-title strong {
  color: #101828;
  font-size: 18px;
}

.wizard-title span {
  margin-top: 5px;
  color: #6b7280;
  font-size: 12px;
}

.wizard-steps {
  min-width: 0;
}

.wizard-steps button {
  padding: 4px 8px;
  border: 0;
  color: #667085;
  background: transparent;
  font-weight: 650;
  cursor: pointer;
}

.wizard-steps button:disabled {
  cursor: not-allowed;
}

.wizard-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.wizard-actions :deep(.el-button) {
  min-width: 110px;
  min-height: 42px;
}

@media (max-width: 1100px) {
  .wizard-header {
    grid-template-columns: 1fr auto;
  }

  .wizard-steps {
    grid-column: 1 / -1;
    grid-row: 2;
  }
}

@media (max-width: 700px) {
  .wizard-header {
    position: relative;
    grid-template-columns: 1fr;
    padding: 14px;
  }

  .wizard-actions {
    justify-content: stretch;
  }

  .wizard-actions :deep(.el-button) {
    flex: 1;
  }

  .wizard-steps button {
    font-size: 12px;
  }
}
</style>
