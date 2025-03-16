package case35

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"interview-cases/test"
	"sync"
	"testing"
	"time"
)

func TestConsumerPerformance(t *testing.T) {
	test.InitTopic(kafka.TopicSpecification{
		Topic:         "logs",
		NumPartitions: 3,
	})
	go producerLoop(t)
	time.Sleep(1 * time.Second)
	// 优化前配置（默认参数）
	defaultConfig := &kafka.ConfigMap{
		"bootstrap.servers":  "localhost:9092",
		"group.id":           "default-consumer-group",
		"auto.offset.reset":  "earliest",
		"fetch.min.bytes":    1,   // 默认值
		"fetch.wait.max.ms":  100, // 默认值
		"enable.auto.commit": false,
	}

	// 优化后配置
	optimizedConfig := &kafka.ConfigMap{
		"bootstrap.servers":  "localhost:9092",
		"group.id":           "optimized-consumer-group",
		"auto.offset.reset":  "earliest",
		"fetch.min.bytes":    32768, // 32KB
		"fetch.wait.max.ms":  500,   // 500ms
		"enable.auto.commit": false,
	}

	// 初始化统计
	defaultStats := &ConsumerStats{StartTime: time.Now()}
	optimizedStats := &ConsumerStats{StartTime: time.Now()}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	// 启动消费者组
	var wg sync.WaitGroup
	startConsumerGroup(t, ctx, "default", defaultConfig, 1, defaultStats, &wg)       // 优化前单线程
	startConsumerGroup(t, ctx, "optimized", optimizedConfig, 4, optimizedStats, &wg) // 优化后4线程

	// 启动统计打印
	go printStatsPeriodically(defaultStats, optimizedStats)

	// 等待所有消费者退出
	wg.Wait()
	printFinalStats(defaultStats, optimizedStats)
}

func producerLoop(t *testing.T) {
	producer := newProducer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go producer.ProduceLoop(ctx)
	var eg errgroup.Group
	for i := 0; i < 10; i++ {
		eg.Go(func() error {
			producer.ProduceLoop(ctx)
			return nil
		})
	}
	err := eg.Wait()
	require.NoError(t, err)
}

func newProducer(t *testing.T) *Producer {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	})
	require.NoError(t, err)
	return &Producer{
		producer: producer,
	}
}

func startConsumerGroup(t *testing.T, ctx context.Context, name string, config *kafka.ConfigMap, numConsumers int, stats *ConsumerStats, wg *sync.WaitGroup) {
	for i := 0; i < numConsumers; i++ {
		consumer, err := kafka.NewConsumer(config)
		if err != nil {
			require.NoError(t, err)
		}

		if err := consumer.Subscribe("logs", nil); err != nil {
			require.NoError(t, err)
		}
		wg.Add(1)
		go func(id int, c *kafka.Consumer) {
			defer wg.Done()
			defer c.Close()
			fmt.Printf("[%s-%d] Consumer started\n", name, id)

			for {
				if ctx.Err() != nil {
					return
				}
				msg, err := c.ReadMessage(100 * time.Millisecond)
				if err != nil {
					if err.(kafka.Error).Code() == kafka.ErrTimedOut {
						continue
					}
					break
				}

				start := time.Now()
				processMessage(msg)
				stats.Add(time.Since(start))

				if _, err := c.CommitMessage(msg); err != nil {
					fmt.Printf("[%s-%d] Commit failed: %v\n", name, id, err)
				}
			}
		}(i, consumer)
	}
}

func processMessage(msg *kafka.Message) {
	// 模拟处理耗时：优化前/后处理逻辑相同
	time.Sleep(50 * time.Millisecond)
}

func printStatsPeriodically(defaultStats, optimizedStats *ConsumerStats) {
	for {
		time.Sleep(5 * time.Second)
		fmt.Println("\n=== 性能报告 ===")
		fmt.Println("[默认配置]   ", defaultStats.String())
		fmt.Println("[优化配置] ", optimizedStats.String())
		fmt.Println("===========================")
	}
}

func printFinalStats(defaultStats, optimizedStats *ConsumerStats) {
	fmt.Println("\n=== 最终性能对比 ===")
	fmt.Println("[默认配置]   ", defaultStats.String())
	fmt.Println("[优化配置] ", optimizedStats.String())

	defaultTime := time.Since(defaultStats.StartTime).Seconds()
	optimizedTime := time.Since(optimizedStats.StartTime).Seconds()

	fmt.Printf("\n性能提升:\n")
	fmt.Printf("吞吐量: %.1fx 提升\n",
		float64(optimizedStats.TotalMessages)/optimizedTime/(float64(defaultStats.TotalMessages)/defaultTime))
	fmt.Printf("延迟: 优化后为原来的 %.1f%%\n",
		(float64(optimizedStats.TotalDuration)/float64(optimizedStats.TotalMessages))/
			(float64(defaultStats.TotalDuration)/float64(defaultStats.TotalMessages))*100)
}
