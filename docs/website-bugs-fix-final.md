# Web站点管理 - 三个关键Bug修复

## Bug 1: 新增内部站点会覆盖已有站点

### 问题描述
每次新增内部站点时，都会覆盖掉之前创建的站点，导致只能保留一个内部站点。

### 根本原因
前端在新增站点时，`formData` 对象中残留了之前编辑操作的 `id` 字段。当提交时，后端判断有 `id` 就执行更新操作而不是创建操作。

### 问题代码
```typescript
// 新增站点时没有清除 id 字段
const handleAdd = () => {
  Object.assign(formData, {
    name: '',
    // ... 其他字段
    // 缺少: id: undefined
  })
}
```

### 修复方案
在 `handleAdd` 函数中明确设置 `id: undefined`，确保是新增操作：

```typescript
const handleAdd = () => {
  dialogTitle.value = '新增站点'
  Object.assign(formData, {
    id: undefined,  // 清除 id 字段，确保是新增而不是更新
    name: '',
    url: '',
    icon: '',
    type: 'external',
    credential: '',
    secureCopyUrl: false,
    accessUser: '',
    accessPassword: '',
    description: '',
    status: 1,
    groupIds: [],
    agentHostIds: []
  })
  // 重置 Agent 主机列表为全部
  agentHosts.value = allAgentHosts.value
  dialogVisible.value = true
}
```

### 验证方法
1. 创建第一个内部站点
2. 再创建第二个内部站点
3. 检查站点列表，应该显示两个站点
4. 查询数据库：`SELECT * FROM websites WHERE type='internal';`

---

## Bug 2: 访问内部站点返回 401 未登录错误

### 问题描述
点击"访问站点"按钮访问内部站点时，新打开的窗口显示：
```json
{"code":401,"message":"未登录","timestamp":1773068849}
```

### 根本原因
1. 内部站点通过代理访问，代理路由在认证中间件保护下
2. 前端使用 `window.open()` 打开新窗口时，没有携带认证 token
3. 新窗口的请求无法通过认证中间件

### 修复方案

#### 前端修复
在打开代理 URL 时，通过 URL 参数传递 token：

```typescript
const handleAccess = async (record: Website) => {
  try {
    const res = await accessWebsite(record.id)
    if (res.type === 'external') {
      window.open(res.url, '_blank')
    } else {
      // 内部站点通过代理访问
      if (res.proxyUrl) {
        // 获取当前的 token
        const token = localStorage.getItem('token')
        if (!token) {
          Message.error('未登录，无法访问内部站点')
          return
        }
        // 构建带 token 的代理 URL
        const proxyUrl = `${res.proxyUrl}?token=${token}`
        window.open(proxyUrl, '_blank')
      } else {
        Message.error('无法获取代理访问地址')
      }
    }
  } catch (error: any) {
    Message.error('访问失败: ' + error.message)
  }
}
```

#### 后端修复
在代理请求时，从 URL 参数中移除 token（不传递给目标站点）：

```go
// 添加查询参数（排除 token 参数）
if c.Request.URL.RawQuery != "" {
    query := c.Request.URL.Query()
    query.Del("token") // 移除 token 参数，不传递给目标站点
    if len(query) > 0 {
        fullURL += "?" + query.Encode()
    }
}
```

同时，在复制请求头时跳过 Authorization 头：

```go
// 复制请求头
for key, values := range c.Request.Header {
    if len(values) > 0 {
        // 跳过一些不需要转发的头
        if key == "Host" || key == "Connection" ||
           strings.HasPrefix(key, "X-Forwarded") ||
           key == "Authorization" {  // 新增：跳过 Authorization
            continue
        }
        probeReq.Headers[key] = values[0]
    }
}
```

### 工作流程
```
用户点击访问
  → 前端获取 token
  → 构建 URL: /api/v1/websites/1/proxy?token=xxx
  → 打开新窗口
  → 后端认证中间件验证 token（从 URL 参数）
  → 通过认证
  → 代理请求到 Agent
  → Agent 访问内部站点
  → 返回响应
```

### 验证方法
1. 创建一个内部站点，绑定在线的 Agent
2. 点击"访问站点"按钮
3. 新窗口应该正常显示内部站点内容，不再返回 401 错误

---

## Bug 3: 选择业务分组后无法查询到 Agent 主机

### 问题描述
新增内部站点时，选择不同的业务分组后，Agent 主机下拉列表为空，无法选择主机。只有不选择业务分组时，才能看到 Agent 主机列表。

### 根本原因
业务分组过滤逻辑错误：当主机没有分组信息时（`groupIds` 为空），被过滤掉了。

### 问题代码
```typescript
const filterAgentHostsByGroups = () => {
  if (!formData.groupIds || formData.groupIds.length === 0) {
    agentHosts.value = allAgentHosts.value
  } else {
    agentHosts.value = allAgentHosts.value.filter((host: any) => {
      if (!host.groupIds || host.groupIds.length === 0) {
        return false  // ❌ 错误：过滤掉了未分组的主机
      }
      return host.groupIds.some((gid: number) => formData.groupIds.includes(gid))
    })
  }
}
```

### 修复方案
修改过滤逻辑，允许显示未分组的主机：

```typescript
const filterAgentHostsByGroups = () => {
  if (!formData.groupIds || formData.groupIds.length === 0) {
    // 没有选择分组，显示所有Agent主机
    agentHosts.value = allAgentHosts.value
  } else {
    // 根据分组过滤主机：显示属于选中分组的主机
    agentHosts.value = allAgentHosts.value.filter((host: any) => {
      // 如果主机没有分组信息，也显示（允许选择未分组的主机）
      if (!host.groupIds || host.groupIds.length === 0) {
        return true  // ✅ 修复：显示未分组的主机
      }
      // 检查主机是否属于选中的任一分组
      return host.groupIds.some((gid: number) => formData.groupIds.includes(gid))
    })
  }
}
```

### 设计考虑
**为什么要显示未分组的主机？**

1. **灵活性**: 用户可能有一些主机还没有分配到分组
2. **可用性**: 避免因为分组配置不完整导致无法选择主机
3. **用户体验**: 用户选择分组后，应该看到"属于该分组的主机 + 未分组的主机"

**替代方案**（如果需要严格过滤）：
```typescript
// 严格模式：只显示属于选中分组的主机
if (!host.groupIds || host.groupIds.length === 0) {
  return false  // 不显示未分组的主机
}
```

### 验证方法
1. 创建一些主机，部分分配到分组A，部分分配到分组B，部分不分配分组
2. 新增内部站点
3. 选择分组A
4. Agent 主机列表应该显示：分组A的主机 + 未分组的主机
5. 选择分组B
6. Agent 主机列表应该显示：分组B的主机 + 未分组的主机

---

## 额外优化

### 1. 编辑站点时的分组过滤
在编辑站点时，也需要根据已选择的分组过滤 Agent 主机：

```typescript
const handleEdit = (record: Website) => {
  dialogTitle.value = '编辑站点'
  Object.assign(formData, {
    id: record.id,
    // ... 其他字段
    groupIds: record.groupIds || [],
    agentHostIds: record.agentHostIds || []
  })
  // 根据已选择的分组过滤 Agent 主机
  filterAgentHostsByGroups()
  dialogVisible.value = true
}
```

### 2. Spin 组件警告修复
Arco Design 的 Spin 组件不接受 "large" 字符串作为 size：

```vue
<!-- 修复前 -->
<a-spin size="large" tip="加载中..." />

<!-- 修复后 -->
<a-spin tip="加载中..." />
```

---

## 测试清单

### Bug 1 测试
- [ ] 创建第一个内部站点，成功
- [ ] 创建第二个内部站点，成功
- [ ] 两个站点都显示在列表中
- [ ] 编辑第一个站点，不影响第二个站点
- [ ] 数据库中有两条记录

### Bug 2 测试
- [ ] 访问外部站点，正常打开
- [ ] 访问内部站点（Agent 在线），正常显示内容
- [ ] 访问内部站点（Agent 离线），提示 Agent 离线
- [ ] 未登录时访问内部站点，提示未登录
- [ ] 代理请求不会将 token 传递给目标站点

### Bug 3 测试
- [ ] 不选择分组，显示所有 Agent 主机
- [ ] 选择分组A，显示分组A的主机 + 未分组的主机
- [ ] 选择分组B，显示分组B的主机 + 未分组的主机
- [ ] 选择多个分组，显示所有选中分组的主机 + 未分组的主机
- [ ] 切换分组时，已选择的主机如果不在新列表中会被清空

---

## 相关文件

### 前端
- `web/src/views/asset/Websites.vue` - 站点管理页面

### 后端
- `internal/server/asset/website_proxy.go` - 代理处理器
- `internal/data/asset/website.go` - 数据访问层

### 数据库
- `websites` - 站点表
- `website_agents` - 站点与 Agent 主机关联表
- `website_groups` - 站点与业务分组关联表

---

## 后续优化建议

1. **认证方式优化**: 考虑使用 session cookie 而不是 URL 参数传递 token，更安全
2. **代理性能**: 对于大文件或流式响应，考虑使用流式代理
3. **缓存策略**: 对于静态资源，可以在代理层添加缓存
4. **错误处理**: 增强代理错误的用户提示，区分网络错误、Agent 错误、目标站点错误
5. **权限控制**: 添加站点访问权限控制，不是所有用户都能访问所有站点
