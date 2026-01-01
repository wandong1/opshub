<template>
  <div class="cluster-auth-content" v-if="modelValue">
    <!-- 操作栏 -->
    <div class="action-bar">
      <div class="page-info">
        <div class="info-icon">
          <el-icon><User /></el-icon>
        </div>
        <div class="info-text">
          <div class="info-title">已申请凭据的用户</div>
          <div class="info-desc">共 {{ uniqueCredentialUsers.length }} 位用户已申请该集群的 kubeconfig 访问凭据</div>
        </div>
      </div>
      <el-button @click="handleRefresh" :loading="loading">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <!-- 用户表格 -->
    <div class="table-wrapper">
      <el-table
        :data="uniqueCredentialUsers"
        v-loading="loading"
        class="user-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column width="60">
          <template #default="{ row }">
            <div class="table-avatar">
              <el-icon><User /></el-icon>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="realName" label="用户名" min-width="120">
          <template #default="{ row }">
            <div class="user-cell">
              <div class="user-name">{{ row.realName || row.username }}</div>
              <div class="user-username">@{{ row.username }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="serviceAccount" label="ServiceAccount" min-width="180">
          <template #default="{ row }">
            <span class="mono-text">{{ row.serviceAccount }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="namespace" label="命名空间" width="150">
          <template #default="{ row }">
            <span class="mono-text">{{ row.namespace }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="申请时间" width="170">
          <template #default="{ row }">
            <span class="time-text">{{ formatDate(row.createdAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default>
            <el-tag type="success" effect="plain" size="small">已授权</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button link class="action-view" @click="handleViewCredential(row)" title="查看凭据">
                <el-icon><Document /></el-icon>
              </el-button>
              <el-button link class="action-revoke" @click="handleRevoke(row)" title="吊销">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-empty
      v-if="!loading && !uniqueCredentialUsers.length"
      description="暂无用户申请凭据"
      :image-size="100"
    />

    <!-- KubeConfig 查看对话框 -->
    <el-dialog
      v-model="showKubeConfigDialog"
      title="查看 KubeConfig 凭据"
      width="800px"
      append-to-body
    >
      <div class="kubeconfig-dialog">
        <div class="kubeconfig-info">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="用户名">{{ currentUser?.username }}</el-descriptions-item>
            <el-descriptions-item label="真实姓名">{{ currentUser?.realName }}</el-descriptions-item>
            <el-descriptions-item label="ServiceAccount">{{ currentUser?.serviceAccount }}</el-descriptions-item>
            <el-descriptions-item label="命名空间">{{ currentUser?.namespace }}</el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="kubeconfig-actions">
          <el-button type="primary" @click="handleCopyKubeConfig">
            <el-icon><DocumentCopy /></el-icon>
            复制
          </el-button>
          <el-button @click="handleDownloadKubeConfig">
            <el-icon><Download /></el-icon>
            下载
          </el-button>
        </div>

        <div class="code-editor-wrapper">
          <div class="line-numbers">
            <div v-for="n in configLineCount" :key="n" class="line-number">{{ n }}</div>
          </div>
          <textarea
            v-model="currentKubeConfig"
            class="code-textarea"
            readonly
            spellcheck="false"
          ></textarea>
        </div>

        <div class="code-tip">
          <el-icon><Warning /></el-icon>
          <span>此凭据文件包含您的集群访问权限，请妥善保管，不要泄露给他人</span>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  User,
  Refresh,
  FolderOpened,
  Document,
  DocumentCopy,
  Download,
  Delete,
  Warning
} from '@element-plus/icons-vue'
import {
  getServiceAccountKubeConfig,
  getClusterCredentialUsers,
  revokeCredentialFully,
  type Cluster,
  type CredentialUser
} from '@/api/kubernetes'

interface Props {
  cluster: Cluster | null
  modelValue: boolean
  credentialUsers?: CredentialUser[]
}

const props = defineProps<Props>()
const emit = defineEmits(['update:modelValue', 'refresh'])

const loading = ref(false)
const showKubeConfigDialog = ref(false)
const currentUser = ref<any>(null)
const currentKubeConfig = ref('')
const configLineCount = ref(1)

// 去重的用户列表
const uniqueCredentialUsers = computed(() => {
  if (!props.credentialUsers || props.credentialUsers.length === 0) return []
  const userMap = new Map<number, CredentialUser>()
  props.credentialUsers.forEach(user => {
    const existing = userMap.get(user.userId)
    if (!existing || new Date(user.createdAt) > new Date(existing.createdAt)) {
      userMap.set(user.userId, user)
    }
  })
  return Array.from(userMap.values()).sort((a, b) =>
    new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  )
})

// 监听 kubeconfig 内容变化
watch(currentKubeConfig, () => {
  configLineCount.value = currentKubeConfig.value.split('\n').length || 1
})

// 方法
const handleRefresh = () => {
  emit('refresh')
  ElMessage.success('刷新成功')
}

const handleViewCredential = async (user: any) => {
  try {
    if (!props.cluster) return
    currentUser.value = user
    loading.value = true
    const result = await getServiceAccountKubeConfig(props.cluster.id, user.serviceAccount)
    currentKubeConfig.value = result.kubeconfig
    showKubeConfigDialog.value = true
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '获取 kubeconfig 失败')
  } finally {
    loading.value = false
  }
}

const handleCopyKubeConfig = async () => {
  try {
    await navigator.clipboard.writeText(currentKubeConfig.value)
    ElMessage.success('复制成功')
  } catch {
    ElMessage.error('复制失败')
  }
}

const handleDownloadKubeConfig = () => {
  const blob = new Blob([currentKubeConfig.value], { type: 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `kubeconfig-${currentUser?.username || 'user'}-${props.cluster?.name || 'cluster'}.yaml`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('下载成功')
}

const handleRevoke = async (user: any) => {
  try {
    await ElMessageBox.confirm(
      `确定要吊销用户 "${user.realName || user.username}" 的凭据吗？\n\n吊销将同时删除：\n• K8s 中的 ServiceAccount\n• 所有相关的 RoleBinding\n• 数据库中的凭据记录\n\n吊销后用户将无法访问该集群！`,
      '确认吊销',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        dangerouslyUseHTMLString: true
      }
    )

    if (!props.cluster) return

    loading.value = true
    await revokeCredentialFully(props.cluster.id, user.serviceAccount, user.username)
    ElMessage.success('吊销成功')
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '吊销失败')
    }
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}
</script>

<style scoped lang="scss">
.cluster-auth-content {
  padding: 0;
}

/* 操作栏 */
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 16px 20px;
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  border-radius: 12px;
  border: 1px solid #d4af37;
}

.page-info {
  display: flex;
  align-items: center;
  gap: 14px;
}

.info-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 20px;
  flex-shrink: 0;
}

.info-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-title {
  font-size: 16px;
  font-weight: 600;
  color: #d4af37;
}

.info-desc {
  font-size: 12px;
  color: #909399;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.user-table {
  width: 100%;

  :deep(.el-table__body-wrapper) {
    border-radius: 0 0 12px 12px;
  }

  :deep(.el-table__row) {
    transition: background-color 0.2s ease;

    &:hover {
      background-color: #f8fafc !important;
    }
  }

  :deep(.el-table__cell) {
    padding: 12px 0;
  }
}

/* 表格头像 */
.table-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border: 1px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 16px;
  margin: 0 auto;
}

/* 用户单元格 */
.user-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.user-username {
  font-size: 12px;
  color: #909399;
}

/* 单行文本 */
.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 13px;
  color: #606266;
}

.time-text {
  font-size: 13px;
  color: #909399;
}

.namespace-tag {
  max-width: 160px;
}

.namespace-text {
  display: inline-block;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
  line-height: 1.5;
}

/* 表格操作 */
.table-actions {
  display: flex;
  gap: 16px;
  align-items: center;
}

.action-view {
  color: #000;
  font-weight: 500;

  &:hover {
    color: #d4af37;
  }

  // 当只有图标时的样式
  &:has(> .el-icon:only-child) {
    :deep(.el-icon) {
      margin-right: 0;
    }
  }
}

.action-revoke {
  color: #f56c6c;

  &:hover {
    color: #f56c6c;
    background-color: #fef0f0;
  }
}

/* KubeConfig 对话框 */
.kubeconfig-dialog {
  .kubeconfig-info {
    margin-bottom: 20px;
  }

  .kubeconfig-actions {
    display: flex;
    gap: 12px;
    margin-bottom: 16px;
  }

  .code-editor-wrapper {
    display: flex;
    border: 1px solid #dcdfe6;
    border-radius: 8px;
    overflow: hidden;
    background-color: #282c34;
  }

  .line-numbers {
    display: flex;
    flex-direction: column;
    padding: 12px 8px;
    background-color: #21252b;
    border-right: 1px solid #3e4451;
    user-select: none;
    min-width: 40px;
    text-align: right;
  }

  .line-number {
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 13px;
    line-height: 1.6;
    color: #5c6370;
    min-height: 20.8px;
  }

  .code-textarea {
    flex: 1;
    min-height: 350px;
    padding: 12px;
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 13px;
    line-height: 1.6;
    color: #abb2bf;
    background-color: #282c34;
    border: none;
    outline: none;
    resize: vertical;
  }

  .code-tip {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    padding: 10px 14px;
    background: #fef0f0;
    border-radius: 6px;
    color: #f56c6c;
    font-size: 13px;

    :deep(.el-icon) {
      font-size: 16px;
    }
  }
}
</style>
