package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/mediocregopher/radix.v2/pool"
)

// This defines a global scope variable for the cache which is populated in main
var c *Cache

// Cache is a key/value cache. Currently we only support strings
type Cache struct {
	pool *pool.Pool
}

// Get retrieves the value for the given key
func (c *Cache) Get(k string) (string, error) {
	resp, err := c.pool.Cmd("GET", k).Str()
	return resp, err
}

// Set assigns a given value to the given key
func (c *Cache) Set(k, v string, ex int) error {
	err := c.pool.Cmd("SET", k, v, "EX", ex).Err
	return err
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	keyMap := r.URL.Query()
	for param, vals := range keyMap {
		if param == "key" {
			fmt.Printf("getting values for %s\n", vals)
			for _, k := range vals {
				v, err := c.Get(k)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("Got value %s for %s\n", v, k)
				fmt.Fprintf(w, "%s = %s\n", k, v)
			}
		}
	}
	return
}

func setHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var kv map[string]string
	if r.Body == nil {
		http.Error(w, "Request contained no body", 400)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", 400)
	}

	err = json.Unmarshal(body, &kv)
	if err != nil {
		http.Error(w, "Error unmarshaling json payload", 400)
	}

	fmt.Printf("recieved set requests: %s\n", kv)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	for k, v := range kv {
		fmt.Printf("Setting %s to %s\n", k, v)
		err = c.Set(k, v, 666)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Set %s to %s\n", k, v)
		fmt.Fprintf(w, "%s set to %s\n", k, v)
	}
	return
}

func main() {
	cs := os.Getenv("REDIS_CACHE")
	pool, err := pool.New("tcp", cs, 10)
	if err != nil {
		fmt.Println("Failed to create redis connection pool!")
		fmt.Println(err)
		return
	}
	defer pool.Empty()
	c = &Cache{pool}

	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/set", setHandler)
	http.ListenAndServe(":8080", nil)
}
