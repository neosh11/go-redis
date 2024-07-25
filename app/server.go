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

func getEndOfLine(startingIndex int, data []byte) (int, error) {
	index := startingIndex
	for index < len(data) && data[index-1] != '\r' && data[index] != '\n' {
		index++
	}
	if index >= len(data) {
		return -1, fmt.Errorf("Invalid data")
	}
	return index, nil
}

func processParameter(startIndex int, data []byte) (value string, EOL int, err error) {
	// check if the first character is $
	if data[startIndex] != '$' {
		return "", -1, fmt.Errorf("Invalid data $")
	}
	// read the first line to get the length of the first argument
	sizeEOL, err := getEndOfLine(startIndex+1, data)
	if err != nil {
		return "", -1, fmt.Errorf("Invalid data EOL - param")
	}
	length := string(data[startIndex+1 : sizeEOL-1])
	lengthInt, err := strconv.Atoi(length)

	if err != nil {
		return "", -1, fmt.Errorf("Invalid data", length)
	}
	// ensure that the length of the first argument is correct
	EOL = sizeEOL + 2 + lengthInt

	if EOL > len(data) || data[EOL-1] != '\r' || data[EOL] != '\n' {
		return "", -1, fmt.Errorf("Invalid data", string(data[sizeEOL+1:EOL-1]))
	}
	// read the first argument
	return string(data[sizeEOL+1 : EOL-1]), EOL, nil
}

func redisProtocolParser(data []byte) (command string, args []string, err error) {
	//*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n
	//	Read the first line to get the number of arguments
	// check if the first character is *
	if data[0] != '*' {
		err = fmt.Errorf("Invalid data - *")
		return
	}

	EOL, err := getEndOfLine(1, data)
	if err != nil {
		err = fmt.Errorf("Invalid data - EOL1")
		return
	}

	numbParams := string(data[1 : EOL-1])
	numbParamsInt, err := strconv.Atoi(numbParams)
	if err != nil {
		err = fmt.Errorf("Invalid data", numbParams)
		return
	}

	if numbParamsInt > 10 {
		err = fmt.Errorf("Only 1-2 parameters allowed at the moment")
	}

	// Read the first parameter which is the command
	command, EOL, err = processParameter(EOL+1, data)
	if err != nil {
		err = fmt.Errorf("failed to process parameter", err)
		return
	}

	if numbParamsInt == 1 {
		return
	}

	args = make([]string, 0)
	for i := 1; i < numbParamsInt; i++ {
		// Read the second parameter
		value, localEol, localErr := processParameter(EOL+1, data)
		EOL = localEol
		if localErr != nil {
			err = fmt.Errorf("failed to process parameter", err)
			return
		}
		args = append(args, value)
	}
	return
}
func processResponse(req net.Conn) error {
	// read the data from the connection
	data := make([]byte, 1024)
	read, err := req.Read(data)
	if err != nil {
		return err
	}

	fmt.Println("Command: ", string(data[:read]))

	command, args, err := redisProtocolParser(data[:read])
	if err != nil {
		return err
	}

	val := ""
	if command == "PING" {
		_, err = req.Write([]byte("+PONG\r\n"))
	} else if command == "ECHO" {
		val = redis.Echo(args)
	} else if command == "SET" {
		val = redis.Set(args)
	} else if command == "GET" {
		val = redis.Get(args)
	} else if command == "INFO" {
		val = redis.Info(args)
	} else {
		val = "-ERR unknown command '" + command + "'\r\n"
	}

	_, err = req.Write([]byte(val))
	if err != nil {
		return err
	}
	return err
}

func handleConnection(req net.Conn) {
	defer func() {
		err := req.Close()
		if err != nil {
			fmt.Println("Error closing connection: ", err.Error())
		}
	}()
	for {
		err := processResponse(req)
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
