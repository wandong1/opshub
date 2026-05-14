<template>
  <a-modal
    v-model:visible="visible"
    title="推送规则巡检配置"
    width="800px"
    :mask-closable="false"
    @ok="handleSave"
    @cancel="handleCancel"
  >
    <a-form :model="form" layout="vertical">
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="启用巡检">
            <a-switch v-model="form.enabled" />
            <div style="font-size: 12px; color: var(--ops-text-secondary); margin-top: 4px;">
              启用后将按配置的间隔或时间自动巡检此规则的告警
            </div>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="推送模式">
            <a-select v-model="form.sendMode">
              <a-option value="always">总是推送（包括无告警）</a-option>
              <a-option value="only_firing">仅有告警时推送</a-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>

      <a-divider orientation="left">巡检时间配置</a-divider>

      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="巡检模式">
            <a-radio-group v-model="form.patrolMode">
              <a-radio value="interval">按间隔巡检</a-radio>
              <a-radio value="fixed">固定时间巡检</a-radio>
            </a-radio-group>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item v-if="form.patrolMode === 'interval'" label="巡检间隔">
            <a-input-number v-model="form.patrolInterval" :min="60" :max="86400" style="width: 100%">
              <template #suffix>秒</template>
            </a-input-number>
            <div style="font-size: 12px; color: var(--ops-text-secondary); margin-top: 4px;">
              建议：1小时=3600秒，4小时=14400秒
            </div>
          </a-form-item>
          <a-form-item v-else label="巡检时间点">
            <a-select
              v-model="patrolTimes"
              multiple
              allow-create
              allow-search
              placeholder="选择或输入时间（如 09:00），按回车添加"
              style="width: 100%"
            >
              <a-option value="00:00">00:00 - 凌晨</a-option>
              <a-option value="06:00">06:00 - 早晨</a-option>
              <a-option value="09:00">09:00 - 上午</a-option>
              <a-option value="12:00">12:00 - 中午</a-option>
              <a-option value="14:00">14:00 - 下午</a-option>
              <a-option value="18:00">18:00 - 傍晚</a-option>
              <a-option value="21:00">21:00 - 晚上</a-option>
              <a-option value="23:00">23:00 - 深夜</a-option>
            </a-select>
            <div style="font-size: 12px; color: var(--ops-text-secondary); margin-top: 4px;">
              💡 可选择预设时间或输入自定义时间（格式：HH:mm，如 09:30、15:45），支持添加多个时间点
            </div>
          </a-form-item>
        </a-col>
      </a-row>

      <a-divider orientation="left">巡检范围配置</a-divider>

      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item label="时间范围">
            <a-input-number v-model="form.timeRange" :min="0" :max="604800" style="width: 100%">
              <template #suffix>秒</template>
            </a-input-number>
            <div style="font-size: 12px; color: var(--ops-text-secondary); margin-top: 4px;">
              0=所有未恢复告警，>0=最近N秒内的告警
            </div>
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="最大告警数">
            <a-input-number v-model="form.maxAlertsPerReport" :min="10" :max="500" style="width: 100%" />
            <div style="font-size: 12px; color: var(--ops-text-secondary); margin-top: 4px;">
              单次巡检报告最多包含的告警数量
            </div>
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="包含已恢复">
            <a-switch v-model="form.includeResolved" />
            <div style="font-size: 12px; color: var(--ops-text-secondary); margin-top: 4px;">
              是否在报告中统计已恢复的告警
            </div>
          </a-form-item>
        </a-col>
      </a-row>

      <a-divider orientation="left">报告样式配置</a-divider>

      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="报告样式">
            <a-select v-model="form.reportStyle">
              <a-option value="detailed">详细报告（显示每条告警）</a-option>
              <a-option value="summary">摘要报告（仅显示统计）</a-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="分组维度">
            <a-select v-model="form.groupBy">
              <a-option value="severity">按告警级别分组</a-option>
              <a-option value="ruleName">按规则名称分组</a-option>
              <a-option value="assetGroup">按业务分组</a-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>

      <a-space style="margin-top: 16px;">
        <a-button type="outline" @click="handleExecute" :loading="executing">
          <template #icon><icon-play-arrow /></template>
          手动执行巡检
        </a-button>
        <a-button type="outline" @click="handleViewReports">
          <template #icon><icon-file /></template>
          查看巡检报告
        </a-button>
      </a-space>
    </a-form>
  </a-modal>

  <!-- 巡检报告列表弹窗 -->
  <a-modal
    v-model:visible="reportsVisible"
    title="巡检报告列表"
    width="1200px"
    :footer="false"
  >
    <a-table :data="reports" :loading="reportsLoading" :pagination="reportsPagination" @page-change="handleReportsPageChange">
      <template #columns>
        <a-table-column title="巡检时间" data-index="createdAt" :width="180" />
        <a-table-column title="活跃告警" :width="100">
          <template #cell="{ record }">
            <a-tag :color="record.firingCount > 0 ? 'red' : 'green'">{{ record.firingCount }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="严重" :width="80">
          <template #cell="{ record }"><a-tag color="red">{{ record.criticalCount }}</a-tag></template>
        </a-table-column>
        <a-table-column title="警告" :width="80">
          <template #cell="{ record }"><a-tag color="orange">{{ record.warningCount }}</a-tag></template>
        </a-table-column>
        <a-table-column title="提示" :width="80">
          <template #cell="{ record }"><a-tag color="blue">{{ record.infoCount }}</a-tag></template>
        </a-table-column>
        <a-table-column title="已推送" :width="80">
          <template #cell="{ record }">
            <a-tag :color="record.sent ? 'green' : 'gray'">{{ record.sent ? '是' : '否' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="推送时间" data-index="sentAt" :width="180" />
      </template>
    </a-table>
  </a-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import request from '@/utils/request'

interface PatrolConfig {
  id?: number
  subscriptionRuleId: number
  enabled: boolean
  patrolMode: string
  patrolInterval: number
  patrolTimes: string
  includeResolved: boolean
  timeRange: number
  maxAlertsPerReport: number
  sendMode: string
  reportStyle: string
  groupBy: string
}

interface PatrolReport {
  id: number
  createdAt: string
  firingCount: number
  criticalCount: number
  warningCount: number
  infoCount: number
  sent: boolean
  sentAt: string
}

const props = defineProps<{
  ruleId: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const visible = ref(false)
const form = ref<PatrolConfig>({
  subscriptionRuleId: 0,
  enabled: false,
  patrolMode: 'interval',
  patrolInterval: 3600,
  patrolTimes: '[]',
  includeResolved: false,
  timeRange: 0,
  maxAlertsPerReport: 100,
  sendMode: 'always',
  reportStyle: 'detailed',
  groupBy: 'severity'
})

const patrolTimes = ref<string[]>([])

const executing = ref(false)
const reportsVisible = ref(false)
const reports = ref<PatrolReport[]>([])
const reportsLoading = ref(false)
const reportsPagination = ref({
  current: 1,
  pageSize: 20,
  total: 0
})

// 打开配置弹窗
const open = async () => {
  visible.value = true
  form.value.subscriptionRuleId = props.ruleId
  await loadConfig()
}

// 加载巡检配置
const loadConfig = async () => {
  try {
    const data = await request.get(`/api/v1/alert/patrol/rule/${props.ruleId}`)
    console.log('巡检配置响应:', data)

    // request 已经自动解包，直接使用返回的数据
    if (data) {
      form.value = data
      // 确保 subscriptionRuleId 正确设置为当前的 ruleId
      form.value.subscriptionRuleId = props.ruleId
      // 解析 JSON 字段
      patrolTimes.value = JSON.parse(form.value.patrolTimes || '[]')
    }
  } catch (error: any) {
    console.error('加载巡检配置错误:', error)
    Message.error('加载巡检配置失败：' + (error.message || '未知错误'))
  }
}

// 保存配置
const handleSave = async () => {
  try {
    // 确保 subscriptionRuleId 正确
    form.value.subscriptionRuleId = props.ruleId

    // 验证固定时间模式下的时间格式
    if (form.value.patrolMode === 'fixed' && patrolTimes.value.length > 0) {
      const timeRegex = /^([0-1][0-9]|2[0-3]):[0-5][0-9]$/
      const invalidTimes = patrolTimes.value.filter(time => !timeRegex.test(time))
      if (invalidTimes.length > 0) {
        Message.warning(`时间格式不正确：${invalidTimes.join(', ')}，请使用 HH:mm 格式（如 09:00、18:30）`)
        return
      }
    }

    // 序列化 JSON 字段
    form.value.patrolTimes = JSON.stringify(patrolTimes.value)

    console.log('保存巡检配置，数据:', form.value)

    await request.post('/api/v1/alert/patrol/rule', form.value)
    Message.success('保存成功')
    visible.value = false
    emit('close')
  } catch (error: any) {
    console.error('保存配置错误:', error)
    Message.error('保存失败：' + (error.message || '未知错误'))
  }
}

// 取消
const handleCancel = () => {
  visible.value = false
  emit('close')
}

// 手动执行巡检
const handleExecute = async () => {
  executing.value = true
  try {
    await request.post(`/api/v1/alert/patrol/rule/${props.ruleId}/execute`)
    Message.success('巡检任务已启动，请稍后查看报告')
  } catch (error: any) {
    console.error('执行巡检错误:', error)
    Message.error('执行失败：' + (error.message || '未知错误'))
  } finally {
    executing.value = false
  }
}

// 查看巡检报告
const handleViewReports = async () => {
  reportsVisible.value = true
  await loadReports()
}

// 加载巡检报告列表
const loadReports = async () => {
  reportsLoading.value = true
  try {
    const data = await request.get(`/api/v1/alert/patrol/rule/${props.ruleId}/reports`, {
      params: {
        page: reportsPagination.value.current,
        page_size: reportsPagination.value.pageSize
      }
    })

    if (data) {
      reports.value = data.data || []
      reportsPagination.value.total = data.total || 0
    }
  } catch (error: any) {
    console.error('加载报告错误:', error)
    Message.error('加载报告失败：' + (error.message || '未知错误'))
  } finally {
    reportsLoading.value = false
  }
}

// 分页变化
const handleReportsPageChange = (page: number) => {
  reportsPagination.value.current = page
  loadReports()
}

defineExpose({
  open
})
</script>

<style scoped>
.page-container {
  padding: 20px;
}
</style>
