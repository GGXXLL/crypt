package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/GGXXLL/crypt/backend"
	"github.com/GGXXLL/crypt/backend/consul"
	"github.com/GGXXLL/crypt/backend/etcd"
	"github.com/GGXXLL/crypt/backend/redis"
	"github.com/GGXXLL/crypt/encoding/secconf"
)

func getCmd(flagset *flag.FlagSet) {
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s get [args...] key\n", os.Args[0])
		flagset.PrintDefaults()
	}
	flagset.StringVar(&secretKeyring, "secret-keyring", ".secring.gpg", "path to armored secret keyring")
	flagset.Parse(os.Args[2:])
	if key == "" {
		flagset.Usage()
		os.Exit(1)
	}
	backendStore, err := getBackendStore(backendName, endpoint)
	if err != nil {
		log.Fatal(err)
	}
	if plaintext {
		value, err := getPlain(key, backendStore)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", value)
		return
	}
	value, err := getEncrypted(key, secretKeyring, backendStore)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", value)
}

func getEncrypted(key, keyring string, store backend.Store) ([]byte, error) {
	var value []byte
	kr, err := os.Open(secretKeyring)
	if err != nil {
		return value, err
	}
	defer kr.Close()
	data, err := store.Get(context.TODO(), key)
	if err != nil {
		return value, err
	}
	value, err = secconf.Decode(data, kr)
	if err != nil {
		return value, err
	}
	return value, err

}

func getPlain(key string, store backend.Store) ([]byte, error) {
	var value []byte
	data, err := store.Get(context.TODO(), key)
	if err != nil {
		return value, err
	}
	return data, err
}

func setCmd(flagset *flag.FlagSet) {
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s set [args...] key file\n", os.Args[0])
		flagset.PrintDefaults()
	}
	flagset.StringVar(&keyring, "keyring", ".pubring.gpg", "path to armored public keyring")
	flagset.Parse(os.Args[2:])
	if key == "" {
		flagset.Usage()
		os.Exit(1)
	}
	if data == "" {
		flagset.Usage()
		os.Exit(1)
	}
	backendStore, err := getBackendStore(backendName, endpoint)
	if err != nil {
		log.Fatal(err)
	}
	d, err := ioutil.ReadFile(data)
	if err != nil {
		log.Fatal(err)
	}

	if plaintext {
		err := setPlain(key, backendStore, d)
		if err != nil {
			log.Fatal(err)
			return
		}
		return
	}
	err = setEncrypted(key, keyring, d, backendStore)
	if err != nil {
		log.Fatal(err)
	}
	return

}
func setPlain(key string, store backend.Store, d []byte) error {
	err := store.Set(context.TODO(), key, d)
	return err

}

func setEncrypted(key, keyring string, d []byte, store backend.Store) error {
	kr, err := os.Open(keyring)
	if err != nil {
		return err
	}
	defer kr.Close()
	secureValue, err := secconf.Encode(d, kr)
	if err != nil {
		return err
	}
	err = store.Set(context.TODO(), key, secureValue)
	return err
}

func getBackendStore(provider string, endpoint string) (backend.Store, error) {
	if endpoint == "" {
		switch provider {
		case "consul":
			endpoint = "127.0.0.1:8500"
		case "etcd":
			endpoint = "http://127.0.0.1:4001"
		case "redis":
			endpoint = "http://127.0.0.1:6379"
		}
	}
	machines := []string{endpoint}
	switch provider {
	case "etcd":
		return etcd.New(machines)
	case "consul":
		return consul.New(machines)
	case "redis":
		return redis.New(machines)
	default:
		return nil, errors.New("invalid backend " + provider)
	}
}
