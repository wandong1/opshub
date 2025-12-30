<template>
  <router-view />
</template>

<script setup lang="ts">
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body,
#app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial,
    sans-serif;
}

/* 全局字体大小增加 */
html {
  font-size: 20px; /* 增加到20px */
}

body {
  font-size: 1rem; /* 20px */
}

/* 调整Element Plus组件字体大小 */
.el-button {
  font-size: 17px !important;
}

.el-input__inner,
.el-textarea__inner {
  font-size: 17px !important;
}

.el-form-item__label {
  font-size: 17px !important;
}

.el-table {
  font-size: 17px !important;
}

.el-table th,
.el-table td {
  font-size: 17px !important;
}

.el-menu-item {
  font-size: 17px !important;
}

.el-dropdown-menu__item {
  font-size: 17px !important;
}

.el-dialog__title {
  font-size: 20px !important;
}

.el-breadcrumb__inner {
  font-size: 17px !important;
}

.el-tag {
  font-size: 16px !important;
}

.el-pagination {
  font-size: 17px !important;
}

.el-card__header {
  font-size: 18px !important;
}

/* 侧边栏特殊字体 */
.el-aside .user-name {
  font-size: 16px !important;
}

.el-aside .user-role {
  font-size: 14px !important;
}

.logo h3 {
  font-size: 22px !important;
}
</style>
