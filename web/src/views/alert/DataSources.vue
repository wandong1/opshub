<template>
  <div class="page-container">
    <a-card :bordered="false">
      <template #title>数据源管理</template>
      <template #extra>
        <a-button type="primary" @click="openCreate"><template #icon><icon-plus /></template>新增数据源</a-button>
      </template>
      <a-table :data="list" :loading="loading" row-key="id">
        <template #columns>
          <a-table-column title="名称" data-index="name" />
          <a-table-column title="类型" data-index="type">
            <template #cell="{ record }"><a-tag :color="typeColor(record.type)">{{ typeLabel(record.type) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="地址">
            <template #cell="{ record }">{{ record.url }}</template>
          </a-table-column>
          <a-table-column title="接入方式">
            <template #cell="{ record }"><a-tag>{{ record.access_mode === 'agent' ? 'Agent代理' : '直连' }}</a-tag></template>
          </a-table-column>
          <a-table-column title="状态" data-index="status">
            <template #cell="{ record }"><a-badge :status="record.status===1?'success':'danger'" :text="record.status===1?'启用':'禁用'" /></template>
          </a-table-column>
          <a-table-column title="操作">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="testConn(record)">测试</a-link>
                <a-link @click="openEdit(record)">编辑</a-link>
                <a-popconfirm content="确认删除？" @ok="remove(record.id)">
                  <a-link status="danger">删除</a-link>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="modalVisible" :title="form.id?'编辑数据源':'新增数据源'" @ok="saveDatasource" @cancel="modalVisible=false" width="700px">
      <a-form :model="form" layout="vertical">
        <a-form-item label="名称" required><a-input v-model="form.name" placeholder="请输入数据源名称" /></a-form-item>
        <a-form-item label="类型" required>
          <a-select v-model="form.type" placeholder="请选择数据源类型">
            <a-option value="prometheus">Prometheus</a-option>
            <a-option value="victoriametrics">VictoriaMetrics</a-option>
            <a-option value="influxdb">InfluxDB</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="接入方式" required>
          <a-select v-model="form.access_mode" :disabled="!!form.id" placeholder="请选择接入方式">
            <a-option value="direct">直连</a-option>
            <a-option value="agent">Agent代理</a-option>
          </a-select>
        </a-form-item>

        <!-- Agent代理模式：选择Agent主机 -->
        <template v-if="form.access_mode === 'agent'">
          <a-form-item label="关联Agent主机" required>
            <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: flex-start;">
              <!-- 已关联的主机列表 -->
              <div style="width: 100%; display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px;">
                <a-tag v-for="(rel, index) in agentRelations" :key="index" closable @close="() => removeAgentRelation(index)">
                  {{ getHostName(rel.agent_host_id) }} (优先级: {{ rel.priority }})
                </a-tag>
              </div>
              <!-- 添加新关联的下拉菜单 -->
              <div style="width: 100%;">
                <div style="display: flex; gap: 8px; align-items: center;">
                  <a-select
                    v-model="selectedAgentHostId"
                    placeholder="选择要添加的Agent主机"
                    style="flex: 1;"
                    allow-clear
                  >
                    <a-option v-for="host in onlineAgents" :key="host.id" :value="host.id">
                      {{ host.name }} ({{ host.ip }})
                    </a-option>
                  </a-select>
                  <a-input-number
                    v-model="newAgentPriority"
                    :min="0"
                    :max="10"
                    placeholder="优先级"
                    style="width: 100px;"
                  />
                  <a-button type="primary" size="small" @click="addAgentRelationLocal">添加</a-button>
                </div>
              </div>
            </div>
          </a-form-item>
        </template>

        <!-- 直连模式：输入完整URL -->
        <a-form-item v-if="form.access_mode === 'direct'" label="数据源地址" required>
          <a-input v-model="form.url" placeholder="http://prometheus:9090" />
          <span class="help-text">完整的数据源访问URL</span>
        </a-form-item>

        <!-- Agent代理模式：输入URL -->
        <template v-if="form.access_mode === 'agent'">
          <a-form-item label="数据源地址" required>
            <a-input v-model="form.url" placeholder="http://prometheus:9090" />
            <span class="help-text">Agent主机上可访问的数据源地址（完整URL）</span>
          </a-form-item>

          <!-- 代理转发URL（已创建后显示） -->
          <a-form-item v-if="form.id && form.proxy_token" label="代理转发URL">
            <a-input v-model="form.proxy_url" readonly />
            <span class="help-text">复制此URL到Grafana数据源配置</span>
          </a-form-item>
        </template>
        <a-form-item label="用户名"><a-input v-model="form.username" /></a-form-item>
        <a-form-item label="密码"><a-input-password v-model="form.password" /></a-form-item>
        <a-form-item label="Token"><a-input v-model="form.token" /></a-form-item>
        <a-form-item label="描述"><a-textarea v-model="form.description" /></a-form-item>
        <a-form-item label="状态">
          <a-switch v-model="form.status" :checked-value="1" :unchecked-value="0" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getDataSources, createDataSource, updateDataSource, deleteDataSource, testDataSource, getAgentRelations, deleteAgentRelation, createAgentRelation, type AlertDataSource } from '@/api/alert'
import request from '@/utils/request'

const list = ref<AlertDataSource[]>([])
const loading = ref(false)
const modalVisible = ref(false)
const form = ref<Partial<AlertDataSource>>({ status: 1, access_mode: 'direct' })
const agentRelations = ref<any[]>([])
const hostMap = ref<Map<number, any>>(new Map())
const onlineAgents = ref<any[]>([])
const selectedAgentHostId = ref<number>()
const newAgentPriority = ref<number>(0)

const typeLabel = (t: string) => ({ prometheus: 'Prometheus', victoriametrics: 'VictoriaMetrics', influxdb: 'InfluxDB' }[t] || t)
const typeColor = (t: string) => ({ prometheus: 'orange', victoriametrics: 'purple', influxdb: 'blue' }[t] || 'gray')

const openCreate = async () => {
  form.value = {
    status: 1,
    access_mode: 'direct',
    // 通用字段
    url: '',
    username: '',
    password: '',
    token: '',
    description: '',
  }
  agentRelations.value = []
  selectedAgentHostId.value = undefined
  newAgentPriority.value = 0
  await loadHosts()
  modalVisible.value = true
}

// 本地添加 Agent 关联（不需要先保存数据源）
const addAgentRelationLocal = () => {
  if (!selectedAgentHostId.value) {
    Message.error('请选择Agent主机')
    return
  }
  // 检查是否已存在
  if (agentRelations.value.some(rel => rel.agent_host_id === selectedAgentHostId.value)) {
    Message.warning('该Agent主机已关联')
    return
  }
  // 添加到本地数组
  agentRelations.value.push({
    agent_host_id: selectedAgentHostId.value,
    priority: newAgentPriority.value
  })
  Message.success('Agent主机已添加')
  selectedAgentHostId.value = undefined
  newAgentPriority.value = 0
}

// 删除本地 Agent 关联
const removeAgentRelation = (index: number) => {
  agentRelations.value.splice(index, 1)
  Message.success('Agent主机已移除')
}

// 保存时同步 Agent 关联到数据库
const syncAgentRelations = async () => {
  if (!form.value.id) return

  try {
    // 获取数据库中的现有关联
    const existingRels = await getAgentRelations(form.value.id)
    const existingRelIds = ((existingRels as any)?.data || []).map((r: any) => r.agent_host_id)
    const newRelIds = agentRelations.value.filter(r => !r.id).map(r => r.agent_host_id)

    // 删除被移除的关联
    for (const rel of (existingRels as any)?.data || []) {
      if (!agentRelations.value.some(r => r.agent_host_id === rel.agent_host_id)) {
        await deleteAgentRelation(rel.id)
      }
    }

    // 添加新的关联
    for (const agentHostId of newRelIds) {
      const rel = agentRelations.value.find(r => r.agent_host_id === agentHostId)
      if (rel) {
        await createAgentRelation({
          data_source_id: form.value.id,
          agent_host_id: rel.agent_host_id,
          priority: rel.priority
        })
      }
    }
  } catch (err) {
    console.error('同步 Agent 关联失败:', err)
  }
}
const openEdit = async (row: AlertDataSource) => {
  form.value = { ...row }
  selectedAgentHostId.value = undefined
  newAgentPriority.value = 0
  await loadHosts()

  // 加载已关联的 Agent 主机
  if (row.access_mode === 'agent') {
    try {
      const res = await getAgentRelations(row.id!)
      console.log('获取关联数据:', res)
      // 后端返回格式：{ code, message, data: [...] } 或直接返回数组
      const rels = (res as any)?.data || res || []
      agentRelations.value = Array.isArray(rels) ? rels : []
      console.log('加载的关联:', agentRelations.value)
    } catch (err) {
      console.error('加载关联失败:', err)
      agentRelations.value = []
    }
  } else {
    agentRelations.value = []
  }

  modalVisible.value = true
}
const getHostName = (hostId: number) => {
  const host = hostMap.value.get(hostId)
  return host ? `${host.name} (${host.ip})` : `主机#${hostId}`
}
const loadHosts = async () => {
  try {
    // 使用request库调用API，自动携带认证信息
    const res = await request.get('/api/v1/hosts', { params: { page: 1, page_size: 1000 } })
    // 后端返回的是分页数据格式：{ list: [...], page: 1, pageSize: 10, total: 5 }
    const hosts = (res as any).data?.list || (res as any).list || []
    hosts.forEach((h: any) => {
      hostMap.value.set(h.id, h)
    })
    // 过滤在线的Agent主机（agentStatus为online）
    onlineAgents.value = hosts.filter((h: any) => h.agentStatus === 'online')
  } catch (err) {
    console.error('加载主机列表失败:', err)
    onlineAgents.value = []
  }
}

const load = async () => {
  loading.value = true
  try {
    await loadHosts()
    const res = await getDataSources()
    list.value = res?.data || res || []
  }
  finally { loading.value = false }
}

const saveDatasource = async () => {
  try {
    // 验证必填字段
    if (!form.value.name) {
      Message.error('请输入数据源名称')
      return
    }
    if (!form.value.type) {
      Message.error('请选择数据源类型')
      return
    }
    if (!form.value.access_mode) {
      Message.error('请选择接入方式')
      return
    }

    // 根据接入方式验证
    if (!form.value.url) {
      Message.error('请输入数据源地址')
      return
    }
    if (form.value.access_mode === 'agent') {
      if (agentRelations.value.length === 0) {
        Message.error('Agent代理模式下请至少关联一个Agent主机')
        return
      }
    }

    // 构建提交数据
    const submitData: any = {
      name: form.value.name,
      type: form.value.type,
      access_mode: form.value.access_mode,
      url: form.value.url,  // 两种模式都使用 URL
      username: form.value.username || '',
      password: form.value.password || '',
      token: form.value.token || '',
      description: form.value.description || '',
      status: form.value.status || 1,
    }

    // 保存或更新数据源
    if (form.value.id) {
      await updateDataSource(form.value.id, submitData)
      Message.success('数据源更新成功')
    } else {
      const res = await createDataSource(submitData)
      const responseData = (res as any)?.data || res
      const dataSourceId = (responseData as any)?.id
      if (dataSourceId) {
        form.value.id = dataSourceId
        Message.success('数据源保存成功')
      } else {
        throw new Error('未能获取数据源ID')
      }
    }

    // 同步 Agent 关联到数据库
    if (form.value.access_mode === 'agent') {
      await syncAgentRelations()
    }

    // 关闭弹窗
    modalVisible.value = false
    selectedAgentHostId.value = undefined
    newAgentPriority.value = 0
    await load()
  } catch (err) {
    Message.error('保存失败: ' + (err as any).message)
    console.error(err)
  }
}

const remove = async (id: number) => {
  try { await deleteDataSource(id); Message.success('删除成功'); load() }
  catch { Message.error('删除失败') }
}

const testConn = async (row: AlertDataSource) => {
  try {
    await testDataSource(row.id!)
    Message.success('连接成功')
  } catch (e: any) {
    Message.error('连接失败: ' + (e?.response?.data?.message || ''))
  }
}

onMounted(load)
</script>

<style scoped>
.page-container { padding: 20px; background: var(--ops-content-bg); min-height: 100%; }
.agent-relations {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
  margin-bottom: 12px;
}
.help-text {
  font-size: 12px;
  color: var(--ops-text-tertiary);
  margin-top: 4px;
  display: block;
}
</style>
