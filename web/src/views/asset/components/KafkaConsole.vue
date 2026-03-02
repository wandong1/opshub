<template>
  <a-drawer
    :visible="visible"
    :title="'Kafka 控制台 - ' + (middleware?.name || '')"
    placement="right"
    :width="'100%'"
    unmount-on-close
    class="kafka-console-drawer"
    @cancel="handleClose"
  >
    <template #title>
      <div class="console-header">
        <div class="header-left">
          <icon-link style="font-size: 18px; color: #86909c;" />
          <span class="header-title">Kafka 控制台 - {{ middleware?.name }}</span>
        </div>
        <div class="header-actions">
          <a-button size="small" @click="refreshAll">
            <template #icon><icon-refresh /></template> 刷新
          </a-button>
        </div>
      </div>
    </template>

    <div class="console-body">
      <!-- 左侧 Topic 列表 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <icon-list />
          <span>Topics</span>
          <div style="flex:1"></div>
          <a-tooltip content="新建 Topic" position="top">
            <icon-plus class="sidebar-action" @click="showCreateTopicDialog" />
          </a-tooltip>
          <a-tooltip content="刷新" position="top">
            <icon-refresh class="sidebar-action" @click="loadTopics" />
          </a-tooltip>
        </div>
        <a-input v-model="topicSearch" placeholder="搜索 Topic..." allow-clear size="small" class="sidebar-search">
          <template #prefix><icon-search /></template>
        </a-input>
        <a-spin :loading="topicsLoading" style="width: 100%; flex: 1;">
          <div class="topic-list">
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
        </a-spin>
      </div>
      <!-- 右侧主区域 -->
      <div class="main-area">
        <a-tabs v-model:active-key="activeTab" type="card-gutter">
          <!-- Topics Tab -->
          <a-tab-pane title="Topics" key="topics">
            <div v-if="!selectedTopic" class="empty-tip">请从左侧选择一个 Topic</div>
            <div v-else class="topic-detail">
              <a-descriptions :column="4" bordered size="small" style="margin-bottom: 16px">
                <a-descriptions-item label="Topic">{{ selectedTopic }}</a-descriptions-item>
                <a-descriptions-item label="分区数">{{ topicDetail?.partitions?.length || 0 }}</a-descriptions-item>
                <a-descriptions-item label="副本数">{{ currentTopicInfo?.replicas || 0 }}</a-descriptions-item>
                <a-descriptions-item label="系统 Topic">{{ currentTopicInfo?.isInternal ? '是' : '否' }}</a-descriptions-item>
              </a-descriptions>
              <a-tabs v-model:active-key="topicSubTab" type="card-gutter">
                <a-tab-pane title="分区详情" key="partitions">
                  <a-table :data="topicDetail?.partitions || []" size="small" :bordered="{ cell: true }" stripe :pagination="false" :scroll="{ y: 400 }">
                    <template #columns>
                      <a-table-column title="分区 ID" data-index="id" :width="80" />
                      <a-table-column title="Leader" data-index="leader" :width="80" />
                      <a-table-column title="Replicas"><template #cell="{ record }">{{ record.replicas?.join(', ') }}</template></a-table-column>
                      <a-table-column title="ISR"><template #cell="{ record }">{{ record.isr?.join(', ') }}</template></a-table-column>
                    </template>
                  </a-table>
                </a-tab-pane>
                <a-tab-pane title="关联消费组" key="groups">
                  <a-table :data="topicDetail?.consumerGroups || []" size="small" :bordered="{ cell: true }" stripe :pagination="false" :scroll="{ y: 400 }">
                    <template #columns>
                      <a-table-column title="消费组" data-index="groupId" />
                      <a-table-column title="总 Lag" data-index="totalLag" :width="120" />
                    </template>
                  </a-table>
                </a-tab-pane>
                <a-tab-pane title="配置" key="config">
                  <div style="margin-bottom: 8px; text-align: right"><a-button size="small" @click="loadTopicConfig">刷新</a-button></div>
                  <a-table :data="topicConfigs" size="small" :bordered="{ cell: true }" stripe :pagination="false" :scroll="{ y: 400 }">
                    <template #columns>
                      <a-table-column title="配置项" data-index="name" :min-width="200" />
                      <a-table-column title="值" :min-width="200">
                        <template #cell="{ record }">
                          <a-input v-if="editingConfigKey === record.name" v-model="editingConfigValue" size="small" style="width: 200px" @keyup.enter="saveConfigEdit(record.name)" />
                          <span v-else>{{ record.value }}</span>
                        </template>
                      </a-table-column>
                      <a-table-column title="只读" data-index="readOnly" :width="60"><template #cell="{ record }">{{ record.readOnly ? '是' : '否' }}</template></a-table-column>
                      <a-table-column title="操作" :width="100">
                        <template #cell="{ record }">
                          <a-button v-if="!record.readOnly && editingConfigKey !== record.name" type="text" size="small" @click="startConfigEdit(record)">编辑</a-button>
                          <template v-if="editingConfigKey === record.name">
                            <a-button type="text" size="small" status="success" @click="saveConfigEdit(record.name)">保存</a-button>
                            <a-button type="text" size="small" @click="editingConfigKey = ''">取消</a-button>
                          </template>
                        </template>
                      </a-table-column>
                    </template>
                  </a-table>
                </a-tab-pane>
              </a-tabs>
            </div>
          </a-tab-pane>
<!-- KAFKA_CONSUMER_GROUP_TAB -->
          <!-- 消费组 Tab -->
          <a-tab-pane title="消费组" key="consumerGroups">
            <div style="margin-bottom: 8px; text-align: right"><a-button size="small" @click="loadConsumerGroups">刷新</a-button></div>
            <a-table :data="consumerGroups" :loading="groupsLoading" size="small" :bordered="{ cell: true }" stripe :pagination="false" :scroll="{ y: 250 }" row-key="groupId" :row-selection="undefined" @row-click="handleGroupRowClick">
              <template #columns>
                <a-table-column title="消费组 ID" data-index="groupId" />
                <a-table-column title="协议类型" data-index="protocolType" :width="120" />
                <a-table-column title="操作" :width="80">
                  <template #cell="{ record }">
                    <a-popconfirm content="确定删除该消费组？" @ok="handleDeleteGroup(record.groupId)">
                      <a-button type="text" size="small" status="danger">删除</a-button>
                    </a-popconfirm>
                  </template>
                </a-table-column>
              </template>
            </a-table>
            <div v-if="selectedGroupDetail" style="margin-top: 16px">
              <h4 style="margin: 0 0 8px">消费组详情 - {{ selectedGroupDetail.groupId }}</h4>
              <a-descriptions :column="3" bordered size="small" style="margin-bottom: 12px">
                <a-descriptions-item label="状态">{{ selectedGroupDetail.state }}</a-descriptions-item>
                <a-descriptions-item label="协议">{{ selectedGroupDetail.protocol }}</a-descriptions-item>
                <a-descriptions-item label="成员数">{{ selectedGroupDetail.members?.length || 0 }}</a-descriptions-item>
              </a-descriptions>
              <a-tabs type="card-gutter" :default-active-key="'members'">
                <a-tab-pane title="成员" key="members">
                  <a-table :data="selectedGroupDetail.members || []" size="small" :bordered="{ cell: true }" stripe :pagination="false" :scroll="{ y: 200 }">
                    <template #columns>
                      <a-table-column title="Client ID" data-index="clientId" />
                      <a-table-column title="Host" data-index="clientHost" :width="150" />
                      <a-table-column title="分配的分区"><template #cell="{ record }">{{ record.assignments?.join(', ') || '-' }}</template></a-table-column>
                    </template>
                  </a-table>
                </a-tab-pane>
                <a-tab-pane title="Lag" key="lag">
                  <a-table :data="selectedGroupDetail.lag || []" size="small" :bordered="{ cell: true }" stripe :pagination="false" :scroll="{ y: 200 }">
                    <template #columns>
                      <a-table-column title="Topic" data-index="topic" />
                      <a-table-column title="分区" data-index="partition" :width="60" />
                      <a-table-column title="当前偏移" data-index="currentOffset" :width="100" />
                      <a-table-column title="日志末尾" data-index="logEndOffset" :width="100" />
                      <a-table-column title="Lag" data-index="lag" :width="80">
                        <template #cell="{ record }"><a-tag :color="record.lag > 0 ? 'orangered' : 'green'" size="small">{{ record.lag }}</a-tag></template>
                      </a-table-column>
                    </template>
                  </a-table>
                </a-tab-pane>
              </a-tabs>
            </div>
          </a-tab-pane>
<!-- KAFKA_MESSAGE_TAB -->
          <!-- 消息监控 Tab -->
          <a-tab-pane title="消息监控" key="messages">
            <!-- 生产消息 -->
            <a-collapse v-model:active-key="produceCollapse">
              <a-collapse-item header="发送消息" key="produce">
                <a-form :model="produceForm" auto-label-width layout="horizontal" size="small">
                  <a-form-item label="Topic" field="topic">
                    <a-select v-model="produceForm.topic" :allow-search="true" placeholder="选择 Topic" style="width: 100%">
                      <a-option v-for="t in topics" :key="t.name" :label="t.name" :value="t.name" />
                    </a-select>
                  </a-form-item>
                  <a-form-item label="Key" field="key"><a-input v-model="produceForm.key" placeholder="可选" /></a-form-item>
                  <a-form-item label="Value" field="value"><a-textarea v-model="produceForm.value" :auto-size="{ minRows: 3, maxRows: 3 }" placeholder="消息内容" /></a-form-item>
                  <a-form-item label="Headers" field="headers">
                    <div v-for="(h, i) in produceForm.headers" :key="i" style="display: flex; gap: 8px; margin-bottom: 4px; width: 100%">
                      <a-input v-model="h.key" placeholder="Key" style="flex: 1" />
                      <a-input v-model="h.value" placeholder="Value" style="flex: 1" />
                      <a-button type="text" status="danger" @click="produceForm.headers.splice(i, 1)"><icon-delete /></a-button>
                    </div>
                    <a-button size="small" @click="produceForm.headers.push({ key: '', value: '' })">添加 Header</a-button>
                  </a-form-item>
                  <a-form-item><a-button type="primary" @click="handleProduce" :loading="produceLoading">发送</a-button></a-form-item>
                </a-form>
              </a-collapse-item>
            </a-collapse>
            <!-- 消费监听 -->
            <div class="consume-section">
              <div class="consume-controls">
                <a-select v-model="consumeForm.topic" :allow-search="true" placeholder="选择 Topic" size="small" style="width: 200px">
                  <a-option v-for="t in topics" :key="t.name" :label="t.name" :value="t.name" />
                </a-select>
                <a-select v-model="consumeForm.startOffset" size="small" style="width: 120px">
                  <a-option label="最新" value="latest" />
                  <a-option label="最早" value="earliest" />
                </a-select>
                <a-input v-model="consumeForm.keyword" placeholder="关键字过滤" allow-clear size="small" style="width: 180px" />
                <a-button v-if="!consuming" type="primary" status="success" size="small" @click="startConsume">开始监听</a-button>
                <a-button v-else type="primary" status="danger" size="small" @click="stopConsume">停止监听</a-button>
                <span v-if="consuming" class="consume-status">监听中... ({{ consumedMessages.length }} 条)</span>
              </div>
              <a-table :data="consumedMessages" size="small" :bordered="{ cell: true }" stripe :pagination="false" :scroll="{ y: 350 }" style="margin-top: 8px" @row-click="showMessageDetail">
                <template #columns>
                  <a-table-column title="分区" data-index="partition" :width="60" />
                  <a-table-column title="偏移量" data-index="offset" :width="80" />
                  <a-table-column title="时间" :width="160"><template #cell="{ record }">{{ formatTime(record.timestamp) }}</template></a-table-column>
                  <a-table-column title="Key" :width="120" ellipsis tooltip><template #cell="{ record }">{{ record.key || '-' }}</template></a-table-column>
                  <a-table-column title="Value" ellipsis tooltip><template #cell="{ record }">{{ truncate(record.value, 100) }}</template></a-table-column>
                  <a-table-column title="大小" data-index="size" :width="70"><template #cell="{ record }">{{ record.size }}B</template></a-table-column>
                </template>
              </a-table>
            </div>
          </a-tab-pane>
<!-- KAFKA_CLUSTER_TAB -->
          <!-- 集群信息 Tab -->
          <a-tab-pane title="集群信息" key="cluster">
            <div style="margin-bottom: 8px; text-align: right"><a-button size="small" @click="loadBrokers">刷新</a-button></div>
            <a-table :data="brokers" :loading="brokersLoading" size="small" :bordered="{ cell: true }" stripe :pagination="false">
              <template #columns>
                <a-table-column title="Broker ID" data-index="id" :width="100" />
                <a-table-column title="地址" data-index="addr" />
                <a-table-column title="Rack" data-index="rack" :width="100"><template #cell="{ record }">{{ record.rack || '-' }}</template></a-table-column>
                <a-table-column title="Controller" data-index="isController" :width="100"><template #cell="{ record }"><a-tag v-if="record.isController" color="green" size="small">是</a-tag><span v-else>否</span></template></a-table-column>
              </template>
            </a-table>
          </a-tab-pane>
        </a-tabs>
      </div>
    </div>

    <!-- 创建 Topic 弹窗 -->
    <a-modal v-model:visible="createTopicVisible" title="新建 Topic" :width="500" unmount-on-close :mask-closable="false">
      <a-form :model="createTopicForm" auto-label-width layout="horizontal" size="small">
        <a-form-item label="Topic 名称" field="name" :rules="[{ required: true, message: '请输入 Topic 名称' }]"><a-input v-model="createTopicForm.name" placeholder="请输入 Topic 名称" /></a-form-item>
        <a-form-item label="分区数" field="numPartitions"><a-input-number v-model="createTopicForm.numPartitions" :min="1" :max="1000" /></a-form-item>
        <a-form-item label="副本因子" field="replicationFactor"><a-input-number v-model="createTopicForm.replicationFactor" :min="1" :max="10" /></a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="createTopicVisible = false">取消</a-button>
        <a-button type="primary" @click="handleCreateTopic" :loading="createTopicLoading">创建</a-button>
      </template>
    </a-modal>
    <!-- 消息详情弹窗 -->
    <a-modal v-model:visible="messageDetailVisible" title="消息详情" :width="600" unmount-on-close :mask-closable="false">
      <a-descriptions :column="2" bordered size="small" v-if="messageDetailData">
        <a-descriptions-item label="分区">{{ messageDetailData.partition }}</a-descriptions-item>
        <a-descriptions-item label="偏移量">{{ messageDetailData.offset }}</a-descriptions-item>
        <a-descriptions-item label="时间">{{ formatTime(messageDetailData.timestamp) }}</a-descriptions-item>
        <a-descriptions-item label="大小">{{ messageDetailData.size }}B</a-descriptions-item>
        <a-descriptions-item label="Key" :span="2">{{ messageDetailData.key || '-' }}</a-descriptions-item>
      </a-descriptions>
      <div v-if="messageDetailData" style="margin-top: 12px">
        <h4 style="margin: 0 0 8px">Value</h4>
        <a-textarea :model-value="messageDetailData.value" :auto-size="{ minRows: 8, maxRows: 8 }" readonly />
      </div>
      <div v-if="messageDetailData?.headers && Object.keys(messageDetailData.headers).length" style="margin-top: 12px">
        <h4 style="margin: 0 0 8px">Headers</h4>
        <a-table :data="Object.entries(messageDetailData.headers).map(([k,v]) => ({key:k,value:v}))" size="small" :bordered="{ cell: true }" stripe :pagination="false">
          <template #columns>
            <a-table-column title="Key" data-index="key" />
            <a-table-column title="Value" data-index="value" />
          </template>
        </a-table>
      </div>
      <template #footer><a-button @click="messageDetailVisible = false">关闭</a-button></template>
    </a-modal>
  </a-drawer>
</template>

<!-- KAFKA_SCRIPT_START -->
<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, reactive } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconLink, IconRefresh, IconList, IconPlus, IconSearch, IconDelete
} from '@arco-design/web-vue/es/icon'
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
    Message.error(e.message || '加载 Topic 失败')
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
    Message.error(e.message || '加载详情失败')
  }
}

const loadTopicConfig = async () => {
  if (!middlewareId.value || !selectedTopic.value) return
  try {
    const res = await getKafkaTopicConfig(middlewareId.value, selectedTopic.value)
    topicConfigs.value = res || []
  } catch (e: any) {
    Message.error(e.message || '加载配置失败')
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
    Message.success('修改成功')
    editingConfigKey.value = ''
    loadTopicConfig()
  } catch (e: any) {
    Message.error(e.message || '修改失败')
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
    Message.success('创建成功')
    createTopicVisible.value = false
    loadTopics()
  } catch (e: any) {
    Message.error(e.message || '创建失败')
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
    Message.error(e.message || '加载消费组失败')
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
    Message.error(e.message || '加载详情失败')
  }
}

const handleGroupRowClick = (record: any) => {
  handleGroupSelect(record)
}

const handleDeleteGroup = async (groupId: string) => {
  if (!middlewareId.value) return
  try {
    await deleteKafkaConsumerGroup(middlewareId.value, groupId)
    Message.success('删除成功')
    selectedGroupDetail.value = null
    loadConsumerGroups()
  } catch (e: any) {
    Message.error(e.message || '删除失败')
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
    Message.error(e.message || '加载 Broker 失败')
  } finally {
    brokersLoading.value = false
  }
}

const handleProduce = async () => {
  if (!middlewareId.value || !produceForm.topic || !produceForm.value) {
    Message.warning('请填写 Topic 和消息内容')
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
    Message.success(`发送成功 (分区: ${res?.partition}, 偏移量: ${res?.offset})`)
  } catch (e: any) {
    Message.error(e.message || '发送失败')
  } finally {
    produceLoading.value = false
  }
}

const startConsume = async () => {
  if (!middlewareId.value || !consumeForm.topic) {
    Message.warning('请选择 Topic')
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
    Message.error(e.message || '启动消费失败')
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
.sidebar { width: 280px; flex-shrink: 0; border-right: 1px solid var(--ops-border-color, #e5e6eb); display: flex; flex-direction: column; }
.sidebar-header { display: flex; align-items: center; gap: 6px; padding: 12px; font-weight: 600; font-size: 14px; border-bottom: 1px solid var(--ops-border-color, #e5e6eb); }
.sidebar-action { cursor: pointer; color: var(--ops-primary, #165dff); font-size: 16px; }
.sidebar-action:hover { color: rgb(var(--primary-5, 60, 126, 255)); }
.sidebar-search { padding: 8px 12px; }
.topic-list { flex: 1; overflow: auto; padding: 4px 0; }
.topic-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; cursor: pointer; font-size: 13px; border-left: 3px solid transparent; }
.topic-item:hover { background: var(--color-fill-1, #f7f8fa); }
.topic-item.active { background: var(--color-primary-light-1, #e8f3ff); border-left-color: var(--ops-primary, #165dff); }
.topic-item.internal .topic-name { color: var(--ops-text-tertiary, #86909c); }
.topic-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.topic-meta { font-size: 11px; color: var(--ops-text-tertiary, #86909c); margin-left: 8px; white-space: nowrap; }
.topic-empty { padding: 20px; text-align: center; color: var(--ops-text-tertiary, #86909c); font-size: 13px; }
.main-area { flex: 1; overflow: auto; }
.empty-tip { padding: 40px; text-align: center; color: var(--ops-text-tertiary, #86909c); }
.topic-detail { padding: 0 4px; }
.consume-section { margin-top: 16px; }
.consume-controls { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.consume-status { font-size: 12px; color: #00b42a; }
</style>
