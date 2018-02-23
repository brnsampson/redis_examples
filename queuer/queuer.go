package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/mediocregopher/radix.v2/pool"
)

// This defines a global scope variable for the queue which is populated in main
var q *Queue

// Queue is a queue based on a redis list. Because this is a simple example the struct only contains a name and connection pool
// and the only methods on it are a simple push and pop. Normally you would not expose just a queue on a web interface, but there
// are applications for something like this. For example, you might have a number of worker applications that are pulling jobs
// from a shared queue. The single threaded nature of redis ensures that each job can be popped off exactly once and eliminates
// race conditions between multiple workers.
type Queue struct {
	name string
	pool *pool.Pool
}

// Pop retrieves the oldest value pushed to our queue. If the queue is empty you will get an error, but this could be avoided by
// executing a blocking pop (BLPOP) in a goroutine instead.
func (q *Queue) Pop() (string, error) {
	resp, err := q.pool.Cmd("LPOP", q.name).Str()
	// always propagate your errors
	return resp, err
}

// Push adds a new value to our queue. If you wanted to ensure the queue does not exceed some size, you would add in an LTRIM
// command after your push: err := q.pool.Cmd("LTRIM", q.name, 0, maxSize).Err
func (q *Queue) Push(v string) error {
	err := q.pool.Cmd("RPUSH", q.name, v).Err
	return err
}

// popHandler is, similar to the cacher example, a function called when a requests comes into the /pop endpoint. Since there is
// no query data needed to pop from a queue, this is a pretty simple handler.
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

// pushHandler reads the json payload of requests to the /push endpoint and adds each value passed to the queue
func pushHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// vals is what we are unmarshaling our json into. In this case we only accept a json list. Anything besides a flat array should cause errors.
	var vals []string
	if r.Body == nil {
		http.Error(w, "Request contained no body", 400)
		return
	}

	// r.Body is actually an io.ReadCloser so we have to fully read it into a string before passing it to json.Unmarshal
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", 400)
	}

	// note that we are actually passing a pointer to vals. This populates vals for us and returns an error if anything went wrong.
	err = json.Unmarshal(body, &vals)
	if err != nil {
		http.Error(w, "Error unmarshaling json payload", 400)
	}

	fmt.Printf("recieved push requests: %s\n", vals)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// we are allowing an arbitrary sized array to be pushed in a single call. The items are pushed from left to right
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
	// using env variables is preferred to config files
	cs := os.Getenv("REDIS_ADDR")
	qa := os.Getenv("QUEUE_ADDR")
	qn := os.Getenv("QUEUE_NAME")

	// we again use a pool instead of a single connection because each http request is executed in a new goroutine and individual clients are not thread safe.
	pool, err := pool.New("tcp", cs, 10)
	if err != nil {
		fmt.Println("Failed to create redis connection pool!")
		// normally this would either be logged or output to stderr. For simplicity I am just printing it, but generally you shouldn't do this.
		// See https://golang.org/pkg/log/#Fatalln, https://golang.org/pkg/os/#Exit, and consider using fmt.Fprintf() with os.Stderr
		fmt.Println(err)
		os.Exit(1)
	}
	defer pool.Empty()

	// our queue is actually an application of a redis list. As such, we could have multiple queues on a single redis server. I'm just lazy.
	q = &Queue{
		qn,
		pool,
	}

	http.HandleFunc("/pop", popHandler)
	http.HandleFunc("/push", pushHandler)
	http.ListenAndServe(qa, nil)
}
