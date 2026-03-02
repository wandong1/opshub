<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-file />
        </div>
        <div>
          <div class="page-title">模板管理</div>
          <div class="page-desc">创建和管理执行模板，支持模板复用和参数化</div>
        </div>
      </div>
    </div>

    <!-- 搜索区域 -->
    <a-card class="search-card" :bordered="false">
      <a-row :gutter="16" align="center">
        <a-col :flex="'auto'">
          <a-space :size="16" wrap>
            <a-space>
              <span class="search-label">模板类型:</span>
              <a-select v-model="searchForm.type" placeholder="请选择" allow-clear style="width: 180px" @change="handleSearch">
                <a-option v-for="t in templateTypes" :key="t.value" :value="t.value">{{ t.label }}</a-option>
              </a-select>
            </a-space>
            <a-space>
              <span class="search-label">模板名称:</span>
              <a-input v-model="searchForm.name" placeholder="请输入" allow-clear style="width: 240px" @press-enter="handleSearch" />
            </a-space>
            <a-button type="primary" @click="handleSearch">
              <template #icon><icon-search /></template>
              搜索
            </a-button>
            <a-button @click="handleReset">
              <template #icon><icon-refresh /></template>
              重置
            </a-button>
          </a-space>
        </a-col>
      </a-row>
    </a-card>

    <!-- 模板列表 -->
    <a-card class="table-card" :bordered="false">
      <template #title>
        <span class="card-title">模板列表</span>
      </template>
      <template #extra>
        <a-space>
          <a-button type="primary" @click="handleCreate">
            <template #icon><icon-plus /></template>
            新建
          </a-button>
          <a-button @click="loadTemplates">
            <template #icon><icon-refresh /></template>
          </a-button>
        </a-space>
      </template>

      <a-table
        :data="templates"
        :loading="loading"
        row-key="id"
        :pagination="tablePagination"
        @page-change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      >
        <template #columns>
          <a-table-column title="序号" :width="70" align="center">
            <template #cell="{ rowIndex }">
              {{ (pagination.page - 1) * pagination.pageSize + rowIndex + 1 }}
            </template>
          </a-table-column>
          <a-table-column title="模板名称" data-index="name" :min-width="200" />
          <a-table-column title="模板类型" data-index="category" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag size="small" color="arcoblue">{{ getCategoryLabel(record.category) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="模板内容" data-index="content" ellipsis tooltip :min-width="280" />
          <a-table-column title="描述信息" data-index="description" ellipsis tooltip :min-width="200" />
          <a-table-column title="操作" :width="120" align="center" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="handleEdit(record)">编辑</a-link>
                <a-popconfirm content="确定要删除该模板吗？" @ok="handleDelete(record)">
                  <a-link status="danger">删除</a-link>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 新建/编辑模板对话框 -->
    <a-modal
      v-model:visible="showTemplateDialog"
      :title="isEdit ? '编辑模板' : '新建模板'"
      :width="860"
      :unmount-on-close="true"
      @ok="handleSaveTemplate"
      @cancel="showTemplateDialog = false"
    >
      <a-form :model="templateForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="模板类型" required>
              <a-space style="width: 100%">
                <a-select v-model="templateForm.type" placeholder="请选择模板类型" style="flex: 1">
                  <a-option v-for="t in templateTypes" :key="t.value" :value="t.value">{{ t.label }}</a-option>
                </a-select>
                <a-link @click="showAddTypeDialog = true">添加类型</a-link>
              </a-space>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="模板名称" required>
              <a-input v-model="templateForm.name" placeholder="请输入模板名称" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="脚本语言" required>
          <a-radio-group v-model="templateForm.scriptType" type="button">
            <a-radio value="Shell">Shell</a-radio>
            <a-radio value="Python">Python</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item label="模板内容" required>
          <a-textarea
            v-model="templateForm.content"
            placeholder="请输入脚本内容..."
            :auto-size="{ minRows: 8, maxRows: 18 }"
            class="code-textarea"
          />
        </a-form-item>

        <a-form-item label="参数化">
          <a-link @click="showParamDialog = true">
            <template #icon><icon-plus /></template>
            添加参数
          </a-link>
          <div v-if="templateForm.parameters.length > 0" class="tag-list">
            <a-tag
              v-for="(param, index) in templateForm.parameters"
              :key="index"
              closable
              color="arcoblue"
              @close="removeParameter(index)"
            >
              {{ param.name }} ({{ param.varName }})
            </a-tag>
          </div>
        </a-form-item>

        <a-form-item label="备注信息">
          <a-textarea
            v-model="templateForm.remark"
            placeholder="请输入模板备注信息"
            :auto-size="{ minRows: 2, maxRows: 5 }"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑参数对话框 -->
    <a-modal
      v-model:visible="showParamDialog"
      title="编辑参数"
      :width="520"
      :unmount-on-close="true"
      @ok="handleSaveParameter"
      @cancel="showParamDialog = false"
    >
      <a-form :model="paramForm" layout="vertical">
        <a-form-item label="参数名" required>
          <a-input v-model="paramForm.name" placeholder="请输入参数名称" />
        </a-form-item>
        <a-form-item label="变量名" required>
          <a-input v-model="paramForm.varName" placeholder="请输入变量名" />
        </a-form-item>
        <a-form-item label="参数类型" required>
          <a-radio-group v-model="paramForm.type" type="button">
            <a-radio value="text">文本框</a-radio>
            <a-radio value="password">密码框</a-radio>
            <a-radio value="select">下拉选择</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="必填">
          <a-switch v-model="paramForm.required" />
        </a-form-item>
        <a-form-item label="默认值">
          <a-input v-model="paramForm.defaultValue" placeholder="请输入" />
        </a-form-item>
        <a-form-item label="提示信息">
          <a-textarea v-model="paramForm.helpText" placeholder="请输入该参数的帮助提示信息" :auto-size="{ minRows: 2, maxRows: 4 }" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 添加类型对话框 -->
    <a-modal
      v-model:visible="showAddTypeDialog"
      title="添加模板类型"
      :width="440"
      :unmount-on-close="true"
      @ok="handleAddType"
      @cancel="showAddTypeDialog = false"
    >
      <a-form :model="newTypeForm" layout="vertical">
        <a-form-item label="类型名称" required>
          <a-input v-model="newTypeForm.label" placeholder="请输入类型名称" />
        </a-form-item>
        <a-form-item label="类型值" required>
          <a-input v-model="newTypeForm.value" placeholder="请输入类型值（英文）" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconFile, IconSearch, IconRefresh, IconPlus } from '@arco-design/web-vue/es/icon'
import { getJobTemplateList, createJobTemplate, updateJobTemplate, deleteJobTemplate } from '@/api/task'

const searchForm = ref({ type: '', name: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })
const loading = ref(false)

const tablePagination = computed(() => ({
  current: pagination.value.page,
  pageSize: pagination.value.pageSize,
  total: pagination.value.total,
  showTotal: true,
  showPageSize: true,
  pageSizeOptions: [10, 20, 50, 100],
}))

const templateTypes = ref([
  { label: '系统信息', value: 'system' },
  { label: '部署', value: 'deploy' },
  { label: '监控', value: 'monitor' },
  { label: '备份', value: 'backup' },
])

const getCategoryLabel = (category: string) => {
  return templateTypes.value.find(t => t.value === category)?.label || category
}

const templates = ref<any[]>([])

const showTemplateDialog = ref(false)
const isEdit = ref(false)
const templateForm = ref({
  id: 0,
  type: '',
  name: '',
  code: '',
  scriptType: 'Shell',
  content: '',
  parameters: [] as any[],
  remark: '',
})

const showParamDialog = ref(false)
const paramForm = ref({
  name: '',
  varName: '',
  type: 'text',
  required: false,
  defaultValue: '',
  helpText: '',
})

const showAddTypeDialog = ref(false)
const newTypeForm = ref({ label: '', value: '' })

const loadTemplates = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keyword: searchForm.value.name || undefined,
      category: searchForm.value.type || undefined,
    }
    const response = await getJobTemplateList(params)
    if (Array.isArray(response)) {
      templates.value = response
      pagination.value.total = response.length
    } else if (response.list) {
      templates.value = response.list
      pagination.value.total = response.total || response.list.length
    } else if (response.data) {
      templates.value = response.data
      pagination.value.total = response.total || response.data.length
    } else {
      templates.value = []
      pagination.value.total = 0
    }
  } catch {
    Message.error('加载模板列表失败')
    templates.value = []
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.value.page = 1
  loadTemplates()
}

const handleReset = () => {
  searchForm.value = { type: '', name: '' }
  pagination.value.page = 1
  loadTemplates()
}

const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadTemplates()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1
  loadTemplates()
}

const handleCreate = () => {
  isEdit.value = false
  templateForm.value = { id: 0, type: '', name: '', code: '', scriptType: 'Shell', content: '', parameters: [], remark: '' }
  showTemplateDialog.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  let parameters: any[] = []
  if (row.variables) {
    try { parameters = JSON.parse(row.variables) } catch { parameters = [] }
  }
  templateForm.value = {
    id: row.id,
    type: row.category,
    name: row.name,
    code: row.code,
    scriptType: 'Shell',
    content: row.content,
    parameters,
    remark: row.description || '',
  }
  showTemplateDialog.value = true
}

const handleDelete = async (row: any) => {
  try {
    await deleteJobTemplate(row.id)
    Message.success('删除成功')
    loadTemplates()
  } catch (error: any) {
    Message.error(error.message || '删除失败')
  }
}

const handleSaveTemplate = async () => {
  if (!templateForm.value.type) { Message.warning('请选择模板类型'); return }
  if (!templateForm.value.name) { Message.warning('请输入模板名称'); return }
  if (!templateForm.value.content) { Message.warning('请输入模板内容'); return }

  try {
    const requestData = {
      name: templateForm.value.name,
      code: templateForm.value.code || `TPL_${Date.now()}`,
      description: templateForm.value.remark || '',
      content: templateForm.value.content,
      category: templateForm.value.type,
      platform: 'linux',
      timeout: 300,
      variables: templateForm.value.parameters.length > 0 ? JSON.stringify(templateForm.value.parameters) : '',
    }
    if (isEdit.value) {
      await updateJobTemplate(templateForm.value.id, requestData)
      Message.success('编辑成功')
    } else {
      await createJobTemplate(requestData)
      Message.success('创建成功')
    }
    showTemplateDialog.value = false
    await loadTemplates()
  } catch (error: any) {
    Message.error(error.message || '保存失败')
  }
}

const handleAddType = () => {
  if (!newTypeForm.value.label) { Message.warning('请输入类型名称'); return }
  if (!newTypeForm.value.value) { Message.warning('请输入类型值'); return }
  if (templateTypes.value.some(t => t.value === newTypeForm.value.value)) {
    Message.warning('该类型值已存在'); return
  }
  templateTypes.value.push({ ...newTypeForm.value })
  showAddTypeDialog.value = false
  newTypeForm.value = { label: '', value: '' }
  Message.success('添加类型成功')
}

const removeParameter = (index: number) => {
  templateForm.value.parameters.splice(index, 1)
}

const handleSaveParameter = () => {
  if (!paramForm.value.name) { Message.warning('请输入参数名'); return }
  if (!paramForm.value.varName) { Message.warning('请输入变量名'); return }
  templateForm.value.parameters.push({ ...paramForm.value })
  showParamDialog.value = false
  paramForm.value = { name: '', varName: '', type: 'text', required: false, defaultValue: '', helpText: '' }
  Message.success('参数添加成功')
}

onMounted(() => { loadTemplates() })
</script>

<style scoped lang="scss">
.page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-card {
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
  padding: 20px 24px;
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--ops-primary, #165dff) 0%, #4080ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.4;
}

.page-desc {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 2px;
}

.search-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.search-label {
  font-size: 14px;
  color: var(--ops-text-secondary, #4e5969);
  white-space: nowrap;
}

.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
  flex: 1;

  .card-title {
    font-weight: 600;
  }
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.code-textarea {
  :deep(textarea) {
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.6;
  }
}
</style>
