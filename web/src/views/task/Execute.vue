<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-play-circle-fill />
        </div>
        <div>
          <div class="page-title">执行任务</div>
          <div class="page-desc">执行脚本任务，实时查看执行结果和日志</div>
        </div>
      </div>
    </div>

    <!-- 主要内容区 -->
    <a-card class="main-card" :bordered="false">
      <!-- 目标主机 -->
      <div class="form-section">
        <div class="section-label"><span class="required">*</span> 目标主机</div>
        <div class="section-body">
          <a-button @click="showHostDialog = true">
            <template #icon><icon-plus /></template>
            添加目标主机
          </a-button>
          <div v-if="selectedHosts.length > 0" class="tag-list">
            <a-tag
              v-for="host in selectedHosts"
              :key="host.id"
              closable
              @close="removeHost(host.id)"
              :color="host.agentStatus === 'online' ? 'green' : 'arcoblue'"
            >
              {{ host.name }} ({{ host.ip }})
              <span v-if="host.agentStatus === 'online'" style="margin-left: 4px; font-size: 11px;">Agent</span>
            </a-tag>
          </div>
        </div>
      </div>

      <!-- 执行命令 -->
      <div class="form-section">
        <div class="section-label"><span class="required">*</span> 执行命令</div>
        <div class="section-body">
          <div class="command-toolbar">
            <a-radio-group v-model="scriptType" type="button">
              <a-radio value="Shell">Shell</a-radio>
              <a-radio value="Python">Python</a-radio>
            </a-radio-group>
            <div class="toolbar-right">
              <a-link @click="showTemplateDialog = true">
                <template #icon><icon-plus /></template>
                从执行模板中选择
              </a-link>
            </div>
          </div>
          <a-textarea
            v-model="scriptContent"
            placeholder="请输入脚本内容..."
            :auto-size="{ minRows: 12, maxRows: 24 }"
            class="code-textarea"
          />
        </div>
      </div>

      <!-- 执行方式 -->
      <div class="form-section">
        <div class="section-label">执行方式</div>
        <div class="section-body">
          <a-radio-group v-model="executionMode" type="button">
            <a-radio value="auto">自动选择（Agent在线优先）</a-radio>
            <a-radio value="ssh">SSH 直连</a-radio>
            <a-radio value="agent">Agent 执行</a-radio>
          </a-radio-group>
          <div v-if="selectedHosts.length > 0 && executionMode === 'auto'" class="execution-hint">
            <span style="color: #00b42a;">Agent在线: {{ selectedHosts.filter(h => h.agentStatus === 'online').length }}台</span>
            <span style="margin-left: 12px; color: #86909c;">SSH回退: {{ selectedHosts.filter(h => h.agentStatus !== 'online').length }}台</span>
          </div>
        </div>
      </div>

      <!-- 执行按钮 -->
      <div class="execute-btn-wrap">
        <a-button
          v-permission="'tasks:execute'"
          type="primary"
          size="large"
          :loading="executing"
          @click="handleExecute"
        >
          <template #icon><icon-play-arrow-fill /></template>
          {{ executing ? '执行中...' : '开始执行' }}
        </a-button>
      </div>
    </a-card>

    <!-- 执行记录 -->
    <a-card class="log-card" :bordered="false">
      <template #title>
        <span class="card-title">执行记录</span>
      </template>
      <div class="log-terminal">
        <div v-if="executionLogs.length === 0" class="log-empty">暂无执行记录</div>
        <div v-else class="log-list">
          <div
            v-for="log in executionLogs"
            :key="log.id"
            class="log-item"
            :class="log.status"
          >
            <div class="log-meta">
              <span class="log-time">{{ log.time }}</span>
              <a-tag :color="log.status === 'success' ? 'green' : log.status === 'error' ? 'red' : 'arcoblue'" size="small">
                {{ log.status === 'success' ? '成功' : log.status === 'error' ? '失败' : '信息' }}
              </a-tag>
              <span v-if="log.host" class="log-host">{{ log.host }}</span>
            </div>
            <pre class="log-output">{{ log.message }}</pre>
          </div>
        </div>
      </div>
    </a-card>

    <!-- 选择主机对话框 -->
    <a-modal
      v-model:visible="showHostDialog"
      title="主机列表"
      :width="960"
      :unmount-on-close="true"
      @ok="confirmHostSelection"
      @cancel="showHostDialog = false"
    >
      <div class="host-dialog-body">
        <div class="host-groups-panel">
          <div class="panel-title">分组列表</div>
          <a-tree
            :data="hostGroups"
            :field-names="{ key: 'id', title: 'name', children: 'children' }"
            default-expand-all
            @select="handleGroupClick"
          />
        </div>
        <div class="host-list-panel">
          <a-input
            v-model="hostSearchKeyword"
            placeholder="输入名称/IP搜索"
            allow-clear
            style="margin-bottom: 12px"
          >
            <template #prefix><icon-search /></template>
          </a-input>
          <a-table
            :data="filteredHosts"
            :row-selection="{ type: 'checkbox', showCheckedAll: true }"
            v-model:selectedKeys="tempSelectedHostIds"
            row-key="id"
            :loading="hostsLoading"
            :scroll="{ y: 360 }"
            :pagination="false"
            @selection-change="handleHostSelectionChange"
          >
            <template #columns>
              <a-table-column title="主机名称" data-index="name" />
              <a-table-column title="IP地址" data-index="ip">
                <template #cell="{ record }">
                  <a-tag size="small">{{ record.ip }}</a-tag>
                </template>
              </a-table-column>
              <a-table-column title="备注信息" data-index="description" />
            </template>
          </a-table>
        </div>
      </div>
    </a-modal>

    <!-- 选择执行模板对话框 -->
    <a-modal
      v-model:visible="showTemplateDialog"
      title="选择执行模板"
      :width="1100"
      :unmount-on-close="true"
      @ok="confirmTemplateSelection"
      @cancel="showTemplateDialog = false"
    >
      <div class="template-filter-bar">
        <a-select v-model="templateFilter.type" placeholder="请选择" allow-clear style="width: 180px">
          <a-option value="system">系统信息</a-option>
          <a-option value="deploy">部署</a-option>
          <a-option value="monitor">监控</a-option>
          <a-option value="backup">备份</a-option>
        </a-select>
        <a-input v-model="templateFilter.name" placeholder="请输入" allow-clear style="width: 260px" />
        <a-button @click="refreshTemplates">
          <template #icon><icon-refresh /></template>
          刷新
        </a-button>
      </div>
      <a-table
        :data="filteredTemplates"
        :loading="templatesLoading"
        row-key="id"
        :pagination="false"
        :row-class="() => 'clickable-row'"
        @row-click="selectTemplate"
      >
        <template #columns>
          <a-table-column title="名称" data-index="name" :width="180" />
          <a-table-column title="类型" data-index="category" :width="120">
            <template #cell="{ record }">
              <a-tag size="small">{{ record.category }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="内容" data-index="content" ellipsis tooltip />
          <a-table-column title="备注" data-index="description" ellipsis tooltip />
        </template>
      </a-table>
    </a-modal>

    <!-- 参数填写对话框 -->
    <a-modal
      v-model:visible="showParamDialog"
      title="填写模板参数"
      :width="560"
      :unmount-on-close="true"
      @ok="applyTemplateWithParams"
      @cancel="showParamDialog = false"
    >
      <a-alert type="info" style="margin-bottom: 16px">
        模板: <strong>{{ currentTemplate?.name }}</strong>
      </a-alert>
      <a-form :model="paramValues" layout="vertical">
        <a-form-item
          v-for="(param, index) in templateParams"
          :key="index"
          :label="param.name"
          :required="param.required"
        >
          <a-input
            v-if="param.type === 'text'"
            v-model="paramValues[param.varName]"
            :placeholder="param.helpText || `请输入${param.name}`"
          />
          <a-input-password
            v-else-if="param.type === 'password'"
            v-model="paramValues[param.varName]"
            :placeholder="param.helpText || `请输入${param.name}`"
          />
          <a-select
            v-else-if="param.type === 'select'"
            v-model="paramValues[param.varName]"
            :placeholder="param.helpText || `请选择${param.name}`"
          >
            <a-option v-for="opt in (param.options || [])" :key="opt" :value="opt">{{ opt }}</a-option>
          </a-select>
          <a-input
            v-else
            v-model="paramValues[param.varName]"
            :placeholder="param.helpText || `请输入${param.name}`"
          />
          <div v-if="param.helpText" class="param-help">{{ param.helpText }}</div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, reactive } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconPlayCircleFill, IconPlus, IconPlayArrowFill,
  IconSearch, IconRefresh
} from '@arco-design/web-vue/es/icon'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'
import { getAgentStatuses } from '@/api/agent'
import { executeTask, getAllJobTemplates } from '@/api/task'

const scriptType = ref('Shell')
const scriptContent = ref('')
const selectedHosts = ref<any[]>([])
const executionMode = ref('auto')
const executing = ref(false)
const executionLogs = ref<any[]>([])

// 主机对话框
const showHostDialog = ref(false)
const hostSearchKeyword = ref('')
const tempSelectedHostIds = ref<number[]>([])
const hostGroups = ref<any[]>([])
const allHosts = ref<any[]>([])
const hostsLoading = ref(false)
const selectedGroupId = ref<number | null>(null)

const filteredHosts = computed(() => {
  let hosts = allHosts.value
  if (selectedGroupId.value !== null) {
    hosts = hosts.filter((host) => host.groupId === selectedGroupId.value)
  }
  if (hostSearchKeyword.value) {
    const keyword = hostSearchKeyword.value.toLowerCase()
    hosts = hosts.filter(
      (host) => host.name.toLowerCase().includes(keyword) || host.ip.includes(keyword)
    )
  }
  return hosts
})

// 模板对话框
const showTemplateDialog = ref(false)
const templateFilter = ref({ type: '', name: '' })
const allTemplates = ref<any[]>([])
const templatesLoading = ref(false)
const selectedTemplate = ref<any>(null)

const filteredTemplates = computed(() => {
  let result = allTemplates.value
  if (templateFilter.value.type) {
    result = result.filter((t) => t.category === templateFilter.value.type)
  }
  if (templateFilter.value.name) {
    const kw = templateFilter.value.name.toLowerCase()
    result = result.filter((t) => t.name.toLowerCase().includes(kw))
  }
  return result
})

// 参数对话框
const showParamDialog = ref(false)
const currentTemplate = ref<any>(null)
const templateParams = ref<any[]>([])
const paramValues = reactive<Record<string, string>>({})

const loadHostGroups = async () => {
  try {
    const data = await getGroupTree()
    hostGroups.value = data || []
  } catch {}
}

const agentStatusMap = ref<Record<number, any>>({})

const loadHostList = async () => {
  hostsLoading.value = true
  try {
    const response = await getHostList({ page: 1, pageSize: 1000 })
    if (Array.isArray(response)) allHosts.value = response
    else if (response.list) allHosts.value = response.list
    else if (response.data) allHosts.value = response.data
    else allHosts.value = []
    // 获取Agent实时状态并合并
    try {
      const statuses = await getAgentStatuses()
      if (Array.isArray(statuses)) {
        const map: Record<number, any> = {}
        for (const s of statuses) {
          map[s.hostId] = s
        }
        agentStatusMap.value = map
        for (const host of allHosts.value) {
          const agentInfo = map[host.id]
          if (agentInfo) {
            host.agentStatus = agentInfo.status
            host.connectionMode = 'agent'
          }
        }
      }
    } catch (e) {}
  } catch { allHosts.value = [] }
  finally { hostsLoading.value = false }
}

const loadTemplates = async () => {
  templatesLoading.value = true
  try {
    const response = await getAllJobTemplates()
    if (Array.isArray(response)) allTemplates.value = response
    else if (response.list) allTemplates.value = response.list
    else if (response.data) allTemplates.value = response.data
    else allTemplates.value = []
  } catch { allTemplates.value = [] }
  finally { templatesLoading.value = false }
}

const handleGroupClick = (keys: string[]) => {
  selectedGroupId.value = keys.length > 0 ? Number(keys[0]) : null
}

const handleHostSelectionChange = (rowKeys: (string | number)[]) => {
  tempSelectedHostIds.value = rowKeys.map(Number)
}

const confirmHostSelection = () => {
  selectedHosts.value = allHosts.value.filter(h => tempSelectedHostIds.value.includes(h.id))
  showHostDialog.value = false
  Message.success(`已选择 ${selectedHosts.value.length} 台主机`)
}

const removeHost = (id: number) => {
  selectedHosts.value = selectedHosts.value.filter(h => h.id !== id)
}

const selectTemplate = (record: any) => {
  selectedTemplate.value = record
  showTemplateDialog.value = false
  let params: any[] = []
  if (record.variables) {
    if (typeof record.variables === 'string' && record.variables !== '[]') {
      try { params = JSON.parse(record.variables) } catch { params = [] }
    } else if (Array.isArray(record.variables)) {
      params = record.variables
    }
  }
  if (params.length > 0) {
    currentTemplate.value = record
    templateParams.value = params
    const values: Record<string, string> = {}
    params.forEach((p: any) => { values[p.varName] = p.defaultValue || '' })
    Object.keys(paramValues).forEach(k => delete paramValues[k])
    Object.assign(paramValues, values)
    showParamDialog.value = true
  } else {
    scriptContent.value = record.content
    Message.success('已应用模板: ' + record.name)
  }
}

const applyTemplateWithParams = () => {
  for (const param of templateParams.value) {
    if (param.required && !paramValues[param.varName]) {
      Message.warning(`请填写参数: ${param.name}`)
      return
    }
  }
  let content = currentTemplate.value.content
  for (const [varName, value] of Object.entries(paramValues)) {
    content = content.replace(new RegExp(`\\{\\{\\s*${varName}\\s*\\}\\}`, 'g'), value)
  }
  scriptContent.value = content
  showParamDialog.value = false
  Message.success('已应用模板: ' + currentTemplate.value.name)
}

const refreshTemplates = async () => {
  await loadTemplates()
  Message.success('刷新成功')
}

const confirmTemplateSelection = () => {
  if (selectedTemplate.value) {
    scriptContent.value = selectedTemplate.value.content
    showTemplateDialog.value = false
    Message.success('已应用模板')
  }
}

const addLog = (message: string, status = 'info', host = '') => {
  const now = new Date()
  const time = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`
  executionLogs.value.unshift({ id: Date.now(), time, message, status, host })
}

const handleExecute = async () => {
  if (selectedHosts.value.length === 0) { Message.warning('请先选择目标主机'); return }
  if (!scriptContent.value.trim()) { Message.warning('请输入执行命令'); return }
  executing.value = true
  addLog(`开始执行任务，目标主机: ${selectedHosts.value.length} 台`, 'info')
  try {
    const response = await executeTask({
      hostIds: selectedHosts.value.map(h => h.id),
      scriptType: scriptType.value,
      content: scriptContent.value,
      executionMode: executionMode.value,
    })
    response.results.forEach((result) => {
      const hostInfo = `${result.hostName} (${result.hostIp})`
      if (result.status === 'success') {
        addLog(result.output || '执行完成，无输出', 'success', hostInfo)
      } else {
        addLog(`错误: ${result.error}\n${result.output || ''}`, 'error', hostInfo)
      }
    })
    const allSuccess = response.results.every(r => r.status === 'success')
    if (allSuccess) Message.success('任务执行成功')
    else Message.warning('部分任务执行失败，请查看执行记录')
  } catch (error: any) {
    addLog('任务执行失败: ' + (error.message || error), 'error')
    Message.error('任务执行失败: ' + (error.message || error))
  } finally {
    executing.value = false
  }
}

onMounted(() => {
  loadHostGroups()
  loadHostList()
})

watch(showTemplateDialog, (v) => { if (v) loadTemplates() })
</script>

<style scoped lang="scss">
.page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-card {
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
  padding: 20px 24px;
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--ops-primary, #165dff) 0%, #4080ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.4;
}

.page-desc {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 2px;
}

.main-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.form-section {
  margin-bottom: 24px;
}

.section-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 12px;

  .required {
    color: var(--ops-danger, #f53f3f);
    margin-right: 4px;
  }
}

.section-body {
  .tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 12px;
  }
}

.execution-hint {
  margin-top: 8px;
  font-size: 12px;
}

.command-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

.code-textarea {
  :deep(textarea) {
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.6;
  }
}

.execute-btn-wrap {
  display: flex;
  justify-content: center;
  padding-top: 8px;
}

.log-card {
  border-radius: var(--ops-border-radius-md, 8px);

  .card-title {
    font-weight: 600;
  }
}

.log-terminal {
  background: #1e1e1e;
  border-radius: 6px;
  min-height: 120px;
  max-height: 500px;
  overflow-y: auto;
  padding: 16px;
}

.log-empty {
  text-align: center;
  color: #666;
  padding: 40px 0;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.log-item {
  background: #2a2a2a;
  border-radius: 6px;
  overflow: hidden;
  border-left: 3px solid #555;

  &.success { border-left-color: var(--ops-success, #00b42a); }
  &.error { border-left-color: var(--ops-danger, #f53f3f); }
  &.info { border-left-color: var(--ops-primary, #165dff); }
}

.log-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: #252525;
  border-bottom: 1px solid #333;
}

.log-time {
  color: #888;
  font-family: 'Consolas', monospace;
  font-size: 12px;
}

.log-host {
  color: var(--ops-warning, #ff7d00);
  font-family: 'Consolas', monospace;
  font-size: 13px;
}

.log-output {
  margin: 0;
  padding: 10px 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 主机对话框 */
.host-dialog-body {
  display: flex;
  gap: 16px;
  height: 460px;
}

.host-groups-panel {
  width: 220px;
  border-right: 1px solid var(--ops-border-color, #e5e6eb);
  padding-right: 16px;
  overflow-y: auto;

  .panel-title {
    font-weight: 600;
    margin-bottom: 12px;
    color: var(--ops-text-primary, #1d2129);
  }
}

.host-list-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 模板筛选 */
.template-filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.clickable-row {
  cursor: pointer;
}

.param-help {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 4px;
}
</style>
