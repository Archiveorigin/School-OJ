<template>
  <el-aside width="232px" class="aside">
    <div class="brand">
      <img class="brand-logo" src="/logo.jpg" alt="黄海在线题测平台" />
      <span>黄海在线题测平台</span>
    </div>
    <el-menu router mode="vertical" :default-active="activeMenu" class="nav">
      <el-menu-item-group v-for="group in groups" :key="group.label" :title="group.label">
        <el-menu-item v-for="item in group.items" :key="item.path" :index="item.path">
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu-item-group>
    </el-menu>
  </el-aside>
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
</script>

<style scoped>
.aside {
  min-height: 100vh;
  background: var(--surface-strong);
  border-right: 1px solid var(--border);
  box-shadow: 8px 0 28px rgba(15, 23, 42, 0.04);
}

.brand {
  min-height: 64px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  font-weight: 800;
  line-height: 1.2;
  color: var(--text);
  border-bottom: 1px solid var(--border);
}

.brand-logo {
  width: 36px;
  height: 36px;
  object-fit: cover;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 20px rgba(10, 94, 166, 0.15);
}

.nav {
  border-right: 0;
  background: transparent;
  padding: 10px 10px 18px;
}

.nav :deep(.el-menu-item-group__title) {
  padding: 14px 10px 6px !important;
  color: var(--muted);
  font-size: 12px;
  font-weight: 700;
}

.nav :deep(.el-menu-item) {
  height: 42px;
  margin: 2px 0;
  border-radius: 8px;
}

.nav :deep(.el-menu-item.is-active) {
  background: color-mix(in srgb, var(--accent) 12%, transparent);
  color: var(--accent);
  font-weight: 700;
}

@media (max-width: 760px) {
  .aside {
    width: 100% !important;
    min-height: auto;
    border-right: 0;
    border-bottom: 1px solid var(--border);
    box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
  }

  .brand {
    min-height: 56px;
  }
}
</style>
