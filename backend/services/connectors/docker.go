package connectors

import (
	"MPET/backend/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DockerInfo Docker信息结构
type DockerInfo struct {
	ID                string `json:"ID"`
	Containers        int    `json:"Containers"`
	ContainersRunning int    `json:"ContainersRunning"`
	ContainersPaused  int    `json:"ContainersPaused"`
	ContainersStopped int    `json:"ContainersStopped"`
	Images            int    `json:"Images"`
	Driver            string `json:"Driver"`
	ServerVersion     string `json:"ServerVersion"`
	OperatingSystem   string `json:"OperatingSystem"`
	Architecture      string `json:"Architecture"`
	NCPU              int    `json:"NCPU"`
	MemTotal          int64  `json:"MemTotal"`
	Name              string `json:"Name"`
	KernelVersion     string `json:"KernelVersion"`
}

// DockerVersion Docker版本信息
type DockerVersion struct {
	Version       string `json:"Version"`
	ApiVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinAPIVersion"`
	GitCommit     string `json:"GitCommit"`
	GoVersion     string `json:"GoVersion"`
	Os            string `json:"Os"`
	Arch          string `json:"Arch"`
	KernelVersion string `json:"KernelVersion"`
	BuildTime     string `json:"BuildTime"`
}

// DockerContainer Docker容器信息
type DockerContainer struct {
	Id     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Image  string            `json:"Image"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Ports  []DockerPort      `json:"Ports"`
	Labels map[string]string `json:"Labels"`
}

// DockerPort Docker端口信息
type DockerPort struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

// DockerImage Docker镜像信息
type DockerImage struct {
	Id          string   `json:"Id"`
	RepoTags    []string `json:"RepoTags"`
	RepoDigests []string `json:"RepoDigests"`
	Created     int64    `json:"Created"`
	Size        int64    `json:"Size"`
	VirtualSize int64    `json:"VirtualSize"`
}

// ConnectDocker 连接 Docker API
func (s *ConnectorService) ConnectDocker(conn *models.Connection) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	// 尝试 HTTP 和 HTTPS
	schemes := []string{"http"}
	if conn.Port == "2376" {
		schemes = []string{"https", "http"}
	}

	var lastErr error
	for _, scheme := range schemes {
		baseURL := fmt.Sprintf("%s://%s", scheme, addr)
		s.AddLog(conn, fmt.Sprintf("尝试 %s 协议", strings.ToUpper(scheme)))

		// 创建 HTTP 客户端
		client := s.createDockerHTTPClient()

		// 尝试获取版本信息
		version, err := s.getDockerVersion(client, baseURL)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ 获取版本信息失败: %v", err))
			lastErr = err
			continue
		}

		s.AddLog(conn, fmt.Sprintf("✓ Docker 版本: %s", version.Version))

		// 获取系统信息
		info, err := s.getDockerInfo(client, baseURL)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ 获取系统信息失败: %v", err))
			lastErr = err
			continue
		}

		s.AddLog(conn, fmt.Sprintf("✓ 主机名: %s", info.Name))
		s.AddLog(conn, fmt.Sprintf("✓ 容器: %d (运行: %d)", info.Containers, info.ContainersRunning))
		s.AddLog(conn, fmt.Sprintf("✓ 镜像: %d", info.Images))

		// 获取容器列表
		containers, err := s.getDockerContainers(client, baseURL)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("⚠ 获取容器列表失败: %v", err))
		} else {
			s.AddLog(conn, fmt.Sprintf("✓ 获取到 %d 个容器", len(containers)))
		}

		// 获取镜像列表
		images, err := s.getDockerImages(client, baseURL)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("⚠ 获取镜像列表失败: %v", err))
		} else {
			s.AddLog(conn, fmt.Sprintf("✓ 获取到 %d 个镜像", len(images)))
		}

		// 构建结果信息
		result := s.buildDockerResult(version, info, containers, images)

		conn.Status = "success"
		conn.Message = fmt.Sprintf("Docker API 未授权访问 (版本: %s)", version.Version)
		s.SetConnectionResult(conn, result)
		conn.ConnectedAt = time.Now()
		return
	}

	// 所有尝试都失败
	conn.Status = "failed"
	conn.Message = fmt.Sprintf("连接失败: %v", lastErr)
	s.AddLog(conn, "所有连接尝试均失败")
}

// createDockerHTTPClient 创建 Docker HTTP 客户端
func (s *ConnectorService) createDockerHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.DialContextWithProxy(ctx, network, addr)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// getDockerVersion 获取 Docker 版本信息
func (s *ConnectorService) getDockerVersion(client *http.Client, baseURL string) (*DockerVersion, error) {
	resp, err := client.Get(baseURL + "/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var version DockerVersion
	if err := json.Unmarshal(body, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// getDockerInfo 获取 Docker 系统信息
func (s *ConnectorService) getDockerInfo(client *http.Client, baseURL string) (*DockerInfo, error) {
	resp, err := client.Get(baseURL + "/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info DockerInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// getDockerContainers 获取 Docker 容器列表
func (s *ConnectorService) getDockerContainers(client *http.Client, baseURL string) ([]DockerContainer, error) {
	resp, err := client.Get(baseURL + "/containers/json?all=true")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var containers []DockerContainer
	if err := json.Unmarshal(body, &containers); err != nil {
		return nil, err
	}

	return containers, nil
}

// getDockerImages 获取 Docker 镜像列表
func (s *ConnectorService) getDockerImages(client *http.Client, baseURL string) ([]DockerImage, error) {
	resp, err := client.Get(baseURL + "/images/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var images []DockerImage
	if err := json.Unmarshal(body, &images); err != nil {
		return nil, err
	}

	return images, nil
}

// buildDockerResult 构建 Docker 结果信息
func (s *ConnectorService) buildDockerResult(version *DockerVersion, info *DockerInfo, containers []DockerContainer, images []DockerImage) string {
	var result strings.Builder

	result.WriteString("=== Docker API 未授权访问 ===\n\n")

	// 版本信息
	if version != nil {
		result.WriteString("【版本信息】\n")
		result.WriteString(fmt.Sprintf("  版本: %s\n", version.Version))
		result.WriteString(fmt.Sprintf("  API 版本: %s\n", version.ApiVersion))
		result.WriteString(fmt.Sprintf("  Go 版本: %s\n", version.GoVersion))
		result.WriteString(fmt.Sprintf("  系统: %s/%s\n", version.Os, version.Arch))
		if version.KernelVersion != "" {
			result.WriteString(fmt.Sprintf("  内核版本: %s\n", version.KernelVersion))
		}
		result.WriteString("\n")
	}

	// 系统信息
	if info != nil {
		result.WriteString("【系统信息】\n")
		result.WriteString(fmt.Sprintf("  主机名: %s\n", info.Name))
		result.WriteString(fmt.Sprintf("  操作系统: %s\n", info.OperatingSystem))
		result.WriteString(fmt.Sprintf("  架构: %s\n", info.Architecture))
		result.WriteString(fmt.Sprintf("  CPU 核心数: %d\n", info.NCPU))
		result.WriteString(fmt.Sprintf("  内存: %.2f GB\n", float64(info.MemTotal)/(1024*1024*1024)))
		result.WriteString(fmt.Sprintf("  存储驱动: %s\n", info.Driver))
		result.WriteString(fmt.Sprintf("  容器总数: %d (运行: %d, 暂停: %d, 停止: %d)\n",
			info.Containers, info.ContainersRunning, info.ContainersPaused, info.ContainersStopped))
		result.WriteString(fmt.Sprintf("  镜像数: %d\n", info.Images))
		result.WriteString("\n")
	}

	// 容器列表
	if len(containers) > 0 {
		result.WriteString(fmt.Sprintf("【容器列表】(共 %d 个)\n", len(containers)))
		displayCount := len(containers)
		if displayCount > 10 {
			displayCount = 10
		}
		for i := 0; i < displayCount; i++ {
			container := containers[i]
			name := "未命名"
			if len(container.Names) > 0 {
				name = strings.TrimPrefix(container.Names[0], "/")
			}
			result.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, name))
			result.WriteString(fmt.Sprintf("      镜像: %s\n", container.Image))
			result.WriteString(fmt.Sprintf("      状态: %s (%s)\n", container.State, container.Status))
			if len(container.Ports) > 0 {
				var ports []string
				for _, port := range container.Ports {
					if port.PublicPort > 0 {
						ports = append(ports, fmt.Sprintf("%d->%d/%s", port.PublicPort, port.PrivatePort, port.Type))
					} else {
						ports = append(ports, fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
					}
				}
				result.WriteString(fmt.Sprintf("      端口: %s\n", strings.Join(ports, ", ")))
			}
		}
		if len(containers) > 10 {
			result.WriteString(fmt.Sprintf("  ... 还有 %d 个容器\n", len(containers)-10))
		}
		result.WriteString("\n")
	}

	// 镜像列表
	if len(images) > 0 {
		result.WriteString(fmt.Sprintf("【镜像列表】(共 %d 个)\n", len(images)))
		displayCount := len(images)
		if displayCount > 10 {
			displayCount = 10
		}
		for i := 0; i < displayCount; i++ {
			image := images[i]
			tags := "无标签"
			if len(image.RepoTags) > 0 && image.RepoTags[0] != "<none>:<none>" {
				tags = strings.Join(image.RepoTags, ", ")
			}
			result.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, tags))
			result.WriteString(fmt.Sprintf("      大小: %.2f MB\n", float64(image.Size)/(1024*1024)))
			result.WriteString(fmt.Sprintf("      创建时间: %s\n", time.Unix(image.Created, 0).Format("2006-01-02 15:04:05")))
		}
		if len(images) > 10 {
			result.WriteString(fmt.Sprintf("  ... 还有 %d 个镜像\n", len(images)-10))
		}
		result.WriteString("\n")
	}

	// result.WriteString("【安全建议】\n")
	// result.WriteString("  ⚠ Docker API 未启用访问控制，存在严重安全风险\n")
	// result.WriteString("  ⚠ 攻击者可以完全控制 Docker 守护进程\n")
	// result.WriteString("  ⚠ 建议配置 TLS 认证或使用防火墙限制访问\n")
	// result.WriteString("  ⚠ 不要将 Docker API 暴露在公网\n")

	return result.String()
}

// ExecuteDockerCommand 执行 Docker 命令
func (s *ConnectorService) ExecuteDockerCommand(conn *models.Connection, command string) (string, error) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	
	// 确定使用的协议
	scheme := "http"
	if conn.Port == "2376" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	client := s.createDockerHTTPClient()

	// 解析命令
	command = strings.TrimSpace(command)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("命令为空")
	}

	// 支持的命令
	switch parts[0] {
	case "ps", "containers":
		// 列出容器
		containers, err := s.getDockerContainers(client, baseURL)
		if err != nil {
			return "", err
		}
		return s.formatContainersList(containers), nil

	case "images":
		// 列出镜像
		images, err := s.getDockerImages(client, baseURL)
		if err != nil {
			return "", err
		}
		return s.formatImagesList(images), nil

	case "info":
		// 系统信息
		info, err := s.getDockerInfo(client, baseURL)
		if err != nil {
			return "", err
		}
		return s.formatDockerInfo(info), nil

	case "version":
		// 版本信息
		version, err := s.getDockerVersion(client, baseURL)
		if err != nil {
			return "", err
		}
		return s.formatDockerVersion(version), nil

	default:
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: ps, containers, images, info, version", parts[0])
	}
}

// ExecuteContainerCommand 在容器中执行命令
func (s *ConnectorService) ExecuteContainerCommand(conn *models.Connection, containerID, cmd string) (string, error) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	
	// 确定使用的协议
	scheme := "http"
	if conn.Port == "2376" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	client := s.createDockerHTTPClient()

	// 创建 exec 实例
	execConfig := map[string]interface{}{
		"AttachStdin":  false,
		"AttachStdout": true,
		"AttachStderr": true,
		"Tty":          false,
		"Cmd":          []string{"/bin/sh", "-c", cmd},
	}

	execConfigJSON, err := json.Marshal(execConfig)
	if err != nil {
		return "", fmt.Errorf("序列化 exec 配置失败: %v", err)
	}

	// 创建 exec
	createURL := fmt.Sprintf("%s/containers/%s/exec", baseURL, containerID)
	resp, err := client.Post(createURL, "application/json", strings.NewReader(string(execConfigJSON)))
	if err != nil {
		return "", fmt.Errorf("创建 exec 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("创建 exec 失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var execCreateResp struct {
		Id string `json:"Id"`
	}
	if err := json.Unmarshal(body, &execCreateResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 启动 exec
	startURL := fmt.Sprintf("%s/exec/%s/start", baseURL, execCreateResp.Id)
	startConfig := map[string]interface{}{
		"Detach": false,
		"Tty":    false,
	}
	startConfigJSON, _ := json.Marshal(startConfig)

	resp2, err := client.Post(startURL, "application/json", strings.NewReader(string(startConfigJSON)))
	if err != nil {
		return "", fmt.Errorf("启动 exec 失败: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		return "", fmt.Errorf("启动 exec 失败 (状态码: %d): %s", resp2.StatusCode, string(body))
	}

	// 读取输出
	output, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("读取输出失败: %v", err)
	}

	// Docker API 返回的是原始流，可能包含 Docker 流头部（8字节）
	// 尝试去除流头部
	result := string(output)
	if len(output) > 8 {
		// 检查是否有 Docker 流头部
		if output[0] <= 2 { // stdout(1) 或 stderr(2)
			// 跳过头部
			result = string(output[8:])
		}
	}

	return result, nil
}

// GetDockerContainersJSON 获取 Docker 容器列表的 JSON 数据
func (s *ConnectorService) GetDockerContainersJSON(conn *models.Connection) (string, error) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	
	// 确定使用的协议
	scheme := "http"
	if conn.Port == "2376" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	client := s.createDockerHTTPClient()

	// 获取容器列表
	containers, err := s.getDockerContainers(client, baseURL)
	if err != nil {
		return "", err
	}

	// 转换为 JSON
	jsonData, err := json.Marshal(containers)
	if err != nil {
		return "", fmt.Errorf("序列化容器列表失败: %v", err)
	}

	return string(jsonData), nil
}

// formatContainersList 格式化容器列表
func (s *ConnectorService) formatContainersList(containers []DockerContainer) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("容器列表 (共 %d 个):\n\n", len(containers)))

	for i, container := range containers {
		name := "未命名"
		if len(container.Names) > 0 {
			name = strings.TrimPrefix(container.Names[0], "/")
		}
		result.WriteString(fmt.Sprintf("[%d] %s\n", i+1, name))
		result.WriteString(fmt.Sprintf("    ID: %s\n", container.Id[:12]))
		result.WriteString(fmt.Sprintf("    镜像: %s\n", container.Image))
		result.WriteString(fmt.Sprintf("    状态: %s (%s)\n", container.State, container.Status))
		if len(container.Ports) > 0 {
			var ports []string
			for _, port := range container.Ports {
				if port.PublicPort > 0 {
					ports = append(ports, fmt.Sprintf("%d->%d/%s", port.PublicPort, port.PrivatePort, port.Type))
				} else {
					ports = append(ports, fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
				}
			}
			result.WriteString(fmt.Sprintf("    端口: %s\n", strings.Join(ports, ", ")))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// formatImagesList 格式化镜像列表
func (s *ConnectorService) formatImagesList(images []DockerImage) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("镜像列表 (共 %d 个):\n\n", len(images)))

	for i, image := range images {
		tags := "无标签"
		if len(image.RepoTags) > 0 && image.RepoTags[0] != "<none>:<none>" {
			tags = strings.Join(image.RepoTags, ", ")
		}
		result.WriteString(fmt.Sprintf("[%d] %s\n", i+1, tags))
		result.WriteString(fmt.Sprintf("    ID: %s\n", image.Id[:19]))
		result.WriteString(fmt.Sprintf("    大小: %.2f MB\n", float64(image.Size)/(1024*1024)))
		result.WriteString(fmt.Sprintf("    创建时间: %s\n", time.Unix(image.Created, 0).Format("2006-01-02 15:04:05")))
		result.WriteString("\n")
	}

	return result.String()
}

// formatDockerInfo 格式化 Docker 系统信息
func (s *ConnectorService) formatDockerInfo(info *DockerInfo) string {
	var result strings.Builder
	result.WriteString("Docker 系统信息:\n\n")
	result.WriteString(fmt.Sprintf("主机名: %s\n", info.Name))
	result.WriteString(fmt.Sprintf("操作系统: %s\n", info.OperatingSystem))
	result.WriteString(fmt.Sprintf("架构: %s\n", info.Architecture))
	result.WriteString(fmt.Sprintf("CPU 核心数: %d\n", info.NCPU))
	result.WriteString(fmt.Sprintf("内存: %.2f GB\n", float64(info.MemTotal)/(1024*1024*1024)))
	result.WriteString(fmt.Sprintf("存储驱动: %s\n", info.Driver))
	result.WriteString(fmt.Sprintf("容器总数: %d (运行: %d, 暂停: %d, 停止: %d)\n",
		info.Containers, info.ContainersRunning, info.ContainersPaused, info.ContainersStopped))
	result.WriteString(fmt.Sprintf("镜像数: %d\n", info.Images))
	return result.String()
}

// formatDockerVersion 格式化 Docker 版本信息
func (s *ConnectorService) formatDockerVersion(version *DockerVersion) string {
	var result strings.Builder
	result.WriteString("Docker 版本信息:\n\n")
	result.WriteString(fmt.Sprintf("版本: %s\n", version.Version))
	result.WriteString(fmt.Sprintf("API 版本: %s\n", version.ApiVersion))
	result.WriteString(fmt.Sprintf("Go 版本: %s\n", version.GoVersion))
	result.WriteString(fmt.Sprintf("系统: %s/%s\n", version.Os, version.Arch))
	if version.KernelVersion != "" {
		result.WriteString(fmt.Sprintf("内核版本: %s\n", version.KernelVersion))
	}
	return result.String()
}
