package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	broker := "192.168.0.158:9094"
	topic := "test"

	fmt.Println("=========================================")
	fmt.Println(" Kafka 消费监听（Partition Consumer）")
	fmt.Printf(" Broker:    %s\n", broker)
	fmt.Printf(" Topic:     %s\n", topic)
	fmt.Println(" 起始位置:  earliest (从最早开始)")
	fmt.Println("=========================================")
	fmt.Println()

	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumer([]string{broker}, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建消费者失败: %v\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	partitions, err := consumer.Partitions(topic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取分区失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Topic [%s] 共 %d 个分区\n", topic, len(partitions))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	count := 0
	for _, p := range partitions {
		pc, err := consumer.ConsumePartition(topic, p, sarama.OffsetOldest)
		if err != nil {
			fmt.Fprintf(os.Stderr, "消费分区 %d 失败: %v\n", p, err)
			continue
		}
		go func(pc sarama.PartitionConsumer) {
			defer pc.Close()
			for msg := range pc.Messages() {
				ts := msg.Timestamp.Format("2006-01-02 15:04:05")
				key := string(msg.Key)
				value := string(msg.Value)
				if len(value) > 200 {
					value = value[:200] + "..."
				}
				var headers []string
				for _, h := range msg.Headers {
					headers = append(headers, fmt.Sprintf("%s=%s", string(h.Key), string(h.Value)))
				}
				headerStr := ""
				if len(headers) > 0 {
					headerStr = fmt.Sprintf(" | Headers: {%s}", strings.Join(headers, ", "))
				}
				count++
				fmt.Printf("[%s] P:%d O:%-6d | Key: %-20s | %s%s\n",
					ts, msg.Partition, msg.Offset, key, value, headerStr)
			}
		}(pc)
	}

	fmt.Println("消费监听已启动，按 Ctrl+C 停止...")
	fmt.Println("-----------------------------------------")

	<-sig
	fmt.Printf("\n共消费 %d 条消息，正在退出...\n", count)
}
