import request from '@/utils/request'

export interface AIModelProxyRequest {
  name: string
  description?: string
  modelType: 'ollama' | 'openai' | 'custom'
  targetUrl: string
  apiKey?: string
  timeout: number
  status: number
  groupId: number
  agentHostIds: number[]
}

export interface AIModelProxyVO {
  id: number
  name: string
  description: string
  modelType: string
  modelTypeText: string
  status: number
  statusText: string
  targetUrl: string
  timeout: number
  proxyToken: string
  proxyUrl: string
  groupId: number
  groupName: string
  agentHostIds: number[]
  agentHostNames: string[]
  agentOnline: boolean
  apiKey: string  // 脱敏显示
  createTime: string
  updateTime: string
}

export interface AIModelProxyListResponse {
  list: AIModelProxyVO[]
  total: number
  page: number
  pageSize: number
}

// 获取列表
export function getAIModelProxies(params: {
  page: number
  pageSize: number
  groupId?: number
  status?: number
  keyword?: string
}) {
  return request.get<AIModelProxyListResponse>('/api/v1/ai-model-proxies', { params })
}

// 获取详情
export function getAIModelProxy(id: number) {
  return request.get<AIModelProxyVO>(`/api/v1/ai-model-proxies/${id}`)
}

// 创建
export function createAIModelProxy(data: AIModelProxyRequest) {
  return request.post<AIModelProxyVO>('/api/v1/ai-model-proxies', data)
}

// 更新
export function updateAIModelProxy(id: number, data: AIModelProxyRequest) {
  return request.put<AIModelProxyVO>(`/api/v1/ai-model-proxies/${id}`, data)
}

// 删除
export function deleteAIModelProxy(id: number) {
  return request.delete(`/api/v1/ai-model-proxies/${id}`)
}

// 重新生成Token
export function regenerateAIModelProxyToken(id: number) {
  return request.post<AIModelProxyVO>(`/api/v1/ai-model-proxies/${id}/regenerate-token`)
}

// 测试连接
export function testAIModelProxyConnection(id: number) {
  return request.get<{ success: boolean; message: string; latency?: number }>(`/api/v1/ai-model-proxies/${id}/test`)
}
