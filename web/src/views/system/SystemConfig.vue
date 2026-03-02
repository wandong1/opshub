<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-settings />
        </div>
        <div>
          <div class="page-title">系统配置</div>
          <div class="page-desc">管理系统基础配置、安全设置</div>
        </div>
      </div>
      <a-button v-permission="'system-config:update'" type="primary" :loading="saving" @click="handleSave">
        <template #icon><icon-check /></template>
        保存配置
      </a-button>
    </div>

    <!-- 配置内容 -->
    <div class="config-content">
      <!-- 左侧导航 -->
      <a-card class="config-nav-card" :bordered="false">
        <div class="nav-header">配置分类</div>
        <div class="nav-list">
          <div
            v-for="(item, index) in navItems"
            :key="index"
            :class="['nav-item', { active: activeNav === index }]"
            @click="activeNav = index"
          >
            <component :is="item.icon" class="nav-icon" />
            <span>{{ item.label }}</span>
          </div>
        </div>
      </a-card>

      <!-- 右侧配置表单 -->
      <a-card class="config-form-card" :bordered="false">
        <!-- 基础配置 -->
        <div v-show="activeNav === 0" class="config-section">
          <div class="section-header">
            <icon-home class="section-icon" />
            <span>基础配置</span>
          </div>
          <a-form :model="config" layout="horizontal" :label-col-props="{ span: 5 }" :wrapper-col-props="{ span: 16 }">
            <a-form-item label="系统名称">
              <a-input v-model="config.systemName" placeholder="请输入系统名称" />
            </a-form-item>
            <a-form-item label="系统Logo">
              <div class="logo-upload-container">
                <div class="logo-preview" v-if="config.systemLogo">
                  <img :src="config.systemLogo" alt="Logo预览" />
                  <div class="logo-actions">
                    <a-button type="primary" status="danger" size="small" @click="removeLogo">
                      <template #icon><icon-delete /></template>
                      删除
                    </a-button>
                  </div>
                </div>
                <a-upload
                  v-else
                  :auto-upload="false"
                  :show-file-list="false"
                  accept=".png,.jpg,.jpeg,.ico,.svg"
                  @change="handleLogoChange"
                >
                  <template #upload-button>
                    <div class="upload-trigger">
                      <icon-plus class="upload-icon" />
                      <span class="upload-text">上传Logo</span>
                    </div>
                  </template>
                </a-upload>
                <div class="upload-tip">支持 png/jpg/jpeg/ico/svg 格式，大小不超过 2MB</div>
              </div>
            </a-form-item>
            <a-form-item label="系统描述">
              <a-textarea v-model="config.systemDescription" placeholder="请输入系统描述" :auto-size="{ minRows: 3, maxRows: 6 }" />
            </a-form-item>
          </a-form>
        </div>

        <!-- 安全配置 -->
        <div v-show="activeNav === 1" class="config-section">
          <div class="section-header">
            <icon-lock class="section-icon" />
            <span>安全配置</span>
          </div>
          <a-form :model="config" layout="horizontal" :label-col-props="{ span: 5 }" :wrapper-col-props="{ span: 16 }">
            <a-form-item label="密码最小长度">
              <a-space>
                <a-input-number v-model="config.passwordMinLength" :min="6" :max="20" style="width: 160px;" />
                <span class="form-tip">建议设置 8 位以上</span>
              </a-space>
            </a-form-item>
            <a-form-item label="Session超时">
              <a-space>
                <a-input-number v-model="config.sessionTimeout" :min="300" :step="300" style="width: 160px;" />
                <span class="form-tip">单位：秒</span>
              </a-space>
            </a-form-item>
            <a-form-item label="开启验证码">
              <a-switch v-model="config.enableCaptcha" checked-text="开启" unchecked-text="关闭" />
            </a-form-item>
            <a-form-item label="最大登录失败">
              <a-space>
                <a-input-number v-model="config.maxLoginAttempts" :min="3" :max="10" style="width: 160px;" />
                <span class="form-tip">超过次数将锁定账户</span>
              </a-space>
            </a-form-item>
            <a-form-item label="账户锁定时间">
              <a-space>
                <a-input-number v-model="config.lockoutDuration" :min="60" :step="60" style="width: 160px;" />
                <span class="form-tip">单位：秒</span>
              </a-space>
            </a-form-item>
          </a-form>
        </div>
      </a-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconSettings,
  IconCheck,
  IconHome,
  IconLock,
  IconPlus,
  IconDelete,
} from '@arco-design/web-vue/es/icon'
import {
  getAllConfig,
  saveBasicConfig,
  saveSecurityConfig,
  uploadLogo
} from '@/api/system'
import { useSystemStore } from '@/stores/system'

const systemStore = useSystemStore()
const saving = ref(false)
const activeNav = ref(0)

const navItems = [
  { label: '基础配置', icon: IconHome },
  { label: '安全配置', icon: IconLock }
]

const config = reactive({
  systemName: 'OpsHub',
  systemLogo: '',
  systemDescription: '运维管理平台',
  passwordMinLength: 8,
  sessionTimeout: 3600,
  enableCaptcha: true,
  maxLoginAttempts: 5,
  lockoutDuration: 300
})

const loadConfig = async () => {
  try {
    const res = await getAllConfig()
    if (res) {
      if (res.basic) {
        config.systemName = res.basic.systemName || 'OpsHub'
        config.systemLogo = res.basic.systemLogo || ''
        config.systemDescription = res.basic.systemDescription || '运维管理平台'
      }
      if (res.security) {
        config.passwordMinLength = res.security.passwordMinLength || 8
        config.sessionTimeout = res.security.sessionTimeout || 3600
        config.enableCaptcha = res.security.enableCaptcha !== false
        config.maxLoginAttempts = res.security.maxLoginAttempts || 5
        config.lockoutDuration = res.security.lockoutDuration || 300
      }
    }
  } catch (error) {
    console.error('加载配置失败', error)
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    await saveBasicConfig({
      systemName: config.systemName,
      systemLogo: config.systemLogo,
      systemDescription: config.systemDescription
    })

    await saveSecurityConfig({
      passwordMinLength: config.passwordMinLength,
      sessionTimeout: config.sessionTimeout,
      enableCaptcha: config.enableCaptcha,
      maxLoginAttempts: config.maxLoginAttempts,
      lockoutDuration: config.lockoutDuration
    })

    systemStore.updateConfig({
      systemName: config.systemName,
      systemLogo: config.systemLogo,
      systemDescription: config.systemDescription
    })

    Message.success('配置保存成功')
  } catch {
    Message.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleLogoChange = async (fileList: any[], fileItem: any) => {
  const file = fileItem.file
  if (!file) return

  const validTypes = ['image/png', 'image/jpeg', 'image/jpg', 'image/x-icon', 'image/svg+xml']
  const isValidType = validTypes.includes(file.type) || file.name.endsWith('.ico') || file.name.endsWith('.svg')
  const isLt2M = file.size / 1024 / 1024 < 2

  if (!isValidType) {
    Message.error('只能上传 png/jpg/jpeg/ico/svg 格式的图片!')
    return
  }
  if (!isLt2M) {
    Message.error('图片大小不能超过 2MB!')
    return
  }

  try {
    const res = await uploadLogo(file)
    if (res && res.url) {
      config.systemLogo = res.url
      Message.success('Logo上传成功')
    }
  } catch {
    Message.error('Logo上传失败')
  }
}

const removeLogo = () => {
  config.systemLogo = ''
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped lang="scss">
.page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-card {
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
  padding: 20px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--ops-primary, #165dff) 0%, #4080ff 100%);
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

/* 配置内容 */
.config-content {
  display: flex;
  gap: 16px;
  min-height: calc(100vh - 220px);
}

.config-nav-card {
  width: 200px;
  min-width: 200px;
  border-radius: var(--ops-border-radius-md, 8px);
  :deep(.arco-card-body) {
    padding: 0;
  }
}

.nav-header {
  padding: 16px 20px;
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}

.nav-list {
  padding: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  margin-bottom: 4px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: var(--ops-text-secondary, #4e5969);
  font-size: 14px;
}

.nav-item:hover {
  background: var(--color-fill-2, #f2f3f5);
  color: var(--ops-text-primary, #1d2129);
}

.nav-item.active {
  background: var(--ops-primary, #165dff);
  color: #fff;
  font-weight: 500;
}

.nav-item.active .nav-icon {
  color: #fff;
}

.nav-icon {
  font-size: 18px;
  color: var(--ops-text-tertiary, #86909c);
  transition: color 0.2s ease;
}

.nav-item:hover .nav-icon {
  color: var(--ops-text-secondary, #4e5969);
}

/* 右侧表单 */
.config-form-card {
  flex: 1;
  border-radius: var(--ops-border-radius-md, 8px);
}

.config-section {
  max-width: 700px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid var(--ops-border-color, #e5e6eb);
  font-size: 16px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}

.section-icon {
  font-size: 22px;
  color: var(--ops-primary, #165dff);
}

.form-tip {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

/* Logo上传 */
.logo-upload-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.logo-preview {
  position: relative;
  width: 120px;
  height: 120px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-fill-1, #f7f8fa);
}

.logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.logo-preview .logo-actions {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 8px;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
}

.logo-preview:hover .logo-actions {
  opacity: 1;
}

.upload-trigger {
  width: 120px;
  height: 120px;
  border: 2px dashed var(--ops-border-color, #e5e6eb);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--color-fill-1, #f7f8fa);
}

.upload-trigger:hover {
  border-color: var(--ops-primary, #165dff);
  background: #fff;
}

.upload-icon {
  font-size: 32px;
  color: var(--ops-text-tertiary, #86909c);
  margin-bottom: 8px;
}

.upload-text {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

.upload-tip {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 响应式 */
@media (max-width: 768px) {
  .config-content {
    flex-direction: column;
  }
  .config-nav-card {
    width: 100%;
    min-width: auto;
  }
  .nav-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .nav-item {
    flex: 1;
    min-width: 120px;
    justify-content: center;
    margin-bottom: 0;
  }
}
</style>
