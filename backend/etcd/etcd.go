package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/bketelsen/crypt/backend"
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

func (c *Client) Get(key string) ([]byte, error) {
	return c.GetWithContext(context.TODO(), key)
}

func (c *Client) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, fmt.Errorf("no such config key: %s", key)
	}

	return resp.Kvs[0].Value, nil
}

func (c *Client) List(key string) (backend.KVPairs, error) {
	return c.ListWithContext(context.TODO(), key)
}

func (c *Client) ListWithContext(ctx context.Context, key string) (backend.KVPairs, error) {
	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	list := backend.KVPairs{}
	for _, kv := range resp.Kvs {
		list = append(list, &backend.KVPair{Key: string(kv.Key), Value: kv.Value})
	}

	return list, nil
}

func (c *Client) Set(key string, value []byte) error {
	return c.SetWithContext(context.TODO(), key, value)
}

func (c *Client) SetWithContext(ctx context.Context, key string, value []byte) error {
	_, err := c.client.Put(ctx, key, string(value))
	return err
}

func (c *Client) Watch(key string, stop chan bool) <-chan *backend.Response {
	return c.WatchWithContext(context.Background(), key, stop)
}

func (c *Client) WatchWithContext(ctx context.Context, key string, stop chan bool) <-chan *backend.Response {
	respChan := make(chan *backend.Response, 0)
	go func() {
		defer func() {
			c.client.Close()
		}()
		rch := c.client.Watch(ctx, key)
		for {
			select {
			case resp := <-rch:
				if resp.Err() != nil {
					respChan <- &backend.Response{Error: resp.Err()}
					time.Sleep(time.Second * 5)
					continue
				}
				for _, e := range resp.Events {
					respChan <- &backend.Response{Value: e.Kv.Value}
				}
			case <-stop:
				close(respChan)
				return
			case <-ctx.Done():
				respChan <- &backend.Response{Error: ctx.Err()}
				return
			}
		}
	}()
	return respChan
}
