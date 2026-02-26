<template>
  <div class="var-input-wrapper" :class="{ focused: isFocused, multiline: multiline }">
    <div
      ref="editorRef"
      class="var-input-editor"
      :class="{ 'var-input-multiline': multiline }"
      contenteditable="true"
      :data-placeholder="placeholder"
      @input="handleInput"
      @keydown="handleKeydown"
      @focus="isFocused = true"
      @blur="handleBlur"
      @contextmenu="handleContextMenu"
      @click="handleEditorClick"
    />
    <div v-if="multiline" class="var-input-toolbar">
      <a-button size="mini" type="text" @mousedown.prevent="openFullscreenEdit">
        <template #icon><icon-expand /></template>全屏编辑
      </a-button>
      <a-button size="mini" type="text" @mousedown.prevent="formatJson">
        <template #icon><icon-code /></template>JSON 美化
      </a-button>
    </div>
    <!-- Dropdown rendered as teleported fixed-position panel -->
    <teleport to="body">
      <div
        v-if="popoverVisible"
        ref="dropdownRef"
        class="var-dropdown-panel"
        :style="dropdownStyle"
        @mousedown.prevent
      >
        <input
          ref="searchInputRef"
          v-model="searchText"
          class="var-dropdown-search"
          placeholder="搜索变量..."
          @keydown="handleSearchKeydown"
        />
        <div class="var-dropdown-list">
          <div v-if="filteredVars.length === 0" class="var-empty">暂无匹配变量</div>
          <div
            v-for="(v, i) in filteredVars"
            :key="v.name"
            class="var-option"
            :class="{ active: highlightIndex === i }"
            @mousedown.prevent="insertVariable(v)"
          >
            <icon-code style="font-size: 14px; color: #165dff; flex-shrink: 0;" />
            <span class="var-option-name">{{ v.name }}</span>
            <span v-if="v.description" class="var-option-desc">{{ v.description }}</span>
          </div>
        </div>
      </div>
    </teleport>
    <!-- Fullscreen edit modal -->
    <a-modal v-model:visible="fullscreenVisible" title="全屏编辑" :width="750" unmount-on-close @before-ok="handleFullscreenConfirm">
      <textarea
        ref="fullscreenTextareaRef"
        v-model="fullscreenText"
        class="var-fullscreen-textarea"
        placeholder="输入内容，变量使用 {{变量名}} 语法"
      />
    </a-modal>
    <!-- Collapse value viewer/editor modal -->
    <a-modal v-model:visible="collapseViewVisible" title="查看完整内容" :width="700" unmount-on-close @before-ok="handleCollapseViewConfirm">
      <div style="display: flex; justify-content: flex-end; margin-bottom: 8px;">
        <a-button size="small" @click="copyCollapseText"><template #icon><icon-copy /></template>复制</a-button>
      </div>
      <textarea v-model="collapseViewText" class="var-fullscreen-textarea" style="min-height: 300px;" />
    </a-modal>
    <!-- Right-click context menu -->
    <teleport to="body">
      <div
        v-if="contextMenuVisible"
        class="var-context-menu"
        :style="contextMenuStyle"
        @mousedown.prevent
      >
        <div class="var-context-item" @click="insertFileAsBase64('image')">
          <icon-image style="font-size: 14px;" /> 插入图片（Base64）
        </div>
        <div class="var-context-item" @click="insertFileAsBase64('audio')">
          <icon-music style="font-size: 14px;" /> 插入音频（Base64）
        </div>
      </div>
    </teleport>
    <input ref="fileInputRef" type="file" style="display: none;" @change="handleFileSelected" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount, reactive } from 'vue'
import { IconCode, IconExpand, IconImage, IconMusic, IconCopy } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'

interface Variable {
  name: string
  description?: string
}

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  variables?: Variable[]
  multiline?: boolean
}>(), {
  modelValue: '',
  placeholder: '输入内容，按 / 插入变量',
  variables: () => [],
  multiline: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorRef = ref<HTMLDivElement>()
const dropdownRef = ref<HTMLDivElement>()
const searchInputRef = ref<HTMLInputElement>()
const isFocused = ref(false)
const popoverVisible = ref(false)
const searchText = ref('')
const highlightIndex = ref(0)
const dropdownStyle = reactive({ top: '0px', left: '0px' })
// Save editor cursor so we can restore it after search input gets focus
const savedRange = ref<Range | null>(null)

// Collapse long JSON string values into pills
const COLLAPSE_THRESHOLD = 500
const collapsedMap = new Map<string, string>()
let collapseIdCounter = 0
const collapseViewVisible = ref(false)
const collapseViewText = ref('')
const collapseViewId = ref('')

// Right-click context menu
const contextMenuVisible = ref(false)
const contextMenuStyle = reactive({ top: '0px', left: '0px' })
const fileInputRef = ref<HTMLInputElement>()
const pendingFileType = ref<'image' | 'audio'>('image')

const filteredVars = computed(() => {
  const kw = searchText.value.toLowerCase()
  if (!kw) return props.variables
  return props.variables.filter(v =>
    v.name.toLowerCase().includes(kw) || (v.description || '').toLowerCase().includes(kw)
  )
})

function pillHtml(name: string): string {
  return `<span class="var-pill" contenteditable="false" data-var="${name}"><svg viewBox="0 0 48 48" width="12" height="12" fill="currentColor" style="flex-shrink:0"><path d="M16 4l-8 22h10l-4 18 24-26H26l6-14z"/></svg>${name}</span>`
}

function collapsePillHtml(id: string, byteLen: number): string {
  const label = byteLen > 1024 ? `${(byteLen / 1024).toFixed(1)}KB` : `${byteLen}B`
  // Folder/file icon SVG — visually distinct from the variable lightning bolt
  return `<span class="var-collapse-pill" contenteditable="false" data-collapse-id="${id}" title="点击查看完整内容"><svg viewBox="0 0 48 48" width="12" height="12" fill="currentColor" style="flex-shrink:0"><path d="M6 9a3 3 0 0 1 3-3h12l4 4h14a3 3 0 0 1 3 3v24a3 3 0 0 1-3 3H9a3 3 0 0 1-3-3V9z"/></svg>${label}</span>`
}

function valueToHtml(val: string): string {
  if (!val) return ''
  // Step 1: Collapse long quoted JSON string values into pills
  let processed = val.replace(/"((?:[^"\\]|\\.)*)"/g, (match, content: string) => {
    if (content.length <= COLLAPSE_THRESHOLD) return match
    const id = `c_${collapseIdCounter++}`
    collapsedMap.set(id, match) // store full quoted string
    return collapsePillHtml(id, content.length)
  })
  // Step 2: Replace {{var}} with variable pills
  processed = processed.replace(/\{\{(\w+)\}\}/g, (_, name) => pillHtml(name))
  return processed
}

function htmlToValue(el: HTMLElement): string {
  let result = ''
  for (const node of Array.from(el.childNodes)) {
    if (node.nodeType === Node.TEXT_NODE) {
      result += node.textContent || ''
    } else if (node.nodeType === Node.ELEMENT_NODE) {
      const elem = node as HTMLElement
      if (elem.classList.contains('var-pill')) {
        result += `{{${elem.dataset.var || ''}}}`
      } else if (elem.classList.contains('var-collapse-pill')) {
        const id = elem.dataset.collapseId || ''
        result += collapsedMap.get(id) || ''
      } else if (elem.tagName === 'BR') {
        result += '\n'
      } else {
        result += elem.textContent || ''
      }
    }
  }
  return result.replace(/\u200B/g, '')
}

function syncFromModel() {
  if (!editorRef.value) return
  const html = valueToHtml(props.modelValue)
  if (editorRef.value.innerHTML !== html) {
    editorRef.value.innerHTML = html
  }
}

function emitValue() {
  if (!editorRef.value) return
  emit('update:modelValue', htmlToValue(editorRef.value))
}

function handleInput() {
  emitValue()
}

// Save current editor selection
function saveEditorRange() {
  const sel = window.getSelection()
  if (sel && sel.rangeCount > 0 && editorRef.value?.contains(sel.anchorNode)) {
    savedRange.value = sel.getRangeAt(0).cloneRange()
  }
}

// Restore saved editor selection
function restoreEditorRange() {
  if (!savedRange.value || !editorRef.value) return
  editorRef.value.focus()
  const sel = window.getSelection()
  if (sel) {
    sel.removeAllRanges()
    sel.addRange(savedRange.value)
  }
}

function showDropdown() {
  if (!editorRef.value) return
  const rect = editorRef.value.getBoundingClientRect()
  dropdownStyle.top = `${rect.bottom + 4}px`
  dropdownStyle.left = `${rect.left}px`
  searchText.value = ''
  highlightIndex.value = 0
  popoverVisible.value = true
  nextTick(() => searchInputRef.value?.focus())
}

function cancelPopover(restoreFocus = true) {
  popoverVisible.value = false
  if (restoreFocus) restoreEditorRange()
}

// Insert `/` into editor manually at cursor, then save range
function insertSlashAndSave() {
  const sel = window.getSelection()
  if (!sel || sel.rangeCount === 0 || !editorRef.value) return
  const range = sel.getRangeAt(0)
  const textNode = document.createTextNode('/')
  range.deleteContents()
  range.insertNode(textNode)
  range.setStartAfter(textNode)
  range.setEndAfter(textNode)
  sel.removeAllRanges()
  sel.addRange(range)
  saveEditorRange()
  emitValue()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === '/') {
    // Prevent default so `/` doesn't leak anywhere
    e.preventDefault()
    // Do NOT insert `/` — only show dropdown
    // Save current cursor position for pill insertion later
    saveEditorRange()
    showDropdown()
    return
  }
  // Block Enter in single-line mode
  if (!props.multiline && e.key === 'Enter') {
    e.preventDefault()
  }
}

function handleSearchKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    // Cancel: insert the `/` that was suppressed
    cancelPopover()
    nextTick(() => insertTextAtCursor('/'))
  } else if (e.key === '/') {
    // // escape: cancel popover, insert literal `/`
    e.preventDefault()
    cancelPopover()
    nextTick(() => insertTextAtCursor('/'))
  } else if (e.key === ' ' && searchText.value === '') {
    // / + space: cancel popover, insert `/ `
    e.preventDefault()
    cancelPopover()
    nextTick(() => insertTextAtCursor('/ '))
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    highlightIndex.value = Math.min(highlightIndex.value + 1, filteredVars.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    highlightIndex.value = Math.max(highlightIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (filteredVars.value[highlightIndex.value]) {
      insertVariable(filteredVars.value[highlightIndex.value])
    }
  }
}

// Insert plain text at saved cursor position in editor
function insertTextAtCursor(text: string) {
  if (!editorRef.value) return
  const sel = window.getSelection()
  if (sel && sel.rangeCount > 0) {
    const range = sel.getRangeAt(0)
    const textNode = document.createTextNode(text)
    range.collapse(false)
    range.insertNode(textNode)
    range.setStartAfter(textNode)
    range.setEndAfter(textNode)
    sel.removeAllRanges()
    sel.addRange(range)
  }
  emitValue()
}

function handleBlur(e: FocusEvent) {
  const related = e.relatedTarget as HTMLElement
  if (dropdownRef.value?.contains(related)) return
  if (searchInputRef.value === related) return
  isFocused.value = false
  setTimeout(() => {
    if (!isFocused.value) popoverVisible.value = false
  }, 150)
}

function insertVariable(v: Variable) {
  if (!editorRef.value) return
  popoverVisible.value = false

  // Restore cursor to editor (no `/` to remove — it was never inserted)
  editorRef.value.focus()
  if (savedRange.value) {
    const sel = window.getSelection()
    if (sel) {
      sel.removeAllRanges()
      sel.addRange(savedRange.value)
    }
  }

  // Create and insert pill
  const pill = document.createElement('span')
  pill.className = 'var-pill'
  pill.contentEditable = 'false'
  pill.dataset.var = v.name
  pill.innerHTML = `<svg viewBox="0 0 48 48" width="12" height="12" fill="currentColor" style="flex-shrink:0"><path d="M16 4l-8 22h10l-4 18 24-26H26l6-14z"/></svg>${v.name}`

  const selection = window.getSelection()
  if (selection && selection.rangeCount > 0) {
    const range = selection.getRangeAt(0)
    range.collapse(false)
    range.insertNode(pill)
    range.setStartAfter(pill)
    range.setEndAfter(pill)
    const spacer = document.createTextNode('\u200B')
    range.insertNode(spacer)
    range.setStartAfter(spacer)
    range.setEndAfter(spacer)
    selection.removeAllRanges()
    selection.addRange(range)
  } else {
    editorRef.value.appendChild(pill)
  }
  emitValue()
  editorRef.value.focus()
}

const fullscreenVisible = ref(false)
const fullscreenText = ref('')
const fullscreenTextareaRef = ref<HTMLTextAreaElement>()

function openFullscreenEdit() {
  fullscreenText.value = props.modelValue || ''
  fullscreenVisible.value = true
}

function handleFullscreenConfirm() {
  emit('update:modelValue', fullscreenText.value)
  nextTick(() => syncFromModel())
  return true
}

function handleEditorClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  const pill = target.closest('.var-collapse-pill') as HTMLElement | null
  if (!pill) return
  const id = pill.dataset.collapseId || ''
  const raw = collapsedMap.get(id) || ''
  // Strip outer quotes for editing
  collapseViewId.value = id
  collapseViewText.value = raw.startsWith('"') && raw.endsWith('"') ? raw.slice(1, -1).replace(/\\"/g, '"').replace(/\\\\/g, '\\') : raw
  collapseViewVisible.value = true
}

function handleCollapseViewConfirm() {
  const id = collapseViewId.value
  // Re-escape and wrap in quotes
  const escaped = collapseViewText.value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
  const quoted = `"${escaped}"`
  collapsedMap.set(id, quoted)
  // Re-emit full value
  emitValue()
  return true
}

async function copyCollapseText() {
  try { await navigator.clipboard.writeText(collapseViewText.value); Message.success('已复制') } catch { Message.error('复制失败') }
}

function handleContextMenu(e: MouseEvent) {
  e.preventDefault()
  saveEditorRange()
  contextMenuStyle.top = `${e.clientY}px`
  contextMenuStyle.left = `${e.clientX}px`
  contextMenuVisible.value = true
}

function insertFileAsBase64(type: 'image' | 'audio') {
  contextMenuVisible.value = false
  pendingFileType.value = type
  if (fileInputRef.value) {
    fileInputRef.value.accept = type === 'image' ? 'image/*' : 'audio/*'
    fileInputRef.value.value = ''
    fileInputRef.value.click()
  }
}

function handleFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const maxSize = 10 * 1024 * 1024 // 10MB
  if (file.size > maxSize) {
    Message.warning('文件大小不能超过 10MB')
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    const base64 = reader.result as string
    restoreEditorRange()
    nextTick(() => {
      insertTextAtCursor(base64)
      // After inserting, re-render to collapse the long base64 into a pill
      nextTick(() => {
        const raw = htmlToValue(editorRef.value!)
        emit('update:modelValue', raw)
        nextTick(() => syncFromModel())
      })
      Message.success(`${pendingFileType.value === 'image' ? '图片' : '音频'}已转为 Base64 插入`)
    })
  }
  reader.onerror = () => Message.error('文件读取失败')
  reader.readAsDataURL(file)
}

function closeContextMenu() {
  contextMenuVisible.value = false
}

function formatJson() {
  if (!editorRef.value) return
  const val = htmlToValue(editorRef.value)
  const placeholders: Record<string, { original: string; quoted: boolean }> = {}
  let idx = 0
  // Step 1: Replace "{{var}}" (quoted) with placeholder string
  let replaced = val.replace(/"(\{\{\w+\}\})"/g, (_, inner) => {
    const key = `__QVAR_${idx++}__`
    placeholders[key] = { original: inner, quoted: true }
    return `"${key}"`
  })
  // Step 2: Replace remaining {{var}} (unquoted) with quoted placeholder
  replaced = replaced.replace(/\{\{(\w+)\}\}/g, (match) => {
    const key = `__VAR_${idx++}__`
    placeholders[key] = { original: match, quoted: false }
    return `"${key}"`
  })
  try {
    const parsed = JSON.parse(replaced)
    let formatted = JSON.stringify(parsed, null, 2)
    // Restore: quoted vars get quotes back, unquoted vars stay bare
    for (const [key, info] of Object.entries(placeholders)) {
      if (info.quoted) {
        formatted = formatted.replace(`"${key}"`, `"${info.original}"`)
      } else {
        formatted = formatted.replace(`"${key}"`, info.original)
      }
    }
    emit('update:modelValue', formatted)
    nextTick(() => syncFromModel())
    Message.success('JSON 格式化成功')
  } catch {
    Message.warning('内容不是有效的 JSON')
  }
}

// Close dropdown on outside click
function handleOutsideClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!editorRef.value?.contains(target) && !dropdownRef.value?.contains(target)) {
    popoverVisible.value = false
  }
}

watch(() => props.modelValue, (val) => {
  if (!editorRef.value) return
  const current = htmlToValue(editorRef.value)
  if (current !== val) syncFromModel()
})

onMounted(() => {
  syncFromModel()
  document.addEventListener('mousedown', handleOutsideClick)
  document.addEventListener('click', closeContextMenu)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', handleOutsideClick)
  document.removeEventListener('click', closeContextMenu)
})
</script>

<style scoped>
.var-input-wrapper {
  position: relative;
  width: 100%;
  border: 1px solid var(--color-border-2, #e5e6eb);
  border-radius: 4px;
  background: #fff;
  transition: border-color 0.2s;
}
.var-input-wrapper.focused {
  border-color: var(--color-primary-light-3, #6aa1ff);
  box-shadow: 0 0 0 2px rgba(22, 93, 255, 0.1);
}
.var-input-editor {
  min-height: 32px;
  padding: 4px 8px;
  outline: none;
  font-size: 14px;
  line-height: 22px;
  color: var(--ops-text-primary, #1d2129);
  word-break: break-all;
  white-space: pre-wrap;
}
.var-input-multiline {
  min-height: 80px;
  max-height: 300px;
  overflow-y: auto;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 13px;
}
.var-input-editor:empty::before {
  content: attr(data-placeholder);
  color: var(--color-text-3, #c9cdd4);
  pointer-events: none;
}
.var-input-toolbar {
  display: flex;
  justify-content: flex-end;
  padding: 2px 4px;
  border-top: 1px solid var(--color-border-1, #f2f3f5);
  gap: 4px;
}
.var-fullscreen-textarea {
  width: 100%;
  min-height: 400px;
  padding: 12px;
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 13px;
  line-height: 1.6;
  border: 1px solid var(--color-border-2, #e5e6eb);
  border-radius: 4px;
  resize: vertical;
  outline: none;
  box-sizing: border-box;
}
.var-fullscreen-textarea:focus {
  border-color: var(--color-primary-light-3, #6aa1ff);
  box-shadow: 0 0 0 2px rgba(22, 93, 255, 0.1);
}
</style>

<style>
/* Global styles (unscoped for contenteditable + teleported dropdown) */
.var-pill {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  background: #e8f3ff;
  color: #165dff;
  border-radius: 4px;
  padding: 0 6px;
  font-size: 12px;
  line-height: 20px;
  user-select: all;
  vertical-align: middle;
  margin: 0 1px;
  font-weight: 500;
  white-space: nowrap;
}
.var-collapse-pill {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: #fff3e0;
  color: #d46b08;
  border: 1px solid #ffd591;
  border-radius: 4px;
  padding: 0 6px;
  font-size: 12px;
  line-height: 20px;
  user-select: all;
  vertical-align: middle;
  margin: 0 1px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s;
}
.var-collapse-pill:hover {
  background: #ffe7ba;
}
.var-dropdown-panel {
  position: fixed;
  z-index: 9999;
  background: #fff;
  border: 1px solid var(--color-border-2, #e5e6eb);
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  min-width: 240px;
  max-width: 340px;
  max-height: 280px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.var-dropdown-search {
  margin: 8px 8px 4px;
  padding: 4px 8px;
  border: 1px solid var(--color-border-2, #e5e6eb);
  border-radius: 4px;
  font-size: 13px;
  outline: none;
  width: calc(100% - 16px);
  box-sizing: border-box;
}
.var-dropdown-search:focus {
  border-color: #165dff;
}
.var-dropdown-list {
  overflow-y: auto;
  max-height: 220px;
  padding: 4px 0;
}
.var-option {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.15s;
}
.var-option:hover, .var-option.active {
  background: var(--color-fill-2, #f2f3f5);
}
.var-option-name {
  font-weight: 500;
  color: var(--ops-text-primary, #1d2129);
}
.var-option-desc {
  color: var(--ops-text-tertiary, #86909c);
  font-size: 12px;
  margin-left: auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 120px;
}
.var-empty {
  padding: 12px;
  text-align: center;
  color: var(--ops-text-tertiary, #86909c);
  font-size: 13px;
}
.var-context-menu {
  position: fixed;
  z-index: 9999;
  background: #fff;
  border: 1px solid var(--color-border-2, #e5e6eb);
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  min-width: 180px;
  padding: 4px 0;
}
.var-context-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  font-size: 13px;
  cursor: pointer;
  color: var(--ops-text-primary, #1d2129);
  transition: background 0.15s;
}
.var-context-item:hover {
  background: var(--color-fill-2, #f2f3f5);
}
</style>