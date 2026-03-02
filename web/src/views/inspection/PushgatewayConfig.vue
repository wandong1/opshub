<template>
  <div class="pushgateway-config-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-cloud /></div>
        <div>
          <h2 class="page-title">Pushgateway 配置</h2>
          <p class="page-subtitle">管理 Pushgateway 实例，用于拨测指标推送</p>
        </div>
      </div>
      <a-button v-permission="'inspection:pushgateways:create'" type="primary" @click="handleAdd"><template #icon><icon-plus /></template>新增Pushgateway</a-button>
    </div>

    <a-table :data="tableData" :loading="loading" :bordered="{ cell: true }" stripe>
      <template #columns>
        <a-table-column title="名称" data-index="name" :width="140" />
        <a-table-column title="URL" data-index="url" :width="240" ellipsis tooltip />
        <a-table-column title="用户名" :width="120">
          <template #cell="{ record }">{{ record.username || '-' }}</template>
        </a-table-column>
        <a-table-column title="默认" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.isDefault === 1 ? 'orangered' : 'gray'">{{ record.isDefault === 1 ? '是' : '否' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="状态" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.status === 1 ? 'green' : 'red'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="200" fixed="right" align="center">
          <template #cell="{ record }">
            <a-button v-permission="'inspection:pushgateways:update'" type="text" size="small" @click="handleEdit(record)">编辑</a-button>
            <a-button v-permission="'inspection:pushgateways:delete'" type="text" size="small" status="danger" @click="handleDelete(record)">删除</a-button>
            <a-button v-permission="'inspection:pushgateways:test'" type="text" size="small" status="success" :loading="record._testing" @click="handleTest(record)">测试</a-button>
          </template>
        </a-table-column>
      </template>
    </a-table>
    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑 Pushgateway' : '新增 Pushgateway'" :width="500" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="名称" field="name"><a-input v-model="formData.name" placeholder="请输入名称" /></a-form-item>
        <a-form-item label="URL" field="url"><a-input v-model="formData.url" placeholder="http://pushgateway:9091" /></a-form-item>
        <a-form-item label="用户名"><a-input v-model="formData.username" placeholder="Basic Auth 用户名（可选）" /></a-form-item>
        <a-form-item label="密码"><a-input-password v-model="formData.password" placeholder="Basic Auth 密码（可选）" /></a-form-item>
        <a-form-item label="默认">
          <a-radio-group v-model="formData.isDefault"><a-radio :value="1">是</a-radio><a-radio :value="0">否</a-radio></a-radio-group>
        </a-form-item>
        <a-form-item label="状态">
          <a-radio-group v-model="formData.status"><a-radio :value="1">启用</a-radio><a-radio :value="0">禁用</a-radio></a-radio-group>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" :loading="submitting" @click="handleSubmit">确定</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconCloud, IconPlus } from '@arco-design/web-vue/es/icon'
import { getPushgatewayList, createPushgateway, updatePushgateway, deletePushgateway, testPushgateway } from '@/api/networkProbe'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<any[]>([])

const defaultForm = () => ({ id: 0, name: '', url: '', username: '', password: '', isDefault: 0, status: 1 })
const formData = reactive(defaultForm())
const formRules = { name: [{ required: true, message: '请输入名称' }], url: [{ required: true, message: '请输入URL' }] }

const loadData = async () => { loading.value = true; try { tableData.value = (await getPushgatewayList()) || [] } catch {} finally { loading.value = false } }
const handleAdd = () => { isEdit.value = false; Object.assign(formData, defaultForm()); dialogVisible.value = true }
const handleEdit = (row: any) => { isEdit.value = true; Object.assign(formData, { ...row }); dialogVisible.value = true }

const handleDelete = (row: any) => {
  Modal.warning({ title: '提示', content: '确定删除？', hideCancel: false, onOk: async () => { await deletePushgateway(row.id); Message.success('删除成功'); loadData() } })
}

const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return
  submitting.value = true
  try {
    if (isEdit.value) { await updatePushgateway(formData.id, formData); Message.success('更新成功') }
    else { await createPushgateway(formData); Message.success('创建成功') }
    dialogVisible.value = false; loadData()
  } catch {} finally { submitting.value = false }
}

const handleTest = async (row: any) => {
  row._testing = true
  try { await testPushgateway(row.id); Message.success('连接成功') } catch {} finally { row._testing = false }
}

onMounted(() => { loadData() })
</script>

<style scoped>
.pushgateway-config-container { padding: 0; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 36px; height: 36px; border-radius: 8px; background: var(--ops-primary); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 18px; }
.page-title { margin: 0; font-size: 17px; font-weight: 600; color: var(--ops-text-primary); }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: var(--ops-text-tertiary); }
</style>
