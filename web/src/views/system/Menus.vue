<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-menu />
        </div>
        <div>
          <div class="page-title">菜单管理</div>
          <div class="page-desc">管理系统菜单和插件菜单，支持目录、菜单、按钮三级层级结构</div>
        </div>
      </div>
      <div class="page-header-actions">
        <a-button v-permission="'menus:create'" type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          新增菜单
        </a-button>
        <a-button @click="toggleExpandAll">
          <template #icon><icon-swap /></template>
          {{ expandAll ? '折叠全部' : '展开全部' }}
        </a-button>
      </div>
    </div>

    <!-- 搜索区域 -->
    <a-card class="search-card" :bordered="false">
      <a-space :size="16" wrap>
        <a-space>
          <span class="search-label">菜单名称:</span>
          <a-input v-model="searchForm.name" placeholder="搜索菜单名称" allow-clear style="width: 200px" />
        </a-space>
        <a-space>
          <span class="search-label">状态:</span>
          <a-select v-model="searchForm.status" placeholder="全部" allow-clear style="width: 120px">
            <a-option :value="1">启用</a-option>
            <a-option :value="0">禁用</a-option>
          </a-select>
        </a-space>
        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
      </a-space>
    </a-card>

    <!-- 菜单树表格 -->
    <a-card class="table-card" :bordered="false">
      <a-table
        :data="filteredMenuList"
        :loading="loading"
        :row-key="getRowKey"
        v-model:expanded-keys="expandedKeys"
        :pagination="false"
      >
        <template #columns>
          <a-table-column title="菜单名称" :width="280">
            <template #cell="{ record }">
              <span style="display: inline-flex; align-items: center; gap: 6px;">
                <span v-if="record.icon" class="menu-icon-preview">{{ record.icon }}</span>
                {{ record.name }}
              </span>
            </template>
          </a-table-column>
          <a-table-column title="菜单编码" data-index="code" :min-width="150" />
          <a-table-column title="类型" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.type === 1" size="small" color="arcoblue">目录</a-tag>
              <a-tag v-else-if="record.type === 2" size="small" color="green">菜单</a-tag>
              <a-tag v-else size="small" color="orangered">按钮</a-tag>
              <a-tag v-if="record.isPlugin" size="small" color="gold" style="margin-left: 4px;">插件</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="路由路径" data-index="path" :min-width="180" />
          <a-table-column title="API绑定" :min-width="220">
            <template #cell="{ record }">
              <template v-if="record.type === 3 && record.apis && record.apis.length > 0">
                <div v-for="(api, idx) in record.apis" :key="idx" style="line-height: 1.8;">
                  <a-tag size="small" :color="getMethodColor(api.apiMethod)">{{ api.apiMethod }}</a-tag>
                  <span class="api-path-text">{{ api.apiPath }}</span>
                </div>
              </template>
              <template v-else-if="record.type === 3 && record.apiPath">
                <a-tag size="small" :color="getMethodColor(record.apiMethod)">{{ record.apiMethod }}</a-tag>
                <span class="api-path-text">{{ record.apiPath }}</span>
              </template>
              <span v-else class="text-muted">-</span>
            </template>
          </a-table-column>
          <a-table-column title="排序" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag size="small" color="gray">{{ record.sort || 0 }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.status === 1 ? 'green' : 'red'" size="small">
                {{ record.status === 1 ? '启用' : '禁用' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="180" align="center" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-tooltip v-if="record.isPlugin" content="调整排序">
                  <a-link status="warning" @click="handleEditPluginSort(record)">
                    <icon-swap />
                  </a-link>
                </a-tooltip>
                <a-tooltip v-if="!record.isPlugin && record.type === 1" content="新增下级菜单">
                  <a-link v-permission="'menus:create'" status="success" @click="handleAddChildMenu(record)">
                    <icon-plus />
                  </a-link>
                </a-tooltip>
                <a-tooltip v-if="!record.isPlugin && record.type === 2" content="新增下级按钮">
                  <a-link v-permission="'menus:create'" status="success" @click="handleAddChildButton(record)">
                    <icon-plus />
                  </a-link>
                </a-tooltip>
                <a-link v-if="!record.isPlugin" v-permission="'menus:update'" @click="handleEdit(record)">编辑</a-link>
                <a-popconfirm
                  v-if="!record.isPlugin"
                  :content="record.children && record.children.length > 0 ? '该菜单包含子菜单，删除后将同时删除所有子菜单及其关联的权限配置，确定要删除吗？' : '确定要删除该菜单吗？'"
                  @ok="handleDelete(record)"
                >
                  <a-link v-permission="'menus:delete'" status="danger">删除</a-link>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="dialogTitle"
      :width="680"
      :unmount-on-close="true"
      @ok="handleSubmit"
      @cancel="handleDialogClose"
      :ok-loading="submitting"
    >
      <a-alert v-if="editingPluginMenu" type="info" style="margin-bottom: 16px;">
        您正在编辑插件菜单，只能修改排序字段。菜单名称: {{ menuForm.name }} | 路径: {{ menuForm.path }}
      </a-alert>

      <a-form :model="menuForm" layout="vertical" ref="formRef">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="菜单名称" field="name" :rules="[{ required: true, message: '请输入菜单名称' }]">
              <a-input v-model="menuForm.name" :disabled="editingPluginMenu" placeholder="请输入菜单名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="菜单编码" field="code" :rules="[{ required: true, message: '请输入菜单编码' }]">
              <a-input v-model="menuForm.code" :disabled="editingPluginMenu" placeholder="请输入菜单编码" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="类型" field="type" :rules="[{ required: true, message: '请选择类型' }]">
          <a-radio-group v-model="menuForm.type" type="button" :disabled="editingPluginMenu">
            <a-radio :value="1">目录</a-radio>
            <a-radio :value="2">菜单</a-radio>
            <a-radio :value="3">按钮</a-radio>
          </a-radio-group>
          <div class="form-tip">
            目录：用于组织菜单的容器，不对应具体页面；菜单：对应具体页面的导航项；按钮：页面内的功能按钮，用于权限控制
          </div>
        </a-form-item>

        <a-form-item v-if="!editingPluginMenu" label="上级菜单" field="parentId">
          <a-cascader
            v-model="menuForm.parentId"
            :options="menuTreeOptions"
            :field-names="{ value: 'ID', label: 'name', children: 'children' }"
            check-strictly
            allow-clear
            placeholder="请选择上级菜单"
            style="width: 100%"
          />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12" v-if="menuForm.type !== 3">
            <a-form-item label="路由路径" field="path">
              <a-input v-model="menuForm.path" :disabled="editingPluginMenu" placeholder="请输入路由路径" />
            </a-form-item>
          </a-col>
          <a-col :span="12" v-if="menuForm.type === 2 && !editingPluginMenu">
            <a-form-item label="组件路径" field="component">
              <a-input v-model="menuForm.component" placeholder="请输入组件路径" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="图标" field="icon">
              <a-input v-model="menuForm.icon" :disabled="editingPluginMenu" placeholder="请输入图标名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="排序" field="sort">
              <a-input-number v-model="menuForm.sort" :min="0" style="width: 100%" />
              <div class="form-tip">数值越小越靠前</div>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16" v-if="!editingPluginMenu">
          <a-col :span="12">
            <a-form-item label="显示状态" field="visible">
              <a-radio-group v-model="menuForm.visible" type="button">
                <a-radio :value="1">显示</a-radio>
                <a-radio :value="0">隐藏</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态" field="status">
              <a-radio-group v-model="menuForm.status" type="button">
                <a-radio :value="1">启用</a-radio>
                <a-radio :value="0">禁用</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
        </a-row>

        <!-- API绑定 -->
        <a-form-item v-if="menuForm.type === 3 && !editingPluginMenu" label="API绑定">
          <div class="api-list">
            <div v-for="(api, index) in menuForm.apis" :key="index" class="api-row">
              <a-select v-model="api.apiMethod" placeholder="方法" style="width: 110px;">
                <a-option value="GET">GET</a-option>
                <a-option value="POST">POST</a-option>
                <a-option value="PUT">PUT</a-option>
                <a-option value="DELETE">DELETE</a-option>
              </a-select>
              <a-input v-model="api.apiPath" placeholder="如 /api/v1/users/:id" style="flex: 1;" />
              <a-link status="danger" @click="menuForm.apis.splice(index, 1)">
                <icon-delete />
              </a-link>
            </div>
            <a-link @click="menuForm.apis.push({ apiPath: '', apiMethod: '' })">
              <icon-plus style="margin-right: 4px;" />添加API
            </a-link>
          </div>
          <div class="form-tip">一个按钮可绑定多个后端API，支持 :id 等路径参数占位符</div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconMenu,
  IconPlus,
  IconRefresh,
  IconSwap,
  IconDelete,
} from '@arco-design/web-vue/es/icon'
import { getMenuTree, createMenu, updateMenu, deleteMenu } from '@/api/menu'
import { pluginManager } from '@/plugins/manager'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const editingPluginMenu = ref(false)
const formRef = ref()
const expandAll = ref(false)
const expandedKeys = ref<(string | number)[]>([])

const getRowKey = (row: any) => row.ID || row.id || row.path

// 搜索
const searchForm = reactive({
  name: '',
  status: undefined as number | undefined
})

const menuList = ref<any[]>([])
const menuTreeOptions = ref<any[]>([])

// HTTP 方法颜色
const getMethodColor = (method: string) => {
  switch (method) {
    case 'GET': return 'green'
    case 'POST': return 'arcoblue'
    case 'PUT': return 'orangered'
    case 'DELETE': return 'red'
    default: return 'gray'
  }
}

// 过滤菜单树
const filteredMenuList = computed(() => {
  if (!searchForm.name && searchForm.status === undefined) {
    return menuList.value
  }
  return filterTree(menuList.value)
})

const filterTree = (nodes: any[]): any[] => {
  const result: any[] = []
  for (const node of nodes) {
    const matchName = !searchForm.name || node.name?.toLowerCase().includes(searchForm.name.toLowerCase())
    const matchStatus = searchForm.status === undefined || node.status === searchForm.status

    let filteredChildren: any[] | undefined = undefined
    if (node.children && node.children.length > 0) {
      const children = filterTree(node.children)
      if (children.length > 0) filteredChildren = children
    }

    if ((matchName && matchStatus) || filteredChildren) {
      result.push({ ...node, children: filteredChildren })
    }
  }
  return result
}

// 收集所有有children的节点key
const collectAllKeys = (nodes: any[]): (string | number)[] => {
  const keys: (string | number)[] = []
  const traverse = (items: any[]) => {
    items.forEach(item => {
      const key = getRowKey(item)
      if (item.children && item.children.length > 0) {
        keys.push(key)
        traverse(item.children)
      }
    })
  }
  traverse(nodes)
  return keys
}

// 搜索时自动展开
watch([() => searchForm.name, () => searchForm.status], () => {
  if (searchForm.name || searchForm.status !== undefined) {
    expandAll.value = true
    expandedKeys.value = collectAllKeys(filteredMenuList.value)
  }
})

const handleReset = () => {
  searchForm.name = ''
  searchForm.status = undefined
  expandAll.value = false
  expandedKeys.value = []
}

const toggleExpandAll = () => {
  expandAll.value = !expandAll.value
  if (expandAll.value) {
    expandedKeys.value = collectAllKeys(filteredMenuList.value)
  } else {
    expandedKeys.value = []
  }
}

// 表单
const menuForm = reactive({
  id: 0,
  name: '',
  code: '',
  type: 2,
  parentId: 0 as number | number[],
  path: '',
  component: '',
  icon: '',
  sort: 0,
  visible: 1,
  status: 1,
  apiPath: '',
  apiMethod: '',
  apis: [] as Array<{ apiPath: string; apiMethod: string }>
})

// 插件菜单排序
const PLUGIN_MENU_SORT_KEY = 'opshub_plugin_menu_sort'

const loadPluginMenuSort = (): Map<string, number> => {
  try {
    const stored = localStorage.getItem(PLUGIN_MENU_SORT_KEY)
    if (stored) return new Map(Object.entries(JSON.parse(stored)))
  } catch { /* ignore */ }
  return new Map()
}

const savePluginMenuSort = (menuPath: string, sort: number) => {
  try {
    const sortMap = loadPluginMenuSort()
    sortMap.set(menuPath, sort)
    localStorage.setItem(PLUGIN_MENU_SORT_KEY, JSON.stringify(Object.fromEntries(sortMap)))
  } catch { /* ignore */ }
}

// 构建插件菜单
const buildPluginMenuList = () => {
  const pluginMenus: any[] = []
  const customSort = loadPluginMenuSort()

  pluginManager.getInstalled().forEach(plugin => {
    if (plugin.getMenus) {
      plugin.getMenus().forEach(menu => {
        const parentId = (menu.parentPath && menu.parentPath !== '') ? menu.parentPath : null
        const sort = customSort.get(menu.path) ?? menu.sort

        pluginMenus.push({
          ID: menu.path, id: menu.path,
          name: menu.name, code: menu.path.replace(/\//g, '_'),
          type: menu.parentPath && menu.parentPath !== '' ? 2 : 1,
          parentId, path: menu.path, component: '', icon: menu.icon,
          sort, visible: menu.hidden ? 0 : 1, status: 1,
          isPlugin: true, pluginName: plugin.name
        })
      })
    }
  })
  return pluginMenus
}

// 在树中查找菜单
const findMenuInTree = (tree: any[], menuId: string | number): any => {
  for (const menu of tree) {
    if ((menu.ID || menu.id) === menuId) return menu
    if (menu.children?.length) {
      const found = findMenuInTree(menu.children, menuId)
      if (found) return found
    }
  }
  return null
}

// 检查菜单code是否已存在
const menuCodeExists = (tree: any[], code: string): boolean => {
  for (const menu of tree) {
    if (menu.code === code) return true
    if (menu.children?.length && menuCodeExists(menu.children, code)) return true
  }
  return false
}

// 插入插件菜单
const insertPluginMenus = (tree: any[], pluginMenus: any[]) => {
  pluginMenus.forEach(pm => {
    if (menuCodeExists(tree, pm.code)) return
    if (!pm.parentId) {
      tree.push(pm)
    } else {
      const parent = findMenuInTree(tree, pm.parentId)
      if (parent) {
        if (!parent.children) parent.children = []
        parent.children.push(pm)
      }
    }
  })
}

// 清理空children
const cleanEmptyChildren = (nodes: any[]) => {
  nodes.forEach(node => {
    if (node.children && Array.isArray(node.children)) {
      if (node.children.length === 0) {
        delete node.children
      } else {
        cleanEmptyChildren(node.children)
      }
    }
  })
}

const loadMenus = async () => {
  loading.value = true
  try {
    let systemMenus: any[] = []
    try {
      systemMenus = await getMenuTree() || []
    } catch { /* ignore */ }

    const pluginMenus = buildPluginMenuList()

    if (systemMenus.length > 0) {
      systemMenus = JSON.parse(JSON.stringify(systemMenus))
      cleanEmptyChildren(systemMenus)
    }

    menuList.value = systemMenus || []

    if (pluginMenus.length > 0) {
      insertPluginMenus(menuList.value, pluginMenus)
    }

    menuTreeOptions.value = JSON.parse(JSON.stringify(systemMenus || []))
    menuTreeOptions.value.unshift({ ID: 0, name: '顶级菜单' })

    if (expandAll.value) {
      expandedKeys.value = collectAllKeys(menuList.value)
    }
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  editingPluginMenu.value = false
  Object.assign(menuForm, {
    id: 0, name: '', code: '', type: 2, parentId: 0,
    path: '', component: '', icon: '', sort: 0,
    visible: 1, status: 1, apiPath: '', apiMethod: '', apis: []
  })
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  isEdit.value = false
  editingPluginMenu.value = false
  dialogTitle.value = '新增菜单'
  resetForm()
  dialogVisible.value = true
}

const handleAddChildMenu = (row: any) => {
  isEdit.value = false
  editingPluginMenu.value = false
  dialogTitle.value = '新增下级菜单'
  resetForm()
  menuForm.parentId = row.ID || row.id
  menuForm.type = 2
  dialogVisible.value = true
}

const handleAddChildButton = (row: any) => {
  isEdit.value = false
  editingPluginMenu.value = false
  dialogTitle.value = '新增下级按钮'
  resetForm()
  menuForm.parentId = row.ID || row.id
  menuForm.type = 3
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  editingPluginMenu.value = false
  dialogTitle.value = '编辑菜单'
  menuForm.id = row.ID || row.id
  menuForm.name = row.name
  menuForm.code = row.code
  menuForm.type = row.type
  menuForm.parentId = row.parentId === 0 ? 0 : (row.parentId || 0)
  menuForm.path = row.path
  menuForm.component = row.component
  menuForm.icon = row.icon
  menuForm.sort = row.sort
  menuForm.visible = row.visible
  menuForm.status = row.status
  menuForm.apiPath = row.apiPath || ''
  menuForm.apiMethod = row.apiMethod || ''
  if (row.apis && row.apis.length > 0) {
    menuForm.apis = row.apis.map((a: any) => ({ apiPath: a.apiPath, apiMethod: a.apiMethod }))
  } else if (row.apiPath && row.apiMethod) {
    menuForm.apis = [{ apiPath: row.apiPath, apiMethod: row.apiMethod }]
  } else {
    menuForm.apis = []
  }
  dialogVisible.value = true
}

const handleEditPluginSort = (row: any) => {
  isEdit.value = true
  editingPluginMenu.value = true
  dialogTitle.value = '调整插件菜单排序'
  menuForm.id = row.ID || row.id
  menuForm.name = row.name
  menuForm.code = row.code
  menuForm.type = row.type
  menuForm.parentId = row.parentId || 0
  menuForm.path = row.path
  menuForm.component = row.component || ''
  menuForm.icon = row.icon
  menuForm.sort = row.sort
  menuForm.visible = row.visible
  menuForm.status = row.status
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await deleteMenu(row.ID || row.id)
    Message.success('删除成功')
    loadMenus()
  } catch {
    Message.error('删除失败')
  }
}

const handleSubmit = async () => {
  const errors = await formRef.value?.validate()
  if (errors) return

  submitting.value = true
  try {
    if (editingPluginMenu.value) {
      savePluginMenuSort(menuForm.path, menuForm.sort)
      Message.success(`插件菜单 "${menuForm.name}" 排序已更新`)
      dialogVisible.value = false
      resetForm()
      loadMenus()
      window.dispatchEvent(new CustomEvent('plugins-changed'))
      return
    }

    const data = { ...menuForm }
    if (Array.isArray(data.parentId)) {
      const lastValue = data.parentId[data.parentId.length - 1]
      data.parentId = (lastValue !== null && lastValue !== undefined) ? lastValue : 0
    }
    data.parentId = Number(data.parentId)

    const apis = menuForm.apis.filter(a => a.apiPath && a.apiMethod)
    ;(data as any).apis = apis

    if (isEdit.value) {
      await updateMenu(menuForm.id, data)
      Message.success('更新成功')
    } else {
      await createMenu(data)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    resetForm()
    loadMenus()
  } catch {
    Message.error('操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDialogClose = () => {
  resetForm()
  dialogVisible.value = false
}

onMounted(() => {
  loadMenus()
})
</script>

<style scoped lang="scss">
.page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-card {
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
  padding: 20px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-header-actions {
  display: flex;
  gap: 12px;
}

.page-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--ops-primary, #165dff) 0%, #4080ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.4;
}

.page-desc {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 2px;
}

.search-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.search-label {
  font-size: 14px;
  color: var(--ops-text-secondary, #4e5969);
  white-space: nowrap;
}

.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.menu-icon-preview {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

.api-path-text {
  font-size: 12px;
  color: var(--ops-text-secondary, #4e5969);
  margin-left: 4px;
}

.text-muted {
  color: var(--ops-text-tertiary, #86909c);
}

/* 表单 */
.form-tip {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 4px;
  line-height: 1.5;
}

.api-list {
  width: 100%;
}

.api-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
</style>
