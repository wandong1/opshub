package agent

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// ExecuteCommand Agent命令执行
func (s *HTTPServer) ExecuteCommand(c *gin.Context) {
	_, as, err := s.getAgentStream(c)
	if err != nil {
		return
	}

	var req struct {
		Command string `json:"command" binding:"required"`
		Timeout int    `json:"timeout"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = 60
	}

	requestID := uuid.New().String()
	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_CmdRequest{
			CmdRequest: &pb.CommandRequest{
				RequestId: requestID,
				Command:   req.Command,
				Timeout:   int32(req.Timeout),
			},
		},
	})

	result, err := s.hub.WaitResponse(as, requestID, time.Duration(req.Timeout+10)*time.Second)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"code": 504, "message": err.Error()})
		return
	}

	cmdResult, ok := result.(*pb.CommandResult)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "响应类型错误"})
		return
	}

	if cmdResult.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": cmdResult.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"exitCode": cmdResult.ExitCode,
			"stdout":   cmdResult.Stdout,
			"stderr":   cmdResult.Stderr,
		},
	})
}

// GetAllStatuses 获取所有Agent状态
func (s *HTTPServer) GetAllStatuses(c *gin.Context) {
	agents, err := s.grpcServer.AgentRepo().List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}

	statuses := make([]gin.H, 0, len(agents))
	for _, a := range agents {
		online := s.hub.IsOnline(a.HostID)
		status := a.Status
		if online {
			status = "online"
		} else if status == "online" {
			status = "offline"
		}
		statuses = append(statuses, gin.H{
			"hostId":   a.HostID,
			"agentId":  a.AgentID,
			"status":   status,
			"version":  a.Version,
			"lastSeen": a.LastSeen,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": statuses})
}

// GetAgentStatus 获取单个Agent状态
func (s *HTTPServer) GetAgentStatus(c *gin.Context) {
	_, as, _ := s.getAgentStream(c)
	hostIDStr := c.Param("hostId")

	agentInfo, err := s.grpcServer.AgentRepo().GetByHostID(c.Request.Context(), parseHostID(hostIDStr))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 0, "message": "success",
			"data": gin.H{"status": "none", "online": false},
		})
		return
	}

	online := as != nil
	status := agentInfo.Status
	if online {
		status = "online"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0, "message": "success",
		"data": gin.H{
			"agentId":  agentInfo.AgentID,
			"status":   status,
			"online":   online,
			"version":  agentInfo.Version,
			"hostname": agentInfo.Hostname,
			"os":       agentInfo.OS,
			"arch":     agentInfo.Arch,
			"lastSeen": agentInfo.LastSeen,
		},
	})
}

func parseHostID(s string) uint {
	id, _ := strconv.ParseUint(s, 10, 64)
	return uint(id)
}
