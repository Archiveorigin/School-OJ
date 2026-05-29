<template>
  <main class="auth-page">
    <section class="auth-panel">
      <div class="auth-brand">
        <img src="/logo.jpg" alt="黄海在线题测平台" />
        <div>
          <h1>黄海在线题测平台</h1>
          <p>面向课程、作业与考试的在线评测平台</p>
        </div>
      </div>

      <el-form class="auth-form" :model="form" @submit.prevent="submit">
        <el-form-item>
          <el-input v-model="form.email" placeholder="邮箱" autocomplete="email" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" placeholder="密码" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <el-button class="full-button" type="primary" native-type="submit" :loading="loading">登录</el-button>
      </el-form>

      <div class="auth-links">
        <router-link to="/register">注册账号</router-link>
        <router-link v-if="resetAvailable" to="/forgot-password">通过邮箱找回密码</router-link>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const resetAvailable = ref(false)
const form = reactive({ email: '', password: '' })

async function submit() {
  if (!form.email || !form.password) {
    ElMessage.error('请填写邮箱和密码')
    return
  }
  loading.value = true
  try {
    await auth.login(form.email, form.password)
    router.push('/')
  } catch (err: any) {
    resetAvailable.value = Boolean(err.response?.data?.password_reset_available)
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}
</script>
