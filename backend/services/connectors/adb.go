package connectors

import (
	"MPET/backend/models"
	"fmt"
	"strings"
	"time"

	"github.com/electricbubble/gadb"
)

// ConnectADB 连接 ADB (Android Debug Bridge)
func (s *ConnectorService) ConnectADB(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "5555"
	}

	s.AddLog(conn, fmt.Sprintf("目标地址: %s:%s", conn.IP, port))
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("⚠ 注意: gadb 库暂不支持代理，将直接连接"))
	}

	// 创建 ADB 客户端（连接到本地 ADB 服务器）
	client, err := gadb.NewClient()
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("创建 ADB 客户端失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	s.AddLog(conn, "✓ ADB 客户端创建成功")

	// 获取 ADB 服务器版本
	version, err := client.ServerVersion()
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("获取 ADB 服务器版本失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	s.AddLog(conn, fmt.Sprintf("✓ ADB 服务器版本: %d", version))

	// 连接到远程设备
	portInt := 5555
	fmt.Sscanf(port, "%d", &portInt)
	
	err = client.Connect(conn.IP, portInt)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接到设备失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	s.AddLog(conn, "✓ 已连接到设备")

	// 获取设备列表
	devices, err := client.DeviceList()
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("获取设备列表失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	if len(devices) == 0 {
		conn.Status = "failed"
		conn.Message = "未找到连接的设备"
		s.AddLog(conn, conn.Message)
		return
	}

	// 使用第一个设备
	device := devices[0]
	s.AddLog(conn, fmt.Sprintf("✓ 设备序列号: %s", device.Serial()))
	
	// 获取设备状态
	state, err := device.State()
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("⚠ 获取设备状态失败: %v", err))
	} else {
		s.AddLog(conn, fmt.Sprintf("设备状态: %s", state))
	}

	// 构建结果信息
	var result strings.Builder
	result.WriteString("=== ADB 设备信息 ===\n")
	result.WriteString(fmt.Sprintf("ADB 服务器版本: %d\n", version))
	result.WriteString(fmt.Sprintf("设备序列号: %s\n", device.Serial()))
	if err == nil {
		result.WriteString(fmt.Sprintf("设备状态: %s\n", state))
	}

	// 获取设备型号
	if deviceModel, err := device.RunShellCommand("getprop ro.product.model"); err == nil {
		deviceModel = strings.TrimSpace(deviceModel)
		s.AddLog(conn, fmt.Sprintf("设备型号: %s", deviceModel))
		result.WriteString(fmt.Sprintf("设备型号: %s\n", deviceModel))
	}

	// 获取 Android 版本
	if androidVersion, err := device.RunShellCommand("getprop ro.build.version.release"); err == nil {
		androidVersion = strings.TrimSpace(androidVersion)
		s.AddLog(conn, fmt.Sprintf("Android 版本: %s", androidVersion))
		result.WriteString(fmt.Sprintf("Android 版本: %s\n", androidVersion))
	}

	// 获取 SDK 版本
	if sdkVersion, err := device.RunShellCommand("getprop ro.build.version.sdk"); err == nil {
		sdkVersion = strings.TrimSpace(sdkVersion)
		result.WriteString(fmt.Sprintf("SDK 版本: %s\n", sdkVersion))
	}

	// 获取设备品牌
	if brand, err := device.RunShellCommand("getprop ro.product.brand"); err == nil {
		brand = strings.TrimSpace(brand)
		result.WriteString(fmt.Sprintf("设备品牌: %s\n", brand))
	}

	conn.Status = "success"
	conn.Message = "连接成功"
	s.SetConnectionResult(conn, result.String())
	conn.ConnectedAt = time.Now()
	s.AddLog(conn, "✓ ADB 连接测试完成")
}

// ExecuteADBCommand 执行 ADB shell 命令
func (s *ConnectorService) ExecuteADBCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("ADB 连接未建立")
	}

	port := conn.Port
	if port == "" {
		port = "5555"
	}

	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	// 创建 ADB 客户端
	client, err := gadb.NewClient()
	if err != nil {
		return "", fmt.Errorf("创建 ADB 客户端失败: %v", err)
	}

	// 连接到远程设备
	portInt := 5555
	fmt.Sscanf(port, "%d", &portInt)
	
	err = client.Connect(conn.IP, portInt)
	if err != nil {
		return "", fmt.Errorf("连接到设备失败: %v", err)
	}

	// 获取设备列表
	devices, err := client.DeviceList()
	if err != nil {
		return "", fmt.Errorf("获取设备列表失败: %v", err)
	}

	if len(devices) == 0 {
		return "", fmt.Errorf("未找到连接的设备")
	}

	// 使用第一个设备执行命令
	device := devices[0]
	output, err := device.RunShellCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("命令执行失败: %v", err)
	}

	if output == "" {
		output = "(命令执行成功，无输出)"
	}

	return output, nil
}

