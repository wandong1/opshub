<template>
  <div class="audit-container">
    <!-- 页面标题和操作按钮 -->
    <a-card class="page-header-card">
      <div class="page-header">
        <div class="page-title-group">
          <div class="page-title-icon">
            <icon-desktop />
          </div>
          <div>
            <h2 class="page-title">终端审计</h2>
            <p class="page-subtitle">查看用户终端操作记录和会话回放</p>
          </div>
        </div>
        <div class="header-actions">
          <a-button type="primary" @click="loadSessions">
            <template #icon><icon-refresh /></template>
            刷新
          </a-button>
        </div>
      </div>
    </a-card>

    <!-- 搜索栏 -->
    <a-card class="search-card">
      <a-form layout="inline" class="search-form">
        <a-form-item>
          <a-input
            v-model="searchPod"
            placeholder="搜索 Pod 名称..."
            allow-clear
            @clear="handleSearch"
            @input="handleSearch"
            style="width: 260px"
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 终端会话列表 -->
    <a-card class="table-card">
      <a-table
        :data="paginatedSessions"
        :loading="loading"
        :bordered="false"
        size="default"
       :columns="tableColumns">
          <template #id="{ record }">
            <span class="id-text">#{{ record.id }}</span>
          </template>
          <template #clusterName="{ record }">
            <div class="cluster-cell">
              <icon-apps />
              <span>{{ record.clusterName }}</span>
            </div>
          </template>
          <template #podName="{ record }">
            <div class="pod-cell">
              <icon-storage />
              <span class="pod-name">{{ record.podName }}</span>
            </div>
          </template>
          <template #containerName="{ record }">
            <a-tag size="small" color="gray">{{ record.containerName }}</a-tag>
          </template>
          <template #username="{ record }">
            <div class="user-cell">
              <icon-user />
              <span>{{ record.username }}</span>
            </div>
          </template>
          <template #duration="{ record }">
            <span class="duration-text">{{ formatDuration(record.duration) }}</span>
          </template>
          <template #createdAt="{ record }">
            {{ formatTime(record.createdAt) }}
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="播放" placement="top">
                <a-button type="text" class="action-btn" @click="handlePlay(record)">
                  <icon-play-arrow />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button type="text" class="action-btn danger" @click="handleDelete(record)">
                  <icon-delete />
                </a-button>
              </a-tooltip>
            </div>
          </template>
        </a-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="currentPage"
          v-model:page-size="pageSize"
          :page-size-options="[10, 20, 50, 100]"
          :total="filteredSessions.length"
          show-total
          show-page-size
        />
      </div>
    </a-card>

    <!-- 播放弹窗 -->
    <a-modal
      v-model:visible="playDialogVisible"
      :title="`终端回放 - ${selectedSession?.podName}`"
      width="90%"
      class="play-dialog"
      @close="handleClosePlay"
    >
      <div class="play-container">
        <div class="play-info">
          <div class="info-item">
            <span class="info-label">集群:</span>
            <span class="info-value">{{ selectedSession?.clusterName || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">命名空间:</span>
            <span class="info-value">{{ selectedSession?.namespace }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Pod:</span>
            <span class="info-value">{{ selectedSession?.podName }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Container:</span>
            <span class="info-value">{{ selectedSession?.containerName }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">用户:</span>
            <span class="info-value">{{ selectedSession?.username }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">时长:</span>
            <span class="info-value">{{ selectedSession ? formatDuration(selectedSession.duration) : '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">大小:</span>
            <span class="info-value">{{ selectedSession ? formatFileSize(selectedSession.fileSize) : '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">时间:</span>
            <span class="info-value">{{ selectedSession ? formatTime(selectedSession.createdAt) : '' }}</span>
          </div>
        </div>
        <div class="player-wrapper" v-if="playDialogVisible">
          <AsciinemaPlayer
            v-if="recordingUrl"
            :src="recordingUrl"
            :cols="120"
            :rows="30"
            :autoplay="true"
            :preload="true"
          />
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { title: 'ID', dataIndex: 'id', slotName: 'id', width: 80, align: 'center' },
  { title: '集群', dataIndex: 'clusterName', slotName: 'clusterName', width: 150 },
  { title: '命名空间', dataIndex: 'namespace', width: 150 },
  { title: 'Pod', dataIndex: 'podName', slotName: 'podName', width: 180 },
  { title: 'Container', dataIndex: 'containerName', slotName: 'containerName', width: 150 },
  { title: '用户', dataIndex: 'username', slotName: 'username', width: 120 },
  { title: '时长', dataIndex: 'duration', slotName: 'duration', width: 100, align: 'center' },
  { title: '创建时间', dataIndex: 'createdAt', slotName: 'createdAt', width: 180 },
  { title: '操作', slotName: 'actions', width: 120, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import request from '@/utils/request'
import axios from 'axios'
import AsciinemaPlayer from '@/components/AsciinemaPlayer.vue'

interface TerminalSession {
  id: number
  clusterId: number
  clusterName: string
  namespace: string
  podName: string
  containerName: string
  userId: number
  username: string
  duration: number
  fileSize: number
  createdAt: string
}

const loading = ref(false)
const sessionList = ref<TerminalSession[]>([])

// 搜索
const searchPod = ref('')

// 分页
const currentPage = ref(1)
const pageSize = ref(10)

// 播放弹窗
const playDialogVisible = ref(false)
const selectedSession = ref<TerminalSession | null>(null)
const recordingUrl = ref('')

// 过滤后的列表
const filteredSessions = computed(() => {
  let result = sessionList.value

  if (searchPod.value) {
    result = result.filter(s =>
      s.podName.toLowerCase().includes(searchPod.value.toLowerCase())
    )
  }

  return result
})

// 分页后的列表
const paginatedSessions = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredSessions.value.slice(start, end)
})

// 格式化时长
const formatDuration = (seconds: number) => {
  if (!seconds) return '-'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60

  if (h > 0) {
    return `${h}h ${m}m ${s}s`
  }
  if (m > 0) {
    return `${m}m ${s}s`
  }
  return `${s}s`
}

// 格式化文件大小
const formatFileSize = (bytes: number) => {
  if (!bytes) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = bytes
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }

  return `${size.toFixed(1)} ${units[unitIndex]}`
}

// 格式化时间
const formatTime = (timeStr: string) => {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 加载会话列表
const loadSessions = async () => {
  loading.value = true
  try {
    const response = await request.get(`/api/v1/plugins/kubernetes/terminal/sessions`)
    // 响应拦截器已经返回了 res.data，所以 response 直接就是数组
    sessionList.value = response || []
  } catch (error: any) {
    sessionList.value = []
    // 如果是404或空列表，显示友好的提示
    if (error.response?.status === 404 || error.response?.data?.data?.length === 0) {
      Message.info('暂无终端会话记录')
    } else {
      Message.error('获取终端会话列表失败')
    }
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// 播放会话
const handlePlay = async (row: TerminalSession) => {
  selectedSession.value = row

  try {
    // 使用原生 axios 获取录制文件，因为后端直接返回文件内容（不是标准响应格式）
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/terminal/sessions/${row.id}/play`, {
      headers: {
        Authorization: `Bearer ${token}`
      }
    })


    // 将数据转换为字符串并创建 blob
    let jsonString: string
    if (typeof response.data === 'string') {
      jsonString = response.data
    } else if (response.data instanceof ArrayBuffer) {
      // 如果是 ArrayBuffer，转换为字符串
      const decoder = new TextDecoder('utf-8')
      jsonString = decoder.decode(response.data)
    } else {
      jsonString = JSON.stringify(response.data)
    }

    const blob = new Blob([jsonString], { type: 'application/json' })
    recordingUrl.value = URL.createObjectURL(blob)

    playDialogVisible.value = true
  } catch (error: any) {
    Message.error('获取录制文件失败')
  }
}

// 关闭播放弹窗
const handleClosePlay = () => {
  if (recordingUrl.value) {
    URL.revokeObjectURL(recordingUrl.value)
    recordingUrl.value = ''
  }
  selectedSession.value = null
}

// 删除会话
const handleDelete = async (row: TerminalSession) => {
  try {
    await confirmModal(
      `确定要删除终端会话记录吗？此操作不可恢复！`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    await request.delete(`/api/v1/plugins/kubernetes/terminal/sessions/${row.id}`)

    Message.success('删除成功')
    await loadSessions()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

onMounted(() => {
  loadSessions()
})
</script>

<style scoped>
.audit-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部卡片 */
.page-header-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  background: var(--ops-primary, #165dff);
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
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 搜索卡片 */
.search-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

/* 表格卡片 */
.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.id-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

.cluster-cell, .pod-cell, .user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pod-name {
  font-weight: 600;
  color: var(--ops-primary, #165dff);
}

.duration-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}

.action-btn {
  color: var(--ops-primary, #165dff);
  padding: 4px;
}

.action-btn:hover {
  color: #4080ff;
}

.action-btn.danger {
  color: var(--ops-danger, #f53f3f);
}

.action-btn.danger:hover {
  color: #f76560;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0 0;
}

/* 播放弹窗 */
.play-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.play-info {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  padding: 16px;
  background: var(--ops-content-bg, #f7f8fa);
  border-radius: var(--ops-border-radius-md, 8px);
  border: 1px solid var(--ops-border-color, #e5e6eb);
}

.info-item {
  display: flex;
  gap: 10px;
  align-items: center;
}

.info-label {
  color: var(--ops-text-tertiary, #86909c);
  font-weight: 500;
  font-size: 13px;
  min-width: 60px;
}

.info-value {
  color: var(--ops-text-primary, #1d2129);
  font-size: 13px;
  font-weight: 500;
}

.player-wrapper {
  background: #000;
  border-radius: var(--ops-border-radius-md, 8px);
  overflow: hidden;
  min-height: 400px;
  aspect-ratio: 16/9;
}
</style>
