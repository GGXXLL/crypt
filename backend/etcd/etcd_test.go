package etcd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/GGXXLL/crypt/backend"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	addr := os.Getenv("ETCD_ADDR")
	if addr == "" {
		t.Skip()
	}
	client, err := New(strings.Split(addr, ","))
	assert.NoError(t, err)

	err = client.Set(context.TODO(), "crypt_test", []byte("test"))
	assert.NoError(t, err)

	val, err := client.Get(context.TODO(), "crypt_test")
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), val)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp := client.Watch(ctx, "crypt_test")

	err = client.Set(context.TODO(), "crypt_test", []byte("update"))
	assert.NoError(t, err)
	var r *backend.Response
	r = <-resp
	assert.NoError(t, r.Error)
	assert.Equal(t, []byte("update"), r.Value)

	cancel()
	r = <-resp
	assert.Error(t, r.Error)
}
