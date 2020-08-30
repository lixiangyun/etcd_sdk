package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lixiangyun/etcdsdk"
)

var gBaseAPI etcdsdk.BaseAPI

func main() {
	baseAPI, err := etcdsdk.ClientInit(5, []string{"localhost:2379"} )
	if err != nil {
		log.Println(err.Error())
		return
	}

	gBaseAPI = baseAPI

	dbAPI, err := etcdsdk.NewDBInit(baseAPI, "db_test")
	if err != nil {
		log.Println(err.Error())
		return
	}

	tableAPI, err := dbAPI.NewTable("table_test_001")
	if err != nil {
		log.Println(err.Error())
		return
	}

	go func() {
		queue := tableAPI.Watch()
		for  {
			item := <- queue
			log.Println(item)
		}
	}()

	for i:=0 ; i<10; i++ {
		key := fmt.Sprintf("key_%d", i)
		err = tableAPI.Insert(key, []byte("value:1234"))
		if err != nil {
			log.Println(err.Error())
		}
		err = tableAPI.Update(key, []byte("value:1234"))
		if err != nil {
			log.Println(err.Error())
		}
	}

	list, _ := tableAPI.Query()

	log.Println(list)

	lock_test1()
}


func lock_test1() {
	ctx, cancel := context.WithCancel(context.Background())
	err := gBaseAPI.Lock(ctx, "123")
	if err != nil {
		log.Println(err.Error())
	}
	time.Sleep(time.Second * 2)

	err = gBaseAPI.Ulock(ctx, "123")
	if err != nil {
		log.Println(err.Error())
	}
	cancel()
}
