package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lixiangyun/etcdsdk"
)

func kv_test01() {

	t1 := time.Now()

	for i := 0; i < 1000; i++ {

		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)

		err := etcdsdk.KeyValuePut(key, value)
		if err != nil {
			log.Println(err.Error())
		}
	}

	t2 := time.Now().Sub(t1)
	t3 := float32(t2) / float32(time.Second)

	log.Printf("delay time : %.3f tps \r\n", float32(1000)/t3)
}
