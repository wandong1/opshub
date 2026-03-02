<template>
  <div class="role-detail" :loading="loading">
    <!-- 基本信息 -->
    <div class="section">
      <h3 class="section-title">基本信息</h3>
      <a-descriptions :column="2" :bordered="true">
        <a-descriptions-item label="角色名称">
          <a-tag type="primary" size="large">{{ role.name }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="命名空间">
          <a-tag v-if="role.namespace">{{ role.namespace }}</a-tag>
          <a-tag v-else color="green">集群级别</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="创建时间">
          {{ role.age }}
        </a-descriptions-item>
        <a-descriptions-item label="标签" v-if="Object.keys(role.labels || {}).length > 0">
          <a-tag
            v-for="(value, key) in role.labels"
            :key="key"
            size="small"
            style="margin-right: 4px;"
          >
            {{ key }}: {{ value }}
          </a-tag>
        </a-descriptions-item>
      </a-descriptions>
    </div>

    <!-- 权限规则 -->
    <div class="section">
      <h3 class="section-title">权限规则</h3>
      <a-tree
        :data="permissionTree"
        :props="treeProps"
        node-key="id"
        :default-expand-all="false"
        :expand-on-click-node="true"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <icon-folder v-if="data.type === 'apiGroup'" />
            <icon-file v-else-if="data.type === 'resource'" />
            <icon-settings v-else />
            <span class="node-label">{{ data.label }}</span>
            <a-tag v-if="data.type === 'verb'" size="small" color="gray">{{ data.value }}</a-tag>
          </span>
        </template>
      </a-tree>
    </div>

    <!-- 绑定的平台用户 -->
    <div class="section">
      <h3 class="section-title">
        绑定的平台用户
        <a-button type="primary" size="small" @click="showBindDialog = true" style="margin-left: 16px;">
          <icon-plus />
          绑定用户
        </a-button>
      </h3>
      <div class="user-bindings">
        <a-table :data="boundUsers" border stripe :columns="tableColumns">
          <template #boundAt="{ record }">
              {{ formatDate(record.boundAt) }}
            </template>
          <template #actions="{ record }">
              <a-button type="text" status="danger" @click="handleUnbind(record)">
                <icon-delete />
                解绑
              </a-button>
            </template>
        </a-table>
      </div>
    </div>

    <!-- 绑定用户对话框 -->
    <a-modal
      v-model:visible="showBindDialog"
      title="绑定用户到角色"
      width="600px"
      :mask-closable="false"
    >
      <a-form :model="bindForm" label-width="100px">
        <a-form-item label="搜索用户">
          <a-input
            v-model="userSearchKeyword"
            placeholder="输入用户名/姓名/邮箱搜索"
            clearable
            @clear="searchUsers"
            @keyup.enter="searchUsers"
          >
            <template #append>
              <a-button @click="searchUsers">
                <icon-search />
              </a-button>
            </template>
          </a-input>
        </a-form-item>

        <a-form-item label="选择用户">
          <a-select
            v-model="bindForm.userId"
            placeholder="请选择用户"
            filterable
            style="width: 100%"
          >
            <a-option
              v-for="user in availableUsers"
              :key="user.id"
              :label="`${user.username} (${user.realName})`"
              :value="user.id"
            >
              <div style="display: flex; justify-content: space-between;">
                <span>{{ user.username }}</span>
                <span style="color: #8492a6; font-size: 12px;">{{ user.realName }}</span>
              </div>
            </a-option>
          </a-select>
        </a-form-item>
      </a-form>

      <template #footer>
        <a-button @click="showBindDialog = false">取消</a-button>
        <a-button type="primary" @click="handleBind" :loading="bindLoading">确定</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { title: '用户名', dataIndex: 'username' },
  { title: '姓名', dataIndex: 'realName' },
  { title: '绑定时间', dataIndex: 'boundAt', slotName: 'boundAt' },
  { title: '操作', slotName: 'actions', width: 120 }
]

import { ref, onMounted, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getRoleDetail, bindUserToRole, unbindUserFromRole, getAvailableUsers, type BoundUser, type AvailableUser } from '@/api/kubernetes'

interface Role {
  name: string
  namespace?: string
  labels: Record<string, string>
  age: string
  rules: any[]
}

interface Props {
  clusterId: number
  role: Role
}

const props = defineProps<Props>()

const emit = defineEmits(['close'])

const loading = ref(false)
const permissionTree = ref<any[]>([])
const boundUsers = ref<BoundUser[]>([])

// 绑定用户对话框
const showBindDialog = ref(false)
const bindLoading = ref(false)
const userSearchKeyword = ref('')
const availableUsers = ref<AvailableUser[]>([])
const bindForm = ref({
  userId: 0 as number | undefined
})

const treeProps = {
  children: 'children',
  label: 'label'
}

// 计算角色类型
const roleType = computed(() => {
  return !props.role.namespace || props.role.namespace === '' ? 'ClusterRole' : 'Role'
})

// 加载角色详情
const loadRoleDetail = async () => {
  if (!props.clusterId || !props.role) return

  try {
    loading.value = true
    const detail = await getRoleDetail(
      props.clusterId,
      props.role.namespace || '',
      props.role.name
    )

    // 构建权限树
    permissionTree.value = buildPermissionTree(detail.rules || [])

    // 加载绑定的用户列表
    await loadBoundUsers()
  } catch (error: any) {
    Message.error(error.response?.data?.message || '加载角色详情失败')
  } finally {
    loading.value = false
  }
}

// 加载已绑定的用户列表
const loadBoundUsers = async () => {
  try {
    const users = await getRoleBoundUsers(
      props.clusterId,
      props.role.name,
      props.role.namespace || ''
    )
    boundUsers.value = users
  } catch (error: any) {
    Message.error('加载绑定用户失败')
  }
}

// 搜索可用用户
const searchUsers = async () => {
  try {
    const result = await getAvailableUsers(userSearchKeyword.value, 1, 50)
    availableUsers.value = result.list
  } catch (error: any) {
    Message.error('搜索用户失败')
  }
}

// 格式化日期
const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

// 构建权限树
const buildPermissionTree = (rules: any[]) => {
  const tree: any[] = []

  rules.forEach((rule, index) => {
    // API Groups - 注意后端返回的是 apiGroups (复数)
    const apiGroups = rule.apiGroups || ['']
    apiGroups.forEach((apiGroup: string, groupIndex: number) => {
      const apiGroupNode = {
        id: `apiGroup-${index}-${groupIndex}`,
        type: 'apiGroup',
        label: apiGroup || 'core',
        children: [] as any[]
      }

      // Resources
      const resources = rule.resources || []
      resources.forEach((resource: string, resIndex: number) => {
        const resourceNode = {
          id: `resource-${index}-${groupIndex}-${resIndex}`,
          type: 'resource',
          label: resource,
          children: [] as any[]
        }

        // Verbs
        const verbs = rule.verbs || ['*']
        verbs.forEach((verb: string, vIndex: number) => {
          const verbLabel = verb === '*' ? '所有操作' : verb
          resourceNode.children.push({
            id: `verb-${index}-${groupIndex}-${resIndex}-${vIndex}`,
            type: 'verb',
            label: '操作',
            value: verbLabel
          })
        })

        apiGroupNode.children.push(resourceNode)
      })

      tree.push(apiGroupNode)
    })
  })

  return tree
}

const handleUnbind = async (user: BoundUser) => {
  try {
    await confirmModal(
      `确定要解绑用户 "${user.username}" 的角色权限吗？`,
      '确认解绑',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      }
    )

    await unbindUserFromRole({
      clusterId: props.clusterId,
      userId: user.userId,
      roleName: props.role.name,
      roleNamespace: props.role.namespace || ''
    })
    Message.success('解绑成功')
    await loadBoundUsers()
  } catch (error) {
    if (error !== 'cancel') {
      Message.error('解绑失败')
    }
  }
}

// 绑定用户
const handleBind = async () => {
  if (!bindForm.value.userId) {
    Message.warning('请选择用户')
    return
  }

  try {
    bindLoading.value = true
    await bindUserToRole({
      clusterId: props.clusterId,
      userId: bindForm.value.userId,
      roleName: props.role.name,
      roleNamespace: props.role.namespace || '',
      roleType: roleType.value
    })

    Message.success('绑定成功')
    showBindDialog.value = false
    bindForm.value.userId = undefined
    userSearchKeyword.value = ''
    availableUsers.value = []

    await loadBoundUsers()
  } catch (error: any) {
    Message.error(error.response?.data?.message || '绑定失败')
  } finally {
    bindLoading.value = false
  }
}

// 监听对话框打开，自动加载用户列表
const handleDialogOpen = () => {
  if (showBindDialog.value && availableUsers.value.length === 0) {
    searchUsers()
  }
}

// 监听对话框显示状态
watch(() => showBindDialog.value, (newVal) => {
  if (newVal) {
    handleDialogOpen()
  }
})

onMounted(() => {
  loadRoleDetail()
})
</script>

<style scoped lang="scss">
.role-detail {
  .section {
    margin-bottom: 30px;

    .section-title {
      font-size: 16px;
      font-weight: 500;
      color: #333;
      margin-bottom: 16px;
      padding-bottom: 8px;
      border-bottom: 2px solid #165dff;
    }
  }

  .tree-node {
    display: flex;
    align-items: center;
    gap: 6px;

    .node-label {
      font-size: 14px;
    }
  }

  :deep(.arco-tree-node__content) {
    padding: 4px 0;
  }

  :deep(.arco-descriptions) {
    margin-top: 16px;
  }
}
</style>
