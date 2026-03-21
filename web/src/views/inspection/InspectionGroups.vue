<template>
  <div class="inspection-groups-container">
    <a-card class="search-card">
      <a-form :model="searchForm" layout="inline">
        <a-form-item label="巡检组名称">
          <a-input v-model="searchForm.name" placeholder="请输入巡检组名称" style="width: 200px" @press-enter="handleSearch" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model="searchForm.status" placeholder="请选择状态" style="width: 120px" allow-clear>
            <a-option value="enabled">启用</a-option>
            <a-option value="disabled">禁用</a-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="handleSearch">查询</a-button>
            <a-button @click="handleReset">重置</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card class="table-card">
      <template #title>
        <a-space>
          <span>巡检组列表</span>
          <a-button type="primary" @click="handleAdd" v-permission="'inspection_groups:create'">
            <template #icon><icon-plus /></template>
            新增巡检组
          </a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        @page-change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      >
        <template #status="{ record }">
          <a-tag :color="record.status === 'enabled' ? 'green' : 'red'">
            {{ record.status === 'enabled' ? '启用' : '禁用' }}
          </a-tag>
        </template>

        <template #executionMode="{ record }">
          <a-tag>{{ getExecutionModeText(record.executionMode) }}</a-tag>
        </template>

        <template #labels="{ record }">
          <template v-if="parseLabels(record.labels).length">
            <a-tag v-for="label in parseLabels(record.labels)" :key="label" color="arcoblue" size="small" style="margin: 2px;">{{ label }}</a-tag>
          </template>
          <span v-else style="color: #86909c;">-</span>
        </template>

        <template #operations="{ record }">
          <a-space>
            <a-button type="text" size="small" @click="handleEdit(record)" v-permission="'inspection_groups:update'">
              编辑
            </a-button>
            <a-button type="text" size="small" status="danger" @click="handleDelete(record)" v-permission="'inspection_groups:delete'">
              删除
            </a-button>
          </a-space>
        </template>
      </a-table>
    </a-card>

    <!-- 新增/编辑弹窗 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="modalTitle"
      width="640px"
      :confirm-loading="submitting"
      :body-style="{ maxHeight: '70vh', overflowY: 'auto' }"
      @ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form :model="formData" :rules="formRules" ref="formRef" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="巡检组名称" field="name">
              <a-input v-model="formData.name" placeholder="请输入巡检组名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态" field="status">
              <a-radio-group v-model="formData.status">
                <a-radio value="enabled">启用</a-radio>
                <a-radio value="disabled">禁用</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="描述" field="description">
          <a-textarea v-model="formData.description" placeholder="请输入描述" :rows="2" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="执行方式" field="executionMode">
              <a-select v-model="formData.executionMode" placeholder="请选择执行方式">
                <a-option value="auto">自动（优先Agent）</a-option>
                <a-option value="agent">仅Agent</a-option>
                <a-option value="ssh">仅SSH</a-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="排序" field="sort">
              <a-input-number v-model="formData.sort" :min="0" style="width: 100%" placeholder="数字越小越靠前" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="自定义标签">
          <div class="label-input-area">
            <div class="label-tags">
              <a-tag
                v-for="(label, idx) in labelList"
                :key="label + idx"
                closable
                color="arcoblue"
                style="margin: 3px;"
                @close="() => removeLabel(idx)"
              >{{ label }}</a-tag>
            </div>
            <div class="label-add-row">
              <template v-if="labelInputVisible">
                <a-input
                  ref="labelInputRef"
                  v-model="labelInputValue"
                  size="small"
                  style="width: 150px"
                  placeholder="如 env:prod"
                  @blur="confirmLabelInput"
                  @press-enter="confirmLabelInput"
                />
                <a-button size="small" type="primary" @click="confirmLabelInput">确认</a-button>
                <a-button size="small" @click="cancelLabelInput">取消</a-button>
              </template>
              <a-button v-else size="small" @click="showLabelInput">
                <template #icon><icon-plus /></template>
                添加标签
              </a-button>
            </div>
            <div class="label-hint">标签格式：key:value，如 env:prod、team:ops，用于 metric 指标标识</div>
          </div>
        </a-form-item>

        <a-form-item label="Prometheus地址" field="prometheusUrl">
          <a-input v-model="formData.prometheusUrl" placeholder="http://prometheus:9090" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="Prometheus用户名" field="prometheusUsername">
              <a-input v-model="formData.prometheusUsername" placeholder="可选" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="Prometheus密码" field="prometheusPassword">
              <a-input-password v-model="formData.prometheusPassword" placeholder="可选" />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, nextTick, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'
import {
  getInspectionGroups,
  createInspectionGroup,
  updateInspectionGroup,
  deleteInspectionGroup
} from '@/api/inspectionManagement'

const loading = ref(false)
const submitting = ref(false)
const tableData = ref<any[]>([])
const modalVisible = ref(false)
const modalTitle = ref('新增巡检组')
const formRef = ref()
const editingId = ref<number | null>(null)

// 标签输入
const labelList = ref<string[]>([])
const labelInputVisible = ref(false)
const labelInputValue = ref('')
const labelInputRef = ref()

const searchForm = reactive({
  name: '',
  status: ''
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: true,
  showPageSize: true
})

const formData = reactive({
  name: '',
  description: '',
  status: 'enabled',
  executionMode: 'auto',
  prometheusUrl: '',
  prometheusUsername: '',
  prometheusPassword: '',
  sort: 0
})

const formRules = {
  name: [{ required: true, message: '请输入巡检组名称' }],
  executionMode: [{ required: true, message: '请选择执行方式' }]
}

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '巡检组名称', dataIndex: 'name' },
  { title: '描述', dataIndex: 'description', ellipsis: true, tooltip: true },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '执行方式', slotName: 'executionMode', width: 120 },
  { title: '自定义标签', slotName: 'labels', width: 200 },
  { title: '排序', dataIndex: 'sort', width: 80 },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
  { title: '操作', slotName: 'operations', width: 150, fixed: 'right' }
]

const getExecutionModeText = (mode: string) => {
  const map: Record<string, string> = {
    auto: '自动',
    agent: '仅Agent',
    ssh: '仅SSH'
  }
  return map[mode] || mode
}

const parseLabels = (labels: string): string[] => {
  if (!labels) return []
  try {
    return JSON.parse(labels)
  } catch {
    return []
  }
}

// 标签操作
const showLabelInput = () => {
  labelInputVisible.value = true
  nextTick(() => labelInputRef.value?.focus())
}

const confirmLabelInput = () => {
  const val = labelInputValue.value.trim()
  if (val && !labelList.value.includes(val)) {
    labelList.value.push(val)
  }
  labelInputVisible.value = false
  labelInputValue.value = ''
}

const cancelLabelInput = () => {
  labelInputVisible.value = false
  labelInputValue.value = ''
}

const removeLabel = (idx: number) => {
  labelList.value.splice(idx, 1)
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getInspectionGroups({
      keyword: searchForm.name,
      status: searchForm.status || undefined,
      page: pagination.current,
      pageSize: pagination.pageSize
    }) as any
    tableData.value = res?.list || res?.data?.list || []
    pagination.total = res?.total || res?.data?.total || 0
  } catch {
    Message.error('加载失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.current = 1
  fetchData()
}

const handleReset = () => {
  searchForm.name = ''
  searchForm.status = ''
  pagination.current = 1
  fetchData()
}

const handlePageChange = (page: number) => {
  pagination.current = page
  fetchData()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.current = 1
  fetchData()
}

const handleAdd = () => {
  editingId.value = null
  modalTitle.value = '新增巡检组'
  Object.assign(formData, {
    name: '', description: '', status: 'enabled',
    executionMode: 'auto', prometheusUrl: '', prometheusUsername: '', prometheusPassword: '', sort: 0
  })
  labelList.value = []
  modalVisible.value = true
}

const handleEdit = (record: any) => {
  editingId.value = record.id
  modalTitle.value = '编辑巡检组'
  Object.assign(formData, {
    name: record.name,
    description: record.description || '',
    status: record.status || 'enabled',
    executionMode: record.executionMode || 'auto',
    prometheusUrl: record.prometheusUrl || '',
    prometheusUsername: record.prometheusUsername || '',
    prometheusPassword: '',
    sort: record.sort || 0
  })
  labelList.value = parseLabels(record.labels)
  modalVisible.value = true
}

const handleDelete = (record: any) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除巡检组「${record.name}」吗？此操作不可撤销。`,
    onOk: async () => {
      try {
        await deleteInspectionGroup(record.id)
        Message.success('删除成功')
        fetchData()
      } catch {
        Message.error('删除失败')
      }
    }
  })
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate()
  if (valid) return
  submitting.value = true
  try {
    const payload: any = {
      ...formData,
      labels: JSON.stringify(labelList.value)
    }
    if (editingId.value) {
      await updateInspectionGroup(editingId.value, payload)
      Message.success('更新成功')
    } else {
      await createInspectionGroup(payload)
      Message.success('创建成功')
    }
    modalVisible.value = false
    fetchData()
  } catch (e: any) {
    Message.error(e?.response?.data?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleCancel = () => {
  modalVisible.value = false
  formRef.value?.resetFields()
  labelList.value = []
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.inspection-groups-container {
  padding: 20px;
}

.search-card {
  margin-bottom: 16px;
}

.table-card {
  background: #fff;
}

.label-input-area {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 4px;
  background: #fafafa;
}

.label-tags {
  display: flex;
  flex-wrap: wrap;
  min-height: 24px;
  margin-bottom: 6px;
}

.label-tags:empty {
  margin-bottom: 0;
}

.label-add-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.label-hint {
  margin-top: 6px;
  font-size: 12px;
  color: #86909c;
}
</style>
