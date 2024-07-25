package base

type KvValue struct {
	Value  string
	Expiry int
}

type Redis struct {
	Memory map[string]KvValue
	port   string
}

func NewRedis(port string) *Redis {
	return &Redis{
		Memory: make(map[string]KvValue),
		port:   port,
	}
}
