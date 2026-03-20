<template>
  <div class="grafana-dashboard">
    <div v-if="loading" class="loading-mask">
      <a-spin :size="36" tip="正在加载监控大屏..." />
    </div>

    <div v-if="!enabled && !loading" class="disabled-mask">
      <a-result status="warning" title="Grafana 集成未启用">
        <template #subtitle>
          请前往<a-link href="/system/integrations">系统管理 → 集成管理</a-link>开启 Grafana 集成并配置访问地址
        </template>
      </a-result>
    </div>

    <iframe
      v-if="enabled && !loading"
      :src="iframeUrl"
      frameborder="0"
      class="grafana-iframe"
      allowfullscreen
      @load="handleIframeLoad"
      @error="handleIframeError"
    />

    <!-- 右上角工具栏 -->
    <div v-if="enabled && !loading" class="toolbar">
      <a-tooltip content="在新窗口打开">
        <a-button type="text" class="toolbar-btn" @click="openInNewWindow">
          <template #icon><icon-launch /></template>
        </a-button>
      </a-tooltip>
      <a-tooltip content="刷新">
        <a-button type="text" class="toolbar-btn" @click="refreshIframe">
          <template #icon><icon-refresh /></template>
        </a-button>
      </a-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getIntegrationConfig } from '@/api/system'

const loading = ref(true)
const enabled = ref(true)
const iframeUrl = ref('')

const loadConfig = async () => {
  loading.value = true
  try {
    const res = await getIntegrationConfig() as any
    const data = res?.data || res
    if (data?.grafana) {
      enabled.value = data.grafana.enabled ?? true
      // 通过后端代理访问 Grafana
      // 代理路由路径与 Grafana sub_path 完全一致
      // 例如 subpath=/grafana_2syulinm/ → iframe src=/grafana_2syulinm/
      iframeUrl.value = data.grafana.subpath || '/'
    }
  } catch (err) {
    console.error('加载 Grafana 配置失败', err)
  } finally {
    loading.value = false
  }
}

const handleIframeLoad = () => {
  // iframe 加载完成
}

const handleIframeError = () => {
  console.error('Grafana 加载失败，请检查配置')
}

const openInNewWindow = () => {
  window.open(iframeUrl.value, '_blank')
}

const refreshIframe = () => {
  const iframe = document.querySelector('.grafana-iframe') as HTMLIFrameElement
  if (iframe) {
    iframe.src = iframe.src
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.grafana-dashboard {
  position: relative;
  width: 100%;
  height: calc(100vh - 60px - 52px);
  min-height: 400px;
  background: #f7f8fa;
  overflow: hidden;
}

.grafana-iframe {
  width: 100%;
  height: 100%;
  border: none;
  display: block;
}

.loading-mask,
.disabled-mask {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f7f8fa;
  z-index: 10;
}

.toolbar {
  position: absolute;
  top: 12px;
  right: 16px;
  display: flex;
  gap: 4px;
  z-index: 20;
  background: rgba(255, 255, 255, 0.9);
  border-radius: 6px;
  padding: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.toolbar-btn {
  color: var(--ops-text-secondary, #4e5969);
}

.toolbar-btn:hover {
  color: var(--ops-primary, #165dff);
}
</style>
