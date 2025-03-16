// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package case1_10

import (
	"fmt"
	"interview-cases/test"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestCase3(t *testing.T) {
	suite.Run(t, new(Case3TestSuite))
}

// Case3TestSuite 模拟索引失效导致表锁的问题
type Case3TestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *Case3TestSuite) SetupSuite() {
	s.db = test.InitDB()
	// 借助 GORM 来初始化表结构
	err := s.db.AutoMigrate(&PaymentLog{})
	require.NoError(s.T(), err)
}

func (s *Case3TestSuite) TearDownSuite() {
	// 如果你不希望测试结束就删除数据，你把这段代码注释掉
	err := s.db.Exec("TRUNCATE TABLE `payment_logs`").Error
	require.NoError(s.T(), err)
}

func (s *Case3TestSuite) TestIndexInvalidationTableLock() {
	// 准备数据
	s.prepareData()

	// 测试索引失效导致的表锁问题
	s.demonstrateTableLock()

	// 测试正确使用索引避免表锁
	s.demonstrateProperIndexUsage()
}

func (s *Case3TestSuite) prepareData() {
	// 首先清空表，确保测试数据是干净的
	err := s.db.Exec("TRUNCATE TABLE `payment_logs`").Error
	require.NoError(s.T(), err)

	// 准备测试数据
	const batchSize = 1000
	const batch = 10
	logs := make([]PaymentLog, 0, batchSize)
	now := time.Now().UnixMilli()

	for i := 0; i < batch; i++ {
		for j := 0; j < batchSize; j++ {
			idx := i*batchSize + j
			logs = append(logs, PaymentLog{
				PaymentID: fmt.Sprintf("%d", 10000+idx),            // 注意这里使用字符串格式的数字，与隐式转换场景契合
				Payer:     fmt.Sprintf("user_%d", idx%100),         // 100个不同的用户
				Ctime:     now - int64(rand.Intn(30*24*3600*1000)), // 随机过去30天内的时间
				Utime:     now,
			})
		}

		err := s.db.Create(&logs).Error
		require.NoError(s.T(), err)
		logs = logs[:0] // 清空slice但保留容量
	}

	s.T().Log("创建了", batchSize*batch, "条测试数据")
}

func (s *Case3TestSuite) demonstrateTableLock() {
	config := lockTestConfig{
		desc:      "索引失效导致表锁",
		tx1Prefix: "事务1",
		tx2Prefix: "事务2",
		paymentID: 11729, // 使用整数而非字符串，发生隐士数据类型转换，导致索引失效
		payer:     "user_29",
	}
	s.runLockTest(config)
}

func (s *Case3TestSuite) demonstrateProperIndexUsage() {
	config := lockTestConfig{
		desc:      "正确使用索引避免表锁",
		tx1Prefix: "事务3",
		tx2Prefix: "事务4",
		paymentID: "11729", // 正确使用字符串类型
		payer:     "user_39",
	}
	s.runLockTest(config)
}

// 执行锁测试的通用方法
func (s *Case3TestSuite) runLockTest(config lockTestConfig) {
	s.T().Log("开始演示", config.desc)

	// 使用通道进行并发控制
	done := make(chan bool)

	// 第一个事务
	go func() {
		tx := s.db.Begin()
		defer tx.Rollback()

		txLabel := config.tx1Prefix
		s.T().Log(txLabel+": 开始执行 - 使用", fmt.Sprintf("%T", config.paymentID), "类型查询")

		// 首先检查执行计划
		var explain []map[string]interface{}
		s.db.Raw("EXPLAIN SELECT * FROM `payment_logs` WHERE payment_id = ?", config.paymentID).Scan(&explain)
		s.T().Log(txLabel+"的执行计划:", explain)

		// 执行查询和更新
		var result PaymentLog
		err := tx.Where("payment_id = ?", config.paymentID).First(&result).Error
		if err != nil {
			s.T().Log("查询未找到结果，这是正常的，因为我们用的是随机ID")
		}

		// 更新一批数据
		err = tx.Model(&PaymentLog{}).Where("payment_id = ?", config.paymentID).
			Updates(map[string]interface{}{"utime": time.Now().UnixMilli()}).Error
		require.NoError(s.T(), err)

		s.T().Log(txLabel + ": 睡眠5秒模拟长事务")
		time.Sleep(5 * time.Second)

		s.T().Log(txLabel + ": 提交事务")
		tx.Commit()

		done <- true
	}()

	// 给第一个事务一些启动时间
	time.Sleep(1 * time.Second)

	// 第二个事务
	go func() {
		tx := s.db.Begin()
		defer tx.Rollback()

		txLabel := config.tx2Prefix
		s.T().Log(txLabel + ": 尝试更新不同的支付记录")
		start := time.Now()

		// 尝试更新不同的支付记录
		err := tx.Model(&PaymentLog{}).Where("payer = ?", config.payer).
			Updates(map[string]interface{}{"utime": time.Now().UnixMilli()}).Error
		duration := time.Since(start)
		require.NoError(s.T(), err)

		s.T().Log(txLabel+": 更新完成，耗时:", duration)

		// 根据测试场景输出不同的结论提示
		if config.desc == "索引失效导致表锁" {
			s.T().Log(txLabel + ": 如果耗时超过4秒，说明被" + config.tx1Prefix + "阻塞了，证明发生了表锁")
		} else {
			s.T().Log(txLabel + ": 如果耗时远小于5秒，说明没有被" + config.tx1Prefix + "阻塞，证明使用了行锁")
		}

		tx.Commit()
		done <- true
	}()

	// 等待两个goroutine完成
	<-done
	<-done

	s.T().Log(config.desc, "演示完成")
}

// PaymentLog 模型定义
type PaymentLog struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	PaymentID string `gorm:"column:payment_id;type:varchar(32);not null;index"`
	Payer     string `gorm:"column:payer;type:varchar(32);not null;index"`
	Ctime     int64  `gorm:"column:ctime;not null"`
	Utime     int64  `gorm:"column:utime;not null"`
}

// TableName 指定表名
func (p PaymentLog) TableName() string {
	return "payment_logs"
}

// 场景配置结构体
type lockTestConfig struct {
	// 测试标识符（如"表锁"或"行锁"）
	desc string

	// 事务标识前缀（如"事务1"、"事务3"）
	tx1Prefix string
	tx2Prefix string

	// 支付ID，用数字和字符串两种方式来验证索引失效
	paymentID any
	// 支付者名称
	payer string
}
