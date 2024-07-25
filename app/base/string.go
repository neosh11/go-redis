package base

import "strconv"

func BulkStringEncode(s string) string {
	return "$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n"
}
func BulkStringNil() string {
	return "$-1\r\n"
}

type RedisStringBuilder struct {
	value string
}

func NewRedisStringBuilder() RedisStringBuilder {
	return RedisStringBuilder{
		value: "",
	}
}

func (r *RedisStringBuilder) AddLine(line string) {
	r.value += line + "\r\n"
}

func (r *RedisStringBuilder) String() string {
	return r.value
}

func (r *RedisStringBuilder) BulkStringEncode() string {
	return BulkStringEncode(r.value)
}

type RequestBuilder struct {
	lines []string
}

func NewRequestBuilder() RequestBuilder {
	return RequestBuilder{
		lines: make([]string, 0),
	}
}

func (r *RequestBuilder) Reset() {
	r.lines = make([]string, 0)
}

func (r *RequestBuilder) AddLine(line string) {
	r.lines = append(r.lines, line)
}

func (r *RequestBuilder) String() string {
	rsb := NewRedisStringBuilder()
	rsb.AddLine("*" + strconv.Itoa(len(r.lines)))
	for _, line := range r.lines {
		rsb.AddLine("$" + strconv.Itoa(len(line)))
		rsb.AddLine(line)
	}
	return rsb.String()
}
func (r *RequestBuilder) Bytes() []byte {
	return []byte(r.String())
}
