package cli

import (
	"github.com/codecrafters-io/redis-starter-go/app/base"
	"log"
	"os"
	"strconv"
)

type Flag string

// mapping from command to array of arguments

func GetRedisConfig() *base.RedisConfig {
	var data = os.Args[1:]
	var index = 0

	port := 6379
	replicaOf := ""

	for index < len(data) {
		if data[index] == "--port" {
			if index+1 < len(data) {
				portStr := data[index+1]
				lPort, err := strconv.Atoi(portStr)
				if err != nil {
					log.Panic("Invalid port")
				}
				port = lPort
				index += 2
			} else {
				log.Panic("Invalid port")
			}
		} else if data[index] == "--replicaof" {
			if index+1 < len(data) {
				replicaOf = data[index+1]
				index += 2
			} else {
				log.Panic("Invalid replicaof")
			}
		} else {
			log.Panic("Invalid flag" + string(data[index]))
		}
	}

	// generate a UUID for replication id
	return &base.RedisConfig{
		Port:              port,
		ReplicaOf:         replicaOf,
		ReplicationId:     "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb",
		ReplicationOffset: 0,
	}
}
