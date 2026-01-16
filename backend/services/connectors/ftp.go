package connectors

import (
	"MPET/backend/models"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

// ConnectFTP 连接 FTP
func (s *ConnectorService) ConnectFTP(conn *models.Connection) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: FTP 连接暂不支持代理，将尝试直接连接")
	}

	var ftpConn *ftp.ServerConn
	var err error
	var connected bool
	var loginType string

	// 如果用户提供了用户名和密码，直接使用，跳过匿名登录
	if conn.User != "" && conn.Pass != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		ftpConn, err = ftp.Dial(addr, ftp.DialWithTimeout(5*time.Second))
		if err == nil {
			err = ftpConn.Login(conn.User, conn.Pass)
			if err == nil {
				s.AddLog(conn, "✓ 密码认证成功")
				connected = true
				loginType = "使用用户名密码"
			} else {
				s.AddLog(conn, fmt.Sprintf("✗ 密码认证失败: %v", err))
				ftpConn.Quit()
			}
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ FTP 连接失败: %v", err))
		}
		if !connected {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("连接失败: %v", err)
			s.AddLog(conn, "密码认证失败")
			return
		}
	} else {
		// 尝试匿名登录
		s.AddLog(conn, "尝试匿名登录（anonymous/anonymous）")
		ftpConn, err = ftp.Dial(addr, ftp.DialWithTimeout(5*time.Second))
		if err == nil {
			err = ftpConn.Login("anonymous", "anonymous")
			if err == nil {
				s.AddLog(conn, "✓ 匿名登录成功")
				connected = true
				loginType = "匿名登录"
			} else {
				s.AddLog(conn, fmt.Sprintf("✗ 匿名登录失败: %v", err))
				ftpConn.Quit()
			}
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ FTP 连接失败: %v", err))
		}

		// 尝试未授权访问（无密码）
		if !connected && conn.User != "" && conn.Pass == "" {
			s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码登录", conn.User))
			ftpConn, err = ftp.Dial(addr, ftp.DialWithTimeout(5*time.Second))
			if err == nil {
				err = ftpConn.Login(conn.User, "")
				if err == nil {
					s.AddLog(conn, "✓ 无密码登录成功")
					connected = true
					loginType = "无密码"
				} else {
					s.AddLog(conn, fmt.Sprintf("✗ 无密码登录失败: %v", err))
					ftpConn.Quit()
				}
			}
		}
	}

	if !connected {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, "所有连接尝试均失败")
		return
	}

	// 连接成功，获取根目录文件列表
	s.AddLog(conn, "获取根目录文件列表")
	result := s.GetFTPDirectoryListJSON(ftpConn, "/")
	conn.Status = "success"
	conn.Message = fmt.Sprintf("连接成功（%s）", loginType)
	s.SetConnectionResult(conn, result)
	conn.ConnectedAt = time.Now()
	ftpConn.Quit()
}

// GetFTPDirectoryList 获取 FTP 目录列表（相当于 dir 命令）
func (s *ConnectorService) GetFTPDirectoryList(ftpConn *ftp.ServerConn) string {
	var results []string
	results = append(results, "目录列表:")
	results = append(results, strings.Repeat("-", 80))

	// 获取当前工作目录
	pwd, err := ftpConn.CurrentDir()
	if err == nil {
		results = append(results, fmt.Sprintf("当前目录: %s", pwd))
		results = append(results, "")
	}

	// 执行 LIST 命令获取目录列表
	entries, err := ftpConn.List(".")
	if err != nil {
		results = append(results, fmt.Sprintf("获取目录列表失败: %v", err))
		return strings.Join(results, "\n")
	}

	if len(entries) == 0 {
		results = append(results, "当前目录为空")
	} else {
		results = append(results, fmt.Sprintf("共找到 %d 个项目:", len(entries)))
		results = append(results, "")
		results = append(results, fmt.Sprintf("%-10s %-15s %-20s %-30s", "类型", "大小", "修改时间", "名称"))
		results = append(results, strings.Repeat("-", 80))

		for _, entry := range entries {
			fileType := "文件"
			if entry.Type == ftp.EntryTypeFolder {
				fileType = "目录"
			}

			size := fmt.Sprintf("%d", entry.Size)
			if entry.Size == 0 && entry.Type == ftp.EntryTypeFolder {
				size = "-"
			}

			timeStr := entry.Time.Format("2006-01-02 15:04:05")
			if entry.Time.IsZero() {
				timeStr = "-"
			}

			results = append(results, fmt.Sprintf("%-10s %-15s %-20s %-30s", fileType, size, timeStr, entry.Name))
		}
	}

	return strings.Join(results, "\n")
}

// FTPFileInfo FTP 文件信息
type FTPFileInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "file" 或 "folder"
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}

// FTPDirectoryResponse FTP 目录响应
type FTPDirectoryResponse struct {
	CurrentPath string        `json:"currentPath"`
	Files       []FTPFileInfo `json:"files"`
}

// GetFTPDirectoryListJSON 获取 FTP 目录列表（JSON 格式）
func (s *ConnectorService) GetFTPDirectoryListJSON(ftpConn *ftp.ServerConn, dirPath string) string {
	response := FTPDirectoryResponse{
		CurrentPath: dirPath,
		Files:       []FTPFileInfo{},
	}

	// 执行 LIST 命令获取目录列表
	entries, err := ftpConn.List(dirPath)
	if err != nil {
		response.Files = append(response.Files, FTPFileInfo{
			Name: fmt.Sprintf("错误: %v", err),
			Type: "error",
		})
	} else {
		for _, entry := range entries {
			fileType := "file"
			if entry.Type == ftp.EntryTypeFolder {
				fileType = "folder"
			}

			timeStr := entry.Time.Format("2006-01-02 15:04:05")
			if entry.Time.IsZero() {
				timeStr = "-"
			}

			filePath := path.Join(dirPath, entry.Name)
			response.Files = append(response.Files, FTPFileInfo{
				Name:    entry.Name,
				Type:    fileType,
				Size:    int64(entry.Size),
				ModTime: timeStr,
				Path:    filePath,
			})
		}
	}

	jsonData, _ := json.Marshal(response)
	return string(jsonData)
}

// ConnectToFTP 建立 FTP 连接（辅助函数）
func (s *ConnectorService) ConnectToFTP(conn *models.Connection) (*ftp.ServerConn, error) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	
	ftpConn, err := ftp.Dial(addr, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}

	// 登录
	user := conn.User
	pass := conn.Pass
	if user == "" {
		user = "anonymous"
		pass = "anonymous"
	}

	err = ftpConn.Login(user, pass)
	if err != nil {
		ftpConn.Quit()
		return nil, fmt.Errorf("登录失败: %v", err)
	}

	return ftpConn, nil
}
