<template>
  <a-drawer
    :visible="visible"
    :title="'Milvus 控制台 - ' + (middleware?.name || '')"
    placement="right"
    :width="'100%'"
    unmount-on-close
    class="milvus-console-drawer"
    @cancel="emit('update:visible', false)"
  >
    <template #title>
      <div class="console-header">
        <div class="header-left">
          <icon-link style="font-size: 18px; color: #165dff;" />
          <span class="header-title">Milvus 控制台 - {{ middleware?.name }}</span>
        </div>
        <div class="header-actions">
          <a-select v-model="currentDatabase" placeholder="选择数据库" size="small" style="width: 160px" @change="handleDatabaseChange">
            <a-option v-for="db in databases" :key="db" :label="db" :value="db" />
          </a-select>
          <a-button size="small" @click="showCreateDbDialog = true">新建数据库</a-button>
          <a-button size="small" @click="showMetricsDialog = true; loadMetrics()">系统信息</a-button>
        </div>
      </div>
    </template>

    <div class="console-body">
      <!-- 左侧栏 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <icon-common />
          <span>Collections</span>
          <div style="flex:1"></div>
          <a-tooltip content="新建 Collection" position="top">
            <icon-plus class="sidebar-action" @click="showCreateCollDialog = true" />
          </a-tooltip>
          <a-tooltip content="刷新" position="top">
            <icon-refresh class="sidebar-action" @click="loadCollections" />
          </a-tooltip>
        </div>
        <a-input v-model="collSearchKeyword" placeholder="搜索..." allow-clear size="small" style="padding: 4px 8px;" />
        <a-spin :loading="collectionsLoading" style="flex: 1; overflow: auto;">
          <div class="collection-list">
            <div
              v-for="col in filteredCollections"
              :key="col.name"
              class="collection-item"
              :class="{ active: currentCollection === col.name }"
              @click="handleSelectCollection(col.name)"
            >
              <icon-file style="color: #165dff;" />
              <div class="collection-info">
                <span class="collection-name">{{ col.name }}</span>
                <a-tag v-if="col.loaded" color="green" size="small">已加载</a-tag>
              </div>
            </div>
            <div v-if="!filteredCollections.length && !collectionsLoading" class="collection-empty">暂无 Collection</div>
          </div>
        </a-spin>
      </div>

      <!-- 右侧主区域 -->
      <div class="main-area">
        <a-tabs v-model:active-key="activeTab" type="card-gutter">
          <!-- Collections Tab -->
          <a-tab-pane title="Collections" key="collections">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else>
              <a-tabs v-model:active-key="activeSubTab" type="card">
                <a-tab-pane title="Schema" key="schema">
                  <a-table :data="collectionDetail?.fields || []" :bordered="{ cell: true }" stripe size="small" style="width:100%" :pagination="false">
                    <template #columns>
                      <a-table-column title="字段名" data-index="name" :width="180" />
                      <a-table-column title="类型" data-index="dataType" :width="140" />
                      <a-table-column title="主键" :width="70" align="center">
                        <template #cell="{ record }">
                          <a-tag v-if="record.primaryKey" color="red" size="small">PK</a-tag>
                        </template>
                      </a-table-column>
                      <a-table-column title="AutoID" :width="80" align="center">
                        <template #cell="{ record }">
                          <a-tag v-if="record.autoID" color="orangered" size="small">Auto</a-tag>
                        </template>
                      </a-table-column>
                      <a-table-column title="维度" :width="100">
                        <template #cell="{ record }">{{ record.typeParams?.dim || '-' }}</template>
                      </a-table-column>
                      <a-table-column title="描述" data-index="description" />
                    </template>
                  </a-table>
                </a-tab-pane>
                <a-tab-pane title="索引" key="indexes">
                  <div style="margin-bottom: 8px;">
                    <a-button type="primary" size="small" @click="showCreateIndexDialog = true">创建索引</a-button>
                  </div>
                  <a-table :data="collectionDetail?.indexes || []" :bordered="{ cell: true }" stripe size="small" style="width:100%" :pagination="false">
                    <template #columns>
                      <a-table-column title="字段" data-index="fieldName" :width="160" />
                      <a-table-column title="索引名" data-index="indexName" :width="160" />
                      <a-table-column title="索引类型" data-index="indexType" :width="140" />
                      <a-table-column title="参数">
                        <template #cell="{ record }">{{ JSON.stringify(record.params) }}</template>
                      </a-table-column>
                      <a-table-column title="操作" :width="80" align="center">
                        <template #cell="{ record }">
                          <a-button type="text" status="danger" size="small" @click="handleDropIndex(record.fieldName)">删除</a-button>
                        </template>
                      </a-table-column>
                    </template>
                  </a-table>
                </a-tab-pane>

                <a-tab-pane title="属性" key="properties">
                  <a-descriptions :column="2" bordered size="small">
                    <a-descriptions-item label="Collection ID">{{ collectionDetail?.id }}</a-descriptions-item>
                    <a-descriptions-item label="加载状态">
                      <a-tag :color="collectionDetail?.loadState === 'Loaded' ? 'green' : 'gray'" size="small">
                        {{ collectionDetail?.loadState }}
                      </a-tag>
                    </a-descriptions-item>
                    <a-descriptions-item label="动态字段">{{ collectionDetail?.enableDynamic ? '是' : '否' }}</a-descriptions-item>
                    <a-descriptions-item label="一致性级别">{{ collectionDetail?.consistencyLevel }}</a-descriptions-item>
                    <a-descriptions-item label="描述" :span="2">{{ collectionDetail?.description || '-' }}</a-descriptions-item>
                    <a-descriptions-item label="统计信息" :span="2">{{ JSON.stringify(collectionDetail?.statistics) }}</a-descriptions-item>
                  </a-descriptions>
                  <div style="margin-top: 12px; display: flex; gap: 8px;">
                    <a-button type="primary" size="small" :disabled="collectionDetail?.loadState === 'Loaded'" @click="handleLoadCollection">加载到内存</a-button>
                    <a-button status="warning" size="small" :disabled="collectionDetail?.loadState !== 'Loaded'" @click="handleReleaseCollection">释放内存</a-button>
                    <a-button status="danger" size="small" @click="handleDropCollection">删除 Collection</a-button>
                  </div>
                </a-tab-pane>
              </a-tabs>
            </div>
          </a-tab-pane>
          <!-- 数据 Tab -->
          <a-tab-pane title="数据" key="data">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else class="data-panel">
              <div class="query-section">
                <div class="query-row">
                  <a-input v-model="queryFilter" placeholder="过滤表达式，如 id > 100" size="small" style="flex:1" />
                  <a-input-number v-model="queryLimit" :min="1" :max="10000" size="small" style="width: 120px" />
                  <a-button type="primary" size="small" :loading="queryLoading" @click="handleQuery">查询</a-button>
                </div>
                <div class="query-row" style="margin-top: 6px;">
                  <a-select v-model="queryOutputFields" multiple placeholder="输出字段（留空=全部）" size="small" style="flex:1" allow-clear>
                    <a-option v-for="f in collectionDetail?.fields || []" :key="f.name" :label="f.name" :value="f.name" />
                  </a-select>
                </div>
              </div>
              <a-table :data="queryResults" :bordered="{ cell: true }" stripe size="small" style="width:100%; margin-top: 8px;" :scroll="{ y: 400 }" :pagination="false">
                <template #columns>
                  <a-table-column v-for="col in queryColumns" :key="col" :data-index="col" :title="col" :width="140" ellipsis tooltip />
                </template>
              </a-table>
              <div style="margin-top: 4px; color: #86909c; font-size: 12px;">共 {{ queryResults.length }} 条</div>

              <a-divider />
              <div class="insert-section">
                <h4 style="margin: 0 0 8px;">插入数据</h4>
                <a-textarea v-model="insertJson" :auto-size="{ minRows: 5 }" placeholder='[{"field1": "value1", ...}]' />
                <a-button type="primary" size="small" style="margin-top: 6px;" @click="handleInsert">插入</a-button>
              </div>
              <a-divider />
              <div class="delete-section">
                <h4 style="margin: 0 0 8px;">删除数据</h4>
                <div class="query-row">
                  <a-input v-model="deleteFilter" placeholder="删除表达式，如 id in [1,2,3]" size="small" style="flex:1" />
                  <a-button status="danger" size="small" @click="handleDelete">删除</a-button>
                </div>
              </div>
            </div>
          </a-tab-pane>
          <!-- 向量搜索 Tab -->
          <a-tab-pane title="向量搜索" key="search">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else class="search-panel">
              <div class="query-row">
                <a-select v-model="searchVectorField" placeholder="向量字段" size="small" style="width: 180px">
                  <a-option v-for="f in vectorFields" :key="f.name" :label="f.name" :value="f.name" />
                </a-select>
                <a-select v-model="searchMetricType" size="small" style="width: 120px">
                  <a-option label="L2" value="L2" />
                  <a-option label="IP" value="IP" />
                  <a-option label="COSINE" value="COSINE" />
                </a-select>
                <a-input-number v-model="searchTopK" :min="1" :max="16384" size="small" style="width: 120px" />
                <a-button type="primary" size="small" :loading="searchLoading" @click="handleSearch">搜索</a-button>
              </div>
              <a-textarea v-model="searchVectors" :auto-size="{ minRows: 3 }" placeholder="向量数据 (JSON 数组)，如 [[0.1, 0.2, ...]]" style="margin-top: 6px;" />
              <div class="query-row" style="margin-top: 6px;">
                <a-input v-model="searchFilter" placeholder="过滤表达式（可选）" size="small" style="flex:1" />
              </div>
              <div class="query-row" style="margin-top: 6px;">
                <a-select v-model="searchOutputFields" multiple placeholder="输出字段" size="small" style="flex:1" allow-clear>
                  <a-option v-for="f in collectionDetail?.fields || []" :key="f.name" :label="f.name" :value="f.name" />
                </a-select>
              </div>
              <div v-if="searchResults.length" style="margin-top: 12px;">
                <div v-for="(group, gi) in searchResults" :key="gi" style="margin-bottom: 12px;">
                  <h4 style="margin: 0 0 6px;">查询向量 #{{ gi + 1 }} 结果</h4>
                  <a-table :data="group" :bordered="{ cell: true }" stripe size="small" :scroll="{ y: 300 }" :pagination="false">
                    <template #columns>
                      <a-table-column title="距离/分数" data-index="score" :width="120" />
                      <a-table-column v-for="col in searchResultColumns" :key="col" :title="col" :width="140" ellipsis tooltip>
                        <template #cell="{ record }">{{ record.fields?.[col] }}</template>
                      </a-table-column>
                    </template>
                  </a-table>
                </div>
              </div>
            </div>
          </a-tab-pane>
          <!-- 分区 Tab -->
          <a-tab-pane title="分区" key="partitions">
            <div v-if="!currentCollection" class="empty-tip">请从左侧选择一个 Collection</div>
            <div v-else>
              <div style="margin-bottom: 8px; display: flex; gap: 8px;">
                <a-input v-model="newPartitionName" placeholder="分区名称" size="small" style="width: 200px" />
                <a-button type="primary" size="small" @click="handleCreatePartition">创建分区</a-button>
                <a-button size="small" @click="loadPartitions">刷新</a-button>
              </div>
              <a-table :data="partitions" :bordered="{ cell: true }" stripe size="small" style="width:100%" :pagination="false">
                <template #columns>
                  <a-table-column title="分区名" data-index="name" />
                  <a-table-column title="ID" data-index="id" :width="120" />
                  <a-table-column title="已加载" :width="80" align="center">
                    <template #cell="{ record }">
                      <a-tag :color="record.loaded ? 'green' : 'gray'" size="small">{{ record.loaded ? '是' : '否' }}</a-tag>
                    </template>
                  </a-table-column>
                  <a-table-column title="操作" :width="80" align="center">
                    <template #cell="{ record }">
                      <a-button type="text" status="danger" size="small" @click="handleDropPartition(record.name)" :disabled="record.name === '_default'">删除</a-button>
                    </template>
                  </a-table-column>
                </template>
              </a-table>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
    </div>
    <!-- 创建数据库对话框 -->
    <a-modal v-model:visible="showCreateDbDialog" title="创建数据库" :width="400" unmount-on-close>
      <a-input v-model="newDbName" placeholder="数据库名称" />
      <template #footer>
        <a-button @click="showCreateDbDialog = false">取消</a-button>
        <a-button type="primary" @click="handleCreateDatabase">确定</a-button>
      </template>
    </a-modal>

    <!-- 创建 Collection 对话框 -->
    <a-modal v-model:visible="showCreateCollDialog" title="创建 Collection" :width="700" unmount-on-close>
      <a-form auto-label-width size="small">
        <a-form-item label="名称" field="name"><a-input v-model="newCollForm.name" /></a-form-item>
        <a-form-item label="描述" field="description"><a-input v-model="newCollForm.description" /></a-form-item>
        <a-form-item label="AutoID" field="autoID"><a-switch v-model="newCollForm.autoID" /></a-form-item>
        <a-form-item label="动态字段" field="enableDynamic"><a-switch v-model="newCollForm.enableDynamic" /></a-form-item>
        <a-form-item label="字段">
          <div v-for="(f, i) in newCollForm.fields" :key="i" style="display:flex; gap:6px; margin-bottom:6px; align-items:center; flex-wrap:wrap;">
            <a-input v-model="f.name" placeholder="字段名" style="width:120px" />
            <a-select v-model="f.dataType" placeholder="类型" style="width:140px">
              <a-option v-for="t in fieldTypes" :key="t" :label="t" :value="t" />
            </a-select>
            <a-checkbox v-model="f.primaryKey">主键</a-checkbox>
            <a-checkbox v-model="f.autoID">AutoID</a-checkbox>
            <a-input-number v-if="f.dataType === 'FloatVector' || f.dataType === 'BinaryVector'" v-model="f.dim" :min="1" placeholder="维度" style="width:100px" />
            <a-input-number v-if="f.dataType === 'VarChar'" v-model="f.maxLength" :min="1" placeholder="最大长度" style="width:120px" />
            <a-button type="text" status="danger" @click="newCollForm.fields.splice(i, 1)"><icon-delete /></a-button>
          </div>
          <a-button size="small" @click="newCollForm.fields.push({ name: '', dataType: 'Int64', primaryKey: false, autoID: false, dim: 128, maxLength: 256, description: '' })">添加字段</a-button>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="showCreateCollDialog = false">取消</a-button>
        <a-button type="primary" @click="handleCreateCollection">确定</a-button>
      </template>
    </a-modal>
    <!-- 创建索引对话框 -->
    <a-modal v-model:visible="showCreateIndexDialog" title="创建索引" :width="500" unmount-on-close>
      <a-form auto-label-width size="small">
        <a-form-item label="字段" field="fieldName">
          <a-select v-model="newIndexForm.fieldName" style="width:100%">
            <a-option v-for="f in collectionDetail?.fields || []" :key="f.name" :label="f.name" :value="f.name" />
          </a-select>
        </a-form-item>
        <a-form-item label="索引名" field="indexName"><a-input v-model="newIndexForm.indexName" /></a-form-item>
        <a-form-item label="索引类型" field="indexType">
          <a-select v-model="newIndexForm.indexType" style="width:100%">
            <a-option v-for="t in indexTypes" :key="t" :label="t" :value="t" />
          </a-select>
        </a-form-item>
        <a-form-item label="度量类型" field="metricType">
          <a-select v-model="newIndexForm.metricType" style="width:100%">
            <a-option label="L2" value="L2" />
            <a-option label="IP" value="IP" />
            <a-option label="COSINE" value="COSINE" />
          </a-select>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="showCreateIndexDialog = false">取消</a-button>
        <a-button type="primary" @click="handleCreateIndex">确定</a-button>
      </template>
    </a-modal>

    <!-- 系统信息对话框 -->
    <a-modal v-model:visible="showMetricsDialog" title="系统信息" :width="500" unmount-on-close>
      <a-spin :loading="metricsLoading" style="width: 100%;">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="版本">{{ metrics?.version }}</a-descriptions-item>
          <a-descriptions-item label="数据库">{{ metrics?.databases?.join(', ') }}</a-descriptions-item>
          <a-descriptions-item label="Collection 数量">{{ metrics?.collectionCount }}</a-descriptions-item>
        </a-descriptions>
      </a-spin>
    </a-modal>
  </a-drawer>
</template>

<script setup lang="ts">
import { ref, computed, watch, reactive } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconLink, IconCommon, IconPlus, IconRefresh, IconFile, IconDelete
} from '@arco-design/web-vue/es/icon'
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
    Message.error('加载数据库列表失败: ' + (e.message || e))
  }
}

async function loadCollections() {
  collectionsLoading.value = true
  try {
    const res = await getMilvusCollections(mwId.value, currentDatabase.value)
    collections.value = res || []
  } catch (e: any) {
    Message.error('加载 Collection 列表失败: ' + (e.message || e))
  } finally {
    collectionsLoading.value = false
  }
}
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
    Message.error('获取 Collection 详情失败: ' + (e.message || e))
  }
  loadPartitions()
}

async function handleCreateDatabase() {
  if (!newDbName.value) return
  try {
    await createMilvusDatabase(mwId.value, newDbName.value)
    Message.success('创建成功')
    showCreateDbDialog.value = false
    newDbName.value = ''
    loadDatabases()
  } catch (e: any) {
    Message.error('创建失败: ' + (e.message || e))
  }
}

async function handleCreateCollection() {
  if (!newCollForm.name) return
  try {
    await createMilvusCollection(mwId.value, { ...newCollForm, database: currentDatabase.value })
    Message.success('创建成功')
    showCreateCollDialog.value = false
    newCollForm.name = ''
    loadCollections()
  } catch (e: any) {
    Message.error('创建失败: ' + (e.message || e))
  }
}
async function handleDropCollection() {
  Modal.warning({
    title: '确认',
    content: `确定删除 Collection "${currentCollection.value}"？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await dropMilvusCollection(mwId.value, currentCollection.value, currentDatabase.value)
        Message.success('删除成功')
        currentCollection.value = ''
        collectionDetail.value = null
        loadCollections()
      } catch (e: any) {
        Message.error('删除失败: ' + (e.message || e))
      }
    }
  })
}

async function handleLoadCollection() {
  try {
    await loadMilvusCollection(mwId.value, { collection: currentCollection.value, database: currentDatabase.value })
    Message.success('加载请求已提交')
    handleSelectCollection(currentCollection.value)
  } catch (e: any) {
    Message.error('加载失败: ' + (e.message || e))
  }
}

async function handleReleaseCollection() {
  try {
    await releaseMilvusCollection(mwId.value, { collection: currentCollection.value, database: currentDatabase.value })
    Message.success('释放成功')
    handleSelectCollection(currentCollection.value)
  } catch (e: any) {
    Message.error('释放失败: ' + (e.message || e))
  }
}

async function handleCreateIndex() {
  try {
    await createMilvusIndex(mwId.value, { ...newIndexForm, collection: currentCollection.value, database: currentDatabase.value })
    Message.success('创建成功')
    showCreateIndexDialog.value = false
    handleSelectCollection(currentCollection.value)
  } catch (e: any) {
    Message.error('创建失败: ' + (e.message || e))
  }
}
async function handleDropIndex(fieldName: string) {
  Modal.warning({
    title: '确认',
    content: `确定删除字段 "${fieldName}" 的索引？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await dropMilvusIndex(mwId.value, currentCollection.value, fieldName, currentDatabase.value)
        Message.success('删除成功')
        handleSelectCollection(currentCollection.value)
      } catch (e: any) {
        Message.error('删除失败: ' + (e.message || e))
      }
    }
  })
}

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
    Message.error('查询失败: ' + (e.message || e))
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
    Message.success(`插入成功，共 ${res?.insertCount || 0} 条`)
    insertJson.value = ''
  } catch (e: any) {
    Message.error('插入失败: ' + (e.message || e))
  }
}

async function handleDelete() {
  if (!deleteFilter.value.trim()) return
  Modal.warning({
    title: '确认',
    content: '确定执行删除操作？',
    hideCancel: false,
    onOk: async () => {
      try {
        await deleteMilvusData(mwId.value, {
          database: currentDatabase.value,
          collection: currentCollection.value,
          filter: deleteFilter.value,
        })
        Message.success('删除成功')
        deleteFilter.value = ''
      } catch (e: any) {
        Message.error('删除失败: ' + (e.message || e))
      }
    }
  })
}
async function handleSearch() {
  if (!searchVectors.value.trim()) {
    Message.warning('请输入向量数据')
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
    Message.error('搜索失败: ' + (e.message || e))
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
    Message.error('加载分区失败: ' + (e.message || e))
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
    Message.success('创建成功')
    newPartitionName.value = ''
    loadPartitions()
  } catch (e: any) {
    Message.error('创建失败: ' + (e.message || e))
  }
}

async function handleDropPartition(name: string) {
  Modal.warning({
    title: '确认',
    content: `确定删除分区 "${name}"？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await dropMilvusPartition(mwId.value, currentCollection.value, name, currentDatabase.value)
        Message.success('删除成功')
        loadPartitions()
      } catch (e: any) {
        Message.error('删除失败: ' + (e.message || e))
      }
    }
  })
}

async function loadMetrics() {
  metricsLoading.value = true
  try {
    const res = await getMilvusMetrics(mwId.value)
    metrics.value = res
  } catch (e: any) {
    Message.error('获取系统信息失败: ' + (e.message || e))
  } finally {
    metricsLoading.value = false
  }
}
</script>

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
  border-right: 1px solid var(--ops-border-color, #e5e6eb);
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
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
  background: #f5f7fa;
}
.sidebar-action {
  cursor: pointer;
  color: #86909c;
  font-size: 16px;
}
.sidebar-action:hover {
  color: #165dff;
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
  background: #e8f3ff;
}
.collection-item.active {
  background: #bedaff;
  font-weight: 600;
}
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
  color: #c9cdd4;
  font-size: 13px;
}
.main-area {
  flex: 1;
  overflow: auto;
  padding: 0;
}
.empty-tip {
  padding: 40px;
  text-align: center;
  color: #c9cdd4;
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
.milvus-console-drawer .arco-drawer-header {
  margin-bottom: 0;
  padding: 12px 20px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}
.milvus-console-drawer .arco-drawer-body {
  padding: 0;
}
</style>