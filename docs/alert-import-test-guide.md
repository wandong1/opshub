# 告警规则导入功能测试指南

## 快速测试

### 1. 重新构建前端
```bash
cd web
npm run build
```

或者使用开发模式：
```bash
cd web
npm run dev
```

### 2. 访问告警规则管理页面
```
http://localhost:5173/alert/rules  (开发模式)
或
http://localhost:9876/alert/rules  (生产模式)
```

### 3. 测试导入功能

#### 方式一：使用提供的测试文件
1. 点击页面右上角的"导入"按钮
2. 选择项目根目录下的 `test_rules.json` 文件
3. 观察导入结果提示

#### 方式二：创建自己的测试文件
创建一个 JSON 文件，内容如下：

```json
[
  {
    "name": "测试规则-磁盘使用率",
    "description": "磁盘使用率超过85%时告警",
    "assetGroupId": 1,
    "ruleGroupId": 1,
    "dataSourceIds": "[1]",
    "expr": "(node_filesystem_size_bytes - node_filesystem_free_bytes) / node_filesystem_size_bytes * 100 > 85",
    "evalInterval": 60,
    "duration": "5m",
    "severity": "warning",
    "enabled": false
  }
]
```

### 4. 预期结果

#### 成功情况
- ✅ 浏览器控制台输出：`开始导入规则文件: test_rules.json application/json 1234`
- ✅ 页面提示：`导入成功：新增 X 条，更新 Y 条`
- ✅ 规则列表自动刷新，显示新导入的规则

#### 失败情况（修复前）
- ❌ 页面提示：`导入失败: 未知错误`
- ❌ 浏览器控制台报错
- ❌ 后端无日志输出

### 5. 查看后端日志（可选）
```bash
tail -f logs/app.log | grep "导入"
```

预期输出：
```
{"level":"info","msg":"开始导入规则","filename":"test_rules.json","size":1234}
{"level":"info","msg":"JSON解析成功","规则数量":2}
{"level":"info","msg":"创建新规则","name":"测试规则-CPU使用率过高"}
{"level":"info","msg":"创建新规则","name":"测试规则-内存使用率过高"}
{"level":"info","msg":"导入完成","新增":2,"更新":0,"失败":0}
```

## 支持的文件格式

### JSON 格式
```json
[
  {
    "name": "规则名称",
    "expr": "PromQL表达式",
    "severity": "warning",
    ...
  }
]
```

### YAML 格式
```yaml
- name: 规则名称
  expr: PromQL表达式
  severity: warning
  ...
```

## 导入规则说明

1. **规则名称唯一性**
   - 如果导入的规则名称已存在，会更新现有规则
   - 如果规则名称不存在，会创建新规则

2. **默认值**
   - 导入的规则默认为禁用状态（`enabled: false`）
   - 评估间隔默认为 15 秒（如果未指定）

3. **必填字段**
   - `name` - 规则名称
   - `expr` - PromQL 表达式
   - `severity` - 告警级别（critical/warning/info）

4. **可选字段**
   - `description` - 规则描述
   - `assetGroupId` - 业务分组 ID
   - `ruleGroupId` - 规则分类 ID
   - `dataSourceIds` - 数据源 ID 列表（JSON 字符串）
   - `evalInterval` - 评估间隔（秒）
   - `duration` - 持续时间（如 "5m"）
   - `labels` - 标签（JSON 字符串）
   - `annotations` - 注解（JSON 字符串）

## 常见问题

### Q1: 导入后规则没有显示？
**A**: 检查筛选条件，导入的规则默认是禁用状态，可能被过滤掉了。

### Q2: 导入提示"文件格式错误"？
**A**: 确保文件是有效的 JSON 或 YAML 格式，可以使用在线工具验证格式。

### Q3: 部分规则导入失败？
**A**: 查看导入结果提示中的失败原因，通常是字段验证失败或数据源不存在。

### Q4: 如何批量更新规则？
**A**: 
1. 先导出现有规则（点击"导出"按钮）
2. 修改导出的文件
3. 重新导入（同名规则会被更新）

## 修复内容总结

本次修复解决了以下问题：

1. ✅ **Content-Type 问题**
   - 移除了手动设置的 `multipart/form-data`
   - 让浏览器自动添加正确的 boundary

2. ✅ **文件对象获取问题**
   - 修复了 Arco Upload 组件参数解析
   - 添加了文件验证和错误提示

3. ✅ **日志增强**
   - 前端添加了详细的控制台日志
   - 后端已有完整的日志记录

## 反馈

如果测试过程中遇到任何问题，请提供：
1. 浏览器控制台的完整错误信息
2. 后端日志输出
3. 导入的文件内容（脱敏后）
