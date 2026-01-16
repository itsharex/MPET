package connectors

import (
	"MPET/backend/models"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tomatome/grdp/core"
	"github.com/tomatome/grdp/glog"
	"github.com/tomatome/grdp/plugin"
	"github.com/tomatome/grdp/protocol/nla"
	"github.com/tomatome/grdp/protocol/pdu"
	"github.com/tomatome/grdp/protocol/sec"
	"github.com/tomatome/grdp/protocol/t125"
	"github.com/tomatome/grdp/protocol/tpkt"
	"github.com/tomatome/grdp/protocol/x224"
)

// RDPClient RDP 客户端结构
type RDPClient struct {
	Host         string          // 服务地址(ip:port)
	tpkt         *tpkt.TPKT      // TPKT协议层
	x224         *x224.X224      // X224协议层
	mcs          *t125.MCSClient // MCS协议层
	sec          *sec.Client     // 安全层
	pdu          *pdu.Client     // PDU协议层
	channels     *plugin.Channels // 插件通道
	screenImage  *image.RGBA     // 屏幕图像
	width        int             // 屏幕宽度
	height       int             // 屏幕高度
	connected    bool            // 连接状态
	mu           sync.Mutex      // 互斥锁
}

// RDPBitmap RDP 位图数据
type RDPBitmap struct {
	DestLeft     int
	DestTop      int
	DestRight    int
	DestBottom   int
	Width        int
	Height       int
	BitsPerPixel int
	IsCompress   bool
	Data         []byte
}

// ConnectRDP 连接到 RDP 服务器
func (s *ConnectorService) ConnectRDP(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "3389"
	}

	target := fmt.Sprintf("%s:%s", conn.IP, port)
	s.AddLog(conn, fmt.Sprintf("正在连接到 RDP 服务器 %s", target))

	// 创建 RDP 客户端配置
	domain := ""
	username := conn.User
	password := conn.Pass

	if username == "" {
		username = "Administrator"
	}

	// 解析域名（如果用户名包含域）
	if strings.Contains(username, "\\") {
		parts := strings.Split(username, "\\")
		domain = parts[0]
		username = parts[1]
	} else if strings.Contains(username, "/") {
		parts := strings.Split(username, "/")
		domain = parts[0]
		username = parts[1]
	}

	if domain == "" {
		domain = conn.IP
	}

	s.AddLog(conn, fmt.Sprintf("使用用户名: %s", username))
	if domain != conn.IP {
		s.AddLog(conn, fmt.Sprintf("域: %s", domain))
	}

	// 尝试 RDP 连接并获取截图
	s.AddLog(conn, "正在建立 RDP 连接...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	success, errMsg := s.tryRDPConnection(ctx, target, domain, username, password)
	
	if success {
		s.AddLog(conn, "✓ RDP 连接成功")
		s.AddLog(conn, "✓ 身份验证成功")

		// 保存结果
		result := fmt.Sprintf("RDP 服务连接成功\n")
		result += strings.Repeat("=", 45) + "\n\n"
		result += fmt.Sprintf("目标地址: %s:%s\n", conn.IP, conn.Port)
		result += fmt.Sprintf("用户名: %s\n", username)
		if domain != conn.IP {
			result += fmt.Sprintf("域: %s\n", domain)
		}
		
		result += "\n服务信息:\n"
		result += strings.Repeat("=", 45) + "\n"
		result += "- RDP 端口开放\n"
		result += "- RDP 协议握手成功\n"
		result += "- 身份验证成功\n"
		
		// result += "\n可用命令:\n"
		// result += strings.Repeat("=", 45) + "\n"
		// result += "- screenshot: 获取屏幕截图\n"
		
		// result += "\n安全风险:\n"
		// result += strings.Repeat("=", 45) + "\n"
		// result += "- RDP 服务已开放，建议使用强密码\n"
		// result += "- 建议启用网络级别身份验证 (NLA)\n"
		// result += "- 建议限制访问 IP 地址\n"
		// result += "- 建议使用 VPN 或 SSH 隧道加密连接\n"
		
		s.SetConnectionResult(conn, result)
		conn.Status = "success"
		conn.Message = "RDP 连接成功"
		conn.ConnectedAt = time.Now()
		s.AddLog(conn, "✓ RDP 检测完成")
	} else {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("RDP 连接失败: %s", errMsg)
		s.AddLog(conn, fmt.Sprintf("✗ %s", conn.Message))
	}
}

// tryRDPConnection 尝试 RDP 连接（仅测试连接，不截图）
func (s *ConnectorService) tryRDPConnection(ctx context.Context, target, domain, username, password string) (bool, string) {
	// 创建结果通道
	type result struct {
		success bool
		errMsg  string
	}
	resultChan := make(chan result, 1)

	// 在协程中进行 RDP 连接尝试
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("连接 panic: %v", r)
				select {
				case <-ctx.Done():
				case resultChan <- result{false, errMsg}:
				}
			}
		}()

		// 尝试 RDP 连接（仅测试）
		err := s.rdpConnectTest(target, domain, username, password, 25)

		if err == nil {
			select {
			case <-ctx.Done():
			case resultChan <- result{true, ""}:
			}
			return
		}

		// 连接失败
		errMsg := "认证失败"
		if err != nil {
			errMsg = err.Error()
			// 简化错误消息
			if strings.Contains(errMsg, "连接失败") {
				errMsg = "连接失败"
			} else if strings.Contains(errMsg, "超时") {
				errMsg = "连接超时"
			} else if strings.Contains(errMsg, "连接关闭") {
				errMsg = "连接被服务器关闭"
			} else if strings.Contains(errMsg, "refused") {
				errMsg = "连接被拒绝"
			}
		}

		select {
		case <-ctx.Done():
		case resultChan <- result{false, errMsg}:
		}
	}()

	// 等待连接结果或超时
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return false, "连接超时"
		}
		return false, "操作取消"
	case res := <-resultChan:
		return res.success, res.errMsg
	}
}

// rdpConnectTest RDP 连接测试（不截图）
func (s *ConnectorService) rdpConnectTest(target, domain, user, password string, timeout int64) error {
	// 配置日志级别为 NONE，避免输出大量日志
	glog.SetLevel(glog.NONE)
	logger := log.New(os.Stdout, "", 0)
	glog.SetLogger(logger)

	// 建立 TCP 连接
	var conn net.Conn
	var err error
	
	if s.Config.Proxy.Enabled {
		conn, err = s.DialWithProxy("tcp", target)
	} else {
		conn, err = net.DialTimeout("tcp", target, time.Duration(timeout)*time.Second)
	}
	
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()

	// 创建 RDP 客户端
	client := &RDPClient{
		Host:   target,
		width:  1280,
		height: 800,
	}

	// 初始化协议栈
	s.initProtocolStack(client, conn, domain, user, password)

	// 使用通道等待连接结果
	resultChan := make(chan error, 1)

	// 设置事件处理（仅测试连接）
	s.setupEventHandlersTest(client, resultChan)

	// 建立 X224 连接
	if err = client.x224.Connect(); err != nil {
		return fmt.Errorf("X224 连接错误: %v", err)
	}

	// 等待连接完成或超时
	select {
	case err := <-resultChan:
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		return fmt.Errorf("连接超时")
	}
}

// setupEventHandlersTest 设置 PDU 事件处理器（仅测试连接）
func (s *ConnectorService) setupEventHandlersTest(client *RDPClient, resultChan chan error) {
	// 错误处理
	client.pdu.On("error", func(e error) {
		select {
		case resultChan <- e:
		default:
		}
	})

	// 连接关闭
	client.pdu.On("close", func() {
		select {
		case resultChan <- fmt.Errorf("连接已关闭"):
		default:
		}
	})

	// 连接成功
	client.pdu.On("success", func() {
		client.mu.Lock()
		client.connected = true
		client.mu.Unlock()
		
		select {
		case resultChan <- nil:
		default:
		}
	})

	// 连接就绪
	client.pdu.On("ready", func() {
		client.mu.Lock()
		client.connected = true
		client.mu.Unlock()
		
		select {
		case resultChan <- nil:
		default:
		}
	})
}

// rdpConnectAndCapture RDP 连接并捕获屏幕
func (s *ConnectorService) rdpConnectAndCapture(target, domain, user, password string, timeout int64) (image.Image, error) {
	// 配置日志级别为 NONE，避免输出大量日志
	glog.SetLevel(glog.NONE)
	logger := log.New(os.Stdout, "", 0)
	glog.SetLogger(logger)

	// 创建 RDP 客户端
	client := &RDPClient{
		Host:   target,
		width:  1280,
		height: 800,
	}
	client.screenImage = image.NewRGBA(image.Rect(0, 0, client.width, client.height))

	// 登录并捕获屏幕
	screenshot, err := s.rdpLoginAndCapture(client, domain, user, password, timeout)
	if err != nil {
		return nil, err
	}

	return screenshot, nil
}

// rdpLoginAndCapture 执行 RDP 登录并捕获屏幕
func (s *ConnectorService) rdpLoginAndCapture(client *RDPClient, domain, user, pwd string, timeout int64) (image.Image, error) {
	// 建立 TCP 连接
	var conn net.Conn
	var err error
	
	if s.Config.Proxy.Enabled {
		conn, err = s.DialWithProxy("tcp", client.Host)
	} else {
		conn, err = net.DialTimeout("tcp", client.Host, time.Duration(timeout)*time.Second)
	}
	
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()

	// 初始化协议栈
	s.initProtocolStack(client, conn, domain, user, pwd)

	// 使用通道等待连接结果和截图
	resultChan := make(chan error, 1)
	screenshotChan := make(chan image.Image, 1)

	// 先设置事件处理（包括位图处理），再建立连接
	s.setupEventHandlersWithCapture(client, resultChan, screenshotChan)

	// 建立 X224 连接
	if err = client.x224.Connect(); err != nil {
		return nil, fmt.Errorf("X224 连接错误: %v", err)
	}

	// 等待连接完成或超时
	select {
	case err := <-resultChan:
		if err != nil {
			return nil, err
		}
		// 连接成功，等待截图（增加等待时间以接收更多位图数据）
		time.Sleep(3 * time.Second) // 等待3秒让系统完成登录和桌面加载
		
		select {
		case screenshot := <-screenshotChan:
			return screenshot, nil
		case <-time.After(5 * time.Second):
			// 如果5秒内没有收到截图，返回当前屏幕图像
			client.mu.Lock()
			hasData := client.screenImage != nil
			client.mu.Unlock()
			
			if hasData {
				return client.screenImage, nil
			}
			return nil, fmt.Errorf("未能获取屏幕截图")
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		return nil, fmt.Errorf("连接超时")
	}
}

// initProtocolStack 初始化 RDP 协议栈
func (s *ConnectorService) initProtocolStack(client *RDPClient, conn net.Conn, domain, user, pwd string) {
	// 创建协议层实例
	client.tpkt = tpkt.New(core.NewSocketLayer(conn), nla.NewNTLMv2(domain, user, pwd))
	client.x224 = x224.New(client.tpkt)
	client.mcs = t125.NewMCSClient(client.x224)
	client.sec = sec.NewClient(client.mcs)
	client.pdu = pdu.NewClient(client.sec)
	client.channels = plugin.NewChannels(client.sec)

	// 设置认证信息
	client.sec.SetUser(user)
	client.sec.SetPwd(pwd)
	client.sec.SetDomain(domain)

	// 配置协议层关联
	client.tpkt.SetFastPathListener(client.sec)
	client.sec.SetFastPathListener(client.pdu)
	client.sec.SetChannelSender(client.mcs)
	client.channels.SetChannelSender(client.sec)
	
	// 不设置特定协议，让服务器选择（更好的兼容性）
	// client.x224.SetRequestedProtocol(x224.PROTOCOL_SSL)
}

// setupEventHandlersWithCapture 设置 PDU 事件处理器（包括位图捕获）
func (s *ConnectorService) setupEventHandlersWithCapture(client *RDPClient, resultChan chan error, screenshotChan chan image.Image) {
	// 错误处理
	client.pdu.On("error", func(e error) {
		resultChan <- e
	})

	// 连接关闭
	client.pdu.On("close", func() {
		resultChan <- fmt.Errorf("连接已关闭")
	})

	// 连接成功
	client.pdu.On("success", func() {
		client.mu.Lock()
		client.connected = true
		client.mu.Unlock()
		
		// 发送同步事件以触发屏幕更新
		go func() {
			time.Sleep(500 * time.Millisecond)
			// 发送一个空的鼠标移动事件来触发屏幕更新
			p := &pdu.PointerEvent{}
			p.PointerFlags = pdu.PTRFLAGS_MOVE
			p.XPos = 0
			p.YPos = 0
			client.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p})
		}()
		
		resultChan <- nil // 成功时发送 nil 错误
	})

	// 连接就绪，也视为成功
	client.pdu.On("ready", func() {
		client.mu.Lock()
		client.connected = true
		client.mu.Unlock()
		
		// 发送同步事件以触发屏幕更新
		go func() {
			time.Sleep(500 * time.Millisecond)
			// 发送一个空的鼠标移动事件来触发屏幕更新
			p := &pdu.PointerEvent{}
			p.PointerFlags = pdu.PTRFLAGS_MOVE
			p.XPos = 0
			p.YPos = 0
			client.pdu.SendInputEvents(pdu.INPUT_EVENT_MOUSE, []pdu.InputEventsInterface{p})
		}()
		
		resultChan <- nil
	})

	// 位图更新事件
	client.pdu.On("bitmap", func(rectangles []pdu.BitmapData) {
		client.mu.Lock()
		defer client.mu.Unlock()
		
		if !client.connected {
			return
		}

		s.processBitmapUpdate(client, rectangles, screenshotChan)
	})
	
	// update 事件（实际的屏幕更新事件）
	client.pdu.On("update", func(rectangles []pdu.BitmapData) {
		client.mu.Lock()
		defer client.mu.Unlock()
		
		if !client.connected {
			return
		}

		s.processBitmapUpdate(client, rectangles, screenshotChan)
	})
}

// processBitmapUpdate 处理位图更新
func (s *ConnectorService) processBitmapUpdate(client *RDPClient, rectangles []pdu.BitmapData, screenshotChan chan image.Image) {
	// 转换位图数据
	bitmaps := make([]RDPBitmap, 0, len(rectangles))
	for _, rect := range rectangles {
		isCompress := rect.IsCompress()
		data := rect.BitmapDataStream
		
		// 如果位图是压缩的，需要解压缩
		if isCompress {
			data = s.bitmapDecompress(&rect)
		}
		
		// 临时调试：记录前几个矩形的坐标
		// if i < 3 {
		// 	log.Printf("[RDP DEBUG] 矩形%d: DestLeft=%d, DestTop=%d, DestRight=%d, DestBottom=%d, Width=%d, Height=%d",
		// 		i, rect.DestLeft, rect.DestTop, rect.DestRight, rect.DestBottom, rect.Width, rect.Height)
		// }
		
		bitmaps = append(bitmaps, RDPBitmap{
			DestLeft:     int(rect.DestLeft),
			DestTop:      int(rect.DestTop),
			DestRight:    int(rect.DestRight),
			DestBottom:   int(rect.DestBottom),
			Width:        int(rect.Width),
			Height:       int(rect.Height),
			BitsPerPixel: int(rect.BitsPerPixel),
			IsCompress:   false, // 已解压
			Data:         data,
		})
	}

	// 绘制位图到屏幕图像
	if len(bitmaps) > 0 {
		s.paintBitmap(client, bitmaps)
		
		// 发送截图（只发送一次）
		select {
		case screenshotChan <- client.screenImage:
		default:
			// 通道已满，不阻塞
		}
	}
}

// ExecuteRDPCommand 执行 RDP 命令
func (s *ConnectorService) ExecuteRDPCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("RDP 连接未建立")
	}

	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	s.AddLog(conn, fmt.Sprintf("开始执行 RDP 命令: %s", cmd))

	// 支持的命令
	switch strings.ToLower(cmd) {
	case "screenshot":
		return s.executeRDPScreenshot(conn)
	default:
		return "", fmt.Errorf("不支持的命令: %s (可用命令: screenshot)", cmd)
	}
}

// executeRDPScreenshot 执行 RDP 截图命令
func (s *ConnectorService) executeRDPScreenshot(conn *models.Connection) (string, error) {
	port := conn.Port
	if port == "" {
		port = "3389"
	}

	target := fmt.Sprintf("%s:%s", conn.IP, port)
	
	// 解析域名和用户名
	domain := ""
	username := conn.User
	password := conn.Pass

	if username == "" {
		username = "Administrator"
	}

	if strings.Contains(username, "\\") {
		parts := strings.Split(username, "\\")
		domain = parts[0]
		username = parts[1]
	} else if strings.Contains(username, "/") {
		parts := strings.Split(username, "/")
		domain = parts[0]
		username = parts[1]
	}

	if domain == "" {
		domain = conn.IP
	}

	s.AddLog(conn, "开始连接 RDP 服务进行截图...")
	s.AddLog(conn, fmt.Sprintf("目标地址: %s", target))
	s.AddLog(conn, fmt.Sprintf("用户名: %s", username))

	// 尝试 RDP 连接并获取截图
	screenshot, err := s.rdpConnectAndCapture(target, domain, username, password, 25)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("截图失败: %v", err))
		return "", fmt.Errorf("截图失败: %v", err)
	}

	if screenshot == nil {
		s.AddLog(conn, "未获取到屏幕数据")
		return "", fmt.Errorf("未获取到屏幕数据")
	}

	// 将截图转换为 Base64
	s.AddLog(conn, "正在编码截图...")
	screenshotBase64, err := s.rdpImageToBase64(screenshot)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("截图编码失败: %v", err))
		return "", fmt.Errorf("截图编码失败: %v", err)
	}

	s.AddLog(conn, fmt.Sprintf("截图编码成功，大小: %d 字节", len(screenshotBase64)))
	
	// 构建结果
	result := fmt.Sprintf("RDP 截图执行成功\n")
	result += strings.Repeat("=", 45) + "\n\n"
	result += fmt.Sprintf("目标地址: %s:%s\n", conn.IP, conn.Port)
	result += fmt.Sprintf("用户名: %s\n", username)
	if domain != conn.IP {
		result += fmt.Sprintf("域: %s\n", domain)
	}
	result += fmt.Sprintf("分辨率: %dx%d\n", 1280, 800)
	result += fmt.Sprintf("\n截图: [BASE64_IMAGE]%s[/BASE64_IMAGE]\n\n", screenshotBase64)
	
	return result, nil
}

// paintBitmap 绘制位图到屏幕图像
func (s *ConnectorService) paintBitmap(client *RDPClient, bitmaps []RDPBitmap) {
	for _, bm := range bitmaps {
		// 检查数据有效性
		if len(bm.Data) == 0 {
			continue
		}
		
		// 创建位图图像
		m := image.NewRGBA(image.Rect(0, 0, bm.Width, bm.Height))
		
		// 获取每像素字节数（解压后的数据格式）
		var bytesPerPixel int
		switch bm.BitsPerPixel {
		case 15:
			bytesPerPixel = 2
		case 16:
			bytesPerPixel = 2
		case 24:
			bytesPerPixel = 3
		case 32:
			bytesPerPixel = 4
		default:
			continue
		}

		// 填充像素数据
		i := 0
		for y := 0; y < bm.Height && i+bytesPerPixel <= len(bm.Data); y++ {
			for x := 0; x < bm.Width && i+bytesPerPixel <= len(bm.Data); x++ {
				var r, g, b, a uint8
				a = 255
				
				switch bytesPerPixel {
				case 2: // 15位或16位
					if bm.BitsPerPixel == 15 {
						// RGB555
						rgb555 := uint16(bm.Data[i]) | (uint16(bm.Data[i+1]) << 8)
						r, g, b = s.rgb555ToRGB(rgb555)
					} else {
						// RGB565
						rgb565 := uint16(bm.Data[i]) | (uint16(bm.Data[i+1]) << 8)
						r, g, b = s.rgb565ToRGB(rgb565)
					}
				case 3: // 24位 BGR
					b = bm.Data[i]
					g = bm.Data[i+1]
					r = bm.Data[i+2]
				case 4: // 32位 BGRA
					b = bm.Data[i]
					g = bm.Data[i+1]
					r = bm.Data[i+2]
					// 忽略alpha通道 (bm.Data[i+3])
				}
				
				m.Set(x, y, color.RGBA{r, g, b, a})
				i += bytesPerPixel
			}
		}

		// 使用 RDP 提供的绝对坐标（DestRight 和 DestBottom 是包含性的，需要 +1）
		destRect := image.Rect(bm.DestLeft, bm.DestTop, bm.DestRight+1, bm.DestBottom+1)
		
		// 确保目标矩形在屏幕范围内
		screenBounds := client.screenImage.Bounds()
		destRect = destRect.Intersect(screenBounds)
		
		if !destRect.Empty() {
			// 绘制到屏幕图像
			draw.Draw(
				client.screenImage,
				destRect,
				m,
				image.Point{0, 0},
				draw.Src,
			)
		}
	}
}

// bitmapDecompress 解压缩位图数据
func (s *ConnectorService) bitmapDecompress(bitmap *pdu.BitmapData) []byte {
	return core.Decompress(bitmap.BitmapDataStream, int(bitmap.Width), int(bitmap.Height), s.bppBytes(bitmap.BitsPerPixel))
}

// bppBytes 获取每像素字节数（用于解压缩）
func (s *ConnectorService) bppBytes(bitsPerPixel uint16) int {
	switch bitsPerPixel {
	case 15:
		return 1
	case 16:
		return 2
	case 24:
		return 3
	case 32:
		return 4
	default:
		return 0
	}
}

// bpp 获取每像素字节数
func (s *ConnectorService) bpp(bitsPerPixel int) int {
	switch bitsPerPixel {
	case 15:
		return 2
	case 16:
		return 2
	case 24:
		return 3
	case 32:
		return 4
	default:
		return 0
	}
}

// rgb555ToRGB 将 RGB555 转换为 RGB
func (s *ConnectorService) rgb555ToRGB(rgb555 uint16) (r, g, b uint8) {
	r = uint8((rgb555 >> 10) & 0x1F)
	g = uint8((rgb555 >> 5) & 0x1F)
	b = uint8(rgb555 & 0x1F)
	// 扩展到 8 位
	r = (r << 3) | (r >> 2)
	g = (g << 3) | (g >> 2)
	b = (b << 3) | (b >> 2)
	return
}

// rgb565ToRGB 将 RGB565 转换为 RGB
func (s *ConnectorService) rgb565ToRGB(rgb565 uint16) (r, g, b uint8) {
	r = uint8((rgb565 >> 11) & 0x1F)
	g = uint8((rgb565 >> 5) & 0x3F)
	b = uint8(rgb565 & 0x1F)
	// 扩展到 8 位
	r = (r << 3) | (r >> 2)
	g = (g << 2) | (g >> 4)
	b = (b << 3) | (b >> 2)
	return
}

// rdpImageToBase64 将图片转换为 Base64 字符串
func (s *ConnectorService) rdpImageToBase64(img image.Image) (string, error) {
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
