<template>
  <div class="inspection-management-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-check-circle /></div>
        <div>
          <h2 class="page-title">巡检管理</h2>
          <p class="page-subtitle">管理巡检组和巡检项，支持命令、脚本、PromQL 三种执行方式</p>
        </div>
      </div>
    </div>

    <!-- 搜索与操作 -->
    <div class="filter-bar">
      <a-input v-model="searchForm.keyword" placeholder="搜索巡检组名称" allow-clear style="width: 220px;" @press-enter="loadData" />
      <a-select v-model="searchForm.status" placeholder="状态" allow-clear style="width: 120px;">
        <a-option label="启用" value="enabled" />
        <a-option label="禁用" value="disabled" />
      </a-select>
      <a-button type="primary" @click="loadData"><template #icon><icon-search /></template>搜索</a-button>
      <a-button @click="handleReset"><template #icon><icon-refresh /></template>重置</a-button>
      <div style="flex: 1;" />
      <a-button v-if="selectedGroupIds.length > 0" status="danger" @click="handleBatchDelete">
        <template #icon><icon-delete /></template>批量删除 ({{ selectedGroupIds.length }})
      </a-button>
      <a-button @click="handleExportAll">
        <template #icon><icon-download /></template>导出全部
      </a-button>
      <a-button @click="handleImport">
        <template #icon><icon-upload /></template>导入配置
      </a-button>
      <a-button v-permission="'inspection_groups:create'" type="primary" @click="handleCreate">
        <template #icon><icon-plus /></template>新增巡检组
      </a-button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card"><div class="stat-value">{{ stats.total }}</div><div class="stat-label">总数</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-success);">{{ stats.enabled }}</div><div class="stat-label">启用</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-danger);">{{ stats.disabled }}</div><div class="stat-label">禁用</div></div>
      <div class="stat-card"><div class="stat-value" style="color: var(--ops-primary);">{{ stats.items }}</div><div class="stat-label">巡检项</div></div>
    </div>

    <!-- 数据表格 -->
    <a-table
      :data="tableData"
      :loading="loading"
      :bordered="{ cell: true }"
      stripe
      row-key="id"
      :row-selection="{
        type: 'checkbox',
        showCheckedAll: true
      }"
      v-model:selected-keys="selectedGroupIds"
      :pagination="{
        current: pagination.page,
        pageSize: pagination.pageSize,
        total: pagination.total,
        showTotal: true,
        showPageSize: true,
        pageSizeOptions: [10, 20, 50]
      }"
      @page-change="(p: number) => { pagination.page = p; loadData() }"
      @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }"
    >
      <template #columns>
        <a-table-column title="ID" data-index="id" :width="80" />
        <a-table-column title="巡检组名称" data-index="name" :width="180" />
        <a-table-column title="描述" data-index="description" ellipsis tooltip />
        <a-table-column title="巡检项" :width="280">
          <template #cell="{ record }">
            <div v-if="record.itemNames && record.itemNames.length > 0" class="inspection-items-cell">
              <template v-if="record.itemNames.length <= 3">
                <a-tag v-for="(name, idx) in record.itemNames" :key="idx" size="small" color="arcoblue" style="margin: 2px;">
                  {{ name }}
                </a-tag>
              </template>
              <template v-else>
                <a-tag v-for="(name, idx) in record.itemNames.slice(0, 3)" :key="idx" size="small" color="arcoblue" style="margin: 2px;">
                  {{ name }}
                </a-tag>
                <a-popover position="bottom" trigger="click">
                  <a-tag size="small" color="gray" style="margin: 2px; cursor: pointer;">
                    +{{ record.itemNames.length - 3 }}
                  </a-tag>
                  <template #content>
                    <div style="max-width: 400px; max-height: 300px; overflow-y: auto;">
                      <a-tag v-for="(name, idx) in record.itemNames" :key="idx" size="small" color="arcoblue" style="margin: 4px;">
                        {{ name }}
                      </a-tag>
                    </div>
                  </template>
                </a-popover>
              </template>
            </div>
            <span v-else style="color: var(--ops-text-tertiary);">暂无巡检项</span>
          </template>
        </a-table-column>
        <a-table-column title="执行方式" :width="120" align="center">
          <template #cell="{ record }">
            <a-tag size="small">{{ getExecutionModeText(record.executionMode) }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="执行策略" :width="120" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.executionStrategy === 'concurrent' ? 'arcoblue' : 'orange'">
              {{ record.executionStrategy === 'concurrent' ? '并发' : '顺序' }}
              <span v-if="record.executionStrategy === 'concurrent' && record.concurrency">({{ record.concurrency }})</span>
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="巡检项数" :width="100" align="center">
          <template #cell="{ record }">
            <a-tag>{{ record.itemCount || 0 }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="状态" :width="80" align="center">
          <template #cell="{ record }">
            <a-tag size="small" :color="record.status === 'enabled' ? 'green' : 'red'">
              {{ record.status === 'enabled' ? '启用' : '禁用' }}
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="创建时间" data-index="createdAt" :width="180" />
        <a-table-column title="操作" :width="300" fixed="right" align="center">
          <template #cell="{ record }">
            <a-space :size="8">
              <a-button v-permission="'inspection_items:test'" type="text" size="small" @click="handleTestRun(record)">
                <template #icon><icon-play-arrow /></template>
                测试
              </a-button>
              <a-button v-permission="'inspection_groups:update'" type="text" size="small" @click="handleEdit(record)">
                <template #icon><icon-edit /></template>
                编辑
              </a-button>
              <a-dropdown trigger="hover">
                <a-button type="text" size="small">
                  <template #icon><icon-more /></template>
                  更多
                </a-button>
                <template #content>
                  <a-doption v-permission="'inspection_groups:create'" @click="handleCopy(record)">
                    <template #icon><icon-copy /></template>
                    复制
                  </a-doption>
                  <a-doption @click="handleExport(record)">
                    <template #icon><icon-download /></template>
                    导出
                  </a-doption>
                  <a-doption v-permission="'inspection_groups:delete'" @click="handleDelete(record)">
                    <template #icon><icon-delete /></template>
                    <span style="color: var(--color-danger);">删除</span>
                  </a-doption>
                </template>
              </a-dropdown>
            </a-space>
          </template>
        </a-table-column>
      </template>
    </a-table>

    <!-- 新建/编辑对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="isEdit ? '编辑巡检组' : '新增巡检组'"
      :width="900"
      :ok-loading="loading"
      unmount-on-close
      @ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="horizontal" auto-label-width>
        <a-form-item label="巡检组名称" field="name">
          <a-input v-model="formData.name" placeholder="请输入巡检组名称" />
        </a-form-item>

        <a-form-item label="描述" field="description">
          <a-textarea v-model="formData.description" placeholder="请输入描述" :rows="2" />
        </a-form-item>

        <a-form-item label="执行方式" field="executionMode">
          <a-radio-group v-model="formData.executionMode" type="button">
            <a-radio value="auto">自动（优先Agent）</a-radio>
            <a-radio value="agent">仅Agent</a-radio>
            <a-radio value="ssh">仅SSH</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item label="业务分组" field="groupIds">
          <a-select
            v-model="formData.groupIds"
            placeholder="请选择业务分组（可多选）"
            multiple
            allow-search
            :max-tag-count="3"
          >
            <a-option v-for="group in assetGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </a-option>
          </a-select>
        </a-form-item>

        <a-form-item label="Prometheus地址" field="prometheusUrl">
          <a-input v-model="formData.prometheusUrl" placeholder="http://prometheus:9090（可选）" />
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="用户名" field="prometheusUsername">
              <a-input v-model="formData.prometheusUsername" placeholder="可选" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="密码" field="prometheusPassword">
              <a-input-password v-model="formData.prometheusPassword" placeholder="可选" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="状态" field="status">
          <a-radio-group v-model="formData.status">
            <a-radio value="enabled">启用</a-radio>
            <a-radio value="disabled">禁用</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-divider>自定义变量</a-divider>

        <a-form-item label="自定义变量" field="customVariables">
          <div style="width: 100%;">
            <div v-for="(item, index) in customVariablesList" :key="index" style="display: flex; gap: 8px; margin-bottom: 8px;">
              <a-input v-model="item.key" placeholder="变量名（如：api_host）" style="flex: 1;" />
              <a-input v-model="item.value" placeholder="变量值" style="flex: 2;" />
              <a-button type="text" status="danger" @click="removeCustomVariable(index)">
                <template #icon><icon-delete /></template>
              </a-button>
            </div>
            <a-button type="dashed" long @click="addCustomVariable">
              <template #icon><icon-plus /></template>
              添加变量
            </a-button>
          </div>
          <template #extra>
            <span style="color: var(--ops-text-tertiary); font-size: 12px;">
              自定义变量仅在当前巡检组的巡检项中可用，优先级高于全局变量
            </span>
          </template>
        </a-form-item>

        <a-form-item label="自定义标签">
          <div class="label-input-area">
            <div class="label-tags">
              <a-tag
                v-for="(label, idx) in labelList"
                :key="label + idx"
                closable
                color="arcoblue"
                style="margin: 3px;"
                @close="() => removeLabel(idx)"
              >{{ label }}</a-tag>
            </div>
            <div class="label-add-row">
              <template v-if="labelInputVisible">
                <a-input
                  ref="labelInputRef"
                  v-model="labelInputValue"
                  size="small"
                  style="width: 150px"
                  placeholder="如 env:prod"
                  @blur="confirmLabelInput"
                  @press-enter="confirmLabelInput"
                />
                <a-button size="small" type="primary" @click="confirmLabelInput">确认</a-button>
                <a-button size="small" @click="cancelLabelInput">取消</a-button>
              </template>
              <a-button v-else size="small" @click="showLabelInput">
                <template #icon><icon-plus /></template>
                添加标签
              </a-button>
            </div>
            <div class="label-hint">标签格式：key:value，如 env:prod、team:ops，用于 metric 指标标识</div>
          </div>
        </a-form-item>

        <a-divider>巡检项配置</a-divider>

        <!-- 执行策略配置 -->
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="执行策略" field="executionStrategy">
              <a-radio-group v-model="formData.executionStrategy">
                <a-radio value="concurrent">并发执行</a-radio>
                <a-radio value="sequential">顺序执行</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="并发数量" field="concurrency" v-if="formData.executionStrategy === 'concurrent'">
              <a-input-number v-model="formData.concurrency" :min="1" :max="200" placeholder="默认50" style="width: 100%;" />
              <template #extra>
                <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                  同时执行的巡检项数量，默认50
                </span>
              </template>
            </a-form-item>
          </a-col>
        </a-row>

        <!-- 巡检项列表 -->
        <a-form-item label="巡检项编排">
          <div style="width: 100%;">
            <div
              v-for="(item, index) in inspectionItems"
              :key="index"
              class="inspection-item-card"
              draggable="true"
              @dragstart="onItemDragStart(index)"
              @dragover.prevent="onItemDragOver(index)"
              @drop="onItemDrop(index)"
              @dragend="itemDragIndex = -1"
            >
              <div class="inspection-item-header" @click="activeItemIndex = activeItemIndex === index ? -1 : index">
                <span class="inspection-item-drag-handle"><icon-drag-dot-vertical /></span>
                <span class="inspection-item-index">{{ index + 1 }}</span>
                <span class="inspection-item-name">{{ item.name || `巡检项 ${index + 1}` }}</span>
                <span style="flex: 1;" />
                <a-tag v-if="item.executionType" size="small" :color="getExecutionTypeColor(item.executionType)">
                  {{ getExecutionTypeText(item.executionType) }}
                </a-tag>
                <a-button type="text" size="mini" @click.stop="copyInspectionItem(index)">
                  <template #icon><icon-copy /></template>
                </a-button>
                <a-button type="text" status="danger" size="mini" @click.stop="deleteInspectionItem(index)">
                  <icon-minus />
                </a-button>
                <icon-down v-if="activeItemIndex !== index" style="color: var(--ops-text-tertiary);" />
                <icon-up v-else style="color: var(--ops-text-tertiary);" />
              </div>

              <div v-show="activeItemIndex === index" class="inspection-item-body">
                <a-row :gutter="12">
                  <a-col :span="12">
                    <a-form-item label="巡检项名称" :label-col-flex="'100px'">
                      <a-input v-model="item.name" placeholder="如：CPU使用率检查" />
                    </a-form-item>
                  </a-col>
                  <a-col :span="12">
                    <a-form-item label="执行类型" :label-col-flex="'100px'">
                      <a-select v-model="item.executionType" @change="handleExecutionTypeChange(item)">
                        <a-option value="command">命令</a-option>
                        <a-option value="script">脚本</a-option>
                        <a-option value="promql">PromQL</a-option>
                        <a-option value="probe">拨测</a-option>
                      </a-select>
                    </a-form-item>
                  </a-col>
                </a-row>

                <!-- 主机匹配配置（拨测类型不需要） -->
                <template v-if="item.executionType !== 'probe'">
                  <a-form-item label="主机匹配方式" :label-col-flex="'100px'">
                    <a-radio-group v-model="item.hostMatchType" @change="loadHostsForItem(item)">
                      <a-radio value="tag">按标签匹配</a-radio>
                      <a-radio value="name">按主机名匹配</a-radio>
                      <a-radio value="id">按主机ID匹配</a-radio>
                    </a-radio-group>
                  </a-form-item>

                <a-form-item v-if="item.hostMatchType === 'tag'" label="主机标签" :label-col-flex="'100px'">
                  <a-select
                    v-model="item.hostTags"
                    placeholder="请选择主机标签（可多选）"
                    multiple
                    allow-create
                    allow-search
                    :loading="loadingHosts"
                    @focus="loadHostsForItem(item)"
                  >
                    <a-option v-for="tag in availableTags" :key="tag" :value="tag">
                      {{ tag }}
                    </a-option>
                  </a-select>
                  <template #extra>
                    <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                      从巡检组关联的业务分组中匹配包含这些标签的主机
                    </span>
                  </template>
                </a-form-item>

                <a-form-item v-if="item.hostMatchType === 'name'" label="主机名匹配" :label-col-flex="'100px'">
                  <a-select
                    v-model="item.hostTags"
                    placeholder="请选择主机名（可多选）"
                    multiple
                    allow-create
                    allow-search
                    :loading="loadingHosts"
                    @focus="loadHostsForItem(item)"
                  >
                    <a-option v-for="hostName in availableHostNames" :key="hostName" :value="hostName">
                      {{ hostName }}
                    </a-option>
                  </a-select>
                  <template #extra>
                    <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                      从巡检组关联的业务分组中匹配主机名包含这些关键词的主机
                    </span>
                  </template>
                </a-form-item>

                <a-form-item v-if="item.hostMatchType === 'id'" label="选择主机" :label-col-flex="'100px'">
                  <a-select
                    v-model="item.hostIds"
                    placeholder="请选择主机（可多选）"
                    multiple
                    allow-search
                    :loading="loadingHosts"
                    @focus="loadHostsForItem(item)"
                  >
                    <a-option v-for="host in availableHosts" :key="host.id" :value="host.id">
                      {{ host.name }} ({{ host.ip }})
                    </a-option>
                  </a-select>
                  <template #extra>
                    <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                      从巡检组关联的业务分组中选择主机
                    </span>
                  </template>
                </a-form-item>
                </template>

                <!-- 命令执行 -->
                <template v-if="item.executionType === 'command'">
                  <a-form-item label="执行命令" :label-col-flex="'100px'">
                    <VariableInput
                      v-model="item.command"
                      placeholder="如：uptime 或 echo {{api_host}}"
                      :variables="variableOptions"
                      :multiline="true"
                    />
                    <template #extra>
                      <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                        支持变量引用，输入 / 可选择变量
                      </span>
                    </template>
                  </a-form-item>
                </template>

                <!-- 脚本执行 -->
                <template v-if="item.executionType === 'script'">
                  <a-form-item label="脚本类型" :label-col-flex="'100px'">
                    <a-select v-model="item.scriptType" style="width: 200px;">
                      <a-option value="shell">Shell</a-option>
                      <a-option value="python">Python</a-option>
                      <a-option value="binary">二进制</a-option>
                    </a-select>
                  </a-form-item>

                  <a-form-item label="脚本来源" :label-col-flex="'100px'">
                    <a-radio-group v-model="item.scriptSource" @change="handleScriptSourceChange(item)">
                      <a-radio value="content">直接输入</a-radio>
                      <a-radio value="file">文件上传</a-radio>
                    </a-radio-group>
                  </a-form-item>

                  <a-form-item v-if="item.scriptSource === 'content'" label="脚本内容" :label-col-flex="'100px'">
                    <VariableInput
                      v-if="item.scriptType !== 'binary'"
                      v-model="item.scriptContent"
                      :placeholder="item.scriptType === 'shell' ? '#!/bin/bash\necho Hello {{username}}' : '#!/usr/bin/env python3\nprint(Hello {{username}})'"
                      :variables="variableOptions"
                      :multiline="true"
                    />
                    <a-textarea
                      v-else
                      v-model="item.scriptContent"
                      placeholder="请上传二进制文件"
                      :rows="8"
                      disabled
                    />
                    <template #extra>
                      <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                        {{ item.scriptType === 'binary' ? '二进制文件请使用文件上传方式' : '脚本内容将在目标主机上执行，支持变量引用，输入 / 可选择变量' }}
                      </span>
                    </template>
                  </a-form-item>

                  <a-form-item v-if="item.scriptSource === 'file'" label="上传脚本" :label-col-flex="'100px'">
                    <a-upload
                      :auto-upload="false"
                      :show-file-list="true"
                      :limit="1"
                      @change="(fileList) => handleScriptFileChange(item, fileList)"
                    >
                      <template #upload-button>
                        <a-button type="outline">
                          <template #icon><icon-upload /></template>
                          选择文件
                        </a-button>
                      </template>
                    </a-upload>
                    <template #extra>
                      <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                        {{ item.scriptFile ? `已上传: ${item.scriptFile}` : '支持 .sh、.py、二进制文件等' }}
                      </span>
                    </template>
                  </a-form-item>

                  <a-form-item label="脚本参数" :label-col-flex="'100px'">
                    <VariableInput
                      v-model="item.scriptArgs"
                      placeholder="如：{{env}} {{region}} prod（支持变量引用）"
                      :variables="variableOptions"
                    />
                    <template #extra>
                      <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                        脚本执行时的位置参数，多个参数用空格分隔，支持变量引用
                      </span>
                    </template>
                  </a-form-item>
                </template>

                <!-- PromQL 查询 -->
                <template v-if="item.executionType === 'promql'">
                  <a-form-item label="PromQL查询" :label-col-flex="'100px'">
                    <VariableInput
                      v-model="item.promqlQuery"
                      placeholder="如：node_cpu_usage_percent{instance='{{instance}}'}"
                      :variables="variableOptions"
                      :multiline="true"
                    />
                    <template #extra>
                      <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                        支持变量引用，输入 / 可选择变量
                      </span>
                    </template>
                  </a-form-item>
                </template>

                <!-- 拨测配置 -->
                <template v-if="item.executionType === 'probe'">
                  <a-form-item label="拨测分类" :label-col-flex="'100px'">
                    <a-select
                      v-model="item.probeCategory"
                      placeholder="请选择拨测分类"
                      @change="handleProbeCategoryChange(item)"
                    >
                      <a-option value="network">基础网络</a-option>
                      <a-option value="layer4">四层协议</a-option>
                      <a-option value="application">应用服务</a-option>
                      <a-option value="workflow">业务流程</a-option>
                    </a-select>
                  </a-form-item>

                  <a-form-item v-if="item.probeCategory" label="拨测类型" :label-col-flex="'100px'">
                    <a-select
                      v-model="item.probeType"
                      placeholder="请选择拨测类型（可选）"
                      allow-clear
                      @change="handleProbeTypeChange(item)"
                    >
                      <a-option
                        v-for="type in getProbeTypesByCategory(item.probeCategory)"
                        :key="type"
                        :value="type"
                      >
                        {{ type.toUpperCase() }}
                      </a-option>
                    </a-select>
                  </a-form-item>

                  <a-form-item v-if="item.probeCategory" label="拨测配置" :label-col-flex="'100px'">
                    <a-select
                      v-model="item.probeConfigId"
                      placeholder="请选择拨测配置"
                      :loading="probeConfigsLoading"
                      allow-search
                      :filter-option="false"
                    >
                      <template #empty>
                        <a-empty description="暂无可用的拨测配置" />
                      </template>
                      <a-option
                        v-for="config in getFilteredProbeConfigs(item)"
                        :key="config.id"
                        :value="config.id"
                      >
                        <div style="display: flex; justify-content: space-between;">
                          <span>{{ config.name }}</span>
                          <span style="color: var(--ops-text-tertiary); font-size: 12px;">
                            {{ config.target }}{{ config.port ? ':' + config.port : '' }}
                          </span>
                        </div>
                      </a-option>
                    </a-select>
                  </a-form-item>
                </template>

                <!-- 断言配置和变量提取（拨测类型不需要） -->
                <template v-if="item.executionType !== 'probe'">
                  <a-form-item label="断言规则" :label-col-flex="'100px'">
                    <a-row :gutter="8">
                      <a-col :span="10">
                        <a-select v-model="item.assertionType" placeholder="选择断言类型" allow-clear>
                          <a-option value="gt">大于 (&gt;)</a-option>
                          <a-option value="gte">大于等于 (&gt;=)</a-option>
                          <a-option value="lt">小于 (&lt;)</a-option>
                          <a-option value="lte">小于等于 (&lt;=)</a-option>
                          <a-option value="eq">等于 (==)</a-option>
                          <a-option value="contains">包含</a-option>
                          <a-option value="not_contains">不包含</a-option>
                          <a-option value="regex">正则匹配</a-option>
                          <a-option value="not_regex">反正则匹配</a-option>
                        </a-select>
                      </a-col>
                      <a-col :span="14">
                        <a-input v-model="item.assertionValue" placeholder="断言值" :disabled="!item.assertionType" />
                      </a-col>
                    </a-row>
                  </a-form-item>

                  <!-- 变量提取 -->
                  <a-form-item label="变量提取" :label-col-flex="'100px'">
                    <a-row :gutter="8">
                      <a-col :span="10">
                        <a-input v-model="item.variableName" placeholder="变量名（如：token）" />
                      </a-col>
                      <a-col :span="14">
                        <a-input v-model="item.variableRegex" placeholder="提取正则（如：token=(.+)）" />
                      </a-col>
                    </a-row>
                  </a-form-item>
                </template>

                <a-row :gutter="12">
                  <a-col :span="12">
                    <a-form-item label="超时时间(秒)" :label-col-flex="'100px'">
                      <a-input-number v-model="item.timeout" :min="1" :max="600" placeholder="默认60秒" style="width: 100%;" />
                    </a-form-item>
                  </a-col>
                  <a-col :span="12">
                    <a-form-item label="状态" :label-col-flex="'100px'">
                      <a-radio-group v-model="item.status">
                        <a-radio value="enabled">启用</a-radio>
                        <a-radio value="disabled">禁用</a-radio>
                      </a-radio-group>
                    </a-form-item>
                  </a-col>
                </a-row>

                <a-form-item label="描述" :label-col-flex="'100px'">
                  <a-textarea v-model="item.description" placeholder="请输入描述" :rows="2" />
                </a-form-item>

                <!-- 测试执行按钮 -->
                <a-form-item :label-col-flex="'100px'">
                  <a-button type="outline" @click="handleTestRunItem(index)">
                    <template #icon><icon-play-arrow /></template>
                    测试执行
                  </a-button>
                </a-form-item>
              </div>
            </div>

            <a-button type="dashed" long @click="addInspectionItem">
              <template #icon><icon-plus /></template>
              添加巡检项
            </a-button>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
    <!-- 测试执行日志弹窗 -->
    <a-modal
      v-model:visible="testLogVisible"
      title="测试执行日志"
      :width="900"
      :footer="false"
      unmount-on-close
    >
      <div class="test-log-container">
        <a-spin :loading="testRunning" tip="正在执行测试...">
          <div class="test-log-content">
            <div v-for="(log, index) in testLogs" :key="index" class="log-item">
              <div class="log-header">
                <a-tag :color="getLogStatusColor(log.status)">{{ log.status }}</a-tag>
                <span class="log-host">{{ log.hostName }} ({{ log.hostIp }})</span>
                <span class="log-time">{{ log.duration }}ms</span>
              </div>
              <div class="log-output">
                <pre>{{ log.output || log.errorMessage }}</pre>
              </div>
              <div v-if="log.assertionResult" class="log-assertion">
                <a-tag :color="log.assertionResult === 'pass' ? 'green' : 'red'">
                  断言: {{ log.assertionResult === 'pass' ? '通过' : '失败' }}
                </a-tag>
                <span v-if="log.assertionDetails">{{ log.assertionDetails.message }}</span>
              </div>
            </div>
            <a-empty v-if="testLogs.length === 0 && !testRunning" description="暂无执行日志" />
          </div>
        </a-spin>
      </div>
    </a-modal>

    <!-- 导出配置对话框 -->
    <a-modal
      v-model:visible="exportDialogVisible"
      title="导出巡检组配置"
      :width="800"
      unmount-on-close
    >
      <div style="margin-bottom: 16px;">
        <a-radio-group v-model="exportFormat" type="button">
          <a-radio value="json">JSON</a-radio>
          <a-radio value="yaml">YAML</a-radio>
        </a-radio-group>
        <a-space style="float: right;">
          <a-button size="small" @click="handleCopyExport">
            <template #icon><icon-copy /></template>
            复制
          </a-button>
          <a-button size="small" type="primary" @click="handleDownloadExport">
            <template #icon><icon-download /></template>
            下载
          </a-button>
        </a-space>
      </div>
      <a-textarea
        v-model="exportData"
        :rows="20"
        readonly
        :style="{ fontFamily: 'monospace', fontSize: '12px' }"
      />
      <template #footer>
        <a-button @click="exportDialogVisible = false">关闭</a-button>
      </template>
    </a-modal>

    <!-- 导入配置对话框 -->
    <a-modal
      v-model:visible="importDialogVisible"
      title="导入巡检组配置"
      :width="800"
      unmount-on-close
      @ok="handleImportSubmit"
    >
      <a-tabs default-active-key="text">
        <a-tab-pane key="text" title="文本导入">
          <div style="margin-bottom: 16px;">
            <a-radio-group v-model="importFormat" type="button">
              <a-radio value="json">JSON</a-radio>
              <a-radio value="yaml">YAML</a-radio>
            </a-radio-group>
            <span style="margin-left: 16px; color: var(--ops-text-secondary); font-size: 12px;">
              支持单个或多个巡检组（数组格式）
            </span>
          </div>
          <a-textarea
            v-model="importData"
            :rows="18"
            placeholder="请输入 JSON 或 YAML 格式的配置内容"
            :style="{ fontFamily: 'monospace', fontSize: '12px' }"
          />
        </a-tab-pane>
        <a-tab-pane key="file" title="文件导入">
          <a-upload
            :custom-request="handleFileUpload"
            :show-file-list="true"
            :limit="1"
            accept=".json,.yaml,.yml"
            :auto-upload="false"
            @change="handleFileChange"
          >
            <template #upload-button>
              <div class="upload-area">
                <icon-upload :size="48" style="color: var(--ops-primary);" />
                <div style="margin-top: 16px; font-size: 14px;">
                  点击或拖拽文件到此区域上传
                </div>
                <div style="margin-top: 8px; font-size: 12px; color: var(--ops-text-secondary);">
                  支持 .json、.yaml、.yml 格式文件
                </div>
              </div>
            </template>
          </a-upload>
        </a-tab-pane>
      </a-tabs>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch, nextTick } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconCheckCircle,
  IconSearch,
  IconRefresh,
  IconPlus,
  IconMinus,
  IconDragDotVertical,
  IconDown,
  IconUp,
  IconPlayArrow,
  IconEdit,
  IconDelete,
  IconDownload,
  IconUpload,
  IconCopy,
  IconMore
} from '@arco-design/web-vue/es/icon'
import {
  getInspectionGroups,
  getInspectionGroup,
  createInspectionGroup,
  updateInspectionGroup,
  deleteInspectionGroup,
  getInspectionItems,
  batchSaveInspectionItems,
  testRunInspection,
  testRunInspectionWithoutSave,
  getInspectionStats,
  exportInspectionGroup,
  exportAllInspectionGroups,
  importInspectionGroup,
  importInspectionGroupFile,
  getProbeConfigsForInspection,
  type InspectionGroup,
  type InspectionItem
} from '@/api/inspectionManagement'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'
import { getVariableList } from '@/api/networkProbe'
import VariableInput from '@/components/VariableInput.vue'

const loading = ref(false)
const tableData = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref()
const activeItemIndex = ref(-1)
const itemDragIndex = ref(-1)
const assetGroups = ref<any[]>([])
const availableHosts = ref<any[]>([])
const loadingHosts = ref(false)
const testLogVisible = ref(false)
const testRunning = ref(false)
const testLogs = ref<any[]>([])
const availableTags = ref<string[]>([])
const availableHostNames = ref<string[]>([])

// 变量选项（用于 VariableInput 组件）
const variableOptions = ref<Array<{ name: string; description?: string }>>([])

// 导入导出相关
const exportDialogVisible = ref(false)
const exportFormat = ref<'json' | 'yaml'>('json')
const exportData = ref('')
const exportRecordId = ref(0)
const importDialogVisible = ref(false)
const importFormat = ref<'json' | 'yaml'>('json')
const importData = ref('')

const searchForm = reactive({
  keyword: '',
  status: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const stats = reactive({
  total: 0,
  enabled: 0,
  disabled: 0,
  items: 0
})

const selectedGroupIds = ref<number[]>([])

const formData = reactive({
  id: 0,
  name: '',
  description: '',
  status: 'enabled',
  executionMode: 'auto',
  executionStrategy: 'concurrent',
  concurrency: 50,
  prometheusUrl: '',
  prometheusUsername: '',
  prometheusPassword: '',
  sort: 0,
  groupIds: [] as number[],
  customVariables: '{}' // 自定义变量（JSON 字符串）
})

const inspectionItems = ref<any[]>([])

// 自定义变量列表（用于表单编辑）
const customVariablesList = ref<Array<{ key: string; value: string }>>([])

// 自定义标签
const labelList = ref<string[]>([])
const labelInputVisible = ref(false)
const labelInputValue = ref('')
const labelInputRef = ref()

const showLabelInput = () => {
  labelInputVisible.value = true
  nextTick(() => labelInputRef.value?.focus())
}
const confirmLabelInput = () => {
  const val = labelInputValue.value.trim()
  if (val && !labelList.value.includes(val)) labelList.value.push(val)
  labelInputVisible.value = false
  labelInputValue.value = ''
}
const cancelLabelInput = () => {
  labelInputVisible.value = false
  labelInputValue.value = ''
}
const removeLabel = (idx: number) => {
  labelList.value.splice(idx, 1)
}

const formRules = {
  name: [{ required: true, message: '请输入巡检组名称' }],
  executionMode: [{ required: true, message: '请选择执行方式' }]
}

const getExecutionModeText = (mode: string) => {
  const map: Record<string, string> = {
    auto: '自动',
    agent: '仅Agent',
    ssh: '仅SSH'
  }
  return map[mode] || mode
}

const getExecutionTypeText = (type: string) => {
  const map: Record<string, string> = {
    command: '命令',
    script: '脚本',
    promql: 'PromQL',
    probe: '拨测'
  }
  return map[type] || type
}

const getExecutionTypeColor = (type: string) => {
  const map: Record<string, string> = {
    command: 'arcoblue',
    script: 'green',
    promql: 'orange',
    probe: 'purple'
  }
  return map[type] || 'gray'
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await getInspectionGroups({
      keyword: searchForm.keyword,
      status: searchForm.status,
      page: pagination.page,
      pageSize: pagination.pageSize
    })
    tableData.value = res.list
    pagination.total = res.total

    // 加载统计数据
    const statsRes = await getInspectionStats()
    Object.assign(stats, statsRes)
  } catch (error: any) {
    Message.error(error.message || '获取数据失败')
  } finally {
    loading.value = false
  }
}

const handleReset = () => {
  searchForm.keyword = ''
  searchForm.status = ''
  loadData()
}

const handleCreate = async () => {
  isEdit.value = false
  Object.assign(formData, {
    id: 0,
    name: '',
    description: '',
    status: 'enabled',
    executionMode: 'auto',
    executionStrategy: 'concurrent',
    concurrency: 50,
    prometheusUrl: '',
    prometheusUsername: '',
    prometheusPassword: '',
    sort: 0,
    groupIds: [],
    customVariables: '{}'
  })
  inspectionItems.value = []
  customVariablesList.value = []
  labelList.value = []
  await loadAssetGroups()
  dialogVisible.value = true
}

const handleEdit = async (record: any) => {
  isEdit.value = true

  console.log('编辑巡检组，原始数据:', record)

  // 解析 groupIds
  let groupIds: number[] = []
  if (record.groupIds) {
    try {
      groupIds = typeof record.groupIds === 'string' ? JSON.parse(record.groupIds) : record.groupIds
      console.log('解析后的 groupIds:', groupIds)
    } catch (e) {
      console.error('解析 groupIds 失败:', e, record.groupIds)
      groupIds = []
    }
  }

  // 解析 customVariables
  let customVars: Record<string, string> = {}
  if (record.customVariables) {
    try {
      customVars = typeof record.customVariables === 'string' ? JSON.parse(record.customVariables) : record.customVariables
      console.log('解析后的 customVariables:', customVars)
    } catch (e) {
      console.error('解析 customVariables 失败:', e, record.customVariables)
      customVars = {}
    }
  }

  // 转换为列表格式
  customVariablesList.value = Object.entries(customVars).map(([key, value]) => ({ key, value }))

  // 解析 labels
  try {
    labelList.value = record.labels ? (typeof record.labels === 'string' ? JSON.parse(record.labels) : record.labels) : []
  } catch {
    labelList.value = []
  }

  Object.assign(formData, {
    ...record,
    groupIds,
    executionStrategy: record.executionStrategy || 'concurrent',
    concurrency: record.concurrency || 50
  })

  console.log('formData 赋值后:', formData)

  // 加载资产分组列表
  await loadAssetGroups()

  // 加载主机列表、标签和主机名（用于后续巡检项配置）
  if (groupIds.length > 0) {
    await loadHostsForItem(null)
  }

  // 加载该巡检组的巡检项列表
  try {
    const res = await getInspectionItems({ groupId: record.id })
    console.log('加载巡检项列表:', res)

    inspectionItems.value = res.list.map((item: any, index: number) => {
      console.log(`巡检项 ${index + 1} 原始数据:`, item)

      // 解析 hostTags
      let hostTags: any[] = []
      if (item.hostTags) {
        try {
          hostTags = typeof item.hostTags === 'string' ? JSON.parse(item.hostTags) : item.hostTags
          console.log(`  - hostTags 解析成功:`, hostTags)
        } catch (e) {
          console.error(`  - 解析 hostTags 失败:`, e, item.hostTags)
          hostTags = []
        }
      }

      // 解析 hostIds
      let hostIds: any[] = []
      if (item.hostIds) {
        try {
          hostIds = typeof item.hostIds === 'string' ? JSON.parse(item.hostIds) : item.hostIds
          console.log(`  - hostIds 解析成功:`, hostIds)
        } catch (e) {
          console.error(`  - 解析 hostIds 失败:`, e, item.hostIds)
          hostIds = []
        }
      }

      return {
        id: item.id,
        name: item.name,
        description: item.description,
        executionType: item.executionType,
        command: item.command || '',
        scriptType: item.scriptType || 'shell',
        scriptContent: item.scriptContent || '',
        scriptFile: item.scriptFile || '',
        scriptSource: item.scriptFile ? 'file' : 'content',
        scriptArgs: item.scriptArgs || '',
        promqlQuery: item.promqlQuery || '',
        probeCategory: item.probeCategory || '',
        probeType: item.probeType || '',
        probeConfigId: item.probeConfigId || undefined,
        assertionType: item.assertionType || '',
        assertionValue: item.assertionValue || '',
        variableName: item.variableName || '',
        variableRegex: item.variableRegex || '',
        timeout: item.timeout || 60,
        status: item.status,
        sort: item.sort,
        hostMatchType: item.hostMatchType || 'tag',
        hostTags: hostTags,
        hostIds: hostIds
      }
    })

    console.log('解析后的巡检项列表:', inspectionItems.value)
  } catch (error: any) {
    console.error('加载巡检项失败:', error)
    Message.error(error.message || '加载巡检项失败')
  }

  // 加载拨测配置列表（如果有拨测类型的巡检项）
  if (groupIds.length > 0) {
    await fetchProbeConfigs()
  }

  dialogVisible.value = true
}

const handleDelete = async (record: any) => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除巡检组"${record.name}"吗？删除后将同时删除该组下的所有巡检项。`,
    onOk: async () => {
      try {
        await deleteInspectionGroup(record.id)
        Message.success('删除成功')
        selectedGroupIds.value = []
        loadData()
      } catch (error: any) {
        Message.error(error.message || '删除失败')
      }
    }
  })
}

// 批量删除
const handleBatchDelete = () => {
  Modal.confirm({
    title: '确认批量删除',
    content: `确定要删除选中的 ${selectedGroupIds.value.length} 个巡检组吗？删除后将同时删除这些组下的所有巡检项。`,
    onOk: async () => {
      try {
        const deletePromises = selectedGroupIds.value.map(id => deleteInspectionGroup(id))
        await Promise.all(deletePromises)
        Message.success(`成功删除 ${selectedGroupIds.value.length} 个巡检组`)
        selectedGroupIds.value = []
        loadData()
      } catch (error: any) {
        Message.error(error.message || '批量删除失败')
      }
    }
  })
}


// 导出配置
const handleExport = async (record: any) => {
  exportRecordId.value = record.id
  exportDialogVisible.value = true
  exportFormat.value = 'json'
  exportData.value = '正在导出...'

  try {
    const data = await exportInspectionGroup(record.id, exportFormat.value)
    exportData.value = data
  } catch (error: any) {
    Message.error(error.message || '导出失败')
    exportDialogVisible.value = false
  }
}

// 监听导出格式变化
watch(exportFormat, async (newFormat) => {
  if (exportDialogVisible.value) {
    exportData.value = '正在导出...'
    try {
      let data
      if (exportRecordId.value === 0) {
        // 导出全部
        data = await exportAllInspectionGroups(newFormat)
      } else {
        // 导出单个
        data = await exportInspectionGroup(exportRecordId.value, newFormat)
      }
      exportData.value = data
    } catch (error: any) {
      Message.error(error.message || '导出失败')
    }
  }
})

// 下载导出文件
const handleDownloadExport = () => {
  const blob = new Blob([exportData.value], {
    type: exportFormat.value === 'yaml' ? 'application/x-yaml' : 'application/json'
  })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  const filename = exportRecordId.value === 0
    ? `inspection_groups_all.${exportFormat.value}`
    : `inspection_group_${exportRecordId.value}.${exportFormat.value}`
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
  Message.success('下载成功')
}

// 复制导出内容
const handleCopyExport = () => {
  navigator.clipboard.writeText(exportData.value)
  Message.success('已复制到剪贴板')
}

// 导入配置
const handleImport = () => {
  importDialogVisible.value = true
  importFormat.value = 'json'
  importData.value = ''
}

// 提交导入
const handleImportSubmit = async () => {
  if (!importData.value.trim()) {
    Message.warning('请输入配置内容')
    return
  }

  try {
    const res = await importInspectionGroup({
      format: importFormat.value,
      data: importData.value
    })
    Message.success(`导入成功，共导入 ${res.count} 个巡检组`)
    importDialogVisible.value = false
    loadData()
  } catch (error: any) {
    Message.error(error.message || '导入失败')
  }
}

// 导出全部
const handleExportAll = async () => {
  exportDialogVisible.value = true
  exportRecordId.value = 0 // 0 表示导出全部
  exportFormat.value = 'json'
  exportData.value = '正在导出...'

  try {
    const data = await exportAllInspectionGroups(exportFormat.value)
    exportData.value = data
  } catch (error: any) {
    Message.error(error.message || '导出失败')
    exportDialogVisible.value = false
  }
}

// 文件上传
const uploadFile = ref<File | null>(null)

const handleFileChange = (fileList: any) => {
  if (fileList && fileList.length > 0) {
    uploadFile.value = fileList[0].file
  }
}

const handleFileUpload = async () => {
  if (!uploadFile.value) {
    Message.warning('请选择文件')
    return
  }

  try {
    const res = await importInspectionGroupFile(uploadFile.value)
    Message.success(`导入成功，共导入 ${res.count} 个巡检组`)
    importDialogVisible.value = false
    uploadFile.value = null
    loadData()
  } catch (error: any) {
    Message.error(error.message || '导入失败')
  }
}

const handleTestRun = async (record: any) => {
  testLogVisible.value = true
  testLogs.value = []
  testRunning.value = true

  try {
    // 获取该巡检组的所有巡检项
    const itemsRes = await getInspectionItems({ groupId: record.id })
    if (!itemsRes.list || itemsRes.list.length === 0) {
      Message.warning('该巡检组没有巡检项')
      testRunning.value = false
      return
    }

    const itemIds = itemsRes.list.map((item: any) => item.id)
    const res = await testRunInspection({ groupId: record.id, itemIds })

    if (res.success && res.results) {
      testLogs.value = res.results.map((result: any) => ({
        status: result.status,
        hostName: result.hostName,
        hostIp: result.hostIp || '',
        output: result.output,
        errorMessage: result.errorMessage,
        duration: Math.round(result.duration * 1000),
        assertionResult: result.assertionResult,
        assertionDetails: result.assertionDetails
      }))
      Message.success(`测试执行完成，共执行 ${res.results.length} 条记录`)
    } else {
      Message.warning(res.message || '测试执行失败')
    }
  } catch (error: any) {
    Message.error(error.message || '测试执行失败')
  } finally {
    testRunning.value = false
  }
}

const handleTestRunItem = async (itemIndex: number) => {
  const item = inspectionItems.value[itemIndex]

  if (!item.name) {
    Message.warning('请先填写巡检项名称')
    return
  }

  if (!formData.groupIds || formData.groupIds.length === 0) {
    Message.warning('请先选择业务分组')
    return
  }

  testLogVisible.value = true
  testLogs.value = []
  testRunning.value = true

  try {
    // 如果巡检组还未保存，需要先创建
    let groupId = formData.id
    if (!groupId) {
      const groupData: any = {
        name: formData.name || '临时测试组',
        description: formData.description,
        status: 'disabled', // 测试时创建为禁用状态
        executionMode: formData.executionMode,
        executionStrategy: formData.executionStrategy,
        concurrency: formData.concurrency || 50,
        prometheusUrl: formData.prometheusUrl,
        prometheusUsername: formData.prometheusUsername,
        prometheusPassword: formData.prometheusPassword,
        groupIds: JSON.stringify(formData.groupIds)
      }
      const res = await createInspectionGroup(groupData)
      groupId = res.id
      // 重要：将创建的临时组 ID 设置到 formData，避免后续保存时重复创建
      formData.id = groupId
      isEdit.value = true // 切换为编辑模式
      Message.info('已创建临时巡检组用于测试，请继续完善配置后保存')
    }

    // 准备巡检项数据（不保存到数据库）
    console.log('[TestRun] item.scriptContent:', JSON.stringify(item.scriptContent))
    console.log('[TestRun] item.executionType:', item.executionType)
    const itemData = {
      ...item,
      groupId: groupId,
      sort: itemIndex,
      executionStrategy: 'concurrent',
      hostTags: JSON.stringify(item.hostTags || []),
      hostIds: JSON.stringify(item.hostIds || [])
    }

    // 使用新的不保存数据的测试 API
    const res = await testRunInspectionWithoutSave({ groupId, items: [itemData] })

    if (res.success && res.results) {
      testLogs.value = res.results.map((result: any) => ({
        status: result.status,
        hostName: result.hostName,
        hostIp: result.hostIp || '',
        output: result.output,
        errorMessage: result.errorMessage,
        duration: Math.round(result.duration * 1000),
        assertionResult: result.assertionResult,
        assertionDetails: result.assertionDetails
      }))
      Message.success(`测试执行完成`)
    }
  } catch (error: any) {
    Message.error(error.message || '测试执行失败')
  } finally {
    testRunning.value = false
  }
}

const getLogStatusColor = (status: string) => {
  const map: Record<string, string> = {
    success: 'green',
    failed: 'red',
    running: 'blue'
  }
  return map[status] || 'gray'
}

const handleSubmit = async () => {
  // 防止重复提交
  if (loading.value) {
    return
  }

  const valid = await formRef.value?.validate()
  if (valid) {
    return
  }

  loading.value = true
  try {
    // 将自定义变量列表转换为 JSON 对象
    const customVarsObj: Record<string, string> = {}
    customVariablesList.value.forEach(item => {
      if (item.key && item.value) {
        customVarsObj[item.key] = item.value
      }
    })

    // 准备巡检组数据
    const groupData: any = {
      name: formData.name,
      description: formData.description,
      status: formData.status,
      executionMode: formData.executionMode,
      executionStrategy: formData.executionStrategy,
      concurrency: formData.concurrency || 50,
      prometheusUrl: formData.prometheusUrl,
      prometheusUsername: formData.prometheusUsername,
      prometheusPassword: formData.prometheusPassword,
      groupIds: JSON.stringify(formData.groupIds),
      customVariables: JSON.stringify(customVarsObj),
      labels: JSON.stringify(labelList.value)
    }

    console.log('保存巡检组数据:', groupData)
    console.log('业务分组 IDs:', formData.groupIds)

    let groupId = formData.id

    if (isEdit.value) {
      // 更新巡检组
      await updateInspectionGroup(formData.id, groupData)
      Message.success('更新成功')
    } else {
      // 创建巡检组
      const res = await createInspectionGroup(groupData)
      groupId = res.id
      Message.success('创建成功')
    }

    // 保存巡检项
    if (inspectionItems.value.length > 0) {
      const items = inspectionItems.value.map((item, index) => {
        const itemData = {
          ...item,
          groupId: groupId,
          sort: index,
          executionStrategy: 'concurrent', // 默认并发执行
          hostTags: JSON.stringify(item.hostTags || []),
          hostIds: JSON.stringify(item.hostIds || [])
        }
        console.log(`巡检项 ${index + 1} 数据:`, itemData)
        console.log(`  - hostMatchType: ${item.hostMatchType}`)
        console.log(`  - hostTags (原始):`, item.hostTags)
        console.log(`  - hostTags (序列化):`, itemData.hostTags)
        console.log(`  - hostIds (原始):`, item.hostIds)
        console.log(`  - hostIds (序列化):`, itemData.hostIds)
        return itemData
      })
      await batchSaveInspectionItems(groupId, items)
    }

    dialogVisible.value = false
    loadData()
  } catch (error: any) {
    console.error('保存失败:', error)
    Message.error(error.message || '保存失败')
  } finally {
    loading.value = false
  }
}

const handleCancel = () => {
  dialogVisible.value = false
  formRef.value?.resetFields()
}

// 加载资产分组列表
const loadAssetGroups = async () => {
  try {
    const res = await getGroupTree()
    // 将树形结构扁平化
    const flattenGroups = (groups: any[]): any[] => {
      let result: any[] = []
      groups.forEach(group => {
        result.push({ id: group.id, name: group.name })
        if (group.children && group.children.length > 0) {
          result = result.concat(flattenGroups(group.children))
        }
      })
      return result
    }
    assetGroups.value = flattenGroups(res)
  } catch (error: any) {
    Message.error(error.message || '加载资产分组失败')
  }
}

// 加载主机列表（根据选中的业务分组）
const loadHostsForItem = async (item: any) => {
  if (!formData.groupIds || formData.groupIds.length === 0) {
    Message.warning('请先选择业务分组')
    return
  }

  loadingHosts.value = true
  try {
    const hosts: any[] = []
    const tagsSet = new Set<string>()
    const hostNamesSet = new Set<string>()

    for (const groupId of formData.groupIds) {
      const res = await getHostList({ groupId, page: 1, pageSize: 1000 })
      if (res.list) {
        hosts.push(...res.list)

        // 收集所有主机标签
        res.list.forEach((host: any) => {
          if (host.tags) {
            // 确保 tags 是字符串类型
            const tagsStr = typeof host.tags === 'string' ? host.tags : String(host.tags)
            const tags = tagsStr.split(',').map((t: string) => t.trim()).filter((t: string) => t)
            tags.forEach((tag: string) => tagsSet.add(tag))
          }
          // 收集所有主机名
          if (host.name) {
            hostNamesSet.add(host.name)
          }
        })
      }
    }

    // 去重
    const uniqueHosts = hosts.filter((host, index, self) =>
      index === self.findIndex(h => h.id === host.id)
    )

    availableHosts.value = uniqueHosts
    availableTags.value = Array.from(tagsSet).sort()
    availableHostNames.value = Array.from(hostNamesSet).sort()
  } catch (error: any) {
    Message.error(error.message || '加载主机列表失败')
  } finally {
    loadingHosts.value = false
  }
}

// 巡检项操作
const addInspectionItem = () => {
  inspectionItems.value.push({
    name: '',
    description: '',
    executionType: 'command',
    command: '',
    scriptType: 'shell',
    scriptContent: '',
    scriptFile: '',
    scriptSource: 'content',
    scriptArgs: '',
    promqlQuery: '',
    probeCategory: '',
    probeType: '',
    probeConfigId: undefined,
    assertionType: '',
    assertionValue: '',
    variableName: '',
    variableRegex: '',
    timeout: 60,
    status: 'enabled',
    sort: inspectionItems.value.length,
    hostMatchType: 'tag',
    hostTags: [],
    hostIds: []
  })
  activeItemIndex.value = inspectionItems.value.length - 1
}

const handleScriptSourceChange = (item: any) => {
  // 切换脚本来源时清空相关字段
  if (item.scriptSource === 'content') {
    item.scriptFile = ''
  } else {
    item.scriptContent = ''
  }
}

const handleExecutionTypeChange = (item: any) => {
  // 清除其他类型的字段
  if (item.executionType !== 'command') {
    item.command = ''
  }
  if (item.executionType !== 'script') {
    item.scriptType = 'shell'
    item.scriptContent = ''
    item.scriptFile = ''
    item.scriptSource = 'content'
    item.scriptArgs = ''
  }
  if (item.executionType !== 'promql') {
    item.promqlQuery = ''
  }
  if (item.executionType !== 'probe') {
    item.probeCategory = ''
    item.probeType = ''
    item.probeConfigId = undefined
  }

  // 如果切换到拨测类型，加载拨测配置
  if (item.executionType === 'probe') {
    fetchProbeConfigs()
  }
}

const handleScriptFileChange = async (item: any, fileList: any[]) => {
  if (fileList.length === 0) {
    item.scriptFile = ''
    return
  }

  const file = fileList[0].file
  if (!file) return

  // 读取文件内容（对于文本文件）或保存文件名（对于二进制文件）
  if (item.scriptType === 'binary') {
    // 二进制文件：只保存文件名，实际上传由后端处理
    item.scriptFile = file.name
    item.scriptFileObject = file
  } else {
    // 文本文件：读取内容
    const reader = new FileReader()
    reader.onload = (e) => {
      item.scriptContent = e.target?.result as string
      item.scriptFile = file.name
    }
    reader.readAsText(file)
  }
}

const deleteInspectionItem = (index: number) => {
  inspectionItems.value.splice(index, 1)
  if (activeItemIndex.value === index) {
    activeItemIndex.value = -1
  } else if (activeItemIndex.value > index) {
    activeItemIndex.value--
  }
}

// 复制巡检项
const copyInspectionItem = (index: number) => {
  const originalItem = inspectionItems.value[index]
  const copiedItem = JSON.parse(JSON.stringify(originalItem))
  copiedItem.name = originalItem.name + '_副本'
  copiedItem.id = undefined
  copiedItem.sort = inspectionItems.value.length
  inspectionItems.value.push(copiedItem)
  activeItemIndex.value = inspectionItems.value.length - 1
  Message.success('巡检项已复制')
}

// 复制巡检组
const handleCopy = async (record: InspectionGroup) => {
  isEdit.value = false

  // 复制巡检组数据
  let groupIds: number[] = []
  if (record.groupIds) {
    try {
      groupIds = typeof record.groupIds === 'string' ? JSON.parse(record.groupIds) : record.groupIds
    } catch (e) {
      console.error('解析 groupIds 失败:', e)
      groupIds = []
    }
  }

  // 解析 customVariables
  let customVars: Record<string, string> = {}
  if (record.customVariables) {
    try {
      customVars = typeof record.customVariables === 'string' ? JSON.parse(record.customVariables) : record.customVariables
    } catch (e) {
      console.error('解析 customVariables 失败:', e)
      customVars = {}
    }
  }

  // 转换为列表格式
  customVariablesList.value = Object.entries(customVars).map(([key, value]) => ({ key, value }))

  Object.assign(formData, {
    id: undefined,
    name: record.name + '_副本',
    description: record.description,
    status: record.status,
    executionMode: record.executionMode,
    executionStrategy: record.executionStrategy || 'concurrent',
    concurrency: record.concurrency || 50,
    prometheusUrl: record.prometheusUrl || '',
    prometheusUsername: record.prometheusUsername || '',
    prometheusPassword: record.prometheusPassword || '',
    groupIds
  })

  // 加载资产分组列表
  await loadAssetGroups()

  // 加载主机列表、标签和主机名（用于后续巡检项配置）
  if (groupIds.length > 0) {
    await loadHostsForItem(null)
  }

  // 加载该巡检组的巡检项列表并复制
  try {
    const res = await getInspectionItems({ groupId: record.id })

    inspectionItems.value = res.list.map((item: any, index: number) => {
      // 解析 hostTags
      let hostTags: any[] = []
      if (item.hostTags) {
        try {
          hostTags = typeof item.hostTags === 'string' ? JSON.parse(item.hostTags) : item.hostTags
        } catch (e) {
          console.error('解析 hostTags 失败:', e, item.hostTags)
          hostTags = []
        }
      }

      // 解析 hostIds
      let hostIds: any[] = []
      if (item.hostIds) {
        try {
          hostIds = typeof item.hostIds === 'string' ? JSON.parse(item.hostIds) : item.hostIds
        } catch (e) {
          console.error('解析 hostIds 失败:', e, item.hostIds)
          hostIds = []
        }
      }

      return {
        id: undefined,
        name: item.name + '_副本',
        description: item.description,
        executionType: item.executionType,
        executionStrategy: item.executionStrategy,
        command: item.command || '',
        scriptType: item.scriptType || 'shell',
        scriptContent: item.scriptContent || '',
        promqlQuery: item.promqlQuery || '',
        probeCategory: item.probeCategory || '',
        probeType: item.probeType || '',
        probeConfigId: item.probeConfigId || undefined,
        assertionType: item.assertionType || '',
        assertionValue: item.assertionValue || '',
        variableName: item.variableName || '',
        variableRegex: item.variableRegex || '',
        timeout: item.timeout || 60,
        status: item.status,
        sort: index,
        hostMatchType: item.hostMatchType || 'tag',
        hostTags: hostTags,
        hostIds: hostIds
      }
    })

    activeItemIndex.value = inspectionItems.value.length > 0 ? 0 : -1
  } catch (error: any) {
    console.error('加载巡检项失败:', error)
    Message.error(error.message || '加载巡检项失败')
  }

  dialogVisible.value = true
  Message.success('巡检组已复制，请修改后保存')
}

// 拖拽排序
const onItemDragStart = (index: number) => {
  itemDragIndex.value = index
}

const onItemDragOver = (index: number) => {
  if (itemDragIndex.value === -1 || itemDragIndex.value === index) return

  const dragItem = inspectionItems.value[itemDragIndex.value]
  inspectionItems.value.splice(itemDragIndex.value, 1)
  inspectionItems.value.splice(index, 0, dragItem)

  if (activeItemIndex.value === itemDragIndex.value) {
    activeItemIndex.value = index
  } else if (activeItemIndex.value === index) {
    activeItemIndex.value = itemDragIndex.value > index ? activeItemIndex.value + 1 : activeItemIndex.value - 1
  }

  itemDragIndex.value = index
}

const onItemDrop = (index: number) => {
  itemDragIndex.value = -1
}

// 自定义变量管理
const addCustomVariable = () => {
  customVariablesList.value.push({ key: '', value: '' })
}

const removeCustomVariable = (index: number) => {
  customVariablesList.value.splice(index, 1)
}

// 拨测相关
const probeConfigsLoading = ref(false)
const probeConfigs = ref<any[]>([])

// 拨测类型映射
const CATEGORY_TYPE_MAP: Record<string, string[]> = {
  network: ['ping'],
  layer4: ['tcp', 'udp'],
  application: ['http', 'https', 'websocket'],
  workflow: ['workflow']
}

// 获取分类对应的拨测类型
const getProbeTypesByCategory = (category: string) => {
  return CATEGORY_TYPE_MAP[category] || []
}

// 获取过滤后的拨测配置
const getFilteredProbeConfigs = (item: any) => {
  let configs = probeConfigs.value

  // 按分类过滤
  if (item.probeCategory) {
    configs = configs.filter((c: any) => c.category === item.probeCategory)
  }

  // 按类型过滤（可选）
  if (item.probeType) {
    configs = configs.filter((c: any) => c.type === item.probeType)
  }

  return configs
}

// 拨测分类变更处理
const handleProbeCategoryChange = (item: any) => {
  item.probeType = ''
  item.probeConfigId = undefined
  fetchProbeConfigs()
}

// 拨测类型变更处理
const handleProbeTypeChange = (item: any) => {
  item.probeConfigId = undefined
}

// 获取拨测配置列表
const fetchProbeConfigs = async () => {
  probeConfigsLoading.value = true
  try {
    // 如果没有选择业务分组，传递空数组，后端会返回所有 group_id = 0 的通用配置
    const groupIds = formData.groupIds && formData.groupIds.length > 0 ? formData.groupIds : [0]

    const res = await getProbeConfigsForInspection({
      groupIds: groupIds,
      status: 1
    })
    probeConfigs.value = res.data || []
    console.log('[fetchProbeConfigs] 加载拨测配置:', res.data?.length || 0, '条', 'groupIds:', groupIds)
  } catch (error: any) {
    console.error('[fetchProbeConfigs] 获取拨测配置失败:', error)
    Message.error('获取拨测配置失败: ' + (error.message || '未知错误'))
    probeConfigs.value = []
  } finally {
    probeConfigsLoading.value = false
  }
}

// 加载变量选项（全局变量 + 巡检组自定义变量）
const loadVariableOptions = async () => {
  const options: Array<{ name: string; description?: string }> = []

  try {
    // 1. 加载全局环境变量
    const res = await getVariableList({
      page: 1,
      pageSize: 1000
    })

    if (res.data && Array.isArray(res.data)) {
      res.data.forEach((v: any) => {
        options.push({
          name: v.name,
          description: v.description || '全局变量'
        })
      })
    }
  } catch (error: any) {
    console.error('加载全局变量失败:', error)
  }

  // 2. 添加巡检组自定义变量
  customVariablesList.value.forEach(item => {
    if (item.key) {
      options.push({
        name: item.key,
        description: '组变量'
      })
    }
  })

  variableOptions.value = options
  console.log('[loadVariableOptions] 加载变量选项:', options.length, '个')
}

// 监听自定义变量列表变化，更新变量选项
watch(customVariablesList, () => {
  loadVariableOptions()
}, { deep: true })

// 监听对话框打开，加载变量选项
watch(dialogVisible, (visible) => {
  if (visible) {
    loadVariableOptions()
  }
})

// 初始化加载数据
onMounted(() => {
  loadData()
})
</script>

<style scoped>
.inspection-management-container {
  padding: 20px;
}

.label-input-area {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 4px;
  background: #fafafa;
}
.label-tags {
  display: flex;
  flex-wrap: wrap;
  min-height: 4px;
  margin-bottom: 6px;
}
.label-add-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.label-hint {
  margin-top: 6px;
  font-size: 12px;
  color: #86909c;
}

.page-header {
  margin-bottom: 20px;
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-title-icon {
  font-size: 32px;
  color: var(--ops-primary);
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--ops-text-primary);
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 14px;
  color: var(--ops-text-secondary);
}

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
}

.stats-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.stat-card {
  flex: 1;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  text-align: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--ops-text-primary);
}

.stat-label {
  margin-top: 8px;
  font-size: 14px;
  color: var(--ops-text-secondary);
}

.inspection-item-card {
  margin-bottom: 12px;
  border: 1px solid var(--ops-border-color);
  border-radius: 8px;
  background: #fff;
  cursor: move;
  transition: all 0.2s;
}

.inspection-item-card:hover {
  border-color: var(--ops-primary);
  box-shadow: 0 2px 8px rgba(22, 93, 255, 0.1);
}

.inspection-item-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  user-select: none;
}

.inspection-item-drag-handle {
  cursor: move;
  color: var(--ops-text-tertiary);
  font-size: 16px;
}

.inspection-item-index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--ops-primary);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.inspection-item-name {
  font-weight: 500;
  color: var(--ops-text-primary);
}

.inspection-item-body {
  padding: 16px;
  border-top: 1px solid var(--ops-border-color);
  background: #f7f8fa;
}

.test-log-container {
  max-height: 600px;
  overflow-y: auto;
}

.test-log-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.log-item {
  padding: 12px;
  background: #f7f8fa;
  border-radius: 4px;
  border: 1px solid var(--ops-border-color);
}

.log-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.log-host {
  font-weight: 500;
  color: var(--ops-text-primary);
}

.log-time {
  margin-left: auto;
  color: var(--ops-text-tertiary);
  font-size: 12px;
}

.log-output {
  margin: 8px 0;
  padding: 8px;
  background: #fff;
  border-radius: 4px;
  border: 1px solid var(--ops-border-color);
}

.log-output pre {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.5;
  color: var(--ops-text-primary);
  white-space: pre-wrap;
  word-wrap: break-word;
}

.log-assertion {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--ops-text-secondary);
}

.upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  border: 2px dashed var(--ops-border-color);
  border-radius: 8px;
  background: var(--ops-content-bg);
  cursor: pointer;
  transition: all 0.3s;
}

.upload-area:hover {
  border-color: var(--ops-primary);
  background: rgba(22, 93, 255, 0.05);
}
</style>
