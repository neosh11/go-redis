package cli

import (
	"github.com/codecrafters-io/redis-starter-go/app/base"
	"log"
	"os"
)

type Flag string

// mapping from command to array of arguments

func GetRedisConfig() *base.RedisConfig {
	var data = os.Args[1:]
	var index = 0

	port := "6379"
	replicaOf := ""

	for index < len(data) {
		if data[index] == "--port" {
			if index+1 < len(data) {
				port = data[index+1]
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

	return &base.RedisConfig{
		Port:      port,
		ReplicaOf: replicaOf,
	}
}
