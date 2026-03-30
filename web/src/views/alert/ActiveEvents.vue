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
        <a-col :span="7"><a-input v-model="keyword" placeholder="搜索规则名称" allow-clear @press-enter="load" /></a-col>
        <a-col :span="4"><a-button type="primary" @click="load">查询</a-button></a-col>
      </a-row>

      <a-table :data="events" :loading="loading" row-key="id"
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

    <!-- 屏蔽弹窗 -->
    <a-modal v-model:visible="silenceVisible" title="屏蔽告警" @ok="doSilence" @cancel="silenceVisible=false" width="400px">
      <a-form layout="vertical" :model="{}">
        <a-form-item label="屏蔽时长">
          <a-radio-group v-model="silenceDuration" type="button">
            <a-radio value="1h">1小时</a-radio>
            <a-radio value="2h">2小时</a-radio>
            <a-radio value="4h">4小时</a-radio>
            <a-radio value="8h">8小时</a-radio>
            <a-radio value="24h">24小时</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="屏蔽原因"><a-textarea v-model="silenceReason" :auto-size="{minRows:2}" /></a-form-item>
      </a-form>
    </a-modal>

    <!-- 人工介入弹窗 -->
    <a-modal v-model:visible="handleVisible" title="人工介入标记" @ok="doHandle" @cancel="handleVisible=false" width="440px">
      <a-alert type="info" style="margin-bottom:12px">
        标记「人工介入」后，该告警仍处于 firing 状态。当监控指标恢复正常时，系统将自动恢复该告警，并记录恢复方式为「人工介入后自动恢复」。
      </a-alert>
      <a-form layout="vertical" :model="{}">
        <a-form-item label="介入备注（可选）">
          <a-textarea v-model="handleNote" :auto-size="{minRows:3}" placeholder="记录处理过程、原因或操作说明..." />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import * as echarts from 'echarts'
import { useUserStore } from '@/stores/user'
import { getActiveEvents, getEventStats, getEventTrend, silenceEvent, handleEvent } from '@/api/alert'

const router = useRouter()
const userStore = useUserStore()

const events = ref<any[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const keyword = ref('')
const filterSeverity = ref('')
const lastRefresh = ref('')
const chartDays = ref(7)

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
    const d = await getActiveEvents({ page: page.value, pageSize: 50, severity: filterSeverity.value, keyword: keyword.value }) as any
    events.value = Array.isArray(d) ? d : (d?.data || [])
    total.value = Array.isArray(d) ? d.length : (d?.total || 0)
    lastRefresh.value = new Date().toLocaleTimeString()
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
      legend: { bottom: 0, textStyle: { fontSize: 10 } },
      grid: { top: 10, bottom: 40, left: 36, right: 10 },
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
      legend: { data: ['新增告警', '已恢复'], bottom: 0, textStyle: { fontSize: 11 } },
      grid: { top: 20, bottom: 40, left: 40, right: 20 },
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

const openSilence = (row: any) => { currentEventId = row.id; silenceReason.value = ''; silenceVisible.value = true }
const doSilence = async () => {
  try { await silenceEvent(currentEventId, { duration: silenceDuration.value, reason: silenceReason.value }); Message.success('屏蔽成功'); silenceVisible.value = false; load() }
  catch { Message.error('操作失败') }
}
const openHandle = (row: any) => { currentEventId = row.id; handleNote.value = ''; handleVisible.value = true }
const doHandle = async () => {
  try { await handleEvent(currentEventId, { note: handleNote.value, userId: userStore.userInfo?.id }); Message.success('已标记人工介入'); handleVisible.value = false; load() }
  catch { Message.error('操作失败') }
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
