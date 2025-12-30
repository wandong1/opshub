package rbac

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type UserUseCase struct {
	userRepo UserRepo
}

func NewUserUseCase(userRepo UserRepo) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

func (uc *UserUseCase) Create(ctx context.Context, user *SysUser) error {
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	return uc.userRepo.Create(ctx, user)
}

func (uc *UserUseCase) Update(ctx context.Context, user *SysUser) error {
	return uc.userRepo.Update(ctx, user)
}

func (uc *UserUseCase) Delete(ctx context.Context, id uint) error {
	return uc.userRepo.Delete(ctx, id)
}

func (uc *UserUseCase) GetByID(ctx context.Context, id uint) (*SysUser, error) {
	return uc.userRepo.GetByID(ctx, id)
}

func (uc *UserUseCase) GetByUsername(ctx context.Context, username string) (*SysUser, error) {
	return uc.userRepo.GetByUsername(ctx, username)
}

func (uc *UserUseCase) List(ctx context.Context, page, pageSize int, keyword string) ([]*SysUser, int64, error) {
	return uc.userRepo.List(ctx, page, pageSize, keyword)
}

func (uc *UserUseCase) AssignRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return uc.userRepo.AssignRoles(ctx, userID, roleIDs)
}

func (uc *UserUseCase) ValidatePassword(ctx context.Context, username, password string) (*SysUser, error) {
	user, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("密码错误")
	}

	return user, nil
}

func (uc *UserUseCase) UpdatePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
	if err != nil {
		return errors.New("原密码错误")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	return uc.userRepo.Update(ctx, user)
}

func (uc *UserUseCase) ResetPassword(ctx context.Context, userID uint, newPassword string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	return uc.userRepo.Update(ctx, user)
}

type RoleUseCase struct {
	roleRepo RoleRepo
}

func NewRoleUseCase(roleRepo RoleRepo) *RoleUseCase {
	return &RoleUseCase{
		roleRepo: roleRepo,
	}
}

func (uc *RoleUseCase) Create(ctx context.Context, role *SysRole) error {
	return uc.roleRepo.Create(ctx, role)
}

func (uc *RoleUseCase) Update(ctx context.Context, role *SysRole) error {
	return uc.roleRepo.Update(ctx, role)
}

func (uc *RoleUseCase) Delete(ctx context.Context, id uint) error {
	return uc.roleRepo.Delete(ctx, id)
}

func (uc *RoleUseCase) GetByID(ctx context.Context, id uint) (*SysRole, error) {
	return uc.roleRepo.GetByID(ctx, id)
}

func (uc *RoleUseCase) List(ctx context.Context, page, pageSize int, keyword string) ([]*SysRole, int64, error) {
	return uc.roleRepo.List(ctx, page, pageSize, keyword)
}

func (uc *RoleUseCase) GetAll(ctx context.Context) ([]*SysRole, error) {
	return uc.roleRepo.GetAll(ctx)
}

func (uc *RoleUseCase) AssignMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	return uc.roleRepo.AssignMenus(ctx, roleID, menuIDs)
}

func (uc *RoleUseCase) GetByUserID(ctx context.Context, userID uint) ([]*SysRole, error) {
	return uc.roleRepo.GetByUserID(ctx, userID)
}

type DepartmentUseCase struct {
	deptRepo DepartmentRepo
}

func NewDepartmentUseCase(deptRepo DepartmentRepo) *DepartmentUseCase {
	return &DepartmentUseCase{
		deptRepo: deptRepo,
	}
}

func (uc *DepartmentUseCase) Create(ctx context.Context, dept *SysDepartment) error {
	return uc.deptRepo.Create(ctx, dept)
}

func (uc *DepartmentUseCase) Update(ctx context.Context, dept *SysDepartment) error {
	return uc.deptRepo.Update(ctx, dept)
}

func (uc *DepartmentUseCase) Delete(ctx context.Context, id uint) error {
	return uc.deptRepo.Delete(ctx, id)
}

func (uc *DepartmentUseCase) GetByID(ctx context.Context, id uint) (*SysDepartment, error) {
	return uc.deptRepo.GetByID(ctx, id)
}

func (uc *DepartmentUseCase) GetTree(ctx context.Context) ([]*SysDepartment, error) {
	return uc.deptRepo.GetTree(ctx)
}

type MenuUseCase struct {
	menuRepo MenuRepo
}

func NewMenuUseCase(menuRepo MenuRepo) *MenuUseCase {
	return &MenuUseCase{
		menuRepo: menuRepo,
	}
}

func (uc *MenuUseCase) Create(ctx context.Context, menu *SysMenu) error {
	return uc.menuRepo.Create(ctx, menu)
}

func (uc *MenuUseCase) Update(ctx context.Context, menu *SysMenu) error {
	return uc.menuRepo.Update(ctx, menu)
}

func (uc *MenuUseCase) Delete(ctx context.Context, id uint) error {
	return uc.menuRepo.Delete(ctx, id)
}

func (uc *MenuUseCase) GetByID(ctx context.Context, id uint) (*SysMenu, error) {
	return uc.menuRepo.GetByID(ctx, id)
}

func (uc *MenuUseCase) GetTree(ctx context.Context) ([]*SysMenu, error) {
	return uc.menuRepo.GetTree(ctx)
}

func (uc *MenuUseCase) GetByUserID(ctx context.Context, userID uint) ([]*SysMenu, error) {
	return uc.menuRepo.GetByUserID(ctx, userID)
}

func (uc *MenuUseCase) GetByRoleID(ctx context.Context, roleID uint) ([]*SysMenu, error) {
	return uc.menuRepo.GetByRoleID(ctx, roleID)
}
