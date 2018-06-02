package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lixiangyun/etcdsdk"
)

func main() {

	endpoints := []string{"localhost:2379"}

	err := etcdsdk.ServiceConnect(endpoints)
	if err != nil {
		log.Fatalln(err.Error())
	}

	err = etcdsdk.ServiceLockInit()
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer etcdsdk.ServiceDisconnect()

	id, err := etcdsdk.ServcieRegister("demo1", []string{"127.0.0.1:8080"})
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	log.Println("instance : ", id)

	inst, err := etcdsdk.ServiceQuery("demo1")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}

	log.Println("total instance : ", len(inst))
	for _, v := range inst {
		log.Println("get instance : ", v)
	}

	watchch := etcdsdk.ServiceWatch("demo1")

	go func() {
		for v := range watchch {
			log.Println("watch instance : ", v)
		}
	}()

	time.Sleep(time.Second * 1)

	etcdsdk.ServiceStatusUpdate(id, 1)

	time.Sleep(time.Second * 1)

	etcdsdk.ServiceDelete(id)

	time.Sleep(time.Second * 1)

	inst, err = etcdsdk.ServiceQuery("demo1")
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	log.Println("total instance : ", len(inst))
	for _, v := range inst {
		log.Println("get instance : ", v)
	}

	ctx, cancel := context.WithCancel(context.Background())
	err = etcdsdk.ServiceLock(ctx, "123")
	if err != nil {
		log.Println(err.Error())
	}
	time.Sleep(time.Second * 2)

	err = etcdsdk.ServiceUnlock(ctx, "123")
	if err != nil {
		log.Println(err.Error())
	}

	cancel()

	t1 := time.Now()

	for i := 0; i < 1000; i++ {

		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)

		kv := etcdsdk.KeyValue{Key: key, Value: value}
		err := etcdsdk.KeyValuePut(kv)
		if err != nil {
			log.Println(err.Error())
		}
	}

	t2 := time.Now().Sub(t1)
	t3 := float32(t2) / float32(time.Second)

	log.Printf("delay time : %.3f tps \r\n", float32(1000)/t3)
}
