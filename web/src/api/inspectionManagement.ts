import request from '@/utils/request'

// ==================== 类型定义 ====================

export interface InspectionGroup {
  id?: number
  name: string
  description: string
  status: string
  executionMode: string
  executionStrategy?: string
  concurrency?: number
  prometheusUrl?: string
  prometheusUsername?: string
  prometheusPassword?: string
  groupIds?: number[]
  customVariables?: string  // JSON 字符串格式的自定义变量
  itemCount?: number
  itemNames?: string[]
  createdAt?: string
  updatedAt?: string
}

export interface InspectionItem {
  id?: number
  groupId: number
  name: string
  description: string
  executionType: string
  executionStrategy: string
  command?: string
  scriptType?: string
  scriptContent?: string
  scriptFile?: string
  scriptArgs?: string
  promqlQuery?: string
  probeCategory?: string
  probeType?: string
  probeConfigId?: number
  hostMatchType: string
  hostTags?: string[]
  hostIds?: number[]
  assertionType?: string
  assertionValue?: string
  variableName?: string
  variableRegex?: string
  timeout: number
  status: string
  sort: number
  createdAt?: string
  updatedAt?: string
}

export interface InspectionRecord {
  id: number
  taskId?: number
  groupId: number
  groupName: string
  itemId: number
  itemName: string
  hostId: number
  hostName: string
  status: string
  output: string
  errorMessage?: string
  duration: number
  assertionResult?: string
  assertionDetails?: any
  extractedVariables?: any
  executedAt: string
}

// 巡检执行记录（主表）
export interface ExecutionRecord {
  id: number
  taskId: number
  taskName: string
  totalItems: number
  totalHosts: number
  totalExecutions: number
  successCount: number
  failedCount: number
  assertionPassCount: number
  assertionFailCount: number
  assertionSkipCount: number
  status: string
  duration: number
  startedAt: string
  completedAt?: string
  groupNames: string[]
  createdAt: string
}

// 巡检执行明细
export interface ExecutionDetail {
  id: number
  executionId: number
  groupId: number
  groupName: string
  itemId: number
  itemName: string
  hostId: number
  hostName: string
  hostIp: string
  status: string
  output: string
  errorMessage: string
  duration: number
  assertionResult: string
  assertionDetails: string
  extractedVariables: string
  executedAt: string
}

export interface TestRunRequest {
  groupId: number
  itemIds?: number[]
}

export interface TestRunResult {
  success: boolean
  message: string
  results: InspectionRecord[]
}

// 拨测配置接口
export interface ProbeConfig {
  id: number
  name: string
  type: string
  category: string
  target: string
  port: number
  groupId: number
  groupIds: string
  description: string
  status: number
}

// ==================== 巡检组 API ====================

/**
 * 获取巡检组列表
 */
export function getInspectionGroups(params: {
  keyword?: string
  status?: string
  page?: number
  pageSize?: number
}) {
  return request<{
    total: number
    list: InspectionGroup[]
    page: number
    pageSize: number
  }>({
    url: '/api/v1/inspection/groups',
    method: 'get',
    params
  })
}

/**
 * 获取巡检组详情
 */
export function getInspectionGroup(id: number) {
  return request<InspectionGroup>({
    url: `/api/v1/inspection/groups/${id}`,
    method: 'get'
  })
}

/**
 * 创建巡检组
 */
export function createInspectionGroup(data: InspectionGroup) {
  return request<{ id: number }>({
    url: '/api/v1/inspection/groups',
    method: 'post',
    data
  })
}

/**
 * 更新巡检组
 */
export function updateInspectionGroup(id: number, data: InspectionGroup) {
  return request({
    url: `/api/v1/inspection/groups/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除巡检组
 */
export function deleteInspectionGroup(id: number) {
  return request({
    url: `/api/v1/inspection/groups/${id}`,
    method: 'delete'
  })
}

/**
 * 获取所有巡检组（用于下拉选择）
 */
export function getAllInspectionGroups() {
  return request<InspectionGroup[]>({
    url: '/api/v1/inspection/groups/all',
    method: 'get'
  })
}

// ==================== 巡检项 API ====================

/**
 * 获取巡检项列表
 */
export function getInspectionItems(params: {
  groupId?: number
  keyword?: string
  status?: string
  page?: number
  pageSize?: number
}) {
  return request<{
    total: number
    list: InspectionItem[]
    page: number
    pageSize: number
  }>({
    url: '/api/v1/inspection/items',
    method: 'get',
    params
  })
}

/**
 * 获取巡检项详情
 */
export function getInspectionItem(id: number) {
  return request<InspectionItem>({
    url: `/api/v1/inspection/items/${id}`,
    method: 'get'
  })
}

/**
 * 创建巡检项
 */
export function createInspectionItem(data: InspectionItem) {
  return request<{ id: number }>({
    url: '/api/v1/inspection/items',
    method: 'post',
    data
  })
}

/**
 * 更新巡检项
 */
export function updateInspectionItem(id: number, data: InspectionItem) {
  return request({
    url: `/api/v1/inspection/items/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除巡检项
 */
export function deleteInspectionItem(id: number) {
  return request({
    url: `/api/v1/inspection/items/${id}`,
    method: 'delete'
  })
}

/**
 * 批量创建/更新巡检项
 */
export function batchSaveInspectionItems(groupId: number, items: InspectionItem[]) {
  return request({
    url: `/api/v1/inspection/groups/${groupId}/items`,
    method: 'post',
    data: { items }
  })
}

/**
 * 测试运行巡检项
 */
export function testRunInspection(data: TestRunRequest) {
  return request<TestRunResult>({
    url: '/api/v1/inspection/items/test-run',
    method: 'post',
    data
  })
}

// ==================== 执行记录 API（旧版，保留兼容） ====================

/**
 * 获取执行记录列表
 */
export function getInspectionRecords(params: {
  groupId?: number
  itemId?: number
  hostId?: number
  status?: string
  startTime?: string
  endTime?: string
  page?: number
  pageSize?: number
}) {
  return request<{
    total: number
    list: InspectionRecord[]
    page: number
    pageSize: number
  }>({
    url: '/api/v1/inspection/records',
    method: 'get',
    params
  })
}

/**
 * 获取执行记录详情
 */
export function getInspectionRecord(id: number) {
  return request<InspectionRecord>({
    url: `/api/v1/inspection/records/${id}`,
    method: 'get'
  })
}

/**
 * 删除执行记录
 */
export function deleteInspectionRecord(id: number) {
  return request({
    url: `/api/v1/inspection/records/${id}`,
    method: 'delete'
  })
}

/**
 * 导出巡检记录为 Excel
 */
export function exportInspectionRecord(id: number) {
  return `/api/v1/inspection/records/${id}/export`
}

// ==================== 巡检执行记录 API（新版） ====================

/**
 * 获取巡检执行记录列表
 */
export function getExecutionRecords(params: {
  taskId?: number
  status?: string
  startTime?: string
  endTime?: string
  page?: number
  pageSize?: number
}) {
  return request<{
    total: number
    list: ExecutionRecord[]
    page: number
    pageSize: number
  }>({
    url: '/api/v1/inspection/execution-records',
    method: 'get',
    params
  })
}

/**
 * 获取巡检执行记录详情
 */
export function getExecutionRecord(id: number) {
  return request<ExecutionRecord>({
    url: `/api/v1/inspection/execution-records/${id}`,
    method: 'get'
  })
}

/**
 * 获取巡检执行明细列表
 */
export function getExecutionDetails(id: number) {
  return request<ExecutionDetail[]>({
    url: `/api/v1/inspection/execution-records/${id}/details`,
    method: 'get'
  })
}

/**
 * 删除巡检执行记录
 */
export function deleteExecutionRecord(id: number) {
  return request({
    url: `/api/v1/inspection/execution-records/${id}`,
    method: 'delete'
  })
}

/**
 * 导出巡检执行报告为 Excel
 */
export function exportExecutionReport(id: number) {
  return `/api/v1/inspection/execution-records/${id}/export`
}

// ==================== 统计 API ====================

/**
 * 获取巡检统计数据
 */
export function getInspectionStats() {
  return request<{
    total: number
    enabled: number
    disabled: number
    items: number
  }>({
    url: '/api/v1/inspection/stats',
    method: 'get'
  })
}

/**
 * 导出巡检组配置
 */
export function exportInspectionGroup(id: number, format: 'json' | 'yaml' = 'json') {
  return request<string>({
    url: `/api/v1/inspection/groups/${id}/export`,
    method: 'get',
    params: { format },
    responseType: 'text'
  })
}

/**
 * 导出所有巡检组配置
 */
export function exportAllInspectionGroups(format: 'json' | 'yaml' = 'json') {
  return request<string>({
    url: '/api/v1/inspection/groups/export-all',
    method: 'get',
    params: { format },
    responseType: 'text'
  })
}

/**
 * 导入巡检组配置
 */
export function importInspectionGroup(data: {
  format: 'json' | 'yaml'
  data: string
}) {
  return request<{ ids: number[]; count: number }>({
    url: '/api/v1/inspection/groups/import',
    method: 'post',
    data
  })
}

/**
 * 导入巡检组配置文件
 */
export function importInspectionGroupFile(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return request<{ ids: number[]; count: number }>({
    url: '/api/v1/inspection/groups/import-file',
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

// ==================== 拨测配置 API ====================

/**
 * 获取拨测配置列表（用于巡检项选择）
 */
export function getProbeConfigsForInspection(params: {
  groupIds: number[]
  category?: string
  type?: string
  status?: number
}) {
  return request<{
    total: number
    page: number
    page_size: number
    data: ProbeConfig[]
  }>({
    url: '/api/v1/inspection/probes',
    method: 'get',
    params: {
      groupId: params.groupIds.join(','),
      category: params.category,
      probeType: params.type,
      status: params.status !== undefined ? params.status : 1,
      page: 1,
      pageSize: 1000
    }
  })
}
