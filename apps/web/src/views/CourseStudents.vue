<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ course?.name || '课程学生' }}</h2>
        <p class="muted">
          {{ course ? `${course.code} · ${course.term || '未设置学期'}` : '加载中...' }}
        </p>
      </div>
      <div class="toolbar">
        <el-button @click="router.push('/courses/list')">返回课程</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>

    <div class="stats-row">
      <div class="stat-card">
        <span class="stat-value">{{ students.length }}</span>
        <span class="stat-label">课程学生</span>
      </div>
      <div class="stat-card">
        <span class="stat-value">{{ classCount }}</span>
        <span class="stat-label">下属班级</span>
      </div>
      <div class="stat-card">
        <span class="stat-value">{{ directStudents }}</span>
        <span class="stat-label">直接加入课程</span>
      </div>
    </div>

    <section class="panel">
      <div class="list-tools">
        <el-input
          v-model="search"
          placeholder="搜索姓名、邮箱或学号"
          clearable
          class="search-input"
        />
        <el-select
          v-if="classOptions.length > 1"
          v-model="classFilter"
          clearable
          placeholder="按班级筛选"
          class="class-filter"
        >
          <el-option
            v-for="option in classOptions"
            :key="option.value"
            :label="option.label"
            :value="option.value"
          />
        </el-select>
      </div>

      <el-table
        :data="filteredStudents"
        v-loading="loading"
        empty-text="暂无学生加入此课程"
      >
        <el-table-column label="学生" min-width="200">
          <template #default="{ row }">
            <div class="student-cell">
              <div class="student-avatar">
                <img v-if="row.avatar_url" :src="row.avatar_url" alt="" />
                <span v-else>{{ (row.name || row.email || '?').trim().slice(0, 1).toUpperCase() }}</span>
              </div>
              <div class="student-info">
                <div class="student-name">{{ row.name }}</div>
                <div class="student-email muted">{{ row.email }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="student_no" label="学号" width="140" />
        <el-table-column label="归属班级" min-width="160">
          <template #default="{ row }">
            <span v-if="row.class_name">{{ row.class_name }}</span>
            <el-tag v-else type="info" size="small" effect="plain">未归属班级</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="加入时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.joined_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              type="danger"
              plain
              :loading="removingId === row.user_id"
              @click="removeStudent(row)"
            >
              移除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="list-footer">
        <span class="muted">
          共 {{ filteredStudents.length }} 名学生
          <template v-if="search || classFilter">
            （已筛选，总计 {{ students.length }} 名）
          </template>
        </span>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../api/client'

interface CourseStudent {
  user_id: number
  name: string
  email: string
  student_no: string
  avatar_url: string
  class_name: string
  joined_at: string
}

const route = useRoute()
const router = useRouter()
const course = ref<any>(null)
const students = ref<CourseStudent[]>([])
const loading = ref(false)
const search = ref('')
const classFilter = ref('')
const removingId = ref<number | null>(null)

const courseId = computed(() => Number(route.params.id))

const classOptions = computed(() => {
  const names = new Set<string>()
  for (const s of students.value) {
    if (s.class_name) {
      for (const name of s.class_name.split(', ')) {
        names.add(name)
      }
    }
  }
  return Array.from(names).sort().map((name) => ({ value: name, label: name }))
})

const classCount = computed(() => classOptions.value.length)
const directStudents = computed(() => students.value.filter((s) => !s.class_name).length)

const filteredStudents = computed(() => {
  let list = students.value
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        s.email.toLowerCase().includes(q) ||
        (s.student_no || '').toLowerCase().includes(q)
    )
  }
  if (classFilter.value) {
    list = list.filter((s) => (s.class_name || '').includes(classFilter.value))
  }
  return list
})

function formatDate(value: string) {
  if (!value) return '-'
  const d = new Date(value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const [courseRes, studentsRes] = await Promise.all([
      client.get('/courses'),
      client.get(`/courses/${courseId.value}/students`)
    ])
    course.value = (courseRes.data as any[]).find((c: any) => c.id === courseId.value) || null
    students.value = Array.isArray(studentsRes.data) ? studentsRes.data : []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

async function removeStudent(row: CourseStudent) {
  try {
    await ElMessageBox.confirm(
      `确定要将 ${row.name || row.email} 从课程中移除吗？该生在此课程下所有班级的归属也会被清除。`,
      '移除学生',
      { type: 'warning' }
    )
  } catch {
    return
  }
  removingId.value = row.user_id
  try {
    await client.delete(`/courses/${courseId.value}/members/${row.user_id}`)
    ElMessage.success('学生已移除')
    students.value = students.value.filter((s) => s.user_id !== row.user_id)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    removingId.value = null
  }
}

onMounted(load)
</script>

<style scoped>
.stats-row {
  display: grid;
  grid-template-columns: repeat(3, minmax(160px, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 18px 16px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.stat-card:hover {
  border-color: var(--accent);
  box-shadow: 0 8px 24px rgba(10, 94, 166, 0.08);
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--accent);
  line-height: 1.2;
}

.stat-label {
  font-size: 13px;
  color: var(--muted);
}

.list-tools {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}

.search-input {
  width: 280px;
}

.class-filter {
  width: 180px;
}

.student-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.student-avatar {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--accent), #14b8a6);
  color: #fff;
  font-weight: 700;
  font-size: 14px;
  overflow: hidden;
  flex-shrink: 0;
}

.student-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.student-info {
  min-width: 0;
}

.student-name {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.student-email {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.list-footer {
  margin-top: 14px;
  text-align: right;
}

@media (max-width: 760px) {
  .stats-row {
    grid-template-columns: 1fr;
  }

  .list-tools {
    flex-direction: column;
    align-items: stretch;
  }

  .search-input,
  .class-filter {
    width: 100%;
  }
}
</style>
