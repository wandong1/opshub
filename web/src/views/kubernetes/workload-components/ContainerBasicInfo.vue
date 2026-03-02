<template>
  <a-row :gutter="20">
    <a-col :span="12">
      <a-form label-position="top" size="default">
        <a-form-item label="容器名称">
          <a-input v-model="localContainer.name" placeholder="输入容器名称" @input="update">
            <template #prefix>
              <icon-storage />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item label="容器镜像">
          <a-input v-model="localContainer.image" placeholder="例如: nginx:latest" @input="update">
            <template #prefix>
              <icon-image />
            </template>
          </a-input>
        </a-form-item>
      </a-form>
    </a-col>
    <a-col :span="12">
      <a-form label-position="top" size="default">
        <a-form-item label="拉取策略">
          <a-select v-model="localContainer.imagePullPolicy" style="width: 100%" placeholder="选择拉取策略" @change="update">
            <a-option label="IfNotPresent - 本地不存在时拉取" value="IfNotPresent" />
            <a-option label="Always - 总是拉取最新" value="Always" />
            <a-option label="Never - 从不拉取" value="Never" />
          </a-select>
        </a-form-item>
      </a-form>
    </a-col>
  </a-row>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'

interface Container {
  name: string
  image: string
  imagePullPolicy?: string
}

const props = defineProps<{
  container: Container
}>()

const emit = defineEmits<{
  update: [container: Container]
}>()

const localContainer = reactive({ ...props.container })

watch(() => props.container, (newVal) => {
  Object.assign(localContainer, newVal)
}, { deep: true })

const update = () => {
  emit('update', { ...localContainer })
}
</script>

<style scoped>
.arco-row {
  padding: 20px;
  background: #ffffff;
  border-radius: 10px;
  border: 1px solid #e8e8e8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.arco-col {
  padding: 0 8px;
}

.arco-form-item {
  margin-bottom: 0;
}

.arco-form-item :deep(.arco-form-item__label) {
  font-weight: 600;
  color: #333;
  font-size: 13px;
  letter-spacing: 0.3px;
  margin-bottom: 8px;
}

.arco-input :deep(.arco-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.arco-input :deep(.arco-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.arco-input :deep(.arco-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.arco-input :deep(.arco-input__prefix) {
  color: #d4af37;
}

.arco-input :deep(.arco-input__prefix-inner) {
  color: #d4af37;
}

.arco-select :deep(.arco-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
}
</style>
