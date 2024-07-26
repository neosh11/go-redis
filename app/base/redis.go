package base

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type KvValue struct {
	Value  string
	Expiry int
}

type RedisConfig struct {
	Port              int
	ReplicaOf         string
	ReplicationId     string
	ReplicationOffset int
	Connections       map[string]net.Conn
}

type Redis struct {
	Memory map[string]KvValue
	Config RedisConfig
}

func NewRedis(config *RedisConfig) *Redis {
	redis := Redis{
		Memory: make(map[string]KvValue),
		Config: *config,
	}
	if config.ReplicaOf != "" {
		redis.ConnectToMaster()
	}
	return &redis
}

func (r Redis) Replicate(data []byte) {

	// get count of keys in r.Config.Connections
	fmt.Println("Replicating data to ", len(r.Config.Connections), " connections")
	for _, conn := range r.Config.Connections {
		_, err := conn.Write(data)
		if err != nil {
			fmt.Println("Error replicating data: ", err.Error())
		}
	}
}

func (r Redis) AddConnection(conn net.Conn) {
	r.Config.Connections[conn.RemoteAddr().String()] = conn
}

func (r Redis) RemoveConnection(conn net.Conn) {
	if _, ok := r.Config.Connections[conn.RemoteAddr().String()]; !ok {
		return
	}
	delete(r.Config.Connections, conn.RemoteAddr().String())
}

func (r Redis) ConnectToMaster() {
	// send a message to the master
	replicaOf := r.Config.ReplicaOf
	// split the string to get the host and port
	split := strings.Split(replicaOf, " ")
	host := split[0]
	port := split[1]
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		panic(err)
		return
	}

	r.SendPingToMaster(conn)
	r.SendReplConfToMaster(conn)
	r.SendPsyncToMaster(conn)

	fmt.Println("Connected to master")
	go r.HandleMasterConnection(conn)
}

func (r Redis) HandleMasterConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		fmt.Println("Closing connection to master")
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(conn)
	err := r.ProcessIncomingMessage(conn, true)
	if err != nil {
		fmt.Println("Error processing request: ", err.Error())
		return
	}

}
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

func redisProtocolParser(data []byte) (command string, args []string, EOL int, err error) {
	//*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n
	//	Read the first line to get the number of arguments
	// check if the first character is *
	if data[0] != '*' {
		err = fmt.Errorf("Invalid data - *")
		return
	}

	EOL, err = getEndOfLine(1, data)
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

func (r Redis) ProcessIncomingMessage(req net.Conn, silent bool) error {
	// read the data from the connection
	reader := bufio.NewReader(req)
	for {
		data, _, err := reader.ReadLine()
		data = append(data, "\r\n"...)
		if err != nil {
			return err
		}
		// check it starts with *
		if len(data) == 0 {
			break
		}
		if data[0] != '*' {
			return fmt.Errorf("Invalid data", string(data))
		}
		// check number of arguments
		numbParams := string(data[1 : len(data)-2])
		numbParamsInt, err := strconv.Atoi(numbParams)
		if err != nil {
			return fmt.Errorf("Invalid data", numbParams)
		}

		//read the next numbParamsInt lines
		for i := 0; i < numbParamsInt*2; i++ {
			line, _, err := reader.ReadLine()
			if err != nil {
				return err
			}
			line = append(line, "\r\n"...)
			data = append(data, line...)
		}

		// process the data
		_, err = r.ProcessCommand(req, data, silent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r Redis) ProcessCommand(req net.Conn, data []byte, silent bool) (EOL int, returnEror error) {

	fmt.Println("Command: ", string(data))

	command, args, EOL, err := redisProtocolParser(data)
	if err != nil {
		return 0, err
	}

	val := ""
	replicate := false
	if command == "PING" {
		_, err = req.Write([]byte("+PONG\r\n"))
	} else if command == "ECHO" {
		val = r.Echo(args)
	} else if command == "SET" {
		val = r.Set(args)
		replicate = true
	} else if command == "GET" {
		val = r.Get(args)
	} else if command == "INFO" {
		val = r.Info(args)
	} else if command == "REPLCONF" {
		val = r.REPLCONF(args)
		_, err = req.Write([]byte(val))
		if err != nil {
			fmt.Println("Error writing to connection: ", err.Error())
		}
		r.AddConnection(req)
		return EOL, err
	} else if command == "PSYNC" {
		bytes := r.PSYNC(args, req)
		_, err = req.Write(bytes)
		return EOL, err
	} else {
		val = "-ERR unknown command '" + command + "'\r\n"
	}

	if !silent {
		_, err = req.Write([]byte(val))
		if replicate {
			r.Replicate(data)
		}
	}

	if err != nil {
		return EOL, err
	}
	return EOL, err
}

func (r Redis) SendPingToMaster(conn net.Conn) {
	rb := NewRequestBuilder()
	rb.AddLine("PING")
	r.SendBytesToMaster(conn, rb.Bytes())
}

func (r Redis) SendReplConfToMaster(conn net.Conn) {
	rb := NewRequestBuilder()
	rb.AddLine("REPLCONF")
	rb.AddLine("listening-port")
	rb.AddLine(strconv.Itoa(r.Config.Port))
	r.SendBytesToMaster(conn, rb.Bytes())

	rb.Reset()
	rb.AddLine("REPLCONF")
	rb.AddLine("capa")
	rb.AddLine("psync2")
	r.SendBytesToMaster(conn, rb.Bytes())
}

func (r Redis) SendPsyncToMaster(conn net.Conn) {
	rb := NewRequestBuilder()
	rb.AddLine("PSYNC")
	rb.AddLine("?")
	rb.AddLine("-1")

	_, err := conn.Write(rb.Bytes())
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(conn)
	// $
	data, _, err := reader.ReadLine()
	// read the next bytesToReadInt bytes
	data, _, err = reader.ReadLine()

	data, _, err = reader.ReadLine()
	bytesToRead := data[1:]

	bytesToReadInt, err := strconv.Atoi(string(bytesToRead))

	// read the next bytesToReadInt bytes
	data = make([]byte, bytesToReadInt)
	_, err := reader.Read(data)

	if err != nil {
		fmt.Println("Error reading from master: ", err.Error())
	}

	fmt.Println("RDB dump: ", string(data))
}

func (r Redis) SendBytesToMaster(conn net.Conn, message []byte) []byte {
	_, err := conn.Write(message)
	if err != nil {
		panic(err)
	}
	response := make([]byte, 1024)
	resLen, err := conn.Read(response)
	if err != nil {
		panic(err)
	}
	return response[:resLen]
}
