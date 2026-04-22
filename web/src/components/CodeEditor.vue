<template>
  <div class="code-editor-wrapper">
    <Codemirror
      v-model="code"
      :placeholder="placeholder"
      :style="{ height: height, fontSize: '14px' }"
      :autofocus="false"
      :indent-with-tab="true"
      :tab-size="2"
      :extensions="extensions"
      @ready="handleReady"
      @change="handleChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, shallowRef } from 'vue'
import { Codemirror } from 'vue-codemirror'
import { python } from '@codemirror/lang-python'
import { javascript } from '@codemirror/lang-javascript'
import { sql } from '@codemirror/lang-sql'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'

interface Props {
  modelValue: string
  language?: 'shell' | 'python' | 'javascript' | 'sql' | 'promql'
  placeholder?: string
  height?: string
  theme?: 'light' | 'dark'
}

const props = withDefaults(defineProps<Props>(), {
  language: 'shell',
  placeholder: '请输入代码...',
  height: '200px',
  theme: 'light'
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const code = ref(props.modelValue)

// 监听外部值变化
watch(() => props.modelValue, (newVal) => {
  if (newVal !== code.value) {
    code.value = newVal
  }
})

// 语言扩展
const languageExtension = computed(() => {
  switch (props.language) {
    case 'python':
      return python()
    case 'javascript':
      return javascript()
    case 'sql':
      return sql()
    case 'promql':
      return sql() // PromQL 使用 SQL 高亮
    case 'shell':
    default:
      return javascript() // Shell 使用 JavaScript 高亮作为替代
  }
})

// 自定义浅色主题
const lightTheme = EditorView.theme({
  '&': {
    backgroundColor: '#ffffff',
    color: '#1d2129',
    fontSize: '14px',
    border: '1px solid #e5e6eb',
    borderRadius: '4px'
  },
  '.cm-content': {
    caretColor: '#165dff',
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    padding: '8px 0'
  },
  '.cm-cursor, .cm-dropCursor': {
    borderLeftColor: '#165dff'
  },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
    backgroundColor: '#d4e3fc'
  },
  '.cm-activeLine': {
    backgroundColor: '#f7f8fa'
  },
  '.cm-gutters': {
    backgroundColor: '#f7f8fa',
    color: '#86909c',
    border: 'none',
    borderRight: '1px solid #e5e6eb'
  },
  '.cm-activeLineGutter': {
    backgroundColor: '#e8f3ff'
  },
  '.cm-lineNumbers .cm-gutterElement': {
    padding: '0 8px',
    minWidth: '32px'
  }
}, { dark: false })

// 扩展配置
const extensions = computed(() => {
  const exts = [
    languageExtension.value,
    EditorView.lineWrapping
  ]

  if (props.theme === 'dark') {
    exts.push(oneDark)
  } else {
    exts.push(lightTheme)
  }

  return exts
})

const view = shallowRef()

const handleReady = (payload: any) => {
  view.value = payload.view
}

const handleChange = (value: string) => {
  emit('update:modelValue', value)
}
</script>

<style scoped>
.code-editor-wrapper {
  width: 100%;
}

.code-editor-wrapper :deep(.cm-editor) {
  outline: none;
}

.code-editor-wrapper :deep(.cm-scroller) {
  overflow: auto;
  font-family: Menlo, Monaco, "Courier New", monospace;
}

.code-editor-wrapper :deep(.cm-focused) {
  outline: none;
  border-color: #165dff;
  box-shadow: 0 0 0 2px rgba(22, 93, 255, 0.1);
}
</style>
