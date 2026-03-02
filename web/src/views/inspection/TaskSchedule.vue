<template>
  <div class="task-schedule-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-schedule /></div>
        <div>
          <h2 class="page-title">任务调度</h2>
          <p class="page-subtitle">管理拨测调度任务，支持 Cron 定时执行与结果推送</p>
        </div>
      </div>
    </div>

    <div class="filter-bar">
      <a-input v-model="searchForm.keyword" placeholder="搜索任务名称" allow-clear style="width: 220px;" @press-enter="loadData" />
      <a-select v-model="searchForm.status" placeholder="状态" allow-clear style="width: 120px;">
        <a-option label="启用" :value="1" />
        <a-option label="禁用" :value="0" />
      </a-select>
      <a-button type="primary" @click="loadData"><template #icon><icon-search /></template>搜索</a-button>
      <a-button @click="handleReset"><template #icon><icon-refresh /></template>重置</a-button>
      <div style="flex: 1;" />
      <a-button v-permission="'inspection:tasks:create'" type="primary" @click="handleCreate"><template #icon><icon-plus /></template>新增任务</a-button>
    </div>

    <a-table :data="taskList" :loading="loading" :bordered="{ cell: true }" stripe :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50] }" @page-change="(p: number) => { pagination.page = p; loadData() }" @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }">
      <template #columns>
        <a-table-column title="任务名称" data-index="name" :width="140" />
        <a-table-column title="关联拨测" :width="140">
          <template #cell="{ record }">
            <a-tooltip v-if="record.probeConfigIds?.length" position="top">
              <template #content><div v-for="id in record.probeConfigIds" :key="id">{{ getProbeLabel(id) }}</div></template>
              <a-tag size="small">{{ record.probeConfigIds.length }} 个配置</a-tag>
            </a-tooltip>
            <span v-else>-</span>
          </template>
        </a-table-column>
        <a-table-column title="并发数" data-index="concurrency" :width="80" align="center" />
        <a-table-column title="Cron表达式" data-index="cronExpr" :width="160" ellipsis tooltip />
        <a-table-column title="Pushgateway" :width="140">
          <template #cell="{ record }">{{ getPgwLabel(record.pushgatewayId) }}</template>
        </a-table-column>
        <a-table-column title="状态" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.status === 1 ? 'green' : 'red'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="最后执行" :width="170">
          <template #cell="{ record }">{{ record.lastRunAt || '-' }}</template>
        </a-table-column>
        <a-table-column title="结果" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag v-if="record.lastResult" size="small" :color="record.lastResult === 'success' ? 'green' : 'red'">{{ record.lastResult }}</a-tag>
            <span v-else>-</span>
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="220" fixed="right" align="center">
          <template #cell="{ record }">
            <a-button v-permission="'inspection:tasks:toggle'" type="text" size="small" status="warning" @click="handleToggle(record)">{{ record.status === 1 ? '禁用' : '启用' }}</a-button>
            <a-button v-permission="'inspection:tasks:update'" type="text" size="small" @click="handleEdit(record)">编辑</a-button>
            <a-button v-permission="'inspection:tasks:delete'" type="text" size="small" status="danger" @click="handleDelete(record)">删除</a-button>
            <a-button v-permission="'inspection:tasks:results'" type="text" size="small" @click="handleViewResults(record)">结果</a-button>
          </template>
        </a-table-column>
      </template>
    </a-table>
    <!-- 新建/编辑对话框 -->
    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑任务' : '新增任务'" :width="660" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="任务名称" field="name">
          <a-input v-model="formData.name" placeholder="请输入任务名称" />
        </a-form-item>
        <a-form-item label="拨测分类">
          <a-radio-group v-model="selectedCategory" type="button" @change="handleTaskCategoryChange">
            <a-radio v-for="c in PROBE_CATEGORIES" :key="c.value" :value="c.value" :disabled="!c.enabled">{{ c.label }}</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="拨测配置" field="probeConfigIds">
          <a-select v-model="formData.probeConfigIds" multiple allow-search placeholder="选择拨测配置（可多选）" style="width: 100%;">
            <a-option v-for="p in filteredProbeOptions" :key="p.id" :label="p.name" :value="p.id">
              {{ p.name }} <span style="float: right; color: var(--ops-text-tertiary); font-size: 12px;">{{ p.type?.toUpperCase() }} · {{ p.target }}</span>
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="执行计划" field="cronExpr">
          <div class="cron-picker">
            <div class="cron-presets">
              <a-button v-for="preset in cronPresets" :key="preset.value" size="small"
                :type="formData.cronExpr === preset.value ? 'primary' : 'secondary'"
                @click="formData.cronExpr = preset.value">{{ preset.label }}</a-button>
            </div>
            <a-input v-model="formData.cronExpr" placeholder="秒级cron，如: 0/30 * * * * ?" style="margin-top: 8px;">
              <template #prepend>Cron</template>
            </a-input>
            <div class="cron-description">{{ cronDescription }}</div>
          </div>
        </a-form-item>
        <a-form-item label="并发数">
          <a-input-number v-model="formData.concurrency" :min="1" :max="50" />
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">同时执行的最大拨测数</span>
        </a-form-item>
        <a-form-item label="Pushgateway">
          <a-select v-model="formData.pushgatewayId" placeholder="选择Pushgateway（可选）" allow-clear style="width: 100%;">
            <a-option v-for="p in pgwOptions" :key="p.id" :label="p.name" :value="p.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="业务分组">
          <a-select v-model="formData.groupId" placeholder="选择业务分组" allow-clear allow-search style="width: 100%;">
            <a-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model="formData.description" :max-length="200" :auto-size="{ minRows: 2 }" />
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

    <!-- 执行结果抽屉 -->
    <a-drawer v-model:visible="resultsVisible" title="执行结果" :width="720" unmount-on-close>
      <a-table :data="resultList" :loading="resultsLoading" :bordered="{ cell: true }" stripe :pagination="{ current: resultPagination.page, pageSize: resultPagination.pageSize, total: resultPagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [20, 50] }" @page-change="(p: number) => { resultPagination.page = p; loadResults() }" @page-size-change="(s: number) => { resultPagination.pageSize = s; resultPagination.page = 1; loadResults() }">
        <template #columns>
          <a-table-column title="时间" data-index="createdAt" :width="170" />
          <a-table-column title="成功" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.success ? 'green' : 'red'">{{ record.success ? '是' : '否' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="延迟(ms)" :width="100" align="center">
            <template #cell="{ record }">{{ record.latency?.toFixed(2) }}</template>
          </a-table-column>
          <a-table-column title="丢包率" :width="90" align="center">
            <template #cell="{ record }">{{ record.packetLoss !== undefined ? (record.packetLoss * 100).toFixed(1) + '%' : '-' }}</template>
          </a-table-column>
          <a-table-column title="执行方式" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.agentHostId > 0" size="small" color="purple">Agent #{{ record.agentHostId }}</a-tag>
              <span v-else>本地</span>
            </template>
          </a-table-column>
          <a-table-column title="重试" :width="70" align="center">
            <template #cell="{ record }">{{ record.retryAttempt > 0 ? record.retryAttempt : '-' }}</template>
          </a-table-column>
          <a-table-column title="错误信息" data-index="errorMessage" ellipsis tooltip />
        </template>
      </a-table>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconSchedule, IconSearch, IconRefresh, IconPlus } from '@arco-design/web-vue/es/icon'
import { getTaskList, createTask, updateTask, deleteTask, toggleTask, getTaskResults, getProbeList, getPushgatewayList, PROBE_CATEGORIES, CATEGORY_LABEL_MAP } from '@/api/networkProbe'
import { getGroupTree } from '@/api/assetGroup'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const taskList = ref<any[]>([])
const probeOptions = ref<any[]>([])
const pgwOptions = ref<any[]>([])
const groupOptions = ref<any[]>([])
const resultsVisible = ref(false)
const resultsLoading = ref(false)
const resultList = ref<any[]>([])
const currentTaskId = ref(0)
const selectedCategory = ref('network')

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const resultPagination = reactive({ page: 1, pageSize: 20, total: 0 })
const searchForm = reactive({ keyword: '', status: undefined as number | undefined })

const cronPresets = [
  { label: '每30秒', value: '0/30 * * * * ?' },
  { label: '每分钟', value: '0 * * * * ?' },
  { label: '每5分钟', value: '0 0/5 * * * ?' },
  { label: '每15分钟', value: '0 0/15 * * * ?' },
  { label: '每小时', value: '0 0 * * * ?' },
  { label: '每天0点', value: '0 0 0 * * ?' },
]

const cronDescriptionMap: Record<string, string> = {
  '0/30 * * * * ?': '每30秒执行一次',
  '0 * * * * ?': '每分钟执行一次',
  '0 0/5 * * * ?': '每5分钟执行一次',
  '0 0/15 * * * ?': '每15分钟执行一次',
  '0 0 * * * ?': '每小时执行一次',
  '0 0 0 * * ?': '每天凌晨0点执行一次',
}

const cronDescription = computed(() => cronDescriptionMap[formData.cronExpr] || (formData.cronExpr ? '自定义 Cron 表达式' : ''))

const defaultForm = () => ({
  id: 0, name: '', probeConfigIds: [] as number[], groupId: 0, cronExpr: '',
  pushgatewayId: undefined as number | undefined, concurrency: 5, status: 1, description: ''
})
const formData = reactive(defaultForm())

const formRules = {
  name: [{ required: true, message: '请输入任务名称' }],
  probeConfigIds: [{ required: true, type: 'array' as const, min: 1, message: '请选择至少一个拨测配置' }],
  cronExpr: [{ required: true, message: '请输入或选择Cron表达式' }],
}

const filteredProbeOptions = computed(() => {
  return probeOptions.value.filter((p: any) => {
    if (!selectedCategory.value) return true
    return p.category === selectedCategory.value
  })
})

const getProbeLabel = (id: number) => probeOptions.value.find((p: any) => p.id === id)?.name || id
const getPgwLabel = (id: number) => pgwOptions.value.find((p: any) => p.id === id)?.name || id || '-'

const loadData = async () => {
  loading.value = true
  try {
    const res = await getTaskList({ page: pagination.page, page_size: pagination.pageSize, keyword: searchForm.keyword, status: searchForm.status })
    taskList.value = res.data || []; pagination.total = res.total || 0
  } catch {} finally { loading.value = false }
}

const flattenGroups = (tree: any[], result: any[] = []): any[] => {
  for (const node of tree) {
    result.push({ id: node.id, name: node.name })
    if (node.children?.length) flattenGroups(node.children, result)
  }
  return result
}

const loadOptions = async () => {
  try {
    const probeRes = await getProbeList({ page: 1, page_size: 1000, status: 1 })
    probeOptions.value = probeRes.data || []
  } catch {}
  try { pgwOptions.value = (await getPushgatewayList()) || [] } catch {}
  try {
    const res = await getGroupTree()
    groupOptions.value = flattenGroups(res.data || res || [])
  } catch {}
}

const handleTaskCategoryChange = () => {
  formData.probeConfigIds = formData.probeConfigIds.filter(id =>
    filteredProbeOptions.value.some((p: any) => p.id === id)
  )
}

const handleReset = () => { searchForm.keyword = ''; searchForm.status = undefined; pagination.page = 1; loadData() }

const handleCreate = async () => {
  isEdit.value = false; Object.assign(formData, defaultForm()); selectedCategory.value = 'network'
  await loadOptions(); dialogVisible.value = true
}

const handleEdit = async (row: any) => {
  isEdit.value = true
  Object.assign(formData, {
    id: row.id, name: row.name, probeConfigIds: row.probeConfigIds || [],
    groupId: row.groupId, cronExpr: row.cronExpr, pushgatewayId: row.pushgatewayId,
    concurrency: row.concurrency || 5, status: row.status, description: row.description
  })
  await loadOptions()
  if (formData.probeConfigIds.length > 0) {
    const firstConfig = probeOptions.value.find((p: any) => p.id === formData.probeConfigIds[0])
    if (firstConfig?.category) selectedCategory.value = firstConfig.category
  }
  dialogVisible.value = true
}

const handleDelete = (row: any) => {
  Modal.warning({ title: '提示', content: '确定删除该任务？', hideCancel: false, onOk: async () => { await deleteTask(row.id); Message.success('删除成功'); loadData() } })
}

const handleToggle = async (row: any) => {
  try { await toggleTask(row.id); Message.success('操作成功'); loadData() } catch {}
}

const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return
  submitting.value = true
  try {
    if (isEdit.value) { await updateTask(formData.id, formData); Message.success('更新成功') }
    else { await createTask(formData); Message.success('创建成功') }
    dialogVisible.value = false; loadData()
  } catch {} finally { submitting.value = false }
}

const handleViewResults = (row: any) => {
  currentTaskId.value = row.id; resultPagination.page = 1; resultsVisible.value = true; loadResults()
}

const loadResults = async () => {
  resultsLoading.value = true
  try {
    const res = await getTaskResults(currentTaskId.value, { page: resultPagination.page, page_size: resultPagination.pageSize })
    resultList.value = res.data || []; resultPagination.total = res.total || 0
  } catch {} finally { resultsLoading.value = false }
}

onMounted(() => { loadData(); loadOptions() })
</script>
<style scoped>
.task-schedule-container { padding: 0; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 36px; height: 36px; border-radius: 8px; background: var(--ops-primary); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 18px; }
.page-title { margin: 0; font-size: 17px; font-weight: 600; color: var(--ops-text-primary); }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: var(--ops-text-tertiary); }
.filter-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; }
.cron-picker { width: 100%; }
.cron-presets { display: flex; flex-wrap: wrap; gap: 6px; }
.cron-description { font-size: 12px; color: var(--ops-success); margin-top: 4px; min-height: 18px; }
</style>
