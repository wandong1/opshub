<template>
  <div class="mount-config-content">
    <div class="mount-section">
      <div class="mount-header">
        <span class="mount-header-title">
          <icon-folder />
          卷挂载配置
        </span>
        <a-button type="primary" @click="addMount" size="default">
          添加挂载点
        </a-button>
      </div>
      <div v-if="localMounts.length > 0" class="mount-table-wrapper">
        <a-table :data="localMounts" class="mount-table" size="default" :columns="tableColumns">
          <template #col_2412="{ record }">
              <a-select
                v-model="record.name"
                placeholder="选择数据卷"
                filterable
                size="small"
                @change="updateMounts"
              >
                <a-option
                  v-for="vol in volumes"
                  :key="vol.name"
                  :label="vol.name"
                  :value="vol.name"
                />
              </a-select>
            </template>
          <template #col_9531="{ record }">
              <a-input
                v-model="record.mountPath"
                placeholder="/container/path"
                size="small"
                @input="updateMounts"
              />
            </template>
          <template #col_9624="{ record }">
              <a-input
                v-model="record.subPath"
                placeholder="可选，挂载卷的子路径"
                size="small"
                @input="updateMounts"
              />
            </template>
          <template #col_2797="{ record }">
              <a-switch
                v-model="record.readOnly"
                active-text="只读"
                inactive-text="读写"
                @change="updateMounts"
              />
            </template>
          <template #actions="{ record }">
              <a-button status="danger" type="text" @click="removeMount(rowIndex)" />
            </template>
        </a-table>
      </div>
      <a-empty v-else description="暂无挂载点配置" :image-size="80" />
    </div>
  </div>
</template>

<script setup lang="ts">
const tableColumns = [
  { title: '卷名', slotName: 'col_2412', width: 150 },
  { title: '挂载路径', slotName: 'col_9531', width: 200 },
  { title: '子路径', slotName: 'col_9624', width: 180 },
  { title: '读写模式', slotName: 'col_2797', width: 120, align: 'center' },
  { title: '操作', slotName: 'actions', width: 80, align: 'center' }
]

import { ref, watch } from 'vue'

interface VolumeMount {
  name: string
  mountPath: string
  subPath?: string
  readOnly?: boolean
}

const props = defineProps<{
  volumeMounts: VolumeMount[]
  volumes: { name: string; type?: string }[]
}>()

const emit = defineEmits<{
  update: [mounts: VolumeMount[]]
}>()

const localMounts = ref<VolumeMount[]>([])

watch(() => props.volumeMounts, (newVal) => {
  localMounts.value = (newVal || []).map(m => ({
    name: m.name || '',
    mountPath: m.mountPath || '',
    subPath: m.subPath || '',
    readOnly: m.readOnly || false
  }))
}, { immediate: true, deep: true })

const addMount = () => {
  localMounts.value.push({
    name: '',
    mountPath: '',
    subPath: '',
    readOnly: false
  })
  updateMounts()
}

const removeMount = (index: number) => {
  localMounts.value.splice(index, 1)
  updateMounts()
}

const updateMounts = () => {
  emit('update', [...localMounts.value])
}
</script>

<style scoped>
.mount-config-content {
  padding: 0;
  background: transparent;
}

.mount-section {
  width: 100%;
}

.mount-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: #d4af37;
  border: 1px solid #d4af37;
  border-radius: 12px 12px 0 0;
  margin-bottom: 0;
}

.mount-header-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  letter-spacing: 0.3px;
}

.mount-header-title .arco-icon {
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: #ffffff;
  border-radius: 6px;
  color: #d4af37;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.mount-header .arco-btn {
  font-weight: 500;
  border-radius: 8px;
  background: #ffffff;
  border: 1px solid #d4af37;
  color: #d4af37;
}

.mount-header .arco-btn:hover {
  background: #fafafa;
  border-color: #c9a227;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.mount-table-wrapper {
  border: 1px solid #e8e8e8;
  border-top: none;
  border-radius: 0 0 12px 12px;
  padding: 20px;
  background: #ffffff;
}

.mount-table {
  background: #ffffff;
  border-radius: 8px;
  overflow: hidden;
}

.mount-table :deep(.arco-table__header-wrapper) {
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
}

.mount-table :deep(.arco-table__header th) {
  background: transparent;
  color: #333;
  font-weight: 600;
  font-size: 13px;
  letter-spacing: 0.3px;
  border-bottom: 1px solid #e8e8e8;
}

.mount-table :deep(.arco-table__body) {
  font-size: 13px;
}

.mount-table :deep(.arco-table__body tr) {
  transition: all 0.3s ease;
}

.mount-table :deep(.arco-table__body tr:hover) {
  background: #fafafa;
}

.mount-table :deep(.arco-table__body td) {
  border-bottom: 1px solid #f0f0f0;
}

.mount-table :deep(.arco-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.mount-table :deep(.arco-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.mount-table :deep(.arco-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.mount-table :deep(.arco-select .arco-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
}

.mount-table :deep(.arco-switch.arco-switch-checked) {
  background-color: #d4af37;
}

:deep(.arco-empty) {
  padding: 60px 0;
}

:deep(.arco-empty__description) {
  color: #999;
}
</style>
