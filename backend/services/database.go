package services

import (
	"MPET/backend/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const dbFileName = "connections.db"

// initDatabase 初始化数据库
func initDatabase() (*sql.DB, error) {
	dbPath := getDBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Printf("数据库文件不存在，将创建新数据库: %s", dbPath)
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			log.Printf("创建数据库目录失败: %v", err)
		}
	}

	dsnPath := strings.ReplaceAll(dbPath, "\\", "/")
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", dsnPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("创建表失败: %v", err)
	}

	log.Printf("数据库初始化成功: %s", dbPath)
	return db, nil
}

func createTables(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS connections (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		ip TEXT NOT NULL,
		port TEXT NOT NULL,
		user TEXT,
		pass TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		message TEXT,
		result TEXT,
		logs TEXT,
		created_at TEXT NOT NULL,
		connected_at TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_type ON connections(type);
	CREATE INDEX IF NOT EXISTS idx_status ON connections(status);
	CREATE INDEX IF NOT EXISTS idx_created_at ON connections(created_at);

	CREATE TABLE IF NOT EXISTS vulnerabilities (
		id TEXT PRIMARY KEY,
		service_type TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		level TEXT NOT NULL,
		description TEXT NOT NULL,
		repair TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_service_type ON vulnerabilities(service_type);
	`

	_, err := db.Exec(createTableSQL)
	return err
}

func connectionFromRow(row *sql.Row) (*models.Connection, error) {
	var conn models.Connection
	var logsJSON string
	var createdAtStr, connectedAtStr string

	err := row.Scan(
		&conn.ID, &conn.Type, &conn.IP, &conn.Port,
		&conn.User, &conn.Pass, &conn.Status, &conn.Message,
		&conn.Result, &logsJSON, &createdAtStr, &connectedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if logsJSON != "" {
		json.Unmarshal([]byte(logsJSON), &conn.Logs)
	}
	if conn.Logs == nil {
		conn.Logs = []string{}
	}

	if createdAtStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			conn.CreatedAt = t
		}
	}
	if connectedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, connectedAtStr); err == nil {
			conn.ConnectedAt = t
		}
	}

	return &conn, nil
}

func connectionFromRows(rows *sql.Rows) (*models.Connection, error) {
	var conn models.Connection
	var logsJSON string
	var createdAtStr, connectedAtStr string

	err := rows.Scan(
		&conn.ID, &conn.Type, &conn.IP, &conn.Port,
		&conn.User, &conn.Pass, &conn.Status, &conn.Message,
		&conn.Result, &logsJSON, &createdAtStr, &connectedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if logsJSON != "" {
		json.Unmarshal([]byte(logsJSON), &conn.Logs)
	}
	if conn.Logs == nil {
		conn.Logs = []string{}
	}

	if createdAtStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			conn.CreatedAt = t
		}
	}
	if connectedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, connectedAtStr); err == nil {
			conn.ConnectedAt = t
		}
	}

	return &conn, nil
}

func connectionToValues(conn *models.Connection) ([]interface{}, error) {
	logsJSON := "[]"
	if conn.Logs != nil && len(conn.Logs) > 0 {
		jsonData, _ := json.Marshal(conn.Logs)
		logsJSON = string(jsonData)
	}

	createdAtStr := conn.CreatedAt.Format(time.RFC3339)
	connectedAtStr := ""
	if !conn.ConnectedAt.IsZero() {
		connectedAtStr = conn.ConnectedAt.Format(time.RFC3339)
	}

	return []interface{}{
		conn.ID, conn.Type, conn.IP, conn.Port,
		conn.User, conn.Pass, conn.Status, conn.Message,
		conn.Result, logsJSON, createdAtStr, connectedAtStr,
	}, nil
}

func getDBPath() string {
	exePath, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exePath), dbFileName)
	}
	return dbFileName
}
