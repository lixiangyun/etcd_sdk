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

	etcdsdk.ServiceDisconnect()
}
