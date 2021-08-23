package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DoNewsCode/crypt/backend"
	"github.com/DoNewsCode/crypt/internal"
	"github.com/go-redis/redis/v8"
)

type Client struct {
	client        redis.UniversalClient
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
	newClient := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: machines,
	})
	if err := newClient.Ping(context.TODO()).Err(); err != nil {
		return nil, fmt.Errorf("creating new redis client for crypt.backend.Client: %v", err)
	}
	cli := &Client{client: newClient, cache: &sync.Map{}, watchInterval: 10 * time.Second}
	for _, opt := range opts {
		opt(cli)
	}

	return cli, nil
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return []byte(resp), nil
}

func (c *Client) Set(ctx context.Context, key string, value []byte) error {
	return c.client.Set(ctx, key, string(value), 0).Err()
}

func (c *Client) Watch(ctx context.Context, key string) <-chan *backend.Response {
	respChan := make(chan *backend.Response, 0)
	go func() {
		defer func() {
			close(respChan)
			c.client.Close()
		}()

		for {
			select {
			case <-time.After(c.watchInterval):
				res, err := c.Get(ctx, key)
				if err != nil {
					respChan <- &backend.Response{Error: err}
					continue
				}
				internal.WatchCache(c.cache, key, res, respChan)
			case <-ctx.Done():
				respChan <- &backend.Response{Error: ctx.Err()}
				return
			}
		}
	}()
	return respChan
}
