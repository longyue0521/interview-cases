package case30

import (
	"github.com/stretchr/testify/require"
	"interview-cases/test"
	"testing"
)

// 复现表锁的案例
func TestMetaDataLock(t *testing.T) {
	db := test.InitDB()
	uDao := NewUserDAO(db)
	err := uDao.ClearTable()
	require.NoError(t, err)
	InitTable(db)
	err = InitData(db)
	require.NoError(t, err)
	uDao.MetaDataLock()
}

// 加上超时时间
func TestMetaDataLockWithTimeout(t *testing.T) {
	db := test.InitDB()
	uDao := NewUserDAO(db)
	err := uDao.ClearTable()
	require.NoError(t, err)
	InitTable(db)
	err = InitData(db)
	require.NoError(t, err)
	uDao.MetaDataLockWithTimeout()
}
