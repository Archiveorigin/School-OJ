<template>
  <div v-if="total > 0" class="list-pagination">
    <el-pagination
      :current-page="page"
      :page-size="pageSize"
      :page-sizes="pageSizes"
      :total="total"
      background
      layout="total, sizes, prev, pager, next, jumper"
      @update:current-page="emit('update:page', $event)"
      @update:page-size="emit('update:pageSize', $event)"
    />
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    page: number
    pageSize: number
    total: number
    pageSizes?: number[]
  }>(),
  {
    pageSizes: () => [10, 20, 50, 100]
  }
)

const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()
</script>

<style scoped>
.list-pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 14px;
}

@media (max-width: 760px) {
  .list-pagination {
    justify-content: flex-start;
    overflow-x: auto;
  }
}
</style>
