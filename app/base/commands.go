package base

import (
	base642 "encoding/base64"
	"fmt"
	"net"
	"strconv"
	"time"
)

func (r Redis) Set(args []string) string {
	// check first 2 arguments
	if len(args) < 2 {
		return "-ERR wrong number of arguments for 'set' command\r\n"
	}
	expiry := -1
	if len(args) >= 4 && args[2] == "px" {
		ex, lErr := strconv.Atoi(args[3])
		if lErr != nil {
			return "-ERR invalid expiry value\r\n"
		}
		expiry = int((int64)(time.Now().UnixMilli())) + ex
	}
	r.Memory[args[0]] = KvValue{
		Value:  args[1],
		Expiry: expiry,
	}
	return "+OK\r\n"
}

func (r Redis) Get(args []string) string {
	if len(args) < 1 {
		return "-ERR wrong number of arguments for 'get' command\r\n"
	}
	value, ok := r.Memory[args[0]]

	if !ok {
		return BulkStringNil()
	}
	if value.Expiry != -1 {
		if value.Expiry < int((int64)(time.Now().UnixMilli())) {
			delete(r.Memory, args[0])
			return BulkStringNil()
		}
	}
	return BulkStringEncode(value.Value)
}

func (r Redis) Echo(args []string) string {
	returnVal := ""
	if len(args) > 0 {
		returnVal = args[0]
	}
	return BulkStringEncode(returnVal)
}

func (r Redis) Ping() string {
	return "+PONG\r\n"
}

func (r Redis) REPLCONF(args []string) string {
	if len(args) < 2 {
		return "-ERR wrong number of arguments for 'replconf' command\r\n"
	}
	if args[0] == "listening-port" {
		port, lErr := strconv.Atoi(args[1])
		if lErr != nil {
			return "-ERR invalid port number\r\n"
		}
		fmt.Println("Listening port updated to", port)
		return "+OK\r\n"
	} else if args[0] == "capa" && args[1] == "psync2" {
		return "+OK\r\n"
	}
	return "-ERR invalid argument for 'replconf' command\r\n"
}

func (r Redis) PSYNC(args []string, req net.Conn) []byte {
	if len(args) < 2 {
		return []byte("-ERR wrong number of arguments for 'psync' command\r\n")
	}
	if args[0] == "?" && args[1] == "-1" {
		resp := "+FULLRESYNC " + r.Config.ReplicationId + " 0"
		_, err := req.Write([]byte(BulkStringEncode(resp)))
		if err != nil {
			return []byte("- Err writing to master failed")
		}
		// send an RDB dump to the replica
		base64 := "UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog=="
		// convert the base64 string to bytes
		rdbDump, err := base642.StdEncoding.DecodeString(base64)
		if err != nil {
			return []byte("- Err decoding base64 string")
		}
		rdb := []byte("$" + strconv.Itoa(len(rdbDump)) + "\r\n")
		// add rdbDump
		rdb = append(rdb, rdbDump...)
		return rdb
	}
	return []byte("-ERR invalid argument for 'psync' command\r\n" + args[0] + args[1])
}

func (r Redis) Info(args []string) string {
	if len(args) < 1 {
		return "-ERR wrong number of arguments for 'info' command\r\n"
	}
	if args[0] == "replication" {
		responseBuilder := NewRedisStringBuilder()
		responseBuilder.AddLine("# Replication")
		if r.Config.ReplicaOf == "" {
			responseBuilder.AddLine("role:master")
			responseBuilder.AddLine("master_replid:" + r.Config.ReplicationId)
			responseBuilder.AddLine("master_repl_offset:" + strconv.Itoa(r.Config.ReplicationOffset))
			return BulkStringEncode(responseBuilder.String())

		} else {
			responseBuilder.AddLine("role:slave")
			return BulkStringEncode(responseBuilder.String())
		}

	} else {
		return "-ERR invalid argument for 'info' command\r\n"
	}
}
