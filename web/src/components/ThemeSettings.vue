<template>
  <a-drawer
    :visible="visible"
    :width="320"
    :footer="false"
    unmount-on-close
    @cancel="handleClose"
  >
    <template #title>
      <div class="drawer-title">
        <icon-settings />
        <span>界面设置</span>
      </div>
    </template>

    <div class="theme-settings">
      <!-- 主题模式 -->
      <div class="setting-section">
        <div class="section-title">主题模式</div>
        <div class="theme-options">
          <div
            class="theme-option"
            :class="{ active: appStore.theme === 'light' }"
            @click="handleThemeChange('light')"
          >
            <div class="option-icon">
              <icon-sun />
            </div>
            <div class="option-label">浅色</div>
          </div>

          <div
            class="theme-option"
            :class="{ active: appStore.theme === 'dark' }"
            @click="handleThemeChange('dark')"
          >
            <div class="option-icon">
              <icon-moon />
            </div>
            <div class="option-label">深色</div>
          </div>

          <div
            class="theme-option"
            :class="{ active: appStore.theme === 'auto' }"
            @click="handleThemeChange('auto')"
          >
            <div class="option-icon">
              <icon-computer />
            </div>
            <div class="option-label">跟随系统</div>
          </div>
        </div>
      </div>

      <!-- 布局模式 -->
      <div class="setting-section">
        <div class="section-title">布局模式</div>
        <div class="layout-options">
          <div
            class="layout-option"
            :class="{ active: appStore.layout === 'sidebar' }"
            @click="handleLayoutChange('sidebar')"
          >
            <div class="option-preview sidebar-preview">
              <div class="preview-sidebar"></div>
              <div class="preview-content">
                <div class="preview-header"></div>
                <div class="preview-body"></div>
              </div>
            </div>
            <div class="option-label">侧边栏</div>
          </div>

          <div
            class="layout-option"
            :class="{ active: appStore.layout === 'topbar' }"
            @click="handleLayoutChange('topbar')"
          >
            <div class="option-preview topbar-preview">
              <div class="preview-header"></div>
              <div class="preview-body"></div>
            </div>
            <div class="option-label">顶部导航</div>
          </div>
        </div>
      </div>

      <!-- 重置按钮 -->
      <div class="setting-section">
        <a-button long @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置为默认设置
        </a-button>
      </div>
    </div>
  </a-drawer>
</template>

<script setup lang="ts">
import { useAppStore } from '@/stores/app'
import { Message } from '@arco-design/web-vue'
import {
  IconSettings,
  IconSun,
  IconMoon,
  IconComputer,
  IconRefresh
} from '@arco-design/web-vue/es/icon'

interface Props {
  visible: boolean
}

interface Emits {
  (e: 'update:visible', value: boolean): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const appStore = useAppStore()

const handleClose = () => {
  emit('update:visible', false)
}

const handleThemeChange = (theme: 'light' | 'dark' | 'auto') => {
  appStore.setTheme(theme)
  Message.success(`已切换到${theme === 'light' ? '浅色' : theme === 'dark' ? '深色' : '跟随系统'}主题`)
}

const handleLayoutChange = (layout: 'sidebar' | 'topbar') => {
  appStore.setLayout(layout)
  Message.success(`已切换到${layout === 'sidebar' ? '侧边栏' : '顶部导航'}布局`)
  // 刷新页面以应用新布局
  setTimeout(() => {
    window.location.reload()
  }, 500)
}

const handleReset = () => {
  appStore.resetSettings()
  Message.success('已重置为默认设置')
  setTimeout(() => {
    window.location.reload()
  }, 500)
}
</script>

<style scoped>
.drawer-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
}

.theme-settings {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.setting-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--ops-text-primary);
}

/* 主题选项 */
.theme-options {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.theme-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 8px;
  border: 2px solid var(--ops-border-color);
  border-radius: var(--ops-border-radius-md);
  cursor: pointer;
  transition: all 0.2s;
}

.theme-option:hover {
  border-color: var(--ops-primary);
  background-color: var(--ops-primary-bg);
}

.theme-option.active {
  border-color: var(--ops-primary);
  background-color: var(--ops-primary-bg);
}

.option-icon {
  font-size: 24px;
  color: var(--ops-text-secondary);
  transition: color 0.2s;
}

.theme-option.active .option-icon {
  color: var(--ops-primary);
}

.option-label {
  font-size: 13px;
  color: var(--ops-text-secondary);
  transition: color 0.2s;
}

.theme-option.active .option-label {
  color: var(--ops-primary);
  font-weight: 500;
}

/* 布局选项 */
.layout-options {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.layout-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px;
  border: 2px solid var(--ops-border-color);
  border-radius: var(--ops-border-radius-md);
  cursor: pointer;
  transition: all 0.2s;
}

.layout-option:hover {
  border-color: var(--ops-primary);
  background-color: var(--ops-primary-bg);
}

.layout-option.active {
  border-color: var(--ops-primary);
  background-color: var(--ops-primary-bg);
}

/* 布局预览 */
.option-preview {
  width: 100%;
  height: 80px;
  border-radius: 4px;
  overflow: hidden;
  background-color: var(--ops-content-bg);
  border: 1px solid var(--ops-border-color);
}

/* 侧边栏布局预览 */
.sidebar-preview {
  display: flex;
}

.preview-sidebar {
  width: 30%;
  background-color: var(--ops-sidebar-bg);
}

.preview-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.sidebar-preview .preview-header {
  height: 20%;
  background-color: var(--ops-header-bg);
  border-bottom: 1px solid var(--ops-border-color);
}

.sidebar-preview .preview-body {
  flex: 1;
  background-color: var(--ops-content-bg);
}

/* 顶部导航布局预览 */
.topbar-preview {
  display: flex;
  flex-direction: column;
}

.topbar-preview .preview-header {
  height: 25%;
  background-color: var(--ops-header-bg);
  border-bottom: 1px solid var(--ops-border-color);
}

.topbar-preview .preview-body {
  flex: 1;
  background-color: var(--ops-content-bg);
}
</style>
