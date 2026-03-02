package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
)

const (
	broker = "192.168.0.158:9094"
	topic  = "test"
	group  = "opshub-console-group"
)

func main() {
	fmt.Println("=========================================")
	fmt.Println(" Kafka 消费组消费监听")
	fmt.Printf(" Broker:  %s\n", broker)
	fmt.Printf(" Topic:   %s\n", topic)
	fmt.Printf(" Group:   %s\n", group)
	fmt.Println("=========================================")
	fmt.Println()

	// ---- 第一步：修复 __consumer_offsets ----
	fmt.Println("[1/3] 检查 __consumer_offsets ...")
	if err := ensureConsumerOffsets(); err != nil {
		fmt.Fprintf(os.Stderr, "修复失败: %v\n", err)
		fmt.Fprintln(os.Stderr, "\n请在 Kafka server.properties 中设置:")
		fmt.Fprintln(os.Stderr, "  offsets.topic.replication.factor=1")
		fmt.Fprintln(os.Stderr, "  transaction.state.log.replication.factor=1")
		fmt.Fprintln(os.Stderr, "然后重启 Kafka")
		os.Exit(1)
	}

	// ---- 第二步：等待 Coordinator 就绪 ----
	fmt.Println("[2/3] 等待 Coordinator 就绪 ...")
	if err := waitCoordinator(); err != nil {
		fmt.Fprintf(os.Stderr, "Coordinator 未就绪: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("      Coordinator 已就绪")

	// ---- 第三步：启动消费组消费 ----
	fmt.Println("[3/3] 启动消费组消费 ...")
	startConsumerGroup()
}

func ensureConsumerOffsets() error {
	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second
	config.Version = sarama.V2_6_0_0

	admin, err := sarama.NewClusterAdmin([]string{broker}, config)
	if err != nil {
		return fmt.Errorf("创建 admin 失败: %w", err)
	}
	defer admin.Close()

	topics, err := admin.ListTopics()
	if err != nil {
		return fmt.Errorf("列出 topic 失败: %w", err)
	}

	if _, exists := topics["__consumer_offsets"]; exists {
		fmt.Println("      __consumer_offsets 已存在")
		return nil
	}

	fmt.Println("      __consumer_offsets 不存在，正在创建 (replication.factor=1) ...")
	err = admin.CreateTopic("__consumer_offsets", &sarama.TopicDetail{
		NumPartitions:     50,
		ReplicationFactor: 1,
		ConfigEntries: map[string]*string{
			"cleanup.policy": strPtr("compact"),
			"compression.type": strPtr("producer"),
		},
	}, false)
	if err != nil {
		return fmt.Errorf("创建 __consumer_offsets 失败: %w", err)
	}
	fmt.Println("      __consumer_offsets 创建成功，等待 5 秒让 broker 初始化 ...")
	time.Sleep(5 * time.Second)
	return nil
}

func waitCoordinator() error {
	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second
	config.Version = sarama.V2_6_0_0

	for i := 0; i < 10; i++ {
		client, err := sarama.NewClient([]string{broker}, config)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		coord, err := client.Coordinator(group)
		client.Close()
		if err == nil && coord != nil {
			return nil
		}
		fmt.Printf("      重试 %d/10 ...\n", i+1)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("超时")
}

func startConsumerGroup() {
	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
	}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := sarama.NewConsumerGroup([]string{broker}, group, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建消费组失败: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	handler := &handler{}

	// 错误日志
	go func() {
		for err := range client.Errors() {
			fmt.Fprintf(os.Stderr, "[错误] %v\n", err)
		}
	}()

	// 消费循环
	go func() {
		for {
			if err := client.Consume(ctx, []string{topic}, handler); err != nil {
				fmt.Fprintf(os.Stderr, "消费错误: %v\n", err)
				time.Sleep(2 * time.Second)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println()
	fmt.Println("消费组监听已启动，按 Ctrl+C 停止...")
	fmt.Println("-----------------------------------------")

	<-sig
	fmt.Printf("\n共消费 %d 条消息，正在退出...\n", handler.count)
	cancel()
}

type handler struct{ count int }

func (h *handler) Setup(s sarama.ConsumerGroupSession) error {
	fmt.Printf("[消费组] 加入成功, MemberID: %s, 分配: %v\n", s.MemberID(), s.Claims())
	return nil
}
func (h *handler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.count++
		ts := msg.Timestamp.Format("2006-01-02 15:04:05")
		key := string(msg.Key)
		value := string(msg.Value)
		if len(value) > 200 {
			value = value[:200] + "..."
		}
		var hdrs []string
		for _, hdr := range msg.Headers {
			hdrs = append(hdrs, fmt.Sprintf("%s=%s", string(hdr.Key), string(hdr.Value)))
		}
		extra := ""
		if len(hdrs) > 0 {
			extra = " | Headers: {" + strings.Join(hdrs, ", ") + "}"
		}
		fmt.Printf("[%s] P:%d O:%-6d | Key: %-20s | %s%s\n",
			ts, msg.Partition, msg.Offset, key, value, extra)
		session.MarkMessage(msg, "")
	}
	return nil
}

func strPtr(s string) *string { return &s }
