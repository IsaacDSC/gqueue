package cfg

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type ConfigDatabase struct {
	Driver string `env:"DB_DRIVER"`
	DbConn string `env:"DB_CONNECTION_STRING"`
}

type Cache struct {
	CacheAddr  string        `env:"CACHE_ADDR"`
	DefaultTTL time.Duration `env:"CACHE_DEFAULT_TTL" default:"24h"`
}

type AsynqConfig struct {
	Concurrency int `env:"WQ_CONCURRENCY"`
}

type WQ string

func (wq WQ) String() string {
	return string(wq)
}

func (wq WQ) IsValid() error {
	switch wq {
	case WQGooglePubSub, WQAWS, WQRedis:
		return nil
	default:
		return fmt.Errorf("invalid worker type: %s", wq)
	}
}

const (
	WQGooglePubSub WQ = "googlepubsub"
	WQAWS          WQ = "aws"
	WQRedis        WQ = "redis"
)

type Config struct {
	ConfigDatabase ConfigDatabase
	Cache          Cache
	AsynqConfig    AsynqConfig
	WQ             WQ `env:"WQ"`
}

var cfg Config

func Get() Config {
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		panic(err)
	}

	if err := cfg.WQ.IsValid(); err != nil {
		panic(err)
	}

	return cfg
}

func SetConfig(c Config) {
	cfg = c
}
