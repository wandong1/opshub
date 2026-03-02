<template>
  <div class="cluster-roles">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <a-form :inline="true">
        <a-form-item>
          <a-input
            v-model="searchKeyword"
            placeholder="搜索角色名称"
            clearable
            @clear="handleSearch"
            @keyup.enter="handleSearch"
            style="width: 240px"
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
        </a-form-item>

        <a-form-item>
          <a-button type="primary" @click="handleSearch">
            <icon-search />
            搜索
          </a-button>
        </a-form-item>
      </a-form>
    </div>

    <!-- 角色列表 -->
    <a-table
      :data="filteredRoles"
      border
      stripe
      :loading="loading"
      style="width: 100%"
      @row-click="handleRowClick"
     :columns="tableColumns">
          <template #name="{ record }">
          <span class="role-name-link">
            <icon-safe />
            {{ record.name }}
          </span>
        </template>
          <template #labels="{ record }">
          <a-tag
            v-for="(value, key) in record.labels"
            :key="key"
            size="small"
            style="margin-right: 4px; margin-bottom: 4px;"
          >
            {{ key }}: {{ value }}
          </a-tag>
        </template>
          <template #age="{ record }">
          {{ record.age }}
        </template>
          <template #actions="{ record }">
          <a-button type="text" status="normal" @click.stop="handleViewDetail(record)">
            <icon-eye />
          </a-button>
          <a-button type="text" status="danger" @click.stop="handleDelete(record)" v-if="record.isCustom">
            <icon-delete />
          </a-button>
        </template>
        </a-table>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { title: '角色名称', dataIndex: 'name', slotName: 'name', width: 250 },
  { title: '标签', dataIndex: 'labels', slotName: 'labels', width: 200 },
  { title: '创建时间', dataIndex: 'age', slotName: 'age', width: 180 },
  { title: '操作', slotName: 'actions', width: 150, fixed: 'right' }
]

import { ref, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getClusterRoles, deleteRole } from '@/api/kubernetes'

interface ClusterRole {
  name: string
  labels: Record<string, string>
  age: string
  rules: any[]
  isCustom: boolean
}

const props = defineProps({
  clusterId: {
    type: Number,
    required: true
  }
})

const emit = defineEmits(['role-click'])

const loading = ref(false)
const searchKeyword = ref('')
const roleList = ref<ClusterRole[]>([])

// 过滤后的角色列表
const filteredRoles = computed(() => {
  if (!searchKeyword.value) {
    return roleList.value
  }
  const keyword = searchKeyword.value.toLowerCase()
  return roleList.value.filter(role =>
    role.name.toLowerCase().includes(keyword) ||
    Object.entries(role.labels).some(([key, value]) =>
      `${key}:${value}`.toLowerCase().includes(keyword)
    )
  )
})

// 加载集群角色列表
const loadClusterRoles = async () => {
  if (!props.clusterId) return

  try {
    loading.value = true
    const roles = await getClusterRoles(props.clusterId)
    roleList.value = roles || []
  } catch (error: any) {
    Message.error(error.response?.data?.message || '加载集群角色失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  // 搜索逻辑通过 computed 自动处理
}

const handleRowClick = (row: ClusterRole) => {
  emit('role-click', row)
}

const handleViewDetail = (row: ClusterRole) => {
  emit('role-click', row)
}

const handleDelete = async (row: ClusterRole) => {
  try {
    await confirmModal(`确定要删除角色 "${row.name}" 吗？`, '提示', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })

    // 集群角色的 namespace 为空字符串
    await deleteRole(props.clusterId, '', row.name)
    Message.success('删除成功')
    await loadClusterRoles()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '删除失败')
    }
  }
}

onMounted(() => {
  loadClusterRoles()
})

// 暴露刷新方法供父组件调用
defineExpose({
  refresh: loadClusterRoles
})
</script>

<style scoped lang="scss">
.cluster-roles {
  .search-bar {
    margin-bottom: 20px;
    padding: 16px;
    background: #f5f5f5;
    border-radius: 8px;
  }

  .role-name-link {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    color: #165dff;
    font-weight: 500;

    &:hover {
      text-decoration: underline;
    }
  }

  :deep(.arco-table) {
    cursor: pointer;

    .arco-table__row:hover {
      background-color: #f5f7fa;
    }
  }
}
</style>
