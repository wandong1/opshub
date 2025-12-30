package rbac

import (
	"context"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"gorm.io/gorm"
)

type departmentRepo struct {
	db *gorm.DB
}

func NewDepartmentRepo(db *gorm.DB) rbac.DepartmentRepo {
	return &departmentRepo{db: db}
}

func (r *departmentRepo) Create(ctx context.Context, dept *rbac.SysDepartment) error {
	return r.db.WithContext(ctx).Create(dept).Error
}

func (r *departmentRepo) Update(ctx context.Context, dept *rbac.SysDepartment) error {
	return r.db.WithContext(ctx).Model(dept).Omit("created_at").Updates(dept).Error
}

func (r *departmentRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否有子部门
		var count int64
		if err := tx.Model(&rbac.SysDepartment{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrRegistered // 存在子部门，不能删除
		}

		// 删除部门
		return tx.Delete(&rbac.SysDepartment{}, id).Error
	})
}

func (r *departmentRepo) GetByID(ctx context.Context, id uint) (*rbac.SysDepartment, error) {
	var dept rbac.SysDepartment
	err := r.db.WithContext(ctx).First(&dept, id).Error
	return &dept, err
}

func (r *departmentRepo) GetTree(ctx context.Context) ([]*rbac.SysDepartment, error) {
	var departments []*rbac.SysDepartment
	err := r.db.WithContext(ctx).Order("sort ASC").Find(&departments).Error
	if err != nil {
		return nil, err
	}
	return r.buildTree(departments, 0), nil
}

func (r *departmentRepo) buildTree(departments []*rbac.SysDepartment, parentID uint) []*rbac.SysDepartment {
	var tree []*rbac.SysDepartment
	for _, dept := range departments {
		if dept.ParentID == parentID {
			children := r.buildTree(departments, dept.ID)
			if len(children) > 0 {
				dept.Children = children
			}
			tree = append(tree, dept)
		}
	}
	return tree
}
