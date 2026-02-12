<template>
  <div class="middlewares-page-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Coin /></el-icon>
        </div>
        <div>
          <h2 class="page-title">中间件管理</h2>
          <p class="page-subtitle">管理 MySQL、Redis、ClickHouse、MongoDB、Kafka、Milvus 等中间件连接</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button v-permission="'middlewares:create'" type="primary" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增中间件
        </el-button>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 左侧分组树 -->
      <div class="left-panel">
        <div class="panel-header">
          <div class="panel-title">
            <el-icon class="panel-icon"><Collection /></el-icon>
            <span>业务分组</span>
          </div>
        </div>
        <div class="panel-body">
          <el-input v-model="groupSearchKeyword" placeholder="搜索分组..." clearable size="small" class="group-search">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <div class="tree-container" v-loading="groupLoading">
            <el-tree
              ref="groupTreeRef"
              :data="sidebarGroupTree"
              :props="{ label: 'name', children: 'children' }"
              :default-expand-all="true"
              :highlight-current="true"
              node-key="id"
              @node-click="handleGroupClick"
            />
          </div>
        </div>
      </div>

      <!-- 右侧列表 -->
      <div class="right-panel">
        <!-- 搜索和筛选 -->
        <div class="filter-bar">
          <el-input v-model="searchKeyword" placeholder="搜索名称/地址..." clearable style="width: 240px" @keyup.enter="handleSearch" @clear="handleSearch">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-select v-model="filterType" placeholder="类型筛选" clearable style="width: 150px" @change="handleSearch">
            <el-option label="MySQL" value="mysql" />
            <el-option label="Redis" value="redis" />
            <el-option label="ClickHouse" value="clickhouse" />
            <el-option label="MongoDB" value="mongodb" />
            <el-option label="Kafka" value="kafka" />
            <el-option label="Milvus" value="milvus" />
          </el-select>
          <el-select v-model="filterStatus" placeholder="状态筛选" clearable style="width: 120px" @change="handleSearch">
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
            <el-option label="未知" :value="-1" />
          </el-select>
          <div style="flex: 1"></div>
          <el-button v-permission="'middlewares:batch-delete'" :disabled="!selectedIds.length" type="danger" plain @click="handleBatchDelete">
            批量删除
          </el-button>
        </div>

        <!-- 表格 -->
        <el-table :data="tableData" v-loading="tableLoading" @selection-change="handleSelectionChange" style="width: 100%">
          <el-table-column type="selection" width="50" />
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="typeText" label="类型" width="120">
            <template #default="{ row }">
              <el-tag :type="getTypeTagColor(row.type)" size="small">{{ row.typeText }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="连接地址" min-width="180">
            <template #default="{ row }">{{ row.host }}:{{ row.port }}</template>
          </el-table-column>
          <el-table-column prop="groupName" label="业务分组" width="120" />
          <el-table-column prop="hostName" label="关联主机" width="120" show-overflow-tooltip />
          <el-table-column prop="statusText" label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : row.status === 0 ? 'danger' : 'info'" size="small">{{ row.statusText }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="version" label="版本" width="100" />
          <el-table-column label="操作" width="260" fixed="right">
            <template #default="{ row }">
              <el-button v-permission="'middlewares:update'" link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button v-permission="'middlewares:connect'" link type="success" size="small" @click="handleTestConnection(row)">测试</el-button>
              <el-button v-permission="'middlewares:execute'" link type="warning" size="small" @click="handleExecute(row)">操作</el-button>
              <el-button v-permission="'middlewares:delete'" link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <div class="pagination-container">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            @size-change="loadList"
            @current-change="loadList"
          />
        </div>
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px" destroy-on-close>
      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入中间件名称" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="formData.type" placeholder="请选择类型" style="width: 100%" @change="handleTypeChange">
            <el-option label="MySQL" value="mysql" />
            <el-option label="Redis" value="redis" />
            <el-option label="ClickHouse" value="clickhouse" />
            <el-option label="MongoDB" value="mongodb" />
            <el-option label="Kafka" value="kafka" />
            <el-option label="Milvus" value="milvus" />
          </el-select>
        </el-form-item>
        <el-form-item label="业务分组" prop="groupId">
          <el-tree-select v-model="formData.groupId" :data="groupTreeOptions" :props="{ label: 'name', children: 'children', value: 'id' }" placeholder="请选择分组" style="width: 100%" check-strictly />
        </el-form-item>
        <el-form-item label="关联主机">
          <el-select v-model="formData.hostIds" multiple filterable placeholder="请选择关联主机（可多选）" style="width: 100%">
            <el-option v-for="h in hostOptions" :key="h.id" :label="`${h.name} (${h.ip || h.host})`" :value="h.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="连接地址" prop="host">
          <el-input v-model="formData.host" placeholder="请输入连接地址" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="formData.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="formData.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <!-- Kafka 认证配置 -->
        <template v-if="formData.type === 'kafka'">
          <el-form-item label="认证模式">
            <el-select v-model="kafkaParams.authMode" style="width: 100%" @change="handleKafkaAuthChange">
              <el-option label="免密" value="none" />
              <el-option label="SASL/PLAIN" value="sasl_plain" />
              <el-option label="SASL/SCRAM-256" value="sasl_scram256" />
              <el-option label="SASL/SCRAM-512" value="sasl_scram512" />
              <el-option label="Kerberos" value="kerberos" />
            </el-select>
          </el-form-item>
          <template v-if="kafkaParams.authMode === 'kerberos'">
            <el-form-item label="Service Name">
              <el-input v-model="kafkaParams.kerberosServiceName" placeholder="默认 kafka" />
            </el-form-item>
            <el-form-item label="Realm">
              <el-input v-model="kafkaParams.kerberosRealm" placeholder="Kerberos Realm" />
            </el-form-item>
            <el-form-item label="Principal">
              <el-input v-model="kafkaParams.kerberosPrincipal" placeholder="Kerberos Principal（或复用用户名）" />
            </el-form-item>
            <el-form-item label="Keytab">
              <el-radio-group v-model="keytabMode" size="small" style="margin-bottom: 8px">
                <el-radio-button value="data">粘贴内容</el-radio-button>
                <el-radio-button value="path">指定路径</el-radio-button>
              </el-radio-group>
              <el-input v-if="keytabMode === 'path'" v-model="kafkaParams.kerberosKeytab" placeholder="/path/to/keytab" />
              <div v-else>
                <el-input v-model="kafkaParams.kerberosKeytabData" type="textarea" :rows="2" placeholder="Base64 编码的 keytab 内容" />
                <el-upload :auto-upload="false" :show-file-list="false" :on-change="handleKeytabUpload" style="margin-top: 4px">
                  <el-button size="small">上传 keytab 文件</el-button>
                </el-upload>
              </div>
            </el-form-item>
            <el-form-item label="krb5.conf">
              <el-radio-group v-model="krb5Mode" size="small" style="margin-bottom: 8px">
                <el-radio-button value="data">粘贴内容</el-radio-button>
                <el-radio-button value="path">指定路径</el-radio-button>
              </el-radio-group>
              <el-input v-if="krb5Mode === 'path'" v-model="kafkaParams.kerberosKrb5Conf" placeholder="/etc/krb5.conf" />
              <div v-else>
                <el-input v-model="kafkaParams.kerberosKrb5Data" type="textarea" :rows="3" placeholder="krb5.conf 文本内容" />
                <el-upload :auto-upload="false" :show-file-list="false" :on-change="handleKrb5Upload" style="margin-top: 4px">
                  <el-button size="small">上传 krb5.conf</el-button>
                </el-upload>
              </div>
            </el-form-item>
          </template>
          <el-form-item label="TLS">
            <el-switch v-model="kafkaParams.useTLS" />
          </el-form-item>
          <el-form-item>
            <el-alert type="info" :closable="false" show-icon>
              连接地址支持逗号分隔的多 Broker 地址，如 broker1:9092,broker2:9092
            </el-alert>
          </el-form-item>
        </template>
        <el-form-item label="默认数据库">
          <el-input v-model="formData.databaseName" placeholder="请输入默认数据库名" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="formData.tags" placeholder="多个标签用逗号分隔" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="formData.description" type="textarea" :rows="2" placeholder="请输入备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>

    <!-- SQL 控制台 -->
    <SqlConsole v-model:visible="consoleVisible" :middleware="currentMiddleware" />

    <!-- Redis 控制台 -->
    <RedisConsole v-model:visible="redisConsoleVisible" :middleware="currentMiddleware" />

    <!-- ClickHouse 控制台 -->
    <ClickHouseConsole v-model:visible="clickhouseConsoleVisible" :middleware="currentMiddleware" />

    <!-- MongoDB 控制台 -->
    <MongoConsole v-model:visible="mongoConsoleVisible" :middleware="currentMiddleware" />

    <!-- Kafka 控制台 -->
    <KafkaConsole v-model:visible="kafkaConsoleVisible" :middleware="currentMiddleware" />

    <!-- Milvus 控制台 -->
    <MilvusConsole v-model:visible="milvusConsoleVisible" :middleware="currentMiddleware" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search, Collection, Coin } from '@element-plus/icons-vue'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'
import {
  getMiddlewareList, createMiddleware, updateMiddleware, deleteMiddleware,
  batchDeleteMiddlewares, testMiddlewareConnection
} from '@/api/middleware'
import SqlConsole from './components/SqlConsole.vue'
import RedisConsole from './components/RedisConsole.vue'
import ClickHouseConsole from './components/ClickHouseConsole.vue'
import MongoConsole from './components/MongoConsole.vue'
import KafkaConsole from './components/KafkaConsole.vue'
import MilvusConsole from './components/MilvusConsole.vue'

const defaultPorts: Record<string, number> = {
  mysql: 3306, redis: 6379, clickhouse: 9000, mongodb: 27017, kafka: 9092, milvus: 19530
}

// 分组树（左侧面板用，带"全部分组"虚拟根节点）
const sidebarGroupTree = ref<any[]>([])
// 分组树（表单 tree-select 用，不带虚拟根节点）
const groupTreeOptions = ref<any[]>([])
const groupLoading = ref(false)
const groupSearchKeyword = ref('')
const groupTreeRef = ref()
const selectedGroupId = ref<number | undefined>(undefined)

// 主机列表（表单选择用）
const hostOptions = ref<any[]>([])

// 列表
const tableData = ref<any[]>([])
const tableLoading = ref(false)
const selectedIds = ref<number[]>([])
const searchKeyword = ref('')
const filterType = ref('')
const filterStatus = ref<number | undefined>(undefined)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })

// 表单
const dialogVisible = ref(false)
const dialogTitle = ref('新增中间件')
const formRef = ref()
const submitLoading = ref(false)
const formData = reactive({
  id: 0, name: '', type: '', groupId: undefined as number | undefined,
  hostIds: [] as number[],
  host: '', port: 3306, username: '', password: '', databaseName: '',
  connectionParams: '', tags: '', description: ''
})
const formRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  host: [{ required: true, message: '请输入连接地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
}

// SQL 控制台
const consoleVisible = ref(false)
const redisConsoleVisible = ref(false)
const clickhouseConsoleVisible = ref(false)
const mongoConsoleVisible = ref(false)
const kafkaConsoleVisible = ref(false)
const milvusConsoleVisible = ref(false)
const currentMiddleware = ref<any>(null)

// Kafka 认证参数
const kafkaParams = reactive({
  authMode: 'none',
  kerberosServiceName: 'kafka',
  kerberosRealm: '',
  kerberosKeytab: '',
  kerberosKeytabData: '',
  kerberosPrincipal: '',
  kerberosKrb5Conf: '',
  kerberosKrb5Data: '',
  useTLS: false,
})
const keytabMode = ref<'data' | 'path'>('data')
const krb5Mode = ref<'data' | 'path'>('data')

const handleKafkaAuthChange = () => {
  // Reset kerberos fields when switching auth mode
}

const handleKeytabUpload = (file: any) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    const result = e.target?.result as ArrayBuffer
    const bytes = new Uint8Array(result)
    let binary = ''
    bytes.forEach(b => binary += String.fromCharCode(b))
    kafkaParams.kerberosKeytabData = btoa(binary)
  }
  reader.readAsArrayBuffer(file.raw)
}

const handleKrb5Upload = (file: any) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    kafkaParams.kerberosKrb5Data = e.target?.result as string
  }
  reader.readAsText(file.raw)
}

const loadGroupTree = async () => {
  groupLoading.value = true
  try {
    const data = await getGroupTree()
    const tree = data || []
    groupTreeOptions.value = tree
    sidebarGroupTree.value = [{ id: 0, name: '全部分组', children: tree }]
  } finally {
    groupLoading.value = false
  }
}

const loadHostOptions = async () => {
  try {
    const res = await getHostList({ page: 1, pageSize: 1000 })
    hostOptions.value = res?.list || []
  } catch {}
}

const loadList = async () => {
  tableLoading.value = true
  try {
    const params: any = { page: pagination.page, pageSize: pagination.pageSize }
    if (searchKeyword.value) params.keyword = searchKeyword.value
    if (filterType.value) params.type = filterType.value
    if (filterStatus.value !== undefined && filterStatus.value !== null) params.status = filterStatus.value
    if (selectedGroupId.value) params.groupId = selectedGroupId.value
    const res = await getMiddlewareList(params)
    tableData.value = res?.list || []
    pagination.total = res?.total || 0
  } finally {
    tableLoading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  loadList()
}

const handleGroupClick = (data: any) => {
  selectedGroupId.value = data.id === 0 ? undefined : data.id
  pagination.page = 1
  loadList()
}

const handleSelectionChange = (rows: any[]) => {
  selectedIds.value = rows.map((r: any) => r.id)
}

const getTypeTagColor = (type: string) => {
  const map: Record<string, string> = { mysql: '', redis: 'danger', clickhouse: 'warning', mongodb: 'success', kafka: 'info', milvus: '' }
  return map[type] || ''
}

const handleTypeChange = (type: string) => {
  if (defaultPorts[type]) formData.port = defaultPorts[type]
}

const handleAdd = () => {
  dialogTitle.value = '新增中间件'
  Object.assign(formData, { id: 0, name: '', type: '', groupId: selectedGroupId.value, hostIds: [], host: '', port: 3306, username: '', password: '', databaseName: '', connectionParams: '', tags: '', description: '' })
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  dialogTitle.value = '编辑中间件'
  const hostIds = row.hostId ? [row.hostId] : (row.hostIds || [])
  Object.assign(formData, { id: row.id, name: row.name, type: row.type, groupId: row.groupId, hostIds, host: row.host, port: row.port, username: row.username, password: '', databaseName: row.databaseName, connectionParams: row.connectionParams, tags: Array.isArray(row.tags) ? row.tags.join(',') : row.tags, description: row.description })
  // Deserialize Kafka params
  if (row.type === 'kafka' && row.connectionParams) {
    try {
      const parsed = JSON.parse(row.connectionParams)
      Object.assign(kafkaParams, {
        authMode: parsed.authMode || 'none',
        kerberosServiceName: parsed.kerberosServiceName || 'kafka',
        kerberosRealm: parsed.kerberosRealm || '',
        kerberosKeytab: parsed.kerberosKeytab || '',
        kerberosKeytabData: parsed.kerberosKeytabData || '',
        kerberosPrincipal: parsed.kerberosPrincipal || '',
        kerberosKrb5Conf: parsed.kerberosKrb5Conf || '',
        kerberosKrb5Data: parsed.kerberosKrb5Data || '',
        useTLS: parsed.useTLS || false,
      })
    } catch {}
  } else {
    Object.assign(kafkaParams, { authMode: 'none', kerberosServiceName: 'kafka', kerberosRealm: '', kerberosKeytab: '', kerberosKeytabData: '', kerberosPrincipal: '', kerberosKrb5Conf: '', kerberosKrb5Data: '', useTLS: false })
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitLoading.value = true
  try {
    const data: any = { ...formData, hostId: formData.hostIds.length > 0 ? formData.hostIds[0] : 0 }
    delete data.hostIds
    // Serialize Kafka connection params
    if (formData.type === 'kafka') {
      data.connectionParams = JSON.stringify(kafkaParams)
    }
    if (formData.id) {
      await updateMiddleware(formData.id, data)
      ElMessage.success('更新成功')
    } else {
      await createMiddleware(data)
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
  ElMessageBox.confirm(`确定删除中间件「${row.name}」？`, '提示', { type: 'warning' }).then(async () => {
    await deleteMiddleware(row.id)
    ElMessage.success('删除成功')
    loadList()
  }).catch(() => {})
}

const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.length} 个中间件？`, '提示', { type: 'warning' }).then(async () => {
    await batchDeleteMiddlewares(selectedIds.value)
    ElMessage.success('批量删除成功')
    loadList()
  }).catch(() => {})
}

const handleTestConnection = async (row: any) => {
  try {
    const res = await testMiddlewareConnection(row.id)
    if (res?.success) {
      ElMessage.success(`连接成功，版本: ${res.version || '未知'}，延迟: ${res.latency}ms`)
    } else {
      ElMessage.error(res?.message || '连接失败')
    }
    loadList()
  } catch (e: any) {
    ElMessage.error(e.message || '测试连接失败')
  }
}

const handleExecute = (row: any) => {
  currentMiddleware.value = row
  if (row.type === 'redis') {
    redisConsoleVisible.value = true
  } else if (row.type === 'clickhouse') {
    clickhouseConsoleVisible.value = true
  } else if (row.type === 'mongodb') {
    mongoConsoleVisible.value = true
  } else if (row.type === 'kafka') {
    kafkaConsoleVisible.value = true
  } else if (row.type === 'milvus') {
    milvusConsoleVisible.value = true
  } else {
    consoleVisible.value = true
  }
}

onMounted(() => {
  loadGroupTree()
  loadHostOptions()
  loadList()
})
</script>

<style scoped>
.middlewares-page-container {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-title-group {
  display: flex;
  align-items: center;
  gap: 12px;
}
.page-title-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
}
.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}
.page-subtitle {
  margin: 2px 0 0;
  font-size: 13px;
  color: #909399;
}
.main-content {
  flex: 1;
  display: flex;
  gap: 16px;
  min-height: 0;
}
.left-panel {
  width: 240px;
  flex-shrink: 0;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #ebeef5;
  display: flex;
  flex-direction: column;
}
.panel-header {
  padding: 12px 16px;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.panel-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 14px;
}
.panel-body {
  padding: 12px;
  flex: 1;
  overflow: auto;
}
.group-search {
  margin-bottom: 10px;
}
.right-panel {
  flex: 1;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #ebeef5;
  padding: 16px;
  display: flex;
  flex-direction: column;
}
.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
  align-items: center;
}
.pagination-container {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
