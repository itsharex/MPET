package connectors

import (
	"MPET/backend/models"
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

// ConnectOracle 连接 Oracle 数据库（使用纯 Go 实现的 go-ora 驱动，无需 Oracle Instant Client）
func (s *ConnectorService) ConnectOracle(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "1521" // Oracle 默认端口
	}
	addr := net.JoinHostPort(conn.IP, port)
	s.AddLog(conn, fmt.Sprintf("目标 Oracle 数据库地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: Oracle 连接暂不支持代理，将尝试直接连接")
	}

	portInt := 1521
	if port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			portInt = p
		}
	}

	// 常见的 Oracle 服务名列表
	serviceNames := []string{"XE", "ORCL", "XEPDB1", "ORCLPDB", "ORCLCDB", "PDBORCL"}

	var username, password string
	// 确定要尝试的用户名和密码
	if conn.User != "" && conn.Pass != "" {
		username = conn.User
		password = conn.Pass
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", username))
	} else if conn.User != "" {
		username = conn.User
		password = ""
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码连接", username))
	} else {
		// 尝试常见的默认用户组合
		username = "sys"
		password = "system"
		s.AddLog(conn, "尝试默认用户 sys/system 连接")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var db *sql.DB
	var err error
	var successServiceName string
	var successUsername string
	var successPassword string

	// 尝试不同的服务名
	for _, serviceName := range serviceNames {
		s.AddLog(conn, fmt.Sprintf("尝试服务名: %s", serviceName))

		var dsn string
		if password != "" {
			dsn = go_ora.BuildUrl(conn.IP, portInt, serviceName, username, password, nil)
		} else {
			dsn = go_ora.BuildUrl(conn.IP, portInt, serviceName, username, "", nil)
		}

		db, err = sql.Open("oracle", dsn)
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("  创建连接失败: %v", err))
			continue
		}

		// 测试连接
		err = db.PingContext(ctx)
		if err != nil {
			errMsg := err.Error()
			// 如果是服务名错误，尝试下一个服务名
			if strings.Contains(errMsg, "ORA-12514") || strings.Contains(errMsg, "TNS:listener does not currently know of service") {
				s.AddLog(conn, fmt.Sprintf("  服务名 %s 不存在，尝试下一个", serviceName))
				db.Close()
				continue
			}
			// 如果是认证失败，尝试下一个服务名（可能是服务名不对）
			if strings.Contains(errMsg, "ORA-01017") || strings.Contains(errMsg, "invalid username/password") {
				s.AddLog(conn, fmt.Sprintf("  认证失败，服务名 %s 可能不正确，尝试下一个", serviceName))
				db.Close()
				continue
			}
			// 其他错误，也尝试下一个服务名
			s.AddLog(conn, fmt.Sprintf("  连接失败: %v，尝试下一个服务名", err))
			db.Close()
			continue
		}

		// 连接成功
		successServiceName = serviceName
		successUsername = username
		successPassword = password
		s.AddLog(conn, fmt.Sprintf("✓ 使用服务名 %s 连接成功", serviceName))
		err = nil // 标记连接成功
		break
	}

	// 如果所有服务名都失败，且使用的是默认用户，尝试其他用户组合
	if err != nil && successServiceName == "" && username == "sys" && password == "system" {
		s.AddLog(conn, "尝试默认用户 scott/tiger 连接")
		username = "scott"
		password = "tiger"

		for _, serviceName := range serviceNames {
			s.AddLog(conn, fmt.Sprintf("尝试服务名: %s (用户: scott/tiger)", serviceName))
			dsn := go_ora.BuildUrl(conn.IP, portInt, serviceName, username, password, nil)

			db, err = sql.Open("oracle", dsn)
			if err != nil {
				continue
			}

			err = db.PingContext(ctx)
			if err != nil {
				errMsg := err.Error()
				if strings.Contains(errMsg, "ORA-12514") || strings.Contains(errMsg, "TNS:listener does not currently know of service") {
					db.Close()
					continue
				}
				if strings.Contains(errMsg, "ORA-01017") || strings.Contains(errMsg, "invalid username/password") {
					db.Close()
					continue
				}
				db.Close()
				continue
			}

			// 连接成功
			successServiceName = serviceName
			successUsername = username
			successPassword = password
			s.AddLog(conn, fmt.Sprintf("✓ 使用服务名 %s 和用户 scott/tiger 连接成功", serviceName))
			err = nil // 标记连接成功
			break
		}
	}

	// 如果所有尝试都失败
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("✗ Oracle 连接失败: %v", err))
		s.AddLog(conn, "提示: 已尝试常见服务名 (XE, ORCL, XEPDB1, ORCLPDB, ORCLCDB, PDBORCL)")
		s.AddLog(conn, "如果您的数据库使用其他服务名，请检查 Oracle 监听器配置")
		if db != nil {
			db.Close()
		}
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		return
	}

	// 更新用户名和密码变量
	username = successUsername
	password = successPassword

	s.AddLog(conn, fmt.Sprintf("✓ Oracle 连接成功 (服务名: %s, 用户: %s)", successServiceName, username))
	s.AddLog(conn, "执行查询: SELECT name FROM v$database")

	// 获取数据库信息
	result := s.GetOracleDatabases(db, ctx)
	conn.Status = "success"
	if username != "" {
		if password != "" {
			conn.Message = fmt.Sprintf("连接成功（用户: %s）", username)
		} else {
			conn.Message = fmt.Sprintf("连接成功（用户: %s，无密码）", username)
		}
	} else {
		conn.Message = "连接成功"
	}
	s.SetConnectionResult(conn, result)
	conn.ConnectedAt = time.Now()
	db.Close()
}

// GetOracleDatabases 获取 Oracle 数据库信息
func (s *ConnectorService) GetOracleDatabases(db *sql.DB, ctx context.Context) string {
	var results []string
	results = append(results, "数据库信息:")
	results = append(results, strings.Repeat("-", 50))

	// 执行查询: SELECT name FROM v$database
	query := "SELECT name FROM v$database"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		results = append(results, fmt.Sprintf("查询失败: %v", err))
		// 尝试其他查询获取数据库信息
		results = append(results, "")
		results = append(results, "尝试获取其他数据库信息...")

		// 尝试查询实例信息
		altQuery := "SELECT instance_name, host_name, version FROM v$instance"
		altRows, altErr := db.QueryContext(ctx, altQuery)
		if altErr == nil {
			defer altRows.Close()
			results = append(results, "实例信息:")
			for altRows.Next() {
				var instanceName, hostName, version string
				if err := altRows.Scan(&instanceName, &hostName, &version); err == nil {
					results = append(results, fmt.Sprintf("  实例名: %s", instanceName))
					results = append(results, fmt.Sprintf("  主机名: %s", hostName))
					results = append(results, fmt.Sprintf("  版本: %s", version))
				}
			}
		}
		return strings.Join(results, "\n")
	}
	defer rows.Close()

	results = append(results, "数据库名称:")
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			results = append(results, fmt.Sprintf("读取行失败: %v", err))
			continue
		}
		results = append(results, fmt.Sprintf("  - %s", name))
	}

	if err := rows.Err(); err != nil {
		results = append(results, fmt.Sprintf("遍历行时出错: %v", err))
	}

	// 尝试获取更多信息
	results = append(results, "")
	results = append(results, "实例信息:")
	instanceQuery := "SELECT instance_name, host_name, version FROM v$instance"
	instanceRows, err := db.QueryContext(ctx, instanceQuery)
	if err == nil {
		defer instanceRows.Close()
		for instanceRows.Next() {
			var instanceName, hostName, version string
			if err := instanceRows.Scan(&instanceName, &hostName, &version); err == nil {
				results = append(results, fmt.Sprintf("  实例名: %s", instanceName))
				results = append(results, fmt.Sprintf("  主机名: %s", hostName))
				results = append(results, fmt.Sprintf("  版本: %s", version))
			}
		}
	}

	return strings.Join(results, "\n")
}

// ExecuteOracleCommand 执行 Oracle 命令
func (s *ConnectorService) ExecuteOracleCommand(conn *models.Connection, command string) (string, error) {
	// 注意：这里使用 godror 驱动，需要确保已安装
	dsn := fmt.Sprintf("%s/%s@%s:%s/XE", conn.User, conn.Pass, conn.IP, conn.Port)
	
	db, err := sql.Open("godror", dsn)
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
			if val != nil {
				row[i] = fmt.Sprintf("%v", val)
			} else {
				row[i] = "NULL"
			}
		}
		results = append(results, strings.Join(row, " | "))
	}

	return strings.Join(results, "\n"), nil
}
