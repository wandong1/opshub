<template>
  <div class="variable-management-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-code /></div>
        <div>
          <h2 class="page-title">环境变量</h2>
          <p class="page-subtitle">管理全局环境变量，支持在拨测配置中通过 {{ '\{\{name\}\}' }} 引用</p>
        </div>
      </div>
      <a-button type="primary" @click="handleAdd"><template #icon><icon-plus /></template>新增变量</a-button>
    </div>

    <div class="search-bar" style="margin-bottom: 16px;">
      <a-form :model="{}" layout="inline">
        <a-form-item><a-input v-model="searchKeyword" placeholder="搜索名称/描述" allow-clear @change="loadData" style="width: 200px" /></a-form-item>
        <a-form-item>
          <a-select v-model="searchVarType" placeholder="类型" allow-clear @change="loadData" style="width: 120px">
            <a-option value="plain">明文</a-option>
            <a-option value="secret">密钥</a-option>
          </a-select>
        </a-form-item>
      </a-form>
    </div>

    <a-table :data="tableData" :loading="loading" :bordered="{ cell: true }" stripe :pagination="pagination" @page-change="onPageChange">
      <template #columns>
        <a-table-column title="名称" data-index="name" :width="160">
          <template #cell="{ record }"><a-tag size="small" color="blue">{{ record.name }}</a-tag></template>
        </a-table-column>
        <a-table-column title="类型" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.varType === 'secret' ? 'orangered' : 'arcoblue'">{{ record.varType === 'secret' ? '密钥' : '明文' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="值" data-index="value" :width="200" ellipsis tooltip />
        <a-table-column title="业务分组" data-index="groupIds" :width="160">
          <template #cell="{ record }">{{ getGroupNames(record.groupIds) }}</template>
        </a-table-column>
        <a-table-column title="描述" data-index="description" :width="200" ellipsis tooltip>
          <template #cell="{ record }">{{ record.description || '-' }}</template>
        </a-table-column>
        <a-table-column title="引用语法" :width="160">
          <template #cell="{ record }">
            <span class="var-pill-display"><svg viewBox="0 0 48 48" width="12" height="12" fill="currentColor" style="flex-shrink:0"><path d="M16 4l-8 22h10l-4 18 24-26H26l6-14z"/></svg>{{ '\{\{' + record.name + '\}\}' }}</span>
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="160" fixed="right" align="center">
          <template #cell="{ record }">
            <a-button type="text" size="small" @click="handleEdit(record)">编辑</a-button>
            <a-button type="text" size="small" status="danger" @click="handleDelete(record)">删除</a-button>
          </template>
        </a-table-column>
      </template>
    </a-table>

    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑变量' : '新增变量'" :width="520" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="变量名" field="name"><a-input v-model="formData.name" placeholder="如 base_url（仅字母数字下划线）" /></a-form-item>
        <a-form-item label="类型" field="varType">
          <a-radio-group v-model="formData.varType">
            <a-radio value="plain">明文</a-radio>
            <a-radio value="secret">密钥</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="值" field="value">
          <a-input-password v-if="formData.varType === 'secret'" v-model="formData.value" :placeholder="isEdit ? '留空则不修改' : '请输入值'" />
          <a-input v-else v-model="formData.value" placeholder="请输入值" />
        </a-form-item>
        <a-form-item label="业务分组">
          <a-select v-model="selectedGroupIds" multiple placeholder="选择业务分组（留空则不限）" allow-clear allow-search style="width: 100%;">
            <a-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="描述"><a-textarea v-model="formData.description" placeholder="变量用途说明" :max-length="500" /></a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" :loading="submitting" @click="handleSubmit">确定</a-button>
      </template>
    </a-modal>
  </div>
</template>


<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconCode, IconPlus } from '@arco-design/web-vue/es/icon'
import { getVariableList, createVariable, updateVariable, deleteVariable } from '@/api/networkProbe'
import { getGroupTree } from '@/api/assetGroup'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<any[]>([])
const searchKeyword = ref('')
const searchVarType = ref('')
const pagination = reactive({ current: 1, pageSize: 15, total: 0 })
const groupOptions = ref<any[]>([])

const flattenGroups = (tree: any[], result: any[] = []): any[] => {
  for (const node of tree) { result.push({ id: node.id, name: node.name }); if (node.children?.length) flattenGroups(node.children, result) }
  return result
}
const loadGroups = async () => {
  try { const res = await getGroupTree(); groupOptions.value = flattenGroups(res.data || res || []) } catch {}
}
const groupNameMap = computed(() => {
  const m: Record<number, string> = {}
  for (const g of groupOptions.value) m[g.id] = g.name
  return m
})
const getGroupNames = (groupIds: string) => {
  if (!groupIds) return '不限'
  return groupIds.split(',').filter(Boolean).map(id => groupNameMap.value[Number(id)] || `#${id}`).join(', ')
}

const defaultForm = () => ({ id: 0, name: '', value: '', varType: 'plain', groupIds: '', description: '' })
const formData = reactive(defaultForm())
const formRules = {
  name: [{ required: true, message: '请输入变量名' }],
  value: [{ required: true, message: '请输入值' }],
  varType: [{ required: true, message: '请选择类型' }],
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await getVariableList({ page: pagination.current, page_size: pagination.pageSize, keyword: searchKeyword.value, var_type: searchVarType.value })
    tableData.value = res?.list || res?.data || []
    pagination.total = res?.total || 0
  } catch {} finally { loading.value = false }
}

const onPageChange = (page: number) => { pagination.current = page; loadData() }

const selectedGroupIds = computed({
  get: () => formData.groupIds ? formData.groupIds.split(',').filter(Boolean).map(Number) : [],
  set: (val: number[]) => { formData.groupIds = val.join(',') }
})

const handleAdd = () => { isEdit.value = false; Object.assign(formData, defaultForm()); loadGroups(); dialogVisible.value = true }
const handleEdit = (row: any) => {
  isEdit.value = true
  Object.assign(formData, { ...row, value: row.varType === 'secret' ? '' : row.value })
  loadGroups(); dialogVisible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate()
  if (valid) return
  submitting.value = true
  try {
    const data = { ...formData }
    if (isEdit.value && data.varType === 'secret' && !data.value) {
      // Don't send empty value for secret edit
      delete (data as any).value
    }
    if (isEdit.value) { await updateVariable(data.id, data) } else { await createVariable(data) }
    Message.success(isEdit.value ? '更新成功' : '创建成功')
    dialogVisible.value = false
    loadData()
  } catch {} finally { submitting.value = false }
}

const handleDelete = (row: any) => {
  Modal.warning({
    title: '确认删除',
    content: `确定删除变量 "${row.name}" 吗？引用该变量的拨测配置将无法解析。`,
    hideCancel: false,
    onOk: async () => { await deleteVariable(row.id); Message.success('删除成功'); loadData() }
  })
}

onMounted(() => { loadData(); loadGroups() })
</script>

<style scoped>
.variable-management-container { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #165dff 0%, #722ed1 100%); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 20px; }
.page-title { margin: 0; font-size: 18px; font-weight: 600; color: var(--ops-text-primary); }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: var(--ops-text-tertiary); }
code { background: #f2f3f5; padding: 2px 6px; border-radius: 3px; font-size: 12px; color: #165dff; }
.var-pill-display {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  background: #e8f3ff;
  color: #165dff;
  border-radius: 4px;
  padding: 0 8px;
  font-size: 12px;
  line-height: 22px;
  font-weight: 500;
  white-space: nowrap;
}
</style>
