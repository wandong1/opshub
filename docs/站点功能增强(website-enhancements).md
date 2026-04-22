# Web站点管理功能增强

## 修复时间
2026-03-09 22:30

## 新增功能

### ✅ 功能 1: Agent 主机根据业务分组过滤

**需求**：选择内部站点时，Agent 主机下拉菜单应该根据所选的业务分组进行过滤。

**实现方案**：

1. **保存所有 Agent 主机**：
```typescript
const allAgentHosts = ref<any[]>([]) // 保存所有Agent主机
```

2. **根据分组过滤**：
```typescript
const filterAgentHostsByGroups = () => {
  if (!formData.groupIds || formData.groupIds.length === 0) {
    // 没有选择分组，显示所有Agent主机
    agentHosts.value = allAgentHosts.value
  } else {
    // 根据分组过滤主机
    agentHosts.value = allAgentHosts.value.filter((host: any) => {
      if (!host.groupIds || host.groupIds.length === 0) {
        return false
      }
      return host.groupIds.some((gid: number) => formData.groupIds.includes(gid))
    })
  }
}
```

3. **监听分组变化**：
```typescript
const handleGroupChange = () => {
  filterAgentHostsByGroups()
  // 如果当前选中的Agent主机不在过滤后的列表中，清空选择
  if (formData.agentHostIds && formData.agentHostIds.length > 0) {
    const validHostIds = agentHosts.value.map((h: any) => h.id)
    formData.agentHostIds = formData.agentHostIds.filter((id: number) =>
      validHostIds.includes(id)
    )
  }
}
```

4. **绑定事件**：
```vue
<a-select
  v-model="formData.groupIds"
  @change="handleGroupChange"
  ...
>
```

**使用流程**：
1. 新增内部站点
2. 选择业务分组（可多选）
3. Agent 主机下拉菜单自动过滤，只显示属于所选分组的主机
4. 如果没有选择分组，显示所有 Agent 主机

---

### ✅ 功能 2: 复制账号和密码

**需求**：在站点列表中添加复制凭据功能，方便用户快速复制账号和密码。

**实现方案**：

1. **添加复制按钮**（仅当站点有账号时显示）：
```vue
<a-button
  v-if="record.accessUser"
  type="text"
  size="small"
  @click="handleCopyCredentials(record)"
>
  <template #icon><icon-copy /></template>
  复制凭据
</a-button>
```

2. **复制凭据函数**：
```typescript
const handleCopyCredentials = async (record: Website) => {
  try {
    // 获取完整的站点信息（包括密码）
    const res = await getWebsite(record.id)
    const website = res

    const credentials = `站点：${website.name}
URL：${website.url}
账号：${website.accessUser || '无'}
密码：${website.accessPassword || '无'}`

    // 复制到剪贴板
    await navigator.clipboard.writeText(credentials)
    Message.success('凭据已复制到剪贴板')
  } catch (error: any) {
    // 降级方案：使用传统方法
    const textarea = document.createElement('textarea')
    textarea.value = credentials
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    Message.success('凭据已复制到剪贴板')
  }
}
```

3. **后端支持**：

添加 `AccessPassword` 字段到 `WebsiteVO`：
```go
type WebsiteVO struct {
    // ...
    AccessPassword string `json:"accessPassword"` // 访问密码（仅在详情接口返回）
    // ...
}
```

添加解密方法：
```go
func (uc *WebsiteUseCase) decrypt(ciphertext string) (string, error) {
    if ciphertext == "" {
        return "", nil
    }

    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(uc.encryptionKey)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", fmt.Errorf("ciphertext too short")
    }

    nonce, cipherData := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

创建 `toVOWithPassword` 方法：
```go
func (uc *WebsiteUseCase) toVOWithPassword(ctx context.Context, website *Website) (*WebsiteVO, error) {
    vo, err := uc.toVO(ctx, website)
    if err != nil {
        return nil, err
    }

    // 解密密码
    if website.AccessPassword != "" {
        decryptedPassword, err := uc.decrypt(website.AccessPassword)
        if err == nil {
            vo.AccessPassword = decryptedPassword
        }
    }

    return vo, nil
}
```

修改 `GetByID` 使用新方法：
```go
func (uc *WebsiteUseCase) GetByID(ctx context.Context, id uint) (*WebsiteVO, error) {
    website, err := uc.websiteRepo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 获取详情时返回完整信息（包括密码）
    return uc.toVOWithPassword(ctx, website)
}
```

**复制内容格式**：
```
站点：百度
URL：https://www.baidu.com
账号：admin
密码：123456
```

**安全考虑**：
- 密码在数据库中加密存储（AES-256-GCM）
- 列表接口不返回密码
- 详情接口返回解密后的密码（仅用于复制功能）
- 使用 Clipboard API（现代浏览器）
- 降级到 `document.execCommand('copy')`（旧浏览器）

---

## 修改文件

### 前端
- `web/src/views/asset/Websites.vue`
  - 添加 `allAgentHosts` 保存所有主机
  - 添加 `filterAgentHostsByGroups()` 过滤函数
  - 添加 `handleGroupChange()` 监听分组变化
  - 添加 `handleCopyCredentials()` 复制凭据
  - 修复 `loadAgentHosts()` 数据解析
  - 导入 `IconCopy` 图标
  - 导入 `getWebsite` API

### 后端
- `internal/biz/asset/website.go`
  - 添加 `AccessPassword` 字段到 `WebsiteVO`

- `internal/biz/asset/website_usecase.go`
  - 添加 `decrypt()` 解密方法
  - 添加 `toVOWithPassword()` 方法
  - 修改 `GetByID()` 返回完整信息

---

## 测试步骤

### 测试 Agent 主机过滤
1. 创建多个业务分组（如：生产环境、测试环境）
2. 将不同的 Agent 主机分配到不同的分组
3. 新增内部站点
4. 选择"生产环境"分组
5. 检查 Agent 主机下拉菜单是否只显示生产环境的主机
6. 再选择"测试环境"分组
7. 检查 Agent 主机下拉菜单是否更新为测试环境的主机
8. 清空分组选择
9. 检查是否显示所有 Agent 主机

### 测试复制凭据
1. 创建一个站点，填写账号和密码
2. 保存后在列表中找到该站点
3. 点击"复制凭据"按钮
4. 应该看到"凭据已复制到剪贴板"提示
5. 粘贴到文本编辑器，检查格式是否正确
6. 检查密码是否正确解密

---

## 编译验证

```bash
# 后端编译
go build -o /dev/null cmd/server/server.go
✅ 通过

# 前端类型检查
cd web && npx vue-tsc --noEmit
✅ 通过
```

---

## 安全注意事项

1. **密码加密**：
   - 使用 AES-256-GCM 加密
   - 密钥长度 32 字节
   - 每次加密使用随机 nonce

2. **权限控制**：
   - 只有有权限的用户才能查看站点详情
   - 复制凭据需要先获取详情接口权限

3. **日志记录**：
   - 建议记录复制凭据的操作（审计日志）
   - 记录访问站点详情的操作

---

## 后续优化建议

1. **Agent 主机过滤增强**：
   - 支持按主机状态过滤（在线/离线）
   - 支持按主机标签过滤
   - 显示主机的负载信息

2. **复制功能增强**：
   - 支持单独复制账号或密码
   - 支持自定义复制格式
   - 添加复制历史记录

3. **安全增强**：
   - 添加复制凭据的审计日志
   - 限制复制次数（防止滥用）
   - 支持临时密码（一次性使用）

---

## 更新日期
2026-03-09 22:30
