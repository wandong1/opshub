<template>
  <div class="nodes-container">
    <!-- 统计卡片 -->
    <a-row :gutter="20" class="stats-row">
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-primary, #165dff)">
              <icon-desktop :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeList.length }}</div>
              <div class="stat-label">节点总数</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-success, #00b42a)">
              <icon-check-circle :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ readyNodeCount }}</div>
              <div class="stat-label">运行正常</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-warning, #ff7d00)">
              <icon-dashboard :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ totalPodCount }}</div>
              <div class="stat-label">Pod总数</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: #722ed1">
              <icon-thunderbolt :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ totalCPUCores }}</div>
              <div class="stat-label">总CPU核数</div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 页面标题和操作按钮 -->
    <a-card class="page-header-card">
      <div class="page-header">
        <div class="page-title-group">
          <div class="page-title-icon">
            <icon-desktop />
          </div>
          <div>
            <h2 class="page-title">节点管理</h2>
            <p class="page-subtitle">管理 Kubernetes 集群节点，监控节点状态和资源使用</p>
          </div>
        </div>
        <div class="header-actions">
          <a-select
            v-model="selectedClusterId"
            placeholder="选择集群"
            style="width: 260px"
            @change="handleClusterChange"
          >
            <template #prefix>
              <icon-apps />
            </template>
            <a-option
              v-for="cluster in clusterList"
              :key="cluster.id"
              :label="cluster.alias || cluster.name"
              :value="cluster.id"
            />
          </a-select>
          <a-button type="primary" @click="loadNodes">
            <template #icon><icon-refresh /></template>
            刷新
          </a-button>
        </div>
      </div>
    </a-card>

    <!-- 搜索和筛选 -->
    <a-card class="search-card">
      <a-form layout="inline" class="search-form">
        <a-form-item>
          <a-input
            v-model="searchName"
            placeholder="搜索节点名称..."
            allow-clear
            @clear="handleSearch"
            @keyup.enter="handleSearch"
            @input="handleSearch"
            style="width: 260px"
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item>
          <a-select
            v-model="searchStatus"
            placeholder="节点状态"
            allow-clear
            @change="handleSearch"
            style="width: 140px"
          >
            <a-option label="正常" value="Ready" />
            <a-option label="异常" value="NotReady" />
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-select
            v-model="searchRole"
            placeholder="节点角色"
            allow-clear
            @change="handleSearch"
            style="width: 140px"
          >
            <a-option label="Master" value="master" />
            <a-option label="Control Plane" value="control-plane" />
            <a-option label="Worker" value="worker" />
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button
            type="outline"
            status="warning"
            :loading="cloudttyLoading"
            @click="handleCloudTTY"
          >
            <template #icon><icon-desktop /></template>
            {{ cloudttyInstalled ? '打开 CloudTTY' : '部署 CloudTTY' }}
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- CloudTTY 部署对话框 -->
    <a-modal
      v-model:visible="cloudttyDialogVisible"
      title="部署 CloudTTY"
      width="70%"
      :close-on-click-modal="false"
    >
      <div class="cloudtty-dialog-content">
        <a-alert
          title="CloudTTY 是一个 Kubernetes Web Terminal 解决方案"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        >
          <template #default>
            <p>CloudTTY 可以提供更强大的节点 Shell 功能，支持：</p>
            <ul style="margin: 10px 0; padding-left: 20px">
              <li>完整的终端模拟</li>
              <li>文件上传/下载</li>
              <li>多标签页支持</li>
              <li>会话审计和录制</li>
            </ul>
          </template>
        </a-alert>

        <div v-if="cloudttyDeploying" class="deploy-status">
          <a-progress :percentage="deployProgress" :status="deployStatus" />
          <p style="margin-top: 10px; text-align: center">{{ deployMessage }}</p>
        </div>

        <div v-else class="deploy-methods">
          <h4>部署步骤：</h4>
          <a-steps direction="vertical" :active="1" class="deploy-steps">
            <a-step title="复制下方命令" />
            <a-step title="在控制台执行命令" />
            <a-step title="等待部署完成（约2-3分钟）" />
            <a-step title="点击已完成部署按钮刷新状态" />
          </a-steps>

          <h4 style="margin-top: 20px">部署命令：</h4>
          <div class="code-block-wrapper">
            <div class="code-line-numbers">
              <div v-for="line in cloudttyCommandLines" :key="line" class="code-line-number">{{ line }}</div>
            </div>
            <textarea
              readonly
              :value="cloudttyCommands"
              class="code-textarea"
            ></textarea>
          </div>
          <a-button
            type="primary"
            @click="copyCommands"
            style="margin-top: 10px"
          >
            复制命令
          </a-button>
        </div>
      </div>

      <template #footer>
        <span class="dialog-footer">
          <a-button @click="cloudttyDialogVisible = false">关闭</a-button>
          <a-button
            type="success"
            @click="handleDeployComplete"
          >
            我已完成部署
          </a-button>
        </span>
      </template>
    </a-modal>

    <!-- CloudTTY 终端对话框 -->
    <a-modal
      v-model:visible="cloudttyTerminalVisible"
      :title="`Shell - ${selectedNode?.name || ''}`"
      width="70%"
      :close-on-click-modal="false"
      @close="handleCloseCloudTTY"
      class="cloudtty-terminal-dialog"
    >
      <div class="cloudtty-terminal-wrapper" @click="focusCloudTTYIframe">
        <iframe
          v-if="cloudttyTerminalVisible"
          id="cloudtty-iframe"
          class="cloudtty-iframe"
          frameborder="0"
          allow="clipboard-read; clipboard-write"
        ></iframe>
      </div>
    </a-modal>

    <!-- 批量操作栏 -->
    <div v-if="selectedNodes.length > 0" class="batch-actions-bar">
      <div class="batch-actions-left">
        <a-checkbox
          v-model="selectAllCurrentPage"
          :indeterminate="isIndeterminate"
          @change="handleSelectAllCurrentPage"
        >
          <span class="selected-count">已选择 {{ selectedNodes.length }} 个节点</span>
        </a-checkbox>
      </div>
      <a-space>
        <a-button type="primary" @click="showBatchLabelsDialog">
          批量打标签
        </a-button>
        <a-button status="warning" @click="showBatchTaintsDialog">
          批量设置污点
        </a-button>
        <a-button @click="handleBatchDrain">
          批量排空
        </a-button>
        <a-button status="warning" @click="handleBatchCordon">
          批量不可调度
        </a-button>
        <a-button status="success" @click="handleBatchUncordon">
          批量可调度
        </a-button>
        <a-button status="danger" @click="handleBatchDelete">
          批量删除
        </a-button>
        <a-button @click="clearSelection">取消选择</a-button>
      </a-space>
    </div>

    <!-- 节点列表 -->
    <a-card class="table-card">
      <a-table
        ref="nodeTableRef"
        :data="paginatedNodeList"
        :loading="loading"
        :bordered="false"
        :pagination="false"
        size="default"
        @selection-change="handleSelectionChange"
       :columns="tableColumns4" :row-selection="{ type: 'checkbox', showCheckedAll: true }">
        <template #nodeName="{ record }">
          <div class="node-name-cell">
            <div class="node-icon-wrapper">
              <icon-apps />
            </div>
            <div class="node-name-content">
              <div class="node-name link-text" @click="goToNodeDetail(record)">{{ record.name }}</div>
              <div class="node-ip">{{ record.internalIP }}</div>
            </div>
          </div>
        </template>
          <template #status="{ record }">
          <a-tag
            :type="record.status === 'Ready' ? 'success' : 'danger'"
            effect="dark"
            size="small"
            class="status-tag"
          >
            {{ record.status === 'Ready' ? '正常' : '异常' }}
          </a-tag>
        </template>
          <template #role="{ record }">
          <div :class="['role-badge', 'role-' + (record.roles || 'worker')]">
            <icon-user />
            <span>{{ getRoleText(record.roles) }}</span>
          </div>
        </template>
          <template #kubeletVersion="{ record }">
          <div class="version-cell">
            <icon-info-circle />
            <span class="version-text">{{ record.version || '-' }}</span>
          </div>
        </template>
          <template #labels="{ record }">
          <div class="label-cell" @click="showLabels(record)">
            <div class="label-badge-wrapper">
              <span class="label-count">{{ Object.keys(record.labels || {}).length }}</span>
              <icon-tag />
            </div>
          </div>
        </template>
          <template #uptime="{ record }">
          <div class="age-cell">
            <icon-clock-circle />
            <span>{{ record.age || '-' }}</span>
          </div>
        </template>
          <template #cpu="{ record }">
          <div class="resource-cell">
            <div class="resource-icon resource-icon-cpu">
              <icon-thunderbolt />
            </div>
            <span class="resource-value">{{ formatCPUWithUsage(record) }}</span>
          </div>
        </template>
          <template #memory="{ record }">
          <div class="resource-cell">
            <div class="resource-icon resource-icon-memory">
              <icon-common />
            </div>
            <span class="resource-value">{{ formatMemoryWithUsage(record) }}</span>
          </div>
        </template>
          <template #podCount="{ record }">
          <div class="pod-count-cell">
            <span class="pod-count">{{ record.podCount ?? 0 }}/{{ record.podCapacity ?? 0 }}</span>
            <span class="pod-label">Pods</span>
          </div>
        </template>
          <template #schedulable="{ record }">
          <a-tag
            :type="record.schedulable ? 'success' : 'warning'"
            effect="dark"
            size="small"
          >
            {{ record.schedulable ? '可调度' : '不可' }}
          </a-tag>
        </template>
          <template #taints="{ record }">
          <div class="taint-cell" @click="showTaints(record)">
            <div class="taint-badge-wrapper">
              <icon-exclamation-circle-fill />
              <span class="taint-count">{{ record.taintCount ?? 0 }}</span>
            </div>
          </div>
        </template>
          <template #actions="{ record }">
          <a-dropdown trigger="click" @select="(command: string) => handleActionCommand(command, record)">
            <a-button type="text" class="action-btn">
              <icon-more />
            </a-button>
            <template #content>
                <a-doption value="shell">
                  <icon-desktop />
                  <span>Shell</span>
                </a-doption>
                <a-doption value="monitor">
                  <icon-line-chart />
                  <span>监控</span>
                </a-doption>
                <a-doption value="yaml">
                  <icon-file />
                  <span>YAML</span>
                </a-doption>
                <a-doption value="drain">
                  <icon-close-circle />
                  <span>节点排空</span>
                </a-doption>
                <a-doption value="cordon">
                  <icon-exclamation-circle />
                  <span>设为不可调度</span>
                </a-doption>
                <a-doption value="uncordon">
                  <icon-check-circle />
                  <span>设为可调度</span>
                </a-doption>
                <a-doption value="delete" class="danger-item">
                  <icon-delete />
                  <span>删除</span>
                </a-doption>
            </template>
          </a-dropdown>
        </template>
        </a-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="currentPage"
          v-model:page-size="pageSize"
          :page-size-options="[10, 20, 50, 100]"
          :total="filteredNodeList.length"
          show-total show-page-size show-jumper
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </a-card>

    <!-- 标签弹窗 -->
    <a-modal
      v-model:visible="labelDialogVisible"
      :title="labelEditMode ? '编辑节点标签' : '节点标签'"
      width="850px"
      class="label-dialog"
      :close-on-click-modal="!labelEditMode"
    >
      <div class="label-dialog-content">
        <!-- 编辑模式 -->
        <div v-if="labelEditMode" class="label-edit-container">
          <div class="label-edit-header">
            <div class="label-edit-info">
              <icon-info-circle />
              <span>编辑 {{ selectedNode?.name }} 的标签</span>
            </div>
            <div class="label-edit-count">
              共 {{ editLabelList.length }} 个标签
            </div>
          </div>

          <div class="label-edit-list">
            <div v-for="(label, index) in editLabelList" :key="index" class="label-edit-row">
              <div class="label-row-number">{{ index + 1 }}</div>
              <div class="label-row-content">
                <div class="label-input-group">
                  <div class="label-input-wrapper">
                    <span class="label-input-label">Key</span>
                    <a-input
                      v-model="label.key"
                      placeholder="如: app"
                      size="default"
                      class="label-edit-input"
                    />
                  </div>
                  <span class="label-separator">=</span>
                  <div class="label-input-wrapper">
                    <span class="label-input-label">Value</span>
                    <a-input
                      v-model="label.value"
                      placeholder="可为空"
                      size="default"
                      class="label-edit-input"
                    />
                  </div>
                </div>
              </div>
              <a-button
                type="danger"
               
                size="default"
                @click="removeEditLabel(index)"
                class="remove-btn"
                circle
              />
            </div>
            <div v-if="editLabelList.length === 0" class="empty-labels">
              <icon-tag />
              <p>暂无标签</p>
              <span>点击下方按钮添加新标签</span>
            </div>
          </div>

          <a-button
            type="primary"
           
            @click="addEditLabel"
            class="add-label-btn"
            plain
          >
            添加标签
          </a-button>
        </div>

        <!-- 查看模式 -->
        <a-table v-else :data="labelList" class="label-table" max-height="500" :columns="tableColumns3">
          <template #key="{ record }">
              <div class="label-key-wrapper" @click="copyToClipboard(record.key, 'Key')">
                <span class="label-key-text">{{ record.key }}</span>
                <icon-copy />
              </div>
            </template>
          <template #value="{ record }">
              <span class="label-value">{{ record.value || '-' }}</span>
            </template>
        </a-table>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <template v-if="labelEditMode">
            <a-button @click="cancelLabelEdit" size="large">取消</a-button>
            <a-button type="primary" @click="saveLabels" :loading="labelSaving" size="large" class="save-btn">
              <icon-check />
              {{ labelSaving ? '保存中...' : '保存更改' }}
            </a-button>
          </template>
          <template v-else>
            <a-button @click="labelDialogVisible = false" size="large">关闭</a-button>
            <a-button type="primary" @click="startLabelEdit" size="large" class="edit-btn">
              编辑标签
            </a-button>
          </template>
        </div>
      </template>
    </a-modal>

    <!-- 污点弹窗 -->
    <a-modal
      v-model:visible="taintDialogVisible"
      :title="taintEditMode ? '编辑节点污点' : '节点污点'"
      width="900px"
      class="taint-dialog"
      :close-on-click-modal="!taintEditMode"
    >
      <div class="taint-dialog-content">
        <!-- 编辑模式 -->
        <div v-if="taintEditMode" class="taint-edit-container">
          <div class="taint-edit-header">
            <div class="taint-edit-info">
              <icon-exclamation-circle-fill />
              <span>编辑 {{ selectedNode?.name }} 的污点</span>
            </div>
            <div class="taint-edit-count">
              共 {{ editTaintList.length }} 个污点
            </div>
          </div>

          <div class="taint-edit-list">
            <div v-for="(taint, index) in editTaintList" :key="index" class="taint-edit-row">
              <div class="taint-row-number">{{ index + 1 }}</div>
              <div class="taint-row-content">
                <div class="taint-input-group">
                  <div class="taint-input-wrapper">
                    <span class="taint-input-label">Key</span>
                    <a-input
                      v-model="taint.key"
                      placeholder="如: key1"
                      size="default"
                      class="taint-edit-input"
                    />
                  </div>
                  <span class="taint-separator">=</span>
                  <div class="taint-input-wrapper">
                    <span class="taint-input-label">Value</span>
                    <a-input
                      v-model="taint.value"
                      placeholder="可选"
                      size="default"
                      class="taint-edit-input"
                    />
                  </div>
                  <span class="taint-separator">:</span>
                  <div class="taint-effect-wrapper">
                    <span class="taint-input-label">Effect</span>
                    <a-select
                      v-model="taint.effect"
                      placeholder="选择"
                      size="default"
                      class="taint-effect-select"
                    >
                      <a-option label="NoSchedule" value="NoSchedule">
                        <div class="effect-option">
                          <a-tag color="orangered" size="small">NoSchedule</a-tag>
                          <span class="effect-desc">Pod 不会被调度</span>
                        </div>
                      </a-option>
                      <a-option label="PreferNoSchedule" value="PreferNoSchedule">
                        <div class="effect-option">
                          <a-tag color="gray" size="small">PreferNoSchedule</a-tag>
                          <span class="effect-desc">尽量不调度</span>
                        </div>
                      </a-option>
                      <a-option label="NoExecute" value="NoExecute">
                        <div class="effect-option">
                          <a-tag color="red" size="small">NoExecute</a-tag>
                          <span class="effect-desc">驱逐已有 Pod</span>
                        </div>
                      </a-option>
                    </a-select>
                  </div>
                </div>
              </div>
              <a-button
                type="danger"
               
                size="default"
                @click="removeEditTaint(index)"
                class="remove-btn"
                circle
              />
            </div>
            <div v-if="editTaintList.length === 0" class="empty-taints">
              <icon-exclamation-circle-fill />
              <p>暂无污点</p>
              <span>点击下方按钮添加新污点</span>
            </div>
          </div>

          <a-button
            type="primary"
           
            @click="addEditTaint"
            class="add-taint-btn"
            plain
          >
            添加污点
          </a-button>
        </div>

        <!-- 查看模式 -->
        <a-table v-else :data="taintList" class="taint-table" max-height="500" :columns="tableColumns2">
          <template #key="{ record }">
              <div class="taint-key-wrapper" @click="copyToClipboard(record.key, 'Key')">
                <span class="taint-key-text">{{ record.key }}</span>
                <icon-copy />
              </div>
            </template>
          <template #value="{ record }">
              <span class="taint-value">{{ record.value || '-' }}</span>
            </template>
          <template #effect="{ record }">
              <a-tag :type="getEffectTagType(record.effect)" class="effect-tag">
                {{ record.effect }}
              </a-tag>
            </template>
        </a-table>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <template v-if="taintEditMode">
            <a-button @click="cancelTaintEdit" size="large">取消</a-button>
            <a-button type="primary" @click="saveTaints" :loading="taintSaving" size="large" class="save-btn">
              <icon-check />
              {{ taintSaving ? '保存中...' : '保存更改' }}
            </a-button>
          </template>
          <template v-else>
            <a-button @click="taintDialogVisible = false" size="large">关闭</a-button>
            <a-button type="primary" @click="startTaintEdit" size="large" class="edit-btn">
              编辑污点
            </a-button>
          </template>
        </div>
      </template>
    </a-modal>

    <!-- 批量标签弹窗 -->
    <a-modal
      v-model:visible="batchLabelDialogVisible"
      title="批量设置节点标签"
      width="700px"
      class="batch-label-dialog"
      :close-on-click-modal="false"
    >
      <div class="batch-label-content">
        <a-alert
          title="将对选中的 {{ selectedNodes.length }} 个节点执行标签操作"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        />

        <a-form :model="batchLabelForm" label-width="100px">
          <a-form-item label="操作类型">
            <a-radio-group v-model="batchLabelForm.operation">
              <a-radio value="add">添加标签</a-radio>
              <a-radio value="remove">删除标签</a-radio>
              <a-radio value="replace">替换标签</a-radio>
            </a-radio-group>
            <div class="form-tip">
              添加：仅添加新标签，不覆盖已有标签 | 删除：删除指定标签 | 替换：覆盖指定标签的值
            </div>
          </a-form-item>

          <a-form-item label="标签列表">
            <div class="batch-label-list">
              <div v-for="(label, index) in batchLabelForm.labels" :key="index" class="batch-label-row">
                <a-input
                  v-model="label.key"
                  placeholder="标签键（如: app）"
                  style="width: 200px; margin-right: 10px"
                />
                <span class="label-separator">=</span>
                <a-input
                  v-model="label.value"
                  placeholder="标签值（如: nginx）"
                  style="width: 200px; margin-right: 10px"
                />
                <a-button
                  type="danger"
                 
                  circle
                  size="small"
                  @click="removeBatchLabel(index)"
                />
              </div>
              <a-button
                type="primary"
               
                @click="addBatchLabel"
                plain
                style="width: 100%; margin-top: 10px"
              >
                添加标签
              </a-button>
            </div>
          </a-form-item>
        </a-form>
      </div>

      <template #footer>
        <a-button @click="batchLabelDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleBatchLabels" :loading="batchLabelSaving">
          确定执行
        </a-button>
      </template>
    </a-modal>

    <!-- 批量污点弹窗 -->
    <a-modal
      v-model:visible="batchTaintDialogVisible"
      title="批量设置节点污点"
      width="800px"
      class="batch-taint-dialog"
      :close-on-click-modal="false"
    >
      <div class="batch-taint-content">
        <a-alert
          title="将对选中的 {{ selectedNodes.length }} 个节点执行污点操作"
          type="warning"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        />

        <a-form :model="batchTaintForm" label-width="100px">
          <a-form-item label="操作类型">
            <a-radio-group v-model="batchTaintForm.operation">
              <a-radio value="add">添加污点</a-radio>
              <a-radio value="remove">删除污点</a-radio>
            </a-radio-group>
          </a-form-item>

          <a-form-item label="污点列表">
            <div class="batch-taint-list">
              <div v-for="(taint, index) in batchTaintForm.taints" :key="index" class="batch-taint-row">
                <a-input
                  v-model="taint.key"
                  placeholder="键（如: key1）"
                  style="width: 150px; margin-right: 8px"
                />
                <span class="taint-separator">=</span>
                <a-input
                  v-model="taint.value"
                  placeholder="值（可选）"
                  style="width: 150px; margin-right: 8px"
                />
                <span class="taint-separator">:</span>
                <a-select
                  v-model="taint.effect"
                  placeholder="Effect"
                  style="width: 150px; margin-right: 8px"
                >
                  <a-option label="NoSchedule" value="NoSchedule" />
                  <a-option label="PreferNoSchedule" value="PreferNoSchedule" />
                  <a-option label="NoExecute" value="NoExecute" />
                </a-select>
                <a-button
                  type="danger"
                 
                  circle
                  size="small"
                  @click="removeBatchTaint(index)"
                />
              </div>
              <a-button
                type="primary"
               
                @click="addBatchTaint"
                plain
                style="width: 100%; margin-top: 10px"
              >
                添加污点
              </a-button>
            </div>
          </a-form-item>
        </a-form>
      </div>

      <template #footer>
        <a-button @click="batchTaintDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleBatchTaints" :loading="batchTaintSaving">
          确定执行
        </a-button>
      </template>
    </a-modal>

    <!-- 批量操作结果弹窗 -->
    <a-modal
      v-model:visible="batchResultDialogVisible"
      title="批量操作结果"
      width="700px"
      class="batch-result-dialog"
    >
      <div class="batch-result-content">
        <div class="result-summary">
          <a-tag :type="getBatchResultSummary().successCount === getBatchResultSummary().totalCount ? 'success' : 'warning'" size="large">
            成功: {{ getBatchResultSummary().successCount }} / {{ getBatchResultSummary().totalCount }}
          </a-tag>
        </div>
        <a-table :data="batchResults" max-height="400" class="result-table" :columns="tableColumns">
          <template #status="{ record }">
              <a-tag :type="record.success ? 'success' : 'danger'" size="small">
                {{ record.success ? '成功' : '失败' }}
              </a-tag>
            </template>
        </a-table>
      </div>
      <template #footer>
        <a-button type="primary" @click="batchResultDialogVisible = false">关闭</a-button>
      </template>
    </a-modal>

    <!-- YAML 编辑弹窗 -->
    <a-modal
      v-model:visible="yamlDialogVisible"
      :title="`节点 YAML - ${selectedNode?.name || ''}`"
      width="900px"
      class="yaml-dialog"
    >
      <div class="yaml-dialog-content">
        <div class="yaml-editor-wrapper">
          <div class="yaml-line-numbers">
            <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
          </div>
          <textarea
            v-model="yamlContent"
            class="yaml-textarea"
            placeholder="YAML 内容"
            spellcheck="false"
            @input="handleYamlInput"
            @scroll="handleYamlScroll"
            ref="yamlTextarea"
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="yamlDialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSaveYAML" :loading="yamlSaving">
            保存
          </a-button>
        </div>
      </template>
    </a-modal>

    <!-- Shell 终端弹窗 -->
    <a-modal
      v-model:visible="shellDialogVisible"
      :title="`Shell - ${selectedNode?.name || ''}`"
      width="70%"
      class="shell-dialog"
      @close="handleCloseShell"
      @opened="handleShellOpened"
    >
      <div class="shell-dialog-content">
        <div ref="terminalRef" class="terminal-container"></div>
      </div>
      <template #footer>
        <span></span>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns4 = [
  { title: '节点名称', slotName: 'nodeName', width: 180, fixed: 'left' },
  { title: '状态', slotName: 'status', width: 90, align: 'center' },
  { title: '角色', slotName: 'role', width: 100, align: 'center' },
  { title: 'Kubelet版本', slotName: 'kubeletVersion', width: 110 },
  { title: '标签', slotName: 'labels', width: 85, align: 'center' },
  { title: '运行时间', slotName: 'uptime', width: 110 },
  { title: 'CPU', slotName: 'cpu', width: 130 },
  { title: '内存', slotName: 'memory', width: 130 },
  { title: 'Pod数量', slotName: 'podCount', width: 95, align: 'center' },
  { title: '调度', slotName: 'schedulable', width: 90, align: 'center' },
  { title: '污点', slotName: 'taints', width: 90, align: 'center' },
  { title: '操作', slotName: 'actions', width: 75, fixed: 'right', align: 'center' }
]

const tableColumns3 = [
  { title: 'Key', dataIndex: 'key', slotName: 'key', width: 200 },
  { title: 'Value', dataIndex: 'value', slotName: 'value', width: 200 }
]

const tableColumns2 = [
  { title: 'Key', dataIndex: 'key', slotName: 'key', width: 200 },
  { title: 'Value', dataIndex: 'value', slotName: 'value', width: 200 },
  { title: 'Effect', dataIndex: 'effect', slotName: 'effect', width: 120, align: 'center' }
]

const tableColumns = [
  { title: '节点名称', dataIndex: 'nodeName', width: 200 },
  { title: '状态', slotName: 'status', width: 100, align: 'center' },
  { title: '消息', dataIndex: 'message', width: 200 }
]

import { ref, onMounted, computed, onUnmounted, nextTick, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Message, Modal } from '@arco-design/web-vue'
import axios from 'axios'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { getClusterList, type Cluster, getNodes, type NodeInfo } from '@/api/kubernetes'

const loading = ref(false)
const router = useRouter()
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const nodeList = ref<NodeInfo[]>([])

// 批量操作相关
const nodeTableRef = ref()
const selectedNodes = ref<NodeInfo[]>([])
const selectAllCurrentPage = ref(false)
const isIndeterminate = ref(false)
const batchLabelDialogVisible = ref(false)
const batchTaintDialogVisible = ref(false)
const batchResultDialogVisible = ref(false)
const batchLabelSaving = ref(false)
const batchTaintSaving = ref(false)
const batchResults = ref<{ nodeName: string; success: boolean; message: string }[]>([])

const batchLabelForm = ref({
  operation: 'add',
  labels: [] as { key: string; value: string }[]
})

const batchTaintForm = ref({
  operation: 'add',
  taints: [] as { key: string; value: string; effect: string }[]
})

// 搜索条件
const searchName = ref('')
const searchStatus = ref('')
const searchRole = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)
const paginationStorageKey = ref('nodes_pagination')

// 标签弹窗
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])
const labelEditMode = ref(false)
const labelSaving = ref(false)
const editLabelList = ref<{ key: string; value: string }[]>([])
const labelOriginalYaml = ref('')

// 污点弹窗
const taintDialogVisible = ref(false)
const taintList = ref<{ key: string; value: string; effect: string }[]>([])
const taintEditMode = ref(false)
const taintSaving = ref(false)
const editTaintList = ref<{ key: string; value: string; effect: string }[]>([])
const taintOriginalYaml = ref('')

// YAML 编辑弹窗
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlSaving = ref(false)
const selectedNode = ref<NodeInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// Shell 终端弹窗
const shellDialogVisible = ref(false)
const terminalRef = ref<HTMLElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null

// CloudTTY 相关状态
const cloudttyInstalled = ref(false)
const cloudttyLoading = ref(false)
const cloudttyDialogVisible = ref(false)
const cloudttyTerminalVisible = ref(false)
const cloudttyDeploying = ref(false)
const deployProgress = ref(0)
const deployStatus = ref<'success' | 'exception' | ''>('')
const deployMessage = ref('')
const deployMethod = ref('auto')
const cloudttyCommands = ref(`#1、安装并等待 Pod 运行起来
helm repo add cloudtty https://cloudtty.github.io/cloudtty
helm repo update
helm install cloudtty-operator cloudtty/cloudtty \\
  --version 0.5.0 \\
  --create-namespace \\
  --namespace cloudtty-system

#2、创建 cloudshell.yaml
cat <<EOF > cloudshell.yaml
apiVersion: cloudshell.cloudtty.io/v1alpha1
kind: CloudShell
metadata:
  name: permanent-terminal
  namespace: cloudtty-system
spec:
  # 命令 - 使用交互式 bash
  commandAction: "bash -il"
  # 暴露方式
  exposureMode: NodePort
  # 单次连接模式 - false 表示允许多次连接
  once: false
  # 不自动清理
  cleanup: false
  ttlSecondsAfterStarted: 315360000
  # 允许 URL 参数
  urlArg: true
  # 环境变量 - 配置 ttyd 参数
  env:
  - name: TTYD_WRITABLE
    value: "true"
  - name: TTYD_SERVER_BUFFER_SIZE
    value: "4096"
  - name: TTYD_CLIENT_BUFFER_SIZE
    value: "4096"
  # ttyd 客户端选项
  ttydClientOptions:
    fontSize: "14"
    fontFamily: "Monaco, Menlo, Consolas, 'Courier New', monospace"
    cursorBlink: "true"
    rendererType: "canvas"
  # 镜像
  image: "cloudshell/cloudshell:latest"
EOF

kubectl apply -f cloudshell.yaml

#3、观察 CR 状态，获取访问接入点
kubectl get cloudshell -w`)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 计算CloudTTY命令行数
const cloudttyCommandLines = computed(() => {
  if (!cloudttyCommands.value) return 1
  return cloudttyCommands.value.split('\n').length
})

// 过滤后的节点列表
const filteredNodeList = computed(() => {
  let result = nodeList.value

  if (searchName.value) {
    result = result.filter(node =>
      node.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (searchStatus.value) {
    result = result.filter(node => node.status === searchStatus.value)
  }

  if (searchRole.value) {
    result = result.filter(node => node.roles === searchRole.value)
  }

  return result
})

// 分页后的节点列表
const paginatedNodeList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredNodeList.value.slice(start, end)
})

// 统计数据
const readyNodeCount = computed(() => {
  return nodeList.value.filter(node => node.status === 'Ready').length
})

const totalPodCount = computed(() => {
  const usedPods = nodeList.value.reduce((sum, node) => sum + (node.podCount || 0), 0)
  const totalPods = nodeList.value.reduce((sum, node) => sum + (node.podCapacity || 0), 0)
  return `${usedPods}/${totalPods}`
})

const totalCPUCores = computed(() => {
  let totalCores = 0
  nodeList.value.forEach(node => {
    if (node.cpuCapacity) {
      const cores = parseCPU(node.cpuCapacity)
      totalCores += cores
    }
  })
  return totalCores.toFixed(1)
})

// 解析 CPU 核数
const parseCPU = (cpu: string): number => {
  if (!cpu) return 0
  if (cpu.endsWith('m')) {
    return parseInt(cpu) / 1000
  }
  return parseFloat(cpu) || 0
}

// 格式化 CPU 显示
const formatCPU = (cpu: string) => {
  if (!cpu) return '-'
  if (cpu.endsWith('m')) {
    const millicores = parseInt(cpu)
    if (isNaN(millicores)) return cpu
    return (millicores / 1000).toFixed(2) + ' 核'
  }
  return cpu + ' 核'
}

// 格式化内存显示
const formatMemory = (memory: string) => {
  if (!memory) return '-'

  const match = memory.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)?$/i)
  if (!match) return memory

  const value = parseFloat(match[1])
  const unit = match[2]?.toUpperCase()

  if (!unit) {
    const bytes = value
    const tb = bytes / (1024 * 1024 * 1024 * 1024)
    if (tb >= 1) return Math.ceil(tb) + ' TB'
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return Math.ceil(gb) + ' GB'
    const mb = bytes / (1024 * 1024)
    if (mb >= 1) return Math.ceil(mb) + ' MB'
    return memory
  }

  let bytes = 0
  switch (unit) {
    case 'KI':
      bytes = value * 1024
      break
    case 'MI':
      bytes = value * 1024 * 1024
      break
    case 'GI':
      bytes = value * 1024 * 1024 * 1024
      break
    case 'TI':
      bytes = value * 1024 * 1024 * 1024 * 1024
      break
  }

  const tb = bytes / (1024 * 1024 * 1024 * 1024)
  if (tb >= 1) return Math.ceil(tb) + ' TB'

  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return Math.ceil(gb) + ' GB'

  const mb = bytes / (1024 * 1024)
  if (mb >= 1) return Math.ceil(mb) + ' MB'

  return memory
}

// 格式化 CPU 显示（包含使用量）
const formatCPUWithUsage = (node: NodeInfo) => {
  const usedCores = (node.cpuUsed || 0) / 1000 // 毫核转核
  const totalCores = parseCPU(node.cpuCapacity)

  const used = usedCores.toFixed(1) // 已使用保留1位小数
  const total = Math.round(totalCores) // 总数不保留小数

  return `${used}/${total}核`
}

// 格式化内存显示（包含使用量）
const formatMemoryWithUsage = (node: NodeInfo) => {
  const usedBytes = node.memoryUsed || 0

  // 解析总内存
  const match = node.memoryCapacity.match(/^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)?$/i)
  if (!match) return '-'

  const value = parseFloat(match[1])
  const unit = match[2]?.toUpperCase()

  let totalBytes = 0
  if (!unit) {
    totalBytes = value
  } else {
    switch (unit) {
      case 'KI':
        totalBytes = value * 1024
        break
      case 'MI':
        totalBytes = value * 1024 * 1024
        break
      case 'GI':
        totalBytes = value * 1024 * 1024 * 1024
        break
      case 'TI':
        totalBytes = value * 1024 * 1024 * 1024 * 1024
        break
    }
  }

  // 转换为GB
  const usedGB = usedBytes / (1024 * 1024 * 1024)
  const totalGB = totalBytes / (1024 * 1024 * 1024)

  const used = usedGB >= 1 ? usedGB.toFixed(1) : (usedBytes / (1024 * 1024)).toFixed(1)
  const total = totalGB >= 1 ? Math.ceil(totalGB) + 'G' : Math.ceil(totalBytes / (1024 * 1024)) + 'M'

  return `内存:${used}/${total}`
}

// 获取角色文本
const getRoleText = (role: string | undefined) => {
  if (!role) return 'Worker'
  if (role === 'master') return 'Master'
  if (role === 'control-plane') return 'Control Plane'
  if (role === 'worker') return 'Worker'
  return role
}

// 获取 Effect 标签类型
const getEffectTagType = (effect: string) => {
  switch (effect) {
    case 'NoSchedule':
      return 'warning'
    case 'NoExecute':
      return 'danger'
    case 'PreferNoSchedule':
      return 'info'
    default:
      return ''
  }
}

// 显示标签弹窗
const showLabels = (row: NodeInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelEditMode.value = false
  selectedNode.value = row
  labelDialogVisible.value = true
}

// 开始编辑标签
const startLabelEdit = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')
    const nodeName = selectedNode.value.name

    // 获取节点当前 YAML
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml?clusterId=${selectedClusterId.value}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    labelOriginalYaml.value = response.data.data?.yaml || ''

    // 复制当前标签到编辑列表
    editLabelList.value = labelList.value.map(label => ({
      key: label.key,
      value: label.value
    }))

    labelEditMode.value = true
  } catch (error) {
    Message.error('获取节点信息失败')
  }
}

// 取消编辑标签
const cancelLabelEdit = () => {
  labelEditMode.value = false
  editLabelList.value = []
}

// 添加编辑标签
const addEditLabel = () => {
  editLabelList.value.push({ key: '', value: '' })
}

// 删除编辑标签
const removeEditLabel = (index: number) => {
  editLabelList.value.splice(index, 1)
}

// 保存标签
const saveLabels = async () => {
  if (!selectedNode.value) return

  // 验证标签 - 只验证 key，value 可以为空
  const validLabels = editLabelList.value.filter(label => label.key.trim() !== '')
  if (validLabels.some(label => !label.key)) {
    Message.warning('标签键不能为空')
    return
  }

  // 检查是否有重复的键
  const keys = validLabels.map(l => l.key)
  const uniqueKeys = new Set(keys)
  if (keys.length !== uniqueKeys.size) {
    Message.warning('存在重复的标签键，请检查')
    return
  }

  labelSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const nodeName = selectedNode.value.name

    // 判断是否为系统标签
    const isSystemLabel = (key: string) => {
      return key.startsWith('kubernetes.io/') ||
             key.startsWith('node-role.') ||
             key.startsWith('node.kubernetes.io/') ||
             key.startsWith('beta.kubernetes.io/')
    }

    // 从 editLabelList 中分离系统标签和用户标签
    const systemLabels: { key: string; value: string }[] = []
    const userLabels: { key: string; value: string }[] = []

    validLabels.forEach(l => {
      if (isSystemLabel(l.key)) {
        systemLabels.push(l)
      } else {
        userLabels.push(l)
      }
    })

    // 合并系统标签和用户标签
    const allLabels = [...systemLabels, ...userLabels]

    // 构建包含所有标签的 YAML
    const labelsStr = allLabels
      .map(l => {
        if (l.value === '') {
          return `    ${l.key}: ""`
        }
        return `    ${l.key}: ${l.value}`
      })
      .join('\n')

    const labelsYaml = `apiVersion: v1
kind: Node
metadata:
  name: ${nodeName}
  labels:
${labelsStr}
`

    // 调用 API 保存
    const response = await axios.put(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml`,
      {
        clusterId: selectedClusterId.value,
        yaml: labelsYaml
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('标签保存成功')
    labelEditMode.value = false
    // 刷新节点列表
    await loadNodes()
    // 从刷新后的节点数据中重新获取标签
    const updatedNode = nodeList.value.find(n => n.name === nodeName)
    if (updatedNode) {
      selectedNode.value = updatedNode
      labelList.value = Object.keys(updatedNode.labels || {}).map(key => ({
        key,
        value: updatedNode.labels![key]
      }))
    }
  } catch (error: any) {
    Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    labelSaving.value = false
  }
}

// 跳转到节点详情页
const goToNodeDetail = (row: NodeInfo) => {
  const cluster = clusterList.value.find(c => c.id === selectedClusterId.value)
  router.push({
    name: 'K8sNodeDetail',
    params: {
      clusterId: selectedClusterId.value,
      nodeName: row.name
    },
    query: {
      clusterName: cluster?.alias || cluster?.name
    }
  })
}

// 显示污点弹窗
const showTaints = (row: NodeInfo) => {
  const taints = row.taints || []
  taintList.value = taints.map(taint => ({
    key: taint.key,
    value: taint.value || '',
    effect: taint.effect
  }))
  // 重置编辑模式
  taintEditMode.value = false
  selectedNode.value = row
  taintDialogVisible.value = true
}

// 开始编辑污点
const startTaintEdit = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')
    const nodeName = selectedNode.value.name

    // 获取节点当前 YAML
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml?clusterId=${selectedClusterId.value}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    taintOriginalYaml.value = response.data.data?.yaml || ''

    // 复制当前污点到编辑列表
    editTaintList.value = taintList.value.map(taint => ({
      key: taint.key,
      value: taint.value || '',
      effect: taint.effect
    }))

    taintEditMode.value = true
  } catch (error) {
    Message.error('获取节点信息失败')
  }
}

// 取消编辑污点
const cancelTaintEdit = () => {
  taintEditMode.value = false
  editTaintList.value = []
}

// 添加编辑污点
const addEditTaint = () => {
  editTaintList.value.push({ key: '', value: '', effect: 'NoSchedule' })
}

// 删除编辑污点
const removeEditTaint = (index: number) => {
  editTaintList.value.splice(index, 1)
}

// 保存污点
const saveTaints = async () => {
  if (!selectedNode.value) return

  // 验证污点
  const validTaints = editTaintList.value.filter(taint => taint.key.trim() !== '')
  if (validTaints.some(taint => !taint.key || !taint.effect)) {
    Message.warning('请填写完整的污点键和Effect')
    return
  }

  // 检查是否有重复的键
  const keys = validTaints.map(t => t.key)
  const uniqueKeys = new Set(keys)
  if (keys.length !== uniqueKeys.size) {
    Message.warning('存在重复的污点键，请检查')
    return
  }

  taintSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const nodeName = selectedNode.value.name

    // 将 YAML 按行分割处理
    const lines = taintOriginalYaml.value.split('\n')
    let updatedLines: string[] = []
    let i = 0
    let inSpec = false
    let taintsFound = false

    while (i < lines.length) {
      const line = lines[i]

      // 检测 spec: 开始
      if (/^spec:\s*$/.test(line)) {
        inSpec = true
        updatedLines.push(line)
        i++
        // 处理 spec 下的内容
        while (i < lines.length) {
          const specLine = lines[i]

          // 检测 taints: 开始（在 spec 下的 2 空格缩进）
          if (/^  taints:\s*$/.test(specLine)) {
            taintsFound = true
            // 如果有污点，保留 taints 并添加新内容
            if (validTaints.length > 0) {
              updatedLines.push(specLine)
              // 添加污点内容（列表项2空格，属性4空格）
              for (const taint of validTaints) {
                if (taint.value) {
                  updatedLines.push(`  - key: ${taint.key}`)
                  updatedLines.push(`    value: ${taint.value}`)
                  updatedLines.push(`    effect: ${taint.effect}`)
                } else {
                  updatedLines.push(`  - key: ${taint.key}`)
                  updatedLines.push(`    effect: ${taint.effect}`)
                }
              }
            }
            // 跳过原有的污点内容
            i++
            // 跳过所有污点条目（2 空格缩进的 "- " 开头）
            while (i < lines.length && /^  -\s/.test(lines[i])) {
              i++
              // 跳过污点的属性行（4 空格缩进）
              while (i < lines.length && /^    /.test(lines[i])) {
                i++
              }
            }
            continue
          }

          // 如果遇到新的2空格缩进字段，且还没找到taints，则在这里插入
          if (!taintsFound && /^  [a-z]/.test(specLine)) {
            // 先添加 taints（如果有）
            if (validTaints.length > 0) {
              updatedLines.push(`  taints:`)
              for (const taint of validTaints) {
                if (taint.value) {
                  updatedLines.push(`  - key: ${taint.key}`)
                  updatedLines.push(`    value: ${taint.value}`)
                  updatedLines.push(`    effect: ${taint.effect}`)
                } else {
                  updatedLines.push(`  - key: ${taint.key}`)
                  updatedLines.push(`    effect: ${taint.effect}`)
                }
              }
            }
            taintsFound = true
          }

          // 添加当前行
          updatedLines.push(specLine)
          i++
        }
        // 如果 spec 下没有其他字段且没有找到 taints，添加 taints
        if (!taintsFound && validTaints.length > 0) {
          updatedLines.push(`  taints:`)
          for (const taint of validTaints) {
            if (taint.value) {
              updatedLines.push(`  - key: ${taint.key}`)
              updatedLines.push(`    value: ${taint.value}`)
              updatedLines.push(`    effect: ${taint.effect}`)
            } else {
              updatedLines.push(`  - key: ${taint.key}`)
              updatedLines.push(`    effect: ${taint.effect}`)
            }
          }
        }

        inSpec = false
        continue
      }

      // 如果不在 spec 中，直接添加行
      if (!inSpec) {
        updatedLines.push(line)
      }
      i++
    }

    const updatedYaml = updatedLines.join('\n')

    // 调用 API 保存
    await axios.put(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml`,
      {
        clusterId: selectedClusterId.value,
        yaml: updatedYaml
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('污点保存成功')
    taintEditMode.value = false
    // 刷新节点列表
    await loadNodes()
    // 更新当前选中的节点为最新数据
    const updatedNode = nodeList.value.find(n => n.name === nodeName)
    if (updatedNode) {
      selectedNode.value = updatedNode
      // 更新当前显示的污点列表
      taintList.value = (updatedNode.taints || []).map(t => ({
        key: t.key,
        value: t.value,
        effect: t.effect
      }))
    } else {
      taintList.value = validTaints
    }
  } catch (error: any) {
    Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    taintSaving.value = false
  }
}

// 复制到剪贴板
const copyToClipboard = async (text: string, type: string) => {
  try {
    await navigator.clipboard.writeText(text)
    Message.success(`${type} 已复制到剪贴板`)
  } catch (error) {
    // 降级方案：使用传统方法
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      Message.success(`${type} 已复制到剪贴板`)
    } catch (err) {
      Message.error('复制失败')
    }
    document.body.removeChild(textarea)
  }
}

// 保存分页状态到 localStorage
const savePaginationState = () => {
  try {
    localStorage.setItem(paginationStorageKey.value, JSON.stringify({
      currentPage: currentPage.value,
      pageSize: pageSize.value
    }))
  } catch (error) {
    // 保存分页状态失败
  }
}

// 从 localStorage 恢复分页状态
const restorePaginationState = () => {
  try {
    const saved = localStorage.getItem(paginationStorageKey.value)
    if (saved) {
      const state = JSON.parse(saved)
      currentPage.value = state.currentPage || 1
      pageSize.value = state.pageSize || 10
    }
  } catch (error) {
    currentPage.value = 1
    pageSize.value = 10
  }
}

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
  savePaginationState()
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  const maxPage = Math.ceil(filteredNodeList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
  savePaginationState()
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('nodes_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNodes()
    }
  } catch (error) {
    Message.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('nodes_selected_cluster_id', selectedClusterId.value.toString())
  }
  // 切换集群时重置分页
  currentPage.value = 1
  await loadNodes()
}

// 加载节点列表
const loadNodes = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const data = await getNodes(selectedClusterId.value)
    nodeList.value = data || []
    // 恢复分页状态
    restorePaginationState()
  } catch (error) {
    nodeList.value = []
    Message.error('获取节点列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  // 搜索时重置到第一页
  currentPage.value = 1
  savePaginationState()
}

// 查看详情
const handleViewDetails = (row: NodeInfo) => {
  Message.info('详情功能开发中...')
}

// 处理下拉菜单命令
const handleActionCommand = (command: string, row: NodeInfo) => {
  selectedNode.value = row

  switch (command) {
    case 'shell':
      handleShell()
      break
    case 'monitor':
      Message.info('监控功能开发中...')
      break
    case 'yaml':
      handleShowYAML()
      break
    case 'drain':
      handleDrainNode()
      break
    case 'cordon':
      handleCordonNode()
      break
    case 'uncordon':
      handleUncordonNode()
      break
    case 'delete':
      handleDeleteNode()
      break
    case 'schedule':
      Message.info('调度设置功能开发中...')
      break
  }
}

// 节点排空
const handleDrainNode = async () => {
  if (!selectedNode.value) return

  try {
    await confirmModal(
      `确定要排空节点 ${selectedNode.value.name} 吗？这将会驱逐该节点上的所有 Pod。`,
      '节点排空确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    // 用户确认后执行排空
    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/drain`,
      {
        clusterId: selectedClusterId.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('节点排空成功')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`节点排空失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 设为不可调度
const handleCordonNode = async () => {
  if (!selectedNode.value) return

  try {
    await confirmModal(
      `确定要将节点 ${selectedNode.value.name} 设为不可调度吗？该节点将不再接受新的Pod调度。`,
      '设为不可调度确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/cordon`,
      {
        clusterId: selectedClusterId.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('节点已设为不可调度')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`设为不可调度失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 设为可调度
const handleUncordonNode = async () => {
  if (!selectedNode.value) return

  try {
    await confirmModal(
      `确定要将节点 ${selectedNode.value.name} 设为可调度吗？该节点将重新接受新的Pod调度。`,
      '设为可调度确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info',
      }
    )

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/uncordon`,
      {
        clusterId: selectedClusterId.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('节点已设为可调度')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`设为可调度失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 删除节点
const handleDeleteNode = async () => {
  if (!selectedNode.value) return

  try {
    await confirmModal(
      `确定要删除节点 ${selectedNode.value.name} 吗？此操作不可恢复！`,
      '删除节点确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error',
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}?clusterId=${selectedClusterId.value}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('节点删除成功')
    await loadNodes()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`删除节点失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 显示 YAML 编辑器
const handleShowYAML = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const nodeName = selectedNode.value.name

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/nodes/${nodeName}/yaml?clusterId=${clusterId}`,
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    yamlContent.value = response.data.data?.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    Message.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (!selectedNode.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    await axios.put(
      `/api/v1/plugins/kubernetes/resources/nodes/${selectedNode.value.name}/yaml`,
      {
        clusterId: selectedClusterId.value,
        yaml: yamlContent.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    Message.success('保存成功')
    yamlDialogVisible.value = false
    await loadNodes()
  } catch (error) {
    Message.error('保存 YAML 失败')
  } finally {
    yamlSaving.value = false
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 输入时自动调整滚动
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 打开 Shell 终端
const handleShell = async () => {
  if (!selectedNode.value) return

  try {
    const token = localStorage.getItem('token')

    // 先获取CloudTTY的Service信息
    const serviceResponse = await axios.get(
      `/api/v1/plugins/kubernetes/cloudtty/service`,
      {
        params: {
          clusterId: selectedClusterId.value
        },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    if (serviceResponse.data.code !== 0) {
      Message.error('CloudTTY服务未找到，请先部署CloudTTY')
      return
    }

    const service = serviceResponse.data.data
    if (!service) {
      Message.error('CloudTTY服务信息不完整，请检查CloudTTY是否正确部署')
      return
    }
    const nodeIp = service.nodeIP || selectedNode.value.internalIP
    const port = service.port || 30000
    const path = service.path || '/cloudtty'

    // 构建CloudTTY访问地址
    // CloudTTY NodePort 模式下，直接访问服务地址即可
    const cloudttyUrl = `http://${nodeIp}:${port}/`

    // 打开 CloudTTY 终端对话框
    cloudttyTerminalVisible.value = true

    nextTick(() => {
      const iframe = document.getElementById('cloudtty-iframe') as HTMLIFrameElement
      if (iframe) {
        iframe.src = cloudttyUrl
        // 添加焦点处理，确保 iframe 可以接收键盘输入
        iframe.addEventListener('load', () => {
          try {
            iframe.contentWindow?.focus()
          } catch (e) {
            // 无法设置 iframe 焦点
          }
        })
      }
    })
  } catch (error: any) {
    Message.error('无法连接到CloudTTY服务: ' + (error.response?.data?.message || error.message))
  }
}

// 关闭 CloudTTY 终端
const handleCloseCloudTTY = () => {
  const iframe = document.getElementById('cloudtty-iframe') as HTMLIFrameElement
  if (iframe) {
    iframe.src = '' // 清空iframe以停止加载
  }
  cloudttyTerminalVisible.value = false
}

// 聚焦 CloudTTY iframe
const focusCloudTTYIframe = () => {
  const iframe = document.getElementById('cloudtty-iframe') as HTMLIFrameElement
  if (iframe && iframe.contentWindow) {
    try {
      iframe.contentWindow.focus()
    } catch (e) {
      // 无法聚焦 iframe
    }
  }
}

// Shell 终端初始化
const handleShellOpened = async () => {
  await nextTick()
  const container = terminalRef.value
  if (!container || !selectedNode.value) return

  // 清空容器
  container.innerHTML = ''

  // 创建终端实例
  terminal = new Terminal({
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#aeafad',
      selection: '#264f78'
    },
    fontFamily: 'Monaco, Menlo, Courier New, monospace',
    fontSize: 14,
    lineHeight: 1.2,
    cursorBlink: true,
    scrollback: 1000
  })

  // 加载插件
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  // 打开终端
  terminal.open(container)
  fitAddon.fit()

  // 建立WebSocket连接
  const token = localStorage.getItem('token')
  const wsUrl = `ws://localhost:9876/api/v1/plugins/kubernetes/shell/nodes/${selectedNode.value.name}?clusterId=${selectedClusterId.value}&token=${token}`

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    terminal.writeln('连接成功...\r\n')
  }

  ws.onmessage = (event) => {
    terminal.write(event.data)
  }

  ws.onerror = (error) => {
    terminal.writeln('\r\n\x1b[31m连接错误\x1b[0m')
  }

  ws.onclose = () => {
    terminal.writeln('\r\n\x1b[33m连接已关闭\x1b[0m')
  }

  // 监听终端输入
  terminal.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  // 监听窗口大小变化
  const resizeObserver = new ResizeObserver(() => {
    if (fitAddon) {
      fitAddon.fit()
    }
  })
  resizeObserver.observe(container)
}

// 关闭 Shell 终端
const handleCloseShell = () => {
  if (ws) {
    ws.close()
    ws = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  if (fitAddon) {
    fitAddon = null
  }
  shellDialogVisible.value = false
}

// 检查 CloudTTY 是否已安装
const checkCloudTTY = async () => {
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/cloudtty/status`,
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    cloudttyInstalled.value = response.data.data?.installed || false
  } catch (error) {
    // CloudTTY check failed
  }
}

// 处理 CloudTTY 按钮
const handleCloudTTY = () => {
  cloudttyDialogVisible.value = true
}

// 开始部署
const startDeploy = async () => {
  // 自动部署改为提示用户使用手动部署
  deployMethod.value = 'manual'
  await copyCommands()
  Message.success('命令已复制，请在控制台执行')
}

// 复制命令
const copyCommands = () => {
  navigator.clipboard.writeText(cloudttyCommands.value)
  Message.success('命令已复制到剪贴板')
}

// 处理部署完成
const handleDeployComplete = async () => {
  cloudttyLoading.value = true
  await checkCloudTTY()
  cloudttyLoading.value = false

  if (cloudttyInstalled.value) {
    Message.success('CloudTTY 部署成功！')
    cloudttyDialogVisible.value = false
  } else {
    Message.warning('未检测到 CloudTTY，请确认部署已完成')
  }
}

// 打开 CloudTTY (已废弃，使用对话框替代)
const openCloudTTY = () => {
  Message.info('请手动部署 CloudTTY 或使用自动部署功能')
}

// 批量操作相关函数
// 处理选择变化
const handleSelectionChange = (selection: NodeInfo[]) => {
  selectedNodes.value = selection
  updateSelectAllStatus()
}

// 更新全选状态
const updateSelectAllStatus = () => {
  const currentPageCount = paginatedNodeList.value.length
  const selectedCount = selectedNodes.value.length

  if (selectedCount === 0) {
    selectAllCurrentPage.value = false
    isIndeterminate.value = false
  } else if (selectedCount === currentPageCount) {
    selectAllCurrentPage.value = true
    isIndeterminate.value = false
  } else {
    selectAllCurrentPage.value = false
    isIndeterminate.value = true
  }
}

// 处理当前页全选
const handleSelectAllCurrentPage = (checked: boolean) => {
  if (checked) {
    // 添加当前页所有节点到已选择列表（去重）
    const currentPageNames = new Set(selectedNodes.value.map(n => n.name))
    paginatedNodeList.value.forEach(node => {
      if (!currentPageNames.has(node.name)) {
        selectedNodes.value.push(node)
      }
    })
  } else {
    // 移除当前页的节点
    const currentPageNames = new Set(paginatedNodeList.value.map(n => n.name))
    selectedNodes.value = selectedNodes.value.filter(n => !currentPageNames.has(n.name))
  }
  updateSelectAllStatus()
  // 同步表格选择状态
  syncTableSelection()
}

// 同步表格选择状态
const syncTableSelection = () => {
  if (nodeTableRef.value) {
    const selectedNames = new Set(selectedNodes.value.map(n => n.name))
    paginatedNodeList.value.forEach(row => {
      const isSelected = selectedNames.has(row.name)
      nodeTableRef.value.toggleRowSelection(row, isSelected)
    })
  }
}

// 清除选择
const clearSelection = () => {
  selectedNodes.value = []
  selectAllCurrentPage.value = false
  isIndeterminate.value = false
  if (nodeTableRef.value) {
    nodeTableRef.value.clearSelection()
  }
}

// 监听分页列表变化，保持选中状态
watch(paginatedNodeList, () => {
  if (selectedNodes.value.length > 0) {
    nextTick(() => {
      syncTableSelection()
    })
  }
}, { flush: 'post' })

// 显示批量标签对话框
const showBatchLabelsDialog = () => {
  batchLabelForm.value = {
    operation: 'add',
    labels: [{ key: '', value: '' }]
  }
  batchLabelDialogVisible.value = true
}

// 添加批量标签
const addBatchLabel = () => {
  batchLabelForm.value.labels.push({ key: '', value: '' })
}

// 删除批量标签
const removeBatchLabel = (index: number) => {
  if (batchLabelForm.value.labels.length > 1) {
    batchLabelForm.value.labels.splice(index, 1)
  } else {
    Message.warning('至少保留一个标签')
  }
}

// 处理批量标签
const handleBatchLabels = async () => {
  const validLabels = batchLabelForm.value.labels.filter(l => l.key.trim() !== '')
  if (validLabels.length === 0) {
    Message.warning('请至少填写一个有效的标签')
    return
  }

  // 检查是否有重复的键
  const keys = validLabels.map(l => l.key)
  const uniqueKeys = new Set(keys)
  if (keys.length !== uniqueKeys.size) {
    Message.warning('存在重复的标签键，请检查')
    return
  }

  batchLabelSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const labelsMap: Record<string, string> = {}
    validLabels.forEach(l => {
      labelsMap[l.key] = l.value
    })

    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/nodes/batch/labels',
      {
        clusterId: selectedClusterId.value,
        nodeNames: selectedNodes.value.map(n => n.name),
        labels: labelsMap,
        operation: batchLabelForm.value.operation
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    batchResults.value = response.data.data?.results || []
    batchResultDialogVisible.value = true
    batchLabelDialogVisible.value = false

    await loadNodes()
    clearSelection()
  } catch (error: any) {
    Message.error(`批量设置标签失败: ${error.response?.data?.message || error.message}`)
  } finally {
    batchLabelSaving.value = false
  }
}

// 显示批量污点对话框
const showBatchTaintsDialog = () => {
  batchTaintForm.value = {
    operation: 'add',
    taints: [{ key: '', value: '', effect: 'NoSchedule' }]
  }
  batchTaintDialogVisible.value = true
}

// 添加批量污点
const addBatchTaint = () => {
  batchTaintForm.value.taints.push({ key: '', value: '', effect: 'NoSchedule' })
}

// 删除批量污点
const removeBatchTaint = (index: number) => {
  if (batchTaintForm.value.taints.length > 1) {
    batchTaintForm.value.taints.splice(index, 1)
  } else {
    Message.warning('至少保留一个污点')
  }
}

// 处理批量污点
const handleBatchTaints = async () => {
  const validTaints = batchTaintForm.value.taints.filter(t => t.key.trim() !== '')
  if (validTaints.length === 0) {
    Message.warning('请至少填写一个有效的污点')
    return
  }

  batchTaintSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/nodes/batch/taints',
      {
        clusterId: selectedClusterId.value,
        nodeNames: selectedNodes.value.map(n => n.name),
        taints: validTaints,
        operation: batchTaintForm.value.operation
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    batchResults.value = response.data.data?.results || []
    batchResultDialogVisible.value = true
    batchTaintDialogVisible.value = false

    await loadNodes()
    clearSelection()
  } catch (error: any) {
    Message.error(`批量设置污点失败: ${error.response?.data?.message || error.message}`)
  } finally {
    batchTaintSaving.value = false
  }
}

// 批量排空
const handleBatchDrain = async () => {
  try {
    await confirmModal(
      `确定要排空选中的 ${selectedNodes.value.length} 个节点吗？这将会驱逐这些节点上的所有 Pod。`,
      '批量排空确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const token = localStorage.getItem('token')
    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/nodes/batch/drain',
      {
        clusterId: selectedClusterId.value,
        nodeNames: selectedNodes.value.map(n => n.name)
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    batchResults.value = response.data.data?.results || []
    batchResultDialogVisible.value = true

    await loadNodes()
    clearSelection()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`批量排空失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 批量设为不可调度
const handleBatchCordon = async () => {
  try {
    await confirmModal(
      `确定要将选中的 ${selectedNodes.value.length} 个节点设为不可调度吗？这些节点将不再接受新的Pod调度。`,
      '批量设为不可调度确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const token = localStorage.getItem('token')
    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/nodes/batch/cordon',
      {
        clusterId: selectedClusterId.value,
        nodeNames: selectedNodes.value.map(n => n.name)
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    batchResults.value = response.data.data?.results || []
    batchResultDialogVisible.value = true

    await loadNodes()
    clearSelection()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`批量设为不可调度失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 批量设为可调度
const handleBatchUncordon = async () => {
  try {
    await confirmModal(
      `确定要将选中的 ${selectedNodes.value.length} 个节点设为可调度吗？这些节点将重新接受新的Pod调度。`,
      '批量设为可调度确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info',
      }
    )

    const token = localStorage.getItem('token')
    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/nodes/batch/uncordon',
      {
        clusterId: selectedClusterId.value,
        nodeNames: selectedNodes.value.map(n => n.name)
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    batchResults.value = response.data.data?.results || []
    batchResultDialogVisible.value = true

    await loadNodes()
    clearSelection()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`批量设为可调度失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 批量删除节点
const handleBatchDelete = async () => {
  try {
    await confirmModal(
      `确定要删除选中的 ${selectedNodes.value.length} 个节点吗？此操作不可恢复！`,
      '批量删除节点确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error',
      }
    )

    const token = localStorage.getItem('token')
    const response = await axios.post(
      '/api/v1/plugins/kubernetes/resources/nodes/batch/delete',
      {
        clusterId: selectedClusterId.value,
        nodeNames: selectedNodes.value.map(n => n.name)
      },
      { headers: { Authorization: `Bearer ${token}` } }
    )

    batchResults.value = response.data.data?.results || []
    batchResultDialogVisible.value = true

    await loadNodes()
    clearSelection()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`批量删除节点失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 获取批量操作结果摘要
const getBatchResultSummary = () => {
  const totalCount = batchResults.value.length
  const successCount = batchResults.value.filter(r => r.success).length
  return { totalCount, successCount }
}

onMounted(() => {
  loadClusters()
  checkCloudTTY()
})
</script>

<style scoped>
.nodes-container {
  padding: 0;
}

/* 统计卡片 */
.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.stat-content {
  display: flex;
  align-items: center;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1;
  margin-bottom: 6px;
}

.stat-label {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 页面头部 */
.page-header-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 14px;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  background: var(--ops-primary, #165dff);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

/* 搜索卡片 */
.search-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.search-form :deep(.arco-form-item) {
  margin-bottom: 0;
}

/* 节点名称单元格 */
.node-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.node-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--ops-primary-bg, #e8f0ff);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--ops-primary, #165dff);
  font-size: 18px;
  flex-shrink: 0;
}

.node-name-content {
  min-width: 0;
}

.node-name {
  font-weight: 500;
  color: var(--ops-primary, #165dff);
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: color 0.2s;
}

.node-name:hover {
  color: var(--ops-primary-light, #306fff);
}

.node-ip {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 2px;
}

/* 状态标签 */
.status-tag {
  border-radius: 10px;
  font-weight: 500;
}

/* 角色徽章 */
.role-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
}

.role-master, .role-control-plane {
  background: var(--ops-primary-bg, #e8f0ff);
  color: var(--ops-primary, #165dff);
}

.role-worker {
  background: #f7f8fa;
  color: var(--ops-text-secondary, #4e5969);
}

/* 版本单元格 */
.version-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--ops-text-secondary, #4e5969);
  font-size: 13px;
}

/* 标签单元格 */
.label-cell {
  cursor: pointer;
  display: flex;
  justify-content: center;
}

.label-badge-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 12px;
  background: var(--ops-primary-bg, #e8f0ff);
  color: var(--ops-primary, #165dff);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.label-badge-wrapper:hover {
  background: var(--ops-primary-lighter, #6694ff);
  color: #fff;
}

/* 运行时间 */
.age-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--ops-text-secondary, #4e5969);
  font-size: 13px;
}

/* 资源单元格 */
.resource-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  flex-shrink: 0;
}

.resource-icon-cpu {
  background: #fff7e8;
  color: var(--ops-warning, #ff7d00);
}

.resource-icon-memory {
  background: #e8ffea;
  color: var(--ops-success, #00b42a);
}

.resource-value {
  font-size: 13px;
  color: var(--ops-text-primary, #1d2129);
  font-weight: 500;
}

/* Pod 数量 */
.pod-count-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.pod-count {
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}

.pod-label {
  font-size: 11px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 污点单元格 */
.taint-cell {
  cursor: pointer;
  display: flex;
  justify-content: center;
}

.taint-badge-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 12px;
  background: #fff7e8;
  color: var(--ops-warning, #ff7d00);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.taint-badge-wrapper:hover {
  background: var(--ops-warning, #ff7d00);
  color: #fff;
}

/* 操作按钮 */
.action-btn {
  color: var(--ops-text-secondary, #4e5969);
  transition: color 0.2s;
}

.action-btn:hover {
  color: var(--ops-primary, #165dff);
}

/* 对话框通用 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 标签/污点编辑 */
.label-edit-container, .taint-edit-container {
  padding: 0;
}

.label-edit-header, .taint-edit-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f7f8fa;
  border-radius: var(--ops-border-radius-sm, 4px);
  margin-bottom: 16px;
}

.label-edit-info, .taint-edit-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--ops-text-secondary, #4e5969);
  font-size: 14px;
}

.label-edit-count, .taint-edit-count {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
}

.label-edit-list, .taint-edit-list {
  max-height: 400px;
  overflow-y: auto;
}

.label-edit-row, .taint-edit-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}

.label-row-number, .taint-row-number {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--ops-primary-bg, #e8f0ff);
  color: var(--ops-primary, #165dff);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}

.label-row-content, .taint-row-content {
  flex: 1;
}

.label-input-group, .taint-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label-input-wrapper, .taint-input-wrapper, .taint-effect-wrapper {
  flex: 1;
}

.label-input-label, .taint-input-label {
  display: block;
  font-size: 11px;
  color: var(--ops-text-tertiary, #86909c);
  margin-bottom: 4px;
}

.label-separator, .taint-separator {
  color: var(--ops-text-tertiary, #86909c);
  font-weight: 600;
  font-size: 16px;
}

.remove-btn {
  flex-shrink: 0;
}

.empty-labels, .empty-taints {
  text-align: center;
  padding: 40px 0;
  color: var(--ops-text-tertiary, #86909c);
}

.empty-labels p, .empty-taints p {
  margin: 8px 0 4px;
  font-size: 14px;
}

.empty-labels span, .empty-taints span {
  font-size: 12px;
}

.add-label-btn, .add-taint-btn {
  width: 100%;
  margin-top: 12px;
}

/* 标签查看表格 */
.label-key-wrapper, .taint-key-wrapper {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: var(--ops-text-primary, #1d2129);
  transition: color 0.2s;
}

.label-key-wrapper:hover, .taint-key-wrapper:hover {
  color: var(--ops-primary, #165dff);
}

.label-value, .taint-value {
  color: var(--ops-text-secondary, #4e5969);
}

.effect-tag {
  font-weight: 500;
}

/* Effect 选项 */
.effect-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.effect-desc {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 批量操作表单 */
.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

.batch-label-row, .batch-taint-row {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

/* 批量结果 */
.result-summary {
  text-align: center;
  margin-bottom: 16px;
}

/* 代码块 */
.code-block-wrapper {
  display: flex;
  width: 100%;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: var(--ops-border-radius-sm, 4px);
  overflow: hidden;
  background-color: #282c34;
}

.code-line-numbers {
  display: flex;
  flex-direction: column;
  padding: 12px 8px;
  background-color: #21252b;
  border-right: 1px solid #3e4451;
  user-select: none;
  min-width: 40px;
  text-align: right;
}

.code-line-number {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #5c6370;
  min-height: 20.8px;
}

.code-textarea {
  flex: 1;
  min-height: 200px;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #abb2bf;
  background-color: #282c34;
  border: none;
  outline: none;
  resize: vertical;
}

.code-textarea::placeholder {
  color: #5c6370;
}

/* YAML 编辑器 */
.yaml-editor-wrapper {
  display: flex;
  width: 100%;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: var(--ops-border-radius-sm, 4px);
  overflow: hidden;
  background-color: #282c34;
  max-height: 60vh;
}

.yaml-line-numbers {
  display: flex;
  flex-direction: column;
  padding: 12px 8px;
  background-color: #21252b;
  border-right: 1px solid #3e4451;
  user-select: none;
  min-width: 40px;
  text-align: right;
  overflow: hidden;
}

.line-number {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #5c6370;
  min-height: 20.8px;
}

.yaml-textarea {
  flex: 1;
  min-height: 400px;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #abb2bf;
  background-color: #282c34;
  border: none;
  outline: none;
  resize: none;
}

.yaml-textarea::placeholder {
  color: #5c6370;
}

/* Shell 终端 */
.terminal-container {
  height: 500px;
  background: #1e1e1e;
  border-radius: var(--ops-border-radius-sm, 4px);
  overflow: hidden;
}

/* CloudTTY */
.cloudtty-iframe {
  width: 100%;
  height: 500px;
  border: none;
  border-radius: var(--ops-border-radius-sm, 4px);
}

.deploy-steps {
  margin: 16px 0;
}

/* 部署状态 */
.deploy-status {
  padding: 20px;
}

.deploy-methods h4 {
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 12px;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0 0;
  border-top: 1px solid var(--ops-border-color, #e5e6eb);
}

/* 批量操作栏 */
.batch-actions-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  margin-bottom: 16px;
  background: var(--ops-primary-bg, #e8f0ff);
  border: 1px solid var(--ops-primary-lighter, #6694ff);
  border-radius: var(--ops-border-radius-md, 8px);
}

.batch-actions-left {
  display: flex;
  align-items: center;
}

.selected-count {
  font-size: 14px;
  color: var(--ops-text-primary, #1d2129);
  font-weight: 500;
}

/* 表格卡片 */
.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

/* 下拉菜单危险项 */
.danger-item {
  color: var(--ops-danger, #f53f3f) !important;
}

/* 对话框样式 */
:deep(.cloudtty-terminal-dialog .arco-modal-body) {
  padding: 0;
}

:deep(.shell-dialog .arco-modal-body) {
  padding: 0;
}

@media (max-width: 1200px) {
  .stat-value { font-size: 24px; }
  .stat-icon { width: 48px; height: 48px; }
}
</style>
