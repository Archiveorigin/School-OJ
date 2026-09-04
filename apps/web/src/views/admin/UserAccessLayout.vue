<template>
  <section class="user-access-layout">
    <header class="access-heading">
      <div>
        <span>IDENTITY & ACCESS</span>
        <h1>用户与权限</h1>
      </div>
    </header>
    <nav class="access-tabs" aria-label="用户与权限子导航">
      <RouterLink to="/admin/users" :class="{ active: route.path === '/admin/users' }"
        >用户列表</RouterLink
      >
      <RouterLink
        to="/admin/users/permissions"
        :class="{ active: route.path.startsWith('/admin/users/permissions') }"
        >权限管理</RouterLink
      >
    </nav>
    <router-view />
  </section>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<style scoped>
.access-heading {
  padding: 26px 28px 18px;
  border-bottom: 1px solid var(--admin-line);
  background: var(--admin-paper);
}
.access-heading span {
  color: var(--admin-accent);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.14em;
}
.access-heading h1 {
  margin: 6px 0 4px;
  color: var(--admin-ink);
  font-size: 26px;
}
.access-tabs {
  display: flex;
  gap: 4px;
  padding: 0 28px;
  border-bottom: 1px solid var(--admin-line);
  background: var(--admin-paper);
}
.access-tabs a {
  position: relative;
  min-width: 110px;
  padding: 14px 18px;
  color: var(--admin-muted);
  text-align: center;
  font-weight: 700;
}
.access-tabs a.active {
  color: var(--admin-accent);
}
.access-tabs a.active::after {
  position: absolute;
  right: 12px;
  bottom: -1px;
  left: 12px;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: var(--admin-accent);
  content: '';
}
@media (max-width: 640px) {
  .access-heading {
    padding: 20px 16px 14px;
  }
  .access-tabs {
    padding: 0 12px;
  }
  .access-tabs a {
    min-width: 0;
    flex: 1;
    padding: 13px 8px;
  }
}
</style>
