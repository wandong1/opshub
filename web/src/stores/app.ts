import { defineStore } from 'pinia'

type ThemeMode = 'light' | 'dark' | 'auto'
type LayoutMode = 'sidebar' | 'topbar'

interface AppState {
  theme: ThemeMode
  layout: LayoutMode
  sidebarCollapsed: boolean
  actualTheme: 'light' | 'dark' // 实际应用的主题（auto 模式下根据系统主题计算）
}

const STORAGE_KEYS = {
  THEME: 'opshub_theme',
  LAYOUT: 'opshub_layout',
  SIDEBAR_COLLAPSED: 'opshub_sidebar_collapsed'
}

export const useAppStore = defineStore('app', {
  state: (): AppState => ({
    theme: (localStorage.getItem(STORAGE_KEYS.THEME) as ThemeMode) || 'light',
    layout: (localStorage.getItem(STORAGE_KEYS.LAYOUT) as LayoutMode) || 'sidebar',
    sidebarCollapsed: localStorage.getItem(STORAGE_KEYS.SIDEBAR_COLLAPSED) === 'true',
    actualTheme: 'light'
  }),

  getters: {
    isDark: (state) => state.actualTheme === 'dark',
    isSidebarLayout: (state) => state.layout === 'sidebar',
    isTopbarLayout: (state) => state.layout === 'topbar'
  },

  actions: {
    // 设置主题模式
    setTheme(mode: ThemeMode) {
      this.theme = mode
      localStorage.setItem(STORAGE_KEYS.THEME, mode)
      this.applyTheme()
    },

    // 设置布局模式
    setLayout(mode: LayoutMode) {
      this.layout = mode
      localStorage.setItem(STORAGE_KEYS.LAYOUT, mode)
    },

    // 切换侧边栏折叠状态
    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed
      localStorage.setItem(STORAGE_KEYS.SIDEBAR_COLLAPSED, String(this.sidebarCollapsed))
    },

    // 设置侧边栏折叠状态
    setSidebarCollapsed(collapsed: boolean) {
      this.sidebarCollapsed = collapsed
      localStorage.setItem(STORAGE_KEYS.SIDEBAR_COLLAPSED, String(collapsed))
    },

    // 应用主题到 DOM
    applyTheme() {
      let targetTheme: 'light' | 'dark' = 'light'

      if (this.theme === 'auto') {
        // 跟随系统主题
        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
        targetTheme = mediaQuery.matches ? 'dark' : 'light'
      } else {
        targetTheme = this.theme
      }

      this.actualTheme = targetTheme

      // 应用 Arco Design 深色主题
      if (targetTheme === 'dark') {
        document.body.setAttribute('arco-theme', 'dark')
      } else {
        document.body.removeAttribute('arco-theme')
      }
    },

    // 监听系统主题变化
    watchSystemTheme() {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

      const handleChange = (e: MediaQueryListEvent | MediaQueryList) => {
        if (this.theme === 'auto') {
          this.applyTheme()
        }
      }

      // 兼容不同浏览器的事件监听方式
      if (mediaQuery.addEventListener) {
        mediaQuery.addEventListener('change', handleChange)
      } else if (mediaQuery.addListener) {
        // Safari 旧版本兼容
        mediaQuery.addListener(handleChange)
      }

      // 返回清理函数
      return () => {
        if (mediaQuery.removeEventListener) {
          mediaQuery.removeEventListener('change', handleChange)
        } else if (mediaQuery.removeListener) {
          mediaQuery.removeListener(handleChange)
        }
      }
    },

    // 重置所有设置
    resetSettings() {
      this.setTheme('light')
      this.setLayout('sidebar')
      this.setSidebarCollapsed(false)
    },

    // 初始化（在 App.vue 中调用）
    init() {
      this.applyTheme()
      this.watchSystemTheme()
    }
  }
})
