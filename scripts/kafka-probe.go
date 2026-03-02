package main

import (
	"fmt"
	"os"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	broker := "192.168.0.158:9094"

	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second

	client, err := sarama.NewClient([]string{broker}, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接失败: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Broker 信息
	fmt.Println("===== Broker 信息 =====")
	for _, b := range client.Brokers() {
		fmt.Printf("  ID: %d  Addr: %s\n", b.ID(), b.Addr())
	}

	// 尝试获取 API 版本来推断 Kafka 版本
	fmt.Println("\n===== 尝试不同版本配置 =====")
	versions := []sarama.KafkaVersion{
		sarama.V3_6_0_0,
		sarama.V3_5_0_0,
		sarama.V3_0_0_0,
		sarama.V2_8_0_0,
		sarama.V2_6_0_0,
		sarama.V2_4_0_0,
		sarama.V2_0_0_0,
		sarama.V1_1_0_0,
		sarama.V0_11_0_0,
		sarama.V0_10_2_0,
	}
	for _, v := range versions {
		cfg := sarama.NewConfig()
		cfg.Net.DialTimeout = 5 * time.Second
		cfg.Version = v
		c, err := sarama.NewClient([]string{broker}, cfg)
		if err != nil {
			fmt.Printf("  版本 %-10s -> 连接失败: %v\n", v, err)
			continue
		}
		// 尝试获取 coordinator
		coord, err := c.Coordinator("test-probe")
		if err != nil {
			fmt.Printf("  版本 %-10s -> 连接成功, Coordinator 失败: %v\n", v, err)
		} else {
			fmt.Printf("  版本 %-10s -> 连接成功, Coordinator: %s (ID:%d) ✓\n", v, coord.Addr(), coord.ID())
		}
		c.Close()
	}

	// Topic 列表
	fmt.Println("\n===== Topic 列表 =====")
	topics, _ := client.Topics()
	for _, t := range topics {
		parts, _ := client.Partitions(t)
		fmt.Printf("  %-40s  分区数: %d\n", t, len(parts))
	}

	// 现有消费组
	fmt.Println("\n===== 现有消费组 =====")
	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		fmt.Printf("  创建 admin 失败: %v\n", err)
		return
	}
	groups, err := admin.ListConsumerGroups()
	if err != nil {
		fmt.Printf("  获取消费组失败: %v\n", err)
	} else if len(groups) == 0 {
		fmt.Println("  (无消费组)")
	} else {
		for id, proto := range groups {
			fmt.Printf("  %s  (protocol: %s)\n", id, proto)
		}
	}
}
