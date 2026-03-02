package agent

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// ListFiles Agent文件列表
func (s *HTTPServer) ListFiles(c *gin.Context) {
	hostID, as, err := s.getAgentStream(c)
	if err != nil {
		return
	}

	path := c.DefaultQuery("path", "/")
	requestID := uuid.New().String()

	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_FileRequest{
			FileRequest: &pb.FileRequest{
				RequestId: requestID,
				Action:    "list",
				Path:      path,
			},
		},
	})

	result, err := s.hub.WaitResponse(as, requestID, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"code": 504, "message": err.Error()})
		return
	}

	fileList, ok := result.(*pb.FileListResult)
	if !ok || fileList.Error != "" {
		msg := "文件列表获取失败"
		if fileList != nil {
			msg = fileList.Error
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	_ = hostID
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": fileList.Files})
}

// UploadFile Agent文件上传
func (s *HTTPServer) UploadFile(c *gin.Context) {
	_, as, err := s.getAgentStream(c)
	if err != nil {
		return
	}

	path := c.PostForm("path")
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件参数错误"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文件失败"})
		return
	}

	requestID := uuid.New().String()
	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_FileRequest{
			FileRequest: &pb.FileRequest{
				RequestId: requestID,
				Action:    "upload",
				Path:      path,
				Filename:  header.Filename,
				Data:      data,
			},
		},
	})

	result, err := s.hub.WaitResponse(as, requestID, 60*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"code": 504, "message": err.Error()})
		return
	}

	chunk, ok := result.(*pb.FileChunk)
	if !ok || (chunk != nil && chunk.Error != "") {
		msg := "上传失败"
		if chunk != nil {
			msg = chunk.Error
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "上传成功"})
}

// DownloadFile Agent文件下载
func (s *HTTPServer) DownloadFile(c *gin.Context) {
	_, as, err := s.getAgentStream(c)
	if err != nil {
		return
	}

	path := c.Query("path")
	requestID := uuid.New().String()

	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_FileRequest{
			FileRequest: &pb.FileRequest{
				RequestId: requestID,
				Action:    "download",
				Path:      path,
			},
		},
	})

	result, err := s.hub.WaitResponse(as, requestID, 60*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"code": 504, "message": err.Error()})
		return
	}

	chunk, ok := result.(*pb.FileChunk)
	if !ok || (chunk != nil && chunk.Error != "") {
		msg := "下载失败"
		if chunk != nil {
			msg = chunk.Error
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Writer.Write(chunk.Data)
}

// DeleteFile Agent文件删除
func (s *HTTPServer) DeleteFile(c *gin.Context) {
	_, as, err := s.getAgentStream(c)
	if err != nil {
		return
	}

	path := c.Query("path")
	requestID := uuid.New().String()

	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_FileRequest{
			FileRequest: &pb.FileRequest{
				RequestId: requestID,
				Action:    "delete",
				Path:      path,
			},
		},
	})

	result, err := s.hub.WaitResponse(as, requestID, 30*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"code": 504, "message": err.Error()})
		return
	}

	chunk, ok := result.(*pb.FileChunk)
	if !ok || (chunk != nil && chunk.Error != "") {
		msg := "删除失败"
		if chunk != nil {
			msg = chunk.Error
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// getAgentStream 获取Agent流的公共方法
func (s *HTTPServer) getAgentStream(c *gin.Context) (uint, *AgentStream, error) {
	hostIDStr := c.Param("hostId")
	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的主机ID"})
		return 0, nil, err
	}

	as, ok := s.hub.GetByHostID(uint(hostID))
	if !ok {
		err := fmt.Errorf("Agent不在线")
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return 0, nil, err
	}

	return uint(hostID), as, nil
}
