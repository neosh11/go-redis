package main

import (
	"fmt"
	"net"
	"os"
)

func processResponse(req net.Conn) error {
	// read the data from the connection
	data := make([]byte, 1024)
	read, err := req.Read(data)
	if err != nil {
		return err
	}

	fmt.Println("Received: ", string(data[:read]))

	// send back the response: +PONG\r\n
	_, err = req.Write([]byte("+PONG\r\n"))
	return err
}

func main() {
	fmt.Println("Logs from your program will appear here!")
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	// Close the listener when the application closes.
	defer func(l net.Listener) {
		err := l.Close()
		if err != nil {
			fmt.Println("Error closing listener: ", err.Error())
		}
	}(l)

	req, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	for {
		err := processResponse(req)
		if err != nil {
			fmt.Println("Error processing request: ", err.Error())
			os.Exit(1)
		}
	}
}
