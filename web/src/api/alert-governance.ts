import request from '@/utils/request'

// 去重规则
export interface DedupRule {
  id?: number
  subscriptionId: number
  name: string
  enabled: boolean
  fingerprintKeys: string
  dedupWindow: number
  createdAt?: string
  updatedAt?: string
}

export const getDedupRules = (subscriptionId?: number) => {
  return request.get('/api/v1/alert/dedup-rules', {
    params: { subscriptionId }
  })
}

export const createDedupRule = (data: DedupRule) => {
  return request.post('/api/v1/alert/dedup-rules', data)
}

export const updateDedupRule = (id: number, data: DedupRule) => {
  return request.put(`/api/v1/alert/dedup-rules/${id}`, data)
}

export const deleteDedupRule = (id: number) => {
  return request.delete(`/api/v1/alert/dedup-rules/${id}`)
}

// 分组规则
export interface GroupRule {
  id?: number
  subscriptionId: number
  name: string
  enabled: boolean
  groupBy: string
  groupWait: number
  groupInterval: number
  maxGroupSize: number
  createdAt?: string
  updatedAt?: string
}

export const getGroupRules = (subscriptionId?: number) => {
  return request.get('/api/v1/alert/group-rules', {
    params: { subscriptionId }
  })
}

export const createGroupRule = (data: GroupRule) => {
  return request.post('/api/v1/alert/group-rules', data)
}

export const updateGroupRule = (id: number, data: GroupRule) => {
  return request.put(`/api/v1/alert/group-rules/${id}`, data)
}

export const deleteGroupRule = (id: number) => {
  return request.delete(`/api/v1/alert/group-rules/${id}`)
}

// 抑制规则
export interface InhibitRule {
  id?: number
  subscriptionId: number
  name: string
  enabled: boolean
  sourceMatchers: string
  targetMatchers: string
  equalLabels: string
  createdAt?: string
  updatedAt?: string
}

export const getInhibitRules = (subscriptionId?: number) => {
  return request.get('/api/v1/alert/inhibit-rules', {
    params: { subscriptionId }
  })
}

export const createInhibitRule = (data: InhibitRule) => {
  return request.post('/api/v1/alert/inhibit-rules', data)
}

export const updateInhibitRule = (id: number, data: InhibitRule) => {
  return request.put(`/api/v1/alert/inhibit-rules/${id}`, data)
}

export const deleteInhibitRule = (id: number) => {
  return request.delete(`/api/v1/alert/inhibit-rules/${id}`)
}
