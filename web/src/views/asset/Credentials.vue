<template>
  <div class="credentials-page-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-lock /></div>
        <div>
          <h2 class="page-title">凭证管理</h2>
          <p class="page-subtitle">管理SSH认证凭证，支持密码和密钥两种认证方式</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button v-permission="'credentials:create'" type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          新建凭证
        </a-button>
      </div>
    </div>

    <!-- 搜索和筛选栏 -->
    <div class="filter-bar">
      <div class="filter-inputs">
        <a-input
          v-model="searchForm.keyword"
          placeholder="搜索凭证名称..."
          allow-clear
          style="width: 300px;"
          @press-enter="handleSearch"
          @clear="handleSearch"
        >
          <template #prefix>
            <icon-search />
          </template>
        </a-input>

        <a-select
          v-model="searchForm.type"
          placeholder="认证方式"
          allow-clear
          style="width: 220px;"
          @change="handleSearch"
        >
          <a-option label="密码认证" value="password" />
          <a-option label="密钥认证" value="key" />
        </a-select>
      </div>

      <div class="filter-actions">
        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
        <a-button @click="loadCredentialList">
          <template #icon><icon-refresh /></template>
          刷新
        </a-button>
      </div>
    </div>

    <!-- 凭证列表 -->
    <div class="table-wrapper">
      <a-table
        :data="credentialList"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        :pagination="{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showTotal: true,
          showPageSize: true,
          pageSizeOptions: [10, 20, 50, 100]
        }"
        @page-change="(p: number) => { pagination.page = p; loadCredentialList() }"
        @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadCredentialList() }"
      >
        <template #columns>
          <a-table-column title="凭证名称" data-index="name" :min-width="180">
            <template #cell="{ record }">
              <div class="name-cell">
                <icon-safe v-if="record.type === 'key'" :style="{ color: '#67c23a', fontSize: '20px' }" />
                <icon-lock v-else :style="{ color: '#e6a23c', fontSize: '20px' }" />
                <span class="name">{{ record.name }}</span>
              </div>
            </template>
          </a-table-column>

          <a-table-column title="认证方式" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.type === 'password' ? 'orangered' : 'green'">
                {{ record.typeText }}
              </a-tag>
            </template>
          </a-table-column>

          <a-table-column title="用户名" data-index="username" :min-width="120">
            <template #cell="{ record }">
              <span v-if="record.username">{{ record.username }}</span>
              <span v-else class="text-muted">-</span>
            </template>
          </a-table-column>

          <a-table-column title="使用情况" :min-width="150">
            <template #cell="{ record }">
              <div class="usage-cell">
                <span class="usage-count">{{ record.hostCount || 0 }} 台主机</span>
              </div>
            </template>
          </a-table-column>

          <a-table-column title="描述" data-index="description" :min-width="200" ellipsis tooltip>
            <template #cell="{ record }">
              <span v-if="record.description">{{ record.description }}</span>
              <span v-else class="text-muted">-</span>
            </template>
          </a-table-column>

          <a-table-column title="创建时间" data-index="createTime" :width="180" />

          <a-table-column title="操作" :width="180" fixed="right" align="center">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="编辑" position="top">
                  <a-button v-permission="'credentials:update'" type="text" size="small" class="action-btn action-edit" @click="handleEdit(record)">
                    <template #icon><icon-edit /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="删除" position="top">
                  <a-button v-permission="'credentials:delete'" type="text" size="small" class="action-btn action-delete" @click="handleDelete(record)" :disabled="record.hostCount > 0">
                    <template #icon><icon-delete /></template>
                  </a-button>
                </a-tooltip>
              </div>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 新增/编辑凭证对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="isEdit ? '编辑凭证' : '新建凭证'"
      :width="600"
      unmount-on-close
      @close="handleDialogClose"
    >
      <a-alert v-if="isEdit" type="info" :closable="false" style="margin-bottom: 20px;">
        出于安全考虑，私钥和密码不会在编辑时显示。如需修改，请重新填写。留空则保持原值不变。
      </a-alert>

      <a-form ref="formRef" :model="form" :rules="rules" layout="horizontal" auto-label-width>
        <a-form-item label="凭证名称" field="name">
          <a-input v-model="form.name" placeholder="请输入凭证名称，如：生产环境root凭证" />
        </a-form-item>

        <a-form-item label="认证方式" field="type">
          <a-radio-group v-model="form.type" @change="handleAuthTypeChange">
            <a-radio value="password">密码认证</a-radio>
            <a-radio value="key">密钥认证</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="form.type === 'password'" label="用户名">
          <a-input v-model="form.username" placeholder="如：root" />
        </a-form-item>

        <a-form-item v-if="form.type === 'password'" label="密码" field="password">
          <a-input-password v-model="form.password" :placeholder="isEdit ? '如需修改密码请在此填写，留空则保持不变' : '请输入密码'" />
        </a-form-item>

        <a-form-item v-if="form.type === 'key'" label="用户名">
          <a-input v-model="form.username" placeholder="如：root（可选）" />
        </a-form-item>

        <a-form-item v-if="form.type === 'key'" label="私钥" field="privateKey">
          <a-textarea
            v-model="form.privateKey"
            :auto-size="{ minRows: 8 }"
            :placeholder="isEdit ? '如需修改私钥请在此填写，留空则保持不变' : '请粘贴PEM格式的私钥内容'"
          />
        </a-form-item>

        <a-form-item v-if="form.type === 'key'" label="私钥密码">
          <a-input-password v-model="form.passphrase" placeholder="如果私钥有密码请输入（可选）" />
        </a-form-item>

        <a-form-item label="备注">
          <a-textarea v-model="form.description" :auto-size="{ minRows: 2 }" placeholder="请输入备注信息" />
        </a-form-item>
      </a-form>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="dialogVisible = false">取消</a-button>
          <a-button type="primary" :loading="submitting" @click="handleSubmit">{{ isEdit ? '保存' : '确定' }}</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconSearch,
  IconRefresh,
  IconLock,
  IconSafe
} from '@arco-design/web-vue/es/icon'
import {
  getCredentialList,
  getCredential,
  createCredential,
  updateCredential,
  deleteCredential
} from '@/api/host'

// 加载状态
const loading = ref(false)
const submitting = ref(false)

// 对话框状态
const dialogVisible = ref(false)
const isEdit = ref(false)

// 表单引用
const formRef = ref<FormInstance>()

// 搜索表单
const searchForm = reactive({
  keyword: '',
  type: undefined as string | undefined
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 凭证列表
const credentialList = ref([])

// 表单
const form = reactive({
  id: 0,
  name: '',
  type: 'password',
  username: '',
  password: '',
  privateKey: '',
  passphrase: '',
  description: ''
})

// 表单验证规则
const rules = {
  name: [{ required: true, message: '请输入凭证名称' }],
  type: [{ required: true, message: '请选择认证方式' }],
  password: [{ required: true, message: '请输入密码' }],
  privateKey: [{ required: true, message: '请输入私钥' }]
}

// 加载凭证列表
const loadCredentialList = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword || undefined
    }
    if (searchForm.type !== undefined) {
      params.type = searchForm.type
    }

    const res = await getCredentialList(params)
    credentialList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    Message.error('获取凭证列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadCredentialList()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.type = undefined
  loadCredentialList()
}

// 新增凭证
const handleAdd = () => {
  Object.assign(form, {
    id: 0,
    name: '',
    type: 'password',
    username: '',
    password: '',
    privateKey: '',
    passphrase: '',
    description: ''
  })
  isEdit.value = false
  dialogVisible.value = true
}

// 编辑凭证
const handleEdit = async (row: any) => {
  try {
    const credential = await getCredential(row.id)

    Object.assign(form, {
      id: credential.id,
      name: credential.name,
      type: credential.type,
      username: credential.username || '',
      password: credential.password || '',
      privateKey: credential.privateKey || '',
      passphrase: credential.passphrase || '',
      description: credential.description || ''
    })
  } catch (error) {
    Message.error('获取凭证详情失败')
    return
  }

  isEdit.value = true
  dialogVisible.value = true
}

// 删除凭证
const handleDelete = (row: any) => {
  if (row.hostCount > 0) {
    Message.warning('该凭证正在被使用，无法删除')
    return
  }

  Modal.warning({
    title: '提示',
    content: `确定要删除凭证"${row.name}"吗？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await deleteCredential(row.id)
        Message.success('删除成功')
        loadCredentialList()
      } catch (error: any) {
        Message.error(error.message || '删除失败')
      }
    }
  })
}

// 认证方式变化
const handleAuthTypeChange = (type: string | number | boolean) => {
  if (type === 'password') {
    form.privateKey = ''
    form.passphrase = ''
  } else {
    form.password = ''
  }
}

// 对话框关闭
const handleDialogClose = () => {
  formRef.value?.resetFields()
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return

  submitting.value = true
  try {
    if (isEdit.value) {
      const updateData: any = {
        id: form.id,
        name: form.name,
        type: form.type,
        username: form.username,
        description: form.description
      }
      if (form.password) {
        updateData.password = form.password
      }
      if (form.privateKey) {
        updateData.privateKey = form.privateKey
      }
      if (form.passphrase) {
        updateData.passphrase = form.passphrase
      }
      await updateCredential(form.id, updateData)
      Message.success('更新成功')
    } else {
      await createCredential(form)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadCredentialList()
  } catch (error: any) {
    Message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadCredentialList()
})
</script>

<style scoped>
.credentials-page-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
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
  flex-shrink: 0;
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 36px;
  height: 36px;
  background: var(--ops-primary);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  flex-shrink: 0;
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

/* 筛选栏 */
.filter-bar {
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.filter-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.filter-actions {
  display: flex;
  gap: 10px;
}

/* 表格 */
.table-wrapper {
  flex: 1;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name {
  font-weight: 500;
  color: #303133;
}

.usage-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.usage-count {
  font-size: 13px;
  color: #606266;
}

.text-muted {
  color: #c0c4cc;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.action-btn:disabled:hover {
  transform: none;
  background-color: transparent;
  color: inherit;
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

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
