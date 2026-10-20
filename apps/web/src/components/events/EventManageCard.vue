<template>
  <section class="manage-card panel">
    <div>
      <h3>本场试卷与附件</h3>
      <p>在独立工作区添加题目、预览题面和上传本场附件，完成后统一提交审核。</p>
    </div>
    <router-link :to="`/event-papers/${kind}/${eventId}/edit`"
      ><el-button type="primary">{{
        editable ? "组题" : "查看试卷与审核"
      }}</el-button></router-link
    >
  </section>
</template>
<script setup lang="ts">
import type { EventKind } from "../../api/modules/events";
defineProps<{ kind: EventKind; eventId: number; editable: boolean }>();
</script>
<style scoped>
.manage-card {
  padding: 22px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
}
.manage-card h3 {
  margin: 0 0 8px;
  font-size: 16px;
}
.manage-card p {
  margin: 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.65;
}
@media (max-width: 600px) {
  .manage-card {
    align-items: flex-start;
    flex-direction: column;
    padding: 18px;
  }
}
</style>
