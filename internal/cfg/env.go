package cfg

import (
	"encoding/json"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"strings"
)

func init() {
	str := os.Getenv("ASYNQ_QUEUES")
	if str != "" {
		json.Unmarshal([]byte(str), &cfg.AsynqConfig.Queues)
	}

	if !cfg.AsynqConfig.Queues.IsValid() {
		panic("invalid ASYNQ_QUEUES")
	}

}

type ConfigDatabase struct {
	DbConn string `env:"DB_CONNECTION_STRING"`
}

type Cache struct {
	CacheAddr string `env:"CACHE_ADDR"`
}

type AsynqConfig struct {
	Concurrency int `env:"ASYNQ_CONCURRENCY"`
	Queues      AsynqQueues
}

type AsynqQueues map[string]int

func (aq AsynqQueues) Contains(queueName string) bool {
	_, exists := aq[queueName]
	return exists
}

func (aq AsynqQueues) IsValid() bool {
	var (
		internalValid bool
		externalValid bool
	)

	for k, v := range cfg.AsynqConfig.Queues {
		if strings.Contains(k, "internal.") && v > 0 {
			internalValid = true
		}
		if strings.Contains(k, "external.") && v > 0 {
			externalValid = true
		}
	}

	return internalValid && externalValid
}

type Config struct {
	ConfigDatabase ConfigDatabase
	Cache          Cache
	AsynqConfig    AsynqConfig
}

var cfg Config

func Get() Config {
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}

func SetConfig(c Config) {
	cfg = c
}
