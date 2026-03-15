<template>
  <div class="inspection-records-container">
    <a-card class="search-card">
      <a-form :model="searchForm" layout="inline">
        <a-form-item label="巡检组">
          <a-select v-model="searchForm.groupId" placeholder="请选择巡检组" style="width: 200px" allow-clear>
            <a-option v-for="group in groupList" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="巡检项">
          <a-select v-model="searchForm.itemId" placeholder="请选择巡检项" style="width: 200px" allow-clear>
            <a-option v-for="item in itemList" :key="item.id" :value="item.id">
              {{ item.name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="主机">
          <a-select v-model="searchForm.hostId" placeholder="请选择主机" style="width: 200px" allow-clear>
            <a-option v-for="host in hostList" :key="host.id" :value="host.id">
              {{ host.name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model="searchForm.status" placeholder="请选择状态" style="width: 120px" allow-clear>
            <a-option value="success">成功</a-option>
            <a-option value="failed">失败</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="断言结果">
          <a-select v-model="searchForm.assertionResult" placeholder="请选择断言结果" style="width: 120px" allow-clear>
            <a-option value="pass">通过</a-option>
            <a-option value="fail">失败</a-option>
            <a-option value="skip">跳过</a-option>
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
        <span>执行记录</span>
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
          <a-tag :color="record.status === 'success' ? 'green' : 'red'">
            {{ record.status === 'success' ? '成功' : '失败' }}
          </a-tag>
        </template>

        <template #assertionResult="{ record }">
          <a-tag v-if="record.assertion_result === 'pass'" color="green">通过</a-tag>
          <a-tag v-else-if="record.assertion_result === 'fail'" color="red">失败</a-tag>
          <a-tag v-else color="gray">跳过</a-tag>
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
              下载报告
            </a-button>
          </a-space>
        </template>
      </a-table>
    </a-card>

    <!-- 详情弹窗 -->
    <a-modal
      v-model:visible="detailVisible"
      title="执行记录详情"
      width="800px"
      :footer="false"
    >
      <a-descriptions :column="2" bordered>
        <a-descriptions-item label="记录ID">{{ detailData.id }}</a-descriptions-item>
        <a-descriptions-item label="巡检组">{{ detailData.group_name }}</a-descriptions-item>
        <a-descriptions-item label="巡检项">{{ detailData.item_name }}</a-descriptions-item>
        <a-descriptions-item label="主机">{{ detailData.host_name }}</a-descriptions-item>
        <a-descriptions-item label="执行状态">
          <a-tag :color="detailData.status === 'success' ? 'green' : 'red'">
            {{ detailData.status === 'success' ? '成功' : '失败' }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="断言结果">
          <a-tag v-if="detailData.assertion_result === 'pass'" color="green">通过</a-tag>
          <a-tag v-else-if="detailData.assertion_result === 'fail'" color="red">失败</a-tag>
          <a-tag v-else color="gray">跳过</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="执行时长">
          {{ detailData.duration ? detailData.duration.toFixed(2) + 's' : '-' }}
        </a-descriptions-item>
        <a-descriptions-item label="执行时间" :span="2">{{ detailData.executed_at }}</a-descriptions-item>
        <a-descriptions-item label="输出内容" :span="2">
          <a-textarea v-model="detailData.output" :rows="8" readonly />
        </a-descriptions-item>
        <a-descriptions-item v-if="detailData.error_message" label="错误信息" :span="2">
          <a-textarea v-model="detailData.error_message" :rows="4" readonly />
        </a-descriptions-item>
        <a-descriptions-item v-if="detailData.assertion_details" label="断言详情" :span="2">
          <pre>{{ formatJSON(detailData.assertion_details) }}</pre>
        </a-descriptions-item>
        <a-descriptions-item v-if="detailData.extracted_variables" label="提取变量" :span="2">
          <pre>{{ formatJSON(detailData.extracted_variables) }}</pre>
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  getInspectionRecords,
  getInspectionRecord,
  getAllInspectionGroups,
  getInspectionItems,
  exportInspectionRecord
} from '@/api/inspectionManagement'
import { getHostList } from '@/api/host'

const loading = ref(false)
const tableData = ref([])
const groupList = ref([])
const itemList = ref([])
const hostList = ref([])
const detailVisible = ref(false)
const detailData = ref<any>({})

const searchForm = reactive({
  groupId: undefined,
  itemId: undefined,
  hostId: undefined,
  status: '',
  assertionResult: ''
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
  { title: '巡检组', dataIndex: 'group_name', width: 150 },
  { title: '巡检项', dataIndex: 'item_name', width: 150 },
  { title: '主机', dataIndex: 'host_name', width: 150 },
  { title: '执行状态', slotName: 'status', width: 100 },
  { title: '断言结果', slotName: 'assertionResult', width: 100 },
  { title: '执行时长', slotName: 'duration', width: 100 },
  { title: '执行时间', dataIndex: 'executed_at', width: 180 },
  { title: '操作', slotName: 'operations', width: 180, fixed: 'right' }
]

const formatJSON = (str: string) => {
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

const fetchGroups = async () => {
  try {
    const res = await getAllInspectionGroups()
    groupList.value = res
  } catch (error: any) {
    Message.error(error.message || '获取巡检组列表失败')
  }
}

const fetchItems = async () => {
  try {
    const res = await getInspectionItems({ page: 1, pageSize: 1000 })
    itemList.value = res.list
  } catch (error: any) {
    Message.error(error.message || '获取巡检项列表失败')
  }
}

const fetchHosts = async () => {
  try {
    const res = await getHostList({ page: 1, pageSize: 1000 })
    hostList.value = res.list
  } catch (error: any) {
    Message.error(error.message || '获取主机列表失败')
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getInspectionRecords({
      groupId: searchForm.groupId,
      itemId: searchForm.itemId,
      hostId: searchForm.hostId,
      status: searchForm.status,
      page: pagination.current,
      pageSize: pagination.pageSize
    })
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
  searchForm.groupId = undefined
  searchForm.itemId = undefined
  searchForm.hostId = undefined
  searchForm.status = ''
  searchForm.assertionResult = ''
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

const handleViewDetail = async (record: any) => {
  try {
    const res = await getInspectionRecord(record.id)
    detailData.value = res
    detailVisible.value = true
  } catch (error: any) {
    Message.error(error.message || '获取详情失败')
  }
}

const handleExport = async (record: any) => {
  try {
    const token = localStorage.getItem('token')
    const url = exportInspectionRecord(record.id)

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
    let filename = `inspection_record_${record.id}.xlsx`
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

onMounted(() => {
  fetchGroups()
  fetchItems()
  fetchHosts()
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
