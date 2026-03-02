<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-history />
        </div>
        <div>
          <div class="page-title">执行记录</div>
          <div class="page-desc">查看用户执行任务和文件分发的历史记录</div>
        </div>
      </div>
      <a-space>
        <a-button
          v-if="selectedIds.length > 0"
          v-permission="'task-history:batch-delete'"
          status="danger"
          @click="handleBatchDelete"
        >
          <template #icon><icon-delete /></template>
          删除选中 ({{ selectedIds.length }})
        </a-button>
        <a-button v-permission="'task-history:export'" @click="handleExport">
          <template #icon><icon-download /></template>
          {{ selectedIds.length > 0 ? `导出选中 (${selectedIds.length})` : '导出全部' }}
        </a-button>
      </a-space>
    </div>

    <!-- 搜索区域 -->
    <a-card class="search-card" :bordered="false">
      <a-space :size="16" wrap>
        <a-input
          v-model="searchForm.keyword"
          placeholder="搜索任务名称..."
          allow-clear
          style="width: 220px"
          @press-enter="handleSearch"
        >
          <template #prefix><icon-search /></template>
        </a-input>
        <a-select
          v-model="searchForm.taskType"
          placeholder="任务类型"
          allow-clear
          style="width: 160px"
          @change="handleSearch"
        >
          <a-option value="manual">手动执行</a-option>
          <a-option value="script">脚本执行</a-option>
          <a-option value="file">文件分发</a-option>
          <a-option value="command">系统命令</a-option>
        </a-select>
        <a-select
          v-model="searchForm.status"
          placeholder="执行状态"
          allow-clear
          style="width: 160px"
          @change="handleSearch"
        >
          <a-option value="pending">等待中</a-option>
          <a-option value="running">执行中</a-option>
          <a-option value="success">成功</a-option>
          <a-option value="failed">失败</a-option>
        </a-select>
        <a-range-picker
          v-model="searchForm.dateRange"
          style="width: 280px"
          @change="handleSearch"
        />
        <a-button type="primary" @click="handleSearch">
          <template #icon><icon-search /></template>
          搜索
        </a-button>
        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
      </a-space>
    </a-card>

    <!-- 表格 -->
    <a-card class="table-card" :bordered="false">
      <a-table
        :data="historyList"
        :loading="loading"
        row-key="id"
        :row-selection="{ type: 'checkbox', showCheckedAll: true }"
        :selected-keys="selectedIds"
        :pagination="tablePagination"
        @selection-change="handleSelectionChange"
        @page-change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      >
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="70" align="center" />
          <a-table-column title="任务名称" data-index="name" :min-width="180">
            <template #cell="{ record }">
              <div class="task-name-cell">
                <icon-play-circle v-if="record.taskType === 'manual' || record.taskType === 'script'" class="task-icon task-icon-exec" />
                <icon-folder v-else-if="record.taskType === 'file'" class="task-icon task-icon-file" />
                <icon-desktop v-else class="task-icon task-icon-cmd" />
                <span>{{ record.name || '-' }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="任务类型" data-index="taskType" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="getTaskTypeColor(record.taskType)">
                {{ getTaskTypeLabel(record.taskType) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="状态" data-index="status" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="getStatusColor(record.status)">
                {{ getStatusLabel(record.status) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="目标主机" data-index="targetHostsDisplay" :min-width="200" ellipsis tooltip />
          <a-table-column title="执行用户" data-index="createdByName" :width="120">
            <template #cell="{ record }">
              <div class="user-cell">
                <a-avatar :size="24" class="user-avatar">
                  {{ (record.createdByName || 'U').charAt(0).toUpperCase() }}
                </a-avatar>
                <span>{{ record.createdByName || '未知用户' }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="执行时间" data-index="createdAt" :width="180">
            <template #cell="{ record }">
              {{ formatDateTime(record.createdAt) }}
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="100" align="center" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="handleView(record)">详情</a-link>
                <a-popconfirm content="确定要删除该记录吗？" @ok="handleDelete(record)">
                  <a-link v-permission="'task-history:delete'" status="danger">删除</a-link>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 查看详情对话框 -->
    <a-modal
      v-model:visible="viewDialogVisible"
      title="执行记录详情"
      :width="960"
      :unmount-on-close="true"
      :footer="false"
    >
      <a-descriptions :column="2" bordered>
        <a-descriptions-item label="任务ID">{{ currentRecord.id }}</a-descriptions-item>
        <a-descriptions-item label="任务名称">{{ currentRecord.name || '-' }}</a-descriptions-item>
        <a-descriptions-item label="任务类型">
          <a-tag size="small" :color="getTaskTypeColor(currentRecord.taskType)">
            {{ getTaskTypeLabel(currentRecord.taskType) }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="执行状态">
          <a-tag size="small" :color="getStatusColor(currentRecord.status)">
            {{ getStatusLabel(currentRecord.status) }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="执行用户">{{ currentRecord.createdByName || '未知用户' }}</a-descriptions-item>
        <a-descriptions-item label="执行时间">{{ formatDateTime(currentRecord.createdAt) }}</a-descriptions-item>
        <a-descriptions-item label="目标主机" :span="2">
          {{ currentRecord.targetHostsDisplay || '-' }}
        </a-descriptions-item>
      </a-descriptions>

      <!-- 执行结果 -->
      <div v-if="parsedResults.length > 0" class="result-section">
        <div class="result-title">执行结果</div>
        <div class="result-list">
          <div
            v-for="(hostResult, index) in parsedResults"
            :key="index"
            class="host-result"
          >
            <div class="host-header" :class="hostResult.status">
              <span class="host-name">{{ hostResult.hostName }}</span>
              <span class="host-ip">({{ hostResult.hostIp }})</span>
              <a-tag :color="hostResult.status === 'success' ? 'green' : 'red'" size="small">
                {{ hostResult.status === 'success' ? '成功' : '失败' }}
              </a-tag>
            </div>
            <div v-if="hostResult.error" class="host-error">
              错误: {{ hostResult.error }}
            </div>
            <div class="host-output">
              <div class="output-header">
                <span>输出:</span>
                <a-link size="small" @click="copyOutput(hostResult.output)">
                  <template #icon><icon-copy /></template>
                  复制
                </a-link>
              </div>
              <pre class="output-content">{{ hostResult.output || '(无输出)' }}</pre>
            </div>
          </div>
        </div>
      </div>

      <div v-if="currentRecord.errorMessage" class="error-section">
        <div class="error-title">错误信息</div>
        <div class="error-content">{{ currentRecord.errorMessage }}</div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconHistory, IconSearch, IconRefresh, IconDelete, IconDownload,
  IconPlayCircle, IconFolder, IconDesktop, IconCopy
} from '@arco-design/web-vue/es/icon'
import {
  getExecutionHistoryList,
  deleteExecutionHistory,
  batchDeleteExecutionHistory,
  exportExecutionHistory
} from '@/api/task'

const loading = ref(false)
const viewDialogVisible = ref(false)
const pagination = ref({ page: 1, pageSize: 10, total: 0 })
const historyList = ref<any[]>([])
const currentRecord = ref<any>({})
const selectedIds = ref<number[]>([])

const searchForm = ref({
  keyword: '',
  taskType: '',
  status: '',
  dateRange: undefined as string[] | undefined,
})

const tablePagination = computed(() => ({
  current: pagination.value.page,
  pageSize: pagination.value.pageSize,
  total: pagination.value.total,
  showTotal: true,
  showPageSize: true,
  pageSizeOptions: [10, 20, 50, 100],
}))

const parsedResults = computed(() => {
  if (!currentRecord.value.result) return []
  try {
    const results = JSON.parse(currentRecord.value.result)
    return Array.isArray(results) ? results : []
  } catch { return [] }
})

const getTaskTypeColor = (type: string) => {
  const map: Record<string, string> = { manual: 'arcoblue', script: 'green', file: 'orangered', command: 'purple' }
  return map[type] || 'gray'
}

const getTaskTypeLabel = (type: string) => {
  const map: Record<string, string> = { manual: '手动执行', script: '脚本执行', file: '文件分发', command: '系统命令' }
  return map[type] || type || '-'
}

const getStatusColor = (status: string) => {
  const map: Record<string, string> = { pending: 'gray', running: 'orangered', success: 'green', failed: 'red' }
  return map[status] || 'gray'
}

const getStatusLabel = (status: string) => {
  const map: Record<string, string> = { pending: '等待中', running: '执行中', success: '成功', failed: '失败' }
  return map[status] || status || '-'
}

const formatDateTime = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

const handleSearch = () => {
  pagination.value.page = 1
  loadHistory()
}

const handleReset = () => {
  searchForm.value = { keyword: '', taskType: '', status: '', dateRange: undefined }
  pagination.value.page = 1
  loadHistory()
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadHistory()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1
  loadHistory()
}

const handleSelectionChange = (rowKeys: number[]) => {
  selectedIds.value = rowKeys
}

const loadHistory = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keyword: searchForm.value.keyword || undefined,
      taskType: searchForm.value.taskType || undefined,
      status: searchForm.value.status || undefined,
    }
    if (searchForm.value.dateRange && searchForm.value.dateRange.length === 2) {
      params.startDate = searchForm.value.dateRange[0]
      params.endDate = searchForm.value.dateRange[1]
    }
    const res = await getExecutionHistoryList(params)
    historyList.value = res.list || []
    pagination.value.total = res.total || 0
  } catch {
    Message.error('获取执行记录列表失败')
  } finally {
    loading.value = false
  }
}

const handleView = (row: any) => {
  currentRecord.value = { ...row }
  viewDialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await deleteExecutionHistory(row.id)
    Message.success('删除成功')
    loadHistory()
  } catch (error: any) {
    Message.error(error.message || '删除失败')
  }
}

const handleBatchDelete = () => {
  if (selectedIds.value.length === 0) { Message.warning('请先选择要删除的记录'); return }
  Modal.warning({
    title: '确认删除',
    content: `确定要删除选中的 ${selectedIds.value.length} 条记录吗？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await batchDeleteExecutionHistory(selectedIds.value)
        Message.success('删除成功')
        selectedIds.value = []
        loadHistory()
      } catch (error: any) {
        Message.error(error.message || '删除失败')
      }
    },
  })
}

const handleExport = async () => {
  try {
    const ids = selectedIds.value.length > 0 ? selectedIds.value : undefined
    const data = await exportExecutionHistory(ids)

    const headers = ['ID', '任务名称', '任务类型', '状态', '目标主机', '执行用户', '执行时间']
    const typeMap: Record<string, string> = { manual: '手动执行', script: '脚本执行', file: '文件分发', command: '系统命令' }
    const statusMap: Record<string, string> = { pending: '等待中', running: '执行中', success: '成功', failed: '失败' }

    const csvContent = [
      headers.join(','),
      ...data.map((item: any) => [
        item.id,
        `"${(item.name || '').replace(/"/g, '""')}"`,
        typeMap[item.taskType] || item.taskType,
        statusMap[item.status] || item.status,
        `"${(item.targetHostsDisplay || '').replace(/"/g, '""')}"`,
        `"${(item.createdByName || '未知用户').replace(/"/g, '""')}"`,
        item.createdAt,
      ].join(','))
    ].join('\n')

    const BOM = '\uFEFF'
    const blob = new Blob([BOM + csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `执行记录_${new Date().toISOString().slice(0, 10)}.csv`
    link.click()
    URL.revokeObjectURL(url)
    Message.success('导出成功')
  } catch (error: any) {
    Message.error(error.message || '导出失败')
  }
}

const copyOutput = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text || '')
    Message.success('已复制到剪贴板')
  } catch { Message.error('复制失败') }
}

onMounted(() => { loadHistory() })
</script>

<style scoped lang="scss">
.page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-card {
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
  padding: 20px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--ops-primary, #165dff) 0%, #4080ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.4;
}

.page-desc {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 2px;
}

.search-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
  flex: 1;
}

.task-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.task-icon-exec { color: var(--ops-primary, #165dff); }
.task-icon-file { color: var(--ops-warning, #ff7d00); }
.task-icon-cmd { color: #722ed1; }

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  background: var(--ops-primary, #165dff);
  color: #fff;
  font-size: 12px;
  flex-shrink: 0;
}

/* 详情对话框 */
.result-section {
  margin-top: 20px;
}

.result-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}

.result-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.host-result {
  background: var(--ops-bg-secondary, #f7f8fa);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--ops-border-color, #e5e6eb);
}

.host-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);

  &.success { background: #e8ffea; }
  &.failed { background: #ffece8; }
}

.host-name {
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}

.host-ip {
  color: var(--ops-text-tertiary, #86909c);
  font-size: 13px;
}

.host-error {
  padding: 10px 16px;
  background: #ffece8;
  color: var(--ops-danger, #f53f3f);
  font-size: 13px;
}

.host-output {
  padding: 12px 16px;
}

.output-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  color: var(--ops-text-secondary, #4e5969);
  font-size: 13px;
}

.output-content {
  margin: 0;
  padding: 12px;
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 6px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
}

.error-section {
  margin-top: 20px;
}

.error-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--ops-danger, #f53f3f);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #ffcfc6;
}

.error-content {
  padding: 12px;
  background: #ffece8;
  color: var(--ops-danger, #f53f3f);
  border-radius: 6px;
  font-size: 13px;
}
</style>
