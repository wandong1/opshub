# 告警数据源Agent代理转发 - 快速开始指南

## 🎯 功能概述

此功能解决边缘机房Prometheus等数据源无法被中心平台直接访问的问题，通过在Agent上转发请求，实现：

1. ✅ 告警规则可以查询边缘数据源
2. ✅ Grafana可以展示边缘数据
3. ✅ 支持多Agent故障转移

## 📋 操作步骤

### 第1步：创建Agent代理数据源

1. 进入 **告警管理 → 数据源管理**
2. 点击 **新增数据源** 按钮
3. 填写基本信息：
   - 名称：边缘Prometheus
   - 类型：Prometheus
   - 接入方式：**Agent代理** （关键！）
4. 配置数据源信息：
   - 数据源地址：`prometheus` （Agent可访问）
   - 数据源端口：`9090`
5. 如需认证，填写用户名、密码或Token
6. 点击 **保存** → 系统自动生成ProxyToken和ProxyURL

### 第2步：关联Agent主机

1. 数据源保存后，在 **关联主机** 部分看到 **+ 添加主机** 按钮
2. 点击打开主机选择模态框
3. 在列表中选择在线的Agent主机
4. 设置优先级（0最高，10最低）
5. 点击 **添加关联**
6. 可添加多个Agent实现高可用

### 第3步：在Grafana中使用

1. 打开Grafana（内嵌在OpsHub中）
2. 创建新的数据源：
   - 类型：Prometheus
   - URL：复制OpsHub中的 **代理转发URL**
   - 例如：`http://localhost:8080/api/v1/alert/proxy/datasource/uuid-xxx`
3. 点击 **Save & Test**
4. 成功后可在Grafana中创建图表查看数据

### 第4步：在告警规则中使用

1. 进入 **告警管理 → 告警规则**
2. 创建新规则或编辑现有规则
3. 在 **数据源** 选择刚创建的Agent代理数据源
4. 编写PromQL表达式
5. 保存规则
6. 系统会通过Agent自动转发查询请求

## 🔧 故障排查

### 问题1：连接失败

**现象**：保存数据源时提示"连接失败"

**排查步骤**：
1. 确认Agent主机在线（图标为绿色）
2. 确认Agent可访问数据源（Host和Port正确）
3. 检查数据源认证信息
4. 检查网络连接

### 问题2：没有可用的Agent

**现象**：告警规则执行时提示"没有可用的Agent"

**排查步骤**：
1. 确认已关联Agent主机
2. 检查Agent是否在线
3. 检查Agent与数据源的网络连接
4. 查看Agent日志

### 问题3：Grafana数据源测试失败

**现象**：Grafana中ProxyURL测试失败

**排查步骤**：
1. 确认ProxyToken正确（从OpsHub复制）
2. 确认OpsHub服务正常运行
3. 确认关联的Agent在线
4. 检查网络/防火墙

## 📊 查看运行状态

### 数据源列表显示

| 接入方式 | 地址显示 | 备注 |
|---------|--------|------|
| 直连 | 显示完整URL | 平台可直接访问 |
| Agent代理 | 显示Host:Port | 通过Agent转发 |

### 关联主机列表显示

- 主机名（IP）
- 优先级：0最高，10最低
- 删除按钮：可移除关联

## 🎓 常见场景

### 场景1：边缘机房单机部署

```
中心平台 --无法直连--> 边缘Prometheus
           ↓ 通过Agent转发
          Agent
           ↓
        边缘Prometheus
```

**操作**：
1. 选择"Agent代理"模式
2. 关联该机房的Agent主机
3. 优先级不重要（只有一个）

### 场景2：边缘机房高可用（多Agent）

```
中心平台 ──┬→ Agent1 ──→ Prometheus
          ├→ Agent2 ──→ Prometheus
          └→ Agent3 ──→ Prometheus
```

**操作**：
1. 选择"Agent代理"模式
2. 关联3个Agent主机
3. 设置优先级：Agent1=0, Agent2=1, Agent3=2
4. Agent1离线时自动转移到Agent2

### 场景3：直连+代理混合

```
直连数据源（平台网络可达）
├─ 中心Prometheus：直连模式
└─ 云服务商Prometheus：直连模式

代理数据源（通过Agent）
├─ 分公司1 Prometheus：Agent代理
├─ 分公司2 VictoriaMetrics：Agent代理
└─ IDC Prometheus：Agent代理
```

**操作**：创建多个数据源，分别选择直连或代理模式

## 💡 最佳实践

1. **优先级设置**：重要的Agent设置较低优先级（0优先）
2. **监控Agent健康**：定期检查关联Agent的在线状态
3. **数据源认证**：敏感环境使用Token而非明文密码
4. **备份Agent**：至少配置2个Agent实现高可用
5. **定期测试**：通过"测试连接"定期验证可用性

## 📞 获取帮助

- 查看日志：应用日志中会记录Agent转发详情
- 联系管理员：检查Agent安装和配置
- 检查文档：完整文档见 ALERT_DATASOURCE_AGENT_PROXY_IMPLEMENTATION.md
