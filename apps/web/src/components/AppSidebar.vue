<template>
  <nav class="top-nav">
    <RouterLink class="brand" to="/">
      <img class="brand-logo" src="/logo.jpg" alt="黄海在线题测平台" />
      <span>黄海在线题测平台</span>
    </RouterLink>
    <div class="nav-scroll">
      <el-menu router mode="horizontal" :ellipsis="false" :default-active="activeMenu" class="nav">
        <el-menu-item v-for="item in items" :key="item.path" :index="item.path">
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Role } from '../api/client'
import { visibleNavGroups } from '../features/navigation/menu'

const props = defineProps<{
  activeMenu: string
  role?: Role
}>()

const groups = computed(() => visibleNavGroups(props.role))
const items = computed(() => groups.value.flatMap((group) => group.items))
</script>

<style scoped>
.top-nav {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1 1 auto;
  min-width: 0;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 0 0 auto;
  font-weight: 800;
  line-height: 1.2;
  color: var(--text);
}

.brand-logo {
  width: 36px;
  height: 36px;
  object-fit: cover;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 20px rgba(10, 94, 166, 0.15);
}

.nav-scroll {
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
}

.nav {
  min-width: max-content;
  border-bottom: 0;
  background: transparent;
}

.nav :deep(.el-menu-item) {
  height: 62px;
  padding: 0 14px;
  border-bottom: 2px solid transparent;
}

.nav :deep(.el-menu-item.is-active) {
  background: transparent;
  border-bottom-color: var(--accent);
  color: var(--accent);
  font-weight: 700;
}

@media (max-width: 760px) {
  .top-nav {
    width: 100%;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .nav-scroll {
    width: 100%;
  }

  .nav :deep(.el-menu-item) {
    height: 40px;
  }
}
</style>
