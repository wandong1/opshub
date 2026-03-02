import request from '@/utils/request'

// 服务标签管理
export const getServiceLabels = (params: { page: number; pageSize: number; keyword?: string }) => {
  return request.get('/api/v1/service-labels', { params })
}

export const getServiceLabel = (id: number) => {
  return request.get(`/api/v1/service-labels/${id}`)
}

export const createServiceLabel = (data: { name: string; matchProcesses: string; description?: string; status: number }) => {
  return request.post('/api/v1/service-labels', data)
}

export const updateServiceLabel = (id: number, data: { name: string; matchProcesses: string; description?: string; status: number }) => {
  return request.put(`/api/v1/service-labels/${id}`, data)
}

export const deleteServiceLabel = (id: number) => {
  return request.delete(`/api/v1/service-labels/${id}`)
}
