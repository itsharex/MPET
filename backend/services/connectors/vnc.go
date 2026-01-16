package connectors

import (
	"MPET/backend/models"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"strings"
	"time"

	vnc "github.com/kward/go-vnc"
	"github.com/kward/go-vnc/rfbflags"
)

// ConnectVNC 检测 VNC 服务
func (s *ConnectorService) ConnectVNC(conn *models.Connection) {
	address := net.JoinHostPort(conn.IP, conn.Port)
	
	s.AddLog(conn, "开始连接 VNC 服务")
	s.AddLog(conn, fmt.Sprintf("目标地址: %s", address))

	// 使用代理拨号
	netConn, err := s.DialWithProxy("tcp", address)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("连接失败: %v", err))
		return
	}

	// 设置连接超时
	netConn.SetDeadline(time.Now().Add(30 * time.Second))

	s.AddLog(conn, "TCP 连接成功，开始 VNC 握手")

	// 创建上下文
	ctx := context.Background()

	// VNC 连接配置
	var vncConfig *vnc.ClientConfig
	
	// 如果提供了密码，使用密码认证
	if conn.Pass != "" {
		s.AddLog(conn, "使用密码认证")
		vncConfig = vnc.NewClientConfig(conn.Pass)
	} else {
		s.AddLog(conn, "尝试无密码连接")
		vncConfig = &vnc.ClientConfig{
			Auth: []vnc.ClientAuth{&vnc.ClientAuthNone{}},
		}
	}

	// 建立 VNC 连接
	vncClient, err := vnc.Connect(ctx, netConn, vncConfig)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("VNC 握手失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("VNC 握手失败: %v", err))
		netConn.Close()
		return
	}
	defer vncClient.Close()

	s.AddLog(conn, "VNC 握手成功")
	s.AddLog(conn, fmt.Sprintf("桌面名称: %s", vncClient.DesktopName()))
	s.AddLog(conn, fmt.Sprintf("分辨率: %dx%d", vncClient.FramebufferWidth(), vncClient.FramebufferHeight()))

	// 设置编码类型，确保支持 RawEncoding
	s.AddLog(conn, "设置编码类型...")
	err = vncClient.SetEncodings(nil)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("设置编码失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("设置编码失败: %v", err))
		return
	}

	conn.Status = "success"
	conn.Message = "VNC 连接成功"
	conn.ConnectedAt = time.Now()

	// 构建结果
	result := fmt.Sprintf("VNC 服务连接成功\n")
	result += strings.Repeat("=", 45) + "\n\n"
	result += fmt.Sprintf("目标地址: %s:%s\n", conn.IP, conn.Port)
	result += fmt.Sprintf("桌面名称: %s\n", vncClient.DesktopName())
	result += fmt.Sprintf("分辨率: %dx%d\n", vncClient.FramebufferWidth(), vncClient.FramebufferHeight())

	result += "\n服务信息:\n"
	result += strings.Repeat("=", 45) + "\n"
	result += "- VNC 服务已开放\n"
	result += "- 远程桌面可访问\n"
	if conn.Pass == "" {
		result += "- 未使用密码认证（安全风险）\n"
	} else {
		result += "- 使用密码认证\n"
	}

	result += "\n可用命令:\n"
	result += strings.Repeat("=", 45) + "\n"
	result += "- screenshot: 获取屏幕截图\n"

	result += "\n安全风险:\n"
	result += strings.Repeat("=", 45) + "\n"
	result += "- VNC 未加密传输，可能被中间人攻击\n"
	result += "- 弱密码或无密码可能导致未授权访问\n"
	result += "- 建议使用 SSH 隧道或 VPN 加密 VNC 连接\n"
	result += "- 建议启用强密码认证\n"

	s.SetConnectionResult(conn, result)
	s.AddLog(conn, "VNC 检测完成")
}

// imageToBase64 将图像转换为 Base64 编码的 PNG
func (s *ConnectorService) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	
	// 编码为 PNG
	err := png.Encode(&buf, img)
	if err != nil {
		return "", fmt.Errorf("PNG 编码失败: %v", err)
	}
	
	// 转换为 Base64
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return base64Str, nil
}

// framebufferToImage 将 FramebufferUpdate 转换为图像
func (s *ConnectorService) framebufferToImage(fb *vnc.FramebufferUpdate, width, height uint16) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	
	// 检测颜色范围以确定正确的转换方法
	var maxR, maxG, maxB uint16 = 0, 0, 0
	for _, rect := range fb.Rects {
		if rawEnc, ok := rect.Enc.(*vnc.RawEncoding); ok {
			for _, color := range rawEnc.Colors {
				if color.R > maxR {
					maxR = color.R
				}
				if color.G > maxG {
					maxG = color.G
				}
				if color.B > maxB {
					maxB = color.B
				}
			}
		}
	}
	
	// 根据颜色范围选择转换方法
	// 如果最大值小于 256，说明颜色值已经是 8 位的，不需要右移
	// 如果最大值大于 256，说明颜色值是 16 位的，需要右移 8 位
	useShift := maxR > 255 || maxG > 255 || maxB > 255
	
	for _, rect := range fb.Rects {
		switch enc := rect.Enc.(type) {
		case *vnc.RawEncoding:
			for y := uint16(0); y < rect.Height; y++ {
				for x := uint16(0); x < rect.Width; x++ {
					idx := int(y)*int(rect.Width) + int(x)
					if idx < len(enc.Colors) {
						vncColor := enc.Colors[idx]
						
						pixelX := int(rect.X + x)
						pixelY := int(rect.Y + y)
						if pixelX < int(width) && pixelY < int(height) {
							var r8, g8, b8 uint8
							if useShift {
								// 16 位颜色，右移 8 位
								r8 = uint8(vncColor.R >> 8)
								g8 = uint8(vncColor.G >> 8)
								b8 = uint8(vncColor.B >> 8)
							} else {
								// 8 位颜色，直接使用
								r8 = uint8(vncColor.R)
								g8 = uint8(vncColor.G)
								b8 = uint8(vncColor.B)
							}
							
							img.Set(pixelX, pixelY, color.RGBA{
								R: r8,
								G: g8,
								B: b8,
								A: 255,
							})
						}
					}
				}
			}
		case *vnc.DesktopSizePseudoEncoding:
			continue
		default:
			continue
		}
	}
	
	return img
}

// ExecuteVNCCommand 执行 VNC 命令
func (s *ConnectorService) ExecuteVNCCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("VNC 连接未建立")
	}

	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	s.AddLog(conn, fmt.Sprintf("开始执行 VNC 命令: %s", cmd))

	// 支持的命令
	switch strings.ToLower(cmd) {
	case "screenshot":
		return s.executeVNCScreenshot(conn)
	default:
		return "", fmt.Errorf("不支持的命令: %s", cmd)
	}
}

// executeVNCScreenshot 执行 VNC 截图命令
func (s *ConnectorService) executeVNCScreenshot(conn *models.Connection) (string, error) {
	address := net.JoinHostPort(conn.IP, conn.Port)
	
	s.AddLog(conn, "开始连接 VNC 服务进行截图...")
	s.AddLog(conn, fmt.Sprintf("目标地址: %s", address))

	// 使用代理拨号
	netConn, err := s.DialWithProxy("tcp", address)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("连接失败: %v", err))
		return "", fmt.Errorf("连接失败: %v", err)
	}

	// 设置连接超时
	netConn.SetDeadline(time.Now().Add(30 * time.Second))

	s.AddLog(conn, "TCP 连接成功，开始 VNC 握手")

	// 创建上下文
	ctx := context.Background()

	// 创建消息通道接收截图数据
	msgCh := make(chan vnc.ServerMessage, 10)

	// VNC 连接配置
	var vncConfig *vnc.ClientConfig
	
	// 如果提供了密码，使用密码认证
	if conn.Pass != "" {
		s.AddLog(conn, "使用密码认证")
		vncConfig = vnc.NewClientConfig(conn.Pass)
	} else {
		s.AddLog(conn, "尝试无密码连接")
		vncConfig = &vnc.ClientConfig{
			Auth: []vnc.ClientAuth{&vnc.ClientAuthNone{}},
		}
	}
	
	// 设置消息通道
	vncConfig.ServerMessageCh = msgCh
	vncConfig.ServerMessages = []vnc.ServerMessage{
		&vnc.FramebufferUpdate{},
		&vnc.SetColorMapEntries{},
		&vnc.Bell{},
		&vnc.ServerCutText{},
	}

	// 建立 VNC 连接
	vncClient, err := vnc.Connect(ctx, netConn, vncConfig)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("VNC 握手失败: %v", err))
		netConn.Close()
		return "", fmt.Errorf("VNC 握手失败: %v", err)
	}
	defer vncClient.Close()

	s.AddLog(conn, "VNC 握手成功")
	s.AddLog(conn, fmt.Sprintf("分辨率: %dx%d", vncClient.FramebufferWidth(), vncClient.FramebufferHeight()))

	// 设置编码类型，确保支持 RawEncoding
	s.AddLog(conn, "设置编码类型...")
	err = vncClient.SetEncodings(nil)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("设置编码失败: %v", err))
		return "", fmt.Errorf("设置编码失败: %v", err)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 启动消息监听器（在 goroutine 中）
	done := make(chan struct{})
	go func() {
		defer close(done)
		vncClient.ListenAndHandle()
	}()

	// 请求屏幕更新并获取截图
	s.AddLog(conn, "请求屏幕截图...")
	
	// 请求完整的屏幕更新
	err = vncClient.FramebufferUpdateRequest(rfbflags.RFBFalse, 0, 0, vncClient.FramebufferWidth(), vncClient.FramebufferHeight())
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("请求屏幕更新失败: %v", err))
		cancel()
		return "", fmt.Errorf("请求屏幕更新失败: %v", err)
	}

	// 等待接收截图数据
	s.AddLog(conn, "等待接收屏幕数据...")
	var screenshot image.Image
	
	select {
	case msg := <-msgCh:
		// 检查是否是 FramebufferUpdate 消息
		if fbUpdate, ok := msg.(*vnc.FramebufferUpdate); ok {
			s.AddLog(conn, fmt.Sprintf("收到屏幕更新消息，包含 %d 个矩形", len(fbUpdate.Rects)))
			// 将 FramebufferUpdate 转换为图像
			screenshot = s.framebufferToImage(fbUpdate, vncClient.FramebufferWidth(), vncClient.FramebufferHeight())
		} else {
			s.AddLog(conn, fmt.Sprintf("收到非预期的消息类型: %T", msg))
		}
	case <-ctx.Done():
		s.AddLog(conn, "等待截图超时")
		return "", fmt.Errorf("等待截图超时")
	}

	if screenshot == nil {
		s.AddLog(conn, "未获取到屏幕数据")
		return "", fmt.Errorf("未获取到屏幕数据")
	}

	// 将截图转换为 Base64
	s.AddLog(conn, "正在编码截图...")
	screenshotBase64, err := s.imageToBase64(screenshot)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("截图编码失败: %v", err))
		return "", fmt.Errorf("截图编码失败: %v", err)
	}

	s.AddLog(conn, fmt.Sprintf("截图编码成功，大小: %d 字节", len(screenshotBase64)))
	
	// 构建结果
	result := fmt.Sprintf("VNC 截图执行成功\n")
	result += strings.Repeat("=", 45) + "\n\n"
	result += fmt.Sprintf("目标地址: %s:%s\n", conn.IP, conn.Port)
	result += fmt.Sprintf("分辨率: %dx%d\n", vncClient.FramebufferWidth(), vncClient.FramebufferHeight())
	result += fmt.Sprintf("\n截图: [BASE64_IMAGE]%s[/BASE64_IMAGE]\n\n", screenshotBase64)
	
	return result, nil
}
