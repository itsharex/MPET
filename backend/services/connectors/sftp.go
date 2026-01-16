package connectors

import (
	"MPET/backend/models"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ConnectSFTP 连接 SFTP
func (s *ConnectorService) ConnectSFTP(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "22"
	}

	address := net.JoinHostPort(conn.IP, port)
	s.AddLog(conn, fmt.Sprintf("目标地址: %s", address))
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	// 配置 SSH 客户端
	config := &ssh.ClientConfig{
		User: conn.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(conn.Pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if conn.User == "" {
		config.User = "root"
	}

	// 建立 SSH 连接
	var sshClient *ssh.Client
	var err error

	if s.Config.Proxy.Enabled {
		// 通过代理连接
		proxyConn, err := s.DialWithProxy("tcp", address)
		if err != nil {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("代理连接失败: %v", err)
			s.AddLog(conn, conn.Message)
			return
		}
		defer proxyConn.Close()

		sshConn, chans, reqs, err := ssh.NewClientConn(proxyConn, address, config)
		if err != nil {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("SSH 连接失败: %v", err)
			s.AddLog(conn, conn.Message)
			return
		}
		sshClient = ssh.NewClient(sshConn, chans, reqs)
	} else {
		// 直接连接
		sshClient, err = ssh.Dial("tcp", address, config)
		if err != nil {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("SSH 连接失败: %v", err)
			s.AddLog(conn, conn.Message)
			return
		}
	}
	defer sshClient.Close()

	s.AddLog(conn, "✓ SSH 连接成功")

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("SFTP 客户端创建失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}
	defer sftpClient.Close()

	s.AddLog(conn, "✓ SFTP 客户端创建成功")

	// 获取根目录列表
	files, err := sftpClient.ReadDir("/")
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("读取目录失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	s.AddLog(conn, fmt.Sprintf("✓ 成功读取根目录，共 %d 个文件/目录", len(files)))

	// 构建文件列表 JSON
	fileList := s.BuildSFTPFileListJSON(files, "/")

	conn.Status = "success"
	conn.Message = "连接成功"
	s.SetConnectionResult(conn, fileList)
	conn.ConnectedAt = time.Now()
	s.AddLog(conn, "✓ SFTP 连接测试完成")
}

// SFTPFileInfo SFTP 文件信息
type SFTPFileInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Path    string `json:"path"`
}

// SFTPDirectoryResponse SFTP 目录响应
type SFTPDirectoryResponse struct {
	CurrentPath string         `json:"currentPath"`
	Files       []SFTPFileInfo `json:"files"`
}

// BuildSFTPFileListJSON 构建 SFTP 文件列表 JSON
func (s *ConnectorService) BuildSFTPFileListJSON(files []os.FileInfo, dirPath string) string {
	response := SFTPDirectoryResponse{
		CurrentPath: dirPath,
		Files:       []SFTPFileInfo{},
	}

	for _, file := range files {
		fileType := "file"
		if file.IsDir() {
			fileType = "folder"
		}

		filePath := path.Join(dirPath, file.Name())
		response.Files = append(response.Files, SFTPFileInfo{
			Name:    file.Name(),
			Type:    fileType,
			Size:    file.Size(),
			ModTime: file.ModTime().Format("2006-01-02 15:04:05"),
			Path:    filePath,
		})
	}

	jsonData, _ := json.Marshal(response)
	return string(jsonData)
}

// ConnectToSFTP 连接到 SFTP 服务器
func (s *ConnectorService) ConnectToSFTP(conn *models.Connection) (*sftp.Client, *ssh.Client, error) {
	port := conn.Port
	if port == "" {
		port = "22"
	}

	address := net.JoinHostPort(conn.IP, port)

	// 配置 SSH 客户端
	config := &ssh.ClientConfig{
		User: conn.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(conn.Pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if conn.User == "" {
		config.User = "root"
	}

	// 建立 SSH 连接
	var sshClient *ssh.Client
	var err error

	if s.Config.Proxy.Enabled {
		// 通过代理连接
		proxyConn, err := s.DialWithProxy("tcp", address)
		if err != nil {
			return nil, nil, fmt.Errorf("代理连接失败: %v", err)
		}

		sshConn, chans, reqs, err := ssh.NewClientConn(proxyConn, address, config)
		if err != nil {
			proxyConn.Close()
			return nil, nil, fmt.Errorf("SSH 连接失败: %v", err)
		}
		sshClient = ssh.NewClient(sshConn, chans, reqs)
	} else {
		// 直接连接
		sshClient, err = ssh.Dial("tcp", address, config)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH 连接失败: %v", err)
		}
	}

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, nil, fmt.Errorf("SFTP 客户端创建失败: %v", err)
	}

	return sftpClient, sshClient, nil
}

// GetSFTPDirectoryListJSON 获取 SFTP 目录列表 JSON
func (s *ConnectorService) GetSFTPDirectoryListJSON(sftpClient *sftp.Client, dirPath string) string {
	response := SFTPDirectoryResponse{
		CurrentPath: dirPath,
		Files:       []SFTPFileInfo{},
	}

	// 读取目录
	files, err := sftpClient.ReadDir(dirPath)
	if err != nil {
		response.Files = append(response.Files, SFTPFileInfo{
			Name: fmt.Sprintf("错误: %v", err),
			Type: "error",
		})
		jsonData, _ := json.Marshal(response)
		return string(jsonData)
	}

	for _, file := range files {
		fileType := "file"
		if file.IsDir() {
			fileType = "folder"
		}

		filePath := path.Join(dirPath, file.Name())
		response.Files = append(response.Files, SFTPFileInfo{
			Name:    file.Name(),
			Type:    fileType,
			Size:    file.Size(),
			ModTime: file.ModTime().Format("2006-01-02 15:04:05"),
			Path:    filePath,
		})
	}

	jsonData, _ := json.Marshal(response)
	return string(jsonData)
}

// DownloadSFTPFile 下载 SFTP 文件
func (s *ConnectorService) DownloadSFTPFile(sftpClient *sftp.Client, remotePath string) (string, error) {
	// 打开远程文件
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return "", fmt.Errorf("打开远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	// 获取文件信息
	stat, err := remoteFile.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %v", err)
	}

	// 创建本地文件
	localPath := path.Join(os.TempDir(), stat.Name())
	localFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer localFile.Close()

	// 复制文件内容
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %v", err)
	}

	return localPath, nil
}
