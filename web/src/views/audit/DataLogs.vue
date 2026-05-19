<template>
  <div class="data-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-bar-chart />
        </div>
        <div>
          <h2 class="page-title">数据日志</h2>
          <p class="page-subtitle">记录系统数据的变更历史</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="handleSearch">
          <template #icon><icon-search /></template>
          查询
        </a-button>
        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
        <a-button v-permission="'data-logs:batch-delete'" status="danger" @click="handleBatchDelete" :disabled="selectedIds.length === 0">
          <template #icon><icon-delete /></template>
          批量删除
        </a-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <a-input
        v-model="searchForm.username"
        placeholder="搜索用户名..."
        allow-clear
        class="filter-input"
      >
        <template #prefix>
          <icon-user />
        </template>
      </a-input>
      <a-select
        v-model="searchForm.tableName"
        placeholder="数据表"
        allow-clear
        class="filter-select"
      >
        <a-option label="用户表" value="sys_user" />
        <a-option label="角色表" value="sys_role" />
        <a-option label="部门表" value="sys_department" />
        <a-option label="菜单表" value="sys_menu" />
        <a-option label="岗位表" value="sys_position" />
      </a-select>
      <a-select
        v-model="searchForm.action"
        placeholder="操作类型"
        allow-clear
        class="filter-select"
      >
        <a-option label="创建" value="create" />
        <a-option label="更新" value="update" />
        <a-option label="删除" value="delete" />
      </a-select>
      <a-range-picker
        v-model="dateRange"
        format="YYYY-MM-DD"
        class="filter-date"
      />
    </div>

    <!-- 数据表格 -->
    <div class="table-wrapper">
      <a-table
        :data="logList"
        :loading="loading"
        :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: false }"
        @selection-change="handleSelectionChange"
        :pagination="false"
        class="modern-table"
      >
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="80" align="center">
            <template #cell="{ record }">
              <span class="id-text">#{{ record.id }}</span>
            </template>
          </a-table-column>
          <a-table-column title="操作用户" data-index="username" :width="140">
            <template #cell="{ record }">
              <div class="user-cell">
                <icon-user class="user-icon" />
                <span>{{ record.realName || record.username }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="数据表" data-index="tableName" :width="140">
            <template #cell="{ record }">
              <a-tag color="blue">{{ record.tableName }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="记录ID" data-index="recordId" :width="90" align="center">
            <template #cell="{ record }">
              <span class="id-text">{{ record.recordId }}</span>
            </template>
          </a-table-column>
          <a-table-column title="操作" data-index="action" :width="80">
            <template #cell="{ record }">
              <a-tag :color="getActionColor(record.action)">
                {{ getActionLabel(record.action) }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="变更字段" data-index="diffFields" :width="150" :ellipsis="true" :tooltip="true" />
          <a-table-column title="数据变更" :width="100" align="center">
            <template #cell="{ record }">
              <a-button type="text" @click="showDataDiff(record)">
                查看详情
              </a-button>
            </template>
          </a-table-column>
          <a-table-column title="IP地址" data-index="ip" :width="130" />
          <a-table-column title="操作时间" data-index="createdAt" :width="170" />
          <a-table-column title="操作" :width="80" fixed="right" align="center">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="删除">
                  <a-button v-permission="'data-logs:delete'" type="text" status="danger" @click="handleDelete(record)">
                    <template #icon><icon-delete /></template>
                  </a-button>
                </a-tooltip>
              </div>
            </template>
          </a-table-column>
        </template>
      </a-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-size-options="[10, 20, 50, 100]"
          show-total
          show-page-size
          @change="loadLogList"
          @page-size-change="loadLogList"
        />
      </div>
    </div>

    <!-- 数据变更详情对话框 -->
    <a-modal
      v-model:visible="detailDialogVisible"
      title="数据变更详情"
      width="700px"
      :footer="false"
      unmount-on-close
    >
      <div v-if="currentRow" class="detail-content">
        <div class="detail-info">
          <div class="info-item">
            <span class="info-label">操作用户:</span>
            <span class="info-value">{{ currentRow.realName || currentRow.username }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">数据表:</span>
            <a-tag color="blue">{{ currentRow.tableName }}</a-tag>
          </div>
          <div class="info-item">
            <span class="info-label">记录ID:</span>
            <span class="info-value">{{ currentRow.recordId }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">操作类型:</span>
            <a-tag :color="getActionColor(currentRow.action)">
              {{ getActionLabel(currentRow.action) }}
            </a-tag>
          </div>
        </div>

        <div v-if="currentRow.action === 'create'" class="data-section">
          <h4 class="section-title">新增数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.newData) }}</pre>
        </div>

        <div v-else-if="currentRow.action === 'update'" class="data-section">
          <h4 class="section-title">原始数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.oldData) }}</pre>
          <h4 class="section-title">新数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.newData) }}</pre>
          <div v-if="currentRow.diffFields" class="diff-section">
            <h4 class="section-title">变更字段</h4>
            <a-tag color="orange">{{ currentRow.diffFields }}</a-tag>
          </div>
        </div>

        <div v-else-if="currentRow.action === 'delete'" class="data-section">
          <h4 class="section-title">删除数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.oldData) }}</pre>
        </div>
      </div>

      <template #footer>
        <a-button @click="detailDialogVisible = false">关闭</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconBarChart,
  IconDelete,
  IconSearch,
  IconRefresh,
  IconUser
} from '@arco-design/web-vue/es/icon'
import { getDataLogList, deleteDataLog, deleteDataLogsBatch } from '@/api/audit'

// 搜索表单
const searchForm = reactive({
  username: '',
  tableName: '',
  action: '',
  startTime: '',
  endTime: ''
})

// 日期范围
const dateRange = ref<[string, string]>()

// 监听日期范围变化
watch(dateRange, (newVal) => {
  if (newVal && newVal.length === 2) {
    searchForm.startTime = newVal[0]
    searchForm.endTime = newVal[1]
  } else {
    searchForm.startTime = ''
    searchForm.endTime = ''
  }
})

// 日志列表
const logList = ref<any[]>([])
const loading = ref(false)

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 选中的ID
const selectedIds = ref<number[]>([])

// 详情对话框
const detailDialogVisible = ref(false)
const currentRow = ref<any>(null)

// 加载日志列表
const loadLogList = async () => {
  loading.value = true
  try {
    const res: any = await getDataLogList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...searchForm
    })
    logList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    Message.error('获取日志列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadLogList()
}

// 重置
const handleReset = () => {
  searchForm.username = ''
  searchForm.tableName = ''
  searchForm.action = ''
  searchForm.startTime = ''
  searchForm.endTime = ''
  dateRange.value = undefined
  pagination.page = 1
  loadLogList()
}

// 删除
const handleDelete = (row: any) => {
  Modal.confirm({
    title: '提示',
    content: '确定要删除这条日志吗?',
    onOk: async () => {
      try {
        await deleteDataLog(row.id)
        Message.success('删除成功')
        loadLogList()
      } catch (error) {
        Message.error('删除失败')
      }
    }
  })
}

// 批量删除
const handleBatchDelete = () => {
  Modal.confirm({
    title: '提示',
    content: `确定要删除选中的 ${selectedIds.value.length} 条日志吗?`,
    onOk: async () => {
      try {
        await deleteDataLogsBatch(selectedIds.value)
        Message.success('删除成功')
        selectedIds.value = []
        loadLogList()
      } catch (error) {
        Message.error('删除失败')
      }
    }
  })
}

// 选择变化
const handleSelectionChange = (rowKeys: (string | number)[]) => {
  selectedIds.value = rowKeys as number[]
}

// 显示数据差异
const showDataDiff = (row: any) => {
  currentRow.value = row
  detailDialogVisible.value = true
}

// 获取操作类型标签颜色
const getActionColor = (action: string) => {
  const map: Record<string, string> = {
    'create': 'green',
    'update': 'orange',
    'delete': 'red'
  }
  return map[action] || 'blue'
}

// 获取操作类型标签
const getActionLabel = (action: string) => {
  const map: Record<string, string> = {
    'create': '创建',
    'update': '更新',
    'delete': '删除'
  }
  return map[action] || action
}

// 格式化JSON
const formatJson = (jsonStr: string) => {
  try {
    const obj = JSON.parse(jsonStr)
    return JSON.stringify(obj, null, 2)
  } catch {
    return jsonStr
  }
}

// 实时搜索
watch([() => searchForm.username, () => searchForm.tableName, () => searchForm.action], () => {
  pagination.page = 1
  loadLogList()
})

onMounted(() => {
  loadLogList()
})
</script>

<style scoped>
.data-logs-container {
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
  background: var(--ops-header-bg);
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
  background: linear-gradient(135deg, var(--ops-primary) 0%, var(--ops-primary-light) 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--ops-text-primary);
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--ops-text-secondary);
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 筛选栏 */
.filter-bar {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--ops-header-bg);
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  gap: 12px;
  align-items: center;
}

.filter-input {
  width: 200px;
}

.filter-select {
  width: 140px;
}

.filter-date {
  width: 260px;
}

/* 表格容器 */
.table-wrapper {
  background: var(--ops-header-bg);
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.id-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: var(--ops-text-tertiary);
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-icon {
  color: var(--ops-primary);
  font-size: 16px;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: var(--ops-header-bg);
  border-top: 1px solid var(--ops-border-color);
}

/* 详情对话框 */
.detail-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-info {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  padding: 16px;
  background: var(--ops-content-bg);
  border-radius: 8px;
  border: 1px solid var(--ops-border-color);
}

.info-item {
  display: flex;
  gap: 10px;
  align-items: center;
}

.info-label {
  color: var(--ops-text-secondary);
  font-weight: 600;
  font-size: 14px;
  min-width: 70px;
}

.info-value {
  color: var(--ops-text-primary);
  font-size: 14px;
  font-weight: 500;
}

.data-section {
  margin-top: 8px;
}

.section-title {
  margin: 16px 0 8px 0;
  font-size: 14px;
  color: var(--ops-text-secondary);
  font-weight: 600;
}

.diff-section {
  margin-top: 16px;
}

.json-data {
  background: #1a1a1a;
  color: #4ade80;
  padding: 16px;
  border-radius: 8px;
  font-size: 13px;
  font-family: 'Monaco', 'Menlo', monospace;
  max-height: 300px;
  overflow-y: auto;
  margin: 0;
  border: 1px solid rgba(74, 222, 128, 0.3);
}

/* 深色模式适配 */
body[arco-theme='dark'] .json-data {
  background: #0a0a0a;
  color: #4ade80;
  border-color: rgba(74, 222, 128, 0.2);
}
</style>
