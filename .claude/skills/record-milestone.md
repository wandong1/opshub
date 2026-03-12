# Record Milestone Skill

## 用途

自动记录项目开发过程中的需求、实现和里程碑到 `record.md` 文件。

## 使用场景

1. 完成一个功能模块后
2. 修复重要 bug 后
3. 完成重大重构后
4. 达成项目里程碑后

## 记录格式

### 需求记录
```markdown
## 📋 需求记录

### [YYYY-MM-DD] 需求标题
- **提出人**: 用户/产品经理
- **描述**: 详细需求描述
- **状态**: 已完成/进行中/待处理
```

### 实现记录
```markdown
## ✅ 实现记录

### [YYYY-MM-DD] 实现标题
- **开发者**: Claude
- **关联需求**: 需求标题或编号
- **修改文件**:
  - `path/to/file1.go` - 修改说明
  - `path/to/file2.ts` - 修改说明
- **技术要点**:
  - 技术点1
  - 技术点2
```

### 里程碑记录
```markdown
## 🎯 里程碑

### [YYYY-MM-DD] v1.x.x - 里程碑标题
- **参与人员**: 开发者列表
- **主要功能**:
  - 功能1
  - 功能2
- **技术亮点**:
  - 亮点1
  - 亮点2
```

## 使用方法

在完成工作后，调用此 skill 并提供以下信息：
- 记录类型（需求/实现/里程碑）
- 标题
- 详细内容

## 示例

```markdown
## ✅ 实现记录

### [2026-03-04] Agent 日志系统实现
- **开发者**: Claude
- **关联需求**: Agent 日志打印与自动清理
- **修改文件**:
  - `agent/internal/logger/logger.go` - 新增日志模块，支持轮转和多级别
  - `agent/internal/config/config.go` - 添加日志配置字段
  - `agent/cmd/main.go` - 集成日志初始化
  - `agent/internal/client/grpc_client.go` - 添加关键日志点
  - `internal/server/agent/agent_service.go` - 添加心跳接收日志
- **技术要点**:
  - 使用 lumberjack 实现日志轮转
  - 支持 Debug/Info/Warn/Error 四级日志
  - 双输出（文件 + stdout）
  - 自动压缩和清理（100MB/3备份/30天）
```

## 注意事项

1. 记录应该简洁明了，突出重点
2. 修改文件列表应包含主要文件，不必列出所有文件
3. 技术要点应聚焦核心技术和创新点
4. 每次记录自动追加到 `record.md` 文件末尾
5. 保持时间倒序（最新的在前）
