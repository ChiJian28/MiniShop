package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Redis        RedisConfig        `mapstructure:"redis"`
	CacheService CacheServiceConfig `mapstructure:"cache_service"`
	RabbitMQ     RabbitMQConfig     `mapstructure:"rabbitmq"`
	Kafka        KafkaConfig        `mapstructure:"kafka"`
	Seckill      SeckillConfig      `mapstructure:"seckill"`
	Log          LogConfig          `mapstructure:"log"`
	Monitoring   MonitoringConfig   `mapstructure:"monitoring"`
}

type ServerConfig struct {
	Port     int `mapstructure:"port"`
	GrpcPort int `mapstructure:"grpc_port"`
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	MaxRetries   int           `mapstructure:"max_retries"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type CacheServiceConfig struct {
	Host    string        `mapstructure:"host"`
	Port    int           `mapstructure:"port"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type RabbitMQConfig struct {
	URL        string `mapstructure:"url"`
	Exchange   string `mapstructure:"exchange"`
	Queue      string `mapstructure:"queue"`
	RoutingKey string `mapstructure:"routing_key"`
	Durable    bool   `mapstructure:"durable"`
	AutoDelete bool   `mapstructure:"auto_delete"`
}

type KafkaConfig struct {
	Brokers   []string      `mapstructure:"brokers"`
	Topic     string        `mapstructure:"topic"`
	Partition int           `mapstructure:"partition"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type SeckillConfig struct {
	MaxConcurrentRequests int                  `mapstructure:"max_concurrent_requests"`
	QueueSize             int                  `mapstructure:"queue_size"`
	RequestTimeout        time.Duration        `mapstructure:"request_timeout"`
	RateLimit             RateLimitConfig      `mapstructure:"rate_limit"`
	CircuitBreaker        CircuitBreakerConfig `mapstructure:"circuit_breaker"`
	Degradation           DegradationConfig    `mapstructure:"degradation"`
}

type RateLimitConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second"`
	BurstSize         int `mapstructure:"burst_size"`
}

type CircuitBreakerConfig struct {
	FailureThreshold int           `mapstructure:"failure_threshold"`
	RecoveryTimeout  time.Duration `mapstructure:"recovery_timeout"`
	HalfOpenRequests int           `mapstructure:"half_open_requests"`
}

type DegradationConfig struct {
	Enable          bool    `mapstructure:"enable"`
	Threshold       float64 `mapstructure:"threshold"`
	ResponseMessage string  `mapstructure:"response_message"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

type MonitoringConfig struct {
	Enable bool   `mapstructure:"enable"`
	Port   int    `mapstructure:"port"`
	Path   string `mapstructure:"path"`
}

func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
