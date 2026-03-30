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
          <a-table-column title="地址" data-index="url" />
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

    <a-modal v-model:visible="modalVisible" :title="form.id?'编辑数据源':'新增数据源'" @ok="save" @cancel="modalVisible=false" width="600px">
      <a-form :model="form" layout="vertical">
        <a-form-item label="名称" required><a-input v-model="form.name" /></a-form-item>
        <a-form-item label="类型" required>
          <a-select v-model="form.type">
            <a-option value="prometheus">Prometheus</a-option>
            <a-option value="victoriametrics">VictoriaMetrics</a-option>
            <a-option value="influxdb">InfluxDB</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="地址" required><a-input v-model="form.url" placeholder="http://prometheus:9090" /></a-form-item>
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
import { getDataSources, createDataSource, updateDataSource, deleteDataSource, testDataSource, type AlertDataSource } from '@/api/alert'

const list = ref<AlertDataSource[]>([])
const loading = ref(false)
const modalVisible = ref(false)
const form = ref<Partial<AlertDataSource>>({ status: 1 })

const typeLabel = (t: string) => ({ prometheus: 'Prometheus', victoriametrics: 'VictoriaMetrics', influxdb: 'InfluxDB' }[t] || t)
const typeColor = (t: string) => ({ prometheus: 'orange', victoriametrics: 'purple', influxdb: 'blue' }[t] || 'gray')

const load = async () => {
  loading.value = true
  try { const res = await getDataSources(); list.value = res?.data || res || [] }
  finally { loading.value = false }
}

const openCreate = () => { form.value = { status: 1 }; modalVisible.value = true }
const openEdit = (row: AlertDataSource) => { form.value = { ...row }; modalVisible.value = true }

const save = async () => {
  try {
    if (form.value.id) { await updateDataSource(form.value.id, form.value) }
    else { await createDataSource(form.value) }
    Message.success('保存成功'); modalVisible.value = false; load()
  } catch { Message.error('保存失败') }
}

const remove = async (id: number) => {
  try { await deleteDataSource(id); Message.success('删除成功'); load() }
  catch { Message.error('删除失败') }
}

const testConn = async (row: AlertDataSource) => {
  try { await testDataSource(row.id!); Message.success('连接成功') }
  catch (e: any) { Message.error('连接失败: ' + (e?.response?.data?.message || '')) }
}

onMounted(load)
</script>

<style scoped>
.page-container { padding: 20px; background: var(--ops-content-bg); min-height: 100%; }
</style>
