<template>
  <div class="system-info-panel">
    <div v-if="!attached" class="not-attached">
      <a-empty description="请先选择Pod并连接到进程">
        <template #image>
          <icon-link />
        </template>
      </a-empty>
    </div>

    <div v-else class="panel-content" :loading="loading">
      <!-- 工具栏 -->
      <div class="toolbar">
        <a-button type="primary" size="small" @click="loadSystemInfo" :loading="loading">
          <icon-refresh /> 刷新
        </a-button>
        <a-radio-group v-model="activeTab" size="small">
          <a-radio value="sysenv">系统环境变量</a-radio>
          <a-radio value="sysprop">系统属性</a-radio>
        </a-radio-group>
        <a-input
          v-model="searchText"
          placeholder="搜索..."
          style="width: 280px"
          clearable
          size="small"
        >
          <template #prefix>
            <icon-search />
          </template>
        </a-input>
        <span class="item-count" v-if="currentData.length > 0">
          共 {{ filteredData.length }} / {{ currentData.length }} 项
        </span>
      </div>

      <!-- 数据表格 -->
      <div class="table-section" v-if="filteredData.length > 0">
        <a-table
          :data="displayData"
          stripe
          size="small"
          :header-cell-style="{ background: '#f5f7fa', color: '#606266' }"
          style="width: 100%"
         :columns="tableColumns">
          <template #key="{ record }">
              <span class="key-name">{{ record.key }}</span>
            </template>
          <template #value="{ record }">
              <span class="key-value">{{ record.value }}</span>
            </template>
        </a-table>

        <!-- 查看更多 -->
        <div v-if="!showAll && filteredData.length > pageSize" class="load-more">
          <a-link type="primary" @click="showAll = true">
            查看更多 (共 {{ filteredData.length }} 条)
          </a-link>
        </div>
      </div>

      <a-empty v-else-if="!loading" description="暂无数据" />
    </div>
  </div>
</template>

<script setup lang="ts">
const tableColumns = [
  { title: '键', dataIndex: 'key', slotName: 'key', width: 350 },
  { title: '值', dataIndex: 'value', slotName: 'value', width: 400, ellipsis: true, tooltip: true }
]

import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getSysEnv, getSysProp } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

interface KeyValueItem {
  key: string
  value: string
}

const loading = ref(false)
const activeTab = ref<'sysenv' | 'sysprop'>('sysenv')
const searchText = ref('')
const showAll = ref(false)
const pageSize = 50

const sysEnvData = ref<KeyValueItem[]>([])
const sysPropData = ref<KeyValueItem[]>([])

// 当前显示的数据
const currentData = computed(() => {
  return activeTab.value === 'sysenv' ? sysEnvData.value : sysPropData.value
})

// 过滤后的数据
const filteredData = computed(() => {
  if (!searchText.value) {
    return currentData.value
  }
  const keyword = searchText.value.toLowerCase()
  return currentData.value.filter(item =>
    item.key.toLowerCase().includes(keyword) ||
    item.value.toLowerCase().includes(keyword)
  )
})

// 显示的数据（分页）
const displayData = computed(() => {
  if (showAll.value) {
    return filteredData.value
  }
  return filteredData.value.slice(0, pageSize)
})

// 加载系统环境变量
const loadSysEnv = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  try {
    const res = await getSysEnv({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    // 后端现在返回结构化数组数据
    if (Array.isArray(res)) {
      sysEnvData.value = res
    } else if (res?.data && Array.isArray(res.data)) {
      sysEnvData.value = res.data
    } else {
      sysEnvData.value = []
    }
  } catch (error: any) {
    Message.error('获取系统环境变量失败: ' + (error.message || '未知错误'))
  }
}

// 加载系统属性
const loadSysProp = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  try {
    const res = await getSysProp({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    // 后端现在返回结构化数组数据
    if (Array.isArray(res)) {
      sysPropData.value = res
    } else if (res?.data && Array.isArray(res.data)) {
      sysPropData.value = res.data
    } else {
      sysPropData.value = []
    }
  } catch (error: any) {
    Message.error('获取系统属性失败: ' + (error.message || '未知错误'))
  }
}

// 加载所有系统信息
const loadSystemInfo = async () => {
  loading.value = true
  showAll.value = false

  try {
    await Promise.all([loadSysEnv(), loadSysProp()])
  } finally {
    loading.value = false
  }
}

// 切换tab时重置分页
watch(activeTab, () => {
  showAll.value = false
})

// 搜索时重置分页
watch(searchText, () => {
  showAll.value = false
})

watch(() => props.attached, (newVal) => {
  if (newVal) {
    loadSystemInfo()
  } else {
    sysEnvData.value = []
    sysPropData.value = []
  }
})
</script>

<style scoped>
.system-info-panel {
  min-height: 400px;
}

.not-attached {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.panel-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.item-count {
  font-size: 13px;
  color: #909399;
  margin-left: auto;
}

/* 表格 */
.table-section {
  background: #fff;
  border-radius: 6px;
  border: 1px solid #ebeef5;
  overflow: hidden;
}

.key-name {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  color: #409eff;
  font-weight: 500;
}

.key-value {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  color: #303133;
  word-break: break-all;
}

/* 查看更多 */
.load-more {
  text-align: center;
  padding: 16px;
  border-top: 1px solid #ebeef5;
}
</style>
