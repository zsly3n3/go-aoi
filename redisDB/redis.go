package redisDB

import (
	"github.com/go-ini/ini"
	"github.com/go-redis/redis"
	"log"
	"strings"
)

var Redis *redis.Client

// 数据库初始化
func InitRedis(cfg *ini.File) *redis.Client {
	rconf := cfg.Section(`redis`)
	addrs := strings.Split(rconf.Key(`addrs`).String(), `,`)
	index := rconf.Key(`index`).MustInt(0)
	client := redis.NewClient(&redis.Options{
		Addr:     addrs[0],
		Password: rconf.Key(`password`).String(),
		DB:       index,
		PoolSize: rconf.Key(`poolsize`).MustInt(100),
	})
	if err := client.Ping().Err(); err != nil {
		log.Fatal("init redisDB err,", err)
	}
	return client
}
