<template>
  <section class="page personal-tickets">
    <TicketBoard
      :key="route.fullPath"
      :context="context"
      :initial-new="route.path.endsWith('/new')"
      :new-back-path="
        route.query.from === 'catalog' ? '/problems' : '/problem-changes'
      "
      created-path="/problem-changes"
      title="我的工单"
    />
  </section>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import TicketBoard from "../../components/tickets/TicketBoard.vue";
import type { TicketContext } from "../../api/modules/tickets";
const route = useRoute();
const context = computed<TicketContext>(() =>
  Number(route.query.problem_id)
    ? { kind: "problem", id: Number(route.query.problem_id) }
    : {
        kind: "mine",
        targetScope: String(route.query.target_scope || "public"),
        teamProblemSetID: Number(route.query.team_problem_set_id) || undefined,
      },
);
</script>
<style scoped>
.personal-tickets {
  max-width: 1280px;
  margin: 0 auto;
  padding: 28px 20px;
}
@media (max-width: 720px) {
  .personal-tickets {
    padding: 18px 12px;
  }
}
</style>
