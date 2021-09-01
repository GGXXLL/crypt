package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/GGXXLL/crypt/backend"
	goetcd "go.etcd.io/etcd/client/v3"
)

type Client struct {
	client *goetcd.Client
}

func New(machines []string) (*Client, error) {
	newClient, err := goetcd.New(goetcd.Config{
		Endpoints:   machines,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("creating new etcd client for crypt.backend.Client: %v", err)
	}
	return &Client{client: newClient}, nil
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, fmt.Errorf("no such config key: %s", key)
	}

	return resp.Kvs[0].Value, nil
}

func (c *Client) Set(ctx context.Context, key string, value []byte) error {
	_, err := c.client.Put(ctx, key, string(value))
	return err
}

func (c *Client) Watch(ctx context.Context, key string) <-chan *backend.Response {
	respChan := make(chan *backend.Response, 0)
	go func() {
		defer func() {
			close(respChan)
			c.client.Close()
		}()
		rch := c.client.Watch(ctx, key)
		for {
			select {
			case resp := <-rch:
				if resp.Err() != nil {
					respChan <- &backend.Response{Error: resp.Err()}
					continue
				}
				for _, e := range resp.Events {
					respChan <- &backend.Response{Value: e.Kv.Value}
				}
			case <-ctx.Done():
				respChan <- &backend.Response{Error: ctx.Err()}
				return
			}
		}
	}()
	return respChan
}
