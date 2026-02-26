<template>
  <div class="probe-management-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-storage /></div>
        <div>
          <h2 class="page-title">拨测管理</h2>
          <p class="page-subtitle">管理网络拨测配置，支持 Ping / TCP / UDP / HTTP / WebSocket 探测</p>
        </div>
      </div>
    </div>

    <!-- 搜索与操作 -->
    <div class="filter-bar">
      <a-input v-model="searchForm.keyword" placeholder="搜索名称或目标" allow-clear style="width: 220px;" @press-enter="loadData" />
      <a-select v-model="searchForm.category" placeholder="拨测分类" allow-clear style="width: 130px;" @change="handleCategoryFilter">
        <a-option v-for="c in PROBE_CATEGORIES.filter(c => c.enabled)" :key="c.value" :label="c.label" :value="c.value" />
      </a-select>
      <a-select v-model="searchForm.type" placeholder="拨测类型" allow-clear style="width: 130px;">
        <a-option v-for="t in searchTypeOptions" :key="t" :label="t.toUpperCase()" :value="t" />
      </a-select>
      <a-select v-model="searchForm.status" placeholder="状态" allow-clear style="width: 120px;">
        <a-option label="启用" :value="1" />
        <a-option label="禁用" :value="0" />
      </a-select>
      <a-button type="primary" @click="loadData"><template #icon><icon-search /></template>搜索</a-button>
      <a-button @click="handleReset"><template #icon><icon-refresh /></template>重置</a-button>
      <div style="flex: 1;" />
      <a-upload v-permission="'inspection:probes:import'" :auto-upload="false" :show-file-list="false" accept=".yaml,.yml,.json" @change="handleImportChange">
        <template #upload-button><a-button><template #icon><icon-upload /></template>导入</a-button></template>
      </a-upload>
      <a-button v-permission="'inspection:probes:export'" @click="handleExport"><template #icon><icon-download /></template>导出</a-button>
      <a-button v-permission="'inspection:probes:create'" type="primary" @click="handleCreate"><template #icon><icon-plus /></template>新增拨测</a-button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card"><div class="stat-value">{{ stats.total }}</div><div class="stat-label">总数</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-success);">{{ stats.enabled }}</div><div class="stat-label">启用</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-danger);">{{ stats.disabled }}</div><div class="stat-label">禁用</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-primary);">{{ stats.ping }}</div><div class="stat-label">Ping</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-warning);">{{ stats.tcp }}</div><div class="stat-label">TCP</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-info);">{{ stats.udp }}</div><div class="stat-label">UDP</div></div>
      <div class="stat-card"><div class="stat-value" style="color: #165dff;">{{ stats.http }}</div><div class="stat-label">HTTP(S)</div></div>
      <div class="stat-card"><div class="stat-value" style="color: #722ed1;">{{ stats.websocket }}</div><div class="stat-label">WebSocket</div></div>
    </div>
    <!-- 数据表格 -->
    <a-table :data="tableData" :loading="loading" :bordered="{ cell: true }" stripe :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50] }" @page-change="(p: number) => { pagination.page = p; loadData() }" @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }">
      <template #columns>
        <a-table-column title="名称" data-index="name" :width="140" />
        <a-table-column title="分类" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag size="small">{{ CATEGORY_LABEL_MAP[record.category] || record.category }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="类型" :width="90" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.type === 'ping' ? 'arcoblue' : record.type === 'tcp' ? 'orangered' : 'gray'">{{ record.type.toUpperCase() }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="目标" data-index="target" :width="160" ellipsis tooltip />
        <a-table-column title="端口" :width="80" align="center">
          <template #cell="{ record }">{{ record.type === 'ping' ? '-' : record.port }}</template>
        </a-table-column>
        <a-table-column title="超时(s)" data-index="timeout" :width="90" align="center" />
        <a-table-column title="标签" data-index="tags" :width="140" ellipsis tooltip />
        <a-table-column title="执行方式" :width="90" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.execMode === 'agent' ? 'purple' : record.execMode === 'proxy' ? 'orangered' : 'gray'">{{ record.execMode === 'agent' ? 'Agent' : record.execMode === 'proxy' ? '代理' : '普通' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="状态" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.status === 1 ? 'green' : 'red'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="200" fixed="right" align="center">
          <template #cell="{ record }">
            <a-button v-permission="'inspection:probes:execute'" type="text" size="small" @click="handleRunOnce(record)">执行</a-button>
            <a-button v-permission="'inspection:probes:update'" type="text" size="small" @click="handleEdit(record)">编辑</a-button>
            <a-button v-permission="'inspection:probes:delete'" type="text" size="small" status="danger" @click="handleDelete(record)">删除</a-button>
          </template>
        </a-table-column>
      </template>
    </a-table>

    <!-- 新建/编辑对话框 -->
    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑拨测' : '新增拨测'" :width="800" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="名称" field="name"><a-input v-model="formData.name" placeholder="请输入拨测名称" /></a-form-item>
        <a-form-item label="分类" field="category">
          <a-radio-group v-model="formData.category" type="button" @change="handleCategoryChange">
            <a-radio v-for="c in PROBE_CATEGORIES" :key="c.value" :value="c.value" :disabled="!c.enabled">{{ c.label }}</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="类型" field="type">
          <a-select v-model="formData.type" placeholder="选择拨测类型">
            <a-option v-for="t in availableTypes" :key="t" :label="t.toUpperCase()" :value="t" />
          </a-select>
        </a-form-item>
        <a-form-item v-if="formData.category !== 'application'" label="目标地址" field="target"><a-input v-model="formData.target" placeholder="IP 或域名" /></a-form-item>
        <a-form-item v-if="formData.type !== 'ping' && formData.category !== 'application'" label="端口" field="port"><a-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%;" /></a-form-item>
        <!-- 应用服务专属字段 -->
        <template v-if="formData.category === 'application'">
          <a-form-item label="HTTP方法">
            <a-select v-model="formData.method" style="width: 100%;">
              <a-option v-for="m in ['GET','POST','PUT','DELETE','PATCH']" :key="m" :value="m">{{ m }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="URL" field="url"><VariableInput v-model="formData.url" placeholder="https://example.com/api/health（输入 / 插入变量）" :variables="variableOptions" /></a-form-item>
          <a-form-item label="Content-Type">
            <a-select v-model="formData.contentType" allow-clear style="width: 100%;">
              <a-option value="application/json">application/json</a-option>
              <a-option value="application/x-www-form-urlencoded">application/x-www-form-urlencoded</a-option>
              <a-option value="text/plain">text/plain</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="Headers">
            <div style="width: 100%;">
              <div v-for="(h, i) in appHeaders" :key="i" style="display: flex; gap: 8px; margin-bottom: 4px;">
                <a-input v-model="h.key" placeholder="Key" style="flex: 1;" />
                <VariableInput v-model="h.value" placeholder="Value（输入 / 插入变量）" :variables="variableOptions" style="flex: 1;" />
                <a-button type="text" status="danger" size="small" @click="appHeaders.splice(i, 1)"><icon-minus /></a-button>
              </div>
              <a-button type="text" size="small" @click="appHeaders.push({ key: '', value: '' })"><icon-plus /> 添加Header</a-button>
            </div>
          </a-form-item>
          <a-form-item label="Query参数">
            <div style="width: 100%;">
              <div v-for="(p, i) in appParams" :key="i" style="display: flex; gap: 8px; margin-bottom: 4px;">
                <a-input v-model="p.key" placeholder="Key" style="flex: 1;" />
                <a-input v-model="p.value" placeholder="Value" style="flex: 1;" />
                <a-button type="text" status="danger" size="small" @click="appParams.splice(i, 1)"><icon-minus /></a-button>
              </div>
              <a-button type="text" size="small" @click="appParams.push({ key: '', value: '' })"><icon-plus /> 添加参数</a-button>
            </div>
          </a-form-item>
          <a-form-item v-if="formData.method !== 'GET'" label="Body">
            <VariableInput v-model="formData.body" placeholder="请求体（输入 / 插入变量）" :variables="variableOptions" :multiline="true" />
          </a-form-item>
          <a-form-item v-if="formData.execMode === 'proxy'" label="代理URL"><a-input v-model="formData.proxyUrl" placeholder="http://proxy:8080" /></a-form-item>
          <a-form-item label="断言配置">
            <div style="width: 100%;">
              <div v-for="(a, i) in appAssertions" :key="i" style="display: flex; gap: 6px; margin-bottom: 6px; flex-wrap: wrap;">
                <a-input v-model="a.name" placeholder="名称" style="width: 100px;" />
                <a-select v-model="a.source" style="width: 90px;"><a-option value="body">Body</a-option><a-option value="header">Header</a-option></a-select>
                <a-input v-model="a.path" placeholder="路径 $.data.id" style="width: 130px;" />
                <a-select v-model="a.condition" style="width: 110px;">
                  <a-option v-for="c in assertConditions" :key="c.value" :value="c.value">{{ c.label }}</a-option>
                </a-select>
                <a-input v-model="a.value" placeholder="期望值" style="width: 100px;" />
                <a-button type="text" status="danger" size="small" @click="appAssertions.splice(i, 1)"><icon-minus /></a-button>
              </div>
              <a-button type="text" size="small" @click="appAssertions.push({ name: '', source: 'body', path: '', condition: '==', value: '' })"><icon-plus /> 添加断言</a-button>
            </div>
          </a-form-item>
        </template>
        <a-form-item label="超时(秒)"><a-input-number v-model="formData.timeout" :min="1" :max="60" /></a-form-item>
        <a-form-item v-if="formData.category === 'network' && formData.type === 'ping'" label="Ping次数"><a-input-number v-model="formData.count" :min="1" :max="100" /></a-form-item>
        <a-form-item v-if="formData.category === 'network' && formData.type === 'ping'" label="包大小"><a-input-number v-model="formData.packetSize" :min="16" :max="65500" /></a-form-item>
        <a-form-item label="业务分组">
          <a-select v-model="selectedGroupIds" multiple placeholder="选择业务分组（可多选）" allow-clear allow-search style="width: 100%;">
            <a-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="标签"><a-input v-model="formData.tags" placeholder="region=cn-east,env=prod" /></a-form-item>
        <a-form-item label="执行方式">
          <a-radio-group v-model="formData.execMode" type="button">
            <a-radio value="local">普通拨测</a-radio>
            <a-radio value="agent">Agent拨测</a-radio>
            <a-radio v-if="formData.category === 'application'" value="proxy">代理拨测</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="formData.execMode === 'agent'" label="Agent主机">
          <template #label>
            Agent主机
            <a-tooltip position="right">
              <template #content>
                <div style="max-width: 280px;">
                  Agent拨测会从选中的主机中随机挑选一台在线Agent发起探测，实现从不同网络环境进行拨测。<br/><br/>
                  使用前提：目标主机需已部署并启动Agent。多台Agent可提供高可用和多视角探测能力。
                </div>
              </template>
              <icon-question-circle style="margin-left: 4px; color: var(--ops-text-tertiary); cursor: help;" />
            </a-tooltip>
          </template>
          <a-select v-model="selectedAgentHostIds" multiple allow-search placeholder="选择Agent主机（可多选）" style="width: 100%;">
            <a-option v-for="h in filteredHostOptions" :key="h.id" :label="h.name || h.ip" :value="h.id">
              {{ h.name || h.ip }} <span :style="{ float: 'right', fontSize: '12px', color: h.agentOnline ? 'var(--ops-success)' : 'var(--ops-text-tertiary)' }">{{ h.agentOnline ? '在线' : '离线' }}</span>
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="失败重试">
          <a-input-number v-model="formData.retryCount" :min="0" :max="5" />
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">拨测失败时自动重试的次数（0-5）</span>
        </a-form-item>
        <a-form-item label="描述"><a-textarea v-model="formData.description" :max-length="200" :auto-size="{ minRows: 2 }" /></a-form-item>
        <a-form-item label="状态">
          <a-radio-group v-model="formData.status"><a-radio :value="1">启用</a-radio><a-radio :value="0">禁用</a-radio></a-radio-group>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" :loading="submitting" @click="handleSubmit">确定</a-button>
      </template>
    </a-modal>

    <!-- 执行结果对话框 -->
    <a-modal v-model:visible="resultDialogVisible" title="拨测结果" :width="750" :footer="false">
      <a-spin :loading="runLoading" style="width: 100%;">
        <template v-if="runResult">
          <a-descriptions :column="1" bordered>
            <a-descriptions-item label="状态"><a-tag :color="runResult.Success ? 'green' : 'red'">{{ runResult.Success ? '成功' : '失败' }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="延迟">{{ runResult.Latency?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.PacketLoss !== undefined && runResult.PacketLoss > 0" label="丢包率">{{ (runResult.PacketLoss * 100).toFixed(1) }}%</a-descriptions-item>
            <a-descriptions-item v-if="runResult.PingRttAvg" label="平均RTT">{{ runResult.PingRttAvg?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.TCPConnectTime" label="TCP连接">{{ runResult.TCPConnectTime?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.HTTPStatusCode" label="HTTP状态码"><a-tag :color="runResult.HTTPStatusCode < 400 ? 'green' : 'red'">{{ runResult.HTTPStatusCode }}</a-tag></a-descriptions-item>
            <a-descriptions-item v-if="runResult.HTTPResponseTime" label="HTTP响应耗时">{{ runResult.HTTPResponseTime?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.HTTPContentLength" label="响应大小">{{ runResult.HTTPContentLength }} bytes</a-descriptions-item>
            <a-descriptions-item v-if="runResult.AssertionResults?.length" label="断言结果">
              <div v-for="(ar, i) in runResult.AssertionResults" :key="i" style="margin-bottom: 4px;">
                <a-tag size="small" :color="ar.success ? 'green' : 'red'">{{ ar.name || `断言${i+1}` }}</a-tag>
                <span v-if="!ar.success" style="font-size: 12px; color: var(--ops-danger); margin-left: 4px;">{{ ar.error }}</span>
              </div>
            </a-descriptions-item>
            <a-descriptions-item v-if="runResult.Error" label="错误">{{ runResult.Error }}</a-descriptions-item>
            <a-descriptions-item v-if="runResult.agentHostId > 0" label="Agent主机">主机 #{{ runResult.agentHostId }}</a-descriptions-item>
            <a-descriptions-item v-if="runResult.retryAttempt > 0" label="重试次数">{{ runResult.retryAttempt }} 次</a-descriptions-item>
          </a-descriptions>
          <!-- 请求信息 -->
          <template v-if="currentRecord">
            <div class="result-section-title">请求信息</div>
            <a-descriptions :column="1" bordered size="small">
              <!-- 网络/四层拨测：显示目标地址和端口 -->
              <template v-if="currentRecord.category !== 'application'">
                <a-descriptions-item label="目标地址">{{ currentRecord.target || '-' }}</a-descriptions-item>
                <a-descriptions-item v-if="currentRecord.type !== 'ping'" label="端口">{{ currentRecord.port || '-' }}</a-descriptions-item>
                <a-descriptions-item label="拨测类型">{{ (currentRecord.type || '').toUpperCase() }}</a-descriptions-item>
              </template>
              <!-- 应用服务拨测：显示URL、方法、Headers、Params、Body -->
              <template v-else>
                <a-descriptions-item label="请求URL">{{ currentRecord.url || currentRecord.target || '-' }}</a-descriptions-item>
                <a-descriptions-item v-if="currentRecord.method" label="请求方法">{{ currentRecord.method }}</a-descriptions-item>
                <a-descriptions-item v-if="parseJsonSafe(currentRecord.headers)" label="请求Headers">
                  <div v-for="(val, key) in parseJsonSafe(currentRecord.headers)" :key="key" class="kv-row">
                    <span class="kv-key">{{ key }}:</span>
                    <template v-if="isLongText(val)">
                      <span class="kv-val-truncated">{{ truncateText(String(val)) }}</span>
                      <a-button type="text" size="mini" @click="showLongText(String(val), `Header: ${key}`)"><icon-expand /></a-button>
                    </template>
                    <span v-else class="kv-val">{{ val }}</span>
                  </div>
                </a-descriptions-item>
                <a-descriptions-item v-if="parseJsonSafe(currentRecord.params)" label="请求Params">
                  <div v-for="(val, key) in parseJsonSafe(currentRecord.params)" :key="key" class="kv-row">
                    <span class="kv-key">{{ key }}:</span>
                    <template v-if="isLongText(String(val))">
                      <span class="kv-val-truncated">{{ truncateText(String(val)) }}</span>
                      <a-button type="text" size="mini" @click="showLongText(String(val), `Param: ${key}`)"><icon-expand /></a-button>
                    </template>
                    <span v-else class="kv-val">{{ val }}</span>
                  </div>
                </a-descriptions-item>
                <a-descriptions-item v-if="currentRecord.body" label="请求Body">
                  <pre class="response-body-pre" v-html="formatJsonWithCollapse(currentRecord.body)" @click="handleResultBodyClick" />
                </a-descriptions-item>
              </template>
            </a-descriptions>
          </template>
          <!-- 响应信息 -->
          <template v-if="runResult.ResponseHeaders || runResult.ResponseBody">
            <div class="result-section-title">响应信息</div>
            <a-descriptions :column="1" bordered size="small">
              <a-descriptions-item v-if="runResult.ResponseHeaders && Object.keys(runResult.ResponseHeaders).length" label="响应Headers">
                <div v-for="(val, key) in runResult.ResponseHeaders" :key="key" class="kv-row">
                  <span class="kv-key">{{ key }}:</span>
                  <template v-if="isLongText(Array.isArray(val) ? val.join(', ') : String(val))">
                    <span class="kv-val-truncated">{{ truncateText(Array.isArray(val) ? val.join(', ') : String(val)) }}</span>
                    <a-button type="text" size="mini" @click="showLongText(Array.isArray(val) ? val.join(', ') : String(val), `Header: ${key}`)"><icon-expand /></a-button>
                  </template>
                  <span v-else class="kv-val">{{ Array.isArray(val) ? val.join(', ') : val }}</span>
                </div>
              </a-descriptions-item>
              <a-descriptions-item v-if="runResult.ResponseBody" label="响应Body">
                <div>
                  <div style="display: flex; justify-content: flex-end; margin-bottom: 4px;">
                    <a-button type="text" size="mini" @click="copyText(runResult.ResponseBody)"><icon-copy /> 复制</a-button>
                  </div>
                  <pre class="response-body-pre" v-html="formatJsonWithCollapse(runResult.ResponseBody)" @click="handleResultBodyClick" />
                </div>
              </a-descriptions-item>
            </a-descriptions>
          </template>
        </template>
      </a-spin>
    </a-modal>

    <!-- 长文本查看弹窗 -->
    <a-modal v-model:visible="longTextVisible" :title="longTextTitle" :width="700" :footer="false">
      <div style="display: flex; justify-content: flex-end; margin-bottom: 8px;">
        <a-button size="small" @click="copyText(longTextContent)"><template #icon><icon-copy /></template>复制</a-button>
      </div>
      <pre class="long-text-pre">{{ longTextContent }}</pre>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconStorage, IconSearch, IconRefresh, IconPlus, IconUpload, IconDownload, IconQuestionCircle, IconMinus, IconExpand, IconCopy } from '@arco-design/web-vue/es/icon'
import { getProbeList, createProbe, updateProbe, deleteProbe, importProbes, exportProbes, runProbeOnce, PROBE_CATEGORIES, CATEGORY_TYPE_MAP, CATEGORY_LABEL_MAP, getVariableList } from '@/api/networkProbe'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'
import { getAgentStatuses } from '@/api/agent'
import VariableInput from '@/components/VariableInput.vue'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<any[]>([])
const resultDialogVisible = ref(false)
const runLoading = ref(false)
const runResult = ref<any>(null)
const currentRecord = ref<any>(null)
const longTextVisible = ref(false)
const longTextContent = ref('')
const longTextTitle = ref('详情')
const groupOptions = ref<any[]>([])
const hostOptions = ref<any[]>([])
const variableOptions = ref<{ name: string; description?: string }[]>([])
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const searchForm = reactive({ keyword: '', type: '', category: '', status: undefined as number | undefined })

const defaultForm = () => ({
  id: 0, name: '', category: 'network', type: 'ping', target: '', port: 80, groupId: 0, groupIds: '',
  timeout: 5, count: 4, packetSize: 64, description: '', tags: '', status: 1,
  execMode: 'local', agentHostIds: '', retryCount: 0,
  method: 'GET', url: '', headers: '', params: '', body: '', proxyUrl: '', assertions: '', contentType: ''
})
const formData = reactive(defaultForm())

const appHeaders = reactive<{ key: string; value: string }[]>([])
const appParams = reactive<{ key: string; value: string }[]>([])
const appAssertions = reactive<{ name: string; source: string; path: string; condition: string; value: string }[]>([])
const assertConditions = [
  { value: '==', label: '等于' }, { value: '>', label: '大于' }, { value: '>=', label: '大于等于' },
  { value: '<', label: '小于' }, { value: '<=', label: '小于等于' },
  { value: 'contains', label: '包含' }, { value: 'notcontains', label: '不包含' },
  { value: 'regexp', label: '正则匹配' }, { value: 'notregexp', label: '正则不匹配' },
]

const formRules = computed(() => ({
  name: [{ required: true, message: '请输入名称' }],
  type: [{ required: true, message: '请选择类型' }],
  target: [{ required: formData.category !== 'application', message: '请输入目标地址' }],
}))

const stats = computed(() => {
  const all = tableData.value
  return {
    total: pagination.total,
    enabled: all.filter((r: any) => r.status === 1).length,
    disabled: all.filter((r: any) => r.status === 0).length,
    ping: all.filter((r: any) => r.type === 'ping').length,
    tcp: all.filter((r: any) => r.type === 'tcp').length,
    udp: all.filter((r: any) => r.type === 'udp').length,
    http: all.filter((r: any) => r.type === 'http' || r.type === 'https').length,
    websocket: all.filter((r: any) => r.type === 'websocket').length,
  }
})

const loadData = async () => {
  loading.value = true
  try {
    const res = await getProbeList({
      page: pagination.page, page_size: pagination.pageSize,
      keyword: searchForm.keyword, type: searchForm.type, category: searchForm.category, status: searchForm.status
    })
    tableData.value = res.data || []; pagination.total = res.total || 0
  } catch {} finally { loading.value = false }
}

const handleReset = () => {
  searchForm.keyword = ''; searchForm.type = ''; searchForm.category = ''; searchForm.status = undefined
  pagination.page = 1; loadData()
}

const availableTypes = computed(() => CATEGORY_TYPE_MAP[formData.category] || ['ping', 'tcp', 'udp'])
const searchTypeOptions = computed(() => {
  if (searchForm.category) return CATEGORY_TYPE_MAP[searchForm.category] || []
  const all: string[] = []
  for (const cat of PROBE_CATEGORIES.filter(c => c.enabled)) { all.push(...(CATEGORY_TYPE_MAP[cat.value] || [])) }
  return all
})
const handleCategoryChange = () => { const types = availableTypes.value; if (!types.includes(formData.type)) formData.type = types[0] || '' }
const handleCategoryFilter = () => { pagination.page = 1; loadData() }

const loadGroups = async () => {
  try { const res = await getGroupTree(); groupOptions.value = flattenGroups(res.data || res || []) } catch {}
}
const flattenGroups = (tree: any[], result: any[] = []): any[] => {
  for (const node of tree) { result.push({ id: node.id, name: node.name }); if (node.children?.length) flattenGroups(node.children, result) }
  return result
}

const selectedAgentHostIds = computed({
  get: () => formData.agentHostIds ? formData.agentHostIds.split(',').filter(Boolean).map(Number) : [],
  set: (val: number[]) => { formData.agentHostIds = val.join(',') }
})

const selectedGroupIds = computed({
  get: () => formData.groupIds ? formData.groupIds.split(',').filter(Boolean).map(Number) : [],
  set: (val: number[]) => { formData.groupIds = val.join(',') }
})

const filteredHostOptions = computed(() => {
  const gids = selectedGroupIds.value
  if (gids.length === 0) return hostOptions.value
  return hostOptions.value.filter((h: any) => gids.includes(h.groupId))
})

// Clear agent hosts that are no longer in the filtered list when groups change
watch(selectedGroupIds, () => {
  const validIds = new Set(filteredHostOptions.value.map((h: any) => h.id))
  const current = selectedAgentHostIds.value
  const filtered = current.filter(id => validIds.has(id))
  if (filtered.length !== current.length) {
    selectedAgentHostIds.value = filtered
  }
})

const loadHosts = async () => {
  try {
    const res = await getHostList({ page: 1, page_size: 1000 })
    const hosts = res.list || []
    try {
      const data = await getAgentStatuses()
      const list = Array.isArray(data) ? data : []
      const statusMap: Record<number, boolean> = {}
      for (const item of list) {
        statusMap[item.hostId] = item.status === 'online'
      }
      for (const h of hosts) {
        h.agentOnline = !!statusMap[h.id]
      }
    } catch {}
    hostOptions.value = hosts
  } catch {}
}

const loadVariables = async () => {
  try {
    const res = await getVariableList({ page: 1, page_size: 100 })
    const list = res?.list || res?.data || []
    variableOptions.value = list.map((v: any) => ({ name: v.name, description: v.description || '' }))
  } catch {}
}

const handleCreate = () => {
  isEdit.value = false; Object.assign(formData, defaultForm())
  appHeaders.length = 0; appParams.length = 0; appAssertions.length = 0
  loadGroups(); loadHosts(); loadVariables(); dialogVisible.value = true
}
const handleEdit = (row: any) => {
  isEdit.value = true
  Object.assign(formData, {
    ...row, category: row.category || 'network', execMode: row.execMode || 'local',
    agentHostIds: row.agentHostIds || '', retryCount: row.retryCount || 0, groupIds: row.groupIds || '',
    method: row.method || 'GET', url: row.url || '', body: row.body || '',
    proxyUrl: row.proxyUrl || '', contentType: row.contentType || '',
    headers: row.headers || '', params: row.params || '', assertions: row.assertions || ''
  })
  // Parse JSON fields into reactive arrays
  appHeaders.length = 0
  if (row.headers) { try { const h = JSON.parse(row.headers); for (const [k, v] of Object.entries(h)) appHeaders.push({ key: k, value: v as string }) } catch {} }
  appParams.length = 0
  if (row.params) { try { const p = JSON.parse(row.params); for (const [k, v] of Object.entries(p)) appParams.push({ key: k, value: v as string }) } catch {} }
  appAssertions.length = 0
  if (row.assertions) { try { const a = JSON.parse(row.assertions); for (const item of a) appAssertions.push({ ...item }) } catch {} }
  loadGroups(); loadHosts(); loadVariables(); dialogVisible.value = true
}

const handleDelete = (row: any) => {
  Modal.warning({ title: '提示', content: '确定删除该拨测配置？', hideCancel: false, onOk: async () => { await deleteProbe(row.id); Message.success('删除成功'); loadData() } })
}

const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return
  submitting.value = true
  try {
    const data = { ...formData }
    // Serialize app-specific fields
    if (data.category === 'application') {
      const headersObj: Record<string, string> = {}
      for (const h of appHeaders) { if (h.key) headersObj[h.key] = h.value }
      data.headers = Object.keys(headersObj).length ? JSON.stringify(headersObj) : ''
      const paramsObj: Record<string, string> = {}
      for (const p of appParams) { if (p.key) paramsObj[p.key] = p.value }
      data.params = Object.keys(paramsObj).length ? JSON.stringify(paramsObj) : ''
      data.assertions = appAssertions.length ? JSON.stringify(appAssertions.filter(a => a.name || a.path)) : ''
      if (!data.url) data.url = data.target
    }
    if (isEdit.value) { await updateProbe(data.id, data); Message.success('更新成功') }
    else { await createProbe(data); Message.success('创建成功') }
    dialogVisible.value = false; loadData()
  } catch {} finally { submitting.value = false }
}

const handleRunOnce = async (row: any) => {
  currentRecord.value = row
  resultDialogVisible.value = true; runLoading.value = true; runResult.value = null
  try { runResult.value = await runProbeOnce(row.id) } catch {} finally { runLoading.value = false }
}

const isLongText = (val: any) => typeof val === 'string' && val.length > 1000
const truncateText = (val: string, max = 80) => val.length > max ? val.slice(0, max) + '...' : val
const showLongText = (content: string, title = '详情') => {
  longTextTitle.value = title; longTextContent.value = content; longTextVisible.value = true
}
const copyText = async (text: string) => {
  try { await navigator.clipboard.writeText(text); Message.success('已复制') } catch { Message.error('复制失败') }
}
const parseJsonSafe = (str: string): Record<string, string> | null => {
  if (!str) return null
  try { return JSON.parse(str) } catch { return null }
}
const formatResponseBody = (body: string): string => {
  if (!body) return ''
  try { return JSON.stringify(JSON.parse(body), null, 2) } catch { return body }
}

// Collapse long string values in JSON for result display (same visual as VariableInput collapse pills)
const RESULT_COLLAPSE_THRESHOLD = 500
const resultCollapsedMap = new Map<string, string>()
let resultCollapseIdCounter = 0

const escapeHtml = (str: string) => str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')

const formatJsonWithCollapse = (jsonStr: string): string => {
  if (!jsonStr) return ''
  try {
    const parsed = JSON.parse(jsonStr)
    const placeholders: Record<string, string> = {}
    const walk = (obj: any): any => {
      if (typeof obj === 'string' && obj.length > RESULT_COLLAPSE_THRESHOLD) {
        const id = `rc_${resultCollapseIdCounter++}`
        resultCollapsedMap.set(id, obj)
        const label = obj.length > 1024 ? `${(obj.length / 1024).toFixed(1)}KB` : `${obj.length}B`
        placeholders[id] = label
        return `__COLLAPSE_${id}__`
      }
      if (Array.isArray(obj)) return obj.map(walk)
      if (obj && typeof obj === 'object') {
        const result: Record<string, any> = {}
        for (const [k, v] of Object.entries(obj)) result[k] = walk(v)
        return result
      }
      return obj
    }
    const walked = walk(parsed)
    const formatted = JSON.stringify(walked, null, 2)
    // Split around placeholders, escape text parts, insert pill HTML
    const regex = /"__COLLAPSE_(rc_\d+)__"/g
    let html = ''
    let lastIndex = 0
    let m: RegExpExecArray | null
    while ((m = regex.exec(formatted)) !== null) {
      html += escapeHtml(formatted.slice(lastIndex, m.index))
      const id = m[1]
      const label = placeholders[id] || ''
      html += `<span class="result-collapse-pill" data-collapse-id="${id}" title="点击查看完整内容"><svg viewBox="0 0 48 48" width="12" height="12" fill="currentColor" style="flex-shrink:0"><path d="M6 9a3 3 0 0 1 3-3h12l4 4h14a3 3 0 0 1 3 3v24a3 3 0 0 1-3 3H9a3 3 0 0 1-3-3V9z"/></svg>${label}</span>`
      lastIndex = m.index + m[0].length
    }
    html += escapeHtml(formatted.slice(lastIndex))
    return html
  } catch {
    return escapeHtml(jsonStr)
  }
}

const handleResultBodyClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  const pill = target.closest('.result-collapse-pill') as HTMLElement | null
  if (!pill) return
  const id = pill.dataset.collapseId || ''
  const content = resultCollapsedMap.get(id) || ''
  showLongText(content, '查看完整内容')
}

const handleImportChange = async (fileList: any[]) => {
  const file = fileList[fileList.length - 1]?.file
  if (!file) return
  try { await importProbes(file); Message.success('导入成功'); loadData() } catch {}
}

const handleExport = async () => {
  try {
    const blob = await exportProbes('yaml') as any
    const url = window.URL.createObjectURL(new Blob([blob]))
    const a = document.createElement('a'); a.href = url; a.download = 'probe_configs.yaml'
    a.click(); window.URL.revokeObjectURL(url)
  } catch {}
}

onMounted(() => { loadData() })
</script>

<style scoped>
.probe-management-container { padding: 0; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 36px; height: 36px; border-radius: 8px; background: var(--ops-primary); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 18px; }
.page-title { margin: 0; font-size: 17px; font-weight: 600; color: var(--ops-text-primary); }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: var(--ops-text-tertiary); }
.filter-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; }
.stats-row { display: flex; gap: 12px; margin-bottom: 16px; }
.stat-card { flex: 1; padding: 14px; background: #fff; border: 1px solid var(--ops-border-color); border-radius: var(--ops-border-radius-md); text-align: center; }
.stat-card .stat-value { font-size: 22px; font-weight: 600; color: var(--ops-text-primary); }
.stat-card .stat-label { font-size: 12px; color: var(--ops-text-tertiary); margin-top: 4px; }
.result-section-title { font-size: 14px; font-weight: 600; color: var(--ops-text-primary); margin: 16px 0 8px; padding-left: 8px; border-left: 3px solid var(--ops-primary); }
.kv-row { display: flex; align-items: center; gap: 4px; margin-bottom: 2px; font-size: 13px; line-height: 20px; }
.kv-key { color: var(--ops-text-secondary); font-weight: 500; flex-shrink: 0; }
.kv-val { color: var(--ops-text-primary); word-break: break-all; }
.kv-val-truncated { color: var(--ops-text-tertiary); font-style: italic; word-break: break-all; }
.response-body-pre { margin: 0; padding: 8px; background: var(--color-fill-1, #f7f8fa); border-radius: 4px; font-size: 12px; line-height: 1.5; max-height: 300px; overflow: auto; white-space: pre-wrap; word-break: break-all; }
.response-body-pre :deep(.result-collapse-pill) {
  display: inline-flex; align-items: center; gap: 3px;
  background: #fff3e0; color: #d46b08; border: 1px solid #ffd591;
  border-radius: 4px; padding: 0 6px; font-size: 12px; line-height: 20px;
  font-weight: 500; white-space: nowrap; cursor: pointer; transition: background 0.15s;
  vertical-align: middle; margin: 0 1px;
}
.response-body-pre :deep(.result-collapse-pill:hover) { background: #ffe7ba; }
.long-text-pre { margin: 0; padding: 12px; background: var(--color-fill-1, #f7f8fa); border-radius: 4px; font-size: 13px; line-height: 1.6; max-height: 500px; overflow: auto; white-space: pre-wrap; word-break: break-all; }
</style>
