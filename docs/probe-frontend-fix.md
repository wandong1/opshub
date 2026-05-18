# 拨测类型巡检项前端问题修复

## 问题描述

在巡检管理界面对拨测类型的巡检项进行测试执行时，发现两个问题：

### 问题 1：断言结果显示错误
**现象**：
- 后端返回：`assertionResult: "skip"`
- 前端显示：`断言: 失败`（红色标签）
- 预期显示：`断言: 跳过`（灰色标签）

**原因**：
前端代码只判断了 `pass` 和其他情况，没有处理 `skip` 状态。

```vue
<!-- 修复前 -->
<a-tag :color="log.assertionResult === 'pass' ? 'green' : 'red'">
  断言: {{ log.assertionResult === 'pass' ? '通过' : '失败' }}
</a-tag>
```

### 问题 2：拨测类型不显示断言配置
**现象**：
- 拨测类型的巡检项在前端配置页面没有显示断言配置选项
- 无法为拨测类型巡检项配置断言规则

**原因**：
前端代码使用了 `v-if="item.executionType !== 'probe'"` 条件，排除了拨测类型。

```vue
<!-- 修复前 -->
<template v-if="item.executionType !== 'probe'">
  <a-form-item label="断言规则">
    <!-- 断言配置 -->
  </a-form-item>
</template>
```

---

## 修复方案

### 修复 1：断言结果显示逻辑

**文件**：`web/src/views/inspection/InspectionManagement.vue`

**修改位置**：第 800-801 行

**修复后代码**：
```vue
<a-tag :color="log.assertionResult === 'pass' ? 'green' : log.assertionResult === 'skip' ? 'gray' : 'red'">
  断言: {{ log.assertionResult === 'pass' ? '通过' : log.assertionResult === 'skip' ? '跳过' : '失败' }}
</a-tag>
```

**效果**：
- `pass` → 绿色标签，显示"通过"
- `skip` → 灰色标签，显示"跳过"
- `fail` → 红色标签，显示"失败"

---

### 修复 2：拨测类型断言配置

**文件**：`web/src/views/inspection/InspectionManagement.vue`

**修改位置**：第 610-654 行

**修复内容**：

#### 1. 移除拨测类型的排除条件

```vue
<!-- 修复前 -->
<template v-if="item.executionType !== 'probe'">
  <a-form-item label="断言规则">
    <!-- ... -->
  </a-form-item>
</template>

<!-- 修复后 -->
<a-form-item label="断言规则">
  <!-- 所有类型都显示断言配置 -->
</a-form-item>
```

#### 2. 新增拨测专用断言类型

```vue
<a-select v-model="item.assertionType" placeholder="选择断言类型" allow-clear>
  <!-- 拨测类型显示拨测专用断言 -->
  <template v-if="item.executionType === 'probe'">
    <a-optgroup label="拨测专用">
      <a-option value="probe_success">拨测是否成功</a-option>
      <a-option value="probe_latency_lt">响应时间小于（毫秒）</a-option>
      <a-option value="probe_assertion_all">原始断言全部通过</a-option>
      <a-option value="probe_status_code">HTTP状态码等于</a-option>
    </a-optgroup>
  </template>
  <!-- PromQL 类型只显示数值比较 -->
  <template v-else-if="item.executionType === 'promql'">
    <!-- ... -->
  </template>
  <!-- 命令和脚本类型显示所有断言 -->
  <template v-else>
    <!-- ... -->
  </template>
</a-select>
```

#### 3. 新增辅助方法

**getAssertionPlaceholder** - 获取断言值输入框占位符
```typescript
const getAssertionPlaceholder = (assertionType: string, executionType: string) => {
  if (!assertionType) return '请先选择断言类型'

  // 拨测专用断言类型
  const probeAssertionPlaceholders: Record<string, string> = {
    probe_success: '无需填写',
    probe_latency_lt: '输入毫秒数，如: 500',
    probe_assertion_all: '无需填写',
    probe_status_code: '输入状态码，如: 200'
  }

  if (probeAssertionPlaceholders[assertionType]) {
    return probeAssertionPlaceholders[assertionType]
  }

  // 其他类型
  if (executionType === 'promql') {
    return '断言值（数值）'
  }

  return '断言值'
}
```

**isAssertionValueDisabled** - 判断断言值输入框是否禁用
```typescript
const isAssertionValueDisabled = (assertionType: string) => {
  // probe_success 和 probe_assertion_all 不需要断言值
  return assertionType === 'probe_success' || assertionType === 'probe_assertion_all'
}
```

**getAssertionExtraText** - 获取断言配置的额外说明文本
```typescript
const getAssertionExtraText = (executionType: string, assertionType: string) => {
  if (executionType === 'probe') {
    if (assertionType === 'probe_success') {
      return '验证拨测是否执行成功'
    } else if (assertionType === 'probe_latency_lt') {
      return '验证响应时间是否小于指定阈值（毫秒）'
    } else if (assertionType === 'probe_assertion_all') {
      return '验证拨测配置中的原始断言是否全部通过'
    } else if (assertionType === 'probe_status_code') {
      return '验证 HTTP 状态码是否等于期望值'
    }
    return '对拨测结果进行二次断言验证'
  } else if (executionType === 'promql') {
    return 'PromQL 查询结果将自动提取指标值进行数值比较'
  }
  return '对执行结果进行断言验证'
}
```

#### 4. 更新 getAssertionTypeText 方法

```typescript
const getAssertionTypeText = (type: string) => {
  const map: Record<string, string> = {
    eq: '等于',
    ne: '不等于',
    gt: '大于',
    gte: '大于等于',
    lt: '小于',
    lte: '小于等于',
    contains: '包含',
    not_contains: '不包含',
    regex: '正则匹配',
    status_code: '状态码',
    // 新增拨测专用断言类型
    probe_success: '拨测是否成功',
    probe_latency_lt: '响应时间小于',
    probe_assertion_all: '原始断言全部通过',
    probe_status_code: 'HTTP状态码等于'
  }
  return map[type] || type
}
```

#### 5. 调整变量提取配置显示条件

```vue
<!-- 变量提取（PromQL 和拨测类型不需要） -->
<template v-if="item.executionType !== 'promql' && item.executionType !== 'probe'">
  <a-form-item label="变量提取" :label-col-flex="'100px'">
    <!-- ... -->
  </a-form-item>
</template>
```

---

## 修复效果

### 1. 断言结果显示正确

**修复前**：
```
断言: 失败 (红色)
无断言规则，跳过校验
```

**修复后**：
```
断言: 跳过 (灰色)
无断言规则，跳过校验
```

---

### 2. 拨测类型可配置断言

**修复前**：
- 拨测类型巡检项不显示断言配置
- 无法配置断言规则

**修复后**：
- 拨测类型巡检项显示断言配置
- 提供 4 种拨测专用断言类型
- 智能提示断言值格式
- 自动禁用不需要断言值的类型

**界面效果**：

```
断言规则:
┌─────────────────────────────────────────────────┐
│ [拨测专用 ▼]                    │ [输入断言值]   │
│  - 拨测是否成功                  │                │
│  - 响应时间小于（毫秒）          │                │
│  - 原始断言全部通过              │                │
│  - HTTP状态码等于                │                │
└─────────────────────────────────────────────────┘
提示: 对拨测结果进行二次断言验证
```

**示例 1：验证拨测成功**
```
断言类型: 拨测是否成功
断言值: [无需填写] (输入框禁用)
提示: 验证拨测是否执行成功
```

**示例 2：验证响应时间**
```
断言类型: 响应时间小于（毫秒）
断言值: 500
提示: 验证响应时间是否小于指定阈值（毫秒）
```

**示例 3：验证原始断言**
```
断言类型: 原始断言全部通过
断言值: [无需填写] (输入框禁用)
提示: 验证拨测配置中的原始断言是否全部通过
```

**示例 4：验证 HTTP 状态码**
```
断言类型: HTTP状态码等于
断言值: 200
提示: 验证 HTTP 状态码是否等于期望值
```

---

## 验证方案

### 测试场景 1：断言结果显示

**步骤**：
1. 创建拨测类型巡检项，不配置断言
2. 执行测试
3. 查看测试日志

**预期结果**：
- 断言结果显示：`断言: 跳过`（灰色标签）
- 断言详情显示：`无断言规则，跳过校验`

---

### 测试场景 2：拨测专用断言配置

**步骤**：
1. 创建拨测类型巡检项
2. 选择执行类型：probe
3. 选择拨测配置
4. 配置断言规则

**预期结果**：
- 断言类型下拉框显示"拨测专用"分组
- 包含 4 种拨测专用断言类型
- 断言值输入框根据类型智能提示
- `probe_success` 和 `probe_assertion_all` 类型的断言值输入框禁用

---

### 测试场景 3：拨测断言执行

**步骤**：
1. 创建拨测类型巡检项
2. 配置断言：`probe_latency_lt = 500`
3. 执行测试

**预期结果**：
- 响应时间 < 500ms：断言通过，显示"断言: 通过"（绿色）
- 响应时间 >= 500ms：断言失败，显示"断言: 失败"（红色）
- 断言详情显示：`响应时间 123.45ms < 阈值 500.00ms`

---

## 改动文件

| 文件 | 改动内容 | 行数 |
|-----|---------|------|
| `web/src/views/inspection/InspectionManagement.vue` | 修复断言显示 + 新增拨测断言配置 | +80 |

**总计**：1 个文件，约 80 行代码

---

## 技术细节

### 1. 三元运算符嵌套

```typescript
// 颜色判断
log.assertionResult === 'pass' ? 'green' : log.assertionResult === 'skip' ? 'gray' : 'red'

// 文本判断
log.assertionResult === 'pass' ? '通过' : log.assertionResult === 'skip' ? '跳过' : '失败'
```

### 2. 条件渲染优化

```vue
<!-- 使用 v-if 和 v-else-if 替代嵌套的 v-if -->
<template v-if="item.executionType === 'probe'">
  <!-- 拨测专用断言 -->
</template>
<template v-else-if="item.executionType === 'promql'">
  <!-- PromQL 断言 -->
</template>
<template v-else>
  <!-- 通用断言 -->
</template>
```

### 3. 动态禁用输入框

```vue
<a-input
  v-model="item.assertionValue"
  :placeholder="getAssertionPlaceholder(item.assertionType, item.executionType)"
  :disabled="!item.assertionType || isAssertionValueDisabled(item.assertionType)"
/>
```

### 4. 智能提示文本

```typescript
// 根据执行类型和断言类型返回不同的提示文本
getAssertionExtraText(executionType, assertionType)
```

---

## 总结

本次修复解决了拨测类型巡检项的两个前端问题：

### ✅ 已修复

1. **断言结果显示错误**
   - 正确处理 `skip` 状态
   - 使用灰色标签显示"跳过"

2. **拨测类型不显示断言配置**
   - 移除拨测类型的排除条件
   - 新增 4 种拨测专用断言类型
   - 智能提示断言值格式
   - 自动禁用不需要断言值的类型

### 🎯 核心优势

- ✅ 用户体验优化，断言状态显示清晰
- ✅ 功能完整，拨测类型支持二次断言
- ✅ 智能提示，降低配置错误率
- ✅ 向后兼容，不影响现有功能

### 📊 验证结果

- ✅ TypeScript 类型检查通过
- ✅ 代码逻辑正确
- ✅ 用户界面友好
