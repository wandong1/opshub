<template>
  <div class="page-container">
    <a-card :bordered="false">
      <template #title>告警订阅管理</template>
      <template #extra>
        <a-button type="primary" @click="openCreate"><template #icon><icon-plus /></template>新增订阅</a-button>
      </template>
      <a-table :data="list" :loading="loading" row-key="id">
        <template #columns>
          <a-table-column title="名称" data-index="name" />
          <a-table-column title="描述" data-index="description" :ellipsis="true" />
          <a-table-column title="规则数" :width="80">
            <template #cell="{ record }"><a-tag>{{ record.ruleCount }}</a-tag></template>
          </a-table-column>
          <a-table-column title="通道数" :width="80">
            <template #cell="{ record }"><a-tag>{{ record.channelCount }}</a-tag></template>
          </a-table-column>
          <a-table-column title="启用" :width="80">
            <template #cell="{ record }">
              <a-switch :model-value="record.enabled" size="small" @change="(v) => quickToggle(record, !!v)" />
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="120">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="openEdit(record)">编辑</a-link>
                <a-popconfirm content="确认删除？" @ok="remove(record.id)">
                  <a-link status="danger">删除</a-link>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 编辑弹窗 -->
    <a-modal v-model:visible="modalVisible" :title="form.id?'编辑订阅':'新增订阅'"
      @ok="save" @cancel="modalVisible=false" width="1000px" :mask-closable="false">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="订阅名称" required><a-input v-model="form.name" /></a-form-item></a-col>
          <a-col :span="12">
            <a-form-item label="业务分组">
              <a-select v-model="form.assetGroupId" allow-clear placeholder="不限">
                <a-option v-for="g in flatGroups" :key="g.id" :value="g.id">{{ g.name }}</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="16"><a-form-item label="描述"><a-input v-model="form.description" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="启用"><a-switch v-model="form.enabled" /></a-form-item></a-col>
        </a-row>

        <a-divider orientation="left">推送规则配置</a-divider>
        <div style="margin-bottom:8px;color:var(--ops-text-secondary);font-size:12px">
          每行为一条推送规则：选择触发规则、告警级别、生效时间，并独立配置该规则的通知通道和接收用户。留空规则 = 全部规则，留空级别 = 全部级别。
        </div>
        <div style="margin-bottom:10px">
          <a-button size="small" type="outline" @click="addRuleGroup">
            <template #icon><icon-plus /></template>添加推送规则
          </a-button>
        </div>
        <div class="rule-group-list">
          <div v-for="(group, idx) in ruleGroups" :key="idx" class="rule-group-item">
            <!-- 行头：编号 + 时间配置 + 删除 -->
            <div class="rule-group-header">
              <span style="font-weight:500;color:var(--ops-text-secondary);font-size:13px">规则 {{ idx + 1 }}</span>
              <a-space style="margin-left:auto">
                <a-button size="mini" @click="openTimeConfig(idx)">
                  <template #icon><icon-clock-circle /></template>
                  {{ timeRangeSummary(group.timeRanges) }}
                </a-button>
                <a-button size="mini" status="danger" @click="removeRuleGroup(idx)">
                  <template #icon><icon-delete /></template>
                </a-button>
              </a-space>
            </div>
            <!-- 触发规则 -->
            <a-row :gutter="8" style="margin-top:8px">
              <a-col :span="24">
                <div style="font-size:12px;color:var(--ops-text-secondary);margin-bottom:4px">触发规则（留空=全部）</div>
                <div style="display:flex;align-items:center;gap:6px">
                  <a-select v-model="group.ruleIds" multiple :max-tag-count="4"
                    placeholder="选择规则（可多选，留空=全部规则）"
                    allow-search allow-clear style="flex:1">
                    <a-option v-for="r in allRules" :key="r.id" :value="r.id">
                      <a-tag :color="sevColor(r.severity)" size="small" style="margin-right:4px">{{ sevLabel(r.severity) }}</a-tag>
                      {{ r.name }}
                    </a-option>
                  </a-select>
                  <a-button size="small" type="outline" @click="group.ruleIds = allRules.map(r => r.id)">全选</a-button>
                  <a-button size="small" type="outline" @click="group.ruleIds = []">清空</a-button>
                </div>
              </a-col>
            </a-row>
            <!-- 告警级别 -->
            <div style="margin-top:8px">
              <div style="font-size:12px;color:var(--ops-text-secondary);margin-bottom:4px">告警级别（留空=全部级别）</div>
              <a-checkbox-group v-model="group.severities" style="display:flex;flex-wrap:wrap;gap:6px">
                <a-checkbox value="critical"><a-tag color="red" size="small">紧急(P1)</a-tag></a-checkbox>
                <a-checkbox value="major"><a-tag color="orangered" size="small">严重(P2)</a-tag></a-checkbox>
                <a-checkbox value="minor"><a-tag color="orange" size="small">一般(P3)</a-tag></a-checkbox>
                <a-checkbox value="warning"><a-tag color="arcoblue" size="small">提示(P4)</a-tag></a-checkbox>
              </a-checkbox-group>
            </div>
            <!-- 通知通道 -->
            <div style="margin-top:8px">
              <div style="font-size:12px;color:var(--ops-text-secondary);margin-bottom:4px">通知通道</div>
              <a-checkbox-group v-model="group.channelIds" style="display:flex;flex-wrap:wrap;gap:6px">
                <a-checkbox v-for="ch in allChannels" :key="ch.id" :value="ch.id">
                  <a-tag :color="chTypeColor(ch.type)" size="small">{{ ch.name }}</a-tag>
                </a-checkbox>
              </a-checkbox-group>
              <div v-if="allChannels.length===0" style="color:var(--ops-text-tertiary);font-size:12px">暂无通道，请先在「告警通道」页面创建</div>
            </div>
            <!-- 接收用户 -->
            <div style="margin-top:8px">
              <div style="font-size:12px;color:var(--ops-text-secondary);margin-bottom:4px">接收用户（留空=@all）</div>
              <a-select v-model="group.userIds" multiple :max-tag-count="6"
                placeholder="选择接收用户（留空则通知全部人）"
                allow-search allow-clear style="width:100%">
                <a-option :value="0" label="📢 所有人" />
                <a-option v-for="u in allUsers" :key="u.ID" :value="u.ID"
                  :label="u.realName ? u.realName + '(' + u.username + ')' : u.username" />
              </a-select>
            </div>
          </div>
          <div v-if="ruleGroups.length===0" style="color:var(--ops-text-tertiary);padding:12px;border:1px dashed var(--ops-border-color);border-radius:4px;text-align:center">
            暂无配置，点击「添加推送规则」
          </div>
        </div>
      </a-form>
    </a-modal>

    <!-- 生效时间配置弹窗 -->
    <a-modal v-model:visible="timeConfigVisible" title="配置生效时间" @ok="applyTimeConfig" @cancel="timeConfigVisible=false" width="580px">
      <div style="margin-bottom:12px">
        <a-space>
          <a-button size="small" @click="addTimeRange"><template #icon><icon-plus /></template>添加时间段</a-button>
          <a-button size="small" type="outline" @click="setWorkdays">工作日 09-18</a-button>
          <a-button size="small" type="outline" @click="editingTimeRanges = []">全天全周</a-button>
        </a-space>
      </div>
      <div v-for="(tr, i) in editingTimeRanges" :key="i" class="time-range-item">
        <a-checkbox-group v-model="tr.weekdays" style="margin-bottom:8px">
          <a-checkbox v-for="wd in weekdays" :key="wd.v" :value="wd.v">{{ wd.l }}</a-checkbox>
        </a-checkbox-group>
        <a-space>
          <a-time-picker v-model="tr.start" format="HH:mm" style="width:120px" />
          <span>至</span>
          <a-time-picker v-model="tr.end" format="HH:mm" style="width:120px" />
          <a-button size="mini" status="danger" @click="editingTimeRanges.splice(i,1)"><template #icon><icon-delete /></template></a-button>
        </a-space>
      </div>
      <div v-if="editingTimeRanges.length===0" style="color:var(--ops-text-tertiary)">空 = 全天全周生效</div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  getSubscriptions, createSubscription, updateSubscription, deleteSubscription, getSubscription,
  getRules, getChannels,
  type AlertSubscription, type TimeRange
} from '@/api/alert'
import { getGroupTree } from '@/api/assetGroup'
import { getUserList } from '@/api/user'

const weekdays = [
  { v: 1, l: '周一' }, { v: 2, l: '周二' }, { v: 3, l: '周三' },
  { v: 4, l: '周四' }, { v: 5, l: '周五' }, { v: 6, l: '周六' }, { v: 7, l: '周日' }
]

interface RuleGroup {
  ruleIds: number[]
  severities: string[]
  channelIds: number[]
  userIds: number[]
  timeRanges: TimeRange[]
}

const list = ref<any[]>([])
const loading = ref(false)
const modalVisible = ref(false)
const form = ref<Partial<AlertSubscription>>({ enabled: true })
const ruleGroups = ref<RuleGroup[]>([])
const flatGroups = ref<any[]>([])
const allRules = ref<any[]>([])
const allChannels = ref<any[]>([])
const allUsers = ref<any[]>([])
const timeConfigVisible = ref(false)
let editingGroupIdx = -1
const editingTimeRanges = ref<TimeRange[]>([])

const sevColor = (s: string) => ({ critical: 'red', major: 'orangered', minor: 'orange', warning: 'arcoblue' } as any)[s] || 'gray'
const sevLabel = (s: string) => ({ critical: '紧急P1', major: '严重P2', minor: '一般P3', warning: '提示P4' } as any)[s] || s
const chTypeColor = (t: string) => ({ wechat_work: 'green', dingtalk: 'blue', sms: 'orange', phone: 'purple', ai_agent: 'cyan' } as any)[t] || 'gray'

const parseTimeRanges = (v: any): TimeRange[] => {
  if (!v) return []
  if (Array.isArray(v)) return v
  try { const p = JSON.parse(v); return Array.isArray(p) ? p : [] } catch { return [] }
}

const parseNumList = (v: any): number[] => {
  if (!v) return []
  if (Array.isArray(v)) return v
  try { const p = JSON.parse(v); return Array.isArray(p) ? p : [] } catch { return [] }
}

const parseStrList = (v: any): string[] => {
  if (!v) return []
  if (Array.isArray(v)) return v
  try { const p = JSON.parse(v); return Array.isArray(p) ? p : [] } catch { return [] }
}

const timeRangeSummary = (ranges: TimeRange[]) => {
  if (!ranges || ranges.length === 0) return '全天全周'
  const dayNames = ['', '一', '二', '三', '四', '五', '六', '日']
  return ranges.map(r => {
    const days = (r.weekdays || []).map((d: number) => '周' + dayNames[d]).join('/')
    return `${days} ${r.start}-${r.end}`
  }).join('；')
}

const flattenGroups = (nodes: any[], result: any[] = []) => {
  for (const n of nodes) {
    result.push(n)
    if (n.children?.length) flattenGroups(n.children, result)
  }
  return result
}

const load = async () => {
  loading.value = true
  try { const res = await getSubscriptions(); list.value = res?.data || res || [] }
  finally { loading.value = false }
}

const remove = async (id: number) => {
  try { await deleteSubscription(id); Message.success('删除成功'); load() }
  catch { Message.error('删除失败') }
}

const quickToggle = async (row: any, v: boolean) => {
  await updateSubscription(row.id, { ...row, enabled: v }).catch(() => {}); load()
}

const newGroup = (): RuleGroup => ({ ruleIds: [], severities: [], channelIds: [], userIds: [], timeRanges: [] })

const openCreate = () => {
  form.value = { enabled: true }
  ruleGroups.value = [newGroup()]
  modalVisible.value = true
}

const openEdit = async (row: any) => {
  form.value = { id: row.id, name: row.name, description: row.description, enabled: row.enabled, assetGroupId: row.assetGroupId }
  try {
    const res = await getSubscription(row.id)
    const d = res?.data || res || {}
    const flatRules: any[] = d.rules || []
    if (flatRules.length > 0) {
      ruleGroups.value = flatRules.map((r: any) => ({
        ruleIds: r.ruleIds || (r.ruleId ? [r.ruleId] : []),
        severities: parseStrList(r.severities),
        channelIds: parseNumList(r.channelIds),
        userIds: parseNumList(r.userIds),
        timeRanges: parseTimeRanges(r.timeRanges)
      }))
    } else {
      ruleGroups.value = [newGroup()]
    }
  } catch {
    ruleGroups.value = [newGroup()]
  }
  modalVisible.value = true
}

const save = async () => {
  if (!form.value.name) { Message.warning('请输入订阅名称'); return }
  const rules = ruleGroups.value.flatMap(g => {
    const base = {
      severities: g.severities,
      channelIds: g.channelIds,
      userIds: g.userIds,
      timeRanges: g.timeRanges
    }
    if (g.ruleIds.length === 0) return [{ ruleId: 0, ...base }]
    return g.ruleIds.map(rid => ({ ruleId: rid, ...base }))
  })
  const payload: any = { ...form.value, rules, channelIds: [], userIds: [] }
  try {
    if (form.value.id) { await updateSubscription(form.value.id, payload) }
    else { await createSubscription(payload) }
    Message.success('保存成功'); modalVisible.value = false; load()
  } catch { Message.error('保存失败') }
}

const addRuleGroup = () => ruleGroups.value.push(newGroup())
const removeRuleGroup = (idx: number) => ruleGroups.value.splice(idx, 1)

const openTimeConfig = (idx: number) => {
  editingGroupIdx = idx
  const g = ruleGroups.value[idx]
  editingTimeRanges.value = g ? JSON.parse(JSON.stringify(g.timeRanges || [])) : []
  timeConfigVisible.value = true
}

const applyTimeConfig = () => {
  const g = editingGroupIdx >= 0 ? ruleGroups.value[editingGroupIdx] : undefined
  if (g) {
    g.timeRanges = JSON.parse(JSON.stringify(editingTimeRanges.value))
  }
  timeConfigVisible.value = false
}

const addTimeRange = () => editingTimeRanges.value.push({ weekdays: [1,2,3,4,5], start: '09:00', end: '18:00' })
const setWorkdays = () => { editingTimeRanges.value = [{ weekdays: [1,2,3,4,5], start: '09:00', end: '18:00' }] }

onMounted(async () => {
  const [treeRes, rulesRes, chRes, usersRes] = await Promise.all([
    getGroupTree(), getRules({ page: 1, pageSize: 1000 }), getChannels(), getUserList({ page: 1, pageSize: 1000 })
  ])
  flatGroups.value = flattenGroups(treeRes?.data || treeRes || [])
  const rulesData = rulesRes?.data || rulesRes || {}
  allRules.value = rulesData.data || (Array.isArray(rulesData) ? rulesData : [])
  allChannels.value = chRes?.data || chRes || []
  const usersData = usersRes?.data || usersRes || {}
  allUsers.value = (usersData.list || usersData.data || (Array.isArray(usersData) ? usersData : [])) as any[]
  load()
})
</script>

<style scoped>
.page-container { padding: 20px; background: var(--ops-content-bg); min-height: 100%; }
.rule-group-list { border: 1px solid var(--ops-border-color); border-radius: 6px; padding: 8px; }
.rule-group-item {
  padding: 10px 12px;
  margin-bottom: 8px;
  background: #f7f8fa;
  border-radius: 4px;
  border: 1px solid var(--ops-border-color);
}
.rule-group-item:last-child { margin-bottom: 0; }
.rule-group-header { display: flex; align-items: center; }
.time-range-item { padding: 12px; margin-bottom: 8px; background: #fff; border: 1px solid var(--ops-border-color); border-radius: 4px; }
</style>