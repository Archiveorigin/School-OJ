<template>
  <section class="admin-ticket-page">
    <AdminPageHeader title="工单管理" eyebrow="FEEDBACK" heading-tag="h1" />
    <div class="panel ticket-panel">
      <TicketBoard :context="{ kind: 'manage' }" title="工单列表" />
    </div>
  </section>
</template>
<script setup lang="ts">
import AdminPageHeader from "../../components/admin/AdminPageHeader.vue";
import TicketBoard from "../../components/tickets/TicketBoard.vue";
</script>
<style scoped>
.admin-ticket-page {
  padding: 28px;
}
.ticket-panel {
  padding: 20px;
  overflow: hidden;
}
@media (max-width: 720px) {
  .admin-ticket-page {
    padding: 18px 12px;
  }
  .ticket-panel {
    padding: 12px;
  }
}
</style>
