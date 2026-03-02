<template>
  <div class="operation-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-file /></div>
        <div>
          <h2 class="page-title">操作日志</h2>
          <p class="page-subtitle">记录系统内的所有操作行为</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button v-permission="'op-logs:search'" type="primary" @click="handleSearch">
          <template #icon><icon-search /></template>
          查询
        </a-button>
        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
        <a-button v-permission="'op-logs:batch-delete'" status="danger" :disabled="selectedIds.length === 0" @click="handleBatchDelete">
          <template #icon><icon-delete /></template>
          批量删除
        </a-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <a-input v-model="searchForm.username" placeholder="搜索用户名..." allow-clear class="filter-input">
        <template #prefix><icon-user /></template>
      </a-input>
      <a-select v-model="searchForm.module" placeholder="选择模块" allow-clear class="filter-select" @change="handleSearch">
        <a-option value="系统管理">系统管理</a-option>
        <a-option value="个人信息">个人信息</a-option>
        <a-option value="操作审计">操作审计</a-option>
        <a-option value="资产管理">资产管理</a-option>
        <a-option value="容器管理">容器管理</a-option>
        <a-option value="监控中心">监控中心</a-option>
        <a-option value="任务中心">任务中心</a-option>
      </a-select>
      <a-select v-model="searchForm.action" placeholder="操作类型" allow-clear class="filter-select" @change="handleSearch">
        <a-option value="查询">查询</a-option>
        <a-option value="创建">创建</a-option>
        <a-option value="更新">更新</a-option>
        <a-option value="删除">删除</a-option>
        <a-option value="登录">登录</a-option>
        <a-option value="登出">登出</a-option>
      </a-select>
      <a-select v-model="searchForm.status" placeholder="状态码" allow-clear class="filter-select" @change="handleSearch">
        <a-option value="2xx">成功 (2xx)</a-option>
        <a-option value="3xx">重定向 (3xx)</a-option>
        <a-option value="4xx">客户端错误 (4xx)</a-option>
        <a-option value="5xx">服务器错误 (5xx)</a-option>
      </a-select>
      <a-range-picker v-model="dateRange" style="width: 260px;" @change="handleDateChange" />
    </div>

    <!-- 数据表格 -->
    <div class="table-wrapper">
      <a-table
        :data="logList"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        row-key="id"
        :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: false }"
        @selection-change="handleSelectionChange"
        :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50, 100] }"
        @page-change="(p: number) => { pagination.page = p; loadLogList() }"
        @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadLogList() }"
      >
        <template #columns>
          <a-table-column title="ID" :width="80" align="center">
            <template #cell="{ record }">
              <span class="id-text">#{{ record.id }}</span>
            </template>
          </a-table-column>
          <a-table-column title="操作用户" :min-width="140">
            <template #cell="{ record }">
              <div class="user-cell">
                <icon-user style="color: var(--ops-primary); flex-shrink: 0;" />
                <span>{{ record.realName || record.username || '-' }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="模块" :min-width="120">
            <template #cell="{ record }">
              <a-tag color="gray" size="small">{{ record.module }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="90">
            <template #cell="{ record }">
              <a-tag :color="getActionColor(record.action)" size="small">{{ record.action }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="操作描述" data-index="description" :min-width="200" ellipsis tooltip />
          <a-table-column title="请求方法" :width="100">
            <template #cell="{ record }">
              <a-tag :color="getMethodColor(record.method)" size="small">{{ record.method }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="请求路径" data-index="path" :min-width="180" ellipsis tooltip />
          <a-table-column title="状态" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.status >= 200 && record.status < 300 ? 'green' : 'red'" size="small">{{ record.status }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="耗时" :width="100" align="right">
            <template #cell="{ record }">
              <span :class="['cost-text', { 'slow-cost': record.costTime > 1000 }]">{{ record.costTime }}ms</span>
            </template>
          </a-table-column>
          <a-table-column title="IP地址" data-index="ip" :width="130" />
          <a-table-column title="操作时间" data-index="createdAt" :width="170" />
          <a-table-column title="操作" :width="80" fixed="right" align="center">
            <template #cell="{ record }">
              <a-tooltip content="删除" position="top">
                <a-button v-permission="'op-logs:delete'" type="text" class="action-btn action-delete" @click="handleDelete(record)">
                  <template #icon><icon-delete /></template>
                </a-button>
              </a-tooltip>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconFile, IconSearch, IconRefresh, IconDelete, IconUser } from '@arco-design/web-vue/es/icon'
import { getOperationLogList, deleteOperationLog, deleteOperationLogsBatch } from '@/api/audit'

const searchForm = reactive({
  username: '', module: '', action: '', status: '', startTime: '', endTime: ''
})
const dateRange = ref<(string | Date)[]>([])
const logList = ref<any[]>([])
const loading = ref(false)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const selectedIds = ref<number[]>([])

const handleDateChange = (val: (string | Date | undefined)[] | undefined) => {
  if (val && val.length === 2 && val[0] && val[1]) {
    searchForm.startTime = String(val[0])
    searchForm.endTime = String(val[1])
  } else {
    searchForm.startTime = ''
    searchForm.endTime = ''
  }
  handleSearch()
}

const loadLogList = async () => {
  loading.value = true
  try {
    const res: any = await getOperationLogList({ page: pagination.page, pageSize: pagination.pageSize, ...searchForm })
    logList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) { /* 错误已由 request 拦截器处理 */ }
  finally { loading.value = false }
}

const handleSearch = () => { pagination.page = 1; loadLogList() }
const handleReset = () => {
  Object.assign(searchForm, { username: '', module: '', action: '', status: '', startTime: '', endTime: '' })
  dateRange.value = []
  pagination.page = 1
  loadLogList()
}

const handleDelete = (row: any) => {
  Modal.warning({
    title: '提示', content: '确定要删除这条日志吗？', hideCancel: false,
    onOk: async () => {
      try { await deleteOperationLog(row.id); Message.success('删除成功'); loadLogList() }
      catch (error) { /* 错误已由 request 拦截器处理 */ }
    }
  })
}

const handleBatchDelete = () => {
  Modal.warning({
    title: '提示', content: `确定要删除选中的 ${selectedIds.value.length} 条日志吗？`, hideCancel: false,
    onOk: async () => {
      try { await deleteOperationLogsBatch(selectedIds.value); Message.success('删除成功'); selectedIds.value = []; loadLogList() }
      catch (error) { /* 错误已由 request 拦截器处理 */ }
    }
  })
}

const handleSelectionChange = (keys: (string | number)[]) => { selectedIds.value = keys.map(Number) }
const getActionColor = (action: string) => {
  const map: Record<string, string> = { '查询': 'gray', '创建': 'green', '更新': 'orangered', '删除': 'red', '登录': 'green', '登出': 'gray' }
  return map[action] || 'gray'
}

const getMethodColor = (method: string) => {
  const map: Record<string, string> = { GET: 'gray', POST: 'green', PUT: 'orangered', DELETE: 'red' }
  return map[method] || 'gray'
}

onMounted(() => { loadLogList() })
</script>

<style scoped>
.operation-logs-container { padding: 0; background-color: transparent; }

.page-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  margin-bottom: 12px; padding: 16px 20px; background: #fff;
  border-radius: var(--ops-border-radius-md, 8px); box-shadow: 0 2px 12px rgba(0,0,0,0.04);
}
.page-title-group { display: flex; align-items: flex-start; gap: 16px; }
.page-title-icon {
  width: 36px; height: 36px; background: var(--ops-primary, #165dff);
  border-radius: 8px; display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 18px; flex-shrink: 0;
}
.page-title { margin: 0; font-size: 20px; font-weight: 600; color: var(--ops-text-primary, #1d2129); }
.page-subtitle { margin: 4px 0 0 0; font-size: 13px; color: var(--ops-text-tertiary, #86909c); }
.header-actions { display: flex; gap: 12px; align-items: center; }

.filter-bar {
  margin-bottom: 12px; padding: 12px 16px; background: #fff;
  border-radius: var(--ops-border-radius-md, 8px); box-shadow: 0 2px 12px rgba(0,0,0,0.04);
  display: flex; gap: 12px; align-items: center; flex-wrap: wrap;
}
.filter-input { width: 200px; }
.filter-select { width: 150px; }

.table-wrapper {
  background: #fff; border-radius: var(--ops-border-radius-md, 8px);
  box-shadow: 0 2px 12px rgba(0,0,0,0.04); overflow: hidden; padding: 16px;
}
.id-text { font-family: 'Monaco', 'Menlo', monospace; font-size: 12px; color: var(--ops-text-tertiary, #86909c); }
.user-cell { display: flex; align-items: center; gap: 8px; }
.cost-text { font-family: 'Monaco', 'Menlo', monospace; font-size: 13px; color: var(--ops-text-secondary, #4e5969); }
.cost-text.slow-cost { color: var(--ops-danger, #f53f3f); font-weight: 600; }
.action-btn { width: 32px; height: 32px; border-radius: 6px; transition: all 0.2s ease; }
.action-delete:hover { background-color: #ffece8; color: var(--ops-danger, #f53f3f); transform: scale(1.1); }
</style>
