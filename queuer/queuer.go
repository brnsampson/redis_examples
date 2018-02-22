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
var q *Queue

// Cache is a key/value cache. Currently we only support strings
type Queue struct {
	name string
	pool *pool.Pool
}

// Pop retrieves the oldest value pushed to our queue
func (q *Queue) Pop() (string, error) {
	resp, err := q.pool.Cmd("LPOP", q.name).Str()
	return resp, err
}

// Push adds a new value to our queue
func (q *Queue) Push(v string) error {
	err := q.pool.Cmd("RPUSH", q.name, v).Err
	return err
}

func popHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("popping first value off queue %s\n", q.name)
	v, err := q.Pop()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error issueing pop command", 400)
		return
	}
	fmt.Printf("Got value %s\n", v)
	fmt.Fprintf(w, "%s\n", v)
	return
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var vals []string
	if r.Body == nil {
		http.Error(w, "Request contained no body", 400)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", 400)
	}

	err = json.Unmarshal(body, &vals)
	if err != nil {
		http.Error(w, "Error unmarshaling json payload", 400)
	}

	fmt.Printf("recieved push requests: %s\n", vals)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	for _, v := range vals {
		fmt.Printf("Pushing %s\n", v)
		err = q.Push(v)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Pushed %s to queue\n", v)
		fmt.Fprintf(w, "%s pushed\n", v)
	}
	return
}

func main() {
	cs := os.Getenv("REDIS_ADDR")
	qn := os.Getenv("REDIS_QUEUE")
	pool, err := pool.New("tcp", cs, 10)
	if err != nil {
		fmt.Println("Failed to create redis connection pool!")
		fmt.Println(err)
		return
	}
	defer pool.Empty()
	q = &Queue{
		qn,
		pool,
	}

	http.HandleFunc("/pop", popHandler)
	http.HandleFunc("/push", pushHandler)
	http.ListenAndServe(":8080", nil)
}
