<template>
  <main class="authoring-page">
    <router-link class="back" :to="back">← 返回场次说明</router-link
    ><EventReviewPanel
      v-if="valid"
      :key="`${kind}-${id}`"
      :kind="kind"
      :event-id="id"
    /><el-result v-else title="场次地址无效" />
  </main>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import EventReviewPanel from "../../components/events/EventReviewPanel.vue";
import type { EventKind } from "../../api/modules/events";
const route = useRoute(),
  kind = computed(() => route.params.kind as EventKind),
  id = computed(() => Number(route.params.eventId)),
  valid = computed(
    () =>
      ["exam", "contest"].includes(kind.value) &&
      Number.isInteger(id.value) &&
      id.value > 0,
  ),
  back = computed(() =>
    kind.value === "exam"
      ? `/exams/${id.value}/description`
      : `/contest/${id.value}/description`,
  );
</script>
<style scoped>
.authoring-page {
  max-width: 1680px;
  margin: auto;
  padding: 28px;
  display: grid;
  gap: 22px;
  min-width: 0;
}
.back {
  color: var(--muted);
  font-size: 13px;
  width: fit-content;
}
.back:hover {
  color: var(--accent);
}
@media (max-width: 600px) {
  .authoring-page {
    padding: 16px;
    gap: 16px;
  }
}
</style>
