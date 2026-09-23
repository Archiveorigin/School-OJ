<template>
  <el-dialog
    v-model="visible"
    :title="context.kind === 'problem' ? '题目反馈' : '工单修改'"
    width="min(1160px, calc(100vw - 24px))"
    destroy-on-close
    :close-on-click-modal="false"
    align-center
    ><TicketBoard v-if="visible" :context="context"
  /></el-dialog>
</template>
<script setup lang="ts">
import type { TicketContext } from "../../api/modules/tickets";
import TicketBoard from "./TicketBoard.vue";
const visible = defineModel<boolean>({ default: false });
defineProps<{ context: TicketContext }>();
</script>
