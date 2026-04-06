<template>
  <div class="rule-page">
    <!-- 内容区（移除左侧树） -->
    <div class="rule-content">
      <a-card :bordered="false">
        <template #title>告警规则</template>
        <template #extra>
          <a-space>
            <a-upload accept=".json,.yaml,.yml" :show-file-list="false" :custom-request="doImport">
              <a-button><template #icon><icon-upload /></template>导入</a-button>
            </a-upload>
            <a-dropdown>
              <a-button><template #icon><icon-download /></template>导出</a-button>
              <template #content>
                <a-doption @click="doExport('json')">导出 JSON</a-doption>
                <a-doption @click="doExport('yaml')">导出 YAML</a-doption>
              </template>
            </a-dropdown>
            <a-button type="primary" @click="openCreate"><template #icon><icon-plus /></template>新增规则</a-button>
          </a-space>
        </template>

        <a-row :gutter="12" style="margin-bottom:16px">
          <a-col :span="6"><a-input v-model="keyword" placeholder="搜索规则名称" allow-clear @press-enter="load" /></a-col>
          <a-col :span="4">
            <a-select v-model="filterAssetGroupId" placeholder="业务分组" allow-clear @change="onFilterGroupChange">
              <a-option v-for="g in flatGroups" :key="g.id" :value="g.id">{{ g.name }}</a-option>
            </a-select>
          </a-col>
          <a-col :span="4">
            <a-select v-model="filterRuleGroupId" placeholder="规则分类" allow-clear @change="load">
              <a-option v-for="rg in filterRuleGroups" :key="rg.id" :value="rg.id">{{ rg.name }}</a-option>
            </a-select>
          </a-col>
          <a-col :span="4">
            <a-select v-model="filterEnabled" placeholder="启用状态" allow-clear @change="load">
              <a-option :value="true">已启用</a-option>
              <a-option :value="false">已禁用</a-option>
            </a-select>
          </a-col>
          <a-col :span="3"><a-button type="primary" @click="load">查询</a-button></a-col>
        </a-row>

        <!-- 批量操作栏 -->
        <div v-if="selectedRuleIds.length > 0" class="batch-bar">
          <span class="batch-count">已选 <b>{{ selectedRuleIds.length }}</b> 条</span>
          <a-space>
            <a-button size="small" @click="openBatchGroup">
              <template #icon><icon-apps /></template>批量设置业务分组
            </a-button>
            <a-button size="small" @click="openBatchRuleGroup">
              <template #icon><icon-tag /></template>批量设置规则分类
            </a-button>
            <a-popconfirm content="确认批量删除选中的规则？" @ok="batchDelete">
              <a-button size="small" status="danger">
                <template #icon><icon-delete /></template>批量删除
              </a-button>
            </a-popconfirm>
            <a-button size="small" type="text" @click="selectedRuleIds = [] as number[]">取消选择</a-button>
          </a-space>
        </div>

        <a-table :data="rules" :loading="loading" row-key="id"
          :row-selection="rowSelection"
          v-model:selectedKeys="selectedRuleIds"
          :pagination="{ total, pageSize, current: page, onChange: onPageChange }"
          :bordered="false" stripe>
          <template #columns>
            <a-table-column title="规则名称" :width="180">
              <template #cell="{ record }">
                <div class="rule-name">{{ record.name }}</div>
              </template>
            </a-table-column>
            <a-table-column title="业务分组" :width="110">
              <template #cell="{ record }">
                <span style="font-size:12px;color:var(--ops-text-secondary)">{{ getAssetGroupName(record.assetGroupId) || '—' }}</span>
              </template>
            </a-table-column>
            <a-table-column title="规则分类" :width="110">
              <template #cell="{ record }">
                <a-tag v-if="record.ruleGroupId" size="small" color="arcoblue" style="font-size:11px">{{ record.ruleGroupName || ruleGroupName(record.ruleGroupId) }}</a-tag>
                <span v-else style="color:#c9cdd4">—</span>
              </template>
            </a-table-column>
            <a-table-column title="级别" :width="88">
              <template #cell="{ record }">
                <a-tag :color="sevColor(record.severity)" style="font-weight:600">{{ sevLabel(record.severity) }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="数据源" :width="140">
              <template #cell="{ record }">
                <span style="color:var(--ops-text-secondary);font-size:12px">
                  {{ getRecordDsNames(record) || '—' }}
                </span>
              </template>
            </a-table-column>
            <a-table-column title="PromQL" :ellipsis="true" :width="220">
              <template #cell="{ record }">
                <a-tooltip :content="record.expr">
                  <code class="expr-code">{{ record.expr }}</code>
                </a-tooltip>
              </template>
            </a-table-column>
            <a-table-column title="频率" :width="68">
              <template #cell="{ record }">
                <span style="color:var(--ops-text-tertiary);font-size:12px">{{ record.evalInterval }}s</span>
              </template>
            </a-table-column>
            <a-table-column title="持续" data-index="duration" :width="68" />
            <a-table-column title="启用" :width="68">
              <template #cell="{ record }">
                <a-switch :model-value="record.enabled" @change="() => doToggle(record)" size="small" />
              </template>
            </a-table-column>
            <a-table-column title="上次评估" :width="158">
              <template #cell="{ record }">
                <span style="color:var(--ops-text-tertiary);font-size:12px">{{ record.lastEvalAt ? fmtTime(record.lastEvalAt) : '—' }}</span>
              </template>
            </a-table-column>
            <a-table-column title="操作" :width="160" fixed="right">
              <template #cell="{ record }">
                <a-space :size="4">
                  <a-tooltip content="即时测试">
                    <a-button size="mini" type="text" @click="doTest(record)">
                      <template #icon><icon-thunderbolt /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="编辑">
                    <a-button size="mini" type="text" @click="openEdit(record)">
                      <template #icon><icon-edit /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="克隆为副本">
                    <a-button size="mini" type="text" @click="doClone(record)">
                      <template #icon><icon-copy /></template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="删除">
                    <a-popconfirm content="确认删除该规则？" @ok="remove(record.id)">
                      <a-button size="mini" type="text" status="danger">
                        <template #icon><icon-delete /></template>
                      </a-button>
                    </a-popconfirm>
                  </a-tooltip>
                </a-space>
              </template>
            </a-table-column>
          </template>
        </a-table>
      </a-card>

    <!-- 规则表单弹窗 -->
    <a-modal v-model:visible="modalVisible" :title="form.id ? '编辑规则' : (form.name?.startsWith('副本_') ? '克隆规则' : '新增规则')"
      @ok="save" @cancel="modalVisible=false" width="780px" :mask-closable="false">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="规则名称" required><a-input v-model="form.name" /></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="严重级别" required>
              <a-select v-model="form.severity">
                <a-option value="critical"><a-tag color="red" size="small">紧急 P1</a-tag> 核心业务不可用</a-option>
                <a-option value="major"><a-tag color="orangered" size="small">严重 P2</a-tag> 核心功能降级</a-option>
                <a-option value="minor"><a-tag color="orange" size="small">一般 P3</a-tag> 非核心异常</a-option>
                <a-option value="warning"><a-tag color="blue" size="small">提示 P4</a-tag> 指标偏离/潜在风险</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="业务分组">
              <a-select v-model="form.assetGroupId" allow-clear placeholder="选择业务分组" @change="onFormGroupChange">
                <a-option v-for="g in flatGroups" :key="g.id" :value="g.id">{{ g.name }}</a-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="规则分类">
              <div style="display:flex;gap:8px">
                <a-select v-model="form.ruleGroupId" allow-clear placeholder="选择或新建分类" style="flex:1">
                  <a-option v-for="g in ruleGroups" :key="g.id" :value="g.id">{{ g.name }}</a-option>
                </a-select>
                <a-button @click="ruleGroupModalVisible=true">+ 新建</a-button>
              </div>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="描述"><a-textarea v-model="form.description" :auto-size="{minRows:2}" /></a-form-item>
        <a-form-item label="数据源" required>
          <a-select v-model="formDsIds" multiple allow-clear placeholder="选择一个或多个数据源（多选）">
            <a-option v-for="ds in dataSources" :key="ds.id" :value="ds.id!">{{ ds.name }} ({{ ds.type }})</a-option>
          </a-select>
        </a-form-item>
        <!-- PromQL + 临时测试 -->
        <a-form-item label="PromQL 表达式" required>
          <a-textarea v-model="form.expr" :auto-size="{minRows:3}"
            placeholder="e.g. up == 0" style="font-family:monospace" />
        </a-form-item>
        <div style="margin-bottom:12px">
          <a-button type="outline" :loading="adhocLoading" @click="doAdhocTest">
            <template #icon><icon-play-arrow /></template>
            测试 PromQL
          </a-button>
          <span style="margin-left:8px;color:var(--ops-text-tertiary);font-size:12px">立即执行一次查询，查看返回值（方便设置阈值）</span>
        </div>
        <div v-if="adhocResult !== null" class="adhoc-result-box">
          <div v-if="adhocResult.results && adhocResult.results.length > 0">
            <div class="adhoc-firing"><icon-exclamation-circle-fill style="color:#f53f3f" /> 命中 {{ adhocResult.results.length }} 个时间序列</div>
            <div v-for="(r, i) in adhocResult.results" :key="i" class="adhoc-item">
              <span class="adhoc-val">{{ r.Value?.toFixed(4) }}</span>
              <span class="adhoc-labels">{{ JSON.stringify(r.Labels) }}</span>
            </div>
          </div>
          <a-empty v-else description="无匹配数据" :image-size="40" />
        </div>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="采集频率(秒)">
              <a-input-number v-model="form.evalInterval" :min="15" :default-value="15" style="width:100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="持续触发时长"><a-input v-model="form.duration" placeholder="0s / 5m / 1h" /></a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="恢复通知">
              <a-switch v-model="form.notifyOnResolve" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="告警标题模板">
          <a-input v-model="annotationTitle" placeholder="告警: {{.RuleName}}" />
        </a-form-item>
        <a-form-item label="告警内容模板">
          <a-textarea v-model="annotationDesc" :auto-size="{minRows:2}" placeholder="当前值: {{.Value}}" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 新建规则分类弹窗 -->
    <a-modal v-model:visible="ruleGroupModalVisible" title="新建规则分类" @ok="saveRuleGroup" @cancel="ruleGroupModalVisible=false" width="440px">
      <a-form :model="{ name: newGroupName, desc: newGroupDesc }" layout="vertical">
        <a-form-item label="分类名称" required><a-input v-model="newGroupName" /></a-form-item>
        <a-form-item label="描述"><a-input v-model="newGroupDesc" /></a-form-item>
      </a-form>
    </a-modal>

    <!-- 测试结果弹窗（已保存规则的测试）-->
    <a-modal v-model:visible="testVisible" title="规则测试结果" :footer="false" width="560px">
      <div v-if="testResult === null" style="text-align:center;padding:24px"><a-spin /></div>
      <div v-else-if="testResult.results && testResult.results.length > 0">
        <a-alert type="error" style="margin-bottom:12px">当前命中告警条件，共 {{ testResult.results.length }} 个时间序列</a-alert>
        <div v-for="(r, i) in testResult.results" :key="i" class="adhoc-item">
          <span class="adhoc-val">{{ r.Value?.toFixed(4) }}</span>
          <span class="adhoc-labels">{{ JSON.stringify(r.Labels) }}</span>
        </div>
      </div>
      <a-empty v-else description="当前无告警（表达式未命中任何数据）" style="padding:24px 0" />
    </a-modal>

    <!-- 批量设置业务分组弹窗 -->
    <a-modal v-model:visible="batchGroupVisible" title="批量设置业务分组" @ok="doBatchGroup" @cancel="batchGroupVisible=false" width="400px">
      <a-form :model="{ batchAssetGroupId }" layout="vertical">
        <a-form-item label="业务分组" required>
          <a-select v-model="batchAssetGroupId" placeholder="请选择业务分组" style="width:100%">
            <a-option v-for="g in flatGroups" :key="g.id" :value="g.id">{{ g.name }}</a-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 批量设置规则分类弹窗 -->
    <a-modal v-model:visible="batchRuleGroupVisible" title="批量设置规则分类" @ok="doBatchRuleGroup" @cancel="batchRuleGroupVisible=false" width="400px">
      <a-form :model="{ batchRuleGroupId }" layout="vertical">
        <a-form-item label="规则分类" required>
          <a-select v-model="batchRuleGroupId" placeholder="请选择规则分类" style="width:100%">
            <a-option v-for="rg in ruleGroups" :key="rg.id" :value="rg.id">{{ rg.name }}</a-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  getRules, createRule, updateRule, deleteRule, toggleRule,
  testRule, cloneRule, exportRules, importRules, adhocTestRule,
  getRuleGroups, createRuleGroup, getDataSources,
  type AlertRule, type AlertRuleGroup, type AlertDataSource
} from '@/api/alert'
import { getGroupTree } from '@/api/assetGroup'

const groupTree = ref<any[]>([])
const flatGroups = ref<any[]>([])
const selectedKeys = ref<string[]>([])
const selectedAssetGroupId = ref<number | undefined>()
const ruleGroups = ref<AlertRuleGroup[]>([])  // 用于表单编辑的规则分类
const allRuleGroups = ref<AlertRuleGroup[]>([])  // 所有规则分类（用于列表显示）
const filterAssetGroupId = ref<number | undefined>()
const filterRuleGroupId = ref<number | undefined>()
const filterRuleGroups = ref<AlertRuleGroup[]>([])
const batchGroupVisible = ref(false)
const batchRuleGroupVisible = ref(false)
const batchAssetGroupId = ref<number | undefined>()
const batchRuleGroupId = ref<number | undefined>()
const dataSources = ref<AlertDataSource[]>([])
const rules = ref<AlertRule[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const filterEnabled = ref<boolean | undefined>()
const selectedRuleIds = ref<number[]>([])
const rowSelection = computed(() => {
  const config = {
    type: 'checkbox' as const,
    showCheckedAll: true
  }
  console.log('[告警规则] rowSelection computed 执行', {
    selectedKeys: selectedRuleIds.value,
    config
  })
  return config
})

// 监听选择变化
watch(selectedRuleIds, (newVal) => {
  console.log('[告警规则] selectedRuleIds 变化', { newVal })
})
const modalVisible = ref(false)
const testVisible = ref(false)
const testResult = ref<any>(null)
const ruleGroupModalVisible = ref(false)
const newGroupName = ref('')
const newGroupDesc = ref('')
const form = ref<Partial<AlertRule>>({ severity: 'warning', evalInterval: 15, notifyOnResolve: true, enabled: true })
const annotationTitle = ref('')
const annotationDesc = ref('')
// 多选数据源（number[]）
const formDsIds = ref<number[]>([])
// Ad-hoc 测试
const adhocLoading = ref(false)
const adhocResult = ref<any>(null)

// helpers
const sevColor = (s: string) => ({ critical: 'red', major: 'orangered', minor: 'orange', warning: 'blue', info: 'arcoblue' }[s] || 'gray')
const sevLabel = (s: string) => ({ critical: '紧急P1', major: '严重P2', minor: '一般P3', warning: '提示P4', info: '信息' }[s] || s)
const ruleGroupName = (id?: number) => allRuleGroups.value.find(g => g.id === id)?.name || ''
const dsName = (id?: number) => dataSources.value.find(d => d.id === id)?.name || String(id || '')
const getAssetGroupName = (id?: number) => flatGroups.value.find(g => g.id === id)?.name || ''

const fmtTime = (s?: string) => {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
const parseDsIds = (s?: string): number[] => { try { return JSON.parse(s || '[]') } catch { return [] } }
const getRecordDsNames = (r: AlertRule) => {
  const ids = parseDsIds(r.dataSourceIds)
  if (ids.length > 0) return ids.map(dsName).join(', ')
  if (r.dataSourceId) return dsName(r.dataSourceId)
  return ''
}

const enrichTree = (nodes: any[]): any[] =>
  nodes.map(n => ({ ...n, key: String(n.id), title: n.name, children: n.children?.length ? enrichTree(n.children) : undefined }))

const flattenTree = (nodes: any[], result: any[] = []) => {
  for (const n of nodes) { result.push({ id: n.id, name: n.name }); if (n.children?.length) flattenTree(n.children, result) }
  return result
}

const openBatchGroup = () => { batchAssetGroupId.value = undefined; batchGroupVisible.value = true }
const openBatchRuleGroup = () => { batchRuleGroupId.value = undefined; batchRuleGroupVisible.value = true }

const doBatchGroup = async () => {
  if (!batchAssetGroupId.value) { Message.warning('请选择业务分组'); return }
  console.log('[告警规则] 批量设置业务分组', { selectedIds: selectedRuleIds.value, groupId: batchAssetGroupId.value })
  try {
    await Promise.all(selectedRuleIds.value.map(id => updateRule(Number(id), { assetGroupId: batchAssetGroupId.value })))
    Message.success('批量设置成功')
    batchGroupVisible.value = false
    selectedRuleIds.value = []
    load()
  } catch (err) {
    console.error('[告警规则] 批量设置业务分组失败', err)
    Message.error('批量设置失败')
  }
}

const doBatchRuleGroup = async () => {
  if (!batchRuleGroupId.value) { Message.warning('请选择规则分类'); return }
  console.log('[告警规则] 批量设置规则分类', { selectedIds: selectedRuleIds.value, ruleGroupId: batchRuleGroupId.value })
  try {
    await Promise.all(selectedRuleIds.value.map(id => updateRule(Number(id), { ruleGroupId: batchRuleGroupId.value })))
    Message.success('批量设置成功')
    batchRuleGroupVisible.value = false
    selectedRuleIds.value = []
    load()
  } catch (err) {
    console.error('[告警规则] 批量设置规则分类失败', err)
    Message.error('批量设置失败')
  }
}

const batchDelete = async () => {
  console.log('[告警规则] 批量删除', { selectedIds: selectedRuleIds.value })
  try {
    await Promise.all(selectedRuleIds.value.map(id => deleteRule(Number(id))))
    Message.success('批量删除成功')
    selectedRuleIds.value = []
    load()
  } catch { Message.error('批量删除失败') }
}
const onPageChange = (p: number) => { page.value = p; load() }

const onFilterGroupChange = async (val: number | undefined) => {
  filterRuleGroupId.value = undefined
  if (val) {
    const res = await getRuleGroups(val)
    filterRuleGroups.value = (res as any) || []
  } else {
    filterRuleGroups.value = []
  }
  load()
}

const load = async () => {
  loading.value = true
  try {
    const res = await getRules({
      page: page.value, pageSize: pageSize.value,
      assetGroupId: filterAssetGroupId.value ?? selectedAssetGroupId.value,
      ruleGroupId: filterRuleGroupId.value,
      keyword: keyword.value,
      enabled: filterEnabled.value
    })
    // res is already unwrapped by interceptor (returns res.data)
    // backend returns { total, page, pageSize, data: [...] }
    const d = res as any
    if (Array.isArray(d)) {
      rules.value = d
      total.value = d.length
    } else {
      rules.value = d?.data || []
      total.value = d?.total || 0
    }

    // 调试日志
    console.log('[告警规则] 数据加载完成', {
      count: rules.value.length,
      firstRecord: rules.value[0],
      hasId: rules.value.length > 0 && rules.value[0]?.id !== undefined,
      ids: rules.value.slice(0, 3).map(r => r.id)
    })
  } finally { loading.value = false }
}

const clearGroup = () => {
  selectedKeys.value = []
  selectedAssetGroupId.value = undefined
  ruleGroups.value = []
  load()
}

const onGroupSelect = (keys: (string | number)[], { node }: any) => {
  selectedKeys.value = keys.map(String)
  selectedAssetGroupId.value = node?.id
  loadRuleGroups(node?.id)
  load()
}

const loadRuleGroups = async (groupId?: number) => {
  const id = groupId ?? selectedAssetGroupId.value
  if (!id) { ruleGroups.value = []; return }
  const res = await getRuleGroups(id)
  ruleGroups.value = (res as any) || []
}

const onFormGroupChange = (val: number) => {
  form.value.ruleGroupId = undefined
  loadRuleGroups(val)
}

const openCreate = () => {
  form.value = { severity: 'warning', evalInterval: 15, notifyOnResolve: true, enabled: true, assetGroupId: selectedAssetGroupId.value }
  formDsIds.value = []
  annotationTitle.value = ''
  annotationDesc.value = ''
  adhocResult.value = null
  if (selectedAssetGroupId.value) loadRuleGroups(selectedAssetGroupId.value)
  modalVisible.value = true
}

const openEdit = (row: AlertRule) => {
  form.value = { ...row }
  // 恢复多选数据源
  if (row.dataSourceIds) {
    formDsIds.value = parseDsIds(row.dataSourceIds)
  } else if (row.dataSourceId) {
    formDsIds.value = [row.dataSourceId]
  } else {
    formDsIds.value = []
  }
  try {
    const ann = JSON.parse(row.annotations || '{}')
    annotationTitle.value = ann.title || ''
    annotationDesc.value = ann.description || ''
  } catch { annotationTitle.value = ''; annotationDesc.value = '' }
  adhocResult.value = null
  if (row.assetGroupId) loadRuleGroups(row.assetGroupId)
  modalVisible.value = true
}

const save = async () => {
  form.value.annotations = JSON.stringify({ title: annotationTitle.value, description: annotationDesc.value })
  // 保存多选数据源
  form.value.dataSourceIds = JSON.stringify(formDsIds.value)
  form.value.dataSourceId = formDsIds.value[0] || 0
  if (form.value.evalInterval && form.value.evalInterval < 15) form.value.evalInterval = 15
  try {
    if (form.value.id) { await updateRule(form.value.id, form.value) }
    else { await createRule(form.value) }
    Message.success('保存成功'); modalVisible.value = false; load()
  } catch { Message.error('保存失败') }
}

const saveRuleGroup = async () => {
  if (!newGroupName.value.trim()) { Message.warning('请输入分类名称'); return }
  const groupId = form.value.assetGroupId || selectedAssetGroupId.value
  if (!groupId) { Message.warning('请先选择业务分组'); return }
  try {
    const created = await createRuleGroup({ name: newGroupName.value, description: newGroupDesc.value, assetGroupId: groupId })
    Message.success('创建成功')
    ruleGroupModalVisible.value = false
    newGroupName.value = ''; newGroupDesc.value = ''
    // 重新加载规则分类列表
    await loadRuleGroups(groupId)
    // 同时更新 allRuleGroups
    const allRgRes = await getRuleGroups()
    allRuleGroups.value = (allRgRes as any) || []
    // 设置表单的规则分类ID
    form.value.ruleGroupId = (created as any)?.id
  } catch { Message.error('创建失败') }
}

const remove = async (id: number) => {
  try { await deleteRule(id); Message.success('删除成功'); load() }
  catch { Message.error('删除失败') }
}

const doToggle = async (row: AlertRule) => {
  try { await toggleRule(row.id!); load() }
  catch { Message.error('操作失败') }
}

const doTest = async (row: AlertRule) => {
  testResult.value = null; testVisible.value = true
  try { testResult.value = await testRule(row.id!) as any }
  catch { Message.error('测试失败') }
}

const doAdhocTest = async () => {
  if (!formDsIds.value.length) { Message.warning('请先选择数据源'); return }
  if (!form.value.expr?.trim()) { Message.warning('请输入 PromQL 表达式'); return }
  adhocLoading.value = true
  adhocResult.value = null
  try {
    adhocResult.value = await adhocTestRule({ dataSourceIds: formDsIds.value, expr: form.value.expr }) as any
  } catch { Message.error('查询失败') }
  finally { adhocLoading.value = false }
}

const doClone = (row: AlertRule) => {
  form.value = { ...row, id: undefined, name: `副本_${row.name}`, enabled: false }
  if (row.dataSourceIds) {
    formDsIds.value = parseDsIds(row.dataSourceIds)
  } else if (row.dataSourceId) {
    formDsIds.value = [row.dataSourceId]
  } else {
    formDsIds.value = []
  }
  try {
    const ann = JSON.parse(row.annotations || '{}')
    annotationTitle.value = ann.title || ''
    annotationDesc.value = ann.description || ''
  } catch { annotationTitle.value = ''; annotationDesc.value = '' }
  adhocResult.value = null
  if (row.assetGroupId) loadRuleGroups(row.assetGroupId)
  modalVisible.value = true
}

const doExport = async (fmt: string) => {
  const ids = selectedRuleIds.value.length ? selectedRuleIds.value.map(Number) : undefined
  const res = await exportRules(ids, fmt)
  const blob = new Blob([res as any])
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a'); a.href = url; a.download = `alert_rules.${fmt}`; a.click()
  URL.revokeObjectURL(url)
}

const doImport = async ({ file }: any) => {
  try {
    const res = await importRules(file.file) as any
    Message.success(`导入成功 ${res?.imported || 0} 条`); load()
  } catch { Message.error('导入失败') }
}

onMounted(async () => {
  const [treeRes, dsRes, allRgRes] = await Promise.all([getGroupTree(), getDataSources(), getRuleGroups()])
  const rawTree = (treeRes as any) || []
  groupTree.value = enrichTree(rawTree)
  flatGroups.value = flattenTree(rawTree)
  dataSources.value = (dsRes as any) || []
  allRuleGroups.value = (allRgRes as any) || []  // 保存所有规则分类用于列表显示
  ruleGroups.value = (allRgRes as any) || []  // 初始化表单的规则分类
  load()
})
</script>

<style scoped>
.rule-page { display: block; height: 100%; background: var(--ops-content-bg); }
.rule-content { padding: 20px; overflow: auto; }
.batch-bar {
  display: flex; align-items: center; gap: 12px;
  background: #e8f3ff; border: 1px solid #bedaff; border-radius: 6px;
  padding: 8px 14px; margin-bottom: 12px;
}
.batch-count { font-size: 13px; color: var(--ops-primary); flex-shrink: 0; }
.rule-name { font-weight: 500; color: var(--ops-text-primary); }
.expr-code {
  font-size: 12px; color: #1d7aeb; background: #f0f5ff;
  padding: 2px 6px; border-radius: 3px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  display: inline-block; max-width: 200px;
}
.adhoc-result-box {
  border: 1px solid var(--ops-border-color); border-radius: 6px;
  padding: 12px; background: #f7f8fa; margin-bottom: 12px;
  max-height: 240px; overflow-y: auto;
}
.adhoc-firing { color: #f53f3f; font-size: 13px; font-weight: 600; margin-bottom: 8px; display: flex; align-items: center; gap: 6px; }
.adhoc-item { display: flex; align-items: baseline; gap: 8px; padding: 4px 0; border-bottom: 1px solid #e5e6eb; }
.adhoc-item:last-child { border-bottom: none; }
.adhoc-val { font-size: 15px; font-weight: 700; color: #f53f3f; min-width: 80px; }
.adhoc-labels { font-size: 11px; color: var(--ops-text-tertiary); font-family: monospace; word-break: break-all; }
</style>
