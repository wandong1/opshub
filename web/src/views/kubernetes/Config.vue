<template>
  <div class="config-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-safe />
        </div>
        <div>
          <h2 class="page-title">配置管理</h2>
          <p class="page-subtitle">管理 Kubernetes ConfigMaps、Secrets 和其他配置资源</p>
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
        <a-button type="primary" @click="loadCurrentResources">
          <icon-refresh />
          刷新
        </a-button>
      </div>
    </div>

    <!-- 配置类型标签 -->
    <div class="config-types-bar">
      <div
        v-for="type in configTypes"
        :key="type.value"
        :class="['type-tab', { active: activeTab === type.value }]"
        @click="handleTabChange(type.value)"
      >
          <component :is="type.icon" />
        <span class="type-label">{{ type.label }}</span>
        <span v-if="type.count !== undefined" class="type-count">({{ type.count }})</span>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="content-wrapper">
      <!-- ConfigMaps -->
      <ConfigMapList
        v-show="activeTab === 'configmaps' && selectedClusterId"
        ref="configMapListRef"
        :clusterId="selectedClusterId"
        @edit="handleEditConfigMap"
        @yaml="handleEditConfigMapYAML"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('configmaps', count)"
      />

      <!-- Secrets -->
      <SecretList
        v-show="activeTab === 'secrets' && selectedClusterId"
        ref="secretListRef"
        :clusterId="selectedClusterId"
        @edit="handleEditSecret"
        @yaml="handleEditSecretYAML"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('secrets', count)"
      />

      <!-- ResourceQuotas -->
      <ResourceQuotaList
        v-show="activeTab === 'resourcequotas' && selectedClusterId"
        ref="resourceQuotaListRef"
        :clusterId="selectedClusterId"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('resourcequotas', count)"
      />

      <!-- LimitRanges -->
      <LimitRangeList
        v-show="activeTab === 'limitranges' && selectedClusterId"
        ref="limitRangeListRef"
        :clusterId="selectedClusterId"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('limitranges', count)"
      />

      <!-- HPA -->
      <HPAList
        v-show="activeTab === 'hpa' && selectedClusterId"
        ref="hpaListRef"
        :clusterId="selectedClusterId"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('hpa', count)"
      />

      <!-- PodDisruptionBudgets -->
      <PodDisruptionBudgetList
        v-show="activeTab === 'pdb' && selectedClusterId"
        ref="pdbListRef"
        :clusterId="selectedClusterId"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('pdb', count)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconSafe,
  IconLock,
  IconBarChart,
  IconSettings,
  IconDashboard,
  IconThunderbolt
} from '@arco-design/web-vue/es/icon'
import { getClusterList, type Cluster } from '@/api/kubernetes'
import axios from 'axios'
import ConfigMapList from './config-components/ConfigMapList.vue'
import SecretList from './config-components/SecretList.vue'
import ResourceQuotaList from './config-components/ResourceQuotaList.vue'
import LimitRangeList from './config-components/LimitRangeList.vue'
import HPAList from './config-components/HPAList.vue'
import PodDisruptionBudgetList from './config-components/PodDisruptionBudgetList.vue'

// 配置类型定义
interface ConfigType {
  label: string
  value: string
  icon: any
  count: number
}

const configTypes = ref<ConfigType[]>([
  { label: 'ConfigMaps', value: 'configmaps', icon: IconSafe, count: 0 },
  { label: 'Secrets', value: 'secrets', icon: IconLock, count: 0 },
  { label: 'ResourceQuotas', value: 'resourcequotas', icon: IconBarChart, count: 0 },
  { label: 'LimitRanges', value: 'limitranges', icon: IconSettings, count: 0 },
  { label: 'HPA', value: 'hpa', icon: IconDashboard, count: 0 },
  { label: 'PodDisruptionBudgets', value: 'pdb', icon: IconThunderbolt, count: 0 },
])

const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const activeTab = ref('configmaps')

// 子组件引用
const configMapListRef = ref()
const secretListRef = ref()
const resourceQuotaListRef = ref()
const limitRangeListRef = ref()
const hpaListRef = ref()
const pdbListRef = ref()

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('config_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
    }
  } catch (error) {
    Message.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('config_selected_cluster_id', selectedClusterId.value.toString())
  }
}

// Tab 切换
const handleTabChange = (tab: string) => {
  activeTab.value = tab
  localStorage.setItem('config_active_tab', tab)
}

// 加载当前资源
const loadCurrentResources = async () => {
  if (!selectedClusterId.value) return

  // 根据当前激活的 tab 刷新对应的子组件数据
  switch (activeTab.value) {
    case 'configmaps':
      await configMapListRef.value?.loadConfigMaps?.()
      break
    case 'secrets':
      await secretListRef.value?.loadSecrets?.()
      break
    case 'resourcequotas':
      await resourceQuotaListRef.value?.loadResourceQuotas?.()
      break
    case 'limitranges':
      await limitRangeListRef.value?.loadLimitRanges?.()
      break
    case 'hpa':
      await hpaListRef.value?.loadHPAs?.()
      break
    case 'pdb':
      await pdbListRef.value?.loadPDBs?.()
      break
  }
}

// ConfigMap 操作
const handleEditConfigMap = (configMap: any) => {
  Message.info('编辑 ConfigMap 功能开发中...')
}

const handleEditConfigMapYAML = (configMap: any) => {
  Message.info('编辑 ConfigMap YAML 功能开发中...')
}

// Secret 操作
const handleEditSecret = (secret: any) => {
  Message.info('编辑 Secret 功能开发中...')
}

const handleEditSecretYAML = (secret: any) => {
  Message.info('编辑 Secret YAML 功能开发中...')
}

// 更新数量
const updateCount = (type: string, count: number) => {
  const configType = configTypes.value.find(t => t.value === type)
  if (configType) {
    configType.count = count
  }
}

onMounted(() => {
  loadClusters()
  const savedTab = localStorage.getItem('config_active_tab')
  if (savedTab) {
    activeTab.value = savedTab
  }
})
</script>

<style scoped>
.config-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
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

/* 配置类型标签栏 */
.config-types-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  flex-wrap: wrap;
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
  font-size: 14px;
  font-weight: 500;
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

.type-icon {
  font-size: 16px;
}

.type-label {
  white-space: nowrap;
}

.type-count {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 2px;
}

/* 内容区域 */
.content-wrapper {
  background: transparent;
}

.cluster-select :deep(.arco-input__wrapper) {
  border-radius: 8px;
}
</style>
