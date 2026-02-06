package utils

import (
	"gorm.io/gorm"
)

// APIBinding API绑定定义
type APIBinding struct {
	ApiMethod string
	ApiPath   string
}

// MenuPermission 菜单权限定义
type MenuPermission struct {
	Code      string
	Name      string
	ApiMethod string       // 单 API 向后兼容
	ApiPath   string       // 单 API 向后兼容
	APIs      []APIBinding // 多 API 绑定（优先使用）
	Sort      int
}

// EnsureMenuPermissions 幂等初始化菜单按钮权限
// parentCode: 父菜单的 code
// perms: 需要创建的按钮权限列表
func EnsureMenuPermissions(db *gorm.DB, parentCode string, perms []MenuPermission) error {
	// 查找父菜单
	var parentID uint
	err := db.Raw("SELECT id FROM sys_menu WHERE code = ? AND deleted_at IS NULL LIMIT 1", parentCode).Scan(&parentID).Error
	if err != nil || parentID == 0 {
		return nil // 父菜单不存在，跳过
	}

	for _, perm := range perms {
		// 检查是否已存在
		var menuID uint
		db.Raw("SELECT id FROM sys_menu WHERE code = ? AND deleted_at IS NULL LIMIT 1", perm.Code).Scan(&menuID)

		if menuID == 0 {
			// 创建按钮菜单
			db.Exec(`INSERT INTO sys_menu (name, code, type, parent_id, api_path, api_method, sort, visible, status, created_at, updated_at)
				VALUES (?, ?, 3, ?, ?, ?, ?, 1, 1, NOW(), NOW())`,
				perm.Name, perm.Code, parentID, perm.ApiPath, perm.ApiMethod, perm.Sort)
			db.Raw("SELECT id FROM sys_menu WHERE code = ? AND deleted_at IS NULL LIMIT 1", perm.Code).Scan(&menuID)
		}

		if menuID == 0 {
			continue
		}

		// 确定 API 列表
		var apis []APIBinding
		if len(perm.APIs) > 0 {
			apis = perm.APIs
		} else if perm.ApiPath != "" && perm.ApiMethod != "" {
			apis = []APIBinding{{ApiMethod: perm.ApiMethod, ApiPath: perm.ApiPath}}
		}

		// 幂等写入 sys_menu_api
		for _, api := range apis {
			var count int64
			db.Raw("SELECT COUNT(*) FROM sys_menu_api WHERE menu_id = ? AND api_path = ? AND api_method = ? AND deleted_at IS NULL",
				menuID, api.ApiPath, api.ApiMethod).Scan(&count)
			if count == 0 {
				db.Exec(`INSERT INTO sys_menu_api (menu_id, api_path, api_method, created_at, updated_at)
					VALUES (?, ?, ?, NOW(), NOW())`, menuID, api.ApiPath, api.ApiMethod)
			}
		}
	}

	return nil
}

// EnsureMenu 幂等创建菜单（如果不存在则创建）
// 返回菜单ID
func EnsureMenu(db *gorm.DB, code, name string, menuType int, parentCode string, path, component, icon string, sort int) uint {
	// 检查是否已存在
	var menuID uint
	db.Raw("SELECT id FROM sys_menu WHERE code = ? AND deleted_at IS NULL LIMIT 1", code).Scan(&menuID)
	if menuID > 0 {
		return menuID
	}

	// 查找父菜单ID
	var parentID uint
	if parentCode != "" {
		db.Raw("SELECT id FROM sys_menu WHERE code = ? AND deleted_at IS NULL LIMIT 1", parentCode).Scan(&parentID)
	}

	// 创建菜单
	db.Exec(`INSERT INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 1, NOW(), NOW())`,
		name, code, menuType, parentID, path, component, icon, sort)

	// 获取新创建的ID
	db.Raw("SELECT id FROM sys_menu WHERE code = ? AND deleted_at IS NULL LIMIT 1", code).Scan(&menuID)
	return menuID
}

// AssignMenusToAdminRole 将所有按钮菜单分配给admin角色
func AssignMenusToAdminRole(db *gorm.DB) error {
	// 获取admin角色ID
	var adminRoleID uint
	db.Raw("SELECT id FROM sys_role WHERE code = 'admin' AND deleted_at IS NULL LIMIT 1").Scan(&adminRoleID)
	if adminRoleID == 0 {
		return nil
	}

	// 获取所有未分配给admin的菜单
	db.Exec(`INSERT IGNORE INTO sys_role_menu (role_id, menu_id)
		SELECT ?, sm.id FROM sys_menu sm
		WHERE sm.deleted_at IS NULL
		AND sm.id NOT IN (SELECT menu_id FROM sys_role_menu WHERE role_id = ?)`,
		adminRoleID, adminRoleID)

	return nil
}
