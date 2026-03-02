<template>
  <div class="service-labels-page-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-tag /></div>
        <div>
          <h2 class="page-title">服务标签管理</h2>
          <p class="page-subtitle">配置服务标签与进程匹配规则，Agent自动注册时根据运行进程匹配标签</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          新建标签
        </a-button>
      </div>
    </div>

    <div class="filter-bar">
      <div class="filter-inputs">
        <a-input
          v-model="searchForm.keyword"
          placeholder="搜索标签名称或匹配进程..."
          allow-clear
          style="width: 300px;"
          @press-enter="handleSearch"
          @clear="handleSearch"
        >
          <template #prefix><icon-search /></template>
        </a-input>
      </div>
      <div class="filter-actions">
        <a-button @click="handleReset"><template #icon><icon-refresh /></template>重置</a-button>
        <a-button @click="loadList"><template #icon><icon-refresh /></template>刷新</a-button>
      </div>
    </div>

    <div class="table-wrapper">
      <a-table
        :data="dataList"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        :pagination="{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showTotal: true,
          showPageSize: true,
          pageSizeOptions: [10, 20, 50]
        }"
        @page-change="(p: number) => { pagination.page = p; loadList() }"
        @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadList() }"
      >
        <template #columns>
          <a-table-column title="标签名称" data-index="name" :width="160">
            <template #cell="{ record }">
              <a-tag color="arcoblue">{{ record.name }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="匹配进程" data-index="matchProcesses" :min-width="300">
            <template #cell="{ record }">
              <a-space wrap>
                <a-tag v-for="proc in record.matchProcesses.split(',')" :key="proc" size="small" color="gray">{{ proc.trim() }}</a-tag>
              </a-space>
            </template>
          </a-table-column>
          <a-table-column title="描述" data-index="description" :min-width="200" />
          <a-table-column title="状态" data-index="status" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.status === 1 ? 'green' : 'gray'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="createTime" :width="180" />
          <a-table-column title="操作" :width="160" align="center" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" size="small" @click="handleEdit(record)">编辑</a-button>
                <a-popconfirm content="确定删除该标签？" @ok="handleDelete(record.id)">
                  <a-button type="text" status="danger" size="small">删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 新建/编辑弹窗 -->
    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑标签' : '新建标签'" :width="520" @ok="handleSubmit" @cancel="dialogVisible = false">
      <a-form ref="formRef" :model="form" :rules="rules" layout="vertical">
        <a-form-item label="标签名称" field="name">
          <a-input v-model="form.name" placeholder="如: mysqld, k8s-master" />
        </a-form-item>
        <a-form-item label="匹配进程" field="matchProcesses">
          <a-input v-model="form.matchProcesses" placeholder="多个进程名用逗号分隔，如: kube-apiserver,kube-scheduler" />
          <template #extra>Agent注册时会执行 ps 命令获取进程列表，与此处配置的进程名匹配</template>
        </a-form-item>
        <a-form-item label="描述" field="description">
          <a-textarea v-model="form.description" placeholder="标签描述" :max-length="500" />
        </a-form-item>
        <a-form-item label="状态" field="status">
          <a-switch v-model="form.status" :checked-value="1" :unchecked-value="0" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getServiceLabels, createServiceLabel, updateServiceLabel, deleteServiceLabel } from '@/api/serviceLabel'

const loading = ref(false)
const dataList = ref<any[]>([])
const searchForm = reactive({ keyword: '' })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })

const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref()
const form = reactive({ id: 0, name: '', matchProcesses: '', description: '', status: 1 })
const rules = {
  name: [{ required: true, message: '请输入标签名称' }],
  matchProcesses: [{ required: true, message: '请输入匹配进程' }]
}

const loadList = async () => {
  loading.value = true
  try {
    const res: any = await getServiceLabels({ page: pagination.page, pageSize: pagination.pageSize, keyword: searchForm.keyword })
    dataList.value = res?.list || []
    pagination.total = res?.total || 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { pagination.page = 1; loadList() }
const handleReset = () => { searchForm.keyword = ''; handleSearch() }

const handleAdd = () => {
  isEdit.value = false
  Object.assign(form, { id: 0, name: '', matchProcesses: '', description: '', status: 1 })
  dialogVisible.value = true
}

const handleEdit = (record: any) => {
  isEdit.value = true
  Object.assign(form, { id: record.id, name: record.name, matchProcesses: record.matchProcesses, description: record.description, status: record.status })
  dialogVisible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate()
  if (valid) return
  try {
    const data = { name: form.name, matchProcesses: form.matchProcesses, description: form.description, status: form.status }
    if (isEdit.value) {
      await updateServiceLabel(form.id, data)
      Message.success('更新成功')
    } else {
      await createServiceLabel(data)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadList()
  } catch (e: any) {
    Message.error(e.message || '操作失败')
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteServiceLabel(id)
    Message.success('删除成功')
    loadList()
  } catch (e: any) {
    Message.error(e.message || '删除失败')
  }
}

onMounted(() => loadList())
</script>

<style scoped>
.service-labels-page-container {
  padding: 0;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-title-group {
  display: flex;
  align-items: center;
  gap: 12px;
}
.page-title-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background: linear-gradient(135deg, #165dff 0%, #3b82f6 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
}
.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-1);
}
.page-subtitle {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--color-text-3);
}
.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.filter-inputs {
  display: flex;
  gap: 12px;
}
.filter-actions {
  display: flex;
  gap: 8px;
}
.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
}
</style>
