import request from '@/utils/request'

// ==================== 数据源 ====================
export interface AlertDataSource {
  id?: number
  name: string
  type: string // prometheus | victoriametrics | influxdb
  url: string
  username?: string
  password?: string
  token?: string
  description?: string
  status?: number
  createdAt?: string
}

export const getDataSources = () => request.get('/api/v1/alert/datasources')
export const createDataSource = (data: Partial<AlertDataSource>) => request.post('/api/v1/alert/datasources', data)
export const getDataSource = (id: number) => request.get(`/api/v1/alert/datasources/${id}`)
export const updateDataSource = (id: number, data: Partial<AlertDataSource>) => request.put(`/api/v1/alert/datasources/${id}`, data)
export const deleteDataSource = (id: number) => request.delete(`/api/v1/alert/datasources/${id}`)
export const testDataSource = (id: number) => request.post(`/api/v1/alert/datasources/${id}/test`)

// ==================== 规则分类 ====================
export interface AlertRuleGroup {
  id?: number
  name: string
  assetGroupId: number
  description?: string
  sort?: number
}

export const getRuleGroups = (assetGroupId?: number) =>
  request.get('/api/v1/alert/rule-groups', { params: { assetGroupId } })
export const createRuleGroup = (data: Partial<AlertRuleGroup>) => request.post('/api/v1/alert/rule-groups', data)
export const updateRuleGroup = (id: number, data: Partial<AlertRuleGroup>) => request.put(`/api/v1/alert/rule-groups/${id}`, data)
export const deleteRuleGroup = (id: number) => request.delete(`/api/v1/alert/rule-groups/${id}`)

// ==================== 告警规则 ====================
export interface AlertRule {
  id?: number
  name: string
  description?: string
  assetGroupId?: number
  ruleGroupId?: number
  dataSourceId?: number        // 单数据源（兼容旧数据）
  dataSourceIds?: string       // JSON数组字符串 "[1,2,3]"
  expr: string
  evalInterval?: number // 秒，默认15
  duration?: string    // e.g. "5m"
  severity?: string    // critical | warning | info
  labels?: string      // JSON
  annotations?: string // JSON {title, description}
  enabled?: boolean
  notifyOnResolve?: boolean
  lastEvalAt?: string
  createdAt?: string
}

export interface RuleListParams {
  page?: number
  pageSize?: number
  assetGroupId?: number
  ruleGroupId?: number
  keyword?: string
  enabled?: boolean
}

export const getRules = (params?: RuleListParams) => request.get('/api/v1/alert/rules', { params })
export const createRule = (data: Partial<AlertRule>) => request.post('/api/v1/alert/rules', data)
export const getRule = (id: number) => request.get(`/api/v1/alert/rules/${id}`)
export const updateRule = (id: number, data: Partial<AlertRule>) => request.put(`/api/v1/alert/rules/${id}`, data)
export const deleteRule = (id: number) => request.delete(`/api/v1/alert/rules/${id}`)
export const toggleRule = (id: number) => request.put(`/api/v1/alert/rules/${id}/toggle`)
export const testRule = (id: number) => request.post(`/api/v1/alert/rules/${id}/test`)
export const cloneRule = (id: number) => request.post(`/api/v1/alert/rules/${id}/clone`)
export const exportRules = (ids?: number[], format = 'json') => {
  const params = new URLSearchParams()
  params.set('format', format)
  ids?.forEach(id => params.append('ids', String(id)))
  return request.get(`/api/v1/alert/rules/export?${params.toString()}`, { responseType: 'blob' })
}
export const adhocTestRule = (data: { dataSourceIds: number[], expr: string }) =>
  request.post('/api/v1/alert/rules/adhoc-test', data)

export const importRules = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return request.post('/api/v1/alert/rules/import', form, { headers: { 'Content-Type': 'multipart/form-data' } })
}

// ==================== 告警事件 ====================
export interface AlertEvent {
  id?: number
  alertRuleId: number
  ruleName?: string
  assetGroupId?: number
  fingerprint?: string
  severity: string
  status: string // firing | resolved
  labels?: string
  annotations?: string
  value?: number
  resolveValue?: number
  firedAt?: string
  resolvedAt?: string
  resolveType?: string // auto | manual
  silenced?: boolean
  silenceUntil?: string
  silenceReason?: string
  manualHandled?: boolean
  handledBy?: number
  handledAt?: string
  handledNote?: string
}

export interface EventListParams {
  page?: number
  pageSize?: number
  assetGroupId?: number
  severity?: string
  keyword?: string
  status?: string
  resolveType?: string
  startTime?: string
  endTime?: string
}

export const getActiveEvents = (params?: EventListParams) => request.get('/api/v1/alert/events/active', { params })
export const getHistoryEvents = (params?: EventListParams) => request.get('/api/v1/alert/events/history', { params })
export const silenceEvent = (id: number, data: { duration: string; reason?: string }) =>
  request.post(`/api/v1/alert/events/${id}/silence`, data)
export const handleEvent = (id: number, data: { note: string; userId?: number }) =>
  request.post(`/api/v1/alert/events/${id}/handle`, data)
export const getEventStats = () => request.get('/api/v1/alert/events/stats')
export const getEventTrend = (days = 30) => request.get('/api/v1/alert/events/trend', { params: { days } })

// ==================== 通知通道 ====================
export interface AlertNotifyChannel {
  id?: number
  name: string
  type: string // wechat_work | dingtalk | sms | phone | ai_agent
  config?: string  // JSON
  alertTemplate?: string
  resolveTemplate?: string
  enabled?: boolean
  aiHookEnabled?: boolean
}

export const getChannels = () => request.get('/api/v1/alert/channels')
export const createChannel = (data: Partial<AlertNotifyChannel>) => request.post('/api/v1/alert/channels', data)
export const getChannel = (id: number) => request.get(`/api/v1/alert/channels/${id}`)
export const updateChannel = (id: number, data: Partial<AlertNotifyChannel>) => request.put(`/api/v1/alert/channels/${id}`, data)
export const deleteChannel = (id: number) => request.delete(`/api/v1/alert/channels/${id}`)
export const testChannel = (id: number) => request.post(`/api/v1/alert/channels/${id}/test`)

// ==================== 告警订阅 ====================
export interface TimeRange {
  weekdays: number[] // 1=周一...7=周日
  start: string      // "08:00"
  end: string        // "18:00"
}

export interface SubscriptionRuleItem {
  ruleId: number
  timeRanges: TimeRange[]
}

export interface AlertSubscription {
  id?: number
  name: string
  assetGroupId?: number
  description?: string
  enabled?: boolean
  rules?: SubscriptionRuleItem[]
  channelIds?: number[]
  userIds?: number[]
  ruleCount?: number
  channelCount?: number
}

export const getSubscriptions = (assetGroupId?: number) =>
  request.get('/api/v1/alert/subscriptions', { params: { assetGroupId } })
export const createSubscription = (data: Partial<AlertSubscription>) => request.post('/api/v1/alert/subscriptions', data)
export const getSubscription = (id: number) => request.get(`/api/v1/alert/subscriptions/${id}`)
export const updateSubscription = (id: number, data: Partial<AlertSubscription>) =>
  request.put(`/api/v1/alert/subscriptions/${id}`, data)
export const deleteSubscription = (id: number) => request.delete(`/api/v1/alert/subscriptions/${id}`)
