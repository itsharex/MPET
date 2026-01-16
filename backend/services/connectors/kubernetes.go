package connectors

import (
	"MPET/backend/models"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// K8sVersion Kubernetes版本信息
type K8sVersion struct {
	Major        string `json:"major"`
	Minor        string `json:"minor"`
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

// K8sPod Pod信息
type K8sPod struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Status struct {
		Phase string `json:"phase"`
	} `json:"status"`
	Spec struct {
		Containers []struct {
			Name  string `json:"name"`
			Image string `json:"image"`
		} `json:"containers"`
	} `json:"spec"`
}

// K8sPodList Pod列表
type K8sPodList struct {
	Items []K8sPod `json:"items"`
}

// K8sNamespace 命名空间信息
type K8sNamespace struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

// K8sNamespaceList 命名空间列表
type K8sNamespaceList struct {
	Items []K8sNamespace `json:"items"`
}

// K8sSecret Secret信息
type K8sSecret struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

// K8sSecretList Secret列表
type K8sSecretList struct {
	Items []K8sSecret `json:"items"`
}

// ConnectKubernetes 连接 Kubernetes API
func (s *ConnectorService) ConnectKubernetes(conn *models.Connection) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	// 尝试 HTTPS 和 HTTP
	schemes := []string{"https", "http"}
	if conn.Port == "8080" || conn.Port == "10255" {
		schemes = []string{"http", "https"}
	}

	var lastErr error
	for _, scheme := range schemes {
		baseURL := fmt.Sprintf("%s://%s", scheme, addr)
		s.AddLog(conn, fmt.Sprintf("尝试 %s 协议", strings.ToUpper(scheme)))

		// 创建 HTTP 客户端
		client := s.createK8sHTTPClient()

		// 尝试获取版本信息
		version, err := s.getK8sVersion(client, baseURL)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ 获取版本信息失败: %v", err))
			lastErr = err
			continue
		}

		s.AddLog(conn, fmt.Sprintf("✓ Kubernetes 版本: %s", version.GitVersion))

		// 获取命名空间
		namespaces, err := s.getK8sNamespaces(client, baseURL)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("⚠ 获取命名空间失败: %v", err))
			namespaces = []string{}
		} else {
			s.AddLog(conn, fmt.Sprintf("✓ 命名空间: %d 个", len(namespaces)))
		}

		// 获取 Pod 列表
		pods, err := s.getK8sPods(client, baseURL)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("⚠ 获取 Pod 列表失败: %v", err))
		} else {
			s.AddLog(conn, fmt.Sprintf("✓ 获取到 %d 个 Pod", len(pods)))
		}

		// 获取 Secret（限制5个）
		secrets, err := s.getK8sSecrets(client, baseURL, 5)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("⚠ 获取 Secret 失败: %v", err))
		} else if len(secrets) > 0 {
			s.AddLog(conn, fmt.Sprintf("⚠ 发现 %d 个 Secret（可能包含敏感信息）", len(secrets)))
		}

		// 构建结果信息
		result := s.buildK8sResult(version, namespaces, pods, secrets)

		conn.Status = "success"
		conn.Message = fmt.Sprintf("Kubernetes API 未授权访问 (版本: %s)", version.GitVersion)
		s.SetConnectionResult(conn, result)
		conn.ConnectedAt = time.Now()
		return
	}

	// 所有尝试都失败
	conn.Status = "failed"
	conn.Message = fmt.Sprintf("连接失败: %v", lastErr)
	s.AddLog(conn, "所有连接尝试均失败")
}

// createK8sHTTPClient 创建 Kubernetes HTTP 客户端
func (s *ConnectorService) createK8sHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.DialContextWithProxy(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// getK8sVersion 获取 Kubernetes 版本信息
func (s *ConnectorService) getK8sVersion(client *http.Client, baseURL string) (*K8sVersion, error) {
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

	var version K8sVersion
	if err := json.Unmarshal(body, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// getK8sNamespaces 获取命名空间列表
func (s *ConnectorService) getK8sNamespaces(client *http.Client, baseURL string) ([]string, error) {
	resp, err := client.Get(baseURL + "/api/v1/namespaces")
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

	var nsList K8sNamespaceList
	if err := json.Unmarshal(body, &nsList); err != nil {
		return nil, err
	}

	namespaces := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Metadata.Name)
	}

	return namespaces, nil
}

// getK8sPods 获取 Pod 列表
func (s *ConnectorService) getK8sPods(client *http.Client, baseURL string) ([]K8sPod, error) {
	resp, err := client.Get(baseURL + "/api/v1/pods")
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

	var podList K8sPodList
	if err := json.Unmarshal(body, &podList); err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// getK8sSecrets 获取 Secret 列表
func (s *ConnectorService) getK8sSecrets(client *http.Client, baseURL string, limit int) ([]K8sSecret, error) {
	url := fmt.Sprintf("%s/api/v1/secrets?limit=%d", baseURL, limit)
	resp, err := client.Get(url)
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

	var secretList K8sSecretList
	if err := json.Unmarshal(body, &secretList); err != nil {
		return nil, err
	}

	return secretList.Items, nil
}

// buildK8sResult 构建 Kubernetes 结果信息
func (s *ConnectorService) buildK8sResult(version *K8sVersion, namespaces []string, pods []K8sPod, secrets []K8sSecret) string {
	var result strings.Builder

	result.WriteString("=== Kubernetes API 未授权访问 ===\n\n")

	// 版本信息
	if version != nil {
		result.WriteString("【版本信息】\n")
		result.WriteString(fmt.Sprintf("  版本: %s\n", version.GitVersion))
		result.WriteString(fmt.Sprintf("  Go 版本: %s\n", version.GoVersion))
		result.WriteString(fmt.Sprintf("  平台: %s\n", version.Platform))
		result.WriteString(fmt.Sprintf("  构建日期: %s\n", version.BuildDate))
		result.WriteString("\n")
	}

	// 命名空间列表
	if len(namespaces) > 0 {
		result.WriteString(fmt.Sprintf("【命名空间】(共 %d 个)\n", len(namespaces)))
		displayCount := len(namespaces)
		if displayCount > 10 {
			displayCount = 10
		}
		for i := 0; i < displayCount; i++ {
			result.WriteString(fmt.Sprintf("  [%d] %s\n", i+1, namespaces[i]))
		}
		if len(namespaces) > 10 {
			result.WriteString(fmt.Sprintf("  ... 还有 %d 个命名空间\n", len(namespaces)-10))
		}
		result.WriteString("\n")
	}

	// Pod 列表
	if len(pods) > 0 {
		result.WriteString(fmt.Sprintf("【Pod 列表】(共 %d 个)\n", len(pods)))
		displayCount := len(pods)
		if displayCount > 10 {
			displayCount = 10
		}
		for i := 0; i < displayCount; i++ {
			pod := pods[i]
			result.WriteString(fmt.Sprintf("  [%d] %s/%s\n", i+1, pod.Metadata.Namespace, pod.Metadata.Name))
			result.WriteString(fmt.Sprintf("      状态: %s\n", pod.Status.Phase))
			if len(pod.Spec.Containers) > 0 {
				result.WriteString(fmt.Sprintf("      镜像: %s\n", pod.Spec.Containers[0].Image))
			}
		}
		if len(pods) > 10 {
			result.WriteString(fmt.Sprintf("  ... 还有 %d 个 Pod\n", len(pods)-10))
		}
		result.WriteString("\n")
	}

	// Secret 列表
	if len(secrets) > 0 {
		result.WriteString(fmt.Sprintf("【⚠️  Secret】(显示前 %d 个)\n", len(secrets)))
		for i, secret := range secrets {
			result.WriteString(fmt.Sprintf("  [%d] %s/%s\n", i+1, secret.Metadata.Namespace, secret.Metadata.Name))
			result.WriteString(fmt.Sprintf("      类型: %s\n", secret.Type))
			result.WriteString(fmt.Sprintf("      数据项: %d 个\n", len(secret.Data)))
		}
		result.WriteString("  ⚠️  Secret 可能包含敏感信息（密码、Token 等）\n")
		result.WriteString("\n")
	}

	// result.WriteString("【安全建议】\n")
	// result.WriteString("  ⚠ Kubernetes API Server 未启用访问控制，存在严重安全风险\n")
	// result.WriteString("  ⚠ 攻击者可以完全控制集群资源\n")
	// result.WriteString("  ⚠ 建议启用 RBAC（基于角色的访问控制）\n")
	// result.WriteString("  ⚠ 配置 TLS 客户端证书认证\n")
	// result.WriteString("  ⚠ 使用防火墙限制 API Server 访问\n")
	// result.WriteString("  ⚠ 不要将 API Server 暴露在公网\n")

	return result.String()
}

// ExecuteKubernetesCommand 执行 Kubernetes 命令
func (s *ConnectorService) ExecuteKubernetesCommand(conn *models.Connection, command string) (string, error) {
	addr := net.JoinHostPort(conn.IP, conn.Port)

	// 确定使用的协议
	scheme := "https"
	if conn.Port == "8080" || conn.Port == "10255" {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	client := s.createK8sHTTPClient()

	// 解析命令
	command = strings.TrimSpace(command)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("命令为空")
	}

	// 支持的命令
	switch parts[0] {
	case "namespaces", "ns":
		// 列出命名空间
		namespaces, err := s.getK8sNamespaces(client, baseURL)
		if err != nil {
			return "", err
		}
		return s.formatK8sNamespaces(namespaces), nil

	case "pods":
		// 列出 Pod
		pods, err := s.getK8sPods(client, baseURL)
		if err != nil {
			return "", err
		}
		return s.formatK8sPods(pods), nil

	case "secrets":
		// 列出 Secret
		limit := 10
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &limit)
		}
		secrets, err := s.getK8sSecrets(client, baseURL, limit)
		if err != nil {
			return "", err
		}
		return s.formatK8sSecrets(secrets), nil

	case "version":
		// 版本信息
		version, err := s.getK8sVersion(client, baseURL)
		if err != nil {
			return "", err
		}
		return s.formatK8sVersion(version), nil

	default:
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: namespaces, pods, secrets, version", parts[0])
	}
}

// formatK8sNamespaces 格式化命名空间列表
func (s *ConnectorService) formatK8sNamespaces(namespaces []string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间列表 (共 %d 个):\n\n", len(namespaces)))

	for i, ns := range namespaces {
		result.WriteString(fmt.Sprintf("[%d] %s\n", i+1, ns))
	}

	return result.String()
}

// formatK8sPods 格式化 Pod 列表
func (s *ConnectorService) formatK8sPods(pods []K8sPod) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Pod 列表 (共 %d 个):\n\n", len(pods)))

	for i, pod := range pods {
		result.WriteString(fmt.Sprintf("[%d] %s/%s\n", i+1, pod.Metadata.Namespace, pod.Metadata.Name))
		result.WriteString(fmt.Sprintf("    状态: %s\n", pod.Status.Phase))
		if len(pod.Spec.Containers) > 0 {
			result.WriteString("    容器:\n")
			for _, container := range pod.Spec.Containers {
				result.WriteString(fmt.Sprintf("      - %s: %s\n", container.Name, container.Image))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// formatK8sSecrets 格式化 Secret 列表
func (s *ConnectorService) formatK8sSecrets(secrets []K8sSecret) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Secret 列表 (共 %d 个):\n\n", len(secrets)))

	for i, secret := range secrets {
		result.WriteString(fmt.Sprintf("[%d] %s/%s\n", i+1, secret.Metadata.Namespace, secret.Metadata.Name))
		result.WriteString(fmt.Sprintf("    类型: %s\n", secret.Type))
		result.WriteString(fmt.Sprintf("    数据项: %d 个\n", len(secret.Data)))
		if len(secret.Data) > 0 {
			result.WriteString("    键:\n")
			for key := range secret.Data {
				result.WriteString(fmt.Sprintf("      - %s\n", key))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// formatK8sVersion 格式化版本信息
func (s *ConnectorService) formatK8sVersion(version *K8sVersion) string {
	var result strings.Builder
	result.WriteString("Kubernetes 版本信息:\n\n")
	result.WriteString(fmt.Sprintf("版本: %s\n", version.GitVersion))
	result.WriteString(fmt.Sprintf("Go 版本: %s\n", version.GoVersion))
	result.WriteString(fmt.Sprintf("平台: %s\n", version.Platform))
	result.WriteString(fmt.Sprintf("构建日期: %s\n", version.BuildDate))
	result.WriteString(fmt.Sprintf("Git Commit: %s\n", version.GitCommit))
	return result.String()
}

// GetK8sPodsJSON 获取 Pod 列表的 JSON 数据
func (s *ConnectorService) GetK8sPodsJSON(conn *models.Connection) (string, error) {
	addr := net.JoinHostPort(conn.IP, conn.Port)

	// 确定使用的协议
	scheme := "https"
	if conn.Port == "8080" || conn.Port == "10255" {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	client := s.createK8sHTTPClient()

	// 获取 Pod 列表
	pods, err := s.getK8sPods(client, baseURL)
	if err != nil {
		return "", err
	}

	// 转换为 JSON
	jsonData, err := json.Marshal(pods)
	if err != nil {
		return "", fmt.Errorf("序列化 Pod 列表失败: %v", err)
	}

	return string(jsonData), nil
}
