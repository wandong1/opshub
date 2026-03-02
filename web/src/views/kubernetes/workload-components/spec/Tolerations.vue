<template>
  <div class="spec-content-wrapper">
    <div class="spec-content-header">
      <h3>容忍度</h3>
      <p>配置 Pod 对节点污点的容忍度</p>
    </div>
    <div class="spec-content">
      <div class="tolerations-table-wrapper">
        <a-table :columns="columns" :data="tolerations" :bordered="{ cell: true }" stripe>
          <template #key="{ record }">
            <a-input v-model="record.key" placeholder="键名，如: key" size="small" />
          </template>
          <template #operator="{ record }">
            <a-select v-model="record.operator" placeholder="运算符" size="small">
              <a-option label="Equal" value="Equal" />
              <a-option label="Exists" value="Exists" />
            </a-select>
          </template>
          <template #value="{ record }">
            <a-input v-model="record.value" placeholder="值" size="small" :disabled="record.operator === 'Exists'" />
          </template>
          <template #effect="{ record }">
            <a-select v-model="record.effect" placeholder="影响类型" size="small">
              <a-option label="NoExecute" value="NoExecute" />
              <a-option label="NoSchedule" value="NoSchedule" />
              <a-option label="PreferNoSchedule" value="PreferNoSchedule" />
            </a-select>
          </template>
          <template #tolerationSeconds="{ record }">
            <a-input v-model="record.tolerationSeconds" placeholder="仅 NoExecute 生效" size="small" :disabled="record.effect !== 'NoExecute'" />
          </template>
          <template #actions="{ rowIndex }">
            <a-button type="text" status="danger" size="small" @click="emit('removeToleration', rowIndex)">删除</a-button>
          </template>
        </a-table>
        <div class="tolerations-actions">
          <a-button type="primary" @click="emit('addToleration')">添加容忍</a-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TableColumnData } from '@arco-design/web-vue'

const columns: TableColumnData[] = [
  { title: '键', dataIndex: 'key', slotName: 'key', width: 150 },
  { title: '运算符', dataIndex: 'operator', slotName: 'operator', width: 120 },
  { title: '值', dataIndex: 'value', slotName: 'value', width: 150 },
  { title: '影响', dataIndex: 'effect', slotName: 'effect', width: 150 },
  { title: '容忍时间(秒)', dataIndex: 'tolerationSeconds', slotName: 'tolerationSeconds', width: 150 },
  { title: '操作', slotName: 'actions', width: 80, fixed: 'right' },
]

interface Toleration {
  key: string
  operator: 'Equal' | 'Exists'
  value: string
  effect: 'NoExecute' | 'NoSchedule' | 'PreferNoSchedule'
  tolerationSeconds?: string
}

const props = defineProps<{
  tolerations: Toleration[]
}>()

const emit = defineEmits<{
  addToleration: []
  removeToleration: [index: number]
}>()
</script>

<style scoped>
.spec-content-wrapper {
  padding: 0;
  background: transparent;
}

.spec-content-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #f0f0f0;
}

.spec-content-header h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.spec-content-header p {
  margin: 0;
  font-size: 13px;
  color: #909399;
}

.spec-content {
  background: #fff;
}

.tolerations-table-wrapper {
  width: 100%;
}

.tolerations-table-wrapper .arco-table {
  width: 100%;
  margin-bottom: 16px;
}

.tolerations-actions {
  display: flex;
  justify-content: flex-start;
  padding-top: 8px;
}
</style>
