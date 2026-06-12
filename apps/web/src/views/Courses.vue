<template>
  <section class="page sub-page">
    <div class="sub-hero">
      <div class="sub-hero-inner">
        <div class="sub-hero-text">
          <h1 class="sub-hero-title">课程班级</h1>
          <p class="sub-hero-sub">按课程和班级分别查看教学范围</p>
        </div>
        <div class="sub-hero-stats">
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ courseCount }}</span>
            <span class="sub-hero-stat-label">课程</span>
          </div>
          <div class="sub-hero-stat">
            <span class="sub-hero-stat-val">{{ classCount }}</span>
            <span class="sub-hero-stat-label">班级</span>
          </div>
        </div>
      </div>
    </div>

    <div class="sub-content">
      <div class="course-class-entry">
        <div class="panel entry-panel">
          <div class="entry-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="28" height="28"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
          </div>
          <div>
            <h3>{{ canManage ? '全部可管理课程' : '我的课程' }}</h3>
            <p class="muted">{{ canManage ? '查看你创建或可协作管理的课程。' : '查看你已加入班级所属的课程。' }}</p>
          </div>
          <el-button type="primary" @click="router.push('/courses/list')">进入课程列表</el-button>
        </div>

        <div class="panel entry-panel">
          <div class="entry-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" width="28" height="28"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          </div>
          <div>
            <h3>{{ canManage ? '全部可管理班级' : '我的班级' }}</h3>
            <p class="muted">{{ canManage ? '按课程查看班级，并进入班级相关教学内容。' : '查看或切换你已加入的班级。' }}</p>
          </div>
          <el-button type="primary" plain @click="router.push('/classes')">进入班级列表</el-button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client } from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useClassroomStore } from '../stores/classroom'

const auth = useAuthStore()
const classroom = useClassroomStore()
const router = useRouter()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const courseCount = ref(0)
const classCount = ref(0)

onMounted(async () => {
  try {
    const [courseRes] = await Promise.all([
      client.get('/courses'),
      classroom.load()
    ])
    courseCount.value = Array.isArray(courseRes.data) ? courseRes.data.length : 0
    classCount.value = classroom.classes.length
  } catch {
    // stats silently fail
  }
})
</script>

<style scoped>
.sub-page {
  padding: 0;
  overflow-x: hidden;
}

.sub-hero {
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0a5ea6 100%);
}

.sub-hero-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 36px 40px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.sub-hero-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sub-hero-title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  color: #f8fafc;
}

.sub-hero-sub {
  margin: 0;
  font-size: 14px;
  color: rgba(248, 250, 252, 0.6);
}

.sub-hero-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.sub-hero-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 20px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 10px;
  min-width: 80px;
  text-align: center;
  transition: background 0.2s;
}

.sub-hero-stat:hover {
  background: rgba(255, 255, 255, 0.18);
}

.sub-hero-stat-val {
  font-size: 22px;
  font-weight: 700;
  color: #fff;
}

.sub-hero-stat-label {
  font-size: 12px;
  color: rgba(248, 250, 252, 0.55);
}

.sub-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px 20px 32px;
}

.course-class-entry {
  display: grid;
  grid-template-columns: repeat(2, minmax(260px, 1fr));
  gap: 16px;
}

.entry-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.entry-icon {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 12px;
  background: linear-gradient(135deg, color-mix(in srgb, var(--accent) 18%, transparent), color-mix(in srgb, #14b8a6 18%, transparent));
  color: var(--accent);
}

.entry-panel h3 {
  margin: 0 0 6px;
  color: var(--text);
}

.entry-panel p {
  margin: 0;
}

@media (max-width: 760px) {
  .sub-hero-inner {
    padding: 24px 20px 32px;
    gap: 16px;
  }

  .course-class-entry {
    grid-template-columns: 1fr;
  }
}
</style>
