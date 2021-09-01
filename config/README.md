# crypt/config

## Usage

### Get configuration from a backend

```
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ggxxll/crypt/config"
)

var (
	key           = "/app/config"
	secretKeyring = ".secring.gpg"
)

func main() {
	kr, err := os.Open(secretKeyring)
	if err != nil {
		log.Fatal(err)
	}
	defer kr.Close()
	machines := []string{"http://127.0.0.1:2379"}
	cm, err := config.NewConfigManager("etcd", machines, kr)
	if err != nil {
        log.Fatal(err)
    }
	value, err := cm.Get(context.TODO(), key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", value)
}
```

### Monitor backend for configuration changes

```
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ggxxll/crypt/config"
)

var (
	key           = "/app/config"
	secretKeyring = ".secring.gpg"
)

func main() {
	kr, err := os.Open(secretKeyring)
	if err != nil {
		log.Fatal(err)
	}
	defer kr.Close()
	secret, err := ioutil.ReadAll(kr)
	if err != nil {
		log.Fatal(err)
	}
	cfg := config.Config{
		Name:          "etcd",
		Machines:      []string{"http://127.0.0.1:2379"},
		Secret:        secret,
		WatchInterval: 5 * time.Second,
	}
	cm, err := config.NewConfigManager(cfg)
	if err != nil {
		log.Fatal(err)
	}
	
	value, err := cm.Get(context.TODO(), key)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", value)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp := cm.Watch(ctx, key)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case r := <-resp:
			if r.Error != nil {
				fmt.Println(r.Error.Error())
			}
			fmt.Printf("%s\n", r.Value)
		}
	}
}
```
