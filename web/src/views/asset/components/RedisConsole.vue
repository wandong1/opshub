<template>
  <a-drawer
    :visible="visible"
    :title="'Redis 控制台 - ' + (middleware?.name || '')"
    placement="right"
    :width="'100%'"
    :unmount-on-close="true"
    class="redis-console-drawer"
    @cancel="emit('update:visible', false)"
  >
    <template #title>
      <div class="console-header">
        <div class="header-left">
          <icon-storage style="font-size: 18px; color: #f56c6c;" />
          <span class="header-title">Redis 控制台 - {{ middleware?.name }}</span>
        </div>
        <div class="header-actions">
          <a-select v-model="currentDb" size="small" style="width: 150px" @change="handleDbChange">
            <a-option v-for="db in databases" :key="db.db" :label="`db${db.db} (${db.keys})`" :value="db.db" />
          </a-select>
          <a-button size="small" @click="refreshKeys">
            <template #icon><icon-refresh /></template>刷新
          </a-button>
          <a-button type="primary" size="small" @click="showNewKeyDialog">
            <template #icon><icon-plus /></template>新建 Key
          </a-button>
          <a-button size="small" @click="activeRightTab = 'info'">服务器信息</a-button>
        </div>
      </div>
    </template>

    <div class="console-body">
      <!-- 左侧键浏览器 -->
      <div class="sidebar">
        <div class="sidebar-search">
          <a-input v-model="searchPattern" placeholder="搜索键名 (支持*通配符)" size="small" allow-clear @keyup.enter="refreshKeys" @clear="refreshKeys">
            <template #prefix><icon-search /></template>
          </a-input>
        </div>
        <div class="key-list" ref="keyListRef" @scroll="onKeyListScroll">
          <a-spin :loading="keysLoading" style="width: 100%;">
            <div :style="{ height: totalHeight + 'px', position: 'relative' }">
              <div :style="{ position: 'absolute', top: offsetTop + 'px', left: 0, right: 0 }">
                <div
                  v-for="item in visibleKeys"
                  :key="item.key"
                  class="key-item"
                  :class="{ active: selectedKey === item.key }"
                  @click="handleSelectKey(item.key)"
                  @contextmenu.prevent="handleKeyContextMenu($event, item)"
                >
                  <span class="key-type-badge" :class="'type-' + item.type">{{ getTypeLabel(item.type) }}</span>
                  <span class="key-name" :title="item.key">{{ item.key }}</span>
                  <span class="key-ttl" v-if="item.ttl >= 0">{{ formatTTL(item.ttl) }}</span>
                  <span class="key-ttl-forever" v-else>-</span>
                </div>
              </div>
            </div>
            <div v-if="!keyList.length && !keysLoading" class="key-empty">暂无数据</div>
            <div v-if="scanCursor !== 0" class="load-more">
              <a-button type="text" status="normal" size="small" @click="loadMoreKeys" :loading="keysLoading">加载更多</a-button>
            </div>
          </a-spin>
        </div>
      </div>

      <!-- 右侧主区域 -->
      <div class="main-area">
        <a-tabs v-model:active-key="activeRightTab" type="card-gutter">
          <!-- 键值详情 Tab -->
          <a-tab-pane title="键值详情" key="detail">
            <div v-if="!selectedKey" class="empty-tip">请从左侧选择一个键</div>
            <a-spin v-else-if="detailLoading" :loading="true" style="height: 200px; width: 100%; display: flex; align-items: center; justify-content: center;" />
            <div v-else-if="keyDetail" class="key-detail">
              <!-- 公共头部 -->
              <div class="detail-header">
                <a-tag :color="getTypeTagColor(keyDetail.type)" size="small">{{ keyDetail.type }}</a-tag>
                <span class="detail-key-name">{{ keyDetail.key }}</span>
                <a-tag v-if="keyDetail.ttl >= 0" size="small" color="orangered">TTL: {{ keyDetail.ttl }}s</a-tag>
                <a-tag v-else size="small" color="gray">永不过期</a-tag>
                <a-tag v-if="keyDetail.size" size="small">{{ formatBytes(keyDetail.size) }}</a-tag>
                <div style="flex:1"></div>
                <a-button size="small" @click="loadKeyDetail(selectedKey)"><template #icon><icon-refresh /></template></a-button>
                <a-button size="small" @click="showTTLDialog(keyDetail.key, keyDetail.ttl)">设置 TTL</a-button>
                <a-button size="small" @click="showRenameDialog(keyDetail.key)">重命名</a-button>
                <a-button size="small" status="danger" @click="handleDeleteKey(keyDetail.key)">删除</a-button>
              </div>

              <!-- String 类型 -->
              <div v-if="keyDetail.type === 'string'" class="detail-body">
                <a-textarea v-model="stringValue" :auto-size="{ minRows: 12 }" />
                <div class="detail-actions">
                  <a-button type="primary" size="small" @click="saveStringValue">保存</a-button>
                </div>
              </div>

              <!-- Hash 类型 -->
              <div v-if="keyDetail.type === 'hash'" class="detail-body">
                <div class="detail-actions" style="margin-bottom:8px;">
                  <a-button type="primary" size="small" @click="showHashAddDialog">添加字段</a-button>
                </div>
                <a-table :data="hashRows" size="small" :bordered="{ cell: true }" stripe :scroll="{ y: 400 }" :pagination="false">
                  <template #columns>
                    <a-table-column title="字段" data-index="field" :width="200" ellipsis tooltip />
                    <a-table-column title="值" data-index="value" :width="300" ellipsis tooltip />
                    <a-table-column title="操作" :width="120" fixed="right">
                      <template #cell="{ record }">
                        <a-button type="text" size="small" @click="editHashField(record)">编辑</a-button>
                        <a-button type="text" status="danger" size="small" @click="deleteHashField(record.field)">删除</a-button>
                      </template>
                    </a-table-column>
                  </template>
                </a-table>
              </div>

              <!-- List 类型 -->
              <div v-if="keyDetail.type === 'list'" class="detail-body">
                <div class="detail-actions" style="margin-bottom:8px;">
                  <a-button type="primary" size="small" @click="showListAddDialog('LPUSH')">LPUSH</a-button>
                  <a-button type="primary" size="small" @click="showListAddDialog('RPUSH')">RPUSH</a-button>
                  <a-button size="small" @click="listPop('LPOP')">LPOP</a-button>
                  <a-button size="small" @click="listPop('RPOP')">RPOP</a-button>
                </div>
                <a-table :data="listRows" size="small" :bordered="{ cell: true }" stripe :scroll="{ y: 400 }" :pagination="false">
                  <template #columns>
                    <a-table-column title="索引" data-index="index" :width="80" />
                    <a-table-column title="值" data-index="value" :width="400" ellipsis tooltip />
                    <a-table-column title="操作" :width="80" fixed="right">
                      <template #cell="{ record }">
                        <a-button type="text" size="small" @click="editListItem(record)">编辑</a-button>
                      </template>
                    </a-table-column>
                  </template>
                </a-table>
              </div>

              <!-- Set 类型 -->
              <div v-if="keyDetail.type === 'set'" class="detail-body">
                <div class="detail-actions" style="margin-bottom:8px;">
                  <a-button type="primary" size="small" @click="showSetAddDialog">添加成员</a-button>
                </div>
                <a-table :data="setRows" size="small" :bordered="{ cell: true }" stripe :scroll="{ y: 400 }" :pagination="false">
                  <template #columns>
                    <a-table-column title="#" :width="60">
                      <template #cell="{ rowIndex }">{{ rowIndex + 1 }}</template>
                    </a-table-column>
                    <a-table-column title="成员" data-index="value" :width="400" ellipsis tooltip />
                    <a-table-column title="操作" :width="80" fixed="right">
                      <template #cell="{ record }">
                        <a-button type="text" status="danger" size="small" @click="deleteSetMember(record.value)">删除</a-button>
                      </template>
                    </a-table-column>
                  </template>
                </a-table>
              </div>

              <!-- ZSet 类型 -->
              <div v-if="keyDetail.type === 'zset'" class="detail-body">
                <div class="detail-actions" style="margin-bottom:8px;">
                  <a-button type="primary" size="small" @click="showZsetAddDialog">添加成员</a-button>
                </div>
                <a-table :data="zsetRows" size="small" :bordered="{ cell: true }" stripe :scroll="{ y: 400 }" :pagination="false">
                  <template #columns>
                    <a-table-column title="#" :width="60">
                      <template #cell="{ rowIndex }">{{ rowIndex + 1 }}</template>
                    </a-table-column>
                    <a-table-column title="成员" data-index="member" :width="300" ellipsis tooltip />
                    <a-table-column title="分数" data-index="score" :width="150" :sortable="{ sortDirections: ['ascend', 'descend'] }" />
                    <a-table-column title="操作" :width="80" fixed="right">
                      <template #cell="{ record }">
                        <a-button type="text" status="danger" size="small" @click="deleteZsetMember(record.member)">删除</a-button>
                      </template>
                    </a-table-column>
                  </template>
                </a-table>
              </div>
            </div>
          </a-tab-pane>

          <!-- 命令行 Tab -->
          <a-tab-pane title="命令行" key="cli">
            <div class="cli-container">
              <div class="cli-output" ref="cliOutputRef">
                <div v-for="(item, idx) in cliHistory" :key="idx" class="cli-line">
                  <div class="cli-command">&gt; {{ item.command }}</div>
                  <pre class="cli-result" :class="{ 'cli-error': item.error }">{{ item.result }}</pre>
                </div>
              </div>
              <div class="cli-input-row">
                <span class="cli-prompt">{{ middleware?.host }}:{{ middleware?.port }}&gt;</span>
                <a-input
                  v-model="cliCommand"
                  placeholder="输入 Redis 命令..."
                  size="small"
                  @keyup.enter="executeCli"
                  @keyup.up="cliHistoryUp"
                  @keyup.down="cliHistoryDown"
                  ref="cliInputRef"
                />
              </div>
            </div>
          </a-tab-pane>

          <!-- 服务器信息 Tab -->
          <a-tab-pane title="服务器信息" key="info">
            <a-spin :loading="infoLoading" class="info-container" style="width: 100%; display: block;">
              <div class="info-actions">
                <a-button size="small" @click="loadServerInfo"><template #icon><icon-refresh /></template> 刷新</a-button>
              </div>
              <a-collapse v-model:active-key="infoActiveNames">
                <a-collapse-item v-for="(section, name) in serverInfo" :key="String(name)" :header="String(name)">
                  <a-descriptions :column="2" bordered size="small">
                    <a-descriptions-item v-for="(val, key) in section" :key="key" :label="String(key)">{{ val }}</a-descriptions-item>
                  </a-descriptions>
                </a-collapse-item>
              </a-collapse>
              <div v-if="!Object.keys(serverInfo).length && !infoLoading" class="empty-tip">点击刷新加载服务器信息</div>
            </a-spin>
          </a-tab-pane>
        </a-tabs>
      </div>
    </div>

    <!-- 右键菜单 -->
    <div v-if="contextMenu.visible" class="context-menu" :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }" @click="contextMenu.visible = false">
      <div class="context-menu-item" @click="handleCopyKeyName">复制键名</div>
      <div class="context-menu-item" @click="showTTLDialog(contextMenu.key, -1)">设置 TTL</div>
      <div class="context-menu-item" @click="showRenameDialog(contextMenu.key)">重命名</div>
      <div class="context-menu-item danger" @click="handleDeleteKey(contextMenu.key)">删除</div>
    </div>

    <!-- 新建 Key 弹窗 -->
    <a-modal v-model:visible="newKeyVisible" title="新建 Key" :width="500" unmount-on-close :mask-closable="false">
      <a-form :model="newKeyForm" auto-label-width layout="horizontal">
        <a-form-item label="键名" field="key"><a-input v-model="newKeyForm.key" /></a-form-item>
        <a-form-item label="类型" field="type">
          <a-select v-model="newKeyForm.type" style="width:100%">
            <a-option label="String" value="string" />
            <a-option label="Hash" value="hash" />
            <a-option label="List" value="list" />
            <a-option label="Set" value="set" />
            <a-option label="ZSet" value="zset" />
          </a-select>
        </a-form-item>
        <a-form-item label="值" field="value">
          <a-textarea v-model="newKeyForm.value" :auto-size="{ minRows: 4 }" placeholder="String: 直接输入值; 其他类型暂设为空" />
        </a-form-item>
        <a-form-item label="TTL(秒)" field="ttl">
          <a-input-number v-model="newKeyForm.ttl" :min="-1" style="width:100%" />
          <div style="font-size:12px;color:#86909c;">-1 表示永不过期</div>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="newKeyVisible = false">取消</a-button>
        <a-button type="primary" @click="createNewKey">确定</a-button>
      </template>
    </a-modal>

    <!-- 通用输入弹窗 -->
    <a-modal v-model:visible="inputDialog.visible" :title="inputDialog.title" :width="450" unmount-on-close :mask-closable="false">
      <a-form auto-label-width layout="horizontal">
        <a-form-item v-if="inputDialog.showField" :label="inputDialog.fieldLabel || '字段'">
          <a-input v-model="inputDialog.field" />
        </a-form-item>
        <a-form-item v-if="inputDialog.showValue" :label="inputDialog.valueLabel || '值'">
          <a-textarea v-model="inputDialog.value" :auto-size="{ minRows: 3 }" />
        </a-form-item>
        <a-form-item v-if="inputDialog.showScore" label="分数">
          <a-input-number v-model="inputDialog.score" style="width:100%" />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="inputDialog.visible = false">取消</a-button>
        <a-button type="primary" @click="inputDialog.onConfirm?.()">确定</a-button>
      </template>
    </a-modal>

    <!-- TTL 弹窗 -->
    <a-modal v-model:visible="ttlDialog.visible" title="设置 TTL" :width="400" unmount-on-close :mask-closable="false">
      <a-form auto-label-width layout="horizontal">
        <a-form-item label="键名"><a-input :model-value="ttlDialog.key" disabled /></a-form-item>
        <a-form-item label="TTL(秒)">
          <a-input-number v-model="ttlDialog.ttl" :min="-1" style="width:100%" />
          <div style="font-size:12px;color:#86909c;">-1 表示移除过期时间（永不过期）</div>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="ttlDialog.visible = false">取消</a-button>
        <a-button type="primary" @click="confirmSetTTL">确定</a-button>
      </template>
    </a-modal>

    <!-- 重命名弹窗 -->
    <a-modal v-model:visible="renameDialog.visible" title="重命名" :width="400" unmount-on-close :mask-closable="false">
      <a-form auto-label-width layout="horizontal">
        <a-form-item label="原键名"><a-input :model-value="renameDialog.oldKey" disabled /></a-form-item>
        <a-form-item label="新键名"><a-input v-model="renameDialog.newKey" /></a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="renameDialog.visible = false">取消</a-button>
        <a-button type="primary" @click="confirmRename">确定</a-button>
      </template>
    </a-modal>
  </a-drawer>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, reactive } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { IconStorage, IconRefresh, IconPlus, IconSearch } from '@arco-design/web-vue/es/icon'
import {
  getRedisDatabases, scanRedisKeys, getRedisKeyDetail, setRedisKey,
  redisKeyAction, deleteRedisKeys, setRedisKeyTTL, renameRedisKey,
  getRedisInfo, executeMiddleware
} from '@/api/middleware'

interface Props {
  visible: boolean
  middleware: any
}

const props = defineProps<Props>()
const emit = defineEmits(['update:visible'])

// DB 选择
const currentDb = ref(0)
const databases = ref<{ db: number; keys: number }[]>([])

// 键列表
const keyList = ref<{ key: string; type: string; ttl: number }[]>([])
const keysLoading = ref(false)
const searchPattern = ref('*')
const scanCursor = ref(0)
const selectedKey = ref('')

// 虚拟滚动
const keyListRef = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const KEY_ITEM_HEIGHT = 33
const OVERSCAN = 10

const totalHeight = computed(() => keyList.value.length * KEY_ITEM_HEIGHT)

const visibleKeys = computed(() => {
  const list = keyList.value
  if (!list.length) return []
  const containerH = keyListRef.value?.clientHeight || 500
  const startIdx = Math.max(0, Math.floor(scrollTop.value / KEY_ITEM_HEIGHT) - OVERSCAN)
  const endIdx = Math.min(list.length, Math.ceil((scrollTop.value + containerH) / KEY_ITEM_HEIGHT) + OVERSCAN)
  return list.slice(startIdx, endIdx)
})

const offsetTop = computed(() => {
  const containerH = keyListRef.value?.clientHeight || 500
  const startIdx = Math.max(0, Math.floor(scrollTop.value / KEY_ITEM_HEIGHT) - OVERSCAN)
  return startIdx * KEY_ITEM_HEIGHT
})

const onKeyListScroll = () => {
  if (keyListRef.value) {
    scrollTop.value = keyListRef.value.scrollTop
    // 滚动到底部附近时自动加载更多
    const el = keyListRef.value
    if (scanCursor.value !== 0 && !keysLoading.value && el.scrollTop + el.clientHeight >= el.scrollHeight - 100) {
      loadMoreKeys()
    }
  }
}

// 键详情
const keyDetail = ref<any>(null)
const detailLoading = ref(false)
const stringValue = ref('')

// 命令行
const cliCommand = ref('')
const cliHistory = ref<{ command: string; result: string; error?: boolean }[]>([])
const cliCommandHistory = ref<string[]>([])
const cliHistoryIndex = ref(-1)
const cliOutputRef = ref<HTMLElement>()
const cliInputRef = ref()

// 服务器信息
const serverInfo = ref<Record<string, Record<string, string>>>({})
const infoLoading = ref(false)
const infoActiveNames = ref<string[]>(['Server', 'Clients', 'Memory', 'Stats', 'Keyspace'])

// 右键菜单
const contextMenu = reactive({ visible: false, x: 0, y: 0, key: '' })

// 右侧 Tab
const activeRightTab = ref('detail')

// 新建 Key
const newKeyVisible = ref(false)
const newKeyForm = reactive({ key: '', type: 'string', value: '', ttl: -1 })

// 通用输入弹窗
const inputDialog = reactive({
  visible: false, title: '', field: '', value: '', score: 0,
  showField: false, showValue: true, showScore: false,
  fieldLabel: '字段', valueLabel: '值',
  onConfirm: null as (() => void) | null
})

// TTL 弹窗
const ttlDialog = reactive({ visible: false, key: '', ttl: -1 })

// 重命名弹窗
const renameDialog = reactive({ visible: false, oldKey: '', newKey: '' })

// ===== Computed =====
const hashRows = computed(() => {
  if (!keyDetail.value || keyDetail.value.type !== 'hash') return []
  const val = keyDetail.value.value || {}
  return Object.entries(val).map(([field, value]) => ({ field, value }))
})

const listRows = computed(() => {
  if (!keyDetail.value || keyDetail.value.type !== 'list') return []
  return (keyDetail.value.value || []).map((v: string, i: number) => ({ index: i, value: v }))
})

const setRows = computed(() => {
  if (!keyDetail.value || keyDetail.value.type !== 'set') return []
  return (keyDetail.value.value || []).map((v: string) => ({ value: v }))
})

const zsetRows = computed(() => {
  if (!keyDetail.value || keyDetail.value.type !== 'zset') return []
  return keyDetail.value.value || []
})

// ===== Methods =====
const getTypeLabel = (type: string) => {
  const map: Record<string, string> = { string: 'S', hash: 'H', list: 'L', set: 'SET', zset: 'ZS' }
  return map[type] || type
}

const getTypeTagColor = (type: string) => {
  const map: Record<string, string> = { string: 'blue', hash: 'green', list: 'orangered', set: 'red', zset: 'gray' }
  return map[type] || 'blue'
}

const formatTTL = (ttl: number) => {
  if (ttl < 0) return ''
  if (ttl < 60) return `${ttl}s`
  if (ttl < 3600) return `${Math.floor(ttl / 60)}m`
  if (ttl < 86400) return `${Math.floor(ttl / 3600)}h`
  return `${Math.floor(ttl / 86400)}d`
}

const formatBytes = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const loadDatabases = async () => {
  if (!props.middleware?.id) return
  try {
    const res: any = await getRedisDatabases(props.middleware.id)
    databases.value = res || []
    if (databases.value.length > 0 && !databases.value.find(d => d.db === currentDb.value)) {
      currentDb.value = databases.value[0].db
    }
  } catch (e: any) {
    Message.error('获取数据库列表失败: ' + (e.message || ''))
  }
}

const refreshKeys = async () => {
  scanCursor.value = 0
  keyList.value = []
  await loadKeys()
}

const loadKeys = async () => {
  if (!props.middleware?.id) return
  keysLoading.value = true
  try {
    const res: any = await scanRedisKeys(props.middleware.id, {
      db: currentDb.value,
      cursor: scanCursor.value,
      count: 100,
      pattern: searchPattern.value || '*'
    })
    const newKeys = res?.keys || []
    if (scanCursor.value === 0) {
      keyList.value = newKeys
    } else {
      keyList.value = [...keyList.value, ...newKeys]
    }
    scanCursor.value = res?.cursor || 0
  } catch (e: any) {
    Message.error('扫描键失败: ' + (e.message || ''))
  } finally {
    keysLoading.value = false
  }
}

const loadMoreKeys = () => {
  loadKeys()
}

const handleDbChange = () => {
  selectedKey.value = ''
  keyDetail.value = null
  refreshKeys()
}

const handleSelectKey = (key: string) => {
  selectedKey.value = key
  activeRightTab.value = 'detail'
  loadKeyDetail(key)
}

const loadKeyDetail = async (key: string) => {
  if (!props.middleware?.id) return
  detailLoading.value = true
  try {
    const res: any = await getRedisKeyDetail(props.middleware.id, key, currentDb.value)
    keyDetail.value = res
    if (res.type === 'string') {
      stringValue.value = res.value || ''
    }
  } catch (e: any) {
    Message.error('获取键详情失败: ' + (e.message || ''))
  } finally {
    detailLoading.value = false
  }
}

const saveStringValue = async () => {
  if (!props.middleware?.id || !keyDetail.value) return
  try {
    await setRedisKey(props.middleware.id, {
      key: keyDetail.value.key,
      type: 'string',
      value: stringValue.value,
      ttl: keyDetail.value.ttl
    }, currentDb.value)
    Message.success('保存成功')
  } catch (e: any) {
    Message.error('保存失败: ' + (e.message || ''))
  }
}

// Hash 操作
const showHashAddDialog = () => {
  inputDialog.title = '添加 Hash 字段'
  inputDialog.field = ''
  inputDialog.value = ''
  inputDialog.showField = true
  inputDialog.showValue = true
  inputDialog.showScore = false
  inputDialog.fieldLabel = '字段名'
  inputDialog.valueLabel = '字段值'
  inputDialog.onConfirm = async () => {
    try {
      await redisKeyAction(props.middleware.id, {
        key: keyDetail.value.key, action: 'HSET',
        field: inputDialog.field, value: inputDialog.value
      }, currentDb.value)
      inputDialog.visible = false
      loadKeyDetail(selectedKey.value)
    } catch (e: any) { Message.error(e.message || '操作失败') }
  }
  inputDialog.visible = true
}

const editHashField = (row: { field: string; value: any }) => {
  inputDialog.title = '编辑 Hash 字段'
  inputDialog.field = row.field
  inputDialog.value = String(row.value)
  inputDialog.showField = true
  inputDialog.showValue = true
  inputDialog.showScore = false
  inputDialog.fieldLabel = '字段名'
  inputDialog.valueLabel = '字段值'
  inputDialog.onConfirm = async () => {
    try {
      await redisKeyAction(props.middleware.id, {
        key: keyDetail.value.key, action: 'HSET',
        field: inputDialog.field, value: inputDialog.value
      }, currentDb.value)
      inputDialog.visible = false
      loadKeyDetail(selectedKey.value)
    } catch (e: any) { Message.error(e.message || '操作失败') }
  }
  inputDialog.visible = true
}

const deleteHashField = async (field: string) => {
  try {
    await redisKeyAction(props.middleware.id, {
      key: keyDetail.value.key, action: 'HDEL', field
    }, currentDb.value)
    loadKeyDetail(selectedKey.value)
  } catch (e: any) { Message.error(e.message || '操作失败') }
}

// List 操作
const showListAddDialog = (action: string) => {
  inputDialog.title = action === 'LPUSH' ? '左侧插入' : '右侧插入'
  inputDialog.value = ''
  inputDialog.showField = false
  inputDialog.showValue = true
  inputDialog.showScore = false
  inputDialog.valueLabel = '值'
  inputDialog.onConfirm = async () => {
    try {
      await redisKeyAction(props.middleware.id, {
        key: keyDetail.value.key, action, value: inputDialog.value
      }, currentDb.value)
      inputDialog.visible = false
      loadKeyDetail(selectedKey.value)
    } catch (e: any) { Message.error(e.message || '操作失败') }
  }
  inputDialog.visible = true
}

const listPop = async (action: string) => {
  try {
    const res = await redisKeyAction(props.middleware.id, {
      key: keyDetail.value.key, action
    }, currentDb.value)
    Message.success(`弹出值: ${res}`)
    loadKeyDetail(selectedKey.value)
  } catch (e: any) { Message.error(e.message || '操作失败') }
}

const editListItem = (row: { index: number; value: string }) => {
  inputDialog.title = `编辑索引 ${row.index}`
  inputDialog.field = String(row.index)
  inputDialog.value = row.value
  inputDialog.showField = false
  inputDialog.showValue = true
  inputDialog.showScore = false
  inputDialog.valueLabel = '值'
  inputDialog.onConfirm = async () => {
    try {
      await redisKeyAction(props.middleware.id, {
        key: keyDetail.value.key, action: 'LSET',
        field: inputDialog.field, value: inputDialog.value
      }, currentDb.value)
      inputDialog.visible = false
      loadKeyDetail(selectedKey.value)
    } catch (e: any) { Message.error(e.message || '操作失败') }
  }
  inputDialog.visible = true
}

// Set 操作
const showSetAddDialog = () => {
  inputDialog.title = '添加成员'
  inputDialog.value = ''
  inputDialog.showField = false
  inputDialog.showValue = true
  inputDialog.showScore = false
  inputDialog.valueLabel = '成员值'
  inputDialog.onConfirm = async () => {
    try {
      await redisKeyAction(props.middleware.id, {
        key: keyDetail.value.key, action: 'SADD', value: inputDialog.value
      }, currentDb.value)
      inputDialog.visible = false
      loadKeyDetail(selectedKey.value)
    } catch (e: any) { Message.error(e.message || '操作失败') }
  }
  inputDialog.visible = true
}

const deleteSetMember = async (value: string) => {
  try {
    await redisKeyAction(props.middleware.id, {
      key: keyDetail.value.key, action: 'SREM', value
    }, currentDb.value)
    loadKeyDetail(selectedKey.value)
  } catch (e: any) { Message.error(e.message || '操作失败') }
}

// ZSet 操作
const showZsetAddDialog = () => {
  inputDialog.title = '添加成员'
  inputDialog.value = ''
  inputDialog.score = 0
  inputDialog.showField = false
  inputDialog.showValue = true
  inputDialog.showScore = true
  inputDialog.valueLabel = '成员'
  inputDialog.onConfirm = async () => {
    try {
      await redisKeyAction(props.middleware.id, {
        key: keyDetail.value.key, action: 'ZADD',
        value: inputDialog.value, score: inputDialog.score
      }, currentDb.value)
      inputDialog.visible = false
      loadKeyDetail(selectedKey.value)
    } catch (e: any) { Message.error(e.message || '操作失败') }
  }
  inputDialog.visible = true
}

const deleteZsetMember = async (member: string) => {
  try {
    await redisKeyAction(props.middleware.id, {
      key: keyDetail.value.key, action: 'ZREM', value: member
    }, currentDb.value)
    loadKeyDetail(selectedKey.value)
  } catch (e: any) { Message.error(e.message || '操作失败') }
}

// 删除键
const handleDeleteKey = (key: string) => {
  Modal.warning({
    title: '提示',
    content: `确定删除键「${key}」？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await deleteRedisKeys(props.middleware.id, [key], currentDb.value)
        Message.success('删除成功')
        if (selectedKey.value === key) {
          selectedKey.value = ''
          keyDetail.value = null
        }
        refreshKeys()
      } catch (e: any) { Message.error(e.message || '删除失败') }
    }
  })
}

// TTL
const showTTLDialog = (key: string, ttl: number) => {
  ttlDialog.key = key
  ttlDialog.ttl = ttl >= 0 ? ttl : -1
  ttlDialog.visible = true
}

const confirmSetTTL = async () => {
  try {
    await setRedisKeyTTL(props.middleware.id, { key: ttlDialog.key, ttl: ttlDialog.ttl }, currentDb.value)
    Message.success('设置成功')
    ttlDialog.visible = false
    if (selectedKey.value === ttlDialog.key) loadKeyDetail(selectedKey.value)
    refreshKeys()
  } catch (e: any) { Message.error(e.message || '设置失败') }
}

// 重命名
const showRenameDialog = (key: string) => {
  renameDialog.oldKey = key
  renameDialog.newKey = key
  renameDialog.visible = true
}

const confirmRename = async () => {
  try {
    await renameRedisKey(props.middleware.id, { oldKey: renameDialog.oldKey, newKey: renameDialog.newKey }, currentDb.value)
    Message.success('重命名成功')
    renameDialog.visible = false
    if (selectedKey.value === renameDialog.oldKey) {
      selectedKey.value = renameDialog.newKey
      loadKeyDetail(renameDialog.newKey)
    }
    refreshKeys()
  } catch (e: any) { Message.error(e.message || '重命名失败') }
}

// 右键菜单
const handleKeyContextMenu = (e: MouseEvent, item: any) => {
  contextMenu.visible = true
  contextMenu.x = e.clientX
  contextMenu.y = e.clientY
  contextMenu.key = item.key
}

const handleCopyKeyName = () => {
  navigator.clipboard.writeText(contextMenu.key).then(() => {
    Message.success('已复制')
  })
}

// 新建 Key
const showNewKeyDialog = () => {
  newKeyForm.key = ''
  newKeyForm.type = 'string'
  newKeyForm.value = ''
  newKeyForm.ttl = -1
  newKeyVisible.value = true
}

const createNewKey = async () => {
  if (!newKeyForm.key) return Message.warning('请输入键名')
  try {
    let value: any = newKeyForm.value
    if (newKeyForm.type === 'hash') value = {}
    else if (newKeyForm.type === 'list' || newKeyForm.type === 'set') value = newKeyForm.value ? [newKeyForm.value] : []
    else if (newKeyForm.type === 'zset') value = newKeyForm.value ? [{ member: newKeyForm.value, score: 0 }] : []

    await setRedisKey(props.middleware.id, {
      key: newKeyForm.key, type: newKeyForm.type, value, ttl: newKeyForm.ttl
    }, currentDb.value)
    Message.success('创建成功')
    newKeyVisible.value = false
    refreshKeys()
  } catch (e: any) { Message.error(e.message || '创建失败') }
}

// 命令行
const executeCli = async () => {
  const cmd = cliCommand.value.trim()
  if (!cmd) return
  cliCommandHistory.value.push(cmd)
  cliHistoryIndex.value = cliCommandHistory.value.length
  cliCommand.value = ''

  try {
    const res: any = await executeMiddleware(props.middleware.id, {
      command: cmd,
      database: String(currentDb.value)
    })
    cliHistory.value.push({ command: cmd, result: res?.message || JSON.stringify(res?.rawResult, null, 2) || 'OK' })
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

// 服务器信息
const loadServerInfo = async () => {
  if (!props.middleware?.id) return
  infoLoading.value = true
  try {
    const res: any = await getRedisInfo(props.middleware.id, currentDb.value)
    serverInfo.value = res || {}
  } catch (e: any) {
    Message.error('获取服务器信息失败: ' + (e.message || ''))
  } finally {
    infoLoading.value = false
  }
}

// 关闭右键菜单
const handleDocClick = () => { contextMenu.visible = false }

// Watch
watch(() => props.visible, (val) => {
  if (val && props.middleware) {
    currentDb.value = 0
    selectedKey.value = ''
    keyDetail.value = null
    keyList.value = []
    cliHistory.value = []
    cliCommandHistory.value = []
    serverInfo.value = {}
    searchPattern.value = '*'
    activeRightTab.value = 'detail'
    loadDatabases()
    refreshKeys()
    loadServerInfo()
    document.addEventListener('click', handleDocClick)
  } else {
    document.removeEventListener('click', handleDocClick)
  }
})
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
  height: calc(100vh - 60px);
  overflow: hidden;
}
.sidebar {
  width: 280px;
  flex-shrink: 0;
  border-right: 1px solid var(--ops-border-color, #e5e6eb);
  display: flex;
  flex-direction: column;
  background: #fafafa;
}
.sidebar-search {
  padding: 8px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}
.key-list {
  flex: 1;
  overflow: auto;
  padding: 4px 0;
}
.key-item {
  display: flex;
  align-items: center;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 13px;
  gap: 6px;
  transition: background 0.15s;
}
.key-item:hover {
  background: rgb(var(--primary-1, 232, 243, 255));
}
.key-item.active {
  background: rgb(var(--primary-2, 201, 225, 255));
}
.key-type-badge {
  display: inline-block;
  min-width: 26px;
  text-align: center;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
}
.type-string { background: #165dff; }
.type-hash { background: #00b42a; }
.type-list { background: #ff7d00; }
.type-set { background: #f53f3f; }
.type-zset { background: #86909c; }
.key-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.key-ttl, .key-ttl-forever {
  font-size: 11px;
  color: var(--ops-text-tertiary, #86909c);
  flex-shrink: 0;
}
.key-empty {
  text-align: center;
  color: var(--ops-text-tertiary, #86909c);
  padding: 30px;
  font-size: 13px;
}
.load-more {
  text-align: center;
  padding: 8px;
}
.main-area {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.main-area :deep(.arco-tabs) {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.main-area :deep(.arco-tabs-content) {
  flex: 1;
  overflow: auto;
  padding: 12px;
}
.main-area :deep(.arco-tabs-content-item) {
  height: 100%;
}
.empty-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--ops-text-tertiary, #86909c);
  font-size: 14px;
}
.key-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.detail-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.detail-key-name {
  font-weight: 600;
  font-size: 14px;
  word-break: break-all;
}
.detail-body {
  flex: 1;
}
.detail-actions {
  margin-top: 8px;
  display: flex;
  gap: 8px;
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
.cli-line {
  margin-bottom: 8px;
}
.cli-command {
  color: #4fc1ff;
}
.cli-result {
  color: #d4d4d4;
  margin: 2px 0 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: inherit;
  font-size: inherit;
}
.cli-result.cli-error {
  color: #f53f3f;
}
.cli-input-row {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: #252526;
  border-top: 1px solid #333;
  gap: 8px;
}
.cli-prompt {
  color: #00b42a;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  white-space: nowrap;
}
.cli-input-row :deep(.arco-input-wrapper) {
  background: transparent;
  border: none;
}
.cli-input-row :deep(.arco-input) {
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
}
/* Info */
.info-container {
  padding: 4px;
}
.info-actions {
  margin-bottom: 12px;
}
/* Context menu */
.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.1);
  z-index: 9999;
  min-width: 120px;
  padding: 4px 0;
}
.context-menu-item {
  padding: 6px 16px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}
.context-menu-item:hover {
  background: rgb(var(--primary-1, 232, 243, 255));
}
.context-menu-item.danger {
  color: #f53f3f;
}
.context-menu-item.danger:hover {
  background: #ffece8;
}
</style>