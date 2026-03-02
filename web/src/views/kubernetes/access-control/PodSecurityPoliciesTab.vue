<template>
  <div class="psp-tab">
    <div class="search-bar">
      <a-input v-model="searchName" placeholder="搜索 PodSecurityPolicy 名称..." allow-clear @input="handleSearch" class="search-input">
        <template #prefix>
          <icon-search />
        </template>
      </a-input>
    </div>

    <div class="table-wrapper">
      <a-table :data="paginatedList" :loading="loading" class="modern-table" :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }" :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <div class="name-icon-wrapper"><icon-lock /></div>
              <span class="name-text">{{ record.name }}</span>
            </div>
          </template>
          <template #actions="{ record }">
            <a-button type="text" @click="handleEdit(record)" class="action-btn">
              <icon-edit />
            </a-button>
            <a-button type="text" @click="handleDelete(record)" class="action-btn danger">
              <icon-delete />
            </a-button>
          </template>
        </a-table>
      <div class="pagination-wrapper">
        <a-pagination v-model:current="currentPage" v-model:page-size="pageSize" :page-size-options="[10, 20, 50]" :total="filteredData.length" show-total show-page-size show-jumper />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const tableColumns = [
  { title: '名称', slotName: 'name', width: 200, fixed: 'left' },
  { title: '存活时间', dataIndex: 'age', width: 140 },
  { title: '操作', slotName: 'actions', width: 100, fixed: 'right', align: 'center' }
]

import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getPodSecurityPolicies, type PodSecurityPolicyInfo } from '@/api/kubernetes'

interface Props {
  clusterId: number
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'count-update': [count: number]
}>()
const loading = ref(false)
const psps = ref<PodSecurityPolicyInfo[]>([])
const searchName = ref('')
const currentPage = ref(1)
const pageSize = ref(10)

const filteredData = computed(() => {
  let result = psps.value
  if (searchName.value) {
    result = result.filter(item => item.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  return result
})

const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredData.value.slice(start, start + pageSize.value)
})

const loadData = async () => {
  loading.value = true
  try {
    const data = await getPodSecurityPolicies(props.clusterId)
    psps.value = data || []
  } catch (error) {
    Message.error('获取 PodSecurityPolicy 列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { currentPage.value = 1 }

const handleEdit = (row: PodSecurityPolicyInfo) => {
  Message.info('编辑功能开发中...')
}

const handleDelete = (row: PodSecurityPolicyInfo) => {
  Message.info('删除功能开发中...')
}

const handleCreate = () => {
  Message.info('新增功能开发中...')
}

watch(() => props.clusterId, () => {
  if (props.clusterId) {
    loadData()
  }
}, { immediate: true })

// 监听筛选后的数据变化，更新计数
watch(filteredData, (newData) => {
  emit('count-update', newData?.length || 0)
})

// 暴露方法给父组件
defineExpose({
  handleCreate
})
</script>

<style scoped>
.psp-tab { width: 100%; }
.search-bar { margin-bottom: 16px; }
.search-input { width: 300px; }
.table-wrapper { background: #fff; border-radius: 8px; overflow: hidden; }
.name-cell { display: flex; align-items: center; gap: 10px; }
.name-icon-wrapper { width: 32px; height: 32px; border-radius: 6px; background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%); display: flex; align-items: center; justify-content: center; border: none; }
.name-icon { color: #165dff; }
.name-text { font-weight: 600; color: #303133; }
.action-btn { color: #165dff; margin: 0 4px; padding: 0; font-size: 16px; display: inline-flex; align-items: center; justify-content: center; }
.action-btn.danger { color: #f56c6c; }
.action-btn:hover { transform: scale(1.1); }
.pagination-wrapper { display: flex; justify-content: flex-end; padding: 16px 20px; border-top: 1px solid #f0f0f0; }
</style>
