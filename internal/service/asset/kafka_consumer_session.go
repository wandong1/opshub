package asset

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

// KafkaMessage 消费到的消息
type KafkaMessage struct {
	Partition int32             `json:"partition"`
	Offset    int64             `json:"offset"`
	Timestamp time.Time         `json:"timestamp"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Size      int               `json:"size"`
	Headers   map[string]string `json:"headers"`
}

// KafkaConsumerSession 消费会话
type KafkaConsumerSession struct {
	ID        string
	Topic     string
	Consumer  sarama.Consumer
	Cancel    context.CancelFunc
	Messages  []KafkaMessage
	mu        sync.Mutex
	CreatedAt time.Time
	LastPoll  time.Time
	pollIndex int
	maxSize   int
}

// KafkaConsumerSessionManager 全局会话管理器
type KafkaConsumerSessionManager struct {
	sessions map[string]*KafkaConsumerSession
	mu       sync.RWMutex
}

var kafkaSessionManager = &KafkaConsumerSessionManager{
	sessions: make(map[string]*KafkaConsumerSession),
}

// StartSession 启动消费会话
func (m *KafkaConsumerSessionManager) StartSession(consumer sarama.Consumer, topic string, startOffset int64) (string, error) {
	m.cleanupStale()

	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithCancel(context.Background())
	sessionID := uuid.New().String()
	session := &KafkaConsumerSession{
		ID:        sessionID,
		Topic:     topic,
		Consumer:  consumer,
		Cancel:    cancel,
		Messages:  make([]KafkaMessage, 0, 256),
		CreatedAt: time.Now(),
		LastPoll:  time.Now(),
		maxSize:   5000,
	}

	for _, partition := range partitions {
		pc, err := consumer.ConsumePartition(topic, partition, startOffset)
		if err != nil {
			cancel()
			consumer.Close()
			return "", err
		}
		go session.consumePartition(ctx, pc)
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return sessionID, nil
}

// consumePartition 消费单个分区的消息
func (s *KafkaConsumerSession) consumePartition(ctx context.Context, pc sarama.PartitionConsumer) {
	defer pc.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-pc.Messages():
			if !ok {
				return
			}
			headers := make(map[string]string)
			for _, h := range msg.Headers {
				headers[string(h.Key)] = string(h.Value)
			}
			km := KafkaMessage{
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Timestamp: msg.Timestamp,
				Key:       string(msg.Key),
				Value:     string(msg.Value),
				Size:      len(msg.Key) + len(msg.Value),
				Headers:   headers,
			}
			s.mu.Lock()
			if len(s.Messages) >= s.maxSize {
				// 环形缓冲：丢弃前半部分
				half := s.maxSize / 2
				copy(s.Messages, s.Messages[half:])
				s.Messages = s.Messages[:len(s.Messages)-half]
				if s.pollIndex > half {
					s.pollIndex -= half
				} else {
					s.pollIndex = 0
				}
			}
			s.Messages = append(s.Messages, km)
			s.mu.Unlock()
		}
	}
}

// PollMessages 拉取新消息
func (m *KafkaConsumerSessionManager) PollMessages(sessionID, keyword string, limit int) ([]KafkaMessage, bool) {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	session.LastPoll = time.Now()

	if limit <= 0 {
		limit = 200
	}

	var result []KafkaMessage
	for i := session.pollIndex; i < len(session.Messages); i++ {
		msg := session.Messages[i]
		if keyword != "" {
			if !strings.Contains(msg.Key, keyword) && !strings.Contains(msg.Value, keyword) {
				continue
			}
		}
		result = append(result, msg)
		if len(result) >= limit {
			break
		}
	}
	session.pollIndex = len(session.Messages)
	return result, true
}

// StopSession 停止消费会话
func (m *KafkaConsumerSessionManager) StopSession(sessionID string) {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if ok {
		session.Cancel()
		session.Consumer.Close()
	}
}

// cleanupStale 清理超过 5 分钟未 poll 的会话
func (m *KafkaConsumerSessionManager) cleanupStale() {
	m.mu.Lock()
	defer m.mu.Unlock()

	threshold := time.Now().Add(-5 * time.Minute)
	for id, session := range m.sessions {
		if session.LastPoll.Before(threshold) {
			session.Cancel()
			session.Consumer.Close()
			delete(m.sessions, id)
		}
	}
}
