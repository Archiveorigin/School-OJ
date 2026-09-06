<template>
  <section class="page activity-page">
    <AdminPageHeader :title="`${title || '考试'} · 行为日志`"><el-button @click="router.push('/admin/exams')">返回考试管理</el-button></AdminPageHeader>
    <div class="panel activity-panel">
      <el-form class="activity-filters" inline @submit.prevent="search">
        <el-form-item label="学生 ID"><el-input v-model="userID" inputmode="numeric" clearable placeholder="全部学生" /></el-form-item>
        <el-form-item label="事件"><el-select v-model="event" clearable placeholder="全部事件"><el-option v-for="(label, key) in eventLabels" :key="key" :label="label" :value="key" /></el-select></el-form-item>
        <el-form-item label="时间"><el-date-picker v-model="range" type="datetimerange" start-placeholder="开始时间" end-placeholder="结束时间" /></el-form-item>
        <el-button native-type="submit" type="primary">查询</el-button>
      </el-form>
      <el-alert v-if="error" :title="error" type="error" :closable="false" />
      <el-table v-loading="loading" :data="items" row-key="id">
        <el-table-column type="expand"><template #default="{ row }">
          <dl class="event-detail">
            <dt>客户端时间</dt><dd>{{ row.client_at ? formatDateTime(row.client_at) : '—' }}</dd>
            <template v-if="row.meta?.login_at"><dt>最近登录时间</dt><dd>{{ formatDateTime(row.meta.login_at) }}</dd><dt>登录 IP</dt><dd>{{ row.meta.login_ip || '—' }}</dd></template>
            <template v-if="row.meta?.route"><dt>拦截接口</dt><dd>{{ row.meta.method }} {{ row.meta.route }}</dd></template>
            <dt>事件标识</dt><dd>{{ row.event_id }}</dd>
          </dl>
        </template></el-table-column>
        <el-table-column prop="user_name" label="学生" min-width="100" />
        <el-table-column prop="user_id" label="学生 ID" width="90" />
        <el-table-column label="事件" min-width="140"><template #default="{ row }">{{ eventLabels[row.event] || row.event }}</template></el-table-column>
        <el-table-column prop="ip" label="IP 地址" min-width="140" />
        <el-table-column label="记录时间" min-width="180"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column>
      </el-table>
      <ListPagination v-model:page="page" v-model:page-size="pageSize" :total="total" />
    </div>
  </section>
</template>
<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { generatedApi, type components } from '../../api/client'
import AdminPageHeader from '../../components/admin/AdminPageHeader.vue'
import ListPagination from '../../components/ListPagination.vue'
import { formatDateTime } from '../../features/time'
const route = useRoute(), router = useRouter()
const title = ref(''), userID = ref(''), event = ref(''), range = ref<[Date, Date] | null>(null)
const items = ref<components['schemas']['ExamActivity'][]>([]), total = ref(0), page = ref(1), pageSize = ref(20), loading = ref(false), error = ref('')
const eventLabels: Record<string, string> = { enter: '进入考试', login: '登录', finish: '交卷', window_leave: '离开窗口', window_return: '返回窗口', devtools_shortcut: '受限快捷键', access_blocked: '访问拦截' }
async function load() {
  loading.value = true; error.value = ''
  try {
    const { data } = await generatedApi.listExamActivity({ path: { id: String(route.params.id) }, query: { page: page.value, page_size: pageSize.value, user_id: userID.value || undefined, event: event.value || undefined, from: range.value?.[0].toISOString(), to: range.value?.[1].toISOString() } })
    items.value = data.items; total.value = data.total; title.value = data.exam_title
  } catch (err: any) { items.value = []; total.value = 0; error.value = err.response?.data?.error || '日志加载失败' }
  finally { loading.value = false }
}
function search() { if (page.value !== 1) page.value = 1; else void load() }
watch([page, pageSize, () => route.params.id], load)
onMounted(load)
</script>
<style scoped>
.activity-panel { padding: 20px; min-width: 0; }
.activity-filters { display: flex; align-items: start; flex-wrap: wrap; gap: 8px; }
.activity-filters :deep(.el-select) { width: 150px; }
.event-detail { display: grid; grid-template-columns: 120px minmax(0, 1fr); gap: 10px; padding: 8px 24px; }
.event-detail dt { color: var(--muted); }
.event-detail dd { margin: 0; overflow-wrap: anywhere; }
@media (max-width: 600px) { .activity-panel { padding: 12px; } .activity-filters :deep(.el-form-item) { width: 100%; margin-right: 0; } .activity-filters :deep(.el-date-editor) { width: 100%; } }
</style>
