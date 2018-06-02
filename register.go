package etcdsdk

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	v3 "github.com/coreos/etcd/clientv3"
	mvcc "github.com/coreos/etcd/mvcc/mvccpb"
)

const (
	publicRegPrefix = "/service/register/"

	defaultTTL     = 5
	defaultTimeout = 3 * time.Second
)

type InstanceID string

type EVENT_TYPE int

const (
	_ EVENT_TYPE = iota
	EVENT_ADD
	EVENT_UPDATE
	EVENT_DELETE
	EVENT_EXPIRE
)

type WatchRsqonse struct {
	Act  EVENT_TYPE
	Name string
	Inst Instance
}

type Instance struct {
	ID        InstanceID `json:"instanceid"`
	Timestamp string     `json:"timestamp"`
	Endpoints []string   `json:"endpoints"`
	Status    int        `json:"status"`

	modversion int64
}

type Service struct {
	Name      string `json:"servicename"`
	Timestamp string `json:"timestamp"`
}

type InstanceCtrl struct {
	inst  *Instance
	svc   *Service
	ttl   int
	lease v3.LeaseID
	stop  chan struct{}
	sync.RWMutex
}

type ServiceMap struct {
	i map[InstanceID]*InstanceCtrl
	sync.RWMutex
}

var gServiceMap = &ServiceMap{
	i: make(map[InstanceID]*InstanceCtrl),
}

func servicePath(name string) string {
	return publicRegPrefix + name
}

func serviceInstancePath(name string, id InstanceID) string {
	return publicRegPrefix + name + "/" + string(id)
}

func NewInstanceID() InstanceID {
	return InstanceID(UUID())
}

func keepalive(ctrl *InstanceCtrl) {
	var trycnt int

	for {

		ctrl.RLock()
		ttl := ctrl.ttl
		lease := ctrl.lease
		ctrl.RUnlock()

		select {
		case <-time.After(time.Duration(ttl) * time.Second / 3):
			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			_, err := Call().KeepAliveOnce(ctx, lease)
			cancel()

			if err != nil {
				trycnt++
				if trycnt > 3 {
					log.Println("instance " + ctrl.inst.ID + " heartbeat fail!")
					return
				}
				continue
			}
			trycnt = 0

		case <-ctrl.stop:
			{
				log.Println("instance " + ctrl.inst.ID + " is stop to heartbeat!")
				return
			}
		}
	}
}

func ServcieRegister(name string, endpoints []string) (InstanceID, error) {
	svc := &Service{Name: name, Timestamp: TimestampGet()}

	key := servicePath(name)
	value, err := json.Marshal(svc)
	if err != nil {
		return "", err
	}

	cmp := v3.Compare(v3.CreateRevision(key), "=", 0)
	put := v3.OpPut(key, string(value))

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	resp, err := Call().Txn(ctx).If(cmp).Then(put).Commit()
	cancel()
	if err != nil {
		return "", err
	}

	if resp.Succeeded {
		log.Println("register service " + name + "success!")
	}

	for {
		inst := &Instance{ID: NewInstanceID(), Timestamp: TimestampGet(), Endpoints: endpoints, Status: 0}

		key = serviceInstancePath(name, inst.ID)
		value, err := json.Marshal(inst)
		if err != nil {
			return "", err
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		rasp, err := Call().Grant(ctx, int64(defaultTTL))
		cancel()
		if err != nil {
			return "", err
		}

		cmp = v3.Compare(v3.CreateRevision(key), "=", 0)
		put = v3.OpPut(key, string(value), v3.WithLease(rasp.ID))

		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		resp, err = Call().Txn(ctx).If(cmp).Then(put).Commit()
		cancel()
		if err != nil {
			return "", err
		}

		if resp.Succeeded {

			inst.modversion = resp.Header.GetRevision()

			instctrl := new(InstanceCtrl)
			instctrl.inst = inst
			instctrl.lease = rasp.ID
			instctrl.svc = svc
			instctrl.ttl = defaultTTL
			instctrl.stop = make(chan struct{}, 1)

			gServiceMap.Lock()
			gServiceMap.i[inst.ID] = instctrl
			gServiceMap.Unlock()

			go keepalive(instctrl)

			log.Println("register instance  " + inst.ID + "success!")

			return inst.ID, nil
		}
	}
}

func instanceUpdate(ctrl *InstanceCtrl) (*Instance, error) {

	key := serviceInstancePath(ctrl.svc.Name, ctrl.inst.ID)

	value, err := json.Marshal(ctrl.inst)
	if err != nil {
		return nil, err
	}

	cmp := v3.Compare(v3.ModRevision(key), "=", ctrl.inst.modversion)
	put := v3.OpPut(key, string(value), v3.WithLease(ctrl.lease))
	get := v3.OpGet(key)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	resp, err := Call().Txn(ctx).If(cmp).Then(put).Else(get).Commit()
	cancel()
	if err != nil {
		return nil, err
	}

	if resp.Succeeded {
		ctrl.inst.modversion = resp.Header.GetRevision()
		log.Println("update instance " + ctrl.inst.ID + " success!")
		return nil, nil
	}

	newinst := new(Instance)
	err = json.Unmarshal(resp.Responses[0].GetResponseRange().Kvs[0].Value, newinst)
	if err != nil {
		return nil, err
	}

	newinst.modversion = resp.Responses[0].GetResponseRange().Kvs[0].ModRevision

	return newinst, nil
}

func ServiceStatusUpdate(id InstanceID, status int) error {

	gServiceMap.RLock()
	instctrl, b := gServiceMap.i[id]
	gServiceMap.RUnlock()

	if b == false {
		return errors.New("instance " + string(id) + " is not exist!")
	}

	instctrl.Lock()
	defer instctrl.Unlock()

	for {
		instctrl.inst.Status = status
		instctrl.inst.Timestamp = TimestampGet()

		inst, err := instanceUpdate(instctrl)
		if err != nil {
			return err
		}

		if inst != nil {
			*(instctrl.inst) = *inst
		} else {
			return nil
		}
	}
}

func ServiceQuery(name string) ([]Instance, error) {

	key := servicePath(name)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	resp, err := Call().Get(ctx, key, v3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}

	var insts []Instance

	for _, kv := range resp.Kvs {
		inst := new(Instance)
		err := json.Unmarshal(kv.Value, inst)
		if err != nil || len(inst.ID) == 0 {
			continue
		}
		insts = append(insts, *inst)
	}

	return insts, nil
}

func ServiceWatch(name string) <-chan WatchRsqonse {

	wtrspch := make(chan WatchRsqonse, 100)

	key := servicePath(name)

	ctx, _ := context.WithTimeout(context.Background(), defaultTimeout)
	wch := Call().Watch(ctx, key, v3.WithPrefix(), v3.WithPrevKV())

	go func() {
		for wrsp := range wch {
			for _, event := range wrsp.Events {

				var act EVENT_TYPE
				var value []byte

				switch event.Type {
				case mvcc.PUT:
					{
						value = event.Kv.Value
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
						value = event.PrevKv.Value
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

				var inst Instance
				err := json.Unmarshal(value, &inst)
				if err != nil {
					continue
				}

				wtrspch <- WatchRsqonse{Act: act, Name: name, Inst: inst}
			}
		}

	}()

	return wtrspch
}

func ServiceDelete(id InstanceID) error {

	gServiceMap.Lock()
	defer gServiceMap.Unlock()

	instctrl, b := gServiceMap.i[id]
	if b == false {
		return errors.New("instance " + string(id) + " is not exist!")
	}

	instctrl.stop <- struct{}{}

	instctrl.Lock()
	defer instctrl.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	_, err := Call().Revoke(ctx, v3.LeaseID(instctrl.lease))
	cancel()
	if err != nil {
		return err
	}

	delete(gServiceMap.i, id)
	return nil
}
