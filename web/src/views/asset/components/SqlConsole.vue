<template>
  <el-drawer
    :model-value="visible"
    :title="'SQL 控制台 - ' + (middleware?.name || '')"
    direction="rtl"
    size="100%"
    :destroy-on-close="true"
    class="sql-console-drawer"
    @close="emit('update:visible', false)"
  >
    <!-- 工具栏 -->
    <template #header>
      <div class="console-header">
        <div class="header-left">
          <el-icon style="font-size: 18px; color: #409eff;"><Monitor /></el-icon>
          <span class="header-title">SQL 控制台 - {{ middleware?.name }}</span>
        </div>
        <div class="header-actions">
          <el-select v-model="currentDatabase" placeholder="选择数据库" size="small" style="width: 180px" @change="handleDatabaseChange">
            <el-option v-for="db in databases" :key="db" :label="db" :value="db" />
          </el-select>
          <el-button type="primary" size="small" @click="executeSQL" :loading="executing">
            <el-icon style="margin-right: 4px;"><CaretRight /></el-icon>执行
          </el-button>
          <el-select v-model="queryLimit" size="small" style="width: 110px">
            <el-option :value="100" label="Limit 100" />
            <el-option :value="200" label="Limit 200" />
            <el-option :value="500" label="Limit 500" />
            <el-option :value="1000" label="Limit 1000" />
            <el-option :value="0" label="No Limit" />
          </el-select>
          <el-button size="small" @click="formatSQL">格式化</el-button>
          <el-popover placement="bottom" :width="360" trigger="click">
            <template #reference>
              <el-button size="small">历史</el-button>
            </template>
            <div class="history-panel">
              <div v-if="!queryHistory.length" class="history-empty">暂无查询历史</div>
              <div v-for="(item, idx) in queryHistory" :key="idx" class="history-item" @click="applyHistory(item.sql)">
                <div class="history-sql">{{ item.sql.substring(0, 80) }}{{ item.sql.length > 80 ? '...' : '' }}</div>
                <div class="history-time">{{ item.time }}</div>
              </div>
            </div>
          </el-popover>
          <el-button size="small" @click="exportCSV" :disabled="!currentTabResult?.columns?.length">导出 CSV</el-button>
        </div>
      </div>
    </template>
<!-- SPLIT_MARKER_BODY -->

    <!-- 主体布局 -->
    <div class="console-body">
      <!-- 左侧数据库树 -->
      <div class="sidebar" :style="{ width: sidebarWidth + 'px' }">
        <div class="sidebar-header">
          <el-icon><Coin /></el-icon>
          <span>数据库</span>
          <div style="flex:1"></div>
          <el-tooltip content="新建数据库" placement="top">
            <el-icon class="sidebar-action" @click="showCreateDbDialog"><Plus /></el-icon>
          </el-tooltip>
          <el-tooltip content="刷新" placement="top">
            <el-icon class="sidebar-action" @click="loadDatabases"><Refresh /></el-icon>
          </el-tooltip>
        </div>
        <div class="sidebar-tree" v-loading="treeLoading">
          <el-tree
            ref="treeRef"
            :data="treeData"
            :props="treeProps"
            lazy
            :load="loadTreeNode"
            node-key="id"
            @node-click="handleTreeNodeClick"
            @node-dblclick="handleTreeNodeDblClick"
          >
            <template #default="{ node, data }">
              <span class="tree-node" @contextmenu.prevent="handleTreeContextMenu($event, data)">
                <el-icon v-if="data.type === 'database'" style="color: #409eff;"><Coin /></el-icon>
                <el-icon v-else-if="data.type === 'table'" style="color: #67c23a;"><Grid /></el-icon>
                <el-icon v-else style="color: #909399;"><Document /></el-icon>
                <el-tooltip :content="node.label" placement="right" :show-after="500" :disabled="node.label.length < 20">
                  <span class="tree-node-label">{{ node.label }}</span>
                </el-tooltip>
                <span v-if="data.type === 'column'" class="column-type">{{ data.colType }}</span>
              </span>
            </template>
          </el-tree>
        </div>
      </div>

      <div class="resize-handle" @mousedown="startResize"></div>

      <!-- 右侧主区域 -->
      <div class="main-area">
        <!-- 查询 Tab 栏 -->
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

        <!-- SQL 编辑器 -->
        <div class="editor-area">
          <Codemirror
            v-model="currentTabContent"
            :extensions="editorExtensions"
            :style="{ height: '200px', fontSize: '14px' }"
            placeholder="输入 SQL 语句，按 Ctrl+Enter 执行..."
            @keydown="handleEditorKeydown"
          />
        </div>

        <!-- 结果面板 -->
        <div class="result-area">
          <div class="result-content" ref="resultContentRef" v-if="currentTabResult">
            <el-table
              v-if="currentTabResult.columns && currentTabResult.columns.length"
              :data="paginatedRows"
              :max-height="tableMaxHeight"
              size="small"
              border
              stripe
              style="width: 100%"
            >
              <el-table-column type="index" label="#" width="55" fixed />
              <el-table-column
                v-for="col in currentTabResult.columns"
                :key="col"
                :prop="col"
                :label="col"
                min-width="140"
                show-overflow-tooltip
                sortable
              />
            </el-table>
            <pre v-else-if="currentTabResult.rawResult !== undefined" class="raw-result">{{ JSON.stringify(currentTabResult.rawResult, null, 2) }}</pre>
            <div v-else-if="currentTabResult.message" class="result-message">
              <el-icon style="color: #67c23a; font-size: 16px;"><CircleCheck /></el-icon>
              <span>{{ currentTabResult.message }}</span>
            </div>
          </div>
          <div v-else class="result-empty">执行 SQL 查询以查看结果</div>

          <!-- 状态栏 + 分页 -->
          <div class="status-bar">
            <span v-if="currentTabResult?.columns">返回 {{ allResultRows.length }} 行</span>
            <span v-if="currentTabResult?.affectedRows">影响 {{ currentTabResult.affectedRows }} 行</span>
            <span v-if="currentTabResult?.duration">耗时 {{ currentTabResult.duration }}ms</span>
            <span v-if="currentDatabase">数据库: {{ currentDatabase }}</span>
            <div style="flex:1"></div>
            <el-pagination
              v-if="allResultRows.length > pageSize"
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :page-sizes="[50, 100, 200, 500]"
              :total="allResultRows.length"
              size="small"
              layout="sizes, prev, pager, next"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 右键上下文菜单 -->
    <div v-if="ctxMenu.visible" ref="ctxMenuRef" class="context-menu" :style="{ top: ctxMenu.y + 'px', left: ctxMenu.x + 'px' }">
      <template v-if="ctxMenu.nodeType === 'table'">
        <div class="ctx-item" @click="ctxOpenTable">打开表</div>
        <div class="ctx-item" @click="ctxNewQuery">新建查询</div>
        <div class="ctx-divider"></div>
        <div class="ctx-item ctx-has-sub">
          查看 DML
          <div class="ctx-submenu">
            <div class="ctx-item" @click="ctxDML('select')">SELECT</div>
            <div class="ctx-item" @click="ctxDML('insert')">INSERT</div>
            <div class="ctx-item" @click="ctxDML('update')">UPDATE</div>
            <div class="ctx-item" @click="ctxDML('delete')">DELETE</div>
          </div>
        </div>
        <div class="ctx-item" @click="ctxViewDDL">查看 DDL</div>
        <div class="ctx-divider"></div>
        <div class="ctx-item" @click="ctxRenameTable">修改表名</div>
        <div class="ctx-item" @click="ctxCopyName">复制表名</div>
        <div class="ctx-item" @click="ctxCopyBackupTable">复制备份表</div>
        <div class="ctx-divider"></div>
        <div class="ctx-item ctx-has-sub">
          导出表
          <div class="ctx-submenu">
            <div class="ctx-item" @click="ctxExportTable('all')">结构和数据</div>
            <div class="ctx-item" @click="ctxExportTable('structure')">仅结构</div>
          </div>
        </div>
        <div class="ctx-divider"></div>
        <div class="ctx-item ctx-danger" @click="ctxTruncateTable">清空表</div>
        <div class="ctx-item ctx-danger" @click="ctxDropTable">删除表</div>
      </template>
      <template v-if="ctxMenu.nodeType === 'database'">
        <div class="ctx-item" @click="ctxNewQuery">新建查询</div>
        <div class="ctx-item" @click="ctxCopyName">复制名称</div>
      </template>
      <template v-if="ctxMenu.nodeType === 'column'">
        <div class="ctx-item" @click="ctxCopyName">复制列名</div>
      </template>
    </div>

    <!-- 修改表名弹窗 -->
    <el-dialog v-model="renameTableDialog.visible" title="修改表名" width="400px" destroy-on-close append-to-body>
      <el-form label-width="80px">
        <el-form-item label="新表名">
          <el-input v-model="renameTableDialog.newName" placeholder="请输入新表名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renameTableDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="confirmRenameTable" :loading="renameTableDialog.loading">确定</el-button>
      </template>
    </el-dialog>

    <!-- 创建数据库弹窗 -->
    <el-dialog v-model="createDbDialog.visible" title="新建数据库" width="400px" destroy-on-close append-to-body>
      <el-form label-width="80px">
        <el-form-item label="数据库名">
          <el-input v-model="createDbDialog.name" placeholder="请输入数据库名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDbDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="confirmCreateDb" :loading="createDbDialog.loading">创建</el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>
<!-- SPLIT_MARKER_SCRIPT -->

<script setup lang="ts">
import { ref, computed, watch, reactive, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Monitor, CaretRight, Coin, Grid, Document, CircleCheck, Plus, Refresh, Close } from '@element-plus/icons-vue'
import { Codemirror } from 'vue-codemirror'
import { sql, MySQL } from '@codemirror/lang-sql'
import { oneDark } from '@codemirror/theme-one-dark'
import { format as formatSqlFn } from 'sql-formatter'
import {
  executeMiddleware,
  getMiddlewareDatabases,
  getMiddlewareTables,
  getMiddlewareColumns,
  createDatabase
} from '@/api/middleware'

interface Props {
  visible: boolean
  middleware: any
}

const props = defineProps<Props>()
const emit = defineEmits(['update:visible'])

// --- Query Tabs ---
interface QueryTab {
  id: string
  name: string
  content: string
  database: string
  result: any | null
  resultPage: number
}

const queryTabs = ref<QueryTab[]>([])
const activeQueryTab = ref('')
let queryTabCounter = 0

const getStorageKey = () => props.middleware?.id ? `opshub_queries_mysql_${props.middleware.id}` : ''

const saveQueryTabs = () => {
  const key = getStorageKey()
  if (!key) return
  try {
    const data = queryTabs.value.map(t => ({ id: t.id, name: t.name, content: t.content, database: t.database }))
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
          id: t.id, name: t.name, content: t.content || '', database: t.database || '',
          result: null, resultPage: 1
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
  queryTabs.value = [{ id: 'query_1', name: 'Query 1', content: '', database: '', result: null, resultPage: 1 }]
  activeQueryTab.value = 'query_1'
}

const currentTab = computed(() => queryTabs.value.find(t => t.id === activeQueryTab.value))

const currentTabContent = computed({
  get: () => currentTab.value?.content || '',
  set: (val: string) => { if (currentTab.value) currentTab.value.content = val }
})

const currentTabResult = computed(() => currentTab.value?.result || null)

const switchQueryTab = (id: string) => { activeQueryTab.value = id }

const addQueryTab = () => {
  queryTabCounter++
  const tab: QueryTab = {
    id: `query_${queryTabCounter}`, name: `Query ${queryTabCounter}`,
    content: '', database: currentDatabase.value, result: null, resultPage: 1
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

// Rename
const renamingTabId = ref('')
const renamingTabName = ref('')
const startRenameTab = (tab: QueryTab) => {
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

// --- Sidebar resize ---
const sidebarWidth = ref(240)
const startResize = (e: MouseEvent) => {
  const startX = e.clientX
  const startW = sidebarWidth.value
  const onMove = (ev: MouseEvent) => {
    sidebarWidth.value = Math.max(180, Math.min(500, startW + ev.clientX - startX))
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// --- Context menu ---
const ctxMenu = reactive({
  visible: false, x: 0, y: 0,
  nodeType: '' as string,
  dbName: '', tableName: '', colName: ''
})

const handleTreeContextMenu = (e: MouseEvent, data: any) => {
  ctxMenu.visible = true
  ctxMenu.x = e.clientX
  ctxMenu.y = e.clientY
  ctxMenu.nodeType = data.type
  ctxMenu.dbName = data.dbName || ''
  ctxMenu.tableName = data.tableName || ''
  ctxMenu.colName = data.type === 'column' ? data.label : ''
}

const ctxMenuRef = ref<HTMLElement>()
const closeCtxMenu = (e?: Event) => {
  if (e && ctxMenuRef.value?.contains(e.target as Node)) return
  ctxMenu.visible = false
}

const ctxOpenTable = () => {
  const query = `SELECT * FROM \`${ctxMenu.dbName}\`.\`${ctxMenu.tableName}\` LIMIT 100`
  if (currentTab.value) currentTab.value.content = query
  currentDatabase.value = ctxMenu.dbName
  closeCtxMenu()
  nextTick(() => executeSQL())
}

const ctxNewQuery = () => {
  addQueryTab()
  if (currentTab.value) {
    if (ctxMenu.nodeType === 'table') {
      currentTab.value.content = `SELECT * FROM \`${ctxMenu.dbName}\`.\`${ctxMenu.tableName}\``
    }
    currentTab.value.database = ctxMenu.dbName
  }
  currentDatabase.value = ctxMenu.dbName
  closeCtxMenu()
}

const ctxDML = async (type: string) => {
  closeCtxMenu()
  const db = ctxMenu.dbName
  const table = ctxMenu.tableName
  try {
    const cols: any[] = await getMiddlewareColumns(props.middleware.id, db, table) as any || []
    const colNames = cols.map((c: any) => c.field)
    const priCol = cols.find((c: any) => c.key === 'PRI')?.field || colNames[0] || 'id'

    const getPlaceholder = (c: any) => {
      const t = (c.type || '').toLowerCase()
      if (/int|decimal|float|double|numeric|bit/.test(t)) return '0'
      if (/date|time/.test(t)) return "'2026-01-01'"
      return "''"
    }

    let sqlStr = ''
    switch (type) {
      case 'select':
        sqlStr = `SELECT ${colNames.map(n => '\`' + n + '\`').join(', ')}\nFROM \`${db}\`.\`${table}\`\nWHERE 1 = 1\nLIMIT 100;`
        break
      case 'insert':
        sqlStr = `INSERT INTO \`${db}\`.\`${table}\` (${colNames.map(n => '\`' + n + '\`').join(', ')})\nVALUES (${cols.map(c => getPlaceholder(c)).join(', ')});`
        break
      case 'update':
        sqlStr = `UPDATE \`${db}\`.\`${table}\`\nSET ${colNames.map(n => '\`' + n + '\` = ' + getPlaceholder(cols.find((c: any) => c.field === n))).join(', ')}\nWHERE \`${priCol}\` = '';`
        break
      case 'delete':
        sqlStr = `DELETE FROM \`${db}\`.\`${table}\`\nWHERE \`${priCol}\` = '';`
        break
    }
    addQueryTab()
    if (currentTab.value) currentTab.value.content = sqlStr
  } catch (e: any) {
    ElMessage.error('获取列信息失败: ' + (e.message || ''))
  }
}

const ctxViewDDL = () => {
  const query = `SHOW CREATE TABLE \`${ctxMenu.dbName}\`.\`${ctxMenu.tableName}\``
  addQueryTab()
  if (currentTab.value) currentTab.value.content = query
  currentDatabase.value = ctxMenu.dbName
  closeCtxMenu()
  nextTick(() => executeSQL())
}

const renameTableDialog = reactive({ visible: false, newName: '', loading: false })

const ctxRenameTable = () => {
  renameTableDialog.newName = ctxMenu.tableName
  renameTableDialog.loading = false
  renameTableDialog.visible = true
  closeCtxMenu()
}

const confirmRenameTable = async () => {
  const newName = renameTableDialog.newName.trim()
  if (!newName) return ElMessage.warning('请输入新表名')
  renameTableDialog.loading = true
  try {
    await executeMiddleware(props.middleware.id, {
      command: `RENAME TABLE \`${ctxMenu.dbName}\`.\`${ctxMenu.tableName}\` TO \`${ctxMenu.dbName}\`.\`${newName}\``,
      database: ctxMenu.dbName
    })
    ElMessage.success('重命名成功')
    renameTableDialog.visible = false
    loadDatabases()
  } catch (e: any) {
    ElMessage.error(e.message || '重命名失败')
  } finally {
    renameTableDialog.loading = false
  }
}

const ctxCopyName = () => {
  let name = ''
  if (ctxMenu.nodeType === 'table') name = ctxMenu.tableName
  else if (ctxMenu.nodeType === 'database') name = ctxMenu.dbName
  else if (ctxMenu.nodeType === 'column') name = ctxMenu.colName
  navigator.clipboard.writeText(name).then(() => ElMessage.success('已复制'))
  closeCtxMenu()
}

const ctxDropTable = () => {
  closeCtxMenu()
  ElMessageBox.confirm(
    `确定要删除表 ${ctxMenu.dbName}.${ctxMenu.tableName} 吗？此操作不可恢复！`,
    '删除表',
    { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
  ).then(async () => {
    try {
      await executeMiddleware(props.middleware.id, {
        command: `DROP TABLE \`${ctxMenu.dbName}\`.\`${ctxMenu.tableName}\``,
        database: ctxMenu.dbName
      })
      ElMessage.success('删除成功')
      loadDatabases()
    } catch (e: any) {
      ElMessage.error(e.message || '删除失败')
    }
  }).catch(() => {})
}

const ctxTruncateTable = () => {
  closeCtxMenu()
  ElMessageBox.confirm(
    `确定要清空表 ${ctxMenu.dbName}.${ctxMenu.tableName} 的所有数据吗？此操作不可恢复！`,
    '清空表',
    { confirmButtonText: '清空', cancelButtonText: '取消', type: 'warning' }
  ).then(async () => {
    try {
      await executeMiddleware(props.middleware.id, {
        command: `TRUNCATE TABLE \`${ctxMenu.dbName}\`.\`${ctxMenu.tableName}\``,
        database: ctxMenu.dbName
      })
      ElMessage.success('清空成功')
    } catch (e: any) {
      ElMessage.error(e.message || '清空失败')
    }
  }).catch(() => {})
}

const ctxCopyBackupTable = () => {
  closeCtxMenu()
  const db = ctxMenu.dbName
  const table = ctxMenu.tableName
  const backupName = `${table}_bak_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}`
  ElMessageBox.prompt('请输入备份表名', '复制备份表', {
    confirmButtonText: '创建',
    cancelButtonText: '取消',
    inputValue: backupName,
    inputPattern: /^[a-zA-Z_]\w*$/,
    inputErrorMessage: '表名格式不正确'
  }).then(async ({ value }) => {
    try {
      executing.value = true
      await executeMiddleware(props.middleware.id, {
        command: `CREATE TABLE \`${db}\`.\`${value}\` LIKE \`${db}\`.\`${table}\``,
        database: db
      })
      await executeMiddleware(props.middleware.id, {
        command: `INSERT INTO \`${db}\`.\`${value}\` SELECT * FROM \`${db}\`.\`${table}\``,
        database: db
      })
      ElMessage.success(`备份表 ${value} 创建成功`)
      loadDatabases()
    } catch (e: any) {
      ElMessage.error(e.message || '备份失败')
    } finally {
      executing.value = false
    }
  }).catch(() => {})
}

const ctxExportTable = async (mode: 'all' | 'structure') => {
  closeCtxMenu()
  const db = ctxMenu.dbName
  const table = ctxMenu.tableName
  executing.value = true
  try {
    const ddlRes: any = await executeMiddleware(props.middleware.id, {
      command: `SHOW CREATE TABLE \`${db}\`.\`${table}\``,
      database: db
    })
    let ddl = ''
    if (ddlRes?.rows?.length) {
      ddl = ddlRes.rows[0][1] || ddlRes.rows[0][0] || ''
    }
    let content = `-- 导出表: ${db}.${table}\n-- 时间: ${new Date().toLocaleString()}\n\n`
    content += ddl + ';\n'

    if (mode === 'all') {
      const dataRes: any = await executeMiddleware(props.middleware.id, {
        command: `SELECT * FROM \`${db}\`.\`${table}\``,
        database: db
      })
      if (dataRes?.rows?.length && dataRes?.columns?.length) {
        content += `\n-- 数据: ${dataRes.rows.length} 行\n`
        for (const row of dataRes.rows) {
          const vals = row.map((v: any) => {
            if (v === null || v === undefined) return 'NULL'
            if (typeof v === 'number') return String(v)
            return "'" + String(v).replace(/'/g, "\\'") + "'"
          })
          content += `INSERT INTO \`${db}\`.\`${table}\` (${dataRes.columns.map((c: string) => '\`' + c + '\`').join(', ')}) VALUES (${vals.join(', ')});\n`
        }
      }
    }

    const blob = new Blob([content], { type: 'text/sql;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${db}_${table}_${mode === 'all' ? 'full' : 'structure'}.sql`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch (e: any) {
    ElMessage.error(e.message || '导出失败')
  } finally {
    executing.value = false
  }
}

// --- Core state ---
const currentDatabase = ref('')
const databases = ref<string[]>([])
const treeData = ref<any[]>([])
const treeLoading = ref(false)
const treeRef = ref<any>(null)
const executing = ref(false)
const queryLimit = ref(200)

// Pagination
const currentPage = ref(1)
const pageSize = ref(100)

// Table height
const resultContentRef = ref<HTMLElement>()
const tableMaxHeight = ref(400)
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  document.addEventListener('click', closeCtxMenu, true)
  document.addEventListener('contextmenu', closeCtxMenu, true)
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      tableMaxHeight.value = Math.max(entry.contentRect.height - 4, 100)
    }
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('click', closeCtxMenu, true)
  document.removeEventListener('contextmenu', closeCtxMenu, true)
  resizeObserver?.disconnect()
})

watch(resultContentRef, (el) => {
  resizeObserver?.disconnect()
  if (el) {
    resizeObserver?.observe(el)
    tableMaxHeight.value = Math.max(el.clientHeight - 4, 100)
  }
})

// Watch tab switch to reset page
watch(activeQueryTab, () => { currentPage.value = currentTab.value?.resultPage || 1 })
watch(currentPage, (val) => { if (currentTab.value) currentTab.value.resultPage = val })

// History
interface QueryHistory { sql: string; time: string }
const queryHistory = ref<QueryHistory[]>([])
const HISTORY_KEY = 'sql_console_history'

const treeProps = { label: 'label', children: 'children', isLeaf: 'isLeaf' }
const editorExtensions = computed(() => [sql({ dialect: MySQL }), oneDark])

// All result rows for current tab
const allResultRows = computed(() => {
  const result = currentTabResult.value
  if (!result?.columns || !result?.rows) return []
  return result.rows.map((row: any[]) => {
    const obj: Record<string, any> = {}
    result.columns.forEach((col: string, i: number) => { obj[col] = row[i] })
    return obj
  })
})

const paginatedRows = computed(() => {
  if (allResultRows.value.length <= pageSize.value) return allResultRows.value
  const start = (currentPage.value - 1) * pageSize.value
  return allResultRows.value.slice(start, start + pageSize.value)
})

// Create DB dialog
const createDbDialog = reactive({ visible: false, name: '', loading: false })

const showCreateDbDialog = () => {
  createDbDialog.name = ''
  createDbDialog.loading = false
  createDbDialog.visible = true
}

const confirmCreateDb = async () => {
  if (!createDbDialog.name.trim()) return ElMessage.warning('请输入数据库名')
  createDbDialog.loading = true
  try {
    await createDatabase(props.middleware.id, createDbDialog.name.trim())
    ElMessage.success('创建成功')
    createDbDialog.visible = false
    loadDatabases()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    createDbDialog.loading = false
  }
}

// Load history
const loadHistory = () => {
  try {
    const data = localStorage.getItem(HISTORY_KEY)
    if (data) queryHistory.value = JSON.parse(data)
  } catch {}
}

const saveHistory = (sqlStr: string) => {
  const trimmed = sqlStr.trim()
  if (!trimmed) return
  queryHistory.value.unshift({ sql: trimmed, time: new Date().toLocaleString() })
  if (queryHistory.value.length > 50) queryHistory.value = queryHistory.value.slice(0, 50)
  try { localStorage.setItem(HISTORY_KEY, JSON.stringify(queryHistory.value)) } catch {}
}

const applyHistory = (sqlStr: string) => {
  if (currentTab.value) currentTab.value.content = sqlStr
}

// Load databases
const loadDatabases = async () => {
  if (!props.middleware?.id) return
  treeLoading.value = true
  try {
    const res: any = await getMiddlewareDatabases(props.middleware.id)
    databases.value = res || []
    treeData.value = databases.value.map(db => ({
      id: `db_${db}`, label: db, type: 'database', dbName: db, isLeaf: false
    }))
    if (props.middleware.databaseName && databases.value.includes(props.middleware.databaseName)) {
      currentDatabase.value = props.middleware.databaseName
    } else if (databases.value.length > 0 && !currentDatabase.value) {
      currentDatabase.value = databases.value[0]!
    }
  } catch (e: any) {
    ElMessage.error('获取数据库列表失败: ' + (e.message || ''))
  } finally {
    treeLoading.value = false
  }
}

const loadTreeNode = async (node: any, resolve: (data: any[]) => void) => {
  if (node.level === 0) { resolve(treeData.value); return }
  const data = node.data
  if (data.type === 'database') {
    try {
      const res: any = await getMiddlewareTables(props.middleware.id, data.dbName)
      resolve((res || []).map((t: string) => ({
        id: `table_${data.dbName}_${t}`, label: t, type: 'table',
        dbName: data.dbName, tableName: t, isLeaf: false
      })))
    } catch { resolve([]) }
  } else if (data.type === 'table') {
    try {
      const res: any = await getMiddlewareColumns(props.middleware.id, data.dbName, data.tableName)
      resolve((res || []).map((c: any) => ({
        id: `col_${data.dbName}_${data.tableName}_${c.field}`,
        label: c.field, type: 'column', colType: c.type, isLeaf: true
      })))
    } catch { resolve([]) }
  } else { resolve([]) }
}

const handleDatabaseChange = (db: string) => { currentDatabase.value = db }
const handleTreeNodeClick = (data: any) => { if (data.type === 'database') currentDatabase.value = data.dbName }
const handleTreeNodeDblClick = (data: any) => {
  if (data.type === 'table') {
    const query = `SELECT * FROM \`${data.dbName}\`.\`${data.tableName}\` LIMIT 100`
    if (currentTab.value) currentTab.value.content = query
    currentDatabase.value = data.dbName
    nextTick(() => executeSQL())
  }
}

// Execute SQL
// 拆分多条 SQL 语句（正确处理字符串、注释中的分号）
const splitSQL = (text: string): string[] => {
  const stmts: string[] = []
  let current = ''
  let i = 0
  while (i < text.length) {
    const ch = text[i]
    // 单行注释
    if (ch === '-' && text[i + 1] === '-') {
      const end = text.indexOf('\n', i)
      if (end === -1) { current += text.slice(i); break }
      current += text.slice(i, end + 1)
      i = end + 1
      continue
    }
    // 多行注释
    if (ch === '/' && text[i + 1] === '*') {
      const end = text.indexOf('*/', i + 2)
      if (end === -1) { current += text.slice(i); break }
      current += text.slice(i, end + 2)
      i = end + 2
      continue
    }
    // 引号字符串（单引号、双引号、反引号）
    if (ch === "'" || ch === '"' || ch === '`') {
      let j = i + 1
      while (j < text.length) {
        if (text[j] === '\\') { j += 2; continue }
        if (text[j] === ch) { j++; break }
        j++
      }
      current += text.slice(i, j)
      i = j
      continue
    }
    // 分号 = 语句分隔符
    if (ch === ';') {
      const trimmed = current.trim()
      if (trimmed) stmts.push(trimmed)
      current = ''
      i++
      continue
    }
    current += ch
    i++
  }
  const trimmed = current.trim()
  if (trimmed) stmts.push(trimmed)
  return stmts
}

const executeSQL = async () => {
  const tab = currentTab.value
  if (!tab) return
  const raw = tab.content.trim()
  if (!raw) return ElMessage.warning('请输入 SQL 语句')

  const stmts = splitSQL(raw)
  if (!stmts.length) return ElMessage.warning('请输入 SQL 语句')

  executing.value = true
  let lastResult: any = null
  let totalAffected = 0
  let totalDuration = 0
  let errorOccurred = false

  try {
    for (let idx = 0; idx < stmts.length; idx++) {
      let stmt = stmts[idx]
      // 最后一条 SELECT 自动加 LIMIT
      if (queryLimit.value > 0 && /^SELECT\b/i.test(stmt) && !/\bLIMIT\b/i.test(stmt)) {
        stmt = stmt + ` LIMIT ${queryLimit.value}`
      }
      try {
        const res: any = await executeMiddleware(props.middleware.id, {
          command: stmt, database: currentDatabase.value
        })
        totalDuration += res?.duration || 0
        if (res?.columns?.length) {
          // 查询语句，保留结果用于展示
          lastResult = res
        } else {
          totalAffected += res?.affectedRows || 0
          if (!lastResult) lastResult = res
        }
      } catch (e: any) {
        ElMessage.error(`第 ${idx + 1} 条语句执行失败: ${e.message || '未知错误'}`)
        errorOccurred = true
        break
      }
    }

    saveHistory(raw)
    if (lastResult) {
      lastResult.duration = totalDuration
      if (!lastResult.columns && totalAffected > 0) {
        lastResult.affectedRows = totalAffected
      }
    } else {
      lastResult = { message: `执行完成，共 ${stmts.length} 条语句`, affectedRows: totalAffected, duration: totalDuration }
    }
    tab.result = lastResult
    tab.resultPage = 1
    currentPage.value = 1
    saveQueryTabs()
    if (!errorOccurred && stmts.length > 1) {
      ElMessage.success(`成功执行 ${stmts.length} 条语句`)
    }
  } finally {
    executing.value = false
  }
}

const formatSQL = () => {
  if (!currentTab.value?.content.trim()) return
  try {
    currentTab.value.content = formatSqlFn(currentTab.value.content, { language: 'mysql' })
  } catch { ElMessage.warning('格式化失败') }
}

const handleEditorKeydown = (e: KeyboardEvent) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); executeSQL() }
}

const exportCSV = () => {
  const result = currentTabResult.value
  if (!result?.columns?.length) return
  const header = result.columns.join(',')
  const rows = (result.rows || []).map((row: any[]) =>
    row.map((cell: any) => {
      const str = String(cell ?? '')
      return str.includes(',') || str.includes('"') || str.includes('\n')
        ? '"' + str.replace(/"/g, '""') + '"' : str
    }).join(',')
  )
  const csv = [header, ...rows].join('\n')
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `query_result_${Date.now()}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

watch(() => props.visible, (val) => {
  if (val && props.middleware) {
    loadHistory()
    loadQueryTabs()
    loadDatabases()
    currentPage.value = 1
  }
})
</script>
<!-- SPLIT_MARKER_STYLE -->

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
.sidebar-tree {
  flex: 1;
  overflow: auto;
  padding: 8px;
}
.tree-node {
  display: flex;
  align-items: center;
  font-size: 13px;
  min-width: 0;
  flex: 1;
}
.tree-node-label {
  margin-left: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.column-type {
  margin-left: 6px;
  color: #909399;
  font-size: 11px;
}
/* Query Tabs */
.query-tabs-bar {
  display: flex;
  align-items: center;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  padding: 0 4px;
  height: 34px;
  flex-shrink: 0;
  overflow-x: auto;
  gap: 2px;
}
.query-tab {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
  border-radius: 4px 4px 0 0;
  background: #e8eaed;
  border: 1px solid #dcdfe6;
  border-bottom: none;
  white-space: nowrap;
  max-width: 160px;
  transition: all 0.15s;
  position: relative;
  top: 1px;
}
.query-tab.active {
  background: #fff;
  border-color: #e4e7ed;
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
  width: 80px;
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
  width: 24px;
  height: 24px;
  cursor: pointer;
  border-radius: 4px;
  color: #606266;
  font-size: 14px;
  transition: all 0.15s;
}
.query-tab-add:hover {
  color: #409eff;
  background: #ecf5ff;
}
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}
.editor-area {
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}
.result-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.result-content {
  flex: 1;
  overflow: auto;
  min-height: 0;
}
.result-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #909399;
  font-size: 14px;
}
.result-message {
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #67c23a;
  font-size: 14px;
}
.raw-result {
  background: #f5f7fa;
  padding: 12px;
  margin: 8px;
  border-radius: 4px;
  max-height: 100%;
  overflow: auto;
  font-size: 13px;
}
.status-bar {
  flex-shrink: 0;
  padding: 4px 12px;
  background: #f5f7fa;
  border-top: 1px solid #e4e7ed;
  font-size: 12px;
  color: #606266;
  display: flex;
  align-items: center;
  gap: 16px;
}
.history-panel {
  max-height: 300px;
  overflow: auto;
}
.history-empty {
  text-align: center;
  color: #909399;
  padding: 20px;
  font-size: 13px;
}
.history-item {
  padding: 8px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.2s;
}
.history-item:hover {
  background: #f5f7fa;
}
.history-sql {
  font-size: 12px;
  color: #303133;
  font-family: monospace;
  word-break: break-all;
}
.history-time {
  font-size: 11px;
  color: #909399;
  margin-top: 4px;
}
.resize-handle {
  width: 4px;
  cursor: col-resize;
  background: transparent;
  flex-shrink: 0;
  transition: background 0.2s;
}
.resize-handle:hover { background: #409eff; }
.context-menu {
  position: fixed;
  z-index: 9999;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.12);
  padding: 4px 0;
  min-width: 160px;
}
.ctx-item {
  padding: 6px 16px;
  font-size: 13px;
  cursor: pointer;
  position: relative;
}
.ctx-item:hover { background: #ecf5ff; color: #409eff; }
.ctx-danger { color: #f56c6c; }
.ctx-danger:hover { background: #fef0f0; color: #f56c6c; }
.ctx-divider { height: 1px; background: #ebeef5; margin: 4px 0; }
.ctx-has-sub { padding-right: 24px; }
.ctx-has-sub::after { content: '▸'; position: absolute; right: 8px; }
.ctx-submenu {
  display: none;
  position: absolute;
  left: 100%;
  top: 0;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.12);
  padding: 4px 0;
  min-width: 120px;
}
.ctx-has-sub:hover > .ctx-submenu { display: block; }
</style>
