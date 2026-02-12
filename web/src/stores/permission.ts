import { defineStore } from 'pinia'

export const usePermissionStore = defineStore('permission', {
  state: () => ({
    buttonCodes: new Set<string>(),
    loaded: false,
    isAdmin: false
  }),
  actions: {
    loadPermissions(menuTree: any[], isAdmin = false) {
      this.buttonCodes.clear()
      this.isAdmin = isAdmin
      this._extract(menuTree)
      this.loaded = true
    },
    _extract(menus: any[]) {
      for (const m of menus) {
        if (m.type === 3 && m.code) this.buttonCodes.add(m.code)
        if (m.children?.length) this._extract(m.children)
      }
    },
    hasPermission(code: string): boolean {
      if (!this.loaded) return true
      if (this.isAdmin) return true
      return this.buttonCodes.has(code)
    },
    clearPermissions() {
      this.buttonCodes.clear()
      this.loaded = false
      this.isAdmin = false
    }
  }
})
