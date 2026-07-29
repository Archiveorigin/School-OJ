<template>
  <section class="page author-management-page">
    <div class="page-header management-header">
      <div>
        <span class="eyebrow">AUTHORING CONTROL</span>
        <h2>出题管理</h2>
      </div>
      <div class="header-actions">
        <button type="button" class="pending-review-button" aria-label="查看待审批出题人员" @click="applicationDialogVisible = true">
          <svg class="message-icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
            <path
              d="M523.946667 85.333333C802.773333 85.333333 982.826667 307.541333 982.826667 511.338667c0 242.837333-197.397333 448-458.837334 448-84.16 0-150.464-16.597333-221.610666-55.402667l-88.618667 50.197333c-27.797333 8.426667-50.474667 5.162667-68.010667-9.834666-17.557333-14.997333-25.130667-34.730667-22.72-59.2a19570.688 19570.688 0 0 0 29.653334-106.368C123.008 741.76 64 656.426667 64 511.338667 64 307.541333 245.141333 85.333333 523.946667 85.333333z m-1.28 64C304.064 149.333333 128 317.12 128 522.666667c0 77.354667 24.874667 151.125333 70.634667 213.184l5.397333 7.125333 18.218667 23.509333-36.970667 128.746667a0.32 0.32 0 0 0 0.490667 0.384l113.408-63.829333 26.752 14.592C385.237333 878.72 452.544 896 522.666667 896 741.269333 896 917.333333 728.213333 917.333333 522.666667S741.269333 149.333333 522.666667 149.333333z m-192 320a53.333333 53.333333 0 1 1 0 106.666667 53.333333 53.333333 0 0 1 0-106.666667z m182.848 0a53.333333 53.333333 0 1 1 0 106.666667 53.333333 53.333333 0 0 1 0-106.666667z m182.869333 0a53.333333 53.333333 0 1 1 0 106.666667 53.333333 53.333333 0 0 1 0-106.666667z"
            />
          </svg>
          <span class="pending-button-copy">
            <strong>待审批人员</strong>
            <small>{{ pendingApplications.length ? '有新的资格申请' : '暂无待处理申请' }}</small>
          </span>
          <span class="pending-count">{{ pendingApplications.length }}</span>
        </button>
        <el-button :loading="loading" @click="load">刷新</el-button>
      </div>
    </div>

    <div class="panel switch-panel">
      <el-radio-group v-model="activePanel" size="large">
        <el-radio-button value="people">人员管理</el-radio-button>
        <el-radio-button value="problems">题目管理</el-radio-button>
      </el-radio-group>
      <span v-if="activePanel === 'people'">仅展示已获出题权限的人员；基础角色仍为学生、教师或管理员。</span>
      <span v-else>非管理员提交的题目通过审批后进入公共题库，也可在发布后撤销。</span>
    </div>

    <section v-if="activePanel === 'people'" class="panel content-panel people-panel">
      <div class="section-heading">
        <div>
          <h3>出题人员</h3>
          <p>管理员为系统内置；其他已授权人员可移除出题权限，不改变其基础角色。</p>
        </div>
      </div>

      <div class="people-summary">
        <article>
          <span>全部出题人员</span>
          <strong>{{ authors.length }}</strong>
          <small>当前拥有出题权限</small>
        </article>
        <article>
          <span>系统管理员</span>
          <strong>{{ adminAuthorCount }}</strong>
          <small>内置权限，不可移除</small>
        </article>
        <article>
          <span>独立授权人员</span>
          <strong>{{ removableAuthorCount }}</strong>
          <small>可单独收回出题权限</small>
        </article>
      </div>

      <div v-loading="loading" class="author-table">
        <div class="author-table-head" aria-hidden="true">
          <span>人员</span>
          <span>角色与权限</span>
          <span>授权记录</span>
          <span>操作</span>
        </div>
        <el-empty v-if="!loading && authors.length === 0" description="暂无出题者人员" />
        <article v-for="author in authors" :key="author.id" class="author-row">
          <div class="person-identity">
            <span class="person-avatar">{{ author.name.trim().slice(0, 1).toUpperCase() }}</span>
            <div>
              <strong>{{ author.name }}</strong>
              <span>{{ author.email }}</span>
            </div>
          </div>
          <div class="role-access">
            <el-tag :type="author.role === 'admin' ? 'danger' : author.role === 'teacher' ? 'warning' : 'info'" effect="plain">
              {{ roleText(author.role) }}
            </el-tag>
            <span><i />已启用出题权限</span>
          </div>
          <div class="authorization-record">
            <strong>{{ authorizationTime(author) }}</strong>
            <span>{{ latestApproval(author.id)?.review_note || (author.role === 'admin' ? '系统内置权限' : '历史授权') }}</span>
          </div>
          <div class="author-action">
            <el-tag v-if="author.role === 'admin'" type="info" effect="plain">内置权限</el-tag>
            <el-button v-else type="danger" plain :loading="removingAuthorID === author.id" @click="removeAuthor(author)"> 删除 </el-button>
          </div>
        </article>
      </div>
    </section>

    <section v-else class="panel content-panel">
      <div class="section-heading problem-heading">
        <div>
          <h3>题目审批</h3>
          <p>核对题面属性与全部后台测试点后，决定发布、退回或撤销题目。</p>
        </div>
        <el-select v-model="problemStatusFilter" class="status-filter">
          <el-option label="全部状态" value="all" />
          <el-option label="待审批" value="pending" />
          <el-option label="已通过" value="approved" />
          <el-option label="已退回" value="rejected" />
          <el-option label="已撤销" value="withdrawn" />
        </el-select>
      </div>

      <div v-loading="loading" class="entity-list problem-list">
        <el-empty v-if="!loading && filteredProblemReviews.length === 0" description="暂无题目审批记录" />
        <article v-for="row in filteredProblemReviews" :key="row.id" class="entity-card problem-card">
          <header class="problem-card-header">
            <div class="problem-title">
              <span>{{ row.problem.display_code || `#${row.problem_id}` }}</span>
              <el-button link type="primary" @click="router.push(`/problems/${row.problem.display_code || row.problem_id}`)">
                {{ row.problem.title }}
              </el-button>
              <p v-if="row.review_note">{{ row.review_note }}</p>
            </div>
            <div class="card-actions problem-actions">
              <template v-if="row.status === 'pending'">
                <el-button type="success" plain @click="openProblemReview(row, 'approved')">通过</el-button>
                <el-button type="danger" plain @click="openProblemReview(row, 'rejected')">退回</el-button>
              </template>
              <el-button
                v-else-if="row.status === 'approved'"
                type="danger"
                plain
                :loading="withdrawingReviewID === row.id"
                @click="withdrawProblem(row)"
              >
                撤销题目
              </el-button>
              <span v-else class="muted">{{ row.status === 'withdrawn' ? '题目已撤销' : '题目已退回' }}</span>
            </div>
          </header>

          <dl class="card-facts problem-facts">
            <div>
              <dt>难度</dt>
              <dd>
                <el-tag effect="plain">{{ row.problem.difficulty || '未设置' }}</el-tag>
              </dd>
            </div>
            <div>
              <dt>标签</dt>
              <dd class="tag-list">
                <el-tag v-for="tag in problemTags(row.problem.tags)" :key="tag" size="small" type="info">{{ tag }}</el-tag>
                <span v-if="!problemTags(row.problem.tags).length" class="muted">-</span>
              </dd>
            </div>
            <div>
              <dt>测试点</dt>
              <dd>
                <ProblemTestDownloads
                  :problem-id="row.problem_id"
                  :problem-code="row.problem.display_code"
                  :button-label="`${row.test_point_count} 个测试点`"
                  :dialog-title="`题目测试点 · ${row.problem.title}`"
                />
              </dd>
            </div>
            <div>
              <dt>出题人</dt>
              <dd>{{ row.author?.name || `用户 #${row.author_id}` }}</dd>
            </div>
            <div>
              <dt>提交时间</dt>
              <dd>{{ formatDateTime(row.submitted_at) }}</dd>
            </div>
            <div>
              <dt>状态</dt>
              <dd>
                <el-tag :type="statusType(row.status)">{{ statusText(row.status) }}</el-tag>
              </dd>
            </div>
          </dl>
        </article>
      </div>
    </section>

    <el-dialog v-model="applicationDialogVisible" width="min(720px, calc(100vw - 24px))" append-to-body align-center destroy-on-close>
      <template #header>
        <div class="dialog-title">
          <span class="dialog-icon">
            <svg class="message-icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
              <path
                d="M523.946667 85.333333C802.773333 85.333333 982.826667 307.541333 982.826667 511.338667c0 242.837333-197.397333 448-458.837334 448-84.16 0-150.464-16.597333-221.610666-55.402667l-88.618667 50.197333c-27.797333 8.426667-50.474667 5.162667-68.010667-9.834666-17.557333-14.997333-25.130667-34.730667-22.72-59.2a19570.688 19570.688 0 0 0 29.653334-106.368C123.008 741.76 64 656.426667 64 511.338667 64 307.541333 245.141333 85.333333 523.946667 85.333333z m-1.28 64C304.064 149.333333 128 317.12 128 522.666667c0 77.354667 24.874667 151.125333 70.634667 213.184l5.397333 7.125333 18.218667 23.509333-36.970667 128.746667a0.32 0.32 0 0 0 0.490667 0.384l113.408-63.829333 26.752 14.592C385.237333 878.72 452.544 896 522.666667 896 741.269333 896 917.333333 728.213333 917.333333 522.666667S741.269333 149.333333 522.666667 149.333333z m-192 320a53.333333 53.333333 0 1 1 0 106.666667 53.333333 53.333333 0 0 1 0-106.666667z m182.848 0a53.333333 53.333333 0 1 1 0 106.666667 53.333333 53.333333 0 0 1 0-106.666667z m182.869333 0a53.333333 53.333333 0 1 1 0 106.666667 53.333333 53.333333 0 0 1 0-106.666667z"
              />
            </svg>
          </span>
          <div>
            <strong>待审批出题人员</strong>
            <span>{{ pendingApplications.length }} 条待处理申请</span>
          </div>
        </div>
      </template>
      <div class="pending-list">
        <el-empty v-if="pendingApplications.length === 0" :image-size="72" description="暂无待处理申请" />
        <article v-for="item in pendingApplications" :key="item.id">
          <div class="pending-copy">
            <strong>{{ item.user?.name || `用户 #${item.user_id}` }}</strong>
            <span>{{ item.user?.email }} · {{ roleText(item.user?.role) }}</span>
            <p>{{ item.motivation }}</p>
            <small>申请于 {{ formatDateTime(item.created_at) }}</small>
          </div>
          <div class="inline-actions">
            <el-button type="success" plain @click="openApplicationReview(item, 'approved')">通过</el-button>
            <el-button type="danger" plain @click="openApplicationReview(item, 'rejected')">驳回</el-button>
          </div>
        </article>
      </div>
    </el-dialog>

    <el-dialog v-model="reviewVisible" :title="reviewDialogTitle" width="min(540px, calc(100vw - 24px))" append-to-body align-center>
      <el-form label-position="top">
        <el-form-item :label="reviewForm.kind === 'application' ? '申请人' : '题目'">
          <el-input :model-value="reviewForm.label" disabled />
        </el-form-item>
        <el-form-item label="审核说明">
          <el-input
            v-model="reviewForm.review_note"
            type="textarea"
            :rows="5"
            :placeholder="reviewForm.status === 'approved' ? '可选：填写审核备注' : '请填写具体退回原因，申请人或出题者将看到该说明'"
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { client, type AuthorApplication, type ProblemReview, type User } from '../../api/client'
import ProblemTestDownloads from '../../components/ProblemTestDownloads.vue'
import { formatDateTime } from '../../features/time'

type ReviewDecision = 'approved' | 'rejected'

const router = useRouter()
const loading = ref(false)
const reviewing = ref(false)
const reviewVisible = ref(false)
const applicationDialogVisible = ref(false)
const removingAuthorID = ref(0)
const withdrawingReviewID = ref(0)
const activePanel = ref<'people' | 'problems'>('people')
const problemStatusFilter = ref<'all' | ProblemReview['status']>('pending')
const authors = ref<User[]>([])
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
const adminAuthorCount = computed(() => authors.value.filter((item) => item.role === 'admin').length)
const removableAuthorCount = computed(() => authors.value.length - adminAuthorCount.value)
const filteredProblemReviews = computed(() =>
  problemStatusFilter.value === 'all'
    ? problemReviews.value
    : problemReviews.value.filter((item) => item.status === problemStatusFilter.value)
)
const reviewDialogTitle = computed(() => {
  if (reviewForm.kind === 'application') return reviewForm.status === 'approved' ? '通过出题资格申请' : '驳回出题资格申请'
  return reviewForm.status === 'approved' ? '通过题目审批' : '退回题目'
})

async function load() {
  loading.value = true
  try {
    const [authorResponse, applicationResponse, problemResponse] = await Promise.all([
      client.get<User[]>('/problem-authors'),
      client.get<AuthorApplication[]>('/author-applications'),
      client.get<ProblemReview[]>('/problem-reviews')
    ])
    authors.value = authorResponse.data
    applications.value = applicationResponse.data
    problemReviews.value = problemResponse.data
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    loading.value = false
  }
}

function latestApproval(userID: number) {
  return applications.value.find((item) => item.user_id === userID && item.status === 'approved')
}

function authorizationTime(author: User) {
  if (author.role === 'admin') return '系统内置'
  const approval = latestApproval(author.id)
  const value = approval?.reviewed_at || approval?.updated_at
  return value ? formatDateTime(value) : '历史授权'
}

async function removeAuthor(author: User) {
  try {
    await ElMessageBox.confirm(`确认移除“${author.name}”的出题权限？该用户的基础角色和既有题目不会被删除。`, '删除出题者人员', {
      type: 'warning',
      confirmButtonText: '确认删除',
      cancelButtonText: '取消'
    })
    removingAuthorID.value = author.id
    await client.delete(`/problem-authors/${author.id}`)
    ElMessage.success('已移除出题权限')
    await load()
  } catch (err: any) {
    if (err === 'cancel' || err === 'close') return
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    removingAuthorID.value = 0
  }
}

function openApplicationReview(row: AuthorApplication, status: ReviewDecision) {
  applicationDialogVisible.value = false
  reviewForm.kind = 'application'
  reviewForm.id = row.id
  reviewForm.label = row.user?.name || `用户 #${row.user_id}`
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
    const endpoint =
      reviewForm.kind === 'application' ? `/author-applications/${reviewForm.id}/review` : `/problem-reviews/${reviewForm.id}/review`
    await client.put(endpoint, {
      status: reviewForm.status,
      review_note: reviewForm.review_note.trim()
    })
    reviewVisible.value = false
    ElMessage.success(
      reviewForm.status === 'approved' ? '审核已通过' : reviewForm.kind === 'problem' ? '题目已退回并保留出题草稿' : '申请已驳回'
    )
    await load()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    reviewing.value = false
  }
}

async function withdrawProblem(row: ProblemReview) {
  try {
    await ElMessageBox.confirm(`确认撤销题目“${row.problem.title}”？撤销后该题目将不再显示于公共题库。`, '撤销题目', {
      type: 'warning',
      confirmButtonText: '确认撤销',
      cancelButtonText: '取消'
    })
    withdrawingReviewID.value = row.id
    await client.put(`/problem-reviews/${row.id}/withdraw`)
    ElMessage.success('题目已撤销')
    await load()
  } catch (err: any) {
    if (err === 'cancel' || err === 'close') return
    ElMessage.error(err.response?.data?.error || err.message)
  } finally {
    withdrawingReviewID.value = 0
  }
}

function statusText(status: ProblemReview['status']) {
  if (status === 'approved') return '已通过'
  if (status === 'rejected') return '已退回'
  if (status === 'withdrawn') return '已撤销'
  return '待审批'
}

function statusType(status: ProblemReview['status']): 'success' | 'danger' | 'warning' | 'info' {
  if (status === 'approved') return 'success'
  if (status === 'rejected') return 'danger'
  if (status === 'withdrawn') return 'info'
  return 'warning'
}

function roleText(role?: string) {
  if (role === 'admin') return '管理员'
  if (role === 'teacher') return '教师'
  return '学生'
}

function problemTags(tags?: Record<string, unknown>) {
  const labels = tags?.labels
  return Array.isArray(labels) ? labels.map(String) : []
}

onMounted(load)
</script>

<style scoped>
.author-management-page {
  padding: 28px;
}
.management-header,
.section-heading,
.header-actions,
.switch-panel,
.inline-actions,
.problem-card-header,
.dialog-title {
  display: flex;
  align-items: center;
}
.management-header,
.section-heading,
.switch-panel,
.problem-card-header {
  justify-content: space-between;
}
.management-header {
  gap: 24px;
  margin-bottom: 18px;
}
.management-header h2 {
  margin: 5px 0;
}
.management-header p,
.section-heading p {
  margin: 0;
  color: var(--muted);
}
.eyebrow {
  color: var(--accent);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.14em;
}
.header-actions {
  gap: 12px;
}
.pending-review-button {
  display: flex;
  align-items: center;
  min-height: 46px;
  gap: 11px;
  padding: 7px 10px 7px 13px;
  color: var(--text);
  border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
  border-radius: 13px;
  background: color-mix(in srgb, var(--accent) 6%, var(--surface));
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    background 0.2s ease,
    transform 0.2s ease;
}
.pending-review-button:hover {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 11%, var(--surface));
  transform: translateY(-1px);
}
.message-icon {
  display: block;
  width: 25px;
  height: 25px;
  flex: 0 0 auto;
  fill: currentColor;
}
.pending-review-button > .message-icon {
  color: var(--accent);
}
.pending-button-copy {
  display: grid;
  min-width: 0;
  gap: 1px;
  text-align: left;
}
.pending-button-copy strong {
  font-size: 13px;
  line-height: 1.3;
}
.pending-button-copy small {
  color: var(--muted);
  font-size: 11px;
  white-space: nowrap;
}
.pending-count {
  display: grid;
  min-width: 25px;
  height: 25px;
  padding: 0 6px;
  place-items: center;
  color: #fff;
  border-radius: 999px;
  background: var(--accent);
  font-size: 12px;
  font-weight: 800;
}
.switch-panel {
  gap: 20px;
  margin-bottom: 18px;
  padding: 14px 18px;
}
.switch-panel > span {
  color: var(--muted);
  font-size: 13px;
}
.content-panel {
  padding: 18px;
  overflow: hidden;
}
.section-heading {
  gap: 16px;
  margin-bottom: 16px;
}
.section-heading h3 {
  margin: 0 0 4px;
}
.people-panel {
  padding: 0;
}
.people-panel .section-heading {
  margin: 0;
  padding: 22px 24px 18px;
}
.people-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  background: var(--border);
}
.people-summary article {
  display: grid;
  gap: 5px;
  padding: 18px 24px;
  background: color-mix(in srgb, var(--surface) 97%, var(--app-bg));
}
.people-summary span {
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
}
.people-summary strong {
  color: var(--text);
  font-size: 28px;
  line-height: 1;
}
.people-summary small {
  color: var(--muted);
  font-size: 11px;
}
.author-table {
  min-height: 180px;
  padding: 0 24px 14px;
}
.author-table-head,
.author-row {
  display: grid;
  grid-template-columns:
    minmax(220px, 1.35fr) minmax(150px, 0.75fr) minmax(240px, 1.2fr)
    88px;
  align-items: center;
  gap: 22px;
}
.author-table-head {
  min-height: 46px;
  color: var(--muted);
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  font-weight: 700;
}
.author-row {
  min-width: 0;
  min-height: 82px;
  border-bottom: 1px solid color-mix(in srgb, var(--border) 82%, transparent);
}
.author-row:last-child {
  border-bottom: 0;
}
.status-filter {
  width: 150px;
}
.entity-list {
  display: grid;
  gap: 12px;
  min-height: 120px;
}
.entity-card {
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: color-mix(in srgb, var(--surface) 95%, var(--app-bg));
}
.person-identity {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 12px;
}
.person-avatar {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  place-items: center;
  color: #fff;
  border-radius: 50%;
  background: linear-gradient(145deg, var(--accent), color-mix(in srgb, var(--accent) 72%, #111827));
  box-shadow: 0 5px 14px color-mix(in srgb, var(--accent) 22%, transparent);
  font-weight: 800;
}
.person-identity > div {
  min-width: 0;
}
.person-identity strong,
.person-identity span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.person-identity span {
  margin-top: 3px;
  color: var(--muted);
  font-size: 12px;
}
.role-access {
  display: grid;
  justify-items: start;
  gap: 7px;
}
.role-access > span {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 11px;
  white-space: nowrap;
}
.role-access i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 0 3px color-mix(in srgb, #22c55e 16%, transparent);
}
.authorization-record {
  display: grid;
  min-width: 0;
  gap: 5px;
}
.authorization-record strong {
  color: var(--text);
  font-size: 13px;
  font-weight: 700;
}
.authorization-record span {
  overflow: hidden;
  color: var(--muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.author-action {
  display: flex;
  justify-content: flex-end;
}
.card-facts {
  display: grid;
  margin: 0;
  gap: 14px;
}
.card-facts > div {
  min-width: 0;
}
.card-facts dt {
  margin-bottom: 5px;
  color: var(--muted);
  font-size: 12px;
}
.card-facts dd {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
}
.card-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex: 0 0 auto;
  gap: 8px;
}
.problem-card-header {
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border);
}
.problem-title {
  min-width: 0;
}
.problem-title > span {
  display: block;
  color: var(--muted);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.06em;
}
.problem-title .el-button {
  max-width: 100%;
  height: auto;
  padding: 5px 0 0;
  font-size: 17px;
  font-weight: 800;
  white-space: normal;
  text-align: left;
}
.problem-title p {
  margin: 7px 0 0;
  color: var(--muted);
  font-size: 13px;
  overflow-wrap: anywhere;
}
.problem-facts {
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
  padding-top: 14px;
}
.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}
.dialog-title {
  gap: 12px;
}
.dialog-icon {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  place-items: center;
  color: var(--accent);
  border-radius: 12px;
  background: color-mix(in srgb, var(--accent) 12%, var(--surface));
}
.dialog-icon .message-icon {
  width: 24px;
  height: 24px;
}
.dialog-title > div {
  display: grid;
  gap: 3px;
}
.dialog-title strong {
  font-size: 17px;
}
.dialog-title span:last-child {
  color: var(--muted);
  font-size: 12px;
}
.pending-list {
  display: grid;
  gap: 10px;
}
.pending-list article {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
  gap: 16px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
}
.pending-copy {
  min-width: 0;
}
.pending-copy strong,
.pending-copy span,
.pending-copy small {
  display: block;
}
.pending-copy span,
.pending-copy small {
  color: var(--muted);
  font-size: 12px;
}
.pending-copy p {
  margin: 7px 0;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.inline-actions {
  flex: 0 0 auto;
  gap: 6px;
}

@media (max-width: 980px) {
  .author-table-head,
  .author-row {
    grid-template-columns:
      minmax(210px, 1.25fr) minmax(145px, 0.75fr) minmax(190px, 1fr)
      76px;
    gap: 14px;
  }
}

@media (max-width: 760px) {
  .author-management-page {
    padding: 18px 12px;
  }
  .management-header,
  .switch-panel,
  .section-heading {
    align-items: stretch;
    flex-direction: column;
  }
  .header-actions {
    justify-content: flex-end;
  }
  .pending-review-button {
    flex: 1 1 auto;
  }
  .switch-panel > span {
    line-height: 1.6;
  }
  .people-summary {
    grid-template-columns: 1fr;
  }
  .people-summary article {
    padding: 15px 18px;
  }
  .author-table {
    padding: 0 18px 12px;
  }
  .author-table-head {
    display: none;
  }
  .author-row {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 14px;
    padding: 18px 0;
  }
  .role-access {
    align-items: end;
    justify-items: end;
  }
  .authorization-record {
    grid-column: 1 / -1;
    padding: 12px 14px;
    border-radius: 10px;
    background: color-mix(in srgb, var(--app-bg) 74%, transparent);
  }
  .author-action {
    grid-column: 1 / -1;
    justify-content: stretch;
  }
  .author-action .el-button {
    width: 100%;
  }
  .status-filter {
    width: 100%;
  }
  .problem-actions {
    width: 100%;
    justify-content: flex-start;
  }
  .pending-list article {
    align-items: stretch;
    flex-direction: column;
  }
  .inline-actions {
    justify-content: flex-end;
  }
}

@media (max-width: 480px) {
  .header-actions {
    align-items: stretch;
    flex-direction: column;
    width: 100%;
  }
  .pending-button-copy {
    flex: 1;
  }
}
</style>
