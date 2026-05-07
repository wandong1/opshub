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
          <a-table-column title="操作" :width="220">
            <template #cell="{ record }">
              <a-space>
                <a-link @click="doTest(record)">测试</a-link>
                <a-link @click="openCopy(record)">复制</a-link>
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
      @ok="save" @cancel="modalVisible=false" width="780px">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="通道名称" required><a-input v-model="form.name" /></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="通道类型" required>
              <a-select v-model="form.type" @change="onTypeChange">
                <a-option value="email">邮件</a-option>
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
        <template v-if="form.type === 'email'">
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="SMTP服务器" required>
                <a-input v-model="cfg.smtpHost" placeholder="smtp.example.com" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="SMTP端口" required>
                <a-input-number v-model="cfg.smtpPort" :min="1" :max="65535" :default-value="465" style="width: 100%;" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="发件人邮箱" required>
                <a-input v-model="cfg.fromEmail" placeholder="noreply@example.com" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="发件人名称">
                <a-input v-model="cfg.fromName" placeholder="OpsHub告警" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="SMTP用户名" required>
                <a-input v-model="cfg.smtpUser" placeholder="user@example.com" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="SMTP密码" required>
                <a-input-password v-model="cfg.smtpPassword" placeholder="请输入SMTP密码" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-alert type="info" style="margin-bottom: 16px;">
            <template #icon><icon-info-circle /></template>
            端口 465 使用直接 TLS 加密，端口 587 使用 STARTTLS。Gmail 需要使用"应用专用密码"。
          </a-alert>
        </template>
        <template v-else-if="form.type === 'wechat_work'">
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
          <div style="margin-top:4px;color:var(--ops-text-secondary);font-size:12px">
            💡 提示：恢复通知应使用 <code v-text="'{{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}}'"></code> 显示恢复时的实际值
            <a-textarea v-model="form.resolveTemplate" :auto-size="{minRows:4}" style="font-family:monospace;font-size:12px" />
          </div>
          
          
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
const cfg = reactive<Record<string, any>>({
  webhookUrl: '',
  secret: '',
  appKey: '',
  appSecret: '',
  provider: 'aliyun',
  templateId: '',
  hookUrl: '',
  smtpHost: '',
  smtpPort: 465,
  fromEmail: '',
  fromName: '',
  smtpUser: '',
  smtpPassword: ''
})

const typeLabel = (t: string) => ({ email: '邮件', wechat_work: '企业微信', dingtalk: '钉钉', sms: '短信', phone: '电话', ai_agent: 'AI智能体' }[t] || t)
const typeColor = (t: string) => ({ email: 'blue', wechat_work: 'green', dingtalk: 'orange', sms: 'blue', phone: 'purple', ai_agent: 'arcoblue' }[t] || 'gray')

const defaultAlertTpl = (type: string) => {
  if (type === 'email') return `<!DOCTYPE html>\n<html lang="zh-CN">\n<head>\n    <meta charset="UTF-8">\n    <meta name="viewport" content="width=device-width, initial-scale=1.0">\n    <title>告警通知</title>\n    <style>\n        * { margin: 0; padding: 0; box-sizing: border-box; }\n        body {\n            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;\n            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);\n            padding: 40px 20px;\n            line-height: 1.6;\n        }\n        .email-container {\n            max-width: 600px;\n            margin: 0 auto;\n            background: #ffffff;\n            border-radius: 16px;\n            overflow: hidden;\n            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);\n        }\n        .header {\n            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);\n            padding: 40px 30px;\n            text-align: center;\n            position: relative;\n        }\n        .header::before {\n            content: '';\n            position: absolute;\n            top: 0;\n            left: 0;\n            right: 0;\n            bottom: 0;\n            background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1440 320"><path fill="%23ffffff" fill-opacity="0.1" d="M0,96L48,112C96,128,192,160,288,160C384,160,480,128,576,122.7C672,117,768,139,864,138.7C960,139,1056,117,1152,106.7C1248,96,1344,96,1392,96L1440,96L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z"></path></svg>') no-repeat bottom;\n            background-size: cover;\n            opacity: 0.3;\n        }\n        .alert-icon {\n            width: 80px;\n            height: 80px;\n            margin: 0 auto 20px;\n            background: rgba(255, 255, 255, 0.2);\n            border-radius: 50%;\n            display: flex;\n            align-items: center;\n            justify-content: center;\n            font-size: 40px;\n            backdrop-filter: blur(10px);\n            border: 3px solid rgba(255, 255, 255, 0.3);\n            position: relative;\n            z-index: 1;\n        }\n        .header h1 {\n            color: #ffffff;\n            font-size: 28px;\n            font-weight: 700;\n            margin-bottom: 10px;\n            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);\n            position: relative;\n            z-index: 1;\n        }\n        .header .subtitle {\n            color: rgba(255, 255, 255, 0.9);\n            font-size: 14px;\n            font-weight: 500;\n            position: relative;\n            z-index: 1;\n        }\n        .content {\n            padding: 40px 30px;\n        }\n        .alert-badge {\n            display: inline-block;\n            padding: 8px 20px;\n            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);\n            color: #ffffff;\n            border-radius: 20px;\n            font-size: 14px;\n            font-weight: 600;\n            margin-bottom: 25px;\n            box-shadow: 0 4px 15px rgba(245, 87, 108, 0.3);\n        }\n        .rule-name {\n            font-size: 24px;\n            font-weight: 700;\n            color: #1a1a1a;\n            margin-bottom: 30px;\n            padding-bottom: 20px;\n            border-bottom: 2px solid #f0f0f0;\n        }\n        .info-grid {\n            display: grid;\n            grid-template-columns: repeat(2, 1fr);\n            gap: 20px;\n            margin-bottom: 30px;\n        }\n        .info-card {\n            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);\n            padding: 20px;\n            border-radius: 12px;\n            border-left: 4px solid #f5576c;\n            transition: transform 0.2s;\n        }\n        .info-card:hover {\n            transform: translateY(-2px);\n        }\n        .info-label {\n            font-size: 12px;\n            color: #666;\n            font-weight: 600;\n            text-transform: uppercase;\n            letter-spacing: 0.5px;\n            margin-bottom: 8px;\n        }\n        .info-value {\n            font-size: 16px;\n            color: #1a1a1a;\n            font-weight: 600;\n        }\n        .severity-critical { border-left-color: #f5576c; }\n        .severity-warning { border-left-color: #ffa726; }\n        .severity-info { border-left-color: #42a5f5; }\n        .details-section {\n            background: #f8f9fa;\n            padding: 25px;\n            border-radius: 12px;\n            margin-top: 30px;\n        }\n        .details-title {\n            font-size: 16px;\n            font-weight: 700;\n            color: #1a1a1a;\n            margin-bottom: 15px;\n            display: flex;\n            align-items: center;\n        }\n        .details-title::before {\n            content: '📋';\n            margin-right: 10px;\n            font-size: 20px;\n        }\n        .details-content {\n            background: #ffffff;\n            padding: 15px;\n            border-radius: 8px;\n            font-size: 14px;\n            color: #333;\n            line-height: 1.8;\n            white-space: pre-wrap;\n            word-wrap: break-word;\n        }\n        .footer {\n            background: #f8f9fa;\n            padding: 30px;\n            text-align: center;\n            border-top: 1px solid #e0e0e0;\n        }\n        .footer-logo {\n            font-size: 24px;\n            font-weight: 700;\n            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);\n            -webkit-background-clip: text;\n            -webkit-text-fill-color: transparent;\n            margin-bottom: 10px;\n        }\n        .footer-text {\n            font-size: 13px;\n            color: #999;\n            margin-bottom: 15px;\n        }\n        .footer-links {\n            margin-top: 15px;\n        }\n        .footer-link {\n            color: #667eea;\n            text-decoration: none;\n            margin: 0 10px;\n            font-size: 13px;\n            font-weight: 500;\n        }\n        @media only screen and (max-width: 600px) {\n            .info-grid {\n                grid-template-columns: 1fr;\n            }\n            .header h1 {\n                font-size: 24px;\n            }\n            .content {\n                padding: 30px 20px;\n            }\n        }\n    </style>\n</head>\n<body>\n    <div class="email-container">\n        <div class="header">\n            <div class="alert-icon">🚨</div>\n            <h1>告警通知</h1>\n            <div class="subtitle">SreHub 智能巡检平台</div>\n        </div>\n        \n        <div class="content">\n            <div class="alert-badge">🔴 告警触发</div>\n            \n            <div class="rule-name">{{.RuleName}}</div>\n            \n            <div class="info-grid">\n                <div class="info-card severity-critical">\n                    <div class="info-label">告警级别</div>\n                    <div class="info-value">{{.SeverityLabel}}</div>\n                </div>\n                \n                <div class="info-card">\n                    <div class="info-label">当前值</div>\n                    <div class="info-value">{{.Value}}</div>\n                </div>\n                \n                <div class="info-card">\n                    <div class="info-label">触发时间</div>\n                    <div class="info-value">{{.FiredAt}}</div>\n                </div>\n                \n                <div class="info-card">\n                    <div class="info-label">持续时长</div>\n                    <div class="info-value">{{if .Duration}}{{.Duration}}{{else}}刚刚触发{{end}}</div>\n                </div>\n            </div>\n            \n            <div class="details-section">\n                <div class="details-title">标签详情</div>\n                <div class="details-content">{{.LabelsDetail}}</div>\n            </div>\n            \n            <div class="details-section">\n                <div class="details-title">注解详情</div>\n                <div class="details-content">{{.AnnotationsDetail}}</div>\n            </div>\n        </div>\n        \n        <div class="footer">\n            <div class="footer-logo">SreHub</div>\n            <div class="footer-text">此邮件由 SreHub 智能巡检平台自动发送，请勿直接回复</div>\n            <div class="footer-links">\n                <a href="#" class="footer-link">查看详情</a>\n                <a href="#" class="footer-link">告警历史</a>\n                <a href="#" class="footer-link">帮助文档</a>\n            </div>\n        </div>\n    </div>\n</body>\n</html>`
  if (type === 'wechat_work') return `## 🔴 SreHub 告警通知\n> **规则**: {{.RuleName}}\n> **级别**: {{.SeverityLabel}}\n> **当前值**: {{.Value}}\n> **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  if (type === 'dingtalk') return `## 🔴 SreHub 告警通知\n- **规则**: {{.RuleName}}\n- **级别**: {{.SeverityLabel}}\n- **当前值**: {{.Value}}\n- **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  return `【SreHub告警】规则: {{.RuleName}} | 级别: {{.SeverityLabel}} | 值: {{.Value}} | 时间: {{.FiredAt}}`
}
const defaultResolveTpl = (type = '') => {
  if (type === 'email') return `<!DOCTYPE html>\n<html lang="zh-CN">\n<head>\n    <meta charset="UTF-8">\n    <meta name="viewport" content="width=device-width, initial-scale=1.0">\n    <title>恢复通知</title>\n    <style>\n        * { margin: 0; padding: 0; box-sizing: border-box; }\n        body {\n            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;\n            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);\n            padding: 40px 20px;\n            line-height: 1.6;\n        }\n        .email-container {\n            max-width: 600px;\n            margin: 0 auto;\n            background: #ffffff;\n            border-radius: 16px;\n            overflow: hidden;\n            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);\n        }\n        .header {\n            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);\n            padding: 40px 30px;\n            text-align: center;\n            position: relative;\n        }\n        .header::before {\n            content: '';\n            position: absolute;\n            top: 0;\n            left: 0;\n            right: 0;\n            bottom: 0;\n            background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1440 320"><path fill="%23ffffff" fill-opacity="0.1" d="M0,96L48,112C96,128,192,160,288,160C384,160,480,128,576,122.7C672,117,768,139,864,138.7C960,139,1056,117,1152,106.7C1248,96,1344,96,1392,96L1440,96L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z"></path></svg>') no-repeat bottom;\n            background-size: cover;\n            opacity: 0.3;\n        }\n        .success-icon {\n            width: 80px;\n            height: 80px;\n            margin: 0 auto 20px;\n            background: rgba(255, 255, 255, 0.2);\n            border-radius: 50%;\n            display: flex;\n            align-items: center;\n            justify-content: center;\n            font-size: 40px;\n            backdrop-filter: blur(10px);\n            border: 3px solid rgba(255, 255, 255, 0.3);\n            position: relative;\n            z-index: 1;\n            animation: pulse 2s infinite;\n        }\n        @keyframes pulse {\n            0%, 100% { transform: scale(1); }\n            50% { transform: scale(1.05); }\n        }\n        .header h1 {\n            color: #ffffff;\n            font-size: 28px;\n            font-weight: 700;\n            margin-bottom: 10px;\n            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);\n            position: relative;\n            z-index: 1;\n        }\n        .header .subtitle {\n            color: rgba(255, 255, 255, 0.9);\n            font-size: 14px;\n            font-weight: 500;\n            position: relative;\n            z-index: 1;\n        }\n        .content {\n            padding: 40px 30px;\n        }\n        .success-badge {\n            display: inline-block;\n            padding: 8px 20px;\n            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);\n            color: #ffffff;\n            border-radius: 20px;\n            font-size: 14px;\n            font-weight: 600;\n            margin-bottom: 25px;\n            box-shadow: 0 4px 15px rgba(56, 239, 125, 0.3);\n        }\n        .rule-name {\n            font-size: 24px;\n            font-weight: 700;\n            color: #1a1a1a;\n            margin-bottom: 30px;\n            padding-bottom: 20px;\n            border-bottom: 2px solid #f0f0f0;\n        }\n        .info-grid {\n            display: grid;\n            grid-template-columns: repeat(2, 1fr);\n            gap: 20px;\n            margin-bottom: 30px;\n        }\n        .info-card {\n            background: linear-gradient(135deg, #e0f7fa 0%, #b2ebf2 100%);\n            padding: 20px;\n            border-radius: 12px;\n            border-left: 4px solid #38ef7d;\n            transition: transform 0.2s;\n        }\n        .info-card:hover {\n            transform: translateY(-2px);\n        }\n        .info-label {\n            font-size: 12px;\n            color: #666;\n            font-weight: 600;\n            text-transform: uppercase;\n            letter-spacing: 0.5px;\n            margin-bottom: 8px;\n        }\n        .info-value {\n            font-size: 16px;\n            color: #1a1a1a;\n            font-weight: 600;\n        }\n        .timeline {\n            background: #f8f9fa;\n            padding: 25px;\n            border-radius: 12px;\n            margin-bottom: 30px;\n        }\n        .timeline-title {\n            font-size: 16px;\n            font-weight: 700;\n            color: #1a1a1a;\n            margin-bottom: 20px;\n            display: flex;\n            align-items: center;\n        }\n        .timeline-title::before {\n            content: '⏱️';\n            margin-right: 10px;\n            font-size: 20px;\n        }\n        .timeline-item {\n            display: flex;\n            align-items: center;\n            margin-bottom: 15px;\n            padding: 15px;\n            background: #ffffff;\n            border-radius: 8px;\n        }\n        .timeline-item:last-child {\n            margin-bottom: 0;\n        }\n        .timeline-dot {\n            width: 12px;\n            height: 12px;\n            border-radius: 50%;\n            margin-right: 15px;\n            flex-shrink: 0;\n        }\n        .timeline-dot.fired {\n            background: #f5576c;\n        }\n        .timeline-dot.resolved {\n            background: #38ef7d;\n        }\n        .timeline-content {\n            flex: 1;\n        }\n        .timeline-label {\n            font-size: 12px;\n            color: #999;\n            margin-bottom: 5px;\n        }\n        .timeline-value {\n            font-size: 14px;\n            color: #333;\n            font-weight: 600;\n        }\n        .details-section {\n            background: #f8f9fa;\n            padding: 25px;\n            border-radius: 12px;\n            margin-top: 30px;\n        }\n        .details-title {\n            font-size: 16px;\n            font-weight: 700;\n            color: #1a1a1a;\n            margin-bottom: 15px;\n            display: flex;\n            align-items: center;\n        }\n        .details-title::before {\n            content: '📋';\n            margin-right: 10px;\n            font-size: 20px;\n        }\n        .details-content {\n            background: #ffffff;\n            padding: 15px;\n            border-radius: 8px;\n            font-size: 14px;\n            color: #333;\n            line-height: 1.8;\n            white-space: pre-wrap;\n            word-wrap: break-word;\n        }\n        .footer {\n            background: #f8f9fa;\n            padding: 30px;\n            text-align: center;\n            border-top: 1px solid #e0e0e0;\n        }\n        .footer-logo {\n            font-size: 24px;\n            font-weight: 700;\n            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);\n            -webkit-background-clip: text;\n            -webkit-text-fill-color: transparent;\n            margin-bottom: 10px;\n        }\n        .footer-text {\n            font-size: 13px;\n            color: #999;\n            margin-bottom: 15px;\n        }\n        .footer-links {\n            margin-top: 15px;\n        }\n        .footer-link {\n            color: #11998e;\n            text-decoration: none;\n            margin: 0 10px;\n            font-size: 13px;\n            font-weight: 500;\n        }\n        @media only screen and (max-width: 600px) {\n            .info-grid {\n                grid-template-columns: 1fr;\n            }\n            .header h1 {\n                font-size: 24px;\n            }\n            .content {\n                padding: 30px 20px;\n            }\n        }\n    </style>\n</head>\n<body>\n    <div class="email-container">\n        <div class="header">\n            <div class="success-icon">✅</div>\n            <h1>恢复通知</h1>\n            <div class="subtitle">SreHub 智能巡检平台</div>\n        </div>\n        \n        <div class="content">\n            <div class="success-badge">✅ 告警已恢复</div>\n            \n            <div class="rule-name">{{.RuleName}}</div>\n            \n            <div class="info-grid">\n                <div class="info-card">\n                    <div class="info-label">告警级别</div>\n                    <div class="info-value">{{.SeverityLabel}}</div>\n                </div>\n                \n                <div class="info-card">\n                    <div class="info-label">当前值</div>\n                    <div class="info-value">{{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}}</div>\n                </div>\n                \n                <div class="info-card">\n                    <div class="info-label">恢复时间</div>\n                    <div class="info-value">{{.ResolvedAt}}</div>\n                </div>\n                \n                <div class="info-card">\n                    <div class="info-label">持续时长</div>\n                    <div class="info-value">{{if .Duration}}{{.Duration}}{{else}}-{{end}}</div>\n                </div>\n            </div>\n            \n            <div class="timeline">\n                <div class="timeline-title">时间线</div>\n                \n                <div class="timeline-item">\n                    <div class="timeline-dot fired"></div>\n                    <div class="timeline-content">\n                        <div class="timeline-label">告警触发</div>\n                        <div class="timeline-value">{{.FiredAt}}</div>\n                    </div>\n                </div>\n                \n                <div class="timeline-item">\n                    <div class="timeline-dot resolved"></div>\n                    <div class="timeline-content">\n                        <div class="timeline-label">告警恢复</div>\n                        <div class="timeline-value">{{.ResolvedAt}}</div>\n                    </div>\n                </div>\n            </div>\n            \n            <div class="details-section">\n                <div class="details-title">标签详情</div>\n                <div class="details-content">{{.LabelsDetail}}</div>\n            </div>\n            \n            <div class="details-section">\n                <div class="details-title">注解详情</div>\n                <div class="details-content">{{.AnnotationsDetail}}</div>\n            </div>\n        </div>\n        \n        <div class="footer">\n            <div class="footer-logo">SreHub</div>\n            <div class="footer-text">此邮件由 SreHub 智能巡检平台自动发送，请勿直接回复</div>\n            <div class="footer-links">\n                <a href="#" class="footer-link">查看详情</a>\n                <a href="#" class="footer-link">告警历史</a>\n                <a href="#" class="footer-link">帮助文档</a>\n            </div>\n        </div>\n    </div>\n</body>\n</html>\n`
  if (type === 'wechat_work') return `## ✅ SreHub 恢复通知\n> **规则**: {{.RuleName}}\n> **级别**: {{.SeverityLabel}}\n> **当前值**: {{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}}\n> **恢复时间**: {{.ResolvedAt}}\n> **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  if (type === 'dingtalk') return `## ✅ SreHub 恢复通知\n- **规则**: {{.RuleName}}\n- **级别**: {{.SeverityLabel}}\n- **当前值**: {{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}}\n- **恢复时间**: {{.ResolvedAt}}\n- **触发时间**: {{.FiredAt}}\n\n**标签详情**:\n{{.LabelsDetail}}\n\n**注解详情**:\n{{.AnnotationsDetail}}`
  return `【SreHub恢复】规则: {{.RuleName}} | 级别: {{.SeverityLabel}} | 值: {{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}} | 恢复时间: {{.ResolvedAt}}`
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
  form.value = { enabled: true, type: 'email', alertTemplate: defaultAlertTpl('email'), resolveTemplate: defaultResolveTpl('email') }
  Object.keys(cfg).forEach(k => cfg[k] = '')
  cfg.provider = 'aliyun'
  cfg.smtpPort = '465'
  modalVisible.value = true
}

const openEdit = (row: AlertNotifyChannel) => {
  form.value = { ...row }
  try { Object.assign(cfg, JSON.parse(row.config || '{}')) } catch {}
  modalVisible.value = true
}

const openCopy = (row: AlertNotifyChannel) => {
  // 复制所有配置，但移除 ID（作为新增处理）
  form.value = {
    ...row,
    id: undefined,
    name: row.name + '副本'
  }
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
