package case33

import (
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"interview-cases/test"
	"testing"
	"time"
)

type Case33TestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *Case33TestSuite) SetupSuite() {
	s.db = test.InitDB()
	require.NoError(s.T(), s.db.AutoMigrate(&User{}, &Order{}))
	// 初始化测试数据
	require.NoError(s.T(), InitializeData(s.db))
}

func (s *Case33TestSuite) TearDownSuite() {
	_ = s.db.Exec("DROP Table users")
	_ = s.db.Exec("DROP Table orders")
}

func (s *Case33TestSuite) TestQueryPerformance() {
	// 测试原始查询性能
	start := time.Now()
	_, err := GetUserTotalsV1(s.db)
	durationV1 := time.Since(start)
	require.NoError(s.T(), err)

	// 创建优化索引
	CreateUserIdx(s.db)

	// 测试优化后查询性能
	start = time.Now()
	_, err = GetUserTotalsV2(s.db)
	durationV2 := time.Since(start)
	require.NoError(s.T(), err)

	// 输出性能对比
	s.T().Logf("V1查询耗时: %v", durationV1)
	s.T().Logf("V2查询耗时: %v", durationV2)
	s.T().Logf("性能提升: %.2f%%",
		float64(durationV1.Nanoseconds()-durationV2.Nanoseconds())/float64(durationV1.Nanoseconds())*100)
}

func TestCase33(t *testing.T) {
	suite.Run(t, new(Case33TestSuite))
}
