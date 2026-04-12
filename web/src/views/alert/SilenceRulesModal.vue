<template>
  <a-modal v-model:visible="visible" title="屏蔽规则管理" width="1400px" :footer="false" @cancel="handleClose">
    <!-- 过滤栏 -->
    <a-row :gutter="12" style="margin-bottom:16px">
      <a-col :span="5">
        <a-select v-model="filterSeverity" placeholder="告警等级" allow-clear @change="load">
          <a-option value="critical">P1-紧急</a-option>
          <a-option value="major">P2-重要</a-option>
          <a-option value="minor">P3-次要</a-option>
          <a-option value="warning">P4-警告</a-option>
        </a-select>
      </a-col>
      <a-col :span="5">
        <a-select v-model="filterType" placeholder="屏蔽类型" allow-clear @change="load">
          <a-option value="fixed">固定时长</a-option>
          <a-option value="periodic">周期性</a-option>
        </a-select>
      </a-col>
      <a-col :span="6">
        <a-input v-model="keyword" placeholder="搜索规则名称" allow-clear @press-enter="load" />
      </a-col>
      <a-col :span="4">
        <a-button type="primary" @click="load">查询</a-button>
      </a-col>
    </a-row>

    <!-- 批量操作栏 -->
    <div v-if="selectedIds.length > 0" class="batch-bar">
      <span class="batch-count">已选 <b>{{ selectedIds.length }}</b> 条</span>
      <a-space>
        <a-popconfirm content="确认删除选中的屏蔽规则？" @ok="batchDelete">
          <a-button size="small" status="danger">
            <template #icon><icon-delete /></template>批量删除
          </a-button>
        </a-popconfirm>
        <a-button size="small" type="text" @click="selectedIds = []">取消选择</a-button>
      </a-space>
    </div>

    <!-- 表格 -->
    <a-table :data="rules" :loading="loading"
      row-key="id"
      :row-selection="rowSelection"
      v-model:selectedKeys="selectedIds"
      :pagination="{ total, pageSize, current: page, onChange: onPageChange }"
      :expandable="{ expandRowByClick: true, defaultExpandAllRows: false }">
      <!-- 展开行：显示标签详情 -->
      <template #expand-row="{ record }">
        <div class="silence-expand">
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

          <div v-if="record.reason" class="expand-section">
            <div class="expand-title">屏蔽原因</div>
            <div class="expand-content">{{ record.reason }}</div>
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
            <a-tag v-if="record.type === 'periodic'" color="blue" size="small">周期性</a-tag>
            <a-tag v-else color="purple" size="small">固定时长</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="屏蔽时间" :width="200">
          <template #cell="{ record }">
            <div v-if="record.type === 'fixed'" style="font-size:12px;color:var(--ops-text-secondary)">
              {{ record.silenceUntil ? '至 ' + fmtTime(record.silenceUntil) : '—' }}
            </div>
            <div v-else style="font-size:11px;color:var(--ops-text-secondary)">
              {{ formatTimeRangesShort(record.timeRanges) }}
            </div>
          </template>
        </a-table-column>
        <a-table-column title="状态" :width="80">
          <template #cell="{ record }">
            <a-badge v-if="record.enabled" status="success" text="启用" />
            <a-badge v-else status="default" text="禁用" />
          </template>
        </a-table-column>
        <a-table-column title="创建时间" :width="160">
          <template #cell="{ record }">
            {{ fmtTime(record.createdAt) }}
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="160" fixed="right">
          <template #cell="{ record }">
            <a-space>
              <a-link @click="openEdit(record)">编辑</a-link>
              <a-link @click="toggleEnabled(record)">{{ record.enabled ? '禁用' : '启用' }}</a-link>
              <a-popconfirm content="确认删除该屏蔽规则？" @ok="deleteRule(record.id)">
                <a-link status="danger">删除</a-link>
              </a-popconfirm>
            </a-space>
          </template>
        </a-table-column>
      </template>
    </a-table>

    <!-- 编辑屏蔽规则弹窗 -->
    <a-modal v-model:visible="editVisible" title="编辑屏蔽规则" @ok="doEdit" width="600px">
      <a-form layout="vertical" :model="editForm">
        <a-form-item label="屏蔽类型">
          <a-radio-group v-model="editForm.type" type="button">
            <a-radio value="fixed">固定时长</a-radio>
            <a-radio value="periodic">周期性</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="editForm.type === 'fixed'" label="屏蔽时长">
          <a-radio-group v-model="editForm.duration" type="button">
            <a-radio value="1h">1小时</a-radio>
            <a-radio value="2h">2小时</a-radio>
            <a-radio value="6h">6小时</a-radio>
            <a-radio value="12h">12小时</a-radio>
            <a-radio value="24h">1天</a-radio>
            <a-radio value="168h">1周</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="editForm.type === 'periodic'" label="生效时间段">
          <a-checkbox-group v-model="editForm.weekdays">
            <a-checkbox :value="1">周一</a-checkbox>
            <a-checkbox :value="2">周二</a-checkbox>
            <a-checkbox :value="3">周三</a-checkbox>
            <a-checkbox :value="4">周四</a-checkbox>
            <a-checkbox :value="5">周五</a-checkbox>
            <a-checkbox :value="6">周六</a-checkbox>
            <a-checkbox :value="7">周日</a-checkbox>
          </a-checkbox-group>
          <a-space style="margin-top:8px">
            <a-time-picker v-model="editForm.start" format="HH:mm" />
            <span>至</span>
            <a-time-picker v-model="editForm.end" format="HH:mm" />
          </a-space>
        </a-form-item>

        <a-form-item label="屏蔽原因">
          <a-textarea v-model="editForm.reason" :auto-size="{minRows:2}" placeholder="请输入屏蔽原因" />
        </a-form-item>
      </a-form>
    </a-modal>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getSilenceRules, updateSilenceRule, deleteSilenceRule, toggleSilenceRule } from '@/api/alert'

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

const rules = ref<any[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filterSeverity = ref('')
const filterType = ref('')
const keyword = ref('')

// 批量选择
const selectedIds = ref<number[]>([])
const rowSelection = computed(() => ({
  type: 'checkbox' as const,
  showCheckedAll: true
}))

// 编辑弹窗
const editVisible = ref(false)
const editForm = ref({
  id: 0,
  type: 'fixed',
  duration: '2h',
  weekdays: [1, 2, 3, 4, 5],
  start: '09:00',
  end: '18:00',
  reason: ''
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

const formatTimeRangesShort = (timeRangesJson?: string): string => {
  if (!timeRangesJson) return '—'
  try {
    const ranges = JSON.parse(timeRangesJson)
    if (!Array.isArray(ranges) || ranges.length === 0) return '—'
    const range = ranges[0]
    const weekdayMap: Record<number, string> = {1:'一',2:'二',3:'三',4:'四',5:'五',6:'六',7:'日'}
    const weekdays = (range.weekdays || []).map((d: number) => weekdayMap[d]).join(',')
    return `周${weekdays} ${range.start || ''}-${range.end || ''}`
  } catch { return '—' }
}

const load = async () => {
  loading.value = true
  try {
    const res = await getSilenceRules({
      page: page.value,
      pageSize: pageSize.value
    }) as any

    let list = res?.data || []

    // 前端过滤
    if (filterSeverity.value) {
      list = list.filter((r: any) => r.severity === filterSeverity.value)
    }
    if (filterType.value) {
      list = list.filter((r: any) => r.type === filterType.value)
    }
    if (keyword.value) {
      list = list.filter((r: any) => r.ruleName && r.ruleName.includes(keyword.value))
    }

    rules.value = list
    total.value = list.length
  } catch (err) {
    console.error('[屏蔽规则] 查询失败', err)
    Message.error('查询失败')
  } finally {
    loading.value = false
  }
}

const onPageChange = (p: number) => {
  page.value = p
  load()
}

const openEdit = (record: any) => {
  editForm.value.id = record.id
  editForm.value.type = record.type
  editForm.value.reason = record.reason || ''

  if (record.type === 'fixed') {
    editForm.value.duration = record.duration || '2h'
  } else if (record.type === 'periodic' && record.timeRanges) {
    try {
      const ranges = JSON.parse(record.timeRanges)
      if (ranges && ranges.length > 0) {
        editForm.value.weekdays = ranges[0].weekdays || [1, 2, 3, 4, 5]
        editForm.value.start = ranges[0].start || '09:00'
        editForm.value.end = ranges[0].end || '18:00'
      }
    } catch (e) {
      console.error('解析时间范围失败', e)
    }
  }

  editVisible.value = true
}

const doEdit = async () => {
  try {
    const data: any = {
      type: editForm.value.type,
      reason: editForm.value.reason
    }

    if (editForm.value.type === 'fixed') {
      data.duration = editForm.value.duration
    } else {
      data.timeRanges = JSON.stringify([{
        weekdays: editForm.value.weekdays,
        start: editForm.value.start,
        end: editForm.value.end
      }])
    }

    await updateSilenceRule(editForm.value.id, data)
    Message.success('修改成功')
    editVisible.value = false
    load()
    emit('refresh')
  } catch (err) {
    console.error('[屏蔽规则] 修改失败', err)
    Message.error('修改失败')
  }
}

const toggleEnabled = async (record: any) => {
  try {
    await toggleSilenceRule(record.id)
    Message.success(record.enabled ? '已禁用' : '已启用')
    load()
    emit('refresh')
  } catch (err) {
    Message.error('操作失败')
  }
}

const deleteRule = async (id: number) => {
  try {
    await deleteSilenceRule(id)
    Message.success('删除成功')
    load()
    emit('refresh')
  } catch (err) {
    Message.error('删除失败')
  }
}

const batchDelete = async () => {
  try {
    await Promise.all(selectedIds.value.map(id => deleteSilenceRule(id)))
    Message.success(`已删除 ${selectedIds.value.length} 条屏蔽规则`)
    selectedIds.value = []
    load()
    emit('refresh')
  } catch (err) {
    Message.error('批量删除失败')
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
.expand-content { font-size: 12px; color: var(--ops-text-primary); }

.labels-grid { display: flex; flex-wrap: wrap; gap: 6px; }
.label-kv {
  display: inline-flex; align-items: center;
  background: #fff; border: 1px solid #e5e6eb; border-radius: 4px; overflow: hidden;
  font-size: 12px;
}
.label-key { background: #f2f3f5; padding: 2px 6px; color: #4e5969; font-weight: 600; }
.label-val { padding: 2px 8px; color: #1d2129; font-family: monospace; }
</style>
