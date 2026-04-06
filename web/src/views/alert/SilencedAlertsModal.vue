<template>
  <a-modal v-model:visible="visible" title="已屏蔽告警" width="1200px" :footer="false" @cancel="handleClose">
    <!-- 过滤栏 -->
    <a-row :gutter="12" style="margin-bottom:16px">
      <a-col :span="6">
        <a-select v-model="filterSeverity" placeholder="告警等级" allow-clear @change="load">
          <a-option value="critical">P1-紧急</a-option>
          <a-option value="major">P2-重要</a-option>
          <a-option value="minor">P3-次要</a-option>
          <a-option value="warning">P4-警告</a-option>
        </a-select>
      </a-col>
      <a-col :span="6">
        <a-select v-model="filterStatus" placeholder="告警状态" allow-clear @change="load">
          <a-option value="firing">告警中</a-option>
          <a-option value="resolved">已恢复</a-option>
        </a-select>
      </a-col>
      <a-col :span="6">
        <a-input v-model="keyword" placeholder="搜索规则名称" allow-clear @press-enter="load" />
      </a-col>
      <a-col :span="6">
        <a-input v-model="labelFilter" placeholder="标签搜索" allow-clear @press-enter="load">
          <template #prefix><icon-tags /></template>
        </a-input>
      </a-col>
    </a-row>

    <!-- 批量操作栏 -->
    <div v-if="selectedIds.length > 0" class="batch-bar">
      <span class="batch-count">已选 <b>{{ selectedIds.length }}</b> 条</span>
      <a-space>
        <a-popconfirm content="确认取消屏蔽选中的告警？" @ok="batchUnsilence">
          <a-button size="small" status="warning">
            <template #icon><icon-sound /></template>批量取消屏蔽
          </a-button>
        </a-popconfirm>
        <a-button size="small" type="text" @click="selectedIds = []">取消选择</a-button>
      </a-space>
    </div>

    <!-- 表格 -->
    <a-table :data="events" :loading="loading"
      row-key="id"
      :row-selection="rowSelection"
      v-model:selectedKeys="selectedIds"
      :pagination="{ total, pageSize, current: page, onChange: onPageChange }"
      :expandable="{ expandRowByClick: true, defaultExpandAllRows: false }">
      <!-- 展开行：显示屏蔽详情和标签 -->
      <template #expand-row="{ record }">
        <div class="silence-expand">
          <div class="expand-section">
            <div class="expand-title">屏蔽信息</div>
            <div class="expand-content">
              <div class="info-row">
                <span class="info-label">屏蔽类型：</span>
                <span class="info-value">{{ record.silenceType === 'periodic' ? '周期性屏蔽' : '固定时长屏蔽' }}</span>
              </div>
              <div v-if="record.silenceType === 'fixed'" class="info-row">
                <span class="info-label">屏蔽时间：</span>
                <span class="info-value">{{ fmtTime(record.silencedAt) }} 至 {{ record.silenceUntil ? fmtTime(record.silenceUntil) : '—' }}</span>
              </div>
              <div v-if="record.silenceType === 'periodic' && record.silenceTimeRanges" class="info-row">
                <span class="info-label">生效时段：</span>
                <span class="info-value">{{ formatTimeRanges(record.silenceTimeRanges) }}</span>
              </div>
              <div v-if="record.silenceReason" class="info-row">
                <span class="info-label">屏蔽原因：</span>
                <span class="info-value">{{ record.silenceReason }}</span>
              </div>
            </div>
          </div>

          <div class="expand-section">
            <div class="expand-title">Labels 详情</div>
            <div class="labels-grid">
              <template v-if="parseLabels(record.labels).length > 0">
                <div v-for="(kv, i) in parseLabels(record.labels)" :key="i" class="label-kv">
                  <span class="label-key">{{ kv.k }}</span>
                  <span class="label-val">{{ kv.v }}</span>
                </div>
              </template>
              <span v-else style="color:var(--ops-text-tertiary);font-size:12px">暂无标签</span>
            </div>
          </div>

          <div v-if="record.annotations" class="expand-section">
            <div class="expand-title">Annotations</div>
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
        <a-table-column title="级别" :width="100">
          <template #cell="{ record }">
            <div class="severity-cell">
              <span :class="['severity-dot', `severity-${record.severity}`]"></span>
              <span class="severity-text">{{ sevLabel(record.severity) }}</span>
            </div>
          </template>
        </a-table-column>
        <a-table-column title="规则名称" :width="180" data-index="ruleName" />
        <a-table-column title="状态" :width="80">
          <template #cell="{ record }">
            <a-tag v-if="record.status === 'firing'" color="red">告警中</a-tag>
            <a-tag v-else color="green">已恢复</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="标签摘要" :width="200">
          <template #cell="{ record }">
            <div class="label-tags">
              <template v-if="parseLabels(record.labels).length">
                <a-tag v-for="(kv,i) in parseLabels(record.labels).slice(0,2)" :key="i"
                  size="small" color="arcoblue" style="margin:1px 2px;font-size:11px;max-width:140px;overflow:hidden;text-overflow:ellipsis">
                  {{ kv.k }}={{ kv.v }}
                </a-tag>
                <a-tag v-if="parseLabels(record.labels).length > 2" size="small" color="gray">
                  +{{ parseLabels(record.labels).length - 2 }}
                </a-tag>
              </template>
              <span v-else style="color:#c9cdd4">—</span>
            </div>
          </template>
        </a-table-column>
        <a-table-column title="屏蔽类型" :width="110">
          <template #cell="{ record }">
            <a-tag v-if="record.silenceType === 'periodic'" color="blue" size="small">周期性</a-tag>
            <a-tag v-else color="purple" size="small">固定时长</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="屏蔽时间" :width="180">
          <template #cell="{ record }">
            <div v-if="record.silenceType === 'fixed'" style="font-size:12px;color:var(--ops-text-secondary)">
              {{ record.silenceUntil ? '至 ' + fmtTime(record.silenceUntil) : '—' }}
            </div>
            <div v-else style="font-size:11px;color:var(--ops-text-secondary)">
              {{ formatTimeRangesShort(record.silenceTimeRanges) }}
            </div>
          </template>
        </a-table-column>
        <a-table-column title="触发时间" :width="160">
          <template #cell="{ record }">
            {{ fmtTime(record.firedAt) }}
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="100" fixed="right">
          <template #cell="{ record }">
            <a-popconfirm content="确认取消屏蔽？" @ok="unsilence(record.id)">
              <a-link>取消屏蔽</a-link>
            </a-popconfirm>
          </template>
        </a-table-column>
      </template>
    </a-table>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getSilencedEvents, batchUnsilenceEvents } from '@/api/alert'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'refresh'): void
}>()

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const events = ref<any[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const filterSeverity = ref('')
const filterStatus = ref('')
const keyword = ref('')
const labelFilter = ref('')

// 批量选择
const selectedIds = ref<number[]>([])
const rowSelection = computed(() => {
  const config = {
    type: 'checkbox' as const,
    showCheckedAll: true
  }
  console.log('[已屏蔽告警] rowSelection computed 执行', {
    selectedKeys: selectedIds.value,
    config
  })
  return config
})

// 监听选择变化
watch(selectedIds, (newVal) => {
  console.log('[已屏蔽告警] selectedIds 变化', { newVal })
})

const fmtTime = (s?: string) => {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const sevLabel = (sev: string) => {
  const map: Record<string, string> = {
    critical: 'P1-紧急',
    major: 'P2-重要',
    minor: 'P3-次要',
    warning: 'P4-警告',
    info: '信息'
  }
  return map[sev] || sev
}

const parseLabels = (s?: string): {k:string,v:string}[] => {
  if (!s || s === '{}' || s === 'null') return []
  try {
    const obj = JSON.parse(s)
    if (!obj || typeof obj !== 'object' || Array.isArray(obj)) return []
    return Object.entries(obj).map(([k, v]) => ({ k, v: String(v) }))
  } catch { return [] }
}

const formatTimeRanges = (timeRangesJson?: string): string => {
  if (!timeRangesJson) return '—'
  try {
    const ranges = JSON.parse(timeRangesJson)
    if (!Array.isArray(ranges) || ranges.length === 0) return '—'
    const range = ranges[0]
    const weekdayMap: Record<number, string> = {1:'周一',2:'周二',3:'周三',4:'周四',5:'周五',6:'周六',7:'周日'}
    const weekdays = (range.weekdays || []).map((d: number) => weekdayMap[d]).join('、')
    return `${weekdays} ${range.start || ''}-${range.end || ''}`
  } catch { return '—' }
}

const formatTimeRangesShort = (timeRangesJson?: string): string => {
  if (!timeRangesJson) return '—'
  try {
    const ranges = JSON.parse(timeRangesJson)
    if (!Array.isArray(ranges) || ranges.length === 0) return '—'
    const range = ranges[0]
    return `${range.start || ''}-${range.end || ''}`
  } catch { return '—' }
}

const load = async () => {
  loading.value = true
  try {
    const res = await getSilencedEvents({
      page: page.value,
      pageSize: pageSize.value,
      severity: filterSeverity.value,
      status: filterStatus.value,
      keyword: keyword.value,
      labelFilter: labelFilter.value
    }) as any
    events.value = res?.data || []
    total.value = res?.total || 0

    // 调试日志
    console.log('[已屏蔽告警] 数据加载完成', {
      count: events.value.length,
      firstRecord: events.value[0],
      hasId: events.value.length > 0 && events.value[0]?.id !== undefined,
      ids: events.value.slice(0, 3).map(e => e.id)
    })
  } catch (err) {
    console.error('[已屏蔽告警] 查询失败', err)
    Message.error('查询失败')
  } finally {
    loading.value = false
  }
}

const onPageChange = (p: number) => {
  page.value = p
  load()
}

const unsilence = async (id: number) => {
  try {
    await batchUnsilenceEvents([id])
    Message.success('取消屏蔽成功')
    load()
    emit('refresh')
  } catch {
    Message.error('操作失败')
  }
}

const batchUnsilence = async () => {
  console.log('[已屏蔽告警] 批量取消屏蔽', { selectedIds: selectedIds.value })
  try {
    await batchUnsilenceEvents(selectedIds.value)
    Message.success(`已取消 ${selectedIds.value.length} 条告警的屏蔽`)
    selectedIds.value = []
    load()
    emit('refresh')
  } catch (err) {
    console.error('[已屏蔽告警] 批量取消屏蔽失败', err)
    Message.error('批量取消屏蔽失败')
  }
}

const handleClose = () => {
  visible.value = false
}

// 监听弹窗打开，自动加载数据
watch(() => props.visible, (val) => {
  if (val) {
    load()
  }
})
</script>

<style scoped>
.batch-bar {
  display: flex; align-items: center; gap: 12px;
  background: #e8f3ff; border: 1px solid #bedaff; border-radius: 6px;
  padding: 8px 14px; margin-bottom: 12px;
}
.batch-count { font-size: 13px; color: var(--ops-primary); flex-shrink: 0; }

.severity-cell { display: flex; align-items: center; gap: 5px; }
.severity-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.severity-text { font-size: 12px; font-weight: 600; }
.severity-critical .severity-dot { background: #f53f3f; box-shadow: 0 0 0 3px rgba(245,63,63,0.2); }
.severity-critical .severity-text { color: #f53f3f; }
.severity-major .severity-dot { background: #d95c2c; box-shadow: 0 0 0 3px rgba(217,92,44,0.2); }
.severity-major .severity-text { color: #d95c2c; }
.severity-minor .severity-dot { background: #ff7d00; box-shadow: 0 0 0 3px rgba(255,125,0,0.2); }
.severity-minor .severity-text { color: #ff7d00; }
.severity-warning .severity-dot { background: #165dff; box-shadow: 0 0 0 3px rgba(22,93,255,0.15); }
.severity-warning .severity-text { color: #165dff; }

.label-tags { display: flex; flex-wrap: wrap; gap: 2px; }

/* 展开行样式 */
.silence-expand { padding: 12px 16px; background: #f7f8fa; border-radius: 6px; }
.expand-section { margin-bottom: 12px; }
.expand-section:last-child { margin-bottom: 0; }
.expand-title { font-size: 12px; font-weight: 600; color: var(--ops-text-secondary); margin-bottom: 6px; }
.expand-content { font-size: 12px; }
.info-row { display: flex; margin-bottom: 4px; }
.info-row:last-child { margin-bottom: 0; }
.info-label { color: var(--ops-text-tertiary); min-width: 80px; flex-shrink: 0; }
.info-value { color: var(--ops-text-primary); }

.labels-grid { display: flex; flex-wrap: wrap; gap: 6px; }
.label-kv {
  display: inline-flex; align-items: center;
  background: #fff; border: 1px solid #e5e6eb; border-radius: 4px; overflow: hidden;
  font-size: 12px;
}
.label-key { background: #f2f3f5; padding: 2px 6px; color: #4e5969; font-weight: 600; }
.label-val { padding: 2px 8px; color: #1d2129; font-family: monospace; }
</style>
