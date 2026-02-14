<template>
  <div class="probe-management-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div>
          <h2 class="page-title">拨测管理</h2>
          <p class="page-subtitle">管理网络拨测配置，支持 Ping / TCP / UDP 探测</p>
        </div>
      </div>
    </div>

    <!-- 搜索与操作 -->
    <div class="search-card">
      <div class="filter-bar">
        <el-input v-model="searchForm.keyword" placeholder="搜索名称或目标" clearable style="width: 220px;" @keyup.enter="loadData" />
        <el-select v-model="searchForm.category" placeholder="拨测分类" clearable style="width: 130px;" @change="handleCategoryFilter">
          <el-option v-for="c in PROBE_CATEGORIES.filter(c => c.enabled)" :key="c.value" :label="c.label" :value="c.value" />
        </el-select>
        <el-select v-model="searchForm.type" placeholder="拨测类型" clearable style="width: 130px;">
          <el-option label="Ping" value="ping" />
          <el-option label="TCP" value="tcp" />
          <el-option label="UDP" value="udp" />
        </el-select>
        <el-select v-model="searchForm.status" placeholder="状态" clearable style="width: 120px;">
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="loadData">搜索</el-button>
        <el-button :icon="Refresh" @click="handleReset">重置</el-button>
        <div style="flex: 1;" />
        <el-upload v-permission="'inspection:probes:import'" :show-file-list="false" :before-upload="handleImport" accept=".yaml,.yml,.json">
          <el-button :icon="Upload">导入</el-button>
        </el-upload>
        <el-button v-permission="'inspection:probes:export'" :icon="Download" @click="handleExport">导出</el-button>
        <el-button v-permission="'inspection:probes:create'" type="primary" :icon="Plus" class="black-button" @click="handleCreate">新增拨测</el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card"><div class="stat-value">{{ stats.total }}</div><div class="stat-label">总数</div></div>
      <div class="stat-card"><div class="stat-value" style="color:#67c23a;">{{ stats.enabled }}</div><div class="stat-label">启用</div></div>
      <div class="stat-card"><div class="stat-value" style="color:#f56c6c;">{{ stats.disabled }}</div><div class="stat-label">禁用</div></div>
      <div class="stat-card"><div class="stat-value" style="color:#409eff;">{{ stats.ping }}</div><div class="stat-label">Ping</div></div>
      <div class="stat-card"><div class="stat-value" style="color:#e6a23c;">{{ stats.tcp }}</div><div class="stat-label">TCP</div></div>
      <div class="stat-card"><div class="stat-value" style="color:#909399;">{{ stats.udp }}</div><div class="stat-label">UDP</div></div>
    </div>

    <!-- 数据表格 -->
    <el-table :data="tableData" v-loading="loading" border stripe style="width: 100%;">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="分类" width="100" align="center">
        <template #default="{ row }">
          <el-tag size="small">{{ CATEGORY_LABEL_MAP[row.category] || row.category }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.type === 'ping' ? 'primary' : row.type === 'tcp' ? 'warning' : 'info'" size="small">{{ row.type.toUpperCase() }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="target" label="目标" min-width="160" />
      <el-table-column prop="port" label="端口" width="80" align="center">
        <template #default="{ row }">{{ row.type === 'ping' ? '-' : row.port }}</template>
      </el-table-column>
      <el-table-column prop="timeout" label="超时(s)" width="90" align="center" />
      <el-table-column prop="tags" label="标签" min-width="140" show-overflow-tooltip />
      <el-table-column label="状态" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right" align="center">
        <template #default="{ row }">
          <el-button v-permission="'inspection:probes:execute'" link type="primary" @click="handleRunOnce(row)">执行</el-button>
          <el-button v-permission="'inspection:probes:update'" link type="primary" @click="handleEdit(row)">编辑</el-button>
          <el-button v-permission="'inspection:probes:delete'" link type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-container">
      <el-pagination v-model:current-page="pagination.page" v-model:page-size="pagination.pageSize"
        :total="pagination.total" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next" @change="loadData" />
    </div>

    <!-- 新建/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑拨测' : '新增拨测'" width="600px" destroy-on-close>
      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入拨测名称" />
        </el-form-item>
        <el-form-item label="分类" prop="category">
          <el-radio-group v-model="formData.category" @change="handleCategoryChange">
            <el-radio-button v-for="c in PROBE_CATEGORIES" :key="c.value" :value="c.value" :disabled="!c.enabled">
              {{ c.label }}
              <el-tooltip v-if="!c.enabled" content="即将推出" placement="top"><template #default><span /></template></el-tooltip>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="formData.type" placeholder="选择拨测类型" style="width: 100%;">
            <el-option v-for="t in availableTypes" :key="t" :label="t.toUpperCase()" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标地址" prop="target">
          <el-input v-model="formData.target" placeholder="IP 或域名" />
        </el-form-item>
        <el-form-item v-if="formData.type !== 'ping'" label="端口" prop="port">
          <el-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%;" />
        </el-form-item>
        <el-form-item label="超时(秒)">
          <el-input-number v-model="formData.timeout" :min="1" :max="60" />
        </el-form-item>
        <el-form-item v-if="formData.type === 'ping'" label="Ping次数">
          <el-input-number v-model="formData.count" :min="1" :max="100" />
        </el-form-item>
        <el-form-item v-if="formData.type === 'ping'" label="包大小">
          <el-input-number v-model="formData.packetSize" :min="16" :max="65500" />
        </el-form-item>
        <el-form-item label="业务分组">
          <el-select v-model="formData.groupId" placeholder="选择业务分组" clearable filterable style="width: 100%;">
            <el-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="formData.tags" placeholder="region=cn-east,env=prod" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="formData.description" type="textarea" :rows="2" />
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

    <!-- 执行结果对话框 -->
    <el-dialog v-model="resultDialogVisible" title="拨测结果" width="500px">
      <div v-loading="runLoading">
        <template v-if="runResult">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="状态">
              <el-tag :type="runResult.Success ? 'success' : 'danger'">{{ runResult.Success ? '成功' : '失败' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="延迟">{{ runResult.Latency?.toFixed(2) }} ms</el-descriptions-item>
            <el-descriptions-item v-if="runResult.PacketLoss !== undefined" label="丢包率">{{ (runResult.PacketLoss * 100).toFixed(1) }}%</el-descriptions-item>
            <el-descriptions-item v-if="runResult.PingRttAvg" label="平均RTT">{{ runResult.PingRttAvg?.toFixed(2) }} ms</el-descriptions-item>
            <el-descriptions-item v-if="runResult.TCPConnectTime" label="TCP连接">{{ runResult.TCPConnectTime?.toFixed(2) }} ms</el-descriptions-item>
            <el-descriptions-item v-if="runResult.Error" label="错误">{{ runResult.Error }}</el-descriptions-item>
          </el-descriptions>
        </template>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Connection, Search, Refresh, Plus, Upload, Download } from '@element-plus/icons-vue'
import { getProbeList, createProbe, updateProbe, deleteProbe, importProbes, exportProbes, runProbeOnce, PROBE_CATEGORIES, CATEGORY_TYPE_MAP, CATEGORY_LABEL_MAP } from '@/api/networkProbe'
import { getGroupTree } from '@/api/assetGroup'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<any[]>([])
const resultDialogVisible = ref(false)
const runLoading = ref(false)
const runResult = ref<any>(null)
const groupOptions = ref<any[]>([])

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const searchForm = reactive({ keyword: '', type: '', category: '', status: undefined as number | undefined })

const defaultForm = () => ({
  id: 0, name: '', category: 'network', type: 'ping', target: '', port: 80, groupId: 0,
  timeout: 5, count: 4, packetSize: 64, description: '', tags: '', status: 1
})
const formData = reactive(defaultForm())

const formRules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  target: [{ required: true, message: '请输入目标地址', trigger: 'blur' }],
}

const stats = computed(() => {
  const all = tableData.value
  return {
    total: pagination.total,
    enabled: all.filter((r: any) => r.status === 1).length,
    disabled: all.filter((r: any) => r.status === 0).length,
    ping: all.filter((r: any) => r.type === 'ping').length,
    tcp: all.filter((r: any) => r.type === 'tcp').length,
    udp: all.filter((r: any) => r.type === 'udp').length,
  }
})

const loadData = async () => {
  loading.value = true
  try {
    const res = await getProbeList({
      page: pagination.page, page_size: pagination.pageSize,
      keyword: searchForm.keyword, type: searchForm.type, category: searchForm.category, status: searchForm.status
    })
    tableData.value = res.data || []
    pagination.total = res.total || 0
  } catch { /* handled by interceptor */ } finally { loading.value = false }
}

const handleReset = () => {
  searchForm.keyword = ''; searchForm.type = ''; searchForm.category = ''; searchForm.status = undefined
  pagination.page = 1; loadData()
}

const availableTypes = computed(() => CATEGORY_TYPE_MAP[formData.category] || ['ping', 'tcp', 'udp'])

const handleCategoryChange = () => {
  const types = availableTypes.value
  if (!types.includes(formData.type)) {
    formData.type = types[0] || ''
  }
}

const handleCategoryFilter = () => {
  pagination.page = 1; loadData()
}

const loadGroups = async () => {
  try {
    const res = await getGroupTree()
    groupOptions.value = flattenGroups(res.data || res || [])
  } catch {}
}

const flattenGroups = (tree: any[], result: any[] = []): any[] => {
  for (const node of tree) {
    result.push({ id: node.id, name: node.name })
    if (node.children?.length) flattenGroups(node.children, result)
  }
  return result
}

const handleCreate = () => {
  isEdit.value = false; Object.assign(formData, defaultForm()); loadGroups(); dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true; Object.assign(formData, { ...row, category: row.category || 'network' }); loadGroups(); dialogVisible.value = true
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定删除该拨测配置？', '提示', { type: 'warning' }).then(async () => {
    await deleteProbe(row.id); ElMessage.success('删除成功'); loadData()
  }).catch(() => {})
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) { await updateProbe(formData.id, formData); ElMessage.success('更新成功') }
      else { await createProbe(formData); ElMessage.success('创建成功') }
      dialogVisible.value = false; loadData()
    } catch { /* handled */ } finally { submitting.value = false }
  })
}

const handleRunOnce = async (row: any) => {
  resultDialogVisible.value = true; runLoading.value = true; runResult.value = null
  try { runResult.value = await runProbeOnce(row.id) } catch { /* handled */ } finally { runLoading.value = false }
}

const handleImport = async (file: File) => {
  try { await importProbes(file); ElMessage.success('导入成功'); loadData() } catch { /* handled */ }
  return false
}

const handleExport = async () => {
  try {
    const blob = await exportProbes('yaml') as any
    const url = window.URL.createObjectURL(new Blob([blob]))
    const a = document.createElement('a'); a.href = url; a.download = 'probe_configs.yaml'
    a.click(); window.URL.revokeObjectURL(url)
  } catch { /* handled */ }
}

onMounted(() => { loadData() })
</script>

<style scoped>
.probe-management-container { padding: 20px; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 20px; }
.page-title { margin: 0; font-size: 18px; font-weight: 600; }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: #909399; }
.search-card { margin-bottom: 16px; }
.filter-bar { display: flex; gap: 10px; align-items: center; }
.stats-row { display: flex; gap: 12px; margin-bottom: 16px; }
.stat-card { flex: 1; padding: 16px; background: #f5f7fa; border-radius: 8px; text-align: center; }
.stat-card .stat-value { font-size: 24px; font-weight: 600; color: #303133; }
.stat-card .stat-label { font-size: 13px; color: #909399; margin-top: 4px; }
.pagination-container { margin-top: 16px; display: flex; justify-content: flex-end; }
</style>
