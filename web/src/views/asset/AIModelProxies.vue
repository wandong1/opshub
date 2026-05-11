<template>
  <div class="ai-model-proxies-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h2>🤖 AI模型代理管理</h2>
      </div>
      <div class="header-right">
        <a-button type="primary" @click="handleCreate">
          <template #icon><icon-plus /></template>
          新建代理
        </a-button>
      </div>
    </div>

    <!-- 帮助提示 -->
    <a-alert type="info" closable style="margin-bottom: 16px">
      <template #icon><icon-info-circle /></template>
      <div>
        <strong>什么是AI模型代理？</strong>
        <p style="margin: 8px 0 0 0">AI模型代理允许您通过Agent主机安全地访问内网的AI模型服务（如Ollama、OpenAI等），无需暴露内网服务到公网。</p>
        <a-link @click="showUsageGuide">查看使用指南</a-link>
      </div>
    </a-alert>

    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <a-space size="medium">
        <a-input
          v-model="searchForm.keyword"
          placeholder="搜索代理名称或目标URL"
          allow-clear
          style="width: 300px"
          @press-enter="handleSearch"
        >
          <template #prefix><icon-search /></template>
        </a-input>

        <a-select
          v-model="searchForm.modelType"
          placeholder="模型类型"
          allow-clear
          style="width: 150px"
          @change="handleSearch"
        >
          <a-option value="ollama">Ollama</a-option>
          <a-option value="openai">OpenAI</a-option>
          <a-option value="custom">Custom</a-option>
        </a-select>

        <a-select
          v-model="searchForm.status"
          placeholder="状态"
          allow-clear
          style="width: 120px"
          @change="handleSearch"
        >
          <a-option :value="1">启用</a-option>
          <a-option :value="0">禁用</a-option>
        </a-select>

        <a-select
          v-model="searchForm.groupId"
          placeholder="业务分组"
          allow-clear
          allow-search
          style="width: 200px"
          @change="handleSearch"
        >
          <a-option v-for="group in flatGroups" :key="group.id" :value="group.id">
            {{ group.name }}
          </a-option>
        </a-select>

        <a-button @click="handleSearch">
          <template #icon><icon-search /></template>
          搜索
        </a-button>

        <a-button @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
      </a-space>
    </div>

    <!-- 代理卡片列表 -->
    <a-spin :loading="loading" style="width: 100%">
      <div v-if="proxyList.length > 0" class="proxy-grid">
        <div
          v-for="proxy in proxyList"
          :key="proxy.id"
          class="proxy-card"
        >
          <!-- 卡片头部 -->
          <div class="card-header">
            <div class="header-left">
              <span class="model-icon">{{ getModelTypeIcon(proxy.modelType) }}</span>
              <span class="proxy-name">{{ proxy.name }}</span>
            </div>
            <div class="header-right">
              <a-tag :color="proxy.status === 1 ? 'green' : 'red'" size="small">
                {{ proxy.statusText }}
              </a-tag>
            </div>
          </div>

          <!-- 卡片内容 -->
          <div class="card-content">
            <div class="info-row">
              <a-tag :color="getModelTypeColor(proxy.modelType)" size="small">
                {{ proxy.modelTypeText }}
              </a-tag>
              <a-tag v-if="proxy.agentOnline" color="green" size="small">
                <template #icon><icon-check-circle-fill /></template>
                Agent在线
              </a-tag>
              <a-tag v-else color="red" size="small">
                <template #icon><icon-close-circle-fill /></template>
                Agent离线
              </a-tag>
            </div>

            <div class="info-row">
              <span class="label">目标:</span>
              <a-tooltip :content="proxy.targetUrl">
                <span class="value truncate">{{ proxy.targetUrl }}</span>
              </a-tooltip>
            </div>

            <div class="info-row">
              <span class="label">超时:</span>
              <span class="value">{{ proxy.timeout }}秒</span>
            </div>

            <div v-if="proxy.groupName" class="info-row">
              <span class="label">分组:</span>
              <span class="value">{{ proxy.groupName }}</span>
            </div>

            <div v-if="proxy.agentHostNames.length > 0" class="info-row">
              <span class="label">Agent:</span>
              <span class="value">{{ proxy.agentHostNames.join(', ') }}</span>
            </div>

            <div v-if="proxy.description" class="info-row description">
              <span class="value">{{ proxy.description }}</span>
            </div>
          </div>

          <!-- 卡片底部操作 -->
          <div class="card-footer">
            <a-button type="primary" size="small" @click="handleCopyProxyUrl(proxy)">
              <template #icon><icon-copy /></template>
              复制URL
            </a-button>

            <a-dropdown>
              <a-button size="small">
                更多
                <icon-down />
              </a-button>
              <template #content>
                <a-doption @click="handleTestConnection(proxy)">
                  <template #icon><icon-thunderbolt /></template>
                  测试连接
                </a-doption>
                <a-doption @click="handleEdit(proxy)">
                  <template #icon><icon-edit /></template>
                  编辑
                </a-doption>
                <a-doption @click="handleRegenerateToken(proxy)">
                  <template #icon><icon-refresh /></template>
                  重新生成Token
                </a-doption>
                <a-doption @click="handleDelete(proxy)">
                  <template #icon><icon-delete /></template>
                  <span style="color: var(--color-danger-6)">删除</span>
                </a-doption>
              </template>
            </a-dropdown>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <a-empty v-else description="暂无AI模型代理" style="margin: 60px 0">
        <a-button type="primary" @click="handleCreate">
          <template #icon><icon-plus /></template>
          新建代理
        </a-button>
      </a-empty>
    </a-spin>

    <!-- 分页 -->
    <div v-if="total > 0" class="pagination">
      <a-pagination
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :total="total"
        show-total
        show-jumper
        show-page-size
        @change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      />
    </div>

    <!-- 创建/编辑弹窗 -->
    <a-modal
      v-model:visible="modalVisible"
      :title="modalTitle"
      width="700px"
      @cancel="handleModalCancel"
      @before-ok="handleModalOk"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        layout="vertical"
      >
        <!-- 基本信息 -->
        <div class="form-section">
          <div class="section-title">基本信息</div>
          <a-form-item label="代理名称" field="name" required>
            <a-input v-model="formData.name" placeholder="请输入代理名称" />
          </a-form-item>

          <a-form-item label="描述" field="description">
            <a-textarea
              v-model="formData.description"
              placeholder="请输入描述（可选）"
              :max-length="200"
              show-word-limit
            />
          </a-form-item>

          <a-form-item label="模型类型" field="modelType" required>
            <a-radio-group v-model="formData.modelType">
              <a-radio value="ollama">
                <span class="model-icon">🦙</span> Ollama
              </a-radio>
              <a-radio value="openai">
                <span class="model-icon">🤖</span> OpenAI
              </a-radio>
              <a-radio value="custom">
                <span class="model-icon">⚙️</span> Custom
              </a-radio>
            </a-radio-group>
          </a-form-item>
        </div>

        <!-- 目标配置 -->
        <div class="form-section">
          <div class="section-title">目标配置</div>
          <a-form-item label="目标URL" field="targetUrl" required>
            <a-input
              v-model="formData.targetUrl"
              placeholder="http://localhost:11434"
            />
          </a-form-item>

          <a-form-item label="API密钥" field="apiKey">
            <a-input-password
              v-model="formData.apiKey"
              placeholder="API密钥（可选，将加密存储）"
            />
            <template #extra>
              <icon-info-circle /> 密钥将加密存储，仅在使用时解密
            </template>
          </a-form-item>

          <a-form-item label="超时时间（秒）" field="timeout" required>
            <a-input-number
              v-model="formData.timeout"
              :min="30"
              :max="600"
              :step="10"
              style="width: 200px"
            />
            <template #extra>
              <icon-info-circle /> 建议300秒，适合长连接和流式响应
            </template>
          </a-form-item>
        </div>

        <!-- 分组和Agent -->
        <div class="form-section">
          <div class="section-title">分组和Agent</div>
          <a-form-item label="业务分组" field="groupId" required>
            <a-select
              v-model="formData.groupId"
              placeholder="请选择业务分组"
              allow-search
              @change="handleGroupChange"
            >
              <a-option v-for="group in flatGroups" :key="group.id" :value="group.id">
                {{ group.name }}
              </a-option>
            </a-select>
          </a-form-item>

          <a-form-item label="Agent主机" field="agentHostIds" required>
            <a-select
              v-model="formData.agentHostIds"
              placeholder="请选择Agent主机（至少1个）"
              multiple
              allow-search
            >
              <a-option
                v-for="host in filteredAgentHosts"
                :key="host.id"
                :value="host.id"
              >
                {{ host.name }} ({{ host.ip }})
                <a-tag
                  v-if="host.agentStatus === 'online'"
                  color="green"
                  size="small"
                  style="margin-left: 8px"
                >
                  在线
                </a-tag>
                <a-tag v-else color="red" size="small" style="margin-left: 8px">
                  离线
                </a-tag>
              </a-option>
            </a-select>
          </a-form-item>
        </div>

        <!-- 状态 -->
        <div class="form-section">
          <div class="section-title">状态</div>
          <a-form-item field="status">
            <a-checkbox v-model="formData.statusChecked">
              启用代理
            </a-checkbox>
          </a-form-item>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted, h } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconPlus,
  IconSearch,
  IconRefresh,
  IconInfoCircle,
  IconCheckCircleFill,
  IconCloseCircleFill,
  IconCopy,
  IconDown,
  IconThunderbolt,
  IconEdit,
  IconDelete
} from '@arco-design/web-vue/es/icon'
import {
  getAIModelProxies,
  createAIModelProxy,
  updateAIModelProxy,
  deleteAIModelProxy,
  regenerateAIModelProxyToken,
  testAIModelProxyConnection,
  type AIModelProxyVO,
  type AIModelProxyRequest
} from '@/api/aiModelProxy'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'

// 搜索表单
const searchForm = reactive({
  keyword: '',
  modelType: '',
  status: undefined as number | undefined,
  groupId: undefined as number | undefined
})

// 分页
const pagination = reactive({
  current: 1,
  pageSize: 12
})

// 数据
const loading = ref(false)
const proxyList = ref<AIModelProxyVO[]>([])
const total = ref(0)

// 业务分组
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

// Agent主机
const agentHosts = ref<any[]>([])
const allAgentHosts = ref<any[]>([]) // 保存所有Agent主机
const filteredAgentHosts = computed(() => {
  if (!formData.groupId) {
    return agentHosts.value
  }
  return agentHosts.value.filter(host => {
    // 如果主机没有分组信息，也显示
    if (!host.groupIds || host.groupIds.length === 0) {
      return true
    }
    // 检查主机是否属于选中的分组
    return host.groupIds.includes(formData.groupId)
  })
})

// 模型类型图标和颜色
const modelTypeIcons: Record<string, string> = {
  ollama: '🦙',
  openai: '🤖',
  custom: '⚙️'
}

const modelTypeColors: Record<string, string> = {
  ollama: 'orange',
  openai: 'green',
  custom: 'blue'
}

const getModelTypeIcon = (type: string) => {
  return modelTypeIcons[type] || '🤖'
}

const getModelTypeColor = (type: string) => {
  return modelTypeColors[type] || 'gray'
}

// 弹窗
const modalVisible = ref(false)
const modalTitle = ref('新建AI模型代理')
const isEdit = ref(false)
const editId = ref(0)

// 表单
const formRef = ref()
const formData = reactive<AIModelProxyRequest & { statusChecked: boolean }>({
  name: '',
  description: '',
  modelType: 'ollama',
  targetUrl: '',
  apiKey: '',
  timeout: 300,
  status: 1,
  groupId: 0,
  agentHostIds: [],
  statusChecked: true
})

const formRules = {
  name: [
    { required: true, message: '请输入代理名称' },
    { minLength: 2, message: '代理名称至少2个字符' }
  ],
  modelType: [
    { required: true, message: '请选择模型类型' }
  ],
  targetUrl: [
    { required: true, message: '请输入目标URL' },
    {
      validator: (value: string, cb: any) => {
        if (!/^https?:\/\/.+/.test(value)) {
          cb('请输入有效的HTTP/HTTPS URL')
        } else {
          cb()
        }
      }
    }
  ],
  timeout: [
    { required: true, message: '请输入超时时间' },
    {
      validator: (value: number, cb: any) => {
        if (value < 30 || value > 600) {
          cb('超时时间必须在30-600秒之间')
        } else {
          cb()
        }
      }
    }
  ],
  groupId: [
    { required: true, message: '请选择业务分组' }
  ],
  agentHostIds: [
    {
      validator: (value: any, cb: any) => {
        if (!value || value.length === 0) {
          cb('请至少选择1台Agent主机')
        } else {
          cb()
        }
      }
    }
  ]
}

// 获取数据
const fetchData = async () => {
  loading.value = true
  try {
    const res = await getAIModelProxies({
      page: pagination.current,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword || undefined,
      modelType: searchForm.modelType || undefined,
      status: searchForm.status,
      groupId: searchForm.groupId
    })
    proxyList.value = res.list || []
    total.value = res.total || 0
  } catch (error: any) {
    Message.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

// 获取业务分组
const fetchGroups = async () => {
  try {
    const res = await getGroupTree()
    groupTree.value = res || []
  } catch (error: any) {
    Message.error(error.message || '获取业务分组失败')
  }
}

// 获取Agent主机
const fetchAgentHosts = async () => {
  try {
    const res = await getHostList({
      page: 1,
      pageSize: 1000
    })
    // 统一处理响应数据：可能在 res.data.list 或 res.list 中
    const list = res.data?.list || res.list || []
    // 过滤出有agentId的主机
    allAgentHosts.value = list.filter((h: any) => h.agentId)
    // 初始加载时显示所有Agent主机
    agentHosts.value = allAgentHosts.value
  } catch (error: any) {
    Message.error(error.message || '获取Agent主机失败')
  }
}

// 搜索
const handleSearch = () => {
  pagination.current = 1
  fetchData()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.modelType = ''
  searchForm.status = undefined
  searchForm.groupId = undefined
  pagination.current = 1
  fetchData()
}

// 分页变化
const handlePageChange = (page: number) => {
  pagination.current = page
  fetchData()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.current = 1
  fetchData()
}

// 复制代理URL
const handleCopyProxyUrl = async (record: AIModelProxyVO) => {
  const proxyUrl = `${window.location.origin}${record.proxyUrl}`

  try {
    await navigator.clipboard.writeText(proxyUrl)
    Message.success('代理URL已复制到剪贴板')
  } catch (err) {
    // 降级方案
    const textarea = document.createElement('textarea')
    textarea.value = proxyUrl
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    Message.success('代理URL已复制到剪贴板')
  }
}

// 测试连接
const handleTestConnection = async (record: AIModelProxyVO) => {
  const modal = Modal.info({
    title: '测试连接',
    content: '正在测试连接...',
    footer: false,
    closable: false
  })

  try {
    const res = await testAIModelProxyConnection(record.id)
    modal.close()

    if (res.success) {
      Modal.success({
        title: '测试成功',
        content: `连接正常${res.latency ? `，响应时间: ${res.latency}ms` : ''}`
      })
    } else {
      Modal.error({
        title: '测试失败',
        content: res.message || '连接失败，请检查配置'
      })
    }
  } catch (err: any) {
    modal.close()
    Modal.error({
      title: '测试失败',
      content: err.message || '连接失败，请检查配置'
    })
  }
}

// 重新生成Token
const handleRegenerateToken = (record: AIModelProxyVO) => {
  Modal.confirm({
    title: '重新生成Token',
    content: '重新生成Token后，旧Token将立即失效。确定要继续吗？',
    onOk: async () => {
      try {
        const res = await regenerateAIModelProxyToken(record.id)
        Message.success('Token已重新生成')

        // 显示新Token
        const proxyUrl = `${window.location.origin}${res.proxyUrl}`
        Modal.info({
          title: '新Token',
          content: () => h('div', [
            h('p', { style: 'margin-bottom: 8px' }, '新的代理URL：'),
            h('a-input', {
              modelValue: proxyUrl,
              readonly: true,
              style: 'margin-bottom: 8px'
            }),
            h('a-button', {
              type: 'primary',
              onClick: () => handleCopyProxyUrl(res)
            }, '复制')
          ])
        })

        // 刷新列表
        fetchData()
      } catch (err: any) {
        Message.error(err.message || '重新生成Token失败')
      }
    }
  })
}

// 创建
const handleCreate = () => {
  isEdit.value = false
  modalTitle.value = '新建AI模型代理'
  resetForm()
  modalVisible.value = true
}

// 编辑
const handleEdit = (record: AIModelProxyVO) => {
  isEdit.value = true
  editId.value = record.id
  modalTitle.value = '编辑AI模型代理'

  formData.name = record.name
  formData.description = record.description
  formData.modelType = record.modelType as any
  formData.targetUrl = record.targetUrl
  formData.apiKey = ''  // 不回显密钥
  formData.timeout = record.timeout
  formData.status = record.status
  formData.statusChecked = record.status === 1
  formData.groupId = record.groupId
  formData.agentHostIds = record.agentHostIds

  modalVisible.value = true
}

// 删除
const handleDelete = (record: AIModelProxyVO) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除代理"${record.name}"吗？此操作不可恢复。`,
    onOk: async () => {
      try {
        await deleteAIModelProxy(record.id)
        Message.success('删除成功')
        fetchData()
      } catch (err: any) {
        Message.error(err.message || '删除失败')
      }
    }
  })
}

// 弹窗确认
const handleModalOk = async () => {
  try {
    await formRef.value?.validate()

    const data: AIModelProxyRequest = {
      name: formData.name,
      description: formData.description,
      modelType: formData.modelType,
      targetUrl: formData.targetUrl,
      apiKey: formData.apiKey,
      timeout: formData.timeout,
      status: formData.statusChecked ? 1 : 0,
      groupId: formData.groupId,
      agentHostIds: formData.agentHostIds
    }

    if (isEdit.value) {
      await updateAIModelProxy(editId.value, data)
      Message.success('更新成功')
    } else {
      await createAIModelProxy(data)
      Message.success('创建成功')
    }

    modalVisible.value = false
    fetchData()
    return true
  } catch (error: any) {
    if (error.message) {
      Message.error(error.message)
    }
    return false
  }
}

// 弹窗取消
const handleModalCancel = () => {
  modalVisible.value = false
}

// 重置表单
const resetForm = () => {
  formData.name = ''
  formData.description = ''
  formData.modelType = 'ollama'
  formData.targetUrl = ''
  formData.apiKey = ''
  formData.timeout = 300
  formData.status = 1
  formData.statusChecked = true
  formData.groupId = 0
  formData.agentHostIds = []
  formRef.value?.clearValidate()
}

// 分组变化
const handleGroupChange = () => {
  // 根据分组过滤Agent主机
  if (!formData.groupId) {
    // 没有选择分组，显示所有Agent主机
    agentHosts.value = allAgentHosts.value
  } else {
    // 根据分组过滤主机
    agentHosts.value = allAgentHosts.value.filter((host: any) => {
      // 如果主机没有分组信息，也显示
      if (!host.groupIds || host.groupIds.length === 0) {
        return true
      }
      // 检查主机是否属于选中的分组
      return host.groupIds.includes(formData.groupId)
    })
  }

  // 清空已选择的Agent主机（如果不在新分组中）
  if (formData.groupId) {
    const validHostIds = agentHosts.value.map(h => h.id)
    formData.agentHostIds = formData.agentHostIds.filter(id => validHostIds.includes(id))
  }
}

// 显示使用指南
const showUsageGuide = () => {
  Modal.info({
    title: 'AI模型代理使用指南',
    width: 800,
    content: () => h('div', { class: 'usage-guide' }, [
      h('h3', '1. 创建代理'),
      h('p', '点击"新建代理"按钮，填写代理信息：'),
      h('ul', [
        h('li', '选择模型类型（Ollama/OpenAI/Custom）'),
        h('li', '填写目标URL（如 http://localhost:11434）'),
        h('li', '选择至少1台在线的Agent主机'),
        h('li', '可选：填写API密钥（将加密存储）')
      ]),

      h('h3', { style: 'margin-top: 16px' }, '2. 复制代理URL'),
      h('p', '创建成功后，点击"复制URL"按钮获取代理地址。'),

      h('h3', { style: 'margin-top: 16px' }, '3. 在代码中使用'),
      h('pre', {
        style: 'background: #f6f6f6; padding: 12px; border-radius: 4px; overflow-x: auto'
      }, `# Python示例
import requests

proxy_url = "http://your-domain/api/v1/ai-model-proxy/{token}"

response = requests.post(
    f"{proxy_url}/api/chat",
    json={
        "model": "llama2",
        "messages": [{"role": "user", "content": "Hello"}],
        "stream": True
    },
    stream=True
)

for line in response.iter_lines():
    if line:
        print(line.decode('utf-8'))`),

      h('h3', { style: 'margin-top: 16px' }, '4. 测试连接'),
      h('p', '点击"测试"按钮验证代理配置是否正确。'),

      h('h3', { style: 'margin-top: 16px' }, '5. 安全提示'),
      h('ul', [
        h('li', '代理Token永久有效，请妥善保管'),
        h('li', 'API密钥加密存储，仅在使用时解密'),
        h('li', '如Token泄露，请立即重新生成')
      ])
    ])
  })
}

// 初始化
onMounted(() => {
  fetchGroups()
  fetchAgentHosts()
  fetchData()
})
</script>

<style lang="scss" scoped>
.ai-model-proxies-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;

  h2 {
    margin: 0;
    font-size: 24px;
    font-weight: 600;
  }
}

.search-bar {
  margin-bottom: 20px;
}

.proxy-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.proxy-card {
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-2);
  border-radius: 8px;
  padding: 16px;
  transition: all 0.3s;
  cursor: pointer;

  &:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;

    .header-left {
      display: flex;
      align-items: center;
      gap: 8px;
      flex: 1;
      min-width: 0;

      .model-icon {
        font-size: 24px;
        flex-shrink: 0;
      }

      .proxy-name {
        font-size: 16px;
        font-weight: 600;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .header-right {
      flex-shrink: 0;
    }
  }

  .card-content {
    margin-bottom: 12px;

    .info-row {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;
      font-size: 14px;

      &:last-child {
        margin-bottom: 0;
      }

      .label {
        color: var(--color-text-3);
        flex-shrink: 0;
      }

      .value {
        color: var(--color-text-1);
        flex: 1;
        min-width: 0;

        &.truncate {
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }

      &.description {
        .value {
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
          color: var(--color-text-3);
        }
      }
    }
  }

  .card-footer {
    display: flex;
    gap: 8px;
    padding-top: 12px;
    border-top: 1px solid var(--color-border-2);
  }
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

.form-section {
  margin-bottom: 24px;

  &:last-child {
    margin-bottom: 0;
  }

  .section-title {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 16px;
    color: var(--color-text-1);
  }
}

.model-icon {
  font-size: 18px;
}

.usage-guide {
  h3 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }

  p {
    margin: 8px 0;
    color: var(--color-text-2);
  }

  ul {
    margin: 8px 0;
    padding-left: 24px;

    li {
      margin: 4px 0;
      color: var(--color-text-2);
    }
  }

  pre {
    margin: 8px 0;
    font-size: 13px;
    line-height: 1.6;
  }
}

// 响应式
@media (max-width: 768px) {
  .proxy-grid {
    grid-template-columns: 1fr;
  }

  .search-bar {
    :deep(.arco-space) {
      flex-wrap: wrap;
    }
  }
}
</style>
