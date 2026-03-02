<template>
  <div class="namespace-roles">
    <!-- 命名空间选择器 -->
    <div class="namespace-selector">
      <a-select
        v-model="selectedNamespace"
        placeholder="选择命名空间"
        filterable
        @change="handleNamespaceChange"
        style="width: 300px"
      >
        <a-option
          v-for="ns in namespaces"
          :key="ns.name"
          :label="ns.name"
          :value="ns.name"
        >
          <span>{{ ns.name }}</span>
          <span style="color: #8492a6; font-size: 12px;">({{ ns.podCount }} pods)</span>
        </a-option>
      </a-select>
    </div>

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
          <template #namespace="{ record }">
          <a-tag size="small">{{ record.namespace }}</a-tag>
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
          <a-button type="text" status="danger" @click.stop="handleDelete(record)">
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
  { title: '命名空间', dataIndex: 'namespace', slotName: 'namespace', width: 180 },
  { title: '标签', dataIndex: 'labels', slotName: 'labels', width: 200 },
  { title: '创建时间', dataIndex: 'age', slotName: 'age', width: 180 },
  { title: '操作', slotName: 'actions', width: 150, fixed: 'right' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getNamespacesForRoles, getNamespaceRoles, createDefaultNamespaceRoles } from '@/api/kubernetes'

interface Namespace {
  name: string
  podCount: number
}

interface NamespaceRole {
  name: string
  namespace: string
  labels: Record<string, string>
  age: string
  rules: any[]
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
const selectedNamespace = ref('default')
const roleList = ref<NamespaceRole[]>([])
const namespaces = ref<Namespace[]>([])

// 过滤后的角色列表
const filteredRoles = computed(() => {
  let roles = roleList.value

  // 按命名空间过滤
  if (selectedNamespace.value) {
    roles = roles.filter(role => role.namespace === selectedNamespace.value)
  }

  // 按关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    roles = roles.filter(role =>
      role.name.toLowerCase().includes(keyword) ||
      Object.entries(role.labels).some(([key, value]) =>
        `${key}:${value}`.toLowerCase().includes(keyword)
      )
    )
  }

  return roles
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return

  try {
    const nsList = await getNamespacesForRoles(props.clusterId)
    namespaces.value = nsList.map(ns => ({
      name: ns.name,
      podCount: ns.podCount || 0
    }))
  } catch (error: any) {
    Message.error('加载命名空间失败')
  }
}

// 加载命名空间角色列表
const loadNamespaceRoles = async () => {
  if (!props.clusterId || !selectedNamespace.value) return

  try {
    loading.value = true
    let roles = await getNamespaceRoles(props.clusterId, selectedNamespace.value)

    // 定义应该有的12个默认命名空间角色
    const expectedNamespaceRoles = [
      'namespace-owner',
      'namespace-viewer',
      'manage-workload',
      'manage-config',
      'manage-rbac',
      'manage-service-discovery',
      'manage-storage',
      'view-workload',
      'view-config',
      'view-rbac',
      'view-service-discovery',
      'view-storage'
    ]

    // 如果角色数量不等于12，说明角色缺失，需要创建
    if (!roles || roles.length !== expectedNamespaceRoles.length) {
      try {
        await createDefaultNamespaceRoles(props.clusterId, selectedNamespace.value)
        // 重新加载角色列表
        roles = await getNamespaceRoles(props.clusterId, selectedNamespace.value)
      } catch (createError) {
      }
    }

    roleList.value = roles || []
  } catch (error: any) {
    Message.error(error.response?.data?.data?.message || '加载命名空间角色失败')
  } finally {
    loading.value = false
  }
}

const handleNamespaceChange = () => {
  loadNamespaceRoles()
}

const handleSearch = () => {
  // 搜索逻辑通过 computed 自动处理
}

const handleRowClick = (row: NamespaceRole) => {
  emit('role-click', row)
}

const handleViewDetail = (row: NamespaceRole) => {
  emit('role-click', row)
}

const handleDelete = async (row: NamespaceRole) => {
  try {
    await confirmModal(
      `确定要删除命名空间 "${row.namespace}" 中的角色 "${row.name}" 吗？`,
      '提示',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      }
    )

    Message.success('删除成功')
    await loadNamespaceRoles()
  } catch (error) {
    if (error !== 'cancel') {
      Message.error('删除失败')
    }
  }
}

onMounted(() => {
  loadNamespaces()
})

// 暴露刷新方法
defineExpose({
  refresh: loadNamespaceRoles
})
</script>

<style scoped lang="scss">
.namespace-roles {
  .namespace-selector {
    margin-bottom: 20px;
  }

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
