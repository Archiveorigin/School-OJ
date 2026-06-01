<template>
  <main class="auth-page">
    <section class="auth-panel">
      <div class="auth-brand">
        <img src="/logo.jpg" alt="黄海在线题测平台" />
        <div>
          <h1>创建账号</h1>
          <p>邮箱验证码通过后即可进入平台</p>
        </div>
      </div>

      <el-form class="auth-form" :model="form" @submit.prevent="submit">
        <el-form-item>
          <el-input v-model="form.email" placeholder="邮箱" autocomplete="email">
            <template #append>
              <el-button :loading="sending" @click="sendCode">{{ codeButtonText }}</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.code" placeholder="六位验证码" maxlength="6" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.name" placeholder="姓名" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.student_no" placeholder="学号" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" placeholder="密码" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-button class="full-button" type="primary" native-type="submit" :loading="submitting">注册</el-button>
      </el-form>

      <div class="auth-links">
        <router-link to="/login">返回登录</router-link>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client } from '../api/client'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const sending = ref(false)
const submitting = ref(false)
const seconds = ref(0)
let timer: number | undefined

const form = reactive({
  email: '',
  code: '',
  name: '',
  student_no: '',
  password: ''
})

const codeButtonText = computed(() => (seconds.value > 0 ? `${seconds.value}s` : '发送验证码'))

async function sendCode() {
  if (!form.email || seconds.value > 0) return
  sending.value = true
  try {
    await client.post('/auth/send-code', { email: form.email, purpose: 'register' })
    ElMessage.success('验证码已发送；本地 Docker 环境请打开 http://localhost:8025 查看邮件')
    startCountdown()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    sending.value = false
  }
}

async function submit() {
  if (!form.email || !form.code || !form.name || !form.password) {
    ElMessage.error('请填写邮箱、验证码、姓名和密码')
    return
  }
  submitting.value = true
  try {
    const { data } = await client.post('/auth/register', { ...form })
    auth.setSession(data.token, data.user)
    router.push('/')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    submitting.value = false
  }
}

function startCountdown() {
  seconds.value = 60
  window.clearInterval(timer)
  timer = window.setInterval(() => {
    seconds.value -= 1
    if (seconds.value <= 0) window.clearInterval(timer)
  }, 1000)
}

onBeforeUnmount(() => window.clearInterval(timer))
</script>
