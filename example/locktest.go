package main

import (
	"context"
	"log"
	"time"

	"github.com/lixiangyun/etcdsdk"
)

func lock_test1() {

	ctx, cancel := context.WithCancel(context.Background())
	err := etcdsdk.ServiceLock(ctx, "123")
	if err != nil {
		log.Println(err.Error())
	}
	time.Sleep(time.Second * 2)

	err = etcdsdk.ServiceUnlock(ctx, "123")
	if err != nil {
		log.Println(err.Error())
	}

	cancel()
}
