<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>{{ roleTitle }}</h2>
        <p class="muted">{{ activeClassText }}</p>
      </div>
      <el-button :loading="loading || classroom.loading" @click="refresh">刷新</el-button>
    </div>

    <div v-if="loadError" class="panel dashboard-error">
      <strong>概览加载失败</strong>
      <span>{{ loadError }}</span>
      <el-button size="small" @click="refresh">重试</el-button>
    </div>

    <div v-if="showNoClassState" class="panel no-class-state">
      <div>
        <h3>尚未加入班级</h3>
        <p class="muted">加入教师提供的班级后，题库、作业、考试和排行榜会按班级显示。</p>
      </div>
      <div class="join-inline">
        <el-input-number v-model="joinClassID" :min="1" placeholder="班级 ID" />
        <el-button type="primary" :loading="joining" @click="joinClass">加入班级</el-button>
      </div>
    </div>

    <template v-else>
      <el-row :gutter="16">
        <el-col :span="6" v-for="item in stats" :key="item.label">
          <div class="panel stat">
            <strong>{{ item.value }}</strong>
            <span class="muted">{{ item.label }}</span>
          </div>
        </el-col>
      </el-row>
      <el-row :gutter="16" class="dashboard-row">
        <el-col :span="10">
          <div class="panel fortune">
            <div class="section-title">
              <h3>今日运势</h3>
              <span>{{ fortune.badge }}</span>
            </div>
            <strong>{{ fortune.title }}</strong>
            <p>{{ fortune.tip }}</p>
            <div class="fortune-tags">
              <el-tag type="success">宜 {{ fortune.good }}</el-tag>
              <el-tag type="info">幸运语言 {{ fortune.lang }}</el-tag>
            </div>
          </div>
        </el-col>
        <el-col :span="14">
          <div class="panel">
            <div class="section-title">
              <h3>{{ rolePanelTitle }}</h3>
            </div>
            <div class="quick-grid">
              <div v-for="item in roleCards" :key="item.label" class="quick-card">
                <strong>{{ item.value }}</strong>
                <span>{{ item.label }}</span>
              </div>
            </div>
          </div>
        </el-col>
      </el-row>
      <div class="panel latest">
        <h3>最近提交</h3>
        <el-table :data="submissions" size="small" v-loading="loading">
          <el-table-column prop="id" label="ID" width="90" />
          <el-table-column prop="problem_id" label="题目" width="90" />
          <el-table-column prop="language" label="语言" width="110" />
          <el-table-column label="状态">
            <template #default="{ row }"><StatusBadge :status="row.status" /></template>
          </el-table-column>
          <el-table-column prop="score" label="分数" width="90" />
        </el-table>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { client, type Submission } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

const courses = ref<any[]>([])
const problems = ref<any[]>([])
const submissions = ref<Submission[]>([])
const assignments = ref<any[]>([])
const exams = ref<any[]>([])
const loading = ref(false)
const loadError = ref('')
const joinClassID = ref<number>()
const joining = ref(false)
const auth = useAuthStore()
const classroom = useClassroomStore()
const showNoClassState = computed(() => auth.role === 'student' && classroom.loaded && classroom.classes.length === 0)
const stats = computed(() => [
  { label: auth.role === 'student' ? '已加入课程' : '课程', value: courses.value.length },
  { label: '当前班级题目', value: problems.value.length },
  { label: '近期提交', value: submissions.value.length },
  { label: '已通过', value: submissions.value.filter((s) => s.status === 'accepted').length }
])
const roleTitle = computed(() => {
  if (auth.role === 'admin') return '管理员概览'
  if (auth.role === 'teacher') return '教学工作台'
  return '学习工作台'
})
const activeClassText = computed(() => {
  if (classroom.loading && !classroom.loaded) return '正在加载班级'
  const item = classroom.activeClass
  if (item) return `${item.course_code} / ${item.class_name}`
  return auth.role === 'student' ? '尚未加入班级' : '尚未选择班级'
})
const rolePanelTitle = computed(() => {
  if (auth.role === 'admin') return '系统运行'
  if (auth.role === 'teacher') return '教学安排'
  return '我的学习'
})
const roleCards = computed(() => [
  { label: auth.role === 'student' ? '我的班级' : '可管理班级', value: classroom.classes.length },
  { label: '作业', value: assignments.value.length },
  { label: '考试', value: exams.value.length }
])
const fortune = computed(() => {
  const seed = `${new Date().toISOString().slice(0, 10)}-${auth.user?.id || 0}`
  const sum = Array.from(seed).reduce((acc, char) => acc + char.charCodeAt(0), 0)
  const options = [
    { badge: '轻盈', title: '适合补齐一个小缺口', tip: '把最小的待办先合并掉，今天的节奏会更顺。', good: '写测试', lang: 'Go' },
    { badge: '清晰', title: '适合读题和拆解边界', tip: '先确认输入输出和极端数据，再动手会更稳。', good: '整理题面', lang: 'Python' },
    { badge: '高效', title: '适合完成一次提交闭环', tip: '从编译到 AC 的反馈链路会给你明确方向。', good: '调试样例', lang: 'C++' },
    { badge: '专注', title: '适合复盘排行榜变化', tip: '看一眼自己和同学的差距，再定下一题。', good: '复盘错题', lang: 'Java' }
  ]
  return options[sum % options.length]
})

function list<T>(value: T[] | null | undefined) {
  return Array.isArray(value) ? value : []
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const params = classroom.activeClassId ? { class_id: classroom.activeClassId } : {}
    const skipClassScoped = showNoClassState.value
    const [c, p, s, a, e] = await Promise.all([
      client.get('/courses'),
      skipClassScoped ? Promise.resolve({ data: [] }) : client.get('/problems', { params }),
      client.get('/submissions'),
      skipClassScoped ? Promise.resolve({ data: [] }) : client.get('/assignments', { params }),
      skipClassScoped ? Promise.resolve({ data: [] }) : client.get('/exams', { params })
    ])
    courses.value = list(c.data)
    problems.value = list(p.data)
    submissions.value = list(s.data)
    assignments.value = list(a.data)
    exams.value = list(e.data)
  } catch (err: any) {
    loadError.value = err.response?.data?.error || err.message || '未知错误'
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await classroom.load({ force: true })
  await load()
}

async function joinClass() {
  if (!joinClassID.value) {
    ElMessage.error('请填写班级 ID')
    return
  }
  joining.value = true
  try {
    await client.post(`/classes/${joinClassID.value}/join`)
    ElMessage.success('已加入班级')
    classroom.setActive(joinClassID.value)
    await classroom.load({ force: true })
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    joining.value = false
  }
}

watch(
  () => classroom.activeClassId,
  () => {
    if (classroom.loaded) load()
  }
)

onMounted(async () => {
  await classroom.load()
  await load()
})
</script>

<style scoped>
.stat {
  display: grid;
  gap: 8px;
}

.stat strong {
  font-size: 28px;
}

.latest {
  margin-top: 16px;
}

.dashboard-row {
  margin-top: 16px;
}

.dashboard-error {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  color: #b42318;
}

.no-class-state {
  display: grid;
  gap: 18px;
  max-width: 720px;
}

.no-class-state h3 {
  margin: 0 0 8px;
  color: var(--text);
}

.no-class-state p {
  margin: 0;
}

.join-inline {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.fortune strong {
  display: block;
  margin-bottom: 8px;
  font-size: 20px;
}

.fortune p {
  margin: 0 0 12px;
  color: var(--muted);
}

.fortune-tags,
.quick-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.quick-card {
  min-width: 130px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: color-mix(in srgb, var(--surface-strong) 72%, transparent);
}

.quick-card strong,
.quick-card span {
  display: block;
}

.quick-card strong {
  font-size: 24px;
}

.quick-card span {
  color: var(--muted);
}
</style>
