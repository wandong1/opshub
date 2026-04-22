<template>
  <div class="inspection-records-container">
    <a-card class="search-card">
      <a-form :model="searchForm" layout="inline">
        <a-form-item label="任务名称">
          <a-select v-model="searchForm.taskId" placeholder="请选择任务" style="width: 200px" allow-clear>
            <a-option v-for="task in taskList" :key="task.id" :value="task.id">
              {{ task.name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="执行状态">
          <a-select v-model="searchForm.status" placeholder="请选择状态" style="width: 120px" allow-clear>
            <a-option value="running">执行中</a-option>
            <a-option value="success">成功</a-option>
            <a-option value="failed">失败</a-option>
            <a-option value="partial">部分成功</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="开始时间">
          <a-range-picker v-model="searchForm.timeRange" style="width: 300px" show-time />
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
        <span>巡检执行记录</span>
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
          <a-tag v-if="record.status === 'running'" color="blue">执行中</a-tag>
          <a-tag v-else-if="record.status === 'success'" color="green">成功</a-tag>
          <a-tag v-else-if="record.status === 'failed'" color="red">失败</a-tag>
          <a-tag v-else color="orange">部分成功</a-tag>
        </template>

        <template #groupNames="{ record }">
          <a-space wrap>
            <a-tag v-for="(name, index) in record.groupNames" :key="index" color="arcoblue">
              {{ name }}
            </a-tag>
          </a-space>
        </template>

        <template #statistics="{ record }">
          <a-space direction="vertical" size="mini">
            <span>总执行: {{ record.totalExecutions }}</span>
            <span>成功: <span style="color: #00b42a">{{ record.successCount }}</span> / 失败: <span style="color: #f53f3f">{{ record.failedCount }}</span></span>
          </a-space>
        </template>

        <template #duration="{ record }">
          {{ record.duration ? record.duration.toFixed(2) + 's' : '-' }}
        </template>

        <template #operations="{ record }">
          <a-space>
            <a-button type="text" size="small" @click="handleViewDetail(record)">
              查看详情
            </a-button>
            <a-button type="text" size="small" @click="handleExport(record)">
              导出报告
            </a-button>
            <a-popconfirm content="确定删除该记录吗？" @ok="handleDelete(record.id)">
              <a-button type="text" size="small" status="danger">
                删除
              </a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </a-table>
    </a-card>

    <!-- 详情弹窗 -->
    <a-modal
      v-model:visible="detailVisible"
      title="巡检执行记录详情"
      width="1200px"
      :footer="false"
    >
      <a-tabs default-active-key="overview">
        <a-tab-pane key="overview" title="执行概览">
          <a-descriptions :column="2" bordered>
            <a-descriptions-item label="记录ID">{{ detailData.id }}</a-descriptions-item>
            <a-descriptions-item label="任务名称">{{ detailData.taskName }}</a-descriptions-item>
            <a-descriptions-item label="执行状态">
              <a-tag v-if="detailData.status === 'running'" color="blue">执行中</a-tag>
              <a-tag v-else-if="detailData.status === 'success'" color="green">成功</a-tag>
              <a-tag v-else-if="detailData.status === 'failed'" color="red">失败</a-tag>
              <a-tag v-else color="orange">部分成功</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="执行时长">
              {{ detailData.duration ? detailData.duration.toFixed(2) + 's' : '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="总巡检项数">{{ detailData.totalItems }}</a-descriptions-item>
            <a-descriptions-item label="总主机数">{{ detailData.totalHosts }}</a-descriptions-item>
            <a-descriptions-item label="总执行次数">{{ detailData.totalExecutions }}</a-descriptions-item>
            <a-descriptions-item label="成功次数">
              <span style="color: #00b42a">{{ detailData.successCount }}</span>
            </a-descriptions-item>
            <a-descriptions-item label="失败次数">
              <span style="color: #f53f3f">{{ detailData.failedCount }}</span>
            </a-descriptions-item>
            <a-descriptions-item label="断言通过">
              <span style="color: #00b42a">{{ detailData.assertionPassCount }}</span>
            </a-descriptions-item>
            <a-descriptions-item label="断言失败">
              <span style="color: #f53f3f">{{ detailData.assertionFailCount }}</span>
            </a-descriptions-item>
            <a-descriptions-item label="断言跳过">
              <span style="color: #86909c">{{ detailData.assertionSkipCount }}</span>
            </a-descriptions-item>
            <a-descriptions-item label="开始时间" :span="2">{{ detailData.startedAt }}</a-descriptions-item>
            <a-descriptions-item label="完成时间" :span="2">{{ detailData.completedAt || '-' }}</a-descriptions-item>
            <a-descriptions-item label="巡检组" :span="2">
              <a-space wrap>
                <a-tag v-for="(name, index) in detailData.groupNames" :key="index" color="arcoblue">
                  {{ name }}
                </a-tag>
              </a-space>
            </a-descriptions-item>
          </a-descriptions>
        </a-tab-pane>

        <a-tab-pane key="details" title="执行明细">
          <!-- 筛选条件 -->
          <div style="margin-bottom: 16px;">
            <a-space>
              <a-select v-model="detailFilter.status" placeholder="执行状态" allow-clear style="width: 120px;" @change="filterDetailData">
                <a-option value="success">成功</a-option>
                <a-option value="failed">失败</a-option>
              </a-select>
              <a-select v-model="detailFilter.assertionResult" placeholder="断言结果" allow-clear style="width: 120px;" @change="filterDetailData">
                <a-option value="pass">通过</a-option>
                <a-option value="fail">失败</a-option>
                <a-option value="skip">跳过</a-option>
              </a-select>
              <a-button @click="resetDetailFilter">重置</a-button>
            </a-space>
          </div>

          <a-table
            :columns="detailColumns"
            :data="filteredDetailTableData"
            :loading="detailLoading"
            :pagination="{ pageSize: 20, showTotal: true }"
            size="small"
          >
            <template #inspectionLevel="{ record }">
              <a-tag v-if="record.inspectionLevel" size="small" :color="getLevelColor(record.inspectionLevel)">
                {{ getLevelText(record.inspectionLevel) }}
              </a-tag>
              <span v-else>-</span>
            </template>

            <template #riskLevel="{ record }">
              <a-tag v-if="record.riskLevel" size="small" :color="getRiskColor(record.riskLevel)">
                {{ getLevelText(record.riskLevel) }}
              </a-tag>
              <span v-else>-</span>
            </template>

            <template #executionType="{ record }">
              <a-tag v-if="record.executionType === 'command'" size="small" color="blue">命令</a-tag>
              <a-tag v-else-if="record.executionType === 'script'" size="small" color="purple">脚本</a-tag>
              <a-tag v-else-if="record.executionType === 'probe'" size="small" color="cyan">拨测</a-tag>
              <a-tag v-else-if="record.executionType === 'promql'" size="small" color="orange">PromQL</a-tag>
              <span v-else>-</span>
            </template>

            <template #status="{ record }">
              <a-tag :color="record.status === 'success' ? 'green' : 'red'">
                {{ record.status === 'success' ? '成功' : '失败' }}
              </a-tag>
            </template>

            <template #assertionResult="{ record }">
              <a-tag v-if="record.assertionResult === 'pass'" color="green">通过</a-tag>
              <a-tag v-else-if="record.assertionResult === 'fail'" color="red">失败</a-tag>
              <a-tag v-else color="gray">跳过</a-tag>
            </template>

            <template #duration="{ record }">
              {{ record.duration ? record.duration.toFixed(2) + 's' : '-' }}
            </template>

            <template #operations="{ record }">
              <a-button type="text" size="mini" @click="handleViewDetailItem(record)">
                查看
              </a-button>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-modal>

    <!-- 明细详情弹窗 -->
    <a-modal
      v-model:visible="detailItemVisible"
      title="执行明细详情"
      width="900px"
      :footer="false"
    >
      <a-descriptions :column="2" bordered>
        <a-descriptions-item label="巡检组">{{ detailItemData.groupName }}</a-descriptions-item>
        <a-descriptions-item label="巡检项">{{ detailItemData.itemName }}</a-descriptions-item>
        <a-descriptions-item label="巡检级别">
          <a-tag v-if="detailItemData.inspectionLevel" size="small" :color="getLevelColor(detailItemData.inspectionLevel)">
            {{ getLevelText(detailItemData.inspectionLevel) }}
          </a-tag>
          <span v-else>-</span>
        </a-descriptions-item>
        <a-descriptions-item label="风险等级">
          <a-tag v-if="detailItemData.riskLevel" size="small" :color="getRiskColor(detailItemData.riskLevel)">
            {{ getLevelText(detailItemData.riskLevel) }}
          </a-tag>
          <span v-else>-</span>
        </a-descriptions-item>
        <a-descriptions-item label="主机">{{ detailItemData.hostName }}</a-descriptions-item>
        <a-descriptions-item label="主机IP">{{ detailItemData.hostIp }}</a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.businessGroup" label="业务分组">{{ detailItemData.businessGroup }}</a-descriptions-item>
        <a-descriptions-item label="执行类型">
          <a-tag v-if="detailItemData.executionType === 'command'" size="small" color="blue">命令</a-tag>
          <a-tag v-else-if="detailItemData.executionType === 'script'" size="small" color="purple">脚本</a-tag>
          <a-tag v-else-if="detailItemData.executionType === 'probe'" size="small" color="cyan">拨测</a-tag>
          <a-tag v-else-if="detailItemData.executionType === 'promql'" size="small" color="orange">PromQL</a-tag>
          <span v-else>-</span>
        </a-descriptions-item>
        <a-descriptions-item label="执行方式">{{ detailItemData.executionMode || '-' }}</a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.command" label="执行命令" :span="2">
          <pre style="margin: 0; padding: 8px; background: #f5f5f5; border-radius: 4px; font-family: monospace; white-space: pre-wrap;">{{ detailItemData.command }}</pre>
        </a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.scriptType" label="脚本类型">{{ detailItemData.scriptType }}</a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.scriptContent" label="脚本内容" :span="2">
          <pre style="margin: 0; padding: 8px; background: #f5f5f5; border-radius: 4px; font-family: monospace; white-space: pre-wrap; max-height: 200px; overflow-y: auto;">{{ detailItemData.scriptContent }}</pre>
        </a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.assertionType" label="断言类型">{{ detailItemData.assertionType }}</a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.assertionValue" label="断言表达式" :span="2">
          <pre style="margin: 0; padding: 8px; background: #f5f5f5; border-radius: 4px; font-family: monospace; white-space: pre-wrap;">{{ detailItemData.assertionValue }}</pre>
        </a-descriptions-item>
        <a-descriptions-item label="执行状态">
          <a-tag :color="detailItemData.status === 'success' ? 'green' : 'red'">
            {{ detailItemData.status === 'success' ? '成功' : '失败' }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="断言结果">
          <a-tag v-if="detailItemData.assertionResult === 'pass'" color="green">通过</a-tag>
          <a-tag v-else-if="detailItemData.assertionResult === 'fail'" color="red">失败</a-tag>
          <a-tag v-else color="gray">跳过</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="执行时长">
          {{ detailItemData.duration ? detailItemData.duration.toFixed(2) + 's' : '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="执行时间">{{ detailItemData.executedAt }}</a-descriptions-item>
        <a-descriptions-item label="输出内容" :span="2">
          <a-textarea v-model="detailItemData.output" :rows="8" readonly />
        </a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.errorMessage" label="错误信息" :span="2">
          <a-textarea v-model="detailItemData.errorMessage" :rows="4" readonly />
        </a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.assertionDetails" label="断言详情" :span="2">
          <pre>{{ formatJSON(detailItemData.assertionDetails) }}</pre>
        </a-descriptions-item>
        <a-descriptions-item v-if="detailItemData.extractedVariables" label="提取变量" :span="2">
          <pre>{{ formatJSON(detailItemData.extractedVariables) }}</pre>
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { useRoute } from 'vue-router'
import {
  getExecutionRecords,
  getExecutionRecord,
  getExecutionDetails,
  deleteExecutionRecord,
  exportExecutionReport,
  type ExecutionRecord,
  type ExecutionDetail
} from '@/api/inspectionManagement'
import { getInspectionTasks } from '@/api/inspectionTask'

const route = useRoute()

const loading = ref(false)
const tableData = ref<ExecutionRecord[]>([])
const taskList = ref([])
const detailVisible = ref(false)
const detailData = ref<ExecutionRecord>({} as ExecutionRecord)
const detailLoading = ref(false)
const detailTableData = ref<ExecutionDetail[]>([])
const detailItemVisible = ref(false)
const detailItemData = ref<ExecutionDetail>({} as ExecutionDetail)

const searchForm = reactive({
  taskId: undefined as number | undefined,
  status: '',
  timeRange: []
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: true,
  showPageSize: true
})

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '任务名称', dataIndex: 'taskName', width: 200 },
  { title: '巡检组', slotName: 'groupNames', width: 250 },
  { title: '执行统计', slotName: 'statistics', width: 150 },
  { title: '执行状态', slotName: 'status', width: 100 },
  { title: '执行时长', slotName: 'duration', width: 100 },
  { title: '开始时间', dataIndex: 'startedAt', width: 180 },
  { title: '操作', slotName: 'operations', width: 220, fixed: 'right' }
]

const detailColumns = [
  { title: '巡检组', dataIndex: 'groupName', width: 120 },
  { title: '巡检项', dataIndex: 'itemName', width: 150 },
  { title: '巡检级别', slotName: 'inspectionLevel', width: 90 },
  { title: '风险等级', slotName: 'riskLevel', width: 90 },
  { title: '主机', dataIndex: 'hostName', width: 120 },
  { title: '主机IP', dataIndex: 'hostIp', width: 130 },
  { title: '业务分组', dataIndex: 'businessGroup', width: 120 },
  { title: '执行类型', slotName: 'executionType', width: 90 },
  { title: '执行状态', slotName: 'status', width: 90 },
  { title: '断言结果', slotName: 'assertionResult', width: 90 },
  { title: '执行时长', slotName: 'duration', width: 90 },
  { title: '执行时间', dataIndex: 'executedAt', width: 170 },
  { title: '操作', slotName: 'operations', width: 80, fixed: 'right' }
]

const detailFilter = reactive({
  status: '',
  assertionResult: ''
})

const filteredDetailTableData = ref<ExecutionDetail[]>([])

const filterDetailData = () => {
  let filtered = [...detailTableData.value]

  if (detailFilter.status) {
    filtered = filtered.filter(item => item.status === detailFilter.status)
  }

  if (detailFilter.assertionResult) {
    filtered = filtered.filter(item => item.assertionResult === detailFilter.assertionResult)
  }

  filteredDetailTableData.value = filtered
}

const resetDetailFilter = () => {
  detailFilter.status = ''
  detailFilter.assertionResult = ''
  filteredDetailTableData.value = [...detailTableData.value]
}

const formatJSON = (str: string) => {
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

const fetchTasks = async () => {
  try {
    const res = await getInspectionTasks({ page: 1, page_size: 100 })
    taskList.value = res.list || []
  } catch (error: any) {
    Message.error(error.message || '获取任务列表失败')
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const params: any = {
      taskId: searchForm.taskId,
      status: searchForm.status,
      page: pagination.current,
      pageSize: pagination.pageSize
    }

    if (searchForm.timeRange && searchForm.timeRange.length === 2) {
      params.startTime = searchForm.timeRange[0]
      params.endTime = searchForm.timeRange[1]
    }

    const res = await getExecutionRecords(params)
    tableData.value = res.list
    pagination.total = res.total
  } catch (error: any) {
    Message.error(error.message || '获取数据失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.current = 1
  fetchData()
}

const handleReset = () => {
  searchForm.taskId = undefined
  searchForm.status = ''
  searchForm.timeRange = []
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

const handleViewDetail = async (record: ExecutionRecord) => {
  try {
    const res = await getExecutionRecord(record.id)
    detailData.value = res
    detailVisible.value = true

    // 加载执行明细
    detailLoading.value = true
    const details = await getExecutionDetails(record.id)
    detailTableData.value = details
    filteredDetailTableData.value = [...details]
    // 重置筛选条件
    detailFilter.status = ''
    detailFilter.assertionResult = ''
  } catch (error: any) {
    Message.error(error.message || '获取详情失败')
  } finally {
    detailLoading.value = false
  }
}

const handleViewDetailItem = (record: ExecutionDetail) => {
  detailItemData.value = record
  detailItemVisible.value = true
}

const handleExport = async (record: ExecutionRecord) => {
  try {
    const token = localStorage.getItem('srehubtoken')
    const url = exportExecutionReport(record.id)

    // 使用 fetch 下载文件，携带 token
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })

    if (!response.ok) {
      throw new Error('下载失败')
    }

    // 获取文件名
    const contentDisposition = response.headers.get('Content-Disposition')
    let filename = `inspection_execution_report_${record.id}.xlsx`
    if (contentDisposition) {
      const matches = /filename=([^;]+)/.exec(contentDisposition)
      if (matches && matches[1]) {
        filename = matches[1]
      }
    }

    // 下载文件
    const blob = await response.blob()
    const downloadUrl = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = downloadUrl
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(downloadUrl)

    Message.success('下载成功')
  } catch (error: any) {
    Message.error(error.message || '下载失败')
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteExecutionRecord(id)
    Message.success('删除成功')
    fetchData()
  } catch (error: any) {
    Message.error(error.message || '删除失败')
  }
}

const getLevelText = (level: string) => {
  const map: Record<string, string> = {
    high: '高',
    medium: '中',
    low: '低'
  }
  return map[level] || '中'
}

const getLevelColor = (level: string) => {
  const map: Record<string, string> = {
    high: 'red',
    medium: 'orange',
    low: 'green'
  }
  return map[level] || 'orange'
}

const getRiskColor = (level: string) => {
  const map: Record<string, string> = {
    high: 'red',
    medium: 'orangered',
    low: 'blue'
  }
  return map[level] || 'orangered'
}

onMounted(() => {
  // 从 URL 参数中获取 taskId
  const taskIdFromQuery = route.query.taskId
  if (taskIdFromQuery) {
    searchForm.taskId = Number(taskIdFromQuery)
  }

  fetchTasks()
  fetchData()
})
</script>

<style scoped>
.inspection-records-container {
  padding: 20px;
}

.search-card {
  margin-bottom: 16px;
}

.table-card {
  background: #fff;
}

pre {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
}
</style>
