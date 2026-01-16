package connectors

import (
	"MPET/backend/models"
	"fmt"
	"net"
	"strings"
	"time"
)

// RMI 协议常量
const (
	RMI_MAGIC       = 0x4a524d49 // "JRMI"
	RMI_VERSION     = 2
	PROTOCOL_JRMP   = 0x4b
	PROTOCOL_STREAM = 0x4c
)

// ConnectRMI 检测 RMI 服务
func (s *ConnectorService) ConnectRMI(conn *models.Connection) {
	address := net.JoinHostPort(conn.IP, conn.Port)
	
	// 使用代理拨号
	netConn, err := s.DialWithProxy("tcp", address)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("连接失败: %v", err))
		return
	}
	defer netConn.Close()

	// 设置超时
	netConn.SetDeadline(time.Now().Add(5 * time.Second))

	s.AddLog(conn, "TCP 连接成功，开始 RMI 协议检测")

	// 发送 JRMI 握手包
	handshake := []byte{0x4a, 0x52, 0x4d, 0x49, 0x00, 0x02, 0x4b}

	_, err = netConn.Write(handshake)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("发送握手包失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("发送握手包失败: %v", err))
		return
	}

	s.AddLog(conn, "已发送 RMI 握手包 (JRMI v2 + SingleOpProtocol)")

	// 读取响应
	response := make([]byte, 256)
	netConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := netConn.Read(response)
	
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("读取响应失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("读取响应失败: %v", err))
		return
	}

	s.AddLog(conn, fmt.Sprintf("收到响应: %d 字节", n))
	
	// 检查响应
	isRMI := false
	responseInfo := ""
	
	if n > 0 && response[0] == 0x4e {
		isRMI = true
		responseInfo = "RMI 协议握手成功 (0x4e - Protocol Accepted)"
		s.AddLog(conn, responseInfo)
		if n > 1 {
			s.AddLog(conn, fmt.Sprintf("响应数据: %x", response[:min(n, 32)]))
		}
	} else {
		conn.Status = "failed"
		conn.Message = "不是有效的 RMI 服务"
		s.AddLog(conn, "不是有效的 RMI 服务")
		return
	}

	if !isRMI {
		conn.Status = "failed"
		conn.Message = "不是有效的 RMI 服务"
		return
	}

	conn.Status = "success"
	conn.Message = "RMI 服务可访问"
	conn.ConnectedAt = time.Now()
	
	result := fmt.Sprintf("RMI 服务检测成功\n")
	result += fmt.Sprintf("目标地址: %s:%s\n", conn.IP, conn.Port)
	result += fmt.Sprintf("协议: Java RMI (Remote Method Invocation)\n")
	result += fmt.Sprintf("响应: %s\n", responseInfo)
	
	result += "\n服务信息:\n"
	result += strings.Repeat("=", 60) + "\n"
	result += "- RMI Registry 端口已开放\n"
	result += "- 服务响应 RMI 协议请求\n"
	result += "- 可能存在未授权访问\n"
	
	// result += "\n安全风险:\n"
	// result += strings.Repeat("=", 60) + "\n"
	// result += "- RMI 服务未授权访问可能导致远程代码执行\n"
	// result += "- 攻击者可以通过反序列化漏洞执行任意代码\n"
	// result += "- 可能暴露敏感的远程对象和方法\n"
	// result += "- 建议启用 RMI 认证和加密\n"
	
	result += "\n推荐工具:\n"
	result += strings.Repeat("=", 60) + "\n"
	result += "1. BaRMIe - RMI 枚举和利用工具\n"
	result += fmt.Sprintf("   java -jar BaRMIe.jar -enum %s %s\n\n", conn.IP, conn.Port)
	result += "2. rmg - Remote Method Guesser\n"
	result += fmt.Sprintf("   java -jar rmg.jar enum %s %s\n\n", conn.IP, conn.Port)
	result += "3. GoEvilRmi - Go 实现的 RMI 利用工具\n"
	result += "   https://github.com/m4xxxxx/GoEvilRmi\n\n"
	result += "4. ysoserial - Java 反序列化利用工具\n"
	result += "   https://github.com/frohoff/ysoserial\n"
	
	s.SetConnectionResult(conn, result)
	s.AddLog(conn, "RMI 检测完成")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
