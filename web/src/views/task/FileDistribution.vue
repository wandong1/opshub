<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-folder />
        </div>
        <div>
          <div class="page-title">文件分发</div>
          <div class="page-desc">上传文件并分发到多台主机指定目录</div>
        </div>
      </div>
    </div>

    <div class="distribution-body">
      <div class="distribution-main">
        <!-- 上传文件 -->
        <a-card class="section-card" :bordered="false">
          <template #title><span class="card-title">上传文件</span></template>
          <a-upload
            draggable
            multiple
            :auto-upload="false"
            @change="handleFileChange"
            :show-file-list="false"
          >
            <template #upload-button>
              <div class="upload-area">
                <icon-upload style="font-size: 40px; color: #c9cdd4" />
                <div class="upload-text">拖拽文件到此处或点击上传</div>
                <div class="upload-hint">支持任意格式文件</div>
              </div>
            </template>
          </a-upload>

          <div v-if="fileList.length > 0" class="file-list">
            <div v-for="(file, index) in fileList" :key="index" class="file-item">
              <icon-file style="font-size: 18px; color: var(--ops-primary, #165dff)" />
              <span class="file-name">{{ file.name }}</span>
              <span class="file-size">{{ formatFileSize(file.file?.size || 0) }}</span>
              <a-button type="text" status="danger" size="mini" @click="removeFile(index)">
                <template #icon><icon-close /></template>
              </a-button>
            </div>
          </div>
          <div v-else class="empty-state">暂无上传文件</div>
        </a-card>

        <!-- 分发目标 -->
        <a-card class="section-card" :bordered="false">
          <template #title><span class="card-title">分发目标</span></template>
          <div class="form-section">
            <div class="section-label"><span class="required">*</span> 目标路径</div>
            <a-input v-model="targetPath" placeholder="请输入目标路径，如 /tmp 或 /home/user" allow-clear />
          </div>
          <div class="form-section">
            <div class="section-label"><span class="required">*</span> 目标主机</div>
            <a-button @click="showHostDialog = true">
              <template #icon><icon-plus /></template>
              添加目标主机
            </a-button>
            <div v-if="selectedHosts.length > 0" class="tag-list">
              <a-tag
                v-for="host in selectedHosts"
                :key="host.id"
                closable
                :color="host.agentStatus === 'online' ? 'green' : 'arcoblue'"
                @close="removeHost(host.id)"
              >
                {{ host.name }} ({{ host.ip }})
                <span v-if="host.agentStatus === 'online'" style="margin-left: 4px; font-size: 11px;">Agent</span>
              </a-tag>
            </div>
          </div>
        </a-card>

        <!-- 分发方式 -->
        <div class="form-section" style="margin-bottom: 16px;">
          <div class="section-label">分发方式</div>
          <div class="section-body">
            <a-radio-group v-model="distributionMode" type="button">
              <a-radio value="auto">自动选择（Agent在线优先）</a-radio>
              <a-radio value="ssh">SSH</a-radio>
              <a-radio value="agent">Agent</a-radio>
            </a-radio-group>
            <div v-if="selectedHosts.length > 0 && distributionMode === 'auto'" class="execution-hint">
              <span style="color: #00b42a;">Agent在线: {{ selectedHosts.filter(h => h.agentStatus === 'online').length }}台</span>
              <span style="margin-left: 12px; color: #86909c;">SSH回退: {{ selectedHosts.filter(h => h.agentStatus !== 'online').length }}台</span>
            </div>
          </div>
        </div>

        <!-- 执行按钮 -->
        <div class="execute-btn-wrap">
          <a-button
            v-permission="'task-distribute:execute'"
            type="primary"
            size="large"
            :loading="distributing"
            @click="handleDistribute"
          >
            <template #icon><icon-play-arrow-fill /></template>
            {{ distributing ? '分发中...' : '开始执行' }}
          </a-button>
        </div>
      </div>

      <!-- 分发记录 -->
      <div class="distribution-log">
        <div class="log-header">
          <span class="log-title">分发记录</span>
          <a-link v-if="distributionLogs.length > 0" @click="clearLogs">清空</a-link>
        </div>
        <div class="log-terminal">
          <div v-if="distributionLogs.length === 0" class="log-empty">暂无分发记录</div>
          <div v-else class="log-list">
            <div
              v-for="log in distributionLogs"
              :key="log.id"
              class="log-item"
              :class="log.status"
            >
              <div class="log-meta">
                <span class="log-time">{{ log.time }}</span>
                <a-tag :color="log.status === 'success' ? 'green' : log.status === 'error' ? 'red' : 'arcoblue'" size="small">
                  {{ log.status === 'success' ? '成功' : log.status === 'error' ? '失败' : '信息' }}
                </a-tag>
              </div>
              <div class="log-message">{{ log.message }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 选择主机对话框 -->
    <a-modal
      v-model:visible="showHostDialog"
      title="选择主机"
      :width="800"
      :unmount-on-close="true"
      @ok="confirmHostSelection"
      @cancel="showHostDialog = false"
    >
      <a-input
        v-model="hostSearchKeyword"
        placeholder="输入名称/IP搜索"
        allow-clear
        style="margin-bottom: 12px"
      >
        <template #prefix><icon-search /></template>
      </a-input>
      <a-table
        :data="filteredHosts"
        :row-selection="{ type: 'checkbox', showCheckedAll: true }"
        v-model:selectedKeys="tempSelectedHostIds"
        row-key="id"
        :loading="hostsLoading"
        :scroll="{ y: 360 }"
        :pagination="false"
      >
        <template #columns>
          <a-table-column title="主机名称" data-index="name" />
          <a-table-column title="IP地址" data-index="ip">
            <template #cell="{ record }">
              <a-tag size="small">{{ record.ip }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="备注信息" data-index="description" />
        </template>
      </a-table>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconFolder, IconUpload, IconFile, IconClose, IconPlus,
  IconPlayArrowFill, IconSearch
} from '@arco-design/web-vue/es/icon'
import { getHostList } from '@/api/host'
import { getAgentStatuses } from '@/api/agent'
import { distributeFiles } from '@/api/task'

const fileList = ref<any[]>([])
const targetPath = ref('')
const selectedHosts = ref<any[]>([])
const distributing = ref(false)
const distributionMode = ref('auto')
const distributionLogs = ref<any[]>([])

// 主机对话框
const showHostDialog = ref(false)
const hostSearchKeyword = ref('')
const tempSelectedHostIds = ref<(string | number)[]>([])
const allHosts = ref<any[]>([])
const hostsLoading = ref(false)

const filteredHosts = computed(() => {
  if (!hostSearchKeyword.value) return allHosts.value
  const keyword = hostSearchKeyword.value.toLowerCase()
  return allHosts.value.filter(
    (host) => host.name.toLowerCase().includes(keyword) || host.ip.includes(keyword)
  )
})

const handleFileChange = (_fileList: any[], fileItem: any) => {
  fileList.value = _fileList
}

const removeFile = (index: number) => {
  fileList.value.splice(index, 1)
}

const formatFileSize = (size: number) => {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
  if (size < 1024 * 1024 * 1024) return (size / (1024 * 1024)).toFixed(2) + ' MB'
  return (size / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const removeHost = (id: number) => {
  selectedHosts.value = selectedHosts.value.filter(h => h.id !== id)
}

const loadHostList = async () => {
  hostsLoading.value = true
  try {
    const response = await getHostList({ page: 1, pageSize: 1000 })
    if (Array.isArray(response)) allHosts.value = response
    else if (response.list) allHosts.value = response.list
    else if (response.data) allHosts.value = response.data
    else allHosts.value = []
    // 获取Agent实时状态并合并
    try {
      const statuses = await getAgentStatuses()
      if (Array.isArray(statuses)) {
        const map: Record<number, any> = {}
        for (const s of statuses) {
          map[s.hostId] = s
        }
        for (const host of allHosts.value) {
          const agentInfo = map[host.id]
          if (agentInfo) {
            host.agentStatus = agentInfo.status
            host.connectionMode = 'agent'
          }
        }
      }
    } catch (e) {}
  } catch { allHosts.value = [] }
  finally { hostsLoading.value = false }
}

const confirmHostSelection = () => {
  selectedHosts.value = allHosts.value.filter(h => tempSelectedHostIds.value.includes(h.id))
  showHostDialog.value = false
  Message.success(`已选择 ${selectedHosts.value.length} 台主机`)
}

const addLog = (message: string, status = 'info') => {
  const now = new Date()
  const time = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`
  distributionLogs.value.unshift({ id: Date.now(), time, message, status })
}

const clearLogs = () => { distributionLogs.value = [] }

const handleDistribute = async () => {
  if (fileList.value.length === 0) { Message.warning('请先上传文件'); return }
  if (!targetPath.value.trim()) { Message.warning('请输入目标路径'); return }
  if (selectedHosts.value.length === 0) { Message.warning('请选择目标主机'); return }

  distributing.value = true
  const fileNames = fileList.value.map(f => f.name).join(', ')
  addLog(`开始分发文件: ${fileNames}`, 'info')
  addLog(`目标路径: ${targetPath.value}`, 'info')
  addLog(`目标主机: ${selectedHosts.value.length} 台`, 'info')

  try {
    const formData = new FormData()
    fileList.value.forEach((file) => {
      if (file.file) formData.append('files', file.file)
    })
    formData.append('targetPath', targetPath.value)
    formData.append('hostIds', JSON.stringify(selectedHosts.value.map(h => h.id)))
    formData.append('distributionMode', distributionMode.value)

    const response = await distributeFiles(formData)

    if (response.results && Array.isArray(response.results)) {
      let successCount = 0
      let failCount = 0
      response.results.forEach((result: any) => {
        if (result.status === 'success') {
          successCount++
          addLog(`${result.hostName}(${result.hostIp}): 分发成功`, 'success')
        } else {
          failCount++
          addLog(`${result.hostName}(${result.hostIp}): ${result.error || '分发失败'}`, 'error')
        }
      })
      if (failCount === 0) {
        Message.success(`文件分发完成，全部 ${successCount} 台主机成功`)
        addLog(`分发完成，全部 ${successCount} 台主机成功`, 'success')
      } else {
        Message.warning(`分发完成，成功 ${successCount} 台，失败 ${failCount} 台`)
        addLog(`分发完成，成功 ${successCount} 台，失败 ${failCount} 台`, 'info')
      }
    } else {
      addLog('文件分发完成', 'success')
      Message.success('文件分发完成')
    }
    fileList.value = []
  } catch (error: any) {
    const errMsg = error.message || '分发失败'
    addLog('文件分发失败: ' + errMsg, 'error')
    Message.error('文件分发失败: ' + errMsg)
  } finally {
    distributing.value = false
  }
}

onMounted(() => { loadHostList() })
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

.distribution-body {
  display: flex;
  gap: 16px;
  flex: 1;
  min-height: 0;
}

.distribution-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-card {
  border-radius: var(--ops-border-radius-md, 8px);

  .card-title {
    font-weight: 600;
  }
}

.upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px 0;
  cursor: pointer;
}

.upload-text {
  font-size: 14px;
  font-weight: 500;
  color: var(--ops-text-primary, #1d2129);
}

.upload-hint {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 16px;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: var(--ops-bg-secondary, #f7f8fa);
  border-radius: 6px;

  .file-name {
    flex: 1;
    font-size: 14px;
    color: var(--ops-text-primary, #1d2129);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file-size {
    font-size: 12px;
    color: var(--ops-text-tertiary, #86909c);
    flex-shrink: 0;
  }
}

.empty-state {
  text-align: center;
  color: var(--ops-text-tertiary, #86909c);
  padding: 40px 0;
}

.form-section {
  margin-bottom: 20px;

  &:last-child {
    margin-bottom: 0;
  }
}

.section-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 10px;

  .required {
    color: var(--ops-danger, #f53f3f);
    margin-right: 4px;
  }
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.execution-hint {
  margin-top: 8px;
  font-size: 12px;
}

.execute-btn-wrap {
  display: flex;
  justify-content: center;
  padding: 16px;
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
}

/* 分发记录侧栏 */
.distribution-log {
  width: 380px;
  min-width: 380px;
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.log-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;

  .log-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--ops-text-primary, #1d2129);
  }
}

.log-terminal {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
  background: #1e1e1e;
}

.log-empty {
  text-align: center;
  color: #666;
  padding: 40px 0;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-item {
  background: #2a2a2a;
  border-radius: 6px;
  padding: 10px 12px;
  border-left: 3px solid #555;

  &.success { border-left-color: var(--ops-success, #00b42a); }
  &.error { border-left-color: var(--ops-danger, #f53f3f); }
  &.info { border-left-color: var(--ops-primary, #165dff); }
}

.log-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.log-time {
  color: #888;
  font-family: 'Consolas', monospace;
  font-size: 12px;
}

.log-message {
  font-size: 13px;
  color: #d4d4d4;
  word-break: break-all;
  line-height: 1.5;
}
</style>
