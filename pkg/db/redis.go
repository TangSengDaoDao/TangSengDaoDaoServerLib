package db

import "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/redis"

func NewRedis(addr string, password string, db ...int) *redis.Conn {
	return redis.New(addr, password, db...)
}
