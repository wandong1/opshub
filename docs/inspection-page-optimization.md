# 智能巡检页面优化总结

## 优化时间
2026-03-12

## 优化目标
将巡检组管理和巡检项管理合并为一个页面，采用类似拨测管理中"业务流程"的实现方式，支持在巡检组下添加多个可拖拽排序的巡检项。

## 优化内容

### 一、前端页面重构

#### 1. 新增合并页面
**文件**：`web/src/views/inspection/InspectionManagement.vue`

**功能特性**：
- ✅ 巡检组列表展示（表格形式）
- ✅ 巡检组 CRUD 操作
- ✅ 巡检项嵌套编辑（在巡检组表单中）
- ✅ 巡检项可拖拽排序（拖拽手柄 + 拖放事件）
- ✅ 巡检项折叠/展开（点击标题切换）
- ✅ 支持三种执行类型：命令、脚本、PromQL
- ✅ 支持 9 种断言类型
- ✅ 支持变量提取配置
- ✅ 测试运行功能
- ✅ 统计卡片展示

**UI 设计参考**：
- 参考拨测管理的"业务流程"模式
- 使用 Arco Design Vue 组件库
- 拖拽手柄图标：`icon-drag-dot-vertical`
- 步骤序号：圆形徽章显示
- 折叠/展开图标：`icon-down` / `icon-up`

#### 2. 保留的独立页面
**文件**：`web/src/views/inspection/InspectionRecords.vue`

**原因**：执行记录是查询展示页面，不需要合并到巡检管理中

#### 3. 废弃的页面
- ~~`InspectionGroups.vue`~~ - 已合并到 InspectionManagement.vue
- ~~`InspectionItems.vue`~~ - 已合并到 InspectionManagement.vue

### 二、路由配置更新

**文件**：`web/src/router/index.ts`

**变更**：
```javascript
// 删除
- /inspection/groups → InspectionGroups.vue
- /inspection/items → InspectionItems.vue

// 新增
+ /inspection/management → InspectionManagement.vue

// 保留
✓ /inspection/records → InspectionRecords.vue
```

### 三、数据库菜单结构优化

#### 1. 菜单变更
**智能巡检模块（ID 352）子菜单**：
```
├── 拨测管理 (ID=353)
├── 任务调度 (ID=354)
├── Pushgateway配置 (ID=355)
├── 环境变量 (ID=356)
├── 巡检管理 (ID=389) ← 合并了巡检组和巡检项
└── 执行记录 (ID=391)
```

**删除的菜单**：
- ~~ID 390: 巡检项管理~~（已合并到 ID 389）

#### 2. 按钮权限优化
**巡检管理（ID 389）的按钮权限**：
- ID 392: 新增巡检组 (`inspection_management:create_group`)
- ID 393: 编辑巡检组 (`inspection_management:update_group`)
- ID 394: 删除巡检组 (`inspection_management:delete_group`)
- ID 395: 新增巡检项 (`inspection_management:create_item`)
- ID 396: 编辑巡检项 (`inspection_management:update_item`)
- ID 397: 删除巡检项 (`inspection_management:delete_item`)
- ID 398: 测试运行 (`inspection_management:test`)

**权限码变更**：
```
旧权限码：
- inspection_groups:create
- inspection_groups:update
- inspection_groups:delete
- inspection_items:create
- inspection_items:update
- inspection_items:delete
- inspection_items:test

新权限码：
- inspection_management:create_group
- inspection_management:update_group
- inspection_management:delete_group
- inspection_management:create_item
- inspection_management:update_item
- inspection_management:delete_item
- inspection_management:test
```

### 四、数据库脚本

#### 1. 优化脚本
**文件**：`migrations/optimize_inspection_menu.sql`

**功能**：
- 删除旧的巡检组管理和巡检项管理菜单
- 创建新的巡检管理菜单
- 更新按钮权限和 API 关联
- 更新角色权限分配

**执行方式**：
```bash
mysql -u root -p opshub < migrations/optimize_inspection_menu.sql
```

#### 2. 初始化脚本更新
**文件**：`migrations/init.sql`

**变更**：已同步优化后的菜单结构

### 五、核心功能实现

#### 1. 巡检项拖拽排序
```javascript
// 拖拽开始
const onItemDragStart = (index: number) => {
  itemDragIndex.value = index
}

// 拖拽悬停
const onItemDragOver = (index: number) => {
  if (itemDragIndex.value === -1 || itemDragIndex.value === index) return

  // 交换位置
  const dragItem = inspectionItems.value[itemDragIndex.value]
  inspectionItems.value.splice(itemDragIndex.value, 1)
  inspectionItems.value.splice(index, 0, dragItem)

  itemDragIndex.value = index
}

// 拖拽结束
const onItemDrop = (index: number) => {
  itemDragIndex.value = -1
}
```

#### 2. 巡检项折叠/展开
```javascript
// 点击标题切换展开状态
<div class="inspection-item-header" @click="activeItemIndex = activeItemIndex === index ? -1 : index">
  <!-- 标题内容 -->
  <icon-down v-if="activeItemIndex !== index" />
  <icon-up v-else />
</div>

// 根据状态显示/隐藏内容
<div v-show="activeItemIndex === index" class="inspection-item-body">
  <!-- 巡检项详细配置 -->
</div>
```

#### 3. 执行类型切换
```javascript
// 根据执行类型显示不同的配置项
<template v-if="item.executionType === 'command'">
  <a-textarea v-model="item.command" placeholder="如：uptime" />
</template>

<template v-if="item.executionType === 'script'">
  <a-select v-model="item.scriptType">
    <a-option value="shell">Shell</a-option>
    <a-option value="python">Python</a-option>
  </a-select>
  <a-textarea v-model="item.scriptContent" />
</template>

<template v-if="item.executionType === 'promql'">
  <a-textarea v-model="item.promqlQuery" />
</template>
```

### 六、样式设计

#### 1. 巡检项卡片样式
```css
.inspection-item-card {
  margin-bottom: 12px;
  border: 1px solid var(--ops-border-color);
  border-radius: 8px;
  background: #fff;
  cursor: move;
  transition: all 0.2s;
}

.inspection-item-card:hover {
  border-color: var(--ops-primary);
  box-shadow: 0 2px 8px rgba(22, 93, 255, 0.1);
}
```

#### 2. 拖拽手柄
```css
.inspection-item-drag-handle {
  cursor: move;
  color: var(--ops-text-tertiary);
  font-size: 16px;
}
```

#### 3. 步骤序号
```css
.inspection-item-index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--ops-primary);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}
```

### 七、使用说明

#### 1. 执行数据库优化
```bash
cd /Users/Zhuanz/golang_project/src/opshub
mysql -u root -p opshub < migrations/optimize_inspection_menu.sql
```

#### 2. 重启服务
```bash
# 后端（如果需要）
make run

# 前端
cd web
npm run dev
```

#### 3. 清除浏览器缓存
- 打开开发者工具（F12）
- 右键刷新按钮 → "清空缓存并硬性重新加载"

#### 4. 访问新页面
登录后在左侧菜单找到：
```
智能巡检
  └── 巡检管理 ← 新的合并页面
```

### 八、操作流程

#### 1. 创建巡检组
1. 点击"新增巡检组"按钮
2. 填写巡检组基本信息
3. 点击"添加巡检项"按钮
4. 配置巡检项详情
5. 可添加多个巡检项
6. 拖拽调整巡检项顺序
7. 提交保存

#### 2. 编辑巡检组
1. 点击表格中的"编辑"按钮
2. 修改巡检组信息
3. 展开/折叠巡检项进行编辑
4. 拖拽调整顺序
5. 添加或删除巡检项
6. 提交保存

#### 3. 测试运行
1. 点击表格中的"测试运行"按钮
2. 系统执行该巡检组下的所有巡检项
3. 查看执行结果

### 九、待实现功能

#### 1. API 接口集成
- [ ] 实现巡检组 CRUD API 调用
- [ ] 实现巡检项批量保存 API
- [ ] 实现测试运行 API 调用
- [ ] 实现巡检项排序保存

#### 2. 数据加载
- [ ] 加载巡检组列表
- [ ] 加载巡检组详情时同时加载巡检项
- [ ] 巡检项按 sort 字段排序

#### 3. 表单验证
- [ ] 巡检组名称必填
- [ ] 巡检项名称必填
- [ ] 执行类型必选
- [ ] 根据执行类型验证对应字段

#### 4. 用户体验优化
- [ ] 拖拽时显示拖拽预览
- [ ] 保存时显示加载状态
- [ ] 删除前二次确认
- [ ] 测试运行显示进度

### 十、技术亮点

1. **拖拽排序**：使用原生 HTML5 拖放 API，无需第三方库
2. **折叠展开**：通过 `v-show` 控制，性能优秀
3. **动态表单**：根据执行类型动态显示配置项
4. **嵌套编辑**：在巡检组表单中直接编辑巡检项
5. **权限控制**：使用 `v-permission` 指令控制按钮显示
6. **样式统一**：遵循 Arco Design 设计规范

### 十一、文件清单

#### 新增文件
```
web/src/views/inspection/
└── InspectionManagement.vue (新增 - 合并页面)

migrations/
└── optimize_inspection_menu.sql (新增 - 优化脚本)
```

#### 修改文件
```
web/src/router/index.ts (修改 - 路由配置)
migrations/init.sql (修改 - 菜单结构)
```

#### 废弃文件（可选删除）
```
web/src/views/inspection/
├── InspectionGroups.vue (废弃)
└── InspectionItems.vue (废弃)
```

### 十二、对比优化前后

#### 优化前
- 2 个独立页面（巡检组管理 + 巡检项管理）
- 需要在两个页面间切换
- 巡检项无法直观看到所属巡检组
- 无法拖拽调整巡检项顺序

#### 优化后
- 1 个合并页面（巡检管理）
- 在一个页面完成所有操作
- 巡检项嵌套在巡检组中，层级清晰
- 支持拖拽调整巡检项执行顺序
- 参考拨测管理的成熟模式

### 十三、注意事项

1. **权限码变更**：前端需要更新权限指令中的权限码
2. **API 兼容性**：后端 API 保持不变，仅前端页面合并
3. **数据结构**：巡检项的 `sort` 字段用于保存拖拽后的顺序
4. **浏览器兼容**：HTML5 拖放 API 需要现代浏览器支持
5. **性能考虑**：巡检项过多时考虑虚拟滚动优化

## 总结

本次优化将巡检组管理和巡检项管理合并为一个页面，采用类似拨测管理"业务流程"的实现方式，支持拖拽排序，提升了用户体验和操作效率。优化后的页面更加直观，层级关系更加清晰，符合用户的操作习惯。
