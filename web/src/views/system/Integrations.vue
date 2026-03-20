<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-link />
        </div>
        <div>
          <div class="page-title">集成管理</div>
          <div class="page-desc">管理第三方系统集成配置，支持 Grafana 等监控平台嵌入</div>
        </div>
      </div>
    </div>

    <!-- 集成列表 Tabs -->
    <a-card :bordered="false" class="content-card">
      <a-tabs v-model:active-key="activeTab" type="card">
        <!-- Grafana 集成 -->
        <a-tab-pane key="grafana">
          <template #title>
            <span style="display: inline-flex; align-items: center; gap: 6px;">
              <icon-bar-chart />
              Grafana
            </span>
          </template>

          <div class="integration-panel">
            <div class="integration-header">
              <div class="integration-title">Grafana 集成配置</div>
              <div class="integration-desc">配置 Grafana 监控平台的访问地址，嵌入监控大屏页面</div>
            </div>

            <a-form
              :model="grafanaForm"
              layout="vertical"
              style="max-width: 640px; margin-top: 24px;"
            >
              <a-form-item label="启用 Grafana 集成">
                <a-switch
                  v-model="grafanaForm.enabled"
                  checked-text="已启用"
                  unchecked-text="已禁用"
                />
                <div class="form-tip">关闭后监控大屏页面将无法访问 Grafana</div>
              </a-form-item>

              <a-form-item label="Grafana 访问地址" :disabled="!grafanaForm.enabled">
                <a-input
                  v-model="grafanaForm.url"
                  placeholder="例如：http://grafana_mon:3000/grafana_2syulinm/"
                  :disabled="!grafanaForm.enabled"
                />
                <div class="form-tip">后端代理将转发请求到此地址，支持内网地址</div>
              </a-form-item>

              <a-form-item label="Grafana Sub-path" :disabled="!grafanaForm.enabled">
                <a-input
                  v-model="grafanaForm.subpath"
                  placeholder="例如：/grafana_2syulinm/"
                  :disabled="!grafanaForm.enabled"
                />
                <div class="form-tip">Grafana 配置的 root_url sub_path，以 / 开头和结尾</div>
              </a-form-item>

              <a-form-item>
                <a-space>
                  <a-button
                    v-permission="'integrations:save'"
                    type="primary"
                    :loading="saving"
                    @click="handleSaveGrafana"
                  >
                    <template #icon><icon-save /></template>
                    保存配置
                  </a-button>
                  <a-button @click="handleLoadConfig">
                    <template #icon><icon-refresh /></template>
                    重置
                  </a-button>
                </a-space>
              </a-form-item>
            </a-form>

            <!-- 连接测试区域 -->
            <a-divider />
            <div class="test-section">
              <div class="test-title">代理地址预览</div>
              <div class="test-url">
                前端访问地址：<a-tag color="arcoblue">/api/v1/grafana/</a-tag>
              </div>
              <div class="test-tip">前端 iframe 通过此地址访问 Grafana，后端自动代理转发到上方配置的地址</div>
            </div>
          </div>
        </a-tab-pane>

        <!-- 预留扩展 Tab 位置，新增集成只需在此处添加 a-tab-pane -->
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getIntegrationConfig, saveIntegrationConfig } from '@/api/system'

const activeTab = ref('grafana')
const saving = ref(false)

const grafanaForm = reactive({
  enabled: true,
  url: 'http://grafana_mon:3000/grafana_2syulinm/',
  subpath: '/grafana_2syulinm/'
})

const handleLoadConfig = async () => {
  try {
    const res = await getIntegrationConfig() as any
    const data = res?.data || res
    if (data?.grafana) {
      grafanaForm.enabled = data.grafana.enabled ?? true
      grafanaForm.url = data.grafana.url || 'http://grafana_mon:3000/grafana_2syulinm/'
      grafanaForm.subpath = data.grafana.subpath || '/grafana_2syulinm/'
    }
  } catch (err) {
    console.error('加载集成配置失败', err)
  }
}

const handleSaveGrafana = async () => {
  if (!grafanaForm.url) {
    Message.warning('请填写 Grafana 访问地址')
    return
  }
  saving.value = true
  try {
    await saveIntegrationConfig({
      grafana: {
        enabled: grafanaForm.enabled,
        url: grafanaForm.url,
        subpath: grafanaForm.subpath
      }
    })
    Message.success('保存成功')
  } catch (err) {
    Message.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  handleLoadConfig()
})
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: linear-gradient(135deg, #165dff 0%, #4080ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.4;
}

.page-desc {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 2px;
}

.content-card {
  border-radius: 8px;
}

.integration-panel {
  padding: 8px 0;
}

.integration-header {
  margin-bottom: 4px;
}

.integration-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}

.integration-desc {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 4px;
}

.form-tip {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 4px;
  line-height: 1.5;
}

.test-section {
  padding: 4px 0;
}

.test-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 12px;
}

.test-url {
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.test-tip {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}
</style>
