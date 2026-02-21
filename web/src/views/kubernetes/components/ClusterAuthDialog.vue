<template>
  <div class="cluster-auth-content" v-if="modelValue">
    <!-- 操作栏 -->
    <div class="action-bar">
      <div class="page-info">
        <div class="info-icon">
          <icon-user />
        </div>
        <div class="info-text">
          <div class="info-title">已申请凭据的用户</div>
          <div class="info-desc">共 {{ uniqueCredentialUsers.length }} 位用户已申请该集群的 kubeconfig 访问凭据</div>
        </div>
      </div>
      <a-button @click="handleRefresh" :loading="loading">
        <icon-refresh />
        刷新
      </a-button>
    </div>

    <!-- 用户表格 -->
    <div class="table-wrapper">
      <a-table
        :data="uniqueCredentialUsers"
        :loading="loading"
        class="user-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
       :columns="tableColumns">
          <template #col_0="{ record }">
            <div class="table-avatar">
              <icon-user />
            </div>
          </template>
          <template #realName="{ record }">
            <div class="user-cell">
              <div class="user-name">{{ record.realName || record.username }}</div>
              <div class="user-username">@{{ record.username }}</div>
            </div>
          </template>
          <template #serviceAccount="{ record }">
            <span class="mono-text">{{ record.serviceAccount }}</span>
          </template>
          <template #namespace="{ record }">
            <span class="mono-text">{{ record.namespace }}</span>
          </template>
          <template #createdAt="{ record }">
            <span class="time-text">{{ formatDate(record.createdAt) }}</span>
          </template>
          <template #status="{ record }">
            <a-tag color="green" size="small">
              <span class="status-dot status-dot-active"></span>
              凭据已申请
            </a-tag>
            <a-tag
              v-if="userRoleCounts[record.userId] > 0"
              color="arcoblue"
              size="small"
              style="margin-left: 4px;"
            >
              {{ userRoleCounts[record.userId] }} 个角色
            </a-tag>
            <a-tag
              v-else-if="userRoleCounts[record.userId] === 0"
              color="orangered"
              size="small"
              style="margin-left: 4px;"
            >
              未分配角色
            </a-tag>
          </template>
          <template #actions="{ record }">
            <a-space>
              <a-button type="text" size="small" class="action-authorize" @click="handleAuthorize(record)">
                <template #icon><icon-safe /></template>
                授权角色
              </a-button>
              <a-button type="text" size="small" class="action-view" @click="handleViewCredential(record)">
                <template #icon><icon-file /></template>
                凭据
              </a-button>
              <a-button type="text" size="small" class="action-revoke" @click="handleRevoke(record)">
                <template #icon><icon-delete /></template>
                吊销
              </a-button>
            </a-space>
          </template>
        </a-table>
    </div>

    <a-empty
      v-if="!loading && !uniqueCredentialUsers.length"
      description="暂无用户申请凭据"
      :image-size="100"
    />

    <!-- KubeConfig 查看对话框 -->
    <a-modal
      v-model:visible="showKubeConfigDialog"
      title="查看 KubeConfig 凭据"
      width="800px"
      :render-to-body="true"
    >
      <div class="kubeconfig-dialog">
        <div class="kubeconfig-info">
          <a-descriptions :column="2" :bordered="true">
            <a-descriptions-item label="用户名">{{ currentUser?.username }}</a-descriptions-item>
            <a-descriptions-item label="真实姓名">{{ currentUser?.realName }}</a-descriptions-item>
            <a-descriptions-item label="ServiceAccount">{{ currentUser?.serviceAccount }}</a-descriptions-item>
            <a-descriptions-item label="命名空间">{{ currentUser?.namespace }}</a-descriptions-item>
          </a-descriptions>
        </div>

        <div class="kubeconfig-actions">
          <a-button type="primary" @click="handleCopyKubeConfig">
            <icon-copy />
            复制
          </a-button>
          <a-button @click="handleDownloadKubeConfig">
            <icon-download />
            下载
          </a-button>
        </div>

        <div class="code-editor-wrapper">
          <div class="line-numbers">
            <div v-for="n in configLineCount" :key="n" class="line-number">{{ n }}</div>
          </div>
          <textarea
            v-model="currentKubeConfig"
            class="code-textarea"
            readonly
            spellcheck="false"
          ></textarea>
        </div>

        <div class="code-tip">
          <icon-exclamation-circle />
          <span>此凭据文件包含您的集群访问权限，请妥善保管，不要泄露给他人</span>
        </div>
      </div>
    </a-modal>

    <!-- 授权对话框 -->
    <a-modal
      v-model:visible="showAuthorizeDialog"
      title="授予用户权限"
      width="800px"
      :render-to-body="true"
      @close="handleAuthorizeDialogClose"
    >
      <div class="authorize-dialog">
        <a-spin :loading="authorizeLoading" style="width: 100%">
        <!-- 用户信息 -->
        <div class="user-info-section">
          <div class="user-info-header">
            <icon-user />
            <span>用户信息</span>
          </div>
          <div class="user-info-content">
            <div class="info-row">
              <span class="info-label">用户名:</span>
              <span class="info-value">{{ authorizeUser?.realName || authorizeUser?.username }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">账号:</span>
              <span class="info-value">@{{ authorizeUser?.username }}</span>
            </div>
          </div>
        </div>

        <!-- 已有权限 -->
        <div class="existing-permissions-section">
          <div class="section-header">
            <icon-safe />
            <span>角色</span>
          </div>

          <!-- 集群级别权限 -->
          <div class="permission-group" v-if="existingClusterRoles.length > 0">
            <div class="permission-group-title">集群级别</div>
            <div class="permission-tags">
              <a-tag
                v-for="role in existingClusterRoles"
                :key="role.roleName"
                type="danger"
                closable
                @close="handleRemoveExistingRole(role)"
              >
                {{ role.roleName }}
              </a-tag>
            </div>
          </div>

          <!-- 命名空间级别权限 -->
          <div class="permission-group" v-show="existingNamespacePermissions.length > 0">
            <div class="permission-group-title">
              命名空间级别
            </div>
            <div class="namespace-permissions">
              <div
                v-for="nsPerm in existingNamespacePermissions"
                :key="nsPerm.namespace"
                class="namespace-permission-item"
              >
                <div class="namespace-name">{{ nsPerm.namespace }}</div>
                <div class="namespace-roles">
                  <a-tag
                    v-for="role in nsPerm.roles"
                    :key="role.id"
                    type="primary"
                    closable
                    @close="handleRemoveExistingRole(role)"
                  >
                    {{ role.roleName }}
                  </a-tag>
                </div>
              </div>
            </div>
          </div>

          <a-empty
            v-if="existingClusterRoles.length === 0 && existingNamespacePermissions.length === 0"
            description="暂无权限"
            :image-size="60"
          />
        </div>

        <!-- 添加新权限 -->
        <div class="add-permission-section">
          <div class="section-header">
            <icon-plus />
            <span>添加新权限</span>
          </div>

          <a-form :model="authorizeForm" label-width="100px">
            <!-- 权限级别 -->
            <a-form-item label="权限级别">
              <a-radio-group v-model="authorizeForm.permissionLevel" @change="handlePermissionLevelChange">
                <a-radio value="cluster">集群级别</a-radio>
                <a-radio value="namespace">命名空间级别</a-radio>
              </a-radio-group>
            </a-form-item>

            <!-- 集群角色选择 -->
            <template v-if="authorizeForm.permissionLevel === 'cluster'">
              <a-form-item label="集群角色">
                <a-select
                  v-model="authorizeForm.clusterRoleNames"
                  placeholder="请选择集群角色"
                  filterable
                  multiple
                  @focus="loadClusterRoles"
                  :loading="loadingRoles"
                  style="width: 100%"
                >
                  <a-option
                    v-for="role in clusterRoles"
                    :key="role.name"
                    :label="role.name"
                    :value="role.name"
                  >
                    <div class="role-option">
                      <icon-safe />
                      <span>{{ role.name }}</span>
                    </div>
                  </a-option>
                </a-select>
              </a-form-item>
            </template>

            <!-- 命名空间角色选择 -->
            <template v-if="authorizeForm.permissionLevel === 'namespace'">
              <a-form-item label="命名空间">
                <a-select
                  v-model="authorizeForm.namespace"
                  placeholder="请选择命名空间"
                  filterable
                  @focus="loadNamespaces"
                  @change="handleNamespaceChange"
                  :loading="loadingNamespaces"
                  style="width: 100%"
                >
                  <a-option
                    v-for="ns in namespaces"
                    :key="ns.name"
                    :label="ns.name"
                    :value="ns.name"
                  >
                    <div class="namespace-option">
                      <icon-folder />
                      <span>{{ ns.name }}</span>
                    </div>
                  </a-option>
                </a-select>
              </a-form-item>

              <a-form-item label="角色">
                <a-select
                  v-model="authorizeForm.namespaceRoleNames"
                  placeholder="请先选择命名空间"
                  filterable
                  multiple
                  :disabled="!authorizeForm.namespace"
                  @focus="loadNamespaceRoles"
                  :loading="loadingRoles"
                  style="width: 100%"
                >
                  <a-option
                    v-for="role in namespaceRoles"
                    :key="role.name"
                    :label="role.name"
                    :value="role.name"
                  >
                    <div class="role-option">
                      <icon-safe />
                      <span>{{ role.name }}</span>
                    </div>
                  </a-option>
                </a-select>
              </a-form-item>
            </template>
          </a-form>
        </div>
        </a-spin>
      </div>

      <template #footer>
        <a-button @click="showAuthorizeDialog = false">取消</a-button>
        <a-button type="primary" @click="handleConfirmAuthorize" :loading="authorizeLoading">
          确认授权
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { slotName: 'col_0', width: 60 },
  { title: '用户名', dataIndex: 'realName', slotName: 'realName', width: 120 },
  { title: 'ServiceAccount', dataIndex: 'serviceAccount', slotName: 'serviceAccount', width: 180 },
  { title: '命名空间', dataIndex: 'namespace', slotName: 'namespace', width: 150 },
  { title: '申请时间', dataIndex: 'createdAt', slotName: 'createdAt', width: 170 },
  { title: '状态', slotName: 'status', width: 200 },
  { title: '操作', slotName: 'actions', width: 260, fixed: 'right' }
]

import { ref, computed, watch, nextTick } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  getServiceAccountKubeConfig,
  getClusterCredentialUsers,
  revokeCredentialFully,
  getClusterRoles,
  getNamespacesForRoles,
  getNamespaceRoles,
  bindUserToRole,
  unbindUserFromRole,
  createDefaultClusterRoles,
  createDefaultNamespaceRoles,
  getUserRoleBindings,
  type Cluster,
  type CredentialUser,
  type Role,
  type UserRoleBinding
} from '@/api/kubernetes'

interface Props {
  cluster: Cluster | null
  modelValue: boolean
  credentialUsers?: CredentialUser[]
}

const props = defineProps<Props>()
const emit = defineEmits(['update:modelValue', 'refresh'])

const loading = ref(false)
const showKubeConfigDialog = ref(false)
const showAuthorizeDialog = ref(false)
const currentUser = ref<any>(null)
const authorizeUser = ref<any>(null)
const currentKubeConfig = ref('')
const configLineCount = ref(1)
const authorizeLoading = ref(false)
const loadingRoles = ref(false)
const loadingNamespaces = ref(false)

// 授权相关数据
const clusterRoles = ref<Role[]>([])
const namespaces = ref<{ name: string; podCount?: number }[]>([])
const namespaceRoles = ref<Role[]>([])
const existingBindings = ref<UserRoleBinding[]>([])
const userRoleCounts = ref<Record<number, number>>({})

const authorizeForm = ref({
  permissionLevel: 'cluster',
  clusterRoleNames: [] as string[],
  namespace: '',
  namespaceRoleNames: [] as string[]
})

// 计算属性：已有的集群角色
const existingClusterRoles = computed(() => {
  if (!existingBindings.value || existingBindings.value.length === 0) return []
  return existingBindings.value
    .filter(b => b.roleType === 'ClusterRole')
    .map(b => ({
      id: b.id,
      roleName: b.roleName,
      roleNamespace: b.roleNamespace,
      userId: b.userId
    }))
})

// 计算属性：已有的命名空间权限（直接从 existingBindings 计算）
const existingNamespacePermissions = computed(() => {
  if (!existingBindings.value || existingBindings.value.length === 0) return []
  const nsBindings = existingBindings.value.filter(b => b.roleType === 'Role')
  const grouped: Record<string, typeof nsBindings> = {}

  nsBindings.forEach(binding => {
    if (!grouped[binding.roleNamespace]) {
      grouped[binding.roleNamespace] = []
    }
    grouped[binding.roleNamespace].push(binding)
  })

  return Object.entries(grouped).map(([namespace, bindings]) => ({
    namespace,
    roles: bindings.map(b => ({
      id: b.id,
      roleName: b.roleName,
      roleNamespace: b.roleNamespace,
      userId: b.userId
    }))
  }))
})

// 去重的用户列表
const uniqueCredentialUsers = computed(() => {
  if (!props.credentialUsers || props.credentialUsers.length === 0) return []
  const userMap = new Map<number, CredentialUser>()
  props.credentialUsers.forEach(user => {
    const existing = userMap.get(user.userId)
    if (!existing || new Date(user.createdAt) > new Date(existing.createdAt)) {
      userMap.set(user.userId, user)
    }
  })
  return Array.from(userMap.values()).sort((a, b) =>
    new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  )
})

// 监听 kubeconfig 内容变化
watch(currentKubeConfig, () => {
  configLineCount.value = currentKubeConfig.value.split('\n').length || 1
})

// 加载每个用户的角色数量
const loadUserRoleCounts = async () => {
  if (!props.cluster) return
  const users = uniqueCredentialUsers.value
  if (users.length === 0) return

  const counts: Record<number, number> = {}
  await Promise.all(
    users.map(async (user) => {
      try {
        const bindings = await getUserRoleBindings(props.cluster!.id, user.userId)
        counts[user.userId] = bindings?.length || 0
      } catch {
        counts[user.userId] = 0
      }
    })
  )
  userRoleCounts.value = counts
}

// 监听用户列表变化，加载角色数量
watch(uniqueCredentialUsers, (newUsers) => {
  if (newUsers.length > 0) {
    loadUserRoleCounts()
  }
}, { immediate: true })

// 方法
const handleRefresh = () => {
  emit('refresh')
  loadUserRoleCounts()
  Message.success('刷新成功')
}

const handleViewCredential = async (user: any) => {
  try {
    if (!props.cluster) return
    currentUser.value = user
    loading.value = true
    const result = await getServiceAccountKubeConfig(props.cluster.id, user.serviceAccount)
    currentKubeConfig.value = result.kubeconfig
    showKubeConfigDialog.value = true
  } catch (error: any) {
    Message.error(error.response?.data?.message || '获取 kubeconfig 失败')
  } finally {
    loading.value = false
  }
}

const handleCopyKubeConfig = async () => {
  try {
    await navigator.clipboard.writeText(currentKubeConfig.value)
    Message.success('复制成功')
  } catch {
    Message.error('复制失败')
  }
}

const handleDownloadKubeConfig = () => {
  const blob = new Blob([currentKubeConfig.value], { type: 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `kubeconfig-${currentUser?.username || 'user'}-${props.cluster?.name || 'cluster'}.yaml`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  Message.success('下载成功')
}

const handleRevoke = async (user: any) => {
  try {
    await confirmModal(
      `确定要吊销用户 "${user.realName || user.username}" 的凭据吗？\n\n吊销将同时删除：\n• K8s 中的 ServiceAccount\n• 所有相关的 RoleBinding\n• 数据库中的凭据记录\n\n吊销后用户将无法访问该集群！`,
      '确认吊销',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        dangerouslyUseHTMLString: true
      }
    )

    if (!props.cluster) return

    loading.value = true
    await revokeCredentialFully(props.cluster.id, user.serviceAccount, user.username)
    Message.success('吊销成功')
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '吊销失败')
    }
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

// 监听计算属性变化，用于调试
watch(existingNamespacePermissions, (newVal) => {
}, { immediate: true })

// 授权相关方法
const handleAuthorize = async (user: any) => {
  if (!props.cluster) return
  authorizeUser.value = user
  authorizeForm.value = {
    permissionLevel: 'cluster',
    clusterRoleNames: [],
    namespace: '',
    namespaceRoleNames: []
  }
  clusterRoles.value = []
  namespaces.value = []
  namespaceRoles.value = []
  existingBindings.value = []

  // 先获取权限，再显示对话框
  try {
    authorizeLoading.value = true
    const bindings = await getUserRoleBindings(props.cluster.id, user.userId)
    existingBindings.value = bindings
    showAuthorizeDialog.value = true
  } catch (error) {
    showAuthorizeDialog.value = true  // 即使失败也显示对话框
  } finally {
    authorizeLoading.value = false
  }
}

const handleAuthorizeDialogClose = () => {
  authorizeUser.value = null
  authorizeForm.value = {
    permissionLevel: 'cluster',
    clusterRoleNames: [],
    namespace: '',
    namespaceRoleNames: []
  }
  clusterRoles.value = []
  namespaces.value = []
  namespaceRoles.value = []
}

const handlePermissionLevelChange = () => {
  authorizeForm.value.clusterRoleNames = []
  authorizeForm.value.namespace = ''
  authorizeForm.value.namespaceRoleNames = []
}

const handleNamespaceChange = () => {
  authorizeForm.value.namespaceRoleNames = []
}

const loadClusterRoles = async () => {
  if (!props.cluster) return
  loadingRoles.value = true
  try {
    let roles = await getClusterRoles(props.cluster.id)

    // 定义应该有的14个默认集群角色
    const expectedClusterRoles = [
      'cluster-owner',
      'cluster-viewer',
      'manage-appmarket',
      'manage-cluster-rbac',
      'manage-cluster-storage',
      'manage-crd',
      'manage-namespaces',
      'manage-nodes',
      'view-cluster-rbac',
      'view-cluster-storage',
      'view-crd',
      'view-events',
      'view-namespaces',
      'view-nodes'
    ]

    // 如果角色数量不等于14，说明角色缺失，需要创建
    if (!roles || roles.length !== expectedClusterRoles.length) {
      try {
        await createDefaultClusterRoles(props.cluster.id)
        // 重新加载角色列表
        roles = await getClusterRoles(props.cluster.id)
      } catch (createError) {
      }
    }

    clusterRoles.value = roles || []
  } catch (error) {
    Message.error('加载集群角色失败')
  } finally {
    loadingRoles.value = false
  }
}

const loadNamespaces = async () => {
  if (!props.cluster) return
  loadingNamespaces.value = true
  try {
    const nsList = await getNamespacesForRoles(props.cluster.id)
    namespaces.value = nsList
  } catch (error) {
    Message.error('加载命名空间失败')
  } finally {
    loadingNamespaces.value = false
  }
}

const loadNamespaceRoles = async () => {
  if (!props.cluster || !authorizeForm.value.namespace) return
  loadingRoles.value = true
  try {
    let roles = await getNamespaceRoles(props.cluster.id, authorizeForm.value.namespace)

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
        await createDefaultNamespaceRoles(props.cluster.id, authorizeForm.value.namespace)
        // 重新加载角色列表
        roles = await getNamespaceRoles(props.cluster.id, authorizeForm.value.namespace)
      } catch (createError) {
      }
    }

    namespaceRoles.value = roles || []
  } catch (error) {
    Message.error('加载命名空间角色失败')
  } finally {
    loadingRoles.value = false
  }
}

const handleConfirmAuthorize = async () => {
  if (!props.cluster || !authorizeUser.value) return

  // 验证表单
  if (authorizeForm.value.permissionLevel === 'cluster') {
    if (!authorizeForm.value.clusterRoleNames || authorizeForm.value.clusterRoleNames.length === 0) {
      Message.warning('请选择集群角色')
      return
    }
  } else {
    if (!authorizeForm.value.namespace) {
      Message.warning('请选择命名空间')
      return
    }
    if (!authorizeForm.value.namespaceRoleNames || authorizeForm.value.namespaceRoleNames.length === 0) {
      Message.warning('请选择角色')
      return
    }
  }

  authorizeLoading.value = true
  try {
    const roleType = authorizeForm.value.permissionLevel === 'cluster' ? 'ClusterRole' : 'Role'
    const roleNamespace = authorizeForm.value.permissionLevel === 'cluster' ? '' : authorizeForm.value.namespace
    const roleNames = authorizeForm.value.permissionLevel === 'cluster'
      ? authorizeForm.value.clusterRoleNames
      : authorizeForm.value.namespaceRoleNames

    // 批量绑定多个角色
    for (const roleName of roleNames) {
      await bindUserToRole({
        clusterId: props.cluster.id,
        userId: authorizeUser.value.userId,
        roleName,
        roleNamespace,
        roleType
      })
    }

    Message.success('授权成功')
    // 重新加载用户权限
    const bindings = await getUserRoleBindings(props.cluster.id, authorizeUser.value.userId)
    existingBindings.value = bindings
    // 清空表单
    authorizeForm.value = {
      permissionLevel: 'cluster',
      clusterRoleNames: [],
      namespace: '',
      namespaceRoleNames: []
    }
    emit('refresh')
    // 刷新角色计数
    loadUserRoleCounts()
  } catch (error: any) {
    Message.error(error.response?.data?.message || '授权失败')
  } finally {
    authorizeLoading.value = false
  }
}

// 删除已有权限
const handleRemoveExistingRole = async (role: any) => {
  try {
    await confirmModal(
      `确定要删除角色 "${role.roleName}" 吗？`,
      '确认删除',
      { type: 'warning' }
    )

    if (!props.cluster || !authorizeUser.value) return

    authorizeLoading.value = true
    await unbindUserFromRole({
      clusterId: props.cluster.id,
      userId: role.userId,
      roleName: role.roleName,
      roleNamespace: role.roleNamespace
    })

    Message.success('删除成功')
    // 重新加载用户权限
    const bindings = await getUserRoleBindings(props.cluster.id, authorizeUser.value.userId)
    existingBindings.value = bindings
    emit('refresh')
    // 刷新角色计数
    loadUserRoleCounts()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '删除失败')
    }
  } finally {
    authorizeLoading.value = false
  }
}
</script>

<style scoped lang="scss">
.cluster-auth-content {
  padding: 0;
}

/* 操作栏 */
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-info {
  display: flex;
  align-items: center;
  gap: 14px;
}

.info-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #165dff;
  font-size: 20px;
  flex-shrink: 0;
}

.info-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-title {
  font-size: 16px;
  font-weight: 600;
  color: #1d2129;
}

.info-desc {
  font-size: 12px;
  color: #86909c;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.user-table {
  width: 100%;

  :deep(.arco-table__body-wrapper) {
    border-radius: 0 0 12px 12px;
  }

  :deep(.arco-table__row) {
    transition: background-color 0.2s ease;

    &:hover {
      background-color: #f8fafc !important;
    }
  }

  :deep(.arco-table__cell) {
    padding: 12px 0;
  }
}

/* 表格头像 */
.table-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #165dff;
  font-size: 16px;
  margin: 0 auto;
}

/* 用户单元格 */
.user-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.user-username {
  font-size: 12px;
  color: #909399;
}

/* 单行文本 */
.mono-text {
  font-family: 'Courier New', monospace;
  font-size: 13px;
  color: #606266;
}

.time-text {
  font-size: 13px;
  color: #909399;
}

.namespace-tag {
  max-width: 160px;
}

.namespace-text {
  display: inline-block;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
  line-height: 1.5;
}

/* 状态指示点 */
.status-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-right: 4px;
  vertical-align: middle;
}

.status-dot-active {
  background-color: #00b42a;
  box-shadow: 0 0 4px rgba(0, 180, 42, 0.4);
  animation: pulse-green 2s infinite;
}

@keyframes pulse-green {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* 表格操作 */
.action-authorize {
  color: #00b42a;
  font-weight: 500;

  &:hover {
    color: #00b42a;
    background-color: #e8ffea;
  }
}

.action-view {
  color: #165dff;
  font-weight: 500;

  &:hover {
    color: #4080ff;
    background-color: #e8f3ff;
  }
}

.action-revoke {
  color: #f53f3f;

  &:hover {
    color: #f53f3f;
    background-color: #ffece8;
  }
}

/* KubeConfig 对话框 */
.kubeconfig-dialog {
  .kubeconfig-info {
    margin-bottom: 20px;
  }

  .kubeconfig-actions {
    display: flex;
    gap: 12px;
    margin-bottom: 16px;
  }

  .code-editor-wrapper {
    display: flex;
    border: 1px solid #dcdfe6;
    border-radius: 8px;
    overflow: hidden;
    background-color: #282c34;
  }

  .line-numbers {
    display: flex;
    flex-direction: column;
    padding: 12px 8px;
    background-color: #21252b;
    border-right: 1px solid #3e4451;
    user-select: none;
    min-width: 40px;
    text-align: right;
  }

  .line-number {
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 13px;
    line-height: 1.6;
    color: #5c6370;
    min-height: 20.8px;
  }

  .code-textarea {
    flex: 1;
    min-height: 350px;
    padding: 12px;
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 13px;
    line-height: 1.6;
    color: #abb2bf;
    background-color: #282c34;
    border: none;
    outline: none;
    resize: vertical;
  }

  .code-tip {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    padding: 10px 14px;
    background: #fef0f0;
    border-radius: 6px;
    color: #f56c6c;
    font-size: 13px;

    :deep(.arco-icon) {
      font-size: 16px;
    }
  }
}

/* 授权对话框 */
.authorize-dialog {
  .user-info-section {
    margin-bottom: 24px;
    padding: 16px;
    background: #f5f7fa;
    border-radius: 8px;
  }

  .user-info-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
    font-weight: 600;
    color: #303133;
    font-size: 14px;

    :deep(.arco-icon) {
      color: #165dff;
      font-size: 18px;
    }
  }

  .user-info-content {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-left: 26px;
  }

  .info-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .info-label {
    font-size: 13px;
    color: #909399;
    min-width: 60px;
  }

  .info-value {
    font-size: 13px;
    color: #303133;
    font-weight: 500;
  }

  // 已有权限区域
  .existing-permissions-section {
    margin-bottom: 24px;
    padding: 16px;
    background: #fff;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
  }

  // 添加新权限区域
  .add-permission-section {
    padding: 16px;
    background: #fff;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    font-weight: 600;
    color: #303133;
    font-size: 14px;

    :deep(.arco-icon) {
      color: #165dff;
      font-size: 18px;
    }
  }

  .permission-group {
    margin-bottom: 16px;

    &:last-child {
      margin-bottom: 0;
    }
  }

  .permission-group-title {
    font-size: 13px;
    font-weight: 500;
    color: #606266;
    margin-bottom: 8px;
  }

  .permission-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .namespace-permissions {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .namespace-permission-item {
    padding: 12px;
    background: #f5f7fa;
    border-radius: 6px;
  }

  .namespace-name {
    font-size: 13px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 8px;
  }

  .namespace-roles {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .role-option {
    display: flex;
    align-items: center;
    gap: 8px;

    .role-icon {
      color: #165dff;
    }
  }

  .namespace-option {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}
</style>
