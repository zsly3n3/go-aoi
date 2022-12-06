package main

import (
	"fmt"
	"github.com/go-ini/ini"
	"go-aoi/redisDB"
	"log"
	"os"
)

func main() {
	client := redisDB.InitRedis(LoadConfig())
	redisDB.Redis = client
}

func LoadConfig() *ini.File {
	pwd, _ := os.Getwd()
	cfg, err := ini.Load(fmt.Sprintf(`%s/tmp/test.ini`, pwd))
	if err != nil {
		log.Fatalln(err)
	}
	return cfg
}
