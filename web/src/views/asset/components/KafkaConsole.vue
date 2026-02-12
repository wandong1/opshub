<template>
  <el-drawer
    :model-value="visible"
    :title="'Kafka 控制台 - ' + (middleware?.name || '')"
    direction="rtl"
    size="100%"
    :destroy-on-close="true"
    class="kafka-console-drawer"
    @close="handleClose"
  >
    <template #header>
      <div class="console-header">
        <div class="header-left">
          <el-icon style="font-size: 18px; color: #909399;"><Connection /></el-icon>
          <span class="header-title">Kafka 控制台 - {{ middleware?.name }}</span>
        </div>
        <div class="header-actions">
          <el-button size="small" @click="refreshAll">
            <el-icon><Refresh /></el-icon> 刷新
          </el-button>
        </div>
      </div>
    </template>

    <div class="console-body">
      <!-- 左侧 Topic 列表 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <el-icon><List /></el-icon>
          <span>Topics</span>
          <div style="flex:1"></div>
          <el-tooltip content="新建 Topic" placement="top">
            <el-icon class="sidebar-action" @click="showCreateTopicDialog"><Plus /></el-icon>
          </el-tooltip>
          <el-tooltip content="刷新" placement="top">
            <el-icon class="sidebar-action" @click="loadTopics"><Refresh /></el-icon>
          </el-tooltip>
        </div>
        <el-input v-model="topicSearch" placeholder="搜索 Topic..." clearable size="small" class="sidebar-search">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <div class="topic-list" v-loading="topicsLoading">
          <div
            v-for="t in filteredTopics"
            :key="t.name"
            class="topic-item"
            :class="{ active: selectedTopic === t.name, internal: t.isInternal }"
            @click="handleSelectTopic(t.name)"
          >
            <span class="topic-name">{{ t.name }}</span>
            <span class="topic-meta">P:{{ t.partitions }} R:{{ t.replicas }}</span>
          </div>
<!-- KAFKA_SIDEBAR_END -->
          <div v-if="!filteredTopics.length && !topicsLoading" class="topic-empty">暂无 Topic</div>
        </div>
      </div>

      <!-- 右侧主区域 -->
      <div class="main-area">
        <el-tabs v-model="activeTab" type="border-card">
          <!-- Topics Tab -->
          <el-tab-pane label="Topics" name="topics">
            <div v-if="!selectedTopic" class="empty-tip">请从左侧选择一个 Topic</div>
            <div v-else class="topic-detail">
              <el-descriptions :column="4" border size="small" style="margin-bottom: 16px">
                <el-descriptions-item label="Topic">{{ selectedTopic }}</el-descriptions-item>
                <el-descriptions-item label="分区数">{{ topicDetail?.partitions?.length || 0 }}</el-descriptions-item>
                <el-descriptions-item label="副本数">{{ currentTopicInfo?.replicas || 0 }}</el-descriptions-item>
                <el-descriptions-item label="系统 Topic">{{ currentTopicInfo?.isInternal ? '是' : '否' }}</el-descriptions-item>
              </el-descriptions>
              <el-tabs v-model="topicSubTab" type="card">
                <el-tab-pane label="分区详情" name="partitions">
                  <el-table :data="topicDetail?.partitions || []" size="small" max-height="400">
                    <el-table-column prop="id" label="分区 ID" width="80" />
                    <el-table-column prop="leader" label="Leader" width="80" />
                    <el-table-column label="Replicas"><template #default="{ row }">{{ row.replicas?.join(', ') }}</template></el-table-column>
                    <el-table-column label="ISR"><template #default="{ row }">{{ row.isr?.join(', ') }}</template></el-table-column>
                  </el-table>
                </el-tab-pane>
                <el-tab-pane label="关联消费组" name="groups">
                  <el-table :data="topicDetail?.consumerGroups || []" size="small" max-height="400">
                    <el-table-column prop="groupId" label="消费组" />
                    <el-table-column prop="totalLag" label="总 Lag" width="120" />
                  </el-table>
                </el-tab-pane>
                <el-tab-pane label="配置" name="config">
                  <div style="margin-bottom: 8px; text-align: right"><el-button size="small" @click="loadTopicConfig">刷新</el-button></div>
                  <el-table :data="topicConfigs" size="small" max-height="400">
                    <el-table-column prop="name" label="配置项" min-width="200" />
                    <el-table-column label="值" min-width="200">
                      <template #default="{ row }">
                        <el-input v-if="editingConfigKey === row.name" v-model="editingConfigValue" size="small" style="width: 200px" @keyup.enter="saveConfigEdit(row.name)" />
                        <span v-else>{{ row.value }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="readOnly" label="只读" width="60"><template #default="{ row }">{{ row.readOnly ? '是' : '否' }}</template></el-table-column>
                    <el-table-column label="操作" width="100">
                      <template #default="{ row }">
                        <el-button v-if="!row.readOnly && editingConfigKey !== row.name" link size="small" type="primary" @click="startConfigEdit(row)">编辑</el-button>
                        <template v-if="editingConfigKey === row.name">
                          <el-button link size="small" type="success" @click="saveConfigEdit(row.name)">保存</el-button>
                          <el-button link size="small" @click="editingConfigKey = ''">取消</el-button>
                        </template>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-tab-pane>
<!-- KAFKA_CONSUMER_GROUP_TAB -->
          <!-- 消费组 Tab -->
          <el-tab-pane label="消费组" name="consumerGroups">
            <div style="margin-bottom: 8px; text-align: right"><el-button size="small" @click="loadConsumerGroups">刷新</el-button></div>
            <el-table :data="consumerGroups" v-loading="groupsLoading" size="small" max-height="250" highlight-current-row @current-change="handleGroupSelect">
              <el-table-column prop="groupId" label="消费组 ID" />
              <el-table-column prop="protocolType" label="协议类型" width="120" />
              <el-table-column label="操作" width="80">
                <template #default="{ row }">
                  <el-popconfirm title="确定删除该消费组？" @confirm="handleDeleteGroup(row.groupId)">
                    <template #reference><el-button link size="small" type="danger">删除</el-button></template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
            <div v-if="selectedGroupDetail" style="margin-top: 16px">
              <h4 style="margin: 0 0 8px">消费组详情 - {{ selectedGroupDetail.groupId }}</h4>
              <el-descriptions :column="3" border size="small" style="margin-bottom: 12px">
                <el-descriptions-item label="状态">{{ selectedGroupDetail.state }}</el-descriptions-item>
                <el-descriptions-item label="协议">{{ selectedGroupDetail.protocol }}</el-descriptions-item>
                <el-descriptions-item label="成员数">{{ selectedGroupDetail.members?.length || 0 }}</el-descriptions-item>
              </el-descriptions>
              <el-tabs type="card" model-value="members">
                <el-tab-pane label="成员" name="members">
                  <el-table :data="selectedGroupDetail.members || []" size="small" max-height="200">
                    <el-table-column prop="clientId" label="Client ID" />
                    <el-table-column prop="clientHost" label="Host" width="150" />
                    <el-table-column label="分配的分区"><template #default="{ row }">{{ row.assignments?.join(', ') || '-' }}</template></el-table-column>
                  </el-table>
                </el-tab-pane>
                <el-tab-pane label="Lag" name="lag">
                  <el-table :data="selectedGroupDetail.lag || []" size="small" max-height="200">
                    <el-table-column prop="topic" label="Topic" />
                    <el-table-column prop="partition" label="分区" width="60" />
                    <el-table-column prop="currentOffset" label="当前偏移" width="100" />
                    <el-table-column prop="logEndOffset" label="日志末尾" width="100" />
                    <el-table-column prop="lag" label="Lag" width="80">
                      <template #default="{ row }"><el-tag :type="row.lag > 0 ? 'warning' : 'success'" size="small">{{ row.lag }}</el-tag></template>
                    </el-table-column>
                  </el-table>
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-tab-pane>
<!-- KAFKA_MESSAGE_TAB -->
          <!-- 消息监控 Tab -->
          <el-tab-pane label="消息监控" name="messages">
            <!-- 生产消息 -->
            <el-collapse v-model="produceCollapse">
              <el-collapse-item title="发送消息" name="produce">
                <el-form :model="produceForm" label-width="80px" size="small">
                  <el-form-item label="Topic">
                    <el-select v-model="produceForm.topic" filterable placeholder="选择 Topic" style="width: 100%">
                      <el-option v-for="t in topics" :key="t.name" :label="t.name" :value="t.name" />
                    </el-select>
                  </el-form-item>
                  <el-form-item label="Key"><el-input v-model="produceForm.key" placeholder="可选" /></el-form-item>
                  <el-form-item label="Value"><el-input v-model="produceForm.value" type="textarea" :rows="3" placeholder="消息内容" /></el-form-item>
                  <el-form-item label="Headers">
                    <div v-for="(h, i) in produceForm.headers" :key="i" style="display: flex; gap: 8px; margin-bottom: 4px; width: 100%">
                      <el-input v-model="h.key" placeholder="Key" style="flex: 1" />
                      <el-input v-model="h.value" placeholder="Value" style="flex: 1" />
                      <el-button link type="danger" @click="produceForm.headers.splice(i, 1)"><el-icon><Delete /></el-icon></el-button>
                    </div>
                    <el-button size="small" @click="produceForm.headers.push({ key: '', value: '' })">添加 Header</el-button>
                  </el-form-item>
                  <el-form-item><el-button type="primary" @click="handleProduce" :loading="produceLoading">发送</el-button></el-form-item>
                </el-form>
              </el-collapse-item>
            </el-collapse>
            <!-- 消费监听 -->
            <div class="consume-section">
              <div class="consume-controls">
                <el-select v-model="consumeForm.topic" filterable placeholder="选择 Topic" size="small" style="width: 200px">
                  <el-option v-for="t in topics" :key="t.name" :label="t.name" :value="t.name" />
                </el-select>
                <el-select v-model="consumeForm.startOffset" size="small" style="width: 120px">
                  <el-option label="最新" value="latest" />
                  <el-option label="最早" value="earliest" />
                </el-select>
                <el-input v-model="consumeForm.keyword" placeholder="关键字过滤" clearable size="small" style="width: 180px" />
                <el-button v-if="!consuming" type="success" size="small" @click="startConsume">开始监听</el-button>
                <el-button v-else type="danger" size="small" @click="stopConsume">停止监听</el-button>
                <span v-if="consuming" class="consume-status">监听中... ({{ consumedMessages.length }} 条)</span>
              </div>
              <el-table :data="consumedMessages" size="small" max-height="350" style="margin-top: 8px" @row-click="showMessageDetail">
                <el-table-column prop="partition" label="分区" width="60" />
                <el-table-column prop="offset" label="偏移量" width="80" />
                <el-table-column label="时间" width="160"><template #default="{ row }">{{ formatTime(row.timestamp) }}</template></el-table-column>
                <el-table-column label="Key" width="120" show-overflow-tooltip><template #default="{ row }">{{ row.key || '-' }}</template></el-table-column>
                <el-table-column label="Value" show-overflow-tooltip><template #default="{ row }">{{ truncate(row.value, 100) }}</template></el-table-column>
                <el-table-column prop="size" label="大小" width="70"><template #default="{ row }">{{ row.size }}B</template></el-table-column>
              </el-table>
            </div>
          </el-tab-pane>
<!-- KAFKA_CLUSTER_TAB -->
          <!-- 集群信息 Tab -->
          <el-tab-pane label="集群信息" name="cluster">
            <div style="margin-bottom: 8px; text-align: right"><el-button size="small" @click="loadBrokers">刷新</el-button></div>
            <el-table :data="brokers" v-loading="brokersLoading" size="small">
              <el-table-column prop="id" label="Broker ID" width="100" />
              <el-table-column prop="addr" label="地址" />
              <el-table-column prop="rack" label="Rack" width="100"><template #default="{ row }">{{ row.rack || '-' }}</template></el-table-column>
              <el-table-column prop="isController" label="Controller" width="100"><template #default="{ row }"><el-tag v-if="row.isController" type="success" size="small">是</el-tag><span v-else>否</span></template></el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <!-- 创建 Topic 弹窗 -->
    <el-dialog v-model="createTopicVisible" title="新建 Topic" width="500px" append-to-body>
      <el-form :model="createTopicForm" label-width="100px" size="small">
        <el-form-item label="Topic 名称" required><el-input v-model="createTopicForm.name" placeholder="请输入 Topic 名称" /></el-form-item>
        <el-form-item label="分区数"><el-input-number v-model="createTopicForm.numPartitions" :min="1" :max="1000" /></el-form-item>
        <el-form-item label="副本因子"><el-input-number v-model="createTopicForm.replicationFactor" :min="1" :max="10" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createTopicVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateTopic" :loading="createTopicLoading">创建</el-button>
      </template>
    </el-dialog>

    <!-- 消息详情弹窗 -->
    <el-dialog v-model="messageDetailVisible" title="消息详情" width="600px" append-to-body>
      <el-descriptions :column="2" border size="small" v-if="messageDetailData">
        <el-descriptions-item label="分区">{{ messageDetailData.partition }}</el-descriptions-item>
        <el-descriptions-item label="偏移量">{{ messageDetailData.offset }}</el-descriptions-item>
        <el-descriptions-item label="时间">{{ formatTime(messageDetailData.timestamp) }}</el-descriptions-item>
        <el-descriptions-item label="大小">{{ messageDetailData.size }}B</el-descriptions-item>
        <el-descriptions-item label="Key" :span="2">{{ messageDetailData.key || '-' }}</el-descriptions-item>
      </el-descriptions>
      <div v-if="messageDetailData" style="margin-top: 12px">
        <h4 style="margin: 0 0 8px">Value</h4>
        <el-input type="textarea" :model-value="messageDetailData.value" :rows="8" readonly />
      </div>
      <div v-if="messageDetailData?.headers && Object.keys(messageDetailData.headers).length" style="margin-top: 12px">
        <h4 style="margin: 0 0 8px">Headers</h4>
        <el-table :data="Object.entries(messageDetailData.headers).map(([k,v]) => ({key:k,value:v}))" size="small">
          <el-table-column prop="key" label="Key" />
          <el-table-column prop="value" label="Value" />
        </el-table>
      </div>
    </el-dialog>
  </el-drawer>
</template>

<!-- KAFKA_SCRIPT_START -->
<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, Refresh, List, Plus, Search, Delete } from '@element-plus/icons-vue'
import {
  getKafkaBrokers, getKafkaTopics, createKafkaTopic, deleteKafkaTopic,
  getKafkaTopicDetail, getKafkaTopicConfig, updateKafkaTopicConfig,
  getKafkaConsumerGroups, getKafkaConsumerGroupDetail, deleteKafkaConsumerGroup,
  produceKafkaMessage, startKafkaConsumerSession, pollKafkaConsumerSession, stopKafkaConsumerSession
} from '@/api/middleware'

const props = defineProps<{ visible: boolean; middleware: any }>()
const emit = defineEmits(['update:visible'])

// State
const activeTab = ref('topics')
const topicSearch = ref('')
const topics = ref<any[]>([])
const topicsLoading = ref(false)
const selectedTopic = ref('')
const topicSubTab = ref('partitions')
const topicDetail = ref<any>(null)
const topicConfigs = ref<any[]>([])
const editingConfigKey = ref('')
const editingConfigValue = ref('')

const consumerGroups = ref<any[]>([])
const groupsLoading = ref(false)
const selectedGroupDetail = ref<any>(null)

const brokers = ref<any[]>([])
const brokersLoading = ref(false)

const createTopicVisible = ref(false)
const createTopicLoading = ref(false)
const createTopicForm = reactive({ name: '', numPartitions: 1, replicationFactor: 1 })

const produceCollapse = ref(['produce'])
const produceForm = reactive({ topic: '', key: '', value: '', headers: [] as { key: string; value: string }[] })
const produceLoading = ref(false)

const consumeForm = reactive({ topic: '', startOffset: 'latest', keyword: '' })
const consuming = ref(false)
const consumeSessionId = ref('')
const consumedMessages = ref<any[]>([])
let pollTimer: ReturnType<typeof setInterval> | null = null

const messageDetailVisible = ref(false)
const messageDetailData = ref<any>(null)

// KAFKA_SCRIPT_COMPUTED

const filteredTopics = computed(() => {
  if (!topicSearch.value) return topics.value
  const kw = topicSearch.value.toLowerCase()
  return topics.value.filter((t: any) => t.name.toLowerCase().includes(kw))
})

const currentTopicInfo = computed(() => topics.value.find((t: any) => t.name === selectedTopic.value))

const middlewareId = computed(() => props.middleware?.id)

// Watchers
watch(() => props.visible, (val) => {
  if (val && middlewareId.value) {
    loadTopics()
  }
})

// Methods
const loadTopics = async () => {
  if (!middlewareId.value) return
  topicsLoading.value = true
  try {
    const res = await getKafkaTopics(middlewareId.value)
    topics.value = res || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载 Topic 失败')
  } finally {
    topicsLoading.value = false
  }
}

const handleSelectTopic = async (name: string) => {
  selectedTopic.value = name
  activeTab.value = 'topics'
  topicSubTab.value = 'partitions'
  produceForm.topic = name
  consumeForm.topic = name
  await loadTopicDetail()
}

const loadTopicDetail = async () => {
  if (!middlewareId.value || !selectedTopic.value) return
  try {
    const res = await getKafkaTopicDetail(middlewareId.value, selectedTopic.value)
    topicDetail.value = res
  } catch (e: any) {
    ElMessage.error(e.message || '加载详情失败')
  }
}

const loadTopicConfig = async () => {
  if (!middlewareId.value || !selectedTopic.value) return
  try {
    const res = await getKafkaTopicConfig(middlewareId.value, selectedTopic.value)
    topicConfigs.value = res || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载配置失败')
  }
}

// KAFKA_SCRIPT_METHODS_PART2

const startConfigEdit = (row: any) => {
  editingConfigKey.value = row.name
  editingConfigValue.value = row.value
}

const saveConfigEdit = async (name: string) => {
  if (!middlewareId.value || !selectedTopic.value) return
  try {
    await updateKafkaTopicConfig(middlewareId.value, { topic: selectedTopic.value, configs: { [name]: editingConfigValue.value } })
    ElMessage.success('修改成功')
    editingConfigKey.value = ''
    loadTopicConfig()
  } catch (e: any) {
    ElMessage.error(e.message || '修改失败')
  }
}

const showCreateTopicDialog = () => {
  createTopicForm.name = ''
  createTopicForm.numPartitions = 1
  createTopicForm.replicationFactor = 1
  createTopicVisible.value = true
}

const handleCreateTopic = async () => {
  if (!middlewareId.value || !createTopicForm.name) return
  createTopicLoading.value = true
  try {
    await createKafkaTopic(middlewareId.value, createTopicForm)
    ElMessage.success('创建成功')
    createTopicVisible.value = false
    loadTopics()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    createTopicLoading.value = false
  }
}

const loadConsumerGroups = async () => {
  if (!middlewareId.value) return
  groupsLoading.value = true
  try {
    const res = await getKafkaConsumerGroups(middlewareId.value)
    consumerGroups.value = res || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载消费组失败')
  } finally {
    groupsLoading.value = false
  }
}

const handleGroupSelect = async (row: any) => {
  if (!row || !middlewareId.value) { selectedGroupDetail.value = null; return }
  try {
    const res = await getKafkaConsumerGroupDetail(middlewareId.value, row.groupId)
    selectedGroupDetail.value = res
  } catch (e: any) {
    ElMessage.error(e.message || '加载详情失败')
  }
}

const handleDeleteGroup = async (groupId: string) => {
  if (!middlewareId.value) return
  try {
    await deleteKafkaConsumerGroup(middlewareId.value, groupId)
    ElMessage.success('删除成功')
    selectedGroupDetail.value = null
    loadConsumerGroups()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

// KAFKA_SCRIPT_METHODS_PART3

const loadBrokers = async () => {
  if (!middlewareId.value) return
  brokersLoading.value = true
  try {
    const res = await getKafkaBrokers(middlewareId.value)
    brokers.value = res || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载 Broker 失败')
  } finally {
    brokersLoading.value = false
  }
}

const handleProduce = async () => {
  if (!middlewareId.value || !produceForm.topic || !produceForm.value) {
    ElMessage.warning('请填写 Topic 和消息内容')
    return
  }
  produceLoading.value = true
  try {
    const headers: Record<string, string> = {}
    for (const h of produceForm.headers) {
      if (h.key) headers[h.key] = h.value
    }
    const res = await produceKafkaMessage(middlewareId.value, {
      topic: produceForm.topic,
      key: produceForm.key || undefined,
      value: produceForm.value,
      headers: Object.keys(headers).length ? headers : undefined,
    })
    ElMessage.success(`发送成功 (分区: ${res?.partition}, 偏移量: ${res?.offset})`)
  } catch (e: any) {
    ElMessage.error(e.message || '发送失败')
  } finally {
    produceLoading.value = false
  }
}

const startConsume = async () => {
  if (!middlewareId.value || !consumeForm.topic) {
    ElMessage.warning('请选择 Topic')
    return
  }
  try {
    const res = await startKafkaConsumerSession(middlewareId.value, {
      topic: consumeForm.topic,
      startOffset: consumeForm.startOffset,
    })
    consumeSessionId.value = res?.sessionId
    consuming.value = true
    consumedMessages.value = []
    pollTimer = setInterval(pollMessages, 1500)
  } catch (e: any) {
    ElMessage.error(e.message || '启动消费失败')
  }
}

const pollMessages = async () => {
  if (!middlewareId.value || !consumeSessionId.value) return
  try {
    const res = await pollKafkaConsumerSession(middlewareId.value, {
      sessionId: consumeSessionId.value,
      keyword: consumeForm.keyword || undefined,
    })
    if (res && res.length) {
      consumedMessages.value = [...consumedMessages.value, ...res]
      if (consumedMessages.value.length > 5000) {
        consumedMessages.value = consumedMessages.value.slice(-5000)
      }
    }
  } catch {
    // Session may have expired
  }
}

const stopConsume = async () => {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
  if (middlewareId.value && consumeSessionId.value) {
    try { await stopKafkaConsumerSession(middlewareId.value, consumeSessionId.value) } catch {}
  }
  consuming.value = false
  consumeSessionId.value = ''
}

const showMessageDetail = (row: any) => {
  messageDetailData.value = row
  messageDetailVisible.value = true
}

const refreshAll = () => {
  loadTopics()
  if (activeTab.value === 'consumerGroups') loadConsumerGroups()
  if (activeTab.value === 'cluster') loadBrokers()
}

const handleClose = () => {
  stopConsume()
  emit('update:visible', false)
}

const formatTime = (ts: string) => {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.toLocaleString()
}

const truncate = (str: string, len: number) => {
  if (!str) return ''
  return str.length > len ? str.slice(0, len) + '...' : str
}

// Tab change handlers
watch(activeTab, (val) => {
  if (val === 'consumerGroups' && !consumerGroups.value.length) loadConsumerGroups()
  if (val === 'cluster' && !brokers.value.length) loadBrokers()
})

watch(topicSubTab, (val) => {
  if (val === 'config' && !topicConfigs.value.length) loadTopicConfig()
})

onBeforeUnmount(() => { stopConsume() })
</script>

<!-- KAFKA_STYLE_START -->
<style scoped>
.console-header { display: flex; justify-content: space-between; align-items: center; width: 100%; }
.header-left { display: flex; align-items: center; gap: 8px; }
.header-title { font-size: 16px; font-weight: 600; }
.header-actions { display: flex; gap: 8px; }
.console-body { display: flex; height: calc(100vh - 80px); gap: 0; }
.sidebar { width: 280px; flex-shrink: 0; border-right: 1px solid #ebeef5; display: flex; flex-direction: column; }
.sidebar-header { display: flex; align-items: center; gap: 6px; padding: 12px; font-weight: 600; font-size: 14px; border-bottom: 1px solid #ebeef5; }
.sidebar-action { cursor: pointer; color: #409eff; font-size: 16px; }
.sidebar-action:hover { color: #66b1ff; }
.sidebar-search { padding: 8px 12px; }
.topic-list { flex: 1; overflow: auto; padding: 4px 0; }
.topic-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; cursor: pointer; font-size: 13px; border-left: 3px solid transparent; }
.topic-item:hover { background: #f5f7fa; }
.topic-item.active { background: #ecf5ff; border-left-color: #409eff; }
.topic-item.internal .topic-name { color: #909399; }
.topic-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.topic-meta { font-size: 11px; color: #909399; margin-left: 8px; white-space: nowrap; }
.topic-empty { padding: 20px; text-align: center; color: #909399; font-size: 13px; }
.main-area { flex: 1; overflow: auto; }
.empty-tip { padding: 40px; text-align: center; color: #909399; }
.topic-detail { padding: 0 4px; }
.consume-section { margin-top: 16px; }
.consume-controls { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.consume-status { font-size: 12px; color: #67c23a; }
</style>
