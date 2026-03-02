<template>
  <div class="middlewares-page-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-storage :size="36" />
        </div>
        <div>
          <h2 class="page-title">中间件管理</h2>
          <p class="page-subtitle">管理 MySQL、Redis、ClickHouse、MongoDB、Kafka、Milvus 等中间件连接</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button v-permission="'middlewares:create'" type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          新增中间件
        </a-button>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 左侧分组树 -->
      <div class="left-panel">
        <div class="panel-header">
          <div class="panel-title">
            <icon-folder class="panel-icon" />
            <span>业务分组</span>
          </div>
        </div>
        <div class="panel-body">
          <a-input v-model="groupSearchKeyword" placeholder="搜索分组..." allow-clear size="small" class="group-search">
            <template #prefix><icon-search /></template>
          </a-input>
          <a-spin :loading="groupLoading" class="tree-container">
            <a-tree
              ref="groupTreeRef"
              :data="sidebarGroupTree"
              :field-names="{ key: 'id', title: 'name', children: 'children' }"
              :default-expand-all="true"
              @select="handleGroupSelect"
            />
          </a-spin>
        </div>
      </div>

      <!-- 右侧列表 -->
      <div class="right-panel">
        <!-- 搜索和筛选 -->
        <div class="filter-bar">
          <a-input v-model="searchKeyword" placeholder="搜索名称/地址..." allow-clear style="width: 240px" @keyup.enter="handleSearch" @clear="handleSearch">
            <template #prefix><icon-search /></template>
          </a-input>
          <a-select v-model="filterType" placeholder="类型筛选" allow-clear allow-search style="width: 150px" @change="handleSearch">
            <a-option label="MySQL" value="mysql" />
            <a-option label="Redis" value="redis" />
            <a-option label="ClickHouse" value="clickhouse" />
            <a-option label="MongoDB" value="mongodb" />
            <a-option label="Kafka" value="kafka" />
            <a-option label="Milvus" value="milvus" />
          </a-select>
          <a-select v-model="filterStatus" placeholder="状态筛选" allow-clear style="width: 120px" @change="handleSearch">
            <a-option label="在线" :value="1" />
            <a-option label="离线" :value="0" />
            <a-option label="未知" :value="-1" />
          </a-select>
          <div style="flex: 1"></div>
          <a-button v-permission="'middlewares:batch-delete'" :disabled="!selectedIds.length" status="danger" @click="handleBatchDelete">
            批量删除
          </a-button>
        </div>

        <!-- 表格 -->
        <a-table :data="tableData" :loading="tableLoading" :bordered="{ cell: true }" stripe row-key="id" :row-selection="{ type: 'checkbox', showCheckedAll: true }" @selection-change="handleSelectionChange" style="width: 100%" :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, pageSizeOptions: [10, 20, 50], showTotal: true, showPageSize: true }" @page-change="handlePageChange" @page-size-change="handlePageSizeChange">
          <template #columns>
            <a-table-column title="名称" data-index="name" :min-width="140" />
            <a-table-column title="类型" data-index="typeText" :width="120">
              <template #cell="{ record }">
                <a-tag :color="getTypeTagColor(record.type)" size="small">{{ record.typeText }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="连接地址" :min-width="180">
              <template #cell="{ record }">{{ record.host }}:{{ record.port }}</template>
            </a-table-column>
            <a-table-column title="业务分组" data-index="groupName" :width="120" />
            <a-table-column title="关联主机" data-index="hostName" :width="120" :tooltip="true" />
            <a-table-column title="状态" data-index="statusText" :width="80">
              <template #cell="{ record }">
                <a-tag :color="record.status === 1 ? 'green' : record.status === 0 ? 'red' : 'gray'" size="small">{{ record.statusText }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="版本" data-index="version" :width="100" />
            <a-table-column title="操作" :width="260" fixed="right">
              <template #cell="{ record }">
                <a-button v-permission="'middlewares:update'" type="text" size="small" @click="handleEdit(record)">编辑</a-button>
                <a-button v-permission="'middlewares:connect'" type="text" status="success" size="small" @click="handleTestConnection(record)">测试</a-button>
                <a-button v-permission="'middlewares:execute'" type="text" status="warning" size="small" @click="handleExecute(record)">操作</a-button>
                <a-button v-permission="'middlewares:delete'" type="text" status="danger" size="small" @click="handleDelete(record)">删除</a-button>
              </template>
            </a-table-column>
          </template>
        </a-table>
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <a-modal v-model:visible="dialogVisible" :title="dialogTitle" :width="600" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" auto-label-width layout="horizontal">
        <a-form-item label="名称" field="name">
          <a-input v-model="formData.name" placeholder="请输入中间件名称" />
        </a-form-item>
        <a-form-item label="类型" field="type">
          <a-select v-model="formData.type" placeholder="请选择类型" allow-clear allow-search style="width: 100%" @change="handleTypeChange">
            <a-option label="MySQL" value="mysql" />
            <a-option label="Redis" value="redis" />
            <a-option label="ClickHouse" value="clickhouse" />
            <a-option label="MongoDB" value="mongodb" />
            <a-option label="Kafka" value="kafka" />
            <a-option label="Milvus" value="milvus" />
          </a-select>
        </a-form-item>
        <a-form-item label="业务分组" field="groupId">
          <a-tree-select v-model="formData.groupId" :data="groupTreeOptions" :field-names="{ key: 'id', title: 'name', children: 'children' }" placeholder="请选择分组" style="width: 100%" />
        </a-form-item>
        <a-form-item label="关联主机">
          <a-select v-model="formData.hostIds" multiple allow-search placeholder="请选择关联主机（可多选）" style="width: 100%">
            <a-option v-for="h in hostOptions" :key="h.id" :label="`${h.name} (${h.ip || h.host})`" :value="h.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="连接地址" field="host">
          <a-input v-model="formData.host" placeholder="请输入连接地址" />
        </a-form-item>
        <a-form-item label="端口" field="port">
          <a-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%" />
        </a-form-item>
        <a-form-item label="用户名">
          <a-input v-model="formData.username" placeholder="请输入用户名" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model="formData.password" placeholder="请输入密码" />
        </a-form-item>
        <!-- Kafka 认证配置 -->
        <template v-if="formData.type === 'kafka'">
          <a-form-item label="认证模式">
            <a-select v-model="kafkaParams.authMode" style="width: 100%" @change="handleKafkaAuthChange">
              <a-option label="免密" value="none" />
              <a-option label="SASL/PLAIN" value="sasl_plain" />
              <a-option label="SASL/SCRAM-256" value="sasl_scram256" />
              <a-option label="SASL/SCRAM-512" value="sasl_scram512" />
              <a-option label="Kerberos" value="kerberos" />
            </a-select>
          </a-form-item>
          <template v-if="kafkaParams.authMode === 'kerberos'">
            <a-form-item label="Service Name">
              <a-input v-model="kafkaParams.kerberosServiceName" placeholder="默认 kafka" />
            </a-form-item>
            <a-form-item label="Realm">
              <a-input v-model="kafkaParams.kerberosRealm" placeholder="Kerberos Realm" />
            </a-form-item>
            <a-form-item label="Principal">
              <a-input v-model="kafkaParams.kerberosPrincipal" placeholder="Kerberos Principal（或复用用户名）" />
            </a-form-item>
            <a-form-item label="Keytab">
              <a-radio-group v-model="keytabMode" size="small" type="button" style="margin-bottom: 8px">
                <a-radio value="data">粘贴内容</a-radio>
                <a-radio value="path">指定路径</a-radio>
              </a-radio-group>
              <a-input v-if="keytabMode === 'path'" v-model="kafkaParams.kerberosKeytab" placeholder="/path/to/keytab" />
              <div v-else>
                <a-textarea v-model="kafkaParams.kerberosKeytabData" :auto-size="{ minRows: 2 }" placeholder="Base64 编码的 keytab 内容" />
                <a-upload :auto-upload="false" :show-file-list="false" @change="handleKeytabUpload" style="margin-top: 4px">
                  <template #upload-button>
                    <a-button size="small">上传 keytab 文件</a-button>
                  </template>
                </a-upload>
              </div>
            </a-form-item>
            <a-form-item label="krb5.conf">
              <a-radio-group v-model="krb5Mode" size="small" type="button" style="margin-bottom: 8px">
                <a-radio value="data">粘贴内容</a-radio>
                <a-radio value="path">指定路径</a-radio>
              </a-radio-group>
              <a-input v-if="krb5Mode === 'path'" v-model="kafkaParams.kerberosKrb5Conf" placeholder="/etc/krb5.conf" />
              <div v-else>
                <a-textarea v-model="kafkaParams.kerberosKrb5Data" :auto-size="{ minRows: 3 }" placeholder="krb5.conf 文本内容" />
                <a-upload :auto-upload="false" :show-file-list="false" @change="handleKrb5Upload" style="margin-top: 4px">
                  <template #upload-button>
                    <a-button size="small">上传 krb5.conf</a-button>
                  </template>
                </a-upload>
              </div>
            </a-form-item>
          </template>
          <a-form-item label="TLS">
            <a-switch v-model="kafkaParams.useTLS" />
          </a-form-item>
          <a-form-item>
            <a-alert type="info" :closable="false">
              连接地址支持逗号分隔的多 Broker 地址，如 broker1:9092,broker2:9092
            </a-alert>
          </a-form-item>
        </template>
        <a-form-item label="默认数据库">
          <a-input v-model="formData.databaseName" placeholder="请输入默认数据库名" />
        </a-form-item>
        <a-form-item label="标签">
          <a-input v-model="formData.tags" placeholder="多个标签用逗号分隔" />
        </a-form-item>
        <a-form-item label="备注">
          <a-textarea v-model="formData.description" :auto-size="{ minRows: 2 }" placeholder="请输入备注" />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</a-button>
      </template>
    </a-modal>

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
import { Message, Modal } from '@arco-design/web-vue'
import { IconPlus, IconSearch, IconFolder, IconStorage } from '@arco-design/web-vue/es/icon'
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
  name: [{ required: true, message: '请输入名称' }],
  type: [{ required: true, message: '请选择类型' }],
  host: [{ required: true, message: '请输入连接地址' }],
  port: [{ required: true, message: '请输入端口' }],
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

const handleKeytabUpload = (_fileList: any[], fileItem: any) => {
  const file = fileItem.file
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    const result = e.target?.result as ArrayBuffer
    const bytes = new Uint8Array(result)
    let binary = ''
    bytes.forEach(b => binary += String.fromCharCode(b))
    kafkaParams.kerberosKeytabData = btoa(binary)
  }
  reader.readAsArrayBuffer(file)
}

const handleKrb5Upload = (_fileList: any[], fileItem: any) => {
  const file = fileItem.file
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    kafkaParams.kerberosKrb5Data = e.target?.result as string
  }
  reader.readAsText(file)
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

const handleGroupSelect = (selectedKeys: (string | number)[], data: { node?: any }) => {
  const node = data.node
  const id = node?.id ?? (selectedKeys.length > 0 ? selectedKeys[0] : undefined)
  selectedGroupId.value = id === 0 ? undefined : id
  pagination.page = 1
  loadList()
}

const handleSelectionChange = (rowKeys: (string | number)[]) => {
  selectedIds.value = rowKeys.map((k) => Number(k))
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadList()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  loadList()
}

const getTypeTagColor = (type: string) => {
  const map: Record<string, string> = { mysql: 'blue', redis: 'red', clickhouse: 'orangered', mongodb: 'green', kafka: 'gray', milvus: 'blue' }
  return map[type] || 'blue'
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
  const errors = await formRef.value?.validate()
  if (errors) return
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
      Message.success('更新成功')
    } else {
      await createMiddleware(data)
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
    content: `确定删除中间件「${row.name}」？`,
    hideCancel: false,
    onOk: async () => {
      await deleteMiddleware(row.id)
      Message.success('删除成功')
      loadList()
    }
  })
}

const handleBatchDelete = () => {
  Modal.warning({
    title: '提示',
    content: `确定删除选中的 ${selectedIds.value.length} 个中间件？`,
    hideCancel: false,
    onOk: async () => {
      await batchDeleteMiddlewares(selectedIds.value)
      Message.success('批量删除成功')
      loadList()
    }
  })
}

const handleTestConnection = async (row: any) => {
  try {
    const res = await testMiddlewareConnection(row.id)
    if (res?.success) {
      Message.success(`连接成功，版本: ${res.version || '未知'}，延迟: ${res.latency}ms`)
    } else {
      Message.error(res?.message || '连接失败')
    }
    loadList()
  } catch (e: any) {
    Message.error(e.message || '测试连接失败')
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
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--ops-primary);
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
  color: var(--ops-text-primary);
}
.page-subtitle {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary);
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
</style>
