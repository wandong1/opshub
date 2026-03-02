<template>
  <div class="hosts-page-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-desktop /></div>
        <div>
          <h2 class="page-title">主机管理</h2>
          <p class="page-subtitle">管理所有服务器和主机资源，支持多种方式导入</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button type="primary" @click="handleOpenTerminal">
          <template #icon><icon-desktop /></template>
          终端</a-button>
        <a-button @click="showInstallPackageDialog = true">
          <template #icon><icon-download /></template>
          生成Agent安装包</a-button>
        <a-dropdown
          v-if="userHasEditPermission && permissionStore.hasPermission('hosts:import')"
          @select="handleImportCommand"
          class="import-dropdown"
        >
          <a-button type="primary">
            <template #icon><icon-plus /></template>
            新增主机
            <icon-down style="margin-left: 6px;" />
          </a-button>
          <template #content>
            <a-doption :value="'direct'">
              <icon-file /> 直接导入
            </a-doption>
            <a-doption :value="'excel'">
              <icon-upload /> Excel导入
            </a-doption>
            <a-doption :value="'cloud'">
              <icon-cloud /> 云主机导入
            </a-doption>
          </template>
        </a-dropdown>
      </div>
    </div>

    <!-- 主内容区域：左侧分组树 + 右侧主机列表 -->
    <div class="main-content">
      <!-- 左侧分组树 - 终端视图时隐藏 -->
      <div class="left-panel" v-show="activeView === 'hosts'">
        <div class="panel-header">
          <div class="panel-title">
            <icon-apps class="panel-icon" />
            <span>资产分组</span>
          </div>
          <div class="panel-actions">
            <a-tooltip content="新增分组" position="top">
              <a-button shape="circle" size="small" @click="handleAddGroup">
                <icon-plus />
              </a-button>
            </a-tooltip>
            <a-tooltip :content="isExpandAll ? '折叠全部' : '展开全部'" position="top">
              <a-button shape="circle" size="small" @click="toggleExpandAll">
                <icon-swap />
              </a-button>
            </a-tooltip>
          </div>
        </div>
        <div class="panel-body">
          <a-input
            v-model="groupSearchKeyword"
            placeholder="搜索分组..."
            allow-clear
            size="small"
            class="group-search"
            @input="filterGroupTree"
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
          <div class="tree-container">
            <a-spin :loading="groupLoading" style="width: 100%;">
              <a-tree
                ref="groupTreeRef"
                :data="filteredGroupTree"
                :field-names="{ key: 'id', title: 'title', children: 'children' }"
                :default-expand-all="false"
                :block-node="true"
                :show-line="false"
                class="group-tree"
                @select="handleGroupSelect"
              >
                <template #title="nodeData">
                  <div class="tree-node">
                    <span class="node-icon">
                      <icon-apps v-if="!nodeData.parentId || nodeData.parentId === 0" style="color: #67c23a;" />
                      <icon-folder v-else style="color: #409eff;" />
                    </span>
                    <span class="node-label">{{ nodeData.title || nodeData.name }}</span>
                    <span class="node-count">({{ nodeData.hostCount || 0 }})</span>
                    <span class="node-actions" @click.stop>
                      <a-dropdown trigger="click" @select="(cmd: any) => handleGroupAction(cmd, nodeData)">
                        <icon-more class="more-icon" />
                        <template #content>
                          <a-doption :value="'edit'">
                            <icon-edit /> 编辑
                          </a-doption>
                          <a-doption :value="'delete'">
                            <icon-delete /> 删除
                          </a-doption>
                        </template>
                      </a-dropdown>
                    </span>
                  </div>
                </template>
              </a-tree>
            </a-spin>
            <a-empty v-if="filteredGroupTree.length === 0 && !groupLoading" description="暂无分组" />
          </div>
        </div>
      </div>

      <!-- 右侧主机列表/终端 -->
      <div class="right-panel">
        <!-- 主机列表视图 -->
        <div v-show="activeView === 'hosts'" class="view-container">
          <!-- 搜索和筛选栏 -->
          <div class="filter-bar">
          <div class="filter-inputs">
            <a-input
              v-model="searchForm.keyword"
              placeholder="搜索主机名/IP..."
              allow-clear
              class="filter-input"
              @input="handleSearch"
            >
              <template #prefix>
                <icon-search class="search-icon" />
              </template>
            </a-input>

            <a-select
              v-model="searchForm.status"
              placeholder="主机状态"
              allow-clear
              class="filter-input"
              @change="handleSearch"
            >
              <a-option label="在线" :value="1" />
              <a-option label="离线" :value="0" />
              <a-option label="未知" :value="-1" />
            </a-select>
          </div>
          <div class="filter-actions">
            <a-button
              v-if="selectedHosts.length > 0"
              type="primary"
              @click="handleBatchDeployAgent"
            >
              <template #icon><icon-cloud-download /></template>
              批量部署Agent ({{ selectedHosts.length }})
            </a-button>
            <a-button
              v-if="selectedHosts.length > 0"
              v-permission="'hosts:batch-delete'"
              type="primary"
              status="danger"
              @click="handleBatchDelete"
            >
              <template #icon><icon-delete /></template>
              批量删除 ({{ selectedHosts.length }})
            </a-button>
            <a-button class="reset-btn" @click="handleReset">
              <template #icon><icon-refresh /></template>
              重置
            </a-button>
            <a-button @click="loadHostList">
              <template #icon><icon-refresh /></template>
              刷新
            </a-button>
          </div>
        </div>

        <!-- 当前选择的分组 -->
        <div v-if="selectedGroup" class="selected-group-bar">
          <icon-folder-add />
          <span class="group-path">{{ getGroupPath(selectedGroup) }}</span>
          <a-tag size="small" color="gray">{{ hostPagination.total }} 台主机</a-tag>
          <a-button type="text" size="small" @click="clearGroupSelection">
            <icon-close />
          </a-button>
        </div>

        <!-- 主机列表 -->
        <div class="table-wrapper">
          <a-table
            :data="hostList"
            :loading="hostLoading"
            :bordered="{ cell: true }"
            stripe
            row-key="id"
            :row-selection="{ type: 'checkbox', showCheckedAll: true }"
            @selection-change="handleHostSelectionChange"
            :pagination="{ current: hostPagination.page, pageSize: hostPagination.pageSize, total: hostPagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50, 100] }"
            @page-change="(page: number) => { hostPagination.page = page; loadHostList() }"
            @page-size-change="(size: number) => { hostPagination.pageSize = size; hostPagination.page = 1; loadHostList() }"
            class="modern-table"
          >
            <template #columns>
              <a-table-column title="主机" data-index="name" :width="200" fixed="left">
                <template #cell="{ record }">
                  <div class="hostname-cell" @click="handleHostnameClick(record)" @dblclick.stop.prevent="handleTableRowDblClick(record)">
                    <div class="host-avatar" :class="`host-status-${record.status}`">
                      <icon-desktop />
                    </div>
                    <div class="host-info">
                      <div class="hostname hostname-clickable">{{ record.name }}</div>
                      <div class="host-meta">
                        <span class="ip">{{ record.ip }}</span>
                        <span class="port">:{{ record.port }}</span>
                      </div>
                    </div>
                  </div>
                </template>
              </a-table-column>
              <a-table-column title="状态" :width="70" align="center">
                <template #cell="{ record }">
                  <div class="status-cell">
                    <span class="status-dot" :class="`status-dot-${record.status}`"></span>
                    <span class="status-text" :class="`status-text-${record.status}`">{{ record.statusText }}</span>
                  </div>
                </template>
              </a-table-column>

              <a-table-column title="类型" :width="90" align="center">
                <template #cell="{ record }">
                  <a-tag v-if="record.type === 'cloud'" size="small" color="orangered">
                    {{ record.cloudProviderText || '云主机' }}
                  </a-tag>
                  <a-tag v-else size="small" color="gray">
                    自建
                  </a-tag>
                </template>
              </a-table-column>

              <a-table-column title="CPU" :width="120" align="center">
                <template #cell="{ record }">
                  <div class="resource-cell">
                    <div v-if="record.cpuCores" class="resource-info">
                      <span class="resource-label">{{ record.cpuCores }}核</span>
                      <a-progress
                        :percent="record.cpuUsage ? parseFloat((record.cpuUsage / 100).toFixed(3)) : 0"
                        :color="getUsageColor(record.cpuUsage)"
                        size="small"
                      />
                    </div>
                    <span v-else class="text-muted">-</span>
                  </div>
                </template>
              </a-table-column>

              <a-table-column title="内存" :width="140" align="center">
                <template #cell="{ record }">
                  <div class="resource-cell">
                    <div v-if="record.memoryTotal" class="resource-info">
                      <span class="resource-label resource-compact">{{ formatBytesCompact(record.memoryUsed) }} / {{ formatBytesCompact(record.memoryTotal) }}</span>
                      <a-progress
                        :percent="record.memoryUsage ? parseFloat((record.memoryUsage / 100).toFixed(3)) : 0"
                        :color="getUsageColor(record.memoryUsage)"
                        size="small"
                      />
                    </div>
                    <span v-else class="text-muted">-</span>
                  </div>
                </template>
              </a-table-column>
              <a-table-column title="磁盘" :width="140" align="center">
                <template #cell="{ record }">
                  <div class="resource-cell">
                    <div v-if="record.diskTotal" class="resource-info">
                      <span class="resource-label resource-compact">{{ formatBytesCompact(record.diskUsed) }} / {{ formatBytesCompact(record.diskTotal) }}</span>
                      <a-progress
                        :percent="record.diskUsage ? parseFloat((record.diskUsage / 100).toFixed(3)) : 0"
                        :color="getUsageColor(record.diskUsage)"
                        size="small"
                      />
                    </div>
                    <span v-else class="text-muted">-</span>
                  </div>
                </template>
              </a-table-column>

              <a-table-column title="标签" :width="120">
                <template #cell="{ record }">
                  <div v-if="record.tags && record.tags.length > 0" class="tags-cell">
                    <a-tag
                      v-for="(tag, index) in record.tags.slice(0, 2)"
                      :key="index"
                      size="small"
                      class="tag-item"
                    >
                      {{ tag }}
                    </a-tag>
                    <a-tag v-if="record.tags.length > 2" size="small" color="gray" class="tag-more">
                      +{{ record.tags.length - 2 }}
                    </a-tag>
                  </div>
                  <span v-else class="text-muted">-</span>
                </template>
              </a-table-column>

              <a-table-column title="系统信息" :width="140">
                <template #cell="{ record }">
                  <div v-if="record.os || record.arch" class="config-cell">
                    <div v-if="record.os" class="config-item">
                      <icon-computer />
                      <span class="config-text">{{ record.os }}</span>
                    </div>
                    <div v-if="record.arch" class="config-item">
                      <icon-code-block />
                      <span class="config-text">{{ record.arch }}</span>
                    </div>
                  </div>
                  <span v-else class="text-muted">-</span>
                </template>
              </a-table-column>

              <a-table-column title="连接方式" :width="100" align="center">
                <template #cell="{ record }">
                  <a-tag v-if="record.connectionMode === 'agent' && record.agentStatus === 'online'" color="green" size="small">
                    <icon-cloud /> Agent
                  </a-tag>
                  <a-tag v-else-if="record.connectionMode === 'agent'" color="orange" size="small">
                    <icon-cloud /> Agent离线
                  </a-tag>
                  <a-tag v-else color="gray" size="small">
                    SSH
                  </a-tag>
                </template>
              </a-table-column>

              <a-table-column title="操作" :width="260" fixed="right" align="center">
                <template #cell="{ record }">
                  <div class="action-buttons">
                    <a-tooltip content="采集信息" position="top">
                      <a-button
                        v-if="isAdmin || hasHostPermission(record.id, PERMISSION.COLLECT)"
                        v-permission="'hosts:collect'"
                        type="text"
                        @click="handleCollectHost(record)"
                      >
                        <icon-refresh />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip content="文件管理" position="top">
                      <a-button
                        v-if="isAdmin || hasHostPermission(record.id, PERMISSION.FILE)"
                        v-permission="'hosts:file-manage'"
                        type="text"
                        @click="handleFileManager(record)"
                      >
                      <template #icon>
        <icon-folder />
      </template>
                      
                      </a-button>
                    </a-tooltip>
                    <a-tooltip content="部署Agent" position="top">
                      <a-button
                        v-if="record.connectionMode !== 'agent' || record.agentStatus === 'none'"
                        type="text"
                        @click="handleDeployAgent(record)"
                        :loading="record._deploying"
                      >
                        <icon-cloud-download />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip content="更新Agent" position="top">
                      <a-button
                        v-if="record.connectionMode === 'agent' && record.agentStatus !== 'none'"
                        type="text"
                        @click="handleUpdateAgent(record)"
                        :loading="record._updating"
                      >
                        <icon-sync />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip content="卸载Agent" position="top">
                      <a-button
                        v-if="record.connectionMode === 'agent' && record.agentStatus !== 'none'"
                        type="text"
                        status="danger"
                        @click="handleUninstallAgent(record)"
                        :loading="record._uninstalling"
                      >
                        <icon-poweroff />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip content="编辑" position="top">
                      <a-button
                        v-if="isAdmin || hasHostPermission(record.id, PERMISSION.EDIT)"
                        v-permission="'hosts:update'"
                        type="text"
                        @click="handleEditHost(record)"
                      >
                        <icon-edit />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip content="删除" position="top">
                      <a-button
                        v-if="isAdmin || hasHostPermission(record.id, PERMISSION.DELETE)"
                        v-permission="'hosts:delete'"
                        type="text"
                        status="danger"
                        @click="handleDeleteHost(record)"
                      >
                        <icon-delete />
                      </a-button>
                    </a-tooltip>
                  </div>
                </template>
              </a-table-column>
            </template>
          </a-table>
        </div>
        </div>

        <!-- 终端视图 -->
        <div v-show="activeView === 'terminal'" class="view-container terminal-view">
          <div class="terminal-view-header">
            <div class="terminal-view-title">
              <icon-desktop />
              <span>Web终端</span>
              <span v-if="activeTerminalHost" class="terminal-current-group">
                / {{ activeTerminalHost.name }}
              </span>
            </div>
            <a-button size="small" @click="switchToHostsView">
              <template #icon><icon-left /></template>
              返回主机列表
            </a-button>
          </div>
          <div class="terminal-content">
            <!-- 资产分组树（左） -->
            <div class="terminal-sidebar">
              <div class="panel-header">
                <div class="panel-title">
                  <icon-apps class="panel-icon" />
                  <span>资产分组</span>
                </div>
                <div class="panel-actions">
                  <a-tooltip :content="isExpandAll ? '折叠全部' : '展开全部'" position="top">
                    <a-button shape="circle" size="small" @click="toggleExpandAll">
                      <icon-swap />
                    </a-button>
                  </a-tooltip>
                </div>
              </div>
              <div class="panel-body">
                <a-input
                  v-model="groupSearchKeyword"
                  placeholder="搜索分组..."
                  allow-clear
                  size="small"
                  class="group-search"
                  @input="filterGroupTree"
                >
                  <template #prefix>
                    <icon-search />
                  </template>
                </a-input>
                <div class="tree-container">
                  <a-spin :loading="groupLoading" style="width: 100%;">
                    <a-tree
                      ref="groupTreeRef"
                      :data="terminalGroupTree"
                      :field-names="{ key: 'id', title: 'title', children: 'children' }"
                      :default-expand-all="false"
                      :block-node="true"
                      :show-line="false"
                    >
                      <template #title="nodeData">
                        <div @dblclick="handleHostDblClick(nodeData)" :style="{ cursor: nodeData.type === 'host' ? 'pointer' : 'default' }">
                          <span>
                            <icon-apps v-if="nodeData.type === 'group' || !nodeData.parentId || nodeData.parentId === 0" style="color: #67c23a;" />
                            <icon-desktop v-else-if="nodeData.type === 'host'" :style="{ color: getStatusColor(nodeData.status) }" />
                            <icon-folder v-else style="color: #409eff;" />
                          </span>
                          <span >{{ nodeData.title || nodeData.name }}</span>
                          <span v-if="nodeData.type === 'group' || !nodeData.type" class="node-count">({{ nodeData.hostCount || 0 }})</span>
                          <span v-if="nodeData.type === 'host'">{{ nodeData.ip }}</span>
                        </div>
                      </template>
                    </a-tree>
                  </a-spin>
                  <a-empty v-if="!terminalGroupTree || terminalGroupTree.length === 0" description="暂无数据" />
                </div>
              </div>
            </div>
            <!-- 终端区域（右） -->
            <div class="terminal-main">
              <div v-if="activeTerminalHost" class="terminal-header">
                <div class="terminal-info">
                  <icon-desktop class="terminal-icon" />
                  <div class="terminal-details">
                    <div class="terminal-title">{{ activeTerminalHost.name }}</div>
                    <div class="terminal-meta">
                      <span class="terminal-ip">{{ activeTerminalHost.ip }}:{{ activeTerminalHost.port }}</span>
                      <span class="terminal-user">{{ activeTerminalHost.sshUser }}</span>
                    </div>
                  </div>
                </div>
                <div class="terminal-actions">
                  <a-button size="small" @click="closeTerminal">
                    <template #icon><icon-close /></template>
                    关闭
                  </a-button>
                </div>
              </div>
              <div v-else class="terminal-placeholder">
                <icon-desktop class="placeholder-icon" />
                <div class="placeholder-text">请双击左侧主机连接终端</div>
                <div class="placeholder-hint">展开分组查看主机，双击主机即可连接</div>
              </div>
              <div v-if="activeTerminalHost" class="terminal-body">
                <div class="terminal-wrapper">
                  <div ref="terminalRef" class="xterm-container"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 直接导入对话框 -->
    <a-modal
      v-model:visible="directImportVisible"
      :title="hostForm.id && hostForm.id > 0 ? '编辑主机' : '新增主机'"
      :width="800"
      unmount-on-close
      class="host-import-dialog responsive-dialog"
      @close="handleDirectImportClose"
    >
      <a-form :model="hostForm" :rules="hostRules" ref="hostFormRef" auto-label-width layout="horizontal">
        <a-row :gutter="20">
          <a-col :span="12">
            <a-form-item label="主机名称" field="name">
              <a-input v-model="hostForm.name" placeholder="请输入主机名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="主机类型" field="type">
              <a-select v-model="hostForm.type" placeholder="请选择主机类型">
                <a-option value="self">
                  <div style="display: flex; align-items: center; gap: 8px;">
                    <icon-desktop />
                    <span>自建主机</span>
                  </div>
                </a-option>
                <a-option value="cloud">
                  <div style="display: flex; align-items: center; gap: 8px;">
                    <icon-cloud />
                    <span>云主机</span>
                  </div>
                </a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="12">
            <a-form-item label="所属分组" field="groupId">
              <a-tree-select
                v-model="hostForm.groupId"
                :data="groupTreeOptions"
                :field-names="{ key: 'id', title: 'name', children: 'children' }"
                allow-clear
                placeholder="请选择分组"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12"></a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="12">
            <a-form-item label="IP地址" field="ip">
              <a-input v-model="hostForm.ip" placeholder="请输入IP地址" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="SSH端口" field="port">
              <a-input-number v-model="hostForm.port" :min="1" :max="65535" :step="1" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="12">
            <a-form-item label="SSH用户名" field="sshUser">
              <a-input v-model="hostForm.sshUser" placeholder="如：root" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="认证凭据">
              <a-select v-model="hostForm.credentialId" placeholder="选择或新建凭证" allow-clear allow-search>
                <a-option
                  v-for="cred in credentialList"
                  :key="cred.id"
                  :label="`${cred.name} (${cred.typeText})`"
                  :value="cred.id"
                />
                <template #footer>
                  <a-button type="text" @click="showCredentialDialog = true" long>
                    <icon-plus /> 新建凭证
                  </a-button>
                </template>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="24">
            <a-form-item label="主机标签">
              <a-select v-model="hostTagsArray" multiple allow-create allow-search placeholder="选择或输入标签，回车确认">
                <a-option v-for="label in serviceLabelOptions" :key="label" :value="label">{{ label }}</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="24">
            <a-form-item label="备注">
              <a-textarea v-model="hostForm.description" :auto-size="{ minRows: 3 }" placeholder="请输入备注信息" />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="directImportVisible = false">取消</a-button>
          <a-button type="primary" @click="handleDirectImportSubmit" :loading="hostSubmitting">
            {{ hostForm.id && hostForm.id > 0 ? '保存' : '确定' }}
          </a-button>
        </div>
      </template>
    </a-modal>

    <!-- 主机详情对话框 -->
    <a-modal
      v-model:visible="showHostDetailDialog"
      title=""
      :width="960"
      unmount-on-close
      class="host-detail-dialog"
      @close="handleCloseHostDetail"
    >
      <template #title>
        <div class="detail-dialog-header">
          <div class="detail-header-left">
            <div class="host-avatar-lg" :class="`host-status-${hostDetail?.status}`">
              <icon-desktop />
            </div>
            <div class="detail-header-info">
              <div class="detail-hostname">{{ hostDetail?.name }}</div>
              <div class="detail-hostmeta">
                <span class="detail-ip">{{ hostDetail?.ip }}:{{ hostDetail?.port }}</span>
                <a-tag :color="getStatusTagColor(hostDetail?.status)" size="small">{{ hostDetail?.statusText }}</a-tag>
                <a-tag v-if="hostDetail?.type === 'cloud'" size="small" color="orangered">{{ hostDetail?.cloudProviderText || '云主机' }}</a-tag>
                <a-tag v-else size="small" color="gray">自建主机</a-tag>
              </div>
            </div>
          </div>
        </div>
      </template>
      <a-spin :loading="hostDetailLoading" class="host-detail-content">
        <template v-if="hostDetail">
          <!-- 资源概览 -->
          <div class="detail-resource-row" v-if="hostDetail.cpuCores || hostDetail.memoryTotal">
            <div class="detail-resource-item">
              <div class="detail-resource-icon cpu-bg"><icon-code-block /></div>
              <div class="detail-resource-info">
                <div class="detail-resource-label">CPU</div>
                <div class="detail-resource-val">{{ hostDetail.cpuCores || '-' }}核</div>
              </div>
              <div class="detail-resource-bar">
                <a-progress
                  :percent="hostDetail.cpuUsage ? parseFloat((hostDetail.cpuUsage / 100).toFixed(3)) : 0"
                  :color="getUsageColor(hostDetail.cpuUsage)"
                  :stroke-width="6"
                  :show-text="false"
                />
                <span class="detail-resource-pct" :class="getUsageLevel(hostDetail.cpuUsage)">{{ hostDetail.cpuUsage ? hostDetail.cpuUsage.toFixed(1) : '0' }}%</span>
              </div>
            </div>
            <div class="detail-resource-item">
              <div class="detail-resource-icon mem-bg"><icon-storage /></div>
              <div class="detail-resource-info">
                <div class="detail-resource-label">内存</div>
                <div class="detail-resource-val">{{ formatBytes(hostDetail.memoryTotal) }}</div>
              </div>
              <div class="detail-resource-bar">
                <a-progress
                  :percent="hostDetail.memoryUsage ? parseFloat((hostDetail.memoryUsage / 100).toFixed(3)) : 0"
                  :color="getUsageColor(hostDetail.memoryUsage)"
                  :stroke-width="6"
                  :show-text="false"
                />
                <span class="detail-resource-pct" :class="getUsageLevel(hostDetail.memoryUsage)">{{ hostDetail.memoryUsage ? hostDetail.memoryUsage.toFixed(1) : '0' }}%</span>
              </div>
            </div>
            <div class="detail-resource-item">
              <div class="detail-resource-icon disk-bg"><icon-common /></div>
              <div class="detail-resource-info">
                <div class="detail-resource-label">磁盘</div>
                <div class="detail-resource-val">{{ formatBytes(hostDetail.diskTotal) }}</div>
              </div>
              <div class="detail-resource-bar">
                <a-progress
                  :percent="hostDetail.diskUsage ? parseFloat((hostDetail.diskUsage / 100).toFixed(3)) : 0"
                  :color="getUsageColor(hostDetail.diskUsage)"
                  :stroke-width="6"
                  :show-text="false"
                />
                <span class="detail-resource-pct" :class="getUsageLevel(hostDetail.diskUsage)">{{ hostDetail.diskUsage ? hostDetail.diskUsage.toFixed(1) : '0' }}%</span>
              </div>
            </div>
          </div>

          <!-- 平铺信息区 -->
          <div class="detail-flat-section">
            <div class="detail-section-title">基本信息</div>
            <div class="detail-flat-grid">
              <div class="detail-flat-item">
                <span class="detail-flat-label">SSH用户</span>
                <span class="detail-flat-value">{{ hostDetail.sshUser || '-' }}</span>
              </div>
              <div class="detail-flat-item">
                <span class="detail-flat-label">所属分组</span>
                <span class="detail-flat-value">{{ hostDetail.groupName || '未分组' }}</span>
              </div>
              <div class="detail-flat-item">
                <span class="detail-flat-label">最后连接</span>
                <span class="detail-flat-value">{{ hostDetail.lastSeen || '未连接' }}</span>
              </div>
              <div class="detail-flat-item">
                <span class="detail-flat-label">创建时间</span>
                <span class="detail-flat-value">{{ hostDetail.createTime || '-' }}</span>
              </div>
              <div class="detail-flat-item" v-if="hostDetail.credential">
                <span class="detail-flat-label">凭证名称</span>
                <span class="detail-flat-value">{{ hostDetail.credential.name }}</span>
              </div>
              <div class="detail-flat-item" v-if="hostDetail.credential">
                <span class="detail-flat-label">认证方式</span>
                <span class="detail-flat-value">
                  <a-tag :color="hostDetail.credential.type === 'password' ? 'orangered' : 'green'" size="small">{{ hostDetail.credential.typeText }}</a-tag>
                </span>
              </div>
            </div>
          </div>

          <div class="detail-flat-section">
            <div class="detail-section-title">系统信息</div>
            <div class="detail-flat-grid">
              <div class="detail-flat-item detail-flat-item-wide">
                <span class="detail-flat-label">操作系统</span>
                <span class="detail-flat-value">{{ hostDetail.os || '-' }}</span>
              </div>
              <div class="detail-flat-item">
                <span class="detail-flat-label">内核版本</span>
                <span class="detail-flat-value">{{ hostDetail.kernel || '-' }}</span>
              </div>
              <div class="detail-flat-item">
                <span class="detail-flat-label">系统架构</span>
                <span class="detail-flat-value">{{ hostDetail.arch || '-' }}</span>
              </div>
              <div class="detail-flat-item">
                <span class="detail-flat-label">主机名</span>
                <span class="detail-flat-value">{{ hostDetail.hostname || '-' }}</span>
              </div>
              <div class="detail-flat-item">
                <span class="detail-flat-label">运行时间</span>
                <span class="detail-flat-value">{{ hostDetail.uptime || '-' }}</span>
              </div>
            </div>
          </div>

          <!-- 标签 -->
          <div class="detail-flat-section" v-if="hostDetail.tags && hostDetail.tags.length > 0">
            <div class="detail-section-title">标签</div>
            <div class="detail-tags-row">
              <a-tag v-for="(tag, index) in hostDetail.tags" :key="index" size="medium" color="arcoblue">{{ tag }}</a-tag>
            </div>
          </div>

          <!-- 备注 -->
          <div class="detail-flat-section" v-if="hostDetail.description">
            <div class="detail-section-title">备注</div>
            <div class="detail-remark-text">{{ hostDetail.description }}</div>
          </div>
        </template>
      </a-spin>
      <template #footer>
        <a-button @click="showHostDetailDialog = false">关闭</a-button>
        <a-button type="primary" @click="handleCollectHostFromDetail" :loading="hostDetailLoading">
          <template #icon><icon-refresh /></template>
          采集信息
        </a-button>
      </template>
    </a-modal>

    <!-- Agent部署/更新对话框 -->
    <a-modal
      v-model:visible="showAgentDeployDialog"
      :title="agentDeployMode === 'deploy' ? '部署Agent' : agentDeployMode === 'update' ? '更新Agent' : '批量部署Agent'"
      :width="520"
      unmount-on-close
      @ok="confirmAgentDeploy"
      @cancel="showAgentDeployDialog = false"
    >
      <div style="margin-bottom: 16px; color: var(--ops-text-secondary, #4e5969); font-size: 14px;">
        <template v-if="agentDeployMode === 'batch'">
          即将在选中的 {{ selectedHosts.length }} 台主机上部署Agent。
        </template>
        <template v-else-if="agentDeployMode === 'update'">
          即将更新主机 {{ agentDeployTarget?.name }} ({{ agentDeployTarget?.ip }}) 上的Agent，更新期间服务将短暂中断。
        </template>
        <template v-else>
          即将在主机 {{ agentDeployTarget?.name }} ({{ agentDeployTarget?.ip }}) 上部署Agent。
        </template>
      </div>
      <a-form :model="{}" layout="vertical">
        <a-form-item label="服务端连接地址">
          <a-input
            v-model="agentDeployServerAddr"
            placeholder="留空则自动检测，如有NAT映射请手动填写"
            allow-clear
          />
          <template #extra>
            <div style="color: var(--ops-text-tertiary, #86909c); font-size: 12px; margin-top: 4px;">
              Agent安装后会通过此地址连接服务端。格式：IP 或 IP:Port。默认自动检测，如果服务端IP经过NAT映射，请填写映射后的地址。
            </div>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Excel导入对话框 -->
    <a-modal
      v-model:visible="excelImportVisible"
      title="Excel批量导入"
      :width="660"
      unmount-on-close
      class="excel-import-dialog responsive-dialog"
      @close="handleExcelImportClose"
    >
      <div class="excel-import-content">
        <a-alert title="导入说明" type="info" :closable="false" style="margin-bottom: 20px;">
          <ul style="margin: 8px 0 0 0; padding-left: 20px;">
            <li>请先下载Excel模板文件</li>
            <li>按照模板格式填写主机信息</li>
            <li>支持批量导入多台主机</li>
            <li>IP地址重复的主机会自动跳过</li>
          </ul>
        </a-alert>

        <a-form :model="excelImportForm" auto-label-width layout="horizontal">
          <a-form-item label="主机类型">
            <a-select v-model="excelImportForm.type" placeholder="请选择主机类型">
              <a-option value="self">
                <div style="display: flex; align-items: center; gap: 8px;">
                  <icon-desktop />
                  <span>自建主机</span>
                </div>
              </a-option>
              <a-option value="cloud">
                <div style="display: flex; align-items: center; gap: 8px;">
                  <icon-cloud />
                  <span>云主机</span>
                </div>
              </a-option>
            </a-select>
          </a-form-item>

          <a-form-item label="所属分组">
            <a-tree-select
              v-model="excelImportForm.groupId"
              :data="groupTreeOptions"
              :field-names="{ key: 'id', title: 'name', children: 'children' }"
              allow-clear
              placeholder="请选择默认分组"
            />
          </a-form-item>

          <a-form-item label="下载模板">
            <a-button @click="downloadTemplate">
              <template #icon><icon-download /></template>
              下载Excel模板
            </a-button>
          </a-form-item>

          <a-form-item label="上传文件">
            <a-upload
              ref="uploadRef"
              draggable
              :auto-upload="false"
              @change="handleFileChange"
              :limit="1"
              accept=".xlsx,.xls"
            >
              <template #upload-button>
                <div class="upload-drag-area">
                  <icon-upload style="font-size: 28px; color: var(--ops-text-tertiary);" />
                  <div>将文件拖到此处，或<span style="color: var(--ops-primary);">点击上传</span></div>
                  <div style="font-size: 12px; color: var(--ops-text-tertiary);">只支持 .xlsx 或 .xls 格式的Excel文件</div>
                </div>
              </template>
            </a-upload>
          </a-form-item>
          <a-form-item v-if="uploadedFile" label="已选择文件">
            <div class="file-info">
              <icon-file />
              <span>{{ uploadedFile.name }}</span>
              <a-tag size="small" color="green">{{ (uploadedFile.size / 1024).toFixed(2) }} KB</a-tag>
            </div>
          </a-form-item>
        </a-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="excelImportVisible = false">取消</a-button>
          <a-button type="primary" @click="handleExcelImportSubmit" :loading="excelImporting">开始导入</a-button>
        </div>
      </template>
    </a-modal>

    <!-- 云主机导入对话框 -->
    <a-modal
      v-model:visible="cloudImportVisible"
      title="云主机导入"
      :width="920"
      unmount-on-close
      class="cloud-import-dialog responsive-dialog"
      @close="handleCloudImportClose"
    >
      <a-steps :current="cloudImportStep + 1" style="margin-bottom: 30px;">
        <a-step title="选择云平台" />
        <a-step title="选择主机" />
        <a-step title="确认导入" />
      </a-steps>

      <!-- 步骤1: 选择云平台 -->
      <div v-if="cloudImportStep === 0" class="cloud-step-content">
        <a-form :model="cloudImportForm" auto-label-width layout="horizontal">
          <a-form-item label="云平台">
            <a-select v-model="cloudImportForm.accountId" placeholder="请选择云平台账号" allow-search>
              <a-option
                v-for="account in enabledCloudAccounts"
                :key="account.id"
                :label="`${account.name} (${account.providerText})`"
                :value="account.id"
              >
                <div style="display: flex; justify-content: space-between; align-items: center;">
                  <span>{{ account.name }}</span>
                  <a-tag size="small" :color="account.provider === 'aliyun' ? 'orangered' : 'arcoblue'">
                    {{ account.providerText }}
                  </a-tag>
                </div>
              </a-option>
              <template #footer>
                <a-button type="text" @click="showCloudAccountDialog = true" long>
                  <icon-plus /> 新增云平台账号
                </a-button>
              </template>
            </a-select>
          </a-form-item>
          <a-form-item label="区域">
            <a-select v-model="cloudImportForm.region" placeholder="请先选择云平台账号" allow-search :loading="loadingCloudRegions">
              <a-option v-for="region in cloudRegions" :key="region.value" :label="region.label" :value="region.value" />
            </a-select>
          </a-form-item>

          <a-form-item label="导入到分组">
            <a-tree-select
              v-model="cloudImportForm.groupId"
              :data="groupTreeOptions"
              :field-names="{ key: 'id', title: 'name', children: 'children' }"
              allow-clear
              placeholder="请选择分组"
            />
          </a-form-item>
        </a-form>

        <a-alert title="提示" type="info" :closable="false">
          <ul style="margin: 8px 0 0 0; padding-left: 20px;">
            <li>请先添加云平台账号（Access Key / Secret Key）</li>
            <li>系统将自动获取该账号下指定区域的ECS实例</li>
            <li>支持阿里云、腾讯云等主流云厂商</li>
          </ul>
        </a-alert>
      </div>

      <!-- 步骤2: 选择主机 -->
      <div v-if="cloudImportStep === 1" class="cloud-step-content">
        <div class="step-header">
          <span>找到 {{ cloudHostList.length }} 台云主机，请选择要导入的主机</span>
          <a-checkbox v-model="selectAllCloudHosts" @change="handleSelectAllCloudHosts">全选</a-checkbox>
        </div>

        <a-table
          :data="cloudHostList"
          :loading="loadingCloudHosts"
          :row-selection="{ type: 'checkbox', showCheckedAll: true }"
          @selection-change="handleCloudHostSelectionChange"
          :scroll="{ y: 400 }"
          row-key="instanceId"
          :bordered="{ cell: true }"
          stripe
        >
          <template #columns>
            <a-table-column title="实例名称" data-index="name" :width="150" />
            <a-table-column title="实例ID" data-index="instanceId" :width="180" />
            <a-table-column title="公网IP" data-index="publicIp" :width="140" />
            <a-table-column title="私网IP" data-index="privateIp" :width="140" />
            <a-table-column title="操作系统" data-index="os" :width="150" :tooltip="true" />
            <a-table-column title="状态" :width="80">
              <template #cell="{ record }">
                <a-tag :color="record.status === 'Running' ? 'green' : 'gray'" size="small">
                  {{ record.status }}
                </a-tag>
              </template>
            </a-table-column>
          </template>
        </a-table>

        <a-empty v-if="cloudHostList.length === 0 && !loadingCloudHosts" description="暂无云主机数据" />
      </div>
      <!-- 步骤3: 确认导入 -->
      <div v-if="cloudImportStep === 2" class="cloud-step-content">
        <a-result status="success" title="准备就绪" subtitle="以下主机将被导入到系统中">
          <template #extra>
            <div class="import-summary">
              <a-descriptions :column="1" bordered>
                <a-descriptions-item label="云平台账号">{{ selectedCloudAccount?.name }}</a-descriptions-item>
                <a-descriptions-item label="目标分组">{{ selectedCloudGroup?.name }}</a-descriptions-item>
                <a-descriptions-item label="待导入主机">{{ selectedCloudHosts.length }} 台</a-descriptions-item>
              </a-descriptions>

              <div class="host-list-preview">
                <h4>主机列表：</h4>
                <a-tag
                  v-for="host in selectedCloudHosts"
                  :key="host.instanceId"
                  style="margin: 4px;"
                >
                  {{ host.name }}
                </a-tag>
              </div>
            </div>
          </template>
        </a-result>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <a-button v-if="cloudImportStep > 0" @click="cloudImportStep--">上一步</a-button>
          <a-button @click="cloudImportVisible = false">取消</a-button>
          <a-button
            v-if="cloudImportStep === 0"
            type="primary"
            @click="handleGetCloudHosts"
            :loading="loadingCloudHosts"
          >
            下一步
          </a-button>
          <a-button
            v-if="cloudImportStep === 1"
            type="primary"
            @click="cloudImportStep++"
            :disabled="selectedCloudHosts.length === 0"
          >
            下一步
          </a-button>
          <a-button
            v-if="cloudImportStep === 2"
            type="primary"
            @click="handleCloudImportSubmit"
            :loading="cloudImporting"
          >
            开始导入
          </a-button>
        </div>
      </template>
    </a-modal>
    <!-- 新建凭证对话框 -->
    <a-modal
      v-model:visible="showCredentialDialog"
      title="新建凭证"
      :width="660"
      unmount-on-close
      class="credential-dialog responsive-dialog"
      @close="handleCredentialDialogClose"
    >
      <a-form :model="credentialForm" :rules="credentialRules" ref="credentialFormRef" auto-label-width layout="horizontal">
        <a-form-item label="凭证名称" field="name">
          <a-input v-model="credentialForm.name" placeholder="请输入凭证名称，如：生产环境root凭证" />
        </a-form-item>

        <a-form-item label="认证方式" field="type">
          <a-radio-group v-model="credentialForm.type" @change="handleAuthTypeChange">
            <a-radio :value="'password'">密码认证</a-radio>
            <a-radio :value="'key'">密钥认证</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="credentialForm.type === 'password'" label="用户名" field="username">
          <a-input v-model="credentialForm.username" placeholder="如：root" />
        </a-form-item>

        <a-form-item v-if="credentialForm.type === 'password'" label="密码" field="password">
          <a-input-password v-model="credentialForm.password" placeholder="请输入密码" />
        </a-form-item>

        <a-form-item v-if="credentialForm.type === 'key'" label="用户名">
          <a-input v-model="credentialForm.username" placeholder="如：root（可选）" />
        </a-form-item>

        <a-form-item v-if="credentialForm.type === 'key'" label="私钥" field="privateKey">
          <a-textarea
            v-model="credentialForm.privateKey"
            :auto-size="{ minRows: 8 }"
            placeholder="请粘贴PEM格式的私钥内容"
          />
        </a-form-item>

        <a-form-item v-if="credentialForm.type === 'key'" label="私钥密码">
          <a-input-password v-model="credentialForm.passphrase" placeholder="如果私钥有密码请输入（可选）" />
        </a-form-item>

        <a-form-item label="备注">
          <a-textarea v-model="credentialForm.description" :auto-size="{ minRows: 2 }" placeholder="请输入备注信息" />
        </a-form-item>
      </a-form>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="showCredentialDialog = false">取消</a-button>
          <a-button type="primary" @click="handleCredentialSubmit" :loading="credentialSubmitting">确定</a-button>
        </div>
      </template>
    </a-modal>
    <!-- 新增云平台账号对话框 -->
    <a-modal
      v-model:visible="showCloudAccountDialog"
      title="新增云平台账号"
      :width="660"
      unmount-on-close
      class="cloud-account-dialog responsive-dialog"
      @close="handleCloudAccountDialogClose"
    >
      <a-form :model="cloudAccountForm" :rules="cloudAccountRules" ref="cloudAccountFormRef" auto-label-width layout="horizontal">
        <a-form-item label="账号名称" field="name">
          <a-input v-model="cloudAccountForm.name" placeholder="请输入账号名称，如：阿里云生产账号" />
        </a-form-item>

        <a-form-item label="云厂商" field="provider">
          <a-select v-model="cloudAccountForm.provider" placeholder="请选择云厂商">
            <a-option label="阿里云" value="aliyun" />
            <a-option label="腾讯云" value="tencent" />
            <a-option label="AWS" value="aws" />
            <a-option label="华为云" value="huawei" />
          </a-select>
        </a-form-item>

        <a-form-item label="Access Key" field="accessKey">
          <a-input v-model="cloudAccountForm.accessKey" placeholder="请输入Access Key ID" />
        </a-form-item>

        <a-form-item label="Secret Key" field="secretKey">
          <a-input-password v-model="cloudAccountForm.secretKey" placeholder="请输入Access Key Secret" />
        </a-form-item>

        <a-form-item label="默认区域">
          <a-input v-model="cloudAccountForm.region" placeholder="如：cn-hangzhou（可选）" />
        </a-form-item>

        <a-form-item label="备注">
          <a-textarea v-model="cloudAccountForm.description" :auto-size="{ minRows: 2 }" placeholder="请输入备注信息" />
        </a-form-item>
      </a-form>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="showCloudAccountDialog = false">取消</a-button>
          <a-button type="primary" @click="handleCloudAccountSubmit" :loading="cloudAccountSubmitting">确定</a-button>
        </div>
      </template>
    </a-modal>
    <!-- 新增/编辑分组对话框 -->
    <a-modal
      v-model:visible="groupDialogVisible"
      :title="groupDialogTitle"
      :width="660"
      unmount-on-close
      class="group-edit-dialog responsive-dialog"
      @close="handleGroupDialogClose"
    >
      <a-form :model="groupForm" :rules="groupRules" ref="groupFormRef" auto-label-width layout="horizontal">
        <a-form-item label="上级分组">
          <a-tree-select
            v-model="groupForm.parentId"
            :data="groupTreeOptions"
            :field-names="{ key: 'id', title: 'name', children: 'children' }"
            allow-clear
            placeholder="不选择则为顶级分组"
          />
        </a-form-item>
        <a-form-item label="分组名称" field="name">
          <a-input v-model="groupForm.name" placeholder="请输入分组名称" />
        </a-form-item>
        <a-form-item label="分组编码" field="code">
          <a-input v-model="groupForm.code" placeholder="请输入分组编码" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model="groupForm.description" :auto-size="{ minRows: 3 }" placeholder="请输入描述" />
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model="groupForm.sort" :min="0" />
        </a-form-item>
        <a-form-item label="状态" field="status">
          <a-radio-group v-model="groupForm.status">
            <a-radio :value="1">正常</a-radio>
            <a-radio :value="0">停用</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="groupDialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleGroupSubmit" :loading="groupSubmitting">确定</a-button>
        </div>
      </template>
    </a-modal>

    <!-- 文件浏览器对话框 -->
    <HostFileBrowser
      v-model:visible="fileBrowserVisible"
      :hostId="selectedHostId"
      :hostName="selectedHostName"
      :connectionMode="selectedHostConnectionMode"
      :agentStatus="selectedHostAgentStatus"
    />

    <!-- 生成Agent安装包对话框 -->
    <a-modal v-model:visible="showInstallPackageDialog" title="生成Agent安装包" :width="520" :footer="false" @cancel="installPackageResult = null">
      <a-space direction="vertical" fill style="width: 100%;">
        <a-form layout="vertical">
          <a-form-item label="服务端地址" help="Agent连接的服务端IP或域名">
            <a-input v-model="installPackageServerAddr" placeholder="如: 192.168.1.100" />
          </a-form-item>
        </a-form>
        <a-button type="primary" :loading="installPackageLoading" long @click="handleGenerateInstallPackage">生成安装包</a-button>
        <template v-if="installPackageResult">
          <a-alert type="success" style="margin-top: 12px;">
            <template #title>安装包已生成</template>
            <div>Agent ID: <a-typography-text copyable>{{ installPackageResult.agentId }}</a-typography-text></div>
            <div style="margin-top: 8px;">
              <a-link :href="installPackageResult.downloadUrl" target="_blank">下载安装包</a-link>
              <span style="color: var(--color-text-3); margin-left: 8px;">（30分钟内有效）</span>
            </div>
          </a-alert>
        </template>
      </a-space>
    </a-modal>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch, nextTick, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'
import 'xterm/lib/xterm.js'
import { Message, Modal } from '@arco-design/web-vue'
import type { FieldRule } from '@arco-design/web-vue'
import {
  IconDesktop,
  IconPlus,
  IconDown,
  IconFile,
  IconUpload,
  IconCloud,
  IconApps,
  IconSwap,
  IconSearch,
  IconFolder,
  IconMore,
  IconEdit,
  IconDelete,
  IconRefresh,
  IconFolderAdd,
  IconClose,
  IconComputer,
  IconCodeBlock,
  IconLeft,
  IconInfoCircle,
  IconLock,
  IconBarChart,
  IconStorage,
  IconCommon,
  IconDownload
} from '@arco-design/web-vue/es/icon'
import HostFileBrowser from './components/HostFileBrowser.vue'
import {
  getGroupTree,
  createGroup,
  updateGroup,
  deleteGroup
} from '@/api/assetGroup'
import {
  getHostList,
  getHost,
  createHost,
  updateHost,
  deleteHost,
  getCredentials,
  createCredential,
  getCloudAccounts,
  createCloudAccount,
  importFromCloud,
  getCloudRegions,
  getCloudInstances,
  collectHostInfo,
  testHostConnection,
  batchCollectHostInfo,
  downloadExcelTemplate,
  importFromExcel,
  batchDeleteHosts
} from '@/api/host'
import type { CloudInstanceVO, CloudRegionVO } from '@/api/host'
import { deployAgent, batchDeployAgent, getAgentStatuses, updateAgent, uninstallAgent, generateInstallPackage } from '@/api/agent'
import { getServiceLabels } from '@/api/serviceLabel'
import { PERMISSION, hasPermission } from '@/utils/permission'
import { getUserHostPermissions } from '@/api/assetPermission'
import { useUserStore } from '@/stores/user'
import { usePermissionStore } from '@/stores/permission'

// 路由
const router = useRouter()

// 用户状态
const userStore = useUserStore()
const permissionStore = usePermissionStore()

// 检查当前用户是否是管理员
const isAdmin = computed(() => {
  const roles = userStore.userInfo?.roles || []
  return roles.some((r: any) => r.code === 'admin')
})

// 加载状态
const groupLoading = ref(false)
const hostLoading = ref(false)
const hostSubmitting = ref(false)
const excelImporting = ref(false)
const cloudImporting = ref(false)
const loadingCloudHosts = ref(false)
const credentialSubmitting = ref(false)
const cloudAccountSubmitting = ref(false)
const groupSubmitting = ref(false)

// 视图状态
const activeView = ref('hosts') // 'hosts' | 'terminal'
const activeTerminalHost = ref<any>(null)
const terminalHostList = ref<any[]>([])
const terminalSearchKeyword = ref('')

// 终端相关
const terminalRef = ref<HTMLElement | null>(null)
const terminal = ref<Terminal | null>(null)
const fitAddon = ref<FitAddon | null>(null)
const ws = ref<WebSocket | null>(null)

// 对话框状态
const directImportVisible = ref(false)
const excelImportVisible = ref(false)
const cloudImportVisible = ref(false)
const showCredentialDialog = ref(false)
const showCloudAccountDialog = ref(false)
const fileBrowserVisible = ref(false)
const selectedHostId = ref(0)
const selectedHostName = ref('')
const selectedHostConnectionMode = ref('')
const selectedHostAgentStatus = ref('')

// Agent安装包生成
const showInstallPackageDialog = ref(false)
const installPackageServerAddr = ref('')
const installPackageLoading = ref(false)
const installPackageResult = ref<{ agentId: string; downloadUrl: string } | null>(null)

// 主机详情
const showHostDetailDialog = ref(false)
const hostDetail = ref<any>(null)
const hostDetailLoading = ref(false)
const groupDialogVisible = ref(false)

const groupDialogTitle = ref('')
const isGroupEdit = ref(false)

// 表单引用
const hostFormRef = ref()
const credentialFormRef = ref()
const cloudAccountFormRef = ref()
const groupFormRef = ref()
const groupTreeRef = ref()
const uploadRef = ref()

// 分组树数据
const groupTree = ref<any[]>([])
const filteredGroupTree = ref<any[]>([])
const groupSearchKeyword = ref('')
const selectedGroup = ref<any>(null)
const isExpandAll = ref(false)

// 主机列表数据
const hostList = ref([])
const hostPermissions = ref<Map<number, number>>(new Map())
const userHasEditPermission = ref(false)
const credentialList = ref([])
const cloudAccountList = ref([])
const serviceLabelOptions = ref<string[]>([])

// 搜索表单
const searchForm = reactive({
  keyword: '',
  status: undefined as number | undefined
})

// 主机分页
const hostPagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 主机表单
const hostForm = reactive({
  id: 0,
  name: '',
  groupId: null as number | null,
  type: 'self',
  cloudProvider: '',
  cloudInstanceId: '',
  cloudAccountId: null as number | null,
  sshUser: 'root',
  ip: '',
  port: 22,
  credentialId: null as number | null,
  tags: '',
  description: ''
})

// 凭证表单
const credentialForm = reactive({
  name: '',
  type: 'password',
  username: '',
  password: '',
  privateKey: '',
  passphrase: '',
  description: ''
})

// 云平台账号表单
const cloudAccountForm = reactive({
  name: '',
  provider: 'aliyun',
  accessKey: '',
  secretKey: '',
  region: '',
  description: '',
  status: 1
})

// Excel导入表单
const excelImportForm = reactive({
  type: 'self',
  groupId: null as number | null
})
const uploadedFile = ref<any>(null)

// 云主机导入
const cloudImportStep = ref(0)
const cloudImportForm = reactive({
  accountId: null as number | null,
  region: '',
  groupId: null as number | null
})
const cloudHostList = ref<any[]>([])
const selectedCloudHosts = ref<any[]>([])
const selectAllCloudHosts = ref(false)
const cloudRegions = ref<any[]>([])
const loadingCloudRegions = ref(false)

// 主机批量选择
const selectedHosts = ref<any[]>([])

const selectedCloudAccount = ref<any>(null)
const selectedCloudGroup = ref<any>(null)

// 分组表单
const groupForm = reactive({
  id: 0,
  parentId: null,
  name: '',
  code: '',
  description: '',
  sort: 0,
  status: 1
})

// 表单验证规则
const hostRules: Record<string, FieldRule[]> = {
  name: [{ required: true, message: '请输入主机名称' }],
  type: [{ required: true, message: '请选择主机类型' }],
  ip: [{ required: true, message: '请输入IP地址' }],
  sshUser: [{ required: true, message: '请输入SSH用户名' }],
  port: [{ required: true, message: '请输入SSH端口' }]
}

const credentialRules: Record<string, FieldRule[]> = {
  name: [{ required: true, message: '请输入凭证名称' }],
  type: [{ required: true, message: '请选择认证方式' }],
  password: [{ required: true, message: '请输入密码' }],
  privateKey: [{ required: true, message: '请输入私钥' }]
}

const cloudAccountRules: Record<string, FieldRule[]> = {
  name: [{ required: true, message: '请输入账号名称' }],
  provider: [{ required: true, message: '请选择云厂商' }],
  accessKey: [{ required: true, message: '请输入Access Key' }],
  secretKey: [{ required: true, message: '请输入Secret Key' }]
}

const groupRules: Record<string, FieldRule[]> = {
  name: [{ required: true, message: '请输入分组名称' }],
  code: [{ required: true, message: '请输入分组编码' }],
  status: [{ required: true, message: '请选择状态' }]
}

// 分组树选项（用于表单中的选择器）
const groupTreeOptions = computed(() => {
  return buildTreeOptions(groupTree.value)
})

// 构建树形选项
const buildTreeOptions = (nodes: any[]): any[] => {
  return nodes.map(node => ({
    id: node.id,
    name: node.name,
    children: node.children ? buildTreeOptions(node.children) : undefined
  }))
}

// 过滤分组树
const filterGroupTree = () => {
  if (!groupSearchKeyword.value) {
    filteredGroupTree.value = groupTree.value
    return
  }
  filteredGroupTree.value = searchTreeNodes(groupTree.value, groupSearchKeyword.value)
}

// 递归搜索树节点
const searchTreeNodes = (nodes: any[], keyword: string): any[] => {
  const result: any[] = []
  for (const node of nodes) {
    const matchName = node.name?.toLowerCase().includes(keyword.toLowerCase())
    let filteredChildren: any[] = []
    if (node.children && node.children.length > 0) {
      filteredChildren = searchTreeNodes(node.children, keyword)
    }
    if (matchName || filteredChildren.length > 0) {
      result.push({
        ...node,
        children: filteredChildren.length > 0 ? filteredChildren : node.children
      })
    }
  }
  return result
}

// 展开/折叠全部
const toggleExpandAll = () => {
  isExpandAll.value = !isExpandAll.value
  if (groupTreeRef.value) {
    if (isExpandAll.value) {
      groupTreeRef.value.expandAll(true)
    } else {
      groupTreeRef.value.expandAll(false)
    }
  }
}

// 获取所有节点key
const getAllNodeKeys = (nodes: any[]): any[] => {
  const keys: any[] = []
  const traverse = (nodeList: any[]) => {
    nodeList.forEach(node => {
      keys.push(node.id)
      if (node.children && node.children.length > 0) {
        traverse(node.children)
      }
    })
  }
  traverse(nodes)
  return keys
}

// 点击分组节点 (a-tree @select handler)
const handleGroupSelect = (selectedKeys: (string | number)[], data: { node?: any }) => {
  const nodeData = data.node
  if (!nodeData) return
  selectedGroup.value = nodeData

  if (activeView.value === 'terminal') {
    loadTerminalHostList(nodeData.id)
  } else {
    hostPagination.page = 1
    loadHostList()
  }
}

// Legacy handler kept for compatibility
const handleGroupClick = (data: any) => {
  selectedGroup.value = data
  if (activeView.value === 'terminal') {
    loadTerminalHostList(data.id)
  } else {
    hostPagination.page = 1
    loadHostList()
  }
}

// 清除分组选择
const clearGroupSelection = () => {
  selectedGroup.value = null
  hostPagination.page = 1
  loadHostList()
}

// 获取分组路径
const getGroupPath = (group: any): string => {
  return group.name || '未知分组'
}

// 分组操作
const handleGroupAction = (command: string, data: any) => {
  if (command === 'edit') {
    handleEditGroup(data)
  } else if (command === 'delete') {
    handleDeleteGroup(data)
  }
}

// 获取状态颜色
const getStatusColor = (status: number) => {
  switch (status) {
    case 1: return '#67c23a'
    case 0: return '#909399'
    default: return '#c0c4cc'
  }
}

// 获取状态类型 (for a-tag color)
const getStatusType = (status: number) => {
  switch (status) {
    case 1: return 'success'
    case 0: return 'info'
    default: return ''
  }
}

// 获取状态标签颜色 (for a-tag)
const getStatusTagColor = (status: number | undefined) => {
  switch (status) {
    case 1: return 'green'
    case 0: return 'gray'
    default: return 'gray'
  }
}

// 加载分组树
const loadGroupTree = async () => {
  groupLoading.value = true
  try {
    const data = await getGroupTree()
    const enriched = enrichTreeData(data || [])
    groupTree.value = enriched
    filteredGroupTree.value = enriched
  } catch (error) {
    Message.error('获取分组树失败')
  } finally {
    groupLoading.value = false
  }
}

// 递归处理树节点数据，添加 title 字段供 Arco tree 渲染
const enrichTreeData = (nodes: any[]): any[] => {
  return nodes.map(node => ({
    ...node,
    title: node.name,
    children: node.children ? enrichTreeData(node.children) : undefined
  }))
}

// 加载主机列表
const loadHostList = async () => {
  hostLoading.value = true
  try {
    const params: any = {
      page: hostPagination.page,
      pageSize: hostPagination.pageSize,
      keyword: searchForm.keyword || undefined
    }
    if (searchForm.status !== undefined) {
      params.status = searchForm.status
    }
    if (selectedGroup.value && selectedGroup.value.id) {
      params.groupId = selectedGroup.value.id
    }

    const res = await getHostList(params)
    hostList.value = res.list || []
    hostPagination.total = res.total || 0

    // 合并已有的Agent状态
    mergeAgentStatuses()

    if (isAdmin.value) {
      userHasEditPermission.value = true
      if (hostList.value && hostList.value.length > 0) {
        const permissionsMap = new Map<number, number>()
        for (const host of hostList.value) {
          permissionsMap.set(host.id, PERMISSION.ALL)
        }
        hostPermissions.value = permissionsMap
      }
    } else if (hostList.value && hostList.value.length > 0) {
      const permissionsMap = new Map<number, number>()
      let hasEditPerm = false
      for (const host of hostList.value) {
        try {
          const permRes = await getUserHostPermissions(host.id)
          if (permRes && permRes.permissions !== undefined) {
            permissionsMap.set(host.id, permRes.permissions)
            if ((permRes.permissions & PERMISSION.EDIT) > 0) {
              hasEditPerm = true
            }
          }
        } catch (err) {
        }
      }
      hostPermissions.value = permissionsMap
      userHasEditPermission.value = hasEditPerm
    } else {
      userHasEditPermission.value = false
    }
  } catch (error) {
    Message.error('获取主机列表失败')
  } finally {
    hostLoading.value = false
  }
}

// 加载凭证列表
const loadCredentialList = async () => {
  try {
    const data = await getCredentials()
    credentialList.value = data || []
  } catch (error) {
  }
}

// 加载服务标签列表
const loadServiceLabels = async () => {
  try {
    const res = await getServiceLabels({ page: 1, pageSize: 1000 })
    const list = res.list || res.data || []
    serviceLabelOptions.value = list.map((item: any) => item.name).filter(Boolean)
  } catch {}
}

// 主机标签：逗号分隔字符串 <-> 数组
const hostTagsArray = computed({
  get: () => hostForm.tags ? hostForm.tags.split(',').map((t: string) => t.trim()).filter(Boolean) : [],
  set: (val: string[]) => { hostForm.tags = val.join(',') }
})

// 加载云平台账号列表
const loadCloudAccountList = async () => {
  try {
    const data = await getCloudAccounts()
    cloudAccountList.value = data || []
  } catch (error) {
  }
}

// 获取启用的云平台账号列表
const enabledCloudAccounts = computed(() => {
  return cloudAccountList.value.filter((a: any) => a.status === 1)
})

// 终端相关方法
const openTerminalTab = () => {
  const url = window.location.origin + '/terminal'
  window.open(url, '_blank')
}

const handleHostDblClick = async (data: any) => {
  if (data.type === 'host' || data.ip) {
    // Use original host ID for terminal connection
    const hostData = { ...data, id: data.hostId || data.id }
    const dblClickHosts = JSON.parse(sessionStorage.getItem('dblClickHosts') || '[]')
    dblClickHosts.push(hostData)
    sessionStorage.setItem('dblClickHosts', JSON.stringify(dblClickHosts))
    router.push('/terminal')
  }
}

// 主机表格行双击打开终端
let hostnameClickTimer: ReturnType<typeof setTimeout> | null = null
const handleHostnameClick = (record: any) => {
  if (hostnameClickTimer) {
    clearTimeout(hostnameClickTimer)
    hostnameClickTimer = null
    return
  }
  hostnameClickTimer = setTimeout(() => {
    hostnameClickTimer = null
    handleShowHostDetail(record)
  }, 250)
}

const handleTableRowDblClick = (record: any) => {
  if (record && record.id) {
    const dblClickHosts = JSON.parse(sessionStorage.getItem('dblClickHosts') || '[]')
    dblClickHosts.push(record)
    sessionStorage.setItem('dblClickHosts', JSON.stringify(dblClickHosts))
    router.push('/terminal')
  }
}

// 终端视图相关方法
const loadTerminalHostList = async (groupId?: number) => {
  try {
    const params: any = {
      page: 1,
      pageSize: 10000
    }
    if (groupId) {
      params.groupId = groupId
    }
    const res = await getHostList(params)
    terminalHostList.value = res.list || []
  } catch (error) {
    terminalHostList.value = []
  }
}

// 过滤终端主机列表
const filteredTerminalHosts = computed(() => {
  if (!terminalSearchKeyword.value) {
    return terminalHostList.value
  }
  const keyword = terminalSearchKeyword.value.toLowerCase()
  return terminalHostList.value.filter((host: any) => {
    return host.name?.toLowerCase().includes(keyword) ||
           host.ip?.includes(keyword) ||
           host.groupName?.toLowerCase().includes(keyword)
  })
})

// 构建终端视图的分组+主机树
const terminalGroupTree = computed(() => {
  if (!groupTree.value || groupTree.value.length === 0) {
    return []
  }

  const copyTree = (groups: any[]): any[] => {
    return groups.map((group: any) => ({
      ...group,
      id: `group-${group.id}`,
      type: 'group',
      label: group.name,
      children: group.children ? copyTree(group.children) : []
    }))
  }

  const tree = copyTree(groupTree.value)

  const addHostsToGroups = (groups: any[], hosts: any[]) => {
    groups.forEach((group: any) => {
      // Extract original group ID from prefixed ID
      const originalGroupId = typeof group.id === 'string' && group.id.startsWith('group-')
        ? parseInt(group.id.replace('group-', ''))
        : group.id
      const groupHosts = hosts.filter((h: any) => h.groupId === originalGroupId)
      if (groupHosts.length > 0) {
        const hostNodes = groupHosts.map((host: any) => ({
          ...host,
          id: `host-${host.id}`,
          hostId: host.id,
          type: 'host',
          title: host.name,
          label: host.name
        }))
        group.children = [...(group.children || []), ...hostNodes]
      }
      if (group.children && group.children.length > 0) {
        addHostsToGroups(group.children, hosts)
      }
    })
  }

  addHostsToGroups(tree, terminalHostList.value)

  return tree
})

// 初始化终端
const initTerminal = async () => {
  await nextTick()

  if (!terminalRef.value) return

  if (terminal.value) {
    terminal.value.dispose()
  }

  terminal.value = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#cccccc',
      cursor: '#cccccc',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#56b6c2',
      white: '#ffffff',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d3b9d8',
      brightCyan: '#61bfff',
      brightWhite: '#ffffff',
    }
  })

  fitAddon.value = new FitAddon()
  terminal.value.loadAddon(fitAddon.value)
  terminal.value.loadAddon(new WebLinksAddon())

  terminal.value.open(terminalRef.value)

  terminal.value.writeln('\x1b[1;32m欢迎使用 SSH Web 终端\x1b[0m')
  terminal.value.writeln('正在连接...')
}

// 连接SSH
const connectSSH = (host: any) => {
  const token = localStorage.getItem('token') || ''
  const wsUrl = `ws://localhost:9876/api/v1/asset/terminal/${host.id}?token=${token}`

  ws.value = new WebSocket(wsUrl)

  ws.value.onopen = () => {
    if (terminal.value) {
      terminal.value.writeln('\x1b[1;32m连接成功！\x1b[0m')
      terminal.value.writeln(`已连接到: ${host.name} (${host.ip}:${host.port})`)
      terminal.value.writeln('')
    }
  }

  ws.value.onmessage = (event) => {
    if (terminal.value) {
      terminal.value.write(event.data)
    }
  }

  ws.value.onerror = (error) => {
    if (terminal.value) {
      terminal.value.writeln('\x1b[1;31m连接错误\x1b[0m')
    }
  }

  ws.value.onclose = () => {
    if (terminal.value) {
      terminal.value.writeln('\r\n\x1b[1;33m连接已关闭\x1b[0m')
    }
  }
}

const getTerminalUrl = (host: any): string => {
  const token = localStorage.getItem('token') || ''
  return `/api/v1/asset/terminal/${host.id}?token=${token}`
}

const closeTerminal = () => {
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }
  if (terminal.value) {
    terminal.value.dispose()
    terminal.value = null
  }
  activeTerminalHost.value = null
}

const switchToHostsView = async () => {
  activeView.value = 'hosts'
  activeTerminalHost.value = null
}

const handleOpenTerminal = () => {
  const url = window.location.origin + '/terminal'
  window.open(url, '_blank')
}

// 搜索
const handleSearch = () => {
  hostPagination.page = 1
  loadHostList()
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.status = undefined
  clearGroupSelection()
}

// 导入命令处理
const handleImportCommand = (command: string) => {
  if (command === 'direct') {
    handleDirectImport()
  } else if (command === 'excel') {
    handleExcelImport()
  } else if (command === 'cloud') {
    handleCloudImport()
  }
}

// 直接导入
const handleDirectImport = async () => {
  await loadCredentialList()
  loadServiceLabels()

  Object.assign(hostForm, {
    id: 0,
    name: '',
    groupId: selectedGroup.value?.id || null,
    type: 'self',
    sshUser: 'root',
    ip: '',
    port: 22,
    credentialId: null,
    tags: '',
    description: ''
  })
  directImportVisible.value = true
}

// 直接导入关闭
const handleDirectImportClose = () => {
  hostFormRef.value?.resetFields()
}

// 直接导入提交
const handleDirectImportSubmit = async () => {
  if (!hostFormRef.value) return
  const errors = await hostFormRef.value.validate()
  if (errors) return
  hostSubmitting.value = true
  try {
    let hostId = 0
    if (hostForm.id && hostForm.id > 0) {
      await updateHost(hostForm.id, hostForm)
      hostId = hostForm.id
      Message.success('主机更新成功')
    } else {
      const result = await createHost(hostForm)
      hostId = result.id
      Message.success('主机导入成功')
    }

    directImportVisible.value = false
    loadHostList()
    loadGroupTree()

    if (hostForm.credentialId && hostId > 0) {
      setTimeout(async () => {
        try {
          await collectHostInfo(hostId)
          Message.success('主机信息采集成功')
          loadHostList()
        } catch (error: any) {
        }
      }, 500)
    }
  } catch (error: any) {
    Message.error(error.message || '操作失败')
  } finally {
    hostSubmitting.value = false
  }
}

// Excel导入
const handleExcelImport = () => {
  Object.assign(excelImportForm, {
    groupId: selectedGroup.value?.id || null
  })
  uploadedFile.value = null
  excelImportVisible.value = true
}

// 下载模板
const downloadTemplate = async () => {
  try {
    const blob = await downloadExcelTemplate()
    const url = window.URL.createObjectURL(new Blob([blob], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }))
    const link = document.createElement('a')
    link.href = url
    link.download = 'host_import_template.xlsx'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    Message.success('模板下载成功')
  } catch (error) {
    Message.error('模板下载失败')
  }
}

// 文件变化 (Arco upload onChange)
const handleFileChange = (fileList: any[], fileItem: any) => {
  if (fileList.length > 0) {
    uploadedFile.value = fileItem.file
  } else {
    uploadedFile.value = null
  }
}

// Excel导入关闭
const handleExcelImportClose = () => {
  uploadedFile.value = null
}

// Excel导入提交
const handleExcelImportSubmit = async () => {
  if (!uploadedFile.value) {
    Message.warning('请先上传Excel文件')
    return
  }
  try {
    excelImporting.value = true
    const file = uploadedFile.value
    const result = await importFromExcel(file, excelImportForm.type, excelImportForm.groupId || undefined)

    if (result.successCount > 0) {
      Message.success(`成功导入 ${result.successCount} 台主机`)
      await loadHostList()
      loadGroupTree()

      await new Promise(resolve => setTimeout(resolve, 500))
      const newHosts = hostList.value.filter((h: any) => h.status === -1)
      if (newHosts.length > 0) {
        Message.info('正在自动采集主机信息...')
        const hostIds = newHosts.map((h: any) => h.id)
        try {
          await batchCollectHostInfo({ hostIds })
          await loadHostList()
          Message.success(`成功采集 ${newHosts.length} 台主机信息`)
        } catch (error) {
        }
      }
    }
    if (result.failedCount > 0) {
      Message.warning(`${result.failedCount} 台主机导入失败`)
      if (result.errors && result.errors.length > 0) {
        Modal.warning({
          title: '导入详情',
          content: result.errors.join('\n'),
        })
      }
    }

    excelImportVisible.value = false
    uploadedFile.value = null
  } catch (error: any) {
    Message.error(error.message || '导入失败')
  } finally {
    excelImporting.value = false
  }
}

// 云主机导入
const handleCloudImport = () => {
  cloudImportStep.value = 0
  cloudHostList.value = []
  selectedCloudHosts.value = []
  cloudRegions.value = []
  Object.assign(cloudImportForm, {
    accountId: null,
    region: '',
    groupId: selectedGroup.value?.id || null
  })
  cloudImportVisible.value = true
}

// 获取云主机列表
const handleGetCloudHosts = async () => {
  if (!cloudImportForm.accountId) {
    Message.warning('请选择云平台账号')
    return
  }
  if (!cloudImportForm.region) {
    Message.warning('请选择区域')
    return
  }

  loadingCloudHosts.value = true
  try {
    const res = await getCloudInstances(cloudImportForm.accountId!, cloudImportForm.region)
    cloudHostList.value = Array.isArray(res) ? res : []
    selectedCloudAccount.value = cloudAccountList.value.find(a => a.id === cloudImportForm.accountId)
    selectedCloudGroup.value = groupTree.value.find((g: any) => g.id === cloudImportForm.groupId)
    cloudImportStep.value = 1
  } catch (error: any) {
    Message.error(error.message || '获取云主机列表失败')
  } finally {
    loadingCloudHosts.value = false
  }
}

// 云主机选择变化
const handleCloudHostSelectionChange = (selection: any[]) => {
  selectedCloudHosts.value = selection
}

// 主机选择变化
const handleHostSelectionChange = (selection: any[]) => {
  selectedHosts.value = selection
}

// 批量删除主机
const handleBatchDelete = async () => {
  if (selectedHosts.value.length === 0) {
    Message.warning('请先选择要删除的主机')
    return
  }

  Modal.warning({
    title: '批量删除确认',
    content: `确定要删除选中的 ${selectedHosts.value.length} 台主机吗？`,
    hideCancel: false,
    onOk: async () => {
      try {
        const hostIds = selectedHosts.value.map((h: any) => h.id)
        await batchDeleteHosts(hostIds)
        Message.success('批量删除成功')
        selectedHosts.value = []
        loadHostList()
        loadGroupTree()
      } catch (error: any) {
        Message.error(error.message || '批量删除失败')
      }
    }
  })
}

// 部署Agent到单台主机
const handleDeployAgent = async (record: any) => {
  agentDeployMode.value = 'deploy'
  agentDeployTarget.value = record
  agentDeployServerAddr.value = ''
  showAgentDeployDialog.value = true
}

// 更新Agent
const handleUpdateAgent = async (record: any) => {
  agentDeployMode.value = 'update'
  agentDeployTarget.value = record
  agentDeployServerAddr.value = ''
  showAgentDeployDialog.value = true
}

// 确认Agent部署/更新
const confirmAgentDeploy = async () => {
  const addr = agentDeployServerAddr.value.trim() || undefined
  showAgentDeployDialog.value = false
  if (agentDeployMode.value === 'batch') {
    try {
      const hostIds = selectedHosts.value.map((h: any) => h.id)
      await batchDeployAgent(hostIds, addr)
      Message.success('批量部署完成')
      selectedHosts.value = []
      loadHostList()
      loadAgentStatuses()
    } catch (error: any) {
      Message.error(error.message || '批量部署失败')
    }
  } else if (agentDeployMode.value === 'update') {
    const record = agentDeployTarget.value
    try {
      record._updating = true
      await updateAgent(record.id, addr)
      Message.success('Agent更新成功')
      loadHostList()
      loadAgentStatuses()
    } catch (error: any) {
      Message.error(error.message || 'Agent更新失败')
    } finally {
      record._updating = false
    }
  } else {
    const record = agentDeployTarget.value
    try {
      record._deploying = true
      await deployAgent(record.id, addr)
      Message.success('Agent部署成功')
      loadHostList()
      loadAgentStatuses()
    } catch (error: any) {
      Message.error(error.message || 'Agent部署失败')
    } finally {
      record._deploying = false
    }
  }
}

// 卸载Agent
const handleUninstallAgent = async (record: any) => {
  Modal.warning({
    title: '卸载Agent',
    content: `确定要卸载主机 ${record.name} (${record.ip}) 上的Agent吗？卸载后将回退为SSH连接方式。`,
    hideCancel: false,
    onOk: async () => {
      try {
        record._uninstalling = true
        await uninstallAgent(record.id)
        Message.success('Agent卸载成功')
        loadHostList()
        loadAgentStatuses()
      } catch (error: any) {
        Message.error(error.message || 'Agent卸载失败')
      } finally {
        record._uninstalling = false
      }
    }
  })
}

// 批量部署Agent
const handleBatchDeployAgent = async () => {
  if (selectedHosts.value.length === 0) {
    Message.warning('请先选择要部署Agent的主机')
    return
  }
  agentDeployMode.value = 'batch'
  agentDeployTarget.value = null
  agentDeployServerAddr.value = ''
  showAgentDeployDialog.value = true
}

// Agent部署对话框状态
const showAgentDeployDialog = ref(false)
const agentDeployMode = ref<'deploy' | 'update' | 'batch'>('deploy')
const agentDeployTarget = ref<any>(null)
const agentDeployServerAddr = ref('')

// 加载Agent状态并合并到主机列表
const agentStatusMap = ref<Record<number, any>>({})
const loadAgentStatuses = async () => {
  try {
    const data = await getAgentStatuses()
    const map: Record<number, any> = {}
    const list = Array.isArray(data) ? data : []
    for (const item of list) {
      map[item.hostId] = item
    }
    agentStatusMap.value = map
    // 合并Agent状态到主机列表
    mergeAgentStatuses()
  } catch {
    // 静默失败
  }
}

// 将agentStatusMap合并到hostList中
const mergeAgentStatuses = () => {
  if (!hostList.value || hostList.value.length === 0) return
  for (const host of hostList.value) {
    const agentInfo = agentStatusMap.value[(host as any).id]
    if (agentInfo) {
      ;(host as any).agentStatus = agentInfo.status
      ;(host as any).connectionMode = 'agent'
    }
  }
}

// 全选云主机
const handleSelectAllCloudHosts = (checked: boolean) => {
  // TODO: 实现全选逻辑
}

// 云主机导入关闭
const handleCloudImportClose = () => {
  cloudImportStep.value = 0
  cloudHostList.value = []
  selectedCloudHosts.value = []
}

// 云主机导入提交
const handleCloudImportSubmit = async () => {
  cloudImporting.value = true
  try {
    const data = {
      accountId: cloudImportForm.accountId,
      region: cloudImportForm.region,
      groupId: cloudImportForm.groupId,
      instanceIds: selectedCloudHosts.value.map(h => h.instanceId)
    }
    await importFromCloud(data)
    Message.success('云主机导入成功')
    cloudImportVisible.value = false
    loadHostList()
    loadGroupTree()
  } catch (error: any) {
    Message.error(error.message || '导入失败')
  } finally {
    cloudImporting.value = false
  }
}

// 认证方式变化
const handleAuthTypeChange = (type: string) => {
  if (type === 'password') {
    credentialForm.privateKey = ''
    credentialForm.passphrase = ''
  } else {
    credentialForm.password = ''
  }
}

// 凭证对话框关闭
const handleCredentialDialogClose = () => {
  credentialFormRef.value?.resetFields()
}

// 提交凭证表单
const handleCredentialSubmit = async () => {
  if (!credentialFormRef.value) return
  const errors = await credentialFormRef.value.validate()
  if (errors) return
  credentialSubmitting.value = true
  try {
    await createCredential(credentialForm)
    Message.success('凭证创建成功')
    showCredentialDialog.value = false
    loadCredentialList()
  } catch (error: any) {
    Message.error(error.message || '创建失败')
  } finally {
    credentialSubmitting.value = false
  }
}

// 云平台账号对话框关闭
const handleCloudAccountDialogClose = () => {
  cloudAccountFormRef.value?.resetFields()
}

// 提交云平台账号表单
const handleCloudAccountSubmit = async () => {
  if (!cloudAccountFormRef.value) return
  const errors = await cloudAccountFormRef.value.validate()
  if (errors) return
  cloudAccountSubmitting.value = true
  try {
    await createCloudAccount(cloudAccountForm)
    Message.success('云平台账号添加成功')
    showCloudAccountDialog.value = false
    loadCloudAccountList()
  } catch (error: any) {
    Message.error(error.message || '添加失败')
  } finally {
    cloudAccountSubmitting.value = false
  }
}

// 编辑主机
const handleEditHost = async (row: any) => {
  await loadCredentialList()
  loadServiceLabels()

  Object.assign(hostForm, {
    id: row.id,
    name: row.name,
    groupId: row.groupId,
    type: row.type || 'self',
    cloudProvider: row.cloudProvider || '',
    cloudInstanceId: row.cloudInstanceId || '',
    cloudAccountId: row.cloudAccountId || null,
    sshUser: row.sshUser,
    ip: row.ip,
    port: row.port,
    credentialId: row.credentialId,
    tags: Array.isArray(row.tags) ? row.tags.join(',') : row.tags,
    description: row.description
  })
  directImportVisible.value = true
}

// 文件管理
const handleFileManager = (row: any) => {
  selectedHostId.value = row.id
  selectedHostName.value = row.name
  selectedHostConnectionMode.value = row.connectionMode || 'ssh'
  selectedHostAgentStatus.value = row.agentStatus || ''
  fileBrowserVisible.value = true
}

// 显示主机详情
const handleShowHostDetail = async (row: any) => {
  try {
    hostDetailLoading.value = true
    showHostDetailDialog.value = true
    const data = await getHost(row.id)
    hostDetail.value = data
  } catch (error: any) {
    Message.error(error.message || '获取主机详情失败')
  } finally {
    hostDetailLoading.value = false
  }
}

// 关闭主机详情
const handleCloseHostDetail = () => {
  showHostDetailDialog.value = false
  hostDetail.value = null
}

// 从详情页采集主机信息
const handleCollectHostFromDetail = async () => {
  if (!hostDetail.value) return
  try {
    hostDetailLoading.value = true
    await collectHostInfo(hostDetail.value.id)
    Message.success('采集成功')
    const data = await getHost(hostDetail.value.id)
    hostDetail.value = data
    loadHostList()
  } catch (error: any) {
    Message.error(error.message || '采集失败')
  } finally {
    hostDetailLoading.value = false
  }
}

// 删除主机
const handleDeleteHost = (row: any) => {
  Modal.warning({
    title: '提示',
    content: `确定要删除主机"${row.name}"吗？`,
    hideCancel: false,
    onOk: async () => {
      try {
        await deleteHost(row.id)
        Message.success('删除成功')
        loadHostList()
        loadGroupTree()
      } catch (error: any) {
        Message.error(error.message || '删除失败')
      }
    }
  })
}

// 新增分组
const handleAddGroup = () => {
  Object.assign(groupForm, {
    id: 0,
    parentId: null,
    name: '',
    code: '',
    description: '',
    sort: 0,
    status: 1
  })
  groupDialogTitle.value = '新增分组'
  isGroupEdit.value = false
  groupDialogVisible.value = true
}

// 编辑分组
const handleEditGroup = (data: any) => {
  Object.assign(groupForm, {
    id: data.id,
    parentId: data.parentId || null,
    name: data.name,
    code: data.code || '',
    description: data.description || '',
    sort: data.sort || 0,
    status: data.status
  })
  groupDialogTitle.value = '编辑分组'
  isGroupEdit.value = true
  groupDialogVisible.value = true
}

// 删除分组
const handleDeleteGroup = (data: any) => {
  const hasChildren = data.children && data.children.length > 0
  const confirmMsg = hasChildren
    ? `该分组下有 ${data.children.length} 个子分组，确定要删除吗？`
    : `确定要删除分组"${data.name}"吗？`

  Modal.warning({
    title: '提示',
    content: confirmMsg,
    hideCancel: false,
    onOk: async () => {
      try {
        await deleteGroup(data.id)
        Message.success('删除成功')
        loadGroupTree()
        if (selectedGroup.value?.id === data.id) {
          clearGroupSelection()
        }
      } catch (error: any) {
        Message.error(error.message || '删除失败')
      }
    }
  })
}

// 提交分组表单
const handleGroupSubmit = async () => {
  if (!groupFormRef.value) return
  const errors = await groupFormRef.value.validate()
  if (errors) return
  groupSubmitting.value = true
  try {
    const data = { ...groupForm }
    if (isGroupEdit.value) {
      await updateGroup(data.id, data)
    } else {
      await createGroup(data)
    }
    Message.success(isGroupEdit.value ? '更新成功' : '创建成功')
    groupDialogVisible.value = false
    loadGroupTree()
  } catch (error: any) {
    Message.error(error.message || (isGroupEdit.value ? '更新失败' : '创建失败'))
  } finally {
    groupSubmitting.value = false
  }
}

// 分组对话框关闭
const handleGroupDialogClose = () => {
  groupFormRef.value?.resetFields()
}

// 检查用户对主机的权限
const hasHostPermission = (hostId: number, permission: number): boolean => {
  const userPermissions = hostPermissions.value.get(hostId) || 0
  return hasPermission(userPermissions, permission)
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatBytesCompact = (bytes: number): string => {
  if (bytes === 0) return '0B'
  const k = 1024
  const sizes = ['B', 'K', 'M', 'G', 'T']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const value = bytes / Math.pow(k, i)
  if (value >= 100) {
    return Math.round(value) + sizes[i]
  }
  return parseFloat(value.toFixed(1)) + sizes[i]
}

// 获取使用率颜色
const getUsageColor = (usage: number): string => {
  if (usage >= 90) return '#f56c6c'
  if (usage >= 70) return '#e6a23c'
  return '#67c23a'
}

// 获取使用率等级
const getUsageLevel = (usage: number | undefined): string => {
  if (!usage) return 'low'
  if (usage >= 90) return 'critical'
  if (usage >= 70) return 'high'
  return 'low'
}

// 采集主机信息
const handleCollectHost = async (row: any) => {
  try {
    await collectHostInfo(row.id)
    Message.success('采集成功')
    loadHostList()
  } catch (error: any) {
    Message.error(error.message || '采集失败')
  }
}

// 监听activeTerminalHost变化，自动连接终端
watch(activeTerminalHost, async (newHost) => {
  if (newHost) {
    await initTerminal()
    await nextTick()
    connectSSH(newHost)
  } else {
    closeTerminal()
  }
})

// 监听云平台账号变化，加载区域列表
watch(() => cloudImportForm.accountId, async (accountId) => {
  if (accountId) {
    cloudRegions.value = []
    cloudImportForm.region = ''
    loadingCloudRegions.value = true
    try {
      const res = await getCloudRegions(accountId)
      cloudRegions.value = Array.isArray(res) ? res : []

      const account = cloudAccountList.value.find((a: any) => a.id === accountId)
      if (account?.region && cloudRegions.value.some((r: any) => r.value === account.region)) {
        cloudImportForm.region = account.region
      }
    } catch (error: any) {
      Message.error(error.message || '加载区域列表失败')
    } finally {
      loadingCloudRegions.value = false
    }
  } else {
    cloudRegions.value = []
    cloudImportForm.region = ''
  }
})

// 生成Agent安装包
const handleGenerateInstallPackage = async () => {
  if (!installPackageServerAddr.value) {
    Message.warning('请输入服务端地址')
    return
  }
  installPackageLoading.value = true
  installPackageResult.value = null
  try {
    const res: any = await generateInstallPackage(installPackageServerAddr.value)
    installPackageResult.value = res
    Message.success('安装包生成成功')
  } catch (e: any) {
    Message.error(e.message || '生成失败')
  } finally {
    installPackageLoading.value = false
  }
}

// 组件销毁时清理资源
onBeforeUnmount(() => {
  closeTerminal()
})

onMounted(async () => {
  // 确保用户信息已加载，否则 isAdmin 判断会失败导致操作按钮不显示
  if (!userStore.userInfo) {
    try {
      await userStore.getProfile()
    } catch (e) {
      // ignore - 用户可能未登录
    }
  }
  loadGroupTree()
  loadHostList()
  loadCredentialList()
  loadServiceLabels()
  loadCloudAccountList()
  loadAgentStatuses()
})
</script>

<style scoped>
.hosts-page-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  flex-shrink: 0;
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-title-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--ops-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: var(--ops-text-primary);
}

.page-subtitle {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary);
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.import-dropdown {
  display: inline-block;
}

/* 主内容区域 */
.main-content {
  display: flex;
  gap: 12px;
  flex: 1;
  min-height: 0;
}

/* 左侧分组面板 */
.left-panel {
  width: 280px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.panel-header {
  padding: 16px;
  border-bottom: 1px solid var(--ops-border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
  color: var(--ops-text-primary);
}

.panel-icon {
  font-size: 18px;
  color: var(--ops-primary);
}

.panel-actions {
  display: flex;
  gap: 8px;
}

.panel-body {
  flex: 1;
  padding: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.group-search {
  margin-bottom: 12px;
}

.tree-container {
  flex: 1;
  overflow-y: auto;
}

.group-tree {
  background: transparent;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  width: 0;
}

.node-icon {
  flex-shrink: 0;
}

.node-label {
  flex: 1;

  white-space: nowrap;
}

.node-count {
  font-size: 12px;
  color: var(--ops-text-tertiary);
  flex-shrink: 0;
}

.node-ip {
  font-size: 11px;
  color: var(--ops-text-tertiary);
  flex-shrink: 0;
}

.node-actions {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.2s;
}

.tree-node:hover .node-actions {
  opacity: 1;
}

.more-icon {
  font-size: 14px;
  cursor: pointer;
  color: var(--ops-text-tertiary);
}

.more-icon:hover {
  color: var(--ops-primary);
}

/* 右侧主机列表面板 */
.right-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.filter-bar {
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px 8px 0 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  border-bottom: 1px solid var(--ops-border-color);
}

.filter-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.filter-input {
  width: 220px;
}

.filter-actions {
  display: flex;
  gap: 10px;
}

.search-icon {
  color: var(--ops-text-tertiary);
}

.reset-btn {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
}

.selected-group-bar {
  padding: 10px 16px;
  background: #f0f9ff;
  border-bottom: 1px solid var(--ops-border-color);
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.group-path {
  color: var(--ops-primary);
  font-weight: 500;
}

.table-wrapper {
  flex: 1;
  background: #fff;
  border-radius: 0 0 8px 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modern-table {
  flex: 1;
}

.hostname-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}

/* 主机头像/状态图标 */
.host-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: #f0f2f5;
  color: #909399;
  font-size: 18px;
}

.host-avatar.host-status-1 {
  background: #e7f8e8;
  color: #67c23a;
}

.host-avatar.host-status-0 {
  background: #fef0f0;
  color: #f56c6c;
}

.host-avatar.host-status--1 {
  background: #f4f4f5;
  color: #909399;
}

/* 主机信息 */
.host-info {
  flex: 1;
  min-width: 0;
}

.hostname {
  font-weight: 500;
  color: var(--ops-text-primary);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host-meta {
  font-size: 12px;
  color: var(--ops-text-tertiary);
  display: flex;
  align-items: center;
  gap: 2px;
}

.ip {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.port {
  flex-shrink: 0;
}

/* 状态单元格 */
.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot-1 {
  background: #67c23a;
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.2);
}

.status-dot-0 {
  background: #f56c6c;
  box-shadow: 0 0 0 2px rgba(245, 108, 108, 0.2);
}

.status-dot--1 {
  background: #909399;
  box-shadow: 0 0 0 2px rgba(144, 148, 153, 0.2);
}

.status-text {
  font-size: 13px;
  font-weight: 500;
}

.status-text-1 {
  color: #67c23a;
}

.status-text-0 {
  color: #f56c6c;
}

.status-text--1 {
  color: #909399;
}

.text-muted {
  color: #c0c4cc;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: var(--ops-primary);
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.action-refresh:hover {
  background-color: #e8f4ff;
  color: var(--ops-primary);
}

.action-files:hover {
  background-color: #fdf6ec;
  color: #e6a23c;
}

/* 资源显示单元格样式 */
.resource-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 40px;
}

.resource-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 0 4px;
}

.resource-label {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
  white-space: nowrap;
}

.resource-compact {
  font-size: 11px;
  white-space: nowrap;
}

/* 配置单元格样式 */
.config-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 0 8px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.config-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 标签单元格样式 */
.tags-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  justify-content: flex-start;
}

.tags-cell .tag-item {
  font-size: 11px;
}

.tags-cell .tag-more {
  font-size: 11px;
}

/* 主机名点击样式 */
.hostname-clickable {
  cursor: pointer;
  transition: color 0.2s ease;
}

.hostname-clickable:hover {
  color: var(--ops-primary);
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* Upload drag area */
.upload-drag-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 30px 20px;
  border: 1px dashed var(--ops-border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.2s;
}

.upload-drag-area:hover {
  border-color: var(--ops-primary);
}

/* Excel导入样式 */
.excel-import-content {
  padding: 10px 0;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 6px;
}

/* 云主机导入样式 */
.cloud-step-content {
  padding: 20px 0;
  min-height: 300px;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 6px;
}

.import-summary {
  width: 100%;
  max-width: 600px;
  margin: 0 auto;
}

.host-list-preview {
  margin-top: 20px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 6px;
}

.host-list-preview h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #606266;
}

/* 视图容器 */
.view-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* 终端视图 */
.terminal-view {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.terminal-view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #1a1a1a;
  border-bottom: 1px solid #333;
}

.terminal-view-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.terminal-current-group {
  margin-left: 8px;
  font-size: 14px;
  color: #858585;
  font-weight: normal;
}

.terminal-content {
  display: flex;
  flex: 1;
  overflow: hidden;
  background: #1e1e1e;
}

/* 终端侧边栏 */
.terminal-sidebar {
  width: 280px;
  min-width: 280px;
  background: #252526;
  border-right: 1px solid #3e3e42;
  display: flex;
  flex-direction: column;
}

/* 终端主区域 */
.terminal-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  overflow: hidden;
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #252526;
  border-bottom: 1px solid #3e3e42;
  flex-shrink: 0;
}

.terminal-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.terminal-icon {
  font-size: 20px;
  color: #4ec9b0;
}

.terminal-details {
  display: flex;
  flex-direction: column;
}

.terminal-title {
  font-size: 14px;
  font-weight: 600;
  color: #cccccc;
}

.terminal-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #858585;
  margin-top: 2px;
}

.terminal-ip {
  color: #9cdcfe;
}

.terminal-user {
  color: #dcdcaa;
}

.terminal-actions {
  display: flex;
  gap: 8px;
}

.terminal-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #858585;
}

.placeholder-icon {
  font-size: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.placeholder-text {
  font-size: 16px;
  margin-bottom: 8px;
}

.placeholder-hint {
  font-size: 13px;
  color: #858585;
}

.terminal-body {
  flex: 1;
  overflow: hidden;
  background: #1e1e1e;
}

.terminal-wrapper {
  width: 100%;
  height: 100%;
  background: #1e1e1e;
  padding: 10px;
}

.xterm-container {
  width: 100%;
  height: 100%;
}

.xterm-container :deep(.xterm) {
  padding: 10px;
}

.xterm-container :deep(.xterm .xterm-viewport) {
  background-color: #1e1e1e !important;
}

.xterm-container :deep(.xterm .xterm-screen) {
  padding: 0;
}

/* 主机详情弹窗样式 */
.detail-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0;
}

.detail-header-left {
  display: flex;
  align-items: center;
  gap: 20px;
  flex: 1;
}

.host-avatar-lg {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 26px;
  flex-shrink: 0;
}

.host-avatar-lg.host-status-1 {
  background: linear-gradient(135deg, #67c23a 0%, #85ce61 100%);
  color: #fff;
}

.host-avatar-lg.host-status-0 {
  background: linear-gradient(135deg, #f56c6c 0%, #f78989 100%);
  color: #fff;
}

.host-avatar-lg.host-status--1 {
  background: linear-gradient(135deg, #909399 0%, #a6a9ad 100%);
  color: #fff;
}

.detail-header-info {
  flex: 1;
}

.detail-hostname {
  font-size: 22px;
  font-weight: 600;
  color: var(--ops-text-primary);
  margin-bottom: 6px;
}

.detail-hostmeta {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.detail-ip {
  font-size: 14px;
  color: var(--ops-text-secondary);
  font-family: 'Monaco', 'Menlo', monospace;
}

.host-detail-content {
  padding: 4px 0;
}

/* 资源概览行 */
.detail-resource-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
  margin-bottom: 24px;
}

.detail-resource-item {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--ops-bg-secondary, #f7f8fa);
  border-radius: 10px;
  padding: 14px 16px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
}

.detail-resource-icon {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  color: #fff;
  flex-shrink: 0;
}

.cpu-bg { background: linear-gradient(135deg, #165dff, #4080ff); }
.mem-bg { background: linear-gradient(135deg, #00b42a, #23c343); }
.disk-bg { background: linear-gradient(135deg, #ff7d00, #ff9a2e); }

.detail-resource-info {
  flex-shrink: 0;
  min-width: 56px;
}

.detail-resource-label {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  line-height: 1;
  margin-bottom: 4px;
}

.detail-resource-val {
  font-size: 16px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.2;
}

.detail-resource-bar {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.detail-resource-pct {
  font-size: 13px;
  font-weight: 600;
  min-width: 40px;
  text-align: right;
  flex-shrink: 0;
}

.detail-resource-pct.low { color: #00b42a; }
.detail-resource-pct.high { color: #ff7d00; }
.detail-resource-pct.critical { color: #f53f3f; }

/* 平铺信息区 */
.detail-flat-section {
  margin-bottom: 20px;
}

.detail-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  margin-bottom: 12px;
  padding-left: 10px;
  border-left: 3px solid var(--ops-primary, #165dff);
  line-height: 1;
}

.detail-flat-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0;
  background: var(--ops-bg-secondary, #f7f8fa);
  border-radius: 8px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  overflow: hidden;
}

.detail-flat-item {
  padding: 12px 16px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
  border-right: 1px solid var(--ops-border-color, #e5e6eb);
}

.detail-flat-item:nth-child(3n) {
  border-right: none;
}

.detail-flat-item-wide {
  grid-column: span 2;
}

.detail-flat-label {
  display: block;
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  margin-bottom: 4px;
  line-height: 1;
}

.detail-flat-value {
  font-size: 14px;
  color: var(--ops-text-primary, #1d2129);
  font-weight: 500;
  word-break: break-all;
  line-height: 1.4;
}

/* 标签行 */
.detail-tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* 备注 */
.detail-remark-text {
  background: var(--ops-bg-secondary, #f7f8fa);
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 8px;
  padding: 12px 16px;
  font-size: 14px;
  color: var(--ops-text-secondary, #4e5969);
  line-height: 1.6;
  white-space: pre-wrap;
}
</style>

