package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

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

func redisProtocolParser(data []byte) (command string, arg1 string, err error) {
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

	if numbParamsInt > 2 {
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
	if numbParamsInt == 2 {
		// Read the second parameter
		arg1, EOL, err = processParameter(EOL+1, data)
		if err != nil {
			err = fmt.Errorf("failed to process parameter", err)
			return
		}
	}
	fmt.Println("Command: ", command)
	return
}

func processResponse(req net.Conn) error {
	// read the data from the connection
	data := make([]byte, 1024)
	read, err := req.Read(data)
	if err != nil {
		return err
	}

	command, arg1, err := redisProtocolParser(data[:read])
	if err != nil {
		return err
	}

	if command == "PING" {
		_, err = req.Write([]byte("+PONG\r\n"))
	} else if command == "ECHO" {
		if arg1 == "" {
			_, err = req.Write([]byte("$0\r\n\r\n"))
		} else {
			_, err = req.Write([]byte("$" + strconv.Itoa(len(arg1)) + "\r\n" + arg1 + "\r\n"))
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

	for {
		req, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(req)
	}
}
