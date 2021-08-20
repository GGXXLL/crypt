// Package backend provides the K/V store interface for crypt backends.
package backend

import "context"

// Response represents a response from a backend store.
type Response struct {
	Value []byte
	Error error
}

// KVPair holds both a key and value when reading a list.
type KVPair struct {
	Key   string
	Value []byte
}

type KVPairs []*KVPair

// A Store is a K/V store backend that retrieves and sets, and monitors
// data in a K/V store.
type Store interface {
	// Get retrieves a value from a K/V store for the provided key.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set sets the provided key to value.
	Set(ctx context.Context, key string, value []byte) error

	// Watch monitors a K/V store for changes to key.
	Watch(ctx context.Context, key string) <-chan *Response
}


type Watcher interface {

}