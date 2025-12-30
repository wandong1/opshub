<template>
  <div class="profile-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>个人信息</span>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <!-- 基本信息标签页 -->
        <el-tab-pane label="基本信息" name="basic">
          <el-form
            :model="profileForm"
            :rules="profileRules"
            ref="profileFormRef"
            label-width="100px"
            class="profile-form"
          >
            <el-form-item label="头像">
              <div class="avatar-section">
                <el-avatar :size="100" :src="profileForm.avatar">
                  <el-icon><UserFilled /></el-icon>
                </el-avatar>
              </div>
            </el-form-item>

            <el-form-item label="用户名">
              <el-input v-model="profileForm.username" disabled />
            </el-form-item>

            <el-form-item label="真实姓名" prop="realName">
              <el-input v-model="profileForm.realName" placeholder="请输入真实姓名" />
            </el-form-item>

            <el-form-item label="邮箱" prop="email">
              <el-input v-model="profileForm.email" placeholder="请输入邮箱" />
            </el-form-item>

            <el-form-item label="手机号" prop="phone">
              <el-input v-model="profileForm.phone" placeholder="请输入手机号" />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="handleUpdateProfile" :loading="updateLoading">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 修改密码标签页 -->
        <el-tab-pane label="修改密码" name="password">
          <el-form
            :model="passwordForm"
            :rules="passwordRules"
            ref="passwordFormRef"
            label-width="100px"
            class="profile-form"
            style="max-width: 600px"
          >
            <el-form-item label="原密码" prop="oldPassword">
              <el-input
                v-model="passwordForm.oldPassword"
                type="password"
                show-password
                placeholder="请输入原密码"
              />
            </el-form-item>

            <el-form-item label="新密码" prop="newPassword">
              <el-input
                v-model="passwordForm.newPassword"
                type="password"
                show-password
                placeholder="请输入新密码（至少6位）"
              />
            </el-form-item>

            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input
                v-model="passwordForm.confirmPassword"
                type="password"
                show-password
                placeholder="请再次输入新密码"
              />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="handleUpdatePassword" :loading="passwordLoading">
                修改密码
              </el-button>
              <el-button @click="handleResetPassword">重置</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, FormInstance } from 'element-plus'
import { UserFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { updateUser, changePassword } from '@/api/user'

const userStore = useUserStore()
const activeTab = ref('basic')
const updateLoading = ref(false)
const passwordLoading = ref(false)

const profileFormRef = ref<FormInstance>()
const passwordFormRef = ref<FormInstance>()

const profileForm = reactive({
  id: 0,
  username: '',
  realName: '',
  email: '',
  phone: '',
  avatar: ''
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const profileRules = {
  realName: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  phone: [{ pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号', trigger: 'blur' }]
}

const validateConfirmPassword = (rule: any, value: any, callback: any) => {
  if (value !== passwordForm.newPassword) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 加载用户信息
const loadUserInfo = async () => {
  try {
    await userStore.getProfile()
    if (userStore.userInfo) {
      profileForm.id = userStore.userInfo.ID || userStore.userInfo.id
      profileForm.username = userStore.userInfo.username
      profileForm.realName = userStore.userInfo.realName || ''
      profileForm.email = userStore.userInfo.email || ''
      profileForm.phone = userStore.userInfo.phone || ''
      profileForm.avatar = userStore.userInfo.avatar || ''
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取用户信息失败')
  }
}

// 更新基本信息
const handleUpdateProfile = async () => {
  if (!profileFormRef.value) return

  await profileFormRef.value.validate(async (valid) => {
    if (valid) {
      updateLoading.value = true
      try {
        await updateUser(profileForm.id, {
          realName: profileForm.realName,
          email: profileForm.email,
          phone: profileForm.phone
        })
        ElMessage.success('保存成功')
        // 重新获取用户信息
        await userStore.getProfile()
      } catch (error) {
        console.error(error)
        ElMessage.error('保存失败')
      } finally {
        updateLoading.value = false
      }
    }
  })
}

// 修改密码
const handleUpdatePassword = async () => {
  if (!passwordFormRef.value) return

  await passwordFormRef.value.validate(async (valid) => {
    if (valid) {
      passwordLoading.value = true
      try {
        await changePassword(passwordForm.oldPassword, passwordForm.newPassword)
        ElMessage.success('密码修改成功，请重新登录')
        handleResetPassword()
        // 延迟后跳转到登录页
        setTimeout(() => {
          userStore.logout()
          window.location.href = '/login'
        }, 1500)
      } catch (error: any) {
        console.error(error)
        const errorMsg = error.response?.data?.message || error.message || '修改密码失败'
        ElMessage.error(errorMsg)
      } finally {
        passwordLoading.value = false
      }
    }
  })
}

// 重置密码表单
const handleResetPassword = () => {
  passwordFormRef.value?.resetFields()
}

onMounted(() => {
  loadUserInfo()
})
</script>

<style scoped>
.profile-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.profile-form {
  max-width: 800px;
  margin-top: 20px;
}

.avatar-section {
  display: flex;
  align-items: center;
}

.avatar-section :deep(.el-avatar) {
  background-color: #1890ff;
  border: 3px solid rgba(255, 255, 255, 0.2);
}

.avatar-section :deep(.el-icon) {
  font-size: 50px;
  color: #fff;
}
</style>
