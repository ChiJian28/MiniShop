package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server         ServerConfig         `mapstructure:"server"`
	Services       ServicesConfig       `mapstructure:"services"`
	Redis          RedisConfig          `mapstructure:"redis"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
	Auth           AuthConfig           `mapstructure:"auth"`
	CORS           CORSConfig           `mapstructure:"cors"`
	Routing        RoutingConfig        `mapstructure:"routing"`
	Monitoring     MonitoringConfig     `mapstructure:"monitoring"`
	Log            LogConfig            `mapstructure:"log"`
	Cache          CacheConfig          `mapstructure:"cache"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
	Retry          RetryConfig          `mapstructure:"retry"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type ServicesConfig struct {
	CacheService     ServiceConfig `mapstructure:"cache-service"`
	SeckillService   ServiceConfig `mapstructure:"seckill-service"`
	OrderService     ServiceConfig `mapstructure:"order-service"`
	InventoryService ServiceConfig `mapstructure:"inventory-service"`
}

type ServiceConfig struct {
	URL             string        `mapstructure:"url"`
	Timeout         time.Duration `mapstructure:"timeout"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxConnsPerHost int           `mapstructure:"max_conns_per_host"`
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

type RateLimitConfig struct {
	Enable    bool                           `mapstructure:"enable"`
	Global    GlobalRateLimitConfig          `mapstructure:"global"`
	User      UserRateLimitConfig            `mapstructure:"user"`
	IP        IPRateLimitConfig              `mapstructure:"ip"`
	Endpoints map[string]EndpointLimitConfig `mapstructure:"endpoints"`
}

type GlobalRateLimitConfig struct {
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	Burst             int     `mapstructure:"burst"`
}

type UserRateLimitConfig struct {
	RequestsPerSecond float64       `mapstructure:"requests_per_second"`
	Burst             int           `mapstructure:"burst"`
	Window            time.Duration `mapstructure:"window"`
}

type IPRateLimitConfig struct {
	RequestsPerSecond float64       `mapstructure:"requests_per_second"`
	Burst             int           `mapstructure:"burst"`
	Window            time.Duration `mapstructure:"window"`
}

type EndpointLimitConfig struct {
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	Burst             int     `mapstructure:"burst"`
}

type AuthConfig struct {
	Enable        bool            `mapstructure:"enable"`
	JWTSecret     string          `mapstructure:"jwt_secret"`
	TokenExpire   time.Duration   `mapstructure:"token_expire"`
	RefreshExpire time.Duration   `mapstructure:"refresh_expire"`
	Signature     SignatureConfig `mapstructure:"signature"`
	Whitelist     []string        `mapstructure:"whitelist"`
}

type SignatureConfig struct {
	Enable          bool          `mapstructure:"enable"`
	Secret          string        `mapstructure:"secret"`
	Expire          time.Duration `mapstructure:"expire"`
	RequiredHeaders []string      `mapstructure:"required_headers"`
}

type CORSConfig struct {
	Enable           bool     `mapstructure:"enable"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

type RoutingConfig struct {
	PrefixMapping map[string]string `mapstructure:"prefix_mapping"`
	HealthChecks  map[string]string `mapstructure:"health_checks"`
	LoadBalancer  string            `mapstructure:"load_balancer"`
}

type MonitoringConfig struct {
	Enable               bool          `mapstructure:"enable"`
	MetricsPort          int           `mapstructure:"metrics_port"`
	MetricsPath          string        `mapstructure:"metrics_path"`
	Tracing              TracingConfig `mapstructure:"tracing"`
	SlowRequestThreshold time.Duration `mapstructure:"slow_request_threshold"`
}

type TracingConfig struct {
	Enable     bool    `mapstructure:"enable"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

type LogConfig struct {
	Level     string          `mapstructure:"level"`
	Format    string          `mapstructure:"format"`
	File      string          `mapstructure:"file"`
	AccessLog AccessLogConfig `mapstructure:"access_log"`
}

type AccessLogConfig struct {
	Enable bool   `mapstructure:"enable"`
	File   string `mapstructure:"file"`
	Format string `mapstructure:"format"`
}

type CacheConfig struct {
	Enable        bool                     `mapstructure:"enable"`
	DefaultExpire time.Duration            `mapstructure:"default_expire"`
	Strategies    map[string]time.Duration `mapstructure:"strategies"`
}

type CircuitBreakerConfig struct {
	Enable           bool          `mapstructure:"enable"`
	FailureThreshold int           `mapstructure:"failure_threshold"`
	SuccessThreshold int           `mapstructure:"success_threshold"`
	Timeout          time.Duration `mapstructure:"timeout"`
	MaxRequests      int           `mapstructure:"max_requests"`
}

type RetryConfig struct {
	Enable          bool          `mapstructure:"enable"`
	MaxAttempts     int           `mapstructure:"max_attempts"`
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	MaxInterval     time.Duration `mapstructure:"max_interval"`
	Multiplier      float64       `mapstructure:"multiplier"`
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
