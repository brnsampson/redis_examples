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

// Cache is a redis cache. Because this is a simple example the struct only contains a connection pool and the only
// methods on it are a simple get and set, but if you were seriously creating a cache struct for a production system you
// would at least have a subscribe and publish channel for cache invalidation and you would probably has some database layer
// as a fallback for cache misses.
type Cache struct {
	pool *pool.Pool
}

// Get retrieves the value for the given key. Normally you would have methods to support all types of data, but
// I'm limiting this to simple key-values.
func (c *Cache) Get(k string) (string, error) {
	resp, err := c.pool.Cmd("GET", k).Str()
	// Generally speaking, the right thing to do is always to propagate any error you recieve.
	return resp, err
}

// Set assigns a given value to the given key. Normally there would be methods to set all data types, but
// this is just an example.
func (c *Cache) Set(k, v string, ex int) error {
	err := c.pool.Cmd("SET", k, v, "EX", ex).Err
	return err
}

// getHandler implements a function with the signature func(ResponseWriter, *Request) as required by the http.HandleFunc method
// in main. Any request that goes to the /get endpoint will trigger this function to be run. As a side note, normally we would
// use some kind of request router like http://www.gorillatoolkit.org/. We used the base http package for clarity.
func getHandler(w http.ResponseWriter, r *http.Request) {
	// This is a pretty simple way to handle query strings. .../get?key=dog will result in a map like {"key": ["dog"]} here.
	keyMap := r.URL.Query()
	for param, vals := range keyMap {
		if param == "key" {
			fmt.Printf("getting values for %s\n", vals)
			for _, k := range vals {
				v, err := c.Get(k)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("Got value %s for %s\n", v, k)
					fmt.Fprintf(w, "%s = %s\n", k, v)
				}
			}
		}
	}
	return
}

// setHandler implements a function with the signature func(ResponseWriter, *Request) as required by the http.HandleFunc method
// in main. Any request that goes to the /set endpoint will trigger this function to be run. Unlike the get endpoint, set
// expects a POST of PUT request and as such the handling is different.
func setHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// kv will be used soon in json.Unmarshal. The structure of this map limits the accepted json objects in the request.
	// If you want to accept arbitrary json with nexted structures then you have to use map[string]interface{} or something
	// and do a bit of slighly annoying stuff.
	var kv map[string]string

	if r.Body == nil {
		http.Error(w, "Request contained no body", 400)
		return
	}

	// r.Body is actually an io.ReadCloser so we have to fully read it into a string before passing it to json.Unmarshal
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

	// We allow multiple values to set in each request
	for k, v := range kv {
		fmt.Printf("Setting %s to %s\n", k, v)
		err = c.Set(k, v, 666)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Set %s to %s\n", k, v)
			fmt.Fprintf(w, "%s set to %s\n", k, v)
		}
	}
	return
}

// main is usually pretty simple; just some initialization and config followed by starting any goroutines and starting
// the main loop.
func main() {
	cs := os.Getenv("REDIS_ADDR")
	ca := os.Getenv("CACHE_ADDR")
	// Create a pool of 10 redis connections that can be used by multiple threads safely.
	pool, err := pool.New("tcp", cs, 10)
	if err != nil {
		fmt.Println("Failed to create redis connection pool!")
		// normally this would either be logged or output to stderr. For simplicity I am just printing it, but generally you shouldn't do this.
		// See https://golang.org/pkg/log/#Fatalln, https://golang.org/pkg/os/#Exit, and consider using fmt.Fprintf() with os.Stderr
		fmt.Println(err)
		os.Exit(1)
	}

	// pool.Empty() closes all connections in the pool and does other cleanup actions. defer tells go to execute this line after
	// the code execution hits an exit point for this function.
	defer pool.Empty()

	// the variable c is actually defined at the top of this file outside of any functions. However, we have to actually give
	// it a value here. This method allows the handler functions to treat c as a global variable.
	c = &Cache{pool}

	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/set", setHandler)
	http.ListenAndServe(ca, nil)
}
