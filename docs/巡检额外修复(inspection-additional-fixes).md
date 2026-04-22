# 智能巡检模块 - 补充修复文档

## 修复日期
2026-03-13

## 本次修复的问题

### 1. 添加巡检项执行策略配置

**问题描述**：
- 在巡检组配置中没有开放执行策略的配置选项
- 无法选择巡检项是并发执行还是顺序执行
- 无法配置并发数量

**修复方案**：
- 在巡检组配置中添加"执行策略"选项：并发执行/顺序执行
- 添加"并发数量"配置，仅在选择并发执行时显示
- 默认值：并发执行，并发数量 50

**修改文件**：

**后端**：
- `internal/data/inspection_mgmt/inspection_group.go`
  - 添加 `ExecutionStrategy` 字段（concurrent/sequential）
  - 添加 `Concurrency` 字段（并发数量，默认50）

- `internal/service/inspection_mgmt/group_dto.go`
  - 在 `GroupCreateRequest` 中添加 `ExecutionStrategy` 和 `Concurrency` 字段
  - 在 `GroupUpdateRequest` 中添加 `ExecutionStrategy` 和 `Concurrency` 字段
  - 在 `GroupResponse` 中添加 `ExecutionStrategy` 和 `Concurrency` 字段

- `internal/service/inspection_mgmt/group_service.go`
  - 修改 `Create()` 方法，处理执行策略和并发数量
  - 修改 `Update()` 方法，支持更新执行策略和并发数量
  - 修改 `toResponse()` 方法，返回执行策略和并发数量

**前端**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 在巡检项配置区域上方添加执行策略配置
  - 添加执行策略单选框（并发执行/顺序执行）
  - 添加并发数量输入框（仅在并发执行时显示）
  - 修改 `formData` 添加 `executionStrategy` 和 `concurrency` 字段
  - 修改 `handleCreate()` 和 `handleSubmit()` 方法处理新字段

- `web/src/api/inspectionManagement.ts`
  - 在 `InspectionGroup` 接口中添加 `executionStrategy` 和 `concurrency` 字段

### 2. 优化巡检管理列表展示

**问题描述**：
- 列表中执行方式、巡检项数、创建时间显示为空
- 缺少执行策略的展示
- 列表信息不够丰富

**修复方案**：
- 添加 ID 列，方便识别
- 添加执行策略列，显示并发/顺序及并发数量
- 调整列宽，优化显示效果
- 确保所有字段正确显示

**修改文件**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 添加 ID 列（宽度80px）
  - 添加执行策略列（宽度120px），显示并发/顺序标签
  - 并发执行时显示并发数量，如"并发(50)"
  - 调整巡检项列宽度为250px
  - 使用不同颜色区分执行策略：并发（蓝色）、顺序（橙色）

### 3. 优化操作列按钮排列

**问题描述**：
- 操作列按钮排列不够美观
- 按钮间距需要优化

**修复方案**：
- 使用 `a-space` 组件的 `:size="8"` 属性设置固定间距
- 确保按钮图标和文字对齐
- 保持操作列宽度为220px

**修改文件**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 修改操作列的 `a-space` 组件，添加 `:size="8"` 属性
  - 确保按钮间距统一

## 数据库变更

需要在 `inspection_groups` 表中添加以下字段：

```sql
ALTER TABLE inspection_groups
ADD COLUMN execution_strategy VARCHAR(20) DEFAULT 'concurrent' COMMENT '执行策略 concurrent/sequential',
ADD COLUMN concurrency INT DEFAULT 50 COMMENT '并发数量';
```

或者通过 GORM 自动迁移：
```go
db.AutoMigrate(&inspectionmgmtdata.InspectionGroup{})
```

## 功能说明

### 执行策略

1. **并发执行（concurrent）**：
   - 多个巡检项同时执行
   - 可配置并发数量（1-200）
   - 默认并发数量为50
   - 适用于大量巡检项，提高执行效率

2. **顺序执行（sequential）**：
   - 巡检项按顺序依次执行
   - 前一个执行完成后才执行下一个
   - 适用于有依赖关系的巡检项

### 列表展示优化

- **ID列**：显示巡检组的唯一标识
- **巡检项列**：显示该组下所有巡检项名称，用顿号分隔
- **执行方式列**：显示 SSH/Agent/自动
- **执行策略列**：显示并发/顺序，并发时显示并发数量
- **巡检项数列**：显示该组包含的巡检项数量
- **状态列**：显示启用/禁用状态
- **创建时间列**：显示创建时间

## 测试建议

1. **执行策略测试**：
   - 创建巡检组时选择并发执行，设置并发数量
   - 创建巡检组时选择顺序执行
   - 编辑巡检组，修改执行策略
   - 验证保存后数据正确存储

2. **列表展示测试**：
   - 验证所有列都正确显示数据
   - 验证执行策略列显示正确的标签和颜色
   - 验证并发数量在并发执行时正确显示
   - 验证创建时间格式正确

3. **按钮排列测试**：
   - 验证操作列按钮间距统一
   - 验证按钮图标和文字对齐
   - 验证按钮点击区域正常

## 技术要点

1. **条件渲染**：使用 `v-if` 控制并发数量输入框的显示
2. **动态标签颜色**：根据执行策略显示不同颜色的标签
3. **默认值处理**：后端在创建时设置默认值，确保数据完整性
4. **字段验证**：并发数量限制在1-200之间

## 注意事项

1. 执行策略配置在巡检组级别，影响该组下所有巡检项的执行方式
2. 并发数量仅在并发执行模式下生效
3. 顺序执行适用于有依赖关系的巡检项，但执行时间会较长
4. 数据库迁移需要在应用启动时自动执行
5. 前端需要正确处理执行策略和并发数量的序列化和反序列化
