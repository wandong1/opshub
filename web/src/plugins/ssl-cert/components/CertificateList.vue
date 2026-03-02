<template>
  <div class="certificate-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <icon-lock />
        </div>
        <div>
          <h2 class="page-title">证书管理</h2>
          <p class="page-subtitle">管理SSL证书，支持Let's Encrypt自动申请和手动导入</p>
        </div>
      </div>
      <div class="header-actions">
        <a-dropdown @select="handleCreate">
          <a-button type="primary">
            <template #icon><icon-plus /></template>
            新增证书
            <icon-down style="margin-left: 6px;" />
          </a-button>
          <template #content>
            <a-doption value="apply">申请证书 (Let's Encrypt)</a-doption>
            <a-doption value="import">导入证书</a-doption>
          </template>
        </a-dropdown>
        <a-button @click="loadData">
          <template #icon><icon-refresh /></template>
          刷新
        </a-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <a-input
          v-model="searchForm.domain"
          placeholder="搜索域名..."
          allow-clear
          class="search-input"
          @keyup.enter="loadData"
        >
          <template #prefix>
            <icon-search />
          </template>
        </a-input>
        <a-select
          v-model="searchForm.status"
          placeholder="证书状态"
          allow-clear
          class="search-input"
          @change="loadData"
        >
          <a-option value="active">正常</a-option>
          <a-option value="expiring">即将过期</a-option>
          <a-option value="expired">已过期</a-option>
          <a-option value="pending">待申请</a-option>
          <a-option value="error">错误</a-option>
        </a-select>

        <a-select
          v-model="searchForm.source_type"
          placeholder="证书来源"
          allow-clear
          class="search-input"
          @change="loadData"
        >
          <a-option value="letsencrypt">Let's Encrypt</a-option>
          <a-option value="aliyun">阿里云</a-option>
          <a-option value="manual">手动导入</a-option>
        </a-select>
      </div>

      <div class="search-actions">
        <a-button class="reset-btn" @click="handleReset">
          <template #icon><icon-refresh /></template>
          重置
        </a-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon stat-icon-primary">
          <icon-file />
        </div>
        <div class="stat-content">
          <div class="stat-label">证书总数</div>
          <div class="stat-value">{{ stats.total || 0 }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-success">
          <icon-check-circle />
        </div>
        <div class="stat-content">
          <div class="stat-label">正常</div>
          <div class="stat-value">{{ stats.active || 0 }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-warning">
          <icon-exclamation-circle />
        </div>
        <div class="stat-content">
          <div class="stat-label">即将过期</div>
          <div class="stat-value">{{ stats.expiring || 0 }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon stat-icon-danger">
          <icon-close-circle />
        </div>
        <div class="stat-content">
          <div class="stat-label">已过期/错误</div>
          <div class="stat-value">{{ (stats.expired || 0) + (stats.error || 0) }}</div>
        </div>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <a-table
        :data="tableData"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: pagination.total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50, 100] }"
        @page-change="(p: number) => { pagination.page = p; loadData() }"
        @page-size-change="(s: number) => { pagination.pageSize = s; pagination.page = 1; loadData() }"
      >
        <template #columns>
          <a-table-column title="证书名称" data-index="name" :width="120" ellipsis tooltip>
            <template #cell="{ record }">
              <div class="cert-name">{{ record.name }}</div>
            </template>
          </a-table-column>

          <a-table-column title="域名" data-index="domain" :width="180" ellipsis tooltip>
            <template #cell="{ record }">
              <div class="domain-cell">
                <span class="domain-main">{{ record.domain }}</span>
                <a-tag v-if="record.san_domains && JSON.parse(record.san_domains || '[]').length > 0" color="gray" size="small" style="margin-left: 8px;">
                  +{{ JSON.parse(record.san_domains || '[]').length }} SAN
                </a-tag>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="80" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.status === 'active'" color="green">正常</a-tag>
              <a-tag v-else-if="record.status === 'expiring'" color="orangered">即将过期</a-tag>
              <a-tag v-else-if="record.status === 'expired'" color="red">已过期</a-tag>
              <a-tag v-else-if="record.status === 'pending'" color="gray">待申请</a-tag>
              <a-tag v-else color="red">错误</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="来源" ellipsis tooltip>
            <template #cell="{ record }">
              <span>{{ getSourceTypeName(record.source_type) }}</span>
            </template>
          </a-table-column>

          <a-table-column title="CA提供商" :min-width="120" ellipsis tooltip>
            <template #cell="{ record }">
              <span>{{ getCAProviderName(record.ca_provider) }}</span>
            </template>
          </a-table-column>

          <a-table-column title="加密算法" :min-width="90" ellipsis tooltip>
            <template #cell="{ record }">
              <span>{{ getKeyAlgorithmName(record.key_algorithm) }}</span>
            </template>
          </a-table-column>

          <a-table-column title="颁发者" data-index="issuer" :min-width="140" ellipsis tooltip>
            <template #cell="{ record }">
              <span>{{ record.issuer || '-' }}</span>
            </template>
          </a-table-column>

          <a-table-column title="到期时间" :min-width="110">
            <template #cell="{ record }">
              <div v-if="record.not_after">
                <span :class="getExpiryClass(record.not_after)">{{ formatDateTime(record.not_after) }}</span>
                <div class="expiry-days">{{ getExpiryDays(record.not_after) }}</div>
              </div>
              <span v-else>-</span>
            </template>
          </a-table-column>

          <a-table-column title="自动续期" :width="80" align="center">
            <template #cell="{ record }">
              <a-switch
                v-model="record.auto_renew"
                @change="handleAutoRenewChange(record)"
                :disabled="record.source_type === 'manual'"
              />
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="260" fixed="right" align="center">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="查看详情" position="top">
                  <a-button type="text" class="action-btn action-view" @click="handleView(record)">
                    <icon-eye />
                  </a-button>
                </a-tooltip>
                <a-tooltip content="下载证书" position="top">
                  <a-button type="text" class="action-btn action-download" @click="handleDownload(record)" :disabled="record.status === 'pending' || record.status === 'error'">
                    <icon-download />
                  </a-button>
                </a-tooltip>
                <a-tooltip content="手动续期" position="top" v-if="record.source_type !== 'manual' && record.status !== 'pending'">
                  <a-button type="text" class="action-btn action-renew" @click="handleRenew(record)">
                    <icon-refresh />
                  </a-button>
                </a-tooltip>
                <a-tooltip content="同步状态" position="top" v-if="record.source_type === 'aliyun' && record.status === 'pending'">
                  <a-button type="text" class="action-btn action-sync" @click="handleSync(record)">
                    <icon-refresh />
                  </a-button>
                </a-tooltip>
                <a-tooltip content="编辑" position="top">
                  <a-button type="text" class="action-btn action-edit" @click="handleEdit(record)">
                    <icon-edit />
                  </a-button>
                </a-tooltip>
                <a-tooltip content="删除" position="top">
                  <a-button type="text" class="action-btn action-delete" @click="handleDelete(record)">
                    <icon-delete />
                  </a-button>
                </a-tooltip>
              </div>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>
    <!-- 申请证书对话框 -->
    <a-modal
      v-model:visible="applyDialogVisible"
      title="申请证书"
      :width="680"
      :mask-closable="false"
      unmount-on-close
    >
      <div class="dialog-scroll-body">
      <a-form :model="applyForm" :rules="applyRules" ref="applyFormRef" auto-label-width>
        <a-form-item label="证书名称" field="name">
          <a-input v-model="applyForm.name" placeholder="请输入证书名称" />
        </a-form-item>
        <a-form-item label="主域名" field="domain">
          <a-input v-model="applyForm.domain" placeholder="请输入主域名，如：example.com" />
        </a-form-item>
        <a-form-item label="SAN域名">
          <a-select
            v-model="applyForm.san_domains"
            :multiple="true"
            allow-search
            allow-create
            placeholder="输入域名后按回车添加，如：www.example.com"
            style="width: 100%"
          />
          <div class="form-tip">可选，让一张证书保护多个域名。例如主域名是 example.com，可添加 www.example.com、api.example.com 等</div>
        </a-form-item>

        <a-divider orientation="left">证书配置</a-divider>

        <a-form-item label="证书类型" field="source_type">
          <a-radio-group v-model="applyForm.source_type" @change="handleSourceTypeChange">
            <a-radio value="acme">ACME免费证书</a-radio>
            <a-radio value="aliyun">阿里云CAS</a-radio>
          </a-radio-group>
          <div class="form-tip" v-if="applyForm.source_type === 'acme'">
            ACME免费证书支持 Let's Encrypt、ZeroSSL 等 CA 提供商，有效期90天，支持自动续期
          </div>
          <div class="form-tip" v-else-if="applyForm.source_type === 'aliyun'">
            <icon-exclamation-circle style="color: #ff7d00; margin-right: 4px;" />
            阿里云CAS免费证书，每个实名账号每年20张额度，有效期1年，不支持自动续期
          </div>
        </a-form-item>
        <!-- 云账号选择 (云厂商证书) -->
        <a-form-item label="云账号" field="cloud_account_id" v-if="applyForm.source_type === 'aliyun'">
          <a-select
            v-model="applyForm.cloud_account_id"
            placeholder="请选择云账号"
            style="width: 100%"
            :loading="cloudAccountsLoading"
          >
            <a-option
              v-for="item in cloudAccounts"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </a-select>
          <div class="form-tip">
            <icon-exclamation-circle style="color: #ff7d00; margin-right: 4px;" />
            请在「资产管理 - 云账号」中添加云账号，需要有证书服务的相关权限
          </div>
        </a-form-item>

        <a-form-item label="CA提供商" field="ca_provider" v-if="applyForm.source_type === 'acme'">
          <a-select v-model="applyForm.ca_provider" placeholder="请选择CA提供商" style="width: 100%">
            <a-option label="Let's Encrypt (推荐)" value="letsencrypt">
              <span>Let's Encrypt</span>
              <span style="color: #00b42a; margin-left: 8px; font-size: 12px;">推荐</span>
            </a-option>
            <a-option label="ZeroSSL" value="zerossl" />
            <a-option label="Google Trust Services" value="google" />
            <a-option label="BuyPass" value="buypass" />
          </a-select>
          <div class="form-tip">不同CA提供商的证书有效期和签发策略可能不同</div>
        </a-form-item>

        <a-form-item label="加密算法" field="key_algorithm">
          <a-select v-model="applyForm.key_algorithm" placeholder="请选择加密算法" style="width: 100%">
            <a-option label="EC P-256 (推荐)" value="ec256">
              <span>EC P-256</span>
              <span style="color: #00b42a; margin-left: 8px; font-size: 12px;">推荐</span>
            </a-option>
            <a-option label="EC P-384" value="ec384" />
            <a-option label="RSA 2048" value="rsa2048" />
            <a-option label="RSA 3072" value="rsa3072" />
            <a-option label="RSA 4096" value="rsa4096" />
          </a-select>
          <div class="form-tip">EC算法性能更好，RSA兼容性更广</div>
        </a-form-item>
        <a-divider orientation="left" v-if="applyForm.source_type === 'acme'">域名验证配置</a-divider>

        <a-form-item label="邮箱地址" field="acme_email" v-if="applyForm.source_type === 'acme'">
          <a-input v-model="applyForm.acme_email" placeholder="请输入邮箱地址" />
          <div class="form-tip">用于ACME账户注册和接收证书过期提醒</div>
        </a-form-item>
        <a-form-item label="DNS验证" field="dns_provider_id" v-if="applyForm.source_type === 'acme'">
          <a-select v-model="applyForm.dns_provider_id" placeholder="请选择DNS服务商" style="width: 100%">
            <a-option
              v-for="item in dnsProviders"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            >
              <span>{{ item.name }}</span>
              <span style="color: #999; margin-left: 8px;">{{ item.provider }}</span>
            </a-option>
          </a-select>
          <div class="form-tip">
            <icon-exclamation-circle style="color: #ff7d00; margin-right: 4px;" />
            选择你域名所在的DNS服务商，系统将通过其API自动完成域名所有权验证（DNS-01验证）
          </div>
        </a-form-item>

        <a-divider orientation="left">续期配置</a-divider>

        <a-form-item label="自动续期">
          <a-switch v-model="applyForm.auto_renew" />
          <span style="margin-left: 12px; color: #86909c; font-size: 13px;">证书到期前自动续期</span>
        </a-form-item>
        <a-form-item label="提前续期天数" v-if="applyForm.auto_renew">
          <a-input-number v-model="applyForm.renew_days_before" :min="7" :max="90" />
          <span style="margin-left: 12px; color: #86909c; font-size: 13px;">天（Let's Encrypt证书建议30天）</span>
        </a-form-item>
      </a-form>
      </div>

      <template #footer>
        <a-button @click="applyDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleApplySubmit" :loading="submitting">申请证书</a-button>
      </template>
    </a-modal>
    <!-- 导入证书对话框 -->
    <a-modal
      v-model:visible="importDialogVisible"
      title="导入证书"
      :width="750"
      :mask-closable="false"
      unmount-on-close
    >
      <CertificateUpload @submit="handleCertificateUploaded" />
    </a-modal>

    <!-- 编辑对话框 -->
    <a-modal
      v-model:visible="editDialogVisible"
      title="编辑证书配置"
      :width="540"
      :mask-closable="false"
      unmount-on-close
    >
      <a-form :model="editForm" ref="editFormRef" auto-label-width>
        <a-form-item label="证书名称" field="name">
          <a-input v-model="editForm.name" placeholder="请输入证书名称" />
        </a-form-item>
        <a-form-item label="自动续期" v-if="editForm.source_type !== 'manual'">
          <a-switch v-model="editForm.auto_renew" />
        </a-form-item>
        <a-form-item label="提前续期天数" v-if="editForm.auto_renew && editForm.source_type !== 'manual'">
          <a-input-number v-model="editForm.renew_days_before" :min="7" :max="90" />
          <span style="margin-left: 12px; color: #86909c; font-size: 13px;">天</span>
        </a-form-item>
        <a-form-item label="DNS Provider" v-if="editForm.source_type !== 'manual'">
          <a-select v-model="editForm.dns_provider_id" placeholder="请选择DNS Provider" style="width: 100%" allow-clear>
            <a-option
              v-for="item in dnsProviders"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </a-select>
          <div class="form-tip" v-if="editForm.source_type === 'aliyun'">
            云厂商证书如需通过ACME自动续期，请配置DNS Provider
          </div>
        </a-form-item>
        <a-form-item label="ACME邮箱" v-if="editForm.source_type !== 'manual'">
          <a-input v-model="editForm.acme_email" placeholder="用于ACME账户注册和证书过期提醒" />
          <div class="form-tip">
            手动续期或自动续期需要配置ACME邮箱
          </div>
        </a-form-item>
      </a-form>

      <template #footer>
        <a-button @click="editDialogVisible = false">取消</a-button>
        <a-button type="primary" @click="handleEditSubmit" :loading="submitting">保存</a-button>
      </template>
    </a-modal>
    <!-- 详情对话框 -->
    <a-modal
      v-model:visible="detailDialogVisible"
      title="证书详情"
      :width="720"
      :footer="false"
    >
      <div v-if="currentCert" class="detail-content">
        <div class="detail-status-bar">
          <a-tag v-if="currentCert.status === 'active'" color="green" size="large">正常</a-tag>
          <a-tag v-else-if="currentCert.status === 'expiring'" color="orangered" size="large">即将过期</a-tag>
          <a-tag v-else-if="currentCert.status === 'expired'" color="red" size="large">已过期</a-tag>
          <a-tag v-else-if="currentCert.status === 'pending'" color="gray" size="large">待申请</a-tag>
          <a-tag v-else color="red" size="large">错误</a-tag>
          <span class="detail-domain">{{ currentCert.domain }}</span>
        </div>
        <div class="detail-info">
          <div class="detail-info-section">
            <div class="detail-section-title">基本信息</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">证书名称</span>
                <span class="info-value">{{ currentCert.name }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">主域名</span>
                <span class="info-value">{{ currentCert.domain }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">来源</span>
                <span class="info-value">{{ getSourceTypeName(currentCert.source_type) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">CA提供商</span>
                <span class="info-value">{{ getCAProviderName(currentCert.ca_provider) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">加密算法</span>
                <span class="info-value">{{ getKeyAlgorithmName(currentCert.key_algorithm) }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">颁发者</span>
                <span class="info-value">{{ currentCert.issuer || '-' }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section">
            <div class="detail-section-title">有效期</div>
            <div class="detail-grid">
              <div class="info-item">
                <span class="info-label">生效时间</span>
                <span class="info-value">{{ formatDateTime(currentCert.not_before) || '-' }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">到期时间</span>
                <span class="info-value" :class="getExpiryClass(currentCert.not_after)">
                  {{ formatDateTime(currentCert.not_after) || '-' }}
                  <span v-if="currentCert.not_after" class="expiry-days-inline">{{ getExpiryDays(currentCert.not_after) }}</span>
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">自动续期</span>
                <span class="info-value">{{ currentCert.auto_renew ? '是' : '否' }}</span>
              </div>
              <div class="info-item" v-if="currentCert.auto_renew">
                <span class="info-label">提前续期</span>
                <span class="info-value">{{ currentCert.renew_days_before }} 天</span>
              </div>
              <div class="info-item" v-if="currentCert.last_renew_at">
                <span class="info-label">上次续期</span>
                <span class="info-value">{{ formatDateTime(currentCert.last_renew_at) }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section">
            <div class="detail-section-title">安全信息</div>
            <div class="detail-grid">
              <div class="info-item full-width">
                <span class="info-label">指纹</span>
                <span class="info-value fingerprint">{{ currentCert.fingerprint || '-' }}</span>
              </div>
            </div>
          </div>
          <div class="detail-info-section" v-if="currentCert.last_error">
            <div class="detail-section-title error-section-title">错误信息</div>
            <div class="error-block">
              {{ formatErrorMessage(currentCert.last_error) }}
            </div>
          </div>
        </div>
      </div>
    </a-modal>
    <!-- 下载对话框 -->
    <a-modal
      v-model:visible="downloadDialogVisible"
      title="下载证书"
      :width="620"
      :footer="false"
    >
      <div class="download-card">
        <div class="download-card-header">
          <div class="download-domain-info">
            <icon-lock class="domain-icon" />
            <div class="domain-text">
              <span class="domain-name">{{ downloadCertDomain }}</span>
              <span class="domain-hint">SSL 证书文件</span>
            </div>
          </div>
        </div>

        <div class="download-format-selector">
          <div
            class="format-option"
            :class="{ active: downloadFormat === 'pem' }"
            @click="downloadFormat = 'pem'"
          >
            <icon-file class="format-icon" />
            <div class="format-info">
              <span class="format-name">PEM 格式</span>
              <span class="format-desc">通用格式</span>
            </div>
          </div>
          <div
            class="format-option"
            :class="{ active: downloadFormat === 'nginx' }"
            @click="downloadFormat = 'nginx'"
          >
            <icon-link class="format-icon" />
            <div class="format-info">
              <span class="format-name">Nginx 格式</span>
              <span class="format-desc">fullchain + key</span>
            </div>
          </div>
        </div>
        <div class="download-files">
          <div class="file-item" v-if="downloadFormat === 'pem'">
            <div class="file-icon cert-icon">
              <icon-file />
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}.pem</span>
              <span class="file-size">证书文件</span>
            </div>
            <div class="file-actions">
              <a-button type="text" status="normal" @click="copyToClipboard(downloadContent.certificate)">
                <icon-copy />
              </a-button>
              <a-button type="text" status="normal" @click="downloadFile(downloadContent.certificate, `${downloadCertDomain}.pem`)">
                <icon-download />
              </a-button>
            </div>
          </div>
          <div class="file-item" v-if="downloadFormat === 'pem'">
            <div class="file-icon key-icon">
              <icon-lock />
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}.key</span>
              <span class="file-size">私钥文件</span>
            </div>
            <div class="file-actions">
              <a-button type="text" status="normal" @click="copyToClipboard(downloadContent.private_key)">
                <icon-copy />
              </a-button>
              <a-button type="text" status="normal" @click="downloadFile(downloadContent.private_key, `${downloadCertDomain}.key`)">
                <icon-download />
              </a-button>
            </div>
          </div>
          <div class="file-item" v-if="downloadFormat === 'nginx'">
            <div class="file-icon cert-icon">
              <icon-file />
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}_fullchain.pem</span>
              <span class="file-size">完整证书链</span>
            </div>
            <div class="file-actions">
              <a-button type="text" status="normal" @click="copyToClipboard(downloadContent.ssl_certificate)">
                <icon-copy />
              </a-button>
              <a-button type="text" status="normal" @click="downloadFile(downloadContent.ssl_certificate, `${downloadCertDomain}_fullchain.pem`)">
                <icon-download />
              </a-button>
            </div>
          </div>
          <div class="file-item" v-if="downloadFormat === 'nginx'">
            <div class="file-icon key-icon">
              <icon-lock />
            </div>
            <div class="file-info">
              <span class="file-name">{{ downloadCertDomain }}.key</span>
              <span class="file-size">私钥文件</span>
            </div>
            <div class="file-actions">
              <a-button type="text" status="normal" @click="copyToClipboard(downloadContent.ssl_certificate_key)">
                <icon-copy />
              </a-button>
              <a-button type="text" status="normal" @click="downloadFile(downloadContent.ssl_certificate_key, `${downloadCertDomain}.key`)">
                <icon-download />
              </a-button>
            </div>
          </div>
        </div>

        <div class="download-action">
          <a-button type="primary" size="large" @click="downloadAllAsZip" :loading="downloadingZip">
            <template #icon><icon-download /></template>
            下载 ZIP 压缩包
          </a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import {
  IconPlus,
  IconSearch,
  IconRefresh,
  IconEdit,
  IconDelete,
  IconEye,
  IconLock,
  IconFile,
  IconCheckCircle,
  IconCloseCircle,
  IconExclamationCircle,
  IconDownload,
  IconDown,
  IconLink,
  IconCopy
} from '@arco-design/web-vue/es/icon'
import JSZip from 'jszip'
import {
  getCertificates,
  getCertificate,
  createCertificate,
  importCertificate,
  updateCertificate,
  deleteCertificate,
  renewCertificate,
  syncCertificate,
  downloadCertificate,
  getCertificateStats,
  getAllDNSProviders,
  getCloudAccounts
} from '../api/ssl-cert'
import CertificateUpload from './CertificateUpload.vue'

const loading = ref(false)
const submitting = ref(false)

// 对话框状态
const applyDialogVisible = ref(false)
const importDialogVisible = ref(false)
const editDialogVisible = ref(false)
const detailDialogVisible = ref(false)
const downloadDialogVisible = ref(false)

// 表单引用
const applyFormRef = ref<FormInstance>()
const importFormRef = ref<FormInstance>()
const editFormRef = ref<FormInstance>()

// 搜索表单
const searchForm = reactive({
  domain: '',
  status: '',
  source_type: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 统计数据
const stats = ref<Record<string, number>>({})

// 表格数据
const tableData = ref<any[]>([])

// DNS Providers
const dnsProviders = ref<any[]>([])

// Cloud Accounts
const cloudAccounts = ref<any[]>([])
const cloudAccountsLoading = ref(false)
// 当前查看的证书
const currentCert = ref<any>(null)

// 申请表单
const applyForm = reactive({
  name: '',
  domain: '',
  san_domains: [] as string[],
  acme_email: '',
  source_type: 'acme',
  ca_provider: 'letsencrypt',
  key_algorithm: 'ec256',
  dns_provider_id: null as number | null,
  cloud_account_id: null as number | null,
  auto_renew: true,
  renew_days_before: 30
})

// 导入表单
const importForm = reactive({
  name: '',
  domain: '',
  san_domains: [] as string[],
  certificate: '',
  private_key: '',
  cert_chain: ''
})

// 编辑表单
const editForm = reactive({
  id: 0,
  name: '',
  auto_renew: true,
  renew_days_before: 30,
  dns_provider_id: null as number | null,
  acme_email: '',
  source_type: ''
})

// 下载内容
const downloadFormat = ref('pem')
const downloadCertDomain = ref('')
const downloadingZip = ref(false)
const downloadContent = reactive({
  certificate: '',
  private_key: '',
  cert_chain: '',
  ssl_certificate: '',
  ssl_certificate_key: ''
})

// 表单验证规则
const applyRules = {
  name: [{ required: true, message: '请输入证书名称' }],
  domain: [{ required: true, message: '请输入主域名' }],
  acme_email: [{ type: 'email' as const, message: '请输入有效的邮箱地址' }]
}

// 获取来源类型名称
const getSourceTypeName = (type: string) => {
  const names: Record<string, string> = {
    acme: 'ACME免费证书',
    letsencrypt: "Let's Encrypt",
    aliyun: '阿里云CAS',
    manual: '手动导入'
  }
  return names[type] || type
}

// 获取CA提供商名称
const getCAProviderName = (provider: string) => {
  const names: Record<string, string> = {
    letsencrypt: "Let's Encrypt",
    zerossl: 'ZeroSSL',
    google: 'Google Trust Services',
    buypass: 'BuyPass',
    aliyun: '阿里云CAS'
  }
  return names[provider] || provider || '-'
}
// 获取密钥算法名称
const getKeyAlgorithmName = (algorithm: string) => {
  const names: Record<string, string> = {
    rsa2048: 'RSA 2048', rsa3072: 'RSA 3072', rsa4096: 'RSA 4096',
    ec256: 'EC P-256', ec384: 'EC P-384'
  }
  return names[algorithm] || algorithm || '-'
}

// 格式化错误信息
const formatErrorMessage = (error: string) => {
  if (!error) return ''
  if (error.includes('InsufficientQuota')) {
    return '阿里云证书额度不足\n\n可能原因：\n1. 账号欠费\n2. 免费证书额度已用完（每年20张）\n3. 未领取免费证书资源包\n\n解决方案：检查账号余额，或使用 ACME 免费证书'
  }
  if (error.includes('InvalidDomain')) {
    return '域名验证失败\n\n请检查域名是否已实名认证、解析是否正常'
  }
  return error
}

// 格式化时间
const formatDateTime = (dateTime: string | null | undefined) => {
  if (!dateTime) return '-'
  return String(dateTime).replace('T', ' ').split('+')[0].split('.')[0]
}

// 获取到期天数
const getExpiryDays = (notAfter: string) => {
  if (!notAfter) return ''
  const days = Math.ceil((new Date(notAfter).getTime() - Date.now()) / 86400000)
  if (days < 0) return `已过期 ${Math.abs(days)} 天`
  if (days === 0) return '今天到期'
  return `剩余 ${days} 天`
}

// 获取到期样式
const getExpiryClass = (notAfter: string) => {
  if (!notAfter) return ''
  const days = Math.ceil((new Date(notAfter).getTime() - Date.now()) / 86400000)
  if (days < 0) return 'expiry-expired'
  if (days <= 7) return 'expiry-danger'
  if (days <= 30) return 'expiry-warning'
  return 'expiry-normal'
}
// 重置搜索
const handleReset = () => {
  searchForm.domain = ''
  searchForm.status = ''
  searchForm.source_type = ''
  pagination.page = 1
  loadData()
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const [certsRes, statsRes] = await Promise.all([
      getCertificates({
        page: pagination.page,
        page_size: pagination.pageSize,
        domain: searchForm.domain || undefined,
        status: searchForm.status || undefined,
        source_type: searchForm.source_type || undefined
      }),
      getCertificateStats()
    ])
    tableData.value = certsRes.list || []
    pagination.total = certsRes.total || 0
    stats.value = statsRes || {}
    if (!stats.value.total) {
      stats.value.total = Object.values(stats.value).reduce((sum: number, val: any) => sum + (Number(val) || 0), 0)
    }
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    loading.value = false
  }
}

// 加载DNS Providers
const loadDNSProviders = async () => {
  try {
    const res = await getAllDNSProviders()
    dnsProviders.value = res || []
  } catch (error) { /* ignore */ }
}

// 加载云账号列表
const loadCloudAccounts = async (provider?: string) => {
  cloudAccountsLoading.value = true
  try {
    const res = await getCloudAccounts(provider)
    cloudAccounts.value = res || []
  } catch (error) {
    cloudAccounts.value = []
  } finally {
    cloudAccountsLoading.value = false
  }
}

// 监听证书类型变化
const handleSourceTypeChange = (val: string) => {
  applyForm.cloud_account_id = null
  if (val === 'aliyun') loadCloudAccounts(val)
}
// 处理创建
const handleCreate = (command: string) => {
  if (command === 'apply') {
    Object.assign(applyForm, {
      name: '', domain: '', san_domains: [], acme_email: '',
      source_type: 'acme', ca_provider: 'letsencrypt', key_algorithm: 'ec256',
      dns_provider_id: null, cloud_account_id: null, auto_renew: true, renew_days_before: 30
    })
    cloudAccounts.value = []
    applyDialogVisible.value = true
  } else if (command === 'import') {
    Object.assign(importForm, { name: '', domain: '', san_domains: [], certificate: '', private_key: '', cert_chain: '' })
    importDialogVisible.value = true
  }
}

// 申请证书提交
const handleApplySubmit = async () => {
  if (!applyFormRef.value) return
  const errors = await applyFormRef.value.validate()
  if (errors) return

  if (applyForm.source_type === 'aliyun' && !applyForm.cloud_account_id) {
    Message.warning('请选择云账号'); return
  }
  if (applyForm.source_type === 'acme') {
    if (!applyForm.acme_email) { Message.warning('请输入邮箱地址'); return }
    if (!applyForm.dns_provider_id) { Message.warning('请选择DNS验证配置'); return }
  }

  submitting.value = true
  try {
    const data: any = {
      name: applyForm.name, domain: applyForm.domain, san_domains: applyForm.san_domains,
      source_type: applyForm.source_type, key_algorithm: applyForm.key_algorithm,
      auto_renew: applyForm.auto_renew, renew_days_before: applyForm.renew_days_before
    }
    if (applyForm.source_type === 'acme') {
      data.acme_email = applyForm.acme_email
      data.ca_provider = applyForm.ca_provider
      data.dns_provider_id = applyForm.dns_provider_id!
    }
    if (applyForm.source_type === 'aliyun') {
      data.cloud_account_id = applyForm.cloud_account_id!
    }
    await createCertificate(data)
    Message.success('证书申请已提交，正在后台处理')
    applyDialogVisible.value = false
    loadData()
  } catch (error: any) {
    // 错误已由 request 拦截器处理
  } finally {
    submitting.value = false
  }
}
// 处理证书上传
const handleCertificateUploaded = async (certData: any) => {
  submitting.value = true
  try {
    await importCertificate({
      name: certData.name, domain: certData.domain,
      certificate: certData.certificate, private_key: certData.privateKey, cert_chain: ''
    })
    Message.success('证书导入成功')
    importDialogVisible.value = false
    loadData()
  } catch (error: any) { /* 错误已由 request 拦截器处理 */ }
  finally { submitting.value = false }
}

// 查看详情
const handleView = async (row: any) => {
  try {
    currentCert.value = await getCertificate(row.id)
    detailDialogVisible.value = true
  } catch (error) { /* 错误已由 request 拦截器处理 */ }
}

// 编辑
const handleEdit = async (row: any) => {
  try {
    const cert = await getCertificate(row.id)
    Object.assign(editForm, {
      id: cert.id, name: cert.name, auto_renew: cert.auto_renew,
      renew_days_before: cert.renew_days_before || 30,
      dns_provider_id: cert.dns_provider_id || null,
      acme_email: cert.acme_email || '', source_type: cert.source_type
    })
  } catch (error) {
    Object.assign(editForm, {
      id: row.id, name: row.name, auto_renew: row.auto_renew,
      renew_days_before: row.renew_days_before || 30,
      dns_provider_id: row.dns_provider_id || null,
      acme_email: row.acme_email || '', source_type: row.source_type
    })
  }
  editDialogVisible.value = true
}

// 编辑提交
const handleEditSubmit = async () => {
  submitting.value = true
  try {
    await updateCertificate(editForm.id, {
      name: editForm.name, auto_renew: editForm.auto_renew,
      renew_days_before: editForm.renew_days_before,
      dns_provider_id: editForm.dns_provider_id || undefined,
      acme_email: editForm.acme_email || undefined
    })
    Message.success('保存成功')
    editDialogVisible.value = false
    loadData()
  } catch (error: any) { /* 错误已由 request 拦截器处理 */ }
  finally { submitting.value = false }
}
// 自动续期切换
const handleAutoRenewChange = async (row: any) => {
  try {
    await updateCertificate(row.id, { auto_renew: row.auto_renew })
    Message.success('更新成功')
  } catch (error) { row.auto_renew = !row.auto_renew }
}

// 同步云证书状态
const handleSync = async (row: any) => {
  loading.value = true
  try {
    await syncCertificate(row.id)
    Message.success('证书同步成功')
    loadData()
  } catch (error: any) { /* 错误已由 request 拦截器处理 */ }
  finally { loading.value = false }
}

// 手动续期证书
const handleRenew = async (row: any) => {
  Modal.warning({
    title: '手动续期',
    content: `确定要手动续期证书 "${row.name}" 吗？续期需要配置DNS Provider和ACME邮箱。`,
    hideCancel: false,
    onOk: async () => {
      loading.value = true
      try {
        await renewCertificate(row.id)
        Message.success('续期任务已提交，请在任务记录中查看进度')
        loadData()
      } catch (error: any) { /* 错误已由 request 拦截器处理 */ }
      finally { loading.value = false }
    }
  })
}

// 下载证书
const handleDownload = async (row: any) => {
  try {
    downloadCertDomain.value = row.domain
    const [pemRes, nginxRes] = await Promise.all([
      downloadCertificate(row.id, 'pem'),
      downloadCertificate(row.id, 'nginx')
    ])
    downloadContent.certificate = pemRes.certificate || ''
    downloadContent.private_key = pemRes.private_key || ''
    downloadContent.cert_chain = pemRes.cert_chain || ''
    downloadContent.ssl_certificate = nginxRes.ssl_certificate || ''
    downloadContent.ssl_certificate_key = nginxRes.ssl_certificate_key || ''
    downloadDialogVisible.value = true
  } catch (error) { /* 错误已由 request 拦截器处理 */ }
}
// 下载单个文件
const downloadFile = (content: string, filename: string) => {
  if (!content) { Message.warning('文件内容为空'); return }
  const blob = new Blob([content], { type: 'application/x-pem-file' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  Message.success(`已下载 ${filename}`)
}

// 下载全部文件为ZIP压缩包
const downloadAllAsZip = async () => {
  downloadingZip.value = true
  try {
    const zip = new JSZip()
    const domain = downloadCertDomain.value || 'certificate'
    if (downloadFormat.value === 'pem') {
      if (downloadContent.certificate) zip.file(`${domain}.pem`, downloadContent.certificate)
      if (downloadContent.private_key) zip.file(`${domain}.key`, downloadContent.private_key)
      if (downloadContent.cert_chain) zip.file(`${domain}_ca.pem`, downloadContent.cert_chain)
    } else {
      if (downloadContent.ssl_certificate) zip.file(`${domain}_fullchain.pem`, downloadContent.ssl_certificate)
      if (downloadContent.ssl_certificate_key) zip.file(`${domain}.key`, downloadContent.ssl_certificate_key)
    }
    const content = await zip.generateAsync({ type: 'blob' })
    const url = URL.createObjectURL(content)
    const link = document.createElement('a')
    link.href = url
    link.download = `${domain}_ssl.zip`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    Message.success('证书已打包下载')
  } catch (error) { Message.error('打包下载失败') }
  finally { downloadingZip.value = false }
}

// 删除
const handleDelete = async (row: any) => {
  Modal.warning({
    title: '提示',
    content: '确定要删除该证书吗？关联的部署配置也将被删除。',
    hideCancel: false,
    onOk: async () => {
      loading.value = true
      try {
        await deleteCertificate(row.id)
        Message.success('删除成功')
        loadData()
      } catch (error: any) { /* 错误已由 request 拦截器处理 */ }
      finally { loading.value = false }
    }
  })
}

// 复制到剪贴板
const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text).then(() => {
    Message.success('已复制到剪贴板')
  }).catch(() => { Message.error('复制失败') })
}

onMounted(() => {
  loadData()
  loadDNSProviders()
})
</script>
<style scoped>
.certificate-container { padding: 0; background-color: transparent; }

.page-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  margin-bottom: 12px; padding: 16px 20px; background: #fff;
  border-radius: var(--ops-border-radius-md, 8px); box-shadow: 0 2px 12px rgba(0,0,0,0.04);
}
.page-title-group { display: flex; align-items: flex-start; gap: 16px; }
.page-title-icon {
  width: 36px; height: 36px; background: var(--ops-primary, #165dff);
  border-radius: 8px; display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 18px; flex-shrink: 0;
}
.page-title { margin: 0; font-size: 20px; font-weight: 600; color: var(--ops-text-primary, #1d2129); }
.page-subtitle { margin: 4px 0 0 0; font-size: 13px; color: var(--ops-text-tertiary, #86909c); }
.header-actions { display: flex; gap: 12px; }

.search-bar {
  margin-bottom: 12px; padding: 12px 16px; background: #fff;
  border-radius: var(--ops-border-radius-md, 8px); box-shadow: 0 2px 12px rgba(0,0,0,0.04);
  display: flex; justify-content: space-between; align-items: center; gap: 16px;
}
.search-inputs { display: flex; gap: 12px; flex: 1; }
.search-input { width: 200px; }
.search-actions { display: flex; gap: 10px; }
.stats-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 12px; }
.stat-card {
  background: #fff; border-radius: var(--ops-border-radius-md, 8px); padding: 20px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.04); display: flex; align-items: center; gap: 16px;
}
.stat-icon {
  width: 56px; height: 56px; border-radius: 12px; display: flex;
  align-items: center; justify-content: center; font-size: 24px; flex-shrink: 0;
}
.stat-icon-primary { background: var(--ops-primary-bg, #e8f0ff); color: var(--ops-primary, #165dff); }
.stat-icon-success { background: #e8ffea; color: var(--ops-success, #00b42a); }
.stat-icon-warning { background: #fff7e8; color: var(--ops-warning, #ff7d00); }
.stat-icon-danger { background: #ffece8; color: var(--ops-danger, #f53f3f); }
.stat-content { flex: 1; }
.stat-label { font-size: 14px; color: var(--ops-text-tertiary, #86909c); margin-bottom: 4px; }
.stat-value { font-size: 28px; font-weight: 600; color: var(--ops-text-primary, #1d2129); }

.table-wrapper {
  background: #fff; border-radius: var(--ops-border-radius-md, 8px);
  box-shadow: 0 2px 12px rgba(0,0,0,0.04); overflow: hidden; padding: 16px;
}
.cert-name { font-weight: 500; }
.domain-cell { display: flex; align-items: center; }
.domain-main { font-family: 'Monaco', 'Menlo', monospace; }
.expiry-days { font-size: 12px; color: var(--ops-text-tertiary, #86909c); }
.expiry-normal { color: var(--ops-success, #00b42a); }
.expiry-warning { color: var(--ops-warning, #ff7d00); }
.expiry-danger { color: var(--ops-danger, #f53f3f); }
.expiry-expired { color: var(--ops-text-tertiary, #86909c); text-decoration: line-through; }
.action-buttons { display: flex; gap: 4px; align-items: center; justify-content: center; }
.action-btn { width: 32px; height: 32px; border-radius: 6px; transition: all 0.2s ease; }
.action-btn:hover { transform: scale(1.1); }
.action-view:hover { background-color: var(--ops-primary-bg, #e8f0ff); color: var(--ops-primary, #165dff); }
.action-download:hover { background-color: #e8ffea; color: var(--ops-success, #00b42a); }
.action-sync:hover { background-color: #e8ffea; color: var(--ops-success, #00b42a); }
.action-renew:hover { background-color: #fff7e8; color: var(--ops-warning, #ff7d00); }
.action-edit:hover { background-color: var(--ops-primary-bg, #e8f0ff); color: var(--ops-primary, #165dff); }
.action-delete:hover { background-color: #ffece8; color: var(--ops-danger, #f53f3f); }

.form-tip { font-size: 12px; color: var(--ops-text-tertiary, #86909c); margin-top: 6px; line-height: 1.5; }
.dialog-scroll-body { max-height: 65vh; overflow-y: auto; }

/* 详情弹窗 */
.detail-content { padding: 0; }
.detail-status-bar {
  display: flex; align-items: center; gap: 12px; margin-bottom: 20px;
  padding: 16px; background: var(--ops-content-bg, #f7f8fa);
  border-radius: 10px; border: 1px solid var(--ops-border-color, #e5e6eb);
}
.detail-domain { font-size: 18px; font-weight: 600; color: var(--ops-text-primary, #1d2129); font-family: 'Monaco', 'Menlo', monospace; }
.detail-info { display: flex; flex-direction: column; gap: 16px; }
.detail-info-section { background: #fff; border: 1px solid var(--ops-border-color, #e5e6eb); border-radius: 10px; overflow: hidden; }
.detail-section-title {
  padding: 10px 16px; font-size: 13px; font-weight: 600;
  color: var(--ops-text-tertiary, #86909c); text-transform: uppercase;
  letter-spacing: 0.5px; background: var(--ops-content-bg, #f7f8fa);
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}
.error-section-title { color: var(--ops-danger, #f53f3f); }
.detail-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0; }
.detail-grid .info-item {
  display: flex; flex-direction: column; gap: 4px; padding: 12px 16px;
  border-bottom: 1px solid #f2f3f5; border-right: 1px solid #f2f3f5;
}
.detail-grid .info-item:nth-child(2n) { border-right: none; }
.detail-grid .info-item:last-child,
.detail-grid .info-item:nth-last-child(2):nth-child(odd) { border-bottom: none; }
.detail-grid .info-item.full-width { grid-column: span 2; border-right: none; }
.info-label { color: var(--ops-text-tertiary, #86909c); font-size: 12px; font-weight: 500; }
.info-value { color: var(--ops-text-primary, #1d2129); font-size: 14px; word-break: break-all; font-weight: 500; }
.fingerprint { font-family: 'Monaco', 'Menlo', monospace; font-size: 12px; }
.expiry-days-inline { font-size: 12px; margin-left: 8px; font-weight: 400; color: var(--ops-text-tertiary, #86909c); }
.error-block { padding: 14px 16px; font-size: 13px; color: var(--ops-danger, #f53f3f); white-space: pre-wrap; word-break: break-word; line-height: 1.6; }

/* 下载弹窗 */
.download-card { background: #fff; border-radius: 12px; }
.download-card-header { padding: 16px 0; border-bottom: 1px solid var(--ops-border-color, #e5e6eb); margin-bottom: 20px; }
.download-domain-info { display: flex; align-items: center; gap: 12px; }
.domain-icon {
  width: 48px; height: 48px; background: var(--ops-primary-bg, #e8f0ff);
  border-radius: 10px; display: flex; align-items: center; justify-content: center;
  color: var(--ops-primary, #165dff); font-size: 22px;
}
.domain-text { display: flex; flex-direction: column; gap: 4px; }
.domain-name { font-size: 16px; font-weight: 600; color: var(--ops-text-primary, #1d2129); font-family: 'Monaco', 'Menlo', monospace; }
.domain-hint { font-size: 13px; color: var(--ops-text-tertiary, #86909c); }
.download-format-selector { display: flex; gap: 12px; margin-bottom: 20px; }
.format-option {
  flex: 1; display: flex; align-items: center; gap: 12px; padding: 16px;
  border: 2px solid var(--ops-border-color, #e5e6eb); border-radius: 10px;
  cursor: pointer; transition: all 0.2s ease;
}
.format-option:hover { border-color: var(--ops-primary-light, #306fff); background: var(--ops-content-bg, #f7f8fa); }
.format-option.active { border-color: var(--ops-primary, #165dff); background: var(--ops-primary-bg, #e8f0ff); }
.format-icon {
  width: 40px; height: 40px; background: var(--ops-content-bg, #f7f8fa);
  border-radius: 8px; display: flex; align-items: center; justify-content: center;
  font-size: 18px; color: var(--ops-text-secondary, #4e5969);
}
.format-option.active .format-icon { background: var(--ops-primary, #165dff); color: #fff; }
.format-info { display: flex; flex-direction: column; gap: 2px; }
.format-name { font-size: 14px; font-weight: 600; color: var(--ops-text-primary, #1d2129); }
.format-desc { font-size: 12px; color: var(--ops-text-tertiary, #86909c); }

.download-files {
  display: flex; flex-direction: column; gap: 10px; margin-bottom: 24px;
  padding: 16px; background: var(--ops-content-bg, #f7f8fa);
  border-radius: 10px; border: 1px solid var(--ops-border-color, #e5e6eb);
}
.file-item {
  display: flex; align-items: center; gap: 12px; padding: 12px;
  background: #fff; border-radius: 8px; border: 1px solid var(--ops-border-color, #e5e6eb);
  transition: all 0.2s ease;
}
.file-item:hover { border-color: var(--ops-primary-light, #306fff); box-shadow: 0 2px 8px rgba(0,0,0,0.04); }
.file-icon { width: 36px; height: 36px; border-radius: 8px; display: flex; align-items: center; justify-content: center; font-size: 16px; }
.cert-icon { background: var(--ops-primary-bg, #e8f0ff); color: var(--ops-primary, #165dff); }
.key-icon { background: #fff7e8; color: var(--ops-warning, #ff7d00); }
.file-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.file-name { font-size: 14px; font-weight: 500; color: var(--ops-text-primary, #1d2129); font-family: 'Monaco', 'Menlo', monospace; }
.file-size { font-size: 12px; color: var(--ops-text-tertiary, #86909c); }
.file-actions { display: flex; gap: 4px; }
.download-action { display: flex; justify-content: center; padding-top: 8px; }
</style>