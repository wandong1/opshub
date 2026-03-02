<template>
  <div class="task-list-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-list :size="20" />
        </div>
        <div>
          <h2 class="page-title">任务记录</h2>
          <p class="page-subtitle">查看证书签发、续期、部署等任务执行记录</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button @click="loadData">
          <template #icon><icon-refresh /></template>
          刷新
        </a-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <a-select
          v-model="searchForm.task_type"
          placeholder="任务类型"
          allow-clear
          class="search-input"
          @change="loadData"
        >
          <a-option value="issue">签发证书</a-option>
          <a-option value="renew">续期证书</a-option>
          <a-option value="deploy">部署证书</a-option>
        </a-select>

        <a-select
          v-model="searchForm.status"
          placeholder="任务状态"
          allow-clear
          class="search-input"
          @change="loadData"
        >
          <a-option value="pending">待执行</a-option>
          <a-option value="running">执行中</a-option>
          <a-option value="success">成功</a-option>
          <a-option value="failed">失败</a-option>
        </a-select>

        <a-select
          v-model="searchForm.trigger_type"
          placeholder="触发方式"
          allow-clear
          class="search-input"
          @change="loadData"
        >
          <a-option value="auto">自动</a-option>
          <a-option value="manual">手动</a-option>
        </a-select>
      </div>

      <div class="search-actions">
        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <a-table
        :data="tableData"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50, 100] }"
        @page-change="(p: number) => { pagination.page = p; loadData() }"
        @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }"
      >
        <template #columns>
          <a-table-column title="ID" data-index="id" :width="80" />

          <a-table-column title="关联证书" :min-width="180">
            <template #cell="{ record }">
              <span v-if="record.certificate">{{ record.certificate.name }} ({{ record.certificate.domain }})</span>
              <span v-else>-</span>
            </template>
          </a-table-column>

          <a-table-column title="任务类型" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.task_type === 'issue'" color="arcoblue">签发证书</a-tag>
              <a-tag v-else-if="record.task_type === 'renew'" color="orangered">续期证书</a-tag>
              <a-tag v-else color="green">部署证书</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="状态" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.status === 'pending'" color="gray">待执行</a-tag>
              <a-tag v-else-if="record.status === 'running'" color="orangered" class="running-tag">
                <icon-loading spin style="margin-right: 4px;" />
                <span>执行中</span>
              </a-tag>
              <a-tag v-else-if="record.status === 'success'" color="green">成功</a-tag>
              <a-tag v-else color="red">失败</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="触发方式" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.trigger_type === 'auto'" size="small">自动</a-tag>
              <a-tag v-else size="small" color="gray">手动</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="开始时间" :width="150">
            <template #cell="{ record }">
              <span>{{ formatDateTime(record.started_at) || '-' }}</span>
            </template>
          </a-table-column>

          <a-table-column title="结束时间" :width="150">
            <template #cell="{ record }">
              <span>{{ formatDateTime(record.finished_at) || '-' }}</span>
            </template>
          </a-table-column>

          <a-table-column title="错误信息" :min-width="300" ellipsis tooltip>
            <template #cell="{ record }">
              <span v-if="record.error_message" class="error-text">{{ record.error_message }}</span>
              <span v-else>-</span>
            </template>
          </a-table-column>

          <a-table-column title="操作" :width="80" fixed="right" align="center">
            <template #cell="{ record }">
              <a-tooltip content="查看详情" position="top">
                <a-button type="text" class="action-btn action-view" @click="handleView(record)">
                  <template #icon><icon-eye /></template>
                </a-button>
              </a-tooltip>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 详情对话框 -->
    <a-modal
      v-model:visible="detailDialogVisible"
      title="任务详情"
      :width="640"
      unmount-on-close
      :mask-closable="false"
      :footer="false"
    >
      <div v-if="currentTask" class="detail-content">
        <div class="detail-status-bar" :class="{
          'status-success': currentTask.status === 'success',
          'status-failed': currentTask.status === 'failed',
          'status-running': currentTask.status === 'running',
          'status-pending': currentTask.status === 'pending'
        }">
          <a-tag v-if="currentTask.status === 'pending'" color="gray" size="large">待执行</a-tag>
          <a-tag v-else-if="currentTask.status === 'running'" color="orangered" size="large">执行中</a-tag>
          <a-tag v-else-if="currentTask.status === 'success'" color="green" size="large">成功</a-tag>
          <a-tag v-else color="red" size="large">失败</a-tag>
          <span class="detail-task-type">{{ getTaskTypeName(currentTask.task_type) }}</span>
          <span class="detail-task-id">#{{ currentTask.id }}</span>
        </div>
        <div class="detail-info">
          <div class="detail-info-section">
            <div class="detail-section-title">任务信息</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">任务类型</span>
                <span class="info-value">{{ getTaskTypeName(currentTask.task_type) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">触发方式</span>
                <span class="info-value">{{ currentTask.trigger_type === 'auto' ? '自动' : '手动' }}</span>
              </div>
              <div class="info-item" v-if="currentTask.certificate">
                <span class="info-label">关联证书</span>
                <span class="info-value">{{ currentTask.certificate.name }} ({{ currentTask.certificate.domain }})</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section">
            <div class="detail-section-title">时间</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">创建时间</span>
                <span class="info-value">{{ formatDateTime(currentTask.created_at) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">开始时间</span>
                <span class="info-value">{{ formatDateTime(currentTask.started_at) || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">结束时间</span>
                <span class="info-value">{{ formatDateTime(currentTask.finished_at) || '-' }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section" v-if="currentTask.error_message">
            <div class="detail-section-title error-section-title">错误信息</div>
            <div class="error-block">{{ currentTask.error_message }}</div>
          </div>
          <div class="detail-info-section" v-if="currentTask.result">
            <div class="detail-section-title">执行结果</div>
            <div class="result-block">
              <pre class="result-json">{{ formatJSON(currentTask.result) }}</pre>
            </div>
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconRefresh, IconList, IconEye, IconLoading } from '@arco-design/web-vue/es/icon'
import { getTasks, getTask } from '../api/ssl-cert'

const loading = ref(false)
const detailDialogVisible = ref(false)

// 搜索
const searchForm = reactive({
  task_type: '',
  status: '',
  trigger_type: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 表格数据
const tableData = ref<any[]>([])

// 当前任务
const currentTask = ref<any>(null)

// 获取任务类型名称
const getTaskTypeName = (type: string) => {
  const names: Record<string, string> = {
    issue: '签发证书',
    renew: '续期证书',
    deploy: '部署证书'
  }
  return names[type] || type
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return null
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 格式化JSON
const formatJSON = (jsonStr: string) => {
  try {
    return JSON.stringify(JSON.parse(jsonStr), null, 2)
  } catch {
    return jsonStr
  }
}

// 重置搜索
const handleReset = () => {
  searchForm.task_type = ''
  searchForm.status = ''
  searchForm.trigger_type = ''
  pagination.page = 1
  loadData()
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getTasks({
      page: pagination.page,
      page_size: pagination.pageSize,
      task_type: searchForm.task_type || undefined,
      status: searchForm.status || undefined,
      trigger_type: searchForm.trigger_type || undefined
    })
    tableData.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 查看详情
const handleView = async (row: any) => {
  try {
    const res = await getTask(row.id)
    currentTask.value = res
    detailDialogVisible.value = true
  } catch (error) {
    // 错误已由 request 拦截器处理
  }
}

onMounted(() => {
  loadData()
  // 每30秒自动刷新
  setInterval(() => {
    loadData()
  }, 30000)
})
</script>

<style scoped>
.task-list-container { padding: 0; background-color: transparent; }

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
.header-actions { display: flex; gap: 12px; }

.search-bar {
  margin-bottom: 12px; padding: 12px 16px; background: #fff;
  border-radius: var(--ops-border-radius-md, 8px); box-shadow: 0 2px 12px rgba(0,0,0,0.04);
  display: flex; justify-content: space-between; align-items: center;
}
.search-inputs { display: flex; gap: 12px; }
.search-input { width: 160px; }
.search-actions { display: flex; gap: 10px; }

.table-wrapper {
  background: #fff; border-radius: var(--ops-border-radius-md, 8px);
  box-shadow: 0 2px 12px rgba(0,0,0,0.04); overflow: hidden; padding: 16px;
}
.error-text { color: var(--ops-danger, #f53f3f); }
.action-btn { width: 32px; height: 32px; border-radius: 6px; transition: all 0.2s ease; }
.action-view:hover { background-color: var(--ops-primary-bg, #e8f0ff); color: var(--ops-primary, #165dff); transform: scale(1.1); }
.running-tag { display: inline-flex; align-items: center; }

/* 详情弹窗 */
.detail-content { padding: 0; }
.detail-status-bar {
  display: flex; align-items: center; gap: 12px; margin-bottom: 20px;
  padding: 16px; background: var(--ops-content-bg, #f7f8fa);
  border-radius: 10px; border: 1px solid var(--ops-border-color, #e5e6eb);
}
.detail-task-type { font-size: 18px; font-weight: 600; color: var(--ops-text-primary, #1d2129); }
.detail-task-id { font-size: 14px; color: var(--ops-text-tertiary, #86909c); font-family: 'Monaco', 'Menlo', monospace; margin-left: auto; }
.detail-info { display: flex; flex-direction: column; gap: 16px; }
.detail-info-section { background: #fff; border: 1px solid var(--ops-border-color, #e5e6eb); border-radius: 10px; overflow: hidden; }
.detail-section-title {
  padding: 10px 16px; font-size: 13px; font-weight: 600;
  color: var(--ops-text-tertiary, #86909c); text-transform: uppercase;
  letter-spacing: 0.5px; background: var(--ops-content-bg, #f7f8fa);
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}
.error-section-title { color: var(--ops-danger, #f53f3f); }
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
.error-block { padding: 14px 16px; font-size: 13px; color: var(--ops-danger, #f53f3f); white-space: pre-wrap; word-break: break-word; line-height: 1.6; }
.result-block { padding: 0; }
.result-json {
  background: var(--ops-content-bg, #f7f8fa); padding: 14px 16px; font-size: 12px;
  overflow-x: auto; margin: 0; white-space: pre-wrap; word-break: break-all;
  font-family: 'Monaco', 'Menlo', monospace; line-height: 1.6;
  color: var(--ops-text-secondary, #4e5969);
}
</style>
