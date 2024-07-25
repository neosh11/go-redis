package base

import (
	"strconv"
	"time"
)

func BulkStringEncode(s string) string {
	return "$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n"
}
func BulkStringNil() string {
	return "$-1\r\n"
}

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

func (r Redis) Info(args []string) string {
	if len(args) < 1 {
		return "-ERR wrong number of arguments for 'info' command\r\n"
	}
	if args[0] == "replication" {
		return BulkStringEncode("# Replication\r\nrole:master\r\n")

	} else {
		return "-ERR invalid argument for 'info' command\r\n"
	}

}
