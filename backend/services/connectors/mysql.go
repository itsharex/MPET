package connectors

import (
	"MPET/backend/models"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectMySQL 连接 MySQL
func (s *ConnectorService) ConnectMySQL(conn *models.Connection) {
	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: MySQL 连接暂不支持代理，将尝试直接连接")
	}

	// 如果提供了密码，直接使用密码认证
	if conn.Pass != "" && conn.User != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql?timeout=5s",
			conn.User, conn.Pass, conn.IP, conn.Port)
		db, err := sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				s.AddLog(conn, "✓ 密码认证成功")
				s.AddLog(conn, "执行查询: SHOW DATABASES")
				result := s.GetMySQLDatabases(db)
				conn.Status = "success"
				conn.Message = "连接成功（使用用户名密码）"
				s.SetConnectionResult(conn, result)
				conn.ConnectedAt = time.Now()
				db.Close()
				return
			}
			s.AddLog(conn, fmt.Sprintf("✗ 密码认证失败: %v", err))
			db.Close()
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 数据库连接失败: %v", err))
		}
		// 密码认证失败，不再尝试其他方式
		conn.Status = "failed"
		conn.Message = "连接失败: 密码认证失败"
		s.AddLog(conn, "密码认证失败，不再尝试无密码连接")
		return
	}

	// 如果没有提供密码，尝试未授权访问（root 无密码）
	s.AddLog(conn, "尝试 root 用户无密码连接")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql?timeout=5s",
		"root", "", conn.IP, conn.Port)
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		err = db.Ping()
		if err == nil {
			s.AddLog(conn, "✓ root 用户无密码连接成功")
			s.AddLog(conn, "执行查询: SHOW DATABASES")
			result := s.GetMySQLDatabases(db)
			conn.Status = "success"
			conn.Message = "连接成功（未授权访问，root 无密码）"
			s.SetConnectionResult(conn, result)
			conn.ConnectedAt = time.Now()
			db.Close()
			return
		}
		s.AddLog(conn, fmt.Sprintf("✗ root 用户连接失败: %v", err))
		db.Close()
	} else {
		s.AddLog(conn, fmt.Sprintf("✗ 数据库连接失败: %v", err))
	}

	// 尝试使用提供的用户名（无密码）
	if conn.User != "" && conn.Pass == "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码连接", conn.User))
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/mysql?timeout=5s",
			conn.User, "", conn.IP, conn.Port)
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				s.AddLog(conn, "✓ 无密码连接成功")
				s.AddLog(conn, "执行查询: SHOW DATABASES")
				result := s.GetMySQLDatabases(db)
				conn.Status = "success"
				conn.Message = "连接成功（无密码）"
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

// GetMySQLDatabases 获取 MySQL 数据库列表
func (s *ConnectorService) GetMySQLDatabases(db *sql.DB) string {
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err)
	}
	defer rows.Close()

	var results []string
	results = append(results, "数据库列表:")
	results = append(results, "数据库名")
	results = append(results, strings.Repeat("-", 30))

	for rows.Next() {
		var database string
		if err := rows.Scan(&database); err != nil {
			results = append(results, fmt.Sprintf("读取行失败: %v", err))
			continue
		}
		results = append(results, database)
	}

	if err := rows.Err(); err != nil {
		results = append(results, fmt.Sprintf("遍历行时出错: %v", err))
	}

	return strings.Join(results, "\n")
}

// ExecuteMySQLCommand 执行 MySQL 命令
func (s *ConnectorService) ExecuteMySQLCommand(conn *models.Connection, command string) (string, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", conn.User, conn.Pass, conn.IP, conn.Port)
	
	db, err := sql.Open("mysql", dsn)
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
