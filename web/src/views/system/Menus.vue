<template>
  <div class="menus-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>菜单管理</span>
          <el-button type="primary" @click="handleAdd">新增菜单</el-button>
        </div>
      </template>

      <el-table
        :data="menuList"
        border
        stripe
        v-loading="loading"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        style="width: 100%"
      >
        <el-table-column prop="name" label="菜单名称" min-width="200" />
        <el-table-column prop="code" label="菜单编码" min-width="150" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.type === 1" type="success">目录</el-tag>
            <el-tag v-else-if="row.type === 2" type="primary">菜单</el-tag>
            <el-tag v-else type="info">按钮</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路由路径" min-width="200" />
        <el-table-column prop="icon" label="图标" width="100" />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button type="danger" size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px">
      <el-form :model="menuForm" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="菜单名称" prop="name">
          <el-input v-model="menuForm.name" />
        </el-form-item>
        <el-form-item label="菜单编码" prop="code">
          <el-input v-model="menuForm.code" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-radio-group v-model="menuForm.type">
            <el-radio :label="1">目录</el-radio>
            <el-radio :label="2">菜单</el-radio>
            <el-radio :label="3">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="上级菜单" prop="parentId">
          <el-cascader
            v-model="menuForm.parentId"
            :options="menuTreeOptions"
            :props="{ checkStrictly: true, value: 'ID', label: 'name' }"
            clearable
            placeholder="请选择上级菜单"
          />
        </el-form-item>
        <el-form-item label="路由路径" prop="path" v-if="menuForm.type !== 3">
          <el-input v-model="menuForm.path" />
        </el-form-item>
        <el-form-item label="组件路径" prop="component" v-if="menuForm.type === 2">
          <el-input v-model="menuForm.component" />
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <el-input v-model="menuForm.icon" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="menuForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="显示状态" prop="visible">
          <el-radio-group v-model="menuForm.visible">
            <el-radio :label="1">显示</el-radio>
            <el-radio :label="0">隐藏</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-radio-group v-model="menuForm.status">
            <el-radio :label="1">启用</el-radio>
            <el-radio :label="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, FormInstance } from 'element-plus'
import { getMenuTree, createMenu, updateMenu, deleteMenu } from '@/api/menu'

const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const formRef = ref<FormInstance>()

const menuList = ref([])
const menuTreeOptions = ref([])

const menuForm = reactive({
  id: 0,
  name: '',
  code: '',
  type: 2,
  parentId: 0,
  path: '',
  component: '',
  icon: '',
  sort: 0,
  visible: 1,
  status: 1
})

const rules = {
  name: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入菜单编码', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

const loadMenus = async () => {
  loading.value = true
  try {
    const res = await getMenuTree()
    menuList.value = res || []
    menuTreeOptions.value = JSON.parse(JSON.stringify(res || []))
    // 添加根节点选项 - 使用大写ID
    menuTreeOptions.value.unshift({ ID: 0, name: '顶级菜单' })
  } finally {
    loading.value = false
  }
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '新增菜单'
  resetForm()
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  dialogTitle.value = '编辑菜单'
  // 正确处理ID字段，兼容大小写
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
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该菜单吗？', '提示', { type: 'warning' })
    await deleteMenu(row.ID || row.id)
    ElMessage.success('删除成功')
    loadMenus()
  } catch (error) {
    if (error !== 'cancel') console.error(error)
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      try {
        const data = { ...menuForm }
        // 处理 parentId
        if (Array.isArray(data.parentId)) {
          const lastValue = data.parentId[data.parentId.length - 1]
          // 级联选择器返回的是数组，取最后一个值
          // 如果是空数组或者最后一项是null/undefined，设置为0（顶级菜单）
          data.parentId = (lastValue !== null && lastValue !== undefined) ? lastValue : 0
        }
        // 确保parentId是数字类型
        data.parentId = Number(data.parentId)

        console.log('提交的数据:', data)

        if (isEdit.value) {
          await updateMenu(menuForm.id, data)
        } else {
          await createMenu(data)
        }
        ElMessage.success('操作成功')
        dialogVisible.value = false
        // 重置表单
        resetForm()
        loadMenus()
      } catch (error) {
        console.error(error)
        ElMessage.error('操作失败')
      }
    }
  })
}

const resetForm = () => {
  Object.assign(menuForm, {
    id: 0,
    name: '',
    code: '',
    type: 2,
    parentId: 0,
    path: '',
    component: '',
    icon: '',
    sort: 0,
    visible: 1,
    status: 1
  })
  formRef.value?.clearValidate()
}

onMounted(() => {
  loadMenus()
})
</script>

<style scoped>
.menus-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
