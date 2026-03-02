package filemanager

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// FileManager 本地文件操作
type FileManager struct{}

// NewFileManager 创建文件管理器
func NewFileManager() *FileManager {
	return &FileManager{}
}

// expandPath 展开路径中的 ~ 为用户home目录
func expandPath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if u, err := user.Current(); err == nil {
			if path == "~" {
				return u.HomeDir
			}
			return filepath.Join(u.HomeDir, path[2:])
		}
	}
	if path == "" {
		return "/"
	}
	return path
}

// HandleRequest 处理文件操作请求
func (f *FileManager) HandleRequest(requestID, action, path, filename string, data []byte) (*pb.AgentMessage, error) {
	path = expandPath(path)
	switch action {
	case "list":
		return f.listFiles(requestID, path)
	case "upload":
		return f.uploadFile(requestID, path, filename, data)
	case "download":
		return f.downloadFile(requestID, path)
	case "delete":
		return f.deleteFile(requestID, path)
	default:
		return nil, fmt.Errorf("未知操作: %s", action)
	}
}

func (f *FileManager) listFiles(requestID, path string) (*pb.AgentMessage, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return &pb.AgentMessage{
			Payload: &pb.AgentMessage_FileList{
				FileList: &pb.FileListResult{RequestId: requestID, Error: err.Error()},
			},
		}, nil
	}

	files := make([]*pb.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, &pb.FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Unix(),
			IsDir:   entry.IsDir(),
		})
	}

	return &pb.AgentMessage{
		Payload: &pb.AgentMessage_FileList{
			FileList: &pb.FileListResult{RequestId: requestID, Files: files},
		},
	}, nil
}

func (f *FileManager) uploadFile(requestID, path, filename string, data []byte) (*pb.AgentMessage, error) {
	// 确保目标目录存在
	if err := os.MkdirAll(path, 0755); err != nil {
		return &pb.AgentMessage{
			Payload: &pb.AgentMessage_FileChunk{
				FileChunk: &pb.FileChunk{RequestId: requestID, Error: fmt.Sprintf("创建目录失败: %v", err)},
			},
		}, nil
	}

	fullPath := filepath.Join(path, filename)

	// 如果文件已存在，先尝试删除（解决不同用户创建的文件权限问题）
	if _, err := os.Stat(fullPath); err == nil {
		os.Remove(fullPath)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return &pb.AgentMessage{
			Payload: &pb.AgentMessage_FileChunk{
				FileChunk: &pb.FileChunk{RequestId: requestID, Error: err.Error()},
			},
		}, nil
	}
	return &pb.AgentMessage{
		Payload: &pb.AgentMessage_FileChunk{
			FileChunk: &pb.FileChunk{RequestId: requestID, Eof: true},
		},
	}, nil
}

func (f *FileManager) downloadFile(requestID, path string) (*pb.AgentMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &pb.AgentMessage{
			Payload: &pb.AgentMessage_FileChunk{
				FileChunk: &pb.FileChunk{RequestId: requestID, Error: err.Error()},
			},
		}, nil
	}
	return &pb.AgentMessage{
		Payload: &pb.AgentMessage_FileChunk{
			FileChunk: &pb.FileChunk{RequestId: requestID, Data: data, Eof: true},
		},
	}, nil
}

func (f *FileManager) deleteFile(requestID, path string) (*pb.AgentMessage, error) {
	if err := os.RemoveAll(path); err != nil {
		return &pb.AgentMessage{
			Payload: &pb.AgentMessage_FileChunk{
				FileChunk: &pb.FileChunk{RequestId: requestID, Error: err.Error()},
			},
		}, nil
	}
	return &pb.AgentMessage{
		Payload: &pb.AgentMessage_FileChunk{
			FileChunk: &pb.FileChunk{RequestId: requestID, Eof: true},
		},
	}, nil
}
