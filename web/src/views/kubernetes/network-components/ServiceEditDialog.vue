<template>
  <a-modal
    v-model="visible"
    :title="isEdit ? '编辑 Service' : '创建 Service'"
    width="900px"
    :close-on-click-modal="false"
    :lock-scroll="false"
    @close="handleClose"
  >
    <!-- 基本信息区域 -->
    <div class="basic-info-section">
      <a-form ref="formRef" :model="formData" :rules="rules" label-width="100px">
        <a-row :gutter="20">
          <a-col :span="8">
            <a-form-item label="名称" field="name">
              <a-input v-model="formData.name" placeholder="Service 名称" :disabled="isEdit" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="命名空间" field="namespace">
              <a-select v-model="formData.namespace" placeholder="选择命名空间" :disabled="isEdit" style="width: 100%">
                <a-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="类型" field="type">
              <a-select v-model="formData.type" placeholder="选择类型" style="width: 100%">
                <a-option label="ClusterIP" value="ClusterIP" />
                <a-option label="NodePort" value="NodePort" />
                <a-option label="LoadBalancer" value="LoadBalancer" />
                <a-option label="ExternalName" value="ExternalName" />
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </div>

    <!-- Tab 导航 -->
    <a-tabs v-model:active-key="activeTab" class="service-tabs">
      <!-- 端口配置 -->
      <a-tab-pane title="端口" key="ports">
        <div class="tab-content">
          <div class="ports-config">
            <div v-for="(port, index) in formData.ports" :key="index" class="port-item">
              <div class="port-header">
                <span>端口 {{ index + 1 }}</span>
                <a-button status="danger" type="text" @click="removePort(index)">
                  <icon-delete />
                </a-button>
              </div>
              <a-row :gutter="12">
                <a-col :span="6">
                  <div class="field-group">
                    <label>名称</label>
                    <a-input v-model="port.name" placeholder="可选" size="small" />
                  </div>
                </a-col>
                <a-col :span="6">
                  <div class="field-group">
                    <label>协议</label>
                    <a-select v-model="port.protocol" size="small" style="width: 100%">
                      <a-option label="TCP" value="TCP" />
                      <a-option label="UDP" value="UDP" />
                      <a-option label="SCTP" value="SCTP" />
                    </a-select>
                  </div>
                </a-col>
                <a-col :span="6">
                  <div class="field-group">
                    <label>端口</label>
                    <a-input-number v-model="port.port" :min="1" :max="65535" size="small" style="width: 100%" />
                  </div>
                </a-col>
                <a-col :span="6">
                  <div class="field-group">
                    <label>目标端口</label>
                    <a-input-number v-model="port.targetPort" :min="1" :max="65535" size="small" style="width: 100%" />
                  </div>
                </a-col>
              </a-row>
              <a-row :gutter="12" v-if="formData.type === 'NodePort'">
                <a-col :span="6">
                  <div class="field-group">
                    <label>NodePort</label>
                    <a-input-number v-model="port.nodePort" :min="30000" :max="32767" size="small" style="width: 100%" />
                  </div>
                </a-col>
              </a-row>
            </div>
            <a-button type="text" @click="addPort" style="margin-top: 8px">
              <icon-plus /> 添加端口
            </a-button>
          </div>
        </div>
      </a-tab-pane>

      <!-- 选择器配置 -->
      <a-tab-pane title="选择器" key="selector">
        <div class="tab-content">
          <div class="selector-config">
            <div class="config-header-with-desc">
              <div class="header-text">
                <div class="title">Pod 标签选择器</div>
                <div class="description">配置用于选择 Pod 的标签</div>
              </div>
              <a-button type="text" @click="addSelector">
                <icon-plus /> 添加
              </a-button>
            </div>
            <div v-if="selectorList.length === 0" class="empty-state">
              <icon-link />
              <p>暂无选择器配置，点击上方"添加"按钮添加</p>
            </div>
            <div v-else>
              <div v-for="(item, index) in selectorList" :key="index" class="selector-item">
                <div class="selector-row">
                  <div class="field-group">
                    <label>键</label>
                    <a-input v-model="item.key" placeholder="例如: app" size="small" />
                  </div>
                  <div class="equal-sign">=</div>
                  <div class="field-group">
                    <label>值</label>
                    <a-input v-model="item.value" placeholder="例如: nginx" size="small" />
                  </div>
                  <div class="action-area">
                    <a-button status="danger" type="text" @click="removeSelector(index)">
                      <icon-delete />
                    </a-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </a-tab-pane>

      <!-- Session Affinity -->
      <a-tab-pane title="Session Affinity" key="sessionAffinity">
        <div class="tab-content">
          <div class="affinity-config-new">
            <!-- 类型选择 -->
            <div class="affinity-type-cards">
              <div
                :class="['affinity-type-card', { 'is-selected': formData.sessionAffinity === 'None' }]"
                @click="formData.sessionAffinity = 'None'"
              >
                <div class="card-icon">
                  <icon-poweroff />
                </div>
                <div class="card-content">
                  <div class="card-title">None</div>
                  <div class="card-desc">不保持会话亲和性，来自同一客户端的请求可能被分发到不同的 Pod</div>
                </div>
                <div class="card-check">
                  <icon-check-circle-fill />
                </div>
              </div>

              <div
                :class="['affinity-type-card', { 'is-selected': formData.sessionAffinity === 'ClientIP' }]"
                @click="formData.sessionAffinity = 'ClientIP'"
              >
                <div class="card-icon">
                  <icon-location />
                </div>
                <div class="card-content">
                  <div class="card-title">ClientIP</div>
                  <div class="card-desc">基于客户端 IP 的会话亲和性，同一客户端的请求将被发送到同一个 Pod</div>
                </div>
                <div class="card-check">
                  <icon-check-circle-fill />
                </div>
              </div>
            </div>

            <!-- 超时配置 -->
            <div v-if="formData.sessionAffinity === 'ClientIP'" class="affinity-timeout-config">
              <div class="timeout-config-card">
                <div class="config-card-header">
                  <div class="header-left">
                    <div class="header-icon">
                      <icon-clock-circle />
                    </div>
                    <div class="header-text">
                      <div class="header-title">会话保持时间</div>
                      <div class="header-desc">同一客户端的请求将被持续发送到同一个 Pod 的时间</div>
                    </div>
                  </div>
                </div>
                <div class="config-card-body">
                  <div class="timeout-input-wrapper">
                    <a-input-number
                      v-model="formData.sessionAffinityConfig.clientIP.timeoutSeconds"
                      :min="1"
                      :max="86400"
                      controls-position="right"
                      size="large"
                      class="timeout-input"
                    />
                    <div class="time-conversions">
                      <div class="conversion-item">
                        <span class="conversion-label">=</span>
                        <span class="conversion-value">{{ formatTimeout(formData.sessionAffinityConfig.clientIP.timeoutSeconds) }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="timeout-info-box">
                    <icon-info-circle />
                    <div class="info-text">
                      <div class="info-main">默认 10800 秒（3 小时），最大 86400 秒（24 小时）</div>
                      <div class="info-sub">建议根据业务需求调整时间，过短可能导致频繁切换 Pod，过长可能影响负载均衡效果</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </a-tab-pane>

      <!-- IP 地址配置 -->
      <a-tab-pane title="IP地址" key="ipAddresses">
        <div class="tab-content">
          <div class="ip-config">
            <!-- Cluster IP -->
            <div class="ip-section">
              <div class="ip-section-header">
                <div class="header-icon">
                  <icon-link />
                </div>
                <div class="header-text">
                  <div class="title">Cluster IP</div>
                  <div class="description">集群内部访问的 IP 地址</div>
                </div>
              </div>
              <div class="ip-input-group">
                <a-input
                  v-model="formData.clusterIP"
                  placeholder="留空自动分配，或输入 'None' 创建无头服务"
                  size="large"
                >
                  <template #prefix>
                    <icon-link />
                  </template>
                </a-input>
              </div>
              <div class="ip-hint">
                <icon-info-circle />
                <span>留空则由系统自动分配 ClusterIP；输入 'None' 可创建无头服务（Headless Service）</span>
              </div>
            </div>

            <!-- External IP -->
            <div class="ip-section" style="margin-top: 30px">
              <div class="ip-section-header">
                <div class="header-icon">
                  <icon-location />
                </div>
                <div class="header-text">
                  <div class="title">外部 IP</div>
                  <div class="description">配置外部可访问的 IP 地址</div>
                </div>
                <a-button type="text" @click="addExternalIP">
                  <icon-plus /> 添加
                </a-button>
              </div>
              <div v-if="formData.externalIPs.length === 0" class="empty-state">
                <icon-location />
                <p>暂无外部 IP 配置，点击上方"添加"按钮添加</p>
              </div>
              <div v-else class="external-ip-list">
                <div v-for="(ip, index) in formData.externalIPs" :key="index" class="external-ip-item">
                  <div class="ip-input-wrapper">
                    <a-input
                      v-model="formData.externalIPs[index]"
                      placeholder="例如: 192.168.1.100"
                      size="small"
                    >
                      <template #prefix>
                        <icon-location />
                      </template>
                    </a-input>
                  </div>
                  <div class="ip-actions">
                    <a-button status="danger" type="text" @click="removeExternalIP(index)">
                      <icon-delete />
                    </a-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </a-tab-pane>

      <!-- 标签/注解 -->
      <a-tab-pane title="标签/注解" key="labelsAnnotations">
        <div class="tab-content">
          <!-- 标签 -->
          <div class="labels-config">
            <div class="config-header-with-desc">
              <div class="header-text">
                <div class="title">标签 (Labels)</div>
                <div class="description">用于标识和选择组织的键值对</div>
              </div>
              <a-button type="text" @click="addLabel">
                <icon-plus /> 添加
              </a-button>
            </div>
            <div v-if="labelsList.length === 0" class="empty-state">
              <icon-tag />
              <p>暂无标签配置，点击上方"添加"按钮添加</p>
            </div>
            <div v-else class="kv-list">
              <div v-for="(item, index) in labelsList" :key="index" class="kv-item">
                <div class="kv-row">
                  <div class="kv-fields">
                    <div class="field-group">
                      <label>键</label>
                      <a-input v-model="item.key" placeholder="键名" size="small" />
                    </div>
                    <div class="equal-sign">=</div>
                    <div class="field-group">
                      <label>值</label>
                      <a-input v-model="item.value" placeholder="键值" size="small" />
                    </div>
                  </div>
                  <div class="kv-actions">
                    <a-button status="danger" type="text" @click="removeLabel(index)">
                      <icon-delete />
                    </a-button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 注解 -->
          <div class="annotations-config" style="margin-top: 40px">
            <div class="config-header-with-desc">
              <div class="header-text">
                <div class="title">注解 (Annotations)</div>
                <div class="description">用于存储任意非标识性数据的键值对</div>
              </div>
              <a-button type="text" @click="addAnnotation">
                <icon-plus /> 添加
              </a-button>
            </div>
            <div v-if="annotationsList.length === 0" class="empty-state">
              <icon-file />
              <p>暂无注解配置，点击上方"添加"按钮添加</p>
            </div>
            <div v-else class="kv-list">
              <div v-for="(item, index) in annotationsList" :key="index" class="kv-item">
                <div class="kv-row">
                  <div class="kv-fields">
                    <div class="field-group">
                      <label>键</label>
                      <a-input v-model="item.key" placeholder="键名" size="small" />
                    </div>
                    <div class="equal-sign">=</div>
                    <div class="field-group">
                      <label>值</label>
                      <a-input v-model="item.value" placeholder="键值" size="small" />
                    </div>
                  </div>
                  <div class="kv-actions">
                    <a-button status="danger" type="text" @click="removeAnnotation(index)">
                      <icon-delete />
                    </a-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </a-tab-pane>
    </a-tabs>

    <template #footer>
      <div class="dialog-footer">
        <a-button @click="handleClose">取消</a-button>
        <a-button type="primary" @click="handleSave" :loading="saving">保存</a-button>
      </div>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getServiceYAML, updateServiceYAML, createService, type ServiceInfo } from '@/api/kubernetes'

interface PortConfig {
  name?: string
  protocol: string
  port: number
  targetPort: number
  nodePort?: number
}

interface KeyValueItem {
  key: string
  value: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['success'])

const visible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const formRef = ref()
const namespaces = ref<any[]>([])
const originalData = ref<any>(null)
const activeTab = ref('ports')

const formData = ref({
  name: '',
  namespace: '',
  type: 'ClusterIP' as any,
  clusterIP: '',
  externalIPs: [] as string[],
  ports: [] as PortConfig[],
  selector: {} as Record<string, string>,
  sessionAffinity: 'None',
  sessionAffinityConfig: {
    clientIP: {
      timeoutSeconds: 10800
    }
  },
  labels: {} as Record<string, string>,
  annotations: {} as Record<string, string>
})

const selectorList = ref<KeyValueItem[]>([])
const labelsList = ref<KeyValueItem[]>([])
const annotationsList = ref<KeyValueItem[]>([])

const rules = {
  name: [{ required: true, content: '请输入 Service 名称', trigger: 'blur' }],
  namespace: [{ required: true, content: '请选择命名空间', trigger: 'change' }],
  type: [{ required: true, content: '请选择类型', trigger: 'change' }]
}

// 打开对话框（编辑模式）
const openEdit = async (service: ServiceInfo, nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = true
  visible.value = true
  activeTab.value = 'ports'

  try {
    const response = await getServiceYAML(props.clusterId!, service.namespace, service.name)
    originalData.value = response.items || response

    const spec = originalData.value.spec || {}
    const metadata = originalData.value.metadata || {}

    formData.value = {
      name: metadata.name || '',
      namespace: metadata.namespace || '',
      type: spec.type || 'ClusterIP',
      clusterIP: spec.clusterIP || '',
      externalIPs: spec.externalIPs || [],
      ports: (spec.ports || []).map((p: any) => ({
        name: p.name || '',
        protocol: p.protocol || 'TCP',
        port: p.port,
        targetPort: p.targetPort,
        nodePort: p.nodePort
      })),
      selector: spec.selector || {},
      sessionAffinity: spec.sessionAffinity || 'None',
      sessionAffinityConfig: spec.sessionAffinityConfig || {
        clientIP: {
          timeoutSeconds: 10800
        }
      },
      labels: metadata.labels || {},
      annotations: metadata.annotations || {}
    }

    // 同步到列表
    syncSelectorFromForm()
    syncLabelsFromForm()
    syncAnnotationsFromForm()

    if (formData.value.ports.length === 0) {
      addPort()
    }
  } catch (error) {
    Message.error('获取 Service 详情失败')
  }
}

// 打开对话框（创建模式）
const openCreate = (nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = false
  visible.value = true
  activeTab.value = 'ports'

  formData.value = {
    name: '',
    namespace: '',
    type: 'ClusterIP',
    clusterIP: '',
    externalIPs: [],
    ports: [{
      name: '',
      protocol: 'TCP',
      port: 80,
      targetPort: 80
    }],
    selector: {},
    sessionAffinity: 'None',
    sessionAffinityConfig: {
      clientIP: {
        timeoutSeconds: 10800
      }
    },
    labels: {},
    annotations: {}
  }

  selectorList.value = []
  labelsList.value = []
  annotationsList.value = []
}

// 同步方法
const syncSelectorFromForm = () => {
  selectorList.value = Object.entries(formData.value.selector).map(([key, value]) => ({ key, value }))
}

const syncSelectorToList = () => {
  formData.value.selector = selectorList.value.reduce((acc, { key, value }) => {
    if (key && value) {
      acc[key] = value
    }
    return acc
  }, {} as Record<string, string>)
}

const syncLabelsFromForm = () => {
  labelsList.value = Object.entries(formData.value.labels).map(([key, value]) => ({ key, value }))
}

const syncLabelsToList = () => {
  formData.value.labels = labelsList.value.reduce((acc, { key, value }) => {
    if (key && value) {
      acc[key] = value
    }
    return acc
  }, {} as Record<string, string>)
}

const syncAnnotationsFromForm = () => {
  annotationsList.value = Object.entries(formData.value.annotations).map(([key, value]) => ({ key, value }))
}

const syncAnnotationsToList = () => {
  formData.value.annotations = annotationsList.value.reduce((acc, { key, value }) => {
    if (key) {
      acc[key] = value || ''
    }
    return acc
  }, {} as Record<string, string>)
}

// 端口操作
const addPort = () => {
  formData.value.ports.push({
    name: '',
    protocol: 'TCP',
    port: 80,
    targetPort: 80
  })
}

const removePort = (index: number) => {
  formData.value.ports.splice(index, 1)
}

// 选择器操作
const addSelector = () => {
  selectorList.value.push({ key: '', value: '' })
}

const removeSelector = (index: number) => {
  selectorList.value.splice(index, 1)
}

// 外部 IP 操作
const addExternalIP = () => {
  formData.value.externalIPs.push('')
}

const removeExternalIP = (index: number) => {
  formData.value.externalIPs.splice(index, 1)
}

// 标签操作
const addLabel = () => {
  labelsList.value.push({ key: '', value: '' })
}

const removeLabel = (index: number) => {
  labelsList.value.splice(index, 1)
}

// 注解操作
const addAnnotation = () => {
  annotationsList.value.push({ key: '', value: '' })
}

const removeAnnotation = (index: number) => {
  annotationsList.value.splice(index, 1)
}

// 格式化超时时间显示
const formatTimeout = (seconds: number) => {
  if (seconds < 60) {
    return `${seconds} 秒`
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    return `${minutes} 分钟`
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600)
    return `${hours} 小时`
  } else {
    const days = Math.floor(seconds / 86400)
    return `${days} 天`
  }
}

// 构建保存的数据
const buildSaveData = () => {
  // 同步所有列表到对象
  syncSelectorToList()
  syncLabelsToList()
  syncAnnotationsToList()

  // 构建端口数组
  const ports = formData.value.ports.map(p => {
    const portObj: any = {
      protocol: p.protocol,
      port: p.port,
      targetPort: p.targetPort
    }
    if (p.name) portObj.name = p.name
    if (formData.value.type === 'NodePort' && p.nodePort) {
      portObj.nodePort = p.nodePort
    }
    return portObj
  })

  // 构建 Service 对象
  const serviceData: any = {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: formData.value.name,
      namespace: formData.value.namespace
    },
    spec: {
      type: formData.value.type,
      selector: formData.value.selector,
      ports: ports,
      sessionAffinity: formData.value.sessionAffinity
    }
  }

  // 添加 SessionAffinityConfig
  if (formData.value.sessionAffinity === 'ClientIP') {
    serviceData.spec.sessionAffinityConfig = formData.value.sessionAffinityConfig
  }

  // 如果是编辑模式，需要包含完整的 spec
  if (isEdit.value && originalData.value) {
    const originalSpec = originalData.value.spec || {}

    // 保留 metadata
    serviceData.metadata = {
      ...originalData.value.metadata,
      name: formData.value.name,
      namespace: formData.value.namespace
    }

    // 添加 labels 和 annotations
    if (Object.keys(formData.value.labels).length > 0) {
      serviceData.metadata.labels = formData.value.labels
    }
    if (Object.keys(formData.value.annotations).length > 0) {
      serviceData.metadata.annotations = formData.value.annotations
    }

    // 保留 ClusterIP
    if (originalSpec.clusterIP) {
      serviceData.spec.clusterIP = originalSpec.clusterIP
    }
    if (formData.value.clusterIP) {
      serviceData.spec.clusterIP = formData.value.clusterIP
    }

    // 保留其他必要的 spec 字段
    if (originalSpec.clusterIPs) {
      serviceData.spec.clusterIPs = originalSpec.clusterIPs
    }
    if (originalSpec.ipFamilies) {
      serviceData.spec.ipFamilies = originalSpec.ipFamilies
    }
    if (originalSpec.ipFamilyPolicy) {
      serviceData.spec.ipFamilyPolicy = originalSpec.ipFamilyPolicy
    }
    if (originalSpec.internalTrafficPolicy) {
      serviceData.spec.internalTrafficPolicy = originalSpec.internalTrafficPolicy
    }
    if (originalSpec.externalTrafficPolicy) {
      serviceData.spec.externalTrafficPolicy = originalSpec.externalTrafficPolicy
    }
    if (originalSpec.healthCheckNodePort) {
      serviceData.spec.healthCheckNodePort = originalSpec.healthCheckNodePort
    }
  } else {
    // 创建模式
    if (formData.value.clusterIP) {
      serviceData.spec.clusterIP = formData.value.clusterIP
    }
    if (Object.keys(formData.value.labels).length > 0) {
      serviceData.metadata.labels = formData.value.labels
    }
    if (Object.keys(formData.value.annotations).length > 0) {
      serviceData.metadata.annotations = formData.value.annotations
    }
  }

  // 添加外部 IP
  const validExternalIPs = formData.value.externalIPs.filter(ip => ip)
  if (validExternalIPs.length > 0) {
    serviceData.spec.externalIPs = validExternalIPs
  }

  return serviceData
}

// 保存
const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  // 验证端口配置
  if (formData.value.ports.length === 0) {
    Message.error('请至少配置一个端口')
    return
  }

  // 验证选择器 - 先同步数据
  syncSelectorToList()
  const hasValidSelector = Object.values(formData.value.selector).some(v => v)
  if (!hasValidSelector && formData.value.type !== 'ExternalName') {
    Message.error('请配置至少一个选择器')
    return
  }

  saving.value = true
  try {
    const serviceData = buildSaveData()

    if (isEdit.value) {
      await updateServiceYAML(
        props.clusterId!,
        formData.value.namespace,
        formData.value.name,
        serviceData
      )
      Message.success('更新成功')
    } else {
      await createService(props.clusterId!, formData.value.namespace, {
        name: formData.value.name,
        type: formData.value.type,
        clusterIP: formData.value.clusterIP || undefined,
        ports: formData.value.ports.map(p => ({
          name: p.name,
          protocol: p.protocol,
          port: p.port,
          targetPort: p.targetPort.toString(),
          nodePort: p.nodePort
        })) as any,
        selector: formData.value.selector,
        sessionAffinity: formData.value.sessionAffinity
      })
      Message.success('创建成功')
    }

    emit('success')
    handleClose()
  } catch (error) {
    Message.error('保存失败')
  } finally {
    saving.value = false
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
  formRef.value?.resetFields()
  originalData.value = null
  activeTab.value = 'ports'
  selectorList.value = []
  labelsList.value = []
  annotationsList.value = []
}

defineExpose({
  openEdit,
  openCreate
})
</script>

<style scoped>
/* 基本信息区域 */
.basic-info-section {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #dcdfe6;
}

/* Tab 样式 */
.service-tabs {
  margin-top: 10px;
}

.service-tabs :deep(.arco-tabs__header) {
  margin-bottom: 20px;
}

.service-tabs :deep(.arco-tabs__item) {
  color: #606266;
  font-weight: 500;
}

.service-tabs :deep(.arco-tabs__item.is-active) {
  color: #165dff;
}

.service-tabs :deep(.arco-tabs__active-bar) {
  background-color: #165dff;
}

.tab-content {
  min-height: 300px;
}

/* 配置区域通用样式 */
.port-item,
.selector-config,
.affinity-config,
.ip-config,
.labels-config,
.annotations-config {
  width: 100%;
}

.port-item {
  margin-bottom: 20px;
  padding: 20px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background-color: #fafafa;
}

.port-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  font-weight: 500;
  color: #165dff;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-group label {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}

.hint {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  font-weight: 500;
  color: #303133;
}

.config-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.config-row span {
  color: #909399;
  font-weight: 500;
}

/* 输入框样式 */
:deep(.arco-input__wrapper) {
  background-color: #fff;
  border-color: #dcdfe6;
  box-shadow: none;
}

:deep(.arco-input__wrapper:hover) {
  border-color: #165dff;
}

:deep(.arco-input__wrapper.is-focus) {
  border-color: #165dff;
  box-shadow: 0 0 0 1px #165dff;
}

:deep(.arco-input__inner) {
  color: #606266;
}

:deep(.arco-select .arco-input__wrapper) {
  background-color: #fff;
}

:deep(.arco-select .arco-input__inner) {
  color: #606266;
}

:deep(.arco-input-number .arco-input__wrapper) {
  background-color: #fff;
}

:deep(.arco-input-number .arco-input__inner) {
  color: #606266;
}

:deep(.arco-input-number__decrease),
:deep(.arco-input-number__increase) {
  background-color: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

:deep(.arco-input-number__decrease:hover),
:deep(.arco-input-number__increase:hover) {
  color: #165dff;
}

/* 单选框样式 */
:deep(.arco-radio__label) {
  color: #606266;
}

:deep(.arco-radio-button.arco-radio-button-checked) {
  border-color: #165dff;
  background: #165dff;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 按钮样式 */
:deep(.arco-button--primary) {
  background-color: #165dff;
  border-color: #165dff;
  color: #ffffff;
  font-weight: 500;
}

:deep(.arco-button--primary:hover) {
  background-color: #4080ff;
  border-color: #4080ff;
  color: #ffffff;
}

:deep(.arco-button--default) {
  background-color: #fff;
  border-color: #dcdfe6;
  color: #606266;
}

:deep(.arco-button--default:hover) {
  border-color: #165dff;
  color: #165dff;
  background-color: #fff;
}

/* Link 按钮样式 */
:deep(.arco-button.is-link) {
  color: #165dff;
  font-weight: 500;
}

:deep(.arco-button.is-link:hover) {
  color: #4080ff;
}

/* Primary Link 按钮样式（添加按钮）- 金色文字 */
:deep(.arco-button--primary.is-link) {
  color: #165dff;
  font-weight: 500;
  background-color: transparent;
}

:deep(.arco-button--primary.is-link:hover) {
  color: #4080ff;
  background-color: transparent;
}

:deep(.arco-button.is-link.is-danger) {
  color: #f56c6c;
}

:deep(.arco-button.is-link.is-danger:hover) {
  color: #f78989;
}

/* Dialog 对话框背景 */
:deep(.arco-dialog) {
  background-color: #fff;
}

:deep(.arco-dialog__header) {
  border-bottom: 1px solid #165dff;
  padding: 20px;
}

:deep(.arco-dialog__title) {
  color: #165dff;
  font-weight: 500;
}

:deep(.arco-dialog__headerbtn .arco-modal__close) {
  color: #165dff;
}

:deep(.arco-dialog__headerbtn .arco-modal__close:hover) {
  color: #4080ff;
}

:deep(.arco-dialog__body) {
  padding: 20px;
  color: #606266;
}

:deep(.arco-dialog__footer) {
  border-top: 1px solid #dcdfe6;
  padding: 15px 20px;
}

/* Form label */
:deep(.arco-form-item__label) {
  color: #606266;
  font-weight: 500;
}

/* 配置头部带描述 */
.config-header-with-desc {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.header-text {
  flex: 1;
}

.header-text .title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.header-text .description {
  font-size: 13px;
  color: #909399;
  font-weight: 400;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background-color: #fafafa;
  border-radius: 8px;
  border: 1px dashed #dcdfe6;
}

.empty-state .empty-icon {
  font-size: 48px;
  color: #c0c4cc;
  margin-bottom: 12px;
}

.empty-state p {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

/* 选择器配置 */
.selector-item {
  margin-bottom: 12px;
  padding: 16px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  transition: all 0.3s;
}

.selector-item:hover {
  border-color: #165dff;
  box-shadow: 0 2px 8px rgba(22, 93, 255, 0.1);
}

.selector-row {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.equal-sign {
  color: #909399;
  font-weight: 600;
  font-size: 16px;
  padding-top: 26px;
  min-width: 20px;
  text-align: center;
}

.action-area {
  padding-top: 26px;
}

/* Session Affinity 配置 - 新设计 */
.affinity-config-new {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 30px;
}

.affinity-type-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.affinity-type-card {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 24px;
  background-color: #fff;
  border: 2px solid #e4e7ed;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.affinity-type-card:hover {
  border-color: #165dff;
  box-shadow: 0 4px 20px rgba(22, 93, 255, 0.15);
  transform: translateY(-2px);
}

.affinity-type-card.is-selected {
  border-color: #165dff;
  background-color: #fff;
  box-shadow: 0 4px 20px rgba(22, 93, 255, 0.2);
}

.affinity-type-card .card-icon {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #e8f3ff;
  border-radius: 12px;
  flex-shrink: 0;
  color: #165dff;
  font-size: 28px;
  border: 2px solid #165dff;
}

.affinity-type-card.is-selected .card-icon {
  background: #165dff;
  color: #fff;
}

.affinity-type-card .card-content {
  flex: 1;
}

.affinity-type-card .card-title {
  font-size: 18px;
  font-weight: 700;
  color: #303133;
  margin-bottom: 8px;
  letter-spacing: 0.5px;
}

.affinity-type-card .card-desc {
  font-size: 13px;
  color: #909399;
  line-height: 1.6;
  font-weight: 400;
}

.affinity-type-card .card-check {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #165dff;
  border-radius: 50%;
  color: #fff;
  font-size: 20px;
  flex-shrink: 0;
}

/* 超时配置卡片 */
.affinity-timeout-config {
  margin-top: 10px;
}

.timeout-config-card {
  padding: 24px;
  background-color: #fff;
  border: 2px solid #165dff;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(22, 93, 255, 0.15);
}

.timeout-config-card .config-card-header {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 2px dashed #165dff;
}

.timeout-config-card .header-left {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.timeout-config-card .header-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #165dff;
  border-radius: 10px;
  color: #fff;
  font-size: 24px;
  flex-shrink: 0;
}

.timeout-config-card .header-text {
  flex: 1;
}

.timeout-config-card .header-title {
  font-size: 17px;
  font-weight: 700;
  color: #303133;
  margin-bottom: 6px;
  letter-spacing: 0.5px;
}

.timeout-config-card .header-desc {
  font-size: 13px;
  color: #909399;
  line-height: 1.6;
  font-weight: 400;
}

.timeout-config-card .config-card-body {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.timeout-config-card .timeout-input-wrapper {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 20px;
  background-color: #fff;
  border-radius: 10px;
  border: 1px solid #165dff;
}

.timeout-config-card .timeout-input {
  flex-shrink: 0;
}

.timeout-config-card .timeout-input :deep(.arco-input__wrapper) {
  width: 200px;
  padding: 8px 16px;
  background-color: #fff;
  border: 2px solid #165dff;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.timeout-config-card .timeout-input :deep(.arco-input__inner) {
  font-size: 18px;
  font-weight: 700;
  color: #165dff;
  text-align: center;
}

.timeout-config-card .time-conversions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.timeout-config-card .conversion-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background-color: #e8f3ff;
  border-radius: 6px;
  border: 1px solid #165dff;
}

.timeout-config-card .conversion-label {
  font-size: 14px;
  font-weight: 600;
  color: #165dff;
  min-width: 24px;
}

.timeout-config-card .conversion-value {
  font-size: 15px;
  font-weight: 700;
  color: #303133;
}

.timeout-config-card .timeout-info-box {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  background-color: #e8f3ff;
  border-radius: 10px;
  border: 1px dashed #165dff;
}

.timeout-config-card .info-icon {
  font-size: 24px;
  color: #165dff;
  flex-shrink: 0;
  margin-top: 2px;
}

.timeout-config-card .info-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.timeout-config-card .info-main {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.timeout-config-card .info-sub {
  font-size: 13px;
  font-weight: 400;
  color: #909399;
  line-height: 1.5;
}

/* IP 地址配置 */
.ip-config {
  width: 100%;
}

.ip-section {
  padding: 20px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
}

.ip-section-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.ip-section-header .header-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #e8f3ff;
  border-radius: 8px;
  color: #165dff;
  font-size: 20px;
  flex-shrink: 0;
}

.ip-section-header .header-text {
  flex: 1;
}

.ip-section-header .header-text .title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.ip-section-header .header-text .description {
  font-size: 13px;
  color: #909399;
  font-weight: 400;
}

.ip-input-group {
  margin-bottom: 12px;
}

.ip-input-group :deep(.arco-input__wrapper) {
  padding-left: 12px;
}

.ip-input-group :deep(.arco-input__prefix) {
  color: #909399;
}

.ip-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #909399;
  padding: 8px 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.ip-hint .arco-icon {
  font-size: 14px;
  color: #165dff;
}

.ip-hint span {
  flex: 1;
}

/* 外部 IP 列表 */
.external-ip-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.external-ip-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ip-input-wrapper {
  flex: 1;
}

.ip-actions {
  display: flex;
  gap: 8px;
}

/* 键值对列表 */
.kv-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.kv-item {
  padding: 16px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  transition: all 0.3s;
}

.kv-item:hover {
  border-color: #165dff;
  box-shadow: 0 2px 8px rgba(22, 93, 255, 0.1);
}

.kv-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.kv-fields {
  display: flex;
  align-items: center;
  flex: 1;
  gap: 16px;
}

.kv-fields .field-group {
  flex: 1;
}

.kv-actions {
  display: flex;
  align-items: center;
  padding-top: 26px;
}

/* 端口配置优化 */
.ports-config {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 标签和注解之间的分隔 */
.labels-config,
.annotations-config {
  width: 100%;
}
</style>
