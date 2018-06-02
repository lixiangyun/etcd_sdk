package etcdsdk

import (
	"context"
	"errors"

	v3 "github.com/coreos/etcd/clientv3"
)

const (
	publicKvsPrefix = "/service/kvs/"
)

type KeyValue struct {
	key   string
	value string
}

func KeyValuePut(kv KeyValue) error {

	key := publicKvsPrefix + kv.key

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	_, err := Call().Put(ctx, key, kv.value)
	cancel()
	if err != nil {
		return err
	}
	return nil
}

func KeyValuePutWithTTL(kv KeyValue, ttl int64) error {

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	resp, err := Call().Grant(ctx, ttl)
	cancel()
	if err != nil {
		return err
	}

	key := publicKvsPrefix + kv.key

	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	_, err = Call().Put(ctx, key, kv.value, v3.WithLease(resp.ID))
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func KeyValueGet(key string) (string, error) {

	key = publicKvsPrefix + key

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	resp, err := Call().Get(ctx, key)
	cancel()
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", errors.New("have not found key/value!")
	}

	return string(resp.Kvs[0].Value), nil
}

func KeyValueGetWithChild(key string) ([]KeyValue, error) {

	return nil, nil
}
