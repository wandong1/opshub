<template>
  <div class="mw-permission-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Lock /></el-icon>
        </div>
        <div>
          <h2 class="page-title">中间件权限</h2>
          <p class="page-subtitle">配置角色对中间件的访问和操作权限</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button v-permission="'middleware-perms:create'" type="primary" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          添加权限
        </el-button>
      </div>
    </div>

    <div class="content-card">
      <div class="filter-bar">
        <el-select v-model="filterRoleId" placeholder="角色筛选" clearable style="width: 180px" @change="loadList">
          <el-option v-for="r in roleList" :key="r.ID" :label="r.name" :value="r.ID" />
        </el-select>
      </div>

      <el-table :data="tableData" v-loading="loading" style="width: 100%">
        <el-table-column prop="roleName" label="角色" width="120" />
        <el-table-column prop="assetGroupName" label="业务分组" width="120" />
        <el-table-column label="中间件类型" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.middlewareType" size="small">{{ mwTypeLabel(row.middlewareType) }}</el-tag>
            <span v-else style="color:#909399">全部类型</span>
          </template>
        </el-table-column>
        <el-table-column label="中间件范围" min-width="180">
          <template #default="{ row }">
            <span v-if="!row.middlewareIds || row.middlewareIds.length === 0">全部实例</span>
            <span v-else>{{ row.middlewareIds.length }} 个指定实例</span>
          </template>
        </el-table-column>
        <el-table-column label="权限" min-width="250">
          <template #default="{ row }">
            <el-tag v-if="row.permissions & 1" size="small" style="margin: 2px;">查看</el-tag>
            <el-tag v-if="row.permissions & 2" size="small" type="warning" style="margin: 2px;">编辑</el-tag>
            <el-tag v-if="row.permissions & 4" size="small" type="danger" style="margin: 2px;">删除</el-tag>
            <el-tag v-if="row.permissions & 8" size="small" type="success" style="margin: 2px;">连接</el-tag>
            <el-tag v-if="row.permissions & 16" size="small" type="info" style="margin: 2px;">执行</el-tag>
            <el-tag v-if="row.permissions & 32" size="small" type="primary" style="margin: 2px;">查询数据</el-tag>
            <el-tag v-if="row.permissions & 64" size="small" type="warning" style="margin: 2px;">修改数据</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button v-permission="'middleware-perms:update'" link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button v-permission="'middleware-perms:delete'" link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-container">
        <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.pageSize" :total="pagination.total" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next" @size-change="loadList" @current-change="loadList" />
      </div>
    </div>

    <!-- 添加/编辑权限弹窗 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px" destroy-on-close>
      <el-form :model="formData" label-width="100px">
        <el-form-item label="角色" required>
          <el-select v-model="formData.roleId" placeholder="请选择角色" style="width: 100%">
            <el-option v-for="r in roleList" :key="r.ID" :label="r.name" :value="r.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="业务分组" required>
          <el-tree-select v-if="!isEdit" v-model="formData.assetGroupIds" :data="groupTree" :props="{ label: 'name', children: 'children', value: 'id' }" placeholder="请选择分组（可多选）" style="width: 100%" check-strictly multiple @change="onGroupOrTypeChange" />
          <el-tree-select v-else v-model="formData.assetGroupId" :data="groupTree" :props="{ label: 'name', children: 'children', value: 'id' }" placeholder="请选择分组" style="width: 100%" check-strictly @change="onGroupOrTypeChange" />
        </el-form-item>
        <el-form-item label="中间件类型">
          <el-select v-model="formData.middlewareType" placeholder="全部类型" clearable style="width: 100%" @change="onGroupOrTypeChange">
            <el-option v-for="t in mwTypeOptions" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="中间件实例">
          <el-select v-model="formData.middlewareIds" multiple placeholder="全部实例（不选则为全部）" style="width: 100%" filterable :loading="mwListLoading">
            <el-option v-for="mw in filteredMiddlewares" :key="mw.id" :label="`${mw.name} (${mw.host}:${mw.port})`" :value="mw.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="权限">
          <el-checkbox-group v-model="selectedPerms">
            <el-checkbox v-for="p in permOptions" :key="p.value" :value="p.value">{{ p.label }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Lock } from '@element-plus/icons-vue'
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
  if (!formData.roleId) return ElMessage.warning('请选择角色')
  const gids = isEdit.value
    ? (formData.assetGroupId ? [formData.assetGroupId] : [])
    : formData.assetGroupIds
  if (gids.length === 0) return ElMessage.warning('请选择至少一个业务分组')

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
      ElMessage.success('更新成功')
    } else {
      await createMiddlewarePermission({
        roleId: formData.roleId,
        assetGroupIds: gids,
        middlewareIds: formData.middlewareIds,
        middlewareType: formData.middlewareType,
        permissions: perms,
      })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadList()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定删除该权限配置？', '提示', { type: 'warning' }).then(async () => {
    await deleteMiddlewarePermission(row.id)
    ElMessage.success('删除成功')
    loadList()
  }).catch(() => {})
}

onMounted(() => {
  loadRoles()
  loadGroupTreeData()
  loadAllMiddlewares()
  loadList()
})
</script>

<style scoped>
.mw-permission-page { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 20px; }
.page-title { margin: 0; font-size: 18px; font-weight: 600; }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: #909399; }
.content-card { background: #fff; border-radius: 8px; border: 1px solid #ebeef5; padding: 16px; }
.filter-bar { display: flex; gap: 10px; margin-bottom: 16px; }
.pagination-container { margin-top: 16px; display: flex; justify-content: flex-end; }
</style>


