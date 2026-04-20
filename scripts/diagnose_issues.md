# 问题诊断和修复指南

## 问题 1：审计日志不消费

### 根本原因
`request_url` 字段类型为 `varchar(500)`，无法存储长 URL（现代 Web 应用 URL 经常超过 500 字符）

### 已修复
✅ 数据库字段已改为 `TEXT` 类型
✅ Worker 增强了错误日志和健壮性
✅ Worker 启动时立即执行一次

### 验证步骤
1. 重启服务：`docker-compose restart opshub-server`
2. 查看启动日志：
   ```bash
   docker logs -f opshub-server | grep -i "audit worker"
   ```
   应该看到：
   - "网站代理访问审计 worker 启动"
   - "网站代理访问审计 worker 首次执行"
   
3. 等待 1-2 分钟，检查队列长度：
   ```bash
   docker exec opshub-redis redis-cli -a '1ujasdJ67Ps' LLEN website:audit:queue
   ```
   队列长度应该逐渐减少

4. 检查数据库记录：
   ```bash
   docker exec -i opshub-mysql mysql -uroot -p'OpsHub@2026' opshub -e "SELECT COUNT(*) FROM website_proxy_audit_logs;"
   ```
   记录数应该逐渐增加

### 如果还不工作
查看详细错误日志：
```bash
docker logs opshub-server | grep -i "audit\|worker" | tail -50
```

---

## 问题 2：平台登录被挤掉

### 诊断步骤

#### 1. 确认平台认证机制
平台使用 **localStorage + JWT**，不使用 Cookie：
- Token 存储在：`localStorage.getItem('token')`
- 通过 HTTP Header 传递：`Authorization: Bearer {token}`

#### 2. 可能的原因

**原因 A：代理站点的 JS 清空了 localStorage**
- 检查方法：打开浏览器开发者工具 → Application → Local Storage
- 访问代理站点前后对比 `token` 是否还存在

**原因 B：代理站点设置了同名 Cookie 覆盖了某些关键信息**
- 检查方法：开发者工具 → Application → Cookies
- 查看是否有 `Path=/` 的 Cookie（这些会影响整个站点）

**原因 C：代理站点的 iframe 或重定向导致页面刷新**
- 检查方法：查看 Network 面板，是否有意外的页面跳转

#### 3. 测试步骤

1. **登录平台**
   - 打开开发者工具
   - 记录 `localStorage.token` 的值
   - 记录所有 Cookie

2. **访问代理站点**
   - 在代理站点登录
   - 操作一段时间

3. **返回平台**
   - 检查 `localStorage.token` 是否还存在
   - 检查 Cookie 是否有变化
   - 尝试刷新页面，看是否还能保持登录

#### 4. 临时解决方案

如果确认是 localStorage 被清空，可以：

**方案 A：使用 iframe 隔离**
修改前端，将代理站点放在 iframe 中：
```vue
<iframe 
  :src="proxyUrl" 
  sandbox="allow-same-origin allow-scripts allow-forms"
  style="width: 100%; height: 100vh; border: none;"
></iframe>
```

**方案 B：在新标签页打开**
当前已经是这样做的（`window.open(proxyUrl, '_blank')`）

**方案 C：添加 localStorage 保护**
在 `web/src/main.ts` 中添加：
```typescript
// 保护平台 token 不被删除
const originalRemoveItem = localStorage.removeItem.bind(localStorage)
const originalClear = localStorage.clear.bind(localStorage)

localStorage.removeItem = function(key: string) {
  if (key === 'token' && !confirm('确定要退出登录吗？')) {
    return
  }
  originalRemoveItem(key)
}

localStorage.clear = function() {
  const token = localStorage.getItem('token')
  originalClear()
  if (token) {
    localStorage.setItem('token', token)
  }
}
```

### Cookie 隔离已增强
✅ 所有代理站点 Cookie 设置 `Path=/api/v1/websites/proxy/t/{token}`
✅ 添加 `SameSite=Lax` 属性
✅ 删除 `Domain` 属性

---

## 快速验证命令

```bash
# 1. 检查审计队列
docker exec opshub-redis redis-cli -a '1ujasdJ67Ps' LLEN website:audit:queue

# 2. 检查审计记录数
docker exec -i opshub-mysql mysql -uroot -p'OpsHub@2026' opshub -e "SELECT COUNT(*) FROM website_proxy_audit_logs;"

# 3. 查看最近的审计日志
docker exec -i opshub-mysql mysql -uroot -p'OpsHub@2026' opshub -e "SELECT * FROM website_proxy_audit_logs ORDER BY id DESC LIMIT 5;"

# 4. 查看 Worker 日志
docker logs opshub-server 2>&1 | grep -i "audit worker" | tail -20

# 5. 手动触发 Worker（重启服务）
docker-compose restart opshub-server
```

---

## 预期结果

### 审计日志
- 队列长度逐渐减少（每分钟最多消费 100 条）
- 数据库记录逐渐增加
- 日志显示 "批量处理网站代理访问审计事件成功"

### Cookie 隔离
- 平台 Cookie：`Path=/`
- 代理站点 Cookie：`Path=/api/v1/websites/proxy/t/{token}; SameSite=Lax`
- 两者完全隔离，互不影响
