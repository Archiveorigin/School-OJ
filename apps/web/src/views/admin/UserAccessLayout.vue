<template>
  <section class="user-access-layout">
    <header class="access-heading">
      <div>
        <span>IDENTITY & ACCESS</span>
        <h1>用户与权限</h1>
        <p>基础角色决定系统身份，附加权限在角色之上独立授予。</p>
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
  border-bottom: 1px solid #dce4ef;
  background: #fff;
}
.access-heading span {
  color: #135ecb;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.14em;
}
.access-heading h1 {
  margin: 6px 0 4px;
  color: #173765;
  font-size: 26px;
}
.access-heading p {
  margin: 0;
  color: #6f7f96;
  font-size: 13px;
}
.access-tabs {
  display: flex;
  gap: 4px;
  padding: 0 28px;
  border-bottom: 1px solid #dce4ef;
  background: #fff;
}
.access-tabs a {
  position: relative;
  min-width: 110px;
  padding: 14px 18px;
  color: #64748b;
  text-align: center;
  font-weight: 700;
}
.access-tabs a.active {
  color: #135ecb;
}
.access-tabs a.active::after {
  position: absolute;
  right: 12px;
  bottom: -1px;
  left: 12px;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: #135ecb;
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
