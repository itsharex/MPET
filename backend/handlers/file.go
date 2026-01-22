package handlers

import (
	"MPET/backend/services"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type FileHandler struct {
	ctx     context.Context
	service *services.ConnectorService
	logger  *services.Logger
}

func NewFileHandler(ctx context.Context, service *services.ConnectorService) *FileHandler {
	return &FileHandler{
		ctx:     ctx,
		service: service,
		logger:  services.GetLogger(),
	}
}

// BrowseFTPDirectory 浏览 FTP 目录
func (h *FileHandler) BrowseFTPDirectory(id string, dirPath string) (string, error) {
	conn, exists := h.service.GetConnection(id)
	if !exists {
		return "", fmt.Errorf("连接不存在")
	}
	if conn.Status != "success" || conn.Type != "FTP" {
		return "", fmt.Errorf("连接未成功或类型不匹配")
	}

	h.logger.Info(fmt.Sprintf("浏览 FTP 目录: %s:%s - %s", conn.IP, conn.Port, dirPath))

	ftpConn, err := h.service.ConnectToFTP(conn)
	if err != nil {
		return "", err
	}
	defer ftpConn.Quit()

	result := h.service.GetFTPDirectoryListJSON(ftpConn, dirPath)
	return result, nil
}

// DownloadFTPFile 下载 FTP 文件
func (h *FileHandler) DownloadFTPFile(id string, filePath string) error {
	conn, exists := h.service.GetConnection(id)
	if !exists || conn.Status != "success" || conn.Type != "FTP" {
		return fmt.Errorf("连接不存在或状态异常")
	}

	ftpConn, err := h.service.ConnectToFTP(conn)
	if err != nil {
		return err
	}
	defer ftpConn.Quit()

	savePath, err := runtime.SaveFileDialog(h.ctx, runtime.SaveDialogOptions{
		DefaultFilename: filePath[strings.LastIndex(filePath, "/")+1:],
		Title:           "保存文件",
	})
	if err != nil || savePath == "" {
		return fmt.Errorf("用户取消了文件保存")
	}

	resp, err := ftpConn.Retr(filePath)
	if err != nil {
		return fmt.Errorf("下载文件失败: %v", err)
	}
	defer resp.Close()

	outFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer outFile.Close()

	_, err = outFile.ReadFrom(resp)
	if err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	h.logger.Info(fmt.Sprintf("FTP 文件下载成功: %s -> %s", filePath, savePath))
	return nil
}

// BrowseSMBDirectory 浏览 SMB 目录
func (h *FileHandler) BrowseSMBDirectory(id string, shareName string, dirPath string) (string, error) {
	conn, exists := h.service.GetConnection(id)
	if !exists || conn.Status != "success" || conn.Type != "SMB" {
		return "", fmt.Errorf("连接不存在或状态异常")
	}

	session, err := h.service.ConnectToSMB(conn)
	if err != nil {
		return "", err
	}
	defer session.Logoff()

	if shareName == "" {
		return h.service.GetSMBSharesJSON(session), nil
	}

	return h.service.GetSMBDirectoryListJSON(session, shareName, dirPath), nil
}

// DownloadSMBFile 下载 SMB 文件
func (h *FileHandler) DownloadSMBFile(id string, shareName string, filePath string) error {
	conn, exists := h.service.GetConnection(id)
	if !exists || conn.Status != "success" || conn.Type != "SMB" {
		return fmt.Errorf("连接不存在或状态异常")
	}

	session, err := h.service.ConnectToSMB(conn)
	if err != nil {
		return err
	}
	defer session.Logoff()

	fs, err := session.Mount(shareName)
	if err != nil {
		return fmt.Errorf("挂载共享失败: %v", err)
	}
	defer fs.Umount()

	savePath, err := runtime.SaveFileDialog(h.ctx, runtime.SaveDialogOptions{
		DefaultFilename: filePath[strings.LastIndex(filePath, "/")+1:],
		Title:           "保存文件",
	})
	if err != nil || savePath == "" {
		return fmt.Errorf("用户取消了文件保存")
	}

	remoteFile, err := fs.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	outFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer outFile.Close()

	_, err = outFile.ReadFrom(remoteFile)
	return err
}

// BrowseSFTPDirectory 浏览 SFTP 目录
func (h *FileHandler) BrowseSFTPDirectory(id string, dirPath string) (string, error) {
	conn, exists := h.service.GetConnection(id)
	if !exists || conn.Status != "success" || conn.Type != "SFTP" {
		return "", fmt.Errorf("连接不存在或状态异常")
	}

	sftpClient, sshClient, err := h.service.ConnectToSFTP(conn)
	if err != nil {
		return "", err
	}
	defer sftpClient.Close()
	defer sshClient.Close()

	return h.service.GetSFTPDirectoryListJSON(sftpClient, dirPath), nil
}

// DownloadSFTPFile 下载 SFTP 文件
func (h *FileHandler) DownloadSFTPFile(id string, filePath string) error {
	conn, exists := h.service.GetConnection(id)
	if !exists || conn.Status != "success" || conn.Type != "SFTP" {
		return fmt.Errorf("连接不存在或状态异常")
	}

	sftpClient, sshClient, err := h.service.ConnectToSFTP(conn)
	if err != nil {
		return err
	}
	defer sftpClient.Close()
	defer sshClient.Close()

	savePath, err := runtime.SaveFileDialog(h.ctx, runtime.SaveDialogOptions{
		DefaultFilename: filePath[strings.LastIndex(filePath, "/")+1:],
		Title:           "保存文件",
	})
	if err != nil || savePath == "" {
		return fmt.Errorf("用户取消了文件保存")
	}

	remoteFile, err := sftpClient.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	outFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, remoteFile)
	return err
}

// SelectDirectory 选择目录
func (h *FileHandler) SelectDirectory() (string, error) {
	dirPath, err := runtime.OpenDirectoryDialog(h.ctx, runtime.OpenDialogOptions{
		Title: "选择报告导出目录",
	})
	if err != nil {
		return "", fmt.Errorf("选择目录失败: %v", err)
	}
	if dirPath == "" {
		return "", fmt.Errorf("用户取消了目录选择")
	}
	return dirPath, nil
}
