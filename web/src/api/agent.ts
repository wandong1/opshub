import request from '@/utils/request'

// Agent管理
export const deployAgent = (hostId: number, serverAddr?: string) => {
  return request.post(`/api/v1/agents/${hostId}/deploy`, { serverAddr })
}

export const batchDeployAgent = (hostIds: number[], serverAddr?: string) => {
  return request.post('/api/v1/agents/batch-deploy', { hostIds, serverAddr })
}

export const getAgentStatuses = () => {
  return request.get('/api/v1/agents/statuses')
}

export const getAgentStatus = (hostId: number) => {
  return request.get(`/api/v1/agents/${hostId}/status`)
}

export const updateAgent = (hostId: number, serverAddr?: string) => {
  return request.put(`/api/v1/agents/${hostId}/update`, { serverAddr })
}

export const uninstallAgent = (hostId: number) => {
  return request.delete(`/api/v1/agents/${hostId}/uninstall`)
}

// Agent文件操作
export const listAgentFiles = (hostId: number, path: string) => {
  return request.get(`/api/v1/agents/${hostId}/files`, { params: { path } })
}

export const uploadAgentFile = (hostId: number, formData: FormData) => {
  return request.post(`/api/v1/agents/${hostId}/files/upload`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export const downloadAgentFile = (hostId: number, path: string) => {
  return request.get(`/api/v1/agents/${hostId}/files/download`, {
    params: { path },
    responseType: 'blob'
  })
}

export const deleteAgentFile = (hostId: number, path: string) => {
  return request.delete(`/api/v1/agents/${hostId}/files`, { params: { path } })
}

// Agent命令执行
export const executeAgentCommand = (hostId: number, command: string, timeout?: number) => {
  return request.post(`/api/v1/agents/${hostId}/execute`, { command, timeout })
}

// 生成Agent安装包（手动部署）
export const generateInstallPackage = (serverAddr: string) => {
  return request.post('/api/v1/agents/generate-install', { serverAddr })
}
