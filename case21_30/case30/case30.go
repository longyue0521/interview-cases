package case30

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"math/rand/v2"
	"time"
)

type User struct {
	ID     int64  `gorm:"primaryKey"` // 设置为主键
	Name   string `gorm:"size:100"`   // 设置字段大小（可选）
	Gender int32  `gorm:"type:int"`   // 指定数据类型（可选）
}

func InitTable(db *gorm.DB) {
	db.AutoMigrate(&User{})
}

func InitData(db *gorm.DB) error {
	for i := 1; i <= 10000; i++ {
		u := &User{
			Name:   fmt.Sprintf("user_%d", i),
			Gender: int32(rand.IntN(2)),
		}
		err := db.WithContext(context.Background()).Create(u).Error
		if err != nil {
			return err
		}
	}
	return nil
}

type UserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		db: db,
	}
}

func (u *UserDAO) MetaDataLock() error {
	// 模拟频繁有事务对表进行操作
	var eg errgroup.Group
	for i := 0; i < 20; i++ {
		eg.Go(func() error {
			return u.queryUser()
		})
	}
	time.Sleep(1 * time.Second)
	eg.Go(func() error {
		return u.AddColumn()
	})
	return eg.Wait()
}

func (u *UserDAO) MetaDataLockWithTimeout() error {
	// 模拟频繁有事务对表进行操作
	var eg errgroup.Group
	for i := 0; i < 20; i++ {
		eg.Go(func() error {
			return u.queryUser()
		})
	}
	time.Sleep(1 * time.Second)
	eg.Go(func() error {
		u.AddColumnWithTimeout()
		return nil
	})
	return eg.Wait()
}

// 长事务
func (u *UserDAO) queryUser() error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		var us User
		err := tx.WithContext(context.Background()).Limit(1).Model(&User{}).Scan(&us).Error
		if err != nil {
			return err
		}
		// 模拟处理很多业务
		time.Sleep(30 * time.Second)
		return nil
	})
}

// 添加字段
func (u *UserDAO) AddColumn() error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		err := tx.WithContext(context.Background()).Exec("alter table users add email VARCHAR(255)").Error
		return err
	})
}

// 有超时时间
func (u *UserDAO) AddColumnWithTimeout() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.WithContext(ctx).Exec("alter table users add email VARCHAR(255)").Error
		return err
	})
}

// 清理表
func (u *UserDAO) ClearTable() error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		err := tx.WithContext(context.Background()).Exec("DROP TABLE IF EXISTS `users`").Error
		return err
	})
}
