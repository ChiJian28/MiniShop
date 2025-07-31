package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server           ServerConfig           `mapstructure:"server"`
	Database         DatabaseConfig         `mapstructure:"database"`
	Redis            RedisConfig            `mapstructure:"redis"`
	Inventory        InventoryConfig        `mapstructure:"inventory"`
	ExternalServices ExternalServicesConfig `mapstructure:"external_services"`
	Log              LogConfig              `mapstructure:"log"`
	Monitoring       MonitoringConfig       `mapstructure:"monitoring"`
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

type InventoryConfig struct {
	DefaultStock      int64              `mapstructure:"default_stock"`
	LowStockThreshold int64              `mapstructure:"low_stock_threshold"`
	HealthCheck       HealthCheckConfig  `mapstructure:"health_check"`
	Sync              SyncConfig         `mapstructure:"sync"`
	Compensation      CompensationConfig `mapstructure:"compensation"`
}

type HealthCheckConfig struct {
	Enable         bool          `mapstructure:"enable"`
	Interval       time.Duration `mapstructure:"interval"`
	Tolerance      int64         `mapstructure:"tolerance"`
	AlertThreshold int64         `mapstructure:"alert_threshold"`
}

type SyncConfig struct {
	BatchSize  int           `mapstructure:"batch_size"`
	Timeout    time.Duration `mapstructure:"timeout"`
	RetryTimes int           `mapstructure:"retry_times"`
}

type CompensationConfig struct {
	Enable        bool          `mapstructure:"enable"`
	CheckInterval time.Duration `mapstructure:"check_interval"`
	AutoFix       bool          `mapstructure:"auto_fix"`
	MaxFixAmount  int64         `mapstructure:"max_fix_amount"`
}

type ExternalServicesConfig struct {
	CacheService ExternalServiceConfig `mapstructure:"cache_service"`
	OrderService ExternalServiceConfig `mapstructure:"order_service"`
}

type ExternalServiceConfig struct {
	Host    string        `mapstructure:"host"`
	Port    int           `mapstructure:"port"`
	Timeout time.Duration `mapstructure:"timeout"`
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
