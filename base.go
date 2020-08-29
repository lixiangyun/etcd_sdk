package etcdsdk

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	mvcc "github.com/coreos/etcd/mvcc/mvccpb"
	dlock "github.com/coreos/etcd/clientv3/concurrency"
)

type EVENT_TYPE int

const (
	_ EVENT_TYPE = iota
	EVENT_ADD
	EVENT_UPDATE
	EVENT_DELETE
	EVENT_EXPIRE
)

type KeyValue struct {
	Key string
	Value []byte
	Version int64
}

type WatchEvent struct {
	Event   EVENT_TYPE
	KeyValue
}

type Client struct {
	sync.RWMutex

	client    *clientv3.Client
	timeout   time.Duration

	prefix    string
	prefixlen int

	session   *dlock.Session
	mlock      map[string]*dlock.Mutex
}

type BaseAPI interface {
	NewLeaseID(ttl int64) (int64, error)
	KeepLease(leaseID int64) error

	Lock(key string) error
	Ulock(key string) error

	PutWithLease(kv KeyValue, leaseID int64) error
	Put(kv KeyValue) error
	PutWithTTL(kv KeyValue, ttl int) error
	Get(key string) ( *KeyValue, error)
	GetWithChild(key string) ([]KeyValue, error)
	Del(key string) error
	Watch(key string) <-chan WatchEvent
}

func EtcdInit(timeout time.Duration, endpoints []string, perfix string) (BaseAPI, error) {
	config := clientv3.Config{
		Endpoints: endpoints,
		DialTimeout: 3*timeout,
	}

	cli, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(perfix, "/") == false {
		perfix = perfix + "/"
	}

	client := new(Client)
	client.client = cli
	client.session, err = dlock.NewSession(cli)
	if err != nil {
		return nil, err
	}
	client.mlock = make(map[string] *dlock.Mutex, 1024)
	client.timeout = timeout
	client.prefix = perfix
	client.prefixlen = len(perfix)

	return client, nil
}

func (this *Client)makePath(key string) string {
	return this.prefix + key
}

func (this *Client)catPath(rawkey []byte) string {
	return string(rawkey[this.prefixlen:])
}

func (this *Client)Put(kv KeyValue) (*KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	if kv.Version == -1 {
		rsp, err := this.client.Put(ctx, this.makePath(kv.Key), string(kv.Value))
		cancel()
		if  err != nil {
			return nil, err
		}
		return &KeyValue{
			Key: this.catPath(rsp.PrevKv.Key),
			Value: rsp.PrevKv.Value,
			Version: rsp.PrevKv.ModRevision,
		},nil
	} else {
		cmp := clientv3.Compare(clientv3.ModRevision(kv.Key), "=", kv.Version)
		put := clientv3.OpPut(kv.Key, string(kv.Value))
		get := clientv3.OpGet(kv.Key)

		txn, err := this.client.Txn(ctx).If(cmp).Then(put).Else(get).Commit()
		cancel()
		if  err != nil {
			return nil, err
		}
		if txn.Succeeded {
			kv.Version = txn.Header.GetRevision()
			return &kv, nil
		}
		prekv := txn.Responses[0].GetResponseRange().Kvs[0]
		return &KeyValue{
			Key: this.catPath(prekv.Key),
			Value: prekv.Value,
			Version: prekv.ModRevision,
		},fmt.Errorf("version is not new")
	}
}

func (this *Client)PutWithTTL(kv KeyValue, ttl int) (*KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	resp, err := this.client.Grant(ctx, int64(ttl))
	cancel()
	if err != nil {
		return nil, err
	}

	ctx, cancel = context.WithTimeout(context.Background(), this.timeout)
	if kv.Version == -1 {
		rsp, err := this.client.Put(ctx, this.makePath(kv.Key),
			string(kv.Value), clientv3.WithLease(resp.ID))
		cancel()
		if  err != nil {
			return nil, err
		}
		return &KeyValue{
			Key: this.catPath(rsp.PrevKv.Key),
			Value: rsp.PrevKv.Value,
			Version: rsp.PrevKv.ModRevision,
		},nil
	} else {
		cmp := clientv3.Compare(clientv3.ModRevision(this.makePath(kv.Key)), "=", kv.Version)
		put := clientv3.OpPut(this.makePath(kv.Key), string(kv.Value), clientv3.WithLease(resp.ID))
		get := clientv3.OpGet(this.makePath(kv.Key))

		txn, err := this.client.Txn(ctx).If(cmp).Then(put).Else(get).Commit()
		cancel()
		if  err != nil {
			return nil, err
		}
		if txn.Succeeded {
			kv.Version = txn.Header.GetRevision()
			return &kv, nil
		}
		prekv := txn.Responses[0].GetResponseRange().Kvs[0]
		return &KeyValue{
			Key: this.catPath(prekv.Key),
			Value: prekv.Value,
			Version: prekv.ModRevision,
		},fmt.Errorf("version is not new")
	}
}

func (this *Client)NewLeaseID(ttl int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	rsp, err := this.client.Grant(ctx, ttl)
	cancel()
	if err != nil {
		return 0, err
	}
	return int64(rsp.ID), nil
}

func (this *Client)KeepLease(leaseID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	_, err := this.client.KeepAliveOnce(ctx, clientv3.LeaseID(leaseID))
	cancel()
	return err
}

func (this *Client)PutWithLease(kv KeyValue, leaseID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	_, err := this.client.Put(ctx, this.makePath(key), string(value),
		clientv3.WithLease(clientv3.LeaseID(leaseID)))
	cancel()
	if  err != nil {
		return err
	}
	return nil
}

func (this *Client)Get(key string) (*KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	rsp, err := this.client.Get(ctx, this.makePath(key))
	cancel()
	if err != nil {
		return nil,err
	}
	if len(rsp.Kvs) == 0 {
		return nil, errors.New(key+" not found")
	}
	return &KeyValue{
		Key: this.catPath(rsp.Kvs[0].Key),
		Value: rsp.Kvs[0].Value,
		Version: rsp.Kvs[0].ModRevision}, nil
}

func (this *Client)GetWithChild(key string) ([]KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	rsp, err := this.client.Get(ctx, this.makePath(key), clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil,err
	}
	if len(rsp.Kvs) == 0 {
		return nil, errors.New(key+" not found")
	}
	var kvList []KeyValue
	for _, v := range rsp.Kvs {
		kvList = append(kvList, KeyValue{
			Key: this.catPath(v.Key),
			Value: v.Value,
			Version: v.ModRevision})
	}
	return kvList, nil
}

func (this *Client)Del(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	_, err := this.client.Delete(ctx, this.makePath(key), clientv3.WithPrefix())
	cancel()
	if  err != nil {
		return err
	}
	return nil
}

func (this *Client)watchPutEvent(event *clientv3.Event) *WatchEvent {
	output := new(WatchEvent)
	output.Key = this.catPath(event.Kv.Key)
	output.Value = event.Kv.Value
	output.Version = event.Kv.ModRevision
	if event.Kv.Version == 1 {
		output.Event = EVENT_ADD
	} else {
		output.Event = EVENT_UPDATE
	}
	return output
}

func (this *Client)watchDelEvent(event *clientv3.Event) *WatchEvent {
	if event.PrevKv == nil {
		return nil
	}
	output := new(WatchEvent)
	output.Key = this.catPath(event.Kv.Key)
	output.Value = event.Kv.Value
	output.Version = event.Kv.ModRevision
	output.Event = EVENT_DELETE

	lease := event.PrevKv.Lease
	if lease == 0 {
		return output
	}

	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	resp, err := this.client.TimeToLive(ctx, clientv3.LeaseID(lease))
	cancel()
	if err != nil {
		return output
	}
	if resp.TTL == -1 {
		output.Event = EVENT_EXPIRE
	}
	return output
}

func (this *Client)watchTask(watchchan clientv3.WatchChan, queue chan WatchEvent)  {
	for  {
		wrsp := <-watchchan
		for _, event := range wrsp.Events {
			var result *WatchEvent
			if event.Type == mvcc.PUT {
				result = this.watchPutEvent(event)
			}else if event.Type == mvcc.DELETE {
				result = this.watchDelEvent(event)
			}
			if result == nil {
				continue
			}
			queue <- *result
		}
	}
}

func (this *Client)Watch(key string) <-chan WatchEvent {
	watchqueue := make(chan WatchEvent, 100)
	watchchan := this.client.Watch(
		context.Background(),
		this.makePath(key),
		clientv3.WithPrefix(),
		clientv3.WithPrevKV())
	go this.watchTask(watchchan, watchqueue)
	return watchqueue
}