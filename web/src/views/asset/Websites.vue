<template>
  <div class="websites-page-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-link /></div>
        <div>
          <h2 class="page-title">Web站点管理</h2>
          <p class="page-subtitle">统一管理内外部Web站点，支持Agent代理访问内网站点</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          新增站点
        </a-button>
      </div>
    </div>

    <!-- 主内容区域：站点列表 -->
    <div class="main-content">
      <!-- 站点列表 -->
      <div class="content-panel">
        <!-- 搜索和筛选 -->
        <div class="search-bar">
          <a-form layout="inline" :model="searchForm">
            <a-form-item label="关键词">
              <a-input
                v-model="searchForm.keyword"
                placeholder="站点名称/URL"
                allow-clear
                style="width: 200px;"
              >
                <template #prefix><icon-search /></template>
              </a-input>
            </a-form-item>
            <a-form-item label="站点类型">
              <a-select
                v-model="searchForm.type"
                placeholder="全部"
                allow-clear
                style="width: 120px;"
              >
                <a-option value="external">外部站点</a-option>
                <a-option value="internal">内部站点</a-option>
              </a-select>
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="handleSearch">
                <template #icon><icon-search /></template>
                搜索
              </a-button>
              <a-button @click="handleReset" style="margin-left: 8px;">
                <template #icon><icon-refresh /></template>
                重置
              </a-button>
            </a-form-item>
          </a-form>
        </div>

        <!-- 站点卡片网格 -->
        <div v-if="tableLoading" class="loading-container">
          <a-spin tip="加载中..." />
        </div>
        <div v-else-if="tableData.length === 0" class="empty-container">
          <a-empty description="暂无站点数据" />
        </div>
        <div v-else class="websites-grid">
          <div v-for="site in tableData" :key="site.id" class="website-card">
            <!-- 卡片头部 -->
            <div class="card-header">
              <div class="site-info">
                <div class="site-icon-large">
                  <icon-link v-if="!site.icon" />
                  <span v-else-if="site.icon.length <= 2" class="icon-emoji">{{ site.icon }}</span>
                  <img v-else :src="site.icon" alt="icon" />
                </div>
                <div class="site-details">
                  <h3 class="site-name">{{ site.name }}</h3>
                  <div class="site-tags">
                    <a-tag :color="site.type === 'external' ? 'blue' : 'orange'" size="small">
                      {{ site.typeText }}
                    </a-tag>
                    <a-tag :color="site.status === 1 ? 'green' : 'red'" size="small">
                      {{ site.statusText }}
                    </a-tag>
                    <a-tag v-if="site.type === 'internal'" :color="site.agentOnline ? 'green' : 'red'" size="small">
                      {{ site.agentOnline ? 'Agent在线' : 'Agent离线' }}
                    </a-tag>
                  </div>
                </div>
              </div>
            </div>

            <!-- 卡片内容 -->
            <div class="card-body">
              <div class="info-row">
                <span class="info-label">访问地址</span>
                <a-tooltip :content="site.url">
                  <span class="info-value url-text">{{ site.url }}</span>
                </a-tooltip>
              </div>
              <div v-if="site.description" class="info-row">
                <span class="info-label">备注</span>
                <span class="info-value">{{ site.description }}</span>
              </div>
              <div v-if="site.groupNames && site.groupNames.length > 0" class="info-row">
                <span class="info-label">业务分组</span>
                <div class="info-value">
                  <a-tag v-for="(name, idx) in site.groupNames" :key="idx" size="small">{{ name }}</a-tag>
                </div>
              </div>
              <div v-if="site.agentHostNames && site.agentHostNames.length > 0" class="info-row">
                <span class="info-label">Agent主机</span>
                <div class="info-value">
                  <a-tag v-for="(name, idx) in site.agentHostNames" :key="idx" size="small" color="arcoblue">{{ name }}</a-tag>
                </div>
              </div>
            </div>

            <!-- 卡片底部操作 -->
            <div class="card-footer">
              <a-button type="primary" size="small" @click="handleAccess(site)">
                <template #icon><icon-export /></template>
                访问站点
              </a-button>
              <a-dropdown v-if="site.accessUser" trigger="hover">
                <a-button size="small">
                  <template #icon><icon-copy /></template>
                  复制凭据
                </a-button>
                <template #content>
                  <a-doption @click="handleCopyUrl(site)">
                    <template #icon><icon-link /></template>
                    复制URL
                  </a-doption>
                  <a-doption @click="handleCopyUsername(site)">
                    <template #icon><icon-user /></template>
                    复制账号
                  </a-doption>
                  <a-doption @click="handleCopyPassword(site)">
                    <template #icon><icon-lock /></template>
                    复制密码
                  </a-doption>
                </template>
              </a-dropdown>
              <a-button size="small" @click="handleEdit(site)">
                <template #icon><icon-edit /></template>
                编辑
              </a-button>
              <a-popconfirm
                content="确定要删除该站点吗？"
                @ok="handleDelete(site.id)"
              >
                <a-button size="small" status="danger">
                  <template #icon><icon-delete /></template>
                  删除
                </a-button>
              </a-popconfirm>
            </div>
          </div>
        </div>

        <!-- 分页 -->
        <div v-if="tableData.length > 0" class="pagination-container">
          <a-pagination
            v-model:current="pagination.current"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :show-total="true"
            :show-page-size="true"
            :page-size-options="pagination.pageSizeOptions"
            @change="handlePageChange"
            @page-size-change="handlePageSizeChange"
          />
        </div>
      </div>
    </div>

    <!-- 新增/编辑站点弹窗 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="dialogTitle"
      width="700px"
      @before-ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form :model="formData" :rules="formRules" ref="formRef" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="站点名称" field="name">
              <a-input v-model="formData.name" placeholder="请输入站点名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="站点类型" field="type">
              <a-select v-model="formData.type" placeholder="请选择站点类型" @change="handleTypeChange">
                <a-option value="external">外部站点</a-option>
                <a-option value="internal">内部站点</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="站点URL" field="url">
          <a-input v-model="formData.url" placeholder="https://example.com" />
        </a-form-item>

        <a-form-item label="站点图标">
          <a-select
            v-model="formData.icon"
            placeholder="请选择图标"
            allow-clear
            allow-search
          >
            <a-option v-for="icon in presetIcons" :key="icon.value" :value="icon.value">
              <div style="display: flex; align-items: center; gap: 8px;">
                <span :style="{ fontSize: '18px' }">{{ icon.emoji }}</span>
                <span>{{ icon.label }}</span>
              </div>
            </a-option>
          </a-select>
          <div style="margin-top: 4px; font-size: 12px; color: var(--ops-text-tertiary);">
            也可以输入自定义图标 URL
          </div>
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="访问用户名">
              <a-input v-model="formData.accessUser" placeholder="可选" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="访问密码">
              <a-input-password v-model="formData.accessPassword" placeholder="可选" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="加密凭据">
          <a-input v-model="formData.credential" placeholder="Token/API密钥等（可选）" />
        </a-form-item>

        <a-form-item label="业务分组">
          <a-select
            v-model="formData.groupIds"
            placeholder="请选择业务分组"
            multiple
            allow-search
            @change="handleGroupChange"
          >
            <a-option v-for="group in flatGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-option>
          </a-select>
        </a-form-item>

        <a-form-item v-if="formData.type === 'internal'" label="绑定Agent主机" field="agentHostIds">
          <a-select
            v-model="formData.agentHostIds"
            placeholder="请选择Agent主机（至少1个）"
            multiple
            allow-search
          >
            <a-option v-for="host in agentHosts" :key="host.id" :value="host.id">
              {{ host.name }} ({{ host.ip }})
              <a-tag v-if="host.agentStatus === 'online'" color="green" size="small" style="margin-left: 8px;">在线</a-tag>
              <a-tag v-else color="red" size="small" style="margin-left: 8px;">离线</a-tag>
            </a-option>
          </a-select>
        </a-form-item>

        <a-form-item label="备注">
          <a-textarea v-model="formData.description" placeholder="请输入备注" :rows="3" />
        </a-form-item>

        <a-form-item label="状态">
          <a-radio-group v-model="formData.status">
            <a-radio :value="1">启用</a-radio>
            <a-radio :value="0">禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item>
          <a-checkbox v-model="formData.secureCopyUrl">
            启用安全复制URL
          </a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconLink,
  IconApps,
  IconFolder,
  IconSearch,
  IconPlus,
  IconEdit,
  IconDelete,
  IconExport,
  IconRefresh,
  IconCopy,
  IconUser,
  IconLock
} from '@arco-design/web-vue/es/icon'
import {
  getWebsiteList,
  getWebsite,
  createWebsite,
  updateWebsite,
  deleteWebsite,
  accessWebsite,
  type Website,
  type WebsiteRequest
} from '@/api/website'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'

// 表格相关
const tableData = ref<Website[]>([])
const tableLoading = ref(false)
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showTotal: true,
  showPageSize: true,
  pageSizeOptions: [10, 20, 50, 100]
})

// 搜索表单
const searchForm = reactive({
  keyword: '',
  type: ''
})

// 弹窗相关
const dialogVisible = ref(false)
const dialogTitle = ref('新增站点')
const formRef = ref()
const formData = reactive<WebsiteRequest>({
  name: '',
  url: '',
  icon: '',
  type: 'external',
  credential: '',
  secureCopyUrl: false,
  accessUser: '',
  accessPassword: '',
  description: '',
  status: 1,
  groupIds: [],
  agentHostIds: []
})

const formRules = {
  name: [{ required: true, message: '请输入站点名称' }],
  url: [{ required: true, message: '请输入站点URL' }],
  type: [{ required: true, message: '请选择站点类型' }],
  agentHostIds: [
    {
      validator: (value: any, cb: any) => {
        if (formData.type === 'internal' && (!value || value.length === 0)) {
          cb('内部站点必须绑定至少1台Agent主机')
        } else {
          cb()
        }
      }
    }
  ]
}

// Agent主机列表
const agentHosts = ref<any[]>([])
const allAgentHosts = ref<any[]>([]) // 保存所有Agent主机

// 预置图标列表
const presetIcons = [
  { value: '🌐', emoji: '🌐', label: '地球' },
  { value: '🏢', emoji: '🏢', label: '办公楼' },
  { value: '💼', emoji: '💼', label: '公文包' },
  { value: '📊', emoji: '📊', label: '图表' },
  { value: '📈', emoji: '📈', label: '上升趋势' },
  { value: '🔧', emoji: '🔧', label: '工具' },
  { value: '⚙️', emoji: '⚙️', label: '设置' },
  { value: '🖥️', emoji: '🖥️', label: '电脑' },
  { value: '📱', emoji: '📱', label: '手机' },
  { value: '🔒', emoji: '🔒', label: '锁' },
  { value: '🔑', emoji: '🔑', label: '钥匙' },
  { value: '📦', emoji: '📦', label: '包裹' },
  { value: '🚀', emoji: '🚀', label: '火箭' },
  { value: '⚡', emoji: '⚡', label: '闪电' },
  { value: '🎯', emoji: '🎯', label: '靶心' },
  { value: '📡', emoji: '📡', label: '卫星天线' },
  { value: '🌟', emoji: '🌟', label: '星星' },
  { value: '💡', emoji: '💡', label: '灯泡' }
]

// 扁平化分组列表（用于表单选择）
const groupTree = ref<any[]>([])
const flatGroups = computed(() => {
  const flatten = (nodes: any[]): any[] => {
    let result: any[] = []
    nodes.forEach(node => {
      result.push(node)
      if (node.children && node.children.length > 0) {
        result = result.concat(flatten(node.children))
      }
    })
    return result
  }
  return flatten(groupTree.value)
})

// 加载分组树（仅用于表单选择）
const loadGroupTree = async () => {
  try {
    const res = await getGroupTree()
    groupTree.value = res || []
  } catch (error: any) {
    console.error('加载分组树失败:', error)
    Message.error('加载业务分组失败: ' + error.message)
  }
}

// 加载Agent主机列表
const loadAgentHosts = async () => {
  try {
    const res = await getHostList({ page: 1, pageSize: 1000 })
    // 修复：统一使用 data.list 获取数据
    const list = res.data?.list || res.list || []
    allAgentHosts.value = list.filter((h: any) => h.agentId)
    // 初始加载时显示所有Agent主机
    agentHosts.value = allAgentHosts.value
  } catch (error: any) {
    Message.error('加载Agent主机列表失败: ' + error.message)
  }
}

// 根据业务分组过滤Agent主机
const filterAgentHostsByGroups = () => {
  if (!formData.groupIds || formData.groupIds.length === 0) {
    // 没有选择分组，显示所有Agent主机
    agentHosts.value = allAgentHosts.value
  } else {
    // 根据分组过滤主机：显示属于选中分组的主机
    agentHosts.value = allAgentHosts.value.filter((host: any) => {
      // 如果主机没有分组信息，也显示（允许选择未分组的主机）
      if (!host.groupIds || host.groupIds.length === 0) {
        return true
      }
      // 检查主机是否属于选中的任一分组
      return host.groupIds.some((gid: number) => formData.groupIds.includes(gid))
    })
  }
}

// 监听业务分组变化
const handleGroupChange = () => {
  filterAgentHostsByGroups()
  // 如果当前选中的Agent主机不在过滤后的列表中，清空选择
  if (formData.agentHostIds && formData.agentHostIds.length > 0) {
    const validHostIds = agentHosts.value.map((h: any) => h.id)
    formData.agentHostIds = formData.agentHostIds.filter((id: number) => validHostIds.includes(id))
  }
}

// 加载站点列表
const loadWebsites = async () => {
  tableLoading.value = true
  try {
    const res = await getWebsiteList({
      page: pagination.current,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword,
      type: searchForm.type,
      groupIds: []
    })
    tableData.value = res.list || []
    pagination.total = res.total || 0
  } catch (error: any) {
    console.error('加载站点列表失败:', error)
    Message.error('加载站点列表失败: ' + error.message)
  } finally {
    tableLoading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.current = 1
  loadWebsites()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.type = ''
  pagination.current = 1
  loadWebsites()
}

// 分页
const handlePageChange = (page: number) => {
  pagination.current = page
  loadWebsites()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.current = 1
  loadWebsites()
}

// 新增
const handleAdd = () => {
  dialogTitle.value = '新增站点'
  Object.assign(formData, {
    id: undefined,  // 清除 id 字段，确保是新增而不是更新
    name: '',
    url: '',
    icon: '',
    type: 'external',
    credential: '',
    secureCopyUrl: false,
    accessUser: '',
    accessPassword: '',
    description: '',
    status: 1,
    groupIds: [],
    agentHostIds: []
  })
  // 重置 Agent 主机列表为全部
  agentHosts.value = allAgentHosts.value
  dialogVisible.value = true
}

// 编辑
const handleEdit = (record: Website) => {
  dialogTitle.value = '编辑站点'
  Object.assign(formData, {
    id: record.id,
    name: record.name,
    url: record.url,
    icon: record.icon,
    type: record.type,
    credential: '',
    secureCopyUrl: record.secureCopyUrl,
    accessUser: record.accessUser,
    accessPassword: '',
    description: record.description,
    status: record.status,
    groupIds: record.groupIds || [],
    agentHostIds: record.agentHostIds || []
  })
  // 根据已选择的分组过滤 Agent 主机
  filterAgentHostsByGroups()
  dialogVisible.value = true
}

// 类型变化
const handleTypeChange = () => {
  if (formData.type === 'external') {
    formData.agentHostIds = []
  }
}

// 提交
const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    if (formData.id) {
      await updateWebsite(formData.id, formData)
      Message.success('更新成功')
    } else {
      await createWebsite(formData)
      Message.success('创建成功')
      // 新增后重置到第一页
      pagination.current = 1
    }
    dialogVisible.value = false
    await loadWebsites()
    return true
  } catch (error: any) {
    Message.error(error.message || '操作失败')
    return false
  }
}

// 取消
const handleCancel = () => {
  formRef.value?.resetFields()
}

// 删除
const handleDelete = async (id: number) => {
  try {
    await deleteWebsite(id)
    Message.success('删除成功')
    loadWebsites()
  } catch (error: any) {
    Message.error('删除失败: ' + error.message)
  }
}

// 复制URL
const handleCopyUrl = async (record: Website) => {
  try {
    await navigator.clipboard.writeText(record.url)
    Message.success('URL已复制到剪贴板')
  } catch (error: any) {
    Message.error('复制失败: ' + error.message)
  }
}

// 复制账号
const handleCopyUsername = async (record: Website) => {
  try {
    if (!record.accessUser) {
      Message.warning('该站点未设置访问账号')
      return
    }
    await navigator.clipboard.writeText(record.accessUser)
    Message.success('账号已复制到剪贴板')
  } catch (error: any) {
    Message.error('复制失败: ' + error.message)
  }
}

// 复制密码
const handleCopyPassword = async (record: Website) => {
  try {
    // 获取完整的站点信息（包括密码）
    const res = await getWebsite(record.id)
    const website = res

    console.log('获取到的站点信息:', website)
    console.log('访问密码:', website.accessPassword)

    if (!website.accessPassword || website.accessPassword === '') {
      Message.warning('该站点未设置访问密码')
      return
    }

    await navigator.clipboard.writeText(website.accessPassword)
    Message.success('密码已复制到剪贴板')
  } catch (error: any) {
    console.error('复制密码失败:', error)
    Message.error('复制失败: ' + error.message)
  }
}

// 访问站点
const handleAccess = async (record: Website) => {
  try {
    const res = await accessWebsite(record.id)
    // axios 拦截器已经返回了 data 字段，直接使用 res
    if (res.type === 'external') {
      window.open(res.url, '_blank')
    } else {
      // 内部站点通过代理访问
      if (res.proxyUrl) {
        // 获取当前的 token
        const token = localStorage.getItem('token')
        if (!token) {
          Message.error('未登录，无法访问内部站点')
          return
        }
        // 构建带 token 的代理 URL
        const proxyUrl = `${res.proxyUrl}?token=${token}`
        window.open(proxyUrl, '_blank')
      } else {
        Message.error('无法获取代理访问地址')
      }
    }
  } catch (error: any) {
    Message.error('访问失败: ' + error.message)
  }
}

onMounted(() => {
  loadGroupTree()
  loadAgentHosts()
  loadWebsites()
})
</script>

<style scoped lang="scss">
.websites-page-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--ops-content-bg);
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  color: #fff;
  font-size: 24px;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--ops-text-primary);
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 14px;
  color: var(--ops-text-secondary);
}

.header-actions {
  display: flex;
  gap: 12px;
}

.main-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.content-panel {
  flex: 1;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.search-bar {
  padding: 16px;
  border-bottom: 1px solid var(--ops-border-color);
}

.loading-container,
.empty-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
}

.websites-grid {
  flex: 1;
  overflow: auto;
  padding: 20px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 20px;
  align-content: start;
}

.website-card {
  background: #fff;
  border: 1px solid var(--ops-border-color);
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s ease;
  display: flex;
  flex-direction: column;

  &:hover {
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }
}

.card-header {
  padding: 20px;
  background: linear-gradient(135deg, #f5f7fa 0%, #ffffff 100%);
  border-bottom: 1px solid var(--ops-border-color);
}

.site-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.site-icon-large {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  font-size: 28px;
  flex-shrink: 0;

  .icon-emoji {
    font-size: 32px;
  }

  img {
    width: 40px;
    height: 40px;
    object-fit: contain;
  }
}

.site-details {
  flex: 1;
  min-width: 0;
}

.site-name {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.card-body {
  padding: 20px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-row {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.info-label {
  font-size: 13px;
  color: var(--ops-text-secondary);
  min-width: 70px;
  flex-shrink: 0;
}

.info-value {
  font-size: 13px;
  color: var(--ops-text-primary);
  flex: 1;
  word-break: break-all;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.url-text {
  color: var(--ops-primary);
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &:hover {
    text-decoration: underline;
  }
}

.card-footer {
  padding: 16px 20px;
  background: var(--ops-content-bg);
  border-top: 1px solid var(--ops-border-color);
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.pagination-container {
  padding: 16px 20px;
  border-top: 1px solid var(--ops-border-color);
  display: flex;
  justify-content: center;
}
</style>
