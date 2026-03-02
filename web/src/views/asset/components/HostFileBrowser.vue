<template>
  <a-modal
    v-model:visible="dialogVisible"
    :title="`文件管理 - ${hostName}`"
    :width="1000"
    :mask-closable="false"
    unmount-on-close
    @close="handleClose"
    class="file-browser-dialog"
  >
    <div v-if="loading" class="loading-container">
      <a-spin :size="32" />
      <span class="loading-text">加载中...</span>
    </div>

    <div v-else class="file-browser">
      <!-- 路径导航 -->
      <div class="breadcrumb-card">
        <div class="breadcrumb-header">
          <icon-location class="location-icon" />
          <span class="breadcrumb-title">当前位置</span>
        </div>
        <a-breadcrumb separator="/" class="path-breadcrumb">
          <a-breadcrumb-item @click="navigateTo('~')" class="breadcrumb-home">
            <icon-home />
            <span>主目录</span>
          </a-breadcrumb-item>
          <a-breadcrumb-item
            v-for="(segment, index) in pathSegments"
            :key="index"
            @click="navigateToSegment(index)"
            class="breadcrumb-segment"
          >
            {{ segment }}
          </a-breadcrumb-item>
        </a-breadcrumb>
        <div class="current-path-display">
          <icon-folder class="path-icon" />
          <a-input
            v-model="pathInput"
            placeholder="输入路径后按回车跳转"
            class="path-input"
            allow-clear
            @keyup.enter="handlePathInput"
          >
            <template #prefix>
              <code class="path-prefix">路径:</code>
            </template>
          </a-input>
        </div>
      </div>

      <!-- 工具栏 -->
      <div class="toolbar-actions">
        <div class="toolbar-left">
          <a-button @click="refreshFiles" :loading="loading" class="toolbar-btn">
            <template #icon><icon-refresh /></template>
            <span>刷新</span>
          </a-button>
          <a-button
            @click="navigateUp"
            :disabled="currentPath === '~' || currentPath === '/'"
            class="toolbar-btn"
          >
            <template #icon><icon-left /></template>
            <span>返回上级</span>
          </a-button>
        </div>
        <div class="toolbar-right">
          <a-upload
            :auto-upload="false"
            :show-file-list="false"
            @change="handleUploadChange"
          >
            <template #upload-button>
              <a-button :loading="uploading" class="upload-btn">
                <template #icon><icon-upload /></template>
                <span>上传文件</span>
              </a-button>
            </template>
          </a-upload>
        </div>
      </div>

      <!-- 上传进度条 -->
      <div v-if="uploading" class="upload-progress-container">
        <div class="upload-info">
          <icon-upload class="upload-icon" />
          <span class="upload-filename">{{ uploadingFileName }}</span>
          <span class="upload-status">{{ uploadStatusText }}</span>
        </div>
        <a-progress
          :percent="uploadProgress / 100"
          :stroke-width="8"
          :status="uploadProgress === 100 ? 'success' : 'normal'"
          :animation="isProcessing"
        />
      </div>

      <!-- 文件列表 -->
      <div class="file-list-container">
        <a-table
          :data="files"
          :loading="loading"
          :bordered="{ cell: true }"
          stripe
          class="file-table"
          :pagination="false"
        >
          <template #columns>
            <a-table-column :width="60" align="center">
              <template #cell="{ record }">
                <div class="file-icon-wrapper">
                  <icon-folder v-if="record.isDir" :size="24" :class="getFileIconClass(record)" />
                  <icon-file v-else :size="24" :class="getFileIconClass(record)" />
                </div>
              </template>
            </a-table-column>

            <a-table-column title="名称" :min-width="280">
              <template #cell="{ record }">
                <div
                  class="file-name-cell"
                  :class="{ 'is-directory': record.isDir }"
                  @click="handleFileClick(record)"
                >
                  <span class="file-name-text">{{ record.name }}</span>
                  <a-tag v-if="record.isDir" size="small" color="arcoblue" class="dir-tag">目录</a-tag>
                </div>
              </template>
            </a-table-column>

            <a-table-column title="大小" :width="120" align="right">
              <template #cell="{ record }">
                <span class="file-size">{{ record.isDir ? '-' : formatSize(record.size) }}</span>
              </template>
            </a-table-column>

            <a-table-column title="权限" :width="130" align="center">
              <template #cell="{ record }">
                <a-tag class="permission-tag" size="small">{{ record.mode || '-' }}</a-tag>
              </template>
            </a-table-column>

            <a-table-column title="修改时间" :width="180">
              <template #cell="{ record }">
                <div class="time-cell">
                  <icon-clock-circle class="time-icon" />
                  <span>{{ record.modTime }}</span>
                </div>
              </template>
            </a-table-column>

            <a-table-column title="操作" :width="180" align="center" fixed="right">
              <template #cell="{ record }">
                <div class="action-buttons">
                  <a-button
                    v-if="!record.isDir"
                    type="text"
                    size="small"
                    @click="downloadFile(record)"
                    :loading="downloadingFiles[record.name]"
                    class="action-btn"
                  >
                    <template #icon><icon-download /></template>
                    <span>下载</span>
                  </a-button>
                  <a-popconfirm
                    content="确定删除此文件吗?"
                    @ok="deleteFile(record)"
                    v-if="!record.isDir"
                  >
                    <a-button
                      type="text"
                      status="danger"
                      size="small"
                      :loading="deletingFiles[record.name]"
                      class="action-btn"
                    >
                      <template #icon><icon-delete /></template>
                      <span>删除</span>
                    </a-button>
                  </a-popconfirm>
                  <span v-if="record.isDir" class="no-action">-</span>
                </div>
              </template>
            </a-table-column>
          </template>
        </a-table>

        <a-empty
          v-if="!loading && files.length === 0"
          description="此目录为空"
        >
          <template #image>
            <icon-folder :size="80" style="color: #c0c4cc" />
          </template>
        </a-empty>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconLocation,
  IconHome,
  IconRefresh,
  IconLeft,
  IconFolder,
  IconFile,
  IconDownload,
  IconUpload,
  IconDelete,
  IconClockCircle
} from '@arco-design/web-vue/es/icon'
import { listHostFiles, downloadHostFile, deleteHostFile } from '@/api/host'
import { listAgentFiles, downloadAgentFile, deleteAgentFile } from '@/api/agent'

interface FileInfo {
  name: string
  size: number
  mode: string
  isDir: boolean
  modTime: string
}

const props = defineProps<{
  visible: boolean
  hostId: number
  hostName: string
  connectionMode?: string
  agentStatus?: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const loading = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadingFileName = ref('')
const isProcessing = ref(false) // 是否在服务器处理中
const downloadingFiles = ref<Record<string, boolean>>({})
const deletingFiles = ref<Record<string, boolean>>({})
const files = ref<FileInfo[]>([])
const currentPath = ref('~')
const pathInput = ref('~')

// 计算上传状态文本
const uploadStatusText = computed(() => {
  if (isProcessing.value) {
    return '服务器处理中...'
  } else if (uploadProgress.value < 100) {
    return `上传中 ${uploadProgress.value}%`
  } else {
    return '上传完成'
  }
})

const pathSegments = computed(() => {
  if (currentPath.value === '~' || currentPath.value === '/') return []
  const path = currentPath.value.startsWith('/') ? currentPath.value.slice(1) : currentPath.value.replace('~/', '')
  return path ? path.split('/').filter(p => p) : []
})

const isAgent = computed(() => props.connectionMode === 'agent' && props.agentStatus === 'online')

// 上传相关
const uploadUrl = computed(() => {
  return isAgent.value
    ? `/api/v1/agents/${props.hostId}/files/upload`
    : `/api/v1/hosts/${props.hostId}/files/upload`
})

const uploadHeaders = computed(() => {
  const token = localStorage.getItem('token')
  return {
    Authorization: `Bearer ${token}`
  }
})

const uploadData = computed(() => {
  return {
    path: currentPath.value
  }
})

const getFileIconClass = (file: FileInfo) => {
  if (file.isDir) {
    return 'icon-directory'
  }
  return 'icon-file'
}

const formatSize = (size: number): string => {
  if (!size || size === 0) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(size) / Math.log(k))
  return Math.round((size / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

const loadFiles = async (path: string = '~') => {
  loading.value = true
  try {
    const response = isAgent.value
      ? await listAgentFiles(props.hostId, path)
      : await listHostFiles(props.hostId, path)
    // 响应拦截器已经返回了 data，所以直接使用 response
    files.value = response || []
  } catch (error: any) {
    Message.error('获取文件列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const refreshFiles = () => {
  loadFiles(currentPath.value)
}

const navigateTo = (path: string) => {
  currentPath.value = path
  pathInput.value = path
  loadFiles(path)
}

const handlePathInput = () => {
  if (!pathInput.value || pathInput.value.trim() === '') {
    Message.warning('请输入有效的路径')
    return
  }
  navigateTo(pathInput.value.trim())
}

const navigateUp = () => {
  if (currentPath.value === '~' || currentPath.value === '/') return

  const segments = currentPath.value.split('/').filter(s => s && s !== '~')

  if (segments.length === 0) {
    // 如果没有分段了，返回主目录或根目录
    navigateTo(currentPath.value.startsWith('/') ? '/' : '~')
    return
  }

  segments.pop()

  if (currentPath.value.startsWith('/')) {
    // 绝对路径
    const parentPath = segments.length > 0 ? '/' + segments.join('/') : '/'
    navigateTo(parentPath)
  } else {
    // 相对路径（主目录）
    const parentPath = segments.length > 0 ? '~/' + segments.join('/') : '~'
    navigateTo(parentPath)
  }
}

const navigateToSegment = (index: number) => {
  const segments = pathSegments.value.slice(0, index + 1)
  const path = currentPath.value.startsWith('/')
    ? '/' + segments.join('/')
    : '~/' + segments.join('/')
  navigateTo(path)
}

const handleFileClick = (file: FileInfo) => {
  if (file.isDir) {
    let newPath: string
    if (currentPath.value === '~') {
      newPath = '~/' + file.name
    } else if (currentPath.value === '/') {
      newPath = '/' + file.name
    } else {
      newPath = currentPath.value + '/' + file.name
    }
    navigateTo(newPath)
  }
}

const handleUploadChange = (_fileList: any[], fileItem: any) => {
  if (fileItem.file) {
    uploading.value = true
    uploadProgress.value = 0
    uploadingFileName.value = fileItem.file.name
    isProcessing.value = false
    handleCustomUpload(fileItem.file)
  }
}

const handleCustomUpload = async (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('path', currentPath.value)

  const token = localStorage.getItem('token')

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()

    // 监听上传进度
    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) {
        // 限制在95%，剩余5%表示服务器处理
        const percentComplete = Math.min(Math.round((event.loaded / event.total) * 95), 95)
        uploadProgress.value = percentComplete
      }
    })

    // 上传完成，开始处理
    xhr.upload.addEventListener('load', () => {
      uploadProgress.value = 95
      isProcessing.value = true
    })

    // 监听完成
    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        // 服务器处理完成
        isProcessing.value = false
        uploadProgress.value = 100

        setTimeout(() => {
          uploading.value = false
          uploadProgress.value = 0
          uploadingFileName.value = ''
          Message.success('文件上传成功')
          refreshFiles()
          resolve(xhr.response)
        }, 500)
      } else {
        uploading.value = false
        uploadProgress.value = 0
        uploadingFileName.value = ''
        isProcessing.value = false

        let errorMsg = '未知错误'
        try {
          const response = JSON.parse(xhr.responseText)
          errorMsg = response.message || errorMsg
        } catch (e) {
          errorMsg = xhr.statusText || errorMsg
        }

        Message.error('文件上传失败: ' + errorMsg)
        reject(new Error(errorMsg))
      }
    })

    // 监听错误
    xhr.addEventListener('error', () => {
      uploading.value = false
      uploadProgress.value = 0
      uploadingFileName.value = ''
      isProcessing.value = false
      Message.error('文件上传失败: 网络错误')
      reject(new Error('网络错误'))
    })

    // 监听中止
    xhr.addEventListener('abort', () => {
      uploading.value = false
      uploadProgress.value = 0
      uploadingFileName.value = ''
      isProcessing.value = false
      Message.warning('文件上传已取消')
      reject(new Error('上传已取消'))
    })

    // 打开请求
    xhr.open('POST', isAgent.value
      ? `/api/v1/agents/${props.hostId}/files/upload`
      : `/api/v1/hosts/${props.hostId}/files/upload`)

    // 设置请求头
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }

    // 发送请求
    xhr.send(formData)
  })
}

const downloadFile = async (file: FileInfo) => {
  downloadingFiles.value[file.name] = true
  try {
    const filePath = currentPath.value === '~' || currentPath.value === '/'
      ? (currentPath.value === '~' ? '~/' : '/') + file.name
      : currentPath.value + '/' + file.name

    const response = isAgent.value
      ? await downloadAgentFile(props.hostId, filePath)
      : await downloadHostFile(props.hostId, filePath)

    // 创建下载链接
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)

    Message.success('文件下载成功')
  } catch (error: any) {
    Message.error('文件下载失败: ' + (error.message || '未知错误'))
  } finally {
    downloadingFiles.value[file.name] = false
  }
}

const deleteFile = async (file: FileInfo) => {
  deletingFiles.value[file.name] = true
  try {
    const filePath = currentPath.value === '~' || currentPath.value === '/'
      ? (currentPath.value === '~' ? '~/' : '/') + file.name
      : currentPath.value + '/' + file.name

    isAgent.value
      ? await deleteAgentFile(props.hostId, filePath)
      : await deleteHostFile(props.hostId, filePath)
    Message.success('文件删除成功')
    refreshFiles()
  } catch (error: any) {
    Message.error('文件删除失败: ' + (error.message || '未知错误'))
  } finally {
    deletingFiles.value[file.name] = false
  }
}

const handleClose = () => {
  emit('update:visible', false)
}

// 监听对话框显示状态
watch(() => props.visible, (visible) => {
  if (visible) {
    currentPath.value = '~'
    pathInput.value = '~'
    loadFiles('~')
  }
})
</script>

<style scoped lang="scss">
.file-browser-dialog {
  :deep(.arco-modal-header) {
    border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
    padding: 20px 24px;
  }

  :deep(.arco-modal-body) {
    padding: 24px;
  }
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px;
  gap: 16px;

  .loading-text {
    font-size: 14px;
    color: var(--ops-text-tertiary, #86909c);
  }
}

.file-browser {
  // 面包屑卡片
  .breadcrumb-card {
    background: #ffffff;
    padding: 20px;
    border-radius: 8px;
    margin-bottom: 20px;
    border: 1px solid var(--ops-border-color, #e5e6eb);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

    .breadcrumb-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;

      .location-icon {
        font-size: 18px;
        color: var(--ops-primary, #165dff);
      }

      .breadcrumb-title {
        font-size: 14px;
        font-weight: 600;
        color: var(--ops-text-primary, #1d2129);
      }
    }

    .path-breadcrumb {
      margin-bottom: 12px;

      :deep(.arco-breadcrumb-item) {
        color: var(--ops-text-secondary, #4e5969);
        font-weight: 500;
        cursor: pointer;
        transition: all 0.3s;
        display: inline-flex;
        align-items: center;
        gap: 6px;

        &:hover {
          color: var(--ops-primary, #165dff);
        }
      }

      .breadcrumb-home {
        :deep(.arco-breadcrumb-item-link) {
          font-weight: 600;
          color: var(--ops-primary, #165dff);
        }
      }
    }

    .current-path-display {
      display: flex;
      align-items: center;
      gap: 8px;

      .path-icon {
        font-size: 16px;
        color: var(--ops-primary, #165dff);
        flex-shrink: 0;
      }

      .path-input {
        flex: 1;

        :deep(.arco-input-wrapper) {
          background: #f7f8fa;
          border-radius: 6px;
          border: 1px solid var(--ops-border-color, #e5e6eb);
          transition: all 0.3s;

          &:hover {
            border-color: #c9cdd4;
          }

          &.arco-input-focus {
            border-color: var(--ops-primary, #165dff);
            background: #ffffff;
          }
        }

        :deep(.arco-input) {
          font-family: 'Consolas', 'Monaco', monospace;
          font-size: 13px;
          color: var(--ops-text-primary, #1d2129);
          font-weight: 500;
          letter-spacing: 0.5px;
        }

        .path-prefix {
          font-family: 'Consolas', 'Monaco', monospace;
          font-size: 12px;
          color: var(--ops-text-tertiary, #86909c);
          margin-right: 6px;
        }
      }
    }
  }

  // 工具栏
  .toolbar-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;

    .toolbar-left,
    .toolbar-right {
      display: flex;
      gap: 12px;
    }

    .toolbar-btn {
      border-radius: 8px;
      padding: 10px 20px;
      font-weight: 500;
      transition: all 0.3s;

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      }
    }

    .upload-btn {
      border-radius: 8px;
      padding: 10px 24px;
      font-weight: 500;
      background-color: #232324;
      border-color: #232324;
      color: #ffffff;
      transition: all 0.3s;

      &:hover {
        background-color: #1d1e1f;
        border-color: #1d1e1f;
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
      }
    }
  }

  // 上传进度条
  .upload-progress-container {
    margin-bottom: 20px;
    padding: 16px;
    background: #f7f8fa;
    border-radius: 8px;
    border: 1px solid var(--ops-border-color, #e5e6eb);

    .upload-info {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 12px;

      .upload-icon {
        font-size: 18px;
        color: var(--ops-primary, #165dff);
        animation: upload-pulse 1.5s ease-in-out infinite;
      }

      .upload-filename {
        flex: 1;
        font-size: 14px;
        color: var(--ops-text-primary, #1d2129);
        font-weight: 500;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .upload-status {
        font-size: 13px;
        font-weight: 600;
        color: var(--ops-primary, #165dff);
        margin-left: auto;
        white-space: nowrap;
      }
    }

    :deep(.arco-progress) {
      .arco-progress-line-text {
        display: none;
      }
    }
  }

  @keyframes upload-pulse {
    0%, 100% {
      transform: scale(1);
      opacity: 1;
    }
    50% {
      transform: scale(1.1);
      opacity: 0.8;
    }
  }

  // 文件列表
  .file-list-container {
    .file-table {
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05);

      .file-icon-wrapper {
        display: flex;
        align-items: center;
        justify-content: center;

        .icon-directory {
          color: var(--ops-primary, #165dff);
          transition: all 0.3s;

          &:hover {
            transform: scale(1.1);
          }
        }

        .icon-file {
          color: var(--ops-text-tertiary, #86909c);
        }
      }

      .file-name-cell {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 4px 0;

        .file-name-text {
          font-size: 14px;
          color: var(--ops-text-primary, #1d2129);
          font-weight: 500;
        }

        &.is-directory {
          cursor: pointer;

          .file-name-text {
            color: var(--ops-primary, #165dff);
          }

          &:hover {
            .file-name-text {
              text-decoration: underline;
            }
          }
        }

        .dir-tag {
          border: none;
          font-size: 12px;
        }
      }

      .file-size {
        font-family: 'Consolas', 'Monaco', monospace;
        font-size: 13px;
        color: var(--ops-text-secondary, #4e5969);
        font-weight: 500;
      }

      .permission-tag {
        font-family: 'Consolas', 'Monaco', monospace;
        font-size: 12px;
        background: #f2f3f5;
        color: var(--ops-text-secondary, #4e5969);
        border: 1px solid var(--ops-border-color, #e5e6eb);
        border-radius: 6px;
        padding: 4px 10px;
      }

      .time-cell {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
        color: var(--ops-text-secondary, #4e5969);

        .time-icon {
          font-size: 14px;
          color: var(--ops-text-tertiary, #86909c);
        }
      }

      .action-buttons {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;

        .action-btn {
          font-weight: 500;
          transition: all 0.3s;

          &:hover {
            transform: scale(1.05);
          }
        }

        .no-action {
          color: #c9cdd4;
          font-size: 14px;
        }
      }
    }

    :deep(.arco-empty) {
      padding: 60px 0;
    }
  }
}
</style>