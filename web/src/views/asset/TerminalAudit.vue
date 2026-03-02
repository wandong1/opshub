<template>
  <div class="terminal-audit-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-desktop /></div>
        <div>
          <h2 class="page-title">终端审计</h2>
          <p class="page-subtitle">查看和管理SSH终端会话录制</p>
        </div>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="filter-bar">
      <div class="search-section">
        <a-input
          v-model="searchKeyword"
          placeholder="搜索主机名、IP或用户名..."
          allow-clear
          class="search-input"
          @press-enter="loadSessions"
          @clear="loadSessions"
        >
          <template #prefix><icon-search /></template>
        </a-input>
      </div>
      <a-button @click="handleRefresh"><template #icon><icon-refresh /></template>重置</a-button>
    </div>

    <!-- 数据表格 -->
    <div class="table-card">
    <a-table
      :data="filteredSessions"
      :loading="loading"
      :bordered="false"
      stripe
      :pagination="{
        current: page,
        pageSize: pageSize,
        total: total,
        showTotal: true,
        showPageSize: true,
        pageSizeOptions: [10, 20, 50, 100]
      }"
      @page-change="handlePageChange"
      @page-size-change="handleSizeChange"
      class="modern-table"
      :header-cell-style="{ background: '#fafbfc', color: '#4e5969', fontWeight: '600' }"
    >
      <template #columns>
        <a-table-column title="ID" data-index="id" :width="80" align="center" />

        <a-table-column title="主机信息" :width="220">
          <template #cell="{ record }">
            <div class="host-info">
              <div class="host-name">
                <icon-desktop style="color: #165dff;" />
                <span>{{ record.hostName }}</span>
              </div>
              <div class="host-ip">{{ record.hostIp }}</div>
            </div>
          </template>
        </a-table-column>

        <a-table-column title="操作用户" :width="150" align="center">
          <template #cell="{ record }">
            <a-tooltip :content="record.username" position="top">
              <a-tag color="gray" class="username-tag">
                <icon-user />
                <span class="username-text">{{ record.username }}</span>
              </a-tag>
            </a-tooltip>
          </template>
        </a-table-column>

        <a-table-column title="连接方式" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag v-if="record.connectionType === 'agent'" color="green" size="small">
              <icon-cloud /> Agent
            </a-tag>
            <a-tag v-else color="gray" size="small">
              SSH
            </a-tag>
          </template>
        </a-table-column>

        <a-table-column title="时长" data-index="durationText" :width="100" align="center" />

        <a-table-column title="文件大小" data-index="fileSizeText" :width="110" align="center" />

        <a-table-column title="状态" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag :color="getStatusColor(record.status)">{{ record.statusText }}</a-tag>
          </template>
        </a-table-column>

        <a-table-column title="创建时间" data-index="createdAtText" :width="180" align="center" />

        <a-table-column title="操作" :width="160" align="center" fixed="right">
          <template #cell="{ record }">
            <div class="action-buttons">
              <a-tooltip content="播放" position="top">
                <a-button
                  type="text"
                  class="action-btn action-play"
                  :loading="playingSession === record.id"
                  @click="handlePlay(record)"
                >
                  <template #icon><icon-video-camera /></template>
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" position="top">
                <a-button
                  v-permission="'terminal-sessions:delete'"
                  type="text"
                  status="danger"
                  class="action-btn"
                  @click="handleDeleteClick(record)"
                >
                  <template #icon><icon-delete /></template>
                </a-button>
              </a-tooltip>
            </div>
          </template>
        </a-table-column>
      </template>
    </a-table>
    </div>

    <!-- 播放对话框 -->
    <a-modal
      v-model:visible="playerVisible"
      :title="`终端回放 - ${currentSession?.hostName}`"
      :width="1000"
      unmount-on-close
      :mask-closable="false"
      class="terminal-player-modal"
      @close="handlePlayerClose"
    >
      <AsciinemaPlayer
        v-if="recordingUrl && playerVisible"
        :src="recordingUrl"
        :autoplay="true"
      />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconDesktop, IconSearch, IconUser, IconVideoCamera, IconDelete, IconRefresh, IconCloud } from '@arco-design/web-vue/es/icon'
import { getTerminalSessions, playTerminalSession, deleteTerminalSession } from '@/api/terminal'
import AsciinemaPlayer from '@/components/AsciinemaPlayer.vue'

interface TerminalSession {
  id: number
  hostId: number
  hostName: string
  hostIp: string
  userId: number
  username: string
  duration: number
  durationText: string
  fileSize: number
  fileSizeText: string
  status: string
  statusText: string
  connectionType: string
  connectionTypeText: string
  createdAt: string
  createdAtText: string
}

const loading = ref(false)
const sessions = ref<TerminalSession[]>([])
const searchKeyword = ref('')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// 播放相关
const playerVisible = ref(false)
const recordingUrl = ref('')
const currentSession = ref<TerminalSession | null>(null)
const playingSession = ref(0)

// 删除相关
const deletingSession = ref(0)

// 过滤后的会话列表
const filteredSessions = computed(() => {
  if (!searchKeyword.value) {
    return sessions.value
  }

  const keyword = searchKeyword.value.toLowerCase()
  return sessions.value.filter(item =>
    item.hostName?.toLowerCase().includes(keyword) ||
    item.hostIp?.toLowerCase().includes(keyword) ||
    item.username?.toLowerCase().includes(keyword)
  )
})

// 加载会话列表
const loadSessions = async () => {
  loading.value = true
  try {
    const response = await getTerminalSessions({
      page: page.value,
      pageSize: pageSize.value,
      keyword: searchKeyword.value
    })
    sessions.value = response.list || []
    total.value = response.total || 0
  } catch (error: any) {
    Message.error('加载会话列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 播放会话
const handlePlay = async (session: TerminalSession) => {
  playingSession.value = session.id
  try {
    const response = await playTerminalSession(session.id)

    // response 是 responseType: 'text' 返回的原始字符串
    if (!response) {
      Message.error('录制文件内容为空')
      return
    }

    // 释放旧的 Blob URL
    if (recordingUrl.value) {
      URL.revokeObjectURL(recordingUrl.value)
      recordingUrl.value = ''
    }

    // 创建Blob URL — asciinema 录制文件是 NDJSON 格式
    const blob = new Blob([response], { type: 'text/plain' })
    recordingUrl.value = URL.createObjectURL(blob)
    currentSession.value = session
    playerVisible.value = true
  } catch (error: any) {
    Message.error('加载录制文件失败: ' + (error.message || '未知错误'))
  } finally {
    playingSession.value = 0
  }
}

// 删除会话
const handleDeleteClick = (row: TerminalSession) => {
  Modal.warning({
    title: '提示',
    content: '确定删除此会话录制吗？',
    hideCancel: false,
    onOk: async () => {
      await handleDelete(row.id)
    }
  })
}

const handleDelete = async (id: number) => {
  deletingSession.value = id
  try {
    await deleteTerminalSession(id)
    Message.success('删除成功')
    loadSessions()
  } catch (error: any) {
    Message.error('删除失败: ' + (error.message || '未知错误'))
  } finally {
    deletingSession.value = 0
  }
}

// 刷新
const handleRefresh = () => {
  searchKeyword.value = ''
  page.value = 1
  loadSessions()
}

// 分页变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  page.value = 1
  loadSessions()
}

const handlePageChange = (p: number) => {
  page.value = p
  loadSessions()
}

// 关闭播放器
const handlePlayerClose = () => {
  if (recordingUrl.value) {
    URL.revokeObjectURL(recordingUrl.value)
    recordingUrl.value = ''
  }
  currentSession.value = null
}

// 获取状态颜色（Arco tag color）
const getStatusColor = (status: string): string => {
  const colorMap: Record<string, string> = {
    completed: 'green',
    recording: 'orangered',
    failed: 'red'
  }
  return colorMap[status] || 'gray'
}

onMounted(() => {
  loadSessions()
})
</script>

<style scoped>
.terminal-audit-container { padding: 0; height: 100%; display: flex; flex-direction: column; }

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #165dff;
  font-size: 22px;
  flex-shrink: 0;
}
.page-title { margin: 0; font-size: 20px; font-weight: 600; color: #1d2129; line-height: 1.3; }
.page-subtitle { margin: 4px 0 0; font-size: 13px; color: #86909c; line-height: 1.4; }

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  align-items: center;
}
.search-section { flex: 1; }
.search-input { width: 280px; }

.table-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 16px;
  flex: 1;
}

/* 主机信息 */
.host-info .host-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: #1d2129;
  margin-bottom: 4px;
}
.host-info .host-ip {
  font-size: 12px;
  color: #86909c;
  font-family: 'Consolas', 'Monaco', monospace;
}

/* 用户名标签 */
.username-tag {
  display: inline-flex !important;
  align-items: center;
  gap: 4px;
}
.username-tag .username-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100px;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  align-items: center;
  justify-content: center;
}
.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
}

/* 播放对话框 */
:deep(.terminal-player-modal .arco-modal-body) {
  padding: 0;
  background: #000;
}
</style>
