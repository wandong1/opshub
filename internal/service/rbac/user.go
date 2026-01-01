package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

type UserService struct {
	userUseCase *rbac.UserUseCase
	authService *AuthService
}

func NewUserService(userUseCase *rbac.UserUseCase, authService *AuthService) *UserService {
	return &UserService{
		userUseCase: userUseCase,
		authService: authService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string       `json:"token"`
	User  *rbac.SysUser `json:"user"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	RealName string `json:"realName"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
}

// Login 用户登录
func (s *UserService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 添加调试日志
	appLogger.Info("用户登录尝试", zap.String("username", req.Username))

	user, err := s.userUseCase.ValidatePassword(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		appLogger.Error("登录失败", zap.String("username", req.Username), zap.Error(err))
		response.ErrorCode(c, http.StatusUnauthorized, err.Error())
		return
	}

	if user.Status != 1 {
		response.ErrorCode(c, http.StatusForbidden, "用户已被禁用")
		return
	}

	token, err := s.authService.GenerateToken(user.ID, user.Username)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "生成token失败")
		return
	}

	// 清空密码字段，防止返回给前端
	user.Password = ""

	// 更新最后登录时间
	_ = s.userUseCase.Update(c.Request.Context(), user)

	appLogger.Info("用户登录成功", zap.String("username", req.Username))

	response.Success(c, LoginResponse{
		Token: token,
		User:  user,
	})
}

// Register 用户注册
func (s *UserService) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	user := &rbac.SysUser{
		Username: req.Username,
		Password: req.Password,
		RealName: req.RealName,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   1,
	}

	if err := s.userUseCase.Create(c.Request.Context(), user); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "注册失败: "+err.Error())
		return
	}

	response.Success(c, user)
}

// GetProfile 获取当前用户信息
func (s *UserService) GetProfile(c *gin.Context) {
	userID := GetUserID(c)
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未登录")
		return
	}

	user, err := s.userUseCase.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 清空密码字段，防止返回给前端
	user.Password = ""

	response.Success(c, user)
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// ChangePassword 修改自己的密码
func (s *UserService) ChangePassword(c *gin.Context) {
	userID := GetUserID(c)
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未登录")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.UpdatePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.SuccessWithMessage(c, "密码修改成功", nil)
}

// CreateUser 创建用户
func (s *UserService) CreateUser(c *gin.Context) {
	var req rbac.SysUser
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	// 清空密码字段，防止返回给前端
	req.Password = ""

	response.Success(c, req)
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req rbac.SysUser
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.userUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, req)
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	if err := s.userUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetUser 获取用户详情
func (s *UserService) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	user, err := s.userUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "用户不存在")
		return
	}

	// 清空密码字段，防止返回给前端
	user.Password = ""

	response.Success(c, user)
}

// ListUsers 用户列表
func (s *UserService) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	users, total, err := s.userUseCase.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	// 清空所有用户的密码字段，防止返回给前端
	for _, user := range users {
		user.Password = ""
	}

	response.Success(c, gin.H{
		"list":     users,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AssignUserRoles 分配用户角色
type AssignUserRolesRequest struct {
	RoleIDs []uint `json:"roleIds" binding:"required"`
}

func (s *UserService) AssignUserRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req AssignUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.AssignRoles(c.Request.Context(), uint(id), req.RoleIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "分配失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// ResetPassword 重置用户密码
func (s *UserService) ResetPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.userUseCase.ResetPassword(c.Request.Context(), uint(id), req.Password); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "重置密码失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "密码重置成功", nil)
}
