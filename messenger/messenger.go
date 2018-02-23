package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/mediocregopher/radix.v2/pubsub"
	"github.com/mediocregopher/radix.v2/redis"
)

// watch basically just subscribes to a given channel and prints out messages as they arrive. It blocks, so use a goroutine
func watch(cs, cname, user string, done <-chan struct{}) {
	// because this will be run with a goroutine and it will be pretty long-lived we just create a single connection here instead of
	// pulling from a connection pool
	c, err := redis.Dial("tcp", cs)
	if err != nil {
		fmt.Println("Failed to create redis client")
		return
	}
	// close the connection when we exit this function
	defer c.Close()

	// create a subclient based on our redis connection. Once we use this to subscribe to a channel we should never use that connection for anything else
	sc := pubsub.NewSubClient(c)
	err = sc.Subscribe(cname).Err
	if err != nil {
		fmt.Printf("Failed to subscribe to channel %s\n", cname)
		fmt.Println(err)
		return
	}
	// defers execute in LIFO order, so our unsubscribe will always happen BEFORE the defer that closes the channel above
	defer sc.Unsubscribe(cname)

	for {
		select {
		// the done channel is provided by the calling function. When the main function closes the channel this case will be hit and this function will return
		case <-done:
			return
		default:
			// Recieve will either error, timeout, or recieve a message, so we just loop on this forever
			r := sc.Receive()
			if r.Err != nil {
				fmt.Println("Error while attempting to recieve from channel")
				return
			}
			if !r.Timeout() {
				if r.Message[:len(user)] != user {
					// Don't echo messages we sent.
					fmt.Println(r.Message)
				}
			}
		}
	}
}

func main() {
	cs := os.Getenv("REDIS_ADDR")
	cn := os.Getenv("CHANNEL_NAME")
	u := os.Getenv("CHANNEL_USER")
	done := make(chan struct{})
	defer close(done)

	// execute the watch function with concurrency
	go watch(cs, cn, u, done)

	// another single channel instead of a pool. Same reason as in watch()
	c, err := redis.Dial("tcp", cs)
	if err != nil {
		fmt.Println("Failed to create redis client for publish")
		return
	}

	// bufio readers are pretty useful things. See https://golang.org/pkg/bufio/#Reader
	reader := bufio.NewReader(os.Stdin)
	for {
		// buffer until we hit a carrage return, then read the buffer in as a string
		s, err := reader.ReadString('\n')
		if err != nil {
			// normally this would either be logged or output to stderr. For simplicity I am just printing it, but generally you shouldn't do this.
			// See https://golang.org/pkg/log/#Fatalln, https://golang.org/pkg/os/#Exit, and consider using fmt.Fprintf() with os.Stderr
			fmt.Println("Error encountered when reading line from stdin")
			os.Exit(1)
		}
		// Strip off line break and attach the username to the message.
		line := s[:len(s)-1]
		msg := fmt.Sprintf("%s \xc2\xbb %s", u, line)
		// push message to all subscribers
		c.Cmd("PUBLISH", cn, msg)
	}
}
