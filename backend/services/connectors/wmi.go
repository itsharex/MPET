package connectors

import (
	"MPET/backend/models"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ConnectWMI 通过 wmic 获取网卡信息
func (s *ConnectorService) ConnectWMI(conn *models.Connection) {
	if runtime.GOOS != "windows" {
		msg := "当前系统不支持 WMI（仅支持 Windows 环境执行 wmic）"
		conn.Status = "failed"
		conn.Message = msg
		s.AddLog(conn, msg)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{}
	if conn.IP != "" {
		args = append(args, "/node:"+conn.IP)
	} else {
		s.AddLog(conn, "未指定 IP，将默认本机")
	}

	if conn.User != "" {
		args = append(args, "/user:"+conn.User)
		if conn.Pass != "" {
			args = append(args, "/password:"+conn.Pass)
		} else {
			s.AddLog(conn, "未提供密码，WMI 可能无法完成认证")
		}
	} else {
		s.AddLog(conn, "未提供用户名，将使用当前系统上下文执行 wmic")
	}

	args = append(args, "nic", "get")
	s.AddLog(conn, fmt.Sprintf("执行命令: wmic %s", strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "wmic", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprintf("wmic 执行失败: %v", err)
		if stderr.Len() > 0 {
			errMsg = fmt.Sprintf("%s（%s）", errMsg, strings.TrimSpace(stderr.String()))
		}
		conn.Status = "failed"
		conn.Message = errMsg
		s.AddLog(conn, errMsg)
		return
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = "命令执行成功，但未返回任何内容"
	}

	conn.Status = "success"
	conn.Message = "WMI 命令执行成功"
	s.SetConnectionResult(conn, output)
	conn.ConnectedAt = time.Now()
	s.AddLog(conn, "✓ WMI 命令执行成功")
}
