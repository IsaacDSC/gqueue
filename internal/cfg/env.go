package cfg

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

func init() {
	str := os.Getenv("WQ_QUEUES")
	if str != "" {
		json.Unmarshal([]byte(str), &cfg.AsynqConfig.Queues)
	}

	if !cfg.AsynqConfig.Queues.IsValid() {
		panic("invalid WQ_QUEUES")
	}

}

type ConfigDatabase struct {
	Driver string `env:"DB_DRIVER"`
	DbConn string `env:"DB_CONNECTION_STRING"`
}

type Cache struct {
	CacheAddr string `env:"CACHE_ADDR"`
}

type AsynqConfig struct {
	Concurrency int `env:"WQ_CONCURRENCY"`
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

func localDebug() {
	os.Setenv("WQ_QUEUES", `{"internal.default":1,"external.default":1}`)
	os.Setenv("CACHE_ADDR", "localhost:6379")
	os.Setenv("DB_DRIVER", "pg")
	os.Setenv("DB_CONNECTION_STRING", "postgresql://idsc:admin@localhost:5432/gqueue?sslmode=disable")
	os.Setenv("WQ_CONCURRENCY", "32")
	os.Setenv("WQ_QUEUES", `{"internal.critical": 7, "internal.high": 5, "internal.medium": 3, "internal.low": 1, "external.critical": 7, "external.high": 5, "external.medium": 3, "external.low": 1}`)
}
