package main

import (
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	broker := "192.168.0.158:9094"
	topic := "test"

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Net.DialTimeout = 10 * time.Second

	producer, err := sarama.NewSyncProducer([]string{broker}, config)
	if err != nil {
		fmt.Printf("创建生产者失败: %v\n", err)
		return
	}
	defer producer.Close()

	messages := []struct{ key, value string }{
		{"user-001", `{"action":"login","user":"张三","ip":"10.0.1.100"}`},
		{"order-1001", `{"action":"create_order","orderId":"1001","amount":299.9}`},
		{"user-002", `{"action":"login","user":"李四","ip":"10.0.1.101"}`},
		{"alert-high", `{"level":"critical","message":"CPU 使用率超过 95%","host":"web-server-03"}`},
		{"order-1002", `{"action":"payment","orderId":"1002","status":"success","amount":1580}`},
	}

	for i, m := range messages {
		msg := &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(m.key),
			Value: sarama.StringEncoder(m.value),
		}
		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			fmt.Printf("[%d] 发送失败: %v\n", i+1, err)
		} else {
			fmt.Printf("[%d] 发送成功  P:%d O:%d  Key: %s\n", i+1, partition, offset, m.key)
		}
	}
	fmt.Println("\n5 条测试消息发送完毕")
}
