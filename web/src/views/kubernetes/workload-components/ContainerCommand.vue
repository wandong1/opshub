<template>
  <div class="command-form-wrapper">
    <!-- 工作目录 -->
    <div class="form-row-group">
      <div class="form-row-item full-width">
        <label class="form-label">工作目录</label>
        <a-input
          v-model="localContainer.workingDir"
          placeholder="容器的工作目录，如: /app"
          clearable
          @input="update"
        />
      </div>
    </div>

    <!-- 运行命令 -->
    <div class="form-section-block">
      <label class="form-label">运行命令 (Command)</label>
      <div class="command-list">
        <div v-for="(cmd, index) in localContainer.command" :key="'cmd-'+index" class="command-item">
          <a-input v-model="localContainer.command[index]" placeholder="命令参数" @input="update">
            <template #prepend>{{ index + 1 }}</template>
          </a-input>
          <a-button status="danger" type="text" @click="removeCommand(index)" />
        </div>
      </div>
      <div class="add-btn-wrapper">
        <a-button class="add-btn" type="primary" size="small" @click="addCommand">
          添加命令
        </a-button>
      </div>
    </div>

    <!-- 启动参数 -->
    <div class="form-section-block">
      <label class="form-label">启动参数 (Args)</label>
      <div class="command-list">
        <div v-for="(arg, index) in localContainer.args" :key="'arg-'+index" class="command-item">
          <a-input v-model="localContainer.args[index]" placeholder="参数值" @input="update">
            <template #prepend>{{ index + 1 }}</template>
          </a-input>
          <a-button status="danger" type="text" @click="removeArg(index)" />
        </div>
      </div>
      <div class="add-btn-wrapper">
        <a-button class="add-btn" type="primary" size="small" @click="addArg">
          添加参数
        </a-button>
      </div>
    </div>

    <!-- 交互选项 -->
    <div class="form-section-block">
      <label class="form-label">交互选项</label>
      <div class="checkbox-group">
        <a-checkbox v-model="localContainer.stdin" @change="update">保持标准输入开启 (stdin)</a-checkbox>
        <a-checkbox v-model="localContainer.tty" @change="update">分配终端 (TTY)</a-checkbox>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'

interface Container {
  workingDir?: string
  command: string[]
  args: string[]
  stdin?: boolean
  tty?: boolean
}

const props = defineProps<{
  container: Container
}>()

const emit = defineEmits<{
  update: [container: Container]
}>()

const localContainer = reactive<Container>({
  workingDir: '',
  command: [],
  args: [],
  stdin: false,
  tty: false
})

watch(() => props.container, (newVal) => {
  localContainer.workingDir = newVal.workingDir || ''
  localContainer.command = newVal.command || []
  localContainer.args = newVal.args || []
  localContainer.stdin = newVal.stdin || false
  localContainer.tty = newVal.tty || false
}, { deep: true, immediate: true })

const update = () => {
  emit('update', { ...localContainer })
}

const addCommand = () => {
  if (!localContainer.command) localContainer.command = []
  localContainer.command.push('')
  update()
}

const removeCommand = (index: number) => {
  localContainer.command.splice(index, 1)
  update()
}

const addArg = () => {
  if (!localContainer.args) localContainer.args = []
  localContainer.args.push('')
  update()
}

const removeArg = (index: number) => {
  localContainer.args.splice(index, 1)
  update()
}
</script>

<style scoped>
.command-form-wrapper {
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0;
}

.form-row-group {
  display: flex;
  gap: 12px;
}

.form-row-item {
  flex: 1;
}

.form-row-item.full-width {
  flex: 100%;
}

.form-label {
  display: block;
  margin-bottom: 8px;
  font-weight: 600;
  color: #333;
  font-size: 13px;
  letter-spacing: 0.3px;
}

.form-section-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 18px;
  background: #ffffff;
  border-radius: 10px;
  border: 1px solid #e8e8e8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.form-section-block .form-label {
  color: #1a1a1a;
  font-size: 14px;
}

.command-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.command-item {
  display: flex;
  gap: 10px;
  align-items: center;
}

.command-item .arco-input :deep(.arco-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.command-item .arco-input :deep(.arco-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.command-item .arco-input :deep(.arco-input-group__prepend) {
  background: #f5f5f5;
  border: 1px solid #e0e0e0;
  color: #d4af37;
  font-weight: 600;
}

.add-btn-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 4px;
}

.add-btn {
  border-radius: 6px;
  font-weight: 500;
  background: #d4af37;
  border: none;
  color: #1a1a1a;
}

.add-btn:hover {
  background: #c9a227;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
}

.checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
  border-radius: 8px;
  border: 1px solid #e8e8e8;
}

.checkbox-group :deep(.arco-checkbox) {
  font-weight: 500;
}

.checkbox-group :deep(.arco-checkbox__label) {
  color: #333;
}

.checkbox-group :deep(.arco-checkbox__inner) {
  background: #ffffff;
  border: 1px solid #d0d0d0;
}

.checkbox-group :deep(.arco-checkbox__input.is-checked .arco-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}
</style>
