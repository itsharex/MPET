package connectors

import (
	"MPET/backend/models"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"
)

// ConnectSMB 连接 SMB
func (s *ConnectorService) ConnectSMB(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "445" // SMB 默认端口
	}
	addr := net.JoinHostPort(conn.IP, port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	// 尝试连接
	connTCP, err := s.DialWithProxy("tcp", addr)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("✗ TCP 连接失败: %v", err))
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		return
	}
	defer connTCP.Close()

	var session *smb2.Session
	var connected bool
	var loginType string

	// 如果用户提供了用户名和密码，直接使用
	if conn.User != "" && conn.Pass != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		d := &smb2.Dialer{
			Initiator: &smb2.NTLMInitiator{
				User:     conn.User,
				Password: conn.Pass,
			},
		}
		session, err = d.Dial(connTCP)
		if err == nil {
			s.AddLog(conn, "✓ 密码认证成功")
			connected = true
			loginType = "使用用户名密码"
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 密码认证失败: %v", err))
		}
		if !connected {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("连接失败: %v", err)
			s.AddLog(conn, "密码认证失败")
			return
		}
	} else {
		// 尝试匿名访问（空用户名和密码）
		s.AddLog(conn, "尝试匿名访问（空用户名和密码）")
		d := &smb2.Dialer{
			Initiator: &smb2.NTLMInitiator{
				User:     "",
				Password: "",
			},
		}
		session, err = d.Dial(connTCP)
		if err == nil {
			s.AddLog(conn, "✓ 匿名访问成功")
			connected = true
			loginType = "匿名访问"
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 匿名访问失败: %v", err))

			// 尝试使用提供的用户名（无密码）
			if conn.User != "" {
				s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码连接", conn.User))
				d := &smb2.Dialer{
					Initiator: &smb2.NTLMInitiator{
						User:     conn.User,
						Password: "",
					},
				}
				session, err = d.Dial(connTCP)
				if err == nil {
					s.AddLog(conn, "✓ 无密码连接成功")
					connected = true
					loginType = "无密码"
				} else {
					s.AddLog(conn, fmt.Sprintf("✗ 无密码连接失败: %v", err))
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
	defer session.Logoff()

	// 连接成功，获取共享列表
	s.AddLog(conn, "获取共享列表")
	result := s.GetSMBSharesJSON(session)
	conn.Status = "success"
	conn.Message = fmt.Sprintf("连接成功（%s）", loginType)
	s.SetConnectionResult(conn, result)
	conn.ConnectedAt = time.Now()
}

// GetSMBFiles 获取 SMB 共享中的文件列表
func (s *ConnectorService) GetSMBFiles(session *smb2.Session) string {
	var results []string
	results = append(results, "文件列表:")
	results = append(results, strings.Repeat("-", 80))

	// 尝试常见的共享名称
	shares := []string{"C$", "IPC$", "ADMIN$", "Share", "Public", "共享"}
	if session != nil {
		// 首先尝试列出所有共享
		sharesList, err := session.ListSharenames()
		if err == nil && len(sharesList) > 0 {
			s.AddLogForSMB(&results, fmt.Sprintf("发现 %d 个共享: %v", len(sharesList), sharesList))
			shares = sharesList
		}
	}

	// 尝试访问每个共享
	for _, shareName := range shares {
		fs, err := session.Mount(shareName)
		if err != nil {
			continue
		}

		results = append(results, "")
		results = append(results, fmt.Sprintf("共享: %s", shareName))
		results = append(results, strings.Repeat("-", 80))

		// 获取根目录的文件列表
		files, err := fs.ReadDir(".")
		if err != nil {
			results = append(results, fmt.Sprintf("读取目录失败: %v", err))
			fs.Umount()
			continue
		}

		if len(files) == 0 {
			results = append(results, "当前目录为空")
		} else {
			results = append(results, fmt.Sprintf("共找到 %d 个项目:", len(files)))
			results = append(results, "")
			results = append(results, fmt.Sprintf("%-10s %-15s %-20s %-30s", "类型", "大小", "修改时间", "名称"))
			results = append(results, strings.Repeat("-", 80))

			for _, file := range files {
				fileType := "文件"
				if file.IsDir() {
					fileType = "目录"
				}

				size := fmt.Sprintf("%d", file.Size())
				if file.IsDir() {
					size = "-"
				}

				timeStr := file.ModTime().Format("2006-01-02 15:04:05")
				if file.ModTime().IsZero() {
					timeStr = "-"
				}

				results = append(results, fmt.Sprintf("%-10s %-15s %-20s %-30s", fileType, size, timeStr, file.Name()))
			}
		}

		// 成功获取一个共享后卸载并返回
		fs.Umount()
		return strings.Join(results, "\n")
	}

	results = append(results, "无法访问任何共享")
	return strings.Join(results, "\n")
}

// AddLogForSMB 为 SMB 结果添加日志（辅助函数）
func (s *ConnectorService) AddLogForSMB(results *[]string, message string) {
	*results = append(*results, message)
}

// SMBFileInfo SMB 文件信息
type SMBFileInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "file", "folder", "share"
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}

// SMBDirectoryResponse SMB 目录响应
type SMBDirectoryResponse struct {
	CurrentPath string        `json:"currentPath"`
	CurrentShare string       `json:"currentShare"`
	Files       []SMBFileInfo `json:"files"`
}

// GetSMBSharesJSON 获取 SMB 共享列表（JSON 格式）
func (s *ConnectorService) GetSMBSharesJSON(session *smb2.Session) string {
	response := SMBDirectoryResponse{
		CurrentPath:  "/",
		CurrentShare: "",
		Files:        []SMBFileInfo{},
	}

	// 列出所有共享
	shares, err := session.ListSharenames()
	if err != nil {
		response.Files = append(response.Files, SMBFileInfo{
			Name: fmt.Sprintf("错误: %v", err),
			Type: "error",
		})
	} else {
		for _, shareName := range shares {
			response.Files = append(response.Files, SMBFileInfo{
				Name: shareName,
				Type: "share",
				Path: "/" + shareName,
			})
		}
	}

	jsonData, _ := json.Marshal(response)
	return string(jsonData)
}

// GetSMBDirectoryListJSON 获取 SMB 目录列表（JSON 格式）
func (s *ConnectorService) GetSMBDirectoryListJSON(session *smb2.Session, shareName string, dirPath string) string {
	response := SMBDirectoryResponse{
		CurrentPath:  dirPath,
		CurrentShare: shareName,
		Files:        []SMBFileInfo{},
	}

	// 挂载共享
	fs, err := session.Mount(shareName)
	if err != nil {
		response.Files = append(response.Files, SMBFileInfo{
			Name: fmt.Sprintf("挂载共享失败: %v", err),
			Type: "error",
		})
		jsonData, _ := json.Marshal(response)
		return string(jsonData)
	}
	defer fs.Umount()

	// 读取目录
	files, err := fs.ReadDir(dirPath)
	if err != nil {
		response.Files = append(response.Files, SMBFileInfo{
			Name: fmt.Sprintf("读取目录失败: %v", err),
			Type: "error",
		})
	} else {
		for _, file := range files {
			fileType := "file"
			if file.IsDir() {
				fileType = "folder"
			}

			timeStr := file.ModTime().Format("2006-01-02 15:04:05")
			if file.ModTime().IsZero() {
				timeStr = "-"
			}

			filePath := path.Join(dirPath, file.Name())
			response.Files = append(response.Files, SMBFileInfo{
				Name:    file.Name(),
				Type:    fileType,
				Size:    file.Size(),
				ModTime: timeStr,
				Path:    filePath,
			})
		}
	}

	jsonData, _ := json.Marshal(response)
	return string(jsonData)
}

// ConnectToSMB 建立 SMB 连接（辅助函数）
func (s *ConnectorService) ConnectToSMB(conn *models.Connection) (*smb2.Session, error) {
	port := conn.Port
	if port == "" {
		port = "445"
	}
	addr := net.JoinHostPort(conn.IP, port)

	connTCP, err := s.DialWithProxy("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("TCP 连接失败: %v", err)
	}

	user := conn.User
	pass := conn.Pass
	if user == "" {
		user = ""
		pass = ""
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     user,
			Password: pass,
		},
	}

	session, err := d.Dial(connTCP)
	if err != nil {
		connTCP.Close()
		return nil, fmt.Errorf("SMB 认证失败: %v", err)
	}

	return session, nil
}
