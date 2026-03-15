<template>
  <div class="inspection-items-container">
    <a-card class="search-card">
      <a-form :model="searchForm" layout="inline">
        <a-form-item label="巡检项名称">
          <a-input v-model="searchForm.name" placeholder="请输入巡检项名称" style="width: 200px" />
        </a-form-item>
        <a-form-item label="巡检组">
          <a-select v-model="searchForm.groupId" placeholder="请选择巡检组" style="width: 200px" allow-clear>
            <a-option v-for="group in groupList" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-option>
          </a-select>
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
          <span>巡检项列表</span>
          <a-button type="primary" @click="handleAdd" v-permission="'inspection_items:create'">
            <template #icon><icon-plus /></template>
            新增巡检项
          </a-button>
          <a-button type="outline" @click="handleTestRun" v-permission="'inspection_items:test'" :disabled="selectedKeys.length === 0">
            <template #icon><icon-play-arrow /></template>
            测试运行
          </a-button>
        </a-space>
      </template>

      <a-table
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        :row-selection="{ type: 'checkbox', showCheckedAll: true }"
        v-model:selected-keys="selectedKeys"
        @page-change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      >
        <template #executionType="{ record }">
          <a-tag>{{ getExecutionTypeText(record.executionType) }}</a-tag>
        </template>

        <template #status="{ record }">
          <a-tag :color="record.status === 'enabled' ? 'green' : 'red'">
            {{ record.status === 'enabled' ? '启用' : '禁用' }}
          </a-tag>
        </template>

        <template #operations="{ record }">
          <a-space>
            <a-button type="text" size="small" @click="handleEdit(record)" v-permission="'inspection_items:update'">
              编辑
            </a-button>
            <a-button type="text" size="small" status="danger" @click="handleDelete(record)" v-permission="'inspection_items:delete'">
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
      width="800px"
      @ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form :model="formData" :rules="formRules" ref="formRef" layout="vertical">
        <a-form-item label="巡检项名称" field="name">
          <a-input v-model="formData.name" placeholder="请输入巡检项名称" />
        </a-form-item>

        <a-form-item label="所属巡检组" field="groupId">
          <a-select v-model="formData.groupId" placeholder="请选择巡检组">
            <a-option v-for="group in groupList" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-option>
          </a-select>
        </a-form-item>

        <a-form-item label="执行类型" field="executionType">
          <a-radio-group v-model="formData.executionType">
            <a-radio value="command">命令</a-radio>
            <a-radio value="script">脚本</a-radio>
            <a-radio value="promql">PromQL</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="formData.executionType === 'command'" label="命令" field="command">
          <a-textarea v-model="formData.command" placeholder="请输入要执行的命令" :rows="3" />
        </a-form-item>

        <a-form-item v-if="formData.executionType === 'script'" label="脚本类型" field="scriptType">
          <a-select v-model="formData.scriptType" placeholder="请选择脚本类型">
            <a-option value="shell">Shell</a-option>
            <a-option value="python">Python</a-option>
          </a-select>
        </a-form-item>

        <a-form-item v-if="formData.executionType === 'script'" label="脚本内容" field="scriptContent">
          <a-textarea v-model="formData.scriptContent" placeholder="请输入脚本内容" :rows="6" />
        </a-form-item>

        <a-form-item v-if="formData.executionType === 'promql'" label="PromQL查询" field="promqlQuery">
          <a-textarea v-model="formData.promqlQuery" placeholder="请输入PromQL查询语句" :rows="3" />
        </a-form-item>

        <a-form-item label="断言类型" field="assertionType">
          <a-select v-model="formData.assertionType" placeholder="请选择断言类型" allow-clear>
            <a-option value="gt">大于</a-option>
            <a-option value="gte">大于等于</a-option>
            <a-option value="lt">小于</a-option>
            <a-option value="lte">小于等于</a-option>
            <a-option value="eq">等于</a-option>
            <a-option value="contains">包含</a-option>
            <a-option value="not_contains">不包含</a-option>
            <a-option value="regex">正则匹配</a-option>
            <a-option value="not_regex">反正则匹配</a-option>
          </a-select>
        </a-form-item>

        <a-form-item v-if="formData.assertionType" label="断言值" field="assertionValue">
          <a-input v-model="formData.assertionValue" placeholder="请输入断言值" />
        </a-form-item>

        <a-form-item label="描述" field="description">
          <a-textarea v-model="formData.description" placeholder="请输入描述" :rows="2" />
        </a-form-item>

        <a-form-item label="超时时间(秒)" field="timeout">
          <a-input-number v-model="formData.timeout" :min="1" :max="600" placeholder="默认60秒" />
        </a-form-item>

        <a-form-item label="状态" field="status">
          <a-radio-group v-model="formData.status">
            <a-radio value="enabled">启用</a-radio>
            <a-radio value="disabled">禁用</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconPlayArrow } from '@arco-design/web-vue/es/icon'

const loading = ref(false)
const tableData = ref([])
const groupList = ref([])
const modalVisible = ref(false)
const modalTitle = ref('新增巡检项')
const formRef = ref()
const selectedKeys = ref([])

const searchForm = reactive({
  name: '',
  groupId: undefined,
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
  groupId: undefined,
  executionType: 'command',
  command: '',
  scriptType: 'shell',
  scriptContent: '',
  promqlQuery: '',
  assertionType: '',
  assertionValue: '',
  timeout: 60,
  status: 'enabled'
})

const formRules = {
  name: [{ required: true, message: '请输入巡检项名称' }],
  groupId: [{ required: true, message: '请选择巡检组' }],
  executionType: [{ required: true, message: '请选择执行类型' }]
}

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '巡检项名称', dataIndex: 'name' },
  { title: '执行类型', slotName: 'executionType', width: 100 },
  { title: '描述', dataIndex: 'description', ellipsis: true, tooltip: true },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '排序', dataIndex: 'sort', width: 80 },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
  { title: '操作', slotName: 'operations', width: 150, fixed: 'right' }
]

const getExecutionTypeText = (type: string) => {
  const map: Record<string, string> = {
    command: '命令',
    script: '脚本',
    promql: 'PromQL'
  }
  return map[type] || type
}

const fetchGroups = async () => {
  try {
    // TODO: 调用API获取巡检组列表
    groupList.value = []
  } catch (error) {
    Message.error('获取巡检组列表失败')
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    // TODO: 调用API获取数据
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
  searchForm.groupId = undefined
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
  modalTitle.value = '新增巡检项'
  Object.assign(formData, {
    id: 0,
    name: '',
    description: '',
    groupId: undefined,
    executionType: 'command',
    command: '',
    scriptType: 'shell',
    scriptContent: '',
    promqlQuery: '',
    assertionType: '',
    assertionValue: '',
    timeout: 60,
    status: 'enabled'
  })
  modalVisible.value = true
}

const handleEdit = (record: any) => {
  modalTitle.value = '编辑巡检项'
  Object.assign(formData, record)
  modalVisible.value = true
}

const handleDelete = async (record: any) => {
  // TODO: 调用删除API
  Message.info('删除功能待实现')
}

const handleTestRun = async () => {
  // TODO: 调用测试运行API
  Message.info('测试运行功能待实现')
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
  fetchGroups()
  fetchData()
})
</script>

<style scoped>
.inspection-items-container {
  padding: 20px;
}

.search-card {
  margin-bottom: 16px;
}

.table-card {
  background: #fff;
}
</style>
