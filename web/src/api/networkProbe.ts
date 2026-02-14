import request from '@/utils/request'

// ===== 拨测配置 =====

export const getProbeList = (params: any) => {
  return request.get('/api/v1/inspection/probes', { params })
}

export const getProbe = (id: number) => {
  return request.get(`/api/v1/inspection/probes/${id}`)
}

export const createProbe = (data: any) => {
  return request.post('/api/v1/inspection/probes', data)
}

export const updateProbe = (id: number, data: any) => {
  return request.put(`/api/v1/inspection/probes/${id}`, data)
}

export const deleteProbe = (id: number) => {
  return request.delete(`/api/v1/inspection/probes/${id}`)
}

export const importProbes = (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  return request.post('/api/v1/inspection/probes/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export const exportProbes = (format: string = 'yaml') => {
  return request.get('/api/v1/inspection/probes/export', {
    params: { format },
    responseType: 'blob'
  })
}

export const runProbeOnce = (id: number) => {
  return request.post(`/api/v1/inspection/probes/${id}/run`)
}

// ===== 调度任务 =====

export const getTaskList = (params: any) => {
  return request.get('/api/v1/inspection/tasks', { params })
}

export const createTask = (data: any) => {
  return request.post('/api/v1/inspection/tasks', data)
}

export const updateTask = (id: number, data: any) => {
  return request.put(`/api/v1/inspection/tasks/${id}`, data)
}

export const deleteTask = (id: number) => {
  return request.delete(`/api/v1/inspection/tasks/${id}`)
}

export const toggleTask = (id: number) => {
  return request.put(`/api/v1/inspection/tasks/${id}/toggle`)
}

export const getTaskResults = (id: number, params: any) => {
  return request.get(`/api/v1/inspection/tasks/${id}/results`, { params })
}

// ===== Pushgateway 配置 =====

export const getPushgatewayList = () => {
  return request.get('/api/v1/inspection/pushgateways')
}

export const createPushgateway = (data: any) => {
  return request.post('/api/v1/inspection/pushgateways', data)
}

export const updatePushgateway = (id: number, data: any) => {
  return request.put(`/api/v1/inspection/pushgateways/${id}`, data)
}

export const deletePushgateway = (id: number) => {
  return request.delete(`/api/v1/inspection/pushgateways/${id}`)
}

export const testPushgateway = (id: number) => {
  return request.post(`/api/v1/inspection/pushgateways/${id}/test`)
}

// ===== 分类常量 =====

export const PROBE_CATEGORIES = [
  { value: 'network', label: '基础网络', enabled: true },
  { value: 'layer4', label: '四层协议', enabled: true },
  { value: 'application', label: '应用服务', enabled: false },
  { value: 'workflow', label: '业务流程', enabled: false },
  { value: 'middleware', label: '中间件', enabled: false },
] as const

export const CATEGORY_TYPE_MAP: Record<string, string[]> = {
  network: ['ping'],
  layer4: ['tcp', 'udp'],
  application: ['http', 'https', 'dns', 'websocket', 'ssl_cert'],
  workflow: ['workflow'],
  middleware: ['mysql', 'redis', 'kafka', 'clickhouse', 'mongodb', 'rabbitmq', 'rocketmq', 'postgresql', 'sqlserver', 'milvus'],
}

export const CATEGORY_LABEL_MAP: Record<string, string> = {
  network: '基础网络',
  layer4: '四层协议',
  application: '应用服务',
  workflow: '业务流程',
  middleware: '中间件',
}
