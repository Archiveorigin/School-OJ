<template>
  <section class="page">
    <div class="page-header">
      <h2>课程班级</h2>
      <div class="toolbar">
        <el-button v-if="canManage" type="primary" @click="openCourseDialog">新建课程</el-button>
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <el-row :gutter="16">
      <el-col :span="14">
        <div class="panel">
          <el-table :data="courses">
            <el-table-column prop="code" label="代码" width="140" />
            <el-table-column prop="name" label="课程" />
            <el-table-column prop="term" label="学期" width="120" />
            <el-table-column label="操作" width="180">
              <template #default="{ row }">
                <el-button size="small" @click="loadClasses(row.id)">班级</el-button>
                <el-button v-if="canManage" size="small" @click="openClassDialog(row.id)">加班级</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="panel">
          <h3>班级</h3>
          <el-table :data="classes" size="small">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="course_id" label="课程" width="90" />
            <el-table-column prop="name" label="名称" />
          </el-table>
        </div>
      </el-col>
    </el-row>
    <el-dialog v-model="courseDialogVisible" title="新建课程" width="560px">
      <el-form :model="courseForm" label-width="90px">
        <el-form-item label="课程代码">
          <el-input v-model="courseForm.code" placeholder="CS101-2026" />
        </el-form-item>
        <el-form-item label="课程名称">
          <el-input v-model="courseForm.name" placeholder="程序设计基础" />
        </el-form-item>
        <el-form-item label="学期">
          <el-input v-model="courseForm.term" placeholder="2026 春" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="courseForm.description" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="courseDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingCourse" @click="submitCourse">创建</el-button>
      </template>
    </el-dialog>
    <el-dialog v-model="classDialogVisible" title="新建班级" width="520px">
      <el-form :model="classForm" label-width="90px">
        <el-form-item label="课程">
          <el-select v-model="classForm.course_id" style="width: 100%">
            <el-option
              v-for="course in courses"
              :key="course.id"
              :label="`${course.code} ${course.name}`"
              :value="course.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="班级名称">
          <el-input v-model="classForm.name" placeholder="计科一班" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="classDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingClass" @click="submitClass">创建</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { client } from '../api/client'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const canManage = computed(() => auth.role === 'admin' || auth.role === 'teacher')
const courses = ref<any[]>([])
const classes = ref<any[]>([])
const courseDialogVisible = ref(false)
const classDialogVisible = ref(false)
const savingCourse = ref(false)
const savingClass = ref(false)
const courseForm = reactive({ code: '', name: '', term: '2026 春', description: '' })
const classForm = reactive({ course_id: 0, name: '' })

async function load() {
  courses.value = (await client.get('/courses')).data
  classes.value = (await client.get('/classes')).data
}

async function loadClasses(courseID: number) {
  classes.value = (await client.get('/classes', { params: { course_id: courseID } })).data
}

function openCourseDialog() {
  courseDialogVisible.value = true
}

function openClassDialog(courseID: number) {
  classForm.course_id = courseID
  classForm.name = ''
  classDialogVisible.value = true
}

async function submitCourse() {
  if (!courseForm.code || !courseForm.name) {
    ElMessage.error('请填写课程代码和课程名称')
    return
  }
  savingCourse.value = true
  try {
    await client.post('/courses', { ...courseForm })
    ElMessage.success('课程已创建')
    courseDialogVisible.value = false
    courseForm.code = ''
    courseForm.name = ''
    courseForm.term = '2026 春'
    courseForm.description = ''
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingCourse.value = false
  }
}

async function submitClass() {
  if (!classForm.course_id || !classForm.name) {
    ElMessage.error('请选择课程并填写班级名称')
    return
  }
  savingClass.value = true
  try {
    await client.post(`/courses/${classForm.course_id}/classes`, { name: classForm.name })
    ElMessage.success('班级已创建')
    classDialogVisible.value = false
    await loadClasses(classForm.course_id)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingClass.value = false
  }
}

onMounted(load)
</script>
