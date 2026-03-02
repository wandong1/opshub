<template>
  <div class="container-config">
    <!-- 标准容器 -->
    <div class="container-section">
      <div class="section-header">
        <span class="section-title">标准容器 (Containers)</span>
        <a-button type="primary" size="small" @click="addContainer('containers')">添加容器</a-button>
      </div>
      <div class="container-list">
        <a-collapse v-model="activeContainers" accordion>
          <a-collapse-item v-for="(container, index) in containers" :key="'container-'+index" :name="index">
            <template #title>
              <div class="container-title">
                <icon-storage />
                <span>{{ container.name || '未命名容器' }}</span>
                <a-tag size="small" color="green">{{ container.image || '无镜像' }}</a-tag>
                <a-button status="danger" type="text" size="small" @click.stop="removeContainer('containers', index)" class="remove-btn">删除</a-button>
              </div>
            </template>
            <div class="container-detail">
              <a-tabs :model-value="getContainerActiveTab('containers', index)" @tab-change="(tab) => setContainerActiveTab('containers', index, tab as string)">
                <a-tab-pane title="基础配置" key="basic">
                  <ContainerBasicInfo :container="container" @update="updateContainer('containers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="运行命令" key="command">
                  <ContainerCommand :container="container" @update="updateContainer('containers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="环境变量" key="env">
                  <EnvConfig :envs="container.env || []" @update="updateContainerEnv('containers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="健康检测" key="health">
                  <HealthCheck
                    :livenessProbe="container.livenessProbe"
                    :readinessProbe="container.readinessProbe"
                    :startupProbe="container.startupProbe"
                    @updateLiveness="updateContainerProbe('containers', index, 'livenessProbe', $event)"
                    @updateReadiness="updateContainerProbe('containers', index, 'readinessProbe', $event)"
                    @updateStartup="updateContainerProbe('containers', index, 'startupProbe', $event)"
                  />
                </a-tab-pane>
                <a-tab-pane title="资源配置" key="resources">
                  <ResourceConfig :resources="container.resources || {}" @update="updateContainerResources('containers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="端口配置" key="ports">
                  <PortConfig :ports="container.ports || []" @update="updateContainerPorts('containers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="存储挂载" key="volumes">
                  <VolumeMounts :volumeMounts="container.volumeMounts || []" :volumes="volumes" @update="updateContainerVolumeMounts('containers', index, $event)" />
                </a-tab-pane>
              </a-tabs>
            </div>
          </a-collapse-item>
        </a-collapse>
        <a-empty v-if="containers.length === 0" description="暂无标准容器" :image-size="60" />
      </div>
    </div>

    <!-- 初始化容器 -->
    <div class="container-section">
      <div class="section-header">
        <span class="section-title">初始化容器 (Init Containers)</span>
        <a-button type="primary" size="small" @click="addContainer('initContainers')">添加初始化容器</a-button>
      </div>
      <div class="container-list">
        <a-collapse v-model="activeInitContainers" accordion>
          <a-collapse-item v-for="(container, index) in initContainers" :key="'init-container-'+index" :name="index">
            <template #title>
              <div class="container-title">
                <icon-storage />
                <span>{{ container.name || '未命名容器' }}</span>
                <a-tag size="small" color="orangered">{{ container.image || '无镜像' }}</a-tag>
                <a-button status="danger" type="text" size="small" @click.stop="removeContainer('initContainers', index)" class="remove-btn">删除</a-button>
              </div>
            </template>
            <div class="container-detail">
              <a-tabs :model-value="getContainerActiveTab('initContainers', index)" @tab-change="(tab) => setContainerActiveTab('initContainers', index, tab as string)">
                <a-tab-pane title="基础配置" key="basic">
                  <ContainerBasicInfo :container="container" @update="updateContainer('initContainers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="运行命令" key="command">
                  <ContainerCommand :container="container" @update="updateContainer('initContainers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="环境变量" key="env">
                  <EnvConfig :envs="container.env || []" @update="updateContainerEnv('initContainers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="健康检测" key="health">
                  <HealthCheck
                    :livenessProbe="container.livenessProbe"
                    :readinessProbe="container.readinessProbe"
                    :startupProbe="container.startupProbe"
                    @updateLiveness="updateContainerProbe('initContainers', index, 'livenessProbe', $event)"
                    @updateReadiness="updateContainerProbe('initContainers', index, 'readinessProbe', $event)"
                    @updateStartup="updateContainerProbe('initContainers', index, 'startupProbe', $event)"
                  />
                </a-tab-pane>
                <a-tab-pane title="资源配置" key="resources">
                  <ResourceConfig :resources="container.resources || {}" @update="updateContainerResources('initContainers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="端口配置" key="ports">
                  <PortConfig :ports="container.ports || []" @update="updateContainerPorts('initContainers', index, $event)" />
                </a-tab-pane>
                <a-tab-pane title="存储挂载" key="volumes">
                  <VolumeMounts :volumeMounts="container.volumeMounts || []" :volumes="volumes" @update="updateContainerVolumeMounts('initContainers', index, $event)" />
                </a-tab-pane>
              </a-tabs>
            </div>
          </a-collapse-item>
        </a-collapse>
        <a-empty v-if="initContainers.length === 0" description="暂无初始化容器" :image-size="60" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ContainerBasicInfo from './ContainerBasicInfo.vue'
import ContainerCommand from './ContainerCommand.vue'
import EnvConfig from './EnvConfig.vue'
import HealthCheck from './HealthCheck.vue'
import ResourceConfig from './ResourceConfig.vue'
import PortConfig from './PortConfig.vue'
import VolumeMounts from './VolumeMounts.vue'

interface Container {
  name: string
  image: string
  imagePullPolicy?: string
  workingDir?: string
  command?: string[]
  args?: string[]
  env?: any[]
  resources?: any
  ports?: any[]
  volumeMounts?: any[]
  stdin?: boolean
  tty?: boolean
  activeTab?: string
}

const props = defineProps<{
  containers: Container[]
  initContainers: Container[]
  volumes: any[]
}>()

const emit = defineEmits<{
  updateContainers: [containers: Container[]]
  updateInitContainers: [initContainers: Container[]]
}>()

const activeContainers = ref<number[]>([])
const activeInitContainers = ref<number[]>([])

// 获取容器的活动标签页
const getContainerActiveTab = (type: 'containers' | 'initContainers', index: number) => {
  const containerList = type === 'containers' ? props.containers : props.initContainers
  return containerList[index]?.activeTab || 'basic'
}

// 设置容器的活动标签页
const setContainerActiveTab = (type: 'containers' | 'initContainers', index: number, tabName: string) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], activeTab: tabName }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], activeTab: tabName }
    emit('updateInitContainers', updated)
  }
}

const addContainer = (type: 'containers' | 'initContainers') => {
  const newContainer: Container = {
    name: '',
    image: '',
    imagePullPolicy: 'IfNotPresent',
    command: [],
    args: [],
    env: [],
    ports: [],
    volumeMounts: [],
    activeTab: 'basic'
  }

  if (type === 'containers') {
    const updated = [...props.containers, newContainer]
    emit('updateContainers', updated)
    activeContainers.value = [updated.length - 1]
  } else {
    const updated = [...props.initContainers, newContainer]
    emit('updateInitContainers', updated)
    activeInitContainers.value = [updated.length - 1]
  }
}

const removeContainer = (type: 'containers' | 'initContainers', index: number) => {
  if (type === 'containers') {
    const updated = props.containers.filter((_, i) => i !== index)
    emit('updateContainers', updated)
  } else {
    const updated = props.initContainers.filter((_, i) => i !== index)
    emit('updateInitContainers', updated)
  }
}

const updateContainer = (type: 'containers' | 'initContainers', index: number, data: Partial<Container>) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], ...data }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], ...data }
    emit('updateInitContainers', updated)
  }
}

const updateContainerEnv = (type: 'containers' | 'initContainers', index: number, envs: any[]) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], env: envs }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], env: envs }
    emit('updateInitContainers', updated)
  }
}

const updateContainerResources = (type: 'containers' | 'initContainers', index: number, resources: any) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], resources }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], resources }
    emit('updateInitContainers', updated)
  }
}

const updateContainerPorts = (type: 'containers' | 'initContainers', index: number, ports: any[]) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], ports }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], ports }
    emit('updateInitContainers', updated)
  }
}

const updateContainerVolumeMounts = (type: 'containers' | 'initContainers', index: number, volumeMounts: any[]) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], volumeMounts }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], volumeMounts }
    emit('updateInitContainers', updated)
  }
}

const updateContainerProbe = (type: 'containers' | 'initContainers', index: number, probeType: string, probe: any) => {
  if (type === 'containers') {
    const updated = [...props.containers]
    updated[index] = { ...updated[index], [probeType]: probe }
    emit('updateContainers', updated)
  } else {
    const updated = [...props.initContainers]
    updated[index] = { ...updated[index], [probeType]: probe }
    emit('updateInitContainers', updated)
  }
}
</script>

<style scoped>
.container-config {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.container-section {
  background: #ffffff;
  border-radius: 4px;
  overflow: hidden;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: #d4af37;
  border-bottom: 1px solid #d4af37;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  letter-spacing: 0.3px;
}

.section-header .arco-btn {
  font-weight: 500;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #d4af37;
  color: #d4af37;
}

.section-header .arco-btn:hover {
  background: #fafafa;
  border-color: #c9a227;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.container-list {
  padding: 20px;
  background: #ffffff;
}

.container-title {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.container-title .arco-icon {
  color: #d4af37;
  font-size: 18px;
}

.remove-btn {
  margin-left: auto;
  opacity: 0.7;
  transition: opacity 0.3s ease;
}

.remove-btn:hover {
  opacity: 1;
}

.container-detail {
  padding: 24px;
  background: #fafafa;
}

.container-detail :deep(.arco-tabs__header) {
  background: #ffffff;
  border-radius: 8px;
  margin-bottom: 16px;
  border: 1px solid #e8e8e8;
}

.container-detail :deep(.arco-tabs__nav) {
  border: none;
}

.container-detail :deep(.arco-tabs__item) {
  color: #666;
  font-weight: 500;
  border: none;
  padding: 0 20px;
  height: 44px;
  line-height: 44px;
  transition: all 0.3s ease;
}

.container-detail :deep(.arco-tabs__item:hover) {
  color: #d4af37;
}

.container-detail :deep(.arco-tabs__item.is-active) {
  color: #d4af37;
  background: transparent;
}

.container-detail :deep(.arco-tabs__active-bar) {
  height: 2px;
  background: #d4af37;
}

.container-detail :deep(.arco-collapse) {
  border: none;
}

.container-detail :deep(.arco-collapse-item__header) {
  background: #ffffff;
  border-radius: 8px;
  margin-bottom: 12px;
  padding: 16px 20px;
  border: 1px solid #e8e8e8;
  font-weight: 600;
  color: #333;
  transition: all 0.3s ease;
}

.container-detail :deep(.arco-collapse-item__header:hover) {
  border-color: #d4af37;
  background: #fafafa;
}

.container-detail :deep(.arco-collapse-item__wrap) {
  background: transparent;
  border: none;
}

.container-detail :deep(.arco-collapse-item__content) {
  padding-bottom: 0;
}
</style>
