<template>
  <a-layout class="ops-layout topbar-layout">
    <!-- 顶部导航栏 -->
    <a-layout-header class="ops-header topbar-header">
      <div class="header-container">
        <!-- Logo 区域 -->
        <div class="header-logo">
          <img v-if="systemStore.systemLogo" :src="systemStore.systemLogo" alt="Logo" class="logo-image" />
          <span class="logo-text">
            <span class="logo-first">{{ systemNameFirst }}</span>
            <span class="logo-second">{{ systemNameSecond }}</span>
          </span>
        </div>

        <!-- 水平菜单 -->
        <div class="header-menu">
          <a-menu
            mode="horizontal"
            :selected-keys="[activeMenu]"
            @menu-item-click="handleMenuClick"
          >
            <template v-for="menu in menuList" :key="menu.ID">
              <!-- 有子菜单 -->
              <a-sub-menu
                v-if="menu.children && menu.children.filter((m: any) => m.type !== 3).length > 0"
                :key="String(menu.ID)"
              >
                <template #icon><component :is="getArcoIcon(menu.icon)" /></template>
                <template #title>{{ menu.name }}</template>
                <a-menu-item
                  v-for="sub in menu.children.filter((m: any) => m.type !== 3)"
                  :key="sub.path"
                  :disabled="sub.status === 0"
                >
                  <template #icon><component :is="getArcoIcon(sub.icon)" /></template>
                  {{ sub.name }}
                </a-menu-item>
              </a-sub-menu>

              <!-- 无子菜单 -->
              <a-menu-item
                v-else
                :key="menu.path || String(menu.ID)"
                :disabled="menu.status === 0"
              >
                <template #icon><component :is="getArcoIcon(menu.icon)" /></template>
                {{ menu.name }}
              </a-menu-item>
            </template>
          </a-menu>
        </div>

        <!-- 右侧工具栏 -->
        <div class="header-right">
          <!-- 主题切换按钮 -->
          <a-button type="text" class="theme-toggle-btn" @click="showThemeSettings = true">
            <icon-sun v-if="appStore.theme === 'light'" />
            <icon-moon v-else-if="appStore.theme === 'dark'" />
            <icon-computer v-else />
          </a-button>

          <!-- 用户下拉菜单 -->
          <a-dropdown trigger="click" @select="handleUserCommand">
            <div class="header-user">
              <a-avatar :size="32" :style="{ backgroundColor: '#165dff' }">
                <img v-if="avatarUrl" :src="avatarUrl" />
                <template v-else>{{ (userStore.userInfo?.realName || userStore.userInfo?.username || 'U').charAt(0) }}</template>
              </a-avatar>
              <span class="header-user-name">{{ userStore.userInfo?.realName || userStore.userInfo?.username }}</span>
              <icon-down class="header-arrow" />
            </div>
            <template #content>
              <a-doption value="profile">
                <template #icon><icon-user /></template>
                个人信息
              </a-doption>
              <a-doption value="logout">
                <template #icon><icon-export /></template>
                退出登录
              </a-doption>
            </template>
          </a-dropdown>
        </div>
      </div>
    </a-layout-header>

    <!-- 内容区 -->
    <a-layout-content class="ops-content" :class="{ 'ops-content--full': currentRoute.meta?.fullContent }">
      <NoPermission v-if="hasNoPermission" />
      <router-view v-else />
    </a-layout-content>

    <!-- 主题设置抽屉 -->
    <ThemeSettings v-model:visible="showThemeSettings" />
  </a-layout>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, type Component } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useSystemStore } from '@/stores/system'
import { useAppStore } from '@/stores/app'
import { Message } from '@arco-design/web-vue'
import NoPermission from '@/views/NoPermission.vue'
import ThemeSettings from '@/components/ThemeSettings.vue'
import {
  IconHome, IconUser, IconUserGroup, IconSettings, IconMenu,
  IconDesktop, IconFile, IconTool, IconDashboard, IconFolder,
  IconStorage, IconLock, IconApps, IconCloud, IconList,
  IconCommon, IconComputer, IconCode, IconCommand, IconRobot,
  IconSchedule, IconSafe, IconThunderbolt, IconNotification,
  IconDown, IconExport, IconSun, IconMoon
} from '@arco-design/web-vue/es/icon'
import { getUserMenu } from '@/api/menu'
import { pluginManager } from '@/plugins/manager'
import { usePermissionStore } from '@/stores/permission'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const systemStore = useSystemStore()
const appStore = useAppStore()

const showThemeSettings = ref(false)

// Element Plus 图标名 → Arco 图标组件映射
const arcoIconMap: Record<string, Component> = {
  'HomeFilled': IconHome,
  'User': IconUser,
  'UserFilled': IconUser,
  'OfficeBuilding': IconCommon,
  'Menu': IconMenu,
  'Platform': IconDesktop,
  'Setting': IconSettings,
  'Document': IconFile,
  'Tools': IconTool,
  'Monitor': IconDashboard,
  'FolderOpened': IconFolder,
  'Connection': IconStorage,
  'Files': IconFile,
  'Lock': IconLock,
  'View': IconApps,
  'Odometer': IconDashboard,
  'Tickets': IconList,
  'List': IconList,
  'Grid': IconApps,
  'Cloudy': IconCloud,
  'Grape': IconThunderbolt,
  'House': IconHome,
  'Computer': IconComputer,
  'Code': IconCode,
  'Command': IconCommand,
  'Robot': IconRobot,
  'Schedule': IconSchedule,
  'Safe': IconSafe,
  'Notification': IconNotification,
}

const getArcoIcon = (iconName: string) => {
  return arcoIconMap[iconName] || IconMenu
}

const systemNameFirst = computed(() => {
  const name = systemStore.systemName || 'OpsHub'
  const match = name.match(/^([A-Z][a-z]*)(.*)$/)
  if (match && match[2]) return match[1]
  const mid = Math.ceil(name.length / 2)
  return name.substring(0, mid)
})

const systemNameSecond = computed(() => {
  const name = systemStore.systemName || 'OpsHub'
  const match = name.match(/^([A-Z][a-z]*)(.*)$/)
  if (match && match[2]) return match[2]
  const mid = Math.ceil(name.length / 2)
  return name.substring(mid)
})

const activeMenu = computed(() => {
  if (route.meta?.activeMenu) return route.meta.activeMenu as string
  return route.path
})

const avatarUrl = computed(() => {
  const avatar = userStore.userInfo?.avatar || ''
  if (!avatar) return ''
  if (avatar.startsWith('data:')) return avatar
  const separator = avatar.includes('?') ? '&' : '?'
  return `${avatar}${separator}t=${userStore.avatarTimestamp}`
})

const currentRoute = computed(() => route)

const menuList = ref<any[]>([])
const hasNoPermission = ref(false)

// 复用 Layout.vue 的菜单构建逻辑
const buildPluginMenus = async (authorizedPaths: Set<string>) => {
  const pluginMenus: any[] = []
  const allPlugins = pluginManager.getAll()
  const roles = userStore.userInfo?.roles || []
  const isSuperAdmin = roles.some((r: any) => r.code === 'admin')

  let enabledPluginNames: Set<string> = new Set()
  try {
    const { listPlugins } = await import('@/api/plugin')
    const backendPlugins = await listPlugins()
    enabledPluginNames = new Set(
      backendPlugins.filter((p: any) => p.enabled).map((p: any) => p.name)
    )
  } catch {
    const installedPlugins = pluginManager.getInstalled()
    enabledPluginNames = new Set(installedPlugins.map(p => p.name))
  }

  const PLUGIN_MENU_SORT_KEY = 'opshub_plugin_menu_sort'
  const customSort: Map<string, number> = (() => {
    try {
      const stored = localStorage.getItem(PLUGIN_MENU_SORT_KEY)
      if (stored) return new Map(Object.entries(JSON.parse(stored)))
    } catch {}
    return new Map()
  })()

  allPlugins.forEach(plugin => {
    if (!enabledPluginNames.has(plugin.name)) return
    if (!plugin.getMenus) return
    const menus = plugin.getMenus()
    menus.forEach(menu => {
      if (!isSuperAdmin && !authorizedPaths.has(menu.path)) return
      const sort = customSort.get(menu.path) ?? menu.sort
      pluginMenus.push({
        ID: menu.path, name: menu.name, path: menu.path,
        icon: menu.icon, sort, hidden: menu.hidden, parentPath: menu.parentPath,
      })
    })
  })
  return pluginMenus
}

const buildMenuTree = (menus: any[]) => {
  const filteredMenus = menus.filter(menu => {
    if (menu.type === 3) return false
    const isVisible = menu.visible === undefined || menu.visible === 1
    return isVisible
  })

  const uniqueMenus: any[] = []
  const seenSignatures = new Set<string>()
  const seenPaths = new Set<string>()
  for (const menu of filteredMenus) {
    let parentKey = 'root'
    if (menu.parentId !== undefined && menu.parentId !== 0) {
      parentKey = `parent_${menu.parentId}`
    } else if (menu.parentPath !== undefined && menu.parentPath !== '' && menu.parentPath !== '/') {
      parentKey = menu.parentPath
    }
    const signature = `${menu.name}_${parentKey}`
    if (seenSignatures.has(signature)) continue
    if (menu.path && seenPaths.has(menu.path)) continue
    seenSignatures.add(signature)
    if (menu.path) seenPaths.add(menu.path)
    uniqueMenus.push(menu)
  }

  const menuMap = new Map()
  const pathToMenuMap = new Map()
  uniqueMenus.forEach(menu => {
    const menuId = menu.ID || menu.id || menu.path
    if (!menuId) return
    const { children, ...menuWithoutChildren } = menu
    menuMap.set(menuId, menuWithoutChildren)
    if (menu.path && menu.path.startsWith('/')) {
      pathToMenuMap.set(menu.path, menuWithoutChildren)
    }
  })

  const tree: any[] = []
  filteredMenus.forEach(menu => {
    const menuId = menu.ID || menu.id || menu.path
    const menuItem = menuMap.get(menuId)
    if (!menuItem) return

    let parentId = null
    if (menu.parentPath !== undefined) {
      parentId = menu.parentPath || null
    } else if (menu.parentId !== undefined) {
      parentId = menu.parentId === 0 ? null : menu.parentId
    }

    if (parentId && menuMap.has(parentId)) {
      const parent = menuMap.get(parentId)
      if (!parent.children) parent.children = []
      parent.children.push(menuItem)
    } else if (parentId && pathToMenuMap.has(parentId)) {
      const parent = pathToMenuMap.get(parentId)
      if (!parent.children) parent.children = []
      parent.children.push(menuItem)
    } else if (parentId) {
      tree.push(menuItem)
    } else {
      tree.push(menuItem)
    }
  })

  const sortMenus = (items: any[]) => {
    items.sort((a: any, b: any) => (a.sort || 0) - (b.sort || 0))
    items.forEach((item: any) => {
      if (item.children?.length > 0) sortMenus(item.children)
    })
  }
  sortMenus(tree)

  const cleanEmptyChildren = (nodes: any[]) => {
    for (const node of nodes) {
      if (Array.isArray(node.children) && node.children.length === 0) {
        delete node.children
        node.hasChildren = false
      } else if (Array.isArray(node.children) && node.children.length > 0) {
        node.hasChildren = true
        cleanEmptyChildren(node.children)
      } else {
        if (node.children) delete node.children
        node.hasChildren = false
      }
    }
  }
  cleanEmptyChildren(tree)
  return tree
}

const loadMenu = async () => {
  try {
    menuList.value = []
    const systemMenus = await getUserMenu() || []

    const permissionStore = usePermissionStore()
    const roles = userStore.userInfo?.roles || []
    const isSuperAdmin = roles.some((r: any) => r.code === 'admin')
    permissionStore.loadPermissions(systemMenus, isSuperAdmin)

    const extractPaths = (menus: any[]): Set<string> => {
      const paths = new Set<string>()
      const traverse = (items: any[]) => {
        items.forEach(item => {
          if (item.path) paths.add(item.path)
          if (item.children?.length > 0) traverse(item.children)
        })
      }
      traverse(menus)
      return paths
    }
    const allAuthorizedPaths = extractPaths(systemMenus)
    const pluginMenus = await buildPluginMenus(allAuthorizedPaths)

    const pluginProvidedMenuCodes = new Set([
      'kubernetes_application_diagnosis', 'kubernetes_cluster_inspection',
      'monitor_domain', 'monitor_alert_channels', 'monitor_alert_receivers',
      'monitor_alert_logs', 'task_templates', 'task_execute',
      'task_file_distribution', 'task_execution_history',
      'kubernetes_clusters', 'kubernetes_nodes',
      'kubernetes_namespaces', 'kubernetes_workloads', 'kubernetes_network',
      'kubernetes_config', 'kubernetes_storage', 'kubernetes_access',
      'kubernetes_audit',
    ])

    const flattenMenus = (menus: any[], result: any[] = []) => {
      menus.forEach(menu => {
        if (menu.code && pluginProvidedMenuCodes.has(menu.code)) return
        const { children, ...menuWithoutChildren } = menu
        result.push(menuWithoutChildren)
        if (children?.length > 0) flattenMenus(children, result)
      })
      return result
    }

    const flatSystemMenus = flattenMenus(systemMenus)
    const allMenus = [...flatSystemMenus, ...pluginMenus]
    menuList.value = buildMenuTree(allMenus)

    if (!isSuperAdmin && menuList.value.length === 0) {
      hasNoPermission.value = true
    } else {
      hasNoPermission.value = false
    }
  } catch {
    Message.error('加载菜单失败')
  }
}

const handleMenuClick = (key: string) => {
  if (key && key.startsWith('/')) {
    router.push(key)
  }
}

const handleUserCommand = (value: string | number | Record<string, any> | undefined) => {
  if (value === 'logout') {
    userStore.logout()
    router.push('/login')
  } else if (value === 'profile') {
    router.push('/profile')
  }
}

const handlePluginChange = () => { loadMenu() }

onUnmounted(() => {
  window.removeEventListener('plugins-changed', handlePluginChange)
})

onMounted(async () => {
  if (!systemStore.loaded) await systemStore.loadFullConfig()
  if (!userStore.userInfo) {
    try { await userStore.getProfile() } catch {}
  }
  await new Promise(resolve => setTimeout(resolve, 100))
  loadMenu()
  window.removeEventListener('plugins-changed', handlePluginChange)
  window.addEventListener('plugins-changed', handlePluginChange)
})
</script>

<style scoped>
.ops-layout.topbar-layout {
  height: 100vh;
}

/* ===== 顶部导航栏 ===== */
.topbar-header {
  height: var(--ops-header-height, 60px);
  background: var(--ops-header-bg, #fff);
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
  padding: 0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.header-container {
  height: 100%;
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 20px;
}

/* Logo */
.header-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.logo-image {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.logo-text {
  font-size: 20px;
  font-weight: 600;
  white-space: nowrap;
}

.logo-first {
  color: var(--ops-text-primary);
}

.logo-second {
  color: var(--ops-primary, #165dff);
}

/* 水平菜单 */
.header-menu {
  flex: 1;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
}

.header-menu::-webkit-scrollbar {
  height: 0;
}

.header-menu :deep(.arco-menu-horizontal) {
  background: transparent;
  border-bottom: none;
}

.header-menu :deep(.arco-menu-item),
.header-menu :deep(.arco-menu-pop-header) {
  padding: 0 16px;
}

/* 右侧工具栏 */
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.theme-toggle-btn {
  font-size: 18px;
  color: var(--ops-text-secondary);
  transition: color 0.2s;
}

.theme-toggle-btn:hover {
  color: var(--ops-primary);
}

.header-user {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--ops-border-radius-sm, 4px);
  transition: background-color 0.2s;
}

.header-user:hover {
  background-color: rgba(0, 0, 0, 0.04);
}

body[arco-theme='dark'] .header-user:hover {
  background-color: rgba(255, 255, 255, 0.08);
}

.header-user-name {
  font-size: 14px;
  color: var(--ops-text-primary);
  font-weight: 500;
  max-width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-arrow {
  color: var(--ops-text-tertiary);
  font-size: 12px;
}

/* ===== 内容区 ===== */
.ops-content {
  background: var(--ops-content-bg, #f7f8fa);
  padding: 20px;
  overflow-y: auto;
}

.ops-content--full {
  padding: 0;
  overflow: hidden;
}
</style>
