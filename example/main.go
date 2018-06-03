package main

import (
	"log"

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

	svcreg_test01()
	kv_test01()
	lock_test1()
}
