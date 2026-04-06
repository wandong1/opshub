<template>
  <div class="page-container">
    <!-- 图表时间范围筛选 -->
    <a-card :bordered="false" style="margin-bottom:16px;padding:4px 0">
      <a-space>
        <span style="font-size:13px;color:var(--ops-text-secondary);font-weight:500">统计时间范围：</span>
        <a-radio-group v-model="chartDays" type="button" size="small" @change="reloadCharts">
          <a-radio :value="7">近7天</a-radio>
          <a-radio :value="14">近14天</a-radio>
          <a-radio :value="30">近30天</a-radio>
        </a-radio-group>
      </a-space>
    </a-card>

    <!-- 统计图表区 -->
    <a-row :gutter="16" style="margin-bottom:20px">
      <a-col :span="6">
        <a-card :bordered="false" class="chart-card">
          <div class="chart-title">告警级别分布</div>
          <div ref="roseChartRef" style="height:200px" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" class="chart-card">
          <div class="chart-title">恢复方式统计</div>
          <div ref="barChartRef" style="height:200px" />
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card :bordered="false" class="chart-card">
          <div class="chart-title">告警趋势</div>
          <div ref="trendChartRef" style="height:200px" />
        </a-card>
      </a-col>
    </a-row>

    <!-- 活跃告警列表 -->
    <a-card :bordered="false">
      <template #title>
        <a-space>
          <span style="font-weight:700;font-size:15px">实时告警</span>
          <a-tag color="red" style="font-size:13px;padding:0 8px">{{ total }}</a-tag>
        </a-space>
      </template>
      <template #extra>
        <a-space>
          <span style="font-size:12px;color:var(--ops-text-tertiary)">上次刷新: {{ lastRefresh }}</span>
          <a-button size="small" @click="load"><template #icon><icon-refresh /></template>刷新</a-button>
        </a-space>
      </template>

      <a-row :gutter="12" style="margin-bottom:16px">
        <a-col :span="5">
          <a-select v-model="filterSeverity" placeholder="严重级别" allow-clear @change="load">
            <a-option value="critical"><a-tag color="red" size="small">紧急 P1</a-tag></a-option>
            <a-option value="major"><a-tag color="orangered" size="small">严重 P2</a-tag></a-option>
            <a-option value="minor"><a-tag color="orange" size="small">一般 P3</a-tag></a-option>
            <a-option value="warning"><a-tag color="blue" size="small">提示 P4</a-tag></a-option>
          </a-select>
        </a-col>
        <a-col :span="6"><a-input v-model="keyword" placeholder="搜索规则名称" allow-clear @press-enter="load" /></a-col>
        <a-col :span="6">
          <a-input v-model="labelFilter" placeholder="标签搜索 (如: job=prome*)" allow-clear @press-enter="load">
            <template #prefix><icon-tags /></template>
          </a-input>
        </a-col>
        <a-col :span="3"><a-button type="primary" @click="load">查询</a-button></a-col>
        <a-col :span="4" style="text-align:right">
          <a-button @click="openSilencedModal">
            <template #icon><icon-eye /></template>已屏蔽告警
          </a-button>
        </a-col>
      </a-row>

      <!-- 批量操作栏 -->
      <div v-if="selectedEventIds.length > 0" class="batch-bar">
        <span class="batch-count">已选 <b>{{ selectedEventIds.length }}</b> 条</span>
        <a-space>
          <a-button size="small" @click="openBatchSilence">
            <template #icon><icon-mute /></template>批量屏蔽
          </a-button>
          <a-button size="small" @click="openBatchHandle">
            <template #icon><icon-check-circle /></template>批量处理
          </a-button>
          <a-button size="small" type="text" @click="selectedEventIds = []">取消选择</a-button>
        </a-space>
      </div>

      <a-table :data="events" :loading="loading"
        row-key="id"
        :row-selection="rowSelection"
        v-model:selectedKeys="selectedEventIds"
        :pagination="{ total, pageSize: 50, current: page, onChange: (p:number)=>{page=p;load()} }"
        :bordered="false" stripe
        :expandable="{ expandRowByClick: true, defaultExpandAllRows: false }">
        <!-- 展开行：显示 Labels 详情 -->
        <template #expand-row="{ record }">
          <div class="labels-expand">
            <div class="labels-expand-title">Labels 详情</div>
            <div class="labels-grid">
              <template v-if="parseLabels(record.labels).length > 0">
                <div v-for="(kv, i) in parseLabels(record.labels)" :key="i" class="label-kv">
                  <span class="label-key">{{ kv.k }}</span>
                  <span class="label-val">{{ kv.v }}</span>
                </div>
              </template>
              <span v-else style="color:var(--ops-text-tertiary);font-size:12px">暂无标签</span>
            </div>
            <div v-if="record.annotations" style="margin-top:8px">
              <div class="labels-expand-title">Annotations</div>
              <div class="labels-grid">
                <div v-for="(kv, i) in parseLabels(record.annotations)" :key="i" class="label-kv">
                  <span class="label-key">{{ kv.k }}</span>
                  <span class="label-val">{{ kv.v }}</span>
                </div>
              </div>
            </div>
          </div>
        </template>
        <template #columns>
          <!-- 级别 -->
          <a-table-column title="级别" :width="100">
            <template #cell="{ record }">
              <div class="sev-cell" :class="'sev-'+record.severity">
                <span class="sev-dot"></span>
                <span class="sev-text">{{ {'critical':'紧急P1','major':'严重P2','minor':'一般P3','warning':'提示P4','info':'信息'}[record.severity] || record.severity }}</span>
              </div>
            </template>
          </a-table-column>
          <!-- 规则名称 -->
          <a-table-column title="规则" :width="180">
            <template #cell="{ record }">
              <a-link @click="goRule(record.alertRuleId)" style="font-weight:600">
                {{ record.ruleName }}
              </a-link>
              <div v-if="record.ruleGroupName" style="font-size:11px;color:var(--ops-text-tertiary);margin-top:1px">
                {{ record.ruleGroupName }}
              </div>
            </template>
          </a-table-column>
          <!-- 业务分组 -->
          <a-table-column title="业务分组" :width="120">
            <template #cell="{ record }">
              <span style="font-size:12px;color:var(--ops-text-secondary)">
                {{ record.assetGroupName || '—' }}
              </span>
            </template>
          </a-table-column>
          <!-- 标签摘要 -->
          <a-table-column title="标签" :width="180">
            <template #cell="{ record }">
              <div class="label-tags">
                <template v-if="parseLabels(record.labels).length">
                  <a-tag v-for="(kv,i) in parseLabels(record.labels).slice(0,3)" :key="i"
                    size="small" color="arcoblue" style="margin:1px 2px;font-size:11px;max-width:160px;overflow:hidden;text-overflow:ellipsis">
                    {{ kv.k }}={{ kv.v }}
                  </a-tag>
                  <a-tag v-if="parseLabels(record.labels).length > 3" size="small" color="gray">
                    +{{ parseLabels(record.labels).length - 3 }}
                  </a-tag>
                </template>
                <span v-else style="color:#c9cdd4">—</span>
              </div>
            </template>
          </a-table-column>
          <!-- 当前值 -->
          <a-table-column title="触发值" :width="90">
            <template #cell="{ record }">
              <span class="val-cell">{{ record.value?.toFixed(4) }}</span>
            </template>
          </a-table-column>
          <!-- 触发时间 -->
          <a-table-column title="触发时间" :width="158">
            <template #cell="{ record }">
              <span style="font-size:12px;color:var(--ops-text-secondary)">{{ fmtTime(record.firedAt) }}</span>
            </template>
          </a-table-column>
          <!-- 持续时长 -->
          <a-table-column title="持续" :width="90">
            <template #cell="{ record }">
              <span class="duration-cell">{{ duration(record.firedAt) }}</span>
            </template>
          </a-table-column>
          <!-- 状态标记 -->
          <a-table-column title="状态" :width="100">
            <template #cell="{ record }">
              <div style="display:flex;flex-direction:column;gap:2px">
                <a-tag v-if="record.silenced" color="gray" size="small">屏蔽中</a-tag>
                <a-tag v-if="record.manualHandled" color="orange" size="small">已介入</a-tag>
                <a-badge v-if="!record.silenced && !record.manualHandled" status="danger" text="告警中" />
              </div>
            </template>
          </a-table-column>
          <!-- 操作 -->
          <a-table-column title="操作" :width="120" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="openSilence(record)">屏蔽</a-link>
                <a-link status="warning" @click="openHandle(record)">介入</a-link>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 屏蔽弹窗（单条） -->
    <a-modal v-model:visible="silenceVisible" title="屏蔽告警" @ok="doSilence" @cancel="silenceVisible=false" width="600px">
      <a-form layout="vertical" :model="{ singleSilenceType, singleEditLabels, silenceDuration, silenceReason }">
        <a-form-item label="屏蔽维度">
          <a-alert type="info" style="margin-bottom:12px">
            默认按 <b>告警等级 + 规则名称 + 标签</b> 三元组屏蔽，相同维度的所有告警都会被屏蔽
          </a-alert>
        </a-form-item>

        <a-form-item label="标签编辑（可选）">
          <a-checkbox v-model="singleEditLabels">自定义标签（移除部分标签可扩大屏蔽范围）</a-checkbox>
          <div v-if="singleEditLabels" style="margin-top:8px">
            <a-space wrap>
              <a-tag v-for="(label, idx) in singleSelectedLabels" :key="idx" closable @close="removeSingleLabel(idx)">
                {{ label.k }}={{ label.v }}
              </a-tag>
            </a-space>
            <div style="margin-top:8px;font-size:12px;color:var(--ops-text-tertiary)">
              提示：移除标签后，屏蔽范围会扩大到所有包含剩余标签的告警
            </div>
          </div>
        </a-form-item>

        <a-form-item label="屏蔽类型">
          <a-radio-group v-model="singleSilenceType" type="button">
            <a-radio value="fixed">固定时长</a-radio>
            <a-radio value="periodic">周期性</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="singleSilenceType === 'fixed'" label="屏蔽时长">
          <a-radio-group v-model="silenceDuration" type="button">
            <a-radio value="1h">1小时</a-radio>
            <a-radio value="2h">2小时</a-radio>
            <a-radio value="4h">4小时</a-radio>
            <a-radio value="8h">8小时</a-radio>
            <a-radio value="24h">24小时</a-radio>
            <a-radio value="168h">1周</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="singleSilenceType === 'periodic'" label="生效时间段">
          <a-checkbox-group v-model="singleSilenceWeekdays">
            <a-checkbox :value="1">周一</a-checkbox>
            <a-checkbox :value="2">周二</a-checkbox>
            <a-checkbox :value="3">周三</a-checkbox>
            <a-checkbox :value="4">周四</a-checkbox>
            <a-checkbox :value="5">周五</a-checkbox>
            <a-checkbox :value="6">周六</a-checkbox>
            <a-checkbox :value="7">周日</a-checkbox>
          </a-checkbox-group>
          <a-space style="margin-top:8px">
            <a-time-picker v-model="singleSilenceStart" format="HH:mm" />
            <span>至</span>
            <a-time-picker v-model="singleSilenceEnd" format="HH:mm" />
          </a-space>
        </a-form-item>

        <a-form-item label="屏蔽原因">
          <a-textarea v-model="silenceReason" :auto-size="{minRows:2}" placeholder="请输入屏蔽原因" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 人工介入弹窗 -->
    <a-modal v-model:visible="handleVisible" title="人工介入标记" @ok="doHandle" @cancel="handleVisible=false" width="440px">
      <a-alert type="info" style="margin-bottom:12px">
        标记「人工介入」后，该告警仍处于 firing 状态。当监控指标恢复正常时，系统将自动恢复该告警，并记录恢复方式为「人工介入后自动恢复」。
      </a-alert>
      <a-form layout="vertical" :model="{ handleNote }">
        <a-form-item label="介入备注（可选）">
          <a-textarea v-model="handleNote" :auto-size="{minRows:3}" placeholder="记录处理过程、原因或操作说明..." />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 批量屏蔽弹窗 -->
    <a-modal v-model:visible="batchSilenceVisible" title="批量屏蔽告警" @ok="doBatchSilence" width="600px">
      <a-form layout="vertical" :model="{ batchSilenceType, batchSilenceDuration, batchSilenceReason }">
        <a-form-item label="屏蔽说明">
          <a-alert type="info" style="margin-bottom:12px">
            已选中 <b>{{ selectedEventIds.length }}</b> 条告警，将按每条告警的 <b>告警等级 + 规则名称 + 标签</b> 三元组分别创建屏蔽规则
          </a-alert>
        </a-form-item>

        <a-form-item label="屏蔽类型">
          <a-radio-group v-model="batchSilenceType" type="button">
            <a-radio value="fixed">固定时长</a-radio>
            <a-radio value="periodic">周期性</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="batchSilenceType === 'fixed'" label="屏蔽时长">
          <a-radio-group v-model="batchSilenceDuration" type="button">
            <a-radio value="1h">1小时</a-radio>
            <a-radio value="2h">2小时</a-radio>
            <a-radio value="6h">6小时</a-radio>
            <a-radio value="12h">12小时</a-radio>
            <a-radio value="24h">1天</a-radio>
            <a-radio value="168h">1周</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="batchSilenceType === 'periodic'" label="生效时间段">
          <a-checkbox-group v-model="batchSilenceWeekdays">
            <a-checkbox :value="1">周一</a-checkbox>
            <a-checkbox :value="2">周二</a-checkbox>
            <a-checkbox :value="3">周三</a-checkbox>
            <a-checkbox :value="4">周四</a-checkbox>
            <a-checkbox :value="5">周五</a-checkbox>
            <a-checkbox :value="6">周六</a-checkbox>
            <a-checkbox :value="7">周日</a-checkbox>
          </a-checkbox-group>
          <a-space style="margin-top:8px">
            <a-time-picker v-model="batchSilenceStart" format="HH:mm" />
            <span>至</span>
            <a-time-picker v-model="batchSilenceEnd" format="HH:mm" />
          </a-space>
        </a-form-item>

        <a-form-item label="屏蔽原因">
          <a-textarea v-model="batchSilenceReason" :auto-size="{minRows:2}" placeholder="请输入屏蔽原因" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 已屏蔽告警弹窗 -->
    <SilencedAlertsModal v-model:visible="silencedModalVisible" @refresh="load" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import * as echarts from 'echarts'
import { useUserStore } from '@/stores/user'
import { getActiveEvents, getEventStats, getEventTrend, silenceEvent, handleEvent, batchSilenceEvents } from '@/api/alert'
import SilencedAlertsModal from './SilencedAlertsModal.vue'

const router = useRouter()
const userStore = useUserStore()

const events = ref<any[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const keyword = ref('')
const filterSeverity = ref('')
const labelFilter = ref('')
const lastRefresh = ref('')
const chartDays = ref(7)

// 批量选择
const selectedEventIds = ref<number[]>([])

const rowSelection = computed(() => {
  const config = {
    type: 'checkbox' as const,
    showCheckedAll: true
  }
  console.log('[批量选择] rowSelection computed 执行', {
    selectedKeys: selectedEventIds.value,
    config
  })
  return config
})

// 监听选择变化（通过 v-model:selectedKeys）
watch(selectedEventIds, (newVal) => {
  console.log('[批量选择] selectedEventIds 变化', { newVal })
})

// 批量屏蔽
const batchSilenceVisible = ref(false)
const batchSilenceType = ref('fixed')
const batchSilenceDuration = ref('2h')
const batchSilenceWeekdays = ref<number[]>([1, 2, 3, 4, 5])
const batchSilenceStart = ref('09:00')
const batchSilenceEnd = ref('18:00')
const batchSilenceReason = ref('')

// 已屏蔽告警弹窗
const silencedModalVisible = ref(false)

const roseChartRef = ref<HTMLElement>()
const barChartRef = ref<HTMLElement>()
const trendChartRef = ref<HTMLElement>()
let roseChart: echarts.ECharts | null = null
let barChart: echarts.ECharts | null = null
let trendChart: echarts.ECharts | null = null
let timer: ReturnType<typeof setInterval> | null = null

const fmtTime = (s?: string) => {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const silenceVisible = ref(false)
const silenceDuration = ref('2h')
const silenceReason = ref('')
const singleSilenceType = ref('fixed')
const singleEditLabels = ref(false)
const singleSelectedLabels = ref<{k:string, v:string}[]>([])
const singleSilenceWeekdays = ref<number[]>([1, 2, 3, 4, 5])
const singleSilenceStart = ref('09:00')
const singleSilenceEnd = ref('18:00')
const handleVisible = ref(false)
const handleNote = ref('')
let currentEventId = 0

const duration = (firedAt: string) => {
  const diff = Date.now() - new Date(firedAt).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 60) return `${m}分钟`
  const h = Math.floor(m / 60); const rm = m % 60
  if (h < 24) return `${h}小时${rm}分`
  return `${Math.floor(h/24)}天${h%24}小时`
}

const parseLabels = (s?: string): {k:string,v:string}[] => {
  if (!s || s === '{}' || s === 'null') return []
  try {
    const obj = JSON.parse(s)
    if (!obj || typeof obj !== 'object' || Array.isArray(obj)) return []
    return Object.entries(obj).map(([k, v]) => ({ k, v: String(v) }))
  } catch { return [] }
}

const goRule = (ruleId?: number) => {
  if (!ruleId) return
  router.push('/alert/rules')
}

const load = async () => {
  loading.value = true
  try {
    const d = await getActiveEvents({
      page: page.value,
      pageSize: 50,
      severity: filterSeverity.value,
      keyword: keyword.value,
      labelFilter: labelFilter.value
    }) as any
    events.value = Array.isArray(d) ? d : (d?.data || [])
    total.value = Array.isArray(d) ? d.length : (d?.total || 0)
    lastRefresh.value = new Date().toLocaleTimeString()

    // 调试日志
    console.log('[数据加载] 告警列表加载完成', {
      count: events.value.length,
      firstRecord: events.value[0],
      hasId: events.value.length > 0 && events.value[0]?.id !== undefined,
      ids: events.value.slice(0, 3).map(e => e.id)
    })
  } finally { loading.value = false }
}

const reloadCharts = () => { initCharts() }

const initCharts = async () => {
  const [statsRes, trendRes] = await Promise.all([getEventStats(), getEventTrend(chartDays.value)])
  const stats = (statsRes as any) || {}
  const trend: any[] = Array.isArray(trendRes) ? trendRes as any[] : []

  if (roseChartRef.value) {
    roseChart = echarts.init(roseChartRef.value)
    roseChart.setOption({
      tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
      series: [{
        type: 'pie', roseType: 'area', radius: ['15%', '70%'],
        data: [
          { value: stats.criticalCount || 0, name: '紧急P1', itemStyle: { color: '#f53f3f' } },
          { value: stats.majorCount || 0, name: '严重P2', itemStyle: { color: '#d95c2c' } },
          { value: stats.minorCount || 0, name: '一般P3', itemStyle: { color: '#ff7d00' } },
          { value: stats.warningCount || 0, name: '提示P4', itemStyle: { color: '#165dff' } },
        ],
        label: { fontSize: 11 },
        emphasis: { itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0,0,0,0.3)' } }
      }]
    })
  }

  if (barChartRef.value) {
    barChart = echarts.init(barChartRef.value)
    const barDates = trend.map((t: any) => t.date)
    const autoRes = trend.map((t: any) => t.autoResolvedCount ?? 0)
    const manualRes = trend.map((t: any) => t.manualResolvedCount ?? 0)
    const firingDay = trend.map((t: any) => t.firingCount ?? 0)
    barChart.setOption({
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      legend: { top: 0, textStyle: { fontSize: 10 } },
      grid: { top: 30, bottom: 40, left: 36, right: 10 },
      xAxis: { type: 'category', data: barDates, axisLabel: { fontSize: 9, rotate: 30 } },
      yAxis: { type: 'value', minInterval: 1 },
      series: [
        { name: '新增告警', type: 'bar', stack: 'total', data: firingDay, barMaxWidth: 18,
          itemStyle: { color: '#f53f3f' } },
        { name: '自动恢复', type: 'bar', stack: 'total', data: autoRes, barMaxWidth: 18,
          itemStyle: { color: '#00b42a' } },
        { name: '人工恢复', type: 'bar', stack: 'total', data: manualRes, barMaxWidth: 18,
          itemStyle: { color: '#ff7d00', borderRadius: [3,3,0,0] } },
      ]
    })
  }

  if (trendChartRef.value) {
    trendChart = echarts.init(trendChartRef.value)
    const dates = trend.map((t: any) => t.date)
    const firing = trend.map((t: any) => t.firingCount)
    const resolved = trend.map((t: any) => t.resolvedCount)
    trendChart.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: ['新增告警', '已恢复'], top: 0, textStyle: { fontSize: 11 } },
      grid: { top: 30, bottom: 40, left: 40, right: 20 },
      xAxis: { type: 'category', data: dates, axisLabel: { fontSize: 10 } },
      yAxis: { type: 'value', minInterval: 1 },
      series: [
        { name: '新增告警', type: 'line', data: firing, smooth: true, symbol: 'circle', symbolSize: 4,
          areaStyle: { color: new echarts.graphic.LinearGradient(0,0,0,1,[{offset:0,color:'rgba(245,63,63,0.4)'},{offset:1,color:'rgba(245,63,63,0)'}]) },
          lineStyle: { color: '#f53f3f' }, itemStyle: { color: '#f53f3f' } },
        { name: '已恢复', type: 'line', data: resolved, smooth: true, symbol: 'circle', symbolSize: 4,
          areaStyle: { color: new echarts.graphic.LinearGradient(0,0,0,1,[{offset:0,color:'rgba(0,180,42,0.4)'},{offset:1,color:'rgba(0,180,42,0)'}]) },
          lineStyle: { color: '#00b42a' }, itemStyle: { color: '#00b42a' } }
      ]
    })
  }
}

const openSilence = (row: any) => {
  currentEventId = row.id
  silenceReason.value = ''
  singleEditLabels.value = false
  singleSilenceType.value = 'fixed'
  singleSelectedLabels.value = parseLabels(row.labels)
  silenceVisible.value = true
}

const removeSingleLabel = (idx: number) => {
  singleSelectedLabels.value.splice(idx, 1)
}

const doSilence = async () => {
  try {
    const data: any = {
      eventIds: [currentEventId],
      type: singleSilenceType.value,
      reason: silenceReason.value
    }

    if (singleEditLabels.value) {
      const labelsObj: Record<string, string> = {}
      singleSelectedLabels.value.forEach(l => { labelsObj[l.k] = l.v })
      data.editLabels = true
      data.labels = JSON.stringify(labelsObj)
    }

    if (singleSilenceType.value === 'fixed') {
      data.duration = silenceDuration.value
    } else {
      data.timeRanges = JSON.stringify([{
        weekdays: singleSilenceWeekdays.value,
        start: singleSilenceStart.value,
        end: singleSilenceEnd.value
      }])
    }

    await batchSilenceEvents(data)
    Message.success('屏蔽成功')
    silenceVisible.value = false
    load()
  } catch {
    Message.error('操作失败')
  }
}
const openHandle = (row: any) => { currentEventId = row.id; handleNote.value = ''; handleVisible.value = true }
const doHandle = async () => {
  try { await handleEvent(currentEventId, { note: handleNote.value, userId: userStore.userInfo?.id }); Message.success('已标记人工介入'); handleVisible.value = false; load() }
  catch { Message.error('操作失败') }
}

// 批量屏蔽
const openBatchSilence = () => {
  if (selectedEventIds.value.length === 0) {
    Message.warning('请先选择告警')
    return
  }

  console.log('[批量屏蔽] 打开弹窗，已选告警数:', selectedEventIds.value.length)
  console.log('[批量屏蔽] 已选告警ID列表:', selectedEventIds.value)

  batchSilenceVisible.value = true
}

const doBatchSilence = async () => {
  console.log('[批量屏蔽] 开始执行，参数:', {
    eventIds: selectedEventIds.value,
    type: batchSilenceType.value,
    duration: batchSilenceType.value === 'fixed' ? batchSilenceDuration.value : undefined,
    timeRanges: batchSilenceType.value === 'periodic' ? {
      weekdays: batchSilenceWeekdays.value,
      start: batchSilenceStart.value,
      end: batchSilenceEnd.value
    } : undefined,
    reason: batchSilenceReason.value
  })

  try {
    const data: any = {
      eventIds: selectedEventIds.value,
      type: batchSilenceType.value,
      reason: batchSilenceReason.value
    }

    if (batchSilenceType.value === 'fixed') {
      data.duration = batchSilenceDuration.value
    } else {
      data.timeRanges = JSON.stringify([{
        weekdays: batchSilenceWeekdays.value,
        start: batchSilenceStart.value,
        end: batchSilenceEnd.value
      }])
    }

    console.log('[批量屏蔽] 发送请求数据:', data)
    await batchSilenceEvents(data)
    console.log('[批量屏蔽] 请求成功')
    Message.success('批量屏蔽成功')
    batchSilenceVisible.value = false
    selectedEventIds.value = []
    load()
  } catch (err) {
    console.error('[批量屏蔽] 请求失败:', err)
    Message.error('批量屏蔽失败')
  }
}

const openBatchHandle = () => {
  Message.info('批量处理功能开发中...')
}

const openSilencedModal = () => {
  silencedModalVisible.value = true
}

onMounted(async () => {
  await load()
  await initCharts()
  timer = setInterval(load, 30000)
})
onUnmounted(() => { if (timer) clearInterval(timer); roseChart?.dispose(); barChart?.dispose(); trendChart?.dispose() })
</script>

<style scoped>
.page-container { padding: 20px; background: var(--ops-content-bg); min-height: 100%; }
.chart-card { border-radius: 8px; }
.chart-title { font-size: 13px; font-weight: 600; color: var(--ops-text-primary); margin-bottom: 8px; }

/* 批量操作栏 */
.batch-bar {
  display: flex; align-items: center; gap: 12px;
  background: #e8f3ff; border: 1px solid #bedaff; border-radius: 6px;
  padding: 8px 14px; margin-bottom: 12px;
}
.batch-count { font-size: 13px; color: var(--ops-primary); flex-shrink: 0; }

/* 级别徽标 */
.sev-cell { display: flex; align-items: center; gap: 5px; }
.sev-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.sev-text { font-size: 12px; font-weight: 600; }
.sev-critical .sev-dot { background: #f53f3f; box-shadow: 0 0 0 3px rgba(245,63,63,0.2); }
.sev-critical .sev-text { color: #f53f3f; }
.sev-major .sev-dot { background: #d95c2c; box-shadow: 0 0 0 3px rgba(217,92,44,0.2); }
.sev-major .sev-text { color: #d95c2c; }
.sev-minor .sev-dot { background: #ff7d00; box-shadow: 0 0 0 3px rgba(255,125,0,0.2); }
.sev-minor .sev-text { color: #ff7d00; }
.sev-warning .sev-dot { background: #165dff; box-shadow: 0 0 0 3px rgba(22,93,255,0.15); }
.sev-warning .sev-text { color: #165dff; }
.sev-info .sev-dot { background: #86909c; box-shadow: 0 0 0 3px rgba(134,144,156,0.15); }
.sev-info .sev-text { color: #86909c; }

.val-cell { font-weight: 700; color: #f53f3f; font-size: 13px; font-family: monospace; }
.duration-cell { font-size: 12px; color: var(--ops-text-secondary); background: #fff7e6;
  padding: 1px 6px; border-radius: 10px; border: 1px solid #ffe7ba; }

/* 标签 */
.label-tags { display: flex; flex-wrap: wrap; gap: 2px; }

/* 展开行 */
.labels-expand { padding: 12px 16px; background: #f7f8fa; border-radius: 6px; }
.labels-expand-title { font-size: 12px; font-weight: 600; color: var(--ops-text-secondary); margin-bottom: 6px; }
.labels-grid { display: flex; flex-wrap: wrap; gap: 6px; }
.label-kv {
  display: inline-flex; align-items: center; gap: 0;
  background: #fff; border: 1px solid #e5e6eb; border-radius: 4px; overflow: hidden;
  font-size: 12px;
}
.label-key { background: #f2f3f5; padding: 2px 6px; color: #4e5969; font-weight: 600; }
.label-val { padding: 2px 8px; color: #1d2129; font-family: monospace; }
</style>
