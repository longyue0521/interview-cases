package case35

import (
	"bytes"
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"sync"
	"time"
)

// 性能统计结构
type ConsumerStats struct {
	TotalMessages int
	TotalDuration time.Duration
	StartTime     time.Time
	mu            sync.Mutex
}

func (s *ConsumerStats) Add(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalMessages++
	s.TotalDuration += duration
}

func (s *ConsumerStats) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.TotalMessages == 0 {
		return "No messages processed"
	}
	avgLatency := s.TotalDuration / time.Duration(s.TotalMessages)
	totalTime := time.Since(s.StartTime)
	tps := float64(s.TotalMessages) / totalTime.Seconds()
	return fmt.Sprintf("Processed %d messages | Avg latency: %v | TPS: %.1f",
		s.TotalMessages, avgLatency, tps)
}

type Producer struct {
	producer *kafka.Producer
}

func (p *Producer) ProduceLoop(ctx context.Context) {
	logsTopic := "logs"
	// 主生产循环
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := p.producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &logsTopic,
					Partition: kafka.PartitionAny,
				},
				Value: []byte(GenerateFixedSizeStringKB(50)),
			}, nil)
			time.Sleep(10 * time.Millisecond)
			if err != nil {
				fmt.Printf("消息生产失败: %v\n", err)
			}
		}
	}
}

func GenerateFixedSizeStringKB(sizeKB int) string {
	const (
		blockSize    = 1024 // 基础块大小1KB
		printableMin = 32   // 可打印字符最小ASCII码（空格）
		printableMax = 126  // 可打印字符最大ASCII码（~）
	)

	if sizeKB <= 0 {
		return ""
	}

	targetSize := sizeKB * blockSize                    // 转换为字节
	repeats := (targetSize + blockSize - 1) / blockSize // 计算需要完整块的数量

	// 预分配精确内存
	buf := bytes.NewBuffer(make([]byte, 0, targetSize))

	// 生成基础块（95个可打印字符循环）
	base := make([]byte, blockSize)
	for i := range base {
		base[i] = byte(printableMin + i%(printableMax-printableMin+1))
	}

	// 写入完整块
	for i := 0; i < repeats; i++ {
		remaining := targetSize - buf.Len()
		if remaining <= 0 {
			break
		}
		writeSize := min(remaining, blockSize)
		buf.Write(base[:writeSize])
	}

	return string(buf.Bytes())
}
