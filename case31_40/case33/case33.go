package case33

import (
	"gorm.io/gorm"
	"math/rand"
	"time"
)

// 初始化用户和订单数据
func InitializeData(db *gorm.DB) error {
	rand.Seed(time.Now().UnixNano())

	var users []User
	for i := 1; i <= 20000; i++ {
		user := User{Name: "User" + string(i)}
		if err := db.Create(&user).Error; err != nil {
			return err // 如果不使用事务，则直接返回错误
		}
		users = append(users, user)

		// 每个用户有3到6个订单
		orderCount := rand.Intn(4) + 3 // 生成3到6之间的随机数
		for j := 0; j < orderCount; j++ {
			order := Order{UserID: user.ID, TotalAmount: rand.Intn(100)} // 随机生成TotalAmount
			if err := db.Create(&order).Error; err != nil {
				return err // 如果不使用事务，则直接返回错误
			}
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

func CreateUserIdx(db *gorm.DB) {
	err := db.Exec("CREATE INDEX IF NOT EXISTS idx_user_id  ON orders(user_id)").Error
	if err != nil {
		// 处理错误
	}
}

func GetUserTotalsV2(db *gorm.DB) ([]UserTotal, error) {
	var results []UserTotal

	query := `
        SELECT u.id AS user_id, u.name, COALESCE(o.total_amount, 0) AS total_amount
        FROM users u
        LEFT JOIN (
            SELECT user_id, SUM(total_amount) AS total_amount
            FROM orders
            GROUP BY user_id
        ) o ON u.id = o.user_id
    `

	// 使用Raw方法执行自定义SQL，并扫描到results
	err := db.Raw(query).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}