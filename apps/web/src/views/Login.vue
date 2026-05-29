<template>
  <main class="login">
    <el-form class="login-box" :model="form" @submit.prevent="submit">
      <h1>School OJ</h1>
      <el-form-item>
        <el-input v-model="form.email" placeholder="邮箱" />
      </el-form-item>
      <el-form-item>
        <el-input v-model="form.password" placeholder="密码" type="password" show-password />
      </el-form-item>
      <el-button type="primary" native-type="submit" :loading="loading">登录</el-button>
      <p class="muted">admin@school.local / teacher@school.local / student@school.local，密码 password</p>
    </el-form>
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
const form = reactive({ email: 'admin@school.local', password: 'password' })

async function submit() {
  loading.value = true
  try {
    await auth.login(form.email, form.password)
    router.push('/')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background: #eef2f7;
}

.login-box {
  width: min(420px, calc(100vw - 32px));
  background: #fff;
  border: 1px solid #d9dee8;
  border-radius: 8px;
  padding: 28px;
}

h1 {
  margin: 0 0 20px;
}
</style>
