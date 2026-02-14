<template>
  <div class="pushgateway-config-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Promotion /></el-icon>
        </div>
        <div>
          <h2 class="page-title">Pushgateway 配置</h2>
          <p class="page-subtitle">管理 Pushgateway 实例，用于拨测指标推送</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button v-permission="'inspection:pushgateways:create'" type="primary" class="black-button" :icon="Plus" @click="handleAdd">新增Pushgateway</el-button>
      </div>
    </div>

    <el-table :data="tableData" v-loading="loading" border stripe>
      <el-table-column label="名称" prop="name" min-width="140" />
      <el-table-column label="URL" prop="url" min-width="240" show-overflow-tooltip />
      <el-table-column label="用户名" prop="username" width="120">
        <template #default="{ row }">{{ row.username || '-' }}</template>
      </el-table-column>
      <el-table-column label="默认" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.isDefault === 1 ? 'warning' : 'info'" size="small">{{ row.isDefault === 1 ? '是' : '否' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right" align="center">
        <template #default="{ row }">
          <el-button v-permission="'inspection:pushgateways:update'" link type="primary" @click="handleEdit(row)">编辑</el-button>
          <el-button v-permission="'inspection:pushgateways:delete'" link type="danger" @click="handleDelete(row)">删除</el-button>
          <el-button v-permission="'inspection:pushgateways:test'" link type="success" :loading="row._testing" @click="handleTest(row)">测试</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑 Pushgateway' : '新增 Pushgateway'" width="500px" destroy-on-close>
      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="URL" prop="url">
          <el-input v-model="formData.url" placeholder="http://pushgateway:9091" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="formData.username" placeholder="Basic Auth 用户名（可选）" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="formData.password" type="password" show-password placeholder="Basic Auth 密码（可选）" />
        </el-form-item>
        <el-form-item label="默认">
          <el-radio-group v-model="formData.isDefault">
            <el-radio :value="1">是</el-radio>
            <el-radio :value="0">否</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="formData.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" class="black-button" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Promotion, Plus } from '@element-plus/icons-vue'
import { getPushgatewayList, createPushgateway, updatePushgateway, deletePushgateway, testPushgateway } from '@/api/networkProbe'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<any[]>([])

const defaultForm = () => ({ id: 0, name: '', url: '', username: '', password: '', isDefault: 0, status: 1 })
const formData = reactive(defaultForm())

const formRules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  url: [{ required: true, message: '请输入URL', trigger: 'blur' }],
}

const loadData = async () => {
  loading.value = true
  try { tableData.value = (await getPushgatewayList()) || [] } catch {} finally { loading.value = false }
}

const handleAdd = () => { isEdit.value = false; Object.assign(formData, defaultForm()); dialogVisible.value = true }
const handleEdit = (row: any) => { isEdit.value = true; Object.assign(formData, { ...row }); dialogVisible.value = true }

const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定删除？', '提示', { type: 'warning' }).then(async () => {
    await deletePushgateway(row.id); ElMessage.success('删除成功'); loadData()
  }).catch(() => {})
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) { await updatePushgateway(formData.id, formData); ElMessage.success('更新成功') }
      else { await createPushgateway(formData); ElMessage.success('创建成功') }
      dialogVisible.value = false; loadData()
    } catch {} finally { submitting.value = false }
  })
}

const handleTest = async (row: any) => {
  row._testing = true
  try { await testPushgateway(row.id); ElMessage.success('连接成功') }
  catch { /* interceptor handles */ }
  finally { row._testing = false }
}

onMounted(() => { loadData() })
</script>

<style scoped>
.pushgateway-config-container { padding: 20px; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 20px; }
.page-title { margin: 0; font-size: 18px; font-weight: 600; }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: #909399; }
</style>
