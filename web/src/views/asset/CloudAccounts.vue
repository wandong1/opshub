<template>
  <div class="cloud-accounts-page">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-cloud /></div>
        <div>
          <h2 class="page-title">云账号管理</h2>
          <p class="page-subtitle">管理云平台账号，用于导入云主机</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button v-permission="'cloud-accounts:create'" @click="handleAdd" type="primary">
          <template #icon><icon-plus /></template>
          新增账号
        </a-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <a-input
          v-model="searchForm.keyword"
          placeholder="搜索账号名称..."
          allow-clear
          class="search-input"
        >
          <template #prefix>
            <icon-search class="search-icon" />
          </template>
        </a-input>

        <a-select
          v-model="searchForm.provider"
          placeholder="云厂商"
          allow-clear
          class="search-input"
        >
          <a-option label="全部" value="" />
          <a-option label="阿里云" value="aliyun" />
          <a-option label="腾讯云" value="tencent" />
          <a-option label="京东云" value="jdcloud" />
        </a-select>

        <a-select
          v-model="searchForm.status"
          placeholder="状态"
          allow-clear
          class="search-input"
        >
          <a-option label="全部" value="" />
          <a-option label="启用" :value="1" />
          <a-option label="禁用" :value="0" />
        </a-select>
      </div>

      <div class="search-actions">
        <a-button class="reset-btn" @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
      </div>
    </div>

    <!-- 账号列表 -->
    <div class="table-wrapper">
      <a-table
        :data="filteredAccountList"
        :loading="loading"
        stripe
        :bordered="{ cell: true }"
        :pagination="false"
        class="modern-table"
      >
        <template #columns>
          <a-table-column title="账号名称" data-index="name" :min-width="150" />
          <a-table-column title="云厂商" align="center" :width="120">
            <template #cell="{ record }">
              <a-tag :color="getProviderColor(record.provider)">
                {{ record.providerText }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="区域" data-index="region" :min-width="120">
            <template #cell="{ record }">
              <span v-if="record.region">{{ record.region }}</span>
              <span v-else class="text-muted">-</span>
            </template>
          </a-table-column>
          <a-table-column title="Access Key" :min-width="200">
            <template #cell>
              <span class="access-key">************</span>
            </template>
          </a-table-column>
          <a-table-column title="状态" align="center" :width="80">
            <template #cell="{ record }">
              <a-switch
                v-model="record.status"
                :checked-value="1"
                :unchecked-value="0"
                @change="handleStatusChange(record)"
              />
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="createTime" :width="180" />
          <a-table-column title="操作" :width="120" align="center" fixed="right">
            <template #cell="{ record }">
              <a-button
                type="text"
                :status="record.status === 1 ? 'normal' : undefined"
                v-permission="'cloud-accounts:import'"
                :disabled="record.status === 0"
                @click="handleImportHost(record)"
                title="导入主机"
              >
                <template #icon><icon-upload /></template>
              </a-button>
              <a-button v-permission="'cloud-accounts:update'" type="text" @click="handleEdit(record)" title="编辑">
                <template #icon><icon-edit /></template>
              </a-button>
              <a-button v-permission="'cloud-accounts:delete'" type="text" status="danger" @click="handleDelete(record)" title="删除">
                <template #icon><icon-delete /></template>
              </a-button>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="isEdit ? '编辑云账号' : '新增云账号'"
      :width="800"
      unmount-on-close
      @close="handleDialogClose"
      class="account-dialog"
    >
      <div class="dialog-content">
        <a-form :model="form" :rules="rules" ref="formRef" auto-label-width layout="horizontal" class="account-form">
          <!-- 云厂商选择 -->
          <a-form-item label="云厂商" field="provider" :rules="[{ required: true, message: '请选择云厂商' }]">
            <div class="provider-options-inline">
              <div
                v-for="provider in providers"
                :key="provider.value"
                :class="['provider-option', { active: form.provider === provider.value }]"
                @click="form.provider = provider.value"
              >
                <span class="provider-short">{{ provider.short }}</span>
                <span class="provider-name">{{ provider.label }}</span>
              </div>
            </div>
          </a-form-item>

          <a-form-item label="账号名称" field="name">
            <a-input v-model="form.name" placeholder="如：生产环境阿里云账号" />
          </a-form-item>
          <template v-if="!isEdit">
            <a-form-item label="Access Key" field="accessKey">
              <a-input v-model="form.accessKey" placeholder="请输入 Access Key ID" />
            </a-form-item>

            <a-form-item label="Secret Key" field="secretKey">
              <a-input-password v-model="form.secretKey" placeholder="请输入 Access Key Secret" />
            </a-form-item>
          </template>

          <a-alert v-else type="info" :closable="false" style="margin-bottom: 20px;">
            如需修改 Access Key 或 Secret Key，请删除后重新创建账号
          </a-alert>

          <a-form-item label="默认区域">
            <a-select v-model="form.region" placeholder="选择默认区域" allow-search style="width: 100%">
              <a-option v-for="region in currentRegions" :key="region.value" :label="region.label" :value="region.value" />
            </a-select>
          </a-form-item>

          <a-form-item label="备注">
            <a-textarea v-model="form.description" :auto-size="{ minRows: 2 }" placeholder="可选，填写备注信息" />
          </a-form-item>

          <a-form-item label="状态">
            <a-radio-group v-model="form.status">
              <a-radio :value="1">启用</a-radio>
              <a-radio :value="0">禁用</a-radio>
            </a-radio-group>
          </a-form-item>
        </a-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="dialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSubmit" :loading="submitting">
            {{ isEdit ? '保存修改' : '创建账号' }}
          </a-button>
        </div>
      </template>
    </a-modal>
    <!-- 导入云主机对话框 -->
    <a-modal
      v-model:visible="importDialogVisible"
      title="导入云主机"
      :width="1200"
      unmount-on-close
      @close="handleImportDialogClose"
      class="import-dialog"
    >
      <a-form :model="importForm" auto-label-width layout="horizontal" class="import-form">
        <div class="form-row">
          <a-form-item label="云账号" class="form-item-full">
            <a-select v-model="importForm.accountId" placeholder="请选择云账号" style="width: 100%" @change="handleAccountChange">
              <a-option v-for="acc in enabledAccountList" :key="acc.id" :label="acc.name" :value="acc.id">
                <span>{{ acc.name }}</span>
                <a-tag :color="getProviderColor(acc.provider)" size="small" style="margin-left: 8px;">
                  {{ acc.providerText }}
                </a-tag>
              </a-option>
            </a-select>
          </a-form-item>
        </div>
        <div class="form-row">
          <a-form-item label="区域" class="form-item-half">
            <a-select v-model="importForm.region" placeholder="请选择区域" style="width: 100%" allow-search>
              <a-option v-for="region in regions" :key="region.value" :label="region.label" :value="region.value" />
            </a-select>
          </a-form-item>
          <a-form-item label="所属分组" class="form-item-half">
            <a-tree-select
              v-model="importForm.groupId"
              :data="groupTreeOptions"
              :field-names="{ key: 'id', title: 'name', children: 'children' }"
              allow-clear
              placeholder="请选择分组"
              style="width: 100%"
            />
          </a-form-item>
        </div>
      </a-form>
      <a-spin :loading="loadingInstances" class="instances-container" style="display: block;">
        <a-alert v-if="!selectedAccount" title="请先选择云账号" type="info" :closable="false" />
        <a-alert v-else-if="!importForm.region" title="请选择区域" type="info" :closable="false" />
        <div v-else-if="cloudHosts.length === 0" class="empty-instances">
          <a-empty description="该区域下没有可导入的云主机" />
        </div>
        <div v-else class="instances-list">
          <div class="instances-header">
            <div class="instances-info">
              <span class="instances-count">找到 <strong>{{ cloudHosts.length }}</strong> 台云主机</span>
              <span class="instances-region">当前区域: {{ importForm.region }}</span>
            </div>
            <a-checkbox v-model="selectAll" @change="handleSelectAll">
              <span class="select-all-text">全选</span>
            </a-checkbox>
          </div>
          <a-table
            ref="cloudHostsTableRef"
            :data="cloudHosts"
            :row-selection="rowSelection"
            v-model:selected-keys="selectedInstanceKeys"
            @selection-change="handleSelectionChange"
            :scroll="{ y: 400 }"
            class="cloud-hosts-table"
            stripe
            :bordered="{ cell: true }"
            :pagination="false"
            row-key="instanceId"
          >
            <template #columns>
              <a-table-column title="实例名称" data-index="name" :min-width="160" ellipsis tooltip>
                <template #cell="{ record }">
                  <div class="instance-name">
                    <icon-desktop class="instance-icon" />
                    <span>{{ record.name }}</span>
                  </div>
                </template>
              </a-table-column>
              <a-table-column title="实例ID" data-index="instanceId" :min-width="180" ellipsis tooltip>
                <template #cell="{ record }">
                  <span class="instance-id">{{ record.instanceId }}</span>
                </template>
              </a-table-column>
              <a-table-column title="IP地址" :min-width="150">
                <template #cell="{ record }">
                  <div class="ip-list">
                    <div v-if="record.publicIp" class="ip-item public-ip">
                      <a-tag size="small" color="green">公</a-tag>
                      <span>{{ record.publicIp }}</span>
                    </div>
                    <div v-if="record.privateIp" class="ip-item private-ip">
                      <a-tag size="small" color="gray">私</a-tag>
                      <span>{{ record.privateIp }}</span>
                    </div>
                    <span v-if="!record.publicIp && !record.privateIp" class="text-muted">-</span>
                  </div>
                </template>
              </a-table-column>
              <a-table-column title="操作系统" data-index="os" :min-width="120" ellipsis tooltip />
              <a-table-column title="状态" :width="90" align="center">
                <template #cell="{ record }">
                  <a-tag :color="getStatusColor(record.status)" size="small">
                    {{ getStatusText(record.status) }}
                  </a-tag>
                </template>
              </a-table-column>
            </template>
          </a-table>
        </div>
      </a-spin>

      <template #footer>
        <a-button @click="importDialogVisible = false" size="large">取消</a-button>
        <a-button type="primary" @click="handleConfirmImport" :loading="importing" :disabled="selectedInstances.length === 0" size="large">
          <template #icon><icon-upload v-if="!importing" /></template>
          <span>导入 {{ selectedInstances.length > 0 ? `(${selectedInstances.length})` : '' }}</span>
        </a-button>
      </template>
    </a-modal>
  </div>
</template>
<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch, nextTick } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconUpload,
  IconCloud,
  IconSearch,
  IconRefresh,
  IconDesktop
} from '@arco-design/web-vue/es/icon'
import {
  getCloudAccounts,
  createCloudAccount,
  updateCloudAccount,
  deleteCloudAccount,
  importFromCloud,
  getCloudInstances,
  getCloudRegions
} from '@/api/host'
import { getGroupTree } from '@/api/assetGroup'

const loading = ref(false)
const accountList = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  keyword: '',
  provider: '',
  status: ''
})

// 过滤后的账号列表
const filteredAccountList = computed(() => {
  return accountList.value.filter((account: any) => {
    const matchKeyword = !searchForm.keyword || account.name?.toLowerCase().includes(searchForm.keyword.toLowerCase())
    const matchProvider = !searchForm.provider || account.provider === searchForm.provider
    const matchStatus = searchForm.status === '' || account.status === parseInt(searchForm.status)
    return matchKeyword && matchProvider && matchStatus
  })
})

// 重置搜索
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.provider = ''
  searchForm.status = ''
}
// 对话框相关
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref()

// 云厂商选项
const providers = [
  { value: 'aliyun', label: '阿里云', short: '阿里' },
  { value: 'tencent', label: '腾讯云', short: '腾讯' },
  { value: 'jdcloud', label: '京东云', short: '京东' }
]

// 当前厂商的区域列表（新增/编辑对话框用）
const currentRegions = ref<any[]>([])

const form = reactive({
  id: 0,
  name: '',
  provider: 'aliyun',
  accessKey: '',
  secretKey: '',
  region: '',
  description: '',
  status: 1
})

// 动态验证规则
const rules = computed(() => {
  const baseRules: any = {
    name: [{ required: true, message: '请输入账号名称' }],
    provider: [{ required: true, message: '请选择云厂商' }]
  }

  // 只有新增时才验证 Access Key 和 Secret Key
  if (!isEdit.value) {
    baseRules.accessKey = [{ required: true, message: '请输入Access Key' }]
    baseRules.secretKey = [{ required: true, message: '请输入Secret Key' }]
  }

  return baseRules
})

// 获取启用的云账号列表
const enabledAccountList = computed(() => {
  return accountList.value.filter((a: any) => a.status === 1)
})
// 导入相关
const importDialogVisible = ref(false)
const importing = ref(false)
const loadingInstances = ref(false)
const selectedAccount = ref<any>(null)
const cloudHosts = ref<any[]>([])
const selectedInstances = ref<string[]>([])
const selectedInstanceKeys = ref<string[]>([])
const selectAll = ref(false)
const groupTreeOptions = ref<any[]>([])
const cloudHostsTableRef = ref()

const importForm = reactive({
  accountId: null as number | null,
  region: '',
  groupId: null as number | null
})

const regions = ref<any[]>([])

// row-selection config for a-table
const rowSelection = reactive({
  type: 'checkbox' as const,
  showCheckedAll: false
})

// 获取云厂商颜色（Arco tag color）
const getProviderColor = (provider: string) => {
  const colorMap: Record<string, string> = {
    aliyun: 'orangered',
    tencent: 'gray',
    jdcloud: 'red'
  }
  return colorMap[provider] || 'gray'
}

// 掩码Access Key
const maskAccessKey = (key: string) => {
  if (!key || key.length <= 8) return key
  return key.substring(0, 4) + '****' + key.substring(key.length - 4)
}

// 加载账号列表
const loadAccountList = async () => {
  loading.value = true
  try {
    const res = await getCloudAccounts()
    accountList.value = Array.isArray(res) ? res : []
  } catch (error) {
  } finally {
    loading.value = false
  }
}
// 加载分组树
const loadGroupTree = async () => {
  try {
    const res = await getGroupTree()
    groupTreeOptions.value = res || []
  } catch (error) {
  }
}

// 新增
const handleAdd = () => {
  Object.assign(form, {
    id: 0,
    name: '',
    provider: 'aliyun',
    accessKey: '',
    secretKey: '',
    region: '',
    description: '',
    status: 1
  })
  isEdit.value = false
  currentRegions.value = getLocalRegions('aliyun')
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = async (row: any) => {
  isEdit.value = true

  Object.assign(form, {
    id: row.id,
    name: row.name,
    provider: row.provider,
    accessKey: '',
    secretKey: '',
    region: row.region || '',
    description: row.description || '',
    status: row.status
  })

  try {
    const res = await getCloudRegions(row.id)
    currentRegions.value = Array.isArray(res) ? res : []
  } catch (error) {
    currentRegions.value = getLocalRegions(row.provider)
  }

  nextTick(() => {
    formRef.value?.clearValidate()
  })
  dialogVisible.value = true
}
// 删除
const handleDelete = (row: any) => {
  Modal.warning({
    title: '提示',
    content: `确定要删除云账号"${row.name}"吗？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await deleteCloudAccount(row.id)
        Message.success('删除成功')
        loadAccountList()
      } catch (error: any) {
        Message.error(error.message || '删除失败')
      }
    }
  })
}

// 状态切换
const handleStatusChange = async (row: any) => {
  try {
    await updateCloudAccount(row.id, {
      id: row.id,
      name: row.name,
      provider: row.provider,
      region: row.region || '',
      description: row.description || '',
      status: row.status
    })
    Message.success('状态更新成功')
  } catch (error: any) {
    row.status = row.status === 1 ? 0 : 1
    Message.error(error.message || '状态更新失败')
  }
}

// 提交表单
const handleSubmit = async () => {
  const errors = await formRef.value?.validate()
  if (errors) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateCloudAccount(form.id, form)
      Message.success('更新成功')
    } else {
      await createCloudAccount(form)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadAccountList()
  } catch (error: any) {
    Message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}
// 对话框关闭
const handleDialogClose = () => {
  formRef.value?.resetFields()
}

// 导入主机
const handleImportHost = (row: any) => {
  if (row.status === 0) {
    Message.warning('该账号已禁用，无法导入主机')
    return
  }
  selectedAccount.value = row
  importForm.accountId = row.id
  importForm.region = row.region || ''
  importForm.groupId = null
  handleAccountChange()
  importDialogVisible.value = true
}

// 账号变化
const handleAccountChange = async () => {
  const account = accountList.value.find((a: any) => a.id === importForm.accountId)
  if (!account) return

  selectedAccount.value = account

  cloudHosts.value = []
  selectedInstances.value = []
  selectedInstanceKeys.value = []
  selectAll.value = false
  regions.value = []
  importForm.region = ''

  try {
    const res = await getCloudRegions(account.id)
    regions.value = Array.isArray(res) ? res : []

    if (account.region) {
      const hasDefaultRegion = regions.value.some((r: any) => r.value === account.region)
      if (hasDefaultRegion) {
        importForm.region = account.region
      }
    }
  } catch (error: any) {
    Message.error(error.message || '加载区域列表失败')
  }
}
// 加载云主机实例列表
const loadCloudInstances = async () => {
  if (!importForm.accountId || !importForm.region) return

  loadingInstances.value = true
  try {
    const res = await getCloudInstances(importForm.accountId, importForm.region)
    cloudHosts.value = Array.isArray(res) ? res : []
  } catch (error: any) {
    Message.error(error.message || '加载云主机列表失败')
    cloudHosts.value = []
  } finally {
    loadingInstances.value = false
  }
}

// 监听区域变化，自动加载实例列表
watch(() => importForm.region, () => {
  if (importForm.region) {
    loadCloudInstances()
  }
})

// 监听表单中的云厂商变化，更新区域列表（新增/编辑对话框用）
watch(() => form.provider, async (newProvider) => {
  if (!newProvider) return
  form.region = ''

  if (isEdit.value && form.id > 0) {
    try {
      const res = await getCloudRegions(form.id)
      currentRegions.value = Array.isArray(res) ? res : []
    } catch (error) {
      currentRegions.value = getLocalRegions(newProvider)
    }
  } else {
    currentRegions.value = getLocalRegions(newProvider)
  }
})
// 本地常用区域列表（新增账号时使用）
const getLocalRegions = (provider: string): any[] => {
  const localMap: Record<string, any[]> = {
    aliyun: [
      { value: 'cn-hangzhou', label: '华东1 (杭州)' },
      { value: 'cn-shanghai', label: '华东2 (上海)' },
      { value: 'cn-beijing', label: '华北2 (北京)' },
      { value: 'cn-shenzhen', label: '华南1 (深圳)' },
      { value: 'cn-guangzhou', label: '华南2 (广州)' },
      { value: 'cn-chengdu', label: '西南1 (成都)' }
    ],
    tencent: [
      { value: 'ap-guangzhou', label: '华南地区 (广州)' },
      { value: 'ap-shanghai', label: '华东地区 (上海)' },
      { value: 'ap-beijing', label: '华北地区 (北京)' },
      { value: 'ap-chengdu', label: '西南地区 (成都)' },
      { value: 'ap-chongqing', label: '西南地区 (重庆)' }
    ]
  }
  return localMap[provider] || []
}

// 全选
const handleSelectAll = (checked: boolean | (string | number | boolean)[]) => {
  if (typeof checked === 'boolean') {
    if (checked) {
      selectedInstanceKeys.value = cloudHosts.value.map((h: any) => h.instanceId)
      selectedInstances.value = [...selectedInstanceKeys.value]
    } else {
      selectedInstanceKeys.value = []
      selectedInstances.value = []
    }
  }
}

// 获取状态颜色（Arco tag color）
const getStatusColor = (status: string) => {
  const statusLower = status.toLowerCase()
  const colorMap: Record<string, string> = {
    'running': 'green',
    'starting': 'orangered',
    'stopping': 'orangered',
    'stopped': 'gray',
    'deleted': 'red'
  }
  return colorMap[statusLower] || 'gray'
}
// 获取状态文本
const getStatusText = (status: string) => {
  const statusLower = status.toLowerCase()
  const textMap: Record<string, string> = {
    'running': '运行中',
    'starting': '启动中',
    'stopping': '停止中',
    'stopped': '已停止',
    'deleted': '已删除'
  }
  return textMap[statusLower] || status
}

// 选择变化
const handleSelectionChange = (rowKeys: string[]) => {
  selectedInstances.value = rowKeys
  selectedInstanceKeys.value = rowKeys
  selectAll.value = rowKeys.length === cloudHosts.value.length && cloudHosts.value.length > 0
}

// 确认导入
const handleConfirmImport = async () => {
  if (!importForm.groupId) {
    Message.warning('请选择所属分组')
    return
  }

  importing.value = true
  try {
    await importFromCloud({
      accountId: importForm.accountId,
      region: importForm.region,
      groupId: importForm.groupId,
      instanceIds: selectedInstances.value
    })
    Message.success('导入成功')
    importDialogVisible.value = false
  } catch (error: any) {
    Message.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

// 导入对话框关闭
const handleImportDialogClose = () => {
  cloudHosts.value = []
  selectedInstances.value = []
  selectedInstanceKeys.value = []
  selectAll.value = false
}

onMounted(() => {
  loadAccountList()
  loadGroupTree()
})
</script>
<style scoped>
.cloud-accounts-page {
  padding: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
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
  background: linear-gradient(135deg, var(--ops-primary) 0%, #4080ff 100%);
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
  font-size: 20px;
  font-weight: 600;
  color: var(--ops-text-primary);
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary);
  line-height: 1.4;
}
.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.search-actions {
  display: flex;
  gap: 10px;
}

.reset-btn {
  background: var(--ops-content-bg);
  border-color: var(--ops-border-color);
  color: var(--ops-text-secondary);
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
}

.search-icon {
  color: var(--ops-primary);
}

/* 表格容器 */
.table-wrapper {
  flex: 1;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}
.modern-table {
  width: 100%;
}

.access-key {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: var(--ops-text-secondary);
}

.text-muted {
  color: #c0c4cc;
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
}

/* 对话框样式 */
.dialog-content {
  padding: 0;
}

/* 云厂商选择器 - 内联样式 */
.provider-options-inline {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.provider-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border: 2px solid var(--ops-border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  background: var(--ops-content-bg);
  min-width: 100px;
  justify-content: center;
}

.provider-option:hover {
  border-color: var(--ops-primary);
  background: #f0f5ff;
}
.provider-option.active {
  border-color: var(--ops-primary);
  background: linear-gradient(135deg, #f0f5ff 0%, #e8f0fe 100%);
}

.provider-short {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 700;
  color: #fff;
}

.provider-option:nth-child(1) .provider-short {
  background: linear-gradient(135deg, #ff6a00 0%, #ff9500 100%);
}

.provider-option:nth-child(2) .provider-short {
  background: linear-gradient(135deg, #00a4ff 0%, #00c6ff 100%);
}

.provider-option:nth-child(3) .provider-short {
  background: linear-gradient(135deg, #e1251b 0%, #f363d6 100%);
}

.provider-name {
  font-size: 14px;
  color: var(--ops-text-secondary);
  font-weight: 500;
}

.provider-option.active .provider-name {
  color: var(--ops-primary);
  font-weight: 600;
}

/* 表单样式 */
.account-form :deep(.arco-form-item) {
  margin-bottom: 20px;
}

/* 对话框底部 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.empty-instances {
  padding: 40px 0;
}
/* 导入对话框样式 */
.import-form {
  margin-bottom: 16px;
}

.form-row {
  display: flex;
  gap: 16px;
}

.form-item-full {
  flex: 1;
}

.form-item-half {
  width: 50%;
}

.instances-container {
  min-height: 200px;
}

.instances-list {
  background: var(--ops-content-bg);
  border-radius: 12px;
  padding: 16px;
}

.instances-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--ops-border-color);
}

.instances-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.instances-count {
  font-size: 15px;
  color: var(--ops-text-primary);
}

.instances-count strong {
  color: var(--ops-primary);
  font-size: 18px;
}

.instances-region {
  font-size: 12px;
  color: var(--ops-text-tertiary);
}
.select-all-text {
  font-size: 14px;
  font-weight: 500;
}

.cloud-hosts-table {
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
}

.instance-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.instance-icon {
  color: var(--ops-primary);
  font-size: 16px;
}

.instance-id {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  color: var(--ops-text-secondary);
  background: var(--ops-content-bg);
  padding: 2px 8px;
  border-radius: 4px;
}

.ip-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ip-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.ip-item span {
  font-family: 'Monaco', 'Menlo', monospace;
}

.public-ip span {
  color: #00b42a;
}

.private-ip span {
  color: var(--ops-text-tertiary);
}
</style>
