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
  font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB',
    'Microsoft YaHei', 'Helvetica Neue', Arial, sans-serif;
}

html {
  font-size: 14px;
  overflow-y: scroll;
}

body {
  font-size: 14px;
  color: var(--ops-text-primary, #1d2129);
  background-color: #fff;
}

/* ===== Element Plus 兼容覆盖（未迁移页面） ===== */
body.el-popup-parent--hidden {
  padding-right: 0 !important;
  overflow: hidden !important;
  width: 100% !important;
}

.el-overlay {
  scrollbar-width: none !important;
}
.el-overlay::-webkit-scrollbar {
  display: none !important;
  width: 0 !important;
}
.el-overlay-dialog {
  overflow: auto !important;
  scrollbar-width: none !important;
}
.el-overlay-dialog::-webkit-scrollbar {
  display: none !important;
  width: 0 !important;
}

/* Element Plus 字体大小保持一致（兼容未迁移页面） */
.el-button { font-size: 14px !important; }
.el-input__inner, .el-textarea__inner { font-size: 14px !important; }
.el-form-item__label { font-size: 14px !important; }
.el-table { font-size: 14px !important; }
.el-table th, .el-table td { font-size: 14px !important; }
.el-menu-item { font-size: 14px !important; }
.el-dropdown-menu__item { font-size: 14px !important; }
.el-dialog__title { font-size: 16px !important; }
.el-breadcrumb__inner { font-size: 14px !important; }
.el-tag { font-size: 12px !important; }
.el-pagination { font-size: 14px !important; }
.el-card__header { font-size: 14px !important; }
</style>
