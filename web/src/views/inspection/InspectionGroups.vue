<template>
  <div class="inspection-groups-container">
    <a-card class="search-card">
      <a-form :model="searchForm" layout="inline">
        <a-form-item label="巡检组名称">
          <a-input v-model="searchForm.name" placeholder="请输入巡检组名称" style="width: 200px" />
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
      width="600px"
      @ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form :model="formData" :rules="formRules" ref="formRef" layout="vertical">
        <a-form-item label="巡检组名称" field="name">
          <a-input v-model="formData.name" placeholder="请输入巡检组名称" />
        </a-form-item>

        <a-form-item label="描述" field="description">
          <a-textarea v-model="formData.description" placeholder="请输入描述" :rows="3" />
        </a-form-item>

        <a-form-item label="状态" field="status">
          <a-radio-group v-model="formData.status">
            <a-radio value="enabled">启用</a-radio>
            <a-radio value="disabled">禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item label="执行方式" field="executionMode">
          <a-select v-model="formData.executionMode" placeholder="请选择执行方式">
            <a-option value="auto">自动（优先Agent）</a-option>
            <a-option value="agent">仅Agent</a-option>
            <a-option value="ssh">仅SSH</a-option>
          </a-select>
        </a-form-item>

        <a-form-item label="Prometheus地址" field="prometheusUrl">
          <a-input v-model="formData.prometheusUrl" placeholder="http://prometheus:9090" />
        </a-form-item>

        <a-form-item label="Prometheus用户名" field="prometheusUsername">
          <a-input v-model="formData.prometheusUsername" placeholder="可选" />
        </a-form-item>

        <a-form-item label="Prometheus密码" field="prometheusPassword">
          <a-input-password v-model="formData.prometheusPassword" placeholder="可选" />
        </a-form-item>

        <a-form-item label="排序" field="sort">
          <a-input-number v-model="formData.sort" :min="0" placeholder="数字越小越靠前" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'

const loading = ref(false)
const tableData = ref([])
const modalVisible = ref(false)
const modalTitle = ref('新增巡检组')
const formRef = ref()

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
  id: 0,
  name: '',
  description: '',
  status: 'enabled',
  executionMode: 'auto',
  prometheusUrl: '',
  prometheusUsername: '',
  prometheusPassword: '',
  sort: 0,
  groupIds: '[]'
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

const fetchData = async () => {
  loading.value = true
  try {
    // TODO: 调用API获取数据
    // const response = await getInspectionGroups({
    //   page: pagination.current,
    //   pageSize: pagination.pageSize,
    //   ...searchForm
    // })
    // tableData.value = response.data.list
    // pagination.total = response.data.total

    // 临时模拟数据
    tableData.value = []
    pagination.total = 0
    Message.info('API接口待实现')
  } catch (error) {
    Message.error('获取数据失败')
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
  handleSearch()
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
  modalTitle.value = '新增巡检组'
  Object.assign(formData, {
    id: 0,
    name: '',
    description: '',
    status: 'enabled',
    executionMode: 'auto',
    prometheusUrl: '',
    prometheusUsername: '',
    prometheusPassword: '',
    sort: 0,
    groupIds: '[]'
  })
  modalVisible.value = true
}

const handleEdit = (record: any) => {
  modalTitle.value = '编辑巡检组'
  Object.assign(formData, record)
  modalVisible.value = true
}

const handleDelete = async (record: any) => {
  // TODO: 调用删除API
  Message.info('删除功能待实现')
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate()
  if (!valid) {
    // TODO: 调用创建/更新API
    Message.info('提交功能待实现')
    modalVisible.value = false
  }
}

const handleCancel = () => {
  modalVisible.value = false
  formRef.value?.resetFields()
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
</style>
