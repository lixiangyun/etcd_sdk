package main

import (
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
}
