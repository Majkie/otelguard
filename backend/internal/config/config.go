package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all application configuration
type Config struct {
	Server     ServerConfig
	Postgres   PostgresConfig
	ClickHouse ClickHouseConfig
	Auth       AuthConfig
	Sampler    SamplerConfig
	GRPC       GRPCConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int           `envconfig:"PORT" default:"8080"`
	Environment  string        `envconfig:"ENV" default:"development"`
	ReadTimeout  time.Duration `envconfig:"READ_TIMEOUT" default:"30s"`
	WriteTimeout time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout  time.Duration `envconfig:"IDLE_TIMEOUT" default:"60s"`
}

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host            string        `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port            int           `envconfig:"POSTGRES_PORT" default:"5432"`
	User            string        `envconfig:"POSTGRES_USER" default:"otelguard"`
	Password        string        `envconfig:"POSTGRES_PASSWORD" default:"otelguard"`
	Database        string        `envconfig:"POSTGRES_DB" default:"otelguard"`
	SSLMode         string        `envconfig:"POSTGRES_SSLMODE" default:"disable"`
	MaxOpenConns    int           `envconfig:"POSTGRES_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"POSTGRES_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"POSTGRES_CONN_MAX_LIFETIME" default:"5m"`
}

// DSN returns the PostgreSQL connection string
func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// MigrationDSN returns the PostgreSQL connection URL for golang-migrate
func (c PostgresConfig) MigrationDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// ClickHouseConfig holds ClickHouse connection configuration
type ClickHouseConfig struct {
	Host        string        `envconfig:"CLICKHOUSE_HOST" default:"localhost"`
	Port        int           `envconfig:"CLICKHOUSE_PORT" default:"9000"`
	Database    string        `envconfig:"CLICKHOUSE_DB" default:"otelguard"`
	User        string        `envconfig:"CLICKHOUSE_USER" default:"default"`
	Password    string        `envconfig:"CLICKHOUSE_PASSWORD" default:""`
	DialTimeout time.Duration `envconfig:"CLICKHOUSE_DIAL_TIMEOUT" default:"5s"`
	MaxOpenConn int           `envconfig:"CLICKHOUSE_MAX_OPEN_CONN" default:"10"`
	MaxIdleConn int           `envconfig:"CLICKHOUSE_MAX_IDLE_CONN" default:"5"`

	// Batch writer settings
	BatchSize     int           `envconfig:"CLICKHOUSE_BATCH_SIZE" default:"1000"`
	FlushInterval time.Duration `envconfig:"CLICKHOUSE_FLUSH_INTERVAL" default:"5s"`
	MaxRetries    int           `envconfig:"CLICKHOUSE_MAX_RETRIES" default:"3"`
	RetryDelay    time.Duration `envconfig:"CLICKHOUSE_RETRY_DELAY" default:"1s"`
	AsyncWrite    bool          `envconfig:"CLICKHOUSE_ASYNC_WRITE" default:"true"`
}

// MigrationDSN returns the ClickHouse connection URL for golang-migrate
func (c ClickHouseConfig) MigrationDSN() string {
	// golang-migrate clickhouse driver format: clickhouse://host:port/database?query_params
	return fmt.Sprintf(
		"clickhouse://%s:%d/%s?username=%s&password=%s&x-multi-statement=true",
		c.Host, c.Port, c.Database, c.User, c.Password,
	)
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret          string        `envconfig:"JWT_SECRET" required:"true"`
	JWTExpiration      time.Duration `envconfig:"JWT_EXPIRATION" default:"24h"`
	APIKeySalt         string        `envconfig:"API_KEY_SALT" required:"true"`
	BcryptCost         int           `envconfig:"BCRYPT_COST" default:"12"`
	RefreshTokenExpiry time.Duration `envconfig:"REFRESH_TOKEN_EXPIRY" default:"168h"` // 7 days
}

// SamplerConfig holds trace sampling configuration
type SamplerConfig struct {
	Enabled       bool    `envconfig:"SAMPLER_ENABLED" default:"false"`
	Type          string  `envconfig:"SAMPLER_TYPE" default:"always"`     // always, random, rate_limit, consistent, priority
	Rate          float64 `envconfig:"SAMPLER_RATE" default:"1.0"`        // 0.0 to 1.0 for random/consistent
	MaxPerSecond  int     `envconfig:"SAMPLER_MAX_PER_SEC" default:"100"` // For rate_limit sampling
	SampleErrors  bool    `envconfig:"SAMPLER_ERRORS" default:"true"`     // Always sample errors
	SampleSlow    bool    `envconfig:"SAMPLER_SLOW" default:"true"`       // Always sample slow traces
	SlowThreshold int     `envconfig:"SAMPLER_SLOW_MS" default:"5000"`    // Threshold for slow in ms
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Enabled          bool `envconfig:"GRPC_ENABLED" default:"true"`
	Port             int  `envconfig:"GRPC_PORT" default:"4317"`
	MaxRecvMsgSize   int  `envconfig:"GRPC_MAX_RECV_MSG_SIZE" default:"16777216"` // 16MB
	MaxSendMsgSize   int  `envconfig:"GRPC_MAX_SEND_MSG_SIZE" default:"16777216"` // 16MB
	EnableReflection bool `envconfig:"GRPC_ENABLE_REFLECTION" default:"true"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("OTELGUARD", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}
	return &cfg, nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}
