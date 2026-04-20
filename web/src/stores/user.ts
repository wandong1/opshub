import { defineStore } from 'pinia'
import { login, register, getProfile } from '@/api/auth'
import type { LoginParams, RegisterParams } from '@/api/auth'
import { usePermissionStore } from '@/stores/permission'

interface UserState {
  srehubtoken: string
  userInfo: any
  avatarTimestamp: number
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    srehubtoken: localStorage.getItem('srehubtoken') || '',
    userInfo: null,
    avatarTimestamp: Date.now()
  }),

  getters: {
    isLogin: (state) => !!state.srehubtoken
  },

  actions: {
    // 登录
    async login(params: LoginParams) {
      const res = await login(params)
      this.srehubtoken = res.token
      this.userInfo = res.user
      localStorage.setItem('srehubtoken', res.token)
      return res
    },

    // 注册
    async register(params: RegisterParams) {
      const res = await register(params)
      return res
    },

    // 获取用户信息
    async getProfile() {
      const res = await getProfile()
      this.userInfo = res
      // 更新时间戳，确保头像等资源能刷新
      this.avatarTimestamp = Date.now()
      return res
    },

    // 退出登录
    logout() {
      this.srehubtoken = ''
      this.userInfo = null
      localStorage.removeItem('srehubtoken')
      const permissionStore = usePermissionStore()
      permissionStore.clearPermissions()
    },

    // 更新头像
    updateAvatar(avatarUrl: string) {
      if (this.userInfo) {
        // 创建新对象以触发响应式更新
        this.userInfo = {
          ...this.userInfo,
          avatar: avatarUrl
        }
        this.avatarTimestamp = Date.now()
      }
    }
  }
})
