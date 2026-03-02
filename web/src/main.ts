import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ArcoVue from '@arco-design/web-vue'
import ArcoVueIcon from '@arco-design/web-vue/es/icon'
import '@arco-design/web-vue/dist/arco.css'
import '@/styles/arco-theme.css'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router, { registerPluginRoutes } from './router'
import { pluginManager } from './plugins/manager'
import { vPermission } from '@/directives/permission'

// 导入插件（插件会自动注册到 pluginManager）
import '@/plugins/kubernetes'
import '@/plugins/monitor'
import '@/plugins/nginx'
import '@/plugins/task'
import '@/plugins/test'
import '@/plugins/ssl-cert'

const app = createApp(App)
const pinia = createPinia()

// 注册所有 Element Plus 图标（兼容未迁移页面）
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(pinia)

// 注册全局指令
app.directive('permission', vPermission)

// 自动安装所有已注册的插件
async function installPlugins() {
  const plugins = pluginManager.getAll()
  for (const plugin of plugins) {
    await pluginManager.install(plugin.name, false)
  }
}

// 安装插件并注册路由
installPlugins().then(() => {
  registerPluginRoutes()

  app.use(router)
  app.use(ArcoVue)
  app.use(ArcoVueIcon)
  app.use(ElementPlus, { locale: zhCn })

  app.mount('#app')
})
