<template>
  <div class="page-container">
    <a-card :bordered="false">
      <template #title>告警通道管理</template>
      <template #extra>
        <a-button type="primary" @click="openCreate"><template #icon><icon-plus /></template>新增通道</a-button>
      </template>

      <a-table :data="list" :loading="loading" row-key="id">
        <template #columns>
          <a-table-column title="名称" data-index="name" />
          <a-table-column title="类型" :width="130">
            <template #cell="{ record }">
              <a-tag :color="typeColor(record.type)">{{ typeLabel(record.type) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="启用" :width="80">
            <template #cell="{ record }">
              <a-switch :model-value="record.enabled" size="small"
                @change="(v) => quickToggle(record, !!v)" />
            </template>
          </a-table-column>
          <a-table-column title="AI智能体" :width="100">
            <template #cell="{ record }">
              <a-tag v-if="record.aiHookEnabled" color="purple">已开启</a-tag>
              <span v-else>—</span>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="160">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="doTest(record)">测试</a-link>
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

    <!-- 编辑弹窗 -->
    <a-modal v-model:visible="modalVisible" :title="form.id?'编辑通道':'新增通道'"
      @ok="save" @cancel="modalVisible=false" width="680px">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="通道名称" required><a-input v-model="form.name" /></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="通道类型" required>
              <a-select v-model="form.type" @change="onTypeChange">
                <a-option value="wechat_work">企业微信</a-option>
                <a-option value="dingtalk">钉钉</a-option>
                <a-option value="sms">短信</a-option>
                <a-option value="phone">电话</a-option>
                <a-option value="ai_agent">AI智能体</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <!-- 动态配置项 -->
        <template v-if="form.type === 'wechat_work'">
          <a-form-item label="Webhook URL" required><a-input v-model="cfg.webhookUrl" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." /></a-form-item>
        </template>
        <template v-else-if="form.type === 'dingtalk'">
          <a-form-item label="Webhook URL" required><a-input v-model="cfg.webhookUrl" /></a-form-item>
          <a-form-item label="加签密钥 (Secret)"><a-input v-model="cfg.secret" /></a-form-item>
        </template>
        <template v-else-if="form.type === 'sms'">
          <a-row :gutter="16">
            <a-col :span="8"><a-form-item label="服务商"><a-select v-model="cfg.provider"><a-option value="aliyun">阿里云</a-option><a-option value="tencent">腾讯云</a-option></a-select></a-form-item></a-col>
            <a-col :span="8"><a-form-item label="AppKey"><a-input v-model="cfg.appKey" /></a-form-item></a-col>
            <a-col :span="8"><a-form-item label="AppSecret"><a-input-password v-model="cfg.appSecret" /></a-form-item></a-col>
          </a-row>
          <a-form-item label="短信模板ID"><a-input v-model="cfg.templateId" /></a-form-item>
        </template>
        <template v-else-if="form.type === 'phone'">
          <a-row :gutter="16">
            <a-col :span="8"><a-form-item label="服务商"><a-select v-model="cfg.provider"><a-option value="aliyun">阿里云</a-option><a-option value="tencent">腾讯云</a-option></a-select></a-form-item></a-col>
            <a-col :span="8"><a-form-item label="AppKey"><a-input v-model="cfg.appKey" /></a-form-item></a-col>
            <a-col :span="8"><a-form-item label="AppSecret"><a-input-password v-model="cfg.appSecret" /></a-form-item></a-col>
          </a-row>
        </template>
        <template v-else-if="form.type === 'ai_agent'">
          <a-form-item label="AI智能体 Hook URL"><a-input v-model="cfg.hookUrl" placeholder="http://ai-agent/webhook" /></a-form-item>
          <a-form-item label="AI智能体总开关">
            <a-switch v-model="form.aiHookEnabled" />
            <span style="margin-left:8px;color:var(--ops-text-secondary);font-size:12px">开启后告警通知将同时发送至AI智能体进行智能收敛</span>
          </a-form-item>
        </template>

        <a-divider />
        <a-form-item label="告警通知模板">
          <a-textarea v-model="form.alertTemplate" :auto-size="{minRows:4}" style="font-family:monospace;font-size:12px" />
        </a-form-item>
        <a-form-item label="恢复通知模板">
          <a-textarea v-model="form.resolveTemplate" :auto-size="{minRows:4}" style="font-family:monospace;font-size:12px" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getChannels, createChannel, updateChannel, deleteChannel, testChannel, type AlertNotifyChannel } from '@/api/alert'

const list = ref<AlertNotifyChannel[]>([])
const loading = ref(false)
const modalVisible = ref(false)
const form = ref<Partial<AlertNotifyChannel>>({ enabled: true })
const cfg = reactive<Record<string, string>>({ webhookUrl: '', secret: '', appKey: '', appSecret: '', provider: 'aliyun', templateId: '', hookUrl: '' })

const typeLabel = (t: string) => ({ wechat_work: '企业微信', dingtalk: '钉钉', sms: '短信', phone: '电话', ai_agent: 'AI智能体' }[t] || t)
const typeColor = (t: string) => ({ wechat_work: 'green', dingtalk: 'orange', sms: 'blue', phone: 'purple', ai_agent: 'arcoblue' }[t] || 'gray')

const defaultAlertTpl = (type: string) => {
  if (type === 'wechat_work') return `## 🔴 SreHub 告警通知\n> **规则**: {{.RuleName}}\n> **级别**: {{.SeverityLabel}}\n> **当前值**: {{.Value}}\n> **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  if (type === 'dingtalk') return `## 🔴 SreHub 告警通知\n- **规则**: {{.RuleName}}\n- **级别**: {{.SeverityLabel}}\n- **当前值**: {{.Value}}\n- **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  return `【SreHub告警】规则: {{.RuleName}} | 级别: {{.SeverityLabel}} | 值: {{.Value}} | 时间: {{.FiredAt}}`
}
const defaultResolveTpl = (type = '') => {
  if (type === 'wechat_work') return `## ✅ SreHub 恢复通知\n> **规则**: {{.RuleName}}\n> **级别**: {{.SeverityLabel}}\n> **当前值**: {{.Value}}\n> **恢复时间**: {{.ResolvedAt}}\n> **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  if (type === 'dingtalk') return `## ✅ SreHub 恢复通知\n- **规则**: {{.RuleName}}\n- **级别**: {{.SeverityLabel}}\n- **当前值**: {{.Value}}\n- **恢复时间**: {{.ResolvedAt}}\n- **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  return `【SreHub恢复】规则: {{.RuleName}} | 级别: {{.SeverityLabel}} | 值: {{.Value}} | 恢复时间: {{.ResolvedAt}}`
}

const load = async () => {
  loading.value = true
  try { const res = await getChannels(); list.value = res?.data || res || [] }
  finally { loading.value = false }
}

const onTypeChange = (type: string) => {
  form.value.alertTemplate = defaultAlertTpl(type)
  form.value.resolveTemplate = defaultResolveTpl(type)
  Object.keys(cfg).forEach(k => cfg[k] = '')
  cfg.provider = 'aliyun'
}

const openCreate = () => {
  form.value = { enabled: true, type: 'wechat_work', alertTemplate: defaultAlertTpl('wechat_work'), resolveTemplate: defaultResolveTpl('wechat_work') }
  Object.keys(cfg).forEach(k => cfg[k] = '')
  cfg.provider = 'aliyun'
  modalVisible.value = true
}

const openEdit = (row: AlertNotifyChannel) => {
  form.value = { ...row }
  try { Object.assign(cfg, JSON.parse(row.config || '{}')) } catch {}
  modalVisible.value = true
}

const save = async () => {
  form.value.config = JSON.stringify(cfg)
  try {
    if (form.value.id) { await updateChannel(form.value.id, form.value) }
    else { await createChannel(form.value) }
    Message.success('保存成功'); modalVisible.value = false; load()
  } catch { Message.error('保存失败') }
}

const remove = async (id: number) => {
  try { await deleteChannel(id); Message.success('删除成功'); load() }
  catch { Message.error('删除失败') }
}

const doTest = async (row: AlertNotifyChannel) => {
  try { await testChannel(row.id!); Message.success('测试通知已发送') }
  catch { Message.error('发送失败') }
}

const quickToggle = async (row: AlertNotifyChannel, v: boolean) => {
  await updateChannel(row.id!, { ...row, enabled: v }).catch(() => {}); load()
}

onMounted(load)
</script>

<style scoped>
.page-container { padding: 20px; background: var(--ops-content-bg); min-height: 100%; }
</style>
