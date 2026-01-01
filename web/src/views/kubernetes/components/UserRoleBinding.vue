<template>
  <div class="user-role-binding">
    <!-- 操作栏 -->
    <div class="action-bar">
      <div class="search-input">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索用户名或真实姓名..."
          clearable
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      <div class="action-buttons">
        <el-button class="bind-btn" @click="showBindDialog = true">
          <el-icon><Plus /></el-icon>
          绑定角色
        </el-button>
        <el-button @click="loadData" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 用户角色绑定列表 -->
    <div class="binding-list" v-loading="loading">
      <div
        v-for="item in filteredBindings"
        :key="item.id"
        class="binding-card"
      >
        <div class="binding-header">
          <div class="user-info">
            <div class="user-avatar">
              <el-icon><User /></el-icon>
            </div>
            <div class="user-details">
              <div class="user-name">{{ item.realName || item.username }}</div>
              <div class="user-username">@{{ item.username }}</div>
            </div>
          </div>
          <el-tag :type="item.roleType === 'ClusterRole' ? 'danger' : 'primary'" size="large">
            {{ item.roleType === 'ClusterRole' ? '集群权限' : '命名空间权限' }}
          </el-tag>
        </div>

        <div class="binding-content">
          <div class="binding-item">
            <span class="label">角色:</span>
            <el-tag type="warning" effect="plain">{{ item.roleName }}</el-tag>
          </div>
          <div class="binding-item" v-if="item.roleType === 'Role'">
            <span class="label">命名空间:</span>
            <el-tag type="success" effect="plain">{{ item.roleNamespace }}</el-tag>
          </div>
          <div class="binding-item">
            <span class="label">绑定时间:</span>
            <span class="value">{{ formatDate(item.createdAt) }}</span>
          </div>
        </div>

        <div class="binding-footer">
          <el-button
            link
            type="danger"
            size="small"
            @click="handleUnbind(item)"
          >
            <el-icon><Delete /></el-icon>
            解绑
          </el-button>
        </div>
      </div>

      <el-empty
        v-if="!loading && !filteredBindings.length"
        description="暂无角色绑定"
        :image-size="100"
      />
    </div>

    <!-- 绑定角色对话框 -->
    <el-dialog
      v-model="showBindDialog"
      title="绑定K8s角色"
      width="700px"
      @close="handleDialogClose"
    >
      <el-form :model="bindForm" :rules="bindRules" ref="bindFormRef" label-width="100px">
        <!-- 选择用户 -->
        <el-form-item label="选择用户" prop="userId">
          <el-select
            v-model="bindForm.userId"
            placeholder="请选择用户"
            filterable
            remote
            :remote-method="searchUsers"
            :loading="searchUsersLoading"
            style="width: 100%"
          >
            <el-option
              v-for="user in availableUsers"
              :key="user.id"
              :label="`${user.realName} (@${user.username})`"
              :value="user.id"
            >
              <div class="user-option">
                <span class="user-option-name">{{ user.realName }}</span>
                <span class="user-option-username">@{{ user.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <!-- 权限级别 -->
        <el-form-item label="权限级别" prop="permissionLevel">
          <el-radio-group v-model="bindForm.permissionLevel" @change="handlePermissionLevelChange">
            <el-radio-button label="cluster">集群级别</el-radio-button>
            <el-radio-button label="namespace">命名空间级别</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <!-- 集群角色选择 -->
        <template v-if="bindForm.permissionLevel === 'cluster'">
          <el-form-item label="集群角色" prop="clusterRoleName">
            <el-select
              v-model="bindForm.clusterRoleName"
              placeholder="请选择集群角色"
              filterable
              @focus="loadClusterRoles"
              :loading="loadingRoles"
              style="width: 100%"
            >
              <el-option
                v-for="role in clusterRoles"
                :key="role.name"
                :label="role.name"
                :value="role.name"
              >
                <div class="role-option">
                  <el-icon class="role-icon"><Key /></el-icon>
                  <span>{{ role.name }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
        </template>

        <!-- 命名空间角色选择 -->
        <template v-if="bindForm.permissionLevel === 'namespace'">
          <el-form-item label="命名空间" prop="namespace">
            <el-select
              v-model="bindForm.namespace"
              placeholder="请选择命名空间"
              filterable
              @focus="loadNamespaces"
              @change="handleNamespaceChange"
              :loading="loadingNamespaces"
              style="width: 100%"
            >
              <el-option
                v-for="ns in namespaces"
                :key="ns.name"
                :label="ns.name"
                :value="ns.name"
              >
                <div class="namespace-option">
                  <el-icon><FolderOpened /></el-icon>
                  <span>{{ ns.name }}</span>
                  <el-tag size="small" type="info" v-if="ns.podCount !== undefined">{{ ns.podCount }} pods</el-tag>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <el-form-item label="角色" prop="namespaceRoleName">
            <el-select
              v-model="bindForm.namespaceRoleName"
              placeholder="请先选择命名空间"
              filterable
              :disabled="!bindForm.namespace"
              @focus="loadNamespaceRoles"
              :loading="loadingRoles"
              style="width: 100%"
            >
              <el-option
                v-for="role in namespaceRoles"
                :key="role.name"
                :label="role.name"
                :value="role.name"
              >
                <div class="role-option">
                  <el-icon class="role-icon"><Key /></el-icon>
                  <span>{{ role.name }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
        </template>

        <!-- 角色权限预览 -->
        <el-form-item v-if="selectedRolePreview" label="权限预览">
          <div class="permission-preview">
            <div
              v-for="(rule, index) in selectedRolePreview.rules"
              :key="index"
              class="preview-rule"
            >
              <div class="rule-header">
                <el-icon><FolderOpened /></el-icon>
                <span>API 组: {{ formatApiGroups(rule.apiGroups) }}</span>
              </div>
              <div class="rule-content">
                <div class="rule-line">
                  <span class="rule-label">资源:</span>
                  <el-tag
                    v-for="res in rule.resources"
                    :key="res"
                    size="small"
                    type="primary"
                    effect="plain"
                  >
                    {{ res }}
                  </el-tag>
                </div>
                <div class="rule-line">
                  <span class="rule-label">操作:</span>
                  <el-tag
                    v-for="verb in rule.verbs"
                    :key="verb"
                    size="small"
                    :type="getVerbType(verb)"
                  >
                    {{ formatVerb(verb) }}
                  </el-tag>
                </div>
              </div>
            </div>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showBindDialog = false">取消</el-button>
        <el-button type="primary" @click="handleBind" :loading="bindLoading">
          确认绑定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules } from 'element-plus'
import {
  Search,
  Refresh,
  Plus,
  User,
  Delete,
  Key,
  FolderOpened
} from '@element-plus/icons-vue'
import {
  getUserRoleBindings,
  bindUserToRole,
  unbindUserFromRole,
  getClusterRoles,
  getNamespacesForRoles,
  getNamespaceRoles,
  getRoleDetail,
  getAvailableUsers,
  type Cluster,
  type UserRoleBinding,
  type Role
} from '@/api/kubernetes'

interface Props {
  cluster: Cluster | null
}

const props = defineProps<Props>()

const loading = ref(false)
const searchKeyword = ref('')
const bindings = ref<UserRoleBinding[]>([])
const showBindDialog = ref(false)
const bindLoading = ref(false)
const bindFormRef = ref<FormInstance>()

// 搜索用户
const availableUsers = ref<any[]>([])
const searchUsersLoading = ref(false)

// 角色数据
const clusterRoles = ref<Role[]>([])
const namespaces = ref<{ name: string; podCount?: number }[]>([])
const namespaceRoles = ref<Role[]>([])
const loadingRoles = ref(false)
const loadingNamespaces = ref(false)
const selectedRolePreview = ref<Role | null>(null)

// 绑定表单
const bindForm = reactive({
  userId: undefined as number | undefined,
  permissionLevel: 'cluster',
  clusterRoleName: '',
  namespace: '',
  namespaceRoleName: ''
})

const bindRules: FormRules = {
  userId: [{ required: true, message: '请选择用户', trigger: 'change' }],
  clusterRoleName: [
    {
      validator: (rule, value, callback) => {
        if (bindForm.permissionLevel === 'cluster' && !value) {
          callback(new Error('请选择集群角色'))
        } else {
          callback()
        }
      },
      trigger: 'change'
    }
  ],
  namespace: [
    {
      validator: (rule, value, callback) => {
        if (bindForm.permissionLevel === 'namespace' && !value) {
          callback(new Error('请选择命名空间'))
        } else {
          callback()
        }
      },
      trigger: 'change'
    }
  ],
  namespaceRoleName: [
    {
      validator: (rule, value, callback) => {
        if (bindForm.permissionLevel === 'namespace' && !value) {
          callback(new Error('请选择角色'))
        } else {
          callback()
        }
      },
      trigger: 'change'
    }
  ]
}

// 计算属性
const filteredBindings = computed(() => {
  if (!searchKeyword.value) return bindings.value
  const keyword = searchKeyword.value.toLowerCase()
  return bindings.value.filter(b =>
    b.username.toLowerCase().includes(keyword) ||
    (b.realName && b.realName.toLowerCase().includes(keyword))
  )
})

// 加载绑定数据
const loadData = async () => {
  if (!props.cluster) return
  loading.value = true
  try {
    const data = await getUserRoleBindings(props.cluster.id)
    bindings.value = data
  } catch (error) {
    ElMessage.error('加载角色绑定失败')
  } finally {
    loading.value = false
  }
}

// 搜索用户
const searchUsers = async (query: string) => {
  if (!query) return
  searchUsersLoading.value = true
  try {
    const result = await getAvailableUsers(query, 1, 20)
    availableUsers.value = result.list
  } catch (error) {
    ElMessage.error('搜索用户失败')
  } finally {
    searchUsersLoading.value = false
  }
}

const handleSearch = () => {
  // filteredBindings 会自动更新
}

// 加载集群角色
const loadClusterRoles = async () => {
  if (!props.cluster) return
  loadingRoles.value = true
  try {
    const roles = await getClusterRoles(props.cluster.id)
    clusterRoles.value = roles
  } catch (error) {
    ElMessage.error('加载集群角色失败')
  } finally {
    loadingRoles.value = false
  }
}

// 加载命名空间
const loadNamespaces = async () => {
  if (!props.cluster) return
  loadingNamespaces.value = true
  try {
    const nsList = await getNamespacesForRoles(props.cluster.id)
    namespaces.value = nsList
  } catch (error) {
    ElMessage.error('加载命名空间失败')
  } finally {
    loadingNamespaces.value = false
  }
}

// 加载命名空间角色
const loadNamespaceRoles = async () => {
  if (!props.cluster || !bindForm.namespace) return
  loadingRoles.value = true
  try {
    const roles = await getNamespaceRoles(props.cluster.id, bindForm.namespace)
    namespaceRoles.value = roles
  } catch (error) {
    ElMessage.error('加载命名空间角色失败')
  } finally {
    loadingRoles.value = false
  }
}

// 权限级别变化
const handlePermissionLevelChange = () => {
  bindForm.clusterRoleName = ''
  bindForm.namespace = ''
  bindForm.namespaceRoleName = ''
  selectedRolePreview.value = null
}

// 命名空间变化
const handleNamespaceChange = () => {
  bindForm.namespaceRoleName = ''
  selectedRolePreview.value = null
}

// 加载角色详情预览
const loadRolePreview = async (roleName: string, namespace: string) => {
  if (!props.cluster) return
  try {
    const detail = await getRoleDetail(props.cluster.id, namespace, roleName)
    selectedRolePreview.value = detail
  } catch (error) {
    selectedRolePreview.value = null
  }
}

// 监听角色选择变化
watch(() => bindForm.clusterRoleName, (newVal) => {
  if (newVal && bindForm.permissionLevel === 'cluster') {
    loadRolePreview(newVal, '')
  }
})

watch(() => bindForm.namespaceRoleName, (newVal) => {
  if (newVal && bindForm.permissionLevel === 'namespace' && bindForm.namespace) {
    loadRolePreview(newVal, bindForm.namespace)
  }
})

// 绑定
const handleBind = async () => {
  if (!bindFormRef.value || !props.cluster) return

  await bindFormRef.value.validate(async (valid) => {
    if (valid) {
      bindLoading.value = true
      try {
        let roleName = ''
        let roleNamespace = ''

        if (bindForm.permissionLevel === 'cluster') {
          roleName = bindForm.clusterRoleName
          roleNamespace = ''
        } else {
          roleName = bindForm.namespaceRoleName
          roleNamespace = bindForm.namespace
        }

        await bindUserToRole({
          clusterId: props.cluster.id,
          userId: bindForm.userId!,
          roleName,
          roleNamespace,
          roleType: bindForm.permissionLevel === 'cluster' ? 'ClusterRole' : 'Role'
        })

        ElMessage.success('绑定成功')
        showBindDialog.value = false
        loadData()
      } catch (error: any) {
        ElMessage.error(error.response?.data?.message || '绑定失败')
      } finally {
        bindLoading.value = false
      }
    }
  })
}

// 解绑
const handleUnbind = async (item: UserRoleBinding) => {
  try {
    await ElMessageBox.confirm(
      `确定要解除用户 "${item.realName || item.username}" 的角色 "${item.roleName}" 绑定吗？`,
      '确认解绑',
      { type: 'warning' }
    )

    if (!props.cluster) return

    await unbindUserFromRole({
      clusterId: props.cluster.id,
      userId: item.userId,
      roleName: item.roleName,
      roleNamespace: item.roleNamespace
    })

    ElMessage.success('解绑成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '解绑失败')
    }
  }
}

// 关闭对话框
const handleDialogClose = () => {
  bindFormRef.value?.resetFields()
  bindForm.userId = undefined
  bindForm.permissionLevel = 'cluster'
  bindForm.clusterRoleName = ''
  bindForm.namespace = ''
  bindForm.namespaceRoleName = ''
  selectedRolePreview.value = null
  availableUsers.value = []
}

// 格式化日期
const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

const formatApiGroups = (groups: string[]) => {
  if (!groups || groups.length === 0) return 'core'
  return groups.map(g => g || 'core').join(', ')
}

const formatVerb = (verb: string) => {
  const verbMap: Record<string, string> = {
    '*': '所有',
    'get': '查看',
    'list': '列表',
    'watch': '监听',
    'create': '创建',
    'update': '更新',
    'patch': '修补',
    'delete': '删除',
    'deletecollection': '批量删除'
  }
  return verbMap[verb] || verb
}

const getVerbType = (verb: string) => {
  const typeMap: Record<string, string> = {
    '*': 'danger',
    'get': '',
    'list': '',
    'watch': '',
    'create': 'success',
    'update': 'warning',
    'patch': 'warning',
    'delete': 'danger',
    'deletecollection': 'danger'
  }
  return typeMap[verb] || 'info'
}

// 初始化
watch(() => props.cluster, (newCluster) => {
  if (newCluster) {
    loadData()
  }
}, { immediate: true })
</script>

<style scoped lang="scss">
.user-role-binding {
  padding: 0;
}

/* 操作栏 */
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  gap: 16px;
}

.search-input {
  flex: 1;
  max-width: 400px;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

.bind-btn {
  background: #000;
  color: #d4af37;
  border-color: #d4af37;

  &:hover {
    background: #1a1a1a;
    border-color: #d4af37;
  }
}

/* 绑定列表 */
.binding-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.binding-card {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e4e7ed;
  transition: all 0.3s ease;

  &:hover {
    border-color: #d4af37;
    box-shadow: 0 4px 20px rgba(212, 175, 55, 0.15);
  }
}

.binding-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border: 2px solid #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 24px;
}

.user-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.user-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.user-username {
  font-size: 13px;
  color: #909399;
}

.binding-content {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  margin-bottom: 16px;
}

.binding-item {
  display: flex;
  align-items: center;
  gap: 8px;

  .label {
    font-size: 13px;
    color: #606266;
  }

  .value {
    font-size: 13px;
    color: #303133;
  }
}

.binding-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  border-top: 1px solid #e4e7ed;
}

/* 用户选项 */
.user-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;

  .user-option-name {
    font-weight: 500;
    color: #303133;
  }

  .user-option-username {
    font-size: 12px;
    color: #909399;
  }
}

/* 角色选项 */
.role-option {
  display: flex;
  align-items: center;
  gap: 8px;

  .role-icon {
    color: #d4af37;
  }
}

/* 命名空间选项 */
.namespace-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

/* 权限预览 */
.permission-preview {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 16px;
  max-height: 300px;
  overflow-y: auto;
}

.preview-rule {
  margin-bottom: 16px;

  &:last-child {
    margin-bottom: 0;
  }
}

.rule-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-weight: 600;
  color: #303133;
}

.rule-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-left: 26px;
}

.rule-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.rule-label {
  font-size: 13px;
  font-weight: 500;
  color: #606266;
  min-width: 50px;
}
</style>
