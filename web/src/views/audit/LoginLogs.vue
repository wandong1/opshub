<template>
  <div class="login-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-check-circle />
        </div>
        <div>
          <h2 class="page-title">登录日志</h2>
          <p class="page-subtitle">记录系统用户登录和登出行为</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button v-permission="'login-logs:search'" type="primary" @click="handleSearch">
          <template #icon><icon-search /></template>
          查询
        </a-button>
        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
        <a-button v-permission="'login-logs:batch-delete'" status="danger" @click="handleBatchDelete" :disabled="selectedIds.length === 0">
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
        v-model="searchForm.loginType"
        placeholder="登录类型"
        allow-clear
        class="filter-select"
      >
        <a-option label="Web" value="web" />
        <a-option label="SSH" value="ssh" />
        <a-option label="API" value="api" />
      </a-select>
      <a-select
        v-model="searchForm.loginStatus"
        placeholder="登录状态"
        allow-clear
        class="filter-select"
      >
        <a-option label="成功" value="success" />
        <a-option label="失败" value="failed" />
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
          <a-table-column title="用户名" data-index="username" :width="140">
            <template #cell="{ record }">
              <div class="user-cell">
                <icon-user class="user-icon" />
                <span>{{ record.realName || record.username }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="登录类型" data-index="loginType" :width="100">
            <template #cell="{ record }">
              <a-tag :color="getLoginTypeColor(record.loginType)">
                {{ record.loginType.toUpperCase() }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="状态" data-index="loginStatus" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.loginStatus === 'success' ? 'green' : 'red'">
                {{ record.loginStatus === 'success' ? '成功' : '失败' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="登录时间" data-index="loginTime" :width="170" />
          <a-table-column title="登出时间" data-index="logoutTime" :width="170">
            <template #cell="{ record }">
              <span>{{ record.logoutTime || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="IP地址" data-index="ip" :width="130" />
          <a-table-column title="登录地点" data-index="location" :width="120" :ellipsis="true" :tooltip="true" />
          <a-table-column title="失败原因" data-index="failReason" :width="150" :ellipsis="true" :tooltip="true">
            <template #cell="{ record }">
              <span v-if="record.loginStatus === 'failed'" class="fail-reason">{{ record.failReason || '未知错误' }}</span>
              <span v-else>-</span>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="80" fixed="right" align="center">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="删除">
                  <a-button v-permission="'login-logs:delete'" type="text" status="danger" @click="handleDelete(record)">
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconCheckCircle,
  IconDelete,
  IconSearch,
  IconRefresh,
  IconUser
} from '@arco-design/web-vue/es/icon'
import { getLoginLogList, deleteLoginLog, deleteLoginLogsBatch } from '@/api/audit'

// 搜索表单
const searchForm = reactive({
  username: '',
  loginType: '',
  loginStatus: '',
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

// 加载日志列表
const loadLogList = async () => {
  loading.value = true
  try {
    const res: any = await getLoginLogList({
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
  searchForm.loginType = ''
  searchForm.loginStatus = ''
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
        await deleteLoginLog(row.id)
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
        await deleteLoginLogsBatch(selectedIds.value)
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

// 获取登录类型标签颜色
const getLoginTypeColor = (type: string) => {
  const map: Record<string, string> = {
    'web': 'green',
    'ssh': 'orange',
    'api': 'blue'
  }
  return map[type] || 'blue'
}

// 实时搜索
watch([() => searchForm.username, () => searchForm.loginType, () => searchForm.loginStatus], () => {
  pagination.page = 1
  loadLogList()
})

onMounted(() => {
  loadLogList()
})
</script>

<style scoped>
.login-logs-container {
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

.fail-reason {
  color: var(--ops-danger);
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
</style>
