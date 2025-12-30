import { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

class KubernetesPlugin implements Plugin {
  name = 'kubernetes'
  description = 'Kubernetes management'
  version = '1.0.0'
  author = 'OpsHub'

  install() {
    console.log('Kubernetes install')
  }

  uninstall() {
    console.log('Kubernetes uninstall')
  }

  getMenus(): PluginMenuConfig[] {
    return [
      { name: 'Kubernetes', path: '/kubernetes', icon: 'Platform', sort: 100, hidden: false, parentPath: '' },
      { name: 'Clusters', path: '/kubernetes/clusters', icon: 'OfficeBuilding', sort: 1, hidden: false, parentPath: '/kubernetes' }
    ]
  }

  getRoutes(): PluginRouteConfig[] {
    return [{
      path: '/kubernetes',
      name: 'Kubernetes',
      component: () => import('@/views/kubernetes/Index.vue'),
      meta: { title: 'Kubernetes' },
      children: [
        { path: 'clusters', name: 'K8sClusters', component: () => import('@/views/kubernetes/Clusters.vue'), meta: { title: 'Clusters' } }
      ]
    }]
  }
}

const plugin = new KubernetesPlugin()
pluginManager.register(plugin)
export default plugin
