package firestore

import (
	"context"
	"errors"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/ggxxll/crypt/backend"
	"github.com/ggxxll/crypt/internal"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

type Client struct {
	client        *firestore.Client
	cache         *sync.Map
	watchInterval time.Duration
}

type data struct {
	Data []byte `firestore:"data"`
}

type OptionFunc func(client *Client)

func WithWatchInterval(duration time.Duration) OptionFunc {
	return func(client *Client) {
		client.watchInterval = duration
	}
}

func New(machines []string, opts ...OptionFunc) (*Client, error) {
	if len(machines) == 0 {
		return nil, errors.New("project should be defined")
	}

	c, err := firestore.NewClient(context.TODO(), machines[0], []option.ClientOption{
		option.WithGRPCDialOption(grpc.WithBlock()),
	}...)
	if err != nil {
		return nil, err
	}
	cli := &Client{
		client:        c,
		cache:         &sync.Map{},
		watchInterval: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(cli)
	}
	return cli, nil
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	snap, err := c.client.Doc(path).Get(ctx)
	if err != nil {
		return nil, err
	}

	d := &data{}
	err = snap.DataTo(&d)
	if err != nil {
		return nil, err
	}
	return d.Data, nil
}

func (c *Client) Set(ctx context.Context, path string, value []byte) error {
	_, err := c.client.Doc(path).Set(ctx, &data{value})
	return err
}

func (c *Client) Watch(ctx context.Context, path string) <-chan *backend.Response {
	ch := make(chan *backend.Response, 0)

	go func() {
		defer func() {
			close(ch)
		}()
		for {
			select {
			case <-time.After(c.watchInterval):
				val, err := c.Get(ctx, path)
				if err != nil {
					ch <- &backend.Response{Error: err}
					continue
				}
				internal.WatchCache(c.cache, path, val, ch)
			case <-ctx.Done():
				ch <- &backend.Response{Error: ctx.Err()}
				return
			}
		}
	}()
	return ch
}
