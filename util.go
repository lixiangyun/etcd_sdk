package etcdsdk

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func UUID() string {
	return fmt.Sprintf("%x", rand.Uint64())
}

func TimestampGet() string {
	return time.Now().Format(time.RFC1123)
}

type AbsPath struct {
	prefix string
}

func NewAbsPath(prefix string) *AbsPath {
	if strings.HasPrefix(prefix, "/") == false {
		prefix = "/" + prefix
	}
	if strings.HasSuffix(prefix, "/") == false {
		prefix = prefix + "/"
	}
	return &AbsPath{prefix: prefix}
}

func (this *AbsPath)CoderKey(key string) string {
	key = this.prefix + key
	if strings.HasSuffix(key, "/") == false {
		key = key + "/"
	}
	return key
}

func (this *AbsPath)DecoderKey(key string) string {
	if 0 == strings.Index(key, this.prefix) {
		key = key[len(this.prefix):]
	}
	if strings.HasSuffix(key, "/") == true {
		key = key[:len(key)-1]
	}
	return key
}

func (this *AbsPath)Coder(kv *KeyValue) *KeyValue {
	kv.Key = this.CoderKey(kv.Key)
	return kv
}

func (this *AbsPath)Decoder(kv *KeyValue) *KeyValue {
	kv.Key = this.DecoderKey(kv.Key)
	return kv
}

func (this *AbsPath)ListCoder(kv []KeyValue) []KeyValue {
	for i:=0; i < len(kv); i++ {
		this.Coder(&kv[i])
	}
	return kv
}

func (this *AbsPath)ListDecoder(kv []KeyValue) []KeyValue {
	for i:=0; i < len(kv); i++ {
		this.Decoder(&kv[i])
	}
	return kv
}
