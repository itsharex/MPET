package connectors

import (
	"MPET/backend/models"
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

// ConnectKafka 连接 Kafka
func (s *ConnectorService) ConnectKafka(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "9092"
	}

	brokers := []string{net.JoinHostPort(conn.IP, port)}
	s.AddLog(conn, fmt.Sprintf("目标 Broker: %s", brokers[0]))
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("⚠ 注意: Kafka 客户端暂不支持 SOCKS5 代理"))
	}

	// 创建 Kafka 配置
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0 // 使用较新的 Kafka 版本
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second
	
	// 如果提供了认证信息
	if conn.User != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = conn.User
		config.Net.SASL.Password = conn.Pass
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		s.AddLog(conn, fmt.Sprintf("使用 SASL 认证: %s", conn.User))
	}

	// 创建客户端
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}
	defer client.Close()

	s.AddLog(conn, "✓ Kafka 客户端创建成功")

	// 获取 Broker 列表
	brokerList := client.Brokers()
	s.AddLog(conn, fmt.Sprintf("✓ 发现 %d 个 Broker", len(brokerList)))

	// 获取 Topic 列表
	topics, err := client.Topics()
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("获取 Topic 列表失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	s.AddLog(conn, fmt.Sprintf("✓ 发现 %d 个 Topic", len(topics)))

	// 构建结果信息
	var result strings.Builder
	result.WriteString("=== Kafka 集群信息 ===\n")
	result.WriteString(fmt.Sprintf("Broker 数量: %d\n", len(brokerList)))
	result.WriteString(fmt.Sprintf("Topic 数量: %d\n\n", len(topics)))

	// 列出 Broker
	result.WriteString("Brokers:\n")
	for _, broker := range brokerList {
		result.WriteString(fmt.Sprintf("  - %s (ID: %d)\n", broker.Addr(), broker.ID()))
	}

	// 列出前 20 个 Topic
	result.WriteString("\nTopics (前 20 个):\n")
	maxTopics := 20
	if len(topics) < maxTopics {
		maxTopics = len(topics)
	}
	for i := 0; i < maxTopics; i++ {
		result.WriteString(fmt.Sprintf("  - %s\n", topics[i]))
	}
	if len(topics) > 20 {
		result.WriteString(fmt.Sprintf("  ... 还有 %d 个 Topic\n", len(topics)-20))
	}

	conn.Status = "success"
	conn.Message = "连接成功"
	s.SetConnectionResult(conn, result.String())
	conn.ConnectedAt = time.Now()
	s.AddLog(conn, "✓ Kafka 连接测试完成")
}

// ExecuteKafkaCommand 执行 Kafka 命令
func (s *ConnectorService) ExecuteKafkaCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("Kafka 连接未建立")
	}

	port := conn.Port
	if port == "" {
		port = "9092"
	}

	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	brokers := []string{net.JoinHostPort(conn.IP, port)}

	// 创建 Kafka 配置
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second

	// 如果提供了认证信息
	if conn.User != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = conn.User
		config.Net.SASL.Password = conn.Pass
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	// 创建客户端
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer client.Close()

	// 解析命令
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("无效的命令")
	}

	cmdType := strings.ToLower(parts[0])
	var result strings.Builder

	switch cmdType {
	case "topics", "list-topics":
		// 列出所有 Topic
		topics, err := client.Topics()
		if err != nil {
			return "", fmt.Errorf("获取 Topic 列表失败: %v", err)
		}
		result.WriteString(fmt.Sprintf("共 %d 个 Topic:\n\n", len(topics)))
		for _, topic := range topics {
			result.WriteString(fmt.Sprintf("%s\n", topic))
		}

	case "brokers", "list-brokers":
		// 列出所有 Broker
		brokers := client.Brokers()
		result.WriteString(fmt.Sprintf("共 %d 个 Broker:\n\n", len(brokers)))
		for _, broker := range brokers {
			result.WriteString(fmt.Sprintf("ID: %d, Address: %s\n", broker.ID(), broker.Addr()))
		}

	case "version", "broker-version":
		// 获取 Broker 版本信息
		brokers := client.Brokers()
		if len(brokers) == 0 {
			return "", fmt.Errorf("没有可用的 Broker")
		}

		result.WriteString("Broker 版本信息:\n\n")
		for _, broker := range brokers {
			if err := broker.Open(config); err != nil {
				result.WriteString(fmt.Sprintf("Broker %d (%s): 连接失败 - %v\n", broker.ID(), broker.Addr(), err))
				continue
			}
			defer broker.Close()

			// 发送 ApiVersions 请求获取版本信息
			request := &sarama.ApiVersionsRequest{}
			response, err := broker.ApiVersions(request)
			if err != nil {
				result.WriteString(fmt.Sprintf("Broker %d (%s): 获取版本失败 - %v\n", broker.ID(), broker.Addr(), err))
				continue
			}

			result.WriteString(fmt.Sprintf("Broker %d (%s):\n", broker.ID(), broker.Addr()))
			result.WriteString(fmt.Sprintf("  支持的 API 数量: %d\n", len(response.ApiKeys)))
			
			// 显示部分 API 版本信息
			if len(response.ApiKeys) > 0 {
				result.WriteString("  主要 API 版本:\n")
				maxDisplay := 10
				if len(response.ApiKeys) < maxDisplay {
					maxDisplay = len(response.ApiKeys)
				}
				for i := 0; i < maxDisplay; i++ {
					apiKey := response.ApiKeys[i]
					result.WriteString(fmt.Sprintf("    API %d: v%d - v%d\n", 
						apiKey.ApiKey, apiKey.MinVersion, apiKey.MaxVersion))
				}
				if len(response.ApiKeys) > maxDisplay {
					result.WriteString(fmt.Sprintf("    ... 还有 %d 个 API\n", len(response.ApiKeys)-maxDisplay))
				}
			}
			result.WriteString("\n")
		}

	case "config", "broker-config":
		// 获取 Broker 配置（使用 franz-go 库）
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: config <broker-id>")
		}
		
		var brokerID int32
		fmt.Sscanf(parts[1], "%d", &brokerID)

		// 使用 franz-go 创建客户端
		opts := []kgo.Opt{
			kgo.SeedBrokers(brokers...),
			kgo.RequestTimeoutOverhead(10 * time.Second),
		}

		// 如果提供了认证信息
		if conn.User != "" {
			opts = append(opts, kgo.SASL(plain.Auth{
				User: conn.User,
				Pass: conn.Pass,
			}.AsMechanism()))
		}

		client, err := kgo.NewClient(opts...)
		if err != nil {
			return "", fmt.Errorf("创建 franz-go 客户端失败: %v", err)
		}
		defer client.Close()

		// 创建 admin 客户端
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		adminClient := kadm.NewClient(client)

		// 获取 Broker 配置
		configs, err := adminClient.DescribeBrokerConfigs(ctx, brokerID)
		if err != nil {
			return "", fmt.Errorf("获取配置失败: %v", err)
		}

		if len(configs) == 0 {
			return "", fmt.Errorf("未找到 Broker %d 的配置", brokerID)
		}

		// 获取第一个 Broker 的配置
		brokerConfig := configs[0]
		if brokerConfig.Err != nil {
			return "", fmt.Errorf("Broker %d 配置错误: %v", brokerID, brokerConfig.Err)
		}

		// 排序配置项
		configList := brokerConfig.Configs
		sort.Slice(configList, func(i, j int) bool {
			return configList[i].Key < configList[j].Key
		})

		result.WriteString(fmt.Sprintf("=== Broker %d 配置 ===\n\n", brokerID))
		result.WriteString(fmt.Sprintf("共 %d 个配置项\n\n", len(configList)))

		// 按类别分组显示
		categories := make(map[string][]string)
		for _, cfg := range configList {
			// 根据配置名称前缀分类
			category := "其他"
			if strings.HasPrefix(cfg.Key, "log.") {
				category = "日志配置"
			} else if strings.HasPrefix(cfg.Key, "num.") {
				category = "线程配置"
			} else if strings.HasPrefix(cfg.Key, "socket.") {
				category = "网络配置"
			} else if strings.HasPrefix(cfg.Key, "replica.") {
				category = "副本配置"
			} else if strings.HasPrefix(cfg.Key, "zookeeper.") {
				category = "Zookeeper 配置"
			} else if strings.HasPrefix(cfg.Key, "security.") || strings.HasPrefix(cfg.Key, "sasl.") || strings.HasPrefix(cfg.Key, "ssl.") {
				category = "安全配置"
			} else if strings.HasPrefix(cfg.Key, "compression.") {
				category = "压缩配置"
			}

			value := cfg.Value
			if cfg.Sensitive && cfg.Value != nil {
				sensitiveValue := "******"
				value = &sensitiveValue
			}

			valueStr := ""
			if value != nil {
				valueStr = *value
			}
			line := fmt.Sprintf("  %s = %s", cfg.Key, valueStr)
			if cfg.Source.String() != "DEFAULT_CONFIG" {
				line += fmt.Sprintf(" [来源: %s]", cfg.Source.String())
			}
			if cfg.Sensitive {
				line += " [敏感]"
			}

			categories[category] = append(categories[category], line)
		}

		// 按类别输出
		categoryOrder := []string{"网络配置", "日志配置", "副本配置", "线程配置", "安全配置", "压缩配置", "Zookeeper 配置", "其他"}
		for _, category := range categoryOrder {
			if lines, ok := categories[category]; ok && len(lines) > 0 {
				result.WriteString(fmt.Sprintf("【%s】\n", category))
				for _, line := range lines {
					result.WriteString(line + "\n")
				}
				result.WriteString("\n")
			}
		}

	case "broker-info":
		// 获取指定 Broker 的详细信息（不需要高版本 API）
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: broker-info <broker-id>")
		}
		
		var brokerID int32
		fmt.Sscanf(parts[1], "%d", &brokerID)

		brokers := client.Brokers()
		var targetBroker *sarama.Broker
		for _, broker := range brokers {
			if broker.ID() == brokerID {
				targetBroker = broker
				break
			}
		}

		if targetBroker == nil {
			return "", fmt.Errorf("未找到 Broker ID: %d", brokerID)
		}

		result.WriteString(fmt.Sprintf("=== Broker %d 信息 ===\n\n", brokerID))
		result.WriteString(fmt.Sprintf("地址: %s\n", targetBroker.Addr()))
		result.WriteString(fmt.Sprintf("ID: %d\n\n", targetBroker.ID()))

		// 尝试连接并获取 API 版本信息
		if err := targetBroker.Open(config); err == nil {
			defer targetBroker.Close()

			// 获取支持的 API 版本
			request := &sarama.ApiVersionsRequest{}
			response, err := targetBroker.ApiVersions(request)
			if err == nil {
				result.WriteString(fmt.Sprintf("支持的 API 数量: %d\n\n", len(response.ApiKeys)))
				result.WriteString("主要 API 版本:\n")
				
				// 显示重要的 API
				apiNames := map[int16]string{
					0: "Produce",
					1: "Fetch",
					2: "ListOffsets",
					3: "Metadata",
					8: "OffsetCommit",
					9: "OffsetFetch",
					10: "FindCoordinator",
					11: "JoinGroup",
					18: "ApiVersions",
					19: "CreateTopics",
					20: "DeleteTopics",
					32: "DescribeConfigs",
					33: "AlterConfigs",
				}

				for _, apiKey := range response.ApiKeys {
					if name, ok := apiNames[apiKey.ApiKey]; ok {
						result.WriteString(fmt.Sprintf("  %s (API %d): v%d - v%d\n", 
							name, apiKey.ApiKey, apiKey.MinVersion, apiKey.MaxVersion))
					}
				}
			} else {
				result.WriteString(fmt.Sprintf("获取 API 版本失败: %v\n", err))
			}
		} else {
			result.WriteString(fmt.Sprintf("连接 Broker 失败: %v\n", err))
		}

	case "cluster-info":
		// 获取集群信息
		brokers := client.Brokers()
		topics, _ := client.Topics()
		
		result.WriteString("=== Kafka 集群信息 ===\n\n")
		result.WriteString(fmt.Sprintf("Broker 数量: %d\n", len(brokers)))
		result.WriteString(fmt.Sprintf("Topic 数量: %d\n\n", len(topics)))

		result.WriteString("Brokers:\n")
		for _, broker := range brokers {
			result.WriteString(fmt.Sprintf("  ID: %d, Address: %s\n", broker.ID(), broker.Addr()))
		}

		// 获取控制器信息
		if len(brokers) > 0 {
			broker := brokers[0]
			if err := broker.Open(config); err == nil {
				defer broker.Close()
				
				metadataRequest := &sarama.MetadataRequest{}
				metadataResponse, err := broker.GetMetadata(metadataRequest)
				if err == nil {
					result.WriteString(fmt.Sprintf("\n控制器 ID: %d\n", metadataResponse.ControllerID))
					result.WriteString(fmt.Sprintf("集群 ID: %s\n", metadataResponse.ClusterID))
				}
			}
		}

	case "describe-topic":
		// 描述指定 Topic
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: describe-topic <topic-name>")
		}
		topicName := parts[1]
		
		partitions, err := client.Partitions(topicName)
		if err != nil {
			return "", fmt.Errorf("获取 Topic 分区失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("Topic: %s\n", topicName))
		result.WriteString(fmt.Sprintf("分区数量: %d\n\n", len(partitions)))

		for _, partition := range partitions {
			leader, err := client.Leader(topicName, partition)
			if err != nil {
				result.WriteString(fmt.Sprintf("分区 %d: 获取 Leader 失败\n", partition))
				continue
			}

			replicas, err := client.Replicas(topicName, partition)
			if err != nil {
				replicas = []int32{}
			}

			result.WriteString(fmt.Sprintf("分区 %d:\n", partition))
			result.WriteString(fmt.Sprintf("  Leader: Broker %d (%s)\n", leader.ID(), leader.Addr()))
			result.WriteString(fmt.Sprintf("  Replicas: %v\n", replicas))
		}

	case "consumer-groups", "list-groups":
		// 列出消费者组
		// 注意：sarama 库需要通过 Broker 来获取消费者组列表
		brokers := client.Brokers()
		if len(brokers) == 0 {
			return "", fmt.Errorf("没有可用的 Broker")
		}

		// 使用第一个 Broker
		broker := brokers[0]
		if err := broker.Open(config); err != nil {
			return "", fmt.Errorf("打开 Broker 连接失败: %v", err)
		}
		defer broker.Close()

		request := &sarama.ListGroupsRequest{}
		response, err := broker.ListGroups(request)
		if err != nil {
			return "", fmt.Errorf("获取消费者组列表失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("共 %d 个消费者组:\n\n", len(response.Groups)))
		for groupID, groupType := range response.Groups {
			result.WriteString(fmt.Sprintf("%s (类型: %s)\n", groupID, groupType))
		}

	case "offsets":
		// 获取 Topic 的偏移量信息
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: offsets <topic-name>")
		}
		topicName := parts[1]

		partitions, err := client.Partitions(topicName)
		if err != nil {
			return "", fmt.Errorf("获取分区失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("Topic: %s 偏移量信息\n\n", topicName))
		for _, partition := range partitions {
			oldest, err := client.GetOffset(topicName, partition, sarama.OffsetOldest)
			if err != nil {
				oldest = -1
			}

			newest, err := client.GetOffset(topicName, partition, sarama.OffsetNewest)
			if err != nil {
				newest = -1
			}

			result.WriteString(fmt.Sprintf("分区 %d: 最旧=%d, 最新=%d, 消息数=%d\n", 
				partition, oldest, newest, newest-oldest))
		}

	default:
		return "", fmt.Errorf("不支持的命令: %s\n\n支持的命令:\n  - topics/list-topics: 列出所有 Topic\n  - brokers/list-brokers: 列出所有 Broker\n  - version/broker-version: 查看 Broker 版本\n  - broker-info <id>: 查看 Broker 详细信息\n  - config <broker-id>: 查看 Broker 配置 (需要 Kafka 0.11.0+)\n  - cluster-info: 查看集群信息\n  - describe-topic <name>: 描述指定 Topic\n  - consumer-groups/list-groups: 列出消费者组\n  - offsets <topic>: 查看 Topic 偏移量", cmdType)
	}

	return result.String(), nil
}
