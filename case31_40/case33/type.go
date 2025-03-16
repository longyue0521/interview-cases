package case33

type User struct {
	ID   int `gorm:"primaryKey;autoIncrement"`
	Name string
}

type Order struct {
	ID          int `gorm:"primaryKey;autoIncrement"`
	UserID      int
	TotalAmount int
}

// 定义接收查询结果的结构体
type UserTotal struct {
	UserID      int    `json:"user_id"`
	Name        string `json:"name"`
	TotalAmount int    `json:"total_amount"`
}


