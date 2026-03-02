<template>
  <div class="thread-list-panel">
    <div v-if="!attached" class="not-attached">
      <a-empty description="请先选择Pod并连接到进程">
        <template #image>
          <icon-link />
        </template>
      </a-empty>
    </div>

    <div v-else class="panel-content" :loading="loading">
      <!-- 状态统计栏 -->
      <div class="status-bar">
        <div class="status-info">
          <span class="total-label">总量:</span>
          <span class="total-value">{{ threads.length }}</span>
          <span class="divider">|</span>
          <span class="status-label">状态分布</span>
          <a-tag
            v-for="(count, state) in stateStats"
            :key="state"
            :type="getStateTagType(state as string)"
            size="small"
            :class="['state-tag', { active: selectedState === state || selectedState === '' }]"
            @click="filterByState(state as string)"
          >
            {{ state }} {{ count }}
          </a-tag>
        </div>
        <a-button type="primary" size="small" @click="loadThreads" :loading="loading">
          <icon-refresh /> 更新
        </a-button>
      </div>

      <!-- 线程表格 -->
      <div class="table-section">
        <a-table
          :data="displayThreads"
          stripe
          size="small"
          :header-cell-style="{ background: '#f5f7fa', color: '#606266' }"
          style="width: 100%"
          @row-click="handleRowClick"
          :columns="tableColumns">
          <template #state="{ record }">
              <a-tag :type="getStateTagType(record.state)" size="small">{{ record.state }}</a-tag>
            </template>
          <template #interrupted="{ record }">
              <span :class="record.interrupted ? 'text-danger' : 'text-muted'">{{ record.interrupted ? 'true' : 'false' }}</span>
            </template>
          <template #daemon="{ record }">
              <span :class="record.daemon ? 'text-primary' : 'text-muted'">{{ record.daemon ? 'true' : 'false' }}</span>
            </template>
        </a-table>

        <!-- 空状态 -->
        <a-empty v-if="displayThreads.length === 0 && !loading" description="暂无数据" :image-size="60">
          <template #default>
            <a-link type="primary" @click="loadThreads">查看更多</a-link>
          </template>
        </a-empty>

        <!-- 查看更多 -->
        <div v-if="!showAll && filteredThreads.length > pageSize" class="load-more">
          <a-link type="primary" @click="showAll = true">查看更多 (共 {{ filteredThreads.length }} 条)</a-link>
        </div>
      </div>

      <!-- 线程堆栈详情对话框 -->
      <a-modal v-model:visible="stackDialogVisible" :title="`线程堆栈 - ${currentThread?.name || ''}`" width="900px" top="5vh">
        <div class="stack-header" v-if="currentThread">
          <a-descriptions :column="4" size="small" :bordered="true">
            <a-descriptions-item label="ID">{{ currentThread.id }}</a-descriptions-item>
            <a-descriptions-item label="名称">{{ currentThread.name }}</a-descriptions-item>
            <a-descriptions-item label="状态">
              <a-tag :type="getStateTagType(currentThread.state)" size="small">{{ currentThread.state }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="CPU">{{ currentThread.cpu }}%</a-descriptions-item>
          </a-descriptions>
        </div>
        <div class="stack-content" v-loading="stackLoading">
          <pre>{{ currentStack || '暂无堆栈信息' }}</pre>
        </div>
      </a-modal>
    </div>
  </div>
</template>

<script setup lang="ts">
const tableColumns = [
  { title: 'ID', dataIndex: 'id', width: 60, align: 'center' },
  { title: '名称', dataIndex: 'name', ellipsis: true, tooltip: true },
  { title: 'Group', dataIndex: 'group', width: 80, align: 'center' },
  { title: 'Priority', dataIndex: 'priority', width: 80, align: 'center' },
  { title: 'State', dataIndex: 'state', slotName: 'state', width: 130, align: 'center' },
  { title: 'CPU(%)', dataIndex: 'cpu', width: 80, align: 'right' },
  { title: 'Time(秒)', dataIndex: 'time', width: 90, align: 'right' },
  { title: 'Interrupted', dataIndex: 'interrupted', slotName: 'interrupted', width: 100, align: 'center' },
  { title: 'Daemon', dataIndex: 'daemon', slotName: 'daemon', width: 80, align: 'center' }
]

import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getThreadList, getThreadStack } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

interface ThreadInfo {
  id: string
  name: string
  group: string
  priority: string
  state: string
  cpu: string
  time: string
  interrupted: boolean
  daemon: boolean
}

const loading = ref(false)
const stackLoading = ref(false)
const threads = ref<ThreadInfo[]>([])
const rawOutput = ref('')
const selectedState = ref('')
const showAll = ref(false)
const pageSize = 20

const stackDialogVisible = ref(false)
const currentThread = ref<ThreadInfo | null>(null)
const currentStack = ref('')

// 状态统计
const stateStats = computed(() => {
  const stats: Record<string, number> = {
    NEW: 0,
    RUNNABLE: 0,
    BLOCKED: 0,
    WAITING: 0,
    TIMED_WAITING: 0,
    TERMINATED: 0
  }
  threads.value.forEach(t => {
    if (stats[t.state] !== undefined) {
      stats[t.state]++
    } else {
      stats[t.state] = 1
    }
  })
  // 只返回有数据的状态
  return Object.fromEntries(Object.entries(stats).filter(([_, v]) => v > 0))
})

// 按状态过滤的线程
const filteredThreads = computed(() => {
  if (!selectedState.value) {
    return threads.value
  }
  return threads.value.filter(t => t.state === selectedState.value)
})

// 显示的线程（分页）
const displayThreads = computed(() => {
  if (showAll.value) {
    return filteredThreads.value
  }
  return filteredThreads.value.slice(0, pageSize)
})

// 获取状态标签类型
const getStateTagType = (state: string): string => {
  const types: Record<string, string> = {
    'RUNNABLE': 'success',
    'BLOCKED': 'danger',
    'WAITING': 'warning',
    'TIMED_WAITING': 'primary',
    'NEW': 'info',
    'TERMINATED': 'info'
  }
  return types[state] || 'info'
}

// 按状态过滤
const filterByState = (state: string) => {
  if (selectedState.value === state) {
    selectedState.value = ''
  } else {
    selectedState.value = state
  }
  showAll.value = false
}

// 解析线程输出
const parseThreadOutput = (output: string): ThreadInfo[] => {
  const threads: ThreadInfo[] = []
  const lines = output.split('\n')

  // 查找表头行
  let headerFound = false
  for (const line of lines) {
    const trimmedLine = line.trim()

    // 跳过空行和信息行
    if (!trimmedLine || trimmedLine.startsWith('[INFO]') || trimmedLine.startsWith('[arthas@')) {
      continue
    }

    // 检查是否是表头
    if (trimmedLine.startsWith('ID') && trimmedLine.includes('NAME')) {
      headerFound = true
      continue
    }

    // 解析数据行
    if (headerFound) {
      // 线程行格式: ID NAME GROUP PRIORITY STATE CPU DELTA_TIME TIME INTERRUPTED DAEMON
      const parts = trimmedLine.split(/\s+/)
      if (parts.length >= 8) {
        const id = parts[0]
        // 验证ID是数字
        if (!/^\d+$/.test(id)) continue

        // 从后往前解析固定字段
        const n = parts.length
        const daemon = parts[n - 1] === 'true'
        const interrupted = parts[n - 2] === 'true'
        const time = parts[n - 3]
        const deltaTime = parts[n - 4]
        const cpu = parts[n - 5]
        const state = parts[n - 6]
        const priority = parts[n - 7]
        const group = parts[n - 8]

        // 名称是ID和group之间的部分
        const nameEndIdx = n - 8
        const name = parts.slice(1, nameEndIdx).join(' ')

        threads.push({
          id,
          name,
          group,
          priority,
          state: normalizeState(state),
          cpu,
          time,
          interrupted,
          daemon
        })
      }
    }
  }

  return threads
}

// 标准化状态名
const normalizeState = (state: string): string => {
  const stateMap: Record<string, string> = {
    'TIMED_': 'TIMED_WAITING',
    'WAITIN': 'WAITING',
    'RUNNAB': 'RUNNABLE',
    'BLOCKE': 'BLOCKED',
    'TERMIN': 'TERMINATED'
  }
  return stateMap[state] || state
}

// 加载线程列表
const loadThreads = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loading.value = true
  showAll.value = false
  selectedState.value = ''

  try {
    const res = await getThreadList({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    const output = typeof res === 'string' ? res : (res?.data || '')
    rawOutput.value = output
    threads.value = parseThreadOutput(output)

    if (threads.value.length === 0 && output) {
    }
  } catch (error: any) {
    Message.error('获取线程列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 点击行查看堆栈
const handleRowClick = async (row: ThreadInfo) => {
  currentThread.value = row
  stackDialogVisible.value = true
  await loadThreadStack(row.id)
}

// 加载线程堆栈
const loadThreadStack = async (threadId: string) => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  stackLoading.value = true
  try {
    const res = await getThreadStack({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId,
      threadId
    })
    currentStack.value = typeof res === 'string' ? res : (res?.data || '暂无堆栈信息')
  } catch (error: any) {
    Message.error('获取线程堆栈失败: ' + (error.message || '未知错误'))
    currentStack.value = '获取失败: ' + (error.message || '未知错误')
  } finally {
    stackLoading.value = false
  }
}

watch(() => props.attached, (newVal) => {
  if (newVal) {
    loadThreads()
  } else {
    threads.value = []
    rawOutput.value = ''
  }
})
</script>

<style scoped>
.thread-list-panel {
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

/* 状态栏 */
.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.total-label {
  font-size: 13px;
  color: #606266;
}

.total-value {
  font-size: 16px;
  font-weight: 600;
  color: #409eff;
  min-width: 30px;
}

.divider {
  color: #dcdfe6;
  margin: 0 4px;
}

.status-label {
  font-size: 13px;
  color: #606266;
  margin-right: 4px;
}

.state-tag {
  cursor: pointer;
  transition: all 0.2s;
}

.state-tag:hover {
  transform: scale(1.05);
}

/* 表格 */
.table-section {
  background: #fff;
  border-radius: 6px;
  border: 1px solid #ebeef5;
  overflow: hidden;
}

.text-danger { color: #f56c6c; }
.text-primary { color: #409eff; }
.text-muted { color: #c0c4cc; }

:deep(.clickable-row) {
  cursor: pointer;
}

:deep(.clickable-row:hover) {
  background-color: #f5f7fa !important;
}

/* 查看更多 */
.load-more {
  text-align: center;
  padding: 16px;
  border-top: 1px solid #ebeef5;
}

/* 堆栈对话框 */
.stack-header {
  margin-bottom: 16px;
}

.stack-content {
  background: #1e1e1e;
  border-radius: 6px;
  padding: 16px;
  max-height: 500px;
  overflow: auto;
}

.stack-content pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
