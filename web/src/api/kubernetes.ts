import request from '@/utils/request'

export interface Cluster {
  id: number
  name: string
  alias: string
  apiEndpoint: string
  version: string
  status: number
  nodeCount: number    // 节点数量
  region: string
  provider: string
  description: string
  createdAt: string
  updatedAt: string
}

export interface CreateClusterParams {
  name: string
  alias?: string
  apiEndpoint: string
  kubeConfig: string
  region?: string
  provider?: string
  description?: string
}

export interface UpdateClusterParams {
  name?: string
  alias?: string
  apiEndpoint?: string
  kubeConfig?: string
  region?: string
  provider?: string
  description?: string
}

/**
 * 获取集群列表
 */
export function getClusterList() {
  return request<Cluster[]>({
    url: '/api/v1/plugins/kubernetes/clusters',
    method: 'get'
  })
}

/**
 * 获取集群详情
 */
export function getClusterDetail(id: number) {
  return request<Cluster>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}`,
    method: 'get'
  })
}

/**
 * 创建集群
 */
export function createCluster(data: CreateClusterParams) {
  return request<Cluster>({
    url: '/api/v1/plugins/kubernetes/clusters',
    method: 'post',
    data
  })
}

/**
 * 更新集群
 */
export function updateCluster(id: number, data: UpdateClusterParams) {
  return request<Cluster>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除集群
 */
export function deleteCluster(id: number) {
  return request({
    url: `/api/v1/plugins/kubernetes/clusters/${id}`,
    method: 'delete'
  })
}

/**
 * 测试集群连接
 */
export function testClusterConnection(id: number) {
  return request<{
    status: string
    version: string
  }>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}/test`,
    method: 'post'
  })
}

/**
 * 获取集群凭证（解密后的 KubeConfig）
 */
export function getClusterConfig(id: number) {
  return request<string>({
    url: `/api/v1/plugins/kubernetes/clusters/${id}/config`,
    method: 'get'
  })
}

// ==================== Kubernetes 资源类型定义 ====================

export interface NodeInfo {
  name: string
  status: string
  roles: string
  age: string
  version: string
  internalIP: string
  osImage: string
  kernelVersion: string
  containerRuntime: string
  labels: Record<string, string>
}

export interface NamespaceInfo {
  name: string
  status: string
  age: string
  labels: Record<string, string>
}

export interface PodInfo {
  name: string
  namespace: string
  ready: string
  status: string
  restarts: number
  age: string
  ip: string
  node: string
  labels: Record<string, string>
}

export interface DeploymentInfo {
  name: string
  namespace: string
  ready: string
  upToDate: number
  available: number
  age: string
  replicas: number
  selector: Record<string, string>
  labels: Record<string, string>
}

export interface ClusterStats {
  nodeCount: number
  workloadCount: number
  podCount: number
  cpuUsage: number
  memoryUsage: number
  cpuCapacity: number
  memoryCapacity: number
  cpuAllocatable: number
  memoryAllocatable: number
  cpuUsed: number
  memoryUsed: number
}

export interface ClusterNetworkInfo {
  serviceCIDR: string
  podCIDR: string
  apiServerAddress: string
  networkPlugin: string
  proxyMode: string
  dnsService: string
}

export interface ComponentInfo {
  name: string
  version: string
  status: string
}

export interface RuntimeInfo {
  containerRuntime: string
  version: string
}

export interface StorageInfo {
  name: string
  provisioner: string
  reclaimPolicy: string
}

export interface ClusterComponentInfo {
  components: ComponentInfo[]
  runtime: RuntimeInfo
  storage: StorageInfo[]
}

// ==================== Kubernetes 资源 API ====================

/**
 * 获取节点列表
 */
export function getNodes(clusterId: number) {
  return request<NodeInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/nodes',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取命名空间列表
 */
export function getNamespaces(clusterId: number) {
  return request<NamespaceInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/namespaces',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取 Pod 列表
 */
export function getPods(clusterId: number, namespace?: string) {
  return request<PodInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/pods',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取 Deployment 列表
 */
export function getDeployments(clusterId: number, namespace?: string) {
  return request<DeploymentInfo[]>({
    url: '/api/v1/plugins/kubernetes/resources/deployments',
    method: 'get',
    params: { clusterId, namespace }
  })
}

/**
 * 获取集群统计信息
 */
export function getClusterStats(clusterId: number) {
  return request<ClusterStats>({
    url: '/api/v1/plugins/kubernetes/resources/stats',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取集群网络信息
 */
export function getClusterNetworkInfo(clusterId: number) {
  return request<ClusterNetworkInfo>({
    url: '/api/v1/plugins/kubernetes/resources/network',
    method: 'get',
    params: { clusterId }
  })
}

/**
 * 获取集群组件信息
 */
export function getClusterComponentInfo(clusterId: number) {
  return request<ClusterComponentInfo>({
    url: '/api/v1/plugins/kubernetes/resources/components',
    method: 'get',
    params: { clusterId }
  })
}
