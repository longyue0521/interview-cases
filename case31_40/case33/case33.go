package case33

import (
	"fmt"
	"gorm.io/gorm"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

func InitializeData(db *gorm.DB) error {
	rand.Seed(time.Now().UnixNano())
	const (
		userBatchSize  = 2000 // 用户批次大小
		orderBatchSize = 5000 // 订单批次大小
		maxConcurrency = 8    // 最大并发数
	)

	// 生成用户数据（保持串行）
	var users []User
	for i := 1; i <= 20000; i++ {
		users = append(users, User{Name: "User" + strconv.Itoa(i)})
	}

	// 批量插入用户（保持串行）
	if err := db.CreateInBatches(users, userBatchSize).Error; err != nil {
		return err
	}

	// 预生成所有订单数据
	var allOrders []Order
	for _, user := range users {
		orderCount := rand.Intn(4) + 3
		for j := 0; j < orderCount; j++ {
			allOrders = append(allOrders, Order{
				UserID:      user.ID,
				TotalAmount: rand.Intn(100) + 1,
			})
		}
	}

	// 使用并发插入订单
	return concurrentInsert(db, allOrders, orderBatchSize, maxConcurrency)
}

// 并发插入核心逻辑
func concurrentInsert(db *gorm.DB, data []Order, batchSize int, maxWorkers int) error {
	var wg sync.WaitGroup
	errChan := make(chan error, maxWorkers)
	sem := make(chan struct{}, maxWorkers) // 并发信号量

	total := len(data)
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		batch := data[i:end]

		wg.Add(1)
		go func(b []Order) {
			defer wg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			// 每个goroutine使用独立的数据库连接
			tx := db.Session(&gorm.Session{CreateBatchSize: batchSize})
			if err := tx.Create(b).Error; err != nil {
				errChan <- fmt.Errorf("batch insert failed: %w", err)
			}
		}(batch)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(errChan)
		close(sem)
	}()

	// 收集错误
	for err := range errChan {
		if err != nil {
			return err // 返回第一个错误
		}
	}
	return nil
}

func GetUserTotalsV1(db *gorm.DB) ([]UserTotal, error) {
	var results []UserTotal

	// 执行查询
	err := db.Model(&User{}).
		Select("users.id AS user_id, users.name, COALESCE(SUM(orders.total_amount), 0) AS total_amount").
		Joins("LEFT JOIN orders ON users.id = orders.user_id").
		Group("users.id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func CreateUserIdx(db *gorm.DB) error {
	return db.Exec("CREATE INDEX idx_user_id ON orders (user_id);").Error

}

func GetUserTotalsV2(db *gorm.DB) ([]UserTotal, error) {
	var results []UserTotal

	query := `
       SELECT u.id, u.name, total_amount  
FROM users u  
LEFT JOIN (  
    SELECT user_id, SUM(total_amount) AS total_amount  
    FROM orders  
    GROUP BY user_id  
) o ON u.id = o.user_id;  

    `

	// 使用Raw方法执行自定义SQL，并扫描到results
	err := db.Raw(query).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}
