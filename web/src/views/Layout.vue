<template>
  <el-container class="layout-container">
    <el-aside width="200px">
      <div class="logo">
        <h3>OpsHub</h3>
      </div>

      <el-menu
        :default-active="activeMenu"
        :default-openeds="defaultOpeneds"
        class="el-menu-vertical"
        router
        background-color="#001529"
        text-color="hsla(0,0%,100%,.65)"
        active-text-color="#fff"
        unique-opened
      >
        <template v-for="menu in menuList" :key="menu.ID">
          <!-- 有子菜单的情况 -->
          <el-sub-menu v-if="menu.children && menu.children.length > 0" :index="String(menu.ID)">
            <template #title>
              <el-icon><component :is="getIcon(menu.icon)" /></el-icon>
              <span>{{ menu.name }}</span>
            </template>
            <el-menu-item
              v-for="subMenu in menu.children"
              :key="subMenu.ID"
              :index="subMenu.path"
            >
              <el-icon><component :is="getIcon(subMenu.icon)" /></el-icon>
              <span>{{ subMenu.name }}</span>
            </el-menu-item>
          </el-sub-menu>

          <!-- 没有子菜单的情况 -->
          <el-menu-item v-else :index="menu.path">
            <el-icon><component :is="getIcon(menu.icon)" /></el-icon>
            <span>{{ menu.name }}</span>
          </el-menu-item>
        </template>
      </el-menu>

      <!-- 用户信息区域 - 放在底部 -->
      <div class="user-section">
        <el-dropdown trigger="click" @command="handleUserCommand">
          <div class="user-info-wrapper">
            <div class="user-avatar">
              <el-avatar :size="40" :src="userStore.userInfo?.avatar || ''">
                <el-icon><UserFilled /></el-icon>
              </el-avatar>
            </div>
            <div class="user-details">
              <div class="user-name">{{ userStore.userInfo?.realName || userStore.userInfo?.username }}</div>
              <div class="user-role">{{ userStore.userInfo?.realName ? '管理员' : '普通用户' }}</div>
            </div>
            <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon><User /></el-icon>
                <span>个人信息</span>
              </el-dropdown-item>
              <el-dropdown-item command="logout" divided>
                <el-icon><SwitchButton /></el-icon>
                <span>退出登录</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-aside>

    <el-container>
      <el-header>
        <div class="header-content">
          <div class="breadcrumb">
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
              <el-breadcrumb-item v-if="currentRoute.meta.title">
                {{ currentRoute.meta.title }}
              </el-breadcrumb-item>
            </el-breadcrumb>
          </div>
        </div>
      </el-header>

      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, shallowRef } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import {
  HomeFilled,
  User,
  UserFilled,
  OfficeBuilding,
  Menu,
  SwitchButton,
  ArrowDown,
  Platform,
  Setting,
  Document,
  Tools,
  Monitor,
  FolderOpened,
  Connection,
  Files,
  Lock,
  View,
  Odometer
} from '@element-plus/icons-vue'
import { getUserMenu } from '@/api/menu'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const activeMenu = computed(() => route.path)
const currentRoute = computed(() => route)
const menuList = ref<any[]>([])
const defaultOpeneds = ref<string[]>([])

// 图标映射
const iconMap: Record<string, any> = {
  'HomeFilled': HomeFilled,
  'User': User,
  'UserFilled': UserFilled,
  'OfficeBuilding': OfficeBuilding,
  'Menu': Menu,
  'Platform': Platform,
  'Setting': Setting,
  'Document': Document,
  'Tools': Tools,
  'Monitor': Monitor,
  'FolderOpened': FolderOpened,
  'Connection': Connection,
  'Files': Files,
  'Lock': Lock,
  'View': View,
  'Odometer': Odometer
}

// 获取图标组件
const getIcon = (iconName: string) => {
  return iconMap[iconName] || Menu
}

// 加载菜单
const loadMenu = async () => {
  try {
    const menus = await getUserMenu()
    menuList.value = menus || []
  } catch (error) {
    console.error('加载菜单失败:', error)
    ElMessage.error('加载菜单失败')
  }
}

const handleUserCommand = (command: string) => {
  if (command === 'logout') {
    handleLogout()
  } else if (command === 'profile') {
    router.push('/profile')
  }
}

const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}

onMounted(() => {
  loadMenu()
})
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.el-aside {
  background-color: #001529 !important;
  color: #fff;
  box-shadow: 2px 0 6px rgba(0, 0, 0, 0.1);
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.logo {
  height: 64px;
  line-height: 64px;
  text-align: center;
  background: rgba(255, 255, 255, 0.05);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  margin: 0;
  flex-shrink: 0;
}

.logo h3 {
  margin: 0;
  color: #fff;
  font-weight: 700;
  letter-spacing: 1px;
  font-size: 20px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

/* 用户信息区域 */
.user-section {
  padding: 0;
  flex-shrink: 0;
}

.user-info-wrapper {
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(0, 0, 0, 0.3);
  cursor: pointer;
  transition: background-color 0.3s;
}

.user-info-wrapper:hover {
  background-color: rgba(255, 255, 255, 0.08);
}

.user-avatar :deep(.el-avatar) {
  background-color: #1890ff;
  border: 2px solid rgba(255, 255, 255, 0.2);
}

.user-avatar :deep(.el-icon) {
  font-size: 20px;
  color: #fff;
}

.user-details {
  flex: 1;
  min-width: 0;
}

.user-name {
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-role {
  color: hsla(0, 0%, 100%, 0.45);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.dropdown-icon {
  color: hsla(0, 0%, 100%, 0.45);
  font-size: 14px;
  transition: transform 0.3s;
}

:deep(.el-dropdown:hover .dropdown-icon) {
  transform: rotate(180deg);
}

/* 下拉菜单样式 */
:deep(.el-dropdown-menu) {
  background-color: #ffffff !important;
  border: 1px solid #e4e7ed;
  padding: 4px 0;
}

:deep(.el-dropdown-menu__item) {
  color: #606266 !important;
  line-height: 40px;
  padding: 0 16px;
}

:deep(.el-dropdown-menu__item:hover) {
  background-color: #ecf5ff !important;
  color: #409eff !important;
}

:deep(.el-dropdown-menu__item.is-divided) {
  border-top: 1px solid #e4e7ed;
  margin-top: 4px;
  padding-top: 8px;
}

:deep(.el-dropdown-menu__item .el-icon) {
  color: #606266 !important;
  margin-right: 8px;
  font-size: 16px;
}

:deep(.el-dropdown-menu__item:hover .el-icon) {
  color: #409eff !important;
}

.el-menu {
  border-right: none !important;
  background-color: #001529 !important;
  flex: 1 1 auto;
  overflow-y: auto;
  overflow-x: hidden;
}

/* 覆盖 Element Plus 菜单样式 */
:deep(.el-menu) {
  background-color: #001529 !important;
}

:deep(.el-menu-item) {
  color: hsla(0, 0%, 100%, 0.65) !important;
  background-color: transparent !important;
  font-size: 17px !important;
}

:deep(.el-menu-item:hover) {
  background-color: #1890ff !important;
  color: #fff !important;
}

:deep(.el-menu-item.is-active) {
  background-color: #1890ff !important;
  color: #fff !important;
}

:deep(.el-menu-item .el-icon) {
  color: inherit;
  font-size: 18px !important;
}

/* 子菜单标题样式 */
:deep(.el-sub-menu__title) {
  color: hsla(0, 0%, 100%, 0.65) !important;
  background-color: transparent !important;
  font-size: 17px !important;
}

:deep(.el-sub-menu__title:hover) {
  background-color: #1890ff !important;
  color: #fff !important;
}

:deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  background-color: #1890ff !important;
  color: #fff !important;
}

:deep(.el-sub-menu__title .el-icon) {
  color: inherit;
  font-size: 18px !important;
}

.el-header {
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
}

.header-content {
  width: 100%;
  display: flex;
  align-items: center;
}

.el-main {
  background-color: #f0f2f5;
  padding: 20px;
}
</style>
