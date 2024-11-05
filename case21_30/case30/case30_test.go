package case30

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"interview-cases/test"
	"testing"
	"time"
)

type Case30 struct {
	ID     int64 `gorm:"primaryKey;autoIncrement"` // 设置为主键
	Name   string
	Status int32
}

type TestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (t *TestSuite) SetupSuite() {
	t.db = test.InitDB()
	// 借助 GORM 来初始化表结构
	err := t.db.AutoMigrate(&Case30{})
	require.NoError(t.T(), err)
	// 初始化一些数据
	t.prepareData()
}

func (t *TestSuite) TestTableLock() {
	t.db.Transaction(func(tx *gorm.DB) error {
		// select for update 查找一个不存在列会导致锁表
		var ca Case30
		err := tx.WithContext(context.Background()).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&Case30{}).
			Where("id = ?", 20000).First(&ca).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 稍微把时间调长一点方便观察表锁现象
				time.Sleep(30 * time.Second)
				return nil
			} else {
				return err
			}
		}
		// 如果存在，就更新
		tx.WithContext(context.Background()).Where("id = ?", 20000).
			Update("status", 1)
		return nil
	})
}

func (t *TestSuite) TestCas() {
	var ca Case30
	err := t.db.WithContext(context.Background()).Create(&Case30{
		ID:     20000,
		Status: 0,
	}).Error
	require.NoError(t.T(), err)
	for {
		err := t.db.WithContext(context.Background()).
			Model(&Case30{}).
			Where("id = ? ", 20000).First(&ca).Error
		if err != nil {
			return
		}
		// 找到就更新
		res := t.db.WithContext(context.Background()).
			Model(&Case30{}).
			Where("id = ? and status = ?", 20000, ca.Status).
			Update("status", 1)
		if res.Error != nil {
			return
		}
		if res.RowsAffected > 0 {
			// 说明修改成功
			return
		}
	}

}

func (t *TestSuite) prepareData() {
	// 先清空数据
	err := t.db.WithContext(context.Background()).Exec("DELETE FROM users;").Error
	require.NoError(t.T(), err)
	users := make([]*Case30, 0, 32)
	// 插入1000条数据
	for i := 1; i <= 1000; i++ {
		u := &Case30{
			Name: fmt.Sprintf("user_%d", i),
		}
		users = append(users, u)
	}
	err = t.db.WithContext(context.Background()).Create(users).Error
	require.NoError(t.T(), err)
}

func TestCase30(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
