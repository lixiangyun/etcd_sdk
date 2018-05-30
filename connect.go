package etcdsdk

import (
	"context"
	"time"

	"github.com/coreos/etcd/clientv3"
)

type Connect struct {
	config clientv3.Config
	conn   *clientv3.Client
	cancel context.CancelFunc
}

const (
	defaultTTL     = 5
	defaultTimeout = 3 * time.Second
)

var gstConnect *Connect

func ServiceConnect(endpoints []string) error {
	conntmp := new(Connect)

	conntmp.config.DialTimeout = defaultTimeout
	conntmp.config.Endpoints = endpoints
	conntmp.config.Context, conntmp.cancel = context.WithCancel(context.Background())

	conn, err := clientv3.New(conntmp.config)
	if err != nil {
		return err
	}

	conntmp.conn = conn
	gstConnect = conntmp

	return nil
}

func ServiceDisconnect() {
	gstConnect.cancel()
	gstConnect.conn.Close()
}
