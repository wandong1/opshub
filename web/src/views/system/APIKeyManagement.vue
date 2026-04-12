<template>
  <div class="apikey-management">
    <a-card :bordered="false">
      <div class="header-actions">
        <a-button type="primary" @click="handleCreate">
          <template #icon><icon-plus /></template>
          新增 API Key
        </a-button>
      </div>

      <a-table
        :columns="columns"
        :data="dataList"
        :loading="loading"
        :pagination="pagination"
        @page-change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      >
        <template #maskedKey="{ record }">
          <a-typography-text code>{{ record.maskedKey }}</a-typography-text>
        </template>

        <template #totalCalls="{ record }">
          <a-tag color="blue">{{ record.totalCalls }}</a-tag>
        </template>

        <template #lastCalledAt="{ record }">
          <span v-if="record.lastCalledAt && record.lastCalledAt !== '0001-01-01T00:00:00Z'">
            {{ formatDateTime(record.lastCalledAt) }}
          </span>
          <span v-else style="color: #86909c">从未调用</span>
        </template>

        <template #createdAt="{ record }">
          {{ formatDateTime(record.createdAt) }}
        </template>

        <template #action="{ record }">
          <a-popconfirm
            content="删除后该 API Key 将立即失效，无法恢复，确认删除？"
            @ok="handleDelete(record.id)"
          >
            <a-button type="text" status="danger" size="small">删除</a-button>
          </a-popconfirm>
        </template>
      </a-table>
    </a-card>

    <!-- 创建 API Key 弹窗 -->
    <a-modal
      v-model:visible="createVisible"
      title="新增 API Key"
      @ok="handleCreateSubmit"
      @cancel="handleCreateCancel"
      :ok-loading="createLoading"
    >
      <a-form :model="createForm" layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model="createForm.name" placeholder="请输入 API Key 名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea
            v-model="createForm.description"
            placeholder="请输入描述信息"
            :max-length="500"
            show-word-limit
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 显示完整密钥弹窗 -->
    <a-modal
      v-model:visible="keyVisible"
      title="API Key 创建成功"
      :footer="false"
      :closable="false"
      :mask-closable="false"
      width="600px"
    >
      <a-alert type="warning" style="margin-bottom: 16px">
        请妥善保管您的 API Key，关闭后将无法再次查看完整密钥！
      </a-alert>

      <div class="key-display">
        <a-typography-text code copyable>{{ fullAPIKey }}</a-typography-text>
      </div>

      <div style="margin-top: 16px; text-align: right">
        <a-button type="primary" @click="handleKeyClose">我已复制，关闭</a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'
import { createAPIKey, listAPIKeys, deleteAPIKey, type APIKey } from '@/api/apikey'
import dayjs from 'dayjs'

const loading = ref(false)
const dataList = ref<APIKey[]>([])
const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showTotal: true,
  showPageSize: true
})

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '名称', dataIndex: 'name', width: 150 },
  { title: 'API Key', slotName: 'maskedKey', width: 200 },
  { title: '描述', dataIndex: 'description', ellipsis: true, tooltip: true },
  { title: '调用次数', slotName: 'totalCalls', width: 120 },
  { title: '最后调用时间', slotName: 'lastCalledAt', width: 180 },
  { title: '创建时间', slotName: 'createdAt', width: 180 },
  { title: '操作', slotName: 'action', width: 100, fixed: 'right' }
]

// 创建表单
const createVisible = ref(false)
const createLoading = ref(false)
const createForm = reactive({
  name: '',
  description: ''
})

// 完整密钥显示
const keyVisible = ref(false)
const fullAPIKey = ref('')

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await listAPIKeys({
      page: pagination.current,
      page_size: pagination.pageSize
    })
    dataList.value = res.data.data || []
    pagination.total = res.data.total
  } catch (error: any) {
    Message.error(error.message || '加载失败')
  } finally {
    loading.value = false
  }
}

// 分页变化
const handlePageChange = (page: number) => {
  pagination.current = page
  loadData()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.current = 1
  loadData()
}

// 创建 API Key
const handleCreate = () => {
  createForm.name = ''
  createForm.description = ''
  createVisible.value = true
}

const handleCreateSubmit = async () => {
  if (!createForm.name.trim()) {
    Message.warning('请输入 API Key 名称')
    return
  }

  createLoading.value = true
  try {
    const res = await createAPIKey({
      name: createForm.name.trim(),
      description: createForm.description.trim()
    })

    Message.success('创建成功')
    createVisible.value = false

    // 显示完整密钥
    fullAPIKey.value = res.data.apiKey
    keyVisible.value = true

    // 刷新列表
    loadData()
  } catch (error: any) {
    Message.error(error.message || '创建失败')
  } finally {
    createLoading.value = false
  }
}

const handleCreateCancel = () => {
  createVisible.value = false
}

const handleKeyClose = () => {
  keyVisible.value = false
  fullAPIKey.value = ''
}

// 删除 API Key
const handleDelete = async (id: number) => {
  try {
    await deleteAPIKey(id)
    Message.success('删除成功')
    loadData()
  } catch (error: any) {
    Message.error(error.message || '删除失败')
  }
}

// 格式化时间
const formatDateTime = (dateStr: string) => {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '-'
  return dayjs(dateStr).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(() => {
  loadData()
})
</script>

<style scoped lang="less">
.apikey-management {
  padding: 20px;

  .header-actions {
    margin-bottom: 16px;
  }

  .key-display {
    padding: 16px;
    background: #f7f8fa;
    border-radius: 4px;
    word-break: break-all;
  }
}
</style>
