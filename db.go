package etcdsdk

type TableAPI interface {

}

type DBAPI interface {

	NewTable(name string) error
	DelTable(name string) error
	WatchTable(name string) error

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


func NewDB() DBAPI {

}

