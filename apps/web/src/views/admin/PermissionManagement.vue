<template>
  <section class="permission-page" v-loading="loading">
    <div class="permission-heading">
      <div>
        <h2>权限管理</h2>
        <p>附加权限不会覆盖学生、教师或管理员等基础角色。</p>
      </div>
      <el-button :loading="loading" @click="load">刷新</el-button>
    </div>

    <section class="permission-summary" aria-label="权限概况">
      <article>
        <span>权限类型</span><strong>{{ definitions.length }}</strong
        ><small>预留统一扩展入口</small>
      </article>
      <article>
        <span>出题者</span><strong>{{ authorCount }}</strong
        ><small>含系统管理员</small>
      </article>
      <article>
        <span>待审核申请</span><strong>{{ pendingApplications.length }}</strong
        ><small>来自基础角色用户</small>
      </article>
    </section>

    <section class="panel permission-definition">
      <div class="definition-mark">A</div>
      <div>
        <span>GLOBAL PERMISSION</span>
        <h3>{{ authorPermission?.name || '出题者' }}</h3>
        <p>
          {{ authorPermission?.description || '可发起题目数据工单，与基础角色并行。' }}
        </p>
      </div>
      <div class="definition-flow"><span>基础角色</span><i>+</i><strong>出题者标识</strong></div>
    </section>

    <section v-if="pendingApplications.length" class="panel application-panel">
      <div class="section-heading">
        <div>
          <h3>待审核申请</h3>
          <p>通过后只添加出题者标识，不改变申请人的基础角色。</p>
        </div>
        <el-tag type="warning">{{ pendingApplications.length }} 条</el-tag>
      </div>
      <div class="application-list">
        <article v-for="application in pendingApplications" :key="application.id">
          <div class="person">
            <span>{{ avatarText(application.user) }}</span>
            <div>
              <strong>{{ application.user?.name || `用户 #${application.user_id}` }}</strong
              ><small>{{ application.user?.email }} · {{ roleText(application.user?.role) }}</small>
            </div>
          </div>
          <p>{{ application.motivation }}</p>
          <time>{{ formatDateTime(application.created_at) }}</time>
          <div>
            <el-button type="danger" plain @click="reviewApplication(application, 'rejected')"
              >驳回</el-button
            ><el-button type="success" @click="reviewApplication(application, 'approved')"
              >授予权限</el-button
            >
          </div>
        </article>
      </div>
    </section>

    <section class="panel people-panel">
      <div class="section-heading">
        <div>
          <h3>人员权限</h3>
          <p>统一查看基础角色和附加权限，便于后续继续增加新的权限类型。</p>
        </div>
        <div class="filters">
          <el-input v-model="keyword" clearable placeholder="搜索姓名或邮箱" /><el-select
            v-model="permissionFilter"
            ><el-option label="全部人员" value="all" /><el-option
              label="仅出题者"
              value="authors" /><el-option label="未授权" value="without"
          /></el-select>
        </div>
      </div>
      <el-table class="permission-table" :data="filteredUsers" row-key="id">
        <el-table-column label="用户" min-width="230"
          ><template #default="{ row }"
            ><div class="table-person">
              <span>{{ avatarText(row) }}</span>
              <div>
                <strong>{{ row.name }}</strong
                ><small>{{ row.email }}</small>
              </div>
            </div></template
          ></el-table-column
        >
        <el-table-column label="基础角色" width="130"
          ><template #default="{ row }"
            ><el-tag :type="roleTagType(row.role)" effect="plain">{{
              roleText(row.role)
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column label="附加权限" min-width="210"
          ><template #default="{ row }"
            ><div class="permission-tags">
              <el-tag v-if="hasAuthorPermission(row)" type="success">出题者</el-tag
              ><span v-else>暂无附加权限</span>
            </div></template
          ></el-table-column
        >
        <el-table-column label="权限说明" min-width="250"
          ><template #default="{ row }">{{
            row.role === 'admin'
              ? '管理员内置出题权限，不可收回'
              : hasAuthorPermission(row)
                ? '可发起题目新增、覆盖修改与覆盖删除工单'
                : '保留当前基础角色，可申请或由管理员授予'
          }}</template></el-table-column
        >
        <el-table-column label="操作" width="150" align="right"
          ><template #default="{ row }"
            ><el-tag v-if="row.role === 'admin'" type="info" effect="plain">系统内置</el-tag
            ><el-button
              v-else-if="hasAuthorPermission(row)"
              type="danger"
              plain
              :loading="updatingUserID === row.id"
              @click="updateAuthorPermission(row, false)"
              >收回权限</el-button
            ><el-button
              v-else
              type="primary"
              plain
              :loading="updatingUserID === row.id"
              @click="updateAuthorPermission(row, true)"
              >授予权限</el-button
            ></template
          ></el-table-column
        >
      </el-table>
      <div class="permission-mobile-list">
        <article v-for="row in filteredUsers" :key="row.id">
          <header>
            <div class="table-person">
              <span>{{ avatarText(row) }}</span>
              <div><strong>{{ row.name }}</strong><small>{{ row.email }}</small></div>
            </div>
            <el-tag :type="roleTagType(row.role)" effect="plain">{{ roleText(row.role) }}</el-tag>
          </header>
          <dl>
            <div>
              <dt>附加权限</dt>
              <dd><el-tag v-if="hasAuthorPermission(row)" type="success">出题者</el-tag><span v-else>暂无附加权限</span></dd>
            </div>
            <div>
              <dt>权限说明</dt>
              <dd>{{ row.role === 'admin' ? '管理员内置出题权限，不可收回' : hasAuthorPermission(row) ? '可发起题目新增、覆盖修改与覆盖删除工单' : '保留当前基础角色，可申请或由管理员授予' }}</dd>
            </div>
          </dl>
          <el-tag v-if="row.role === 'admin'" type="info" effect="plain">系统内置</el-tag>
          <el-button v-else-if="hasAuthorPermission(row)" type="danger" plain :loading="updatingUserID === row.id" @click="updateAuthorPermission(row, false)">收回权限</el-button>
          <el-button v-else type="primary" plain :loading="updatingUserID === row.id" @click="updateAuthorPermission(row, true)">授予权限</el-button>
        </article>
      </div>
      <el-empty v-if="!filteredUsers.length" description="没有符合条件的人员" />
    </section>
  </section>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import {
  client,
  type AuthorApplication,
  type Role,
  type User,
  type UserPermissionDefinition
} from '../../api/client'
import { formatDateTime } from '../../features/time'

const loading = ref(false)
const updatingUserID = ref(0)
const users = ref<User[]>([])
const applications = ref<AuthorApplication[]>([])
const definitions = ref<UserPermissionDefinition[]>([])
const keyword = ref('')
const permissionFilter = ref<'all' | 'authors' | 'without'>('all')
const authorPermission = computed(() =>
  definitions.value.find((item) => item.key === 'problem_author')
)
const pendingApplications = computed(() =>
  applications.value.filter((item) => item.status === 'pending')
)
const authorCount = computed(() => users.value.filter(hasAuthorPermission).length)
const filteredUsers = computed(() => {
  const query = keyword.value.trim().toLocaleLowerCase()
  return users.value.filter((user) => {
    const enabled = hasAuthorPermission(user)
    if (permissionFilter.value === 'authors' && !enabled) return false
    if (permissionFilter.value === 'without' && enabled) return false
    return (
      !query ||
      user.name.toLocaleLowerCase().includes(query) ||
      user.email.toLocaleLowerCase().includes(query)
    )
  })
})

async function load() {
  loading.value = true
  try {
    const [userResponse, applicationResponse, permissionResponse] = await Promise.all([
      client.get<User[]>('/users'),
      client.get<AuthorApplication[]>('/author-applications'),
      client.get<UserPermissionDefinition[]>('/permissions')
    ])
    users.value = userResponse.data || []
    applications.value = applicationResponse.data || []
    definitions.value = permissionResponse.data || []
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || error.message)
  } finally {
    loading.value = false
  }
}

async function reviewApplication(application: AuthorApplication, status: 'approved' | 'rejected') {
  try {
    const result = await ElMessageBox.prompt(
      status === 'approved' ? '可填写授权说明' : '请填写驳回原因',
      status === 'approved' ? '授予出题者权限' : '驳回出题资格申请',
      {
        inputType: 'textarea',
        inputPlaceholder: status === 'approved' ? '例如：已完成题目规范培训' : '必须说明驳回原因',
        inputValidator: (value) =>
          status === 'approved' || Boolean(value.trim()) || '必须填写驳回原因'
      }
    )
    await client.put(`/author-applications/${application.id}/review`, {
      status,
      review_note: result.value.trim()
    })
    ElMessage.success(status === 'approved' ? '已添加出题者标识' : '申请已驳回')
    await load()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close')
      ElMessage.error(error.response?.data?.error || error.message)
  }
}

async function updateAuthorPermission(user: User, enabled: boolean) {
  try {
    const result = await ElMessageBox.prompt(
      enabled
        ? `为“${user.name}”添加出题者标识，不改变其${roleText(user.role)}角色。`
        : `收回“${user.name}”的出题权限，不删除其角色和既有题目。`,
      enabled ? '授予出题者权限' : '收回出题者权限',
      {
        inputType: 'textarea',
        inputPlaceholder: '可填写本次权限变更说明',
        confirmButtonText: enabled ? '确认授予' : '确认收回',
        cancelButtonText: '取消'
      }
    )
    updatingUserID.value = user.id
    await client.put(`/users/${user.id}/permissions/problem_author`, {
      enabled,
      note: result.value.trim()
    })
    ElMessage.success(enabled ? '已添加出题者标识' : '已收回出题权限')
    await load()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close')
      ElMessage.error(error.response?.data?.error || error.message)
  } finally {
    updatingUserID.value = 0
  }
}

function hasAuthorPermission(user: User) {
  return user.role === 'admin' || Boolean(user.can_author)
}
function avatarText(user?: User) {
  return (user?.name || user?.email || 'U').trim().slice(0, 1).toUpperCase()
}
function roleText(role?: Role) {
  return role === 'admin' ? '管理员' : role === 'teacher' ? '教师' : '学生'
}
function roleTagType(role?: Role): 'danger' | 'warning' | 'info' {
  return role === 'admin' ? 'danger' : role === 'teacher' ? 'warning' : 'info'
}

onMounted(load)
</script>

<style scoped>
.permission-page {
  padding: 24px 28px 40px;
}
.permission-heading,
.section-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
}
.permission-heading {
  margin-bottom: 16px;
}
.permission-heading h2,
.section-heading h3 {
  margin: 0 0 4px;
}
.permission-heading p,
.section-heading p {
  margin: 0;
  color: var(--muted);
}
.permission-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 14px;
}
.permission-summary article {
  padding: 17px 18px;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--surface-strong);
}
.permission-summary span,
.permission-summary strong,
.permission-summary small {
  display: block;
}
.permission-summary span,
.permission-summary small {
  color: var(--muted);
}
.permission-summary strong {
  margin: 5px 0;
  font-size: 27px;
}
.permission-definition {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr) auto;
  align-items: center;
  gap: 16px;
  margin-bottom: 14px;
  padding: 18px;
}
.definition-mark {
  width: 46px;
  height: 46px;
  display: grid;
  place-items: center;
  color: #fff;
  border-radius: 9px;
  background: #135ecb;
  font-size: 21px;
  font-weight: 900;
}
.permission-definition span {
  color: #135ecb;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
}
.permission-definition h3 {
  margin: 4px 0;
}
.permission-definition p {
  margin: 0;
  color: var(--muted);
}
.definition-flow {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: #f7f9fc;
}
.definition-flow i {
  color: #94a3b8;
  font-style: normal;
}
.definition-flow strong {
  color: #16803b;
}
.application-panel,
.people-panel {
  padding: 18px;
}
.application-panel {
  margin-bottom: 14px;
}
.section-heading {
  align-items: center;
  margin-bottom: 16px;
}
.application-list {
  display: grid;
  gap: 9px;
}
.application-list article {
  display: grid;
  grid-template-columns: minmax(210px, 0.8fr) minmax(260px, 1.2fr) 150px auto;
  align-items: center;
  gap: 14px;
  padding: 13px 0;
  border-top: 1px solid var(--border);
}
.application-list p {
  margin: 0;
  color: var(--muted);
}
.application-list time {
  color: var(--muted);
  font-size: 12px;
}
.person,
.table-person {
  display: flex;
  align-items: center;
  gap: 10px;
}
.person > span,
.table-person > span {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  color: #135ecb;
  border-radius: 50%;
  background: #edf4ff;
  font-weight: 800;
}
.person div,
.table-person div {
  display: grid;
  gap: 3px;
}
.person small,
.table-person small {
  color: var(--muted);
}
.filters {
  display: flex;
  gap: 8px;
}
.filters .el-input {
  width: 230px;
}
.filters .el-select {
  width: 150px;
}
.permission-tags span {
  color: var(--muted);
  font-size: 13px;
}
.permission-mobile-list {
  display: none;
}
@media (max-width: 900px) {
  .permission-summary {
    grid-template-columns: 1fr;
  }
  .permission-definition {
    grid-template-columns: 52px 1fr;
  }
  .definition-flow {
    grid-column: 1/-1;
  }
  .application-list article {
    grid-template-columns: 1fr;
  }
  .section-heading {
    align-items: stretch;
    flex-direction: column;
  }
  .filters,
  .filters .el-input,
  .filters .el-select {
    width: 100%;
  }
}
@media (max-width: 640px) {
  .permission-page {
    padding: 18px 12px;
  }
  .permission-heading {
    align-items: stretch;
    flex-direction: column;
  }
  .filters {
    flex-direction: column;
  }
  .permission-table {
    display: none;
  }
  .permission-mobile-list {
    display: grid;
    gap: 10px;
  }
  .permission-mobile-list article {
    display: grid;
    gap: 14px;
    padding: 15px;
    border: 1px solid var(--border);
    border-radius: 7px;
    background: #fff;
  }
  .permission-mobile-list header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
  }
  .permission-mobile-list dl {
    display: grid;
    gap: 10px;
    margin: 0;
  }
  .permission-mobile-list dl > div {
    display: grid;
    grid-template-columns: 74px minmax(0, 1fr);
    gap: 8px;
  }
  .permission-mobile-list dt {
    color: var(--muted);
  }
  .permission-mobile-list dd {
    margin: 0;
    line-height: 1.55;
  }
  .permission-mobile-list .el-button,
  .permission-mobile-list > article > .el-tag {
    width: 100%;
  }
}
</style>
