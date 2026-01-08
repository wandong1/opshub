<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑 Ingress' : '创建 Ingress'"
    width="1000px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="formData" :rules="rules" label-width="120px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入 Ingress 名称" :disabled="isEdit" />
      </el-form-item>

      <el-form-item label="命名空间" prop="namespace">
        <el-select v-model="formData.namespace" placeholder="请选择命名空间" :disabled="isEdit" style="width: 100%">
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </el-form-item>

      <el-form-item label="Ingress Class">
        <el-input v-model="formData.ingressClassName" placeholder="例如: nginx" />
      </el-form-item>

      <!-- 主机配置 -->
      <el-form-item label="主机配置">
        <div class="hosts-config">
          <div v-for="(host, hostIndex) in formData.hosts" :key="hostIndex" class="host-item">
            <div class="host-header">
              <span>主机 {{ hostIndex + 1 }}</span>
              <el-button type="danger" link @click="removeHost(hostIndex)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <div class="host-field">
              <label>主机名</label>
              <el-input v-model="host.host" placeholder="例如: example.com" />
            </div>

            <!-- 路径配置 -->
            <div class="paths-config">
              <div class="paths-title">路径配置</div>
              <div v-for="(path, pathIndex) in host.paths" :key="pathIndex" class="path-item">
                <div class="path-header">
                  <span>路径 {{ pathIndex + 1 }}</span>
                  <el-button type="danger" link @click="removePath(hostIndex, pathIndex)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </div>
                <div class="path-grid">
                  <div class="path-field">
                    <label>路径</label>
                    <el-input v-model="path.path" placeholder="例如: /api" size="small" />
                  </div>
                  <div class="path-field">
                    <label>类型</label>
                    <el-select v-model="path.pathType" size="small" style="width: 100%">
                      <el-option label="Prefix" value="Prefix" />
                      <el-option label="Exact" value="Exact" />
                      <el-option label="ImplementationSpecific" value="ImplementationSpecific" />
                    </el-select>
                  </div>
                  <div class="path-field">
                    <label>服务名称</label>
                    <el-input v-model="path.service" placeholder="服务名称" size="small" />
                  </div>
                  <div class="path-field">
                    <label>端口</label>
                    <el-input-number v-model="path.port" :min="1" :max="65535" size="small" style="width: 100%" />
                  </div>
                </div>
              </div>
              <el-button type="primary" link @click="addPath(hostIndex)" style="margin-top: 8px">
                <el-icon><Plus /></el-icon> 添加路径
              </el-button>
            </div>
          </div>
          <el-button type="primary" link @click="addHost">
            <el-icon><Plus /></el-icon> 添加主机
          </el-button>
        </div>
      </el-form-item>

      <!-- TLS 配置 -->
      <el-form-item label="TLS 配置">
        <div class="tls-config">
          <div v-for="(tls, tlsIndex) in formData.tls" :key="tlsIndex" class="tls-item">
            <div class="tls-header">
              <span>TLS {{ tlsIndex + 1 }}</span>
              <el-button type="danger" link @click="removeTLS(tlsIndex)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <div class="tls-hosts">
              <el-tag
                v-for="(host, hostIdx) in tls.hosts"
                :key="hostIdx"
                closable
                @close="removeTLSHost(tlsIndex, hostIdx)"
              >
                {{ host }}
              </el-tag>
              <el-input
                v-model="newTLSHost"
                placeholder="输入主机名后回车添加"
                @keyup.enter="addTLSHost(tlsIndex)"
                size="small"
                style="width: 200px"
              />
            </div>
            <div class="tls-secret">
              <label>Secret 名称</label>
              <el-input v-model="tls.secretName" placeholder="Secret 名称" size="small" />
            </div>
          </div>
          <el-button type="primary" link @click="addTLS">
            <el-icon><Plus /></el-icon> 添加 TLS
          </el-button>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import { getIngressYAML, updateIngressYAML, createIngress, type IngressInfo } from '@/api/kubernetes'

interface PathConfig {
  path: string
  pathType: string
  service: string
  port: number
}

interface HostConfig {
  host: string
  paths: PathConfig[]
}

interface TLSConfig {
  hosts: string[]
  secretName: string
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
const newTLSHost = ref('')

const formData = ref({
  name: '',
  namespace: '',
  ingressClassName: '',
  hosts: [] as HostConfig[],
  tls: [] as TLSConfig[]
})

const rules = {
  name: [{ required: true, message: '请输入 Ingress 名称', trigger: 'blur' }],
  namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }]
}

// 打开对话框（编辑模式）
const openEdit = async (ingress: IngressInfo, nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = true
  visible.value = true

  try {
    const response = await getIngressYAML(props.clusterId!, ingress.namespace, ingress.name)
    originalData.value = response.items || response

    // 解析数据到表单
    const spec = originalData.value.spec || {}
    const metadata = originalData.value.metadata || {}

    // 解析主机和路径
    const hosts: HostConfig[] = []
    if (spec.rules && spec.rules.length > 0) {
      spec.rules.forEach((rule: any) => {
        const hostConfig: HostConfig = {
          host: rule.host || '',
          paths: []
        }

        if (rule.http && rule.http.paths) {
          rule.http.paths.forEach((p: any) => {
            hostConfig.paths.push({
              path: p.path || '/',
              pathType: p.pathType || 'Prefix',
              service: p.backend?.service?.name || '',
              port: p.backend?.service?.port?.number || 0
            })
          })
        }

        hosts.push(hostConfig)
      })
    }

    // 解析 TLS
    const tls: TLSConfig[] = []
    if (spec.tls) {
      spec.tls.forEach((t: any) => {
        tls.push({
          hosts: t.hosts || [],
          secretName: t.secretName || ''
        })
      })
    }

    formData.value = {
      name: metadata.name || '',
      namespace: metadata.namespace || '',
      ingressClassName: spec.ingressClassName || '',
      hosts,
      tls
    }

    // 如果没有主机，添加一个默认主机
    if (formData.value.hosts.length === 0) {
      addHost()
    }
  } catch (error) {
    console.error('获取 Ingress 详情失败:', error)
    ElMessage.error('获取 Ingress 详情失败')
  }
}

// 打开对话框（创建模式）
const openCreate = (nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = false
  visible.value = true

  // 重置表单
  formData.value = {
    name: '',
    namespace: '',
    ingressClassName: '',
    hosts: [],
    tls: []
  }

  // 添加默认主机
  addHost()
}

// 添加主机
const addHost = () => {
  formData.value.hosts.push({
    host: '',
    paths: [{
      path: '/',
      pathType: 'Prefix',
      service: '',
      port: 80
    }]
  })
}

// 删除主机
const removeHost = (index: number) => {
  formData.value.hosts.splice(index, 1)
}

// 添加路径
const addPath = (hostIndex: number) => {
  const host = formData.value.hosts[hostIndex]
  if (host) {
    host.paths.push({
      path: '/',
      pathType: 'Prefix',
      service: '',
      port: 80
    })
  }
}

// 删除路径
const removePath = (hostIndex: number, pathIndex: number) => {
  const host = formData.value.hosts[hostIndex]
  if (host) {
    host.paths.splice(pathIndex, 1)
  }
}

// 添加 TLS
const addTLS = () => {
  formData.value.tls.push({
    hosts: [],
    secretName: ''
  })
}

// 删除 TLS
const removeTLS = (index: number) => {
  formData.value.tls.splice(index, 1)
}

// 添加 TLS 主机
const addTLSHost = (tlsIndex: number) => {
  if (newTLSHost.value) {
    const tls = formData.value.tls[tlsIndex]
    if (tls) {
      tls.hosts.push(newTLSHost.value)
      newTLSHost.value = ''
    }
  }
}

// 删除 TLS 主机
const removeTLSHost = (tlsIndex: number, hostIndex: number) => {
  const tls = formData.value.tls[tlsIndex]
  if (tls) {
    tls.hosts.splice(hostIndex, 1)
  }
}

// 构建保存的数据
const buildSaveData = () => {
  // 构建 rules
  const rules = formData.value.hosts
    .filter(host => host.paths.length > 0)
    .map(host => {
      const rule: any = {}

      if (host.host) {
        rule.host = host.host
      }

      rule.http = {
        paths: host.paths.map(path => ({
          path: path.path,
          pathType: path.pathType,
          backend: {
            service: {
              name: path.service,
              port: {
                number: path.port
              }
            }
          }
        }))
      }

      return rule
    })

  // 构建 Ingress 对象
  const ingressData: any = {
    apiVersion: 'networking.k8s.io/v1',
    kind: 'Ingress',
    metadata: {
      name: formData.value.name,
      namespace: formData.value.namespace
    },
    spec: {
      rules,
      ingressClassName: formData.value.ingressClassName || undefined
    }
  }

  // 添加 TLS
  const validTLS = formData.value.tls.filter(t => t.hosts.length > 0 || t.secretName)
  if (validTLS.length > 0) {
    ingressData.spec.tls = validTLS.map(t => ({
      hosts: t.hosts,
      secretName: t.secretName
    }))
  }

  // 如果是编辑模式，保留原有的 metadata 字段
  if (isEdit.value && originalData.value) {
    ingressData.metadata = {
      ...originalData.value.metadata,
      name: formData.value.name,
      namespace: formData.value.namespace
    }
    // 保留 annotations
    if (originalData.value.metadata?.annotations) {
      ingressData.metadata.annotations = originalData.value.metadata.annotations
    }
  }

  return ingressData
}

// 保存
const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  // 验证配置
  if (formData.value.hosts.length === 0) {
    ElMessage.error('请至少配置一个主机')
    return
  }

  const hasValidPath = formData.value.hosts.some(host =>
    host.paths.some(path => path.service && path.port > 0)
  )

  if (!hasValidPath) {
    ElMessage.error('请至少配置一个有效的路径（包含服务名称和端口）')
    return
  }

  saving.value = true
  try {
    const ingressData = buildSaveData()

    if (isEdit.value) {
      await updateIngressYAML(
        props.clusterId!,
        formData.value.namespace,
        formData.value.name,
        ingressData
      )
      ElMessage.success('更新成功')
    } else {
      // 构建创建请求数据
      const createData = {
        name: formData.value.name,
        ingressClassName: formData.value.ingressClassName || undefined,
        rules: formData.value.hosts
          .filter(host => host.paths.length > 0)
          .map(host => ({
            host: host.host || undefined,
            paths: host.paths
              .filter(p => p.service)
              .map(p => ({
                path: p.path,
                pathType: p.pathType,
                service: p.service,
                port: p.port
              }))
          })),
        tls: formData.value.tls
          .filter(t => t.hosts.length > 0)
          .map(t => ({
            hosts: t.hosts,
            secretName: t.secretName
          }))
      }

      await createIngress(props.clusterId!, formData.value.namespace, createData)
      ElMessage.success('创建成功')
    }

    emit('success')
    handleClose()
  } catch (error) {
    console.error('保存失败:', error)
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
  formRef.value?.resetFields()
  originalData.value = null
  newTLSHost.value = ''
}

defineExpose({
  openEdit,
  openCreate
})
</script>

<style scoped>
.hosts-config,
.tls-config,
.paths-config {
  width: 100%;
}

.host-item,
.tls-item,
.path-item {
  margin-bottom: 16px;
  padding: 16px;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  background-color: #fafafa;
}

.host-header,
.tls-header,
.path-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 500;
  color: #606266;
}

.host-field {
  margin-bottom: 12px;
}

.host-field label {
  display: block;
  font-size: 12px;
  color: #909399;
  font-weight: 500;
  margin-bottom: 4px;
}

.paths-config {
  margin-top: 12px;
}

.paths-title {
  font-size: 13px;
  font-weight: 500;
  color: #606266;
  margin-bottom: 12px;
}

.path-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.path-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.path-field label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.tls-hosts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
  align-items: center;
}

.tls-secret {
  margin-top: 12px;
}

.tls-secret label {
  display: block;
  font-size: 12px;
  color: #909399;
  font-weight: 500;
  margin-bottom: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
