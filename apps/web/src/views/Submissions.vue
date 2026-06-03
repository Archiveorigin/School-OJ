<template>
  <section class="page">
    <div class="page-header">
      <h2>提交</h2>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="items" @row-click="open">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="user_id" label="用户" width="90" />
        <el-table-column prop="problem_id" label="题目" width="90" />
        <el-table-column prop="language" label="语言" width="110" />
        <el-table-column label="状态" width="130">
          <template #default="{ row }"><StatusBadge :status="row.status" /></template>
        </el-table-column>
        <el-table-column prop="score" label="分数" width="90" />
        <el-table-column label="消息" min-width="260">
          <template #default="{ row }">
            <div class="message-preview">{{ row.message || '-' }}</div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="visible" title="提交详情" width="900px">
      <div v-if="detail" class="submission-detail">
        <div class="summary-grid">
          <span>ID</span><strong>#{{ detail.submission.id }}</strong>
          <span>题目</span><strong>{{ detail.submission.problem_id }}</strong>
          <span>语言</span><strong>{{ detail.submission.language }}</strong>
          <span>状态</span><strong><StatusBadge :status="detail.submission.status" /></strong>
          <span>分数</span><strong>{{ detail.submission.score }}</strong>
          <span>时间</span><strong>{{ detail.submission.created_at }}</strong>
        </div>

        <el-collapse v-model="expandedSections">
          <el-collapse-item v-if="detail.submission.message" title="编译/运行消息" name="message">
            <pre class="detail-pre">{{ detail.submission.message }}</pre>
          </el-collapse-item>
          <el-collapse-item title="测试点结果" name="results">
            <el-table :data="detail.results || []" size="small" max-height="320">
              <el-table-column prop="case_name" label="测试点" min-width="160" />
              <el-table-column label="状态" width="130"><template #default="{ row }"><StatusBadge :status="row.status" /></template></el-table-column>
              <el-table-column prop="time_ms" label="耗时 ms" width="100" />
              <el-table-column prop="memory_kb" label="内存 KB" width="100" />
              <el-table-column label="信息" min-width="220">
                <template #default="{ row }"><div class="message-preview">{{ row.message || '-' }}</div></template>
              </el-table-column>
            </el-table>
          </el-collapse-item>
          <el-collapse-item v-if="detail.submission.source_code" title="源码" name="source">
            <pre class="detail-pre source-pre">{{ detail.submission.source_code }}</pre>
          </el-collapse-item>
          <el-collapse-item title="Trace" name="trace">
            <pre class="detail-pre">{{ JSON.stringify(detail.submission.trace || {}, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { client, type Submission } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'

const items = ref<Submission[]>([])
const detail = ref<any>(null)
const expandedSections = ref<string[]>(['results'])
const visible = ref(false)

async function load() {
  items.value = (await client.get('/submissions')).data
}

async function open(row: Submission) {
  detail.value = (await client.get(`/submissions/${row.id}`)).data
  expandedSections.value = ['results']
  visible.value = true
}

onMounted(load)
</script>

<style scoped>
.message-preview {
  display: -webkit-box;
  max-height: 42px;
  overflow: hidden;
  color: #4b5563;
  line-height: 1.35;
  word-break: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.submission-detail {
  display: grid;
  gap: 14px;
}

.summary-grid {
  display: grid;
  grid-template-columns: 80px 1fr 80px 1fr;
  gap: 10px 14px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
}

.summary-grid span {
  color: #6b7280;
}

.detail-pre {
  max-height: 360px;
  overflow: auto;
  margin: 0;
  padding: 12px;
  border-radius: 8px;
  background: #0f172a;
  color: #e2e8f0;
  white-space: pre-wrap;
}

.source-pre {
  white-space: pre;
}

@media (max-width: 760px) {
  .summary-grid {
    grid-template-columns: 80px 1fr;
  }
}
</style>
