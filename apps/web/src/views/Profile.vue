<template>
  <section class="page profile-page">
    <div class="profile-hero panel">
      <div class="profile-identity">
        <button class="profile-avatar" type="button" @click="avatarInput?.click()">
          <img v-if="profile?.user.avatar_url" :src="profile.user.avatar_url" alt="" />
          <span v-else>{{ initials }}</span>
        </button>
        <input ref="avatarInput" class="hidden-input" type="file" accept="image/*" @change="uploadAvatar" />
        <div>
          <h2>{{ profile?.user.name || auth.user?.name }}</h2>
          <p>{{ profile?.user.email || auth.user?.email }}</p>
          <div class="profile-tags">
            <el-tag>{{ roleLabel }}</el-tag>
            <el-tag v-if="profile?.user.email_verified" type="success">邮箱已验证</el-tag>
            <el-tag v-if="profile?.user.student_no" type="info">学号 {{ profile.user.student_no }}</el-tag>
          </div>
        </div>
      </div>
      <div class="profile-stats">
        <div>
          <strong>{{ profile?.submissions || 0 }}</strong>
          <span>提交</span>
        </div>
        <div>
          <strong>{{ profile?.solved || 0 }}</strong>
          <span>通过题目</span>
        </div>
        <div>
          <strong>{{ statusTotal }}</strong>
          <span>评测记录</span>
        </div>
      </div>
    </div>

    <div class="profile-grid">
      <div class="panel">
        <div class="section-title">
          <h3>代码提交活跃度</h3>
          <span class="muted">近 365 天</span>
        </div>
        <div class="activity-scroll">
          <div class="activity-grid">
            <span
              v-for="day in days"
              :key="day.date"
              class="activity-cell"
              :class="levelClass(day.count)"
              :title="`${day.date}: ${day.count} 次提交`"
            ></span>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="section-title">
          <h3>个人信息</h3>
        </div>
        <el-form label-width="86px" :model="profileForm">
          <el-form-item label="姓名">
            <el-input v-model="profileForm.name" />
          </el-form-item>
          <el-button type="primary" :loading="savingProfile" @click="saveProfile">保存信息</el-button>
        </el-form>
      </div>

      <div class="panel">
        <div class="section-title">
          <h3>邮箱换绑</h3>
        </div>
        <el-form label-width="86px" :model="emailForm">
          <el-form-item label="新邮箱">
            <el-input v-model="emailForm.email">
              <template #append>
                <el-button :loading="sendingCode" @click="sendEmailCode">发送验证码</el-button>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item label="验证码">
            <el-input v-model="emailForm.code" maxlength="6" />
          </el-form-item>
          <el-button type="primary" :loading="bindingEmail" @click="bindEmail">换绑邮箱</el-button>
        </el-form>
      </div>

      <div class="panel">
        <div class="section-title">
          <h3>近期提交</h3>
        </div>
        <el-table :data="profile?.recent || []" size="small">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="language" label="语言" width="90" />
          <el-table-column prop="status" label="状态" width="130" />
          <el-table-column prop="score" label="分数" width="80" />
          <el-table-column label="时间" width="130">
            <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel">
        <div class="section-title">
          <h3>提交反馈</h3>
        </div>
        <el-input v-model="feedback" type="textarea" :rows="5" placeholder="告诉我们你遇到的问题或改进建议" />
        <el-button class="panel-action" type="primary" :loading="sendingFeedback" @click="submitFeedback">提交反馈</el-button>
      </div>

      <div class="panel danger-panel">
        <div class="section-title">
          <h3>账号注销</h3>
        </div>
        <p class="muted">注销后当前账号将无法继续登录，个人邮箱会从账号中解绑。</p>
        <el-button type="danger" plain @click="deleteAccount">注销账号</el-button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client, type Submission, type User } from '../api/client'
import { useAuthStore } from '../stores/auth'

interface ProfileData {
  user: User
  solved: number
  submissions: number
  by_status: Array<{ status: string; count: number }>
  activity: Array<{ date: string; count: number }>
  recent: Submission[]
}

const auth = useAuthStore()
const router = useRouter()
const profile = ref<ProfileData | null>(null)
const avatarInput = ref<HTMLInputElement>()
const savingProfile = ref(false)
const sendingCode = ref(false)
const bindingEmail = ref(false)
const sendingFeedback = ref(false)
const feedback = ref('')

const profileForm = reactive({ name: '' })
const emailForm = reactive({ email: '', code: '' })

const initials = computed(() => (profile.value?.user.name || auth.user?.name || 'U').slice(0, 1).toUpperCase())
const roleLabel = computed(() => {
  const role = profile.value?.user.role || auth.user?.role
  return role === 'admin' ? '管理员' : role === 'teacher' ? '教师' : '学生'
})
const statusTotal = computed(() => (profile.value?.by_status || []).reduce((sum, item) => sum + item.count, 0))
const days = computed(() => {
  const map = new Map((profile.value?.activity || []).map((item) => [item.date, item.count]))
  const today = new Date()
  const result: Array<{ date: string; count: number }> = []
  for (let i = 364; i >= 0; i -= 1) {
    const date = new Date(today)
    date.setDate(today.getDate() - i)
    const key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
    result.push({ date: key, count: map.get(key) || 0 })
  }
  return result
})

async function load() {
  const { data } = await client.get('/profile')
  profile.value = data
  profileForm.name = data.user.name
  auth.updateUser(data.user)
}

async function saveProfile() {
  savingProfile.value = true
  try {
    const { data } = await client.put('/profile', { name: profileForm.name })
    auth.updateUser(data)
    await load()
    ElMessage.success('个人信息已保存')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    savingProfile.value = false
  }
}

async function uploadAvatar(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.error('头像文件不能超过 2MB')
    return
  }
  const avatar = await readFile(file)
  try {
    const { data } = await client.put('/profile', { avatar_url: avatar })
    auth.updateUser(data)
    await load()
    ElMessage.success('头像已更新')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function sendEmailCode() {
  if (!emailForm.email) {
    ElMessage.error('请填写新邮箱')
    return
  }
  sendingCode.value = true
  try {
    await client.post('/profile/email-code', { email: emailForm.email })
    ElMessage.success('验证码已发送；本地 Docker 环境请打开 http://localhost:8025 查看邮件')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    sendingCode.value = false
  }
}

async function bindEmail() {
  if (!emailForm.email || !emailForm.code) {
    ElMessage.error('请填写新邮箱和验证码')
    return
  }
  bindingEmail.value = true
  try {
    const { data } = await client.post('/profile/email', { ...emailForm })
    auth.updateUser(data)
    emailForm.email = ''
    emailForm.code = ''
    await load()
    ElMessage.success('邮箱已换绑')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    bindingEmail.value = false
  }
}

async function submitFeedback() {
  if (!feedback.value.trim()) {
    ElMessage.error('请填写反馈内容')
    return
  }
  sendingFeedback.value = true
  try {
    await client.post('/feedback', { message: feedback.value })
    feedback.value = ''
    ElMessage.success('反馈已提交')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    sendingFeedback.value = false
  }
}

async function deleteAccount() {
  try {
    await ElMessageBox.confirm('注销后账号将无法继续登录，确认继续？', '注销账号', { type: 'warning' })
    await client.delete('/profile')
    auth.logout()
    router.push('/login')
  } catch (err: any) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.error || err.message)
  }
}

function levelClass(count: number) {
  if (count >= 8) return 'level-4'
  if (count >= 4) return 'level-3'
  if (count >= 2) return 'level-2'
  if (count >= 1) return 'level-1'
  return 'level-0'
}

function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${month}-${day} ${hour}:${minute}`
}

function readFile(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

onMounted(load)
</script>
