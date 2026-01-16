package connectors

import (
	"MPET/backend/models"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// ConnectPostgreSQL 连接 PostgreSQL
func (s *ConnectorService) ConnectPostgreSQL(conn *models.Connection) {
	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: PostgreSQL 连接暂不支持代理，将尝试直接连接")
	}

	// 如果用户提供了用户名和密码，直接使用，跳过默认用户连接
	if conn.User != "" && conn.Pass != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable connect_timeout=5",
			conn.IP, conn.Port, conn.User, conn.Pass)
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				s.AddLog(conn, "✓ 密码认证成功")
				s.AddLog(conn, "执行查询: SELECT * FROM pg_database")
				result := s.GetPostgreSQLDatabases(db)
				conn.Status = "success"
				conn.Message = "连接成功（使用用户名密码）"
				s.SetConnectionResult(conn, result)
				conn.ConnectedAt = time.Now()
				db.Close()
				return
			}
			s.AddLog(conn, fmt.Sprintf("✗ 用户认证失败: %v", err))
			db.Close()
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 数据库连接失败: %v", err))
		}
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, "密码认证失败")
		return
	}

	// 尝试未授权访问（使用默认用户 postgres，无密码）
	s.AddLog(conn, "尝试默认用户 postgres 无密码连接")
	dsn := fmt.Sprintf("host=%s port=%s user=postgres password= dbname=postgres sslmode=disable connect_timeout=5",
		conn.IP, conn.Port)
	db, err := sql.Open("postgres", dsn)
	if err == nil {
		err = db.Ping()
		if err == nil {
			s.AddLog(conn, "✓ 默认用户 postgres 无密码连接成功")
			s.AddLog(conn, "执行查询: SELECT * FROM pg_database")
			result := s.GetPostgreSQLDatabases(db)
			conn.Status = "success"
			conn.Message = "连接成功（未授权访问，默认用户 postgres）"
			s.SetConnectionResult(conn, result)
			conn.ConnectedAt = time.Now()
			db.Close()
			return
		}
		s.AddLog(conn, fmt.Sprintf("✗ 默认用户连接失败: %v", err))
		db.Close()
	} else {
		s.AddLog(conn, fmt.Sprintf("✗ 数据库连接失败: %v", err))
	}

	// 尝试使用提供的用户名（无密码）
	if conn.User != "" {
		password := conn.Pass
		if password == "" {
			s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码连接", conn.User))
		} else {
			s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable connect_timeout=5",
			conn.IP, conn.Port, conn.User, password)
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				if password == "" {
					s.AddLog(conn, "✓ 无密码连接成功")
					conn.Status = "success"
					conn.Message = "连接成功（无密码）"
				} else {
					s.AddLog(conn, "✓ 密码认证成功")
					conn.Status = "success"
					conn.Message = "连接成功（使用用户名密码）"
				}
				s.AddLog(conn, "执行查询: SELECT * FROM pg_database")
				result := s.GetPostgreSQLDatabases(db)
				s.SetConnectionResult(conn, result)
				conn.ConnectedAt = time.Now()
				db.Close()
				return
			}
			s.AddLog(conn, fmt.Sprintf("✗ 用户认证失败: %v", err))
			db.Close()
		}
	}

	conn.Status = "failed"
	conn.Message = fmt.Sprintf("连接失败: %v", err)
	s.AddLog(conn, "所有连接尝试均失败")
}

// GetPostgreSQLDatabases 获取 PostgreSQL 数据库列表
func (s *ConnectorService) GetPostgreSQLDatabases(db *sql.DB) string {
	query := "SELECT datname, pg_size_pretty(pg_database_size(datname)) as size, datcollate, datctype FROM pg_database ORDER BY datname"
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	defer rows.Close()

	var results []string
	results = append(results, "数据库列表:")
	results = append(results, fmt.Sprintf("%-20s %-15s %-15s %-15s", "数据库名", "大小", "排序规则", "字符集"))
	results = append(results, strings.Repeat("-", 65))

	for rows.Next() {
		var datname, size, datcollate, datctype string
		if err := rows.Scan(&datname, &size, &datcollate, &datctype); err != nil {
			results = append(results, fmt.Sprintf("读取行失败: %v", err))
			continue
		}
		results = append(results, fmt.Sprintf("%-20s %-15s %-15s %-15s", datname, size, datcollate, datctype))
	}

	if err := rows.Err(); err != nil {
		results = append(results, fmt.Sprintf("遍历行时出错: %v", err))
	}

	return strings.Join(results, "\n")
}

// ExecutePostgreSQLCommand 执行 PostgreSQL 命令
func (s *ConnectorService) ExecutePostgreSQLCommand(conn *models.Connection, command string) (string, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable",
		conn.IP, conn.Port, conn.User, conn.Pass)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(command)
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
