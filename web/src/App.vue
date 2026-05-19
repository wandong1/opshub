<template>
  <router-view />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

onMounted(() => {
  // 初始化主题系统
  appStore.init()
})
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

/* ===== 深色模式下的 Element Plus 适配 ===== */
body[arco-theme='dark'] .el-card {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-card__header {
  background-color: #2a2a2b !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-table {
  background-color: #232324 !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-table th,
body[arco-theme='dark'] .el-table tr {
  background-color: #232324 !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-table th.el-table__cell {
  background-color: #2a2a2b !important;
}

body[arco-theme='dark'] .el-table td {
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-table--border,
body[arco-theme='dark'] .el-table--group {
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-table--enable-row-hover .el-table__body tr:hover > td {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

body[arco-theme='dark'] .el-table__empty-text {
  color: rgba(255, 255, 255, 0.6) !important;
}

body[arco-theme='dark'] .el-input__wrapper {
  background-color: #232324 !important;
  box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.15) inset !important;
}

body[arco-theme='dark'] .el-input__inner,
body[arco-theme='dark'] .el-textarea__inner {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-input__inner::placeholder,
body[arco-theme='dark'] .el-textarea__inner::placeholder {
  color: rgba(255, 255, 255, 0.4) !important;
}

body[arco-theme='dark'] .el-select-dropdown {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-select-dropdown__item {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-select-dropdown__item:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

body[arco-theme='dark'] .el-select-dropdown__item.selected {
  color: #4080ff !important;
}

body[arco-theme='dark'] .el-popper {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-dialog {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-dialog__header {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-dialog__title {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-dialog__body {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-dialog__footer {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-form-item__label {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-pagination {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-pagination button {
  background-color: #232324 !important;
  color: rgba(255, 255, 255, 0.9) !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-pagination button:hover {
  color: #4080ff !important;
}

body[arco-theme='dark'] .el-pager li {
  background-color: #232324 !important;
  color: rgba(255, 255, 255, 0.9) !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-pager li:hover {
  color: #4080ff !important;
}

body[arco-theme='dark'] .el-pager li.active {
  color: #4080ff !important;
  background-color: rgba(64, 128, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-tabs__header {
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-tabs__item {
  color: rgba(255, 255, 255, 0.6) !important;
}

body[arco-theme='dark'] .el-tabs__item:hover {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-tabs__item.is-active {
  color: #4080ff !important;
}

body[arco-theme='dark'] .el-tabs__nav-wrap::after {
  background-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-tag {
  background-color: rgba(255, 255, 255, 0.08) !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-breadcrumb__inner {
  color: rgba(255, 255, 255, 0.6) !important;
}

body[arco-theme='dark'] .el-breadcrumb__inner:hover {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-breadcrumb__item:last-child .el-breadcrumb__inner {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-dropdown-menu {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-dropdown-menu__item {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-dropdown-menu__item:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

body[arco-theme='dark'] .el-menu {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-menu-item {
  color: rgba(255, 255, 255, 0.8) !important;
}

body[arco-theme='dark'] .el-menu-item:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

body[arco-theme='dark'] .el-sub-menu__title {
  color: rgba(255, 255, 255, 0.8) !important;
}

body[arco-theme='dark'] .el-sub-menu__title:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

body[arco-theme='dark'] .el-tree {
  background-color: transparent !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-tree-node__content:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

body[arco-theme='dark'] .el-collapse {
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-collapse-item__header {
  background-color: #232324 !important;
  color: rgba(255, 255, 255, 0.9) !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-collapse-item__wrap {
  background-color: #17171a !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-descriptions {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-descriptions__label {
  color: rgba(255, 255, 255, 0.6) !important;
}

body[arco-theme='dark'] .el-descriptions__content {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-empty__description {
  color: rgba(255, 255, 255, 0.6) !important;
}

body[arco-theme='dark'] .el-divider {
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-radio__label,
body[arco-theme='dark'] .el-checkbox__label {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-switch {
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-date-editor {
  background-color: #232324 !important;
}

body[arco-theme='dark'] .el-picker-panel {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-date-table td {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-date-table td.available:hover {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

body[arco-theme='dark'] .el-drawer {
  background-color: #232324 !important;
}

body[arco-theme='dark'] .el-drawer__header {
  color: rgba(255, 255, 255, 0.9) !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-drawer__body {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-loading-mask {
  background-color: rgba(23, 23, 26, 0.8) !important;
}

body[arco-theme='dark'] .el-message-box {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-message-box__title {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-message-box__content {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-notification {
  background-color: #232324 !important;
  border-color: rgba(255, 255, 255, 0.15) !important;
}

body[arco-theme='dark'] .el-notification__title {
  color: rgba(255, 255, 255, 0.9) !important;
}

body[arco-theme='dark'] .el-notification__content {
  color: rgba(255, 255, 255, 0.9) !important;
}

</style>
