package connectors

import (
	"MPET/backend/models"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectSSH 连接 SSH
func (s *ConnectorService) ConnectSSH(conn *models.Connection) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	// 如果用户名为空，尝试常见默认用户名
	users := []string{conn.User}
	if conn.User == "" {
		users = []string{"root", "admin", "ubuntu", "centos"}
		s.AddLog(conn, "用户名为空，将尝试常见默认用户名")
	}

	// 如果提供了密码，只尝试密码认证
	if conn.Pass != "" {
		s.AddLog(conn, fmt.Sprintf("使用提供的密码进行认证（密码长度: %d）", len(conn.Pass)))
		for _, user := range users {
			if user == "" {
				continue
			}
			s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", user))
			config := &ssh.ClientConfig{
				User:            user,
				Auth:            []ssh.AuthMethod{ssh.Password(conn.Pass)},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         5 * time.Second,
			}

			// 使用代理或直接连接
			var client *ssh.Client
			var err error
			if s.Config.Proxy.Enabled {
				proxyDialer, dialErr := s.GetProxyDialer()
				if dialErr != nil {
					s.AddLog(conn, fmt.Sprintf("✗ 创建代理 Dialer 失败: %v", dialErr))
					continue
				}
				connProxy, dialErr := proxyDialer.Dial("tcp", addr)
				if dialErr != nil {
					s.AddLog(conn, fmt.Sprintf("✗ 通过代理连接失败: %v", dialErr))
					continue
				}
				sshConn, chans, reqs, err := ssh.NewClientConn(connProxy, addr, config)
				if err == nil {
					client = ssh.NewClient(sshConn, chans, reqs)
				}
			} else {
				client, err = ssh.Dial("tcp", addr, config)
			}
			if err == nil {
				s.AddLog(conn, fmt.Sprintf("✓ 用户 %s 密码认证成功", user))
				s.AddLog(conn, "执行命令: whoami, ip addr")
				// 执行命令
				result := s.ExecuteSSHCommands(client)
				conn.Status = "success"
				conn.Message = fmt.Sprintf("连接成功（用户: %s）", user)
				s.SetConnectionResult(conn, result)
				conn.ConnectedAt = time.Now()
				client.Close()
				return
			}
			s.AddLog(conn, fmt.Sprintf("✗ 用户 %s 密码认证失败: %v", user, err))
		}
		// 如果提供了密码但所有尝试都失败，不再尝试密钥认证
		conn.Status = "failed"
		conn.Message = "连接失败: 密码认证失败"
		s.AddLog(conn, "密码认证失败，不再尝试密钥认证")
		return
	}

	// 如果没有提供密码，尝试密钥认证或无密码连接
	s.AddLog(conn, "未提供密码，尝试密钥认证或无密码连接")
	for _, user := range users {
		if user == "" {
			continue
		}
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密钥认证", user))
		config := &ssh.ClientConfig{
			User:            user,
			Auth:            []ssh.AuthMethod{},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		}

		var client *ssh.Client
		var err error
		if s.Config.Proxy.Enabled {
			proxyDialer, dialErr := s.GetProxyDialer()
			if dialErr != nil {
				s.AddLog(conn, fmt.Sprintf("✗ 创建代理 Dialer 失败: %v", dialErr))
				continue
			}
			connProxy, dialErr := proxyDialer.Dial("tcp", addr)
			if dialErr != nil {
				s.AddLog(conn, fmt.Sprintf("✗ 通过代理连接失败: %v", dialErr))
				continue
			}
			sshConn, chans, reqs, err := ssh.NewClientConn(connProxy, addr, config)
			if err == nil {
				client = ssh.NewClient(sshConn, chans, reqs)
			}
		} else {
			client, err = ssh.Dial("tcp", addr, config)
		}
		if err == nil {
			s.AddLog(conn, fmt.Sprintf("✓ 用户 %s 密钥认证成功", user))
			s.AddLog(conn, "执行命令: whoami, ip addr")
			result := s.ExecuteSSHCommands(client)
			conn.Status = "success"
			conn.Message = fmt.Sprintf("连接成功（密钥认证或无密码，用户: %s）", user)
			s.SetConnectionResult(conn, result)
			conn.ConnectedAt = time.Now()
			client.Close()
			return
		}
		s.AddLog(conn, fmt.Sprintf("✗ 用户 %s 密钥认证失败: %v", user, err))
	}

	conn.Status = "failed"
	conn.Message = "连接失败: 所有尝试均失败"
	s.AddLog(conn, "所有连接尝试均失败")
}

// ExecuteSSHCommands 执行 SSH 命令
func (s *ConnectorService) ExecuteSSHCommands(client *ssh.Client) string {
	var results []string

	commands := []string{"whoami", "ip addr"}
	for _, cmd := range commands {
		session, err := client.NewSession()
		if err != nil {
			results = append(results, fmt.Sprintf("命令 %s 执行失败: %v", cmd, err))
			continue
		}

		output, err := session.CombinedOutput(cmd)
		session.Close()

		if err != nil {
			results = append(results, fmt.Sprintf("命令 %s 执行失败: %v", cmd, err))
		} else {
			results = append(results, fmt.Sprintf("命令: %s\n%s", cmd, string(output)))
		}
	}

	return strings.Join(results, "")
}

// ExecuteSSHCommand 执行 SSH 命令
func (s *ConnectorService) ExecuteSSHCommand(conn *models.Connection, command string) (string, error) {
	config := &ssh.ClientConfig{
		User: conn.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(conn.Pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(conn.IP, conn.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("SSH连接失败: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("命令执行失败: %v", err)
	}

	return string(output), nil
}
