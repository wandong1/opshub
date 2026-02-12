<template>
  <el-drawer
    :model-value="visible"
    :title="'MongoDB 控制台 - ' + (middleware?.name || '')"
    direction="rtl"
    size="100%"
    :destroy-on-close="true"
    class="mongo-console-drawer"
    @close="emit('update:visible', false)"
  >
    <template #header>
      <div class="console-header">
        <div class="header-left">
          <el-icon style="font-size: 18px; color: #67c23a;"><Connection /></el-icon>
          <span class="header-title">MongoDB 控制台 - {{ middleware?.name }}</span>
        </div>
        <div class="header-actions">
          <el-select v-model="currentDatabase" placeholder="选择数据库" size="small" style="width: 180px" @change="handleDatabaseChange">
            <el-option v-for="db in databases" :key="db" :label="db" :value="db" />
          </el-select>
          <el-button size="small" @click="activeRightTab = 'info'">服务器状态</el-button>
        </div>
      </div>
    </template>

    <div class="console-body">
      <!-- 左侧栏 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <el-icon><Coin /></el-icon>
          <span>集合列表</span>
          <div style="flex:1"></div>
          <el-tooltip content="新建集合" placement="top">
            <el-icon class="sidebar-action" @click="showCreateCollDialog"><Plus /></el-icon>
          </el-tooltip>
          <el-tooltip content="刷新" placement="top">
            <el-icon class="sidebar-action" @click="refreshCollections"><Refresh /></el-icon>
          </el-tooltip>
        </div>
        <div class="collection-list" v-loading="collectionsLoading">
          <div
            v-for="col in collections"
            :key="col"
            class="collection-item"
            :class="{ active: currentCollection === col }"
            @click="handleSelectCollection(col)"
          >
            <el-icon style="color: #67c23a;"><Document /></el-icon>
            <span class="collection-name">{{ col }}</span>
          </div>
          <div v-if="!collections.length && !collectionsLoading" class="collection-empty">暂无集合</div>
        </div>
      </div>
<!-- SPLIT_MONGO_MAIN -->

      <!-- 右侧主区域 -->
      <div class="main-area">
        <el-tabs v-model="activeRightTab" type="border-card">
          <!-- 文档查询 Tab -->
          <el-tab-pane label="文档查询" name="query">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个集合</div>
            <div v-else class="query-panel">
              <!-- 查询子 Tab 栏 -->
              <div class="query-tabs-bar">
                <div
                  v-for="tab in queryTabs"
                  :key="tab.id"
                  class="query-tab"
                  :class="{ active: activeQueryTab === tab.id }"
                  @click="switchQueryTab(tab.id)"
                  @dblclick="startRenameTab(tab)"
                >
                  <span v-if="renamingTabId !== tab.id" class="query-tab-name">{{ tab.name }}</span>
                  <input
                    v-else
                    v-model="renamingTabName"
                    class="query-tab-rename-input"
                    @blur="finishRenameTab"
                    @keyup.enter="finishRenameTab"
                    @keyup.escape="cancelRenameTab"
                    @click.stop
                  />
                  <el-icon v-if="queryTabs.length > 1" class="query-tab-close" @click.stop="closeQueryTab(tab.id)"><Close /></el-icon>
                </div>
                <div class="query-tab-add" @click="addQueryTab">
                  <el-icon><Plus /></el-icon>
                </div>
              </div>

              <div class="query-controls">
                <div class="query-row">
                  <span class="query-label">Filter:</span>
                  <el-input v-model="currentTabFilter" placeholder='{"name": "test"}' size="small" style="flex:1" />
                </div>
                <div class="query-row">
                  <span class="query-label">Sort:</span>
                  <el-input v-model="currentTabSort" placeholder='{"_id": -1}' size="small" style="width: 200px" />
                  <span class="query-label" style="margin-left:12px;">Limit:</span>
                  <el-input-number v-model="currentTabLimit" :min="1" :max="1000" size="small" style="width: 120px" />
                  <span class="query-label" style="margin-left:12px;">Skip:</span>
                  <el-input-number v-model="currentTabSkip" :min="0" size="small" style="width: 120px" />
                </div>
                <div class="query-actions">
                  <el-button type="primary" size="small" @click="executeQuery" :loading="queryLoading">查询</el-button>
                  <el-button type="success" size="small" @click="showInsertDialog">插入文档</el-button>
                  <el-button type="danger" size="small" @click="showDeleteDialog">删除文档</el-button>
                  <div style="flex:1"></div>
                  <span v-if="currentTabTotal >= 0" class="query-total">共 {{ currentTabTotal }} 条</span>
                </div>
              </div>
              <div class="query-results" v-loading="queryLoading">
                <div v-if="!currentTabResults.length && !queryLoading" class="empty-tip">执行查询以查看结果</div>
                <div v-for="(doc, idx) in currentTabResults" :key="idx" class="doc-item">
                  <div class="doc-header">
                    <span class="doc-index">#{{ (currentTab?.skip || 0) + idx + 1 }}</span>
                    <span v-if="doc._id" class="doc-id">_id: {{ doc._id }}</span>
                    <div style="flex:1"></div>
                    <el-button link type="primary" size="small" @click="showEditDialog(doc)">编辑</el-button>
                    <el-button link type="danger" size="small" @click="handleDeleteOne(doc)">删除</el-button>
                  </div>
                  <pre class="doc-json">{{ formatJson(doc) }}</pre>
                </div>
              </div>
            </div>
          </el-tab-pane>

          <!-- 命令行 Tab -->
          <el-tab-pane label="命令行" name="cli">
            <div class="cli-container">
              <div class="cli-output" ref="cliOutputRef">
                <div v-for="(item, idx) in cliHistory" :key="idx" class="cli-line">
                  <div class="cli-command">&gt; {{ item.command }}</div>
                  <pre class="cli-result" :class="{ 'cli-error': item.error }">{{ item.result }}</pre>
                </div>
              </div>
              <div class="cli-input-row">
                <span class="cli-prompt">{{ middleware?.host }}:{{ middleware?.port }}&gt;</span>
                <el-input
                  v-model="cliCommand"
                  placeholder='输入 JSON 命令'
                  size="small"
                  @keyup.enter="executeCli"
                  @keyup.up="cliHistoryUp"
                  @keyup.down="cliHistoryDown"
                />
              </div>
            </div>
          </el-tab-pane>

          <!-- 服务器信息 Tab -->
          <el-tab-pane label="服务器信息" name="info">
            <div v-loading="infoLoading" class="info-container">
              <div class="info-actions">
                <el-button size="small" @click="loadServerStats"><el-icon><Refresh /></el-icon> 刷新</el-button>
              </div>
              <div v-if="serverStats" class="info-sections">
                <el-descriptions title="基本信息" :column="2" border size="small" style="margin-bottom:16px;">
                  <el-descriptions-item label="主机">{{ serverStats.host || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="版本">{{ serverStats.version || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="进程">{{ serverStats.process || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="PID">{{ serverStats.pid || '-' }}</el-descriptions-item>
                  <el-descriptions-item label="运行时间">{{ formatUptime(serverStats.uptime) }}</el-descriptions-item>
                  <el-descriptions-item label="当前连接数">{{ serverStats.connections?.current || '-' }}</el-descriptions-item>
                </el-descriptions>
                <el-descriptions v-if="serverStats.mem" title="内存" :column="2" border size="small" style="margin-bottom:16px;">
                  <el-descriptions-item label="常驻内存">{{ serverStats.mem?.resident || '-' }} MB</el-descriptions-item>
                  <el-descriptions-item label="虚拟内存">{{ serverStats.mem?.virtual || '-' }} MB</el-descriptions-item>
                </el-descriptions>
                <el-descriptions v-if="serverStats.opcounters" title="操作计数" :column="3" border size="small" style="margin-bottom:16px;">
                  <el-descriptions-item label="insert">{{ serverStats.opcounters?.insert || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="query">{{ serverStats.opcounters?.query || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="update">{{ serverStats.opcounters?.update || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="delete">{{ serverStats.opcounters?.delete || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="getmore">{{ serverStats.opcounters?.getmore || 0 }}</el-descriptions-item>
                  <el-descriptions-item label="command">{{ serverStats.opcounters?.command || 0 }}</el-descriptions-item>
                </el-descriptions>
              </div>
              <div v-else-if="!infoLoading" class="empty-tip">点击刷新加载服务器信息</div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- 插入文档弹窗 -->
    <el-dialog v-model="insertDialog.visible" title="插入文档" width="600px" destroy-on-close append-to-body>
      <el-input v-model="insertDialog.json" type="textarea" :rows="12" placeholder='输入 JSON 文档，如 {"name": "test", "age": 18}' />
      <template #footer>
        <el-button @click="insertDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="confirmInsert" :loading="insertDialog.loading">插入</el-button>
      </template>
    </el-dialog>

    <!-- 编辑文档弹窗 -->
    <el-dialog v-model="editDialog.visible" title="编辑文档" width="600px" destroy-on-close append-to-body>
      <el-input v-model="editDialog.json" type="textarea" :rows="12" />
      <template #footer>
        <el-button @click="editDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="confirmEdit" :loading="editDialog.loading">保存</el-button>
      </template>
    </el-dialog>

    <!-- 删除文档弹窗 -->
    <el-dialog v-model="deleteDialog.visible" title="删除文档" width="500px" destroy-on-close append-to-body>
      <p style="margin-bottom:8px;">输入过滤条件（JSON），匹配的文档将被删除：</p>
      <el-input v-model="deleteDialog.filter" type="textarea" :rows="4" placeholder='如 {"name": "test"}' />
      <template #footer>
        <el-button @click="deleteDialog.visible = false">取消</el-button>
        <el-button type="danger" @click="confirmDelete" :loading="deleteDialog.loading">删除</el-button>
      </template>
    </el-dialog>

    <!-- 创建集合弹窗 -->
    <el-dialog v-model="createCollDialog.visible" title="新建集合" width="400px" destroy-on-close append-to-body>
      <el-form label-width="80px">
        <el-form-item label="集合名">
          <el-input v-model="createCollDialog.name" placeholder="请输入集合名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createCollDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="confirmCreateColl" :loading="createCollDialog.loading">创建</el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>
<!-- SPLIT_MONGO_SCRIPT -->

<script setup lang="ts">
import { ref, computed, watch, nextTick, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Connection, Refresh, Coin, Document, Plus, Close } from '@element-plus/icons-vue'
import {
  getMongoDatabases, getMongoCollections, queryMongoDocuments,
  mongoInsertDocument, mongoUpdateDocuments, mongoDeleteDocuments,
  getMongoStats, executeMiddleware, createMongoCollection
} from '@/api/middleware'

interface Props {
  visible: boolean
  middleware: any
}

const props = defineProps<Props>()
const emit = defineEmits(['update:visible'])

// --- Query Tabs ---
interface MongoQueryTab {
  id: string
  name: string
  filter: string
  sort: string
  limit: number
  skip: number
  results: any[]
  total: number
}

const queryTabs = ref<MongoQueryTab[]>([])
const activeQueryTab = ref('')
let queryTabCounter = 0

const getStorageKey = () => props.middleware?.id ? `opshub_queries_mongo_${props.middleware.id}` : ''

const saveQueryTabs = () => {
  const key = getStorageKey()
  if (!key) return
  try {
    const data = queryTabs.value.map(t => ({
      id: t.id, name: t.name, filter: t.filter, sort: t.sort, limit: t.limit, skip: t.skip
    }))
    localStorage.setItem(key, JSON.stringify(data))
  } catch {}
}

const loadQueryTabs = () => {
  const key = getStorageKey()
  if (!key) { initDefaultTab(); return }
  try {
    const raw = localStorage.getItem(key)
    if (raw) {
      const data = JSON.parse(raw) as any[]
      if (data.length > 0) {
        queryTabs.value = data.map((t: any) => ({
          id: t.id, name: t.name, filter: t.filter || '{}', sort: t.sort || '{"_id": -1}',
          limit: t.limit || 50, skip: t.skip || 0, results: [], total: -1
        }))
        const maxNum = Math.max(...data.map((t: any) => parseInt(t.id.replace('query_', '')) || 0))
        queryTabCounter = maxNum
        activeQueryTab.value = queryTabs.value[0].id
        return
      }
    }
  } catch {}
  initDefaultTab()
}

const initDefaultTab = () => {
  queryTabCounter = 1
  queryTabs.value = [{
    id: 'query_1', name: 'Query 1', filter: '{}', sort: '{"_id": -1}',
    limit: 50, skip: 0, results: [], total: -1
  }]
  activeQueryTab.value = 'query_1'
}

const currentTab = computed(() => queryTabs.value.find(t => t.id === activeQueryTab.value))

const currentTabFilter = computed({
  get: () => currentTab.value?.filter || '{}',
  set: (val: string) => { if (currentTab.value) currentTab.value.filter = val }
})
const currentTabSort = computed({
  get: () => currentTab.value?.sort || '{"_id": -1}',
  set: (val: string) => { if (currentTab.value) currentTab.value.sort = val }
})
const currentTabLimit = computed({
  get: () => currentTab.value?.limit || 50,
  set: (val: number) => { if (currentTab.value) currentTab.value.limit = val }
})
const currentTabSkip = computed({
  get: () => currentTab.value?.skip || 0,
  set: (val: number) => { if (currentTab.value) currentTab.value.skip = val }
})
const currentTabResults = computed(() => currentTab.value?.results || [])
const currentTabTotal = computed(() => currentTab.value?.total ?? -1)

const switchQueryTab = (id: string) => { activeQueryTab.value = id }

const addQueryTab = () => {
  queryTabCounter++
  const tab: MongoQueryTab = {
    id: `query_${queryTabCounter}`, name: `Query ${queryTabCounter}`,
    filter: '{}', sort: '{"_id": -1}', limit: 50, skip: 0, results: [], total: -1
  }
  queryTabs.value.push(tab)
  activeQueryTab.value = tab.id
  saveQueryTabs()
}

const closeQueryTab = (id: string) => {
  if (queryTabs.value.length <= 1) return
  const idx = queryTabs.value.findIndex(t => t.id === id)
  if (idx === -1) return
  queryTabs.value.splice(idx, 1)
  if (activeQueryTab.value === id) {
    activeQueryTab.value = queryTabs.value[Math.min(idx, queryTabs.value.length - 1)].id
  }
  saveQueryTabs()
}

const renamingTabId = ref('')
const renamingTabName = ref('')
const startRenameTab = (tab: MongoQueryTab) => {
  renamingTabId.value = tab.id
  renamingTabName.value = tab.name
  nextTick(() => {
    const input = document.querySelector('.query-tab-rename-input') as HTMLInputElement
    input?.focus()
  })
}
const finishRenameTab = () => {
  const tab = queryTabs.value.find(t => t.id === renamingTabId.value)
  if (tab && renamingTabName.value.trim()) tab.name = renamingTabName.value.trim()
  renamingTabId.value = ''
  saveQueryTabs()
}
const cancelRenameTab = () => { renamingTabId.value = '' }

// --- Core state ---
const currentDatabase = ref('')
const databases = ref<string[]>([])
const collections = ref<string[]>([])
const collectionsLoading = ref(false)
const currentCollection = ref('')
const queryLoading = ref(false)

// 命令行
const cliCommand = ref('')
const cliHistory = ref<{ command: string; result: string; error?: boolean }[]>([])
const cliCommandHistory = ref<string[]>([])
const cliHistoryIndex = ref(-1)
const cliOutputRef = ref<HTMLElement>()

// 服务器信息
const serverStats = ref<any>(null)
const infoLoading = ref(false)
const activeRightTab = ref('query')

// 弹窗
const insertDialog = reactive({ visible: false, json: '{}', loading: false })
const editDialog = reactive({ visible: false, json: '{}', loading: false, originalDoc: null as any })
const deleteDialog = reactive({ visible: false, filter: '{}', loading: false })
const createCollDialog = reactive({ visible: false, name: '', loading: false })

// Helpers
const formatJson = (obj: any) => {
  try { return JSON.stringify(obj, null, 2) } catch { return String(obj) }
}

const formatUptime = (seconds: number) => {
  if (!seconds) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}天 ${h}小时`
  if (h > 0) return `${h}小时 ${m}分钟`
  return `${m}分钟`
}

const parseJson = (str: string): any => {
  try { return JSON.parse(str) } catch (e) { throw new Error('JSON 格式错误') }
}

// Create collection
const showCreateCollDialog = () => {
  if (!currentDatabase.value) return ElMessage.warning('请先选择数据库')
  createCollDialog.name = ''
  createCollDialog.loading = false
  createCollDialog.visible = true
}

const confirmCreateColl = async () => {
  if (!createCollDialog.name.trim()) return ElMessage.warning('请输入集合名')
  createCollDialog.loading = true
  try {
    await createMongoCollection(props.middleware.id, currentDatabase.value, createCollDialog.name.trim())
    ElMessage.success('创建成功')
    createCollDialog.visible = false
    loadCollections()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    createCollDialog.loading = false
  }
}

// Data loading
const loadDatabases = async () => {
  if (!props.middleware?.id) return
  try {
    const res: any = await getMongoDatabases(props.middleware.id)
    databases.value = res || []
    if (props.middleware.databaseName && databases.value.includes(props.middleware.databaseName)) {
      currentDatabase.value = props.middleware.databaseName
    } else if (databases.value.length > 0) {
      currentDatabase.value = databases.value[0]!
    }
    if (currentDatabase.value) loadCollections()
  } catch (e: any) {
    ElMessage.error('获取数据库列表失败: ' + (e.message || ''))
  }
}

const loadCollections = async () => {
  if (!props.middleware?.id || !currentDatabase.value) return
  collectionsLoading.value = true
  try {
    const res: any = await getMongoCollections(props.middleware.id, currentDatabase.value)
    collections.value = res || []
    if (collections.value.length > 0 && !collections.value.includes(currentCollection.value)) {
      currentCollection.value = ''
    }
  } catch (e: any) {
    ElMessage.error('获取集合列表失败: ' + (e.message || ''))
  } finally {
    collectionsLoading.value = false
  }
}

const refreshCollections = () => { loadCollections() }

const handleDatabaseChange = () => {
  currentCollection.value = ''
  queryTabs.value.forEach(t => { t.results = []; t.total = -1 })
  loadCollections()
}

const handleSelectCollection = (col: string) => {
  currentCollection.value = col
  activeRightTab.value = 'query'
  queryTabs.value.forEach(t => { t.results = []; t.total = -1 })
}

const executeQuery = async () => {
  const tab = currentTab.value
  if (!tab) return
  if (!currentCollection.value) return ElMessage.warning('请选择集合')
  queryLoading.value = true
  try {
    const filter = parseJson(tab.filter || '{}')
    const sort = tab.sort ? parseJson(tab.sort) : undefined
    const res: any = await queryMongoDocuments(props.middleware.id, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      filter, sort, limit: tab.limit, skip: tab.skip
    })
    tab.results = res?.list || []
    tab.total = res?.total ?? -1
    saveQueryTabs()
  } catch (e: any) {
    ElMessage.error(e.message || '查询失败')
  } finally {
    queryLoading.value = false
  }
}

const showInsertDialog = () => {
  if (!currentCollection.value) return ElMessage.warning('请选择集合')
  insertDialog.json = '{\n  \n}'
  insertDialog.loading = false
  insertDialog.visible = true
}

const confirmInsert = async () => {
  insertDialog.loading = true
  try {
    const doc = parseJson(insertDialog.json)
    await mongoInsertDocument(props.middleware.id, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      document: doc
    })
    ElMessage.success('插入成功')
    insertDialog.visible = false
    executeQuery()
  } catch (e: any) {
    ElMessage.error(e.message || '插入失败')
  } finally {
    insertDialog.loading = false
  }
}

const showEditDialog = (doc: any) => {
  editDialog.originalDoc = doc
  const copy = { ...doc }
  delete copy._id
  editDialog.json = JSON.stringify(copy, null, 2)
  editDialog.loading = false
  editDialog.visible = true
}

const confirmEdit = async () => {
  editDialog.loading = true
  try {
    const newDoc = parseJson(editDialog.json)
    await mongoUpdateDocuments(props.middleware.id, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      filter: { _id: editDialog.originalDoc._id },
      update: { $set: newDoc }
    })
    ElMessage.success('更新成功')
    editDialog.visible = false
    executeQuery()
  } catch (e: any) {
    ElMessage.error(e.message || '更新失败')
  } finally {
    editDialog.loading = false
  }
}

const handleDeleteOne = (doc: any) => {
  if (!doc._id) return ElMessage.warning('文档缺少 _id 字段')
  ElMessageBox.confirm('确定删除该文档？', '提示', { type: 'warning' }).then(async () => {
    try {
      await mongoDeleteDocuments(props.middleware.id, {
        database: currentDatabase.value,
        collection: currentCollection.value,
        filter: { _id: doc._id }
      })
      ElMessage.success('删除成功')
      executeQuery()
    } catch (e: any) { ElMessage.error(e.message || '删除失败') }
  }).catch(() => {})
}

const showDeleteDialog = () => {
  if (!currentCollection.value) return ElMessage.warning('请选择集合')
  deleteDialog.filter = '{}'
  deleteDialog.loading = false
  deleteDialog.visible = true
}

const confirmDelete = async () => {
  deleteDialog.loading = true
  try {
    const filter = parseJson(deleteDialog.filter)
    if (!Object.keys(filter).length) {
      ElMessage.warning('不允许空条件删除')
      deleteDialog.loading = false
      return
    }
    await mongoDeleteDocuments(props.middleware.id, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      filter
    })
    ElMessage.success('删除成功')
    deleteDialog.visible = false
    executeQuery()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  } finally {
    deleteDialog.loading = false
  }
}

// CLI
const executeCli = async () => {
  const cmd = cliCommand.value.trim()
  if (!cmd) return
  cliCommandHistory.value.push(cmd)
  cliHistoryIndex.value = cliCommandHistory.value.length
  cliCommand.value = ''
  try {
    const res: any = await executeMiddleware(props.middleware.id, {
      command: cmd, database: currentDatabase.value
    })
    const output = res?.message || JSON.stringify(res?.rawResult, null, 2) || 'OK'
    cliHistory.value.push({ command: cmd, result: output })
  } catch (e: any) {
    cliHistory.value.push({ command: cmd, result: e.message || '执行失败', error: true })
  }
  nextTick(() => {
    if (cliOutputRef.value) cliOutputRef.value.scrollTop = cliOutputRef.value.scrollHeight
  })
}

const cliHistoryUp = () => {
  if (cliHistoryIndex.value > 0) {
    cliHistoryIndex.value--
    cliCommand.value = cliCommandHistory.value[cliHistoryIndex.value]
  }
}

const cliHistoryDown = () => {
  if (cliHistoryIndex.value < cliCommandHistory.value.length - 1) {
    cliHistoryIndex.value++
    cliCommand.value = cliCommandHistory.value[cliHistoryIndex.value]
  } else {
    cliHistoryIndex.value = cliCommandHistory.value.length
    cliCommand.value = ''
  }
}

// Server stats
const loadServerStats = async () => {
  if (!props.middleware?.id) return
  infoLoading.value = true
  try {
    const res: any = await getMongoStats(props.middleware.id)
    serverStats.value = res || {}
  } catch (e: any) {
    ElMessage.error('获取服务器状态失败: ' + (e.message || ''))
  } finally {
    infoLoading.value = false
  }
}

// Watch
watch(() => props.visible, (val) => {
  if (val && props.middleware) {
    currentDatabase.value = ''
    currentCollection.value = ''
    collections.value = []
    cliHistory.value = []
    cliCommandHistory.value = []
    serverStats.value = null
    activeRightTab.value = 'query'
    loadQueryTabs()
    loadDatabases()
    loadServerStats()
  }
})
</script>
<!-- SPLIT_MONGO_STYLE -->

<style scoped>
.console-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.header-title {
  font-size: 16px;
  font-weight: 600;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.console-body {
  display: flex;
  height: calc(100vh - 60px);
  overflow: hidden;
}
.sidebar {
  width: 240px;
  flex-shrink: 0;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  background: #fafafa;
}
.sidebar-header {
  padding: 10px 12px;
  font-weight: 600;
  font-size: 13px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  gap: 6px;
}
.sidebar-action {
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  padding: 2px;
  border-radius: 3px;
  transition: all 0.15s;
}
.sidebar-action:hover {
  color: #409eff;
  background: #ecf5ff;
}
.collection-list {
  flex: 1;
  overflow: auto;
  padding: 4px 0;
}
.collection-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  font-size: 13px;
  gap: 6px;
  transition: background 0.15s;
}
.collection-item:hover { background: #ecf5ff; }
.collection-item.active { background: #d9ecff; }
.collection-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collection-empty {
  text-align: center;
  color: #909399;
  padding: 30px;
  font-size: 13px;
}
.main-area {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.main-area :deep(.el-tabs) {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.main-area :deep(.el-tabs__content) {
  flex: 1;
  overflow: auto;
  padding: 12px;
}
.main-area :deep(.el-tab-pane) {
  height: 100%;
}
.empty-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #909399;
  font-size: 14px;
}
/* Query Tabs */
.query-tabs-bar {
  display: flex;
  align-items: center;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 0 4px;
  height: 32px;
  margin-bottom: 10px;
  overflow-x: auto;
  gap: 2px;
}
.query-tab {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  font-size: 12px;
  cursor: pointer;
  border-radius: 3px;
  background: transparent;
  white-space: nowrap;
  max-width: 140px;
  transition: all 0.15s;
}
.query-tab.active {
  background: #fff;
  box-shadow: 0 1px 2px rgba(0,0,0,0.1);
  font-weight: 600;
}
.query-tab:hover { background: #ecf5ff; }
.query-tab-name {
  overflow: hidden;
  text-overflow: ellipsis;
}
.query-tab-rename-input {
  border: 1px solid #409eff;
  border-radius: 2px;
  padding: 0 4px;
  font-size: 12px;
  width: 70px;
  outline: none;
}
.query-tab-close {
  font-size: 12px;
  color: #909399;
  cursor: pointer;
  border-radius: 50%;
  padding: 1px;
}
.query-tab-close:hover {
  color: #f56c6c;
  background: #fef0f0;
}
.query-tab-add {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  cursor: pointer;
  border-radius: 3px;
  color: #606266;
  font-size: 13px;
  transition: all 0.15s;
}
.query-tab-add:hover {
  color: #409eff;
  background: #ecf5ff;
}
/* Query panel */
.query-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.query-controls {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
  margin-bottom: 12px;
}
.query-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.query-label {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  white-space: nowrap;
  min-width: 45px;
}
.query-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.query-total {
  font-size: 13px;
  color: #909399;
}
.query-results {
  flex: 1;
  overflow: auto;
}
.doc-item {
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  margin-bottom: 8px;
  overflow: hidden;
}
.doc-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  font-size: 12px;
}
.doc-index {
  font-weight: 600;
  color: #409eff;
}
.doc-id {
  color: #909399;
  font-family: monospace;
}
.doc-json {
  padding: 8px 12px;
  margin: 0;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow: auto;
  background: #fff;
}
/* CLI */
.cli-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1e1e1e;
  border-radius: 4px;
  overflow: hidden;
}
.cli-output {
  flex: 1;
  overflow: auto;
  padding: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  min-height: 300px;
}
.cli-line { margin-bottom: 8px; }
.cli-command { color: #4fc1ff; }
.cli-result {
  color: #d4d4d4;
  margin: 2px 0 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: inherit;
  font-size: inherit;
}
.cli-result.cli-error { color: #f56c6c; }
.cli-input-row {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #252526;
  border-top: 1px solid #333;
  gap: 8px;
}
.cli-prompt {
  color: #67c23a;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  white-space: nowrap;
}
.cli-input-row :deep(.el-input__wrapper) {
  background: transparent;
  box-shadow: none;
}
.cli-input-row :deep(.el-input__inner) {
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
}
/* Info */
.info-container { padding: 4px; }
.info-actions { margin-bottom: 12px; }
.info-sections { max-width: 800px; }
</style>
