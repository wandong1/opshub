<template>
  <div class="certificate-upload-container">
    <!-- 标签页 -->
    <a-tabs v-model:active-key="activeTab" class="certificate-tabs">
      <!-- 文件上传标签页 -->
      <a-tab-pane key="upload">
        <template #title>📁 文件上传</template>
        <div class="upload-content">
          <a-form :model="uploadForm" :rules="uploadRules" ref="uploadFormRef" auto-label-width>
            <a-form-item label="证书名称" field="name">
              <a-input v-model="uploadForm.name" placeholder="请输入证书名称" />
            </a-form-item>

            <a-form-item label="域名" field="domain">
              <a-input v-model="uploadForm.domain" placeholder="请输入域名，如：example.com" />
            </a-form-item>

            <a-form-item label="证书文件" field="certFile">
              <div class="file-upload-area">
                <input
                  type="file"
                  ref="certFileInput"
                  @change="handleCertFileSelect"
                  accept=".pem,.crt,.cer,.x509"
                  style="display: none"
                />
                <div class="upload-box" @click="() => certFileInput?.click()">
                  <icon-file class="upload-icon" />
                  <div class="upload-text">
                    <div class="upload-title">点击选择证书文件或拖拽上传</div>
                    <div class="upload-desc">支持 .pem .crt .cer .x509 格式</div>
                  </div>
                </div>
                <div v-if="uploadForm.certFile" class="file-info">
                  <span class="file-name">✓ {{ uploadForm.certFile.name }}</span>
                </div>
              </div>
            </a-form-item>

            <a-form-item label="私钥文件" field="keyFile">
              <div class="file-upload-area">
                <input
                  type="file"
                  ref="keyFileInput"
                  @change="handleKeyFileSelect"
                  accept=".key,.pem"
                  style="display: none"
                />
                <div class="upload-box" @click="() => keyFileInput?.click()">
                  <icon-lock class="upload-icon" />
                  <div class="upload-text">
                    <div class="upload-title">点击选择私钥文件或拖拽上传</div>
                    <div class="upload-desc">支持 .key .pem 格式（可选）</div>
                  </div>
                </div>
                <div v-if="uploadForm.keyFile" class="file-info">
                  <span class="file-name">✓ {{ uploadForm.keyFile.name }}</span>
                </div>
              </div>
            </a-form-item>

            <div class="form-actions">
              <a-button @click="handleUploadCancel">取消</a-button>
              <a-button type="primary" @click="handleUploadSubmit" :loading="uploading">
                验证并上传
              </a-button>
            </div>
          </a-form>
        </div>
      </a-tab-pane>

      <!-- 手动粘贴标签页 -->
      <a-tab-pane key="paste">
        <template #title>📝 手动粘贴</template>
        <div class="paste-content">
          <a-form :model="pasteForm" :rules="pasteRules" ref="pasteFormRef" auto-label-width>
            <a-form-item label="证书名称" field="name">
              <a-input v-model="pasteForm.name" placeholder="请输入证书名称" />
            </a-form-item>

            <a-form-item label="域名" field="domain">
              <a-input v-model="pasteForm.domain" placeholder="请输入域名，如：example.com" />
            </a-form-item>

            <a-form-item label="证书内容" field="certificate">
              <a-textarea
                v-model="pasteForm.certificate"
                :auto-size="{ minRows: 8 }"
                placeholder="请粘贴证书内容（PEM格式）&#10;-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
              />
            </a-form-item>

            <a-form-item label="私钥内容" field="privateKey">
              <a-textarea
                v-model="pasteForm.privateKey"
                :auto-size="{ minRows: 8 }"
                placeholder="请粘贴私钥内容（PEM格式，可选）&#10;-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
              />
            </a-form-item>
            <div class="form-actions">
              <a-button @click="handlePasteCancel">取消</a-button>
              <a-button type="primary" @click="handlePasteSubmit" :loading="pasting">
                验证并提交
              </a-button>
            </div>
          </a-form>
        </div>
      </a-tab-pane>
    </a-tabs>

    <!-- 证书信息预览对话框 -->
    <a-modal
      v-model:visible="previewDialogVisible"
      title="证书信息预览"
      :width="620"
      unmount-on-close
    >
      <div v-if="certInfo" class="cert-preview">
        <div class="cert-status-bar" :class="`status-bar-${certStatus}`">
          <span class="status-icon">{{ certStatus === 'valid' ? '&#10003;' : '!' }}</span>
          <span class="status-text">{{ certStatus === 'valid' ? '证书有效' : '证书即将过期' }}</span>
        </div>

        <div class="cert-info-sections">
          <div class="cert-info-section">
            <div class="section-title">基本信息</div>
            <div class="section-grid">
              <div class="cert-info-item">
                <div class="cert-label">证书名称</div>
                <div class="cert-value">{{ certInfo.name }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">域名</div>
                <div class="cert-value">{{ certInfo.domain }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">颁发者</div>
                <div class="cert-value cert-mono">{{ certInfo.issuer }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">主体</div>
                <div class="cert-value cert-mono">{{ certInfo.subject }}</div>
              </div>
            </div>
          </div>
          <div class="cert-info-section">
            <div class="section-title">有效期</div>
            <div class="section-grid">
              <div class="cert-info-item">
                <div class="cert-label">有效期起</div>
                <div class="cert-value">{{ certInfo.notBefore }}</div>
              </div>
              <div class="cert-info-item">
                <div class="cert-label">有效期至</div>
                <div class="cert-value" :class="getDaysRemainingClass(certInfo.daysRemaining)">
                  {{ certInfo.notAfter }}
                  <span class="days-remaining">(剩余 {{ certInfo.daysRemaining }} 天)</span>
                </div>
              </div>
            </div>
          </div>

          <div class="cert-info-section">
            <div class="section-title">安全信息</div>
            <div class="section-grid">
              <div class="cert-info-item full-width">
                <div class="cert-label">指纹(SHA256)</div>
                <div class="cert-value cert-mono fingerprint">{{ certInfo.fingerprint }}</div>
              </div>
              <div class="cert-info-item" v-if="certInfo.privateKey">
                <div class="cert-label">私钥</div>
                <div class="cert-value private-key-status">已包含</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <a-button @click="previewDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleConfirmUpload" :loading="confirming">
          确认上传
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Message } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconFile, IconLock } from '@arco-design/web-vue/es/icon'
interface CertInfo {
  name: string
  domain: string
  certificate: string
  privateKey: string
  subject: string
  issuer: string
  notBefore: string
  notAfter: string
  daysRemaining: number
  fingerprint: string
}

const emit = defineEmits<{
  submit: [data: CertInfo]
}>()

const activeTab = ref('upload')

// 上传表单
const uploadForm = reactive({
  name: '',
  domain: '',
  certFile: null as File | null,
  keyFile: null as File | null
})

const uploadRules = {
  name: [{ required: true, message: '请输入证书名称' }],
  domain: [{ required: true, message: '请输入域名' }],
  certFile: [{ required: true, message: '请选择证书文件' }]
}

const uploadFormRef = ref<FormInstance>()
const certFileInput = ref<HTMLInputElement | null>(null)
const keyFileInput = ref<HTMLInputElement | null>(null)

// 粘贴表单
const pasteForm = reactive({
  name: '',
  domain: '',
  certificate: '',
  privateKey: ''
})

const pasteRules = {
  name: [{ required: true, message: '请输入证书名称' }],
  domain: [{ required: true, message: '请输入域名' }],
  certificate: [{ required: true, message: '请粘贴证书内容' }]
}

const pasteFormRef = ref<FormInstance>()
// 预览相关
const previewDialogVisible = ref(false)
const certInfo = ref<CertInfo | null>(null)
const certStatus = ref('valid')
const uploading = ref(false)
const pasting = ref(false)
const confirming = ref(false)

// 处理证书文件选择
const handleCertFileSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.[0]) {
    uploadForm.certFile = input.files[0]
  }
}

// 处理私钥文件选择
const handleKeyFileSelect = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files?.[0]) {
    uploadForm.keyFile = input.files[0]
  }
}

// 获取剩余天数的样式
const getDaysRemainingClass = (days: number) => {
  if (days <= 0) return 'days-expired'
  if (days <= 30) return 'days-warning'
  return 'days-normal'
}

// 验证私钥格式
const isValidPrivateKey = (content: string) => {
  const trimmed = content.trim()
  return (trimmed.includes('BEGIN PRIVATE KEY') || trimmed.includes('BEGIN RSA PRIVATE KEY')) &&
         (trimmed.includes('END PRIVATE KEY') || trimmed.includes('END RSA PRIVATE KEY'))
}

// 读取文件
const readFile = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(new Error('读取文件失败'))
    reader.readAsText(file)
  })
}
// 解析证书信息
const parseCertInfo = async (certPem: string, keyPem: string = ''): Promise<CertInfo> => {
  try {
    // 这里使用前端解析，由于无法直接解析X.509证书，我们返回基本信息
    // 在实际应用中，应该由后端验证并返回详细信息
    const notAfterMatch = certPem.match(/notAfter=([^\n]+)/)
    const notBeforeMatch = certPem.match(/notBefore=([^\n]+)/)

    // 计算剩余天数（前端估算）
    const daysRemaining = 90 // 默认值，应由后端计算

    return {
      name: '',
      domain: '',
      certificate: certPem,
      privateKey: keyPem,
      subject: '证书信息将在上传后解析',
      issuer: '证书信息将在上传后解析',
      notBefore: '待解析',
      notAfter: '待解析',
      daysRemaining: daysRemaining,
      fingerprint: '待解析'
    }
  } catch (error: any) {
    throw new Error('解析证书失败：' + error.message)
  }
}

// 处理上传提交
const handleUploadSubmit = async () => {
  if (!uploadFormRef.value) return

  const errors = await uploadFormRef.value.validate()
  if (errors) return

  uploading.value = true
  try {
    // 直接读取文件内容并提交
    const certContent = await readFile(uploadForm.certFile!)
    let keyContent = ''

    if (uploadForm.keyFile) {
      keyContent = await readFile(uploadForm.keyFile)
    }

    // 验证证书格式
    if (!certContent.includes('BEGIN CERTIFICATE')) {
      Message.error('无效的证书格式')
      uploading.value = false
      return
    }
    if (keyContent && !isValidPrivateKey(keyContent)) {
      Message.error('无效的私钥格式')
      uploading.value = false
      return
    }

    // 解析证书信息
    certInfo.value = await parseCertInfo(certContent, keyContent)
    certStatus.value = certInfo.value.daysRemaining > 30 ? 'valid' : 'warning'
    previewDialogVisible.value = true
  } catch (error: any) {
    Message.error(error.message || '验证失败')
  } finally {
    uploading.value = false
  }
}

// 处理粘贴提交
const handlePasteSubmit = async () => {
  if (!pasteFormRef.value) return

  const errors = await pasteFormRef.value.validate()
  if (errors) return

  pasting.value = true
  try {
    // 验证证书格式
    if (!pasteForm.certificate.includes('BEGIN CERTIFICATE')) {
      Message.error('无效的证书格式')
      pasting.value = false
      return
    }

    if (pasteForm.privateKey && !isValidPrivateKey(pasteForm.privateKey)) {
      Message.error('无效的私钥格式')
      pasting.value = false
      return
    }

    // 解析证书信息
    certInfo.value = await parseCertInfo(pasteForm.certificate, pasteForm.privateKey)
    certStatus.value = certInfo.value.daysRemaining > 30 ? 'valid' : 'warning'
    previewDialogVisible.value = true
  } catch (error: any) {
    Message.error(error.message || '验证失败')
  } finally {
    pasting.value = false
  }
}
// 确认上传
const handleConfirmUpload = async () => {
  if (!certInfo.value) return

  confirming.value = true
  try {
    // 更新证书信息（从表单获取name和domain）
    if (activeTab.value === 'upload') {
      certInfo.value.name = uploadForm.name
      certInfo.value.domain = uploadForm.domain
    } else {
      certInfo.value.name = pasteForm.name
      certInfo.value.domain = pasteForm.domain
    }

    emit('submit', certInfo.value)
    previewDialogVisible.value = false
    resetForms()
  } catch (error: any) {
    Message.error(error.message || '上传失败')
  } finally {
    confirming.value = false
  }
}

// 取消上传
const handleUploadCancel = () => {
  uploadFormRef.value?.resetFields()
  uploadForm.certFile = null
  uploadForm.keyFile = null
  if (certFileInput.value) certFileInput.value.value = ''
  if (keyFileInput.value) keyFileInput.value.value = ''
}

// 取消粘贴
const handlePasteCancel = () => {
  pasteFormRef.value?.resetFields()
}

// 重置所有表单
const resetForms = () => {
  handleUploadCancel()
  handlePasteCancel()
  activeTab.value = 'upload'
}
</script>
<style scoped>
.certificate-upload-container {
  padding: 0;
}

/* 标签页 */
.certificate-tabs {
  margin: 0;
}

/* 上传内容 */
.upload-content,
.paste-content {
  padding: 20px 0;
}

.file-upload-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.upload-box {
  border: 2px dashed var(--ops-border-color, #e5e6eb);
  border-radius: 10px;
  padding: 32px 20px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  background-color: var(--ops-content-bg, #f7f8fa);
}

.upload-box:hover {
  border-color: var(--ops-primary, #165dff);
  background-color: #f0f5ff;
}

.upload-icon {
  font-size: 32px;
  color: var(--ops-primary, #165dff);
  margin-bottom: 12px;
}

.upload-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.upload-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--ops-text-primary, #1d2129);
}

.upload-desc {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background-color: #e8f3ff;
  border-radius: 6px;
  border-left: 3px solid var(--ops-primary, #165dff);
}

.file-name {
  font-size: 13px;
  color: var(--ops-primary, #165dff);
  font-weight: 500;
}

/* 表单操作 */
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--ops-border-color, #e5e6eb);
}

/* 证书预览 */
.cert-preview {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.cert-status-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 18px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 15px;
}
.status-bar-valid {
  background: linear-gradient(135deg, #e8ffea 0%, #d9f7be 100%);
  color: #00b42a;
  border: 1px solid #a9e2ab;
}

.status-bar-warning {
  background: linear-gradient(135deg, #fff7e8 0%, #ffe4ba 100%);
  color: #ff7d00;
  border: 1px solid #ffcf8b;
}

.status-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
}

.status-bar-valid .status-icon {
  background: #00b42a;
  color: #fff;
}

.status-bar-warning .status-icon {
  background: #ff7d00;
  color: #fff;
}

.cert-info-sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.cert-info-section {
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 10px;
  overflow: hidden;
}

.section-title {
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 600;
  color: var(--ops-text-tertiary, #86909c);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: var(--ops-content-bg, #f7f8fa);
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}
.section-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0;
}

.cert-info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 16px;
  border-bottom: 1px solid #f2f3f5;
  border-right: 1px solid #f2f3f5;
}

.cert-info-item:nth-child(2n) {
  border-right: none;
}

.cert-info-item:last-child,
.cert-info-item:nth-last-child(2):nth-child(odd) {
  border-bottom: none;
}

.cert-info-item.full-width {
  grid-column: span 2;
  border-right: none;
}

.cert-label {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  font-weight: 500;
}

.cert-value {
  font-size: 13px;
  color: var(--ops-text-primary, #1d2129);
  word-break: break-all;
  font-weight: 500;
}

.cert-mono {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: var(--ops-text-secondary, #4e5969);
  background-color: var(--ops-content-bg, #f7f8fa);
  padding: 6px 8px;
  border-radius: 6px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  max-height: 80px;
  overflow-y: auto;
}
.fingerprint {
  word-break: break-all;
  letter-spacing: 1px;
}

.private-key-status {
  color: #00b42a;
  font-weight: 600;
}

.days-remaining {
  display: inline-block;
  margin-left: 8px;
  font-weight: 600;
}

.days-normal {
  color: #00b42a;
}

.days-warning {
  color: #ff7d00;
}

.days-expired {
  color: #f53f3f;
}
</style>