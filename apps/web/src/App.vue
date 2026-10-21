<template>
  <router-view v-if="authPage" />
  <el-container
    v-else
    direction="vertical"
    class="shell"
    :class="{
      'exam-shell': studentExamWorkspace,
      'admin-shell': adminWorkspace,
      'course-shell': courseWorkspace,
      'team-shell': teamWorkspace,
      'authoring-shell': examAuthoringWorkspace,
    }"
  >
    <el-header
      v-if="
        !studentExamWorkspace &&
        !adminWorkspace &&
        !courseWorkspace &&
        !teamWorkspace &&
        !examAuthoringWorkspace
      "
      class="topbar"
      height="auto"
    >
      <AppSidebar
        :active-menu="activeMenu"
        :role="auth.role"
        :authenticated="auth.isAuthed"
        :can-author="canAuthor"
      />
      <div class="topbar-actions">
        <el-dropdown
          v-if="auth.isAuthed"
          trigger="hover"
          :show-timeout="120"
          :hide-timeout="180"
          @command="handleCommand"
        >
          <button class="avatar-button" type="button">
            <img
              v-if="auth.user?.avatar_url"
              :src="auth.user.avatar_url"
              alt=""
            />
            <span v-else>{{ initials }}</span>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">个人中心</el-dropdown-item>
              <el-dropdown-item command="update-profile"
                >更新个人资料</el-dropdown-item
              >
              <el-dropdown-item v-if="auth.can('course:teach')" command="admin"
                >后台管理</el-dropdown-item
              >
              <el-dropdown-item command="theme">
                {{ auth.theme === "dark" ? "切换明亮模式" : "切换暗黑模式" }}
              </el-dropdown-item>
              <el-dropdown-item divided command="logout"
                >退出登录</el-dropdown-item
              >
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button v-else type="primary" @click="goLogin">登录</el-button>
      </div>
    </el-header>
    <el-main
      class="main-content"
      :class="{
        'exam-main-content': studentExamWorkspace,
        'admin-main-content': adminWorkspace,
        'course-main-content': courseWorkspace,
        'team-main-content': teamWorkspace,
        'authoring-main-content': examAuthoringWorkspace,
      }"
    >
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { computed, watch, onMounted, onBeforeUnmount } from "vue";
import { flushSavedExamActivity } from "./features/exams/activity";
import { useRoute, useRouter } from "vue-router";
import AppSidebar from "./components/AppSidebar.vue";
import { useAuthStore } from "./stores/auth";
import { useExamLockStore } from "./stores/examLock";

const auth = useAuthStore();
const examLock = useExamLockStore();
const router = useRouter();
const route = useRoute();
let examSyncing = false;
let examSyncTimer: number | undefined;

const authPage = computed(() =>
  ["/login", "/register", "/forgot-password"].includes(route.path),
);
const initials = computed(() =>
  (auth.user?.name || auth.user?.email || "U").trim().slice(0, 1).toUpperCase(),
);
const canAuthor = computed(
  () => auth.can("problem:author"),
);
const activeMenu = computed(() => String(route.meta.activeMenu || route.path));
const currentExamRouteId = computed(() => {
  const value = route.params.id;
  return typeof value === "string" ? Number(value) : undefined;
});
const studentExamWorkspace = computed(
  () =>
    auth.role === "student" &&
    Boolean(currentExamRouteId.value) &&
    route.path.startsWith("/exams/"),
);
const adminWorkspace = computed(
  () => route.path === "/admin" || route.path.startsWith("/admin/"),
);
const courseWorkspace = computed(() =>
  /^\/my\/courses\/[^/]+/.test(route.path),
);
const teamWorkspace = computed(() =>
  /^\/teams\/[^/]+\/(?:contests(?:\/[^/]+)?|problem-sets|members)$/.test(
    route.path,
  ),
);
const examAuthoringWorkspace = computed(() => route.path === "/exams/new");
const activeExamRoot = computed(() =>
  examLock.examId ? `/exams/${examLock.examId}` : "",
);
const inActiveExam = computed(
  () =>
    Boolean(activeExamRoot.value) &&
    (route.path === activeExamRoot.value ||
      route.path.startsWith(`${activeExamRoot.value}/`)),
);

function logout() {
  auth.logout();
  examLock.unlock();
  router.push("/login");
}

function goLogin() {
  router.push({ path: "/login", query: { redirect: route.fullPath } });
}

function handleCommand(command: string) {
  if (command === "profile") {
    router.push("/profile");
    return;
  }
  if (command === "update-profile") {
    router.push({ path: "/profile", query: { edit: "1" } });
    return;
  }
  if (command === "admin") {
    router.push("/admin");
    return;
  }
  if (command === "theme") {
    auth.toggleTheme();
    return;
  }
  if (command === "logout") {
    logout();
  }
}

async function syncExamEnvironment() {
  if (examSyncing || !auth.isAuthed || auth.role !== "student") return;
  examSyncing = true;
  try {
    // Initial URL/legacy redirects must resolve before global lock recovery.
    // Otherwise the startup '/' placeholder overwrites a valid exam deep link.
    await router.isReady();
    await examLock.syncActiveExam();
    if (examLock.locked && examLock.examId && !inActiveExam.value) await router.replace(`/exams/${examLock.examId}/description`);
    if (!examLock.locked && auth.user) await flushSavedExamActivity(auth.user.id);
  } catch { /* Keep the last server-verified state on connection failure. */ }
  finally { examSyncing = false; }
}
onMounted(() => {
  window.addEventListener('focus', syncExamEnvironment);
  window.addEventListener('storage', syncExamEnvironment);
  examSyncTimer = window.setInterval(syncExamEnvironment, 15000);
});
onBeforeUnmount(() => {
  window.removeEventListener('focus', syncExamEnvironment);
  window.removeEventListener('storage', syncExamEnvironment);
  if (examSyncTimer) window.clearInterval(examSyncTimer);
});

watch(
  () => auth.isAuthed,
  (authed) => {
    if (!authed) {
      examLock.unlock();
    }
  },
);

watch(
  () => [auth.isAuthed, auth.role, route.fullPath],
  () => {
    void syncExamEnvironment();
  },
  { immediate: true },
);
</script>

<style scoped>
.shell {
  min-height: 100vh;
  background:
    linear-gradient(180deg, rgba(10, 94, 166, 0.05), transparent 320px),
    var(--app-bg);
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 64px;
  padding: 0 18px;
  background: var(--glass);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(18px);
}

.topbar-actions {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  flex: 0 0 auto;
}

.avatar-button {
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: linear-gradient(135deg, var(--accent), #14b8a6);
  color: #fff;
  font-weight: 800;
  cursor: pointer;
  overflow: hidden;
  transition:
    transform 0.18s ease,
    box-shadow 0.18s ease;
}

.avatar-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 28px rgba(10, 94, 166, 0.18);
}

.avatar-button img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.main-content {
  padding: 0;
}

.exam-shell {
  min-height: 100vh;
}

.admin-shell,
.admin-main-content,
.course-shell,
.course-main-content,
.team-shell,
.team-main-content {
  min-height: 100vh;
}

.admin-main-content,
.course-main-content,
.team-main-content,
.authoring-main-content {
  padding: 0;
}

.authoring-shell,
.authoring-main-content {
  min-height: 100vh;
}

.exam-main-content :deep(.page) {
  min-height: 100vh;
}




@media (max-width: 760px) {
  .topbar {
    align-items: flex-start;
    flex-direction: column;
    padding: 10px 12px;
  }
}
</style>
