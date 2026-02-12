<template>
  <div class="operation-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <h2 class="page-title">中间件审计</h2>
          <p class="page-subtitle">记录中间件控制台执行的所有命令</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleSearch">
          <el-icon style="margin-right: 6px;"><Search /></el-icon>
          查询
        </el-button>
        <el-button class="black-button" @click="handleReset">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          重置
        </el-button>
        <el-button class="black-button danger" @click="handleBatchDelete" :disabled="selectedIds.length === 0">
          <el-icon style="margin-right: 6px;"><Delete /></el-icon>
          批量删除
        </el-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-input v-model="searchForm.username" placeholder="搜索用户名..." clearable class="filter-input">
        <template #prefix><el-icon class="filter-icon"><User /></el-icon></template>
      </el-input>
      <el-select v-model="searchForm.middlewareType" placeholder="中间件类型" clearable class="filter-select">
        <el-option label="MySQL" value="mysql" />
        <el-option label="Redis" value="redis" />
        <el-option label="ClickHouse" value="clickhouse" />
        <el-option label="MongoDB" value="mongodb" />
        <el-option label="Kafka" value="kafka" />
        <el-option label="Milvus" value="milvus" />
      </el-select>
      <el-select v-model="searchForm.commandType" placeholder="命令类型" clearable class="filter-select">
        <el-option label="查询" value="query" />
        <el-option label="新增" value="insert" />
        <el-option label="修改" value="update" />
        <el-option label="删除" value="delete" />
        <el-option label="DDL" value="ddl" />
        <el-option label="其他" value="other" />
      </el-select>
      <el-select v-model="searchForm.status" placeholder="执行结果" clearable class="filter-select">
        <el-option label="成功" value="success" />
        <el-option label="失败" value="failed" />
      </el-select>
      <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" value-format="YYYY-MM-DD" class="filter-date" />
    </div>

    <!-- 数据表格 -->
    <div class="table-wrapper">
      <el-table :data="logList" v-loading="loading" @selection-change="handleSelectionChange" class="modern-table" size="default">
        <el-table-column type="selection" width="55" />
        <el-table-column label="ID" prop="id" width="80" align="center">
          <template #default="{ row }"><span class="id-text">#{{ row.id }}</span></template>
        </el-table-column>
        <el-table-column label="用户" prop="username" width="100">
          <template #default="{ row }">
            <div class="user-cell"><el-icon class="user-icon"><User /></el-icon><span>{{ row.username || '-' }}</span></div>
          </template>
        </el-table-column>
        <el-table-column label="中间件" prop="middlewareName" min-width="120" show-overflow-tooltip />
        <el-table-column label="类型" prop="middlewareType" width="110">
          <template #default="{ row }">
            <el-tag :type="getMwTypeTag(row.middlewareType)" size="small">{{ row.middlewareType }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="数据库" prop="database" width="120" show-overflow-tooltip />
        <el-table-column label="命令" prop="command" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showDetail(row)">{{ truncate(row.command, 60) }}</el-button>
          </template>
        </el-table-column>
        <el-table-column label="命令类型" prop="commandType" width="100">
          <template #default="{ row }">
            <el-tag :type="getCmdTypeTag(row.commandType)" size="small">{{ cmdTypeLabel(row.commandType) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="结果" prop="status" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">{{ row.status === 'success' ? '成功' : '失败' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="耗时" prop="duration" width="90" align="right">
          <template #default="{ row }"><span class="cost-text">{{ row.duration }}ms</span></template>
        </el-table-column>
        <el-table-column label="IP" prop="ip" width="130" />
        <el-table-column label="时间" prop="createdAt" width="170" />
        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-tooltip content="删除" placement="top">
              <el-button link class="action-btn danger" @click="handleDelete(row)"><el-icon :size="18"><Delete /></el-icon></el-button>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrapper">
        <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.pageSize" :page-sizes="[10, 20, 50, 100]" :total="pagination.total" layout="total, sizes, prev, pager, next" @size-change="loadLogList" @current-change="loadLogList" />
      </div>
    </div>

    <!-- 命令详情弹窗 -->
    <el-dialog v-model="detailVisible" title="命令详情" width="700px" destroy-on-close>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="用户">{{ detailRow?.username }}</el-descriptions-item>
        <el-descriptions-item label="中间件">{{ detailRow?.middlewareName }} ({{ detailRow?.middlewareType }})</el-descriptions-item>
        <el-descriptions-item label="数据库">{{ detailRow?.database || '-' }}</el-descriptions-item>
        <el-descriptions-item label="命令类型">{{ cmdTypeLabel(detailRow?.commandType) }}</el-descriptions-item>
        <el-descriptions-item label="结果">{{ detailRow?.status === 'success' ? '成功' : '失败' }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ detailRow?.duration }}ms</el-descriptions-item>
        <el-descriptions-item label="影响行数">{{ detailRow?.affectedRows ?? '-' }}</el-descriptions-item>
        <el-descriptions-item label="IP">{{ detailRow?.ip }}</el-descriptions-item>
      </el-descriptions>
      <div style="margin-top: 16px;">
        <p style="font-weight: 600; margin-bottom: 8px;">执行命令：</p>
        <pre style="background: #f5f7fa; padding: 12px; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; word-break: break-all; font-size: 13px;">{{ detailRow?.command }}</pre>
      </div>
      <div v-if="detailRow?.errorMsg" style="margin-top: 12px;">
        <p style="font-weight: 600; margin-bottom: 8px; color: #f56c6c;">错误信息：</p>
        <pre style="background: #fef0f0; padding: 12px; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; word-break: break-all; font-size: 13px; color: #f56c6c;">{{ detailRow?.errorMsg }}</pre>
      </div>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Document, Delete, Search, Refresh, User } from '@element-plus/icons-vue'
import { getMiddlewareAuditLogList, deleteMiddlewareAuditLog, deleteMiddlewareAuditLogsBatch } from '@/api/audit'

const searchForm = reactive({
  username: '',
  middlewareType: '',
  commandType: '',
  status: '',
  startTime: '',
  endTime: ''
})

const dateRange = ref<[string, string]>([])
watch(dateRange, (newVal) => {
  if (newVal && newVal.length === 2) {
    searchForm.startTime = newVal[0]
    searchForm.endTime = newVal[1]
  } else {
    searchForm.startTime = ''
    searchForm.endTime = ''
  }
})

const logList = ref<any[]>([])
const loading = ref(false)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const selectedIds = ref<number[]>([])
const detailVisible = ref(false)
const detailRow = ref<any>(null)

const loadLogList = async () => {
  loading.value = true
  try {
    const res: any = await getMiddlewareAuditLogList({ page: pagination.page, pageSize: pagination.pageSize, ...searchForm })
    logList.value = res.list || []
    pagination.total = res.total || 0
  } catch { ElMessage.error('获取日志列表失败') } finally { loading.value = false }
}

const handleSearch = () => { pagination.page = 1; loadLogList() }
const handleReset = () => {
  Object.assign(searchForm, { username: '', middlewareType: '', commandType: '', status: '', startTime: '', endTime: '' })
  dateRange.value = []
  pagination.page = 1
  loadLogList()
}
const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定要删除这条日志吗?', '提示', { type: 'warning' }).then(async () => {
    await deleteMiddlewareAuditLog(row.id)
    ElMessage.success('删除成功')
    loadLogList()
  }).catch(() => {})
}

const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 条日志吗?`, '提示', { type: 'warning' }).then(async () => {
    await deleteMiddlewareAuditLogsBatch(selectedIds.value)
    ElMessage.success('删除成功')
    selectedIds.value = []
    loadLogList()
  }).catch(() => {})
}

const handleSelectionChange = (selection: any[]) => { selectedIds.value = selection.map(item => item.id) }
const showDetail = (row: any) => { detailRow.value = row; detailVisible.value = true }
const truncate = (str: string, len: number) => str && str.length > len ? str.substring(0, len) + '...' : str

const cmdTypeLabel = (type: string) => {
  const map: Record<string, string> = { query: '查询', insert: '新增', update: '修改', delete: '删除', ddl: 'DDL', admin: '管理', other: '其他' }
  return map[type] || type || '-'
}

const getCmdTypeTag = (type: string) => {
  const map: Record<string, string> = { query: 'info', insert: 'success', update: 'warning', delete: 'danger', ddl: '', other: 'info' }
  return map[type] || 'info'
}

const getMwTypeTag = (type: string) => {
  const map: Record<string, string> = { mysql: '', redis: 'danger', clickhouse: 'warning', mongodb: 'success', kafka: 'info', milvus: '' }
  return map[type] || 'info'
}

watch([() => searchForm.username, () => searchForm.middlewareType, () => searchForm.commandType, () => searchForm.status], () => {
  pagination.page = 1
  loadLogList()
})

onMounted(() => { loadLogList() })
</script>
<style scoped>
.operation-logs-container { padding: 0; background-color: transparent; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; padding: 16px 20px; background: #fff; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.04); }
.page-title-group { display: flex; align-items: flex-start; gap: 16px; }
.page-title-icon { width: 48px; height: 48px; background: linear-gradient(135deg, #000 0%, #1a1a1a 100%); border-radius: 10px; display: flex; align-items: center; justify-content: center; color: #d4af37; font-size: 22px; flex-shrink: 0; border: 1px solid #d4af37; }
.page-title { margin: 0; font-size: 20px; font-weight: 600; color: #303133; line-height: 1.3; }
.page-subtitle { margin: 4px 0 0 0; font-size: 13px; color: #909399; line-height: 1.4; }
.header-actions { display: flex; gap: 12px; align-items: center; }
.black-button { background-color: #000 !important; color: #fff !important; border-color: #000 !important; border-radius: 8px; padding: 10px 20px; font-weight: 500; }
.black-button:hover { background-color: #333 !important; border-color: #333 !important; }
.black-button.danger { background-color: #f56c6c !important; border-color: #f56c6c !important; }
.black-button.danger:hover { background-color: #f78989 !important; }
.black-button:disabled { background-color: #c0c4cc !important; border-color: #c0c4cc !important; }
.filter-bar { margin-bottom: 16px; padding: 12px 16px; background: #fff; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.04); display: flex; gap: 12px; align-items: center; }
.filter-input { width: 200px; }
.filter-select { width: 140px; }
.filter-date { width: 260px; }
.filter-icon { color: #d4af37; }
.table-wrapper { background: #fff; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.04); overflow: hidden; }
.modern-table { width: 100%; }
.modern-table :deep(.el-table__row) { transition: background-color 0.2s ease; height: 56px !important; }
.modern-table :deep(.el-table__row td) { height: 56px !important; }
.modern-table :deep(.el-table__row:hover) { background-color: #f8fafc !important; }
.id-text { font-family: 'Monaco','Menlo',monospace; font-size: 12px; color: #909399; }
.user-cell { display: flex; align-items: center; gap: 8px; }
.user-icon { color: #d4af37; font-size: 16px; }
.cost-text { font-family: 'Monaco','Menlo',monospace; font-size: 13px; color: #606266; }
.action-btn { color: #d4af37; padding: 4px; }
.action-btn.danger { color: #f56c6c; }
.action-btn.danger:hover { color: #f78989; }
.pagination-wrapper { display: flex; justify-content: flex-end; padding: 16px 20px; background: #fff; border-top: 1px solid #f0f0f0; }
</style>
