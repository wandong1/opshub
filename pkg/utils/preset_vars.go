package utils

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// GeneratePresetVariables 生成系统预置变量
// hostIP: 可选，传入则生成 exec_node_ip 变量（仅巡检模块使用）
func GeneratePresetVariables(hostIP string) map[string]string {
	now := time.Now()

	vars := map[string]string{
		// 时间相关
		"timestamp":        fmt.Sprintf("%d", now.Unix()),
		"timestamp_ms":     fmt.Sprintf("%d", now.UnixMilli()),
		"current_time":     now.Format("150405"),
		"current_date":     now.Format("20060102"),
		"current_datetime": now.Format("20060102150405"),

		// 随机数相关
		"random_number": fmt.Sprintf("%010d", rand.Intn(10000000000)),
		"random_string": GenerateRandomString(10),
		"random_uuid":   uuid.New().String(),
	}

	// 巡检专属：执行主机IP
	if hostIP != "" {
		vars["exec_node_ip"] = hostIP
	}

	return vars
}

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
