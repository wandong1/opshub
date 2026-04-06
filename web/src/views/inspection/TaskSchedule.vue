<template>
  <div class="task-schedule-container">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-schedule /></div>
        <div>
          <h2 class="page-title">任务调度</h2>
          <p class="page-subtitle">管理拨测调度任务，支持 Cron 定时执行与结果推送</p>
        </div>
      </div>
    </div>

    <div class="filter-bar">
      <a-input v-model="searchForm.keyword" placeholder="搜索任务名称" allow-clear style="width: 220px;" @press-enter="loadData" />
      <a-select v-model="searchForm.taskType" placeholder="任务类型" allow-clear style="width: 120px;">
        <a-option label="拨测任务" value="probe" />
        <a-option label="巡检任务" value="inspection" />
      </a-select>
      <a-select v-model="searchForm.status" placeholder="状态" allow-clear style="width: 120px;">
        <a-option label="启用" :value="1" />
        <a-option label="禁用" :value="0" />
      </a-select>
      <a-button type="primary" @click="loadData"><template #icon><icon-search /></template>搜索</a-button>
      <a-button @click="handleReset"><template #icon><icon-refresh /></template>重置</a-button>
      <div style="flex: 1;" />
      <a-button v-permission="'inspection:tasks:create'" type="primary" @click="handleCreate"><template #icon><icon-plus /></template>新增任务</a-button>
    </div>

    <a-table :data="taskList" :loading="loading" :bordered="{ cell: true }" stripe :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50] }" @page-change="(p: number) => { pagination.page = p; loadData() }" @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }">
      <template #columns>
        <a-table-column title="任务名称" data-index="name" :width="140" />
        <a-table-column title="任务类型" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.taskType === 'inspection' ? 'purple' : 'blue'">
              {{ record.taskType === 'inspection' ? '巡检' : '拨测' }}
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="关联配置" :width="140">
          <template #cell="{ record }">
            <a-tooltip v-if="record.taskType === 'probe' && record.probeConfigIds?.length" position="top">
              <template #content><div v-for="id in record.probeConfigIds" :key="id">{{ getProbeLabel(id) }}</div></template>
              <a-tag size="small">{{ record.probeConfigIds.length }} 个拨测</a-tag>
            </a-tooltip>
            <a-tooltip v-else-if="record.taskType === 'inspection' && record.inspectionGroupIds?.length" position="top">
              <template #content><div v-for="id in record.inspectionGroupIds" :key="id">{{ getInspectionGroupLabel(id) }}</div></template>
              <a-tag size="small" color="purple">{{ record.inspectionGroupIds.length }} 个巡检组</a-tag>
            </a-tooltip>
            <span v-else>-</span>
          </template>
        </a-table-column>
        <a-table-column title="并发数" data-index="concurrency" :width="80" align="center" />
        <a-table-column title="Cron表达式" data-index="cronExpr" :width="160" ellipsis tooltip />
        <a-table-column title="Pushgateway" :width="140">
          <template #cell="{ record }">{{ getPgwLabel(record.pushgatewayId) }}</template>
        </a-table-column>
        <a-table-column title="状态" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.enabled ? 'green' : 'red'">{{ record.enabled ? '启用' : '禁用' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="最后执行" :width="170">
          <template #cell="{ record }">{{ record.lastRunAt || '-' }}</template>
        </a-table-column>
        <a-table-column title="结果" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag v-if="record.lastResult" size="small" :color="record.lastResult === 'success' ? 'green' : 'red'">{{ record.lastResult }}</a-tag>
            <span v-else>-</span>
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="420" fixed="right" align="center">
          <template #cell="{ record }">
            <a-tooltip content="启用/禁用">
              <a-button v-permission="'inspection:tasks:toggle'" type="text" size="small" :status="record.enabled ? 'warning' : 'normal'" @click="handleToggle(record)">
                <template #icon><icon-poweroff /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="编辑">
              <a-button v-permission="'inspection:tasks:update'" type="text" size="small" @click="handleEdit(record)">
                <template #icon><icon-edit /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="复制">
              <a-button v-permission="'inspection:tasks:create'" type="text" size="small" @click="handleCopy(record)">
                <template #icon><icon-copy /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="删除">
              <a-button v-permission="'inspection:tasks:delete'" type="text" size="small" status="danger" @click="handleDelete(record)">
                <template #icon><icon-delete /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip content="查看结果">
              <a-button v-permission="'inspection:tasks:results'" type="text" size="small" @click="handleViewResults(record)">
                <template #icon><icon-eye /></template>
              </a-button>
            </a-tooltip>
            <template v-if="runningTaskIds.has(record.id)">
              <a-tooltip content="终止运行">
                <a-button v-permission="'inspection:tasks:run'" type="text" size="small" status="danger" @click="handleStop(record)">
                  <template #icon><icon-stop /></template>
                  <span class="running-text">运行中</span>
                </a-button>
              </a-tooltip>
            </template>
            <template v-else>
              <a-tooltip content="立即运行一次">
                <a-button v-permission="'inspection:tasks:run'" type="text" size="small" status="success" @click="handleRun(record)">
                  <template #icon><icon-play-arrow /></template>
                  立即运行
                </a-button>
              </a-tooltip>
            </template>
          </template>
        </a-table-column>
      </template>
    </a-table>
    <!-- 新建/编辑对话框 -->
    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑任务' : '新增任务'" :width="720" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="任务名称" field="name">
          <a-input v-model="formData.name" placeholder="请输入任务名称" />
        </a-form-item>

        <!-- 任务类型选择 -->
        <a-form-item label="任务类型" field="taskType">
          <a-radio-group v-model="formData.taskType" type="button" @change="handleTaskTypeChange">
            <a-radio value="probe">拨测任务</a-radio>
            <a-radio value="inspection">巡检任务</a-radio>
          </a-radio-group>
        </a-form-item>

        <!-- 拨测任务配置 -->
        <template v-if="formData.taskType === 'probe'">
          <a-form-item label="拨测分类">
            <a-radio-group v-model="selectedCategory" type="button" @change="handleTaskCategoryChange">
              <a-radio v-for="c in PROBE_CATEGORIES" :key="c.value" :value="c.value" :disabled="!c.enabled">{{ c.label }}</a-radio>
            </a-radio-group>
          </a-form-item>
          <a-form-item label="拨测配置" field="probeConfigIds">
            <a-select v-model="formData.probeConfigIds" multiple allow-search placeholder="选择拨测配置（可多选）" style="width: 100%;">
              <a-option v-for="p in filteredProbeOptions" :key="p.id" :label="p.name" :value="p.id">
                {{ p.name }} <span style="float: right; color: var(--ops-text-tertiary); font-size: 12px;">{{ p.type?.toUpperCase() }} · {{ p.target }}</span>
              </a-option>
            </a-select>
          </a-form-item>
        </template>

        <!-- 巡检任务配置 -->
        <template v-if="formData.taskType === 'inspection'">
          <a-form-item label="巡检组" field="inspectionGroupIds">
            <a-select v-model="formData.inspectionGroupIds" multiple allow-search placeholder="选择巡检组（可多选）" style="width: 100%;" @change="handleInspectionGroupChange">
              <a-option v-for="g in inspectionGroupOptions" :key="g.id" :label="g.name" :value="g.id">
                {{ g.name }} <span style="float: right; color: var(--ops-text-tertiary); font-size: 12px;">{{ g.itemCount || 0 }} 个巡检项</span>
              </a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="指定巡检项" v-if="formData.inspectionGroupIds.length > 0">
            <a-select v-model="formData.inspectionItemIds" multiple allow-search placeholder="不选则执行所有巡检项" style="width: 100%;">
              <a-option v-for="item in filteredInspectionItems" :key="item.id" :label="item.name" :value="item.id">
                {{ item.name }} <span style="float: right; color: var(--ops-text-tertiary); font-size: 12px;">{{ getInspectionGroupLabel(item.groupId) }}</span>
              </a-option>
            </a-select>
          </a-form-item>
        </template>

        <a-form-item label="执行计划" field="cronExpr">
          <div class="cron-picker">
            <div class="cron-presets">
              <a-button v-for="preset in cronPresets" :key="preset.value" size="small"
                :type="formData.cronExpr === preset.value ? 'primary' : 'secondary'"
                @click="formData.cronExpr = preset.value">{{ preset.label }}</a-button>
            </div>
            <a-input v-model="formData.cronExpr" placeholder="秒级cron，如: 0/30 * * * * ?" style="margin-top: 8px;">
              <template #prepend>Cron</template>
            </a-input>
            <div class="cron-description">{{ cronDescription }}</div>
          </div>
        </a-form-item>
        <a-form-item label="并发数" v-if="formData.taskType === 'probe'">
          <a-input-number v-model="formData.concurrency" :min="1" :max="50" />
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">同时执行的最大拨测数</span>
        </a-form-item>
        <a-form-item label="Pushgateway">
          <a-select v-model="formData.pushgatewayId" placeholder="选择Pushgateway（可选）" allow-clear style="width: 100%;">
            <a-option v-for="p in pgwOptions" :key="p.id" :label="p.name" :value="p.id" />
          </a-select>
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">
            {{ formData.taskType === 'inspection' ? '推送巡检指标到 Prometheus' : '推送拨测指标到 Prometheus' }}
          </span>
        </a-form-item>
        <!-- 需求一：执行方式覆盖 -->
        <a-form-item label="执行方式">
          <a-select v-model="formData.executionMode" placeholder="不设置则沿用原配置" allow-clear style="width: 220px;">
            <template v-if="formData.taskType === 'probe'">
              <a-option label="本地执行" value="local" />
              <a-option label="Agent 执行" value="agent" />
            </template>
            <template v-else>
              <a-option label="自动（优先 Agent）" value="auto" />
              <a-option label="仅 Agent" value="agent" />
              <a-option label="仅 SSH" value="ssh" />
            </template>
          </a-select>
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">
            {{ formData.executionMode ? '覆盖原配置的执行方式' : '不设置则沿用原配置' }}
          </span>
        </a-form-item>

        <!-- 需求一：Agent 主机选择（仅执行方式为 agent 时显示）-->
        <a-form-item v-if="formData.executionMode === 'agent'" label="Agent 主机">
          <a-select v-model="formData.agentHostIds" multiple allow-search placeholder="选择 Agent 主机" style="width: 100%;">
            <a-option v-for="h in agentHostOptions" :key="h.id" :label="h.name" :value="h.id">
              {{ h.name }} <span style="float: right; color: var(--ops-text-tertiary); font-size: 12px;">{{ h.ip }}</span>
            </a-option>
          </a-select>
        </a-form-item>

        <a-form-item label="业务分组">
          <a-select v-model="formData.businessGroupId" placeholder="不设置则沿用原配置" allow-clear allow-search style="width: 100%;">
            <a-option :value="0" label="不覆盖" />
            <a-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </a-select>
        </a-form-item>

        <!-- 需求三/四：自定义变量（拨测和巡检任务均支持，优先级最高）-->
        <a-divider style="margin: 8px 0; font-size: 13px;">调度变量（优先级最高，覆盖所有同名变量）</a-divider>
        <a-form-item label="自定义变量">
          <div style="width: 100%;">
            <div v-for="(item, index) in customVariablesList" :key="index"
              style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;">
              <a-input v-model="item.key" placeholder="变量名（如：api_host）" style="flex: 1;" />
              <a-input v-model="item.value" placeholder="变量值" style="flex: 2;" />
              <a-button type="text" status="danger" size="small" @click="removeCustomVariable(index)">
                <template #icon><icon-delete /></template>
              </a-button>
            </div>
            <a-button type="dashed" long size="small" @click="addCustomVariable">
              <template #icon><icon-plus /></template>
              添加变量
            </a-button>
          </div>
          <template #extra>
            <span style="color: var(--ops-text-tertiary); font-size: 12px;">
              {{ formData.taskType === 'probe' ? '优先级：调度变量 > 巡检组环境变量 > 拨测流程变量' : '优先级：调度变量 > 巡检组自定义变量 > 全局变量' }}
            </span>
          </template>
        </a-form-item>
        <a-form-item label="负责人">
          <a-input v-model="formData.owner" placeholder="请输入负责人（可选，用于 Metric 标签）" allow-clear />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model="formData.description" :max-length="200" :auto-size="{ minRows: 2 }" />
        </a-form-item>
        <a-form-item label="状态">
          <a-radio-group v-model="formData.enabled"><a-radio :value="true">启用</a-radio><a-radio :value="false">禁用</a-radio></a-radio-group>
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="dialogVisible = false">取消</a-button>
        <a-button type="primary" :loading="submitting" @click="handleSubmit">确定</a-button>
      </template>
    </a-modal>

    <!-- 需求二：立即运行同步结果抽屉 -->
    <a-drawer v-model:visible="syncResultVisible" title="立即运行结果" :width="1260" unmount-on-close>
      <div v-if="syncRunning" style="text-align:center;padding:60px 0;">
        <a-spin size="large" />
        <p style="margin-top:16px;color:var(--ops-text-secondary);">正在执行中，请稍候...</p>
      </div>
      <template v-else-if="syncResultData">
        <!-- 汇总 -->
        <a-descriptions :column="4" bordered size="small" style="margin-bottom:16px;">
          <a-descriptions-item label="任务名称">{{ syncResultData.task_name }}</a-descriptions-item>
          <a-descriptions-item label="类型">
            <a-tag size="small" :color="syncResultData.task_type === 'inspection' ? 'purple' : 'blue'">{{ syncResultData.task_type === 'inspection' ? '巡检' : '拨测' }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="总状态">
            <a-tag size="small" :color="syncResultData.status === 'success' ? 'green' : syncResultData.status === 'partial' ? 'orange' : 'red'">{{ syncResultData.status }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="耗时">{{ syncResultData.duration?.toFixed(2) }} 秒</a-descriptions-item>
          <a-descriptions-item label="总项数">{{ syncResultData.total_items }}</a-descriptions-item>
          <a-descriptions-item label="成功"><span style="color:var(--ops-success)">{{ syncResultData.success_count }}</span></a-descriptions-item>
          <a-descriptions-item label="失败"><span style="color:var(--ops-danger)">{{ syncResultData.failed_count }}</span></a-descriptions-item>
        </a-descriptions>

        <!-- 巡检详情 -->
        <template v-if="syncResultData.task_type === 'inspection'">
          <a-table :data="syncResultData.details || []" :bordered="{ cell: true }" stripe size="small"
            :pagination="{ pageSize: 20, showTotal: true }">
            <template #columns>
              <a-table-column title="巡检组" data-index="group_name" :width="120" />
              <a-table-column title="巡检项" data-index="item_name" :width="140" />
              <a-table-column title="主机" :width="140">
                <template #cell="{ record }">
                  <span v-if="record.host_name">{{ record.host_name }}<br/><span style="font-size:11px;color:var(--ops-text-tertiary)">{{ record.host_ip }}</span></span>
                  <span v-else style="color:var(--ops-text-tertiary)">-</span>
                </template>
              </a-table-column>
              <a-table-column title="执行类型" :width="100" align="center">
                <template #cell="{ record }">
                  <a-tag v-if="record.execution_type === 'command'" size="small" color="blue">命令</a-tag>
                  <a-tag v-else-if="record.execution_type === 'script'" size="small" color="purple">脚本</a-tag>
                  <a-tag v-else-if="record.execution_type === 'probe'" size="small" color="cyan">拨测</a-tag>
                  <a-tag v-else-if="record.execution_type === 'promql'" size="small" color="orange">PromQL</a-tag>
                  <span v-else>-</span>
                </template>
              </a-table-column>
              <a-table-column title="状态" :width="80" align="center">
                <template #cell="{ record }">
                  <a-tag size="small" :color="record.status === 'success' ? 'green' : 'red'">{{ record.status === 'success' ? '成功' : '失败' }}</a-tag>
                </template>
              </a-table-column>
              <a-table-column title="断言" :width="80" align="center">
                <template #cell="{ record }">
                  <a-tag v-if="record.assertion_result" size="small" :color="record.assertion_result === 'pass' ? 'green' : record.assertion_result === 'fail' ? 'red' : 'gray'">{{ record.assertion_result }}</a-tag>
                  <span v-else>-</span>
                </template>
              </a-table-column>
              <a-table-column title="耗时(s)" :width="90" align="center">
                <template #cell="{ record }">{{ record.duration?.toFixed(3) }}</template>
              </a-table-column>
              <a-table-column title="详情">
                <template #cell="{ record }">
                  <a-collapse :bordered="false" size="small">
                    <a-collapse-item header="查看详细信息" key="detail">
                      <div style="font-size: 12px;">
                        <!-- 执行配置 -->
                        <a-descriptions :column="1" size="small" bordered style="margin-bottom: 8px;">
                          <a-descriptions-item v-if="record.command" label="执行命令">
                            <pre class="code-block-inline">{{ record.command }}</pre>
                          </a-descriptions-item>
                          <a-descriptions-item v-if="record.script_type && record.script_content" label="脚本类型">
                            {{ record.script_type }}
                          </a-descriptions-item>
                          <a-descriptions-item v-if="record.script_content" label="脚本内容">
                            <pre class="code-block">{{ record.script_content }}</pre>
                          </a-descriptions-item>
                          <a-descriptions-item v-if="record.assertion_type" label="断言类型">
                            {{ record.assertion_type }}
                          </a-descriptions-item>
                          <a-descriptions-item v-if="record.assertion_value" label="断言表达式">
                            <pre class="code-block-inline">{{ record.assertion_value }}</pre>
                          </a-descriptions-item>
                          <a-descriptions-item v-if="record.output" label="执行输出">
                            <pre class="code-block">{{ record.output }}</pre>
                          </a-descriptions-item>
                          <a-descriptions-item v-if="record.error_message" label="错误信息">
                            <span style="color: var(--ops-danger);">{{ record.error_message }}</span>
                          </a-descriptions-item>
                        </a-descriptions>
                      </div>
                    </a-collapse-item>
                  </a-collapse>
                </template>
              </a-table-column>
            </template>
          </a-table>
        </template>

        <!-- 拨测详情 -->
        <template v-else>
          <a-table :data="syncResultData.details || []" :bordered="{ cell: true }" stripe size="small"
            :pagination="{ pageSize: 20, showTotal: true }">
            <template #columns>
              <a-table-column title="配置名称" data-index="config_name" :width="140" />
              <a-table-column title="类型" :width="120" align="center">
                <template #cell="{ record }"><a-tag size="small" color="arcoblue">{{ record.config_type?.toUpperCase() }}</a-tag></template>
              </a-table-column>
              <a-table-column title="目标" data-index="target" :width="160" ellipsis tooltip />
              <a-table-column title="结果" :width="80" align="center">
                <template #cell="{ record }">
                  <a-tag size="small" :color="record.success ? 'green' : 'red'">{{ record.success ? '成功' : '失败' }}</a-tag>
                </template>
              </a-table-column>
              <a-table-column title="延迟(ms)" :width="100" align="center">
                <template #cell="{ record }">{{ record.latency_ms?.toFixed(2) }}</template>
              </a-table-column>
              <a-table-column title="执行方式" :width="100" align="center">
                <template #cell="{ record }">
                  <a-tag v-if="record.agent_host_id" size="small" color="purple">Agent #{{ record.agent_host_id }}</a-tag>
                  <span v-else>本地</span>
                </template>
              </a-table-column>
              <a-table-column title="详情" ellipsis tooltip>
                <template #cell="{ record }">
                  <div v-if="record.error_message" style="color: var(--ops-danger); font-size: 12px; margin-bottom: 4px;">{{ record.error_message }}</div>

                  <!-- 业务流程详情展示 -->
                  <div v-if="record.output" style="margin-top: 4px;">
                    <a-collapse :bordered="false" size="small">
                      <a-collapse-item :header="`查看流程步骤 (${JSON.parse(record.output).length})`" key="steps">
                        <div v-for="(step, idx) in JSON.parse(record.output)" :key="idx" class="step-detail-item">
                          <div class="step-header">
                            <a-tag size="small" :color="step.success ? 'green' : (step.skipped ? 'gray' : 'red')">
                              {{ step.success ? 'PASS' : (step.skipped ? 'SKIP' : 'FAIL') }}
                            </a-tag>
                            <span class="step-name">Step {{ idx + 1 }}: {{ step.stepName }}</span>
                            <span class="step-latency">{{ step.latency?.toFixed(2) }}ms</span>
                          </div>
                          <div class="step-info">
                            <div v-if="step.url" class="info-line"><strong>URL:</strong> {{ step.url }}</div>
                            <div v-if="step.method" class="info-line"><strong>Method:</strong> {{ step.method }}</div>
                            <div v-if="step.httpStatusCode" class="info-line"><strong>Status Code:</strong> {{ step.httpStatusCode }}</div>
                            <div v-if="step.error" class="info-line error"><strong>Error:</strong> {{ step.error }}</div>

                            <!-- 请求详情 -->
                            <a-descriptions :column="1" size="small" bordered style="margin-top: 8px;">
                              <a-descriptions-item v-if="step.requestHeaders && Object.keys(step.requestHeaders).length > 0" label="请求头">
                                <pre class="code-block">{{ JSON.stringify(step.requestHeaders, null, 2) }}</pre>
                              </a-descriptions-item>
                              <a-descriptions-item v-if="step.requestParams && Object.keys(step.requestParams).length > 0" label="请求参数">
                                <pre class="code-block">{{ JSON.stringify(step.requestParams, null, 2) }}</pre>
                              </a-descriptions-item>
                              <a-descriptions-item v-if="step.requestBody" label="请求体">
                                <pre class="code-block">{{ step.requestBody }}</pre>
                              </a-descriptions-item>
                              // 断言表达式
                              <a-descriptions-item v-if="step.assertions && step.assertions.length > 0" label="断言">
                                <pre class="code-block">{{ JSON.stringify(step.assertions, null, 2) }}</pre>
                              </a-descriptions-item>
                              <a-descriptions-item v-if="step.responseHeaders && Object.keys(step.responseHeaders).length > 0" label="响应头">
                                <pre class="code-block">{{ JSON.stringify(step.responseHeaders, null, 2) }}</pre>
                              </a-descriptions-item>
                              <a-descriptions-item v-if="step.responseBody" label="响应体">
                                <pre class="code-block">{{ step.responseBody }}</pre>
                              </a-descriptions-item>
                              <a-descriptions-item v-if="step.extractedVars && Object.keys(step.extractedVars).length > 0" label="提取变量">
                                <pre class="code-block">{{ JSON.stringify(step.extractedVars, null, 2) }}</pre>
                              </a-descriptions-item>
                            </a-descriptions>

                            <!-- 断言结果 -->
                            <div v-if="step.assertionResults && step.assertionResults.length > 0" style="margin-top: 8px;">
                              <div style="font-weight: 500; margin-bottom: 4px;">断言结果:</div>
                              <div v-for="(ast, astIdx) in step.assertionResults" :key="astIdx" class="assertion-item">
                                <icon-check-circle-fill v-if="ast.success" style="color: var(--ops-success); font-size: 14px;" />
                                <icon-exclamation-circle-fill v-else style="color: var(--ops-danger); font-size: 14px;" />
                                <span class="ast-text">
                                  {{ ast.name }}:
                                  <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                                    ({{ ast.source }}.{{ ast.path }} {{ ast.condition }} "{{ ast.value }}")
                                  </span>
                                  → {{ ast.actual }}
                                </span>
                                <span v-if="ast.error" class="ast-error"> ({{ ast.error }})</span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </a-collapse-item>
                    </a-collapse>
                  </div>

                  <!-- 断言详情展示（非业务流程） -->
                  <div v-if="record.assertion_detail && record.assertion_detail !== '[]'" style="margin-top: 4px;">
                    <a-tag size="small" :color="record.assertion_success ? 'green' : 'red'" style="margin-bottom: 4px;">
                      断言: {{ record.assertion_success ? '通过' : '不通过' }}
                    </a-tag>
                    <div v-for="(ast, idx) in JSON.parse(record.assertion_detail)" :key="idx" class="assertion-item">
                      <icon-check-circle-fill v-if="ast.success" style="color: var(--ops-success); font-size: 14px;" />
                      <icon-exclamation-circle-fill v-else style="color: var(--ops-danger); font-size: 14px;" />
                      <span class="ast-text">{{ ast.name }}: {{ ast.actual }}</span>
                    </div>
                  </div>
                </template>
              </a-table-column>
            </template>
          </a-table>
        </template>
      </template>
    </a-drawer>

    <!-- 执行结果抽屉 -->
    <a-drawer v-model:visible="resultsVisible" title="执行结果" :width="720" unmount-on-close>
      <!-- 拨测任务结果 -->
      <a-table v-if="currentTaskType === 'probe'" :data="resultList" :loading="resultsLoading" :bordered="{ cell: true }" stripe :pagination="{ current: resultPagination.page, pageSize: resultPagination.pageSize, total: resultPagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [20, 50] }" @page-change="(p: number) => { resultPagination.page = p; loadResults() }" @page-size-change="(s: number) => { resultPagination.pageSize = s; resultPagination.page = 1; loadResults() }">
        <template #columns>
          <a-table-column title="时间" data-index="createdAt" :width="170" />
          <a-table-column title="触发方式" :width="90" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.triggerType === 'manual' ? 'orange' : 'arcoblue'">{{ record.triggerType === 'manual' ? '手动' : '调度' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="成功" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.success ? 'green' : 'red'">{{ record.success ? '是' : '否' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="延迟(ms)" :width="100" align="center">
            <template #cell="{ record }">{{ record.latency?.toFixed(2) }}</template>
          </a-table-column>
          <a-table-column title="丢包率" :width="90" align="center">
            <template #cell="{ record }">{{ record.packetLoss !== undefined ? (record.packetLoss * 100).toFixed(1) + '%' : '-' }}</template>
          </a-table-column>
          <a-table-column title="执行方式" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.agentHostId > 0" size="small" color="purple">Agent #{{ record.agentHostId }}</a-tag>
              <span v-else>本地</span>
            </template>
          </a-table-column>
          <a-table-column title="重试" :width="70" align="center">
            <template #cell="{ record }">{{ record.retryAttempt > 0 ? record.retryAttempt : '-' }}</template>
          </a-table-column>
          <a-table-column title="错误信息" data-index="errorMessage" ellipsis tooltip />
        </template>
      </a-table>

      <!-- 巡检任务结果 -->
      <a-table v-else :data="resultList" :loading="resultsLoading" :bordered="{ cell: true }" stripe :pagination="{ current: resultPagination.page, pageSize: resultPagination.pageSize, total: resultPagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [20, 50] }" @page-change="(p: number) => { resultPagination.page = p; loadResults() }" @page-size-change="(s: number) => { resultPagination.pageSize = s; resultPagination.page = 1; loadResults() }">
        <template #columns>
          <a-table-column title="执行时间" data-index="executed_at" :width="170" />
          <a-table-column title="触发方式" :width="90" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.trigger_type === 'manual' ? 'orange' : 'arcoblue'">{{ record.trigger_type === 'manual' ? '手动' : '调度' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="巡检项" data-index="item_name" :width="150" ellipsis tooltip />
          <a-table-column title="主机" data-index="host_name" :width="120" ellipsis tooltip />
          <a-table-column title="状态" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.status === 'success' ? 'green' : 'red'">{{ record.status === 'success' ? '成功' : '失败' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="耗时(ms)" :width="100" align="center" data-index="duration" />
          <a-table-column title="断言结果" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.assertion_result" size="small" :color="record.assertion_result === 'pass' ? 'green' : 'red'">{{ record.assertion_result }}</a-tag>
              <span v-else>-</span>
            </template>
          </a-table-column>
          <a-table-column title="错误信息" data-index="error_message" ellipsis tooltip />
        </template>
      </a-table>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconSchedule, IconSearch, IconRefresh, IconPlus, IconEdit, IconDelete, IconPoweroff, IconPlayArrow, IconStop, IconEye, IconCheckCircleFill, IconExclamationCircleFill } from '@arco-design/web-vue/es/icon'
import { getTaskList, createTask, updateTask, deleteTask, toggleTask, getTaskResults, getProbeList, getPushgatewayList, PROBE_CATEGORIES, CATEGORY_LABEL_MAP } from '@/api/networkProbe'
import { getAllInspectionGroups, getInspectionItems } from '@/api/inspectionManagement'
import { getInspectionTasks, createInspectionTask, updateInspectionTask, deleteInspectionTask, toggleInspectionTask, getInspectionTaskResults, runInspectionTask, stopInspectionTask, runInspectionTaskSync } from '@/api/inspectionTask'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'
import { getAgentStatuses } from '@/api/agent'
import { useRouter } from 'vue-router'

const router = useRouter()

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const taskList = ref<any[]>([])
const probeOptions = ref<any[]>([])
const pgwOptions = ref<any[]>([])
const groupOptions = ref<any[]>([])
const inspectionGroupOptions = ref<any[]>([])
const inspectionItemOptions = ref<any[]>([])
const agentHostOptions = ref<any[]>([])
// 需求四：自定义变量键值对列表
const customVariablesList = ref<Array<{ key: string; value: string }>>([])

const addCustomVariable = () => {
  customVariablesList.value.push({ key: '', value: '' })
}
const removeCustomVariable = (index: number) => {
  customVariablesList.value.splice(index, 1)
}
// 将键值对列表序列化为 JSON 字符串
const serializeCustomVariables = (): string => {
  const obj: Record<string, string> = {}
  for (const item of customVariablesList.value) {
    if (item.key.trim()) obj[item.key.trim()] = item.value
  }
  return Object.keys(obj).length > 0 ? JSON.stringify(obj) : ''
}
// 将 JSON 字符串反序列化为键值对列表
const deserializeCustomVariables = (jsonStr: string) => {
  customVariablesList.value = []
  if (!jsonStr) return
  try {
    const obj = typeof jsonStr === 'string' ? JSON.parse(jsonStr) : jsonStr
    for (const [key, value] of Object.entries(obj)) {
      customVariablesList.value.push({ key, value: String(value) })
    }
  } catch {}
}
const resultsVisible = ref(false)
const resultsLoading = ref(false)
const resultList = ref<any[]>([])
const currentTaskId = ref(0)
const selectedCategory = ref('network')
const runningTaskIds = ref<Set<number>>(new Set())

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const resultPagination = reactive({ page: 1, pageSize: 20, total: 0 })
const searchForm = reactive({ keyword: '', taskType: undefined as string | undefined, status: undefined as number | undefined })

const cronPresets = [
  { label: '每30秒', value: '0/30 * * * * ?' },
  { label: '每分钟', value: '0 * * * * ?' },
  { label: '每5分钟', value: '0 0/5 * * * ?' },
  { label: '每15分钟', value: '0 0/15 * * * ?' },
  { label: '每小时', value: '0 0 * * * ?' },
  { label: '每天0点', value: '0 0 0 * * ?' },
]

const cronDescriptionMap: Record<string, string> = {
  '0/30 * * * * ?': '每30秒执行一次',
  '0 * * * * ?': '每分钟执行一次',
  '0 0/5 * * * ?': '每5分钟执行一次',
  '0 0/15 * * * ?': '每15分钟执行一次',
  '0 0 * * * ?': '每小时执行一次',
  '0 0 0 * * ?': '每天凌晨0点执行一次',
}

const cronDescription = computed(() => cronDescriptionMap[formData.cronExpr] || (formData.cronExpr ? '自定义 Cron 表达式' : ''))

const defaultForm = () => ({
  id: 0, name: '', taskType: 'probe' as 'probe' | 'inspection',
  probeConfigIds: [] as number[],
  inspectionGroupIds: [] as number[],
  inspectionItemIds: [] as number[],
  groupId: 0, cronExpr: '',
  pushgatewayId: undefined as number | undefined, concurrency: 5, enabled: true, description: '', owner: '',
  // 需求一新增字段
  executionMode: '' as string,      // 执行方式覆盖：拨测=local/agent，巡检=auto/agent/ssh，空=不覆盖
  agentHostIds: [] as number[],      // Agent 主机 ID（executionMode=agent 时）
  businessGroupId: 0 as number,      // 业务分组 ID 覆盖
  customVariables: '' as string      // 自定义变量 JSON（序列化后存储，由 customVariablesList 驱动）
})
const formData = reactive(defaultForm())

const formRules = computed(() => {
  const rules: any = {
    name: [{ required: true, message: '请输入任务名称' }],
    taskType: [{ required: true, message: '请选择任务类型' }],
    cronExpr: [{ required: true, message: '请输入或选择Cron表达式' }],
  }

  if (formData.taskType === 'probe') {
    rules.probeConfigIds = [{ required: true, type: 'array' as const, min: 1, message: '请选择至少一个拨测配置' }]
  } else if (formData.taskType === 'inspection') {
    rules.inspectionGroupIds = [{ required: true, type: 'array' as const, min: 1, message: '请选择至少一个巡检组' }]
  }

  return rules
})

const filteredProbeOptions = computed(() => {
  return probeOptions.value.filter((p: any) => {
    if (!selectedCategory.value) return true
    return p.category === selectedCategory.value
  })
})

const filteredInspectionItems = computed(() => {
  if (formData.inspectionGroupIds.length === 0) return []
  return inspectionItemOptions.value.filter((item: any) =>
    formData.inspectionGroupIds.includes(item.groupId)
  )
})

const getProbeLabel = (id: number) => probeOptions.value.find((p: any) => p.id === id)?.name || id
const getPgwLabel = (id: number) => pgwOptions.value.find((p: any) => p.id === id)?.name || id || '-'
const getInspectionGroupLabel = (id: number) => inspectionGroupOptions.value.find((g: any) => g.id === id)?.name || id

const loadData = async () => {
  loading.value = true
  try {
    const res = await getInspectionTasks({
      page: pagination.page,
      page_size: pagination.pageSize,
      name: searchForm.keyword,
      enabled: searchForm.status !== undefined ? searchForm.status === 1 : undefined
    })

    // 转换数据格式以兼容前端显示
    taskList.value = (res.list || []).map((task: any) => {
      const converted: any = {
        id: task.id,
        name: task.name,
        description: task.description,
        taskType: task.task_type || 'probe',
        cronExpr: task.cron_expr,
        enabled: task.enabled,
        concurrency: task.concurrency || 5,
        pushgatewayId: task.pushgateway_id,
        owner: task.owner || '',
        groupId: 0,
        lastRunAt: task.last_run_at,
        lastResult: task.last_run_status,
        nextRunAt: task.next_run_at,
        // 需求一新增字段
        executionMode: task.execution_mode || '',
        agentHostIds: task.agent_host_ids || '',
        businessGroupId: task.business_group_id || 0,
        customVariables: task.custom_variables || ''
      }

      // 解析配置ID
      if (task.task_type === 'inspection') {
        try {
          converted.inspectionGroupIds = task.group_ids ? JSON.parse(task.group_ids) : []
          converted.inspectionItemIds = task.item_ids ? JSON.parse(task.item_ids) : []
        } catch (e) {
          converted.inspectionGroupIds = []
          converted.inspectionItemIds = []
        }
      } else {
        try {
          converted.probeConfigIds = task.item_ids ? JSON.parse(task.item_ids) : []
        } catch (e) {
          converted.probeConfigIds = []
        }
        try {
          const gids = task.group_ids ? JSON.parse(task.group_ids) : []
          converted.groupId = gids.length > 0 ? gids[0] : 0
        } catch (e) {
          converted.groupId = 0
        }
      }

      return converted
    })

    pagination.total = res.total || 0
  } catch {} finally { loading.value = false }
}

const flattenGroups = (tree: any[], result: any[] = []): any[] => {
  for (const node of tree) {
    result.push({ id: node.id, name: node.name })
    if (node.children?.length) flattenGroups(node.children, result)
  }
  return result
}

const loadOptions = async () => {
  try {
    const probeRes = await getProbeList({ page: 1, page_size: 1000, status: 1 })
    probeOptions.value = probeRes.data || []
  } catch {}
  try { pgwOptions.value = (await getPushgatewayList()) || [] } catch {}
  try {
    const res = await getGroupTree()
    groupOptions.value = flattenGroups(res.data || res || [])
  } catch {}
  try {
    inspectionGroupOptions.value = await getAllInspectionGroups()
  } catch {}
  // 加载 Agent 在线主机（用于执行方式=agent 时选择主机）
  try {
    const [hostRes, agentList] = await Promise.all([
      getHostList({ page: 1, page_size: 500 }),
      getAgentStatuses()
    ])
    const hosts = hostRes.data?.list || hostRes.list || []
    // getAgentStatuses 返回数组，字段为 hostId（驼峰）
    const agentMap: Record<number, any> = {}
    for (const a of (Array.isArray(agentList) ? agentList : [])) {
      agentMap[a.hostId] = a
    }
    agentHostOptions.value = hosts
      // .filter((h: any) => agentMap[h.id]?.status === 'online')
      .map((h: any) => ({ id: h.id, name: h.name, ip: h.ip }))
  } catch {}
}

const loadInspectionItems = async (groupIds: number[]) => {
  if (groupIds.length === 0) {
    inspectionItemOptions.value = []
    return
  }
  try {
    const allItems: any[] = []
    for (const groupId of groupIds) {
      const res = await getInspectionItems({ groupId, pageSize: 100 })
      allItems.push(...res.list)
    }
    inspectionItemOptions.value = allItems
  } catch {}
}

const handleTaskTypeChange = () => {
  formData.probeConfigIds = []
  formData.inspectionGroupIds = []
  formData.inspectionItemIds = []
}

const handleInspectionGroupChange = () => {
  loadInspectionItems(formData.inspectionGroupIds)
  formData.inspectionItemIds = formData.inspectionItemIds.filter(id =>
    filteredInspectionItems.value.some((item: any) => item.id === id)
  )
}

const handleTaskCategoryChange = () => {
  formData.probeConfigIds = formData.probeConfigIds.filter(id =>
    filteredProbeOptions.value.some((p: any) => p.id === id)
  )
}

const handleReset = () => { searchForm.keyword = ''; searchForm.taskType = undefined; searchForm.status = undefined; pagination.page = 1; loadData() }

const handleCreate = async () => {
  isEdit.value = false
  Object.assign(formData, defaultForm())
  customVariablesList.value = []
  selectedCategory.value = 'network'
  await loadOptions()
  dialogVisible.value = true
}

const handleEdit = async (row: any) => {
  isEdit.value = true
  Object.assign(formData, {
    id: row.id, name: row.name,
    taskType: row.taskType || 'probe',
    probeConfigIds: row.probeConfigIds || [],
    inspectionGroupIds: row.inspectionGroupIds || [],
    inspectionItemIds: row.inspectionItemIds || [],
    groupId: row.groupId, cronExpr: row.cronExpr, pushgatewayId: row.pushgatewayId,
    concurrency: row.concurrency || 5, enabled: row.enabled, description: row.description, owner: row.owner || '',
    // 需求一新增
    executionMode: row.executionMode || '',
    agentHostIds: (() => { try { return row.agentHostIds ? JSON.parse(row.agentHostIds) : [] } catch { return [] } })(),
    businessGroupId: row.businessGroupId || 0,
    customVariables: row.customVariables || ''
  })
  // 需求四：反序列化变量为键值对列表
  deserializeCustomVariables(row.customVariables || '')
  await loadOptions()

  if (formData.taskType === 'probe' && formData.probeConfigIds.length > 0) {
    const firstConfig = probeOptions.value.find((p: any) => p.id === formData.probeConfigIds[0])
    if (firstConfig?.category) selectedCategory.value = firstConfig.category
  } else if (formData.taskType === 'inspection' && formData.inspectionGroupIds.length > 0) {
    await loadInspectionItems(formData.inspectionGroupIds)
  }

  dialogVisible.value = true
}

const handleCopy = async (row: any) => {
  isEdit.value = false
  Object.assign(formData, {
    name: row.name + '_副本',
    taskType: row.taskType || 'probe',
    probeConfigIds: row.probeConfigIds || [],
    inspectionGroupIds: row.inspectionGroupIds || [],
    inspectionItemIds: row.inspectionItemIds || [],
    groupId: row.groupId, cronExpr: row.cronExpr, pushgatewayId: row.pushgatewayId,
    concurrency: row.concurrency || 5, enabled: row.enabled, description: row.description, owner: row.owner || '',
    // 需求一新增
    executionMode: row.executionMode || '',
    agentHostIds: (() => { try { return row.agentHostIds ? JSON.parse(row.agentHostIds) : [] } catch { return [] } })(),
    businessGroupId: row.businessGroupId || 0,
    customVariables: row.customVariables || ''
  })
  // 需求四：反序列化变量为键值对列表
  deserializeCustomVariables(row.customVariables || '')
  await loadOptions()

  if (formData.taskType === 'probe' && formData.probeConfigIds.length > 0) {
    const firstConfig = probeOptions.value.find((p: any) => p.id === formData.probeConfigIds[0])
    if (firstConfig?.category) selectedCategory.value = firstConfig.category
  } else if (formData.taskType === 'inspection' && formData.inspectionGroupIds.length > 0) {
    await loadInspectionItems(formData.inspectionGroupIds)
  }

  dialogVisible.value = true
}

const handleDelete = (row: any) => {
  Modal.warning({ title: '提示', content: '确定删除该任务？', hideCancel: false, onOk: async () => { await deleteInspectionTask(row.id); Message.success('删除成功'); loadData() } })
}

const handleToggle = async (row: any) => {
  try { await toggleInspectionTask(row.id); Message.success('操作成功'); loadData() } catch {}
}

const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return
  submitting.value = true
  try {
    // 构建请求数据
    const requestData: any = {
      name: formData.name,
      description: formData.description,
      task_type: formData.taskType,
      cron_expr: formData.cronExpr,
      enabled: formData.enabled,
      pushgateway_id: formData.pushgatewayId || 0,
      concurrency: formData.concurrency || 5,
      owner: formData.owner || ''
    }

    // 根据任务类型设置不同的配置
    if (formData.taskType === 'inspection') {
      requestData.group_ids = JSON.stringify(formData.inspectionGroupIds)
      requestData.item_ids = JSON.stringify(formData.inspectionItemIds)
    } else {
      requestData.group_ids = JSON.stringify([formData.groupId])
      requestData.item_ids = JSON.stringify(formData.probeConfigIds)
    }
    // 需求一、三、四：新增执行覆盖配置和变量
    requestData.execution_mode = formData.executionMode || ''
    requestData.agent_host_ids = formData.agentHostIds.length > 0 ? JSON.stringify(formData.agentHostIds) : ''
    requestData.business_group_id = formData.businessGroupId || 0
    requestData.custom_variables = serializeCustomVariables()

    if (isEdit.value) {
      await updateInspectionTask(formData.id, requestData)
      Message.success('更新成功')
    } else {
      await createInspectionTask(requestData)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (error: any) {
    Message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const currentTaskType = ref<'probe' | 'inspection'>('probe')

const handleViewResults = (row: any) => {
  // 如果是巡检任务，直接跳转到执行记录页面
  if (row.taskType === 'inspection') {
    router.push({
      path: '/inspection/records',
      query: { taskId: row.id }
    })
    return
  }

  // 拨测任务显示抽屉
  currentTaskId.value = row.id
  currentTaskType.value = row.taskType || 'probe'
  resultPagination.page = 1
  resultsVisible.value = true
  loadResults()
}

// 需求二：同步执行结果
const syncResultVisible = ref(false)
const syncResultData = ref<any>(null)
const syncRunning = ref(false)

const handleRun = async (row: any) => {
  syncRunning.value = true
  runningTaskIds.value = new Set([...runningTaskIds.value, row.id])
  syncResultData.value = null
  syncResultVisible.value = true
  try {
    const result = await runInspectionTaskSync(row.id)
    syncResultData.value = result
    loadData()
  } catch (e: any) {
    Message.error(e?.response?.data?.message || '执行失败')
    syncResultVisible.value = false
  } finally {
    syncRunning.value = false
    runningTaskIds.value.delete(row.id)
    runningTaskIds.value = new Set(runningTaskIds.value)
  }
}

const handleStop = async (row: any) => {
  try {
    await stopInspectionTask(row.id)
    runningTaskIds.value.delete(row.id)
    runningTaskIds.value = new Set(runningTaskIds.value)
    Message.success('任务已停止')
    loadData()
  } catch (e: any) {
    Message.error(e?.response?.data?.message || '停止任务失败')
  }
}

const loadResults = async () => {
  resultsLoading.value = true
  try {
    if (currentTaskType.value === 'inspection') {
      const res = await getInspectionTaskResults(currentTaskId.value, { page: resultPagination.page, page_size: resultPagination.pageSize })
      resultList.value = res.list || []
      resultPagination.total = res.total || 0
    } else {
      const res = await getTaskResults(currentTaskId.value, { page: resultPagination.page, page_size: resultPagination.pageSize })
      // 拨测任务使用 Pagination 响应格式，数据在 data.data 中
      resultList.value = res.data?.data || []
      resultPagination.total = res.data?.total || 0
    }
  } catch {} finally { resultsLoading.value = false }
}

onMounted(() => { loadData(); loadOptions() })
</script>
<style scoped>
.task-schedule-container { padding: 0; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 36px; height: 36px; border-radius: 8px; background: var(--ops-primary); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 18px; }
.page-title { margin: 0; font-size: 17px; font-weight: 600; color: var(--ops-text-primary); }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: var(--ops-text-tertiary); }
.filter-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; }
.cron-picker { width: 100%; }
.cron-presets { display: flex; flex-wrap: wrap; gap: 6px; }
.cron-description { font-size: 12px; color: var(--ops-success); margin-top: 4px; min-height: 18px; }

/* 运行中动画 */
@keyframes running-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
.running-text {
  animation: running-pulse 1.2s ease-in-out infinite;
  font-size: 12px;
}

/* 同步执行结果样式 */
.sync-result-header {
  background: var(--ops-content-bg);
  padding: 16px;
  border-radius: 4px;
  border: 1px solid var(--ops-border-color);
}
.step-detail-item {
  border-bottom: 1px solid var(--ops-border-color);
  padding: 12px 0;
}
.step-detail-item:last-child { border-bottom: none; }
.step-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.step-name { font-weight: 500; flex: 1; font-size: 13px; }
.step-latency { font-size: 12px; color: var(--ops-text-tertiary); }
.step-info { padding-left: 24px; font-size: 12px; }
.info-line { margin-bottom: 4px; word-break: break-all; line-height: 1.6; }
.info-line.error { color: var(--ops-danger); }
.code-block {
  margin: 0;
  padding: 8px;
  background: #f5f5f5;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
  font-size: 11px;
  line-height: 1.5;
}
.code-block-inline {
  margin: 4px 0 0 0;
  padding: 4px 6px;
  background: #f5f5f5;
  border-radius: 3px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 11px;
  line-height: 1.4;
  display: inline-block;
  max-width: 100%;
}
.assertion-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  line-height: 1.6;
  padding: 4px 0;
  overflow-x: auto;
  white-space: nowrap;
}
.ast-text {
  color: var(--ops-text-secondary);
  display: inline-block;
}
.ast-error {
  color: var(--ops-danger);
  font-size: 11px;
  white-space: normal;
}
</style>
