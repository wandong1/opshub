import request from '@/utils/request'

export interface APIKey {
  id: number
  name: string
  maskedKey: string
  description: string
  totalCalls: number
  lastCalledAt: string
  createdAt: string
}

export interface CreateAPIKeyRequest {
  name: string
  description?: string
}

export interface CreateAPIKeyResponse {
  id: number
  name: string
  apiKey: string // 完整明文密钥（仅创建时返回一次）
  description: string
  createdAt: string
}

export interface ListAPIKeysParams {
  page: number
  page_size: number
}

export interface ListAPIKeysResponse {
  total: number
  page: number
  page_size: number
  data: APIKey[]
}

// 创建 API Key
export const createAPIKey = (data: CreateAPIKeyRequest) => {
  return request.post('/api/v1/system/apikeys', data) as Promise<CreateAPIKeyResponse>
}

// 获取 API Key 列表
export const listAPIKeys = (params: ListAPIKeysParams) => {
  return request.get('/api/v1/system/apikeys', { params }) as Promise<ListAPIKeysResponse>
}

// 删除 API Key
export const deleteAPIKey = (id: number) => {
  return request.delete(`/api/v1/system/apikeys/${id}`)
}
