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
	Key   string
	Value string
}

type KvWatchRsq struct {
	Act   EVENT_TYPE
	Key   string
	Value string
}

func KeyValuePut(key string, value string) error {

	key = publicKvsPrefix + key

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	_, err := Call().Put(ctx, key, value)
	cancel()
	if err != nil {
		return err
	}
	return nil
}

func KeyValuePutWithTTL(key string, value string, ttl int64) error {

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	resp, err := Call().Grant(ctx, ttl)
	cancel()
	if err != nil {
		return err
	}

	key = publicKvsPrefix + key

	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	_, err = Call().Put(ctx, key, value, v3.WithLease(resp.ID))
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

	key = publicKvsPrefix + key

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	resp, err := Call().Get(ctx, key, v3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, errors.New("have not found key/value!")
	}

	var kvs []KeyValue

	for _, v := range resp.Kvs {
		kv := KeyValue{Key: string(v.Key), Value: string(v.Value)}
		kvs = append(kvs, kv)
	}

	return kvs, nil
}

func KeyValueWatch(key string) <-chan KvWatchRsq {

	return nil
}
