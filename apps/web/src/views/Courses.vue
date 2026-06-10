<template>
  <section class="page">
    <div class="page-header">
      <div>
        <h2>课程班级</h2>
        <p class="muted">按课程和班级分别查看教学范围。</p>
      </div>
      <div class="toolbar">
        <el-button type="primary" @click="router.push('/courses/list')">查看课程</el-button>
        <el-button @click="router.push('/classes')">查看班级</el-button>
      </div>
    </div>

    <div class="course-class-entry">
      <div class="panel entry-panel">
        <div>
          <h3>{{ canManage ? '全部可管理课程' : '我的课程' }}</h3>
          <p class="muted">{{ canManage ? '查看你创建或可协作管理的课程。' : '查看你已加入班级所属的课程。' }}</p>
        </div>
        <el-button type="primary" @click="router.push('/courses/list')">进入课程列表</el-button>
      </div>

      <div class="panel entry-panel">
        <div>
          <h3>{{ canManage ? '全部可管理班级' : '我的班级' }}</h3>
          <p class="muted">{{ canManage ? '按课程查看班级，并进入班级相关教学内容。' : '查看或切换你已加入的班级。' }}</p>
        </div>
        <el-button type="primary" plain @click="router.push('/classes')">进入班级列表</el-button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
</script>

<style scoped>
.page-header p {
  margin: 6px 0 0;
}

.course-class-entry {
  display: grid;
  grid-template-columns: repeat(2, minmax(260px, 1fr));
  gap: 16px;
}

.entry-panel {
  display: grid;
  gap: 18px;
  min-height: 180px;
  align-content: space-between;
}

.entry-panel h3 {
  margin: 0 0 8px;
  color: var(--text);
}

.entry-panel p {
  margin: 0;
}

@media (max-width: 760px) {
  .course-class-entry {
    grid-template-columns: 1fr;
  }
}
</style>
