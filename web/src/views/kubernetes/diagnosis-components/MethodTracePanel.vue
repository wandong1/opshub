<template>
  <div class="method-trace-panel">
    <div v-if="!attached" class="not-attached">
      <a-empty description="请先选择Pod并连接到进程">
        <template #image>
          <icon-link />
        </template>
      </a-empty>
    </div>

    <div v-else class="panel-content">
      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">类名</span>
            <a-input
              v-model="classPattern"
              placeholder="类名表达式 (如: com.example.service.*)"
              style="width: 300px"
              size="default"
              clearable
              :disabled="tracing"
            >
              <template #prefix>
                <icon-folder />
              </template>
            </a-input>
          </div>
          <div class="input-group">
            <span class="input-label">方法名</span>
            <a-input
              v-model="methodPattern"
              placeholder="方法名 (如: doSomething)"
              style="width: 200px"
              size="default"
              clearable
              :disabled="tracing"
            >
              <template #prefix>
                <icon-send />
              </template>
            </a-input>
          </div>
        </div>
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">条件表达式</span>
            <a-input
              v-model="condition"
              placeholder="OGNL 条件表达式 (可选, 如: params[0]>100)"
              style="width: 300px"
              size="default"
              clearable
              :disabled="tracing"
            >
              <template #prefix>
                <icon-filter />
              </template>
            </a-input>
          </div>
          <div class="input-group">
            <span class="input-label">最大调用次数</span>
            <a-input-number
              v-model="maxCount"
              :min="1"
              :max="1000"
              size="default"
              :disabled="tracing"
              style="width: 120px"
            />
          </div>
          <a-checkbox v-model="skipJDKMethod" :disabled="tracing">
            跳过JDK方法
          </a-checkbox>
        </div>
        <div class="toolbar-row actions">
          <a-button
            type="primary"
            @click="startTrace"
            :loading="starting"
            :disabled="tracing || !classPattern || !methodPattern"
          >
            <icon-play-arrow />
            {{ starting ? '启动中...' : '开始追踪' }}
          </a-button>
          <a-button
            @click="stopTrace"
            :disabled="!tracing"
            type="danger"
          >
            <icon-pause-circle /> 停止追踪
          </a-button>
          <a-button @click="clearOutput">
            <icon-delete /> 清空输出
          </a-button>
          <a-divider direction="vertical" />
          <a-tag v-if="tracing" color="green">
            <icon-loading />
            追踪中...
          </a-tag>
          <a-tag v-else color="gray">未追踪</a-tag>
          <span class="trace-count" v-if="traceCount > 0">
            已捕获: {{ traceCount }} 次调用
          </span>
        </div>
      </div>

      <!-- 使用说明 -->
      <a-collapse v-model="showHelp" class="help-collapse">
        <a-collapse-item title="使用说明" name="help">
          <div class="help-content">
            <p><strong>trace 命令</strong> 可以追踪方法的调用路径和耗时，帮助分析性能问题。</p>
            <ul>
              <li><strong>类名</strong>: 支持通配符 * 匹配，如 <code>com.example.*</code> 或完整类名</li>
              <li><strong>方法名</strong>: 支持通配符 * 匹配，如 <code>get*</code> 或 <code>*</code></li>
              <li><strong>条件表达式</strong>: OGNL 表达式，如 <code>params[0]>100</code> 表示第一个参数大于100时才追踪</li>
              <li><strong>跳过JDK方法</strong>: 勾选后不追踪 JDK 内部方法调用，输出更简洁</li>
            </ul>
            <p class="tip">提示: 追踪高频方法时建议设置条件表达式或减少最大调用次数，避免影响应用性能。</p>
          </div>
        </a-collapse-item>
      </a-collapse>

      <!-- 追踪输出 -->
      <div class="output-section">
        <div class="section-header">
          <span>追踪输出</span>
          <span class="output-info">
            <a-tag size="small" color="gray">{{ outputLineCount }} 行</a-tag>
          </span>
        </div>
        <div class="trace-output" ref="outputRef">
          <div v-if="!rawOutput && !tracing" class="empty-output">
            <a-empty description="等待追踪结果...">
              <template #image>
                <icon-search />
              </template>
            </a-empty>
          </div>
          <pre v-else>{{ cleanOutput }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { createArthasWebSocket, executeArthasCommand, type ArthasWSMessage } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

// 表单数据
const classPattern = ref('')
const methodPattern = ref('')
const condition = ref('')
const maxCount = ref(100)
const skipJDKMethod = ref(true)

// 状态
const tracing = ref(false)
const starting = ref(false)
const traceCount = ref(0)
const showHelp = ref<string[]>([])
const rawOutput = ref('')  // 使用原始字符串而不是数组
const outputRef = ref<HTMLElement | null>(null)

// WebSocket 连接
let ws: WebSocket | null = null

// 计算输出行数
const outputLineCount = computed(() => {
  return rawOutput.value.split('\n').length
})

// 清理输出中的 ANSI 转义码
const cleanOutput = computed(() => {
  return rawOutput.value
    .replace(/\x1b\[[0-9;]*m/g, '')
    .replace(/\033\[[0-9;]*m/g, '')
    .replace(/\[\d+;\d+m/g, '')
    .replace(/\[\d+m/g, '')
    .replace(/\[0m/g, '')
    .replace(/\[m/g, '')
})

// 构建 trace 命令
const buildTraceCommand = (): string => {
  let cmd = `trace ${classPattern.value} ${methodPattern.value}`

  // 添加条件表达式
  if (condition.value) {
    cmd += ` '${condition.value}'`
  }

  // 添加选项
  cmd += ` -n ${maxCount.value}`

  if (skipJDKMethod.value) {
    cmd += ' --skipJDKMethod'
  }

  return cmd
}

// 开始追踪
const startTrace = async () => {
  if (!classPattern.value || !methodPattern.value) {
    Message.warning('请输入类名和方法名')
    return
  }

  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    Message.warning('请先选择 Pod 和容器')
    return
  }

  starting.value = true
  traceCount.value = 0
  rawOutput.value = ''

  try {
    // 创建 WebSocket 连接
    ws = createArthasWebSocket({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    ws.onopen = () => {
      // 发送 trace 命令
      const command = buildTraceCommand()
      rawOutput.value = `[INFO] 执行命令: ${command}\n\n`

      const msg: ArthasWSMessage = {
        type: 'command',
        command: command
      }
      ws?.send(JSON.stringify(msg))
      tracing.value = true
      starting.value = false
      Message.success('开始追踪')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'output') {
          // 追踪输出 - 直接追加到原始输出
          const content = data.content
          rawOutput.value += content

          // 统计追踪次数（通过检测特定模式）
          if (content.includes('---ts=') || content.includes('`---')) {
            traceCount.value++
          }

          // 自动滚动到底部
          scrollToBottom()
        } else if (data.type === 'error') {
          rawOutput.value += `\n[ERROR] ${data.content}\n`
          Message.error(data.content)
        } else if (data.type === 'info') {
          rawOutput.value += `[INFO] ${data.content}\n`
        }
      } catch (e) {
        // 如果不是 JSON，直接追加原始数据
        rawOutput.value += event.data
      }
    }

    ws.onerror = (error) => {
      rawOutput.value += '\n[ERROR] WebSocket 连接错误\n'
      tracing.value = false
      starting.value = false
      Message.error('WebSocket 连接失败')
    }

    ws.onclose = () => {
      tracing.value = false
      starting.value = false
      rawOutput.value += '\n[INFO] 追踪已停止\n'
    }

  } catch (error: any) {
    Message.error('启动追踪失败: ' + (error.message || '未知错误'))
    starting.value = false
  }
}

// 停止追踪
const stopTrace = () => {
  if (ws) {
    const msg: ArthasWSMessage = {
      type: 'stop'
    }
    ws.send(JSON.stringify(msg))
    ws.close()
    ws = null
  }
  tracing.value = false
  rawOutput.value += '\n[INFO] 用户停止追踪\n'
  Message.info('已停止追踪')
}

// 清空输出
const clearOutput = () => {
  rawOutput.value = ''
  traceCount.value = 0
}

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick()
  if (outputRef.value) {
    outputRef.value.scrollTop = outputRef.value.scrollHeight
  }
}

// 组件卸载时清理
onUnmounted(() => {
  if (ws) {
    ws.close()
    ws = null
  }
})

// 监听 attached 状态变化
watch(() => props.attached, (newVal) => {
  if (!newVal && ws) {
    ws.close()
    ws = null
    tracing.value = false
  }
})
</script>

<style scoped>
.method-trace-panel {
  min-height: 400px;
}

.not-attached {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.panel-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 工具栏 */
.toolbar {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e9ecef;
}

.toolbar-row {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.toolbar-row:last-child {
  margin-bottom: 0;
}

.toolbar-row.actions {
  padding-top: 12px;
  border-top: 1px solid #e9ecef;
}

.input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.input-label {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
  font-weight: 500;
}

.trace-count {
  font-size: 13px;
  color: #67c23a;
  font-weight: 500;
  margin-left: 8px;
}

/* 帮助折叠面板 */
.help-collapse {
  border: 1px solid #e4e7ed;
  border-radius: 6px;
}

.help-collapse :deep(.arco-collapse-item__header) {
  padding: 0 16px;
  font-size: 13px;
  color: #606266;
}

.help-content {
  padding: 0 8px;
  font-size: 13px;
  color: #606266;
  line-height: 1.8;
}

.help-content ul {
  margin: 8px 0;
  padding-left: 20px;
}

.help-content code {
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: #e6a23c;
}

.help-content .tip {
  background: #fdf6ec;
  padding: 8px 12px;
  border-radius: 4px;
  border-left: 3px solid #e6a23c;
  margin-top: 12px;
}

/* 输出区域 */
.output-section {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  overflow: hidden;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.output-info {
  display: flex;
  gap: 8px;
  align-items: center;
}

.trace-output {
  flex: 1;
  overflow: auto;
  min-height: 300px;
  max-height: 500px;
}

.trace-output pre {
  margin: 0;
  padding: 16px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  line-height: 1.6;
  min-height: 100%;
  white-space: pre-wrap;
  word-break: break-all;
}

.empty-output {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  background: #fafafa;
}

/* 加载动画 */
.is-loading {
  animation: rotating 1s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 响应式 */
@media (max-width: 992px) {
  .toolbar-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .input-group {
    width: 100%;
  }

  .input-group :deep(.arco-input) {
    width: 100% !important;
  }
}
</style>
