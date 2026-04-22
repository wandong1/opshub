# 智能巡检模块 Bug 修复文档

## 修复日期
2026-03-13

## 修复内容

### 1. 巡检组配置 - 添加业务分组多选功能

**问题描述**：
- 在巡检组的新增/编辑弹窗中，没有看到支持多选的业务分组选择器

**修复方案**：
- 在 `InspectionManagement.vue` 中添加了业务分组多选下拉框
- 使用 `a-select` 组件，支持 `multiple` 多选模式
- 通过 `getGroupTree()` API 获取资产分组树，并扁平化为列表供选择
- 选中的分组 ID 数组通过 `JSON.stringify()` 转换为字符串存储到 `groupIds` 字段

**修改文件**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 添加业务分组选择表单项
  - 添加 `assetGroups` 状态存储分组列表
  - 添加 `loadAssetGroups()` 方法加载分组数据
  - 修改 `handleCreate()` 和 `handleEdit()` 方法，在打开弹窗时加载分组
  - 修改 `handleSubmit()` 方法，将 `groupIds` 数组序列化为 JSON 字符串

### 2. 巡检项支持主机标签和主机名匹配

**问题描述**：
- 巡检项配置中缺少主机匹配方式选择
- 无法通过主机标签或主机名来筛选执行主机
- 主机来源应该是巡检组关联的业务分组下的主机

**修复方案**：
- 添加主机匹配方式选择：按标签匹配、按主机名匹配、按主机ID匹配
- 按标签匹配：输入标签关键词，从业务分组中筛选包含这些标签的主机
- 按主机名匹配：输入主机名关键词，从业务分组中筛选主机名包含关键词的主机
- 按主机ID匹配：从业务分组的主机列表中直接选择主机
- 对于 PromQL 类型，通过 Agent 执行；SSH 方式不支持 PromQL

**修改文件**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 添加主机匹配方式单选框（tag/name/id）
  - 添加对应的主机标签、主机名、主机ID选择器
  - 添加 `loadHostsForItem()` 方法，根据业务分组加载主机列表
  - 修改巡检项数据结构，添加 `hostMatchType`、`hostTags`、`hostIds` 字段

- `internal/service/inspection_mgmt/item_service.go`
  - 修改 `matchHosts()` 方法，支持按主机名匹配
  - 添加 `filterHostsByNames()` 方法，按主机名关键词过滤
  - 添加 `hostMatchesKeywords()` 辅助方法，实现不区分大小写的主机名匹配

- `internal/data/inspection_mgmt/inspection_item.go`
  - 更新 `HostMatchType` 字段注释，支持 tag/name/id 三种类型
  - 更新 `HostTags` 字段注释，说明用于存储标签或主机名关键词的 JSON 数组

### 3. 添加巡检项测试执行功能

**问题描述**：
- 每个巡检项上没有测试执行按钮
- 无法在配置时测试巡检项是否正确
- 缺少执行日志的实时查看功能

**修复方案**：
- 在每个巡检项配置区域底部添加"测试执行"按钮
- 点击后打开测试日志弹窗，异步执行测试
- 弹窗中实时显示执行结果，包括主机名、执行状态、输出内容、断言结果、执行耗时
- 支持在巡检管理列表中对整个巡检组进行测试运行

**修改文件**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 添加测试日志弹窗 `testLogVisible`
  - 添加 `handleTestRunItem()` 方法，测试单个巡检项
  - 修改 `handleTestRun()` 方法，测试整个巡检组
  - 添加 `getLogStatusColor()` 方法，根据状态显示不同颜色
  - 添加测试日志样式，包括日志容器、日志项、输出区域等

- `internal/service/inspection_mgmt/item_dto.go`
  - 修改 `TestRunRequest` 结构，添加 `GroupID` 字段
  - 修改 `TestRunResponse` 结构，添加 `HostIp` 字段

- `internal/service/inspection_mgmt/item_service.go`
  - 修改 `executeItem()` 方法，返回结果中包含主机 IP

### 4. 优化巡检管理列表展示

**问题描述**：
- 巡检管理列表显示过于单调
- 执行方式、巡检项数、创建时间等字段显示为空
- 没有展示巡检组下的巡检项名称

**修复方案**：
- 在列表中添加"巡检项"列，展示该巡检组下所有巡检项的名称
- 修复执行方式、巡检项数、创建时间字段的显示
- 后端返回 `itemCount` 和 `itemNames` 字段

**修改文件**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 添加"巡检项"列，显示巡检项名称列表
  - 调整列宽，优化显示效果

- `internal/service/inspection_mgmt/group_dto.go`
  - 在 `GroupResponse` 结构中添加 `ItemCount` 和 `ItemNames` 字段

- `internal/service/inspection_mgmt/group_service.go`
  - 修改构造函数，注入 `itemRepo`
  - 修改 `List()` 方法，查询每个巡检组的巡检项并填充到响应中
  - 修改 `toResponse()` 方法，初始化 `ItemCount` 和 `ItemNames` 字段

- `internal/server/inspection/inspection_mgmt_init.go`
  - 修改 `NewGroupService()` 调用，传入 `itemRepo` 参数

### 5. 优化操作按钮布局和图标

**问题描述**：
- 操作列的按钮挤在一起，不美观
- 缺少图标，不够直观

**修复方案**：
- 为每个操作按钮添加图标：测试（播放图标）、编辑（编辑图标）、删除（删除图标）
- 使用 Arco Design 的图标组件
- 调整按钮间距，使用 `a-space` 组件

**修改文件**：
- `web/src/views/inspection/InspectionManagement.vue`
  - 导入 `IconPlayArrow`、`IconEdit`、`IconDelete` 图标
  - 为操作按钮添加图标插槽
  - 调整操作列宽度为 220px

## 技术要点

### 前端
- 使用 Arco Design Vue 组件库
- 使用 `a-select` 的 `multiple` 模式实现多选
- 使用 `allow-create` 和 `allow-search` 支持自定义输入和搜索
- 使用 `a-modal` 实现测试日志弹窗
- 使用 `a-spin` 显示加载状态

### 后端
- 使用 JSON 格式存储数组数据（groupIds、hostTags、hostIds）
- 在 Service 层实现主机匹配逻辑
- 支持三种主机匹配方式：标签、主机名、主机ID
- 通过 Repository 层查询关联数据

## 测试建议

1. **业务分组多选测试**：
   - 创建巡检组时选择多个业务分组
   - 编辑巡检组时修改业务分组选择
   - 验证保存后数据正确存储

2. **主机匹配测试**：
   - 测试按标签匹配：输入标签，验证匹配到正确的主机
   - 测试按主机名匹配：输入主机名关键词，验证匹配逻辑
   - 测试按主机ID匹配：选择主机，验证执行时使用正确的主机

3. **测试执行功能测试**：
   - 在巡检项配置中点击"测试执行"按钮
   - 验证测试日志弹窗正确显示
   - 验证执行结果、输出内容、断言结果正确显示
   - 测试整个巡检组的测试运行功能

4. **列表展示测试**：
   - 验证巡检项名称正确显示
   - 验证执行方式、巡检项数、创建时间正确显示
   - 验证操作按钮图标和布局正确

## 注意事项

1. 业务分组 ID 以 JSON 数组格式存储，前端需要进行序列化和反序列化
2. 主机标签和主机 ID 也以 JSON 数组格式存储
3. 测试执行功能会临时创建禁用状态的巡检组用于测试
4. 主机名匹配支持不区分大小写的模糊匹配
5. PromQL 类型的巡检项需要通过 Agent 执行，SSH 方式不支持
