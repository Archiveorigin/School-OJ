<template>
  <section class="page author-management-page">
    <div class="page-header management-header">
      <div>
        <span class="eyebrow">AUTHORING CONTROL</span>
        <h2>出题管理</h2>
        <p>集中处理出题资格申请与非管理员提交的待审题目。</p>
      </div>
      <div class="header-actions">
        <el-popover placement="bottom-end" :width="520" trigger="click">
          <template #reference>
            <el-badge :value="pendingApplications.length" :hidden="pendingApplications.length === 0" class="reminder-badge">
              <button type="button" class="reminder-button" aria-label="待处理出题者申请">
                <span class="reminder-icon" :style="{ maskImage: `url(${remindIcon})`, WebkitMaskImage: `url(${remindIcon})` }"></span>
              </button>
            </el-badge>
          </template>
          <div class="reminder-popover">
            <header>
              <div><strong>待处理申请</strong><span>{{ pendingApplications.length }} 条出题资格申请</span></div>
              <el-button text @click="activePanel = 'people'">进入人员管理</el-button>
            </header>
            <div v-if="pendingApplications.length" class="pending-list">
              <article v-for="item in pendingApplications" :key="item.id">
                <div>
                  <strong>{{ item.user?.name || `用户 #${item.user_id}` }}</strong>
                  <span>{{ item.user?.email }}</span>
                  <p>{{ item.motivation }}</p>
                </div>
                <div class="inline-actions">
                  <el-button size="small" type="success" @click="openApplicationReview(item, 'approved')">通过</el-button>
                  <el-button size="small" type="danger" plain @click="openApplicationReview(item, 'rejected')">驳回</el-button>
                </div>
              </article>
            </div>
            <el-empty v-else :image-size="72" description="暂无待处理申请" />
          </div>
        </el-popover>
        <el-button :loading="loading" @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel switch-panel">
      <el-radio-group v-model="activePanel" size="large">
        <el-radio-button value="people">人员管理</el-radio-button>
        <el-radio-button value="problems">题目管理</el-radio-button>
      </el-radio-group>
      <span v-if="activePanel === 'people'">审核通过只开通出题标识，不改变学生、教师等基础角色。</span>
      <span v-else>非管理员题目通过审核后才会出现在公共题库。</span>
    </div>

    <section v-if="activePanel === 'people'" class="panel content-panel">
      <div class="section-heading">
        <div><h3>出题人员申请</h3><p>查看申请说明、授权状态和管理员审核记录。</p></div>
        <el-tag type="danger" effect="dark">{{ pendingApplications.length }} 条待处理</el-tag>
      </div>
      <el-table :data="applications" v-loading="loading" empty-text="暂无出题人员申请">
        <el-table-column label="申请人" min-width="190">
          <template #default="{ row }">
            <strong>{{ row.user?.name || `用户 #${row.user_id}` }}</strong>
            <div class="muted">{{ row.user?.email }}</div>
          </template>
        </el-table-column>
        <el-table-column label="基础角色" width="110">
          <template #default="{ row }">{{ roleText(row.user?.role) }}</template>
        </el-table-column>
        <el-table-column prop="motivation" label="申请说明" min-width="280" show-overflow-tooltip />
        <el-table-column label="出题标识" width="110">
          <template #default="{ row }">
            <el-tag :type="row.user?.can_author ? 'success' : 'info'">{{ row.user?.can_author ? '已开通' : '未开通' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="申请状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="申请时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="review_note" label="审核说明" min-width="180">
          <template #default="{ row }">{{ row.review_note || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button size="small" type="success" text @click="openApplicationReview(row, 'approved')">通过</el-button>
              <el-button size="small" type="danger" text @click="openApplicationReview(row, 'rejected')">驳回</el-button>
            </template>
            <span v-else class="muted">已处理</span>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-else class="panel content-panel">
      <div class="section-heading problem-heading">
        <div><h3>题目审批</h3><p>核对题面属性及全部后台测试点后决定发布或退回。</p></div>
        <el-select v-model="problemStatusFilter" style="width: 150px">
          <el-option label="全部状态" value="all" />
          <el-option label="待审批" value="pending" />
          <el-option label="已通过" value="approved" />
          <el-option label="已退回" value="rejected" />
        </el-select>
      </div>
      <el-table :data="filteredProblemReviews" v-loading="loading" empty-text="暂无题目审批记录">
        <el-table-column label="题目名称" min-width="210">
          <template #default="{ row }">
            <el-button link type="primary" @click="router.push(`/problems/${row.problem.display_code || row.problem_id}`)">
              {{ row.problem.title }}
            </el-button>
            <div v-if="row.review_note" class="muted review-note">{{ row.review_note }}</div>
          </template>
        </el-table-column>
        <el-table-column label="题目号" width="110">
          <template #default="{ row }"><strong>{{ row.problem.display_code || `#${row.problem_id}` }}</strong></template>
        </el-table-column>
        <el-table-column label="难度" width="100">
          <template #default="{ row }"><el-tag effect="plain">{{ row.problem.difficulty || '未设置' }}</el-tag></template>
        </el-table-column>
        <el-table-column label="标签" min-width="180">
          <template #default="{ row }">
            <div class="tag-list">
              <el-tag v-for="tag in problemTags(row.problem.tags)" :key="tag" size="small" type="info">{{ tag }}</el-tag>
              <span v-if="!problemTags(row.problem.tags).length" class="muted">-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="测试点" width="145">
          <template #default="{ row }">
            <ProblemTestDownloads
              :problem-id="row.problem_id"
              :problem-code="row.problem.display_code"
              :button-label="`${row.test_point_count} 个测试点`"
              :dialog-title="`题目测试点 · ${row.problem.title}`"
            />
          </template>
        </el-table-column>
        <el-table-column label="出题人" min-width="140">
          <template #default="{ row }">
            <strong>{{ row.author?.name || `用户 #${row.author_id}` }}</strong>
            <div class="muted">{{ row.author?.email }}</div>
          </template>
        </el-table-column>
        <el-table-column label="提交时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.submitted_at) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="105">
          <template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button size="small" type="success" text @click="openProblemReview(row, 'approved')">通过</el-button>
              <el-button size="small" type="danger" text @click="openProblemReview(row, 'rejected')">退回</el-button>
            </template>
            <span v-else class="muted">已处理</span>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog v-model="reviewVisible" :title="reviewDialogTitle" width="540px">
      <el-form label-position="top">
        <el-form-item :label="reviewForm.kind === 'application' ? '申请人' : '题目'">
          <el-input :model-value="reviewForm.label" disabled />
        </el-form-item>
        <el-form-item label="审核说明">
          <el-input
            v-model="reviewForm.review_note"
            type="textarea"
            :rows="5"
            :placeholder="reviewForm.status === 'approved' ? '可选：填写审核备注' : '请填写具体退回原因，出题者将在草稿页看到该说明'"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reviewVisible = false">取消</el-button>
        <el-button :type="reviewForm.status === 'approved' ? 'success' : 'danger'" :loading="reviewing" @click="submitReview">
          确认{{ reviewForm.status === 'approved' ? '通过' : reviewForm.kind === 'problem' ? '退回' : '驳回' }}
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import remindIcon from '../../assets/remind-fill.svg'
import { client, type AuthorApplication, type ProblemReview } from '../../api/client'
import ProblemTestDownloads from '../../components/ProblemTestDownloads.vue'
import { formatDateTime } from '../../features/time'

type ReviewDecision = 'approved' | 'rejected'

const router = useRouter()
const loading = ref(false)
const reviewing = ref(false)
const reviewVisible = ref(false)
const activePanel = ref<'people' | 'problems'>('people')
const problemStatusFilter = ref<'all' | ProblemReview['status']>('pending')
const applications = ref<AuthorApplication[]>([])
const problemReviews = ref<ProblemReview[]>([])
const reviewForm = reactive({
  kind: 'application' as 'application' | 'problem',
  id: 0,
  label: '',
  status: 'approved' as ReviewDecision,
  review_note: ''
})

const pendingApplications = computed(() => applications.value.filter((item) => item.status === 'pending'))
const filteredProblemReviews = computed(() => problemStatusFilter.value === 'all'
  ? problemReviews.value
  : problemReviews.value.filter((item) => item.status === problemStatusFilter.value))
const reviewDialogTitle = computed(() => {
  if (reviewForm.kind === 'application') return reviewForm.status === 'approved' ? '通过出题资格申请' : '驳回出题资格申请'
  return reviewForm.status === 'approved' ? '通过题目审批' : '退回题目'
})

async function load() {
  loading.value = true
  try {
    const [applicationResponse, problemResponse] = await Promise.all([
      client.get('/author-applications'),
      client.get('/problem-reviews')
    ])
    applications.value = applicationResponse.data
    problemReviews.value = problemResponse.data
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function openApplicationReview(row: AuthorApplication, status: ReviewDecision) {
  reviewForm.kind = 'application'
  reviewForm.id = row.id
  reviewForm.label = `${row.user?.name || `用户 #${row.user_id}`} <${row.user?.email || '-'}>`
  reviewForm.status = status
  reviewForm.review_note = ''
  reviewVisible.value = true
}

function openProblemReview(row: ProblemReview, status: ReviewDecision) {
  reviewForm.kind = 'problem'
  reviewForm.id = row.id
  reviewForm.label = `${row.problem.display_code || `#${row.problem_id}`} · ${row.problem.title}`
  reviewForm.status = status
  reviewForm.review_note = ''
  reviewVisible.value = true
}

async function submitReview() {
  if (reviewForm.status === 'rejected' && !reviewForm.review_note.trim()) {
    ElMessage.error('驳回或退回时必须填写审核说明')
    return
  }
  reviewing.value = true
  try {
    const endpoint = reviewForm.kind === 'application'
      ? `/author-applications/${reviewForm.id}/review`
      : `/problem-reviews/${reviewForm.id}/review`
    await client.put(endpoint, {
      status: reviewForm.status,
      review_note: reviewForm.review_note.trim()
    })
    reviewVisible.value = false
    ElMessage.success(reviewForm.status === 'approved' ? '审核已通过' : reviewForm.kind === 'problem' ? '题目已退回并保留出题草稿' : '申请已驳回')
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    reviewing.value = false
  }
}

function statusText(status: string) {
  if (status === 'approved') return '已通过'
  if (status === 'rejected') return '已退回'
  return '待审批'
}

function statusType(status: string): 'success' | 'danger' | 'warning' {
  if (status === 'approved') return 'success'
  if (status === 'rejected') return 'danger'
  return 'warning'
}

function roleText(role?: string) {
  if (role === 'admin') return '管理员'
  if (role === 'teacher') return '教师'
  if (role === 'problem_setter') return '出题者'
  return '学生'
}

function problemTags(tags?: Record<string, unknown>) {
  const labels = tags?.labels
  return Array.isArray(labels) ? labels.map(String) : []
}

onMounted(load)
</script>

<style scoped>
.author-management-page { padding: 28px; }
.management-header, .section-heading, .header-actions, .switch-panel, .reminder-popover header, .inline-actions { display: flex; align-items: center; }
.management-header, .section-heading, .switch-panel, .reminder-popover header { justify-content: space-between; }
.management-header { gap: 24px; margin-bottom: 18px; }
.management-header h2 { margin: 5px 0; }
.management-header p, .section-heading p { margin: 0; color: var(--muted); }
.eyebrow { color: var(--accent); font-size: 11px; font-weight: 800; letter-spacing: .14em; }
.header-actions { gap: 12px; }
.reminder-button { display: grid; width: 42px; height: 42px; place-items: center; color: #dc2626; border: 1px solid color-mix(in srgb, #dc2626 30%, var(--border)); border-radius: 11px; background: color-mix(in srgb, #dc2626 8%, var(--surface)); cursor: pointer; }
.reminder-icon { width: 24px; height: 24px; background: currentColor; mask-repeat: no-repeat; mask-position: center; mask-size: contain; }
.reminder-popover header { padding-bottom: 12px; border-bottom: 1px solid var(--border); }
.reminder-popover header div { display: grid; gap: 3px; }
.reminder-popover header span { color: var(--muted); font-size: 12px; }
.pending-list { display: grid; gap: 10px; max-height: 440px; overflow: auto; padding-top: 12px; }
.pending-list article { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 12px; border: 1px solid var(--border); border-radius: 10px; }
.pending-list article > div:first-child { min-width: 0; }
.pending-list article span { display: block; color: var(--muted); font-size: 12px; }
.pending-list article p { overflow: hidden; margin: 6px 0 0; color: var(--text); text-overflow: ellipsis; white-space: nowrap; }
.inline-actions { flex: 0 0 auto; gap: 6px; }
.switch-panel { gap: 20px; margin-bottom: 18px; padding: 14px 18px; }
.switch-panel > span { color: var(--muted); font-size: 13px; }
.content-panel { padding: 18px; }
.section-heading { gap: 16px; margin-bottom: 16px; }
.section-heading h3 { margin: 0 0 4px; }
.tag-list { display: flex; flex-wrap: wrap; gap: 5px; }
.review-note { overflow: hidden; max-width: 260px; margin-top: 4px; text-overflow: ellipsis; white-space: nowrap; }
@media (max-width: 760px) {
  .author-management-page { padding: 18px 12px; }
  .management-header, .switch-panel, .section-heading { align-items: stretch; flex-direction: column; }
  .switch-panel > span { line-height: 1.6; }
}
</style>
