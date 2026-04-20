import request from '@/utils/request'

// 巡检项断言覆盖结构
export interface ItemAssertionOverride {
  item_id: number
  assertion_type: string
  assertion_value: string
}

// 巡检组业务分组覆盖结构
export interface GroupBusinessGroupOverride {
  group_id: number
  business_group_id: number
}

// 巡检任务接口
export interface InspectionTask {
  id: number
  name: string
  description: string
  taskType: string
  cronExpr: string
  enabled: boolean
  groupIds: number[]
  itemIds: number[]
  pushgatewayId?: number
  concurrency: number
  executionMode?: string
  agentHostIds?: string
  businessGroupId?: number
  customVariables?: string
  itemAssertionOverrides?: string
  groupBusinessGroupOverrides?: string
  createdAt: string
  updatedAt: string
}

// 获取巡检任务列表（包含拨测和巡检任务）
export function getInspectionTasks(params: any) {
  return request.get('/api/v1/inspection/mgmt-tasks', { params })
}

// 创建巡检任务
export function createInspectionTask(data: any) {
  return request.post('/api/v1/inspection/mgmt-tasks', data)
}

// 更新巡检任务
export function updateInspectionTask(id: number, data: any) {
  return request.put(`/api/v1/inspection/mgmt-tasks/${id}`, data)
}

// 删除巡检任务
export function deleteInspectionTask(id: number) {
  return request.delete(`/api/v1/inspection/mgmt-tasks/${id}`)
}

// 切换巡检任务状态
export function toggleInspectionTask(id: number) {
  return request.put(`/api/v1/inspection/mgmt-tasks/${id}/toggle`)
}

// 获取巡检任务详情
export function getInspectionTask(id: number) {
  return request.get(`/api/v1/inspection/mgmt-tasks/${id}`)
}

// 获取巡检任务执行结果
export function getInspectionTaskResults(taskId: number, params: any) {
  return request.get(`/api/v1/inspection/records`, {
    params: {
      ...params,
      task_id: taskId
    }
  })
}

// 手动运行任务
export function runInspectionTask(id: number) {
  return request.post(`/api/v1/inspection/mgmt-tasks/${id}/run`)
}

// 停止手动运行的任务
export function stopInspectionTask(id: number) {
  return request.post(`/api/v1/inspection/mgmt-tasks/${id}/stop`)
}

// 同步执行任务（立即运行，阻塞直到完成，返回完整结果）
export function runInspectionTaskSync(id: number) {
  return request.post(`/api/v1/inspection/mgmt-tasks/${id}/run-sync`, {}, { timeout: 30 * 60 * 1000 })
}

