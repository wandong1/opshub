<template>
  <div class="audit-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <div>
          <h2 class="page-title">终端审计</h2>
          <p class="page-subtitle">查看用户终端操作记录和会话回放</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="loadSessions">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <el-input
        v-model="searchPod"
        placeholder="搜索 Pod 名称..."
        clearable
        class="search-input"
        @input="handleSearch"
      >
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <!-- 终端会话列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedSessions"
        v-loading="loading"
        class="modern-table"
        size="default"
      >
        <el-table-column label="ID" prop="id" width="80" align="center">
          <template #default="{ row }">
            <span class="id-text">#{{ row.id }}</span>
          </template>
        </el-table-column>

        <el-table-column label="集群" prop="clusterName" min-width="150">
          <template #default="{ row }">
            <div class="cluster-cell">
              <el-icon class="cluster-icon"><Platform /></el-icon>
              <span>{{ row.clusterName }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="命名空间" prop="namespace" width="150" />

        <el-table-column label="Pod" prop="podName" min-width="180">
          <template #default="{ row }">
            <div class="pod-cell">
              <el-icon class="pod-icon"><Box /></el-icon>
              <span class="pod-name">{{ row.podName }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Container" prop="containerName" width="150">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.containerName }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="用户" prop="username" width="120">
          <template #default="{ row }">
            <div class="user-cell">
              <el-icon class="user-icon"><User /></el-icon>
              <span>{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="时长" prop="duration" width="100" align="center">
          <template #default="{ row }">
            <span class="duration-text">{{ formatDuration(row.duration) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="播放" placement="top">
                <el-button link class="action-btn" @click="handlePlay(row)">
                  <el-icon :size="18"><VideoPlay /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn danger" @click="handleDelete(row)">
                  <el-icon :size="18"><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredSessions.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- 播放弹窗 -->
    <el-dialog
      v-model="playDialogVisible"
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
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  Refresh,
  Platform,
  Box,
  User,
  VideoPlay,
  Delete,
  Monitor
} from '@element-plus/icons-vue'
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
    console.log('🔍 终端会话响应:', response)
    // 响应拦截器已经返回了 res.data，所以 response 直接就是数组
    sessionList.value = response || []
    console.log('🔍 sessionList 设置后:', sessionList.value)
  } catch (error: any) {
    console.error('获取终端会话列表失败:', error)
    sessionList.value = []
    // 如果是404或空列表，显示友好的提示
    if (error.response?.status === 404 || error.response?.data?.data?.length === 0) {
      ElMessage.info('暂无终端会话记录')
    } else {
      ElMessage.error('获取终端会话列表失败')
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

    console.log('📼 录制文件内容:', response.data)

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
    console.error('获取录制文件失败:', error)
    ElMessage.error('获取录制文件失败')
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
    await ElMessageBox.confirm(
      `确定要删除终端会话记录吗？此操作不可恢复！`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    await request.delete(`/api/v1/plugins/kubernetes/terminal/sessions/${row.id}`)

    ElMessage.success('删除成功')
    await loadSessions()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
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

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff;
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
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  gap: 16px;
}

.search-input {
  width: 320px;
}

.search-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
  height: 56px !important;
}

.modern-table :deep(.el-table__row td) {
  height: 56px !important;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.id-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #909399;
}

.cluster-cell, .pod-cell, .user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cluster-icon, .pod-icon, .user-icon {
  color: #d4af37;
  font-size: 16px;
}

.pod-name {
  font-weight: 600;
  color: #d4af37;
}

.duration-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}

.action-btn {
  color: #d4af37;
  padding: 4px;
}

.action-btn:hover {
  color: #bfa13f;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn.danger:hover {
  color: #f78989;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* 播放弹窗 */
.play-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #000000 0%, #1a1a1a 100%);
  color: #d4af37;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.play-dialog :deep(.el-dialog__title) {
  color: #d4af37;
  font-size: 16px;
  font-weight: 600;
}

.play-dialog :deep(.el-dialog__body) {
  padding: 24px;
  background-color: #1a1a1a;
}

.play-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.play-info {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  border: 1px solid #d4af37;
}

.info-item {
  display: flex;
  gap: 10px;
  align-items: center;
}

.info-label {
  color: #d4af37;
  font-weight: 600;
  font-size: 15px;
  min-width: 60px;
}

.info-value {
  color: #000000;
  font-size: 15px;
  font-weight: 500;
}

.player-wrapper {
  background: #000;
  border-radius: 8px;
  border: 1px solid #d4af37;
  overflow: hidden;
  min-height: 400px;
  aspect-ratio: 16/9;
}
</style>
