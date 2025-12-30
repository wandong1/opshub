import { Plugin } from './types'
import { ElMessage } from 'element-plus'

class PluginManagerImpl {
  private plugins: Map<string, Plugin> = new Map()

  register(plugin: Plugin) {
    this.plugins.set(plugin.name, plugin)
    console.log('Plugin registered:', plugin.name)
  }

  get(name: string): Plugin | undefined {
    return this.plugins.get(name)
  }

  getAll(): Plugin[] {
    return Array.from(this.plugins.values())
  }
}

export const pluginManager = new PluginManagerImpl()
