<template>
  <section class="page profile-page">
    <section class="panel identity-panel">
      <div class="identity-main">
        <button class="profile-avatar" type="button" aria-label="更新个人资料" @click="openEditor">
          <img v-if="profile?.user.avatar_url" :src="profile.user.avatar_url" alt="" />
          <span v-else>{{ initials }}</span>
        </button>
        <div class="identity-copy">
          <span class="eyebrow">PERSONAL ACADEMIC PROFILE</span>
          <h1>{{ profile?.user.name || auth.user?.name }}</h1>
          <p>{{ profile?.user.email || auth.user?.email }}</p>
          <div class="profile-tags">
            <el-tag>{{ roleLabel }}</el-tag>
            <el-tag v-if="canAuthor" type="warning">已获出题资格</el-tag>
            <el-tag v-if="profile?.user.email_verified" type="success">邮箱已验证</el-tag>
            <el-tag v-if="profile?.user.student_no" type="info">学号 {{ profile.user.student_no }}</el-tag>
          </div>
        </div>
      </div>
      <div class="identity-meta">
        <div><span>身份</span><strong>{{ roleLabel }}</strong></div>
        <div><span>账号</span><strong>{{ profile?.user.student_no || '未设置学号' }}</strong></div>
        <div><span>加入平台</span><strong>{{ joinedAt }}</strong></div>
        <el-button type="primary" plain @click="openEditor">更新个人资料</el-button>
      </div>
    </section>

    <section class="panel author-panel">
      <div class="author-copy">
        <span class="eyebrow">PROBLEM AUTHOR PROGRAM</span>
        <h2>{{ canAuthor ? '出题工作台已开放' : '申请成为出题者' }}</h2>
        <p v-if="canAuthor">你可以创建题目、导入 Markdown 题面和测试点压缩包，并维护自己提供的题目。</p>
        <p v-else-if="authorApplication?.status === 'pending'">申请已提交，管理员审核通过后会为当前账号开通独立出题权限，学生角色保持不变。</p>
        <p v-else>说明你的出题经验、擅长方向或计划提供的题目类型，管理员审核通过后即可开始出题。</p>
        <el-alert
          v-if="authorApplication?.status === 'rejected'"
          type="warning"
          :closable="false"
          :title="authorApplication.review_note || '上次申请未通过，你可以补充说明后重新申请。'"
        />
      </div>
      <div v-if="canAuthor" class="author-actions">
        <el-button type="primary" size="large" @click="router.push('/problems/create')">进入出题页面</el-button>
        <el-button size="large" @click="router.push('/problems')">查看题库</el-button>
      </div>
      <div v-else-if="authorApplication?.status === 'pending'" class="application-status">
        <el-tag type="warning" size="large">等待审核</el-tag>
        <span>提交于 {{ formatApplicationTime(authorApplication.created_at) }}</span>
      </div>
      <div v-else class="application-form">
        <el-input
          v-model="authorMotivation"
          type="textarea"
          :rows="4"
          maxlength="800"
          show-word-limit
          placeholder="至少 10 个字，例如：有算法竞赛经验，计划提供基础数据结构与动态规划题目。"
        />
        <el-button type="primary" :loading="applying" @click="applyAsAuthor">
          {{ authorApplication?.status === 'rejected' ? '重新提交申请' : '提交申请' }}
        </el-button>
      </div>
    </section>

    <section class="panel activity-panel">
      <div class="activity-heading">
        <div>
          <span class="eyebrow">LEARNING ACTIVITY</span>
          <h2>{{ profile?.activity_label || '解题活跃度' }}</h2>
          <p>最近一年共记录 {{ totalActivity }} {{ profile?.activity_unit || '道题' }}，活跃 {{ activeDays }} 天。</p>
        </div>
        <div class="activity-metrics">
          <div><strong>{{ profile?.solved || 0 }}</strong><span>累计通过</span></div>
          <div><strong>{{ activeDays }}</strong><span>活跃天数</span></div>
          <div><strong>{{ longestStreak }}</strong><span>最长连续</span></div>
        </div>
      </div>

      <div class="calendar-scroll">
        <div class="calendar-chart" :style="{ '--week-count': String(weekCount) }">
          <div class="month-axis" aria-hidden="true">
            <span v-for="month in monthLabels" :key="`${month.label}-${month.column}`" :style="{ gridColumn: month.column }">
              {{ month.label }}
            </span>
          </div>
          <div class="calendar-body">
            <div class="weekday-axis" aria-hidden="true">
              <span>日</span><span>一</span><span>二</span><span>三</span><span>四</span><span>五</span><span>六</span>
            </div>
            <div class="activity-grid" role="img" aria-label="最近一年解题活跃度热力图">
              <template v-for="(cell, index) in calendarCells" :key="cell?.date || `empty-${index}`">
                <span v-if="!cell" class="activity-cell empty-cell"></span>
                <el-tooltip v-else placement="top" effect="dark" :show-after="120">
                  <template #content>
                    <div class="activity-tooltip">
                      <strong>{{ cell.date }}</strong>
                      <span>{{ cell.count ? `${activityVerb} ${cell.count} ${profile?.activity_unit || '道题'}` : '当日暂无记录' }}</span>
                      <ul v-if="cell.problems.length">
                        <li v-for="problem in cell.problems.slice(0, 8)" :key="problem.id">
                          {{ problem.display_code || `#${problem.id}` }} · {{ problem.title }}
                        </li>
                      </ul>
                      <small v-if="cell.problems.length > 8">另有 {{ cell.problems.length - 8 }} 道题</small>
                    </div>
                  </template>
                  <button
                    type="button"
                    class="activity-cell"
                    :class="levelClass(cell.count)"
                    :aria-label="`${cell.date}，${cell.count} ${profile?.activity_unit || '道题'}`"
                  ></button>
                </el-tooltip>
              </template>
            </div>
          </div>
        </div>
      </div>
      <div class="activity-legend">
        <span>少</span>
        <i class="activity-cell level-0"></i>
        <i class="activity-cell level-1"></i>
        <i class="activity-cell level-2"></i>
        <i class="activity-cell level-3"></i>
        <i class="activity-cell level-4"></i>
        <span>多</span>
      </div>
    </section>

    <el-dialog
      v-model="editVisible"
      title="更新个人资料"
      width="min(1040px, 94vw)"
      align-center
      destroy-on-close
      class="profile-edit-dialog"
      @closed="onEditorClosed"
    >
      <div class="profile-dialog-grid">
        <section class="dialog-form-section">
          <div class="dialog-section-title"><h3>基本资料</h3><span>用于课程与评测记录展示</span></div>
          <el-form label-position="top" :model="profileForm">
            <div class="form-grid">
              <el-form-item label="姓名 / 昵称">
                <el-input v-model="profileForm.name" size="large" />
              </el-form-item>
              <el-form-item label="学号">
                <el-input :model-value="profile?.user.student_no || '未设置'" size="large" disabled />
              </el-form-item>
              <el-form-item label="角色">
                <el-input :model-value="roleLabel" size="large" disabled />
              </el-form-item>
              <el-form-item label="当前邮箱">
                <el-input :model-value="profile?.user.email" size="large" disabled />
              </el-form-item>
            </div>

            <div class="dialog-section-title password-title"><h3>修改密码</h3><span>不修改时保持为空</span></div>
            <el-form-item label="当前密码">
              <el-input v-model="passwordForm.current_password" type="password" show-password size="large" placeholder="修改密码时填写" />
            </el-form-item>
            <div class="form-grid">
              <el-form-item label="新密码">
                <el-input v-model="passwordForm.new_password" type="password" show-password size="large" />
              </el-form-item>
              <el-form-item label="再次输入新密码">
                <el-input v-model="passwordForm.confirm_password" type="password" show-password size="large" />
              </el-form-item>
            </div>
          </el-form>
        </section>

        <aside class="dialog-side-section">
          <div class="avatar-editor">
            <button type="button" class="dialog-avatar" @click="avatarInput?.click()">
              <img v-if="profile?.user.avatar_url" :src="profile.user.avatar_url" alt="" />
              <span v-else>{{ initials }}</span>
              <small>更换头像</small>
            </button>
            <input ref="avatarInput" class="hidden-input" type="file" accept="image/*" @change="uploadAvatar" />
            <div><strong>{{ profile?.user.name }}</strong><span>支持 JPG、PNG、WebP，最大 2MB</span></div>
          </div>

          <div class="email-editor">
            <div class="dialog-section-title"><h3>邮箱换绑</h3><span>新邮箱需完成验证码校验</span></div>
            <el-input v-model="emailForm.email" size="large" placeholder="新邮箱地址" />
            <el-input v-model="emailForm.code" size="large" maxlength="6" placeholder="6 位验证码">
              <template #append>
                <el-button :loading="sendingCode" @click="sendEmailCode">发送验证码</el-button>
              </template>
            </el-input>
          </div>

          <div class="privacy-note">
            <strong>资料说明</strong>
            <p>姓名、头像与学习记录仅在平台教学场景中使用。账号注销后将无法继续登录。</p>
          </div>
        </aside>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="danger" plain @click="deleteAccount">注销账号</el-button>
          <div>
            <el-button @click="editVisible = false">取消</el-button>
            <el-button type="primary" :loading="savingProfile" @click="saveAll">更新</el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { client, type AuthorApplication, type User } from '../api/client'
import { useAuthStore } from '../stores/auth'

interface ActivityProblem {
  id: number
  display_code?: string
  title: string
}

interface ActivityDay {
  date: string
  count: number
  problems: ActivityProblem[]
}

interface ProfileData {
  user: User
  solved: number
  submissions: number
  by_status: Array<{ status: string; count: number }>
  activity: ActivityDay[]
  activity_label?: string
  activity_unit?: string
}

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const profile = ref<ProfileData | null>(null)
const editVisible = ref(false)
const avatarInput = ref<HTMLInputElement>()
const savingProfile = ref(false)
const sendingCode = ref(false)
const applying = ref(false)
const authorApplication = ref<AuthorApplication | null>(null)
const authorMotivation = ref('')

const profileForm = reactive({ name: '' })
const emailForm = reactive({ email: '', code: '' })
const passwordForm = reactive({ current_password: '', new_password: '', confirm_password: '' })

const initials = computed(() => (profile.value?.user.name || auth.user?.name || 'U').trim().slice(0, 1).toUpperCase())
const roleLabel = computed(() => {
  const role = profile.value?.user.role || auth.user?.role
  return role === 'admin' ? '管理员' : role === 'teacher' ? '教师' : role === 'problem_setter' ? '出题者' : '学生'
})
const canAuthor = computed(() => Boolean(profile.value?.user.can_author || auth.user?.can_author) || ['admin', 'teacher', 'problem_setter'].includes(profile.value?.user.role || auth.role || ''))
const joinedAt = computed(() => {
  const value = profile.value?.user.created_at
  if (!value) return '平台成员'
  return new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long' }).format(new Date(value))
})
const activityVerb = computed(() => canAuthor.value ? '上传' : '解出')

const days = computed<ActivityDay[]>(() => {
  const map = new Map((profile.value?.activity || []).map((item) => [item.date, item]))
  const today = new Date()
  const result: ActivityDay[] = []
  for (let i = 364; i >= 0; i -= 1) {
    const date = new Date(today)
    date.setHours(0, 0, 0, 0)
    date.setDate(today.getDate() - i)
    const key = formatDateKey(date)
    const item = map.get(key)
    result.push({ date: key, count: item?.count || 0, problems: item?.problems || [] })
  }
  return result
})
const leadingEmpty = computed(() => {
  const first = days.value[0]
  return first ? new Date(`${first.date}T00:00:00`).getDay() : 0
})
const calendarCells = computed<Array<ActivityDay | null>>(() => [
  ...Array.from({ length: leadingEmpty.value }, () => null),
  ...days.value
])
const weekCount = computed(() => Math.ceil(calendarCells.value.length / 7))
const monthLabels = computed(() => {
  const labels: Array<{ label: string; column: string }> = []
  let lastMonth = -1
  days.value.forEach((day, index) => {
    const date = new Date(`${day.date}T00:00:00`)
    if (date.getMonth() === lastMonth) return
    lastMonth = date.getMonth()
    const week = Math.floor((leadingEmpty.value + index) / 7) + 1
    labels.push({ label: `${date.getMonth() + 1}月`, column: `${week} / span 4` })
  })
  return labels
})
const totalActivity = computed(() => days.value.reduce((sum, day) => sum + day.count, 0))
const activeDays = computed(() => days.value.filter((day) => day.count > 0).length)
const longestStreak = computed(() => {
  let current = 0
  let longest = 0
  days.value.forEach((day) => {
    current = day.count > 0 ? current + 1 : 0
    longest = Math.max(longest, current)
  })
  return longest
})

async function load() {
  try {
    const { data } = await client.get('/profile')
    profile.value = data
    profileForm.name = data.user.name
    auth.updateUser(data.user)
    if (data.user.role === 'student' && !data.user.can_author) {
      authorApplication.value = (await client.get('/author-applications/me')).data.application
    } else {
      authorApplication.value = null
    }
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  }
}

async function applyAsAuthor() {
  const motivation = authorMotivation.value.trim()
  if (motivation.length < 10) {
    ElMessage.error('申请说明至少需要 10 个字')
    return
  }
  applying.value = true
  try {
    authorApplication.value = (await client.post('/author-applications', { motivation })).data
    authorMotivation.value = ''
    ElMessage.success('申请已提交，请等待管理员审核')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    applying.value = false
  }
}

function formatApplicationTime(value: string) {
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function openEditor() {
  resetEditorForms()
  editVisible.value = true
}

async function saveAll() {
  const name = profileForm.name.trim()
  if (!name) {
    ElMessage.error('姓名不能为空')
    return
  }
  const changingPassword = Boolean(passwordForm.current_password || passwordForm.new_password || passwordForm.confirm_password)
  if (changingPassword) {
    if (!passwordForm.current_password || passwordForm.new_password.length < 6) {
      ElMessage.error('请填写当前密码，新密码至少 6 位')
      return
    }
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      ElMessage.error('两次输入的新密码不一致')
      return
    }
  }
  const changingEmail = Boolean(emailForm.email || emailForm.code)
  if (changingEmail && (!emailForm.email || !emailForm.code)) {
    ElMessage.error('请填写新邮箱和验证码')
    return
  }

  savingProfile.value = true
  try {
    if (name !== profile.value?.user.name) {
      const { data } = await client.put('/profile', { name })
      auth.updateUser(data)
    }
    if (changingPassword) {
      await client.post('/profile/password', {
        current_password: passwordForm.current_password,
        new_password: passwordForm.new_password
      })
    }
    if (changingEmail) {
      const { data } = await client.post('/profile/email', {
        email: emailForm.email.trim(),
        code: emailForm.code.trim()
      })
      auth.updateUser(data)
    }
    await load()
    editVisible.value = false
    ElMessage.success('个人资料已更新')
  } catch (err: any) {
    const text = err.response?.data?.error || err.message
    ElMessage.error(text === 'current password is incorrect' ? '当前密码不正确' : text)
  } finally {
    savingProfile.value = false
  }
}

async function uploadAvatar(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.error('头像文件不能超过 2MB')
    input.value = ''
    return
  }
  try {
    const avatar = await readFile(file)
    const { data } = await client.put('/profile', { avatar_url: avatar })
    auth.updateUser(data)
    await load()
    ElMessage.success('头像已更新')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    input.value = ''
  }
}

async function sendEmailCode() {
  if (!emailForm.email.trim()) {
    ElMessage.error('请填写新邮箱')
    return
  }
  sendingCode.value = true
  try {
    await client.post('/profile/email-code', { email: emailForm.email.trim() })
    ElMessage.success('验证码已发送')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    sendingCode.value = false
  }
}

async function deleteAccount() {
  try {
    await ElMessageBox.confirm('注销后账号将无法继续登录，确认继续？', '注销账号', {
      type: 'warning',
      confirmButtonText: '确认注销',
      cancelButtonText: '取消',
      confirmButtonClass: 'el-button--danger'
    })
    await client.delete('/profile')
    auth.logout()
    router.push('/login')
  } catch (err: any) {
    if (err !== 'cancel' && err !== 'close') ElMessage.error(err.response?.data?.error || err.message)
  }
}

function resetEditorForms() {
  profileForm.name = profile.value?.user.name || ''
  emailForm.email = ''
  emailForm.code = ''
  passwordForm.current_password = ''
  passwordForm.new_password = ''
  passwordForm.confirm_password = ''
}

function onEditorClosed() {
  if (route.query.edit !== '1') return
  const query = { ...route.query }
  delete query.edit
  router.replace({ path: route.path, query })
}

function levelClass(count: number) {
  if (count >= 8) return 'level-4'
  if (count >= 4) return 'level-3'
  if (count >= 2) return 'level-2'
  if (count >= 1) return 'level-1'
  return 'level-0'
}

function formatDateKey(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function readFile(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

watch(
  () => route.query.edit,
  (value) => {
    if (value === '1') openEditor()
  },
  { immediate: true }
)
onMounted(load)
</script>

<style scoped>
.profile-page { max-width: 1220px; margin: 0 auto; padding: 38px 28px 68px; }
.identity-panel { display: flex; align-items: center; justify-content: space-between; gap: 34px; padding: 34px 38px; overflow: hidden; background: radial-gradient(circle at 92% 8%, rgba(56,189,248,.12), transparent 30%), var(--surface); }
.identity-main { display: flex; align-items: center; gap: 22px; min-width: 0; }
.profile-avatar { width: 104px; height: 104px; font-size: 38px; }
.identity-copy { min-width: 0; }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .18em; }
.identity-copy h1 { margin: 8px 0 4px; font-family: 'Noto Serif SC', SimSun, serif; font-size: 32px; }
.identity-copy p { margin: 0 0 12px; color: var(--muted); word-break: break-all; }
.identity-meta { min-width: 410px; display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
.identity-meta div { display: grid; gap: 5px; padding: 15px; border: 1px solid var(--border); border-radius: 10px; background: color-mix(in srgb, var(--surface-strong) 74%, transparent); }
.identity-meta span { color: var(--muted); font-size: 12px; }
.identity-meta strong { overflow: hidden; font-size: 14px; text-overflow: ellipsis; white-space: nowrap; }
.identity-meta .el-button { grid-column: 1 / -1; margin-top: 2px; }
.activity-panel { margin-top: 18px; padding: 30px 34px 24px; }
.author-panel { display: grid; grid-template-columns: minmax(0, 1fr) minmax(320px, .8fr); gap: 28px; margin-top: 18px; padding: 28px 34px; }
.author-copy h2 { margin: 7px 0; font-size: 24px; }
.author-copy p { margin: 0 0 12px; color: var(--muted); line-height: 1.7; }
.author-actions { display: flex; align-items: center; justify-content: flex-end; gap: 10px; }
.application-form { display: grid; gap: 10px; }
.application-form .el-button { justify-self: end; }
.application-status { display: flex; align-items: center; justify-content: flex-end; gap: 12px; color: var(--muted); }
.activity-heading { display: flex; align-items: end; justify-content: space-between; gap: 28px; }
.activity-heading h2 { margin: 8px 0 5px; font-family: 'Noto Serif SC', SimSun, serif; font-size: 27px; }
.activity-heading p { margin: 0; color: var(--muted); }
.activity-metrics { display: grid; grid-template-columns: repeat(3, minmax(86px, 1fr)); gap: 8px; }
.activity-metrics div { display: grid; gap: 3px; padding: 12px 15px; text-align: center; border: 1px solid var(--border); border-radius: 9px; }
.activity-metrics strong { color: var(--accent); font-size: 22px; }
.activity-metrics span { color: var(--muted); font-size: 11px; }
.calendar-scroll { margin-top: 30px; overflow-x: auto; padding: 6px 0 10px; }
.calendar-chart { width: max-content; min-width: 100%; --cell-size: 13px; --cell-gap: 4px; }
.month-axis { display: grid; grid-template-columns: repeat(var(--week-count), var(--cell-size)); gap: var(--cell-gap); margin-left: 28px; min-height: 24px; color: var(--muted); font-size: 11px; }
.month-axis span { white-space: nowrap; }
.calendar-body { display: flex; gap: 8px; }
.weekday-axis { width: 20px; display: grid; grid-template-rows: repeat(7, var(--cell-size)); gap: var(--cell-gap); color: var(--muted); font-size: 10px; line-height: var(--cell-size); }
.activity-grid { display: grid; grid-auto-flow: column; grid-template-rows: repeat(7, var(--cell-size)); grid-auto-columns: var(--cell-size); gap: var(--cell-gap); width: max-content; }
.activity-cell { width: var(--cell-size, 13px); height: var(--cell-size, 13px); display: block; padding: 0; border: 0; border-radius: 3px; background: #e8edf2; }
button.activity-cell { cursor: pointer; transition: transform .12s ease, outline-color .12s ease; }
button.activity-cell:hover { z-index: 2; transform: scale(1.32); outline: 1px solid color-mix(in srgb, var(--accent) 62%, transparent); }
.empty-cell { visibility: hidden; }
:global(.dark) .activity-cell.level-0 { background: #293548; }
.activity-cell.level-1 { background: #a7d7c5; }
.activity-cell.level-2 { background: #59b99a; }
.activity-cell.level-3 { background: #16856d; }
.activity-cell.level-4 { background: #075e54; }
.activity-legend { display: flex; align-items: center; justify-content: flex-end; gap: 5px; color: var(--muted); font-size: 11px; }
.activity-legend .activity-cell { --cell-size: 12px; }
.activity-tooltip { max-width: 330px; display: grid; gap: 5px; }
.activity-tooltip span, .activity-tooltip small { color: #cbd5e1; }
.activity-tooltip ul { max-height: 180px; margin: 4px 0 0; padding-left: 18px; overflow: auto; }
.activity-tooltip li { margin: 3px 0; line-height: 1.45; }
.profile-dialog-grid { display: grid; grid-template-columns: minmax(0, 1.45fr) minmax(300px, .75fr); gap: 28px; }
.dialog-form-section { padding-right: 28px; border-right: 1px solid var(--border); }
.dialog-section-title { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin-bottom: 17px; }
.dialog-section-title h3 { margin: 0; font-size: 18px; }
.dialog-section-title span { color: var(--muted); font-size: 12px; }
.password-title { margin-top: 10px; padding-top: 22px; border-top: 1px solid var(--border); }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 16px; }
.dialog-side-section { display: flex; flex-direction: column; gap: 24px; }
.avatar-editor { display: flex; align-items: center; gap: 17px; padding-bottom: 22px; border-bottom: 1px solid var(--border); }
.dialog-avatar { position: relative; width: 92px; height: 92px; display: grid; place-items: center; flex: 0 0 auto; padding: 0; overflow: hidden; color: #fff; border: 0; border-radius: 18px; background: linear-gradient(135deg, var(--accent), #0f766e); font-size: 31px; font-weight: 800; cursor: pointer; }
.dialog-avatar img { width: 100%; height: 100%; object-fit: cover; }
.dialog-avatar small { position: absolute; inset: auto 0 0; padding: 5px; color: #fff; background: rgba(2,6,23,.7); font-size: 10px; }
.avatar-editor > div { display: grid; gap: 6px; }
.avatar-editor > div span { color: var(--muted); font-size: 12px; line-height: 1.5; }
.email-editor { display: grid; gap: 12px; }
.email-editor .dialog-section-title { display: grid; gap: 4px; margin-bottom: 3px; }
.privacy-note { margin-top: auto; padding: 17px; border: 1px solid var(--border); border-radius: 10px; background: var(--app-bg); }
.privacy-note p { margin: 7px 0 0; color: var(--muted); font-size: 12px; line-height: 1.7; }
.dialog-footer { display: flex; align-items: center; justify-content: space-between; gap: 16px; width: 100%; }
.hidden-input { display: none; }
:global(.profile-edit-dialog .el-dialog__body) { padding-top: 12px; }
@media (max-width: 900px) { .identity-panel, .activity-heading { align-items: stretch; flex-direction: column; } .identity-meta { min-width: 0; width: 100%; } .author-panel { grid-template-columns: 1fr; } .author-actions, .application-status { justify-content: flex-start; } .profile-dialog-grid { grid-template-columns: 1fr; } .dialog-form-section { padding-right: 0; border-right: 0; } }
@media (max-width: 620px) { .profile-page { padding: 20px 13px 46px; } .identity-panel, .activity-panel { padding: 24px 20px; } .identity-main { align-items: flex-start; flex-direction: column; } .identity-meta, .activity-metrics, .form-grid { grid-template-columns: 1fr; } .profile-avatar { width: 88px; height: 88px; } .dialog-footer { align-items: stretch; flex-direction: column-reverse; } .dialog-footer > div { display: flex; } .dialog-footer .el-button { flex: 1; } }
</style>
