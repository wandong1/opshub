<template>
  <div class="clusters-container">
    <!-- 统计卡片 -->
    <a-row :gutter="20" class="stats-row">
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-primary, #165dff)">
              <icon-apps :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ clusterList.length }}</div>
              <div class="stat-label">集群总数</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-success, #00b42a)">
              <icon-check-circle :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ clusterList.filter(c => c.status === 1).length }}</div>
              <div class="stat-label">运行正常</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-warning, #ff7d00)">
              <icon-dashboard :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ totalNodeCount }}</div>
              <div class="stat-label">总节点数</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: #722ed1">
              <icon-link :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ clusterList.filter(c => c.provider === 'native').length }}</div>
              <div class="stat-label">自建集群</div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 页面标题和操作按钮 -->
    <a-card class="page-header-card">
      <div class="page-header">
        <div class="page-title-group">
          <div class="page-title-icon">
            <icon-apps />
          </div>
          <div>
            <h2 class="page-title">集群管理</h2>
            <p class="page-subtitle">管理您的 Kubernetes 集群，支持多云平台统一管理</p>
          </div>
        </div>
        <div class="header-actions">
          <a-button v-permission="'k8s-clusters:sync'" status="success" @click="handleSyncAll" :loading="syncing">
            <template #icon><icon-refresh /></template>
            同步状态
          </a-button>
          <a-button v-if="isAdmin" v-permission="'k8s-clusters:create'" type="primary" @click="handleRegister">
            <template #icon><icon-plus /></template>
            注册集群
          </a-button>
        </div>
      </div>
    </a-card>

    <!-- 搜索和筛选 -->
    <a-card class="search-card">
      <a-form :model="searchForm" layout="inline" class="search-form">
        <a-form-item>
          <a-input
            v-model="searchForm.keyword"
            placeholder="搜索集群名称或别名..."
            allow-clear
            @clear="handleSearch"
            @keyup.enter="handleSearch"
            style="width: 260px"
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item>
          <a-select
            v-model="searchForm.status"
            placeholder="集群状态"
            allow-clear
            @change="handleSearch"
            style="width: 150px"
          >
            <a-option label="正常" :value="1" />
            <a-option label="连接失败" :value="2" />
            <a-option label="不可用" :value="3" />
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-input
            v-model="searchForm.version"
            placeholder="集群版本..."
            allow-clear
            @clear="handleSearch"
            @keyup.enter="handleSearch"
            style="width: 150px"
          >
            <template #prefix>
              <icon-info-circle />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button @click="handleReset">
              <template #icon><icon-undo /></template>
              重置
            </a-button>
            <a-button type="primary" @click="handleSearch">
              <template #icon><icon-search /></template>
              搜索
            </a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 集群列表 -->
    <a-card class="table-card">
      <!-- 批量操作栏 -->
      <div v-if="selectedClusters.length > 0" class="batch-actions-bar">
        <div class="batch-actions-left">
          <a-checkbox
            v-model="selectAllCurrentPage"
            :indeterminate="isIndeterminate"
            @change="handleSelectAllCurrentPage"
          >
            <span class="selected-count">已选择 {{ selectedClusters.length }} 个集群</span>
          </a-checkbox>
        </div>
        <a-space>
          <a-button v-permission="'k8s-clusters:batch-sync'" type="primary" @click="handleBatchSync">
            批量同步
          </a-button>
          <a-button v-permission="'k8s-clusters:batch-delete'" status="danger" @click="handleBatchDelete">
            批量删除
          </a-button>
          <a-button @click="clearSelection">取消选择</a-button>
        </a-space>
      </div>

      <a-table
        ref="clusterTableRef"
        :data="paginatedClusterList"
        :loading="loading"
        :bordered="false"
        @selection-change="handleSelectionChange"
        :columns="tableColumns"
        :row-selection="{ type: 'checkbox', showCheckedAll: true }"
        :pagination="false"
      >
        <template #name="{ record }">
          <a-button type="text" @click="handleViewDetail(record)" class="cluster-name-link">
            {{ record.name }}
          </a-button>
        </template>
          <template #alias="{ record }">
          {{ record.alias || '-' }}
        </template>
          <template #status="{ record }">
          <a-tag :color="getStatusType(record.status)" size="small">
            <span class="status-dot" :class="'status-dot-' + record.status"></span>
            {{ getStatusText(record.status) }}
          </a-tag>
        </template>
          <template #provider="{ record }">
          {{ getProviderText(record.provider) }}
        </template>
          <template #region="{ record }">
          {{ record.region || '-' }}
        </template>
          <template #actions="{ record }">
          <div class="action-buttons">
            <a-tooltip content="凭证" placement="top">
              <a-button v-if="isAdmin" type="text" class="action-btn" @click="handleViewConfig(record)">
                <icon-safe />
              </a-button>
            </a-tooltip>
            <a-tooltip content="授权" placement="top">
              <a-button type="text" class="action-btn action-auth" @click="handleAuthorize(record)">
                <icon-lock />
              </a-button>
            </a-tooltip>
            <a-tooltip content="同步" placement="top">
              <a-button v-permission="'k8s-clusters:sync'" type="text" class="action-btn action-sync" @click="handleSync(record)">
                <icon-refresh />
              </a-button>
            </a-tooltip>
            <a-tooltip content="编辑" placement="top">
              <a-button v-if="isAdmin" v-permission="'k8s-clusters:update'" type="text" class="action-btn action-edit" @click="handleEdit(record)">
                <icon-edit />
              </a-button>
            </a-tooltip>
            <a-tooltip content="删除" placement="top">
              <a-button v-if="isAdmin" v-permission="'k8s-clusters:delete'" type="text" class="action-btn action-delete" @click="handleDelete(record)">
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
          :total="filteredClusterList.length"
          show-total show-page-size show-jumper
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </a-card>

    <!-- 注册/编辑集群对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="isEdit ? '编辑集群' : '注册集群'"
      width="70%"
      class="cluster-edit-dialog"
      @close="handleDialogClose"
    >
      <a-form :model="clusterForm" :rules="rules" ref="formRef" label-width="100px">
        <!-- 基本信息 -->
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-row :gutter="20">
            <a-col :span="12">
              <a-form-item label="集群名称" field="name">
                <a-input v-model="clusterForm.name" placeholder="请输入集群名称"  />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="集群别名">
                <a-input v-model="clusterForm.alias" placeholder="可选" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <!-- 认证配置 -->
        <div class="form-section">
          <div class="section-title">认证配置</div>
          <a-form-item label="认证方式">
            <a-radio-group v-model="authType" @change="handleAuthTypeChange">
              <a-radio value="config">KubeConfig 文件</a-radio>
              <a-radio value="token">Service Account Token</a-radio>
            </a-radio-group>
          </a-form-item>

          <!-- KubeConfig 方式 -->
          <template v-if="authType === 'config'">
            <a-alert
              v-if="isEdit"
              title="配置信息"
              type="info"
              :closable="false"
              style="margin-bottom: 12px"
            >
              <template #default>
                <div style="font-size: 12px;">
                  <p style="margin: 0 0 8px 0;">
                    <strong>当前集群配置信息：</strong>
                  </p>
                  <ul style="margin: 0; padding-left: 20px;">
                    <li>API Endpoint: {{ clusterForm.apiEndpoint || '未配置' }}</li>
                    <li>服务商: {{ clusterForm.provider ? getProviderText(clusterForm.provider) : '未配置' }}</li>
                    <li>区域: {{ clusterForm.region || '未配置' }}</li>
                  </ul>
                  <p style="margin: 8px 0 0 0; color: #409eff;">
                    💡 下方显示的是当前的 KubeConfig 配置，您可以直接编辑或上传新文件替换
                  </p>
                </div>
              </template>
            </a-alert>
            <a-form-item label="配置内容" field="kubeConfig">
              <div style="margin-bottom: 8px;">
                <a-button size="small" @click="handleUploadKubeConfig">
                  <icon-upload />
                  上传 KubeConfig 文件
                </a-button>
                <input
                  ref="fileInputRef"
                  type="file"
                  style="display: none"
                  @change="handleFileChange"
                />
              </div>
              <div class="code-editor-wrapper">
                <div class="line-numbers">
                  <div v-for="n in lineCount" :key="n" class="line-number">{{ n }}</div>
                </div>
                <textarea
                  v-model="clusterForm.kubeConfig"
                  class="code-textarea"
                  :placeholder="isEdit ? '' : '请粘贴 KubeConfig 文件内容或点击上方按钮上传'"
                  spellcheck="false"
                  @input="updateLineCount"

                ></textarea>
              </div>
              <div class="code-tip" v-if="!isEdit">
                <icon-info-circle />
                <span>如何获取 KubeConfig？通常位于 ~/.kube/config 文件中</span>
              </div>
            </a-form-item>
          </template>

          <!-- Token 方式 -->
          <template v-if="authType === 'token'">
            <a-form-item label="API 地址" field="apiEndpoint">
              <a-input
                v-model="clusterForm.apiEndpoint"
                placeholder="https://k8s-api.example.com:6443"

              >
                <template #prepend>
                  <icon-link />
                </template>
              </a-input>
            </a-form-item>
            <a-form-item label="TLS 验证">
              <a-switch v-model="skipTLSVerify" active-text="跳过验证" inactive-text="验证证书" />
              <span style="margin-left: 12px; font-size: 12px; color: #909399;">
                ⚠️ 跳过 TLS 验证仅适用于测试环境，生产环境请提供 CA 证书
              </span>
            </a-form-item>
            <a-form-item label="Token" field="token">
              <div class="code-editor-wrapper">
                <div class="line-numbers">
                  <div v-for="n in tokenLineCount" :key="n" class="line-number">{{ n }}</div>
                </div>
                <textarea
                  v-model="clusterForm.token"
                  class="code-textarea"
                  placeholder="请输入 Service Account Token"
                  spellcheck="false"
                  @input="updateTokenLineCount"

                ></textarea>
              </div>
              <div class="code-tip">
                <icon-info-circle />
                <span>如何获取 Token？使用 kubectl create token 命令创建</span>
              </div>
            </a-form-item>
          </template>
        </div>

        <!-- 集群信息 -->
        <div class="form-section">
          <div class="section-title">集群信息</div>
          <a-row :gutter="20">
            <a-col :span="12">
              <a-form-item label="服务商">
                <a-select v-model="clusterForm.provider" placeholder="请选择" style="width: 100%">
                  <a-option label="自建集群" value="native" />
                  <a-option label="阿里云 ACK" value="aliyun" />
                  <a-option label="腾讯云 TKE" value="tencent" />
                  <a-option label="AWS EKS" value="aws" />
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="区域">
                <a-input v-model="clusterForm.region" placeholder="例如: cn-beijing" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item label="备注">
            <a-input
              v-model="clusterForm.description"
              type="textarea"
              :rows="2"
              placeholder="请输入集群备注（可选）"
            />
          </a-form-item>
        </div>
      </a-form>

      <template #footer>
        <div class="dialog-footer">
          <a-button @click="dialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSubmit" :loading="submitLoading">
            {{ isEdit ? '保存' : '注册集群' }}
          </a-button>
        </div>
      </template>
    </a-modal>

    <!-- 查看集群凭证对话框 -->
    <a-modal
      v-model:visible="configDialogVisible"
      title="集群凭证"
      width="700px"
    >
      <div style="margin-bottom: 16px;">
        <a-descriptions :column="2" :bordered="true">
          <a-descriptions-item label="集群名称">{{ currentCluster?.name }}</a-descriptions-item>
          <a-descriptions-item label="别名">{{ currentCluster?.alias || '-' }}</a-descriptions-item>
          <a-descriptions-item label="API Endpoint">{{ currentCluster?.apiEndpoint }}</a-descriptions-item>
          <a-descriptions-item label="版本">{{ currentCluster?.version }}</a-descriptions-item>
        </a-descriptions>
      </div>

      <div style="margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center;">
        <span style="font-weight: 500;">KubeConfig 配置</span>
        <div>
          <a-button size="small" @click="handleCopyConfig">
            <icon-copy />
            复制
          </a-button>
          <a-button size="small" @click="handleDownloadConfig">
            <icon-download />
            下载
          </a-button>
        </div>
      </div>

      <div class="code-editor-wrapper">
        <div class="line-numbers">
          <div v-for="n in configLineCount" :key="n" class="line-number">{{ n }}</div>
        </div>
        <textarea
          v-model="currentConfig"
          class="code-textarea"
          readonly
          spellcheck="false"
        ></textarea>
      </div>

      <div class="code-tip">
        <icon-exclamation-circle />
        <span>请妥善保管集群凭证，不要泄露给他人</span>
      </div>
    </a-modal>

    <!-- 授权对话框 -->
    <a-modal
      v-model:visible="authorizeDialogVisible"
      title="集群授权"
      width="900px"
    >
      <a-tabs v-model:active-key="activeAuthTab" type="border-card">
        <!-- 连接信息 -->
        <a-tab-pane title="连接信息" key="connection">
          <div class="connection-info">
            <div class="info-section">
              <div class="section-title">
                <icon-link />
                <span>集群连接信息</span>
              </div>
              <a-descriptions :column="2" :bordered="true" style="margin-top: 16px;">
                <a-descriptions-item label="集群名称">{{ currentCluster?.name }}</a-descriptions-item>
                <a-descriptions-item label="别名">{{ currentCluster?.alias || '-' }}</a-descriptions-item>
                <a-descriptions-item label="API Endpoint">{{ currentCluster?.apiEndpoint }}</a-descriptions-item>
                <a-descriptions-item label="版本">{{ currentCluster?.version }}</a-descriptions-item>
              </a-descriptions>
            </div>

            <div class="credential-section">
              <div class="section-header">
                <div class="section-title">
                  <icon-safe />
                  <span>凭据管理</span>
                </div>
                <div v-if="!generatedKubeConfig">
                  <a-button
                    v-permission="'k8s-clusters:apply-credential'"
                    type="primary"
                   
                    @click="handleApplyCredential"
                    :loading="credentialLoading"
                  >
                    凭据申请
                  </a-button>
                </div>
                <div v-else>
                  <a-button
                    v-permission="'k8s-clusters:revoke-credential'"
                    type="danger"
                   
                    @click="handleRevokeCredential"
                    :loading="revokeLoading"
                  >
                    吊销凭据
                  </a-button>
                </div>
              </div>

              <div v-if="generatedKubeConfig" class="kubeconfig-display">
                <div class="kubeconfig-header">
                  <span style="font-weight: 500;">生成的 KubeConfig 凭据</span>
                  <a-button
                    type="primary"
                   
                    @click="handleCopyKubeConfig"
                    size="small"
                  >
                    复制
                  </a-button>
                </div>
                <a-input
                  v-model="generatedKubeConfig"
                  type="textarea"
                  :rows="10"
                  readonly
                  class="kubeconfig-textarea"
                />
                <div class="code-tip">
                  <icon-exclamation-circle />
                  <span>此凭据文件包含您的集群访问权限，请妥善保管，不要泄露给他人</span>
                </div>
              </div>

              <div v-else class="no-credential-tip">
                <a-empty description="暂无凭据，请点击上方按钮申请">
                  <template #image>
                    <icon-safe />
                  </template>
                </a-empty>
              </div>
            </div>
          </div>
        </a-tab-pane>

        <!-- 用户 -->
        <a-tab-pane v-if="isAdmin" key="users">
          <template #label>
            <span class="tab-label">
              <icon-user />
              用户
            </span>
          </template>
          <div class="tab-content">
            <ClusterAuthDialog
              v-if="currentCluster"
              :cluster="currentCluster"
              :model-value="true"
              :credential-users="clusterCredentialUsers"
              @refresh="loadClusterCredentials"
            />
            <a-empty v-else description="请先选择集群" />
          </div>
        </a-tab-pane>

        <!-- 角色 -->
        <a-tab-pane v-if="isAdmin" key="roles">
          <template #label>
            <span class="tab-label">
              <icon-safe />
              角色
            </span>
          </template>
          <div class="tab-content">
            <UserRoleBinding
              v-if="currentCluster"
              :cluster="currentCluster"
            />
            <a-empty v-else description="请先选择集群" />
          </div>
        </a-tab-pane>
      </a-tabs>

      <template #footer>
        <div class="dialog-footer">
          <a-button @click="authorizeDialogVisible = false">关闭</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { title: '集群名称', dataIndex: 'name', slotName: 'name', width: 180 },
  { title: '别名', dataIndex: 'alias', slotName: 'alias', width: 120 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 100 },
  { title: '版本', dataIndex: 'version', width: 120 },
  { title: '节点数', dataIndex: 'nodeCount', width: 100 },
  { title: '服务商', dataIndex: 'provider', slotName: 'provider', width: 120 },
  { title: '区域', dataIndex: 'region', slotName: 'region', width: 120 },
  { title: '备注', dataIndex: 'description', width: 150, ellipsis: true, tooltip: true },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
  { title: '操作', slotName: 'actions', width: 220, fixed: 'right' }
]

import { ref, reactive, computed, onMounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { Message, Modal } from '@arco-design/web-vue'

import {
  getClusterList,
  createCluster,
  updateCluster,
  deleteCluster,
  testClusterConnection,
  getClusterDetail,
  getClusterConfig,
  generateKubeConfig,
  revokeKubeConfig,
  getClusterCredentialUsers,
  getExistingKubeConfig,
  syncClusterStatus,
  syncAllClustersStatus,
  createDefaultClusterRoles,
  createDefaultNamespaceRoles,
  type Cluster,
  type CredentialUser
} from '@/api/kubernetes'
import ClusterAuthDialog from './components/ClusterAuthDialog.vue'
import UserRoleBinding from './components/UserRoleBinding.vue'
import { useUserStore } from '@/stores/user'

// 用户权限
const userStore = useUserStore()
const isAdmin = computed(() => {
  if (!userStore.userInfo) {
    return false
  }

  // 确保 roles 是数组，如果不是则返回 false
  if (!Array.isArray(userStore.userInfo.roles)) {
    return false
  }

  // 检查是否有 admin 角色
  return userStore.userInfo.roles.some((role: any) => role.code === 'admin')
})

const loading = ref(false)
const dialogVisible = ref(false)
const configDialogVisible = ref(false)
const authorizeDialogVisible = ref(false)
const showRoleBindingDialog = ref(false)
const activeAuthTab = ref('connection')
const credentialLoading = ref(false)
const revokeLoading = ref(false)
const generatedKubeConfig = ref('')
const currentCredentialUsername = ref('')
const submitLoading = ref(false)
const formRef = ref()
const fileInputRef = ref<HTMLInputElement>()
const authType = ref('config')
const skipTLSVerify = ref(true)  // 默认跳过 TLS 验证，适用于自签名证书
const lineCount = ref(1)
const tokenLineCount = ref(1)
const isEdit = ref(false)
const editClusterId = ref<number>()
const kubeConfigEditable = ref(false)
const currentCluster = ref<Cluster>()
const currentConfig = ref('')
const configLineCount = ref(1)
const router = useRouter()
const syncing = ref(false) // 同步状态

const clusterList = ref<Cluster[]>([])
const clusterCredentialUsers = ref<CredentialUser[]>([])
const selectedClusters = ref<Cluster[]>([]) // 选中的集群
const clusterTableRef = ref() // 表格引用
const selectAllCurrentPage = ref(false) // 全选当前页
const isIndeterminate = ref(false) // 半选状态

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)
const paginationStorageKey = ref('cluster_list_pagination')

// 搜索表单
const searchForm = reactive({
  keyword: '',
  status: undefined as number | undefined,
  version: ''
})

const clusterForm = reactive({
  name: '',
  alias: '',
  apiEndpoint: '',
  kubeConfig: '',
  token: '',
  provider: 'native',
  region: '',
  description: ''
})

const rules = {
  name: [{ required: true, content: '请输入集群名称', trigger: 'blur' }],
  kubeConfig: [
    {
      required: true,
      content: '请输入 KubeConfig',
      trigger: 'blur',
      validator: (rule: any, value: any, callback: any) => {
        // 新增模式必须填写，编辑模式可以留空
        if (!isEdit.value && authType.value === 'config' && !value) {
          callback(new Error('请输入 KubeConfig'))
        } else {
          callback()
        }
      }
    }
  ],
  apiEndpoint: [
    {
      required: true,
      content: '请输入 API Endpoint',
      trigger: 'blur',
      validator: (rule: any, value: any, callback: any) => {
        // 新增模式必须填写，编辑模式可以留空
        if (!isEdit.value && authType.value === 'token' && !value) {
          callback(new Error('请输入 API Endpoint'))
        } else {
          callback()
        }
      }
    }
  ],
  token: [
    {
      required: true,
      content: '请输入 Token',
      trigger: 'blur',
      validator: (rule: any, value: any, callback: any) => {
        // 新增模式必须填写，编辑模式可以留空
        if (!isEdit.value && authType.value === 'token' && !value) {
          callback(new Error('请输入 Token'))
        } else {
          callback()
        }
      }
    }
  ]
}

// 过滤后的集群列表
const filteredClusterList = computed(() => {
  let result = clusterList.value

  // 按关键词搜索（集群名称或别名）
  if (searchForm.keyword) {
    const keyword = searchForm.keyword.toLowerCase()
    result = result.filter(cluster =>
      cluster.name.toLowerCase().includes(keyword) ||
      (cluster.alias || '').toLowerCase().includes(keyword)
    )
  }

  // 按状态筛选
  if (searchForm.status !== undefined) {
    result = result.filter(cluster => cluster.status === searchForm.status)
  }

  // 按版本筛选
  if (searchForm.version) {
    result = result.filter(cluster =>
      cluster.version && cluster.version.toLowerCase().includes(searchForm.version.toLowerCase())
    )
  }

  return result
})

// 分页后的集群列表
const paginatedClusterList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredClusterList.value.slice(start, end)
})

// 总节点数
const totalNodeCount = computed(() => {
  return clusterList.value.reduce((sum, cluster) => sum + (cluster.nodeCount || 0), 0)
})

// 加载集群列表
const loadClusters = async () => {
  loading.value = true
  try {
    const data = await getClusterList()
    // 强制刷新：使用新数组替换旧数组
    clusterList.value = [...(data || [])]
    // 恢复分页状态
    restorePaginationState()
  } catch (error) {
    Message.error('获取集群列表失败')
  } finally {
    loading.value = false
  }
}

// 保存分页状态到 localStorage
const savePaginationState = () => {
  try {
    localStorage.setItem(paginationStorageKey.value, JSON.stringify({
      currentPage: currentPage.value,
      pageSize: pageSize.value
    }))
  } catch (error) {
    // 保存分页状态失败
  }
}

// 从 localStorage 恢复分页状态
const restorePaginationState = () => {
  try {
    const saved = localStorage.getItem(paginationStorageKey.value)
    if (saved) {
      const state = JSON.parse(saved)
      currentPage.value = state.currentPage || 1
      pageSize.value = state.pageSize || 10
    }
  } catch (error) {
    currentPage.value = 1
    pageSize.value = 10
  }
}

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
  savePaginationState()
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  // 当每页数量变化时，可能需要调整当前页码
  const maxPage = Math.ceil(filteredClusterList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
  savePaginationState()
}

// 搜索
const handleSearch = () => {
  // 搜索时重置到第一页
  currentPage.value = 1
  savePaginationState()
  // filteredClusterList 会自动更新
}

// 重置搜索
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.status = undefined
  searchForm.version = ''
}

// 注册集群
const handleRegister = () => {
  isEdit.value = false
  kubeConfigEditable.value = true
  dialogVisible.value = true
}

// 查看集群详情
const handleViewDetail = (row: Cluster) => {
  router.push(`/kubernetes/clusters/${row.id}`)
}

// 编辑集群
const handleEdit = async (row: Cluster) => {
  isEdit.value = true
  editClusterId.value = row.id
  kubeConfigEditable.value = true

  try {
    // 获取现有的 kubeconfig 内容
    const config = await getClusterConfig(row.id)

    // 填充表单数据
    Object.assign(clusterForm, {
      name: row.name,
      alias: row.alias,
      apiEndpoint: row.apiEndpoint,
      kubeConfig: config, // 显示现有的 KubeConfig
      token: "",
      provider: row.provider,
      region: row.region,
      description: row.description
    })

    // 更新行号
    updateLineCount()
  } catch (error: any) {
    Message.error(error.response?.data?.message || '获取集群配置失败')
    // 即使失败也打开对话框，但不显示配置
    Object.assign(clusterForm, {
      name: row.name,
      alias: row.alias,
      apiEndpoint: row.apiEndpoint,
      kubeConfig: "",
      token: "",
      provider: row.provider,
      region: row.region,
      description: row.description
    })
  }

  dialogVisible.value = true
}

// 同步集群信息
const handleSync = async (row: Cluster) => {
  const loadingMsg = Message.loading({
    content: '正在同步集群信息...',
    duration: 0,
    type: 'info'
  })

  try {
    // 调用新的同步状态 API
    await syncClusterStatus(row.id)

    // 等待一小段时间让同步完成
    await new Promise(resolve => setTimeout(resolve, 2000))

    Message.clear()

    // 重新加载列表
    await loadClusters()
    Message.success('同步成功')
  } catch (error: any) {
    Message.clear()
    Message.error(error.response?.data?.message || '同步失败')
  }
}

// 同步所有集群状态
const handleSyncAll = async () => {
  syncing.value = true
  try {
    await syncAllClustersStatus()

    // 等待一小段时间让同步完成
    await new Promise(resolve => setTimeout(resolve, 3000))

    // 重新加载列表
    await loadClusters()
    Message.success('批量同步任务已启动，请稍后刷新查看')
  } catch (error: any) {
    Message.error(error.response?.data?.message || '同步失败')
  } finally {
    syncing.value = false
  }
}

// 处理表格选择变化
const handleSelectionChange = (selection: Cluster[]) => {
  selectedClusters.value = selection
  updateSelectAllStatus()
}

// 更新全选状态
const updateSelectAllStatus = () => {
  const currentPageCount = paginatedClusterList.value.length
  const selectedCount = selectedClusters.value.length

  if (selectedCount === 0) {
    selectAllCurrentPage.value = false
    isIndeterminate.value = false
  } else if (selectedCount === currentPageCount) {
    selectAllCurrentPage.value = true
    isIndeterminate.value = false
  } else {
    selectAllCurrentPage.value = false
    isIndeterminate.value = true
  }
}

// 处理当前页全选
const handleSelectAllCurrentPage = (checked: boolean) => {
  if (checked) {
    // 添加当前页所有集群到已选择列表（去重）
    const currentPageIds = new Set(selectedClusters.value.map(c => c.id))
    paginatedClusterList.value.forEach(cluster => {
      if (!currentPageIds.has(cluster.id)) {
        selectedClusters.value.push(cluster)
      }
    })
  } else {
    // 移除当前页的集群
    const currentPageIds = new Set(paginatedClusterList.value.map(c => c.id))
    selectedClusters.value = selectedClusters.value.filter(c => !currentPageIds.has(c.id))
  }
  updateSelectAllStatus()
  // 同步表格选择状态
  syncTableSelection()
}

// 同步表格选择状态
const syncTableSelection = () => {
  if (clusterTableRef.value) {
    const selectedIds = new Set(selectedClusters.value.map(c => c.id))
    paginatedClusterList.value.forEach(row => {
      const isSelected = selectedIds.has(row.id)
      clusterTableRef.value.toggleRowSelection(row, isSelected)
    })
  }
}

// 清除选择
const clearSelection = () => {
  selectedClusters.value = []
  selectAllCurrentPage.value = false
  isIndeterminate.value = false
  if (clusterTableRef.value) {
    clusterTableRef.value.clearSelection()
  }
}

// 批量同步集群
const handleBatchSync = async () => {
  try {
    await confirmModal(
      `确定要同步选中的 ${selectedClusters.value.length} 个集群吗？`,
      '批量同步确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info'
      }
    )

    const loadingMsg = Message.loading({
      content: `正在同步 ${selectedClusters.value.length} 个集群，请稍候...`,
      duration: 0,
      type: 'info'
    })

    // 并发同步所有选中的集群
    const syncPromises = selectedClusters.value.map(cluster => syncClusterStatus(cluster.id))
    await Promise.all(syncPromises)

    Message.clear()
    clearSelection()
    await loadClusters()
    Message.success('同步成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '同步失败')
    }
  }
}

// 批量删除集群
const handleBatchDelete = async () => {
  if (selectedClusters.value.length === 0) {
    Message.warning('请选择要删除的集群')
    return
  }

  try {
    await confirmModal(
      `确定要删除选中的 ${selectedClusters.value.length} 个集群吗？此操作不可恢复！`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    // 显示正在删除的提示
    const loadingMsg = Message.loading({
      content: `正在删除 ${selectedClusters.value.length} 个集群，请稍候...`,
      duration: 0,
      type: 'info'
    })

    // 并发删除所有选中的集群
    const deletePromises = selectedClusters.value.map(cluster => deleteCluster(cluster.id))
    await Promise.all(deletePromises)

    Message.clear()
    clearSelection()
    await loadClusters()
    Message.success('删除成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '删除失败')
    }
  }
}

// 认证方式切换
const handleAuthTypeChange = () => {
  formRef.value?.clearValidate()
  setTimeout(() => {
    formRef.value?.validate()
  }, 50)
}

// 更新行号
const updateLineCount = () => {
  const lines = clusterForm.kubeConfig.split('\n').length
  lineCount.value = lines || 1
}

// 更新 Token 行号
const updateTokenLineCount = () => {
  const lines = clusterForm.token.split('\n').length
  tokenLineCount.value = lines || 1
}

// 上传 KubeConfig 文件
const handleUploadKubeConfig = () => {
  fileInputRef.value?.click()
}

// 处理文件选择
const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]

  if (!file) return

  const reader = new FileReader()
  reader.onload = (e) => {
    const content = e.target?.result as string
    clusterForm.kubeConfig = content
    updateLineCount()
    Message.success('文件读取成功')
  }
  reader.onerror = () => {
    Message.error('文件读取失败')
  }
  reader.readAsText(file)

  // 清空 input value，允许重复上传同一文件
  target.value = ''
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitLoading.value = true
      try {
        let kubeConfig = clusterForm.kubeConfig
        if (authType.value === 'token') {
          kubeConfig = buildKubeConfigFromToken(
            clusterForm.apiEndpoint,
            clusterForm.token
          )
        }

        if (isEdit.value && editClusterId.value) {
          // 编辑模式 - 可以更新名称、备注、服务商等信息
          // 如果需要更新 KubeConfig，在编辑模式下重新输入即可
          const updateData: any = {
            name: clusterForm.name,
            alias: clusterForm.alias,
            region: clusterForm.region,
            provider: clusterForm.provider,
            description: clusterForm.description
          }

          // 如果重新输入了 KubeConfig，则更新它
          if (clusterForm.kubeConfig && authType.value === 'config') {
            updateData.kubeConfig = clusterForm.kubeConfig
          } else if (clusterForm.token && authType.value === 'token') {
            updateData.kubeConfig = buildKubeConfigFromToken(
              clusterForm.apiEndpoint,
              clusterForm.token
            )
            updateData.apiEndpoint = clusterForm.apiEndpoint
          }

          await updateCluster(editClusterId.value, updateData)
          Message.success('更新成功')
        } else {
          // 新增模式
          const requestData: any = {
            name: clusterForm.name,
            kubeConfig: kubeConfig
          }

          if (authType.value === 'token') {
            requestData.apiEndpoint = clusterForm.apiEndpoint
          }

          if (clusterForm.alias) requestData.alias = clusterForm.alias
          if (clusterForm.provider) requestData.provider = clusterForm.provider
          if (clusterForm.region) requestData.region = clusterForm.region
          if (clusterForm.description) requestData.description = clusterForm.description

          const newCluster = await createCluster(requestData)
          Message.success('集群注册成功')

          // 注册成功后立即创建默认集群角色和常用命名空间角色
          const roleLoadingMsg = Message.loading({
            content: '正在初始化默认角色，请稍候...',
            duration: 0,
            showClose: false
          })

          try {
            // 并行创建集群角色和命名空间角色（ClusterRole）
            const [clusterRolesResult, namespaceRolesResult] = await Promise.all([
              createDefaultClusterRoles(newCluster.id),
              createDefaultNamespaceRoles(newCluster.id).catch(() => {
                // 命名空间角色创建失败不影响整体流程
                return { created: [] }
              })
            ])

            Message.clear()

            const clusterCount = clusterRolesResult?.created?.length || 0
            const namespaceCount = namespaceRolesResult?.created?.length || 0
            Message.success(`默认角色初始化完成（集群角色：${clusterCount}个，命名空间角色：${namespaceCount}个）`)
          } catch (roleError) {
            Message.clear()
            Message.warning('集群注册成功，但创建默认角色失败，请稍后在角色管理页面手动创建')
          }
        }

        dialogVisible.value = false
        loadClusters()
      } catch (error: any) {
        Message.error(error.response?.data?.message || '操作失败')
      } finally {
        submitLoading.value = false
      }
    }
  })
}

// 从 Token 构建 KubeConfig
const buildKubeConfigFromToken = (apiEndpoint: string, token: string) => {
  // 根据 skipTLSVerify 决定是否跳过 TLS 验证
  const tlsConfig = skipTLSVerify.value
    ? '    insecure-skip-tls-verify: true'
    : '    certificate-authority-data: ""'

  return `apiVersion: v1
kind: Config
clusters:
- cluster:
${tlsConfig}
    server: ${apiEndpoint}
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: default-user
  name: default-context
current-context: default-context
users:
- name: default-user
  user:
    token: ${token}
`
}

// 测试连接
const handleTestConnection = async (row: Cluster) => {
  const loadingMsg = Message.loading({
    content: '正在测试连接...',
    duration: 0,
    type: 'info'
  })

  try {
    const result = await testClusterConnection(row.id)
    Message.clear()

    // 重新加载列表以更新节点数
    await loadClusters()

    Message.success(`连接成功！版本: ${result.version}`)
  } catch (error: any) {
    Message.clear()
    Message.error(error.response?.data?.message || '连接失败')
  }
}

// 删除集群
const handleDelete = async (row: Cluster) => {
  try {
    await confirmModal(
      `<div style="line-height: 1.8;">
        <p style="margin-bottom: 12px; font-weight: 600; color: #f56c6c;">
          <i class="el-icon-warning" style="margin-right: 4px;"></i>
          确定要删除集群 <strong>"${row.name}"</strong> 吗？
        </p>
        <div style="padding: 12px; background: #fef0f0; border-left: 3px solid #f56c6c; margin-bottom: 8px; border-radius: 4px;">
          <p style="margin: 0 0 8px 0; color: #606266; font-size: 14px;"><strong>删除集群将同时清理以下资源：</strong></p>
          <ul style="margin: 0; padding-left: 20px; color: #909399; font-size: 13px;">
            <li>所有用户的集群访问凭据（ServiceAccount）</li>
            <li>所有用户的角色绑定（ClusterRoleBinding 和 RoleBinding）</li>
            <li>所有默认集群角色（ClusterRole）</li>
            <li>所有命名空间中的 OpsHub 管理的 RoleBinding</li>
            <li>数据库中的所有集群相关数据</li>
          </ul>
        </div>
        <p style="margin: 8px 0 0 0; color: #e6a23c; font-size: 13px;">
          <i class="el-icon-warning" style="margin-right: 4px;"></i>
          此操作不可恢复，请谨慎操作！
        </p>
      </div>`,
      '删除集群',
      {
        type: 'warning',
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        dangerouslyUseHTMLString: true,
        customClass: 'delete-cluster-confirm'
      }
    )

    // 显示正在删除的提示
    const loadingMsg = Message.loading({
      content: '正在删除集群，请稍候...',
      duration: 0,
      type: 'info'
    })

    await deleteCluster(row.id)
    Message.clear()
    Message.success('集群已删除，所有相关资源已清理')
    loadClusters()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '删除失败')
    }
  }
}

// 查看集群凭证
const handleViewConfig = async (row: Cluster) => {
  try {
    const cluster = await getClusterDetail(row.id)
    currentCluster.value = cluster

    // 获取解密后的 KubeConfig
    const config = await getClusterConfig(row.id)
    currentConfig.value = config

    configDialogVisible.value = true
  } catch (error: any) {
    Message.error(error.response?.data?.message || '获取集群凭证失败')
  }
}

// 监听 config 内容变化，更新行号
watch(currentConfig, () => {
  const lines = currentConfig.value.split('\n').length
  configLineCount.value = lines || 1
})

// 复制配置
const handleCopyConfig = async () => {
  try {
    await navigator.clipboard.writeText(currentConfig.value)
    Message.success('复制成功')
  } catch (error) {
    Message.error('复制失败，请手动复制')
  }
}

// 下载配置
const handleDownloadConfig = () => {
  const blob = new Blob([currentConfig.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  const filename = `kubeconfig-${currentCluster.value?.name || 'cluster'}.conf`
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
  Message.success('下载成功')
}

// 加载集群凭据用户列表
const loadClusterCredentials = async () => {
  if (!currentCluster.value) return

  try {
    const users = await getClusterCredentialUsers(currentCluster.value.id)
    clusterCredentialUsers.value = users
    // 不再自动刷新当前用户凭据，避免误清空
  } catch (error: any) {
    Message.error(error.response?.data?.message || '加载凭据用户失败')
  }
}

// 刷新当前用户的凭据
const refreshCurrentUserCredential = async () => {
  if (!currentCluster.value) return

  try {
    const result = await getExistingKubeConfig(currentCluster.value.id)
    generatedKubeConfig.value = result.kubeconfig
    currentCredentialUsername.value = result.username

    // 保存到localStorage
    const username = getCurrentUsername()
    const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
    const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
    localStorage.setItem(storageKey, result.kubeconfig)
    localStorage.setItem(usernameKey, result.username)
  } catch (error: any) {
    // 只有明确的 404 错误（用户尚未申请凭据）才清空显示
    // 其他错误（如网络错误、后端查找失败）不清空，保持现有状态
    if (error.response?.status === 404) {
      generatedKubeConfig.value = ''
      currentCredentialUsername.value = ''
      // 同时清除 localStorage
      const username = getCurrentUsername()
      const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
      const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
      localStorage.removeItem(storageKey)
      localStorage.removeItem(usernameKey)
    } else {
      // 其他错误，不清空凭据
    }
  }
}

// 打开授权对话框
const handleAuthorize = async (row: Cluster) => {
  try {
    const cluster = await getClusterDetail(row.id)
    currentCluster.value = cluster

    authorizeDialogVisible.value = true
    activeAuthTab.value = 'connection'

    // 先尝试从后端API获取用户现有的kubeconfig
    try {
      const result = await getExistingKubeConfig(cluster.id)
      generatedKubeConfig.value = result.kubeconfig
      currentCredentialUsername.value = result.username

      // 保存到localStorage
      const username = getCurrentUsername()
      const storageKey = `kubeconfig_${cluster.id}_${username}`
      const usernameKey = `kubeconfig_username_${cluster.id}_${username}`
      localStorage.setItem(storageKey, result.kubeconfig)
      localStorage.setItem(usernameKey, result.username)
    } catch (error: any) {
      // 如果是404错误（用户尚未申请凭据），清空显示
      if (error.response?.status === 404) {
        generatedKubeConfig.value = ''
        currentCredentialUsername.value = ''
      } else {
        // 其他错误，也清空显示
        generatedKubeConfig.value = ''
        currentCredentialUsername.value = ''
      }
    }

    // 加载凭据用户列表（从后端API获取）
    await loadClusterCredentials()
  } catch (error: any) {
    Message.error(error.response?.data?.message || '获取集群信息失败')
  }
}

// 申请凭据
const handleApplyCredential = async () => {
  if (!currentCluster.value) return

  try {
    credentialLoading.value = true

    // 获取当前用户名
    const username = getCurrentUsername()

    // 调用后端API生成kubeconfig
    const result = await generateKubeConfig(currentCluster.value.id, username)
    generatedKubeConfig.value = result.kubeconfig
    currentCredentialUsername.value = result.username

    // 保存到 localStorage
    const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
    const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
    localStorage.setItem(storageKey, result.kubeconfig)
    localStorage.setItem(usernameKey, result.username)

    Message.success('凭据申请成功')
  } catch (error: any) {
    Message.error(error.response?.data?.message || '凭据申请失败')
  } finally {
    credentialLoading.value = false
  }
}

// 吊销凭据
const handleRevokeCredential = async () => {
  if (!currentCluster.value || !currentCredentialUsername.value) return

  try {
    await confirmModal('确定要吊销该凭据吗？吊销后将无法使用该 KubeConfig 访问集群。', '提示', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    revokeLoading.value = true

    // 调用后端API撤销kubeconfig
    await revokeKubeConfig(currentCluster.value.id, currentCredentialUsername.value)

    // 清空凭据
    generatedKubeConfig.value = ''
    currentCredentialUsername.value = ''

    // 清除 localStorage 中的凭据
    const username = getCurrentUsername()
    const storageKey = `kubeconfig_${currentCluster.value.id}_${username}`
    const usernameKey = `kubeconfig_username_${currentCluster.value.id}_${username}`
    localStorage.removeItem(storageKey)
    localStorage.removeItem(usernameKey)

    Message.success('凭据吊销成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '凭据吊销失败')
    }
  } finally {
    revokeLoading.value = false
  }
}

// 获取当前用户名
const getCurrentUsername = () => {
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr)
      return user.username || 'opshub-user'
    } catch {
      return 'opshub-user'
    }
  }
  return 'opshub-user'
}

// 复制生成的kubeconfig
const handleCopyKubeConfig = async () => {
  try {
    await navigator.clipboard.writeText(generatedKubeConfig.value)
    Message.success('复制成功')
  } catch (error) {
    Message.error('复制失败，请手动复制')
  }
}

// 关闭对话框
const handleDialogClose = () => {
  formRef.value?.resetFields()
  Object.assign(clusterForm, {
    name: '',
    alias: '',
    apiEndpoint: '',
    kubeConfig: '',
    token: '',
    provider: 'native',
    region: '',
    description: ''
  })
  authType.value = 'config'
  skipTLSVerify.value = true  // 重置 TLS 验证选项
  isEdit.value = false
  editClusterId.value = undefined
  kubeConfigEditable.value = true
}

// 获取状态类型
const getStatusType = (status: number) => {
  const statusMap: Record<number, string> = {
    1: 'green',
    2: 'red',
    3: 'orangered'
  }
  return statusMap[status] || 'gray'
}

// 获取状态文本
const getStatusText = (status: number) => {
  const statusMap: Record<number, string> = {
    1: '正常',
    2: '连接失败',
    3: '不可用'
  }
  return statusMap[status] || '未知'
}

// 获取服务商文本
const getProviderText = (provider: string) => {
  const providerMap: Record<string, string> = {
    native: '自建集群',
    aliyun: '阿里云 ACK',
    tencent: '腾讯云 TKE',
    aws: 'AWS EKS'
  }
  return providerMap[provider] || provider || '未配置'
}

onMounted(async () => {
  // 确保用户信息已加载
  if (!userStore.userInfo) {
    try {
      await userStore.getProfile()
    } catch (error) {
      // 获取用户信息失败
    }
  }

  loadClusters()
})

// 监听标签页切换，当切换到用户标签时加载凭据用户列表，切换到连接信息标签时刷新当前用户凭据
watch(activeAuthTab, async (newTab) => {
  if (!currentCluster.value) return

  if (newTab === 'users') {
    // 切换到用户标签，加载凭据用户列表
    await loadClusterCredentials()
  } else if (newTab === 'connection') {
    // 切换到连接信息标签，刷新当前用户的凭据
    await refreshCurrentUserCredential()
  }
})

// 监听分页列表变化，保持选中状态
watch(paginatedClusterList, () => {
  if (selectedClusters.value.length > 0) {
    nextTick(() => {
      syncTableSelection()
    })
  }
}, { flush: 'post' })
</script>

<style scoped>
.clusters-container {
  padding: 0;
}

/* 统计卡片 */
.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.stat-content {
  display: flex;
  align-items: center;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1;
  margin-bottom: 6px;
}

.stat-label {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 页面头部 */
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
  align-items: center;
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
  gap: 10px;
  align-items: center;
}
/* 搜索卡片 */
.search-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.search-form :deep(.arco-form-item) {
  margin-bottom: 0;
}

/* 表格卡片 */
.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

/* 批量操作栏 */
.batch-actions-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  margin-bottom: 16px;
  background: var(--ops-primary-bg, #e8f0ff);
  border: 1px solid var(--ops-primary-lighter, #6694ff);
  border-radius: var(--ops-border-radius-sm, 4px);
}

.batch-actions-left {
  display: flex;
  align-items: center;
}

.selected-count {
  font-size: 14px;
  color: var(--ops-text-primary, #1d2129);
  font-weight: 500;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0 0;
  border-top: 1px solid var(--ops-border-color, #e5e6eb);
}

/* 集群名称链接 */
.cluster-name-link {
  color: var(--ops-primary, #165dff) !important;
  font-size: 14px;
  font-weight: 500;
}

.cluster-name-link:hover {
  color: var(--ops-primary-light, #306fff) !important;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  align-items: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  padding: 0;
  border-radius: var(--ops-border-radius-sm, 4px);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  color: var(--ops-text-secondary, #4e5969);
  font-size: 16px;
}

.action-btn:deep(.arco-btn-icon) {
  font-size: 16px;
}

.action-btn:hover {
  background-color: var(--ops-primary-bg, #e8f0ff);
  color: var(--ops-primary, #165dff);
}

.action-auth:hover {
  background-color: var(--ops-primary-bg, #e8f0ff);
  color: var(--ops-primary, #165dff);
}

.action-sync:hover {
  background-color: #e8ffea;
  color: var(--ops-success, #00b42a);
}

.action-edit:hover {
  background-color: #fff7e8;
  color: var(--ops-warning, #ff7d00);
}

.action-delete:hover {
  background-color: #ffece8;
  color: var(--ops-danger, #f53f3f);
}

/* 表单分区 */
.form-section {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px dashed var(--ops-border-color, #e5e6eb);
}

.form-section:last-of-type {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 16px;
  padding-left: 8px;
  border-left: 3px solid var(--ops-primary, #165dff);
}

/* 代码编辑器 */
.code-editor-wrapper {
  display: flex;
  width: 100%;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: var(--ops-border-radius-sm, 4px);
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
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #5c6370;
  min-height: 20.8px;
}

.code-textarea {
  flex: 1;
  min-height: 200px;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #abb2bf;
  background-color: #282c34;
  border: none;
  outline: none;
  resize: vertical;
}

.code-textarea::placeholder {
  color: #5c6370;
}

.code-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  padding: 8px 12px;
  background-color: #f7f8fa;
  border-radius: var(--ops-border-radius-sm, 4px);
  font-size: 12px;
  color: var(--ops-text-secondary, #4e5969);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 授权对话框 */
.connection-info {
  padding: 16px;
}

.info-section {
  margin-bottom: 24px;
}

.credential-section {
  margin-top: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.kubeconfig-display {
  margin-top: 16px;
}

.kubeconfig-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.kubeconfig-textarea :deep(.arco-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.5;
  background-color: #f7f8fa;
}

.no-credential-tip {
  padding: 40px 0;
  text-align: center;
}

.tab-content {
  padding: 16px;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

/* 对话框样式 */
:deep(.cluster-edit-dialog) {
  width: 70% !important;
  max-width: 90vw;
}

:deep(.cluster-edit-dialog .arco-modal__body) {
  max-height: 70vh;
  overflow-y: auto;
}

/* 状态指示点 */
.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-right: 4px;
  vertical-align: middle;
}

.status-dot-1 {
  background-color: #00b42a;
  box-shadow: 0 0 4px rgba(0, 180, 42, 0.4);
  animation: pulse-green 2s infinite;
}

.status-dot-2 {
  background-color: #f53f3f;
  box-shadow: 0 0 4px rgba(245, 63, 63, 0.4);
}

.status-dot-3 {
  background-color: #ff7d00;
  box-shadow: 0 0 4px rgba(255, 125, 0, 0.4);
}

@keyframes pulse-green {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@media (max-width: 1200px) {
  .stat-value { font-size: 24px; }
  .stat-icon { width: 48px; height: 48px; }
}
</style>
