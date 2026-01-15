package rbac

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/opshub/internal/biz/rbac"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

type PositionService struct {
	positionUseCase *rbac.PositionUseCase
}

func NewPositionService(positionUseCase *rbac.PositionUseCase) *PositionService {
	return &PositionService{
		positionUseCase: positionUseCase,
	}
}

// PositionListResponse 岗位列表响应
type PositionListResponse struct {
	ID         uint   `json:"id"`
	PostCode   string `json:"postCode"`
	PostName   string `json:"postName"`
	PostStatus int    `json:"postStatus"`
	CreateTime string `json:"createTime"`
	Remark     string `json:"remark"`
}

// toPositionListResponse 转换为岗位列表响应格式
func toPositionListResponse(position *rbac.SysPosition) PositionListResponse {
	return PositionListResponse{
		ID:         position.ID,
		PostCode:   position.PostCode,
		PostName:   position.PostName,
		PostStatus: position.PostStatus,
		CreateTime: position.CreatedAt.Format("2006-01-02 15:04:05"),
		Remark:     position.Remark,
	}
}

// CreatePosition 创建岗位
func (s *PositionService) CreatePosition(c *gin.Context) {
	var req rbac.SysPosition
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.positionUseCase.Create(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, toPositionListResponse(&req))
}

// UpdatePosition 更新岗位
func (s *PositionService) UpdatePosition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的岗位ID")
		return
	}

	var req rbac.SysPosition
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.positionUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, toPositionListResponse(&req))
}

// DeletePosition 删除岗位
func (s *PositionService) DeletePosition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的岗位ID")
		return
	}

	if err := s.positionUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetPosition 获取岗位详情
func (s *PositionService) GetPosition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的岗位ID")
		return
	}

	position, err := s.positionUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "岗位不存在")
		return
	}

	response.Success(c, toPositionListResponse(position))
}

// ListPositions 岗位列表
func (s *PositionService) ListPositions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	postCode := c.Query("postCode")
	postName := c.Query("postName")

	positions, total, err := s.positionUseCase.List(c.Request.Context(), page, pageSize, postCode, postName)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	list := make([]PositionListResponse, 0, len(positions))
	for _, pos := range positions {
		list = append(list, toPositionListResponse(pos))
	}

	response.Success(c, gin.H{
		"list":     list,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

// GetPositionUsers 获取岗位下的用户列表
func (s *PositionService) GetPositionUsers(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的岗位ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	users, total, err := s.positionUseCase.GetUsers(c.Request.Context(), uint(id), page, pageSize)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     users,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AssignUsersToPositionRequest 分配用户到岗位请求
type AssignUsersToPositionRequest struct {
	UserIDs []uint `json:"userIds" binding:"required"`
}

// AssignUsersToPosition 分配用户到岗位
func (s *PositionService) AssignUsersToPosition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的岗位ID")
		return
	}

	var req AssignUsersToPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := s.positionUseCase.AssignUsers(c.Request.Context(), uint(id), req.UserIDs); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "分配失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "分配成功", nil)
}

// RemoveUserFromPosition 移除岗位下的用户
func (s *PositionService) RemoveUserFromPosition(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的岗位ID")
		return
	}

	userIdStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIdStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的用户ID")
		return
	}

	if err := s.positionUseCase.RemoveUser(c.Request.Context(), uint(id), uint(userID)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "移除失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "移除成功", nil)
}
