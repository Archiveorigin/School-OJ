<template>
  <section class="course-overview">
    <div class="overview-heading">
      <div>
        <span>COURSE CATALOG</span>
        <h2>课程列表</h2>
        <p>课程概况仅呈现当前账号可访问的课程，班级成员与任课关系保留在数据层供后续扩展。</p>
      </div>
      <el-input v-model="keyword" clearable placeholder="搜索课程号、名称、学期或学院" />
    </div>
    <div class="panel course-list-panel">
      <el-table :data="pagedCourses" v-loading="loading" row-key="id" :row-class-name="rowClassName" @row-click="openCourse">
        <el-table-column prop="code" label="课程号" width="150" />
        <el-table-column prop="name" label="课程名" min-width="220" />
        <el-table-column prop="term" label="学期" width="150" />
        <el-table-column label="所属学院" min-width="190">
          <template #default="{ row }">{{ row.college || '暂未填写' }}</template>
        </el-table-column>
        <el-table-column label="课程说明" min-width="260">
          <template #default="{ row }">{{ row.description || '暂无课程说明' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="right">
          <template #default="{ row }">
            <el-button size="small" :type="row.id === courseID ? 'primary' : 'default'" @click.stop="openCourse(row)">
              {{ row.id === courseID ? '当前' : '进入' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && !filteredCourses.length" description="暂无可访问课程" />
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="filteredCourses.length" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client } from '../../api/client'
import ListPagination from '../../components/ListPagination.vue'

const route = useRoute()
const router = useRouter()
const courses = ref<any[]>([])
const keyword = ref('')
const loading = ref(false)
const page = ref(1)
const pageSize = ref(10)
const courseID = computed(() => Number(route.params.courseId))
const filteredCourses = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return courses.value
  return courses.value.filter((course) =>
    [course.code, course.name, course.term, course.college, course.description]
      .some((value) => String(value || '').toLowerCase().includes(text))
  )
})
const pagedCourses = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredCourses.value.slice(start, start + pageSize.value)
})

async function load() {
  loading.value = true
  try {
    courses.value = (await client.get('/courses')).data || []
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function openCourse(course: any) {
  if (course.id !== courseID.value) router.push(`/my/courses/${course.id}`)
}

function rowClassName({ row }: { row: any }) {
  return row.id === courseID.value ? 'current-course-row' : ''
}

watch([keyword, pageSize], () => { page.value = 1 })
onMounted(load)
</script>

<style scoped>
.course-overview { max-width: 1220px; margin: 0 auto; padding: 34px 28px 58px; }
.overview-heading { display: flex; align-items: end; justify-content: space-between; gap: 24px; margin-bottom: 20px; }
.overview-heading span { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .16em; }
.overview-heading h2 { margin: 8px 0 6px; font-family: 'Noto Serif SC', SimSun, serif; font-size: 29px; }
.overview-heading p { margin: 0; color: var(--muted); line-height: 1.7; }
.overview-heading .el-input { width: min(380px, 100%); flex: 0 0 auto; }
.course-list-panel { padding: 22px; }
.course-list-panel :deep(.el-table__row) { cursor: pointer; }
.course-list-panel :deep(.current-course-row td.el-table__cell) { background: color-mix(in srgb, var(--accent) 9%, var(--surface-strong)); }
@media (max-width: 760px) { .course-overview { padding: 24px 14px 42px; } .overview-heading { align-items: stretch; flex-direction: column; } .overview-heading .el-input { width: 100%; } }
</style>
