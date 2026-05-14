# 拨测管理 - 批量删除功能实现文档

## 功能概述

为拨测管理页面添加批量删除功能，允许用户一次性删除多个选中的拨测配置。

## 修改内容

### 1. 后端修改

#### 1.1 UseCase 层 - 添加批量删除方法

**文件**: `internal/biz/inspection/probe_config_usecase.go`

```go
func (uc *ProbeConfigUseCase) BatchDelete(ctx context.Context, ids []uint) error {
	for _, id := range ids {
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
```

**说明**：
- 遍历传入的 ID 列表，逐个调用 `Delete` 方法
- 如果任何一个删除失败，立即返回错误
- 复用现有的 `Delete` 方法，保持删除逻辑一致

#### 1.2 Service 层 - 添加批量删除处理器

**文件**: `internal/service/inspection/probe_config_service.go`

```go
func (s *ProbeConfigService) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.useCase.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "批量删除失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": fmt.Sprintf("成功删除 %d 个拨测配置", len(req.IDs))})
}
```

**说明**：
- 接收 JSON 格式的请求体，包含要删除的 ID 列表
- 使用 Gin 的 binding 标签进行参数验证（至少包含一个 ID）
- 返回成功消息，包含删除的数量

#### 1.3 路由注册

**文件**: `internal/server/inspection/http.go`

```go
probes := inspection.Group("/probes")
{
    probes.GET("", s.probeConfigService.List)
    probes.POST("", s.probeConfigService.Create)
    probes.POST("/test", s.probeConfigService.TestProbe)
    probes.POST("/batch-delete", s.probeConfigService.BatchDelete)  // 新增
    probes.GET("/:id", s.probeConfigService.Get)
    probes.PUT("/:id", s.probeConfigService.Update)
    probes.DELETE("/:id", s.probeConfigService.Delete)
    probes.POST("/import", s.probeConfigService.Import)
    probes.GET("/export", s.probeConfigService.Export)
    probes.POST("/:id/run", s.probeConfigService.RunOnce)
}
```

**说明**：
- 路由路径：`POST /api/v1/inspection/probes/batch-delete`
- 放在 `/test` 路由之后，避免路由冲突

### 2. 前端修改

#### 2.1 API 层 - 添加批量删除方法

**文件**: `web/src/api/networkProbe.ts`

```typescript
export const batchDeleteProbes = (ids: number[]) => {
  return request.post('/api/v1/inspection/probes/batch-delete', { ids })
}
```

**说明**：
- 接收 ID 数组作为参数
- 发送 POST 请求到批量删除接口

#### 2.2 页面层 - 添加批量删除 UI 和逻辑

**文件**: `web/src/views/inspection/ProbeManagement.vue`

**1. 导入新的 API 和图标**：
```typescript
import { IconDelete } from '@arco-design/web-vue/es/icon'
import { batchDeleteProbes } from '@/api/networkProbe'
```

**2. 添加批量删除按钮**：
```vue
<a-button v-if="selectedIds.length > 0" v-permission="'inspection:probes:delete'" status="danger" @click="handleBatchDelete">
  <template #icon><icon-delete /></template>批量删除 ({{ selectedIds.length }})
</a-button>
```

**3. 添加表格行选择功能**：
```vue
<a-table 
  :data="tableData" 
  :loading="loading" 
  :row-selection="{ type: 'checkbox', selectedRowKeys: selectedIds, onlyCurrent: false }" 
  @select="handleSelect" 
  @select-all="handleSelectAll"
  ...
>
```

**4. 添加响应式数据**：
```typescript
const selectedIds = ref<number[]>([])
```

**5. 添加处理函数**：
```typescript
const handleSelect = (rowKeys: number[]) => {
  selectedIds.value = rowKeys
}

const handleSelectAll = (checked: boolean) => {
  if (checked) {
    selectedIds.value = tableData.value.map((item: any) => item.id)
  } else {
    selectedIds.value = []
  }
}

const handleBatchDelete = () => {
  Modal.confirm({
    title: '确认批量删除',
    content: `确定要删除选中的 ${selectedIds.value.length} 个拨测配置吗？`,
    onOk: async () => {
      try {
        await batchDeleteProbes(selectedIds.value)
        Message.success(`成功删除 ${selectedIds.value.length} 个拨测配置`)
        selectedIds.value = []
        loadData()
      } catch (error: any) {
        Message.error(error.message || '批量删除失败')
      }
    }
  })
}
```

**说明**：
- `handleSelect`: 处理单行选择
- `handleSelectAll`: 处理全选/取消全选
- `handleBatchDelete`: 批量删除确认和执行
- 删除成功后自动刷新列表并清空选中状态

## API 接口文档

### 批量删除拨测配置

**接口地址**: `POST /api/v1/inspection/probes/batch-delete`

**请求参数**:
```json
{
  "ids": [1, 2, 3]
}
```

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| ids | array | 是 | 要删除的拨测配置 ID 列表，至少包含一个 ID |

**响应示例**:

成功响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "成功删除 3 个拨测配置"
  },
  "timestamp": 1234567890
}
```

失败响应：
```json
{
  "code": 500,
  "message": "批量删除失败: record not found",
  "timestamp": 1234567890
}
```

## 功能特性

1. **权限控制**: 批量删除按钮使用 `v-permission="'inspection:probes:delete'"` 指令控制权限
2. **动态显示**: 只有选中拨测配置时才显示批量删除按钮
3. **选中计数**: 按钮上显示当前选中的数量
4. **用户友好**: 
   - 删除前弹出确认对话框
   - 显示要删除的数量
   - 删除成功后显示成功消息
   - 自动刷新列表
   - 清空选中状态
5. **性能优化**: 使用单次 API 调用替代多次并发请求
6. **跨页选择**: 支持跨页面选择（`onlyCurrent: false`）

## 使用说明

1. 在拨测管理页面，勾选需要删除的拨测配置
2. 点击页面顶部的"批量删除"按钮（显示选中数量）
3. 在确认对话框中点击"确定"
4. 等待删除完成，系统会显示成功消息并刷新列表

## 注意事项

1. 批量删除操作不可逆，请谨慎操作
2. 删除拨测配置不会影响已执行的拨测结果记录
3. 如果某个拨测配置正在被定时任务使用，删除后任务将无法执行
4. 建议在删除前先导出配置作为备份

## 测试验证

### 后端测试
```bash
# 编译验证
go build -o /dev/null ./cmd/... ./internal/... ./pkg/... ./plugins/...
```

### 前端测试
```bash
# 类型检查
cd web && npx vue-tsc --noEmit
```

### 功能测试
1. 创建多个测试拨测配置
2. 勾选多个拨测配置
3. 点击批量删除按钮
4. 验证删除成功且列表刷新
5. 验证选中状态被清空

## 相关文件清单

### 后端文件
- `internal/biz/inspection/probe_config_usecase.go` - 添加批量删除方法
- `internal/service/inspection/probe_config_service.go` - 添加批量删除处理器
- `internal/server/inspection/http.go` - 添加批量删除路由

### 前端文件
- `web/src/api/networkProbe.ts` - 添加批量删除 API 方法
- `web/src/views/inspection/ProbeManagement.vue` - 添加批量删除 UI 和逻辑

## 与巡检管理批量删除的对比

| 特性 | 巡检管理 | 拨测管理 |
|------|----------|----------|
| 后端架构 | Service → UseCase → Repository | Service → UseCase → Repository |
| 路由路径 | `/api/v1/inspection/groups/batch-delete` | `/api/v1/inspection/probes/batch-delete` |
| 前端组件库 | Arco Design Vue | Arco Design Vue |
| 权限指令 | `v-permission` | `v-permission` |
| 跨页选择 | 支持 | 支持 |
| 实现方式 | 一致 | 一致 |

## 版本信息

- 实现日期: 2026-05-14
- 开发者: Claude (Opus 4.6)
- 功能状态: 已完成并通过编译验证
- 关联功能: 巡检管理批量删除（参考实现）
