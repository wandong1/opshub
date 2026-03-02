package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	assetbiz "github.com/ydcloud-dy/opshub/internal/biz/asset"
	assetserver "github.com/ydcloud-dy/opshub/internal/server/asset"
	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// HandleTerminal Agent WebSocket终端
func (s *HTTPServer) HandleTerminal(c *gin.Context) {
	hostIDStr := c.Param("hostId")
	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的主机ID"})
		return
	}

	as, ok := s.hub.GetByHostID(uint(hostID))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Agent不在线"})
		return
	}

	// 获取用户信息
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	var uid uint
	var uname string = "unknown"
	if id, ok := userID.(uint); ok {
		uid = id
	} else if id, ok := userID.(float64); ok {
		uid = uint(id)
	}
	if name, ok := username.(string); ok {
		uname = name
	}

	colsStr := c.DefaultQuery("cols", "80")
	rowsStr := c.DefaultQuery("rows", "24")
	cols, _ := strconv.ParseUint(colsStr, 10, 32)
	rows, _ := strconv.ParseUint(rowsStr, 10, 32)
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	// 升级WebSocket
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		appLogger.Error("WebSocket升级失败", zap.Error(err))
		return
	}
	defer conn.Close()

	sessionID := fmt.Sprintf("agent-%d-%d", hostID, time.Now().UnixNano())
	hostVO, _ := s.hostUseCase.GetByID(c.Request.Context(), uint(hostID))

	// 创建录制器
	recorder, _ := assetserver.NewAsciinemaRecorder("./data/terminal-recordings", int(cols), int(rows))

	// 注册终端输出回调 -> 转发到WebSocket
	var connMu sync.Mutex
	s.hub.RegisterTerminalCallback(sessionID, func(data []byte) {
		if recorder != nil {
			recorder.RecordOutput(data)
		}
		connMu.Lock()
		conn.WriteMessage(websocket.BinaryMessage, data)
		connMu.Unlock()
	})
	defer s.hub.UnregisterTerminalCallback(sessionID)

	// 发送打开终端请求到Agent
	as.Send(&pb.ServerMessage{
		Payload: &pb.ServerMessage_TermOpen{
			TermOpen: &pb.TerminalOpen{
				SessionId: sessionID,
				Cols:      uint32(cols),
				Rows:      uint32(rows),
			},
		},
	})
	defer func() {
		as.Send(&pb.ServerMessage{
			Payload: &pb.ServerMessage_TermClose{
				TermClose: &pb.TerminalClose{SessionId: sessionID},
			},
		})
		if recorder != nil {
			recorder.Close()
			s.saveTerminalSession(recorder, uint(hostID), hostVO, uid, uname)
		}
	}()

	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		return nil
	})

	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			appLogger.Info("Agent终端WebSocket关闭", zap.String("sessionID", sessionID), zap.Error(err))
			break
		}

		if messageType == websocket.TextMessage {
			var msg map[string]any
			if json.Unmarshal(data, &msg) == nil {
				if msgType, _ := msg["type"].(string); msgType == "resize" {
					c, _ := msg["cols"].(float64)
					r, _ := msg["rows"].(float64)
					as.Send(&pb.ServerMessage{
						Payload: &pb.ServerMessage_TermResize{
							TermResize: &pb.TerminalResize{
								SessionId: sessionID,
								Cols:      uint32(c),
								Rows:      uint32(r),
							},
						},
					})
					continue
				}
			}
		}

		if recorder != nil {
			recorder.RecordInput(data)
		}
		as.Send(&pb.ServerMessage{
			Payload: &pb.ServerMessage_TermInput{
				TermInput: &pb.TerminalInput{
					SessionId: sessionID,
					Data:      data,
				},
			},
		})
	}
}

// saveTerminalSession 保存终端会话记录
func (s *HTTPServer) saveTerminalSession(recorder *assetserver.AsciinemaRecorder, hostID uint, hostVO *assetbiz.HostInfoVO, userID uint, username string) {
	var hostName, hostIP string
	if hostVO != nil {
		hostName = hostVO.Name
		hostIP = hostVO.IP
	}
	session := &assetbiz.TerminalSession{
		HostID:         hostID,
		HostName:       hostName,
		HostIP:         hostIP,
		UserID:         userID,
		Username:       username,
		RecordingPath:  recorder.GetRecordingPath(),
		Duration:       recorder.GetDuration(),
		FileSize:       recorder.GetFileSize(),
		Status:         "completed",
		ConnectionType: "agent",
	}
	if err := s.db.Create(session).Error; err != nil {
		appLogger.Error("保存Agent终端会话记录失败", zap.Error(err))
	}
}
