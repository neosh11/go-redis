package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

type Value struct {
	value  string
	expiry int
}

var memory = make(map[string]Value)

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

	if command == "PING" {
		_, err = req.Write([]byte("+PONG\r\n"))
	} else if command == "ECHO" {
		if len(args) == 0 {
			_, err = req.Write([]byte("$0\r\n\r\n"))
		} else {
			_, err = req.Write([]byte("$" + strconv.Itoa(len(args[0])) + "\r\n" + args[0] + "\r\n"))
		}
	} else if command == "SET" {
		// check first 2 arguments
		if len(args) < 2 {
			_, err = req.Write([]byte("-ERR wrong number of arguments for 'set' command\r\n"))
			return err
		}
		expiry := -1
		if len(args) >= 4 && args[2] == "px" {
			ex, lErr := strconv.Atoi(args[3])
			if lErr != nil {
				_, err = req.Write([]byte("-ERR invalid expiry value\r\n"))
				return err
			}
			expiry = int((int64)(time.Now().UnixMilli())) + ex
		}
		memory[args[0]] = Value{
			value:  args[1],
			expiry: expiry,
		}
		_, err = req.Write([]byte("+OK\r\n"))
	} else if command == "GET" {
		if len(args) < 1 {
			_, err = req.Write([]byte("-ERR wrong number of arguments for 'get' command\r\n"))
			return err
		}
		value, ok := memory[args[0]]
		if !ok {
			_, err = req.Write([]byte("$-1\r\n"))
		} else {
			if value.expiry != -1 {
				fmt.Println("Expiry: ", value.expiry, " Time: ", int((int64)(time.Now().UnixMilli())))
				if value.expiry < int((int64)(time.Now().UnixMilli())) {
					delete(memory, args[0])
					_, err = req.Write([]byte("$-1\r\n"))
					if err != nil {
						return err
					}
					return nil
				}
			}
			_, err = req.Write([]byte("$" + strconv.Itoa(len(value.value)) + "\r\n" + value.value + "\r\n"))
		}
	} else {
		_, err = req.Write([]byte("-ERR unknown command '" + command + "'\r\n"))
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
	// get port from command line --port flag
	port := "6379"
	flags := os.Args[1:]
	if len(flags) > 0 && flags[0] == "--port" {
		if len(flags) < 2 {
			fmt.Println("Missing port number")
			os.Exit(1)
		}
		port = flags[1]
	}
	fmt.Println("Logs from your program will appear here!")
	l, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		fmt.Println("Failed to bind to port", port)
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
