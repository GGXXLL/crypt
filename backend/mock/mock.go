package mock

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/GGXXLL/crypt/backend"
	"github.com/GGXXLL/crypt/internal"
)

var (
	mockedStore map[string][]byte

	once = sync.Once{}
	lock = sync.RWMutex{}
)

type Client struct {
	cache *sync.Map
}

func New(_ []string) (*Client, error) {
	once.Do(func() {
		mockedStore = make(map[string][]byte, 2)
	})
	return &Client{cache: &sync.Map{}}, nil
}

func (c *Client) Get(_ context.Context, key string) ([]byte, error) {
	lock.RLock()
	defer lock.RUnlock()

	if v, ok := mockedStore[key]; ok {
		return v, nil
	}
	err := errors.New("Could not find key: " + key)
	return nil, err
}

func (c *Client) Set(_ context.Context, key string, value []byte) error {
	lock.Lock()
	defer lock.Unlock()

	mockedStore[key] = value
	return nil
}

func (c *Client) Watch(ctx context.Context, key string) <-chan *backend.Response {
	lock.RLock()
	defer lock.RUnlock()

	respChan := make(chan *backend.Response, 0)
	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				b, err := c.Get(ctx, key)
				if err != nil {
					respChan <- &backend.Response{Error: err}
					continue
				}
				internal.WatchCache(c.cache, key, b, respChan)

			case <-ctx.Done():
				respChan <- &backend.Response{Error: ctx.Err()}
				return

			}
		}
	}()
	return respChan
}
