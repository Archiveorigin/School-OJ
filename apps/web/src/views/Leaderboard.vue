<template>
  <section class="page">
    <div class="page-header">
      <h2>排行榜</h2>
      <div class="toolbar">
        <el-select v-model="selectedClassID" clearable style="width: 240px" @change="load">
          <el-option
            v-for="item in classroom.classes"
            :key="item.class_id"
            :label="`${item.course_code} / ${item.class_name}`"
            :value="item.class_id"
          />
        </el-select>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <div v-if="selectedClassID" class="panel">
      <el-table :data="rows" :row-class-name="rowClass">
        <el-table-column prop="rank" label="#" width="70" />
        <el-table-column prop="name" label="学生" />
        <el-table-column prop="solved" label="通过题数" width="120" />
        <el-table-column prop="score" label="积分" width="100" />
        <el-table-column label="最后提交">
          <template #default="{ row }">{{ formatTime(row.last_submission) }}</template>
        </el-table-column>
      </el-table>
    </div>
    <div v-else class="leaderboard-groups">
      <div v-for="group in groups" :key="group.class_id" class="panel">
        <div class="section-title">
          <h3>{{ group.course_code }} / {{ group.class_name }}</h3>
          <span class="muted">{{ group.course_name }}</span>
        </div>
        <el-table :data="group.rows" :row-class-name="rowClass">
          <el-table-column prop="rank" label="#" width="70" />
          <el-table-column prop="name" label="学生" />
          <el-table-column prop="solved" label="通过题数" width="120" />
          <el-table-column prop="score" label="积分" width="100" />
        </el-table>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { client } from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

const rows = ref<any[]>([])
const groups = ref<any[]>([])
const auth = useAuthStore()
const classroom = useClassroomStore()
const selectedClassID = ref<number>()

async function load() {
  const classID = selectedClassID.value || (auth.role === 'student' ? classroom.activeClassId : undefined)
  if (classID) {
    rows.value = (await client.get('/leaderboard', { params: { class_id: classID } })).data
    groups.value = []
  } else {
    groups.value = (await client.get('/leaderboard')).data
    rows.value = []
  }
}

function rowClass({ row }: any) {
  return row.user_id === auth.user?.id ? 'self-row' : ''
}

function formatTime(value?: string) {
  if (!value) return '暂无'
  return new Date(value).toLocaleString()
}

watch(
  () => classroom.activeClassId,
  (id) => {
    selectedClassID.value = id || undefined
    load()
  }
)

onMounted(async () => {
  await classroom.load()
  selectedClassID.value = classroom.activeClassId || undefined
  await load()
})
</script>

<style scoped>
.leaderboard-groups {
  display: grid;
  gap: 16px;
}

:deep(.self-row) {
  --el-table-tr-bg-color: color-mix(in srgb, var(--accent) 10%, transparent);
}
</style>
