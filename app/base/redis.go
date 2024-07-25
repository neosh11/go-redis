package base

import (
	"fmt"
	"net"
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
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(conn)
	fmt.Println("Connected to master")

	// send a PING to the master
	r.SendPingToMaster(conn)
}

func (r Redis) SendPingToMaster(conn net.Conn) {
	rb := NewRequestBuilder()
	rb.AddLine("PING")
	_, err := conn.Write(rb.Bytes())
	if err != nil {
		panic(err)
	}

	response := make([]byte, 1024)

	resLen, err := conn.Read(response)
	if err != nil {
		panic(err)
	}

	fmt.Println("Sent PING to master", string(response[:resLen]))
}
