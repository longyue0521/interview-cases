package case36

import (
	"context"
	"fmt"
	"interview-cases/test"
	"strings"
	"testing"
)

func BenchmarkOptimization(b *testing.B) {
	rdb := test.InitRedis()
	defer rdb.FlushDB(context.Background())

	productID := 10010
	longDesc := strings.Repeat("A", 10240) // 10KB长描述

	// 场景1：未优化的存储结构（所有字段在一个Hash）
	bigKey := fmt.Sprintf("product:%d:all", productID)
	rdb.HSet(context.Background(), bigKey,
		"title", "商品标题",
		"price", 99.99,
		"stock", 100,
		"description", longDesc,
		"spec", `{"color":"red","size":"XL"}`,
	)

	// 场景2：优化后的存储结构（拆分两个Hash）
	baseKey := fmt.Sprintf("product:%d:base", productID)
	detailKey := fmt.Sprintf("product:%d:detail", productID)
	rdb.HSet(context.Background(), baseKey,
		"title", "商品标题",
		"price", 99.99,
		"stock", 100,
	)
	rdb.HSet(context.Background(), detailKey,
		"description", longDesc,
		"spec", `{"color":"red","size":"XL"}`,
	)

	b.Run("未优化-全量读取", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 需要从大Hash中获取全部字段（实际只关注基础字段）
			rdb.HGetAll(context.Background(), bigKey).Result()
		}
	})

	b.Run("优化后-基础读取", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 访问基础字段的部分字段
			rdb.HMGet(context.Background(), baseKey, "title", "price", "stock").Result()
		}
	})
}
