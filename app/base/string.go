package base

import "strconv"

func BulkStringEncode(s string) string {
	return "$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n"
}
func BulkStringNil() string {
	return "$-1\r\n"
}

type ResponseBuilder struct {
	value string
}

func NewResponseBuilder() ResponseBuilder {
	return ResponseBuilder{
		value: "",
	}
}

func (r *ResponseBuilder) AddLine(line string) {
	r.value += line + "\r\n"
}

func (r *ResponseBuilder) String() string {
	return r.value
}

func (r *ResponseBuilder) BulkStringEncode() string {
	return BulkStringEncode(r.value)
}
