package main

import (
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/base"
	"github.com/codecrafters-io/redis-starter-go/app/cli"
	"net"
	"os"
	"strconv"
)

var redis *base.Redis

func handleConnection(req net.Conn) {
	defer func() {
		redis.RemoveConnection(req)
		err := req.Close()
		if err != nil {
			fmt.Println("Error closing connection: ", err.Error())
		}
	}()
	for {
		err := redis.ProcessIncomingMessage(req, false)
		if err != nil {
			fmt.Println("Error processing request: ", err.Error())
			return
		}
	}
}

func main() {
	config := cli.GetRedisConfig()
	redis = base.NewRedis(config)
	fmt.Println("Logs from your program will appear here!")
	l, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(config.Port))
	if err != nil {
		fmt.Println("Failed to bind to port", config.Port)
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			fmt.Println("Error closing listener: ", err.Error())
		}
	}(l)

	for {
		req, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(req)
	}
}
