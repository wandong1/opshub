package rbac

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	appError "github.com/ydcloud-dy/opshub/pkg/error"
	"github.com/ydcloud-dy/opshub/pkg/response"
)

// ErrorResponse 错误响应辅助函数
func ErrorResponse(c *gin.Context, status int, message string) {
	var code appError.ErrorCode
	switch status {
	case http.StatusBadRequest:
		code = appError.ErrBadRequest
	case http.StatusUnauthorized:
		code = appError.ErrUnauthorized
	case http.StatusForbidden:
		code = appError.ErrForbidden
	case http.StatusNotFound:
		code = appError.ErrNotFound
	case http.StatusInternalServerError:
		code = appError.ErrInternalServer
	default:
		code = appError.ErrInternalServer
	}

	c.JSON(status, response.Response{
		Code:      int(code),
		Message:   message,
		Timestamp: 0,
	})
}

// AbortWithError 中止并返回错误
func AbortWithError(c *gin.Context, status int, message string) {
	ErrorResponse(c, status, message)
	c.Abort()
}

// Errorf 格式化错误
func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
