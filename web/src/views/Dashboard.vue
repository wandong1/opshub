<template>
  <div class="dashboard">
    <!-- 顶部统计卡片 -->
    <a-row :gutter="20" class="stats-row">
      <a-col :span="6" v-for="(stat, index) in topStats" :key="index">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" :style="{ backgroundColor: stat.color }">
              <component :is="stat.icon" :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stat.value }}</div>
              <div class="stat-label">{{ stat.label }}</div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 图表展示区域 -->
    <a-row :gutter="20" class="chart-row">
      <a-col :span="12">
        <a-card class="chart-card" hoverable :header-style="{ padding: '14px 20px' }">
          <template #title>
            <span class="card-title">主机状态分布</span>
          </template>
          <template #extra>
            <a-link @click="$router.push('/asset/hosts')">查看全部</a-link>
          </template>
          <div ref="hostStatusChart" class="chart-container"></div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card class="chart-card" hoverable :header-style="{ padding: '14px 20px' }">
          <template #title>
            <span class="card-title">K8s集群资源概览</span>
          </template>
          <template #extra>
            <a-link @click="$router.push('/kubernetes/clusters')">查看全部</a-link>
          </template>
          <div ref="k8sResourceChart" class="chart-container"></div>
        </a-card>
      </a-col>
    </a-row>
    <a-row :gutter="20" class="chart-row">
      <a-col :span="12">
        <a-card class="chart-card" hoverable :header-style="{ padding: '14px 20px' }">
          <template #title>
            <span class="card-title">操作趋势（最近7天）</span>
          </template>
          <template #extra>
            <a-link @click="$router.push('/audit/operation-logs')">查看全部</a-link>
          </template>
          <div ref="operationTrendChart" class="chart-container"></div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card class="chart-card" hoverable :header-style="{ padding: '14px 20px' }">
          <template #title>
            <span class="card-title">告警统计</span>
          </template>
          <template #extra>
            <a-link @click="$router.push('/monitor/alert-logs')">查看全部</a-link>
          </template>
          <div ref="alertStatsChart" class="chart-container"></div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 快速入口 -->
    <a-row :gutter="20" class="quick-access-row">
      <a-col :span="24">
        <a-card class="quick-access-card" hoverable :header-style="{ padding: '14px 20px' }">
          <template #title>
            <span class="card-title">快速入口</span>
          </template>
          <div class="quick-access-grid">
            <div class="quick-item" v-for="entry in quickEntries" :key="entry.path" @click="$router.push(entry.path)">
              <component :is="entry.icon" :size="28" :style="{ color: entry.color }" />
              <span>{{ entry.label }}</span>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, markRaw } from 'vue'
import {
  IconCommon, IconStorage, IconFile, IconThunderbolt,
  IconSafe, IconCloud
} from '@arco-design/web-vue/es/icon'
import { getHostList } from '@/api/host'
import { getClusterList } from '@/api/kubernetes'
import { getOperationLogList } from '@/api/audit'
import { getAlertLogs } from '@/api/alert-config'
import * as echarts from 'echarts'

const topStats = ref([
  { label: '主机总数', value: '0', icon: markRaw(IconCommon), color: '#165dff' },
  { label: 'K8s集群', value: '0', icon: markRaw(IconStorage), color: '#00b42a' },
  { label: '今日操作', value: '0', icon: markRaw(IconFile), color: '#ff7d00' },
  { label: '活跃告警', value: '0', icon: markRaw(IconThunderbolt), color: '#f53f3f' },
])

const quickEntries = [
  { path: '/asset/hosts', icon: markRaw(IconCommon), color: '#165dff', label: '主机管理' },
  { path: '/kubernetes/clusters', icon: markRaw(IconStorage), color: '#00b42a', label: 'K8s集群' },
  { path: '/audit/operation-logs', icon: markRaw(IconFile), color: '#ff7d00', label: '操作日志' },
  { path: '/monitor/alert-logs', icon: markRaw(IconThunderbolt), color: '#f53f3f', label: '告警日志' },
  { path: '/asset/credentials', icon: markRaw(IconSafe), color: '#86909c', label: '凭据管理' },
  { path: '/asset/cloud-accounts', icon: markRaw(IconCloud), color: '#4e5969', label: '云账号' },
]

const hostStatusChart = ref<HTMLElement>()
const k8sResourceChart = ref<HTMLElement>()
const operationTrendChart = ref<HTMLElement>()
const alertStatsChart = ref<HTMLElement>()

const hosts = ref<any[]>([])
const clusters = ref<any[]>([])
const operationLogs = ref<any[]>([])
const alertLogs = ref<any[]>([])

const fetchHosts = async () => {
  try {
    const res: any = await getHostList({ page: 1, pageSize: 100 })
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        hosts.value = res.list
        topStats.value[0].value = String(res.total || res.list.length || 0)
      } else if (Array.isArray(res)) {
        hosts.value = res
        topStats.value[0].value = String(res.length || 0)
      }
    }
    await nextTick()
    renderHostStatusChart()
  } catch { topStats.value[0].value = '0' }
}

const fetchClusters = async () => {
  try {
    const res: any = await getClusterList()
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        clusters.value = res.list
        topStats.value[1].value = String(res.total || res.list.length || 0)
      } else if (Array.isArray(res)) {
        clusters.value = res
        topStats.value[1].value = String(res.length || 0)
      }
    }
    await nextTick()
    renderK8sResourceChart()
  } catch { topStats.value[1].value = '0' }
}

const fetchOperationLogs = async () => {
  try {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const res: any = await getOperationLogList({ page: 1, pageSize: 500 })
    if (res) {
      const list = res.list && Array.isArray(res.list) ? res.list : Array.isArray(res) ? res : []
      operationLogs.value = list
      const todayCount = list.filter((log: any) => new Date(log.createdAt) >= today).length
      topStats.value[2].value = String(todayCount)
    }
    await nextTick()
    renderOperationTrendChart()
  } catch { topStats.value[2].value = '0' }
}

const fetchAlertLogs = async () => {
  try {
    const res: any = await getAlertLogs({ page: 1, pageSize: 100 })
    if (res) {
      const list = res.list && Array.isArray(res.list) ? res.list : Array.isArray(res) ? res : []
      alertLogs.value = list
      const activeCount = list.filter((log: any) => log.status === 'failed').length
      topStats.value[3].value = String(activeCount)
    }
    await nextTick()
    renderAlertStatsChart()
  } catch { topStats.value[3].value = '0' }
}

// Arco 品牌色系
const COLORS = {
  primary: '#165dff',
  success: '#00b42a',
  warning: '#ff7d00',
  danger: '#f53f3f',
  gray: '#86909c',
}

const renderHostStatusChart = () => {
  if (!hostStatusChart.value) return
  const chart = echarts.init(hostStatusChart.value)
  const onlineCount = hosts.value.filter(h => h.status === 1).length
  const offlineCount = hosts.value.filter(h => h.status !== 1).length
  chart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { orient: 'vertical', right: 10, top: 'center' },
    series: [{
      name: '主机状态', type: 'pie', radius: ['40%', '70%'], center: ['40%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
      label: { show: false, position: 'center' },
      emphasis: { label: { show: true, fontSize: 20, fontWeight: 'bold' } },
      labelLine: { show: false },
      data: [
        { value: onlineCount, name: '在线', itemStyle: { color: COLORS.success } },
        { value: offlineCount, name: '离线', itemStyle: { color: COLORS.gray } },
      ],
    }],
  })
  window.addEventListener('resize', () => chart.resize())
}

const renderK8sResourceChart = () => {
  if (!k8sResourceChart.value) return
  const chart = echarts.init(k8sResourceChart.value)
  const clusterNames = clusters.value.map(c => c.name || '未命名')
  const nodeCounts = clusters.value.map(c => c.nodeCount || 0)
  const podCounts = clusters.value.map(c => c.podCount || 0)
  chart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { data: ['节点数', 'Pod数'], top: 10 },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '15%', containLabel: true },
    xAxis: {
      type: 'category',
      data: clusterNames.length > 0 ? clusterNames : ['暂无数据'],
      axisLabel: { interval: 0, rotate: clusterNames.length > 3 ? 30 : 0 },
    },
    yAxis: { type: 'value' },
    series: [
      { name: '节点数', type: 'bar', data: nodeCounts.length > 0 ? nodeCounts : [0], itemStyle: { color: COLORS.primary }, barMaxWidth: 40 },
      { name: 'Pod数', type: 'bar', data: podCounts.length > 0 ? podCounts : [0], itemStyle: { color: COLORS.success }, barMaxWidth: 40 },
    ],
  })
  window.addEventListener('resize', () => chart.resize())
}

const renderOperationTrendChart = () => {
  if (!operationTrendChart.value) return
  const chart = echarts.init(operationTrendChart.value)
  const today = new Date()
  const dates: string[] = []
  const counts: number[] = []
  for (let i = 6; i >= 0; i--) {
    const date = new Date(today)
    date.setDate(date.getDate() - i)
    dates.push(`${date.getMonth() + 1}/${date.getDate()}`)
    counts.push(operationLogs.value.filter((log: any) => new Date(log.createdAt).toDateString() === date.toDateString()).length)
  }
  chart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '10%', containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: dates },
    yAxis: { type: 'value' },
    series: [{
      name: '操作次数', type: 'line', smooth: true, data: counts,
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(255, 125, 0, 0.3)' },
          { offset: 1, color: 'rgba(255, 125, 0, 0.05)' },
        ]),
      },
      itemStyle: { color: COLORS.warning }, lineStyle: { width: 2 },
    }],
  })
  window.addEventListener('resize', () => chart.resize())
}

const renderAlertStatsChart = () => {
  if (!alertStatsChart.value) return
  const chart = echarts.init(alertStatsChart.value)
  const typeMap = new Map<string, number>()
  alertLogs.value.forEach((log: any) => {
    const type = log.alertType || '未知'
    typeMap.set(type, (typeMap.get(type) || 0) + 1)
  })
  const typeData = Array.from(typeMap.entries())
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 5)
  chart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { orient: 'vertical', right: 10, top: 'center', data: typeData.map(d => d.name) },
    series: [{
      name: '告警类型', type: 'pie', radius: ['40%', '70%'], center: ['40%', '50%'],
      data: typeData.length > 0 ? typeData : [{ name: '暂无数据', value: 1 }],
      emphasis: { itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0,0,0,0.5)' } },
    }],
  })
  window.addEventListener('resize', () => chart.resize())
}

onMounted(() => {
  fetchHosts()
  fetchClusters()
  fetchOperationLogs()
  fetchAlertLogs()
})
</script>

<style scoped>
.dashboard { padding: 0; }

.stats-row { margin-bottom: 20px; }

.stat-card { border-radius: var(--ops-border-radius-md, 8px); }

.stat-content {
  display: flex;
  align-items: center;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  flex-shrink: 0;
}

.stat-info { flex: 1; }

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1;
  margin-bottom: 6px;
}

.stat-label {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
}

.chart-row { margin-bottom: 20px; }

.chart-card {
  border-radius: var(--ops-border-radius-md, 8px);
  height: 100%;
}

.card-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--ops-text-primary, #1d2129);
}

.chart-container {
  width: 100%;
  height: 300px;
}

.quick-access-row { margin-bottom: 20px; }

.quick-access-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.quick-access-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 16px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px 12px;
  border-radius: var(--ops-border-radius-md, 8px);
  background: #f7f8fa;
  cursor: pointer;
  transition: all 0.2s;
}

.quick-item:hover {
  background: var(--ops-primary-bg, #e8f0ff);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.quick-item span {
  margin-top: 10px;
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
  font-weight: 500;
}

@media (max-width: 1200px) {
  .stat-value { font-size: 24px; }
  .stat-icon { width: 48px; height: 48px; }
  .chart-container { height: 250px; }
}
</style>
