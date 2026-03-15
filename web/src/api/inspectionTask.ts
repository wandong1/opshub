import request from '@/utils/request'

// 获取巡检任务列表（包含拨测和巡检任务）
export function getInspectionTaskList(params: any) {
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
