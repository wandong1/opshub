<template>
  <div class="governance-config">
    <a-collapse :default-active-key="['dedup']" :bordered="false">
      <!-- 去重规则 -->
      <a-collapse-item key="dedup" header="去重规则">
        <a-form :model="dedupRule" layout="vertical">
          <a-form-item label="启用去重">
            <a-switch v-model="dedupRule.enabled" />
          </a-form-item>
          <a-form-item label="指纹字段" v-if="dedupRule.enabled">
            <a-checkbox-group v-model="fingerprintKeys">
              <a-checkbox value="severity">告警级别</a-checkbox>
              <a-checkbox value="ruleName">规则名称</a-checkbox>
              <a-checkbox value="instance">实例</a-checkbox>
              <a-checkbox value="job">任务</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
          <a-form-item label="去重时间窗口(秒)" v-if="dedupRule.enabled">
            <a-input-number v-model="dedupRule.dedupWindow" :min="60" :max="3600" :step="60" style="width:200px" />
            <span style="margin-left:8px;color:var(--ops-text-secondary)">{{ Math.floor(dedupRule.dedupWindow / 60) }}分钟</span>
          </a-form-item>
        </a-form>
      </a-collapse-item>

      <!-- 分组规则 -->
      <a-collapse-item key="group" header="分组规则">
        <a-form :model="groupRule" layout="vertical">
          <a-form-item label="启用分组">
            <a-switch v-model="groupRule.enabled" />
          </a-form-item>
          <a-form-item label="分组字段" v-if="groupRule.enabled">
            <a-checkbox-group v-model="groupByKeys">
              <a-checkbox value="severity">告警级别</a-checkbox>
              <a-checkbox value="ruleName">规则名称</a-checkbox>
              <a-checkbox value="instance">实例</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
          <a-form-item label="分组等待时间(秒)" v-if="groupRule.enabled">
            <a-input-number v-model="groupRule.groupWait" :min="10" :max="300" style="width:200px" />
          </a-form-item>
          <a-form-item label="分组发送间隔(秒)" v-if="groupRule.enabled">
            <a-input-number v-model="groupRule.groupInterval" :min="60" :max="3600" style="width:200px" />
          </a-form-item>
          <a-form-item label="单组最大告警数" v-if="groupRule.enabled">
            <a-input-number v-model="groupRule.maxGroupSize" :min="5" :max="100" style="width:200px" />
          </a-form-item>
        </a-form>
      </a-collapse-item>

      <!-- 抑制规则 -->
      <a-collapse-item key="inhibit" header="抑制规则">
        <a-form :model="inhibitRule" layout="vertical">
          <a-form-item label="启用抑制">
            <a-switch v-model="inhibitRule.enabled" />
          </a-form-item>
          <div v-if="inhibitRule.enabled">
            <a-form-item label="源告警匹配条件（JSON）">
              <a-textarea v-model="inhibitRule.sourceMatchers" :rows="3" placeholder='{"severity": "critical", "ruleName": "节点宕机"}' />
            </a-form-item>
            <a-form-item label="目标告警匹配条件（JSON）">
              <a-textarea v-model="inhibitRule.targetMatchers" :rows="3" placeholder='{"severity": "warning", "ruleName": "服务不可用"}' />
            </a-form-item>
            <a-form-item label="相等标签（JSON数组）">
              <a-input v-model="inhibitRule.equalLabels" placeholder='["instance", "cluster"]' />
            </a-form-item>
          </div>
        </a-form>
      </a-collapse-item>
    </a-collapse>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { DedupRule, GroupRule, InhibitRule } from '@/api/alert-governance'

const props = defineProps<{
  subscriptionId: number
  dedupData?: DedupRule
  groupData?: GroupRule
  inhibitData?: InhibitRule
}>()

const emit = defineEmits<{
  (e: 'update', data: { dedup: DedupRule | null, group: GroupRule | null, inhibit: InhibitRule | null }): void
}>()

// 去重规则
const dedupRule = ref<Partial<DedupRule>>({
  enabled: false,
  dedupWindow: 600,
  name: '去重规则',
  subscriptionId: props.subscriptionId
})

const fingerprintKeys = ref<string[]>(['severity', 'ruleName', 'instance'])

// 分组规则
const groupRule = ref<Partial<GroupRule>>({
  enabled: false,
  groupWait: 30,
  groupInterval: 300,
  maxGroupSize: 20,
  name: '分组规则',
  subscriptionId: props.subscriptionId
})

const groupByKeys = ref<string[]>(['severity', 'ruleName'])

// 抑制规则
const inhibitRule = ref<Partial<InhibitRule>>({
  enabled: false,
  sourceMatchers: '{"severity": "critical"}',
  targetMatchers: '{"severity": "warning"}',
  equalLabels: '["instance"]',
  name: '抑制规则',
  subscriptionId: props.subscriptionId
})

// 监听 props 变化，更新本地状态
watch(() => props.dedupData, (data) => {
  if (data) {
    dedupRule.value.enabled = data.enabled === true
    dedupRule.value.dedupWindow = data.dedupWindow || 600
    if (data.fingerprintKeys) {
      try {
        fingerprintKeys.value = JSON.parse(data.fingerprintKeys)
      } catch {}
    }
  } else {
    dedupRule.value.enabled = false
  }
}, { immediate: true, deep: true })

watch(() => props.groupData, (data) => {
  if (data) {
    groupRule.value.enabled = data.enabled === true
    groupRule.value.groupWait = data.groupWait || 30
    groupRule.value.groupInterval = data.groupInterval || 300
    groupRule.value.maxGroupSize = data.maxGroupSize || 20
    if (data.groupBy) {
      try {
        groupByKeys.value = JSON.parse(data.groupBy)
      } catch {}
    }
  } else {
    groupRule.value.enabled = false
  }
}, { immediate: true, deep: true })

watch(() => props.inhibitData, (data) => {
  if (data) {
    inhibitRule.value.enabled = data.enabled === true
    inhibitRule.value.sourceMatchers = data.sourceMatchers || '{}'
    inhibitRule.value.targetMatchers = data.targetMatchers || '{}'
    inhibitRule.value.equalLabels = data.equalLabels || '[]'
  } else {
    inhibitRule.value.enabled = false
  }
}, { immediate: true, deep: true })

// 监听变化并向父组件发送更新
watch([dedupRule, fingerprintKeys, groupRule, groupByKeys, inhibitRule], () => {
  const dedupData = dedupRule.value.enabled ? {
    ...dedupRule.value,
    fingerprintKeys: JSON.stringify(fingerprintKeys.value),
    subscriptionId: props.subscriptionId,
    name: '去重规则'
  } as DedupRule : null

  const groupData = groupRule.value.enabled ? {
    ...groupRule.value,
    groupBy: JSON.stringify(groupByKeys.value),
    subscriptionId: props.subscriptionId,
    name: '分组规则'
  } as GroupRule : null

  const inhibitData = inhibitRule.value.enabled ? {
    ...inhibitRule.value,
    subscriptionId: props.subscriptionId,
    name: '抑制规则'
  } as InhibitRule : null

  emit('update', { dedup: dedupData, group: groupData, inhibit: inhibitData })
}, { deep: true })
</script>

<style scoped>
.governance-config {
  padding: 12px 0;
}
</style>
