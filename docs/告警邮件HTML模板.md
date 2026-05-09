# 告警邮件 HTML 模板

## 模板说明

这两个模板采用现代化设计风格，支持深色/浅色主题，响应式布局，适配各种邮件客户端。

## 1. 告警通知模板（Alert Template）

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>告警通知</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 40px 20px;
            line-height: 1.6;
        }
        .email-container {
            max-width: 600px;
            margin: 0 auto;
            background: #ffffff;
            border-radius: 16px;
            overflow: hidden;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
        }
        .header {
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            padding: 40px 30px;
            text-align: center;
            position: relative;
        }
        .header::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1440 320"><path fill="%23ffffff" fill-opacity="0.1" d="M0,96L48,112C96,128,192,160,288,160C384,160,480,128,576,122.7C672,117,768,139,864,138.7C960,139,1056,117,1152,106.7C1248,96,1344,96,1392,96L1440,96L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z"></path></svg>') no-repeat bottom;
            background-size: cover;
            opacity: 0.3;
        }
        .alert-icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 20px;
            background: rgba(255, 255, 255, 0.2);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 40px;
            backdrop-filter: blur(10px);
            border: 3px solid rgba(255, 255, 255, 0.3);
            position: relative;
            z-index: 1;
        }
        .header h1 {
            color: #ffffff;
            font-size: 28px;
            font-weight: 700;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            position: relative;
            z-index: 1;
        }
        .header .subtitle {
            color: rgba(255, 255, 255, 0.9);
            font-size: 14px;
            font-weight: 500;
            position: relative;
            z-index: 1;
        }
        .content {
            padding: 40px 30px;
        }
        .alert-badge {
            display: inline-block;
            padding: 8px 20px;
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            color: #ffffff;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 25px;
            box-shadow: 0 4px 15px rgba(245, 87, 108, 0.3);
        }
        .rule-name {
            font-size: 24px;
            font-weight: 700;
            color: #1a1a1a;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 2px solid #f0f0f0;
        }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 20px;
            margin-bottom: 30px;
        }
        .info-card {
            background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
            padding: 20px;
            border-radius: 12px;
            border-left: 4px solid #f5576c;
            transition: transform 0.2s;
        }
        .info-card:hover {
            transform: translateY(-2px);
        }
        .info-label {
            font-size: 12px;
            color: #666;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }
        .info-value {
            font-size: 16px;
            color: #1a1a1a;
            font-weight: 600;
        }
        .severity-critical { border-left-color: #f5576c; }
        .severity-warning { border-left-color: #ffa726; }
        .severity-info { border-left-color: #42a5f5; }
        .details-section {
            background: #f8f9fa;
            padding: 25px;
            border-radius: 12px;
            margin-top: 30px;
        }
        .details-title {
            font-size: 16px;
            font-weight: 700;
            color: #1a1a1a;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
        }
        .details-title::before {
            content: '📋';
            margin-right: 10px;
            font-size: 20px;
        }
        .details-content {
            background: #ffffff;
            padding: 15px;
            border-radius: 8px;
            font-size: 14px;
            color: #333;
            line-height: 1.8;
            white-space: pre-wrap;
            word-wrap: break-word;
        }
        .footer {
            background: #f8f9fa;
            padding: 30px;
            text-align: center;
            border-top: 1px solid #e0e0e0;
        }
        .footer-logo {
            font-size: 24px;
            font-weight: 700;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 10px;
        }
        .footer-text {
            font-size: 13px;
            color: #999;
            margin-bottom: 15px;
        }
        .footer-links {
            margin-top: 15px;
        }
        .footer-link {
            color: #667eea;
            text-decoration: none;
            margin: 0 10px;
            font-size: 13px;
            font-weight: 500;
        }
        @media only screen and (max-width: 600px) {
            .info-grid {
                grid-template-columns: 1fr;
            }
            .header h1 {
                font-size: 24px;
            }
            .content {
                padding: 30px 20px;
            }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            <div class="alert-icon">🚨</div>
            <h1>告警通知</h1>
            <div class="subtitle">OpsHub 智能运维平台</div>
        </div>
        
        <div class="content">
            <div class="alert-badge">🔴 告警触发</div>
            
            <div class="rule-name">{{.RuleName}}</div>
            
            <div class="info-grid">
                <div class="info-card severity-critical">
                    <div class="info-label">告警级别</div>
                    <div class="info-value">{{.SeverityLabel}}</div>
                </div>
                
                <div class="info-card">
                    <div class="info-label">当前值</div>
                    <div class="info-value">{{.Value}}</div>
                </div>
                
                <div class="info-card">
                    <div class="info-label">触发时间</div>
                    <div class="info-value">{{.FiredAt}}</div>
                </div>
                
                <div class="info-card">
                    <div class="info-label">持续时长</div>
                    <div class="info-value">{{if .Duration}}{{.Duration}}{{else}}刚刚触发{{end}}</div>
                </div>
            </div>
            
            <div class="details-section">
                <div class="details-title">标签详情</div>
                <div class="details-content">{{.LabelsDetail}}</div>
            </div>
            
            <div class="details-section">
                <div class="details-title">注解详情</div>
                <div class="details-content">{{.AnnotationsDetail}}</div>
            </div>
        </div>
        
        <div class="footer">
            <div class="footer-logo">OpsHub</div>
            <div class="footer-text">此邮件由 OpsHub 智能运维平台自动发送，请勿直接回复</div>
            <div class="footer-links">
                <a href="#" class="footer-link">查看详情</a>
                <a href="#" class="footer-link">告警历史</a>
                <a href="#" class="footer-link">帮助文档</a>
            </div>
        </div>
    </div>
</body>
</html>
```

## 2. 恢复通知模板（Resolve Template）

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>恢复通知</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
            padding: 40px 20px;
            line-height: 1.6;
        }
        .email-container {
            max-width: 600px;
            margin: 0 auto;
            background: #ffffff;
            border-radius: 16px;
            overflow: hidden;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
        }
        .header {
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
            padding: 40px 30px;
            text-align: center;
            position: relative;
        }
        .header::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1440 320"><path fill="%23ffffff" fill-opacity="0.1" d="M0,96L48,112C96,128,192,160,288,160C384,160,480,128,576,122.7C672,117,768,139,864,138.7C960,139,1056,117,1152,106.7C1248,96,1344,96,1392,96L1440,96L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z"></path></svg>') no-repeat bottom;
            background-size: cover;
            opacity: 0.3;
        }
        .success-icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 20px;
            background: rgba(255, 255, 255, 0.2);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 40px;
            backdrop-filter: blur(10px);
            border: 3px solid rgba(255, 255, 255, 0.3);
            position: relative;
            z-index: 1;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.05); }
        }
        .header h1 {
            color: #ffffff;
            font-size: 28px;
            font-weight: 700;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            position: relative;
            z-index: 1;
        }
        .header .subtitle {
            color: rgba(255, 255, 255, 0.9);
            font-size: 14px;
            font-weight: 500;
            position: relative;
            z-index: 1;
        }
        .content {
            padding: 40px 30px;
        }
        .success-badge {
            display: inline-block;
            padding: 8px 20px;
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
            color: #ffffff;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 25px;
            box-shadow: 0 4px 15px rgba(56, 239, 125, 0.3);
        }
        .rule-name {
            font-size: 24px;
            font-weight: 700;
            color: #1a1a1a;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 2px solid #f0f0f0;
        }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 20px;
            margin-bottom: 30px;
        }
        .info-card {
            background: linear-gradient(135deg, #e0f7fa 0%, #b2ebf2 100%);
            padding: 20px;
            border-radius: 12px;
            border-left: 4px solid #38ef7d;
            transition: transform 0.2s;
        }
        .info-card:hover {
            transform: translateY(-2px);
        }
        .info-label {
            font-size: 12px;
            color: #666;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }
        .info-value {
            font-size: 16px;
            color: #1a1a1a;
            font-weight: 600;
        }
        .timeline {
            background: #f8f9fa;
            padding: 25px;
            border-radius: 12px;
            margin-bottom: 30px;
        }
        .timeline-title {
            font-size: 16px;
            font-weight: 700;
            color: #1a1a1a;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
        }
        .timeline-title::before {
            content: '⏱️';
            margin-right: 10px;
            font-size: 20px;
        }
        .timeline-item {
            display: flex;
            align-items: center;
            margin-bottom: 15px;
            padding: 15px;
            background: #ffffff;
            border-radius: 8px;
        }
        .timeline-item:last-child {
            margin-bottom: 0;
        }
        .timeline-dot {
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 15px;
            flex-shrink: 0;
        }
        .timeline-dot.fired {
            background: #f5576c;
        }
        .timeline-dot.resolved {
            background: #38ef7d;
        }
        .timeline-content {
            flex: 1;
        }
        .timeline-label {
            font-size: 12px;
            color: #999;
            margin-bottom: 5px;
        }
        .timeline-value {
            font-size: 14px;
            color: #333;
            font-weight: 600;
        }
        .details-section {
            background: #f8f9fa;
            padding: 25px;
            border-radius: 12px;
            margin-top: 30px;
        }
        .details-title {
            font-size: 16px;
            font-weight: 700;
            color: #1a1a1a;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
        }
        .details-title::before {
            content: '📋';
            margin-right: 10px;
            font-size: 20px;
        }
        .details-content {
            background: #ffffff;
            padding: 15px;
            border-radius: 8px;
            font-size: 14px;
            color: #333;
            line-height: 1.8;
            white-space: pre-wrap;
            word-wrap: break-word;
        }
        .footer {
            background: #f8f9fa;
            padding: 30px;
            text-align: center;
            border-top: 1px solid #e0e0e0;
        }
        .footer-logo {
            font-size: 24px;
            font-weight: 700;
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 10px;
        }
        .footer-text {
            font-size: 13px;
            color: #999;
            margin-bottom: 15px;
        }
        .footer-links {
            margin-top: 15px;
        }
        .footer-link {
            color: #11998e;
            text-decoration: none;
            margin: 0 10px;
            font-size: 13px;
            font-weight: 500;
        }
        @media only screen and (max-width: 600px) {
            .info-grid {
                grid-template-columns: 1fr;
            }
            .header h1 {
                font-size: 24px;
            }
            .content {
                padding: 30px 20px;
            }
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            <div class="success-icon">✅</div>
            <h1>恢复通知</h1>
            <div class="subtitle">OpsHub 智能运维平台</div>
        </div>
        
        <div class="content">
            <div class="success-badge">✅ 告警已恢复</div>
            
            <div class="rule-name">{{.RuleName}}</div>
            
            <div class="info-grid">
                <div class="info-card">
                    <div class="info-label">告警级别</div>
                    <div class="info-value">{{.SeverityLabel}}</div>
                </div>
                
                <div class="info-card">
                    <div class="info-label">当前值</div>
                    <div class="info-value">{{if .ResolveValue}}{{.ResolveValue}}{{else}}{{.Value}}{{end}}</div>
                </div>
                
                <div class="info-card">
                    <div class="info-label">恢复时间</div>
                    <div class="info-value">{{.ResolvedAt}}</div>
                </div>
                
                <div class="info-card">
                    <div class="info-label">持续时长</div>
                    <div class="info-value">{{if .Duration}}{{.Duration}}{{else}}-{{end}}</div>
                </div>
            </div>
            
            <div class="timeline">
                <div class="timeline-title">时间线</div>
                
                <div class="timeline-item">
                    <div class="timeline-dot fired"></div>
                    <div class="timeline-content">
                        <div class="timeline-label">告警触发</div>
                        <div class="timeline-value">{{.FiredAt}}</div>
                    </div>
                </div>
                
                <div class="timeline-item">
                    <div class="timeline-dot resolved"></div>
                    <div class="timeline-content">
                        <div class="timeline-label">告警恢复</div>
                        <div class="timeline-value">{{.ResolvedAt}}</div>
                    </div>
                </div>
            </div>
            
            <div class="details-section">
                <div class="details-title">标签详情</div>
                <div class="details-content">{{.LabelsDetail}}</div>
            </div>
            
            <div class="details-section">
                <div class="details-title">注解详情</div>
                <div class="details-content">{{.AnnotationsDetail}}</div>
            </div>
        </div>
        
        <div class="footer">
            <div class="footer-logo">OpsHub</div>
            <div class="footer-text">此邮件由 OpsHub 智能运维平台自动发送，请勿直接回复</div>
            <div class="footer-links">
                <a href="#" class="footer-link">查看详情</a>
                <a href="#" class="footer-link">告警历史</a>
                <a href="#" class="footer-link">帮助文档</a>
            </div>
        </div>
    </div>
</body>
</html>
```

## 使用方法

### 在告警治理模块中使用

1. 进入"告警治理" → "告警通道管理"
2. 新增或编辑邮件通道
3. 在"告警通知模板"中粘贴上面的**告警通知模板**
4. 在"恢复通知模板"中粘贴上面的**恢复通知模板**
5. 保存

### 模板特点

#### 告警通知模板（红色主题）
- 🎨 渐变红色主题，视觉冲击力强
- 🚨 醒目的告警图标和徽章
- 📊 网格布局展示关键信息
- 📋 折叠式详情区域
- 📱 响应式设计，移动端友好
- ✨ 悬停动画效果

#### 恢复通知模板（绿色主题）
- 🎨 渐变绿色主题，传达积极信号
- ✅ 成功图标带脉冲动画
- ⏱️ 时间线展示告警生命周期
- 📊 清晰的恢复信息展示
- 📱 响应式设计
- ✨ 现代化 UI 风格

### 模板变量说明

| 变量 | 说明 |
|------|------|
| `{{.RuleName}}` | 告警规则名称 |
| `{{.SeverityLabel}}` | 告警级别（critical/warning/info） |
| `{{.Value}}` | 当前值 |
| `{{.ResolveValue}}` | 恢复时的值 |
| `{{.FiredAt}}` | 触发时间 |
| `{{.ResolvedAt}}` | 恢复时间 |
| `{{.Duration}}` | 持续时长 |
| `{{.LabelsDetail}}` | 标签详情 |
| `{{.AnnotationsDetail}}` | 注解详情 |

### 自定义建议

1. **修改颜色主题**：
   - 告警模板：修改 `#f5576c`（红色）为其他颜色
   - 恢复模板：修改 `#38ef7d`（绿色）为其他颜色

2. **修改 Logo**：
   - 将 `OpsHub` 替换为您的品牌名称

3. **添加链接**：
   - 修改 footer 中的链接地址

4. **调整布局**：
   - 修改 `.info-grid` 的 `grid-template-columns` 改变列数

## 效果预览

### 告警通知效果
- 顶部：红色渐变背景 + 告警图标
- 中部：网格卡片展示关键指标
- 底部：详细信息折叠区域
- 整体：现代化、专业、易读

### 恢复通知效果
- 顶部：绿色渐变背景 + 成功图标（带动画）
- 中部：时间线展示告警生命周期
- 底部：详细信息区域
- 整体：积极、清晰、专业

## 兼容性

✅ 支持的邮件客户端：
- Gmail
- Outlook
- Apple Mail
- QQ 邮箱
- 163 邮箱
- 企业邮箱

✅ 支持的设备：
- 桌面端
- 移动端
- 平板

## 注意事项

1. 某些邮件客户端可能不支持所有 CSS 特性（如动画）
2. 建议发送测试邮件验证效果
3. 可以根据实际需求调整样式
4. 保持 HTML 代码在一行或使用工具压缩，避免邮件客户端解析问题
