package etcdsdk

import (
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
)

type Instance struct {
	ID        string        `json:"instanceid"`
	Timestamp time.Duration `json:"timestamp"`
	Endpoints []string      `json:"endpoints"`
}

type Service struct {
	Name      string        `json:"servicename"`
	Timestamp time.Duration `json:"timestamp"`
}

func ServcieRegister(name string, endpoints []string) error {

}
