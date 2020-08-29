package etcdsdk

import (
	"context"
	"errors"
	"fmt"
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
	mutex     sync.RWMutex

	client    *clientv3.Client
	timeout   time.Duration

	session   *dlock.Session
	mlock     map[string]*dlock.Mutex
}

type BaseAPI interface {
	NewLeaseID(ttl int) (int64, error)
	KeepLease(leaseID int64) error

	PutWithLease(kv KeyValue, leaseID int64) (*KeyValue, error)
	PutWithTTL(kv KeyValue, ttl int) (*KeyValue, error)
	Put(kv KeyValue) (*KeyValue, error)

	Get(key string) (*KeyValue, error)
	GetWithChild(key string) ([]KeyValue, error)

	Del(key string) error
	Watch(key string) <-chan WatchEvent

	Lock(ctx context.Context, key string) error
	Ulock(ctx context.Context, key string) error
}

func ClientInit(timeout int, endpoints []string) (*Client, error) {
	config := clientv3.Config{
		Endpoints: endpoints,
		DialTimeout: 3*time.Duration(timeout)*time.Second,
	}

	cli, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}

	client := new(Client)
	client.client = cli
	client.session, err = dlock.NewSession(cli,  dlock.WithTTL(timeout))
	if err != nil {
		return nil, err
	}

	client.mlock = make(map[string] *dlock.Mutex, 1024)
	client.timeout = time.Duration(timeout)*time.Second

	return client, nil
}

func (this *Client)put(kv KeyValue, leaseID clientv3.LeaseID) (*KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	var opt []clientv3.OpOption
	if leaseID != -1 {
		opt = append(opt, clientv3.WithLease(leaseID))
	}
	rsp, err := this.client.Put(ctx, kv.Key, string(kv.Value), opt...)
	cancel()
	if  err != nil {
		return nil, err
	}
	return &KeyValue{
		Key: string(rsp.PrevKv.Key),
		Value: rsp.PrevKv.Value,
		Version: rsp.PrevKv.ModRevision,
	}, nil
}

func (this *Client)putCmp(kv KeyValue, leaseID clientv3.LeaseID) (*KeyValue, error) {
	var opt []clientv3.OpOption
	if leaseID != -1 {
		opt = append(opt, clientv3.WithLease(leaseID))
	}

	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	cmp := clientv3.Compare(clientv3.ModRevision(kv.Key), "=", kv.Version)
	put := clientv3.OpPut(kv.Key, string(kv.Value), opt...)
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
		Key: string(prekv.Key),
		Value: prekv.Value,
		Version: prekv.ModRevision,
	},fmt.Errorf("version is old")
}

func (this *Client)Put(kv KeyValue) (*KeyValue, error) {
	if kv.Version == -1 {
		return this.put(kv, -1)
	}
	return this.putCmp(kv, -1)
}

func (this *Client)PutWithTTL(kv KeyValue, ttl int) (*KeyValue, error) {
	leaseID, err := this.NewLeaseID(ttl)
	if err != nil {
		return nil, err
	}
	if kv.Version == -1 {
		return this.put(kv, clientv3.LeaseID(leaseID))
	}
	return this.putCmp(kv, clientv3.LeaseID(leaseID))
}

func (this *Client)NewLeaseID(ttl int) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	rsp, err := this.client.Grant(ctx, int64(ttl))
	cancel()
	if err != nil {
		return 0, err
	}
	return int64(rsp.ID), nil
}

func (this *Client)KeepLeaseID(leaseID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	_, err := this.client.KeepAliveOnce(ctx, clientv3.LeaseID(leaseID))
	cancel()
	return err
}

func (this *Client)PutWithLease(kv KeyValue, leaseID int64) (*KeyValue, error) {
	if kv.Version == -1 {
		return this.put(kv, clientv3.LeaseID(leaseID))
	}
	return this.putCmp(kv, clientv3.LeaseID(leaseID))
}

func (this *Client)Get(key string) (*KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	rsp, err := this.client.Get(ctx, key)
	cancel()
	if err != nil {
		return nil,err
	}
	if len(rsp.Kvs) == 0 {
		return nil, errors.New(key + " not found")
	}
	return &KeyValue{
		Key: string(rsp.Kvs[0].Key),
		Value: rsp.Kvs[0].Value,
		Version: rsp.Kvs[0].ModRevision}, nil
}

func (this *Client)GetWithChild(key string) ([]KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	rsp, err := this.client.Get(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, errors.New(key + " not found")
	}
	var kvList []KeyValue
	for _, v := range rsp.Kvs {
		kvList = append(kvList, KeyValue{
			Key: string(v.Key),
			Value: v.Value,
			Version: v.ModRevision})
	}
	return kvList, nil
}

func (this *Client)Del(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	_, err := this.client.Delete(ctx, key, clientv3.WithPrefix())
	cancel()
	if  err != nil {
		return err
	}
	return nil
}

func (this *Client)watchPutEvent(event *clientv3.Event) *WatchEvent {
	output := new(WatchEvent)
	output.Key = string(event.Kv.Key)
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
	output.Key = string(event.Kv.Key)
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
		context.Background(), key,
		clientv3.WithPrefix(),
		clientv3.WithPrevKV())
	go this.watchTask(watchchan, watchqueue)
	return watchqueue
}

func (this *Client)getLock(key string) *dlock.Mutex {
	this.mutex.RLock()
	mlock, b := this.mlock[key]
	this.mutex.RUnlock()

	if b == true {
		return mlock
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	mlock = dlock.NewMutex(this.session, key)
	gstMLock.m[key] = mlock
	return mlock
}

func (this *Client)Ulock(ctx context.Context, key string) error {
	mlock := this.getLock(key)
	return mlock.Unlock(ctx)
}

func (this *Client)Lock(ctx context.Context, key string) error {
	mlock := this.getLock(key)
	return mlock.Lock(ctx)
}