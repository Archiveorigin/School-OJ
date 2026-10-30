<template><SubmissionStatusMark :status="status" compact /></template>
<script setup lang="ts">
import SubmissionStatusMark from "./SubmissionStatusMark.vue";
defineProps<{ status: string }>();
</script>
