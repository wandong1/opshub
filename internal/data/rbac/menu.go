// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package rbac

import (
	"context"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type menuRepo struct {
	db *gorm.DB
}

func NewMenuRepo(db *gorm.DB) rbac.MenuRepo {
	return &menuRepo{db: db}
}

func (r *menuRepo) Create(ctx context.Context, menu *rbac.SysMenu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

func (r *menuRepo) Update(ctx context.Context, menu *rbac.SysMenu) error {
	return r.db.WithContext(ctx).Model(&rbac.SysMenu{}).Where("id = ?", menu.ID).Updates(map[string]interface{}{
		"name":       menu.Name,
		"code":       menu.Code,
		"type":       menu.Type,
		"parent_id":  menu.ParentID,
		"path":       menu.Path,
		"component":  menu.Component,
		"icon":       menu.Icon,
		"sort":       menu.Sort,
		"visible":    menu.Visible,
		"status":     menu.Status,
		"api_path":   menu.ApiPath,
		"api_method": menu.ApiMethod,
	}).Error
}

func (r *menuRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.deleteRecursive(tx, id)
	})
}

// deleteRecursive 递归删除菜单及其所有子菜单
func (r *menuRepo) deleteRecursive(tx *gorm.DB, id uint) error {
	// 查找所有子菜单
	var childIDs []uint
	if err := tx.Model(&rbac.SysMenu{}).Where("parent_id = ?", id).Pluck("id", &childIDs).Error; err != nil {
		return err
	}

	// 递归删除子菜单
	for _, childID := range childIDs {
		if err := r.deleteRecursive(tx, childID); err != nil {
			return err
		}
	}

	// 删除角色菜单关联
	if err := tx.Where("menu_id = ?", id).Delete(&rbac.SysRoleMenu{}).Error; err != nil {
		return err
	}

	// 删除菜单API关联
	if err := tx.Where("menu_id = ?", id).Delete(&rbac.SysMenuAPI{}).Error; err != nil {
		return err
	}

	// 删除菜单本身
	return tx.Delete(&rbac.SysMenu{}, id).Error
}

func (r *menuRepo) GetByID(ctx context.Context, id uint) (*rbac.SysMenu, error) {
	var menu rbac.SysMenu
	err := r.db.WithContext(ctx).First(&menu, id).Error
	return &menu, err
}

func (r *menuRepo) GetTree(ctx context.Context) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Preload("APIs").
		Where("visible = ?", 1).
		Order("sort ASC").
		Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return r.buildTree(menus, 0), nil
}

func (r *menuRepo) buildTree(menus []*rbac.SysMenu, parentID uint) []*rbac.SysMenu {
	var tree []*rbac.SysMenu
	for _, menu := range menus {
		if menu.ParentID == parentID {
			children := r.buildTree(menus, menu.ID)
			if len(children) > 0 {
				menu.Children = children
			}
			tree = append(tree, menu)
		}
	}
	return tree
}

func (r *menuRepo) GetByUserID(ctx context.Context, userID uint) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Preload("APIs").
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role_menu.role_id").
		Where("sys_user_role.user_id = ? AND sys_menu.status = 1 AND sys_menu.visible = 1", userID).
		Distinct().
		Order("sys_menu.sort ASC").
		Find(&menus).Error

	if err != nil {
		return nil, err
	}

	return r.buildTree(menus, 0), nil
}

func (r *menuRepo) GetByRoleID(ctx context.Context, roleID uint) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Where("sys_role_menu.role_id = ?", roleID).
		Find(&menus).Error
	return menus, err
}

func (r *menuRepo) GetAPIPermissionsByUserID(ctx context.Context, userID uint) ([]rbac.APIPermission, error) {
	var perms []rbac.APIPermission
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT sma.api_path, sma.api_method
		FROM sys_menu_api sma
		JOIN sys_menu sm ON sm.id = sma.menu_id AND sm.deleted_at IS NULL
		JOIN sys_role_menu srm ON srm.menu_id = sm.id
		JOIN sys_user_role sur ON sur.role_id = srm.role_id
		WHERE sur.user_id = ? AND sm.status = 1 AND sma.deleted_at IS NULL
		UNION
		SELECT DISTINCT sm.api_path, sm.api_method
		FROM sys_menu sm
		JOIN sys_role_menu srm ON srm.menu_id = sm.id
		JOIN sys_user_role sur ON sur.role_id = srm.role_id
		WHERE sur.user_id = ? AND sm.api_path != '' AND sm.api_method != '' AND sm.status = 1
		AND sm.id NOT IN (SELECT DISTINCT menu_id FROM sys_menu_api WHERE deleted_at IS NULL)
	`, userID, userID).Scan(&perms).Error
	return perms, err
}

func (r *menuRepo) GetAllRegisteredAPIs(ctx context.Context) ([]rbac.APIPermission, error) {
	var perms []rbac.APIPermission
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT sma.api_path, sma.api_method
		FROM sys_menu_api sma
		JOIN sys_menu sm ON sm.id = sma.menu_id AND sm.deleted_at IS NULL
		WHERE sm.status = 1 AND sma.deleted_at IS NULL
		UNION
		SELECT DISTINCT api_path, api_method FROM sys_menu
		WHERE api_path != '' AND api_method != '' AND status = 1 AND deleted_at IS NULL
		AND id NOT IN (SELECT DISTINCT menu_id FROM sys_menu_api WHERE deleted_at IS NULL)
	`).Scan(&perms).Error
	return perms, err
}

func (r *menuRepo) GetMenuAPIs(ctx context.Context, menuID uint) ([]rbac.SysMenuAPI, error) {
	var apis []rbac.SysMenuAPI
	err := r.db.WithContext(ctx).Where("menu_id = ?", menuID).Find(&apis).Error
	return apis, err
}

func (r *menuRepo) SaveMenuAPIs(ctx context.Context, menuID uint, apis []rbac.SysMenuAPI) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除旧的
		if err := tx.Where("menu_id = ?", menuID).Delete(&rbac.SysMenuAPI{}).Error; err != nil {
			return err
		}
		// 批量插入新的
		if len(apis) > 0 {
			for i := range apis {
				apis[i].MenuID = menuID
				apis[i].ID = 0
			}
			return tx.Create(&apis).Error
		}
		return nil
	})
}

func (r *menuRepo) DeleteMenuAPIs(ctx context.Context, menuID uint) error {
	return r.db.WithContext(ctx).Where("menu_id = ?", menuID).Delete(&rbac.SysMenuAPI{}).Error
}
