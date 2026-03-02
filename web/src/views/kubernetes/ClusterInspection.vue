<template>
  <div class="inspection-container">
    <!-- 页面标题和操作按钮 -->
    <a-card class="page-header-card">
      <div class="page-header">
        <div class="page-title-group">
          <div class="page-title-icon">
            <icon-file-search />
          </div>
          <div>
            <h2 class="page-title">集群巡检</h2>
            <p class="page-subtitle">对 Kubernetes 集群进行健康检查，生成巡检报告</p>
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
          <a-button v-permission="'k8s-inspection:start'" type="primary" @click="handleStartInspection" :loading="inspecting" :disabled="!selectedClusterId">
            <template #icon><icon-refresh /></template>
            {{ inspecting ? '巡检中...' : '开始巡检' }}
          </a-button>
          <a-button @click="historyDialogVisible = true">
            <template #icon><icon-clock-circle /></template>
            历史记录
          </a-button>
        </div>
      </div>
    </a-card>

    <!-- 巡检进度 -->
    <a-card class="progress-card" v-if="inspecting">
      <div class="progress-header">
        <div class="progress-animation">
          <div class="progress-spinner"></div>
        </div>
        <div class="progress-info">
          <span class="progress-title">正在巡检集群...</span>
          <span class="progress-step">{{ progressInfo.currentStep }}</span>
        </div>
      </div>
      <a-progress
        :percentage="progressInfo.progress"
        :stroke-width="16"
        :color="progressColors"
        striped
        striped-flow
      >
        <template #default="{ percentage }">
          <span class="progress-text">{{ percentage }}%</span>
        </template>
      </a-progress>
    </a-card>

    <!-- 空状态 - 未执行巡检 -->
    <a-card v-if="!inspecting && !inspectionResult" class="empty-card">
      <div class="empty-content">
        <div class="empty-icon">
          <icon-file-search />
        </div>
        <h3 class="empty-title">开始集群健康巡检</h3>
        <p class="empty-desc">选择一个 Kubernetes 集群，执行全面的健康检查，包括节点状态、工作负载、网络、存储、安全配置等多个维度</p>
        <div class="empty-features">
          <div class="feature-item">
            <icon-desktop />
            <span>节点健康</span>
          </div>
          <div class="feature-item">
            <icon-storage />
            <span>工作负载</span>
          </div>
          <div class="feature-item">
            <icon-link />
            <span>网络状态</span>
          </div>
          <div class="feature-item">
            <icon-storage />
            <span>存储管理</span>
          </div>
          <div class="feature-item">
            <icon-lock />
            <span>安全配置</span>
          </div>
          <div class="feature-item">
            <icon-line-chart />
            <span>容量规划</span>
          </div>
        </div>
        <a-button v-permission="'k8s-inspection:start'" type="primary" size="large" @click="handleStartInspection" :disabled="!selectedClusterId">
          <template #icon><icon-refresh /></template>
          立即开始巡检
        </a-button>
      </div>
    </a-card>

    <!-- 巡检结果详情 -->
    <a-card v-if="inspectionResult && !inspecting" class="result-card">
      <!-- 统计卡片 -->
      <div class="stats-grid">
        <div class="stat-card score-card">
          <div class="score-circle" :class="getScoreClass(inspectionResult.score)">
            <svg viewBox="0 0 100 100">
              <circle class="score-bg" cx="50" cy="50" r="45" />
              <circle
                class="score-progress"
                cx="50" cy="50" r="45"
                :stroke-dasharray="scoreCircle"
                :stroke-dashoffset="scoreOffset"
              />
            </svg>
            <div class="score-value">{{ inspectionResult.score }}</div>
            <div class="score-label">健康评分</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon-blue">
            <icon-file-search />
          </div>
          <div class="stat-content">
            <div class="stat-label">检查项目</div>
            <div class="stat-value">{{ inspectionResult.summary.totalChecks }}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon-green">
            <icon-check-circle-fill />
          </div>
          <div class="stat-content">
            <div class="stat-label">通过项</div>
            <div class="stat-value success-text">{{ inspectionResult.summary.passedChecks }}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" :class="getIssueIconClass()">
            <icon-exclamation-circle-fill />
          </div>
          <div class="stat-content">
            <div class="stat-label">发现问题</div>
            <div class="stat-value" :class="getIssueClass()">
              {{ inspectionResult.summary.warningChecks + inspectionResult.summary.failedChecks }}
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon-blue">
            <icon-clock-circle />
          </div>
          <div class="stat-content">
            <div class="stat-label">巡检耗时</div>
            <div class="stat-value">{{ currentInspection?.duration || 0 }}s</div>
          </div>
        </div>
      </div>
      <a-tabs v-model:active-key="activeTab" class="inspection-tabs">
        <a-tab-pane title="概览" key="overview">
          <div class="overview-content">
            <!-- 结果统计 -->
            <div class="result-summary">
              <div class="summary-item success">
                <icon-check-circle-fill />
                <span class="summary-label">正常</span>
                <span class="summary-value">{{ inspectionResult.summary.passedChecks }}</span>
              </div>
              <div class="summary-item warning">
                <icon-exclamation-circle-fill />
                <span class="summary-label">警告</span>
                <span class="summary-value">{{ inspectionResult.summary.warningChecks }}</span>
              </div>
              <div class="summary-item error">
                <icon-close-circle-fill />
                <span class="summary-label">异常</span>
                <span class="summary-value">{{ inspectionResult.summary.failedChecks }}</span>
              </div>
            </div>

            <!-- 检查项列表 -->
            <div class="check-items-table">
              <a-table :data="allCheckItems" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns13">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="集群信息" key="cluster">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Kubernetes 版本</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.version }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">平台</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.platform }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Go 版本</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.goVersion }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">连接延迟</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.connectionDelay }}ms</span>
              </div>
            </div>
            <div class="check-items-table">
              <a-table :data="inspectionResult.clusterInfo.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns12">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="节点健康" key="nodes">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">节点总数</span>
                <span class="info-value">{{ inspectionResult.nodeHealth.totalNodes }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Ready 节点</span>
                <span class="info-value success-text">{{ inspectionResult.nodeHealth.readyNodes }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">NotReady 节点</span>
                <span class="info-value" :class="{ 'error-text': inspectionResult.nodeHealth.notReadyNodes > 0 }">
                  {{ inspectionResult.nodeHealth.notReadyNodes }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">有资源压力</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.nodeHealth.pressureNodes > 0 }">
                  {{ inspectionResult.nodeHealth.pressureNodes }}
                </span>
              </div>
            </div>
            <!-- 节点资源利用率表格 -->
            <div class="check-items-table" v-if="inspectionResult.nodeHealth.nodeUtilization.length">
              <h4 class="table-title">节点资源利用率</h4>
              <a-table :data="inspectionResult.nodeHealth.nodeUtilization" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns11">
          <template #status="{ record }">
                    <a-tag :type="record.status === 'Ready' ? 'success' : 'danger'" size="small">
                      {{ record.status }}
                    </a-tag>
                  </template>
          <template #podCount="{ record }">
                    {{ record.podCount }}/{{ record.podCapacity }}
                  </template>
        </a-table>
            </div>
            <!-- 检查项 -->
            <div class="check-items-table">
              <h4 class="table-title">检查项</h4>
              <a-table :data="inspectionResult.nodeHealth.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns10">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="工作负载" key="workloads">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Deployment</span>
                <span class="info-value">{{ inspectionResult.workloads.healthyDeployments }}/{{ inspectionResult.workloads.totalDeployments }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">DaemonSet</span>
                <span class="info-value">{{ inspectionResult.workloads.healthyDaemonSets }}/{{ inspectionResult.workloads.totalDaemonSets }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">StatefulSet</span>
                <span class="info-value">{{ inspectionResult.workloads.healthyStatefulSets }}/{{ inspectionResult.workloads.totalStatefulSets }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Pod总数</span>
                <span class="info-value">{{ inspectionResult.workloads.runningPods }}/{{ inspectionResult.workloads.totalPods }}</span>
              </div>
            </div>
            <!-- 异常工作负载 -->
            <div class="check-items-table" v-if="inspectionResult.workloads.unhealthyWorkloads.length">
              <h4 class="table-title">异常工作负载</h4>
              <a-table :data="inspectionResult.workloads.unhealthyWorkloads" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns9">
          
        </a-table>
            </div>
            <!-- 检查项 -->
            <div class="check-items-table">
              <h4 class="table-title">检查项</h4>
              <a-table :data="inspectionResult.workloads.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns8">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="网络" key="network">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Service总数</span>
                <span class="info-value">{{ inspectionResult.network.totalServices }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Ingress总数</span>
                <span class="info-value">{{ inspectionResult.network.totalIngresses }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">NetworkPolicy</span>
                <span class="info-value">{{ inspectionResult.network.networkPolicies }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">无Endpoint的Service</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.network.noEndpointServices > 0 }">
                  {{ inspectionResult.network.noEndpointServices }}
                </span>
              </div>
            </div>
            <div class="check-items-table">
              <a-table :data="inspectionResult.network.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns7">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="存储" key="storage">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">PV总数</span>
                <span class="info-value">{{ inspectionResult.storage.totalPVs }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">PVC总数</span>
                <span class="info-value">{{ inspectionResult.storage.totalPVCs }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">StorageClass</span>
                <span class="info-value">{{ inspectionResult.storage.storageClasses }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Pending PVC</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.storage.pendingPVCs > 0 }">
                  {{ inspectionResult.storage.pendingPVCs }}
                </span>
              </div>
            </div>
            <div class="check-items-table">
              <a-table :data="inspectionResult.storage.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns6">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="安全" key="security">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">ServiceAccount</span>
                <span class="info-value">{{ inspectionResult.security.serviceAccounts }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">特权Pod</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.security.privilegedPods > 0 }">
                  {{ inspectionResult.security.privilegedPods }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">hostNetwork Pod</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.security.hostNetworkPods > 0 }">
                  {{ inspectionResult.security.hostNetworkPods }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">cluster-admin绑定</span>
                <span class="info-value">{{ inspectionResult.security.clusterAdminBindings }}</span>
              </div>
            </div>
            <div class="check-items-table">
              <a-table :data="inspectionResult.security.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns5">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="容量" key="capacity">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">CPU分配率</span>
                <span class="info-value" :class="getCapacityClass(inspectionResult.capacity.cpuAllocatePercent)">
                  {{ inspectionResult.capacity.cpuAllocatePercent.toFixed(1) }}%
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">内存分配率</span>
                <span class="info-value" :class="getCapacityClass(inspectionResult.capacity.memoryAllocatePercent)">
                  {{ inspectionResult.capacity.memoryAllocatePercent.toFixed(1) }}%
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">Pod密度</span>
                <span class="info-value" :class="getCapacityClass(inspectionResult.capacity.podDensityPercent)">
                  {{ inspectionResult.capacity.podDensityPercent.toFixed(1) }}%
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">Pod数量</span>
                <span class="info-value">{{ inspectionResult.capacity.currentPodCount }}/{{ inspectionResult.capacity.totalPodCapacity }}</span>
              </div>
            </div>
            <div class="check-items-table">
              <a-table :data="inspectionResult.capacity.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns4">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>

        <a-tab-pane title="事件" key="events">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Warning事件</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.events.warningEvents > 10 }">
                  {{ inspectionResult.events.warningEvents }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">Error事件</span>
                <span class="info-value" :class="{ 'error-text': inspectionResult.events.errorEvents > 0 }">
                  {{ inspectionResult.events.errorEvents }}
                </span>
              </div>
            </div>
            <!-- 最近事件 -->
            <div class="check-items-table" v-if="inspectionResult.events.recentEvents.length">
              <h4 class="table-title">最近告警事件</h4>
              <a-table :data="inspectionResult.events.recentEvents" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns3">
          
        </a-table>
            </div>
            <!-- 检查项 -->
            <div class="check-items-table">
              <h4 class="table-title">检查项</h4>
              <a-table :data="inspectionResult.events.items" style="width: 100%" :header-cell-style="tableHeaderStyle" :columns="tableColumns2">
          <template #status="{ record }">
                    <a-tag :type="getStatusTagType(record.status)" size="small">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                  </template>
        </a-table>
            </div>
          </div>
        </a-tab-pane>
      </a-tabs>

      <!-- 导出按钮 -->
      <div class="export-actions">
        <a-button type="primary" @click="handleExportExcel" :loading="exporting">
          <template #icon><icon-download /></template>
          导出Excel报告
        </a-button>
        <a-button v-permission="'k8s-inspection:start'" @click="handleStartInspection" :disabled="inspecting">
          <template #icon><icon-refresh /></template>
          重新巡检
        </a-button>
      </div>
    </a-card>

    <!-- 历史记录对话框 -->
    <a-modal v-model:visible="historyDialogVisible" title="巡检历史记录" width="900px">
      <template v-if="historyList.length === 0 && !historyLoading">
        <div class="history-empty">
          <icon-clock-circle />
          <p class="history-empty-text">暂无巡检记录</p>
          <p class="history-empty-hint">执行集群巡检后，历史记录将会显示在这里</p>
        </div>
      </template>
      <template v-else>
        <a-table :data="historyList" style="width: 100%" :header-cell-style="tableHeaderStyle" v-loading="historyLoading" :columns="tableColumns">
          <template #col_8565="{ record }">
              <span :class="getScoreClass(record.score)">{{ record.score }}/100</span>
            </template>
          <template #status="{ record }">
              <a-tag :type="record.status === 'completed' ? 'success' : record.status === 'running' ? 'warning' : 'danger'" size="small">
                {{ record.status === 'completed' ? '完成' : record.status === 'running' ? '进行中' : '失败' }}
              </a-tag>
            </template>
          <template #actions="{ record }">
              <a-button type="text" size="small" @click="handleViewHistory(record)" :disabled="record.status !== 'completed'">
                查看
              </a-button>
              <a-button status="danger" type="text" size="small" @click="handleDeleteHistory(record)">
                删除
              </a-button>
            </template>
        </a-table>
        <div class="pagination-wrapper" v-if="historyTotal > 0">
          <a-pagination
            v-model:current="historyPage"
            v-model:page-size="historyPageSize"
            :total="historyTotal"
            :page-size-options="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            @size-change="loadHistory"
            @current-change="loadHistory"
          />
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns13 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150, ellipsis: true, tooltip: true },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true },
  { title: '建议', dataIndex: 'suggestion', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns12 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns11 = [
  { title: '节点名称', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: 'CPU容量', dataIndex: 'cpuCapacity', width: 120 },
  { title: '内存容量', dataIndex: 'memoryCapacity', width: 120 },
  { title: 'Pod数量', slotName: 'podCount', width: 120 }
]

const tableColumns10 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true },
  { title: '建议', dataIndex: 'suggestion', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns9 = [
  { title: '类型', dataIndex: 'kind', width: 120 },
  { title: '命名空间', dataIndex: 'namespace', width: 150 },
  { title: '名称', dataIndex: 'name', width: 200 },
  { title: '就绪状态', dataIndex: 'ready', width: 120 },
  { title: '原因', dataIndex: 'reason', width: 200 }
]

const tableColumns8 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true },
  { title: '建议', dataIndex: 'suggestion', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns7 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns6 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true },
  { title: '建议', dataIndex: 'suggestion', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns5 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true },
  { title: '建议', dataIndex: 'suggestion', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns4 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true },
  { title: '建议', dataIndex: 'suggestion', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns3 = [
  { title: '类型', dataIndex: 'type', width: 100 },
  { title: '原因', dataIndex: 'reason', width: 150 },
  { title: '对象', dataIndex: 'object', width: 200, ellipsis: true, tooltip: true },
  { title: '命名空间', dataIndex: 'namespace', width: 120 },
  { title: '次数', dataIndex: 'count', width: 80 },
  { title: '最后发生', dataIndex: 'lastSeen', width: 180 },
  { title: '消息', dataIndex: 'message', width: 300, ellipsis: true, tooltip: true }
]

const tableColumns2 = [
  { title: '类别', dataIndex: 'category', width: 120 },
  { title: '检查项', dataIndex: 'name', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查值', dataIndex: 'value', width: 150 },
  { title: '详情', dataIndex: 'detail', width: 200, ellipsis: true, tooltip: true },
  { title: '建议', dataIndex: 'suggestion', width: 200, ellipsis: true, tooltip: true }
]

const tableColumns = [
  { title: '集群', dataIndex: 'clusterName', width: 150 },
  { title: '评分', slotName: 'col_8565', width: 100 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '检查项', dataIndex: 'checkCount', width: 80 },
  { title: '通过', dataIndex: 'passCount', width: 80 },
  { title: '警告', dataIndex: 'warningCount', width: 80 },
  { title: '失败', dataIndex: 'failCount', width: 80 },
  { title: '耗时(s)', dataIndex: 'duration', width: 80 },
  { title: '时间', dataIndex: 'createdAt', width: 180 },
  { title: '操作', slotName: 'actions', width: 150, fixed: 'right' }
]

import { ref, onMounted, computed, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { getClusterList, type Cluster } from '@/api/kubernetes'
import {
  startInspection,
  getInspectionProgress,
  getInspectionResult,
  getInspectionHistory,
  deleteInspection,
  exportInspection,
  type InspectionResult,
  type InspectionProgress,
  type InspectionHistoryItem,
  type ClusterInspection,
  type CheckItem
} from '@/api/inspection'

// 状态
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const inspecting = ref(false)
const currentInspectionId = ref<number>()
const currentInspection = ref<ClusterInspection | null>(null)
const inspectionResult = ref<InspectionResult | null>(null)
const progressInfo = ref<InspectionProgress>({
  inspectionId: 0,
  status: '',
  progress: 0,
  currentStep: '',
  completedClusters: 0,
  totalClusters: 0
})
const activeTab = ref('overview')
const exporting = ref(false)

// 历史记录
const historyDialogVisible = ref(false)
const historyList = ref<InspectionHistoryItem[]>([])
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(10)
const historyTotal = ref(0)

// 进度条颜色
const progressColors = [
  { color: '#909399', percentage: 30 },
  { color: '#e6a23c', percentage: 60 },
  { color: '#67c23a', percentage: 100 }
]

// 表格头样式
const tableHeaderStyle = {
  background: '#fafbfc',
  color: '#4e5969',
  fontSize: '14px',
  fontWeight: '600'
}

// 计算所有检查项
const allCheckItems = computed(() => {
  if (!inspectionResult.value) return []
  const items: CheckItem[] = []
  items.push(...(inspectionResult.value.clusterInfo?.items || []))
  items.push(...(inspectionResult.value.nodeHealth?.items || []))
  items.push(...(inspectionResult.value.components?.items || []))
  items.push(...(inspectionResult.value.workloads?.items || []))
  items.push(...(inspectionResult.value.network?.items || []))
  items.push(...(inspectionResult.value.storage?.items || []))
  items.push(...(inspectionResult.value.security?.items || []))
  items.push(...(inspectionResult.value.config?.items || []))
  items.push(...(inspectionResult.value.capacity?.items || []))
  items.push(...(inspectionResult.value.events?.items || []))
  return items
})

// 圆环进度条计算
const scoreCircle = computed(() => {
  const circumference = 2 * Math.PI * 45
  return `${circumference} ${circumference}`
})

const scoreOffset = computed(() => {
  if (!inspectionResult.value) return 283
  const circumference = 2 * Math.PI * 45
  const progress = inspectionResult.value.score / 100
  return circumference * (1 - progress)
})

// 监听历史对话框打开
watch(historyDialogVisible, (visible) => {
  if (visible) {
    loadHistory()
  }
})

// 获取评分样式类
const getScoreClass = (score: number) => {
  if (score >= 80) return 'score-excellent'
  if (score >= 60) return 'score-good'
  return 'score-poor'
}

// 获取问题图标样式
const getIssueIconClass = () => {
  if (!inspectionResult.value) return 'stat-icon-blue'
  const issues = inspectionResult.value.summary.warningChecks + inspectionResult.value.summary.failedChecks
  if (issues === 0) return 'stat-icon-green'
  if (inspectionResult.value.summary.failedChecks > 0) return 'stat-icon-red'
  return 'stat-icon-orange'
}

// 获取问题数量样式
const getIssueClass = () => {
  if (!inspectionResult.value) return ''
  if (inspectionResult.value.summary.failedChecks > 0) return 'error-text'
  if (inspectionResult.value.summary.warningChecks > 0) return 'warning-text'
  return 'success-text'
}

// 获取状态标签类型
const getStatusTagType = (status: string) => {
  switch (status) {
    case 'success': return 'success'
    case 'warning': return 'warning'
    case 'error': return 'danger'
    default: return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'success': return '正常'
    case 'warning': return '警告'
    case 'error': return '异常'
    default: return '未知'
  }
}

// 获取容量样式
const getCapacityClass = (percent: number) => {
  if (percent > 85) return 'error-text'
  if (percent > 70) return 'warning-text'
  return 'success-text'
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0 && !selectedClusterId.value) {
      selectedClusterId.value = clusterList.value[0].id
    }
  } catch (error: any) {
    Message.error('获取集群列表失败: ' + (error.message || '未知错误'))
  }
}

// 处理集群变更
const handleClusterChange = () => {
  inspectionResult.value = null
  currentInspection.value = null
}

// 开始巡检
const handleStartInspection = async () => {
  if (!selectedClusterId.value) {
    Message.warning('请先选择集群')
    return
  }

  inspecting.value = true
  inspectionResult.value = null
  progressInfo.value = {
    inspectionId: 0,
    status: 'running',
    progress: 0,
    currentStep: '初始化巡检...',
    completedClusters: 0,
    totalClusters: 1
  }

  try {
    const data = await startInspection({
      clusterIds: [selectedClusterId.value]
    })
    currentInspectionId.value = data.inspectionId
    pollProgress()
  } catch (error: any) {
    inspecting.value = false
    Message.error('启动巡检失败: ' + (error.message || '未知错误'))
  }
}

// 轮询进度
const pollProgress = async () => {
  if (!currentInspectionId.value) return

  try {
    const data = await getInspectionProgress(currentInspectionId.value)
    progressInfo.value = data

    if (data.status === 'completed') {
      inspecting.value = false
      await loadInspectionResult()
      Message.success('巡检完成')
    } else if (data.status === 'failed') {
      inspecting.value = false
      Message.error('巡检失败')
    } else {
      setTimeout(pollProgress, 1000)
    }
  } catch (error) {
    setTimeout(pollProgress, 2000)
  }
}

// 加载巡检结果
const loadInspectionResult = async () => {
  if (!currentInspectionId.value) return

  try {
    const data = await getInspectionResult(currentInspectionId.value)
    currentInspection.value = data.inspection
    inspectionResult.value = data.result
  } catch (error: any) {
    Message.error('获取巡检结果失败: ' + (error.message || '未知错误'))
  }
}

// 导出Excel
const handleExportExcel = async () => {
  if (!currentInspectionId.value) return

  exporting.value = true
  try {
    const res = await exportInspection(currentInspectionId.value, 'excel')
    // 创建下载链接
    const blob = new Blob([res as any], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `cluster-inspection-${currentInspection.value?.clusterName || 'report'}-${new Date().toISOString().slice(0, 10)}.xlsx`
    link.click()
    window.URL.revokeObjectURL(url)
    Message.success('导出成功')
  } catch (error: any) {
    Message.error('导出失败: ' + (error.message || '未知错误'))
  } finally {
    exporting.value = false
  }
}

// 加载历史记录
const loadHistory = async () => {
  historyLoading.value = true
  try {
    const data = await getInspectionHistory({
      clusterId: selectedClusterId.value,
      page: historyPage.value,
      pageSize: historyPageSize.value
    })
    historyList.value = data.list || []
    historyTotal.value = data.total
  } catch (error: any) {
    Message.error('获取历史记录失败: ' + (error.message || '未知错误'))
  } finally {
    historyLoading.value = false
  }
}

// 查看历史记录
const handleViewHistory = async (item: InspectionHistoryItem) => {
  currentInspectionId.value = item.id
  await loadInspectionResult()
  historyDialogVisible.value = false
}

// 删除历史记录
const handleDeleteHistory = async (item: InspectionHistoryItem) => {
  try {
    await confirmModal('确定要删除这条巡检记录吗?', '提示', {
      type: 'warning'
    })
    await deleteInspection(item.id)
    Message.success('删除成功')
    loadHistory()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.inspection-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部卡片 */
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
  align-items: flex-start;
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
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 进度卡片 */
.progress-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.progress-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.progress-animation {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.progress-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--ops-border-color, #e5e6eb);
  border-top-color: var(--ops-primary, #165dff);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.progress-info {
  flex: 1;
}

.progress-title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 4px;
}

.progress-step {
  font-size: 14px;
  color: var(--ops-text-tertiary, #86909c);
}

.progress-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-primary, #165dff);
}

/* 空状态卡片 */
.empty-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.empty-content {
  max-width: 600px;
  margin: 0 auto;
  text-align: center;
  padding: 40px 0;
}

.empty-icon {
  width: 100px;
  height: 100px;
  margin: 0 auto 24px;
  background: var(--ops-primary, #165dff);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 48px;
}

.empty-title {
  margin: 0 0 12px;
  font-size: 22px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}

.empty-desc {
  margin: 0 0 32px;
  font-size: 14px;
  color: var(--ops-text-tertiary, #86909c);
  line-height: 1.6;
}

.empty-features {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 16px;
  margin-bottom: 32px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--ops-content-bg, #f7f8fa);
  border-radius: 20px;
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
}

.feature-item :deep(.arco-icon) {
  font-size: 16px;
  color: var(--ops-primary, #165dff);
}

/* 结果卡片 */
.result-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: 140px repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: var(--ops-content-bg, #f7f8fa);
  border-radius: var(--ops-border-radius-md, 8px);
  transition: all 0.3s ease;
}

.stat-card:not(.score-card):hover {
  background: #f0f2f5;
}

/* 评分圆环卡片 */
.score-card {
  grid-row: span 1;
  justify-content: center;
  padding: 20px;
  background: linear-gradient(135deg, #1d2129 0%, #2a2f38 100%);
}

.score-circle {
  position: relative;
  width: 100px;
  height: 100px;
}

.score-circle svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.score-circle .score-bg {
  fill: none;
  stroke: #333;
  stroke-width: 8;
}

.score-circle .score-progress {
  fill: none;
  stroke: var(--ops-primary, #165dff);
  stroke-width: 8;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.8s ease;
}

.score-circle.score-excellent .score-progress {
  stroke: var(--ops-success, #00b42a);
}

.score-circle.score-good .score-progress {
  stroke: var(--ops-warning, #ff7d00);
}

.score-circle.score-poor .score-progress {
  stroke: var(--ops-danger, #f53f3f);
}

.score-value {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -60%);
  font-size: 28px;
  font-weight: 700;
  color: #fff;
}

.score-label {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, 50%);
  font-size: 12px;
  color: #86909c;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.stat-icon :deep(.arco-icon) {
  font-size: 24px;
  color: inherit;
}

.stat-icon-blue {
  background: rgba(22, 93, 255, 0.1);
  color: var(--ops-primary, #165dff);
}

.stat-icon-green {
  background: rgba(0, 180, 42, 0.1);
  color: var(--ops-success, #00b42a);
}

.stat-icon-orange {
  background: rgba(255, 125, 0, 0.1);
  color: var(--ops-warning, #ff7d00);
}

.stat-icon-red {
  background: rgba(245, 63, 63, 0.1);
  color: var(--ops-danger, #f53f3f);
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1;
}

.stat-value.score-excellent {
  color: var(--ops-success, #00b42a);
}

.stat-value.score-good {
  color: var(--ops-warning, #ff7d00);
}

.stat-value.score-poor {
  color: var(--ops-danger, #f53f3f);
}

.inspection-tabs {
  :deep(.arco-tabs__header) {
    margin-bottom: 20px;
  }

  :deep(.arco-tabs__item) {
    font-size: 14px;
    font-weight: 500;
  }
}

/* 概览内容 */
.overview-content {
  padding: 0;
}

.result-summary {
  display: flex;
  gap: 40px;
  margin-bottom: 24px;
  padding: 20px;
  background: var(--ops-content-bg, #f7f8fa);
  border-radius: var(--ops-border-radius-md, 8px);
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.summary-item :deep(.arco-icon) {
  font-size: 24px;
}

.summary-item.success :deep(.arco-icon) {
  color: var(--ops-success, #00b42a);
}

.summary-item.warning :deep(.arco-icon) {
  color: var(--ops-warning, #ff7d00);
}

.summary-item.error :deep(.arco-icon) {
  color: var(--ops-danger, #f53f3f);
}

.summary-label {
  font-size: 14px;
  color: var(--ops-text-tertiary, #86909c);
}

.summary-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--ops-text-primary, #1d2129);
}

/* Tab内容 */
.tab-content {
  padding: 0;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
  padding: 16px;
  background: var(--ops-content-bg, #f7f8fa);
  border-radius: var(--ops-border-radius-md, 8px);
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
}

.info-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}

.success-text {
  color: var(--ops-success, #00b42a);
}

.warning-text {
  color: var(--ops-warning, #ff7d00);
}

.error-text {
  color: var(--ops-danger, #f53f3f);
}

/* 表格 */
.check-items-table {
  margin-top: 16px;
}

.table-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}

/* 导出按钮 */
.export-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--ops-border-color, #e5e6eb);
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

/* 响应式 */
@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  .score-card {
    grid-column: span 1;
  }

  .info-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    flex-wrap: wrap;
  }

  .cluster-select {
    width: 100%;
  }

  .empty-features {
    gap: 8px;
  }

  .feature-item {
    padding: 6px 12px;
    font-size: 12px;
  }
}

/* 历史记录空状态 */
.history-empty {
  padding: 60px 20px;
  text-align: center;
}

.history-empty-icon {
  font-size: 64px;
  color: var(--ops-border-color, #e5e6eb);
  margin-bottom: 16px;
}

.history-empty-text {
  margin: 0 0 8px;
  font-size: 16px;
  color: var(--ops-text-secondary, #4e5969);
}

.history-empty-hint {
  margin: 0;
  font-size: 14px;
  color: var(--ops-text-tertiary, #86909c);
}
</style>
