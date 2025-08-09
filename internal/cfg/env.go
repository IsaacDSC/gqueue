package cfg

import "github.com/ilyakaznacheev/cleanenv"

type ConfigDatabase struct {
	DbConn string `env:"DB_CONNECTION_STRING"`
}

type Cache struct {
	CacheAddr string `env:"CACHE_ADDR"`
}

type Config struct {
	ConfigDatabase ConfigDatabase
	Cache          Cache
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
