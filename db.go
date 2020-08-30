package etcdsdk

import "fmt"

type table struct {
	api   BaseAPI
	path *AbsPath
	table string
	queue chan WatchItem
}

type TableAPI interface {
	Insert(key string, value []byte) error
	Delete(key string) error
	Update(key string, value []byte) error

	Query() ([]KeyValue, error)
	QueryKey(key string) (*KeyValue, error)
	Watch() <- chan WatchItem
}

type WatchItem struct {
	Event   EVENT_TYPE
	KeyValue
}

type DBAPI interface {
	NewTable(name string) (TableAPI, error)
	DelTable(name string) error
}

type DBClient struct {
	api BaseAPI
	name  string
}

var ERR_PARAM_INVALID = fmt.Errorf("input param is invalid")

func NewDBInit(api BaseAPI, name string) (DBAPI, error) {
	kv := KeyValue{
		Key: fmt.Sprintf("/%s", name),
		Value: []byte(TimestampGet()),
		Version: 0,
	}
	_, err := api.Put(kv)
	if err != nil && err != ERR_NOT_NEWEST {
		return nil, err
	}
	return &DBClient{api: api, name: name}, nil
}

func (db *DBClient)NewTable(name string) (TableAPI, error) {
	tabpath := fmt.Sprintf("/%s/%s/", db.name, name)
	abs := NewAbsPath(tabpath)

	kv := KeyValue{
		Key: tabpath,
		Value: []byte(TimestampGet()),
		Version: 0,
	}
	_, err := db.api.Put(kv)
	if err != nil && err != ERR_NOT_NEWEST {
		return nil, err
	}

	tab := new(table)
	tab.table = tabpath
	tab.path = abs
	tab.api = db.api

	return tab, nil
}

func (db *DBClient)DelTable(name string) error  {
	return db.api.Del(fmt.Sprintf("/%s/%s/", db.name, name))
}

func (tab *table)Insert(key string, value []byte) error {
	if len(key) == 0 {
		return ERR_PARAM_INVALID
	}
	kv := KeyValue{Key: key, Value: value, Version: 0}
	tab.path.Coder(&kv)
	_, err := tab.api.Put(kv)
	return err
}

func (tab *table)Delete(key string) error {
	if len(key) == 0 {
		return ERR_PARAM_INVALID
	}
	return tab.api.Del(tab.path.CoderKey(key))
}

func (tab *table)Update(key string, value []byte) error {
	if len(key) == 0 {
		return ERR_PARAM_INVALID
	}
	kv := KeyValue{Key: key, Value: value, Version: -1}
	tab.path.Coder(&kv)
	_, err := tab.api.Put(kv)
	return err
}

func (tab *table)Query() ([]KeyValue, error) {
	kvs, err := tab.api.GetWithChild(tab.table)
	if err != nil {
		return nil, err
	}
	tab.path.ListDecoder(kvs)
	var output []KeyValue
	for _, v := range kvs {
		if len(v.Key) != 0 {
			output = append(output, v)
		}
	}
	return output, nil
}

func (tab *table)QueryKey(key string) (*KeyValue, error) {
	if len(key) == 0 {
		return nil, ERR_PARAM_INVALID
	}
	kv, err := tab.api.Get(tab.path.CoderKey(key))
	if err != nil {
		return nil, err
	}
	return tab.path.Decoder(kv), nil
}

func (tab *table)Watch() <- chan WatchItem {
	if tab.queue != nil {
		return tab.queue
	}
	tab.queue = make(chan WatchItem, 100)

	watchchan := tab.api.Watch(tab.table)
	if watchchan == nil {
		return nil
	}
	go func() {
		for  {
			event := <- watchchan
			event.Key = tab.path.DecoderKey(event.Key)
			tab.queue <- WatchItem{KeyValue: event.KeyValue, Event: event.Event}
		}
	}()
	return tab.queue
}

