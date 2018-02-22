package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/mediocregopher/radix.v2/pubsub"
	"github.com/mediocregopher/radix.v2/redis"
)

// watch basically just subscribes to a given channel and prints out messages as they arrive. It blocks, so use a goroutine
func watch(cs, cname string, done <-chan struct{}) {
	c, err := redis.Dial("tcp", cs)
	if err != nil {
		fmt.Println("Failed to create redis client")
		return
	}

	sc := pubsub.NewSubClient(c)
	err = sc.PSubscribe(cname).Err
	if err != nil {
		fmt.Printf("Failed to subscribe to channel %s\n", cname)
		fmt.Println(err)
		return
	}
	for {
		select {
		case <-done:
			return
		default:
			r := sc.Receive()
			if r.Err != nil {
				fmt.Println("Error while attempting to recieve from channel")
				return
			}
			if !r.Timeout() {
				fmt.Println(r.Message)
			}
		}
	}
}

func main() {
	cs := os.Getenv("REDIS_ADDR")
	cn := os.Getenv("CHANNEL_NAME")
	done := make(chan struct{})
	defer close(done)

	go watch(cs, cn, done)

	c, err := redis.Dial("tcp", cs)
	if err != nil {
		fmt.Println("Failed to create redis client for publish")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error encountered when reading line from stdin")
			return
		}
		// Strip off line break
		msg = msg[:len(msg)-1]
		c.Cmd("PUBLISH", cn, msg)
	}
}
