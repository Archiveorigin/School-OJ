<template>
  <aside
    class="authoring-sidebar"
    :class="{ compact: step === 1 || collapsed }"
  >
    <RouterLink class="authoring-brand" to="/">
      <img src="/logo1.png" alt="青岛黄海学院" />
      <div>
        <strong>黄海在线</strong>
        <span>HUANGHAI ONLINE JUDGE</span>
      </div>
    </RouterLink>

    <nav class="authoring-nav" aria-label="教师工作区">
      <RouterLink
        v-for="item in items"
        :key="item.label"
        :to="item.to"
        :class="{ active: item.active }"
      >
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div v-if="step !== 1" class="campus-visual" aria-hidden="true">
      <img src="/bg-hero.webp" alt="" />
    </div>

    <button
      class="collapse-action"
      type="button"
      @click="collapsed = !collapsed"
    >
      <el-icon><Fold /></el-icon>
      <span>{{ collapsed ? "展开" : "收起" }}</span>
    </button>
  </aside>
</template>

<script setup lang="ts">
import {
  Bell,
  Collection,
  DataAnalysis,
  EditPen,
  Fold,
  QuestionFilled,
  Setting,
  Trophy,
} from "@element-plus/icons-vue";
import { ref } from "vue";

defineProps<{ step: number }>();

const collapsed = ref(false);
const items = [
  { label: "赛事", to: "/teams", icon: Trophy },
  { label: "练习", to: "/problems", icon: EditPen },
  { label: "教学", to: "/my/courses", icon: Collection, active: true },
  { label: "评测", to: "/submissions", icon: DataAnalysis },
  { label: "管理", to: "/admin", icon: Setting },
  { label: "消息", to: "/profile", icon: Bell },
  { label: "帮助", to: "/", icon: QuestionFilled },
];
</script>

<style scoped>
.authoring-sidebar {
  position: sticky;
  top: 0;
  z-index: 4;
  display: flex;
  width: 226px;
  height: 100vh;
  flex-direction: column;
  flex: 0 0 226px;
  overflow: hidden;
  color: #fff;
  background: #0d4fa8;
}

.authoring-sidebar.compact {
  width: 84px;
  flex-basis: 84px;
  background: #071d3c;
}

.authoring-brand {
  display: flex;
  min-height: 132px;
  align-items: center;
  gap: 12px;
  padding: 20px 18px;
  color: #fff;
}

.authoring-brand img {
  width: 58px;
  height: 58px;
  flex: 0 0 auto;
  object-fit: contain;
  border-radius: 50%;
  background: #fff;
}

.authoring-brand strong,
.authoring-brand span {
  display: block;
}

.authoring-brand strong {
  font-family: "STSong", "Songti SC", serif;
  font-size: 22px;
  letter-spacing: 0.08em;
}

.authoring-brand span {
  margin-top: 5px;
  font-size: 8px;
  letter-spacing: 0.08em;
  opacity: 0.76;
}

.authoring-sidebar.compact .authoring-brand {
  min-height: 84px;
  justify-content: center;
  padding: 12px;
}

.authoring-sidebar.compact .authoring-brand img {
  width: 54px;
  height: 54px;
}

.authoring-sidebar.compact .authoring-brand div {
  display: none;
}

.authoring-nav {
  display: grid;
  gap: 4px;
  padding: 8px 12px;
}

.authoring-nav a {
  display: flex;
  min-height: 54px;
  align-items: center;
  gap: 14px;
  padding: 0 16px;
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.82);
  font-weight: 650;
  transition:
    background 0.18s ease,
    color 0.18s ease;
}

.authoring-nav a:hover,
.authoring-nav a.active {
  color: #fff;
  background: rgba(255, 255, 255, 0.14);
}

.authoring-nav .el-icon {
  font-size: 22px;
}

.authoring-sidebar.compact .authoring-nav {
  padding: 8px 0;
}

.authoring-sidebar.compact .authoring-nav a {
  min-height: 70px;
  flex-direction: column;
  justify-content: center;
  gap: 5px;
  padding: 8px 2px;
  border-radius: 0;
  font-size: 12px;
}

.authoring-sidebar.compact .authoring-nav a.active {
  border-left: 4px solid #2a7bff;
  background: #0c315f;
}

.campus-visual {
  min-height: 150px;
  margin-top: auto;
  overflow: hidden;
}

.campus-visual img {
  width: 100%;
  height: 100%;
  min-height: 150px;
  object-fit: cover;
  filter: saturate(0.88);
}

.collapse-action {
  display: flex;
  min-height: 54px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 0;
  color: rgba(255, 255, 255, 0.82);
  background: rgba(0, 0, 0, 0.12);
  cursor: pointer;
}

@media (max-width: 900px) {
  .authoring-sidebar,
  .authoring-sidebar.compact {
    position: relative;
    width: 100%;
    height: auto;
    flex: none;
  }

  .authoring-brand {
    min-height: 72px;
  }

  .authoring-nav {
    display: flex;
    overflow-x: auto;
  }

  .authoring-nav a,
  .authoring-sidebar.compact .authoring-nav a {
    min-width: 76px;
    min-height: 52px;
    flex-direction: row;
    padding: 8px 12px;
  }

  .campus-visual,
  .collapse-action {
    display: none;
  }
}
</style>
