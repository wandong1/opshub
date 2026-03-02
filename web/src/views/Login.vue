<template>
  <div class="login-container">
    <!-- 左侧：品牌展示区 -->
    <div class="brand-section">
      <div class="brand-content">
        <h1 class="brand-title">{{ systemStore.systemName || 'OpsHub' }}</h1>
        <p class="brand-desc" v-if="systemStore.systemDescription">{{ systemStore.systemDescription }}</p>
        <div class="brand-slogan">
          <span>高效</span>
          <span>安全</span>
          <span>便捷</span>
        </div>
        <p class="brand-subtitle">一键通达所有应用</p>
        <div class="brand-illustration">
          <svg viewBox="0 0 400 300" class="illustration-svg">
            <rect x="100" y="200" width="200" height="20" rx="4" fill="url(#blueGrad)" opacity="0.3"/>
            <rect x="120" y="180" width="160" height="20" rx="4" fill="url(#blueGrad)" opacity="0.5"/>
            <rect x="140" y="160" width="120" height="20" rx="4" fill="url(#blueGrad)" opacity="0.7"/>
            <rect x="180" y="100" width="40" height="60" rx="4" fill="url(#blueGrad)" opacity="0.85"/>
            <circle cx="200" cy="90" r="15" fill="url(#blueGrad)" opacity="0.8"/>
            <line x1="200" y1="75" x2="150" y2="50" stroke="url(#blueGrad)" stroke-width="2" opacity="0.6"/>
            <line x1="200" y1="75" x2="250" y2="50" stroke="url(#blueGrad)" stroke-width="2" opacity="0.6"/>
            <line x1="200" y1="75" x2="200" y2="40" stroke="url(#blueGrad)" stroke-width="2" opacity="0.6"/>
            <circle cx="150" cy="50" r="8" fill="url(#blueGrad)" opacity="0.9"/>
            <circle cx="250" cy="50" r="8" fill="url(#blueGrad)" opacity="0.9"/>
            <circle cx="200" cy="40" r="8" fill="url(#blueGrad)" opacity="0.9"/>
            <defs>
              <linearGradient id="blueGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#306fff;stop-opacity:1" />
                <stop offset="100%" style="stop-color:#6694ff;stop-opacity:1" />
              </linearGradient>
            </defs>
          </svg>
        </div>
      </div>
    </div>

    <!-- 右侧：登录表单区 -->
    <div class="login-section">
      <div class="login-wrapper">
        <div class="login-header">
          <h2>用户登录</h2>
          <div class="header-line"></div>
        </div>
        <a-form ref="formRef" :model="loginForm" :rules="rules" class="login-form" size="large" layout="vertical">
          <a-form-item field="username" hide-label>
            <a-input v-model="loginForm.username" placeholder="请输入用户名" allow-clear>
              <template #prefix><icon-user /></template>
            </a-input>
          </a-form-item>

          <a-form-item field="password" hide-label>
            <a-input-password v-model="loginForm.password" placeholder="请输入密码" @keyup.enter="handleLogin">
              <template #prefix><icon-lock /></template>
            </a-input-password>
          </a-form-item>

          <a-form-item field="captchaCode" hide-label v-if="captchaEnabled">
            <div class="captcha-wrapper">
              <a-input v-model="loginForm.captchaCode" placeholder="请输入验证码" @keyup.enter="handleLogin">
                <template #prefix><icon-safe /></template>
              </a-input>
              <div class="captcha-image" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="验证码" />
                <span v-else class="captcha-loading">加载中...</span>
              </div>
            </div>
          </a-form-item>

          <a-form-item hide-label>
            <a-checkbox v-model="loginForm.remember">记住登录名</a-checkbox>
          </a-form-item>

          <a-form-item hide-label>
            <a-button type="primary" long :loading="loading" class="login-button" @click="handleLogin">
              登录
            </a-button>
          </a-form-item>
        </a-form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconUser, IconLock, IconSafe } from '@arco-design/web-vue/es/icon'
import { useUserStore } from '@/stores/user'
import { useSystemStore } from '@/stores/system'
import request from '@/utils/request'
import { getPublicConfig } from '@/api/system'

const router = useRouter()
const userStore = useUserStore()
const systemStore = useSystemStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const captchaImage = ref('')
const captchaId = ref('')
const captchaEnabled = ref(true)

const loginForm = reactive({
  username: '',
  password: '',
  captchaCode: '',
  captchaId: '',
  remember: false,
})

const rules = computed(() => ({
  username: [{ required: true, message: '请输入用户名' }],
  password: [{ required: true, message: '请输入密码' }],
  captchaCode: captchaEnabled.value ? [{ required: true, message: '请输入验证码' }] : [],
}))

const loadPublicConfig = async () => {
  try {
    const res = await getPublicConfig()
    if (res) {
      captchaEnabled.value = res.enableCaptcha !== false
      systemStore.updateConfig({
        systemName: res.systemName,
        systemLogo: res.systemLogo,
        systemDescription: res.systemDescription,
      })
    }
  } catch {
    captchaEnabled.value = true
  }
}

const refreshCaptcha = async () => {
  if (!captchaEnabled.value) return
  try {
    captchaImage.value = ''
    const res: any = await request.get('/api/v1/captcha')
    captchaImage.value = res.image
    captchaId.value = res.captchaId
    loginForm.captchaId = res.captchaId
  } catch {
    Message.error('获取验证码失败')
  }
}

const handleLogin = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return

  loading.value = true
  try {
    await userStore.login({
      username: loginForm.username,
      password: loginForm.password,
      captchaId: loginForm.captchaId,
      captchaCode: loginForm.captchaCode,
    })
    if (loginForm.remember) {
      localStorage.setItem('rememberedUsername', loginForm.username)
    } else {
      localStorage.removeItem('rememberedUsername')
    }
    Message.success('登录成功')
    await router.push('/')
  } catch (error: any) {
    let errorMessage = '登录失败'
    if (error) {
      if (error.message && typeof error.message === 'string' && error.message !== '400') {
        errorMessage = error.message
      } else if (error.response?.data?.message) {
        errorMessage = error.response.data.message
      } else if (error.response?.data?.data) {
        errorMessage = error.response.data.data
      }
    }
    Message.error(errorMessage)
    refreshCaptcha()
    loginForm.captchaCode = ''
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  const rememberedUsername = localStorage.getItem('rememberedUsername')
  if (rememberedUsername) {
    loginForm.username = rememberedUsername
    loginForm.remember = true
  }
  await loadPublicConfig()
  if (captchaEnabled.value) refreshCaptcha()
})
</script>

<style scoped>
.login-container {
  display: flex;
  min-height: 100vh;
  background: #fff;
  overflow: hidden;
}

/* 左侧品牌区 */
.brand-section {
  flex: 0 0 58%;
  background: linear-gradient(135deg, #1d2129 0%, #2a3042 50%, #1d2129 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.brand-section::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 20% 30%, rgba(22, 93, 255, 0.08) 0%, transparent 50%),
    radial-gradient(circle at 80% 70%, rgba(22, 93, 255, 0.05) 0%, transparent 50%);
}

.brand-content {
  text-align: center;
  color: #fff;
  z-index: 1;
  padding: 60px;
}

.brand-title {
  font-size: 44px;
  font-weight: 700;
  margin-bottom: 12px;
  letter-spacing: 2px;
}

.brand-desc {
  font-size: 16px;
  color: rgba(255, 255, 255, 0.6);
  margin-bottom: 36px;
}

.brand-slogan {
  display: flex;
  justify-content: center;
  gap: 24px;
  margin-bottom: 24px;
  font-size: 18px;
  font-weight: 500;
}

.brand-slogan span {
  padding: 8px 20px;
  background: rgba(22, 93, 255, 0.15);
  border-radius: 8px;
  border: 1px solid rgba(22, 93, 255, 0.25);
}

.brand-subtitle {
  font-size: 15px;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 48px;
  letter-spacing: 1px;
}

.brand-illustration { max-width: 360px; margin: 0 auto; }
.illustration-svg { width: 100%; height: auto; filter: drop-shadow(0 10px 20px rgba(0,0,0,0.2)); }

/* 右侧登录区 */
.login-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  padding: 60px;
  position: relative;
}

.login-section::before {
  content: '';
  position: absolute;
  left: 0;
  top: 15%;
  width: 2px;
  height: 70%;
  background: linear-gradient(180deg, transparent 0%, var(--ops-primary, #165dff) 50%, transparent 100%);
  opacity: 0.3;
}

.login-wrapper {
  width: 100%;
  max-width: 400px;
}

.login-header { margin-bottom: 40px; }

.login-header h2 {
  font-size: 28px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 16px;
}

.header-line {
  width: 60px;
  height: 3px;
  background: var(--ops-primary, #165dff);
  border-radius: 2px;
}

.login-form { margin-top: 32px; }

.login-form :deep(.arco-form-item) { margin-bottom: 24px; }

.login-form :deep(.arco-input-wrapper) {
  height: 44px;
  border-radius: 8px;
}

.captcha-wrapper {
  display: flex;
  gap: 12px;
  width: 100%;
}

.captcha-wrapper :deep(.arco-input-wrapper) { flex: 1; }

.captcha-image {
  flex-shrink: 0;
  width: 130px;
  height: 44px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f7f8fa;
  transition: all 0.2s;
}

.captcha-image:hover {
  border-color: var(--ops-primary, #165dff);
}

.captcha-image img { width: 100%; height: 100%; object-fit: cover; }
.captcha-loading { font-size: 12px; color: var(--ops-text-tertiary, #86909c); }

.login-button {
  height: 44px;
  font-size: 15px;
  font-weight: 500;
  border-radius: 8px;
}

@media (max-width: 1200px) {
  .brand-section { flex: 0 0 50%; }
  .brand-title { font-size: 36px; }
  .brand-slogan { font-size: 16px; gap: 16px; }
}

@media (max-width: 768px) {
  .login-container { flex-direction: column; }
  .brand-section { flex: none; min-height: 40vh; }
  .brand-title { font-size: 28px; }
  .brand-slogan { font-size: 14px; gap: 12px; }
  .brand-slogan span { padding: 6px 14px; }
  .brand-illustration { max-width: 240px; }
  .login-section { padding: 30px; }
}
</style>
