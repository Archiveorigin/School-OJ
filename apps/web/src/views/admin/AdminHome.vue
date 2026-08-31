<template>
  <section class="admin-home" v-loading="loading">
    <header class="admin-hero">
      <div class="admin-hero-copy">
        <span>青岛黄海学院 · SCHOOL OJ</span>
        <h1>教学管理中心</h1>
        <p>统筹课程、班级与考核，让考试发布回到清晰、连续的教学流程中。</p>
      </div>
      <img :src="campusImage" alt="" aria-hidden="true" />
    </header>

    <div class="admin-workspace-grid">
      <section class="admin-surface exam-plan-panel">
        <header class="surface-header">
          <div>
            <span class="section-kicker"
              ><el-icon><Calendar /></el-icon> 考试发布计划</span
            >
            <p>管理浏览器草稿与已创建考试</p>
          </div>
          <el-button text type="primary" @click="router.push('/admin/exams')"
            >查看全部 <el-icon><Right /></el-icon
          ></el-button>
        </header>

        <div class="plan-tabs" role="tablist" aria-label="考试筛选">
          <button
            v-for="item in planFilters"
            :key="item.value"
            type="button"
            :class="{ active: planFilter === item.value }"
            @click="planFilter = item.value"
          >
            {{ item.label }} <span>{{ item.count }}</span>
          </button>
        </div>

        <div v-if="filteredPlans.length" class="exam-plan-list">
          <button
            v-for="entry in filteredPlans"
            :key="entry.key"
            type="button"
            class="exam-plan-row"
            :class="{ selected: activePlan?.key === entry.key }"
            @click="selectedPlanKey = entry.key"
          >
            <span class="plan-date">
              <strong>{{ planDay(entry.startsAt) }}</strong>
              <small>{{ planMonth(entry.startsAt) }}</small>
            </span>
            <span class="plan-main">
              <span class="plan-title-line">
                <strong>{{ entry.title }}</strong>
                <em :class="`tone-${entry.tone}`">{{ entry.status }}</em>
              </span>
              <small
                >{{ entry.course || "尚未选择课程" }} ·
                {{ entry.scope || "尚未选择范围" }}</small
              >
              <span class="plan-meta">
                <span
                  ><el-icon><Clock /></el-icon>{{ planTime(entry) }}</span
                >
                <span
                  ><el-icon><Document /></el-icon
                  >{{ entry.problemCount }} 道题</span
                >
              </span>
            </span>
            <span class="row-action">{{
              entry.kind === "draft" ? "继续发布" : "查看"
            }}</span>
          </button>
        </div>

        <div v-else class="empty-plan">
          <el-icon><Calendar /></el-icon>
          <strong>当前筛选下暂无考试</strong>
          <span>可以新建考试，或切换其他状态查看。</span>
        </div>

        <footer class="plan-footer">
          <el-button @click="load"
            ><el-icon><Refresh /></el-icon>刷新数据</el-button
          >
          <el-button type="primary" @click="startNewExam"
            ><el-icon><EditPen /></el-icon>{{
              storedDraft ? "继续草稿" : "新建考试"
            }}</el-button
          >
        </footer>
      </section>

      <section class="admin-surface publish-panel">
        <header class="surface-header">
          <div>
            <span class="section-kicker"
              ><el-icon><Document /></el-icon>
              {{ activePlan?.kind === "draft" ? "发布准备" : "考试概览" }}</span
            >
            <p>
              {{
                activePlan?.kind === "draft"
                  ? "继续完成草稿，确认后再创建考试"
                  : "查看考试安排与当前状态"
              }}
            </p>
          </div>
          <span
            v-if="activePlan"
            class="publish-status"
            :class="`tone-${activePlan.tone}`"
            >{{ activePlan.status }}</span
          >
        </header>

        <template v-if="activePlan">
          <section class="publish-section">
            <h2>{{ activePlan.title }}</h2>
            <dl class="publish-facts">
              <div v-for="fact in activeFacts" :key="fact.label">
                <dt>{{ fact.label }}</dt>
                <dd>{{ fact.value }}</dd>
              </div>
            </dl>
          </section>

          <template v-if="activePlan.kind === 'draft'">
            <section class="publish-section readiness-section">
              <h3>发布前检查</h3>
              <ul>
                <li
                  v-for="check in draftChecks"
                  :key="check.label"
                  :class="{ ready: check.ready }"
                >
                  <el-icon
                    ><CircleCheck v-if="check.ready" /><WarningFilled v-else
                  /></el-icon>
                  <span>{{ check.label }}</span>
                </li>
              </ul>
            </section>

            <section class="publish-settings">
              <div v-for="setting in draftSettings" :key="setting.label">
                <span>{{ setting.label }}</span>
                <strong>{{ setting.value }}</strong>
              </div>
            </section>
          </template>

          <div class="publish-actions">
            <el-button
              v-if="activePlan.kind === 'draft'"
              @click="router.push('/admin/exams')"
              >返回列表</el-button
            >
            <el-button
              :type="activePlan.kind === 'draft' ? 'danger' : 'primary'"
              @click="openActivePlan"
            >
              {{
                activePlan.kind === "draft" ? "继续编辑与发布" : "查看考试详情"
              }}
              <el-icon><Right /></el-icon>
            </el-button>
          </div>
          <p v-if="activePlan.kind === 'draft'" class="publish-help">
            草稿仅保存在当前浏览器；进入发布流程后仍会执行完整校验。
          </p>
        </template>

        <div v-else class="empty-publish">
          <el-icon><Document /></el-icon>
          <h2>还没有考试安排</h2>
          <p>从基本信息、题目与分值开始创建第一场考试。</p>
          <el-button type="primary" @click="startNewExam">新建考试</el-button>
        </div>
      </section>
    </div>

    <div class="admin-secondary-grid">
      <section class="admin-surface compact-surface">
        <header class="surface-header">
          <div>
            <span class="section-kicker">待处理事项</span>
            <p>需要管理员决策的出题资格与题目修改工单</p>
          </div>
          <el-button
            v-if="auth.role === 'admin'"
            text
            type="primary"
            @click="router.push('/admin/problem-authors')"
            >全部待办 <el-icon><Right /></el-icon
          ></el-button>
        </header>
        <div v-if="pendingItems.length" class="compact-list">
          <button
            v-for="item in pendingItems"
            :key="item.path"
            type="button"
            @click="router.push(item.path)"
          >
            <span class="compact-icon"
              ><el-icon><WarningFilled /></el-icon
            ></span>
            <span
              ><strong>{{ item.title }}</strong
              ><small>{{ item.description }}</small></span
            >
            <em>{{ item.count }}</em>
          </button>
        </div>
        <div v-else class="compact-empty">当前没有待处理审批。</div>
      </section>

      <section class="admin-surface compact-surface">
        <header class="surface-header">
          <div>
            <span class="section-kicker">管理动态</span>
            <p>
              {{
                auth.role === "admin"
                  ? "来自审计日志的最新操作"
                  : "近期考试安排"
              }}
            </p>
          </div>
          <el-button
            v-if="auth.role === 'admin'"
            text
            type="primary"
            @click="router.push('/admin/audit-logs')"
            >查看全部 <el-icon><Right /></el-icon
          ></el-button>
        </header>
        <div v-if="activityItems.length" class="activity-list">
          <div v-for="item in activityItems" :key="item.key">
            <span class="activity-dot"></span>
            <p>
              <strong>{{ item.actor }}</strong
              >{{ item.text }}
            </p>
            <time>{{ item.time }}</time>
          </div>
        </div>
        <div v-else class="compact-empty">暂无可展示的管理动态。</div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  Calendar,
  CircleCheck,
  Clock,
  Document,
  EditPen,
  Refresh,
  Right,
  WarningFilled,
} from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { client } from "../../api/client";
import { apiErrorMessage } from "../../api/errors";
import {
  examTotalScore,
  validateExamDraft,
} from "../../features/exams/builder";
import {
  examDraftStorageKey,
  parseStoredExamDraft,
  type StoredExamDraft,
} from "../../features/exams/draft";
import { formatDateTime } from "../../features/time";
import { useAuthStore } from "../../stores/auth";

type ExamItem = {
  id: number;
  title: string;
  course_id?: number;
  course_code?: string;
  course_name?: string;
  class_id?: number | null;
  class_name?: string;
  starts_at?: string | null;
  ends_at?: string | null;
  scoring_rule?: string;
  problems?: Array<{ score?: number }>;
};

type PlanEntry = {
  key: string;
  kind: "draft" | "exam";
  title: string;
  course: string;
  scope: string;
  startsAt: Date | string | null;
  endsAt: Date | string | null;
  problemCount: number;
  totalScore: number;
  status: string;
  tone: "draft" | "upcoming" | "running" | "closed";
  raw: StoredExamDraft | ExamItem;
};

type CourseItem = { id: number; code?: string; name?: string };
type ClassItem = {
  id?: number;
  class_id?: number;
  name?: string;
  class_name?: string;
  course_id?: number;
};
type AuditItem = {
  id: number;
  actor_name?: string;
  action?: string;
  resource_label?: string;
  created_at?: string;
};

const router = useRouter();
const auth = useAuthStore();
const campusImage = "/bg-hero.webp";
const loading = ref(false);
const exams = ref<ExamItem[]>([]);
const courses = ref<CourseItem[]>([]);
const classes = ref<ClassItem[]>([]);
const storedDraft = ref<StoredExamDraft | null>(null);
const pendingApplications = ref<any[]>([]);
const pendingProblemTickets = ref<any[]>([]);
const auditLogs = ref<AuditItem[]>([]);
const selectedPlanKey = ref("");
const planFilter = ref<"all" | "draft" | "running" | "upcoming" | "closed">(
  "all",
);

const courseMap = computed(
  () => new Map(courses.value.map((course) => [course.id, course])),
);
const classMap = computed(
  () =>
    new Map(
      classes.value.map((item) => [Number(item.class_id || item.id), item]),
    ),
);

const plans = computed<PlanEntry[]>(() => {
  const items: PlanEntry[] = [];
  if (storedDraft.value) {
    const draft = storedDraft.value;
    const course = draft.form.course_id
      ? courseMap.value.get(draft.form.course_id)
      : undefined;
    const classItem =
      draft.form.class_id && draft.form.class_id > 0
        ? classMap.value.get(draft.form.class_id)
        : undefined;
    items.push({
      key: "draft",
      kind: "draft",
      title: draft.form.title.trim() || "未命名考试草稿",
      course: course
        ? [course.code, course.name].filter(Boolean).join(" ")
        : "",
      scope:
        draft.form.class_id === -1
          ? "全课程"
          : classItem?.class_name || classItem?.name || "",
      startsAt: draft.form.starts_at,
      endsAt: draft.form.ends_at,
      problemCount: draft.selectedProblems.length,
      totalScore: examTotalScore(draft.selectedProblems),
      status: `草稿 · 第 ${draft.step + 1} 步`,
      tone: "draft",
      raw: draft,
    });
  }

  const examEntries = exams.value.map((exam) => {
    const tone = examTone(exam);
    return {
      key: `exam-${exam.id}`,
      kind: "exam" as const,
      title: exam.title,
      course: [exam.course_code, exam.course_name].filter(Boolean).join(" "),
      scope: exam.class_name || "全课程",
      startsAt: exam.starts_at || null,
      endsAt: exam.ends_at || null,
      problemCount: exam.problems?.length || 0,
      totalScore: (exam.problems || []).reduce(
        (sum, item) => sum + Number(item.score || 0),
        0,
      ),
      status:
        tone === "running"
          ? "进行中"
          : tone === "upcoming"
            ? "待开始"
            : "已结束",
      tone,
      raw: exam,
    };
  });

  examEntries.sort((a, b) => {
    const order = { running: 0, upcoming: 1, draft: 2, closed: 3 };
    const toneOrder = order[a.tone] - order[b.tone];
    if (toneOrder) return toneOrder;
    return dateValue(a.startsAt) - dateValue(b.startsAt);
  });
  return [...items, ...examEntries];
});

const planFilters = computed(() => [
  { label: "全部", value: "all" as const, count: plans.value.length },
  {
    label: "草稿",
    value: "draft" as const,
    count: plans.value.filter((item) => item.kind === "draft").length,
  },
  {
    label: "进行中",
    value: "running" as const,
    count: plans.value.filter((item) => item.tone === "running").length,
  },
  {
    label: "待开始",
    value: "upcoming" as const,
    count: plans.value.filter((item) => item.tone === "upcoming").length,
  },
  {
    label: "已结束",
    value: "closed" as const,
    count: plans.value.filter((item) => item.tone === "closed").length,
  },
]);

const filteredPlans = computed(() =>
  plans.value
    .filter(
      (item) =>
        planFilter.value === "all" ||
        (planFilter.value === "draft"
          ? item.kind === "draft"
          : item.tone === planFilter.value),
    )
    .slice(0, 6),
);
const activePlan = computed(
  () =>
    plans.value.find((item) => item.key === selectedPlanKey.value) ||
    plans.value[0],
);

const activeFacts = computed(() => {
  const plan = activePlan.value;
  if (!plan) return [];
  return [
    { label: "所属课程", value: plan.course || "未设置" },
    { label: "考试范围", value: plan.scope || "未设置" },
    {
      label: "开始时间",
      value: plan.startsAt ? formatDateTime(plan.startsAt) : "未设置",
    },
    {
      label: "结束时间",
      value: plan.endsAt ? formatDateTime(plan.endsAt) : "未设置",
    },
    { label: "题目数量", value: `${plan.problemCount} 道` },
    { label: "试卷总分", value: `${plan.totalScore} 分` },
  ];
});

const draftChecks = computed(() => {
  const draft = storedDraft.value;
  if (!draft) return [];
  const hasScope =
    draft.form.class_id === -1
      ? Boolean(draft.form.course_id)
      : Boolean(draft.form.class_id);
  const hasTime = Boolean(
    draft.form.starts_at &&
    draft.form.ends_at &&
    dateValue(draft.form.ends_at) > dateValue(draft.form.starts_at),
  );
  const hasProblems = draft.selectedProblems.length > 0;
  const valid = !validateExamDraft(draft.form, draft.selectedProblems);
  return [
    {
      label: "考试名称与可见范围已设置",
      ready: Boolean(draft.form.title.trim() && hasScope),
    },
    { label: "考试起止时间有效", ready: hasTime },
    { label: "已选择题目并设置分值", ready: hasProblems && valid },
  ];
});

const draftSettings = computed(() => {
  const form = storedDraft.value?.form;
  if (!form) return [];
  return [
    { label: "评测方式", value: form.manual_review ? "人工阅卷" : "即时评测" },
    {
      label: "计分规则",
      value: form.scoring_rule.toUpperCase(),
    },
    {
      label: "实时榜单",
      value: form.ranking_visible ? "考试期间可见" : "不公开",
    },
  ];
});

const pendingItems = computed(() => {
  if (auth.role !== "admin") return [];
  return [
    {
      path: "/admin/problem-authors",
      title: "出题资格申请",
      description: "审核教师或学生的出题权限申请",
      count: pendingApplications.value.length,
    },
    {
      path: "/admin/problem-authors",
      title: "题目修改工单",
      description: "核对需求并上传完整最终题包",
      count: pendingProblemTickets.value.length,
    },
  ].filter((item) => item.count > 0);
});

const activityItems = computed(() => {
  if (auth.role === "admin") {
    return auditLogs.value.slice(0, 4).map((item) => ({
      key: `audit-${item.id}`,
      actor: item.actor_name || "系统",
      text: ` ${auditActionLabel(item)}`,
      time: shortDateTime(item.created_at),
    }));
  }
  return exams.value.slice(0, 4).map((item) => ({
    key: `exam-activity-${item.id}`,
    actor: "考试安排",
    text: ` · ${item.title}`,
    time: shortDateTime(item.starts_at),
  }));
});

function examTone(exam: ExamItem): "upcoming" | "running" | "closed" {
  const now = Date.now();
  if (exam.ends_at && dateValue(exam.ends_at) <= now) return "closed";
  if (exam.starts_at && dateValue(exam.starts_at) > now) return "upcoming";
  return "running";
}

function dateValue(value: Date | string | null | undefined) {
  if (!value) return Number.MAX_SAFE_INTEGER;
  const time = new Date(value).getTime();
  return Number.isFinite(time) ? time : Number.MAX_SAFE_INTEGER;
}

function planDay(value: Date | string | null) {
  if (!value) return "--";
  return String(new Date(value).getDate()).padStart(2, "0");
}

function planMonth(value: Date | string | null) {
  if (!value) return "待定";
  return `${new Date(value).getMonth() + 1}月`;
}

function planTime(plan: PlanEntry) {
  if (!plan.startsAt) return "时间待定";
  const start = new Date(plan.startsAt);
  const end = plan.endsAt ? new Date(plan.endsAt) : undefined;
  const time = start.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
  });
  if (!end) return time;
  return `${time}–${end.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" })}`;
}

function shortDateTime(value?: string | null) {
  if (!value) return "-";
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function auditActionLabel(item: AuditItem) {
  const labels: Record<string, string> = {
    "exam.create": "创建了考试",
    "exam.delete": "删除了考试",
    "course.create": "创建了课程",
    "course.update": "更新了课程",
    "class.create": "创建了班级",
    "problem.create": "创建了题目",
    "problem_review.review": "处理了题目审批",
    "author_application.review": "处理了出题资格申请",
    "plagiarism.create": "启动了查重任务",
  };
  return `${labels[item.action || ""] || item.action || "执行了管理操作"}${item.resource_label ? ` · ${item.resource_label}` : ""}`;
}

function startNewExam() {
  router.push("/admin/exams/new");
}

function openActivePlan() {
  const plan = activePlan.value;
  if (!plan) return;
  if (plan.kind === "draft") {
    router.push("/admin/exams/new");
    return;
  }
  router.push(`/exams/${(plan.raw as ExamItem).id}`);
}

async function load() {
  loading.value = true;
  try {
    const [examResponse, courseResponse, classResponse] = await Promise.all([
      client.get("/exams"),
      client.get("/courses"),
      client.get("/classes"),
    ]);
    exams.value = Array.isArray(examResponse.data) ? examResponse.data : [];
    courses.value = Array.isArray(courseResponse.data)
      ? courseResponse.data
      : [];
    classes.value = Array.isArray(classResponse.data) ? classResponse.data : [];
    storedDraft.value = parseStoredExamDraft(
      localStorage.getItem(examDraftStorageKey),
    );

    if (auth.role === "admin") {
      const [applicationResult, reviewResult, auditResult] =
        await Promise.allSettled([
          client.get("/author-applications", { params: { status: "pending" } }),
          client.get("/problem-change-tickets", { params: { status: "pending" } }),
          client.get("/audit-logs"),
        ]);
      pendingApplications.value =
        applicationResult.status === "fulfilled" &&
        Array.isArray(applicationResult.value.data)
          ? applicationResult.value.data
          : [];
      pendingProblemTickets.value =
        reviewResult.status === "fulfilled" &&
        Array.isArray(reviewResult.value.data)
          ? reviewResult.value.data
          : [];
      auditLogs.value =
        auditResult.status === "fulfilled" &&
        Array.isArray(auditResult.value.data)
          ? auditResult.value.data
          : [];
    }

    if (!plans.value.some((item) => item.key === selectedPlanKey.value)) {
      selectedPlanKey.value = plans.value[0]?.key || "";
    }
  } catch (error: unknown) {
    ElMessage.error(apiErrorMessage(error, "加载教学管理数据失败"));
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<style scoped>
.admin-home {
  min-height: calc(100vh - 64px);
  padding: 30px 28px 18px;
  color: #132a4d;
}

.admin-hero {
  position: relative;
  min-height: 138px;
  display: flex;
  align-items: center;
  overflow: hidden;
  margin: -30px -28px 14px;
  padding: 20px 38px;
  border-bottom: 1px solid #dce4ef;
  background: #fff;
}

.admin-hero-copy {
  position: relative;
  z-index: 2;
  max-width: 650px;
}

.admin-hero-copy > span {
  color: #6f7f96;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.13em;
}

.admin-hero h1 {
  margin: 8px 0 5px;
  color: #0a3d86;
  font-family: "STSong", "Songti SC", "Noto Serif SC", serif;
  font-size: clamp(38px, 4.4vw, 60px);
  font-weight: 800;
  letter-spacing: 0.04em;
  line-height: 1.08;
}

.admin-hero p {
  margin: 0;
  color: #69788e;
  font-size: 14px;
}

.admin-hero img {
  position: absolute;
  inset: 0 0 0 auto;
  width: 55%;
  height: 100%;
  object-fit: cover;
  object-position: center 42%;
  opacity: 0.19;
  filter: grayscale(1) sepia(0.15) hue-rotate(170deg) saturate(2.1);
  mask-image: linear-gradient(
    to right,
    transparent 0,
    rgba(0, 0, 0, 0.85) 32%,
    #000 100%
  );
}

.admin-workspace-grid,
.admin-secondary-grid {
  display: grid;
  grid-template-columns: minmax(460px, 0.96fr) minmax(520px, 1.04fr);
  gap: 14px;
}

.admin-secondary-grid {
  margin-top: 14px;
}

.admin-surface {
  min-width: 0;
  overflow: hidden;
  border: 1px solid #d8e1ec;
  border-radius: 5px;
  background: #fff;
}

.surface-header {
  display: flex;
  min-height: 76px;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 16px 20px;
  border-bottom: 1px solid #e3e9f1;
}

.surface-header > div {
  min-width: 0;
}

.section-kicker {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #173765;
  font-size: 17px;
  font-weight: 800;
}

.surface-header p {
  margin: 5px 0 0;
  color: #79869a;
  font-size: 12px;
}

.plan-tabs {
  display: flex;
  gap: 18px;
  overflow-x: auto;
  padding: 0 20px;
  border-bottom: 1px solid #e3e9f1;
}

.plan-tabs button {
  position: relative;
  min-height: 44px;
  padding: 0;
  border: 0;
  color: #61718a;
  background: transparent;
  font-weight: 650;
  white-space: nowrap;
  cursor: pointer;
}

.plan-tabs button span {
  margin-left: 3px;
  color: #98a3b3;
  font-size: 12px;
}

.plan-tabs button.active {
  color: #135ecb;
}

.plan-tabs button.active::after {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 2px;
  background: #135ecb;
  content: "";
}

.exam-plan-list {
  min-height: 390px;
}

.exam-plan-row {
  position: relative;
  width: 100%;
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
  min-height: 94px;
  padding: 13px 18px;
  border: 0;
  border-bottom: 1px solid #e7ecf3;
  color: #1d3559;
  text-align: left;
  background: #fff;
  cursor: pointer;
}

.exam-plan-row:hover,
.exam-plan-row.selected {
  background: #f2f6fc;
}

.exam-plan-row.selected::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: #135ecb;
  content: "";
}

.plan-date {
  display: grid;
  place-items: center;
  align-self: stretch;
  align-content: center;
  border-right: 1px solid #e0e7f0;
}

.plan-date strong {
  color: #0a3d86;
  font-size: 22px;
  line-height: 1;
}

.plan-date small {
  margin-top: 5px;
  color: #8794a7;
  font-size: 11px;
}

.plan-main,
.plan-title-line,
.plan-meta {
  min-width: 0;
  display: flex;
}

.plan-main {
  flex-direction: column;
  gap: 7px;
}

.plan-title-line {
  align-items: center;
  gap: 9px;
}

.plan-title-line strong {
  overflow: hidden;
  color: #153764;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.plan-title-line em,
.publish-status {
  flex: 0 0 auto;
  padding: 3px 7px;
  border-radius: 3px;
  font-size: 10px;
  font-style: normal;
  font-weight: 700;
}

.tone-draft {
  color: #2363bd;
  background: #e8f1ff;
}

.tone-upcoming {
  color: #a85b00;
  background: #fff3dd;
}

.tone-running {
  color: #08734c;
  background: #e4f7ef;
}

.tone-closed {
  color: #6c7685;
  background: #eef1f5;
}

.plan-main > small {
  overflow: hidden;
  color: #6e7b8e;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.plan-meta {
  gap: 16px;
  color: #8290a3;
  font-size: 11px;
}

.plan-meta span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.row-action {
  color: #135ecb;
  font-size: 12px;
  font-weight: 700;
}

.plan-footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 18px;
  background: #fbfcfe;
}

.empty-plan,
.empty-publish {
  display: grid;
  place-items: center;
  align-content: center;
  min-height: 390px;
  padding: 30px;
  color: #8290a3;
  text-align: center;
}

.empty-plan .el-icon,
.empty-publish .el-icon {
  margin-bottom: 12px;
  color: #90a4c1;
  font-size: 36px;
}

.empty-plan strong,
.empty-publish h2 {
  margin: 0;
  color: #24456f;
}

.empty-plan span,
.empty-publish p {
  margin: 7px 0 18px;
}

.publish-panel {
  display: flex;
  flex-direction: column;
}

.publish-section {
  padding: 20px 22px;
  border-bottom: 1px solid #e3e9f1;
}

.publish-section h2,
.publish-section h3 {
  margin: 0 0 18px;
  color: #173765;
}

.publish-section h2 {
  font-size: 21px;
}

.publish-section h3 {
  position: relative;
  padding-left: 11px;
  font-size: 15px;
}

.publish-section h3::before {
  position: absolute;
  inset: 1px auto 1px 0;
  width: 3px;
  background: #135ecb;
  content: "";
}

.publish-facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 24px;
  margin: 0;
}

.publish-facts div {
  min-width: 0;
  display: grid;
  grid-template-columns: 70px minmax(0, 1fr);
  gap: 9px;
  font-size: 12px;
}

.publish-facts dt {
  color: #7b8799;
}

.publish-facts dd {
  overflow: hidden;
  margin: 0;
  color: #243b5c;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.readiness-section ul {
  display: grid;
  gap: 12px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.readiness-section li {
  display: flex;
  align-items: center;
  gap: 9px;
  color: #a85b00;
  font-size: 13px;
}

.readiness-section li.ready {
  color: #0d8a58;
}

.publish-settings {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  margin: 18px 22px;
  border: 1px solid #dfe6ef;
  background: #dfe6ef;
}

.publish-settings div {
  display: grid;
  gap: 6px;
  padding: 12px;
  background: #f9fbfd;
}

.publish-settings span {
  color: #8290a3;
  font-size: 11px;
}

.publish-settings strong {
  color: #334b6b;
  font-size: 12px;
}

.publish-actions {
  display: grid;
  grid-template-columns: 1fr 1.35fr;
  gap: 12px;
  margin-top: auto;
  padding: 18px 22px 10px;
}

.publish-actions :deep(.el-button) {
  min-height: 42px;
}

.publish-actions :deep(.el-button--danger) {
  --el-button-bg-color: #c92c2c;
  --el-button-border-color: #c92c2c;
  --el-button-hover-bg-color: #ac2020;
  --el-button-hover-border-color: #ac2020;
}

.publish-help {
  margin: 0;
  padding: 0 22px 18px;
  color: #7d8a9c;
  font-size: 11px;
  text-align: right;
}

.compact-surface .surface-header {
  min-height: 60px;
  padding-top: 12px;
  padding-bottom: 12px;
}

.compact-list button {
  width: 100%;
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 9px 20px;
  border: 0;
  border-bottom: 1px solid #e7ecf3;
  color: #243b5c;
  text-align: left;
  background: #fff;
  cursor: pointer;
}

.compact-list button:hover {
  background: #f7f9fc;
}

.compact-icon {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  color: #a85b00;
  border-radius: 50%;
  background: #fff3dd;
}

.compact-list button > span:nth-child(2) {
  display: grid;
  gap: 4px;
}

.compact-list strong {
  font-size: 13px;
}

.compact-list small {
  color: #7f8b9d;
}

.compact-list em {
  min-width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  color: #fff;
  border-radius: 50%;
  background: #135ecb;
  font-size: 11px;
  font-style: normal;
  font-weight: 800;
}

.compact-empty {
  min-height: 96px;
  display: grid;
  place-items: center;
  padding: 20px;
  color: #8491a4;
  font-size: 13px;
}

.activity-list {
  padding: 2px 20px 6px;
}

.activity-list > div {
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  color: #4f6078;
  font-size: 12px;
}

.activity-dot {
  width: 6px;
  height: 6px;
  border: 1px solid #56739a;
  border-radius: 50%;
}

.activity-list p {
  overflow: hidden;
  margin: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.activity-list strong {
  color: #2a4367;
}

.activity-list time {
  color: #96a0af;
}

@media (max-width: 1180px) {
  .admin-workspace-grid,
  .admin-secondary-grid {
    grid-template-columns: 1fr;
  }

  .exam-plan-list {
    min-height: 0;
  }
}

@media (max-width: 680px) {
  .admin-home {
    padding: 18px 14px 28px;
  }

  .admin-hero {
    min-height: 130px;
    margin: -18px -14px 16px;
    padding: 22px 18px;
  }

  .admin-hero img {
    width: 78%;
    opacity: 0.11;
  }

  .admin-hero h1 {
    font-size: 36px;
  }

  .admin-hero p {
    max-width: 300px;
    font-size: 12px;
  }

  .surface-header {
    align-items: flex-start;
    padding: 15px;
  }

  .surface-header .el-button {
    padding-right: 0;
  }

  .plan-tabs {
    gap: 15px;
    padding: 0 15px;
  }

  .exam-plan-row {
    grid-template-columns: 42px minmax(0, 1fr);
    padding: 12px 14px;
  }

  .row-action {
    display: none;
  }

  .publish-facts,
  .publish-settings,
  .publish-actions {
    grid-template-columns: 1fr;
  }

  .publish-settings {
    margin: 15px;
  }

  .publish-section,
  .publish-actions {
    padding-right: 15px;
    padding-left: 15px;
  }

  .activity-list > div {
    grid-template-columns: 12px minmax(0, 1fr);
  }

  .activity-list time {
    grid-column: 2;
  }
}
</style>
