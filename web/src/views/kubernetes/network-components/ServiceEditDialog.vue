<template>
  <el-dialog
    v-model="visible"
    :title="isEdit ? '编辑 Service' : '创建 Service'"
    width="900px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="formData" :rules="rules" label-width="120px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入 Service 名称" :disabled="isEdit" />
      </el-form-item>

      <el-form-item label="命名空间" prop="namespace">
        <el-select v-model="formData.namespace" placeholder="请选择命名空间" :disabled="isEdit" style="width: 100%">
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </el-form-item>

      <el-form-item label="类型" prop="type">
        <el-select v-model="formData.type" placeholder="请选择类型" style="width: 100%">
          <el-option label="ClusterIP" value="ClusterIP" />
          <el-option label="NodePort" value="NodePort" />
          <el-option label="LoadBalancer" value="LoadBalancer" />
          <el-option label="ExternalName" value="ExternalName" />
        </el-select>
      </el-form-item>

      <el-form-item label="Cluster IP">
        <el-input v-model="formData.clusterIP" placeholder="自动分配或手动指定" />
      </el-form-item>

      <el-form-item label="外部 IP">
        <div class="external-ips">
          <div v-for="(ip, index) in formData.externalIPs" :key="index" class="ip-item">
            <el-input v-model="formData.externalIPs[index]" placeholder="外部 IP 地址" />
            <el-button type="danger" link @click="removeExternalIP(index)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <el-button type="primary" link @click="addExternalIP">
            <el-icon><Plus /></el-icon> 添加外部 IP
          </el-button>
        </div>
      </el-form-item>

      <el-form-item label="端口配置">
        <div class="ports-config">
          <div v-for="(port, index) in formData.ports" :key="index" class="port-item">
            <div class="port-header">
              <span>端口 {{ index + 1 }}</span>
              <el-button type="danger" link @click="removePort(index)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <div class="port-grid">
              <div class="port-field">
                <label>端口名称</label>
                <el-input v-model="port.name" placeholder="可选" size="small" />
              </div>
              <div class="port-field">
                <label>协议</label>
                <el-select v-model="port.protocol" size="small" style="width: 100%">
                  <el-option label="TCP" value="TCP" />
                  <el-option label="UDP" value="UDP" />
                </el-select>
              </div>
              <div class="port-field">
                <label>端口</label>
                <el-input-number v-model="port.port" :min="1" :max="65535" size="small" style="width: 100%" />
              </div>
              <div class="port-field">
                <label>目标端口</label>
                <el-input-number v-model="port.targetPort" :min="1" :max="65535" size="small" style="width: 100%" />
              </div>
              <div v-if="formData.type === 'NodePort'" class="port-field">
                <label>NodePort</label>
                <el-input-number v-model="port.nodePort" :min="30000" :max="32767" size="small" style="width: 100%" />
              </div>
            </div>
          </div>
          <el-button type="primary" link @click="addPort" style="margin-top: 8px">
            <el-icon><Plus /></el-icon> 添加端口
          </el-button>
        </div>
      </el-form-item>

      <el-form-item label="选择器">
        <div class="selector-config">
          <div v-for="(selector, index) in selectorList" :key="index" class="selector-item">
            <el-input v-model="selector.key" placeholder="键" style="width: 180px" />
            <el-input v-model="selector.value" placeholder="值" style="width: 180px" />
            <el-button type="danger" link @click="removeSelector(index)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <el-button type="primary" link @click="addSelector">
            <el-icon><Plus /></el-icon> 添加选择器
          </el-button>
        </div>
      </el-form-item>

      <el-form-item label="会话亲和性">
        <el-select v-model="formData.sessionAffinity" placeholder="请选择会话亲和性" style="width: 100%">
          <el-option label="None" value="None" />
          <el-option label="ClientIP" value="ClientIP" />
        </el-select>
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
import { getServiceYAML, updateServiceYAML, createService, type ServiceInfo } from '@/api/kubernetes'

interface PortConfig {
  name?: string
  protocol: string
  port: number
  targetPort: string | number
  nodePort?: number
}

interface SelectorItem {
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

const formData = ref({
  name: '',
  namespace: '',
  type: 'ClusterIP' as any,
  clusterIP: '',
  externalIPs: [] as string[],
  ports: [] as PortConfig[],
  selector: {} as Record<string, string>,
  sessionAffinity: 'None'
})

const selectorList = ref<SelectorItem[]>([])

const rules = {
  name: [{ required: true, message: '请输入 Service 名称', trigger: 'blur' }],
  namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

// 同步 selectorList 和 formData.selector
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

// 打开对话框（编辑模式）
const openEdit = async (service: ServiceInfo, nsList: any[]) => {
  namespaces.value = nsList
  isEdit.value = true
  visible.value = true

  try {
    const response = await getServiceYAML(props.clusterId!, service.namespace, service.name)
    originalData.value = response.items || response

    // 解析数据到表单
    const spec = originalData.value.spec || {}
    const metadata = originalData.value.metadata || {}

    formData.value = {
      name: metadata.name || '',
      namespace: metadata.namespace || '',
      type: spec.type || 'ClusterIP',
      clusterIP: spec.clusterIP || '',
      externalIPs: spec.externalIPs || [],
      ports: (spec.ports || []).map((p: any) => ({
        name: p.name,
        protocol: p.protocol || 'TCP',
        port: p.port,
        targetPort: p.targetPort,
        nodePort: p.nodePort
      })),
      selector: spec.selector || {},
      sessionAffinity: spec.sessionAffinity || 'None'
    }

    // 同步 selector 到 selectorList
    syncSelectorFromForm()

    // 如果没有端口，添加默认端口
    if (formData.value.ports.length === 0) {
      formData.value.ports.push({
        protocol: 'TCP',
        port: 80,
        targetPort: 80
      })
    }
  } catch (error) {
    console.error('获取 Service 详情失败:', error)
    ElMessage.error('获取 Service 详情失败')
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
    type: 'ClusterIP',
    clusterIP: '',
    externalIPs: [],
    ports: [{
      protocol: 'TCP',
      port: 80,
      targetPort: 80
    }],
    selector: {},
    sessionAffinity: 'None'
  }

  // 清空 selectorList
  selectorList.value = []
}

// 添加外部 IP
const addExternalIP = () => {
  formData.value.externalIPs.push('')
}

// 删除外部 IP
const removeExternalIP = (index: number) => {
  formData.value.externalIPs.splice(index, 1)
}

// 添加端口
const addPort = () => {
  formData.value.ports.push({
    protocol: 'TCP',
    port: 80,
    targetPort: 80
  })
}

// 删除端口
const removePort = (index: number) => {
  formData.value.ports.splice(index, 1)
}

// 添加选择器
const addSelector = () => {
  selectorList.value.push({ key: '', value: '' })
}

// 删除选择器
const removeSelector = (index: number) => {
  selectorList.value.splice(index, 1)
}

// 构建保存的数据
const buildSaveData = () => {
  // 构建 selector 对象
  const selector: Record<string, string> = {}
  selectorList.value.forEach(({ key, value }) => {
    if (key && value) {
      selector[key] = value
    }
  })

  // 构建端口数组
  const ports = formData.value.ports
    .filter(p => p.port > 0)
    .map(p => {
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

  // 构建完整的 Service 对象
  const serviceData: any = {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: formData.value.name,
      namespace: formData.value.namespace
    },
    spec: {
      type: formData.value.type,
      selector: selector,
      ports: ports,
      sessionAffinity: formData.value.sessionAffinity
    }
  }

  // 添加 ClusterIP
  if (formData.value.clusterIP) {
    serviceData.spec.clusterIP = formData.value.clusterIP
  }

  // 添加外部 IP
  const validExternalIPs = formData.value.externalIPs.filter(ip => ip)
  if (validExternalIPs.length > 0) {
    serviceData.spec.externalIPs = validExternalIPs
  }

  // 如果是编辑模式，保留原有的 metadata 字段
  if (isEdit.value && originalData.value) {
    serviceData.metadata = {
      ...originalData.value.metadata,
      name: formData.value.name,
      namespace: formData.value.namespace
    }
    // 保留 labels
    if (originalData.value.metadata?.labels) {
      serviceData.metadata.labels = originalData.value.metadata.labels
    }
    // 保留 annotations
    if (originalData.value.metadata?.annotations) {
      serviceData.metadata.annotations = originalData.value.metadata.annotations
    }
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
    ElMessage.error('请至少配置一个端口')
    return
  }

  // 同步 selectorList 到 formData.selector
  syncSelectorToList()

  // 验证选择器
  const hasValidSelector = Object.keys(formData.value.selector).length > 0
  if (!hasValidSelector) {
    ElMessage.error('请配置至少一个选择器')
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
      ElMessage.success('更新成功')
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
  selectorList.value = []
}

defineExpose({
  openEdit,
  openCreate
})
</script>

<style scoped>
.external-ips,
.ports-config,
.selector-config {
  width: 100%;
}

.ip-item,
.port-item,
.selector-item {
  margin-bottom: 12px;
  padding: 16px;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  background-color: #fafafa;
}

.ip-item,
.selector-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* 端口配置样式 */
.port-item {
  padding: 16px;
}

.port-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 500;
  color: #606266;
}

.port-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.port-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.port-field label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
