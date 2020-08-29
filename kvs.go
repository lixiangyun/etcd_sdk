package etcdsdk

import (
	"context"
	"errors"
	"log"

	v3 "github.com/coreos/etcd/clientv3"

	mvcc "github.com/coreos/etcd/mvcc/mvccpb"
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

func keyvaluePath(key string) string {
	return publicKvsPrefix + key
}

func KeyValuePut(key string, value string) error {

	key = keyvaluePath(key)

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

	key = keyvaluePath(key)

	ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	_, err = Call().Put(ctx, key, value, v3.WithLease(resp.ID))
	cancel()
	if err != nil {
		return err
	}

	return nil
}

func KeyValueGet(key string) (string, error) {

	key = keyvaluePath(key)

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

	key = keyvaluePath(key)

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
		kv := KeyValue{Key: string(v.Key[len(publicKvsPrefix):]), Value: string(v.Value)}
		kvs = append(kvs, kv)
	}

	return kvs, nil
}

func KeyValueWatch(ctx context.Context, key string) <-chan KvWatchRsq {

	var act EVENT_TYPE
	var value string

	watchrsq := make(chan KvWatchRsq, 100)
	key = keyvaluePath(key)

	wch := Call().Watch(ctx, key, v3.WithPrefix(), v3.WithPrevKV())

	go func() {

		select {
		case wrsp := <-wch:
			{
				for _, event := range wrsp.Events {

					switch event.Type {
					case mvcc.PUT:
						{
							key = string(event.Kv.Key[len(publicKvsPrefix):])
							value = string(event.Kv.Value)
							if event.Kv.Version == 1 {
								act = EVENT_ADD
							} else {
								act = EVENT_UPDATE
							}
						}

					case mvcc.DELETE:
						{
							if event.PrevKv == nil {
								log.Println("prev kv is not exist!")
								continue
							}

							act = EVENT_DELETE
							key = string(event.Kv.Key[len(publicKvsPrefix):])
							value = string(event.Kv.Value)
							lease := event.PrevKv.Lease

							if lease == 0 {
								break
							}

							ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
							resp, err := Call().TimeToLive(ctx, v3.LeaseID(lease))
							cancel()

							if err != nil {
								break
							}

							if resp.TTL == -1 {
								act = EVENT_EXPIRE
							}
						}
					default:
						continue
					}

					watchrsq <- KvWatchRsq{Act: act, Key: key, Value: string(value)}
				}
			}
		case <-ctx.Done():
			return
		}
	}()

	return watchrsq
}
