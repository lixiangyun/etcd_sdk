package etcdsdk

import (
	"fmt"
	"math/rand"
	"time"
)

func UUID() string {
	return fmt.Sprintf("%x", rand.Uint64())
}

func TimestampGet() string {
	return time.Now().Format(time.RFC1123)
}

