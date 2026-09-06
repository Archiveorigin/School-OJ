import ElementPlus, { ElMessage } from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { client } from './api/client'
import router from './router'
import { useAuthStore } from './stores/auth'
import { useExamLockStore } from './stores/examLock'
import './styles/tokens.css'
import './style.css'

const pinia = createPinia()
const app = createApp(App).use(pinia).use(router).use(ElementPlus)

client.interceptors.response.use(
  (response) => response,
  (error) => {
    const data = error.response?.data
    if (error.response?.status === 403 && data?.code === 'EXAM_SCOPE_REQUIRED' && data?.exam_id) {
      const examID = Number(data.exam_id)
      const message = '考试中仅可访问当前考试'
      useExamLockStore().lock(examID, message)
      if (
        !router.currentRoute.value.path.startsWith(`/exams/${examID}/`)
      ) {
        router.replace(`/exams/${examID}/description`)
      }
      ElMessage.warning(message)
    }
    return Promise.reject(error)
  }
)

app.mount('#app')
useAuthStore().applyTheme()
