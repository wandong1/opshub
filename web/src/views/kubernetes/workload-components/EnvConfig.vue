<template>
  <div class="env-config-content">
    <div class="env-tabs-wrapper">
      <a-tabs v-model:active-key="activeEnvTab" class="env-type-tabs">
        <a-tab-pane title="普通变量" key="normal">
          <div class="env-section">
            <div class="env-header">
              <span class="env-header-title">
                <icon-safe />
                普通环境变量
              </span>
              <a-button type="primary" @click="showAddEnvDialog('normal')" size="default">
                添加变量
              </a-button>
            </div>
            <div v-if="normalEnvs.length > 0" class="env-table-wrapper">
              <a-table :data="normalEnvs" class="env-table" size="default" :columns="tableColumns3">
          <template #name="{ record }">
                    <span class="env-name">{{ record.name }}</span>
                  </template>
          <template #value="{ record }">
                    <div class="env-value-cell">
                      <a-input
                        v-if="record.editing"
                        v-model="record.tempValue"
                        placeholder="请输入变量值"
                        size="small"
                      />
                      <span v-else class="env-value">{{ record.value || '-' }}</span>
                    </div>
                  </template>
          <template #actions="{ record }">
                    <div class="action-buttons">
                      <template v-if="record.editing">
                        <a-button type="text" status="success" size="small" @click="saveEnvEdit(record, rowIndex)">
                          <icon-check />
                        </a-button>
                        <a-button type="text" size="small" @click="cancelEnvEdit(record, rowIndex)">
                          <icon-close />
                        </a-button>
                      </template>
                      <template v-else>
                        <a-button type="text" size="small" @click="editEnv(record, rowIndex)">
                          <icon-edit />
                        </a-button>
                        <a-button status="danger" type="text" size="small" @click="removeEnv('normal', rowIndex)">
                          <icon-delete />
                        </a-button>
                      </template>
                    </div>
                  </template>
        </a-table>
            </div>
            <a-empty v-else description="暂无环境变量配置" :image-size="80" />
          </div>
        </a-tab-pane>

        <a-tab-pane title="配置映射引用" key="configmap">
          <div class="env-section">
            <div class="env-header">
              <span class="env-header-title">
                <icon-file />
                ConfigMap 引用
              </span>
              <a-button type="primary" @click="showAddEnvDialog('configmap')" size="default">
                添加引用
              </a-button>
            </div>
            <div v-if="configmapEnvs.length > 0" class="env-table-wrapper">
              <a-table :data="configmapEnvs" class="env-table" size="default" :columns="tableColumns2">
          <template #name="{ record }">
                    <span class="env-name">{{ record.name }}</span>
                  </template>
          <template #configmapName="{ record }">
                    <span class="env-resource">{{ record.configmapName }}</span>
                  </template>
          <template #key="{ record }">
                    <span class="env-key">{{ record.key }}</span>
                  </template>
          <template #actions="{ record }">
                    <div class="action-buttons">
                      <a-button type="text" size="small" @click="editConfigMapEnv(record, rowIndex)">
                        <icon-edit />
                      </a-button>
                      <a-button status="danger" type="text" size="small" @click="removeEnv('configmap', rowIndex)">
                        <icon-delete />
                      </a-button>
                    </div>
                  </template>
        </a-table>
            </div>
            <a-empty v-else description="暂无 ConfigMap 引用" :image-size="80" />
          </div>
        </a-tab-pane>

        <a-tab-pane title="密钥引用" key="secret">
          <div class="env-section">
            <div class="env-header">
              <span class="env-header-title">
                <icon-lock />
                Secret 引用
              </span>
              <a-button type="primary" @click="showAddEnvDialog('secret')" size="default">
                添加引用
              </a-button>
            </div>
            <div v-if="secretEnvs.length > 0" class="env-table-wrapper">
              <a-table :data="secretEnvs" class="env-table" size="default" :columns="tableColumns">
          <template #name="{ record }">
                    <span class="env-name">{{ record.name }}</span>
                  </template>
          <template #secretName="{ record }">
                    <span class="env-resource">{{ record.secretName }}</span>
                  </template>
          <template #key="{ record }">
                    <span class="env-key">{{ record.key }}</span>
                  </template>
          <template #actions="{ record }">
                    <div class="action-buttons">
                      <a-button type="text" size="small" @click="editSecretEnv(record, rowIndex)">
                        <icon-edit />
                      </a-button>
                      <a-button status="danger" type="text" size="small" @click="removeEnv('secret', rowIndex)">
                        <icon-delete />
                      </a-button>
                    </div>
                  </template>
        </a-table>
            </div>
            <a-empty v-else description="暂无 Secret 引用" :image-size="80" />
          </div>
        </a-tab-pane>
      </a-tabs>
    </div>

    <!-- 添加/编辑普通变量对话框 -->
    <a-modal
      v-model="normalEnvDialogVisible"
      :title="editingEnvIndex >= 0 ? '编辑环境变量' : '添加环境变量'"
      width="600px"
    >
      <a-form :model="normalEnvForm" label-width="100px" label-position="left">
        <a-form-item label="变量名称" required>
          <a-input v-model="normalEnvForm.name" placeholder="例如: DATABASE_URL" allow-clear />
        </a-form-item>
        <a-form-item label="变量值" required>
          <a-input
            v-model="normalEnvForm.value"
            type="textarea"
            :rows="3"
            placeholder="请输入变量值"
          />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="normalEnvDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="saveNormalEnv">确定</a-button>
      </template>
    </a-modal>

    <!-- 添加/编辑 ConfigMap 引用对话框 -->
    <a-modal
      v-model="configmapEnvDialogVisible"
      :title="editingConfigMapIndex >= 0 ? '编辑 ConfigMap 引用' : '添加 ConfigMap 引用'"
      width="600px"
    >
      <a-form :model="configmapEnvForm" label-width="120px" label-position="left">
        <a-form-item label="变量名称" required>
          <a-input v-model="configmapEnvForm.name" placeholder="环境变量名称" allow-clear />
        </a-form-item>
        <a-form-item label="ConfigMap" required>
          <a-select
            v-model="configmapEnvForm.configmapName"
            placeholder="选择 ConfigMap"
            style="width: 100%"
            filterable
          >
            <a-option
              v-for="cm in configmapList"
              :key="cm.name"
              :label="cm.name"
              :value="cm.name"
            />
          </a-select>
        </a-form-item>
        <a-form-item label="Key" required>
          <a-input v-model="configmapEnvForm.key" placeholder="ConfigMap 中的键名" allow-clear />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="configmapEnvDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="saveConfigMapEnv">确定</a-button>
      </template>
    </a-modal>

    <!-- 添加/编辑 Secret 引用对话框 -->
    <a-modal
      v-model="secretEnvDialogVisible"
      :title="editingSecretIndex >= 0 ? '编辑 Secret 引用' : '添加 Secret 引用'"
      width="600px"
    >
      <a-form :model="secretEnvForm" label-width="120px" label-position="left">
        <a-form-item label="变量名称" required>
          <a-input v-model="secretEnvForm.name" placeholder="环境变量名称" allow-clear />
        </a-form-item>
        <a-form-item label="Secret" required>
          <a-select
            v-model="secretEnvForm.secretName"
            placeholder="选择 Secret"
            style="width: 100%"
            filterable
          >
            <a-option
              v-for="sec in secretList"
              :key="sec.name"
              :label="sec.name"
              :value="sec.name"
            />
          </a-select>
        </a-form-item>
        <a-form-item label="Key" required>
          <a-input v-model="secretEnvForm.key" placeholder="Secret 中的键名" allow-clear />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="secretEnvDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="saveSecretEnv">确定</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
const tableColumns3 = [
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 180 },
  { title: '值', dataIndex: 'value', slotName: 'value', width: 250 },
  { title: '操作', slotName: 'actions', width: 150, align: 'center' }
]

const tableColumns2 = [
  { title: '变量名称', dataIndex: 'name', slotName: 'name', width: 180 },
  { title: 'ConfigMap', dataIndex: 'configmapName', slotName: 'configmapName', width: 150 },
  { title: 'Key', dataIndex: 'key', slotName: 'key', width: 150 },
  { title: '操作', slotName: 'actions', width: 150, align: 'center' }
]

const tableColumns = [
  { title: '变量名称', dataIndex: 'name', slotName: 'name', width: 180 },
  { title: 'Secret', dataIndex: 'secretName', slotName: 'secretName', width: 150 },
  { title: 'Key', dataIndex: 'key', slotName: 'key', width: 150 },
  { title: '操作', slotName: 'actions', width: 150, align: 'center' }
]

import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'

// 环境变量接口定义
interface NormalEnv {
  name: string
  value: string
  valueFrom?: never
  editing?: boolean
  tempValue?: string
}

interface ConfigMapEnv {
  name: string
  configmapName: string
  key: string
  valueFrom: {
    type: 'configmap'
    configMapName: string
    key: string
  }
}

interface SecretEnv {
  name: string
  secretName: string
  key: string
  valueFrom: {
    type: 'secret'
    secretName: string
    key: string
  }
}

type EnvVar = NormalEnv | ConfigMapEnv | SecretEnv

const props = defineProps<{
  envs: EnvVar[]
  configmapList?: { name: string }[]
  secretList?: { name: string }[]
}>()

const emit = defineEmits<{
  update: [envs: EnvVar[]]
}>()

const activeEnvTab = ref('normal')
const normalEnvDialogVisible = ref(false)
const configmapEnvDialogVisible = ref(false)
const secretEnvDialogVisible = ref(false)
const editingEnvIndex = ref(-1)
const editingConfigMapIndex = ref(-1)
const editingSecretIndex = ref(-1)

// 普通环境变量表单
const normalEnvForm = ref({
  name: '',
  value: ''
})

// ConfigMap 环境变量表单
const configmapEnvForm = ref({
  name: '',
  configmapName: '',
  key: ''
})

// Secret 环境变量表单
const secretEnvForm = ref({
  name: '',
  secretName: '',
  key: ''
})

// 分离不同类型的环境变量
const normalEnvs = computed(() => {
  return props.envs.filter(env => !env.valueFrom) as NormalEnv[]
})

const configmapEnvs = computed(() => {
  return props.envs.filter(env => env.valueFrom?.type === 'configmap') as ConfigMapEnv[]
})

const secretEnvs = computed(() => {
  return props.envs.filter(env => env.valueFrom?.type === 'secret') as SecretEnv[]
})

// 显示添加对话框
const showAddEnvDialog = (type: string) => {
  if (type === 'normal') {
    normalEnvForm.value = { name: '', value: '' }
    editingEnvIndex.value = -1
    normalEnvDialogVisible.value = true
  } else if (type === 'configmap') {
    configmapEnvForm.value = { name: '', configmapName: '', key: '' }
    editingConfigMapIndex.value = -1
    configmapEnvDialogVisible.value = true
  } else if (type === 'secret') {
    secretEnvForm.value = { name: '', secretName: '', key: '' }
    editingSecretIndex.value = -1
    secretEnvDialogVisible.value = true
  }
}

// 编辑普通环境变量
const editEnv = (row: NormalEnv, index: number) => {
  row.editing = true
  row.tempValue = row.value
}

const saveEnvEdit = (row: NormalEnv, index: number) => {
  if (row.tempValue !== undefined) {
    row.value = row.tempValue
  }
  row.editing = false
  row.tempValue = undefined
  updateEnvs()
}

const cancelEnvEdit = (row: NormalEnv, index: number) => {
  row.editing = false
  row.tempValue = undefined
}

// 保存普通环境变量
const saveNormalEnv = () => {
  if (!normalEnvForm.value.name) {
    Message.warning('请输入变量名称')
    return
  }
  if (!normalEnvForm.value.value) {
    Message.warning('请输入变量值')
    return
  }

  const newEnv: NormalEnv = {
    name: normalEnvForm.value.name,
    value: normalEnvForm.value.value
  }

  if (editingEnvIndex.value >= 0) {
    const normalEnvList = normalEnvs.value
    normalEnvList[editingEnvIndex.value] = newEnv
  } else {
    normalEnvs.value.push(newEnv)
  }

  normalEnvDialogVisible.value = false
  updateEnvs()
}

// 编辑 ConfigMap 引用
const editConfigMapEnv = (row: ConfigMapEnv, index: number) => {
  configmapEnvForm.value = {
    name: row.name,
    configmapName: row.configmapName,
    key: row.key
  }
  editingConfigMapIndex.value = index
  configmapEnvDialogVisible.value = true
}

// 保存 ConfigMap 引用
const saveConfigMapEnv = () => {
  if (!configmapEnvForm.value.name) {
    Message.warning('请输入变量名称')
    return
  }
  if (!configmapEnvForm.value.configmapName) {
    Message.warning('请选择 ConfigMap')
    return
  }
  if (!configmapEnvForm.value.key) {
    Message.warning('请输入 Key')
    return
  }

  const newEnv: ConfigMapEnv = {
    name: configmapEnvForm.value.name,
    configmapName: configmapEnvForm.value.configmapName,
    key: configmapEnvForm.value.key,
    valueFrom: {
      type: 'configmap',
      configMapName: configmapEnvForm.value.configmapName,
      key: configmapEnvForm.value.key
    }
  }

  if (editingConfigMapIndex.value >= 0) {
    const configmapEnvList = configmapEnvs.value
    configmapEnvList[editingConfigMapIndex.value] = newEnv
  } else {
    configmapEnvs.value.push(newEnv)
  }

  configmapEnvDialogVisible.value = false
  updateEnvs()
}

// 编辑 Secret 引用
const editSecretEnv = (row: SecretEnv, index: number) => {
  secretEnvForm.value = {
    name: row.name,
    secretName: row.secretName,
    key: row.key
  }
  editingSecretIndex.value = index
  secretEnvDialogVisible.value = true
}

// 保存 Secret 引用
const saveSecretEnv = () => {
  if (!secretEnvForm.value.name) {
    Message.warning('请输入变量名称')
    return
  }
  if (!secretEnvForm.value.secretName) {
    Message.warning('请选择 Secret')
    return
  }
  if (!secretEnvForm.value.key) {
    Message.warning('请输入 Key')
    return
  }

  const newEnv: SecretEnv = {
    name: secretEnvForm.value.name,
    secretName: secretEnvForm.value.secretName,
    key: secretEnvForm.value.key,
    valueFrom: {
      type: 'secret',
      secretName: secretEnvForm.value.secretName,
      key: secretEnvForm.value.key
    }
  }

  if (editingSecretIndex.value >= 0) {
    const secretEnvList = secretEnvs.value
    secretEnvList[editingSecretIndex.value] = newEnv
  } else {
    secretEnvs.value.push(newEnv)
  }

  secretEnvDialogVisible.value = false
  updateEnvs()
}

// 删除环境变量
const removeEnv = (type: string, index: number) => {
  const updatedEnvs = [...props.envs]

  if (type === 'normal') {
    const normalEnvList = normalEnvs.value
    const targetEnv = normalEnvList[index]
    const globalIndex = updatedEnvs.findIndex(env => env === targetEnv)
    if (globalIndex >= 0) {
      updatedEnvs.splice(globalIndex, 1)
    }
  } else if (type === 'configmap') {
    const configmapEnvList = configmapEnvs.value
    const targetEnv = configmapEnvList[index]
    const globalIndex = updatedEnvs.findIndex(env => env === targetEnv)
    if (globalIndex >= 0) {
      updatedEnvs.splice(globalIndex, 1)
    }
  } else if (type === 'secret') {
    const secretEnvList = secretEnvs.value
    const targetEnv = secretEnvList[index]
    const globalIndex = updatedEnvs.findIndex(env => env === targetEnv)
    if (globalIndex >= 0) {
      updatedEnvs.splice(globalIndex, 1)
    }
  }

  emit('update', updatedEnvs)
}

// 更新环境变量列表
const updateEnvs = () => {
  const updatedEnvs: EnvVar[] = [
    ...normalEnvs.value,
    ...configmapEnvs.value,
    ...secretEnvs.value
  ]
  emit('update', updatedEnvs)
}
</script>

<style scoped>
.env-config-content {
  padding: 0;
  background: #fff;
}

.env-tabs-wrapper {
  width: 100%;
}

.env-type-tabs {
  width: 100%;
}

.env-section {
  width: 100%;
}

.env-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: #d4af37;
  border: 1px solid #d4af37;
  border-radius: 12px 12px 0 0;
  margin-bottom: 0;
}

.env-header-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.env-header-title .arco-icon {
  font-size: 18px;
  color: #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: #ffffff;
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.env-header .arco-btn {
  background: #ffffff;
  border: 1px solid #d4af37;
  color: #d4af37;
  font-weight: 500;
}

.env-header .arco-btn:hover {
  background: #fafafa;
  border-color: #c9a227;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.env-table-wrapper {
  border: 1px solid #e8e8e8;
  border-top: none;
  border-radius: 0 0 12px 12px;
  padding: 20px;
  background: #ffffff;
}

.env-table {
  background: #fff;
  border-radius: 8px;
}

.env-table :deep(.arco-table__header-wrapper) {
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
}

.env-table :deep(.arco-table__header th) {
  background: transparent;
  color: #333;
  font-weight: 600;
  border-bottom: 1px solid #e8e8e8;
}

.env-table :deep(.arco-table__body tr) {
  transition: all 0.3s ease;
}

.env-table :deep(.arco-table__body tr:hover) {
  background: #fafafa;
}

.env-table :deep(.arco-table__body td) {
  border-bottom: 1px solid #f0f0f0;
}

.env-name {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  color: #1a1a1a;
  font-weight: 600;
}

.env-value {
  color: #666;
  word-break: break-all;
}

.env-value-cell {
  display: flex;
  align-items: center;
}

.env-resource {
  color: #d4af37;
  font-weight: 600;
}

.env-key {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  color: #52c41a;
  font-weight: 600;
}

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

:deep(.arco-dialog__body) {
  padding: 20px;
}

:deep(.arco-form-item__label) {
  font-weight: 600;
  color: #333;
}

:deep(.arco-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  transition: all 0.3s ease;
}

:deep(.arco-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 0 0 3px rgba(212, 175, 55, 0.1);
}

:deep(.arco-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 0 0 4px rgba(212, 175, 55, 0.15);
}

:deep(.arco-tabs__content) {
  padding-top: 16px;
}

:deep(.arco-tabs__item) {
  font-size: 14px;
  font-weight: 500;
}

:deep(.arco-tabs__item.is-active) {
  color: #d4af37;
}

:deep(.arco-tabs__active-bar) {
  background: #d4af37;
}

:deep(.arco-empty) {
  padding: 40px 0;
}

:deep(.arco-empty__description) {
  color: #999;
}
</style>
