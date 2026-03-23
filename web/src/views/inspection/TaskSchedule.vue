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
      <a-select v-model="searchForm.taskType" placeholder="任务类型" allow-clear style="width: 120px;">
        <a-option label="拨测任务" value="probe" />
        <a-option label="巡检任务" value="inspection" />
      </a-select>
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
        <a-table-column title="任务类型" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.taskType === 'inspection' ? 'purple' : 'blue'">
              {{ record.taskType === 'inspection' ? '巡检' : '拨测' }}
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="关联配置" :width="140">
          <template #cell="{ record }">
            <a-tooltip v-if="record.taskType === 'probe' && record.probeConfigIds?.length" position="top">
              <template #content><div v-for="id in record.probeConfigIds" :key="id">{{ getProbeLabel(id) }}</div></template>
              <a-tag size="small">{{ record.probeConfigIds.length }} 个拨测</a-tag>
            </a-tooltip>
            <a-tooltip v-else-if="record.taskType === 'inspection' && record.inspectionGroupIds?.length" position="top">
              <template #content><div v-for="id in record.inspectionGroupIds" :key="id">{{ getInspectionGroupLabel(id) }}</div></template>
              <a-tag size="small" color="purple">{{ record.inspectionGroupIds.length }} 个巡检组</a-tag>
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
            <a-tag size="small" :color="record.enabled ? 'green' : 'red'">{{ record.enabled ? '启用' : '禁用' }}</a-tag>
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
        <a-table-column title="操作" :width="380" fixed="right" align="center">
          <template #cell="{ record }">
            <a-tooltip content="启用/禁用">
              <a-button v-permission="'inspection:tasks:toggle'" type="text" size="small" :status="record.enabled ? 'warning' : 'normal'" @click="handleToggle(record)">
                <template #icon><icon-poweroff /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="编辑">
              <a-button v-permission="'inspection:tasks:update'" type="text" size="small" @click="handleEdit(record)">
                <template #icon><icon-edit /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="删除">
              <a-button v-permission="'inspection:tasks:delete'" type="text" size="small" status="danger" @click="handleDelete(record)">
                <template #icon><icon-delete /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="查看结果">
              <a-button v-permission="'inspection:tasks:results'" type="text" size="small" @click="handleViewResults(record)">
                <template #icon><icon-eye /></template>
              </a-button>
            </a-tooltip>
            <template v-if="runningTaskIds.has(record.id)">
              <a-tooltip content="终止运行">
                <a-button v-permission="'inspection:tasks:run'" type="text" size="small" status="danger" @click="handleStop(record)">
                  <template #icon><icon-stop /></template>
                  <span class="running-text">运行中</span>
                </a-button>
              </a-tooltip>
            </template>
            <template v-else>
              <a-tooltip content="立即运行一次">
                <a-button v-permission="'inspection:tasks:run'" type="text" size="small" status="success" @click="handleRun(record)">
                  <template #icon><icon-play-arrow /></template>
                  立即运行
                </a-button>
              </a-tooltip>
            </template>
          </template>
        </a-table-column>
      </template>
    </a-table>
    <!-- 新建/编辑对话框 -->
    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑任务' : '新增任务'" :width="720" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="任务名称" field="name">
          <a-input v-model="formData.name" placeholder="请输入任务名称" />
        </a-form-item>

        <!-- 任务类型选择 -->
        <a-form-item label="任务类型" field="taskType">
          <a-radio-group v-model="formData.taskType" type="button" @change="handleTaskTypeChange">
            <a-radio value="probe">拨测任务</a-radio>
            <a-radio value="inspection">巡检任务</a-radio>
          </a-radio-group>
        </a-form-item>

        <!-- 拨测任务配置 -->
        <template v-if="formData.taskType === 'probe'">
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
        </template>

        <!-- 巡检任务配置 -->
        <template v-if="formData.taskType === 'inspection'">
          <a-form-item label="巡检组" field="inspectionGroupIds">
            <a-select v-model="formData.inspectionGroupIds" multiple allow-search placeholder="选择巡检组（可多选）" style="width: 100%;" @change="handleInspectionGroupChange">
              <a-option v-for="g in inspectionGroupOptions" :key="g.id" :label="g.name" :value="g.id">
                {{ g.name }} <span style="float: right; color: var(--ops-text-tertiary); font-size: 12px;">{{ g.itemCount || 0 }} 个巡检项</span>
              </a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="指定巡检项" v-if="formData.inspectionGroupIds.length > 0">
            <a-select v-model="formData.inspectionItemIds" multiple allow-search placeholder="不选则执行所有巡检项" style="width: 100%;">
              <a-option v-for="item in filteredInspectionItems" :key="item.id" :label="item.name" :value="item.id">
                {{ item.name }} <span style="float: right; color: var(--ops-text-tertiary); font-size: 12px;">{{ getInspectionGroupLabel(item.groupId) }}</span>
              </a-option>
            </a-select>
          </a-form-item>
        </template>

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
        <a-form-item label="并发数" v-if="formData.taskType === 'probe'">
          <a-input-number v-model="formData.concurrency" :min="1" :max="50" />
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">同时执行的最大拨测数</span>
        </a-form-item>
        <a-form-item label="Pushgateway">
          <a-select v-model="formData.pushgatewayId" placeholder="选择Pushgateway（可选）" allow-clear style="width: 100%;">
            <a-option v-for="p in pgwOptions" :key="p.id" :label="p.name" :value="p.id" />
          </a-select>
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">
            {{ formData.taskType === 'inspection' ? '推送巡检指标到 Prometheus' : '推送拨测指标到 Prometheus' }}
          </span>
        </a-form-item>
        <a-form-item label="业务分组">
          <a-select v-model="formData.groupId" placeholder="选择业务分组" allow-clear allow-search style="width: 100%;">
            <a-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="负责人">
          <a-input v-model="formData.owner" placeholder="请输入负责人（可选，用于 Metric 标签）" allow-clear />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model="formData.description" :max-length="200" :auto-size="{ minRows: 2 }" />
        </a-form-item>
        <a-form-item label="状态">
          <a-radio-group v-model="formData.enabled"><a-radio :value="true">启用</a-radio><a-radio :value="false">禁用</a-radio></a-radio-group>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" :loading="submitting" @click="handleSubmit">确定</a-button>
      </template>
    </a-modal>

    <!-- 执行结果抽屉 -->
    <a-drawer v-model:visible="resultsVisible" title="执行结果" :width="720" unmount-on-close>
      <!-- 拨测任务结果 -->
      <a-table v-if="currentTaskType === 'probe'" :data="resultList" :loading="resultsLoading" :bordered="{ cell: true }" stripe :pagination="{ current: resultPagination.page, pageSize: resultPagination.pageSize, total: resultPagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [20, 50] }" @page-change="(p: number) => { resultPagination.page = p; loadResults() }" @page-size-change="(s: number) => { resultPagination.pageSize = s; resultPagination.page = 1; loadResults() }">
        <template #columns>
          <a-table-column title="时间" data-index="createdAt" :width="170" />
          <a-table-column title="触发方式" :width="90" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.triggerType === 'manual' ? 'orange' : 'arcoblue'">{{ record.triggerType === 'manual' ? '手动' : '调度' }}</a-tag>
            </template>
          </a-table-column>
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

      <!-- 巡检任务结果 -->
      <a-table v-else :data="resultList" :loading="resultsLoading" :bordered="{ cell: true }" stripe :pagination="{ current: resultPagination.page, pageSize: resultPagination.pageSize, total: resultPagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [20, 50] }" @page-change="(p: number) => { resultPagination.page = p; loadResults() }" @page-size-change="(s: number) => { resultPagination.pageSize = s; resultPagination.page = 1; loadResults() }">
        <template #columns>
          <a-table-column title="执行时间" data-index="executed_at" :width="170" />
          <a-table-column title="触发方式" :width="90" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.trigger_type === 'manual' ? 'orange' : 'arcoblue'">{{ record.trigger_type === 'manual' ? '手动' : '调度' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="巡检项" data-index="item_name" :width="150" ellipsis tooltip />
          <a-table-column title="主机" data-index="host_name" :width="120" ellipsis tooltip />
          <a-table-column title="状态" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.status === 'success' ? 'green' : 'red'">{{ record.status === 'success' ? '成功' : '失败' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="耗时(ms)" :width="100" align="center" data-index="duration" />
          <a-table-column title="断言结果" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.assertion_result" size="small" :color="record.assertion_result === 'pass' ? 'green' : 'red'">{{ record.assertion_result }}</a-tag>
              <span v-else>-</span>
            </template>
          </a-table-column>
          <a-table-column title="错误信息" data-index="error_message" ellipsis tooltip />
        </template>
      </a-table>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconSchedule, IconSearch, IconRefresh, IconPlus, IconEdit, IconDelete, IconPoweroff, IconPlayArrow, IconStop, IconEye } from '@arco-design/web-vue/es/icon'
import { getTaskList, createTask, updateTask, deleteTask, toggleTask, getTaskResults, getProbeList, getPushgatewayList, PROBE_CATEGORIES, CATEGORY_LABEL_MAP } from '@/api/networkProbe'
import { getAllInspectionGroups, getInspectionItems } from '@/api/inspectionManagement'
import { getInspectionTasks, createInspectionTask, updateInspectionTask, deleteInspectionTask, toggleInspectionTask, getInspectionTaskResults, runInspectionTask, stopInspectionTask } from '@/api/inspectionTask'
import { getGroupTree } from '@/api/assetGroup'
import { useRouter } from 'vue-router'

const router = useRouter()

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const taskList = ref<any[]>([])
const probeOptions = ref<any[]>([])
const pgwOptions = ref<any[]>([])
const groupOptions = ref<any[]>([])
const inspectionGroupOptions = ref<any[]>([])
const inspectionItemOptions = ref<any[]>([])
const resultsVisible = ref(false)
const resultsLoading = ref(false)
const resultList = ref<any[]>([])
const currentTaskId = ref(0)
const selectedCategory = ref('network')
const runningTaskIds = ref<Set<number>>(new Set())

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const resultPagination = reactive({ page: 1, pageSize: 20, total: 0 })
const searchForm = reactive({ keyword: '', taskType: undefined as string | undefined, status: undefined as number | undefined })

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
  id: 0, name: '', taskType: 'probe' as 'probe' | 'inspection',
  probeConfigIds: [] as number[],
  inspectionGroupIds: [] as number[],
  inspectionItemIds: [] as number[],
  groupId: 0, cronExpr: '',
  pushgatewayId: undefined as number | undefined, concurrency: 5, enabled: true, description: '', owner: ''
})
const formData = reactive(defaultForm())

const formRules = computed(() => {
  const rules: any = {
    name: [{ required: true, message: '请输入任务名称' }],
    taskType: [{ required: true, message: '请选择任务类型' }],
    cronExpr: [{ required: true, message: '请输入或选择Cron表达式' }],
  }

  if (formData.taskType === 'probe') {
    rules.probeConfigIds = [{ required: true, type: 'array' as const, min: 1, message: '请选择至少一个拨测配置' }]
  } else if (formData.taskType === 'inspection') {
    rules.inspectionGroupIds = [{ required: true, type: 'array' as const, min: 1, message: '请选择至少一个巡检组' }]
  }

  return rules
})

const filteredProbeOptions = computed(() => {
  return probeOptions.value.filter((p: any) => {
    if (!selectedCategory.value) return true
    return p.category === selectedCategory.value
  })
})

const filteredInspectionItems = computed(() => {
  if (formData.inspectionGroupIds.length === 0) return []
  return inspectionItemOptions.value.filter((item: any) =>
    formData.inspectionGroupIds.includes(item.groupId)
  )
})

const getProbeLabel = (id: number) => probeOptions.value.find((p: any) => p.id === id)?.name || id
const getPgwLabel = (id: number) => pgwOptions.value.find((p: any) => p.id === id)?.name || id || '-'
const getInspectionGroupLabel = (id: number) => inspectionGroupOptions.value.find((g: any) => g.id === id)?.name || id

const loadData = async () => {
  loading.value = true
  try {
    const res = await getInspectionTasks({
      page: pagination.page,
      page_size: pagination.pageSize,
      name: searchForm.keyword,
      enabled: searchForm.status !== undefined ? searchForm.status === 1 : undefined
    })

    // 转换数据格式以兼容前端显示
    taskList.value = (res.list || []).map((task: any) => {
      const converted: any = {
        id: task.id,
        name: task.name,
        description: task.description,
        taskType: task.task_type || 'probe',
        cronExpr: task.cron_expr,
        enabled: task.enabled,
        concurrency: task.concurrency || 5,
        pushgatewayId: task.pushgateway_id,
        owner: task.owner || '',
        groupId: 0,
        lastRunAt: task.last_run_at,
        lastResult: task.last_run_status,
        nextRunAt: task.next_run_at
      }

      // 解析配置ID
      if (task.task_type === 'inspection') {
        try {
          converted.inspectionGroupIds = task.group_ids ? JSON.parse(task.group_ids) : []
          converted.inspectionItemIds = task.item_ids ? JSON.parse(task.item_ids) : []
        } catch (e) {
          converted.inspectionGroupIds = []
          converted.inspectionItemIds = []
        }
      } else {
        try {
          converted.probeConfigIds = task.item_ids ? JSON.parse(task.item_ids) : []
        } catch (e) {
          converted.probeConfigIds = []
        }
        try {
          const gids = task.group_ids ? JSON.parse(task.group_ids) : []
          converted.groupId = gids.length > 0 ? gids[0] : 0
        } catch (e) {
          converted.groupId = 0
        }
      }

      return converted
    })

    pagination.total = res.total || 0
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
  try {
    inspectionGroupOptions.value = await getAllInspectionGroups()
  } catch {}
}

const loadInspectionItems = async (groupIds: number[]) => {
  if (groupIds.length === 0) {
    inspectionItemOptions.value = []
    return
  }
  try {
    const allItems: any[] = []
    for (const groupId of groupIds) {
      const res = await getInspectionItems({ groupId, pageSize: 100 })
      allItems.push(...res.list)
    }
    inspectionItemOptions.value = allItems
  } catch {}
}

const handleTaskTypeChange = () => {
  formData.probeConfigIds = []
  formData.inspectionGroupIds = []
  formData.inspectionItemIds = []
}

const handleInspectionGroupChange = () => {
  loadInspectionItems(formData.inspectionGroupIds)
  formData.inspectionItemIds = formData.inspectionItemIds.filter(id =>
    filteredInspectionItems.value.some((item: any) => item.id === id)
  )
}

const handleTaskCategoryChange = () => {
  formData.probeConfigIds = formData.probeConfigIds.filter(id =>
    filteredProbeOptions.value.some((p: any) => p.id === id)
  )
}

const handleReset = () => { searchForm.keyword = ''; searchForm.taskType = undefined; searchForm.status = undefined; pagination.page = 1; loadData() }

const handleCreate = async () => {
  isEdit.value = false; Object.assign(formData, defaultForm()); selectedCategory.value = 'network'
  await loadOptions(); dialogVisible.value = true
}

const handleEdit = async (row: any) => {
  isEdit.value = true
  Object.assign(formData, {
    id: row.id, name: row.name,
    taskType: row.taskType || 'probe',
    probeConfigIds: row.probeConfigIds || [],
    inspectionGroupIds: row.inspectionGroupIds || [],
    inspectionItemIds: row.inspectionItemIds || [],
    groupId: row.groupId, cronExpr: row.cronExpr, pushgatewayId: row.pushgatewayId,
    concurrency: row.concurrency || 5, enabled: row.enabled, description: row.description, owner: row.owner || ''
  })
  await loadOptions()

  if (formData.taskType === 'probe' && formData.probeConfigIds.length > 0) {
    const firstConfig = probeOptions.value.find((p: any) => p.id === formData.probeConfigIds[0])
    if (firstConfig?.category) selectedCategory.value = firstConfig.category
  } else if (formData.taskType === 'inspection' && formData.inspectionGroupIds.length > 0) {
    await loadInspectionItems(formData.inspectionGroupIds)
  }

  dialogVisible.value = true
}

const handleDelete = (row: any) => {
  Modal.warning({ title: '提示', content: '确定删除该任务？', hideCancel: false, onOk: async () => { await deleteInspectionTask(row.id); Message.success('删除成功'); loadData() } })
}

const handleToggle = async (row: any) => {
  try { await toggleInspectionTask(row.id); Message.success('操作成功'); loadData() } catch {}
}

const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return
  submitting.value = true
  try {
    // 构建请求数据
    const requestData: any = {
      name: formData.name,
      description: formData.description,
      task_type: formData.taskType,
      cron_expr: formData.cronExpr,
      enabled: formData.enabled,
      pushgateway_id: formData.pushgatewayId || 0,
      concurrency: formData.concurrency || 5,
      owner: formData.owner || ''
    }

    // 根据任务类型设置不同的配置
    if (formData.taskType === 'inspection') {
      requestData.group_ids = JSON.stringify(formData.inspectionGroupIds)
      requestData.item_ids = JSON.stringify(formData.inspectionItemIds)
    } else {
      requestData.group_ids = JSON.stringify([formData.groupId])
      requestData.item_ids = JSON.stringify(formData.probeConfigIds)
    }

    if (isEdit.value) {
      await updateInspectionTask(formData.id, requestData)
      Message.success('更新成功')
    } else {
      await createInspectionTask(requestData)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (error: any) {
    Message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const currentTaskType = ref<'probe' | 'inspection'>('probe')

const handleViewResults = (row: any) => {
  // 如果是巡检任务，直接跳转到执行记录页面
  if (row.taskType === 'inspection') {
    router.push({
      path: '/inspection/records',
      query: { taskId: row.id }
    })
    return
  }

  // 拨测任务显示抽屉
  currentTaskId.value = row.id
  currentTaskType.value = row.taskType || 'probe'
  resultPagination.page = 1
  resultsVisible.value = true
  loadResults()
}

const handleRun = async (row: any) => {
  try {
    await runInspectionTask(row.id)
    runningTaskIds.value = new Set([...runningTaskIds.value, row.id])
    Message.success('任务已开始执行')
    setTimeout(() => {
      runningTaskIds.value.delete(row.id)
      runningTaskIds.value = new Set(runningTaskIds.value)
      loadData()
    }, 3000)
  } catch (e: any) {
    Message.error(e?.response?.data?.message || '启动任务失败')
  }
}

const handleStop = async (row: any) => {
  try {
    await stopInspectionTask(row.id)
    runningTaskIds.value.delete(row.id)
    runningTaskIds.value = new Set(runningTaskIds.value)
    Message.success('任务已停止')
    loadData()
  } catch (e: any) {
    Message.error(e?.response?.data?.message || '停止任务失败')
  }
}

const loadResults = async () => {
  resultsLoading.value = true
  try {
    if (currentTaskType.value === 'inspection') {
      const res = await getInspectionTaskResults(currentTaskId.value, { page: resultPagination.page, page_size: resultPagination.pageSize })
      resultList.value = res.list || []
      resultPagination.total = res.total || 0
    } else {
      const res = await getTaskResults(currentTaskId.value, { page: resultPagination.page, page_size: resultPagination.pageSize })
      // 拨测任务使用 Pagination 响应格式，数据在 data.data 中
      resultList.value = res.data?.data || []
      resultPagination.total = res.data?.total || 0
    }
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

/* 运行中动画 */
@keyframes running-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
.running-text {
  animation: running-pulse 1.2s ease-in-out infinite;
  font-size: 12px;
}
</style>
