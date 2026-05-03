# 告警规则导入功能修复报告

## 问题描述
用户反馈告警规则导入功能异常，导入时直接报错"导入失败"，且没有看到相关的错误日志。

## 问题分析

### 1. 前端问题

#### 问题 1：Content-Type 设置错误
**位置**: `web/src/api/alert.ts:127`

**原因**: 手动设置了 `Content-Type: 'multipart/form-data'`，导致浏览器无法自动添加 boundary 参数。

**错误代码**:
```typescript
export const importRules = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  return request.post('/api/v1/alert/rules/import', form, { 
    headers: { 'Content-Type': 'multipart/form-data' }  // ❌ 错误
  })
}
```

**正确代码**:
```typescript
export const importRules = (file: File) => {
  const form = new FormData()
  form.append('file', file)
  // 不要手动设置 Content-Type，让浏览器自动设置 boundary
  return request.post('/api/v1/alert/rules/import', form)  // ✅ 正确
}
```

#### 问题 2：文件对象获取错误
**位置**: `web/src/views/alert/RuleManagement.vue`

**原因**: Arco Design Upload 组件的 `custom-request` 回调参数结构不正确，导致无法获取到文件对象。

**错误代码**:
```typescript
const doImport = async ({ file }: any) => {
  try {
    const res = await importRules(file.file) as any  // ❌ file.file 可能为 undefined
    // ...
  }
}
```

**正确代码**:
```typescript
const doImport = async (options: any) => {
  try {
    // Arco Design Upload 的 custom-request 参数结构
    const file = options.fileItem?.file || options.file  // ✅ 兼容多种参数结构
    if (!file) {
      Message.error('未选择文件')
      return
    }

    console.log('开始导入规则文件:', file.name, file.type, file.size)
    const res = await importRules(file) as any
    // ...
  }
}
```

### 2. 后端代码检查

**位置**: `internal/server/alert/rule_handler.go:279-381`

后端代码逻辑正确，包含详细的日志记录：
- ✅ 文件上传处理正确
- ✅ JSON/YAML 格式解析正确
- ✅ 错误日志记录完整
- ✅ 路由注册正确 (`POST /api/v1/alert/rules/import`)

**日志记录**:
```go
logger.Info("开始导入规则", zap.String("filename", file.Filename), zap.Int64("size", file.Size))
logger.Debug("文件内容", zap.String("content", string(buf[:min(len(buf), 500)])))
logger.Info("JSON解析成功", zap.Int("规则数量", len(rules)))
logger.Info("导入完成", zap.Int("新增", success), zap.Int("更新", updated), zap.Int("失败", len(failedRules)))
```

## 修复方案

### 修复 1：移除手动设置的 Content-Type
**文件**: `web/src/api/alert.ts`

```diff
export const importRules = (file: File) => {
  const form = new FormData()
  form.append('file', file)
-  return request.post('/api/v1/alert/rules/import', form, { headers: { 'Content-Type': 'multipart/form-data' } })
+  return request.post('/api/v1/alert/rules/import', form)
}
```

### 修复 2：修复文件对象获取逻辑
**文件**: `web/src/views/alert/RuleManagement.vue`

```diff
-const doImport = async ({ file }: any) => {
+const doImport = async (options: any) => {
  try {
-    const res = await importRules(file.file) as any
+    // Arco Design Upload 的 custom-request 参数结构
+    const file = options.fileItem?.file || options.file
+    if (!file) {
+      Message.error('未选择文件')
+      return
+    }
+
+    console.log('开始导入规则文件:', file.name, file.type, file.size)
+    const res = await importRules(file) as any
    const imported = res?.imported || 0
    const updated = res?.updated || 0
    const failed = res?.failed || []
    // ...
  }
}
```

## 测试验证

### 1. 创建测试文件
**文件**: `test_rules.json`

```json
[
  {
    "name": "测试规则-CPU使用率过高",
    "description": "当CPU使用率超过80%时触发告警",
    "assetGroupId": 1,
    "ruleGroupId": 1,
    "dataSourceIds": "[1]",
    "expr": "100 - (avg by (instance) (irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 80",
    "evalInterval": 60,
    "duration": "5m",
    "severity": "warning",
    "labels": "{\"team\":\"ops\"}",
    "annotations": "{\"title\":\"CPU使用率过高\",\"description\":\"实例 {{ $labels.instance }} CPU使用率为 {{ $value }}%\"}",
    "enabled": false,
    "notifyOnResolve": true
  }
]
```

### 2. 测试步骤
1. 启动前端开发服务器
   ```bash
   cd web && npm run dev
   ```

2. 访问告警规则管理页面
   ```
   http://localhost:5173/alert/rules
   ```

3. 点击"导入"按钮，选择 `test_rules.json` 文件

4. 观察浏览器控制台输出：
   ```
   开始导入规则文件: test_rules.json application/json 1234
   ```

5. 观察后端日志：
   ```bash
   tail -f logs/app.log | grep "导入"
   ```

   预期输出：
   ```
   {"level":"info","msg":"开始导入规则","filename":"test_rules.json","size":1234}
   {"level":"info","msg":"JSON解析成功","规则数量":1}
   {"level":"info","msg":"导入完成","新增":1,"更新":0,"失败":0}
   ```

6. 检查导入结果提示：
   ```
   ✅ 导入成功：新增 1 条，更新 0 条
   ```

## 根本原因总结

1. **Content-Type 问题**: 
   - 手动设置 `multipart/form-data` 会覆盖浏览器自动生成的 boundary
   - 导致后端无法正确解析 multipart 请求

2. **文件对象获取问题**:
   - Arco Design Upload 的 `custom-request` 回调参数结构与预期不符
   - 需要兼容 `options.fileItem.file` 和 `options.file` 两种结构

3. **日志缺失问题**:
   - 由于前端请求根本没有发送成功（或发送了错误的请求）
   - 后端没有收到请求，因此没有日志输出

## 预防措施

1. **FormData 使用规范**:
   - 使用 FormData 时，不要手动设置 `Content-Type`
   - 让浏览器自动处理 boundary

2. **UI 组件参数验证**:
   - 使用第三方 UI 组件时，先打印回调参数结构
   - 添加参数验证和错误提示

3. **日志增强**:
   - 前端添加详细的 console.log
   - 后端保持详细的日志记录

4. **错误处理**:
   - 前端捕获并显示详细的错误信息
   - 后端返回明确的错误提示

## 相关文件

- ✅ `web/src/api/alert.ts` - API 定义（已修复）
- ✅ `web/src/views/alert/RuleManagement.vue` - 导入逻辑（已修复）
- ✅ `internal/server/alert/rule_handler.go` - 后端处理（无需修改）
- ✅ `internal/server/alert/http.go` - 路由注册（无需修改）

## 修复状态

- [x] 问题分析完成
- [x] 前端代码修复
- [x] 测试文件准备
- [ ] 功能测试验证（需要用户测试）
- [ ] 代码提交

## 下一步

1. 用户测试验证修复效果
2. 如果测试通过，提交代码
3. 更新用户文档，说明导入文件格式要求
