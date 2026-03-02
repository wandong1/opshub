<template>
  <div class="dns-provider-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-link />
        </div>
        <div>
          <h2 class="page-title">DNS验证配置</h2>
          <p class="page-subtitle">配置DNS服务商API凭证，用于ACME证书申请时的域名所有权验证（DNS-01验证）</p>
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

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <a-table
        :data="tableData"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50] }"
        @page-change="(p: number) => { pagination.page = p; loadData() }"
        @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }"
      >
        <template #columns>
          <a-table-column title="名称" data-index="name" :min-width="120" />
          <a-table-column title="服务商" :width="130" align="center">
            <template #cell="{ record }">
              <a-tag>{{ getProviderName(record.provider) }}</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="邮箱" data-index="email" :min-width="150" />

          <a-table-column title="电话" data-index="phone" :width="130" />

          <a-table-column title="状态" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.enabled" color="green">启用</a-tag>
              <a-tag v-else color="gray">禁用</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="连接测试" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.last_test_ok" color="green" size="small">正常</a-tag>
              <a-tag v-else-if="record.last_test_at" color="red" size="small">失败</a-tag>
              <a-tag v-else color="gray" size="small">未测试</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="上次测试" :width="170">
            <template #cell="{ record }">
              <span>{{ formatDateTime(record.last_test_at) || '-' }}</span>
            </template>
          </a-table-column>

          <a-table-column title="创建时间" :width="170">
            <template #cell="{ record }">
              <span>{{ formatDateTime(record.created_at) }}</span>
            </template>
          </a-table-column>

          <a-table-column title="操作" :width="150" fixed="right" align="center">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="测试连接" position="top">
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
      :width="620"
      :mask-closable="false"
      unmount-on-close
      @cancel="dialogVisible = false"
    >
      <a-form :model="form" :rules="rules" ref="formRef" auto-label-width>
        <a-form-item label="配置名称" field="name">
          <a-input v-model="form.name" placeholder="请输入配置名称" />
        </a-form-item>

        <a-form-item label="DNS服务商" field="provider">
          <a-select v-model="form.provider" placeholder="请选择DNS服务商" :disabled="!!form.id">
            <a-option label="阿里云DNS" value="aliyun" />
          </a-select>
        </a-form-item>

        <!-- 阿里云配置 -->
        <template v-if="form.provider === 'aliyun'">
          <a-form-item label="AccessKey ID" field="config.access_key_id">
            <a-input v-model="form.config.access_key_id" placeholder="请输入AccessKey ID" />
          </a-form-item>
          <a-form-item label="AccessKey Secret" field="config.access_key_secret">
            <a-input-password v-model="form.config.access_key_secret" placeholder="请输入AccessKey Secret" />
          </a-form-item>
        </template>

        <a-divider />

        <a-form-item label="邮箱" field="email">
          <a-input v-model="form.email" placeholder="请输入邮箱地址" />
        </a-form-item>

        <a-form-item label="电话" field="phone">
          <a-input v-model="form.phone" placeholder="请输入电话号码" />
        </a-form-item>

        <a-form-item label="状态" field="enabled">
          <a-select v-model="form.enabled">
            <a-option label="启用" :value="true" />
            <a-option label="禁用" :value="false" />
          </a-select>
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
import { ref, reactive, onMounted } from 'vue'
import { Message, Modal, FormInstance } from '@arco-design/web-vue'
import { IconPlus, IconRefresh, IconEdit, IconDelete, IconLink } from '@arco-design/web-vue/es/icon'
import {
  getDNSProviders,
  getDNSProviderDetail,
  createDNSProvider,
  updateDNSProvider,
  deleteDNSProvider,
  testDNSProvider
} from '../api/ssl-cert'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const formRef = ref<FormInstance>()

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 表格数据
const tableData = ref<any[]>([])

// 表单数据
const form = reactive({
  id: 0,
  name: '',
  provider: '',
  config: {} as Record<string, string>,
  email: '',
  phone: '',
  enabled: true
})

// 表单验证规则
const rules = {
  name: [{ required: true, message: '请输入配置名称' }],
  provider: [{ required: true, message: '请选择DNS服务商' }],
  email: [
    { required: true, message: '请输入邮箱地址' },
    { type: 'email' as const, message: '请输入有效的邮箱地址' }
  ],
  phone: [{ required: true, message: '请输入电话号码' }]
}

// 获取服务商名称
const getProviderName = (provider: string) => {
  const names: Record<string, string> = {
    aliyun: '阿里云'
  }
  return names[provider] || provider
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return null
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getDNSProviders({
      page: pagination.page,
      page_size: pagination.pageSize
    })
    tableData.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 新增
const handleAdd = () => {
  dialogTitle.value = '新增DNS配置'
  Object.assign(form, {
    id: 0,
    name: '',
    provider: '',
    config: {},
    email: '',
    phone: '',
    enabled: true
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = async (row: any) => {
  dialogTitle.value = '编辑DNS配置'
  try {
    // 获取完整详情(包含配置)
    const detail = await getDNSProviderDetail(row.id)
    Object.assign(form, {
      id: detail.id,
      name: detail.name,
      provider: detail.provider,
      config: detail.config || {},
      email: detail.email || '',
      phone: detail.phone || '',
      enabled: detail.enabled
    })
    dialogVisible.value = true
  } catch (error) {
    // 错误已由 request 拦截器处理
  }
}

// 提交
const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return
  submitting.value = true
  try {
    if (form.id) {
      await updateDNSProvider(form.id, {
        name: form.name,
        config: form.config,
        email: form.email,
        phone: form.phone,
        enabled: form.enabled
      })
      Message.success('保存成功')
      dialogVisible.value = false
      loadData()
    } else {
      await createDNSProvider({
        name: form.name,
        provider: form.provider,
        config: form.config,
        email: form.email,
        phone: form.phone,
        enabled: form.enabled
      })
      Message.success('创建成功')
      dialogVisible.value = false
      loadData()
    }
  } catch (error: any) {
    // 错误已由 request 拦截器处理并显示
  } finally {
    submitting.value = false
  }
}

// 测试连接
const handleTest = async (row: any) => {
  try {
    row.testing = true
    await testDNSProvider(row.id)
    Message.success('连接测试成功')
    loadData()
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    row.testing = false
  }
}

// 删除
const handleDelete = (row: any) => {
  Modal.warning({
    title: '提示',
    content: '确定要删除该DNS配置吗？',
    hideCancel: false,
    onOk: async () => {
      loading.value = true
      try {
        await deleteDNSProvider(row.id)
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
})
</script>

<style scoped>
.dns-provider-container { padding: 0; background-color: transparent; }

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

.table-wrapper {
  background: #fff; border-radius: var(--ops-border-radius-md, 8px);
  box-shadow: 0 2px 12px rgba(0,0,0,0.04); overflow: hidden; padding: 16px;
}
.action-buttons { display: flex; gap: 4px; align-items: center; justify-content: center; }
.action-btn { width: 32px; height: 32px; border-radius: 6px; transition: all 0.2s ease; }
.action-btn:hover { transform: scale(1.1); }
.action-test:hover { background-color: #e8ffea; color: var(--ops-success, #00b42a); }
.action-edit:hover { background-color: var(--ops-primary-bg, #e8f0ff); color: var(--ops-primary, #165dff); }
.action-delete:hover { background-color: #ffece8; color: var(--ops-danger, #f53f3f); }
</style>
