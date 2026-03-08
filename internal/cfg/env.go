package cfg

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type ConfigDatabase struct {
	Driver string `env:"DB_DRIVER"`
	DbConn string `env:"DB_CONNECTION_STRING"`
}

type Cache struct {
	CacheAddr  string        `env:"CACHE_ADDR"`
	DefaultTTL time.Duration `env:"CACHE_DEFAULT_TTL" env-default:"24h"`
}

type AsynqConfig struct {
	Concurrency int `env:"WQ_CONCURRENCY"`
}

type ServerPort int

func (p ServerPort) String() string {
	return fmt.Sprintf(":%d", p)
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
		return fmt.Errorf("invalid WQ type: %s", wq)
	}
}

const (
	WQGooglePubSub WQ = "googlepubsub"
	WQAWS          WQ = "aws"
	WQRedis        WQ = "redis"
)

type Config struct {
	ProjectID      string `env:"PROJECT_ID"`
	SecretKey      string `env:"SECRET_KEY"`
	ConfigDatabase ConfigDatabase
	Cache          Cache
	AsynqConfig    AsynqConfig
	WQ             WQ `env:"WQ"`
	// InternalBaseURL TODO: será utilizado para buscar informações e não compartilhar banco de dados(backoffice, pubsub, task)
	InternalBaseURL     string `env:"INTERNAL_BASE_URL"`
	InternalServiceName string `env:"INTERNAL_SERVICE_NAME"`

	PubsubApiPort     ServerPort    `env:"PUBSUB_API_PORT" env-default:"8082"`
	TaskApiPort       ServerPort    `env:"TASK_API_PORT" env-default:"8083"`
	BackofficeApiPort ServerPort    `env:"BACKOFFICE_API_PORT" env-default:"8081"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"30s"` //TODO: porque não está sendo usado?

	MetricsEnabled           bool   `env:"METRICS_ENABLED" env-default:"true"`
	OTELExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" env-default:""`
	MaxConsumers             int    `env:"MAX_CONSUMERS" env-default:"10"`
	LogLevel                 int    `env:"LOG_LEVEL" env-default:"2"` // 0: debug, 1: info, 2: warn, 3: error
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

	if cfg.LogLevel < 0 || cfg.LogLevel > 3 {
		panic("invalid log level")
	}

	return cfg
}

func SetConfig(c Config) {
	cfg = c
}
