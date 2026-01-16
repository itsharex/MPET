package connectors

import (
	"MPET/backend/models"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectMongoDB 连接 MongoDB
func (s *ConnectorService) ConnectMongoDB(conn *models.Connection) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: MongoDB 连接暂不支持代理，将尝试直接连接")
	}

	var client *mongo.Client
	var err error
	var connected bool
	var username, password string

	// 如果用户提供了用户名和密码，直接使用，跳过未授权访问
	if conn.User != "" && conn.Pass != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		mongoURL := fmt.Sprintf("mongodb://%s:%s@%s:%s", conn.User, conn.Pass, conn.IP, conn.Port)
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
		if err == nil {
			err = client.Ping(ctx, nil)
			if err == nil {
				s.AddLog(conn, "✓ 密码认证成功")
				username = conn.User
				password = conn.Pass
				connected = true
			} else {
				s.AddLog(conn, fmt.Sprintf("✗ Ping 失败: %v", err))
				client.Disconnect(ctx)
			}
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 连接失败: %v", err))
		}
		if !connected {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("连接失败: %v", err)
			s.AddLog(conn, "密码认证失败")
			return
		}
	} else {
		// 尝试未授权访问
		s.AddLog(conn, "尝试未授权访问（无认证）")
		mongoURL := fmt.Sprintf("mongodb://%s:%s", conn.IP, conn.Port)
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
		if err == nil {
			err = client.Ping(ctx, nil)
			if err == nil {
				s.AddLog(conn, "✓ 未授权访问成功")
				connected = true
			} else {
				s.AddLog(conn, fmt.Sprintf("✗ Ping 失败: %v", err))
				client.Disconnect(ctx)
			}
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 连接失败: %v", err))
		}

		// 尝试使用用户名（无密码）
		if !connected && conn.User != "" {
			pass := conn.Pass
			if pass == "" {
				s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码连接", conn.User))
				mongoURL = fmt.Sprintf("mongodb://%s@%s:%s", conn.User, conn.IP, conn.Port)
			} else {
				s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
				mongoURL = fmt.Sprintf("mongodb://%s:%s@%s:%s", conn.User, conn.Pass, conn.IP, conn.Port)
			}
			client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
			if err == nil {
				err = client.Ping(ctx, nil)
				if err == nil {
					if pass == "" {
						s.AddLog(conn, "✓ 无密码连接成功")
					} else {
						s.AddLog(conn, "✓ 密码认证成功")
					}
					username = conn.User
					password = pass
					connected = true
				} else {
					s.AddLog(conn, fmt.Sprintf("✗ Ping 失败: %v", err))
					client.Disconnect(ctx)
				}
			} else {
				s.AddLog(conn, fmt.Sprintf("✗ 连接失败: %v", err))
			}
		}
	}

	if !connected {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, "所有连接尝试均失败")
		return
	}

	// 连接成功，执行 show dbs
	s.AddLog(conn, "执行 show dbs")
	result := s.GetMongoDBDatabases(client, ctx)
	conn.Status = "success"
	if username != "" && password != "" {
		conn.Message = "连接成功（使用用户名密码）"
	} else if username != "" {
		conn.Message = "连接成功（无密码）"
	} else {
		conn.Message = "连接成功（未授权访问）"
	}
	s.SetConnectionResult(conn, result)
	conn.ConnectedAt = time.Now()
	client.Disconnect(ctx)
}

// GetMongoDBDatabases 获取 MongoDB 数据库列表（相当于 show dbs）
func (s *ConnectorService) GetMongoDBDatabases(client *mongo.Client, ctx context.Context) string {
	var results []string
	results = append(results, "数据库列表:")
	results = append(results, strings.Repeat("-", 50))

	// 使用 ListDatabaseNames 获取数据库列表
	databases, err := client.ListDatabaseNames(ctx, nil)
	if err != nil {
		results = append(results, fmt.Sprintf("获取数据库列表失败: %v", err))
		return strings.Join(results, "\n")
	}

	if len(databases) == 0 {
		results = append(results, "当前没有数据库")
	} else {
		results = append(results, fmt.Sprintf("共找到 %d 个数据库:", len(databases)))
		results = append(results, "")

		// 获取每个数据库的详细信息
		for _, dbName := range databases {
			// 跳过系统数据库（可选，根据需求决定是否显示）
			if dbName == "admin" || dbName == "local" || dbName == "config" {
				results = append(results, fmt.Sprintf("  %s (系统数据库)", dbName))
			} else {
				results = append(results, fmt.Sprintf("  %s", dbName))
			}

			// 尝试获取数据库统计信息
			db := client.Database(dbName)
			stats := db.RunCommand(ctx, map[string]interface{}{"dbStats": 1})
			if stats.Err() == nil {
				var statsResult map[string]interface{}
				if err := stats.Decode(&statsResult); err == nil {
					if size, ok := statsResult["dataSize"].(float64); ok {
						results = append(results, fmt.Sprintf("    数据大小: %.2f MB", size/1024/1024))
					}
					if collections, ok := statsResult["collections"].(float64); ok {
						results = append(results, fmt.Sprintf("    集合数: %.0f", collections))
					}
				}
			}
			results = append(results, "")
		}
	}

	return strings.Join(results, "\n")
}

// ExecuteMongoDBCommand 执行 MongoDB 命令
func (s *ConnectorService) ExecuteMongoDBCommand(conn *models.Connection, command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/", conn.User, conn.Pass, conn.IP, conn.Port)
	if conn.User == "" {
		uri = fmt.Sprintf("mongodb://%s:%s/", conn.IP, conn.Port)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer client.Disconnect(ctx)

	// MongoDB 命令执行（简化版本，仅支持基本命令）
	result := client.Database("admin").RunCommand(ctx, bson.D{{Key: "eval", Value: command}})
	
	var cmdResult bson.M
	if err := result.Decode(&cmdResult); err != nil {
		return "", fmt.Errorf("命令执行失败: %v", err)
	}

	jsonData, _ := json.MarshalIndent(cmdResult, "", "  ")
	return string(jsonData), nil
}
