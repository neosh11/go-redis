package base

type KvValue struct {
	Value  string
	Expiry int
}

type RedisConfig struct {
	Port      string
	ReplicaOf string
}

type Redis struct {
	Memory map[string]KvValue
	Config RedisConfig
}

func NewRedis(config *RedisConfig) *Redis {
	return &Redis{
		Memory: make(map[string]KvValue),
		Config: *config,
	}
}
