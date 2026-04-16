<template>
  <a-modal v-model:visible="visible" title="关联Agent主机" @ok="handleOk" width="600px">
    <a-alert type="info">
      <template #title>关联说明</template>
      选择一个或多个在线的Agent主机作为数据源代理，系统会按优先级选择在线的Agent转发请求
    </a-alert>

    <a-divider />

    <a-table :data="relations" :loading="loading" row-key="id" :pagination="false">
      <template #columns>
        <a-table-column title="主机名" :width="200">
          <template #cell="{ record }">{{ getHostName(record.agent_host_id) }}</template>
        </a-table-column>
        <a-table-column title="优先级" :width="100">
          <template #cell="{ record }">
            <a-input-number v-model="record.priority" :min="0" :max="10" size="small" />
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="80">
          <template #cell="{ record }">
            <a-link status="danger" size="small" @click="removeRelation(record.id)">删除</a-link>
          </template>
        </a-table-column>
      </template>
    </a-table>

    <a-divider />

    <a-form :model="newRelation" layout="vertical">
      <a-form-item label="选择主机">
        <a-select v-model="newRelation.agent_host_id" placeholder="选择在线的Agent主机">
          <a-option v-for="host in onlineAgents" :key="host.id" :value="host.id">
            {{ host.name }} ({{ host.ip }})
          </a-option>
        </a-select>
      </a-form-item>
      <a-form-item label="优先级">
        <a-input-number v-model="newRelation.priority" :min="0" :max="10" placeholder="0-10，越小优先级越高" />
      </a-form-item>
      <a-button type="primary" @click="addRelation">添加关联</a-button>
    </a-form>
  </a-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import request from '@/utils/request'
import {
  getAgentRelations,
  createAgentRelation,
  deleteAgentRelation,
  type DataSourceAgentRelation
} from '@/api/alert'

interface Host {
  id: number
  name: string
  ip: string
  agentStatus?: string
}

const visible = ref(false)
const loading = ref(false)
const datasourceId = ref<number>()
const relations = ref<DataSourceAgentRelation[]>([])
const onlineAgents = ref<Host[]>([])
const newRelation = ref({ agent_host_id: 0, priority: 0 })

const getHostName = (hostId: number) => {
  const host = onlineAgents.value.find(h => h.id === hostId)
  return host ? `${host.name} (${host.ip})` : `主机#${hostId}`
}

const loadHosts = async () => {
  try {
    // 使用request库调用API，自动携带认证信息
    const res = await request.get('/api/v1/hosts', { params: { page: 1, page_size: 1000 } })

    // 后端返回的是分页数据格式：{ list: [...], page: 1, pageSize: 10, total: 5 }
    const hosts = (res as any).data?.list || (res as any).list || []

    // 过滤在线的Agent主机（agentStatus为online）
    onlineAgents.value = hosts.filter((h: any) => h.agentStatus === 'online')
  } catch (err) {
    console.error('加载主机列表失败', err)
  }
}

const loadRelations = async () => {
  if (!datasourceId.value) return
  try {
    loading.value = true
    const res = await getAgentRelations(datasourceId.value)
    relations.value = res.data || []
  } catch (err) {
    Message.error('加载Agent关联失败')
  } finally {
    loading.value = false
  }
}

const addRelation = async () => {
  if (!newRelation.value.agent_host_id) {
    Message.warning('请选择主机')
    return
  }
  try {
    await createAgentRelation({
      data_source_id: datasourceId.value,
      agent_host_id: newRelation.value.agent_host_id,
      priority: newRelation.value.priority
    })
    Message.success('添加成功')
    newRelation.value = { agent_host_id: 0, priority: 0 }
    await loadRelations()
  } catch (err) {
    Message.error('添加失败')
  }
}

const removeRelation = async (id: number | undefined) => {
  if (!id) return
  try {
    await deleteAgentRelation(id)
    Message.success('删除成功')
    await loadRelations()
  } catch (err) {
    Message.error('删除失败')
  }
}

const handleOk = async () => {
  visible.value = false
}

const open = async (dsId: number) => {
  datasourceId.value = dsId
  visible.value = true
  await loadHosts()
  await loadRelations()
}

defineExpose({ open })
</script>

<style scoped>
.agent-relations {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.help-text {
  font-size: 12px;
  color: var(--ops-text-tertiary);
  margin-top: 4px;
  display: block;
}
</style>
