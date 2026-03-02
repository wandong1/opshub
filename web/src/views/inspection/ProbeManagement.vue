<template>
  <div class="probe-management-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-storage /></div>
        <div>
          <h2 class="page-title">拨测管理</h2>
          <p class="page-subtitle">管理网络拨测配置，支持 Ping / TCP / UDP / HTTP / WebSocket 探测</p>
        </div>
      </div>
    </div>

    <!-- 搜索与操作 -->
    <div class="filter-bar">
      <a-input v-model="searchForm.keyword" placeholder="搜索名称或目标" allow-clear style="width: 220px;" @press-enter="loadData" />
      <a-select v-model="searchForm.category" placeholder="拨测分类" allow-clear style="width: 130px;" @change="handleCategoryFilter">
        <a-option v-for="c in PROBE_CATEGORIES.filter(c => c.enabled)" :key="c.value" :label="c.label" :value="c.value" />
      </a-select>
      <a-select v-model="searchForm.type" placeholder="拨测类型" allow-clear style="width: 130px;">
        <a-option v-for="t in searchTypeOptions" :key="t" :label="t.toUpperCase()" :value="t" />
      </a-select>
      <a-select v-model="searchForm.status" placeholder="状态" allow-clear style="width: 120px;">
        <a-option label="启用" :value="1" />
        <a-option label="禁用" :value="0" />
      </a-select>
      <a-button type="primary" @click="loadData"><template #icon><icon-search /></template>搜索</a-button>
      <a-button @click="handleReset"><template #icon><icon-refresh /></template>重置</a-button>
      <div style="flex: 1;" />
      <a-upload v-permission="'inspection:probes:import'" :auto-upload="false" :show-file-list="false" accept=".yaml,.yml,.json" @change="handleImportChange">
        <template #upload-button><a-button><template #icon><icon-upload /></template>导入</a-button></template>
      </a-upload>
      <a-button v-permission="'inspection:probes:export'" @click="handleExport"><template #icon><icon-download /></template>导出</a-button>
      <a-button v-permission="'inspection:probes:create'" type="primary" @click="handleCreate"><template #icon><icon-plus /></template>新增拨测</a-button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card"><div class="stat-value">{{ stats.total }}</div><div class="stat-label">总数</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-success);">{{ stats.enabled }}</div><div class="stat-label">启用</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-danger);">{{ stats.disabled }}</div><div class="stat-label">禁用</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-primary);">{{ stats.ping }}</div><div class="stat-label">Ping</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-warning);">{{ stats.tcp }}</div><div class="stat-label">TCP</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-info);">{{ stats.udp }}</div><div class="stat-label">UDP</div></div>
      <div class="stat-card"><div class="stat-value" style="color: #165dff;">{{ stats.http }}</div><div class="stat-label">HTTP(S)</div></div>
      <div class="stat-card"><div class="stat-value" style="color: #722ed1;">{{ stats.websocket }}</div><div class="stat-label">WebSocket</div></div>
      <div class="stat-card"><div class="stat-value" style="color: #eb2f96;">{{ stats.workflow }}</div><div class="stat-label">流程</div></div>
    </div>
    <!-- 数据表格 -->
    <a-table :data="tableData" :loading="loading" :bordered="{ cell: true }" stripe :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50] }" @page-change="(p: number) => { pagination.page = p; loadData() }" @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }">
      <template #columns>
        <a-table-column title="名称" data-index="name" :width="140" />
        <a-table-column title="分类" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag size="small">{{ CATEGORY_LABEL_MAP[record.category] || record.category }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="类型" :width="90" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.type === 'ping' ? 'arcoblue' : record.type === 'tcp' ? 'orangered' : 'gray'">{{ record.type.toUpperCase() }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="目标" data-index="target" :width="160" ellipsis tooltip />
        <a-table-column title="端口" :width="80" align="center">
          <template #cell="{ record }">{{ record.type === 'ping' ? '-' : record.port }}</template>
        </a-table-column>
        <a-table-column title="超时(s)" data-index="timeout" :width="90" align="center" />
        <a-table-column title="标签" data-index="tags" :width="140" ellipsis tooltip />
        <a-table-column title="执行方式" :width="90" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.execMode === 'agent' ? 'purple' : record.execMode === 'proxy' ? 'orangered' : 'gray'">{{ record.execMode === 'agent' ? 'Agent' : record.execMode === 'proxy' ? '代理' : '普通' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="状态" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.status === 1 ? 'green' : 'red'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="200" fixed="right" align="center">
          <template #cell="{ record }">
            <a-button v-permission="'inspection:probes:execute'" type="text" size="small" @click="handleRunOnce(record)">执行</a-button>
            <a-button v-permission="'inspection:probes:update'" type="text" size="small" @click="handleEdit(record)">编辑</a-button>
            <a-button v-permission="'inspection:probes:delete'" type="text" size="small" status="danger" @click="handleDelete(record)">删除</a-button>
          </template>
        </a-table-column>
      </template>
    </a-table>

    <!-- 新建/编辑对话框 -->
    <a-modal v-model:visible="dialogVisible" :title="isEdit ? '编辑拨测' : '新增拨测'" :width="800" unmount-on-close>
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="名称" field="name"><a-input v-model="formData.name" placeholder="请输入拨测名称" /></a-form-item>
        <a-form-item label="分类" field="category">
          <a-radio-group v-model="formData.category" type="button" @change="handleCategoryChange">
            <a-radio v-for="c in PROBE_CATEGORIES" :key="c.value" :value="c.value" :disabled="!c.enabled">{{ c.label }}</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="类型" field="type">
          <a-select v-model="formData.type" placeholder="选择拨测类型">
            <a-option v-for="t in availableTypes" :key="t" :label="t.toUpperCase()" :value="t" />
          </a-select>
        </a-form-item>
        <a-form-item v-if="formData.category !== 'application' && formData.category !== 'workflow'" label="目标地址" field="target"><a-input v-model="formData.target" placeholder="IP 或域名" /></a-form-item>
        <a-form-item v-if="formData.type !== 'ping' && formData.category !== 'application' && formData.category !== 'workflow'" label="端口" field="port"><a-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%;" /></a-form-item>
        <!-- 应用服务专属字段 -->
        <template v-if="formData.category === 'application'">
          <a-form-item v-if="formData.type !== 'websocket'" label="HTTP方法">
            <a-select v-model="formData.method" style="width: 100%;">
              <a-option v-for="m in ['GET','POST','PUT','DELETE','PATCH']" :key="m" :value="m">{{ m }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="URL" field="url"><VariableInput v-model="formData.url" :placeholder="formData.type === 'websocket' ? 'ws://example.com/ws 或 wss://example.com/ws' : 'https://example.com/api/health（输入 / 插入变量）'" :variables="variableOptions" /></a-form-item>
          <a-form-item v-if="formData.type !== 'websocket'" label="Content-Type">
            <a-select v-model="formData.contentType" allow-clear style="width: 100%;">
              <a-option value="application/json">application/json</a-option>
              <a-option value="application/x-www-form-urlencoded">application/x-www-form-urlencoded</a-option>
              <a-option value="text/plain">text/plain</a-option>
            </a-select>
          </a-form-item>
          <a-form-item v-if="formData.type === 'https' || formData.type === 'websocket'" label="跳过证书验证">
            <a-switch v-model="formData.skipVerify" />
            <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">HTTPS/WSS 请求跳过 TLS 证书验证</span>
          </a-form-item>
          <template v-if="formData.type === 'websocket'">
            <a-form-item label="消息类型">
              <a-select v-model="formData.wsMessageFormat" style="width: 200px;">
                <a-option v-for="t in WS_MESSAGE_TYPES" :key="t.value" :label="t.label" :value="t.value" />
              </a-select>
            </a-form-item>
            <a-form-item label="发送消息">
              <VariableInput v-model="formData.wsMessage" placeholder="要发送的 WebSocket 消息内容" :variables="variableOptions" :multiline="true" />
            </a-form-item>
            <a-form-item label="接收超时(秒)">
              <a-input-number v-model="formData.wsReadTimeout" :min="0" :max="60" style="width: 120px;" />
              <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">0 = 不等待接收响应</span>
            </a-form-item>
          </template>
          <a-form-item label="Headers">
            <div style="width: 100%;">
              <div v-for="(h, i) in appHeaders" :key="i" style="display: flex; gap: 8px; margin-bottom: 4px;">
                <a-input v-model="h.key" placeholder="Key" style="flex: 1;" />
                <VariableInput v-model="h.value" placeholder="Value（输入 / 插入变量）" :variables="variableOptions" style="flex: 1;" />
                <a-button type="text" status="danger" size="small" @click="appHeaders.splice(i, 1)"><icon-minus /></a-button>
              </div>
              <a-button type="text" size="small" @click="appHeaders.push({ key: '', value: '' })"><icon-plus /> 添加Header</a-button>
            </div>
          </a-form-item>
          <a-form-item label="Query参数">
            <div style="width: 100%;">
              <div v-for="(p, i) in appParams" :key="i" style="display: flex; gap: 8px; margin-bottom: 4px;">
                <a-input v-model="p.key" placeholder="Key" style="flex: 1;" />
                <a-input v-model="p.value" placeholder="Value" style="flex: 1;" />
                <a-button type="text" status="danger" size="small" @click="appParams.splice(i, 1)"><icon-minus /></a-button>
              </div>
              <a-button type="text" size="small" @click="appParams.push({ key: '', value: '' })"><icon-plus /> 添加参数</a-button>
            </div>
          </a-form-item>
          <a-form-item v-if="formData.method !== 'GET' && formData.type !== 'websocket'" label="Body">
            <VariableInput v-model="formData.body" placeholder="请求体（输入 / 插入变量）" :variables="variableOptions" :multiline="true" />
          </a-form-item>
          <a-form-item v-if="formData.execMode === 'proxy'" label="代理URL"><a-input v-model="formData.proxyUrl" placeholder="http://proxy:8080" /></a-form-item>
          <a-form-item label="断言配置">
            <div style="width: 100%;">
              <div v-for="(a, i) in appAssertions" :key="i" style="display: flex; gap: 6px; margin-bottom: 6px; flex-wrap: wrap;">
                <a-input v-model="a.name" placeholder="名称" style="width: 100px;" />
                <a-select v-model="a.source" style="width: 90px;"><a-option value="body">Body</a-option><a-option value="header">Header</a-option></a-select>
                <a-input v-model="a.path" placeholder="路径 $.data.id" style="width: 130px;" />
                <a-select v-model="a.condition" style="width: 110px;">
                  <a-option v-for="c in assertConditions" :key="c.value" :value="c.value">{{ c.label }}</a-option>
                </a-select>
                <a-input v-model="a.value" placeholder="期望值" style="width: 100px;" />
                <a-button type="text" status="danger" size="small" @click="appAssertions.splice(i, 1)"><icon-minus /></a-button>
              </div>
              <a-button type="text" size="small" @click="appAssertions.push({ name: '', source: 'body', path: '', condition: '==', value: '' })"><icon-plus /> 添加断言</a-button>
            </div>
          </a-form-item>
        </template>
        <!-- 业务流程专属字段 -->
        <template v-if="formData.category === 'workflow'">
          <a-form-item label="失败停止">
            <a-switch v-model="workflowStopOnFailure" />
            <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">某步骤失败后停止执行后续步骤</span>
          </a-form-item>
          <a-form-item label="流程变量">
            <div style="width: 100%;">
              <div v-for="(v, i) in workflowVariables" :key="i" style="display: flex; gap: 8px; margin-bottom: 4px;">
                <a-input v-model="v.key" placeholder="变量名" style="flex: 1;" />
                <a-input v-model="v.value" placeholder="变量值" style="flex: 1;" />
                <a-button type="text" status="danger" size="small" @click="workflowVariables.splice(i, 1)"><icon-minus /></a-button>
              </div>
              <a-button type="text" size="small" @click="workflowVariables.push({ key: '', value: '' })"><icon-plus /> 添加变量</a-button>
            </div>
          </a-form-item>
          <a-form-item label="步骤编排">
            <div style="width: 100%;">
              <div v-for="(step, si) in workflowSteps" :key="si" class="workflow-step-item"
                draggable="true" @dragstart="onStepDragStart(si)" @dragover.prevent="onStepDragOver(si)" @drop="onStepDrop(si)" @dragend="stepDragIndex = -1">
                <div class="workflow-step-header" @click="activeStepIndex = activeStepIndex === si ? -1 : si">
                  <span class="workflow-step-drag-handle"><icon-drag-dot-vertical /></span>
                  <span class="workflow-step-index">{{ si + 1 }}</span>
                  <span class="workflow-step-name">{{ step.name || `步骤 ${si + 1}` }}</span>
                  <span style="flex:1;" />
                  <a-tag v-if="step.stepType && step.stepType !== 'http'" size="small" :color="STEP_TYPE_COLORS[step.stepType] || 'gray'">{{ STEP_TYPE_LABELS[step.stepType] || step.stepType }}</a-tag>
                  <a-tag v-else-if="step.method" size="small" color="arcoblue">{{ step.method }}</a-tag>
                  <a-button type="text" status="danger" size="mini" @click.stop="deleteWorkflowStep(si)"><icon-minus /></a-button>
                  <icon-down v-if="activeStepIndex !== si" style="color: var(--ops-text-tertiary);" />
                  <icon-up v-else style="color: var(--ops-text-tertiary);" />
                </div>
                <div v-show="activeStepIndex === si" class="workflow-step-body">
                  <a-row :gutter="12">
                    <a-col :span="8"><a-form-item label="步骤名称" :label-col-flex="'80px'"><a-input v-model="step.name" placeholder="如：登录" /></a-form-item></a-col>
                    <a-col :span="8"><a-form-item label="步骤类型" :label-col-flex="'80px'">
                      <a-select v-model="step.stepType" :disabled="step.stepType === 'ws_connect' || step.stepType === 'ws_disconnect' || step.stepType === 'ws_send' || step.stepType === 'ws_receive'">
                        <a-option v-for="st in STEP_TYPES" :key="st.value" :value="st.value">{{ st.label }}</a-option>
                      </a-select>
                    </a-form-item></a-col>
                    <a-col :span="8"><a-form-item label="延迟(秒)" :label-col-flex="'80px'"><a-input-number v-model="step.delay" :min="0" :max="60" style="width:100%;" /></a-form-item></a-col>
                  </a-row>
                  <!-- http step fields -->
                  <template v-if="!step.stepType || step.stepType === 'http'">
                  <a-row :gutter="12">
                    <a-col :span="8"><a-form-item label="方法" :label-col-flex="'80px'">
                      <a-select v-model="step.method"><a-option v-for="m in ['GET','POST','PUT','DELETE','PATCH']" :key="m" :value="m">{{ m }}</a-option></a-select>
                    </a-form-item></a-col>
                    <a-col :span="16"><a-form-item label="URL" :label-col-flex="'50px'"><VariableInput v-model="step.url" placeholder="{{baseUrl}}/api/xxx" :variables="allWorkflowVariableOptions(si)" /></a-form-item></a-col>
                  </a-row>
                  <a-form-item label="Content-Type" :label-col-flex="'100px'">
                    <a-select v-model="step.contentType" allow-clear style="width: 100%;">
                      <a-option value="application/json">application/json</a-option>
                      <a-option value="application/x-www-form-urlencoded">application/x-www-form-urlencoded</a-option>
                      <a-option value="text/plain">text/plain</a-option>
                    </a-select>
                  </a-form-item>
                  <a-form-item v-if="step.url && (step.url.startsWith('https') || step.url.startsWith('wss'))" label="跳过证书" :label-col-flex="'100px'">
                    <a-switch v-model="step.skipVerify" /><span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">跳过 TLS 证书验证</span>
                  </a-form-item>
                  <a-form-item label="Headers" :label-col-flex="'100px'">
                    <div style="width: 100%;">
                      <div v-for="(h, hi) in step.headers" :key="hi" style="display: flex; gap: 8px; margin-bottom: 4px;">
                        <a-input v-model="h.key" placeholder="Key" style="flex: 1;" />
                        <VariableInput v-model="h.value" placeholder="Value" :variables="allWorkflowVariableOptions(si)" style="flex: 1;" />
                        <a-button type="text" status="danger" size="small" @click="step.headers.splice(hi, 1)"><icon-minus /></a-button>
                      </div>
                      <a-button type="text" size="small" @click="step.headers.push({ key: '', value: '' })"><icon-plus /> 添加Header</a-button>
                    </div>
                  </a-form-item>
                  <a-form-item label="Query参数" :label-col-flex="'100px'">
                    <div style="width: 100%;">
                      <div v-for="(p, pi) in step.params" :key="pi" style="display: flex; gap: 8px; margin-bottom: 4px;">
                        <a-input v-model="p.key" placeholder="Key" style="flex: 1;" />
                        <a-input v-model="p.value" placeholder="Value" style="flex: 1;" />
                        <a-button type="text" status="danger" size="small" @click="step.params.splice(pi, 1)"><icon-minus /></a-button>
                      </div>
                      <a-button type="text" size="small" @click="step.params.push({ key: '', value: '' })"><icon-plus /> 添加参数</a-button>
                    </div>
                  </a-form-item>
                  <a-form-item v-if="step.method !== 'GET'" label="Body" :label-col-flex="'100px'">
                    <VariableInput v-model="step.body" placeholder="请求体" :variables="allWorkflowVariableOptions(si)" :multiline="true" />
                  </a-form-item>
                  </template>
                  <!-- ws_connect fields -->
                  <template v-if="step.stepType === 'ws_connect'">
                    <a-form-item label="URL" :label-col-flex="'100px'"><VariableInput v-model="step.url" placeholder="ws://example.com/ws 或 wss://..." :variables="allWorkflowVariableOptions(si)" /></a-form-item>
                    <a-form-item v-if="step.url && step.url.startsWith('wss')" label="跳过证书" :label-col-flex="'100px'">
                      <a-switch v-model="step.skipVerify" /><span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">跳过 TLS 证书验证</span>
                    </a-form-item>
                    <a-form-item label="Headers" :label-col-flex="'100px'">
                      <div style="width: 100%;">
                        <div v-for="(h, hi) in step.headers" :key="hi" style="display: flex; gap: 8px; margin-bottom: 4px;">
                          <a-input v-model="h.key" placeholder="Key" style="flex: 1;" />
                          <VariableInput v-model="h.value" placeholder="Value" :variables="allWorkflowVariableOptions(si)" style="flex: 1;" />
                          <a-button type="text" status="danger" size="small" @click="step.headers.splice(hi, 1)"><icon-minus /></a-button>
                        </div>
                        <a-button type="text" size="small" @click="step.headers.push({ key: '', value: '' })"><icon-plus /> 添加Header</a-button>
                      </div>
                    </a-form-item>
                    <a-form-item label="Query参数" :label-col-flex="'100px'">
                      <div style="width: 100%;">
                        <div v-for="(p, pi) in step.params" :key="pi" style="display: flex; gap: 8px; margin-bottom: 4px;">
                          <a-input v-model="p.key" placeholder="Key" style="flex: 1;" />
                          <VariableInput v-model="p.value" placeholder="Value" :variables="allWorkflowVariableOptions(si)" style="flex: 1;" />
                          <a-button type="text" status="danger" size="small" @click="step.params.splice(pi, 1)"><icon-minus /></a-button>
                        </div>
                        <a-button type="text" size="small" @click="step.params.push({ key: '', value: '' })"><icon-plus /> 添加参数</a-button>
                      </div>
                    </a-form-item>
                    <a-form-item label="超时(秒)" :label-col-flex="'100px'"><a-input-number v-model="step.timeout" :min="1" :max="60" style="width: 120px;" /></a-form-item>
                  </template>
                  <!-- ws_send fields -->
                  <template v-if="step.stepType === 'ws_send'">
                    <a-form-item label="消息类型" :label-col-flex="'100px'">
                      <a-select v-model="step.wsMessageFormat" style="width: 200px;">
                        <a-option v-for="t in WS_MESSAGE_TYPES" :key="t.value" :label="t.label" :value="t.value" />
                      </a-select>
                    </a-form-item>
                    <a-form-item label="发送消息" :label-col-flex="'100px'">
                      <VariableInput v-model="step.wsMessage" placeholder="要发送的消息" :variables="allWorkflowVariableOptions(si)" :multiline="true" />
                    </a-form-item>
                    <a-form-item label="超时(秒)" :label-col-flex="'100px'"><a-input-number v-model="step.timeout" :min="1" :max="60" style="width: 120px;" /></a-form-item>
                  </template>
                  <!-- ws_receive fields -->
                  <template v-if="step.stepType === 'ws_receive'">
                    <a-form-item label="接收模式" :label-col-flex="'100px'">
                      <a-select v-model="step.wsReceiveMode" style="width: 160px;">
                        <a-option value="single">单条接收</a-option>
                        <a-option value="stream">流式接收</a-option>
                      </a-select>
                      <span v-if="step.wsReceiveMode === 'stream'" style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">在超时时间内持续接收所有消息，结果为 JSON 数组</span>
                    </a-form-item>
                    <a-form-item label="接收超时(秒)" :label-col-flex="'100px'">
                      <a-input-number v-model="step.wsReadTimeout" :min="1" :max="60" style="width: 120px;" />
                    </a-form-item>
                  </template>
                  <!-- Assertions: http, ws_connect(header), ws_receive(body) -->
                  <template v-if="!step.stepType || step.stepType === 'http' || step.stepType === 'ws_connect' || step.stepType === 'ws_receive'">
                  <a-form-item label="断言配置" :label-col-flex="'100px'">
                    <div style="width: 100%;">
                      <div v-for="(a, ai) in step.assertions" :key="ai" style="display: flex; gap: 6px; margin-bottom: 6px; flex-wrap: wrap;">
                        <a-input v-model="a.name" placeholder="名称" style="width: 90px;" />
                        <a-select v-model="a.source" style="width: 85px;">
                          <a-option v-if="step.stepType !== 'ws_connect'" value="body">Body</a-option>
                          <a-option value="header">Header</a-option>
                        </a-select>
                        <a-input v-model="a.path" placeholder="路径" style="width: 120px;" />
                        <a-select v-model="a.condition" style="width: 100px;">
                          <a-option v-for="c in assertConditions" :key="c.value" :value="c.value">{{ c.label }}</a-option>
                        </a-select>
                        <a-input v-model="a.value" placeholder="期望值" style="width: 90px;" />
                        <a-button type="text" status="danger" size="small" @click="step.assertions.splice(ai, 1)"><icon-minus /></a-button>
                      </div>
                      <a-button type="text" size="small" @click="step.assertions.push({ name: '', source: step.stepType === 'ws_connect' ? 'header' : 'body', path: '', condition: '==', value: '' })"><icon-plus /> 添加断言</a-button>
                    </div>
                  </a-form-item>
                  <a-form-item label="变量提取" :label-col-flex="'100px'">
                    <div style="width: 100%;">
                      <div v-for="(ex, ei) in step.extractions" :key="ei" style="display: flex; gap: 8px; margin-bottom: 4px;">
                        <a-input v-model="ex.name" placeholder="变量名" style="flex: 1;" />
                        <a-select v-model="ex.source" style="width: 90px;">
                          <a-option v-if="step.stepType !== 'ws_connect'" value="body">Body</a-option>
                          <a-option value="header">Header</a-option>
                        </a-select>
                        <a-input v-model="ex.path" placeholder="GJSON路径 / Header名" style="flex: 1;" />
                        <a-button type="text" status="danger" size="small" @click="step.extractions.splice(ei, 1)"><icon-minus /></a-button>
                      </div>
                      <a-button type="text" size="small" @click="step.extractions.push({ name: '', source: step.stepType === 'ws_connect' ? 'header' : 'body', path: '' })"><icon-plus /> 添加提取</a-button>
                    </div>
                  </a-form-item>
                  </template>
                  <a-form-item v-if="!step.stepType || step.stepType === 'http'" label="超时(秒)" :label-col-flex="'100px'"><a-input-number v-model="step.timeout" :min="1" :max="60" style="width: 120px;" /></a-form-item>
                </div>
              </div>
              <div style="display: flex; gap: 8px;">
                <a-button type="dashed" style="flex: 1;" @click="addWorkflowStep"><icon-plus /> 添加 HTTP 步骤</a-button>
                <a-button type="dashed" style="flex: 1;" :disabled="hasWSFlow" @click="addWSFlow"><icon-plus /> 添加 WebSocket 流程</a-button>
              </div>
              <div v-if="hasWSFlow" style="margin-top: 6px; display: flex; gap: 8px;">
                <a-button type="dashed" style="flex: 1;" :disabled="!canAddWSMiddleStep" @click="addWSMiddleStep('ws_send')"><icon-plus /> 插入 WS 发送</a-button>
                <a-button type="dashed" style="flex: 1;" :disabled="!canAddWSMiddleStep" @click="addWSMiddleStep('ws_receive')"><icon-plus /> 插入 WS 接收</a-button>
              </div>
            </div>
          </a-form-item>
        </template>
        <a-form-item label="超时(秒)"><a-input-number v-model="formData.timeout" :min="1" :max="60" /></a-form-item>
        <a-form-item v-if="formData.category === 'network' && formData.type === 'ping'" label="Ping次数"><a-input-number v-model="formData.count" :min="1" :max="100" /></a-form-item>
        <a-form-item v-if="formData.category === 'network' && formData.type === 'ping'" label="包大小"><a-input-number v-model="formData.packetSize" :min="16" :max="65500" /></a-form-item>
        <a-form-item label="业务分组">
          <a-select v-model="selectedGroupIds" multiple placeholder="选择业务分组（可多选）" allow-clear allow-search style="width: 100%;">
            <a-option v-for="g in groupOptions" :key="g.id" :label="g.name" :value="g.id" />
          </a-select>
        </a-form-item>
        <a-form-item label="标签"><a-input v-model="formData.tags" placeholder="region=cn-east,env=prod" /></a-form-item>
        <a-form-item label="执行方式">
          <a-radio-group v-model="formData.execMode" type="button">
            <a-radio value="local">普通拨测</a-radio>
            <a-radio value="agent">Agent拨测</a-radio>
            <a-radio v-if="formData.category === 'application'" value="proxy">代理拨测</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="formData.execMode === 'agent'" label="Agent主机">
          <template #label>
            Agent主机
            <a-tooltip position="right">
              <template #content>
                <div style="max-width: 280px;">
                  Agent拨测会从选中的主机中随机挑选一台在线Agent发起探测，实现从不同网络环境进行拨测。<br/><br/>
                  使用前提：目标主机需已部署并启动Agent。多台Agent可提供高可用和多视角探测能力。
                </div>
              </template>
              <icon-question-circle style="margin-left: 4px; color: var(--ops-text-tertiary); cursor: help;" />
            </a-tooltip>
          </template>
          <a-select v-model="selectedAgentHostIds" multiple allow-search placeholder="选择Agent主机（可多选）" style="width: 100%;">
            <a-option v-for="h in filteredHostOptions" :key="h.id" :label="h.name || h.ip" :value="h.id">
              {{ h.name || h.ip }} <span :style="{ float: 'right', fontSize: '12px', color: h.agentOnline ? 'var(--ops-success)' : 'var(--ops-text-tertiary)' }">{{ h.agentOnline ? '在线' : '离线' }}</span>
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="失败重试">
          <a-input-number v-model="formData.retryCount" :min="0" :max="5" />
          <span style="margin-left: 8px; font-size: 12px; color: var(--ops-text-tertiary);">拨测失败时自动重试的次数（0-5）</span>
        </a-form-item>
        <a-form-item label="描述"><a-textarea v-model="formData.description" :max-length="200" :auto-size="{ minRows: 2 }" /></a-form-item>
        <a-form-item label="状态">
          <a-radio-group v-model="formData.status"><a-radio :value="1">启用</a-radio><a-radio :value="0">禁用</a-radio></a-radio-group>
        </a-form-item>
      </a-form>
      <template #footer>
        <div style="display: flex; justify-content: space-between; width: 100%;">
          <a-button :loading="testing" @click="handleTestProbe" status="warning">测试执行</a-button>
          <div style="display: flex; gap: 8px;">
            <a-button @click="dialogVisible = false">取消</a-button>
            <a-button type="primary" :loading="submitting" @click="handleSubmit">确定</a-button>
          </div>
        </div>
      </template>
    </a-modal>

    <!-- 执行结果对话框 -->
    <a-modal v-model:visible="resultDialogVisible" title="拨测结果" :width="750" :footer="false">
      <a-spin :loading="runLoading" style="width: 100%;">
        <template v-if="runResult">
          <!-- 业务流程结果 -->
          <template v-if="currentRecord?.category === 'workflow'">
            <a-descriptions :column="1" bordered size="small">
              <a-descriptions-item label="状态"><a-tag :color="runResult.success ? 'green' : 'red'">{{ runResult.success ? '成功' : '失败' }}</a-tag></a-descriptions-item>
              <a-descriptions-item label="总耗时">{{ runResult.totalLatency?.toFixed(2) }} ms</a-descriptions-item>
              <a-descriptions-item v-if="runResult.error" label="错误">{{ runResult.error }}</a-descriptions-item>
            </a-descriptions>
            <div class="result-section-title">步骤执行时间线</div>
            <a-timeline style="margin-top: 12px;">
              <a-timeline-item v-for="(sr, si) in (runResult.stepResults || [])" :key="si"
                :dot-color="sr.skipped ? 'var(--ops-text-tertiary)' : sr.success ? 'var(--ops-success)' : 'var(--ops-danger)'">
                <div class="wf-step-result-header">
                  <span style="font-weight: 500;">{{ sr.stepName || `步骤 ${si + 1}` }}</span>
                  <a-tag v-if="sr.stepType && sr.stepType !== 'http'" size="small" :color="STEP_TYPE_COLORS[sr.stepType] || 'gray'">{{ STEP_TYPE_LABELS[sr.stepType] || sr.stepType }}</a-tag>
                  <a-tag v-if="sr.skipped" size="small" color="gray">跳过</a-tag>
                  <a-tag v-else-if="sr.success" size="small" color="green">成功</a-tag>
                  <a-tag v-else size="small" color="red">失败</a-tag>
                  <a-tag v-if="sr.httpStatusCode" size="small" :color="sr.httpStatusCode < 400 ? 'arcoblue' : 'red'">{{ sr.httpStatusCode }}</a-tag>
                  <span v-if="sr.latency" style="font-size: 12px; color: var(--ops-text-tertiary);">{{ sr.latency?.toFixed(2) }} ms</span>
                </div>
                <template v-if="!sr.skipped">
                  <div v-if="sr.assertionResults?.length" style="margin-top: 4px;">
                    <span v-for="(ar, ai) in sr.assertionResults" :key="ai" style="margin-right: 8px;">
                      <a-tag size="small" :color="ar.success ? 'green' : 'red'">{{ ar.name || `断言${ai+1}` }}</a-tag>
                      <span v-if="!ar.success" style="font-size: 12px; color: var(--ops-danger); margin-left: 2px;">{{ ar.error }}</span>
                    </span>
                  </div>
                  <div v-if="sr.extractedVars && Object.keys(sr.extractedVars).length" style="margin-top: 4px;">
                    <span style="font-size: 12px; color: var(--ops-text-secondary);">提取变量：</span>
                    <a-tag v-for="(val, key) in sr.extractedVars" :key="key" size="small" color="orangered">{{ key }}={{ truncateText(String(val), 40) }}</a-tag>
                  </div>
                  <div v-if="sr.error" style="margin-top: 4px; font-size: 12px; color: var(--ops-danger);">{{ sr.error }}</div>
                  <div v-if="sr.responseBody" style="margin-top: 4px;">
                    <div style="display: flex; justify-content: flex-end; margin-bottom: 2px;">
                      <a-button type="text" size="mini" @click="copyText(sr.responseBody)"><icon-copy /> 复制</a-button>
                    </div>
                    <pre class="response-body-pre" v-html="formatJsonWithCollapse(sr.responseBody)" @click="handleResultBodyClick" />
                  </div>
                </template>
              </a-timeline-item>
            </a-timeline>
          </template>
          <!-- 非流程结果（原有逻辑） -->
          <template v-else>
          <a-descriptions :column="1" bordered>
            <a-descriptions-item label="状态"><a-tag :color="runResult.Success ? 'green' : 'red'">{{ runResult.Success ? '成功' : '失败' }}</a-tag></a-descriptions-item>
            <a-descriptions-item label="延迟">{{ runResult.Latency?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.PacketLoss !== undefined && runResult.PacketLoss > 0" label="丢包率">{{ (runResult.PacketLoss * 100).toFixed(1) }}%</a-descriptions-item>
            <a-descriptions-item v-if="runResult.PingRttAvg" label="平均RTT">{{ runResult.PingRttAvg?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.TCPConnectTime" label="TCP连接">{{ runResult.TCPConnectTime?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.HTTPStatusCode" label="HTTP状态码"><a-tag :color="runResult.HTTPStatusCode < 400 ? 'green' : 'red'">{{ runResult.HTTPStatusCode }}</a-tag></a-descriptions-item>
            <a-descriptions-item v-if="runResult.HTTPResponseTime" label="HTTP响应耗时">{{ runResult.HTTPResponseTime?.toFixed(2) }} ms</a-descriptions-item>
            <a-descriptions-item v-if="runResult.HTTPContentLength" label="响应大小">{{ runResult.HTTPContentLength }} bytes</a-descriptions-item>
            <a-descriptions-item v-if="runResult.AssertionResults?.length" label="断言结果">
              <div v-for="(ar, i) in runResult.AssertionResults" :key="i" style="margin-bottom: 4px;">
                <a-tag size="small" :color="ar.success ? 'green' : 'red'">{{ ar.name || `断言${i+1}` }}</a-tag>
                <span v-if="!ar.success" style="font-size: 12px; color: var(--ops-danger); margin-left: 4px;">{{ ar.error }}</span>
              </div>
            </a-descriptions-item>
            <a-descriptions-item v-if="runResult.Error" label="错误">{{ runResult.Error }}</a-descriptions-item>
            <a-descriptions-item v-if="runResult.agentHostId > 0" label="Agent主机">主机 #{{ runResult.agentHostId }}</a-descriptions-item>
            <a-descriptions-item v-if="runResult.retryAttempt > 0" label="重试次数">{{ runResult.retryAttempt }} 次</a-descriptions-item>
          </a-descriptions>
          <!-- 请求信息 -->
          <template v-if="currentRecord">
            <div class="result-section-title">请求信息</div>
            <a-descriptions :column="1" bordered size="small">
              <!-- 网络/四层拨测：显示目标地址和端口 -->
              <template v-if="currentRecord.category !== 'application'">
                <a-descriptions-item label="目标地址">{{ currentRecord.target || '-' }}</a-descriptions-item>
                <a-descriptions-item v-if="currentRecord.type !== 'ping'" label="端口">{{ currentRecord.port || '-' }}</a-descriptions-item>
                <a-descriptions-item label="拨测类型">{{ (currentRecord.type || '').toUpperCase() }}</a-descriptions-item>
              </template>
              <!-- 应用服务拨测：显示URL、方法、Headers、Params、Body -->
              <template v-else>
                <a-descriptions-item label="请求URL">{{ buildDisplayUrl(currentRecord) }}</a-descriptions-item>
                <a-descriptions-item v-if="currentRecord.method" label="请求方法">{{ currentRecord.method }}</a-descriptions-item>
                <a-descriptions-item v-if="parseJsonSafe(currentRecord.headers)" label="请求Headers">
                  <div v-for="(val, key) in parseJsonSafe(currentRecord.headers)" :key="key" class="kv-row">
                    <span class="kv-key">{{ key }}:</span>
                    <template v-if="isLongText(val)">
                      <span class="kv-val-truncated">{{ truncateText(String(val)) }}</span>
                      <a-button type="text" size="mini" @click="showLongText(String(val), `Header: ${key}`)"><icon-expand /></a-button>
                    </template>
                    <span v-else class="kv-val">{{ val }}</span>
                  </div>
                </a-descriptions-item>
                <a-descriptions-item v-if="parseJsonSafe(currentRecord.params)" label="请求Params">
                  <div v-for="(val, key) in parseJsonSafe(currentRecord.params)" :key="key" class="kv-row">
                    <span class="kv-key">{{ key }}:</span>
                    <template v-if="isLongText(String(val))">
                      <span class="kv-val-truncated">{{ truncateText(String(val)) }}</span>
                      <a-button type="text" size="mini" @click="showLongText(String(val), `Param: ${key}`)"><icon-expand /></a-button>
                    </template>
                    <span v-else class="kv-val">{{ val }}</span>
                  </div>
                </a-descriptions-item>
                <a-descriptions-item v-if="currentRecord.body" label="请求Body">
                  <pre class="response-body-pre" v-html="formatJsonWithCollapse(currentRecord.body)" @click="handleResultBodyClick" />
                </a-descriptions-item>
              </template>
            </a-descriptions>
          </template>
          <!-- 响应信息 -->
          <template v-if="runResult.ResponseHeaders || runResult.ResponseBody">
            <div class="result-section-title">响应信息</div>
            <a-descriptions :column="1" bordered size="small">
              <a-descriptions-item v-if="runResult.ResponseHeaders && Object.keys(runResult.ResponseHeaders).length" label="响应Headers">
                <div v-for="(val, key) in runResult.ResponseHeaders" :key="key" class="kv-row">
                  <span class="kv-key">{{ key }}:</span>
                  <template v-if="isLongText(Array.isArray(val) ? val.join(', ') : String(val))">
                    <span class="kv-val-truncated">{{ truncateText(Array.isArray(val) ? val.join(', ') : String(val)) }}</span>
                    <a-button type="text" size="mini" @click="showLongText(Array.isArray(val) ? val.join(', ') : String(val), `Header: ${key}`)"><icon-expand /></a-button>
                  </template>
                  <span v-else class="kv-val">{{ Array.isArray(val) ? val.join(', ') : val }}</span>
                </div>
              </a-descriptions-item>
              <a-descriptions-item v-if="runResult.ResponseBody" label="响应Body">
                <div>
                  <div style="display: flex; justify-content: flex-end; margin-bottom: 4px;">
                    <a-button type="text" size="mini" @click="copyText(runResult.ResponseBody)"><icon-copy /> 复制</a-button>
                  </div>
                  <pre class="response-body-pre" v-html="formatJsonWithCollapse(runResult.ResponseBody)" @click="handleResultBodyClick" />
                </div>
              </a-descriptions-item>
            </a-descriptions>
          </template>
          </template>
        </template>
      </a-spin>
    </a-modal>

    <!-- 长文本查看弹窗 -->
    <a-modal v-model:visible="longTextVisible" :title="longTextTitle" :width="700" :footer="false">
      <div style="display: flex; justify-content: flex-end; margin-bottom: 8px;">
        <a-button size="small" @click="copyText(longTextContent)"><template #icon><icon-copy /></template>复制</a-button>
      </div>
      <pre class="long-text-pre">{{ longTextContent }}</pre>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import { IconStorage, IconSearch, IconRefresh, IconPlus, IconUpload, IconDownload, IconQuestionCircle, IconMinus, IconExpand, IconCopy, IconDragDotVertical, IconDown, IconUp } from '@arco-design/web-vue/es/icon'
import { getProbeList, createProbe, updateProbe, deleteProbe, importProbes, exportProbes, runProbeOnce, testProbe, PROBE_CATEGORIES, CATEGORY_TYPE_MAP, CATEGORY_LABEL_MAP, getVariableList } from '@/api/networkProbe'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'
import { getAgentStatuses } from '@/api/agent'
import VariableInput from '@/components/VariableInput.vue'

const loading = ref(false)
const submitting = ref(false)
const testing = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const tableData = ref<any[]>([])
const resultDialogVisible = ref(false)
const runLoading = ref(false)
const runResult = ref<any>(null)
const currentRecord = ref<any>(null)
const longTextVisible = ref(false)
const longTextContent = ref('')
const longTextTitle = ref('详情')
const groupOptions = ref<any[]>([])
const hostOptions = ref<any[]>([])
const variableOptions = ref<{ name: string; description?: string }[]>([])
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const searchForm = reactive({ keyword: '', type: '', category: '', status: undefined as number | undefined })

const defaultForm = () => ({
  id: 0, name: '', category: 'network', type: 'ping', target: '', port: 80, groupId: 0, groupIds: '',
  timeout: 5, count: 4, packetSize: 64, description: '', tags: '', status: 1,
  execMode: 'local', agentHostIds: '', retryCount: 0,
  method: 'GET', url: '', headers: '', params: '', body: '', proxyUrl: '', assertions: '', contentType: '',
  skipVerify: true, wsMessage: '', wsMessageType: 1, wsMessageFormat: 'text', wsReadTimeout: 5
})
const formData = reactive(defaultForm())

const appHeaders = reactive<{ key: string; value: string }[]>([])
const appParams = reactive<{ key: string; value: string }[]>([])
const appAssertions = reactive<{ name: string; source: string; path: string; condition: string; value: string }[]>([])
// Workflow state
const workflowVariables = reactive<{ key: string; value: string }[]>([])
const workflowSteps = reactive<any[]>([])
const workflowStopOnFailure = ref(true)
const activeStepIndex = ref(-1)
const stepDragIndex = ref(-1)
const assertConditions = [
  { value: '==', label: '等于' }, { value: '>', label: '大于' }, { value: '>=', label: '大于等于' },
  { value: '<', label: '小于' }, { value: '<=', label: '小于等于' },
  { value: 'contains', label: '包含' }, { value: 'notcontains', label: '不包含' },
  { value: 'regexp', label: '正则匹配' }, { value: 'notregexp', label: '正则不匹配' },
]
const WS_MESSAGE_TYPES = [
  { value: 'text', label: 'Text', wireType: 1 },
  { value: 'json', label: 'JSON', wireType: 1 },
  { value: 'xml', label: 'XML', wireType: 1 },
  { value: 'html', label: 'HTML', wireType: 1 },
  { value: 'binary', label: 'Binary', wireType: 2 },
]
const STEP_TYPES = [
  { value: 'http', label: 'HTTP 请求' },
  { value: 'ws_connect', label: 'WS 连接' },
  { value: 'ws_send', label: 'WS 发送' },
  { value: 'ws_receive', label: 'WS 接收' },
  { value: 'ws_disconnect', label: 'WS 断开' },
]
const STEP_TYPE_COLORS: Record<string, string> = {
  http: 'arcoblue', ws_connect: 'purple', ws_send: 'orangered', ws_receive: 'green', ws_disconnect: 'gray'
}
const STEP_TYPE_LABELS: Record<string, string> = {
  http: 'HTTP', ws_connect: 'WS连接', ws_send: 'WS发送', ws_receive: 'WS接收', ws_disconnect: 'WS断开'
}

const formRules = computed(() => ({
  name: [{ required: true, message: '请输入名称' }],
  type: [{ required: true, message: '请选择类型' }],
  target: [{ required: formData.category !== 'application' && formData.category !== 'workflow', message: '请输入目标地址' }],
}))

const stats = computed(() => {
  const all = tableData.value
  return {
    total: pagination.total,
    enabled: all.filter((r: any) => r.status === 1).length,
    disabled: all.filter((r: any) => r.status === 0).length,
    ping: all.filter((r: any) => r.type === 'ping').length,
    tcp: all.filter((r: any) => r.type === 'tcp').length,
    udp: all.filter((r: any) => r.type === 'udp').length,
    http: all.filter((r: any) => r.type === 'http' || r.type === 'https').length,
    websocket: all.filter((r: any) => r.type === 'websocket').length,
    workflow: all.filter((r: any) => r.type === 'workflow').length,
  }
})

const loadData = async () => {
  loading.value = true
  try {
    const res = await getProbeList({
      page: pagination.page, page_size: pagination.pageSize,
      keyword: searchForm.keyword, type: searchForm.type, category: searchForm.category, status: searchForm.status
    })
    tableData.value = res.data || []; pagination.total = res.total || 0
  } catch {} finally { loading.value = false }
}

const handleReset = () => {
  searchForm.keyword = ''; searchForm.type = ''; searchForm.category = ''; searchForm.status = undefined
  pagination.page = 1; loadData()
}

const availableTypes = computed(() => CATEGORY_TYPE_MAP[formData.category] || ['ping', 'tcp', 'udp'])
const searchTypeOptions = computed(() => {
  if (searchForm.category) return CATEGORY_TYPE_MAP[searchForm.category] || []
  const all: string[] = []
  for (const cat of PROBE_CATEGORIES.filter(c => c.enabled)) { all.push(...(CATEGORY_TYPE_MAP[cat.value] || [])) }
  return all
})
const handleCategoryChange = () => { const types = availableTypes.value; if (!types.includes(formData.type)) formData.type = types[0] || '' }
const handleCategoryFilter = () => { pagination.page = 1; loadData() }

// Workflow helpers
const newWorkflowStep = () => ({
  name: '', stepType: 'http', delay: 0, method: 'GET', url: '', contentType: 'application/json',
  headers: [] as { key: string; value: string }[], params: [] as { key: string; value: string }[],
  body: '', timeout: 10, skipVerify: true, wsMessage: '', wsMessageType: 1, wsMessageFormat: 'text', wsReadTimeout: 5, wsReceiveMode: 'single',
  assertions: [] as { name: string; source: string; path: string; condition: string; value: string }[],
  extractions: [] as { name: string; source: string; path: string }[],
  execMode: '', proxyUrl: ''
})
const addWorkflowStep = () => { workflowSteps.push(newWorkflowStep()); activeStepIndex.value = workflowSteps.length - 1 }

// WS flow constraints
const hasWSFlow = computed(() => workflowSteps.some(s => s.stepType === 'ws_connect'))
const wsConnectIndex = computed(() => workflowSteps.findIndex(s => s.stepType === 'ws_connect'))
const wsDisconnectIndex = computed(() => workflowSteps.findIndex(s => s.stepType === 'ws_disconnect'))
const canAddWSMiddleStep = computed(() => wsConnectIndex.value >= 0 && wsDisconnectIndex.value > wsConnectIndex.value)

const addWSFlow = () => {
  if (hasWSFlow.value) return
  const connectStep = newWorkflowStep()
  connectStep.stepType = 'ws_connect'; connectStep.name = 'WS 连接'; connectStep.method = ''
  const disconnectStep = newWorkflowStep()
  disconnectStep.stepType = 'ws_disconnect'; disconnectStep.name = 'WS 断开'; disconnectStep.method = ''
  workflowSteps.push(connectStep, disconnectStep)
  activeStepIndex.value = workflowSteps.length - 2
}

const addWSMiddleStep = (type: string) => {
  if (!canAddWSMiddleStep.value) return
  const step = newWorkflowStep()
  step.stepType = type
  step.name = type === 'ws_send' ? 'WS 发送' : 'WS 接收'
  step.method = ''
  // Insert before ws_disconnect
  workflowSteps.splice(wsDisconnectIndex.value, 0, step)
  activeStepIndex.value = wsDisconnectIndex.value - 1
}

const canDeleteStep = (si: number) => {
  const step = workflowSteps[si]
  if (!step) return true
  // Deleting ws_connect or ws_disconnect removes the entire WS flow
  if (step.stepType === 'ws_connect' || step.stepType === 'ws_disconnect') return true
  return true
}

const deleteWorkflowStep = (si: number) => {
  const step = workflowSteps[si]
  if (step.stepType === 'ws_connect' || step.stepType === 'ws_disconnect') {
    // Remove entire WS flow: connect, all ws_send/ws_receive between, and disconnect
    const ci = wsConnectIndex.value
    const di = wsDisconnectIndex.value
    if (ci >= 0 && di >= ci) {
      workflowSteps.splice(ci, di - ci + 1)
      activeStepIndex.value = -1
      return
    }
  }
  workflowSteps.splice(si, 1)
  if (activeStepIndex.value >= workflowSteps.length) activeStepIndex.value = -1
}
const onStepDragStart = (i: number) => { stepDragIndex.value = i }
const onStepDragOver = (_i: number) => {}
const onStepDrop = (targetIndex: number) => {
  const from = stepDragIndex.value
  if (from < 0 || from === targetIndex) return
  const item = workflowSteps.splice(from, 1)[0]
  workflowSteps.splice(targetIndex, 0, item)
  activeStepIndex.value = targetIndex
}
const allWorkflowVariableOptions = (stepIndex: number) => {
  const opts: { name: string; description?: string }[] = []
  // System variables
  for (const v of variableOptions.value) opts.push(v)
  // Workflow-level variables
  for (const v of workflowVariables) { if (v.key) opts.push({ name: v.key, description: '流程变量' }) }
  // Extracted variables from previous steps
  for (let i = 0; i < stepIndex; i++) {
    const s = workflowSteps[i]
    if (s?.extractions) {
      for (const ex of s.extractions) { if (ex.name) opts.push({ name: ex.name, description: `步骤${i + 1}提取` }) }
    }
  }
  return opts
}

const loadGroups = async () => {
  try { const res = await getGroupTree(); groupOptions.value = flattenGroups(res.data || res || []) } catch {}
}
const flattenGroups = (tree: any[], result: any[] = []): any[] => {
  for (const node of tree) { result.push({ id: node.id, name: node.name }); if (node.children?.length) flattenGroups(node.children, result) }
  return result
}

const selectedAgentHostIds = computed({
  get: () => formData.agentHostIds ? formData.agentHostIds.split(',').filter(Boolean).map(Number) : [],
  set: (val: number[]) => { formData.agentHostIds = val.join(',') }
})

const selectedGroupIds = computed({
  get: () => formData.groupIds ? formData.groupIds.split(',').filter(Boolean).map(Number) : [],
  set: (val: number[]) => { formData.groupIds = val.join(',') }
})

const filteredHostOptions = computed(() => {
  const gids = selectedGroupIds.value
  if (gids.length === 0) return hostOptions.value
  return hostOptions.value.filter((h: any) => gids.includes(h.groupId))
})

// Clear agent hosts that are no longer in the filtered list when groups change
// Also reload variables when groups change to reflect group-scoped variables
watch(selectedGroupIds, () => {
  const validIds = new Set(filteredHostOptions.value.map((h: any) => h.id))
  const current = selectedAgentHostIds.value
  const filtered = current.filter(id => validIds.has(id))
  if (filtered.length !== current.length) {
    selectedAgentHostIds.value = filtered
  }
  if (dialogVisible.value) {
    loadVariables()
  }
})

const loadHosts = async () => {
  try {
    const res = await getHostList({ page: 1, page_size: 1000 })
    const hosts = res.list || []
    try {
      const data = await getAgentStatuses()
      const list = Array.isArray(data) ? data : []
      const statusMap: Record<number, boolean> = {}
      for (const item of list) {
        statusMap[item.hostId] = item.status === 'online'
      }
      for (const h of hosts) {
        h.agentOnline = !!statusMap[h.id]
      }
    } catch {}
    hostOptions.value = hosts
  } catch {}
}

const loadVariables = async () => {
  try {
    const params: any = { page: 1, page_size: 100 }
    // Pass group IDs to filter variables by scope
    if (formData.groupIds) {
      params.group_ids = formData.groupIds
    }
    const res = await getVariableList(params)
    const list = res?.list || res?.data || []
    variableOptions.value = list.map((v: any) => ({ name: v.name, description: v.description || '' }))
  } catch {}
}

const handleCreate = () => {
  isEdit.value = false; Object.assign(formData, defaultForm())
  appHeaders.length = 0; appParams.length = 0; appAssertions.length = 0
  workflowVariables.length = 0; workflowSteps.length = 0; workflowStopOnFailure.value = true; activeStepIndex.value = -1
  loadGroups(); loadHosts(); loadVariables(); dialogVisible.value = true
}
const handleEdit = (row: any) => {
  isEdit.value = true
  Object.assign(formData, {
    ...row, category: row.category || 'network', execMode: row.execMode || 'local',
    agentHostIds: row.agentHostIds || '', retryCount: row.retryCount || 0, groupIds: row.groupIds || '',
    method: row.method || 'GET', url: row.url || '', body: row.body || '',
    proxyUrl: row.proxyUrl || '', contentType: row.contentType || '',
    headers: row.headers || '', params: row.params || '', assertions: row.assertions || '',
    skipVerify: row.skipVerify !== undefined ? row.skipVerify : true,
    wsMessage: row.wsMessage || '', wsMessageType: row.wsMessageType || 1,
    wsMessageFormat: row.wsMessageFormat || (row.wsMessageType === 2 ? 'binary' : 'text'),
    wsReadTimeout: row.wsReadTimeout || 5
  })
  // Parse JSON fields into reactive arrays
  appHeaders.length = 0
  if (row.headers) { try { const h = JSON.parse(row.headers); for (const [k, v] of Object.entries(h)) appHeaders.push({ key: k, value: v as string }) } catch {} }
  appParams.length = 0
  if (row.params) { try { const p = JSON.parse(row.params); for (const [k, v] of Object.entries(p)) appParams.push({ key: k, value: v as string }) } catch {} }
  appAssertions.length = 0
  if (row.assertions) { try { const a = JSON.parse(row.assertions); for (const item of a) appAssertions.push({ ...item }) } catch {} }
  // Restore workflow data
  workflowVariables.length = 0; workflowSteps.length = 0; workflowStopOnFailure.value = true; activeStepIndex.value = -1
  if (row.category === 'workflow' && row.body) {
    try {
      const def = JSON.parse(row.body)
      workflowStopOnFailure.value = def.stopOnFailure !== false
      if (def.variables) { for (const [k, v] of Object.entries(def.variables)) workflowVariables.push({ key: k, value: v as string }) }
      if (def.steps) {
        for (const s of def.steps) {
          const step = newWorkflowStep()
          step.name = s.name || ''; step.stepType = s.stepType || 'http'; step.delay = s.delay || 0; step.method = s.method || 'GET'
          step.url = s.url || ''; step.contentType = s.contentType || ''; step.body = s.body || ''
          step.timeout = s.timeout || 10; step.execMode = s.execMode || ''; step.proxyUrl = s.proxyUrl || ''
          step.skipVerify = s.skipVerify !== undefined ? s.skipVerify : true
          step.wsMessage = s.wsMessage || ''; step.wsMessageType = s.wsMessageType || 1
          step.wsMessageFormat = s.wsMessageFormat || (s.wsMessageType === 2 ? 'binary' : 'text')
          step.wsReadTimeout = s.wsReadTimeout || 5
          step.wsReceiveMode = s.wsReceiveMode || 'single'
          if (s.headers && typeof s.headers === 'object' && !Array.isArray(s.headers)) {
            step.headers = Object.entries(s.headers).map(([k, v]) => ({ key: k, value: v as string }))
          } else if (Array.isArray(s.headers)) { step.headers = s.headers }
          if (s.params && typeof s.params === 'object' && !Array.isArray(s.params)) {
            step.params = Object.entries(s.params).map(([k, v]) => ({ key: k, value: v as string }))
          } else if (Array.isArray(s.params)) { step.params = s.params }
          if (s.assertions) step.assertions = s.assertions.map((a: any) => ({ ...a }))
          if (s.extractions) step.extractions = s.extractions.map((e: any) => ({ ...e }))
          workflowSteps.push(step)
        }
      }
    } catch {}
  }
  loadGroups(); loadHosts(); loadVariables(); dialogVisible.value = true
}

const handleDelete = (row: any) => {
  Modal.warning({ title: '提示', content: '确定删除该拨测配置？', hideCancel: false, onOk: async () => { await deleteProbe(row.id); Message.success('删除成功'); loadData() } })
}

const handleSubmit = async () => {
  if (!formRef.value) return
  const errors = await formRef.value.validate()
  if (errors) return
  submitting.value = true
  try {
    const data = buildProbeData()
    if (isEdit.value) { await updateProbe(data.id, data); Message.success('更新成功') }
    else { await createProbe(data); Message.success('创建成功') }
    dialogVisible.value = false; loadData()
  } catch {} finally { submitting.value = false }
}

const buildProbeData = () => {
  const data: any = { ...formData }
  if (data.category === 'application') {
    const headersObj: Record<string, string> = {}
    for (const h of appHeaders) { if (h.key) headersObj[h.key] = h.value }
    data.headers = Object.keys(headersObj).length ? JSON.stringify(headersObj) : ''
    const paramsObj: Record<string, string> = {}
    for (const p of appParams) { if (p.key) paramsObj[p.key] = p.value }
    data.params = Object.keys(paramsObj).length ? JSON.stringify(paramsObj) : ''
    data.assertions = appAssertions.length ? JSON.stringify(appAssertions.filter((a: any) => a.name || a.path)) : ''
    if (!data.url) data.url = data.target
    data.skipVerify = formData.skipVerify
    data.wsMessage = formData.wsMessage
    data.wsMessageType = WS_MESSAGE_TYPES.find(t => t.value === formData.wsMessageFormat)?.wireType || 1
    data.wsMessageFormat = formData.wsMessageFormat
    data.wsReadTimeout = formData.wsReadTimeout
    if (data.type === 'websocket' && !data.target) data.target = data.url || 'websocket'
  }
  if (data.category === 'workflow') {
    const vars: Record<string, string> = {}
    for (const v of workflowVariables) { if (v.key) vars[v.key] = v.value }
    const steps = workflowSteps.map(s => {
      const headersObj: Record<string, string> = {}
      for (const h of (s.headers || [])) { if (h.key) headersObj[h.key] = h.value }
      const paramsObj: Record<string, string> = {}
      for (const p of (s.params || [])) { if (p.key) paramsObj[p.key] = p.value }
      return {
        name: s.name, stepType: s.stepType || 'http', delay: s.delay || 0, method: s.method || 'GET', url: s.url,
        contentType: s.contentType, headers: headersObj, params: paramsObj,
        body: s.body || '', timeout: s.timeout || 10,
        skipVerify: s.skipVerify !== undefined ? s.skipVerify : true,
        wsMessage: s.wsMessage || '',
        wsMessageType: WS_MESSAGE_TYPES.find(t => t.value === s.wsMessageFormat)?.wireType || s.wsMessageType || 1,
        wsMessageFormat: s.wsMessageFormat || 'text',
        wsReadTimeout: s.wsReadTimeout || 5,
        wsReceiveMode: s.wsReceiveMode || 'single',
        assertions: (s.assertions || []).filter((a: any) => a.name || a.path),
        extractions: (s.extractions || []).filter((e: any) => e.name),
        execMode: s.execMode || '', proxyUrl: s.proxyUrl || ''
      }
    })
    data.body = JSON.stringify({ variables: vars, stopOnFailure: workflowStopOnFailure.value, steps })
    data.target = data.target || 'workflow'
    data.type = 'workflow'
  }
  return data
}

const handleTestProbe = async () => {
  testing.value = true
  try {
    const data = buildProbeData()
    const res = await testProbe(data)
    currentRecord.value = data
    runResult.value = res
    resultDialogVisible.value = true
  } catch (e: any) {
    Message.error('测试执行失败: ' + (e?.message || '未知错误'))
  } finally { testing.value = false }
}

const handleRunOnce = async (row: any) => {
  currentRecord.value = row
  resultDialogVisible.value = true; runLoading.value = true; runResult.value = null
  try { runResult.value = await runProbeOnce(row.id) } catch {} finally { runLoading.value = false }
}

const isLongText = (val: any) => typeof val === 'string' && val.length > 1000
const truncateText = (val: string, max = 80) => val.length > max ? val.slice(0, max) + '...' : val
const showLongText = (content: string, title = '详情') => {
  longTextTitle.value = title; longTextContent.value = content; longTextVisible.value = true
}
const copyText = async (text: string) => {
  try { await navigator.clipboard.writeText(text); Message.success('已复制') } catch { Message.error('复制失败') }
}
const parseJsonSafe = (str: string): Record<string, string> | null => {
  if (!str) return null
  try { return JSON.parse(str) } catch { return null }
}
const buildDisplayUrl = (record: any): string => {
  const base = record.url || record.target || '-'
  if (base === '-') return base
  const params = parseJsonSafe(record.params)
  if (!params || Object.keys(params).length === 0) return base
  const qs = Object.entries(params).map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`).join('&')
  return base + (base.includes('?') ? '&' : '?') + qs
}
const formatResponseBody = (body: string): string => {
  if (!body) return ''
  try { return JSON.stringify(JSON.parse(body), null, 2) } catch { return body }
}

// Collapse long string values in JSON for result display (same visual as VariableInput collapse pills)
const RESULT_COLLAPSE_THRESHOLD = 500
const resultCollapsedMap = new Map<string, string>()
let resultCollapseIdCounter = 0

const escapeHtml = (str: string) => str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')

const formatJsonWithCollapse = (jsonStr: string): string => {
  if (!jsonStr) return ''
  // Try to pretty-print JSON first
  let text = jsonStr
  try {
    text = JSON.stringify(JSON.parse(jsonStr), null, 2)
  } catch {
    // Not valid JSON, use raw text as-is
  }
  // Process: split text around quoted strings, escape non-quoted parts,
  // collapse long quoted strings into pills (same as VariableInput)
  const regex = /"((?:[^"\\]|\\.)*)"/g
  let html = ''
  let lastIndex = 0
  let m: RegExpExecArray | null
  while ((m = regex.exec(text)) !== null) {
    // Escape text before this match
    html += escapeHtml(text.slice(lastIndex, m.index))
    const content = m[1]
    if (content.length > RESULT_COLLAPSE_THRESHOLD) {
      const id = `rc_${resultCollapseIdCounter++}`
      resultCollapsedMap.set(id, content)
      const label = content.length > 1024 ? `${(content.length / 1024).toFixed(1)}KB` : `${content.length}B`
      html += `<span class="result-collapse-pill" data-collapse-id="${id}" title="点击查看完整内容"><svg viewBox="0 0 48 48" width="12" height="12" fill="currentColor" style="flex-shrink:0"><path d="M6 9a3 3 0 0 1 3-3h12l4 4h14a3 3 0 0 1 3 3v24a3 3 0 0 1-3 3H9a3 3 0 0 1-3-3V9z"/></svg>${label}</span>`
    } else {
      html += escapeHtml(m[0])
    }
    lastIndex = m.index + m[0].length
  }
  // Escape remaining text after last match
  html += escapeHtml(text.slice(lastIndex))
  return html
}

const handleResultBodyClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  const pill = target.closest('.result-collapse-pill') as HTMLElement | null
  if (!pill) return
  const id = pill.dataset.collapseId || ''
  const content = resultCollapsedMap.get(id) || ''
  showLongText(content, '查看完整内容')
}

const handleImportChange = async (fileList: any[]) => {
  const file = fileList[fileList.length - 1]?.file
  if (!file) return
  try { await importProbes(file); Message.success('导入成功'); loadData() } catch {}
}

const handleExport = async () => {
  try {
    const blob = await exportProbes('yaml') as any
    const url = window.URL.createObjectURL(new Blob([blob]))
    const a = document.createElement('a'); a.href = url; a.download = 'probe_configs.yaml'
    a.click(); window.URL.revokeObjectURL(url)
  } catch {}
}

onMounted(() => { loadData() })
</script>

<style scoped>
.probe-management-container { padding: 0; height: 100%; display: flex; flex-direction: column; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-title-group { display: flex; align-items: center; gap: 12px; }
.page-title-icon { width: 36px; height: 36px; border-radius: 8px; background: var(--ops-primary); display: flex; align-items: center; justify-content: center; color: #fff; font-size: 18px; }
.page-title { margin: 0; font-size: 17px; font-weight: 600; color: var(--ops-text-primary); }
.page-subtitle { margin: 2px 0 0; font-size: 13px; color: var(--ops-text-tertiary); }
.filter-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; }
.stats-row { display: flex; gap: 12px; margin-bottom: 16px; }
.stat-card { flex: 1; padding: 14px; background: #fff; border: 1px solid var(--ops-border-color); border-radius: var(--ops-border-radius-md); text-align: center; }
.stat-card .stat-value { font-size: 22px; font-weight: 600; color: var(--ops-text-primary); }
.stat-card .stat-label { font-size: 12px; color: var(--ops-text-tertiary); margin-top: 4px; }
.result-section-title { font-size: 14px; font-weight: 600; color: var(--ops-text-primary); margin: 16px 0 8px; padding-left: 8px; border-left: 3px solid var(--ops-primary); }
.kv-row { display: flex; align-items: center; gap: 4px; margin-bottom: 2px; font-size: 13px; line-height: 20px; }
.kv-key { color: var(--ops-text-secondary); font-weight: 500; flex-shrink: 0; }
.kv-val { color: var(--ops-text-primary); word-break: break-all; }
.kv-val-truncated { color: var(--ops-text-tertiary); font-style: italic; word-break: break-all; }
.response-body-pre { margin: 0; padding: 8px; background: var(--color-fill-1, #f7f8fa); border-radius: 4px; font-size: 12px; line-height: 1.5; max-height: 300px; overflow: auto; white-space: pre-wrap; word-break: break-all; }
.response-body-pre :deep(.result-collapse-pill) {
  display: inline-flex; align-items: center; gap: 3px;
  background: #fff3e0; color: #d46b08; border: 1px solid #ffd591;
  border-radius: 4px; padding: 0 6px; font-size: 12px; line-height: 20px;
  font-weight: 500; white-space: nowrap; cursor: pointer; transition: background 0.15s;
  vertical-align: middle; margin: 0 1px;
}
.response-body-pre :deep(.result-collapse-pill:hover) { background: #ffe7ba; }
.long-text-pre { margin: 0; padding: 12px; background: var(--color-fill-1, #f7f8fa); border-radius: 4px; font-size: 13px; line-height: 1.6; max-height: 500px; overflow: auto; white-space: pre-wrap; word-break: break-all; }
.workflow-step-item { border: 1px solid var(--ops-border-color); border-radius: 6px; margin-bottom: 8px; background: #fff; }
.workflow-step-item[draggable="true"] { cursor: grab; }
.workflow-step-header { display: flex; align-items: center; gap: 8px; padding: 8px 12px; cursor: pointer; user-select: none; }
.workflow-step-drag-handle { color: var(--ops-text-tertiary); cursor: grab; display: flex; align-items: center; }
.workflow-step-index { width: 22px; height: 22px; border-radius: 50%; background: var(--ops-primary); color: #fff; font-size: 12px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.workflow-step-name { font-size: 14px; font-weight: 500; color: var(--ops-text-primary); }
.workflow-step-body { padding: 8px 16px 12px; border-top: 1px solid var(--ops-border-color); }
.wf-step-result-header { display: flex; align-items: center; gap: 8px; }
</style>
