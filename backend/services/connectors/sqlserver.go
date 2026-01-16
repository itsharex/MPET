package connectors

import (
	"MPET/backend/models"
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

// ConnectSQLServer 连接 SQL Server
func (s *ConnectorService) ConnectSQLServer(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "1433"
	}
	server := net.JoinHostPort(conn.IP, port)
	s.AddLog(conn, fmt.Sprintf("目标 SQL Server 地址: %s", server))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: SQL Server 连接暂不支持代理，将尝试直接连接")
	}

	type attempt struct {
		user  string
		pass  string
		label string
	}

	var attempts []attempt
	seen := make(map[string]struct{})
	addAttempt := func(user, pass, label string) {
		key := fmt.Sprintf("%s|%s", user, pass)
		if _, exists := seen[key]; exists {
			return
		}
		attempts = append(attempts, attempt{
			user:  user,
			pass:  pass,
			label: label,
		})
		seen[key] = struct{}{}
	}

	// 优先尝试用户提供的凭据
	if conn.User != "" || conn.Pass != "" {
		addAttempt(conn.User, conn.Pass, "用户提供的凭据")
		// 如果用户只提供了用户名，补充一次无密码尝试
		if conn.Pass == "" && conn.User != "" {
			addAttempt(conn.User, "", "用户提供的用户名（无密码）")
		}
	}

	// 追加常见的弱口令/默认凭据
	// defaultAttempts := []attempt{
	// 	{user: "sa", pass: "", label: "默认用户 sa 无密码"},
	// 	{user: "sa", pass: "sa", label: "常见弱口令 sa/sa"},
	// 	{user: "sa", pass: "123456", label: "常见弱口令 sa/123456"},
	// 	{user: "sa", pass: "P@ssw0rd", label: "常见弱口令 sa/P@ssw0rd"},
	// 	{user: "sa", pass: "Password123", label: "常见弱口令 sa/Password123"},
	// 	{user: "", pass: "", label: "无凭据"},
	// }
	// for _, att := range defaultAttempts {
	// 	addAttempt(att.user, att.pass, att.label)
	// }

	// 兜底，至少要有一个尝试
	if len(attempts) == 0 {
		addAttempt("sa", "", "默认用户 sa 无密码")
		addAttempt("", "", "无凭据")
	}

	var lastErr error
	for _, att := range attempts {
		if att.user != "" {
			if att.pass == "" {
				s.AddLog(conn, fmt.Sprintf("尝试 SQL Server 用户 %s 无密码连接（%s）", att.user, att.label))
			} else {
				s.AddLog(conn, fmt.Sprintf("尝试 SQL Server 用户 %s 密码认证（%s）", att.user, att.label))
			}
		} else {
			s.AddLog(conn, fmt.Sprintf("尝试 SQL Server 无凭据连接（%s）", att.label))
		}

		dsn := s.BuildSQLServerDSN(server, att.user, att.pass)
		db, err := sql.Open("sqlserver", dsn)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ 创建 SQL Server 连接失败: %v", err))
			lastErr = err
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ SQL Server 认证失败: %v", err))
			lastErr = err
			db.Close()
			continue
		}

		s.AddLog(conn, "✓ SQL Server 连接成功")
		s.AddLog(conn, "执行查询: SELECT name AS DatabaseName FROM sys.databases")

		result := s.GetSQLServerDatabases(db)
		conn.Status = "success"
		if att.user != "" {
			if att.pass == "" {
				conn.Message = fmt.Sprintf("连接成功（SQL Server 用户 %s 无密码）", att.user)
			} else {
				conn.Message = fmt.Sprintf("连接成功（SQL Server 用户 %s）", att.user)
			}
		} else {
			conn.Message = "连接成功（SQL Server 无凭据）"
		}
		s.SetConnectionResult(conn, result)
		conn.ConnectedAt = time.Now()
		db.Close()
		return
	}

	conn.Status = "failed"
	failMsg := "连接失败: 所有 SQL Server 尝试均失败"
	if lastErr != nil {
		failMsg = fmt.Sprintf("%s（最后错误: %v）", failMsg, lastErr)
	}
	conn.Message = failMsg
	s.AddLog(conn, "所有 SQL Server 连接尝试均失败")
	if lastErr != nil {
		s.AddLog(conn, fmt.Sprintf("最后错误: %v", lastErr))
	}
}

// BuildSQLServerDSN 构建 SQL Server DSN
func (s *ConnectorService) BuildSQLServerDSN(server, user, pass string) string {
	const commonParams = "?encrypt=disable"
	if user == "" {
		return fmt.Sprintf("sqlserver://%s%s", server, commonParams)
	}

	escapedUser := url.QueryEscape(user)
	if pass == "" {
		return fmt.Sprintf("sqlserver://%s@%s%s", escapedUser, server, commonParams)
	}
	escapedPass := url.QueryEscape(pass)
	return fmt.Sprintf("sqlserver://%s:%s@%s%s", escapedUser, escapedPass, server, commonParams)
}

// GetSQLServerDatabases 获取 SQL Server 数据库列表
func (s *ConnectorService) GetSQLServerDatabases(db *sql.DB) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT name AS DatabaseName FROM sys.databases")
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	defer rows.Close()

	var results []string
	results = append(results, "数据库列表:")
	results = append(results, strings.Repeat("-", 40))

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			results = append(results, fmt.Sprintf("读取行失败: %v", err))
			continue
		}
		results = append(results, fmt.Sprintf("- %s", name))
	}

	if err := rows.Err(); err != nil {
		results = append(results, fmt.Sprintf("遍历行时出错: %v", err))
	}

	if len(results) == 2 {
		results = append(results, "未获取到任何数据库")
	}

	return strings.Join(results, "\n")
}

// ExecuteSQLServerCommand 执行 SQL Server 命令
func (s *ConnectorService) ExecuteSQLServerCommand(conn *models.Connection, command string) (string, error) {
	server := net.JoinHostPort(conn.IP, conn.Port)
	dsn := s.BuildSQLServerDSN(server, conn.User, conn.Pass)

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, command)
	if err != nil {
		return "", fmt.Errorf("命令执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("获取列信息失败: %v", err)
	}

	var results []string
	results = append(results, strings.Join(columns, " | "))
	results = append(results, strings.Repeat("-", len(columns)*20))

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		
		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				// 处理字节数组类型
				switch v := val.(type) {
				case []byte:
					row[i] = string(v)
				default:
					row[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		results = append(results, strings.Join(row, " | "))
	}

	return strings.Join(results, "\n"), nil
}
