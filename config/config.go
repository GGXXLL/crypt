package config

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/GGXXLL/crypt/backend"
	"github.com/GGXXLL/crypt/backend/consul"
	"github.com/GGXXLL/crypt/backend/etcd"
	"github.com/GGXXLL/crypt/backend/firestore"
	"github.com/GGXXLL/crypt/backend/redis"
	"github.com/GGXXLL/crypt/encoding/secconf"
)

type KVPair struct {
	backend.KVPair
}

type KVPairs []*KVPair

type configManager struct {
	store      backend.Store
	secret     []byte
	withSecret bool
}

type Config struct {
	Name          string
	Machines      []string
	Secret        []byte
	WatchInterval time.Duration
}

// Manager A ConfigManager retrieves and decrypts configuration from a key/value store.
type Manager interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Watch(ctx context.Context, key string) <-chan *Response
}

type OptionFunc func(c *configManager)

func WithSecretKey(secret []byte) OptionFunc {
	return func(c *configManager) {
		c.secret = secret
		c.withSecret = true
	}
}

func NewConfigManager(cfg Config) (Manager, error) {
	if cfg.WatchInterval == 0 {
		cfg.WatchInterval = 10 * time.Second
	}
	store, err := NewStore(cfg.Name, cfg.Machines, cfg.WatchInterval)
	if err != nil {
		return nil, err
	}
	m := &configManager{
		store:      store,
		secret:     cfg.Secret,
		withSecret: false,
	}

	return m, nil
}

func NewStore(name string, machines []string, watchInterval time.Duration) (backend.Store, error) {
	switch name {
	case "etcd":
		return etcd.New(machines)
	case "consul":
		return consul.New(machines, consul.WithWatchInterval(watchInterval))
	case "redis":
		return redis.New(machines, redis.WithWatchInterval(watchInterval))
	case "firestore":
		return firestore.New(machines, firestore.WithWatchInterval(watchInterval))
	default:
		return nil, errors.New("invalid backend " + name)
	}
}

func NewConfigManagerWithStore(store backend.Store, opts ...OptionFunc) (Manager, error) {
	m := &configManager{store: store}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// Get retrieves and decodes a secconf value stored at key.
func (c *configManager) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if c.withSecret {
		return secconf.Decode(value, bytes.NewBuffer(c.secret))
	}
	return value, nil
}

// Set will put a key/value into the data store
// and encode it with secconf
func (c *configManager) Set(ctx context.Context, key string, value []byte) error {
	if c.withSecret {
		encodedValue, err := secconf.Encode(value, bytes.NewBuffer(c.secret))
		if err != nil {
			return err
		}
		return c.store.Set(ctx, key, encodedValue)
	}

	return c.store.Set(ctx, key, value)
}

type Response struct {
	Value []byte
	Error error
}

func (c *configManager) Watch(ctx context.Context, key string) <-chan *Response {
	resp := make(chan *Response, 0)
	backendResp := c.store.Watch(ctx, key)
	go func() {
		for {
			select {
			case r := <-backendResp:
				if r.Error != nil {
					resp <- &Response{nil, r.Error}
					continue
				}
				if c.withSecret {
					value, err := secconf.Decode(r.Value, bytes.NewBuffer(c.secret))
					resp <- &Response{value, err}
					continue
				}
				resp <- &Response{r.Value, r.Error}
			case <-ctx.Done():
				resp <- &Response{Error: ctx.Err()}
				return

			}
		}
	}()
	return resp
}
