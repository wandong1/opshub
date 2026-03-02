<template>
  <div class="access-control-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-lock />
        </div>
        <div>
          <h2 class="page-title">访问控制</h2>
          <p class="page-subtitle">管理 Kubernetes 集群的访问控制和权限</p>
        </div>
      </div>
      <div class="header-actions">
        <a-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          class="cluster-select"
          @change="handleClusterChange"
        >
          <template #prefix>
            <icon-apps />
          </template>
          <a-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </a-select>
        <a-button type="primary" @click="loadData">
          <icon-refresh />
          刷新
        </a-button>
      </div>
    </div>

    <!-- 访问控制类型标签 -->
    <div class="access-types-bar">
      <div
        v-for="type in accessTypes"
        :key="type.value"
        :class="['type-tab', { active: activeTab === type.value }]"
        @click="handleTabChange(type.value)"
      >
          <component :is="type.icon" />
        <span class="type-label">{{ type.label }}</span>
        <span v-if="type.count !== undefined" class="type-count">({{ type.count }})</span>
      </div>
    </div>

    <!-- 操作栏 -->
    <div class="action-bar" v-if="selectedClusterId">
      <div class="search-section">
        <a-input
          v-model="searchName"
          placeholder="搜索名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <icon-search />
          </template>
        </a-input>
        <a-select
          v-model="selectedNamespace"
          placeholder="选择命名空间"
          class="namespace-select"
          @change="handleNamespaceChange"
        >
          <template #prefix>
            <icon-folder />
          </template>
          <a-option label="所有命名空间" value="" />
          <a-option
            v-for="ns in namespaceList"
            :key="ns.name"
            :label="ns.name"
            :value="ns.name"
          />
        </a-select>
      </div>

      <div class="action-buttons">
        <a-button type="primary" @click="handleCreate">
          <icon-plus />
          {{ getCreateButtonText() }}
        </a-button>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="content-wrapper">
      <!-- ServiceAccounts -->
      <ServiceAccountsTab
        v-show="activeTab === 'serviceaccounts' && selectedClusterId"
        ref="serviceAccountsTabRef"
        :cluster-id="selectedClusterId"
        :namespace="selectedNamespace"
        :search-name="searchName"
        @count-update="(count) => updateCount('serviceaccounts', count)"
      />

      <!-- Roles -->
      <RolesTab
        v-show="activeTab === 'roles' && selectedClusterId"
        ref="rolesTabRef"
        :cluster-id="selectedClusterId"
        :namespace="selectedNamespace || ''"
        :search-name="searchName"
        @count-update="(count) => updateCount('roles', count)"
      />

      <!-- RoleBindings -->
      <RoleBindingsTab
        v-show="activeTab === 'rolebindings' && selectedClusterId"
        ref="roleBindingsTabRef"
        :cluster-id="selectedClusterId"
        :namespace="selectedNamespace || ''"
        :search-name="searchName"
        @count-update="(count) => updateCount('rolebindings', count)"
      />

      <!-- ClusterRoles -->
      <ClusterRolesTab
        v-show="activeTab === 'clusterroles' && selectedClusterId"
        ref="clusterRolesTabRef"
        :cluster-id="selectedClusterId"
        :search-name="searchName"
        @count-update="(count) => updateCount('clusterroles', count)"
      />

      <!-- ClusterRoleBindings -->
      <ClusterRoleBindingsTab
        v-show="activeTab === 'clusterrolebindings' && selectedClusterId"
        ref="clusterRoleBindingsTabRef"
        :cluster-id="selectedClusterId"
        :search-name="searchName"
        @count-update="(count) => updateCount('clusterrolebindings', count)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getClusterList, getNamespaces, type Cluster, type NamespaceInfo } from '@/api/kubernetes'
import {
  IconUser,
  IconSafe,
  IconLink,
  IconUserGroup,
  IconShareInternal
} from '@arco-design/web-vue/es/icon'
import ServiceAccountsTab from './access-control/ServiceAccountsTab.vue'
import RolesTab from './access-control/RolesTab.vue'
import RoleBindingsTab from './access-control/RoleBindingsTab.vue'
import ClusterRolesTab from './access-control/ClusterRolesTab.vue'
import ClusterRoleBindingsTab from './access-control/ClusterRoleBindingsTab.vue'

// 访问控制类型定义
interface AccessType {
  label: string
  value: string
  icon: any
  count: number
}

const accessTypes = ref<AccessType[]>([
  { label: 'ServiceAccounts', value: 'serviceaccounts', icon: IconUser, count: 0 },
  { label: 'Roles', value: 'roles', icon: IconSafe, count: 0 },
  { label: 'RoleBindings', value: 'rolebindings', icon: IconLink, count: 0 },
  { label: 'ClusterRoles', value: 'clusterroles', icon: IconUserGroup, count: 0 },
  { label: 'ClusterRoleBindings', value: 'clusterrolebindings', icon: IconShareInternal, count: 0 },
])

const activeTab = ref('serviceaccounts')
const selectedClusterId = ref<number>()
const selectedNamespace = ref<string>()
const clusterList = ref<Cluster[]>([])
const namespaceList = ref<NamespaceInfo[]>([])
const searchName = ref('')

// 子组件引用
const serviceAccountsTabRef = ref()
const rolesTabRef = ref()
const roleBindingsTabRef = ref()
const clusterRolesTabRef = ref()
const clusterRoleBindingsTabRef = ref()

// 获取新增按钮文本
const getCreateButtonText = () => {
  const buttonMap: Record<string, string> = {
    serviceaccounts: '新增 ServiceAccount',
    roles: '新增 Role',
    rolebindings: '新增 RoleBinding',
    clusterroles: '新增 ClusterRole',
    clusterrolebindings: '新增 ClusterRoleBinding'
  }
  return buttonMap[activeTab.value] || '新增'
}

// 处理新增按钮点击
const handleCreate = () => {
  const refMap: Record<string, any> = {
    serviceaccounts: serviceAccountsTabRef.value,
    roles: rolesTabRef.value,
    rolebindings: roleBindingsTabRef.value,
    clusterroles: clusterRolesTabRef.value,
    clusterrolebindings: clusterRoleBindingsTabRef.value
  }
  const activeRef = refMap[activeTab.value]
  if (activeRef && activeRef.handleCreate) {
    activeRef.handleCreate()
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const list = await getClusterList()
    clusterList.value = list || []

    // 恢复上次选择的集群
    const savedClusterId = localStorage.getItem('access_control_cluster_id')
    if (savedClusterId) {
      const id = parseInt(savedClusterId)
      if (clusterList.value.some(c => c.id === id)) {
        selectedClusterId.value = id
      }
    }

    // 如果有集群但没保存的选择，默认选第一个
    if (!selectedClusterId.value && clusterList.value.length > 0) {
      selectedClusterId.value = clusterList.value[0].id
    }

    if (selectedClusterId.value) {
      await loadNamespaces()
    }
  } catch (error) {
  }
}

// 更新资源数量
const updateCount = (type: string, count: number) => {
  const accessType = accessTypes.value.find(t => t.value === type)
  if (accessType) {
    accessType.count = count
  }
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return

  try {
    const list = await getNamespaces(selectedClusterId.value)
    namespaceList.value = list || []

    // 恢复上次选择的命名空间
    const savedNamespace = localStorage.getItem('access_control_namespace')
    if (savedNamespace && namespaceList.value.some(ns => ns.name === savedNamespace)) {
      selectedNamespace.value = savedNamespace
    } else if (namespaceList.value.length > 0) {
      selectedNamespace.value = namespaceList.value[0].name
    }
  } catch (error) {
  }
}

// 处理集群切换
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('access_control_cluster_id', selectedClusterId.value.toString())
    await loadNamespaces()
  }
}

// 处理命名空间切换
const handleNamespaceChange = () => {
  if (selectedNamespace.value) {
    localStorage.setItem('access_control_namespace', selectedNamespace.value)
  }
}

// 处理标签切换
const handleTabChange = (tab: string) => {
  activeTab.value = tab
}

// 加载数据 - 根据当前激活的Tab刷新对应数据
const loadData = async () => {
  if (!selectedClusterId.value) {
    await loadClusters()
    return
  }

  // 根据当前激活的Tab刷新对应的子组件数据
  const refMap: Record<string, any> = {
    serviceaccounts: serviceAccountsTabRef.value,
    roles: rolesTabRef.value,
    rolebindings: roleBindingsTabRef.value,
    clusterroles: clusterRolesTabRef.value,
    clusterrolebindings: clusterRoleBindingsTabRef.value
  }

  const activeRef = refMap[activeTab.value]
  if (activeRef && activeRef.loadData) {
    await activeRef.loadData()
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.access-control-container {
  padding: 0;
  background-color: transparent;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #165dff;
  font-size: 22px;
  flex-shrink: 0;
  border: none;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.cluster-select {
  width: 280px;
}

.namespace-select {
  width: 240px;
}

.action-bar {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-section {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.access-types-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow-x: auto;
}

.type-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: #f7f8fa;
  color: #4e5969;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 500;
  user-select: none;
}

.type-tab:hover {
  background: #e8f3ff;
  color: #165dff;
}

.type-tab.active {
  background: #165dff;
  color: #fff;
  border: none;
  box-shadow: 0 2px 8px rgba(22, 93, 255, 0.3);
}

.type-label {
  font-size: 14px;
}

.type-count {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 2px;
}

.content-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 16px;
  min-height: 400px;
}
</style>
