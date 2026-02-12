<template>
  <el-drawer
    :model-value="visible"
    :title="'Milvus 控制台 - ' + (middleware?.name || '')"
    direction="rtl"
    size="100%"
    :destroy-on-close="true"
    class="milvus-console-drawer"
    @close="emit('update:visible', false)"
  >
    <template #header>
      <div class="console-header">
        <div class="header-left">
          <el-icon style="font-size: 18px; color: #409eff;"><Connection /></el-icon>
          <span class="header-title">Milvus 控制台 - {{ middleware?.name }}</span>
        </div>
        <div class="header-actions">
          <el-select v-model="currentDatabase" placeholder="选择数据库" size="small" style="width: 160px" @change="handleDatabaseChange">
            <el-option v-for="db in databases" :key="db" :label="db" :value="db" />
          </el-select>
          <el-button size="small" @click="showCreateDbDialog = true">新建数据库</el-button>
          <el-button size="small" @click="showMetricsDialog = true; loadMetrics()">系统信息</el-button>
        </div>
      </div>
    </template>

    <div class="console-body">
      <!-- 左侧栏 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <el-icon><Coin /></el-icon>
          <span>Collections</span>
          <div style="flex:1"></div>
          <el-tooltip content="新建 Collection" placement="top">
            <el-icon class="sidebar-action" @click="showCreateCollDialog = true"><Plus /></el-icon>
          </el-tooltip>
          <el-tooltip content="刷新" placement="top">
            <el-icon class="sidebar-action" @click="loadCollections"><Refresh /></el-icon>
          </el-tooltip>
        </div>
        <el-input v-model="collSearchKeyword" placeholder="搜索..." clearable size="small" style="padding: 4px 8px;" />
        <div class="collection-list" v-loading="collectionsLoading">
          <div
            v-for="col in filteredCollections"
            :key="col.name"
            class="collection-item"
            :class="{ active: currentCollection === col.name }"
            @click="handleSelectCollection(col.name)"
          >
<!-- TEMPLATE_CONT -->
            <el-icon style="color: #409eff;"><Document /></el-icon>
            <div class="collection-info">
              <span class="collection-name">{{ col.name }}</span>
              <el-tag v-if="col.loaded" type="success" size="small" effect="plain">已加载</el-tag>
            </div>
          </div>
          <div v-if="!filteredCollections.length && !collectionsLoading" class="collection-empty">暂无 Collection</div>
        </div>
      </div>

      <!-- 右侧主区域 -->
      <div class="main-area">
        <el-tabs v-model="activeTab" type="border-card">
          <!-- Collections Tab -->
          <el-tab-pane label="Collections" name="collections">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else>
              <el-tabs v-model="activeSubTab" type="card">
                <el-tab-pane label="Schema" name="schema">
                  <el-table :data="collectionDetail?.fields || []" border size="small" style="width:100%">
                    <el-table-column prop="name" label="字段名" width="180" />
                    <el-table-column prop="dataType" label="类型" width="140" />
                    <el-table-column label="主键" width="70" align="center">
                      <template #default="{ row }">
                        <el-tag v-if="row.primaryKey" type="danger" size="small">PK</el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column label="AutoID" width="80" align="center">
                      <template #default="{ row }">
                        <el-tag v-if="row.autoID" type="warning" size="small">Auto</el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column label="维度" width="100">
                      <template #default="{ row }">{{ row.typeParams?.dim || '-' }}</template>
                    </el-table-column>
                    <el-table-column prop="description" label="描述" />
                  </el-table>
                </el-tab-pane>
<!-- TEMPLATE_INDEX_TAB -->
                <el-tab-pane label="索引" name="indexes">
                  <div style="margin-bottom: 8px;">
                    <el-button type="primary" size="small" @click="showCreateIndexDialog = true">创建索引</el-button>
                  </div>
                  <el-table :data="collectionDetail?.indexes || []" border size="small" style="width:100%">
                    <el-table-column prop="fieldName" label="字段" width="160" />
                    <el-table-column prop="indexName" label="索引名" width="160" />
                    <el-table-column prop="indexType" label="索引类型" width="140" />
                    <el-table-column label="参数">
                      <template #default="{ row }">{{ JSON.stringify(row.params) }}</template>
                    </el-table-column>
                    <el-table-column label="操作" width="80" align="center">
                      <template #default="{ row }">
                        <el-button link type="danger" size="small" @click="handleDropIndex(row.fieldName)">删除</el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>

                <el-tab-pane label="属性" name="properties">
                  <el-descriptions :column="2" border size="small">
                    <el-descriptions-item label="Collection ID">{{ collectionDetail?.id }}</el-descriptions-item>
                    <el-descriptions-item label="加载状态">
                      <el-tag :type="collectionDetail?.loadState === 'Loaded' ? 'success' : 'info'" size="small">
                        {{ collectionDetail?.loadState }}
                      </el-tag>
                    </el-descriptions-item>
                    <el-descriptions-item label="动态字段">{{ collectionDetail?.enableDynamic ? '是' : '否' }}</el-descriptions-item>
                    <el-descriptions-item label="一致性级别">{{ collectionDetail?.consistencyLevel }}</el-descriptions-item>
                    <el-descriptions-item label="描述" :span="2">{{ collectionDetail?.description || '-' }}</el-descriptions-item>
                    <el-descriptions-item label="统计信息" :span="2">{{ JSON.stringify(collectionDetail?.statistics) }}</el-descriptions-item>
                  </el-descriptions>
                  <div style="margin-top: 12px; display: flex; gap: 8px;">
                    <el-button type="primary" size="small" :disabled="collectionDetail?.loadState === 'Loaded'" @click="handleLoadCollection">加载到内存</el-button>
                    <el-button type="warning" size="small" :disabled="collectionDetail?.loadState !== 'Loaded'" @click="handleReleaseCollection">释放内存</el-button>
                    <el-button type="danger" size="small" @click="handleDropCollection">删除 Collection</el-button>
                  </div>
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-tab-pane>
<!-- TEMPLATE_DATA_TAB -->

          <!-- 数据 Tab -->
          <el-tab-pane label="数据" name="data">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else class="data-panel">
              <div class="query-section">
                <div class="query-row">
                  <el-input v-model="queryFilter" placeholder="过滤表达式，如 id > 100" size="small" style="flex:1" />
                  <el-input-number v-model="queryLimit" :min="1" :max="10000" size="small" style="width: 120px" />
                  <el-button type="primary" size="small" :loading="queryLoading" @click="handleQuery">查询</el-button>
                </div>
                <div class="query-row" style="margin-top: 6px;">
                  <el-select v-model="queryOutputFields" multiple placeholder="输出字段（留空=全部）" size="small" style="flex:1" clearable>
                    <el-option v-for="f in collectionDetail?.fields || []" :key="f.name" :label="f.name" :value="f.name" />
                  </el-select>
                </div>
              </div>
              <el-table :data="queryResults" border size="small" style="width:100%; margin-top: 8px;" max-height="400">
                <el-table-column v-for="col in queryColumns" :key="col" :prop="col" :label="col" min-width="140" show-overflow-tooltip />
              </el-table>
              <div style="margin-top: 4px; color: #909399; font-size: 12px;">共 {{ queryResults.length }} 条</div>

              <el-divider />
              <div class="insert-section">
                <h4 style="margin: 0 0 8px;">插入数据</h4>
                <el-input v-model="insertJson" type="textarea" :rows="5" placeholder='[{"field1": "value1", ...}]' />
                <el-button type="primary" size="small" style="margin-top: 6px;" @click="handleInsert">插入</el-button>
              </div>
              <el-divider />
              <div class="delete-section">
                <h4 style="margin: 0 0 8px;">删除数据</h4>
                <div class="query-row">
                  <el-input v-model="deleteFilter" placeholder="删除表达式，如 id in [1,2,3]" size="small" style="flex:1" />
                  <el-button type="danger" size="small" @click="handleDelete">删除</el-button>
                </div>
              </div>
            </div>
          </el-tab-pane>
<!-- TEMPLATE_SEARCH_TAB -->

          <!-- 向量搜索 Tab -->
          <el-tab-pane label="向量搜索" name="search">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else class="search-panel">
              <div class="query-row">
                <el-select v-model="searchVectorField" placeholder="向量字段" size="small" style="width: 180px">
                  <el-option v-for="f in vectorFields" :key="f.name" :label="f.name" :value="f.name" />
                </el-select>
                <el-select v-model="searchMetricType" size="small" style="width: 120px">
                  <el-option label="L2" value="L2" />
                  <el-option label="IP" value="IP" />
                  <el-option label="COSINE" value="COSINE" />
                </el-select>
                <el-input-number v-model="searchTopK" :min="1" :max="16384" size="small" style="width: 120px" />
                <el-button type="primary" size="small" :loading="searchLoading" @click="handleSearch">搜索</el-button>
              </div>
              <el-input v-model="searchVectors" type="textarea" :rows="3" placeholder="向量数据 (JSON 数组)，如 [[0.1, 0.2, ...]]" style="margin-top: 6px;" />
              <div class="query-row" style="margin-top: 6px;">
                <el-input v-model="searchFilter" placeholder="过滤表达式（可选）" size="small" style="flex:1" />
              </div>
              <div class="query-row" style="margin-top: 6px;">
                <el-select v-model="searchOutputFields" multiple placeholder="输出字段" size="small" style="flex:1" clearable>
                  <el-option v-for="f in collectionDetail?.fields || []" :key="f.name" :label="f.name" :value="f.name" />
                </el-select>
              </div>
              <div v-if="searchResults.length" style="margin-top: 12px;">
                <div v-for="(group, gi) in searchResults" :key="gi" style="margin-bottom: 12px;">
                  <h4 style="margin: 0 0 6px;">查询向量 #{{ gi + 1 }} 结果</h4>
                  <el-table :data="group" border size="small" max-height="300">
                    <el-table-column prop="score" label="距离/分数" width="120" />
                    <el-table-column v-for="col in searchResultColumns" :key="col" :label="col" min-width="140" show-overflow-tooltip>
                      <template #default="{ row }">{{ row.fields?.[col] }}</template>
                    </el-table-column>
                  </el-table>
                </div>
              </div>
            </div>
          </el-tab-pane>
<!-- TEMPLATE_PARTITION_TAB -->

          <!-- 分区 Tab -->
          <el-tab-pane label="分区" name="partitions">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else>
              <div style="margin-bottom: 8px; display: flex; gap: 8px;">
                <el-input v-model="newPartitionName" placeholder="分区名称" size="small" style="width: 200px" />
                <el-button type="primary" size="small" @click="handleCreatePartition">创建分区</el-button>
                <el-button size="small" @click="loadPartitions">刷新</el-button>
              </div>
              <el-table :data="partitions" border size="small" style="width:100%">
                <el-table-column prop="name" label="分区名" />
                <el-table-column prop="id" label="ID" width="120" />
                <el-table-column label="已加载" width="80" align="center">
                  <template #default="{ row }">
                    <el-tag :type="row.loaded ? 'success' : 'info'" size="small">{{ row.loaded ? '是' : '否' }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="80" align="center">
                  <template #default="{ row }">
                    <el-button link type="danger" size="small" @click="handleDropPartition(row.name)" :disabled="row.name === '_default'">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
<!-- TEMPLATE_DIALOGS -->

    <!-- 创建数据库对话框 -->
    <el-dialog v-model="showCreateDbDialog" title="创建数据库" width="400px">
      <el-input v-model="newDbName" placeholder="数据库名称" />
      <template #footer>
        <el-button @click="showCreateDbDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateDatabase">确定</el-button>
      </template>
    </el-dialog>

    <!-- 创建 Collection 对话框 -->
    <el-dialog v-model="showCreateCollDialog" title="创建 Collection" width="700px">
      <el-form label-width="100px" size="small">
        <el-form-item label="名称"><el-input v-model="newCollForm.name" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="newCollForm.description" /></el-form-item>
        <el-form-item label="AutoID"><el-switch v-model="newCollForm.autoID" /></el-form-item>
        <el-form-item label="动态字段"><el-switch v-model="newCollForm.enableDynamic" /></el-form-item>
        <el-form-item label="字段">
          <div v-for="(f, i) in newCollForm.fields" :key="i" style="display:flex; gap:6px; margin-bottom:6px; align-items:center; flex-wrap:wrap;">
            <el-input v-model="f.name" placeholder="字段名" style="width:120px" />
            <el-select v-model="f.dataType" placeholder="类型" style="width:140px">
              <el-option v-for="t in fieldTypes" :key="t" :label="t" :value="t" />
            </el-select>
            <el-checkbox v-model="f.primaryKey">主键</el-checkbox>
            <el-checkbox v-model="f.autoID">AutoID</el-checkbox>
            <el-input-number v-if="f.dataType === 'FloatVector' || f.dataType === 'BinaryVector'" v-model="f.dim" :min="1" placeholder="维度" style="width:100px" />
            <el-input-number v-if="f.dataType === 'VarChar'" v-model="f.maxLength" :min="1" placeholder="最大长度" style="width:120px" />
            <el-button link type="danger" @click="newCollForm.fields.splice(i, 1)"><el-icon><Delete /></el-icon></el-button>
          </div>
          <el-button size="small" @click="newCollForm.fields.push({ name: '', dataType: 'Int64', primaryKey: false, autoID: false, dim: 128, maxLength: 256, description: '' })">添加字段</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateCollDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateCollection">确定</el-button>
      </template>
    </el-dialog>
<!-- TEMPLATE_MORE_DIALOGS -->

    <!-- 创建索引对话框 -->
    <el-dialog v-model="showCreateIndexDialog" title="创建索引" width="500px">
      <el-form label-width="100px" size="small">
        <el-form-item label="字段">
          <el-select v-model="newIndexForm.fieldName" style="width:100%">
            <el-option v-for="f in collectionDetail?.fields || []" :key="f.name" :label="f.name" :value="f.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="索引名"><el-input v-model="newIndexForm.indexName" /></el-form-item>
        <el-form-item label="索引类型">
          <el-select v-model="newIndexForm.indexType" style="width:100%">
            <el-option v-for="t in indexTypes" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="度量类型">
          <el-select v-model="newIndexForm.metricType" style="width:100%">
            <el-option label="L2" value="L2" />
            <el-option label="IP" value="IP" />
            <el-option label="COSINE" value="COSINE" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateIndexDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateIndex">确定</el-button>
      </template>
    </el-dialog>

    <!-- 系统信息对话框 -->
    <el-dialog v-model="showMetricsDialog" title="系统信息" width="500px">
      <el-descriptions :column="1" border size="small" v-loading="metricsLoading">
        <el-descriptions-item label="版本">{{ metrics?.version }}</el-descriptions-item>
        <el-descriptions-item label="数据库">{{ metrics?.databases?.join(', ') }}</el-descriptions-item>
        <el-descriptions-item label="Collection 数量">{{ metrics?.collectionCount }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </el-drawer>
</template>

<!-- SCRIPT_SECTION -->

<script setup lang="ts">
import { ref, computed, watch, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Connection, Coin, Plus, Refresh, Document, Delete } from '@element-plus/icons-vue'
import {
  getMilvusDatabases, createMilvusDatabase, dropMilvusDatabase,
  getMilvusCollections, describeMilvusCollection, createMilvusCollection, dropMilvusCollection,
  loadMilvusCollection, releaseMilvusCollection,
  createMilvusIndex, dropMilvusIndex,
  queryMilvusData, insertMilvusData, deleteMilvusData, searchMilvusVectors,
  getMilvusPartitions, createMilvusPartition, dropMilvusPartition,
  getMilvusMetrics
} from '@/api/middleware'

const props = defineProps<{ visible: boolean; middleware: any }>()
const emit = defineEmits(['update:visible'])

const databases = ref<string[]>([])
const currentDatabase = ref('default')
const collections = ref<any[]>([])
const collectionsLoading = ref(false)
const collSearchKeyword = ref('')
const currentCollection = ref('')
const collectionDetail = ref<any>(null)
const activeTab = ref('collections')
const activeSubTab = ref('schema')

// 查询相关
const queryFilter = ref('')
const queryLimit = ref(100)
const queryOutputFields = ref<string[]>([])
const queryResults = ref<any[]>([])
const queryColumns = ref<string[]>([])
const queryLoading = ref(false)

// 插入/删除
const insertJson = ref('')
const deleteFilter = ref('')

// 搜索相关
const searchVectorField = ref('')
const searchMetricType = ref('L2')
const searchTopK = ref(10)
const searchVectors = ref('')
const searchFilter = ref('')
const searchOutputFields = ref<string[]>([])
const searchResults = ref<any[][]>([])
const searchResultColumns = ref<string[]>([])
const searchLoading = ref(false)

// 分区
const partitions = ref<any[]>([])
const newPartitionName = ref('')

// 对话框
const showCreateDbDialog = ref(false)
const newDbName = ref('')
const showCreateCollDialog = ref(false)
const showCreateIndexDialog = ref(false)
const showMetricsDialog = ref(false)
const metrics = ref<any>(null)
const metricsLoading = ref(false)

// SCRIPT_CONT

const fieldTypes = ['Bool', 'Int8', 'Int16', 'Int32', 'Int64', 'Float', 'Double', 'VarChar', 'JSON', 'FloatVector', 'BinaryVector']
const indexTypes = ['FLAT', 'IVF_FLAT', 'IVF_SQ8', 'IVF_PQ', 'HNSW', 'DISKANN', 'AUTOINDEX', 'SCANN']

const newCollForm = reactive({
  name: '', description: '', autoID: false, enableDynamic: true,
  fields: [
    { name: 'id', dataType: 'Int64', primaryKey: true, autoID: true, dim: 128, maxLength: 256, description: '' },
    { name: 'embedding', dataType: 'FloatVector', primaryKey: false, autoID: false, dim: 128, maxLength: 256, description: '' },
  ] as any[]
})

const newIndexForm = reactive({ fieldName: '', indexName: '', indexType: 'AUTOINDEX', metricType: 'COSINE' })

const filteredCollections = computed(() => {
  if (!collSearchKeyword.value) return collections.value
  const kw = collSearchKeyword.value.toLowerCase()
  return collections.value.filter((c: any) => c.name.toLowerCase().includes(kw))
})

const vectorFields = computed(() => {
  return (collectionDetail.value?.fields || []).filter((f: any) =>
    f.dataType === 'FloatVector' || f.dataType === 'BinaryVector' || f.dataType === 'Float16Vector' || f.dataType === 'BFloat16Vector'
  )
})

const mwId = computed(() => props.middleware?.id)

watch(() => props.visible, (val) => {
  if (val && mwId.value) {
    loadDatabases()
  }
})

async function loadDatabases() {
  try {
    const res = await getMilvusDatabases(mwId.value)
    databases.value = res || []
    if (databases.value.length && !databases.value.includes(currentDatabase.value)) {
      currentDatabase.value = databases.value[0]
    }
    loadCollections()
  } catch (e: any) {
    ElMessage.error('加载数据库列表失败: ' + (e.message || e))
  }
}

async function loadCollections() {
  collectionsLoading.value = true
  try {
    const res = await getMilvusCollections(mwId.value, currentDatabase.value)
    collections.value = res || []
  } catch (e: any) {
    ElMessage.error('加载 Collection 列表失败: ' + (e.message || e))
  } finally {
    collectionsLoading.value = false
  }
}

// SCRIPT_HANDLERS

function handleDatabaseChange() {
  currentCollection.value = ''
  collectionDetail.value = null
  loadCollections()
}

async function handleSelectCollection(name: string) {
  currentCollection.value = name
  activeTab.value = 'collections'
  activeSubTab.value = 'schema'
  try {
    const res = await describeMilvusCollection(mwId.value, name, currentDatabase.value)
    collectionDetail.value = res
  } catch (e: any) {
    ElMessage.error('获取 Collection 详情失败: ' + (e.message || e))
  }
  loadPartitions()
}

async function handleCreateDatabase() {
  if (!newDbName.value) return
  try {
    await createMilvusDatabase(mwId.value, newDbName.value)
    ElMessage.success('创建成功')
    showCreateDbDialog.value = false
    newDbName.value = ''
    loadDatabases()
  } catch (e: any) {
    ElMessage.error('创建失败: ' + (e.message || e))
  }
}

async function handleCreateCollection() {
  if (!newCollForm.name) return
  try {
    await createMilvusCollection(mwId.value, { ...newCollForm, database: currentDatabase.value })
    ElMessage.success('创建成功')
    showCreateCollDialog.value = false
    newCollForm.name = ''
    loadCollections()
  } catch (e: any) {
    ElMessage.error('创建失败: ' + (e.message || e))
  }
}

async function handleDropCollection() {
  await ElMessageBox.confirm(`确定删除 Collection "${currentCollection.value}"？`, '确认')
  try {
    await dropMilvusCollection(mwId.value, currentCollection.value, currentDatabase.value)
    ElMessage.success('删除成功')
    currentCollection.value = ''
    collectionDetail.value = null
    loadCollections()
  } catch (e: any) {
    ElMessage.error('删除失败: ' + (e.message || e))
  }
}

// SCRIPT_LOAD_RELEASE

async function handleLoadCollection() {
  try {
    await loadMilvusCollection(mwId.value, { collection: currentCollection.value, database: currentDatabase.value })
    ElMessage.success('加载请求已提交')
    handleSelectCollection(currentCollection.value)
  } catch (e: any) {
    ElMessage.error('加载失败: ' + (e.message || e))
  }
}

async function handleReleaseCollection() {
  try {
    await releaseMilvusCollection(mwId.value, { collection: currentCollection.value, database: currentDatabase.value })
    ElMessage.success('释放成功')
    handleSelectCollection(currentCollection.value)
  } catch (e: any) {
    ElMessage.error('释放失败: ' + (e.message || e))
  }
}

async function handleCreateIndex() {
  try {
    await createMilvusIndex(mwId.value, { ...newIndexForm, collection: currentCollection.value, database: currentDatabase.value })
    ElMessage.success('创建成功')
    showCreateIndexDialog.value = false
    handleSelectCollection(currentCollection.value)
  } catch (e: any) {
    ElMessage.error('创建失败: ' + (e.message || e))
  }
}

async function handleDropIndex(fieldName: string) {
  await ElMessageBox.confirm(`确定删除字段 "${fieldName}" 的索引？`, '确认')
  try {
    await dropMilvusIndex(mwId.value, currentCollection.value, fieldName, currentDatabase.value)
    ElMessage.success('删除成功')
    handleSelectCollection(currentCollection.value)
  } catch (e: any) {
    ElMessage.error('删除失败: ' + (e.message || e))
  }
}

// SCRIPT_DATA_OPS

async function handleQuery() {
  queryLoading.value = true
  try {
    const res = await queryMilvusData(mwId.value, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      filter: queryFilter.value,
      outputFields: queryOutputFields.value.length ? queryOutputFields.value : undefined,
      limit: queryLimit.value,
    })
    const data = res
    queryResults.value = data?.rows || []
    if (queryResults.value.length > 0) {
      queryColumns.value = Object.keys(queryResults.value[0])
    } else {
      queryColumns.value = []
    }
  } catch (e: any) {
    ElMessage.error('查询失败: ' + (e.message || e))
  } finally {
    queryLoading.value = false
  }
}

async function handleInsert() {
  if (!insertJson.value.trim()) return
  try {
    const rows = JSON.parse(insertJson.value)
    const res = await insertMilvusData(mwId.value, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      rows: Array.isArray(rows) ? rows : [rows],
    })
    ElMessage.success(`插入成功，共 ${res?.insertCount || 0} 条`)
    insertJson.value = ''
  } catch (e: any) {
    ElMessage.error('插入失败: ' + (e.message || e))
  }
}

async function handleDelete() {
  if (!deleteFilter.value.trim()) return
  await ElMessageBox.confirm('确定执行删除操作？', '确认')
  try {
    await deleteMilvusData(mwId.value, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      filter: deleteFilter.value,
    })
    ElMessage.success('删除成功')
    deleteFilter.value = ''
  } catch (e: any) {
    ElMessage.error('删除失败: ' + (e.message || e))
  }
}

// SCRIPT_SEARCH_OPS

async function handleSearch() {
  if (!searchVectors.value.trim()) {
    ElMessage.warning('请输入向量数据')
    return
  }
  searchLoading.value = true
  try {
    const vectors = JSON.parse(searchVectors.value)
    const res = await searchMilvusVectors(mwId.value, {
      database: currentDatabase.value,
      collection: currentCollection.value,
      vectorField: searchVectorField.value,
      vectors: vectors,
      topK: searchTopK.value,
      metricType: searchMetricType.value,
      filter: searchFilter.value || undefined,
      outputFields: searchOutputFields.value.length ? searchOutputFields.value : undefined,
    })
    searchResults.value = res?.results || []
    // 提取列名
    const cols = new Set<string>()
    for (const group of searchResults.value) {
      for (const hit of group) {
        if (hit.fields) Object.keys(hit.fields).forEach(k => cols.add(k))
      }
    }
    searchResultColumns.value = Array.from(cols)
  } catch (e: any) {
    ElMessage.error('搜索失败: ' + (e.message || e))
  } finally {
    searchLoading.value = false
  }
}

async function loadPartitions() {
  if (!currentCollection.value) return
  try {
    const res = await getMilvusPartitions(mwId.value, currentCollection.value, currentDatabase.value)
    partitions.value = res || []
  } catch (e: any) {
    ElMessage.error('加载分区失败: ' + (e.message || e))
  }
}

async function handleCreatePartition() {
  if (!newPartitionName.value) return
  try {
    await createMilvusPartition(mwId.value, {
      collection: currentCollection.value,
      partition: newPartitionName.value,
      database: currentDatabase.value,
    })
    ElMessage.success('创建成功')
    newPartitionName.value = ''
    loadPartitions()
  } catch (e: any) {
    ElMessage.error('创建失败: ' + (e.message || e))
  }
}

// SCRIPT_FINAL

async function handleDropPartition(name: string) {
  await ElMessageBox.confirm(`确定删除分区 "${name}"？`, '确认')
  try {
    await dropMilvusPartition(mwId.value, currentCollection.value, name, currentDatabase.value)
    ElMessage.success('删除成功')
    loadPartitions()
  } catch (e: any) {
    ElMessage.error('删除失败: ' + (e.message || e))
  }
}

async function loadMetrics() {
  metricsLoading.value = true
  try {
    const res = await getMilvusMetrics(mwId.value)
    metrics.value = res
  } catch (e: any) {
    ElMessage.error('获取系统信息失败: ' + (e.message || e))
  } finally {
    metricsLoading.value = false
  }
}
</script>

<!-- STYLE_SECTION -->

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
  height: calc(100vh - 120px);
  gap: 0;
}
.sidebar {
  width: 280px;
  min-width: 280px;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  background: #fafafa;
}
.sidebar-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 12px;
  font-weight: 600;
  font-size: 13px;
  border-bottom: 1px solid #e4e7ed;
  background: #f5f7fa;
}
.sidebar-action {
  cursor: pointer;
  color: #909399;
  font-size: 16px;
}
.sidebar-action:hover {
  color: #409eff;
}
.collection-list {
  flex: 1;
  overflow: auto;
  padding: 4px 0;
}
.collection-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.2s;
}
.collection-item:hover {
  background: #ecf5ff;
}
.collection-item.active {
  background: #d9ecff;
  font-weight: 600;
}
/* STYLE_CONT */
.collection-info {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}
.collection-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collection-empty {
  padding: 20px;
  text-align: center;
  color: #c0c4cc;
  font-size: 13px;
}
.main-area {
  flex: 1;
  overflow: auto;
  padding: 0;
}
.main-area :deep(.el-tabs--border-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.main-area :deep(.el-tabs__content) {
  flex: 1;
  overflow: auto;
  padding: 12px;
}
.empty-tip {
  padding: 40px;
  text-align: center;
  color: #c0c4cc;
  font-size: 14px;
}
.query-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.data-panel, .search-panel {
  padding: 4px 0;
}
</style>

<style>
.milvus-console-drawer .el-drawer__header {
  margin-bottom: 0;
  padding: 12px 20px;
  border-bottom: 1px solid #e4e7ed;
}
.milvus-console-drawer .el-drawer__body {
  padding: 0;
}
</style>
















