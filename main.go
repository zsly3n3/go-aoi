package main

import (
	"fmt"
	"github.com/go-ini/ini"
	"go-aoi/aoi_list"
	"go-aoi/redisDB"
	"log"
	"os"
)

func main() {
	client := redisDB.InitRedis(LoadConfig())
	redisDB.Redis = client
	list := aoi_list.Create(`test`, `xset`, `yset`)
	ent := new(aoi_list.Entity)
	ent.Radius = 2
	ent.UUID = `p4`
	ent.X = 4
	ent.Y = 5
	//arr, err := list.Add(ent)
	//if err != nil {
	//	log.Println(err.Error())
	//	return
	//}
	//log.Println(`before:`, arr)
	arr, err := list.Leave(ent)
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Println(`arr:`, arr)
}

func LoadConfig() *ini.File {
	pwd, _ := os.Getwd()
	cfg, err := ini.Load(fmt.Sprintf(`%s/tmp/test.ini`, pwd))
	if err != nil {
		log.Fatalln(err)
	}
	return cfg
}
