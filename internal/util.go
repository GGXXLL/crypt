package internal

import (
	"bytes"
	"crypto/md5"
	"sync"

	"github.com/DoNewsCode/crypt/backend"
)

func GenMD5(b []byte) []byte {
	h := md5.New()
	h.Write(b)
	return h.Sum(nil)
}

func WatchCache(cache *sync.Map, key string, value []byte, respChan chan *backend.Response) {
	if h, ok := cache.Load(key); ok {
		if !bytes.Equal(GenMD5(value), h.([]byte)) {
			respChan <- &backend.Response{Value: value}
		}
	} else {
		cache.Store(key, value)
		respChan <- &backend.Response{Value: value}
	}
}
