<template>
  <div class="mw-permission-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-lock />
        </div>
        <div>
          <h2 class="page-title">中间件权限</h2>
          <p class="page-subtitle">配置角色对中间件的访问和操作权限</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button v-permission="'middleware-perms:create'" type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          添加权限
        </a-button>
      </div>
    </div>

    <div class="filter-bar">
      <a-select v-model="filterRoleId" placeholder="角色筛选" allow-clear style="width: 180px" @change="loadList">
        <a-option v-for="r in roleList" :key="r.ID" :label="r.name" :value="r.ID" />
      </a-select>
    </div>

    <a-table
      :data="tableData"
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
        <a-table-column title="角色" data-index="roleName" :width="120" />
        <a-table-column title="业务分组" data-index="assetGroupName" :width="120" />
        <a-table-column title="中间件类型" :width="110">
          <template #cell="{ record }">
            <a-tag v-if="record.middlewareType" size="small">{{ mwTypeLabel(record.middlewareType) }}</a-tag>
            <span v-else style="color:#909399">全部类型</span>
          </template>
        </a-table-column>
        <a-table-column title="中间件范围" :min-width="180">
          <template #cell="{ record }">
            <span v-if="!record.middlewareIds || record.middlewareIds.length === 0">全部实例</span>
            <span v-else>{{ record.middlewareIds.length }} 个指定实例</span>
          </template>
        </a-table-column>
        <a-table-column title="权限" :min-width="250">
          <template #cell="{ record }">
            <a-tag v-if="record.permissions & 1" size="small" style="margin: 2px;">查看</a-tag>
            <a-tag v-if="record.permissions & 2" size="small" color="orangered" style="margin: 2px;">编辑</a-tag>
            <a-tag v-if="record.permissions & 4" size="small" color="red" style="margin: 2px;">删除</a-tag>
            <a-tag v-if="record.permissions & 8" size="small" color="green" style="margin: 2px;">连接</a-tag>
            <a-tag v-if="record.permissions & 16" size="small" color="gray" style="margin: 2px;">执行</a-tag>
            <a-tag v-if="record.permissions & 32" size="small" color="arcoblue" style="margin: 2px;">查询数据</a-tag>
            <a-tag v-if="record.permissions & 64" size="small" color="orangered" style="margin: 2px;">修改数据</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="150" fixed="right">
          <template #cell="{ record }">
            <a-button v-permission="'middleware-perms:update'" type="text" size="small" @click="handleEdit(record)">编辑</a-button>
            <a-button v-permission="'middleware-perms:delete'" type="text" size="small" status="danger" @click="handleDelete(record)">删除</a-button>
          </template>
        </a-table-column>
      </template>
    </a-table>

    <!-- 添加/编辑权限弹窗 -->
    <a-modal v-model:visible="dialogVisible" :title="dialogTitle" :width="600" unmount-on-close>
      <a-form :model="formData" layout="horizontal" auto-label-width>
        <a-form-item label="角色" field="roleId" required>
          <a-select v-model="formData.roleId" placeholder="请选择角色" style="width: 100%" allow-search>
            <a-option v-for="r in roleList" :key="r.ID" :label="r.name" :value="r.ID" />
          </a-select>
        </a-form-item>
        <a-form-item label="业务分组" field="assetGroupIds" required>
          <a-tree-select v-if="!isEdit" v-model="formData.assetGroupIds" :data="groupTree" :field-names="{ key: 'id', title: 'name', children: 'children' }" placeholder="请选择分组（可多选）" style="width: 100%" tree-checkable allow-search @change="onGroupOrTypeChange" />
          <a-tree-select v-else v-model="formData.assetGroupId" :data="groupTree" :field-names="{ key: 'id', title: 'name', children: 'children' }" placeholder="请选择分组" style="width: 100%" allow-search @change="onGroupOrTypeChange" />
        </a-form-item>
        <a-form-item label="中间件类型" field="middlewareType">
          <a-select v-model="formData.middlewareType" placeholder="全部类型" allow-clear style="width: 100%" @change="onGroupOrTypeChange">
            <a-option v-for="t in mwTypeOptions" :key="t.value" :label="t.label" :value="t.value" />
          </a-select>
        </a-form-item>
        <a-form-item label="中间件实例" field="middlewareIds">
          <a-select v-model="formData.middlewareIds" multiple placeholder="全部实例（不选则为全部）" style="width: 100%" allow-search :loading="mwListLoading">
            <a-option v-for="mw in filteredMiddlewares" :key="mw.id" :label="`${mw.name} (${mw.host}:${mw.port})`" :value="mw.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="权限" field="permissions">
          <a-checkbox-group v-model="selectedPerms">
            <a-checkbox v-for="p in permOptions" :key="p.value" :value="p.value">{{ p.label }}</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconLock, IconPlus } from '@arco-design/web-vue/es/icon'
import { getMiddlewarePermissions, createMiddlewarePermission, updateMiddlewarePermission, deleteMiddlewarePermission, getMiddlewareList } from '@/api/middleware'
import { getGroupTree } from '@/api/assetGroup'
import request from '@/utils/request'

const mwTypeOptions = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'Redis', value: 'redis' },
  { label: 'ClickHouse', value: 'clickhouse' },
  { label: 'MongoDB', value: 'mongodb' },
  { label: 'Kafka', value: 'kafka' },
  { label: 'Milvus', value: 'milvus' },
]

const mwTypeLabel = (type: string) => {
  return mwTypeOptions.find(t => t.value === type)?.label || type
}

const roleList = ref<any[]>([])
const groupTree = ref<any[]>([])
const tableData = ref<any[]>([])
const loading = ref(false)
const filterRoleId = ref<number | undefined>(undefined)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })

const dialogVisible = ref(false)
const dialogTitle = ref('添加权限')
const isEdit = ref(false)
const submitLoading = ref(false)
const formData = reactive({
  id: 0,
  roleId: undefined as number | undefined,
  assetGroupId: undefined as number | undefined,
  assetGroupIds: [] as number[],
  middlewareIds: [] as number[],
  middlewareType: '' as string,
  permissions: 1,
})

const allMiddlewares = ref<any[]>([])
const mwListLoading = ref(false)

const filteredMiddlewares = computed(() => {
  let list = allMiddlewares.value
  // 按类型筛选
  if (formData.middlewareType) {
    list = list.filter((m: any) => m.type === formData.middlewareType)
  }
  // 按分组筛选
  const gids = isEdit.value
    ? (formData.assetGroupId ? [formData.assetGroupId] : [])
    : formData.assetGroupIds
  if (gids.length > 0) {
    list = list.filter((m: any) => gids.includes(m.groupId))
  }
  return list
})

const permOptions = [
  { label: '查看', value: 1 },
  { label: '编辑', value: 2 },
  { label: '删除', value: 4 },
  { label: '测试连接', value: 8 },
  { label: '执行操作', value: 16 },
  { label: '查询数据', value: 32 },
  { label: '修改数据', value: 64 },
]

const selectedPerms = ref<number[]>([1])

const loadRoles = async () => {
  try {
    const res = await request.get('/api/v1/roles', { params: { page: 1, pageSize: 100 } })
    roleList.value = res?.list || []
  } catch {}
}

const loadGroupTreeData = async () => {
  try {
    const data = await getGroupTree()
    groupTree.value = data || []
  } catch {}
}

const loadAllMiddlewares = async () => {
  mwListLoading.value = true
  try {
    const res = await getMiddlewareList({ page: 1, pageSize: 9999 })
    allMiddlewares.value = res?.list || []
  } catch {} finally {
    mwListLoading.value = false
  }
}

const onGroupOrTypeChange = () => {
  // 清除已选的不在筛选范围内的实例
  const validIds = new Set(filteredMiddlewares.value.map((m: any) => m.id))
  formData.middlewareIds = formData.middlewareIds.filter(id => validIds.has(id))
}

const loadList = async () => {
  loading.value = true
  try {
    const params: any = { page: pagination.page, pageSize: pagination.pageSize }
    if (filterRoleId.value) params.roleId = filterRoleId.value
    const res = await getMiddlewarePermissions(params)
    tableData.value = res?.list || []
    pagination.total = res?.total || 0
  } finally {
    loading.value = false
  }
}

const permsToArray = (permissions: number) => {
  const arr: number[] = []
  permOptions.forEach(p => { if (permissions & p.value) arr.push(p.value) })
  return arr
}

const arrayToPerms = (arr: number[]) => {
  return arr.reduce((acc, v) => acc | v, 0)
}

const handleAdd = () => {
  dialogTitle.value = '添加权限'
  isEdit.value = false
  Object.assign(formData, { id: 0, roleId: undefined, assetGroupId: undefined, assetGroupIds: [], middlewareIds: [], middlewareType: '', permissions: 1 })
  selectedPerms.value = [1]
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  dialogTitle.value = '编辑权限'
  isEdit.value = true
  Object.assign(formData, {
    id: row.id, roleId: row.roleId, assetGroupId: row.assetGroupId,
    assetGroupIds: [row.assetGroupId],
    middlewareIds: row.middlewareIds || [],
    middlewareType: row.middlewareType || '',
    permissions: row.permissions,
  })
  selectedPerms.value = permsToArray(row.permissions)
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formData.roleId) return Message.warning('请选择角色')
  const gids = isEdit.value
    ? (formData.assetGroupId ? [formData.assetGroupId] : [])
    : formData.assetGroupIds
  if (gids.length === 0) return Message.warning('请选择至少一个业务分组')

  submitLoading.value = true
  try {
    const perms = arrayToPerms(selectedPerms.value)
    if (isEdit.value && formData.id) {
      await updateMiddlewarePermission(formData.id, {
        roleId: formData.roleId,
        assetGroupId: formData.assetGroupId,
        middlewareIds: formData.middlewareIds,
        middlewareType: formData.middlewareType,
        permissions: perms,
      })
      Message.success('更新成功')
    } else {
      await createMiddlewarePermission({
        roleId: formData.roleId,
        assetGroupIds: gids,
        middlewareIds: formData.middlewareIds,
        middlewareType: formData.middlewareType,
        permissions: perms,
      })
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadList()
  } catch (e: any) {
    Message.error(e.message || '操作失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = (row: any) => {
  Modal.warning({
    title: '提示',
    content: '确定删除该权限配置？',
    hideCancel: false,
    onOk: async () => {
      await deleteMiddlewarePermission(row.id)
      Message.success('删除成功')
      loadList()
    }
  })
}

onMounted(() => {
  loadRoles()
  loadGroupTreeData()
  loadAllMiddlewares()
  loadList()
})
</script>

<style scoped>
.mw-permission-page { padding: 0; }
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--ops-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
}
.page-title { margin: 0; font-size: 18px; font-weight: 600; }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: #909399; }
.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}
</style>
