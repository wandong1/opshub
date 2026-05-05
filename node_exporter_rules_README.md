# Node Exporter 告警规则导入测试说明

## 文件位置
```
/Users/Zhuanz/golang_project/src/opshub/node_exporter_rules.yaml
```

## 规则列表

### 1. 主机-CPU使用率过高 ⚠️ Warning
- **触发条件**: CPU 使用率 > 80%
- **持续时间**: 5 分钟
- **评估间隔**: 60 秒
- **PromQL**: `100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80`

### 2. 主机-内存使用率过高 🔴 Critical
- **触发条件**: 内存使用率 > 85%
- **持续时间**: 3 分钟
- **评估间隔**: 60 秒
- **PromQL**: `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85`

### 3. 主机-磁盘使用率过高 ⚠️ Warning
- **触发条件**: 磁盘使用率 > 85%
- **持续时间**: 5 分钟
- **评估间隔**: 300 秒（5分钟）
- **PromQL**: `(node_filesystem_size_bytes{fstype=~"ext4|xfs"} - node_filesystem_free_bytes{fstype=~"ext4|xfs"}) / node_filesystem_size_bytes{fstype=~"ext4|xfs"} * 100 > 85`
- **说明**: 仅监控 ext4 和 xfs 文件系统

### 4. 主机-磁盘IO等待时间过高 ⚠️ Warning
- **触发条件**: IO 等待时间 > 20%
- **持续时间**: 5 分钟
- **评估间隔**: 60 秒
- **PromQL**: `avg by (instance) (irate(node_cpu_seconds_total{mode="iowait"}[5m])) * 100 > 20`
- **说明**: 高 IO 等待可能表示磁盘性能瓶颈

### 5. 主机-网络流量异常 ℹ️ Info
- **触发条件**: 网络接收流量 > 100 MB/s
- **持续时间**: 3 分钟
- **评估间隔**: 60 秒
- **PromQL**: `sum by (instance) (irate(node_network_receive_bytes_total{device!~"lo|docker.*|veth.*"}[5m])) / 1024 / 1024 > 100`
- **说明**: 排除 lo、docker、veth 等虚拟网卡

## 导入测试步骤

### 1. 启动前端开发服务器
```bash
cd /Users/Zhuanz/golang_project/src/opshub/web
npm run dev
```

### 2. 访问告警规则管理页面
```
http://localhost:5173/alert/rules
```

### 3. 导入规则
1. 点击页面右上角的 **"导入"** 按钮
2. 选择文件：`node_exporter_rules.yaml`
3. 等待导入完成

### 4. 预期结果

#### 成功情况
- ✅ 浏览器控制台输出：
  ```
  开始导入规则文件: node_exporter_rules.yaml application/x-yaml 2345
  ```

- ✅ 页面提示：
  ```
  导入成功：新增 5 条，更新 0 条
  ```

- ✅ 规则列表显示 5 条新规则：
  - 主机-CPU使用率过高
  - 主机-内存使用率过高
  - 主机-磁盘使用率过高
  - 主机-磁盘IO等待时间过高
  - 主机-网络流量异常

#### 失败情况（如果修复前）
- ❌ 页面提示：`导入失败: 未知错误`
- ❌ 浏览器控制台报错
- ❌ 规则列表无变化

### 5. 验证导入结果

#### 检查规则详情
1. 点击任意一条规则查看详情
2. 确认字段正确：
   - 规则名称
   - PromQL 表达式
   - 告警级别
   - 评估间隔
   - 持续时间

#### 检查规则状态
- 所有导入的规则默认为 **禁用状态**（enabled: false）
- 可以手动启用需要的规则

#### 测试规则（可选）
1. 点击规则右侧的 **"测试"** 按钮
2. 查看是否能正常查询数据
3. 确认 PromQL 表达式正确

### 6. 查看后端日志（可选）

打开新终端，查看导入日志：
```bash
tail -f /Users/Zhuanz/golang_project/src/opshub/logs/app.log | grep "导入"
```

预期输出：
```json
{"level":"info","msg":"开始导入规则","filename":"node_exporter_rules.yaml","size":2345}
{"level":"info","msg":"YAML解析成功","规则数量":5}
{"level":"info","msg":"创建新规则","name":"主机-CPU使用率过高"}
{"level":"info","msg":"创建新规则","name":"主机-内存使用率过高"}
{"level":"info","msg":"创建新规则","name":"主机-磁盘使用率过高"}
{"level":"info","msg":"创建新规则","name":"主机-磁盘IO等待时间过高"}
{"level":"info","msg":"创建新规则","name":"主机-网络流量异常"}
{"level":"info","msg":"导入完成","新增":5,"更新":0,"失败":0}
```

## 注意事项

### 1. 数据源配置
- 规则中的 `dataSourceIds: "[1]"` 表示使用 ID 为 1 的数据源
- 如果你的数据源 ID 不是 1，需要修改 YAML 文件中的 `dataSourceIds` 字段
- 或者导入后在页面上手动修改

### 2. 业务分组和规则分类
- `assetGroupId: 1` - 业务分组 ID
- `ruleGroupId: 1` - 规则分类 ID
- 如果这些 ID 在你的系统中不存在，可能需要先创建或修改为正确的 ID

### 3. 阈值调整
导入后可以根据实际情况调整阈值：
- CPU 使用率：默认 80%
- 内存使用率：默认 85%
- 磁盘使用率：默认 85%
- IO 等待时间：默认 20%
- 网络流量：默认 100 MB/s

### 4. 启用规则
导入的规则默认是禁用状态，需要手动启用：
1. 在规则列表中找到对应规则
2. 点击右侧的 **"启用/禁用"** 开关
3. 确认启用

## 常见问题

### Q1: 导入后提示"数据源不存在"？
**A**: 修改 YAML 文件中的 `dataSourceIds` 为你系统中实际存在的数据源 ID。

### Q2: 规则测试失败？
**A**: 
1. 检查数据源是否配置正确
2. 确认 Node Exporter 是否正常运行
3. 验证 PromQL 表达式是否正确

### Q3: 导入提示"YAML 解析失败"？
**A**: 
1. 检查 YAML 文件格式是否正确
2. 确保缩进使用空格而非 Tab
3. 确保字符串中的引号正确转义

### Q4: 部分规则导入失败？
**A**: 查看导入结果提示中的失败原因，通常是：
- 字段验证失败
- 数据源不存在
- 业务分组或规则分类不存在

## 修复验证

如果导入成功，说明以下问题已修复：
- ✅ Content-Type boundary 问题
- ✅ 文件对象获取问题
- ✅ YAML 格式解析正常
- ✅ 后端接口工作正常

## 下一步

导入成功后，可以：
1. 根据实际需求调整阈值
2. 启用需要的规则
3. 配置告警通知渠道
4. 测试告警触发和恢复

---

**祝测试顺利！如有问题请随时反馈。** 🎉
