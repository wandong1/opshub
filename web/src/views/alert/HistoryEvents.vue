<template>
  <div class="page-container">
    <a-card :bordered="false">
      <template #title>历史告警查询</template>
      <a-form layout="inline" :model="{}" style="margin-bottom:16px;flex-wrap:wrap;gap:8px">
        <a-form-item label="时间范围">
          <a-range-picker v-model="timeRange" show-time format="YYYY-MM-DD HH:mm:ss" style="width:340px" />
        </a-form-item>
        <a-form-item label="严重级别">
          <a-select v-model="filterSeverity" placeholder="全部" allow-clear style="width:120px">
            <a-option value="critical">紧急 P1</a-option>
            <a-option value="major">严重 P2</a-option>
            <a-option value="minor">一般 P3</a-option>
            <a-option value="warning">提示 P4</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model="filterStatus" placeholder="全部" allow-clear style="width:100px">
            <a-option value="firing">告警中</a-option>
            <a-option value="resolved">已恢复</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="恢复方式">
          <a-select v-model="filterResolveType" placeholder="全部" allow-clear style="width:130px">
            <a-option value="auto">自动</a-option>
            <a-option value="manual_then_auto">人工介入后自动</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="规则名称">
          <a-input v-model="keyword" placeholder="搜索" allow-clear style="width:160px" />
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="load">查询</a-button>
            <a-button @click="reset">重置</a-button>
          </a-space>
        </a-form-item>
      </a-form>

      <a-table :data="list" :loading="loading" row-key="id"
        :pagination="{ total, pageSize, current: page, onChange: (p:number)=>{page=p;load()} }"
        :bordered="false" stripe
        :expandable="{ expandRowByClick: true, defaultExpandAllRows: false }">
        <!-- 展开行：Labels 详情 -->
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
            <div v-if="record.handledNote" style="margin-top:8px">
              <div class="labels-expand-title">介入备注</div>
              <div style="font-size:12px;color:var(--ops-text-secondary);padding:4px 0">{{ record.handledNote }}</div>
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
              <a-link @click="goRule(record.alertRuleId)" style="font-weight:600">{{ record.ruleName }}</a-link>
              <div v-if="record.ruleGroupName" style="font-size:11px;color:var(--ops-text-tertiary);margin-top:1px">{{ record.ruleGroupName }}</div>
            </template>
          </a-table-column>
          <!-- 业务分组 -->
          <a-table-column title="业务分组" :width="110">
            <template #cell="{ record }">
              <span style="font-size:12px;color:var(--ops-text-secondary)">{{ record.assetGroupName || '—' }}</span>
            </template>
          </a-table-column>
          <!-- 状态 -->
          <a-table-column title="状态" :width="90">
            <template #cell="{ record }">
              <a-badge :status="record.status==='firing'?'danger':'success'" :text="record.status==='firing'?'告警中':'已恢复'" />
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
          <!-- 触发时间 -->
          <a-table-column title="触发时间" :width="158">
            <template #cell="{ record }">
              <span style="font-size:12px;color:var(--ops-text-secondary)">{{ fmtTime(record.firedAt) }}</span>
            </template>
          </a-table-column>
          <!-- 恢复时间 -->
          <a-table-column title="恢复时间" :width="158">
            <template #cell="{ record }">
              <span style="font-size:12px;color:var(--ops-text-secondary)">{{ record.resolvedAt ? fmtTime(record.resolvedAt) : '—' }}</span>
            </template>
          </a-table-column>
          <!-- 持续时长 -->
          <a-table-column title="持续" :width="90">
            <template #cell="{ record }">
              <span class="duration-cell">{{ calcDuration(record.firedAt, record.resolvedAt) }}</span>
            </template>
          </a-table-column>
          <!-- 触发值/恢复值 -->
          <a-table-column title="触发值" :width="90">
            <template #cell="{ record }">
              <span class="val-cell">{{ record.value?.toFixed(4) }}</span>
            </template>
          </a-table-column>
          <a-table-column title="恢复值" :width="90">
            <template #cell="{ record }">
              <span style="font-size:13px;color:#00b42a;font-weight:600;font-family:monospace">{{ record.resolveValue != null ? record.resolveValue.toFixed(4) : '—' }}</span>
            </template>
          </a-table-column>
          <!-- 恢复方式 -->
          <a-table-column title="恢复方式" :width="120">
            <template #cell="{ record }">
              <a-tag v-if="record.resolveType === 'auto'" color="green" size="small">自动恢复</a-tag>
              <a-tag v-else-if="record.resolveType === 'manual_then_auto'" color="orange" size="small">人工介入后自动</a-tag>
              <a-tag v-else-if="record.resolveType === 'manual'" color="purple" size="small">手动恢复</a-tag>
              <span v-else style="color:#c9cdd4">—</span>
            </template>
          </a-table-column>
          <!-- 介入标记 -->
          <a-table-column title="介入" :width="70">
            <template #cell="{ record }">
              <a-tooltip v-if="record.manualHandled" content="已有人工介入记录">
                <icon-user style="color:#ff7d00;font-size:16px" />
              </a-tooltip>
              <span v-else style="color:#c9cdd4">—</span>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getHistoryEvents } from '@/api/alert'

const router = useRouter()
const list = ref<any[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const timeRange = ref<[Date, Date] | null>(null)
const filterSeverity = ref('')
const filterStatus = ref('')
const filterResolveType = ref('')
const keyword = ref('')

const fmtTime = (s?: string) => {
  if (!s) return '—'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const parseLabels = (s?: string): {k:string,v:string}[] => {
  if (!s) return []
  try {
    const obj = JSON.parse(s)
    if (typeof obj !== 'object' || Array.isArray(obj)) return []
    return Object.entries(obj).map(([k, v]) => ({ k, v: String(v) }))
  } catch { return [] }
}

const calcDuration = (firedAt: string, resolvedAt?: string) => {
  if (!firedAt) return '—'
  const end = resolvedAt ? new Date(resolvedAt).getTime() : Date.now()
  const m = Math.floor((end - new Date(firedAt).getTime()) / 60000)
  if (m < 60) return `${m}分钟`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}小时${m%60}分`
  return `${Math.floor(h/24)}天${h%24}小时`
}

const goRule = (ruleId?: number) => {
  if (!ruleId) return
  router.push('/alert/rules')
}

const load = async () => {
  loading.value = true
  try {
    const params: any = { page: page.value, pageSize: pageSize.value, keyword: keyword.value }
    if (filterSeverity.value) params.severity = filterSeverity.value
    if (filterStatus.value) params.status = filterStatus.value
    if (filterResolveType.value) params.resolveType = filterResolveType.value
    if (timeRange.value?.[0]) params.startTime = timeRange.value[0].toISOString()
    if (timeRange.value?.[1]) params.endTime = timeRange.value[1].toISOString()
    const d = await getHistoryEvents(params) as any
    list.value = Array.isArray(d) ? d : (d?.data || [])
    total.value = Array.isArray(d) ? d.length : (d?.total || 0)
  } finally { loading.value = false }
}

const reset = () => {
  timeRange.value = null; filterSeverity.value = ''; filterStatus.value = ''
  filterResolveType.value = ''; keyword.value = ''; page.value = 1; load()
}

onMounted(load)
</script>

<style scoped>
.page-container { padding: 20px; background: var(--ops-content-bg); min-height: 100%; }

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
.label-tags { display: flex; flex-wrap: wrap; gap: 2px; }

.labels-expand { padding: 12px 16px; background: #f7f8fa; border-radius: 6px; }
.labels-expand-title { font-size: 12px; font-weight: 600; color: var(--ops-text-secondary); margin-bottom: 6px; }
.labels-grid { display: flex; flex-wrap: wrap; gap: 6px; }
.label-kv {
  display: inline-flex; align-items: center;
  background: #fff; border: 1px solid #e5e6eb; border-radius: 4px; overflow: hidden;
  font-size: 12px;
}
.label-key { background: #f2f3f5; padding: 2px 6px; color: #4e5969; font-weight: 600; }
.label-val { padding: 2px 8px; color: #1d2129; font-family: monospace; }
</style>
