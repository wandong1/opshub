import { defineStore } from 'pinia'
import { login, register, getProfile } from '@/api/auth'
import type { LoginParams, RegisterParams } from '@/api/auth'

interface UserState {
  token: string
  userInfo: any
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    token: localStorage.getItem('token') || '',
    userInfo: null
  }),

  getters: {
    isLogin: (state) => !!state.token
  },

  actions: {
    // 登录
    async login(params: LoginParams) {
      const res = await login(params)
      console.log('登录响应:', res)
      this.token = res.token
      this.userInfo = res.user
      localStorage.setItem('token', res.token)
      console.log('Token已保存:', res.token)
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
      return res
    },

    // 退出登录
    logout() {
      this.token = ''
      this.userInfo = null
      localStorage.removeItem('token')
    }
  }
})
