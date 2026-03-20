import request from '@/utils/request'

// 获取所有配置
export const getAllConfig = () => {
  return request.get('/api/v1/system/config')
}

// 获取基础配置
export const getBasicConfig = () => {
  return request.get('/api/v1/system/config/basic')
}

// 保存基础配置
export const saveBasicConfig = (data: {
  systemName: string
  systemLogo: string
  systemDescription: string
}) => {
  return request.put('/api/v1/system/config/basic', data)
}

// 获取安全配置
export const getSecurityConfig = () => {
  return request.get('/api/v1/system/config/security')
}

// 保存安全配置
export const saveSecurityConfig = (data: {
  passwordMinLength: number
  sessionTimeout: number
  enableCaptcha: boolean
  maxLoginAttempts: number
  lockoutDuration: number
}) => {
  return request.put('/api/v1/system/config/security', data)
}

// 上传Logo
export const uploadLogo = (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  return request.post('/api/v1/system/config/logo', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}

// 获取公开配置（无需认证）
export const getPublicConfig = () => {
  return request.get('/api/v1/public/config')
}

// ========== 集成管理 ==========

export interface GrafanaIntegration {
  enabled: boolean
  url: string
  subpath: string
}

export interface IntegrationConfig {
  grafana: GrafanaIntegration
}

// 获取集成配置
export const getIntegrationConfig = () => {
  return request.get('/api/v1/system/integrations')
}

// 保存集成配置
export const saveIntegrationConfig = (data: { grafana?: GrafanaIntegration }) => {
  return request.put('/api/v1/system/integrations', data)
}
