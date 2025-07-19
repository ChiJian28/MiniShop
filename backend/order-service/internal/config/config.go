package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	RabbitMQ   RabbitMQConfig   `mapstructure:"rabbitmq"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	Order      OrderConfig      `mapstructure:"order"`
	Log        LogConfig        `mapstructure:"log"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

type ServerConfig struct {
	Port     int `mapstructure:"port"`
	GrpcPort int `mapstructure:"grpc_port"`
}

type DatabaseConfig struct {
	Driver          string         `mapstructure:"driver"`
	Postgres        PostgresConfig `mapstructure:"postgres"`
	MySQL           MySQLConfig    `mapstructure:"mysql"`
	MaxIdleConns    int            `mapstructure:"max_idle_conns"`
	MaxOpenConns    int            `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration  `mapstructure:"conn_max_lifetime"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
	TimeZone string `mapstructure:"timezone"`
}

type MySQLConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	DBName    string `mapstructure:"dbname"`
	Charset   string `mapstructure:"charset"`
	ParseTime bool   `mapstructure:"parse_time"`
	Loc       string `mapstructure:"loc"`
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

type RabbitMQConfig struct {
	URL           string `mapstructure:"url"`
	Exchange      string `mapstructure:"exchange"`
	Queue         string `mapstructure:"queue"`
	RoutingKey    string `mapstructure:"routing_key"`
	Durable       bool   `mapstructure:"durable"`
	AutoDelete    bool   `mapstructure:"auto_delete"`
	PrefetchCount int    `mapstructure:"prefetch_count"`
}

type KafkaConfig struct {
	Brokers   []string      `mapstructure:"brokers"`
	Topic     string        `mapstructure:"topic"`
	GroupID   string        `mapstructure:"group_id"`
	Partition int           `mapstructure:"partition"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type OrderConfig struct {
	OrderTimeout   time.Duration      `mapstructure:"order_timeout"`
	PaymentTimeout time.Duration      `mapstructure:"payment_timeout"`
	Retry          RetryConfig        `mapstructure:"retry"`
	Compensation   CompensationConfig `mapstructure:"compensation"`
}

type RetryConfig struct {
	MaxAttempts     int           `mapstructure:"max_attempts"`
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	MaxInterval     time.Duration `mapstructure:"max_interval"`
	Multiplier      float64       `mapstructure:"multiplier"`
}

type CompensationConfig struct {
	Enable        bool          `mapstructure:"enable"`
	CheckInterval time.Duration `mapstructure:"check_interval"`
	MaxRetryHours int           `mapstructure:"max_retry_hours"`
	BatchSize     int           `mapstructure:"batch_size"`
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
