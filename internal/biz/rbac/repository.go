package rbac

import "context"

type UserRepo interface {
	Create(ctx context.Context, user *SysUser) error
	Update(ctx context.Context, user *SysUser) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysUser, error)
	GetByUsername(ctx context.Context, username string) (*SysUser, error)
	List(ctx context.Context, page, pageSize int, keyword string) ([]*SysUser, int64, error)
	AssignRoles(ctx context.Context, userID uint, roleIDs []uint) error
	UpdateLastLogin(ctx context.Context, userID uint) error
}

type RoleRepo interface {
	Create(ctx context.Context, role *SysRole) error
	Update(ctx context.Context, role *SysRole) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysRole, error)
	List(ctx context.Context, page, pageSize int, keyword string) ([]*SysRole, int64, error)
	GetAll(ctx context.Context) ([]*SysRole, error)
	AssignMenus(ctx context.Context, roleID uint, menuIDs []uint) error
	GetByUserID(ctx context.Context, userID uint) ([]*SysRole, error)
}

type DepartmentRepo interface {
	Create(ctx context.Context, dept *SysDepartment) error
	Update(ctx context.Context, dept *SysDepartment) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysDepartment, error)
	GetTree(ctx context.Context) ([]*SysDepartment, error)
}

type MenuRepo interface {
	Create(ctx context.Context, menu *SysMenu) error
	Update(ctx context.Context, menu *SysMenu) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysMenu, error)
	GetTree(ctx context.Context) ([]*SysMenu, error)
	GetByUserID(ctx context.Context, userID uint) ([]*SysMenu, error)
	GetByRoleID(ctx context.Context, roleID uint) ([]*SysMenu, error)
}
