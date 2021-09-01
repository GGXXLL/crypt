package consul

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ggxxll/crypt/backend"
	"github.com/ggxxll/crypt/internal"
	"github.com/hashicorp/consul/api"
)

type Client struct {
	client        *api.KV
	cache         *sync.Map
	watchInterval time.Duration
}

type OptionFunc func(client *Client)

func WithWatchInterval(duration time.Duration) OptionFunc {
	return func(client *Client) {
		client.watchInterval = duration
	}
}

func New(machines []string, opts ...OptionFunc) (*Client, error) {
	conf := api.DefaultConfig()
	if len(machines) > 0 {
		conf.Address = machines[0]
	}
	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}

	cli := &Client{client: client.KV(), cache: &sync.Map{}, watchInterval: 10 * time.Second}
	for _, opt := range opts {
		opt(cli)
	}

	return cli, nil
}

func (c *Client) Get(_ context.Context, key string) ([]byte, error) {
	kv, _, err := c.client.Get(key, nil)
	if err != nil {
		return nil, err
	}
	if kv == nil {
		return nil, fmt.Errorf("key ( %s ) was not found", key)
	}
	return kv.Value, nil
}

func (c *Client) Set(_ context.Context, key string, value []byte) error {
	key = strings.TrimPrefix(key, "/")
	kv := &api.KVPair{
		Key:   key,
		Value: value,
	}
	_, err := c.client.Put(kv, nil)
	return err
}

func (c *Client) Watch(ctx context.Context, key string) <-chan *backend.Response {
	respChan := make(chan *backend.Response, 0)
	go func() {
		defer func() {
			close(respChan)
		}()
		for {
			select {
			case <-time.After(c.watchInterval):
				val, err := c.Get(ctx, key)
				if err != nil {
					respChan <- &backend.Response{Error: err}
					continue
				}
				internal.WatchCache(c.cache, key, val, respChan)
			case <-ctx.Done():
				respChan <- &backend.Response{Error: ctx.Err()}
				return
			}
		}
	}()
	return respChan
}
