package etcdsdk

import (
	"context"
	"errors"
	"sync"

	dlock "github.com/coreos/etcd/clientv3/concurrency"
)

const (
	publicLockPrefix = "/service/lock/"
)

type serviceLockMap struct {
	s *dlock.Session
	m map[string]*dlock.Mutex
	sync.RWMutex
}

var gstMLock = serviceLockMap{
	m: make(map[string]*dlock.Mutex),
}

func ServiceLockInit() error {
	var err error
	gstMLock.s, err = dlock.NewSession(Call())
	if err != nil {
		return err
	}
	return nil
}

func getMutexlock(name string) *dlock.Mutex {

	gstMLock.RLock()
	mlock, b := gstMLock.m[name]
	gstMLock.RUnlock()

	if b == true {
		return mlock
	}

	gstMLock.Lock()
	defer gstMLock.Unlock()

	m := dlock.NewMutex(gstMLock.s, publicLockPrefix+name)
	if m == nil {
		return nil
	}

	gstMLock.m[name] = m
	return m
}

func ServiceLock(ctx context.Context, name string) error {
	m := getMutexlock(name)
	if m == nil {
		return errors.New("create lock failed!")
	}
	return m.Lock(ctx)
}

func ServiceUnlock(ctx context.Context, name string) error {
	gstMLock.RLock()
	m, b := gstMLock.m[name]
	gstMLock.RUnlock()

	if b == false {
		return errors.New("lock is not exist!")
	}
	return m.Unlock(ctx)
}
