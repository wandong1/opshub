<template>
  <div class="task-schedule-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Timer /></el-icon>
        </div>
        <div>
          <h2 class="page-title">任务调度</h2>
          <p class="page-subtitle">管理拨测调度任务，支持 Cron 定时执行与结果推送</p>
        </div>
      </div>
    </div>

    <div class="filter-bar">
      <el-input v-model="searchForm.keyword" placeholder="搜索任务名称" clearable style="width: 220px;" @keyup.enter="loadData" />
      <el-select v-model="searchForm.status" placeholder="状态" clearable style="width: 120px;">
        <el-option label="启用" :value="1" />
        <el-option label="禁用" :value="0" />
      </el-select>
      <el-button type="primary" :icon="Search" @click="loadData">搜索</el-button>
      <el-button :icon="Refresh" @click="handleReset">重置</el-button>
      <div style="flex: 1;" />
      <el-button v-permission="'inspection:tasks:create'" type="primary" :icon="Plus" class="black-button" @click="handleCreate">新增任务</el-button>
    </div>

    <el-table :data="taskList" v-loading="loading" border stripe>
      <el-table-column label="任务名称" prop="name" min-width="140" />
      <el-table-column label="关联拨测" width="140">
        <template #default="{ row }">
          <el-tooltip v-if="row.probeConfigIds?.length" placement="top">
            <template #content>
              <div v-for="id in row.probeConfigIds" :key="id">{{ getProbeLabel(id) }}</div>
            </template>
            <el-tag size="small">{{ row.probeConfigIds.length }} 个配置</el-tag>
          </el-tooltip>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="并发数" width="80" align="center" prop="concurrency" />
      <el-table-column label="Cron表达式" prop="cronExpr" min-width="160" show-overflow-tooltip />
      <el-table-column label="Pushgateway" width="140">
        <template #default="{ row }">{{ getPgwLabel(row.pushgatewayId) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="最后执行" width="170">
        <template #default="{ row }">{{ row.lastRunAt || '-' }}</template>
      </el-table-column>
      <el-table-column label="结果" width="80" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.lastResult" :type="row.lastResult === 'success' ? 'success' : 'danger'" size="small">{{ row.lastResult }}</el-tag>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right" align="center">
        <template #default="{ row }">
          <el-button v-permission="'inspection:tasks:toggle'" link type="warning" @click="handleToggle(row)">{{ row.status === 1 ? '禁用' : '启用' }}</el-button>
          <el-button v-permission="'inspection:tasks:update'" link type="primary" @click="handleEdit(row)">编辑</el-button>
          <el-button v-permission="'inspection:tasks:delete'" link type="danger" @click="handleDelete(row)">删除</el-button>
          <el-button v-permission="'inspection:tasks:results'" link @click="handleViewResults(row)">结果</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-container">
      <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.pageSize"
        :total="pagination.total" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next" @change="loadData" />
    </div>

    <!-- 新建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑任务' : '新增任务'" width="660px" destroy-on-close>
      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="110px">
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="拨测分类">
          <el-radio-group v-model="selectedCategory" @change="handleTaskCategoryChange">
            <el-radio-button v-for="c in PROBE_CATEGORIES" :key="c.value" :value="c.value" :disabled="!c.enabled">
              {{ c.label }}
              <el-tooltip v-if="!c.enabled" content="即将推出" placement="top"><template #default><span /></template></el-tooltip>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="拨测配置" prop="probeConfigIds">
          <el-select v-model="formData.probeConfigIds" multiple filterable placeholder="选择拨测配置（可多选）" style="width: 100%;">
            <el-option v-for="p in filteredProbeOptions" :key="p.id" :label="p.name" :value="p.id">
              <span>{{ p.name }}</span>
              <span style="float: right; color: #909399; font-size: 12px;">{{ p.type?.toUpperCase() }} · {{ p.target }}</span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="执行计划" prop="cronExpr">
          <div class="cron-picker">
            <div class="cron-presets">
              <el-button v-for="preset in cronPresets" :key="preset.value" size="small"
                :type="formData.cronExpr === preset.value ? 'primary' : 'default'"
                @click="formData.cronExpr = preset.value">{{ preset.label }}</el-button>
            </div>
            <el-input v-model="formData.cronExpr" placeholder="秒级cron，如: 0/30 * * * * ?" style="margin-top: 8px;">
              <template #prepend>Cron</template>
            </el-input>
            <div class="cron-description">{{ cronDescription }}</div>
          </div>
        </el-form-item>
        <el-form-item label="并发数">
          <el-input-number v-model="formData.concurrency" :min="1" :max="50" />
          <span style="margin-left: 8px; font-size: 12px; color: #909399;">同时执行的最大拨测数</span>
        </el-form-item>
        <el-form-item label="Pushgateway">
          <el-select v-model="formData.pushgatewayId" placeholder="选择Pushgateway（可选）" clearable style="width: 100%;">
            <el-option v-for="p in pgwOptions" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="业务分组">
          <el-select v-model="formData.groupId" placeholder="选择业务分组" clearable filterable style="width: 100%;">
            <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="formData.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="formData.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" class="black-button" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 执行结果抽屉 -->
    <el-drawer v-model="resultsVisible" title="执行结果" size="60%">
      <el-table :data="resultList" v-loading="resultsLoading" border stripe>
        <el-table-column label="时间" prop="createdAt" width="170" />
        <el-table-column label="成功" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" size="small">{{ row.success ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="延迟(ms)" width="100" align="center">
          <template #default="{ row }">{{ row.latency?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="丢包率" width="90" align="center">
          <template #default="{ row }">{{ row.packetLoss !== undefined ? (row.packetLoss * 100).toFixed(1) + '%' : '-' }}</template>
        </el-table-column>
        <el-table-column label="错误信息" prop="errorMessage" min-width="200" show-overflow-tooltip />
      </el-table>
      <div class="pagination-container">
        <el-pagination v-model:current-page="resultPagination.page" v-model:page-size="resultPagination.pageSize"
          :total="resultPagination.total" :page-sizes="[20, 50]" layout="total, prev, pager, next" @change="loadResults" />
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Timer, Search, Refresh, Plus } from '@element-plus/icons-vue'
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

const formRules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  probeConfigIds: [{ required: true, type: 'array', min: 1, message: '请选择至少一个拨测配置', trigger: 'change' }],
  cronExpr: [{ required: true, message: '请输入或选择Cron表达式', trigger: 'blur' }],
}

const filteredProbeOptions = computed(() => {
  return probeOptions.value.filter((p: any) => {
    if (!selectedCategory.value) return true
    return p.category === selectedCategory.value
  })
})

const getProbeLabel = (id: number) => probeOptions.value.find((p: any) => p.id === id)?.name || id
const getPgwLabel = (id: number) => pgwOptions.value.find((p: any) => p.id === id)?.name || id || '-'
// SCRIPT_CONTINUE_PLACEHOLDER

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
  // Clear selected configs that don't belong to the new category
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
  // Infer category from first config
  if (formData.probeConfigIds.length > 0) {
    const firstConfig = probeOptions.value.find((p: any) => p.id === formData.probeConfigIds[0])
    if (firstConfig?.category) selectedCategory.value = firstConfig.category
  }
  dialogVisible.value = true
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定删除该任务？', '提示', { type: 'warning' }).then(async () => {
    await deleteTask(row.id); ElMessage.success('删除成功'); loadData()
  }).catch(() => {})
}

const handleToggle = async (row: any) => {
  try { await toggleTask(row.id); ElMessage.success('操作成功'); loadData() } catch {}
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) { await updateTask(formData.id, formData); ElMessage.success('更新成功') }
      else { await createTask(formData); ElMessage.success('创建成功') }
      dialogVisible.value = false; loadData()
    } catch {} finally { submitting.value = false }
  })
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
.task-schedule-container { padding: 20px; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 20px; }
.page-title { margin: 0; font-size: 18px; font-weight: 600; }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: #909399; }
.filter-bar { display: flex; gap: 10px; margin-bottom: 16px; align-items: center; }
.pagination-container { margin-top: 16px; display: flex; justify-content: flex-end; }
.cron-picker { width: 100%; }
.cron-presets { display: flex; flex-wrap: wrap; gap: 6px; }
.cron-description { font-size: 12px; color: #67c23a; margin-top: 4px; min-height: 18px; }
</style>
