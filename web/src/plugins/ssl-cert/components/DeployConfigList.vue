<template>
  <div class="deploy-config-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-upload /></div>
        <div>
          <h2 class="page-title">部署配置</h2>
          <p class="page-subtitle">配置证书自动部署到Nginx服务器或K8s集群</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          新增配置
        </a-button>
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
          v-model="searchForm.deploy_type"
          placeholder="部署类型"
          allow-clear
          class="search-input"
          @change="loadData"
        >
          <a-option value="nginx_ssh">Nginx SSH</a-option>
          <a-option value="k8s_secret">K8s Secret</a-option>
        </a-select>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <a-table
        :data="tableData"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        :pagination="{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showTotal: true,
          showPageSize: true,
          pageSizeOptions: [10, 20, 50]
        }"
        @page-change="(p: number) => { pagination.page = p; loadData() }"
        @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }"
      >
        <template #columns>
          <a-table-column title="配置名称" data-index="name" :width="120" ellipsis tooltip />

          <a-table-column title="关联证书" :min-width="200" ellipsis tooltip>
            <template #cell="{ record }">
              <span v-if="record.certificate">{{ record.certificate.name }} ({{ record.certificate.domain }})</span>
              <span v-else>-</span>
            </template>
          </a-table-column>

          <a-table-column title="部署类型" :width="130" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.deploy_type === 'nginx_ssh'" color="arcoblue">Nginx SSH</a-tag>
              <a-tag v-else color="green">K8s Secret</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="自动部署" :width="100" align="center">
            <template #cell="{ record }">
              <a-switch v-model="record.auto_deploy" @change="handleAutoDeployChange(record)" />
            </template>
          </a-table-column>

          <a-table-column title="状态" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.enabled" color="green">启用</a-tag>
              <a-tag v-else color="gray">禁用</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="上次部署" :min-width="180">
            <template #cell="{ record }">
              <div v-if="record.last_deploy_at">
                <span>{{ formatDateTime(record.last_deploy_at) }}</span>
                <a-tag v-if="record.last_deploy_ok" color="green" size="small" style="margin-left: 8px;">成功</a-tag>
                <a-tag v-else color="red" size="small" style="margin-left: 8px;">失败</a-tag>
              </div>
              <span v-else>-</span>
            </template>
          </a-table-column>

          <a-table-column title="操作" :width="200" fixed="right" align="center">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="立即部署" position="top">
                  <a-button type="text" class="action-btn action-deploy" @click="handleDeploy(record)" :loading="record.deploying">
                    <template #icon><icon-upload /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="测试配置" position="top">
                  <a-button type="text" class="action-btn action-test" @click="handleTest(record)" :loading="record.testing">
                    <template #icon><icon-link /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="编辑" position="top">
                  <a-button type="text" class="action-btn action-edit" @click="handleEdit(record)">
                    <template #icon><icon-edit /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="删除" position="top">
                  <a-button type="text" class="action-btn action-delete" @click="handleDelete(record)">
                    <template #icon><icon-delete /></template>
                  </a-button>
                </a-tooltip>
              </div>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="dialogTitle"
      :width="680"
      :mask-closable="false"
      unmount-on-close
    >
      <a-form :model="form" :rules="rules" ref="formRef" auto-label-width>
        <a-form-item label="配置名称" field="name">
          <a-input v-model="form.name" placeholder="请输入配置名称" />
        </a-form-item>

        <a-form-item label="关联证书" field="certificate_id">
          <a-select v-model="form.certificate_id" placeholder="请选择证书" allow-search>
            <a-option
              v-for="cert in certificates"
              :key="cert.id"
              :label="`${cert.name} (${cert.domain})`"
              :value="cert.id"
            />
          </a-select>
        </a-form-item>

        <a-form-item label="部署类型" field="deploy_type">
          <a-select v-model="form.deploy_type" placeholder="请选择部署类型" :disabled="!!form.id">
            <a-option value="nginx_ssh">Nginx SSH</a-option>
            <a-option value="k8s_secret">K8s Secret</a-option>
          </a-select>
        </a-form-item>

        <!-- Nginx SSH配置 -->
        <template v-if="form.deploy_type === 'nginx_ssh'">
          <a-divider orientation="left">Nginx配置</a-divider>
          <a-form-item label="资产分组">
            <a-tree-select
              v-model="selectedGroupId"
              :data="assetGroups"
              :field-names="{ key: 'id', title: 'name', children: 'children' }"
              placeholder="请选择资产分组"
              allow-clear
              @change="onGroupChange"
            />
          </a-form-item>
          <a-form-item label="目标主机" field="target_config.host_id">
            <a-select
              v-model="form.target_config.host_id"
              placeholder="请先选择资产分组"
              allow-search
              :loading="hostsLoading"
              :disabled="!selectedGroupId"
              allow-clear
            >
              <a-option
                v-for="host in hosts"
                :key="host.id"
                :label="`${host.name} (${host.ip})`"
                :value="host.id"
              >
                <div class="host-option">
                  <span class="host-name">{{ host.name }}</span>
                  <span class="host-ip">{{ host.ip }}</span>
                </div>
              </a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="证书路径" field="target_config.cert_path">
            <a-input v-model="form.target_config.cert_path" placeholder="/etc/nginx/ssl/cert.pem" />
          </a-form-item>
          <a-form-item label="私钥路径" field="target_config.key_path">
            <a-input v-model="form.target_config.key_path" placeholder="/etc/nginx/ssl/key.pem" />
          </a-form-item>
          <a-form-item label="备份旧证书">
            <a-switch v-model="form.target_config.backup_enabled" />
            <span style="margin-left: 12px; color: var(--ops-text-tertiary); font-size: 13px;">部署前备份原有证书文件</span>
          </a-form-item>
          <a-form-item label="备份路径" v-if="form.target_config.backup_enabled">
            <a-input v-model="form.target_config.backup_path" placeholder="留空则默认备份到证书目录下的backup文件夹" />
          </a-form-item>
        </template>

        <!-- K8s Secret配置 -->
        <template v-if="form.deploy_type === 'k8s_secret'">
          <a-divider orientation="left">K8s配置</a-divider>

          <!-- 容器插件不可用提示 -->
          <a-alert
            v-if="!k8sPluginAvailable"
            title="容器管理插件未启用"
            type="warning"
            :closable="false"
            style="margin-bottom: 16px;"
          >
            该部署类型需要启用容器管理插件。请先在插件管理中启用 Kubernetes 插件，并添加集群配置。
          </a-alert>

          <a-form-item label="K8s集群" field="target_config.cluster_id">
            <a-select
              v-model="form.target_config.cluster_id"
              placeholder="请选择K8s集群"
              allow-search
              :loading="k8sClustersLoading"
              :disabled="!k8sPluginAvailable"
              @change="onClusterChange"
            >
              <a-option
                v-for="cluster in k8sClusters"
                :key="cluster.id"
                :label="`${cluster.name}${cluster.alias ? ' (' + cluster.alias + ')' : ''}`"
                :value="cluster.id"
              >
                <div class="cluster-option">
                  <span class="cluster-name">{{ cluster.name }}</span>
                  <span class="cluster-alias">{{ cluster.alias || cluster.apiEndpoint }}</span>
                </div>
              </a-option>
            </a-select>
            <div class="form-tip">从容器管理中选择已配置的K8s集群</div>
          </a-form-item>
          <a-form-item label="命名空间" field="target_config.namespace">
            <a-select
              v-model="form.target_config.namespace"
              placeholder="请先选择集群"
              allow-search
              :loading="k8sNamespacesLoading"
              :disabled="!form.target_config.cluster_id"
              @change="onNamespaceChange"
            >
              <a-option
                v-for="ns in k8sNamespaces"
                :key="ns.name"
                :label="ns.name"
                :value="ns.name"
              />
            </a-select>
          </a-form-item>

          <a-form-item label="Secret名称" field="target_config.secret_name">
            <a-select
              v-model="form.target_config.secret_name"
              placeholder="请先选择命名空间"
              allow-search
              allow-create
              :loading="k8sSecretsLoading"
              :disabled="!form.target_config.namespace"
            >
              <a-option
                v-for="secret in k8sSecrets"
                :key="secret.name"
                :label="secret.name"
                :value="secret.name"
              >
                <div class="secret-option">
                  <span class="secret-name">{{ secret.name }}</span>
                  <span class="secret-type">{{ secret.type }}</span>
                </div>
              </a-option>
            </a-select>
            <div class="form-tip">可选择已有Secret或输入新名称创建，将以 kubernetes.io/tls 类型创建/更新</div>
          </a-form-item>

          <a-form-item label="触发滚动更新">
            <a-switch v-model="form.target_config.trigger_rollout" />
            <div class="form-tip">部署后触发关联Deployment滚动更新</div>
          </a-form-item>

          <a-form-item label="Deployment" v-if="form.target_config.trigger_rollout">
            <a-select
              v-model="form.target_config.deployments"
              multiple
              allow-search
              allow-create
              placeholder="选择或输入Deployment名称"
              :loading="k8sDeploymentsLoading"
            >
              <a-option
                v-for="deploy in k8sDeployments"
                :key="deploy.name"
                :label="deploy.name"
                :value="deploy.name"
              >
                <div class="deploy-option">
                  <span class="deploy-name">{{ deploy.name }}</span>
                  <span class="deploy-ready">{{ deploy.ready }}</span>
                </div>
              </a-option>
            </a-select>
            <div class="form-tip">可从列表选择或手动输入Deployment名称</div>
          </a-form-item>
        </template>

        <a-form-item label="自动部署">
          <a-switch v-model="form.auto_deploy" />
          <div class="form-tip">证书续期后自动部署</div>
        </a-form-item>

        <a-form-item label="启用">
          <a-switch v-model="form.enabled" />
        </a-form-item>
      </a-form>

      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleSubmit" :loading="submitting">保存</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import {
  IconPlus,
  IconRefresh,
  IconEdit,
  IconDelete,
  IconUpload,
  IconLink,
  IconExclamationCircle
} from '@arco-design/web-vue/es/icon'
import {
  getDeployConfigs,
  createDeployConfig,
  updateDeployConfig,
  deleteDeployConfig,
  executeDeploy,
  testDeployConfig,
  getCertificates
} from '../api/ssl-cert'
import { getHostList, getHost } from '@/api/host'
import { getGroupTree } from '@/api/assetGroup'
import { getClusterList, getNamespaces, getDeployments, getSecrets } from '@/api/kubernetes'
import type { Cluster, NamespaceInfo, DeploymentInfo, SecretInfo } from '@/api/kubernetes'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const formRef = ref<FormInstance>()
const hostsLoading = ref(false)

// 搜索
const searchForm = reactive({
  deploy_type: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 表格数据
const tableData = ref<any[]>([])

// 证书列表
const certificates = ref<any[]>([])

// 资产分组列表
const assetGroups = ref<any[]>([])

// 选中的分组ID
const selectedGroupId = ref<number | null>(null)

// 主机列表
const hosts = ref<any[]>([])

// K8s 相关
const k8sClusters = ref<Cluster[]>([])
const k8sNamespaces = ref<NamespaceInfo[]>([])
const k8sSecrets = ref<SecretInfo[]>([])
const k8sDeployments = ref<DeploymentInfo[]>([])
const k8sClustersLoading = ref(false)
const k8sNamespacesLoading = ref(false)
const k8sSecretsLoading = ref(false)
const k8sDeploymentsLoading = ref(false)
const k8sPluginAvailable = ref(true)

// 表单数据
const form = reactive({
  id: 0,
  name: '',
  certificate_id: null as number | null,
  deploy_type: '',
  target_config: {} as Record<string, any>,
  auto_deploy: true,
  enabled: true
})

// 表单验证规则
const rules = {
  name: [{ required: true, message: '请输入配置名称' }],
  certificate_id: [{ required: true, message: '请选择证书' }],
  deploy_type: [{ required: true, message: '请选择部署类型' }],
  'target_config.host_id': [{ required: true, message: '请选择目标主机' }],
  'target_config.cert_path': [{ required: true, message: '请输入证书路径' }],
  'target_config.key_path': [{ required: true, message: '请输入私钥路径' }],
  'target_config.cluster_id': [{ required: true, message: '请选择K8s集群' }],
  'target_config.namespace': [{ required: true, message: '请选择命名空间' }],
  'target_config.secret_name': [{ required: true, message: '请选择或输入Secret名称' }]
}

// 监听部署类型变化，初始化 target_config 默认值
watch(() => form.deploy_type, (newType) => {
  // 只在新增模式下初始化默认值
  if (form.id === 0) {
    if (newType === 'nginx_ssh') {
      form.target_config = {
        host_id: null,
        cert_path: '',
        key_path: '',
        backup_enabled: false,
        backup_path: ''
      }
      selectedGroupId.value = null
      hosts.value = []
    } else if (newType === 'k8s_secret') {
      form.target_config = {
        cluster_id: null,
        namespace: '',
        secret_name: '',
        trigger_rollout: false,
        deployments: []
      }
      // 加载 K8s 集群列表
      loadK8sClusters()
    }
  }
})

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return null
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getDeployConfigs({
      page: pagination.page,
      page_size: pagination.pageSize,
      deploy_type: searchForm.deploy_type || undefined
    })
    tableData.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 加载证书列表
const loadCertificates = async () => {
  try {
    const res = await getCertificates({ page: 1, page_size: 1000 })
    certificates.value = res.list || []
  } catch (error) {
    // ignore
  }
}

// 加载资产分组（保持树形结构）
const loadAssetGroups = async () => {
  try {
    const res = await getGroupTree()
    assetGroups.value = res.data || res || []
  } catch (error) {
    // ignore
  }
}

// 当分组改变时加载该分组下的主机
const onGroupChange = async (groupId: number | null, resetHostId: boolean = true) => {
  hosts.value = []
  if (resetHostId) {
    form.target_config.host_id = null
  }
  if (!groupId) return

  hostsLoading.value = true
  try {
    const res = await getHostList({ page: 1, page_size: 500, groupId })
    hosts.value = res.data?.list || res.list || []
  } catch (error) {
    // ignore
  } finally {
    hostsLoading.value = false
  }
}

// 加载单个主机信息（编辑时回显用）
const loadHostById = async (hostId: number) => {
  if (!hostId) return
  try {
    const res = await getHost(hostId)
    const hostData = res.data || res
    if (hostData && hostData.id) {
      // 设置选中的分组
      if (hostData.groupId) {
        selectedGroupId.value = hostData.groupId
        // 加载该分组下的主机，不重置 host_id
        await onGroupChange(hostData.groupId, false)
      } else {
        // 如果主机没有分组，直接添加到列表中
        hosts.value = [hostData]
      }
    }
  } catch (error) {
    // ignore
  }
}

// 加载 K8s 集群列表
const loadK8sClusters = async () => {
  k8sClustersLoading.value = true
  k8sPluginAvailable.value = true
  try {
    const res = await getClusterList()
    // 处理可能的 AxiosResponse 包装
    k8sClusters.value = (res as any)?.data || res || []
  } catch (error: any) {
    k8sClusters.value = []
    // 如果是 404 或其他错误，说明容器插件可能未启用
    if (error?.response?.status === 404 || error?.message?.includes('404')) {
      k8sPluginAvailable.value = false
    }
  } finally {
    k8sClustersLoading.value = false
  }
}

// 当选择集群时加载命名空间
const onClusterChange = async (clusterId: number | null) => {
  k8sNamespaces.value = []
  k8sSecrets.value = []
  k8sDeployments.value = []
  form.target_config.namespace = ''
  form.target_config.secret_name = ''
  form.target_config.deployments = []

  if (!clusterId) return

  k8sNamespacesLoading.value = true
  try {
    const res = await getNamespaces(clusterId)
    k8sNamespaces.value = (res as any)?.data || res || []
  } catch (error) {
    k8sNamespaces.value = []
  } finally {
    k8sNamespacesLoading.value = false
  }
}

// 当选择命名空间时加载 Secrets 和 Deployments
const onNamespaceChange = async (namespace: string) => {
  k8sSecrets.value = []
  k8sDeployments.value = []
  form.target_config.secret_name = ''
  form.target_config.deployments = []

  if (!namespace || !form.target_config.cluster_id) return

  // 并行加载 Secrets 和 Deployments
  k8sSecretsLoading.value = true
  k8sDeploymentsLoading.value = true

  try {
    const [secretsRes, deploymentsRes] = await Promise.all([
      getSecrets(form.target_config.cluster_id, namespace),
      getDeployments(form.target_config.cluster_id, namespace)
    ])
    k8sSecrets.value = (secretsRes as any)?.data || secretsRes || []
    k8sDeployments.value = (deploymentsRes as any)?.data || deploymentsRes || []
  } catch (error) {
    k8sSecrets.value = []
    k8sDeployments.value = []
  } finally {
    k8sSecretsLoading.value = false
    k8sDeploymentsLoading.value = false
  }
}

// 编辑时加载 K8s 相关数据
const loadK8sDataForEdit = async (clusterId: number, namespace: string) => {
  // 先加载集群列表
  await loadK8sClusters()

  // 再加载命名空间
  if (clusterId) {
    k8sNamespacesLoading.value = true
    try {
      const res = await getNamespaces(clusterId)
      k8sNamespaces.value = (res as any)?.data || res || []
    } catch (error) {
      k8sNamespaces.value = []
    } finally {
      k8sNamespacesLoading.value = false
    }
  }

  // 加载 Secrets 和 Deployments
  if (clusterId && namespace) {
    k8sSecretsLoading.value = true
    k8sDeploymentsLoading.value = true
    try {
      const [secretsRes, deploymentsRes] = await Promise.all([
        getSecrets(clusterId, namespace),
        getDeployments(clusterId, namespace)
      ])
      k8sSecrets.value = (secretsRes as any)?.data || secretsRes || []
      k8sDeployments.value = (deploymentsRes as any)?.data || deploymentsRes || []
    } catch (error) {
      k8sSecrets.value = []
      k8sDeployments.value = []
    } finally {
      k8sSecretsLoading.value = false
      k8sDeploymentsLoading.value = false
    }
  }
}

// 新增
const handleAdd = () => {
  dialogTitle.value = '新增部署配置'
  hosts.value = []
  selectedGroupId.value = null
  k8sClusters.value = []
  k8sNamespaces.value = []
  k8sSecrets.value = []
  k8sDeployments.value = []
  k8sPluginAvailable.value = true
  Object.assign(form, {
    id: 0,
    name: '',
    certificate_id: null,
    deploy_type: '',
    target_config: {},
    auto_deploy: true,
    enabled: true
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: any) => {
  dialogTitle.value = '编辑部署配置'
  hosts.value = []
  selectedGroupId.value = null
  k8sNamespaces.value = []
  k8sSecrets.value = []
  k8sDeployments.value = []
  const targetConfig = row.target_config ? JSON.parse(row.target_config) : {}
  Object.assign(form, {
    id: row.id,
    name: row.name,
    certificate_id: row.certificate_id,
    deploy_type: row.deploy_type,
    target_config: targetConfig,
    auto_deploy: row.auto_deploy,
    enabled: row.enabled
  })
  // 如果是 nginx_ssh 类型且有 host_id，加载主机信息以便回显
  if (row.deploy_type === 'nginx_ssh' && targetConfig.host_id) {
    loadHostById(targetConfig.host_id)
  }
  // 如果是 k8s_secret 类型，加载 K8s 相关数据以便回显
  if (row.deploy_type === 'k8s_secret') {
    loadK8sDataForEdit(targetConfig.cluster_id, targetConfig.namespace)
  }
  dialogVisible.value = true
}

// 提交
const handleSubmit = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
    submitting.value = true
    try {
      if (form.id) {
        await updateDeployConfig(form.id, {
          name: form.name,
          target_config: form.target_config,
          auto_deploy: form.auto_deploy,
          enabled: form.enabled
        })
        Message.success('保存成功')
        dialogVisible.value = false
        loadData()
      } else {
        await createDeployConfig({
          name: form.name,
          certificate_id: form.certificate_id!,
          deploy_type: form.deploy_type,
          target_config: form.target_config,
          auto_deploy: form.auto_deploy,
          enabled: form.enabled
        })
        Message.success('创建成功')
        dialogVisible.value = false
        loadData()
      }
    } catch (error: any) {
      // 错误已由 request 拦截器处理
    } finally {
      submitting.value = false
    }
  } catch {
    // 验证失败
  }
}

// 自动部署切换
const handleAutoDeployChange = async (row: any) => {
  try {
    await updateDeployConfig(row.id, { auto_deploy: row.auto_deploy })
    Message.success('更新成功')
  } catch (error) {
    row.auto_deploy = !row.auto_deploy
  }
}

// 立即部署
const handleDeploy = async (row: any) => {
  Modal.warning({
    title: '提示',
    content: '确定要立即部署证书吗？',
    hideCancel: false,
    onOk: async () => {
      row.deploying = true
      try {
        await executeDeploy(row.id)
        Message.success('部署成功')
        loadData()
      } catch (error: any) {
        // 错误已由 request 拦截器处理
      } finally {
        row.deploying = false
      }
    }
  })
}

// 测试配置
const handleTest = async (row: any) => {
  try {
    row.testing = true
    await testDeployConfig(row.id)
    Message.success('配置测试成功')
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    row.testing = false
  }
}

// 删除
const handleDelete = async (row: any) => {
  Modal.warning({
    title: '提示',
    content: '确定要删除该部署配置吗？',
    hideCancel: false,
    onOk: async () => {
      loading.value = true
      try {
        await deleteDeployConfig(row.id)
        Message.success('删除成功')
        loadData()
      } catch (error: any) {
        // 错误已由 request 拦截器处理
      } finally {
        loading.value = false
      }
    }
  })
}

onMounted(() => {
  loadData()
  loadCertificates()
  loadAssetGroups()
})
</script>

<style scoped>
.deploy-config-container { padding: 0; background-color: transparent; }

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
}
.search-inputs { display: flex; gap: 12px; }
.search-input { width: 200px; }

.table-wrapper {
  background: #fff; border-radius: var(--ops-border-radius-md, 8px);
  box-shadow: 0 2px 12px rgba(0,0,0,0.04); overflow: hidden; padding: 16px;
}
.action-buttons { display: flex; gap: 4px; align-items: center; justify-content: center; }
.action-btn { width: 32px; height: 32px; border-radius: 6px; transition: all 0.2s ease; }
.action-btn:hover { transform: scale(1.1); }
.action-deploy:hover { background-color: #e8ffea; color: var(--ops-success, #00b42a); }
.action-test:hover { background-color: #fff7e8; color: var(--ops-warning, #ff7d00); }
.action-edit:hover { background-color: var(--ops-primary-bg, #e8f0ff); color: var(--ops-primary, #165dff); }
.action-delete:hover { background-color: #ffece8; color: var(--ops-danger, #f53f3f); }

.form-tip { font-size: 12px; color: var(--ops-text-tertiary, #86909c); margin-top: 6px; line-height: 1.5; }
.host-option, .cluster-option, .secret-option, .deploy-option {
  display: flex; justify-content: space-between; align-items: center; width: 100%;
}
.host-name, .cluster-name, .secret-name, .deploy-name { color: var(--ops-text-primary, #1d2129); }
.host-ip, .cluster-alias { color: var(--ops-text-tertiary, #86909c); font-size: 12px; }
.secret-type { color: var(--ops-text-tertiary, #86909c); font-size: 11px; max-width: 150px; overflow: hidden; text-overflow: ellipsis; }
.deploy-ready { color: var(--ops-success, #00b42a); font-size: 12px; }
</style>
