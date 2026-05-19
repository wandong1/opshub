<template>
  <div class="login-container">
    <!-- 背景动效层 -->
    <div class="bg-animation">
      <div class="grid-lines"></div>
      <div class="floating-particles">
        <div class="particle" v-for="i in 20" :key="i" :style="getParticleStyle(i)"></div>
      </div>
    </div>

    <!-- 左侧：品牌展示区 -->
    <div class="brand-section">
      <div class="brand-content">
        <!-- Logo 和标题 -->
        <div class="brand-header">
          <div class="logo-wrapper">
            <img v-if="systemStore.systemLogo" :src="systemStore.systemLogo" alt="Logo" class="brand-logo" />
            <div v-else class="brand-logo-placeholder">
              <svg viewBox="0 0 48 48" fill="none">
                <rect width="48" height="48" rx="12" fill="url(#logoGrad)"/>
                <path d="M24 14L32 20V28L24 34L16 28V20L24 14Z" stroke="white" stroke-width="2" fill="none"/>
                <circle cx="24" cy="24" r="3" fill="white"/>
                <defs>
                  <linearGradient id="logoGrad" x1="0" y1="0" x2="48" y2="48">
                    <stop offset="0%" stop-color="#4080ff"/>
                    <stop offset="100%" stop-color="#165dff"/>
                  </linearGradient>
                </defs>
              </svg>
            </div>
          </div>
          <h1 class="brand-title">{{ systemStore.systemName || 'SreHub' }}</h1>
          <p class="brand-tagline">智能巡检管理平台</p>
        </div>

        <!-- 核心价值主张 -->
        <div class="value-propositions">
          <div class="value-item" v-for="(item, index) in valueItems" :key="index" :style="{ animationDelay: `${index * 0.1}s` }">
            <div class="value-icon">
              <component :is="item.icon" />
            </div>
            <div class="value-text">
              <h3>{{ item.title }}</h3>
              <p>{{ item.desc }}</p>
            </div>
          </div>
        </div>

        <!-- 主题标语 -->
        <div class="slogan-section">
          <div class="slogan-main">务实高效风，企业落地运维场景</div>
          <div class="slogan-features">
            <span class="feature-tag">巡检无死角</span>
            <span class="feature-tag">管控零距离</span>
            <span class="feature-tag">智能提质增效</span>
          </div>
          <div class="slogan-sub">前置风险预警 · 全程智能巡检 · 稳保业务畅通</div>
        </div>

        <!-- 版本信息 -->
        <div class="version-info">
          <p class="version-text">{{ systemVersion }}</p>
        </div>
      </div>
    </div>

    <!-- 右侧：登录表单区 -->
    <div class="login-section">
      <div class="login-wrapper">
        <div class="login-header">
          <h2>欢迎登录</h2>
          <p class="login-subtitle">精准排查隐患，减负提质巡检</p>
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
              <span v-if="!loading">立即登录</span>
              <span v-else>登录中...</span>
            </a-button>
          </a-form-item>
        </a-form>

        <!-- 底部提示 -->
        <div class="login-footer">
          <p class="footer-tip">Copyright © 2026 ChinaTelecom All Rights Reserved.</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import {
  IconUser, IconLock, IconSafe,
  IconThunderbolt, IconEye, IconSafe as IconShield
} from '@arco-design/web-vue/es/icon'
import { useUserStore } from '@/stores/user'
import { useSystemStore } from '@/stores/system'
import request from '@/utils/request'
import { getPublicConfig } from '@/api/system'
import { getRsaPublicKey } from '@/api/auth'
import JSEncrypt from 'jsencrypt'

const router = useRouter()
const userStore = useUserStore()
const systemStore = useSystemStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const captchaImage = ref('')
const captchaId = ref('')
const captchaEnabled = ref(true)
const rsaPublicKey = ref('')
const systemVersion = ref('v1.0.0')

const loginForm = reactive({
  username: '',
  password: '',
  captchaCode: '',
  captchaId: '',
  remember: false,
})

// 核心价值主张
const valueItems = [
  {
    icon: IconThunderbolt,
    title: '智能巡检',
    desc: '自动化巡检，实时监控系统健康状态'
  },
  {
    icon: IconEye,
    title: '精准排查',
    desc: '快速定位问题，精准排查系统隐患'
  },
  {
    icon: IconShield,
    title: '风险预警',
    desc: '前置风险预警，保障业务稳定运行'
  }
]

// 粒子动画样式
const getParticleStyle = (index: number) => {
  const size = Math.random() * 4 + 2
  const duration = Math.random() * 20 + 10
  const delay = Math.random() * 5
  return {
    width: `${size}px`,
    height: `${size}px`,
    left: `${Math.random() * 100}%`,
    top: `${Math.random() * 100}%`,
    animationDuration: `${duration}s`,
    animationDelay: `${delay}s`
  }
}

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
      systemVersion.value = res.version || 'v1.0.0'
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

const loadRsaPublicKey = async () => {
  try {
    const res = await getRsaPublicKey()
    rsaPublicKey.value = res.publicKey
  } catch (error) {
    console.error('获取 RSA 公钥失败:', error)
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

const encryptPassword = (password: string): string | null => {
  if (!rsaPublicKey.value) {
    console.error('RSA 公钥未加载')
    return null
  }

  try {
    const encrypt = new JSEncrypt()
    encrypt.setPublicKey(rsaPublicKey.value)
    const encrypted = encrypt.encrypt(password)
    return encrypted || null
  } catch (error) {
    console.error('密码加密失败:', error)
    return null
  }
}

const handleLogin = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return

  loading.value = true
  try {
    const encryptedPassword = encryptPassword(loginForm.password)
    if (!encryptedPassword) {
      Message.error('密码加密失败，请刷新页面重试')
      loading.value = false
      return
    }

    await userStore.login({
      username: loginForm.username,
      encryptedPassword: encryptedPassword,
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
  await loadRsaPublicKey()
  if (captchaEnabled.value) refreshCaptcha()
})
</script>

<style scoped>
.login-container {
  display: flex;
  min-height: 100vh;
  background: #0a0e27;
  overflow: hidden;
  position: relative;
}

/* ===== 背景动效层 ===== */
.bg-animation {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}

.grid-lines {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(64, 128, 255, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(64, 128, 255, 0.03) 1px, transparent 1px);
  background-size: 50px 50px;
  animation: gridMove 20s linear infinite;
}

@keyframes gridMove {
  0% { transform: translate(0, 0); }
  100% { transform: translate(50px, 50px); }
}

.floating-particles {
  position: absolute;
  inset: 0;
}

.particle {
  position: absolute;
  background: radial-gradient(circle, rgba(64, 128, 255, 0.8) 0%, transparent 70%);
  border-radius: 50%;
  animation: float linear infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0) translateX(0); opacity: 0; }
  10% { opacity: 1; }
  90% { opacity: 1; }
  100% { transform: translateY(-100vh) translateX(50px); opacity: 0; }
}

/* ===== 左侧品牌区 ===== */
.brand-section {
  flex: 0 0 55%;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 1;
  padding: 60px;
}

.brand-content {
  max-width: 600px;
  width: 100%;
}

/* Logo 和标题 */
.brand-header {
  text-align: center;
  margin-bottom: 60px;
}

.logo-wrapper {
  display: inline-block;
  margin-bottom: 24px;
  animation: logoFloat 3s ease-in-out infinite;
}

@keyframes logoFloat {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

.brand-logo {
  width: 120px;
  height: 120px;
  object-fit: contain;
  filter: drop-shadow(0 4px 12px rgba(64, 128, 255, 0.3));
}

.brand-logo-placeholder svg {
  width: 80px;
  height: 80px;
  filter: drop-shadow(0 4px 12px rgba(64, 128, 255, 0.3));
}

.brand-title {
  font-size: 48px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 12px;
  letter-spacing: 2px;
  background: linear-gradient(135deg, #ffffff 0%, #a0c4ff 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-tagline {
  font-size: 18px;
  color: rgba(255, 255, 255, 0.6);
  letter-spacing: 4px;
}

/* 核心价值主张 */
.value-propositions {
  display: flex;
  flex-direction: column;
  gap: 24px;
  margin-bottom: 60px;
}

.value-item {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  padding: 24px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(64, 128, 255, 0.2);
  border-radius: 12px;
  backdrop-filter: blur(10px);
  transition: all 0.3s ease;
  animation: slideInLeft 0.6s ease-out backwards;
}

@keyframes slideInLeft {
  from {
    opacity: 0;
    transform: translateX(-30px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.value-item:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(64, 128, 255, 0.4);
  transform: translateX(8px);
}

.value-icon {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(64, 128, 255, 0.2) 0%, rgba(22, 93, 255, 0.1) 100%);
  border-radius: 12px;
  font-size: 28px;
  color: #4080ff;
}

.value-text h3 {
  font-size: 18px;
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 8px;
}

.value-text p {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.6);
  line-height: 1.6;
}

/* 主题标语 */
.slogan-section {
  text-align: center;
  padding: 32px;
  background: linear-gradient(135deg, rgba(64, 128, 255, 0.1) 0%, rgba(22, 93, 255, 0.05) 100%);
  border: 1px solid rgba(64, 128, 255, 0.2);
  border-radius: 16px;
  backdrop-filter: blur(10px);
}

.slogan-main {
  font-size: 20px;
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 20px;
  line-height: 1.6;
}

.slogan-features {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.feature-tag {
  padding: 8px 20px;
  background: rgba(64, 128, 255, 0.15);
  border: 1px solid rgba(64, 128, 255, 0.3);
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
  color: #4080ff;
  white-space: nowrap;
}

.slogan-sub {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.5);
  letter-spacing: 2px;
}

/* 版本信息 */
.version-info {
  margin-top: 40px;
  text-align: center;
  padding-top: 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.version-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 8px;
  font-family: 'Courier New', monospace;
  letter-spacing: 1px;
}

.copyright-text {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  letter-spacing: 1px;
}

/* ===== 右侧登录区 ===== */
.login-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(20px);
  padding: 60px;
  position: relative;
  z-index: 1;
}

.login-wrapper {
  width: 100%;
  max-width: 420px;
  animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.login-header {
  margin-bottom: 40px;
}

.login-header h2 {
  font-size: 32px;
  font-weight: 700;
  color: #1d2129;
  margin-bottom: 8px;
}

.login-subtitle {
  font-size: 14px;
  color: #86909c;
  letter-spacing: 1px;
}

.login-form :deep(.arco-form-item) {
  margin-bottom: 20px;
}

.login-form :deep(.arco-input-wrapper),
.login-form :deep(.arco-input-password) {
  height: 48px;
  border-radius: 8px;
  border: 1px solid #e5e6eb;
  transition: all 0.3s;
}

.login-form :deep(.arco-input-wrapper:hover),
.login-form :deep(.arco-input-wrapper:focus-within) {
  border-color: #4080ff;
  box-shadow: 0 0 0 3px rgba(64, 128, 255, 0.1);
}

.captcha-wrapper {
  display: flex;
  gap: 12px;
  width: 100%;
}

.captcha-wrapper :deep(.arco-input-wrapper) {
  flex: 1;
}

.captcha-image {
  flex-shrink: 0;
  width: 130px;
  height: 48px;
  border: 1px solid #e5e6eb;
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
  border-color: #4080ff;
  transform: scale(1.02);
}

.captcha-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.captcha-loading {
  font-size: 12px;
  color: #86909c;
}

.login-button {
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 8px;
  background: linear-gradient(135deg, #4080ff 0%, #165dff 100%);
  border: none;
  box-shadow: 0 4px 12px rgba(64, 128, 255, 0.3);
  transition: all 0.3s;
}

.login-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(64, 128, 255, 0.4);
}

.login-button:active {
  transform: translateY(0);
}

.login-footer {
  margin-top: 24px;
  text-align: center;
}

.footer-tip {
  font-size: 13px;
  color: #86909c;
  line-height: 1.6;
}

/* ===== 响应式 ===== */
@media (max-width: 1200px) {
  .brand-section {
    flex: 0 0 50%;
    padding: 40px;
  }

  .brand-title {
    font-size: 40px;
  }

  .value-propositions {
    gap: 16px;
  }

  .value-item {
    padding: 20px;
  }
}

@media (max-width: 768px) {
  .login-container {
    flex-direction: column;
  }

  .brand-section {
    flex: none;
    min-height: 50vh;
    padding: 30px 20px;
  }

  .brand-title {
    font-size: 32px;
  }

  .brand-tagline {
    font-size: 14px;
  }

  .value-propositions {
    margin-bottom: 40px;
  }

  .value-item {
    padding: 16px;
  }

  .value-icon {
    width: 48px;
    height: 48px;
    font-size: 24px;
  }

  .value-text h3 {
    font-size: 16px;
  }

  .slogan-main {
    font-size: 16px;
  }

  .feature-tag {
    font-size: 12px;
    padding: 6px 16px;
  }

  .login-section {
    padding: 30px 20px;
  }

  .login-header h2 {
    font-size: 24px;
  }
}
</style>
