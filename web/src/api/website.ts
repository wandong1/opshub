import request from '@/utils/request'

export interface Website {
  id: number
  name: string
  url: string
  icon: string
  type: 'external' | 'internal'
  typeText: string
  credential?: string
  secureCopyUrl: boolean
  accessUser: string
  accessPassword?: string  // 访问密码（仅在详情接口返回）
  description: string
  status: number
  statusText: string
  createTime: string
  updateTime: string
  groupNames: string[] | null
  groupIds: number[] | null
  agentHostIds: number[] | null
  agentHostNames: string[] | null
  agentOnline: boolean
  basePath?: string  // 站点基础路径（仅内部站点）
}

export interface WebsiteRequest {
  id?: number
  name: string
  url: string
  icon?: string
  type: 'external' | 'internal'
  credential?: string
  secureCopyUrl: boolean
  accessUser?: string
  accessPassword?: string
  description?: string
  status: number
  groupIds: number[]
  agentHostIds: number[]
  basePath?: string  // 站点基础路径（仅内部站点）
}

export interface WebsiteListParams {
  page: number
  pageSize: number
  keyword?: string
  type?: string
  groupIds?: number[]
}

export interface WebsiteListResponse {
  list: Website[]
  total: number
  page: number
  pageSize: number
}

// 获取站点列表
export function getWebsiteList(params: WebsiteListParams) {
  return request.get<WebsiteListResponse>('/api/v1/websites', { params })
}

// 获取站点详情
export function getWebsite(id: number) {
  return request.get<Website>(`/api/v1/websites/${id}`)
}

// 创建站点
export function createWebsite(data: WebsiteRequest) {
  return request.post('/api/v1/websites', data)
}

// 更新站点
export function updateWebsite(id: number, data: WebsiteRequest) {
  return request.put(`/api/v1/websites/${id}`, data)
}

// 删除站点
export function deleteWebsite(id: number) {
  return request.delete(`/api/v1/websites/${id}`)
}

// 访问站点（获取访问信息）
export function accessWebsite(id: number) {
  return request.get<{ type: string; url?: string; proxyUrl?: string; hostId?: number }>(`/api/v1/websites/${id}/access`)
}
