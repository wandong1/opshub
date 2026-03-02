<template>
  <div class="mw-audit-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-code /></div>
        <div>
          <h2 class="page-title">中间件审计</h2>
          <p class="page-subtitle">记录中间件控制台执行的所有命令</p>
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
        <a-button status="danger" :disabled="selectedIds.length === 0" @click="handleBatchDelete">
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
      <a-select v-model="searchForm.middlewareType" placeholder="中间件类型" allow-clear class="filter-select" @change="handleSearch">
        <a-option value="mysql">MySQL</a-option>
        <a-option value="redis">Redis</a-option>
        <a-option value="clickhouse">ClickHouse</a-option>
        <a-option value="mongodb">MongoDB</a-option>
        <a-option value="kafka">Kafka</a-option>
        <a-option value="milvus">Milvus</a-option>
      </a-select>
      <a-select v-model="searchForm.commandType" placeholder="命令类型" allow-clear class="filter-select" @change="handleSearch">
        <a-option value="query">查询</a-option>
        <a-option value="insert">新增</a-option>
        <a-option value="update">修改</a-option>
        <a-option value="delete">删除</a-option>
        <a-option value="ddl">DDL</a-option>
        <a-option value="other">其他</a-option>
      </a-select>
      <a-select v-model="searchForm.status" placeholder="执行结果" allow-clear class="filter-select" @change="handleSearch">
        <a-option value="success">成功</a-option>
        <a-option value="failed">失败</a-option>
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
            <template #cell="{ record }"><span class="id-text">#{{ record.id }}</span></template>
          </a-table-column>
          <a-table-column title="用户" :width="100">
            <template #cell="{ record }">
              <div class="user-cell"><icon-user style="color: var(--ops-primary); flex-shrink: 0;" /><span>{{ record.username || '-' }}</span></div>
            </template>
          </a-table-column>
          <a-table-column title="中间件" data-index="middlewareName" :min-width="120" ellipsis tooltip />
          <a-table-column title="类型" :width="110">
            <template #cell="{ record }">
              <a-tag :color="getMwTypeColor(record.middlewareType)" size="small">{{ record.middlewareType }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="数据库" data-index="database" :width="120" ellipsis tooltip />
          <a-table-column title="命令" :min-width="200" ellipsis>
            <template #cell="{ record }">
              <a-link @click="showDetail(record)">{{ truncate(record.command, 60) }}</a-link>
            </template>
          </a-table-column>
          <a-table-column title="命令类型" :width="100">
            <template #cell="{ record }">
              <a-tag :color="getCmdTypeColor(record.commandType)" size="small">{{ cmdTypeLabel(record.commandType) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="结果" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.status === 'success' ? 'green' : 'red'" size="small">{{ record.status === 'success' ? '成功' : '失败' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="耗时" :width="90" align="right">
            <template #cell="{ record }"><span class="cost-text">{{ record.duration }}ms</span></template>
          </a-table-column>
          <a-table-column title="IP" data-index="ip" :width="130" />
          <a-table-column title="时间" data-index="createdAt" :width="170" />
          <a-table-column title="操作" :width="80" fixed="right" align="center">
            <template #cell="{ record }">
              <a-tooltip content="删除" position="top">
                <a-button type="text" class="action-btn action-delete" @click="handleDelete(record)">
                  <template #icon><icon-delete /></template>
                </a-button>
              </a-tooltip>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 命令详情弹窗 -->
    <a-modal v-model:visible="detailVisible" title="命令详情" :width="700" unmount-on-close :footer="false">
      <div v-if="detailRow" class="detail-content">
        <div class="detail-info-section">
          <div class="detail-section-title">基本信息</div>
          <div class="detail-grid">
            <div class="info-item"><span class="info-label">用户</span><span class="info-value">{{ detailRow.username }}</span></div>
            <div class="info-item"><span class="info-label">中间件</span><span class="info-value">{{ detailRow.middlewareName }} ({{ detailRow.middlewareType }})</span></div>
            <div class="info-item"><span class="info-label">数据库</span><span class="info-value">{{ detailRow.database || '-' }}</span></div>
            <div class="info-item"><span class="info-label">命令类型</span><span class="info-value">{{ cmdTypeLabel(detailRow.commandType) }}</span></div>
            <div class="info-item"><span class="info-label">结果</span><span class="info-value">{{ detailRow.status === 'success' ? '成功' : '失败' }}</span></div>
            <div class="info-item"><span class="info-label">耗时</span><span class="info-value">{{ detailRow.duration }}ms</span></div>
            <div class="info-item"><span class="info-label">影响行数</span><span class="info-value">{{ detailRow.affectedRows ?? '-' }}</span></div>
            <div class="info-item"><span class="info-label">IP</span><span class="info-value">{{ detailRow.ip }}</span></div>
          </div>
        </div>
        <div class="detail-info-section" style="margin-top: 16px;">
          <div class="detail-section-title">执行命令</div>
          <pre class="command-block">{{ detailRow.command }}</pre>
        </div>
        <div v-if="detailRow.errorMsg" class="detail-info-section" style="margin-top: 16px;">
          <div class="detail-section-title error-title">错误信息</div>
          <pre class="error-block">{{ detailRow.errorMsg }}</pre>
        </div>
      </div>
    </a-modal>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconCode, IconSearch, IconRefresh, IconDelete, IconUser } from '@arco-design/web-vue/es/icon'
import { getMiddlewareAuditLogList, deleteMiddlewareAuditLog, deleteMiddlewareAuditLogsBatch } from '@/api/audit'

const searchForm = reactive({ username: '', middlewareType: '', commandType: '', status: '', startTime: '', endTime: '' })
const dateRange = ref<(string | Date)[]>([])
const logList = ref<any[]>([])
const loading = ref(false)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const selectedIds = ref<number[]>([])
const detailVisible = ref(false)
const detailRow = ref<any>(null)

const handleDateChange = (val: (string | Date | undefined)[] | undefined) => {
  if (val && val.length === 2 && val[0] && val[1]) {
    searchForm.startTime = String(val[0])
    searchForm.endTime = String(val[1])
  } else { searchForm.startTime = ''; searchForm.endTime = '' }
  handleSearch()
}

const loadLogList = async () => {
  loading.value = true
  try {
    const res: any = await getMiddlewareAuditLogList({ page: pagination.page, pageSize: pagination.pageSize, ...searchForm })
    logList.value = res.list || []
    pagination.total = res.total || 0
  } catch { /* 错误已由 request 拦截器处理 */ }
  finally { loading.value = false }
}

const handleSearch = () => { pagination.page = 1; loadLogList() }
const handleReset = () => {
  Object.assign(searchForm, { username: '', middlewareType: '', commandType: '', status: '', startTime: '', endTime: '' })
  dateRange.value = []
  pagination.page = 1
  loadLogList()
}

const handleDelete = (row: any) => {
  Modal.warning({ title: '提示', content: '确定要删除这条日志吗？', hideCancel: false,
    onOk: async () => { await deleteMiddlewareAuditLog(row.id); Message.success('删除成功'); loadLogList() }
  })
}

const handleBatchDelete = () => {
  Modal.warning({ title: '提示', content: `确定要删除选中的 ${selectedIds.value.length} 条日志吗？`, hideCancel: false,
    onOk: async () => { await deleteMiddlewareAuditLogsBatch(selectedIds.value); Message.success('删除成功'); selectedIds.value = []; loadLogList() }
  })
}

const handleSelectionChange = (keys: (string | number)[]) => { selectedIds.value = keys.map(Number) }
const showDetail = (row: any) => { detailRow.value = row; detailVisible.value = true }
const truncate = (str: string, len: number) => str && str.length > len ? str.substring(0, len) + '...' : str

const cmdTypeLabel = (type: string) => {
  const map: Record<string, string> = { query: '查询', insert: '新增', update: '修改', delete: '删除', ddl: 'DDL', admin: '管理', other: '其他' }
  return map[type] || type || '-'
}
const getCmdTypeColor = (type: string) => {
  const map: Record<string, string> = { query: 'gray', insert: 'green', update: 'orangered', delete: 'red', ddl: 'arcoblue', other: 'gray' }
  return map[type] || 'gray'
}
const getMwTypeColor = (type: string) => {
  const map: Record<string, string> = { mysql: 'arcoblue', redis: 'red', clickhouse: 'orangered', mongodb: 'green', kafka: 'gray', milvus: 'arcoblue' }
  return map[type] || 'gray'
}

onMounted(() => { loadLogList() })
</script>
<style scoped>
.mw-audit-container { padding: 0; background-color: transparent; }

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
.action-btn { width: 32px; height: 32px; border-radius: 6px; transition: all 0.2s ease; }
.action-delete:hover { background-color: #ffece8; color: var(--ops-danger, #f53f3f); transform: scale(1.1); }

/* 详情弹窗 */
.detail-content { padding: 0; }
.detail-info-section { background: #fff; border: 1px solid var(--ops-border-color, #e5e6eb); border-radius: 10px; overflow: hidden; }
.detail-section-title {
  padding: 10px 16px; font-size: 13px; font-weight: 600;
  color: var(--ops-text-tertiary, #86909c); text-transform: uppercase; letter-spacing: 0.5px;
  background: var(--ops-content-bg, #f7f8fa); border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}
.error-title { color: var(--ops-danger, #f53f3f); }
.detail-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0; }
.detail-grid .info-item {
  display: flex; flex-direction: column; gap: 4px; padding: 12px 16px;
  border-bottom: 1px solid #f2f3f5; border-right: 1px solid #f2f3f5;
}
.detail-grid .info-item:nth-child(2n) { border-right: none; }
.detail-grid .info-item:last-child,
.detail-grid .info-item:nth-last-child(2):nth-child(odd) { border-bottom: none; }
.info-label { color: var(--ops-text-tertiary, #86909c); font-size: 12px; font-weight: 500; }
.info-value { color: var(--ops-text-primary, #1d2129); font-size: 14px; word-break: break-all; font-weight: 500; }
.command-block {
  background: var(--ops-content-bg, #f7f8fa); padding: 14px 16px; font-size: 13px;
  overflow-x: auto; margin: 0; white-space: pre-wrap; word-break: break-all;
  font-family: 'Monaco', 'Menlo', monospace; line-height: 1.6; color: var(--ops-text-secondary, #4e5969);
}
.error-block {
  padding: 14px 16px; font-size: 13px; color: var(--ops-danger, #f53f3f);
  white-space: pre-wrap; word-break: break-word; line-height: 1.6; margin: 0;
  font-family: 'Monaco', 'Menlo', monospace;
}
</style>
