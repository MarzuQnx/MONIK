package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig       `yaml:"server"`
	Database  DatabaseConfig     `yaml:"database"`
	Router    RouterConfig       `yaml:"router"`
	Logging   LoggingConfig      `yaml:"logging"`
	WAN       WANDetectionConfig `yaml:"wan"`
	Worker    WorkerPoolConfig   `yaml:"worker"`
	WebSocket WebSocketConfig    `yaml:"websocket"`
	Metrics   MetricsConfig      `yaml:"metrics"`
	Dashboard DashboardConfig    `yaml:"dashboard"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Address returns the full server address
func (s ServerConfig) Address() string {
	return s.Host + ":" + strconv.Itoa(s.Port)
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Path        string `yaml:"path"`
	MaxOpenConn int    `yaml:"max_open_conn"`
	MaxIdleConn int    `yaml:"max_idle_conn"`
}

// RouterConfig holds MikroTik router configuration
type RouterConfig struct {
	IP       string        `yaml:"ip"`
	Port     int           `yaml:"port"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// WANDetectionConfig holds WAN/ISP detection configuration
type WANDetectionConfig struct {
	Enabled          bool          `yaml:"enabled"`
	DetectionMethod  string        `yaml:"detection_method"` // auto, manual, hybrid
	ManualInterface  string        `yaml:"manual_interface"`
	CacheDuration    time.Duration `yaml:"cache_duration"`
	TrafficThreshold uint64        `yaml:"traffic_threshold"` // bytes per minute
}

// WorkerPoolConfig holds worker pool configuration
type WorkerPoolConfig struct {
	MaxWorkers                     int           `yaml:"max_workers"`
	QueueSize                      int           `yaml:"queue_size"`
	WorkerTimeout                  time.Duration `yaml:"worker_timeout"`
	LoadThreshold                  float64       `yaml:"load_threshold"` // 0.0 to 1.0
	RebalanceEvery                 time.Duration `yaml:"rebalance_every"`
	LoadBalancingStrategy          string        `yaml:"load_balancing_strategy"` // round_robin, least_connections, random, weighted
	CircuitBreakerEnabled          bool          `yaml:"circuit_breaker_enabled"`
	CircuitBreakerFailureThreshold int           `yaml:"circuit_breaker_failure_threshold"`
	CircuitBreakerRecoveryTimeout  time.Duration `yaml:"circuit_breaker_recovery_timeout"`
	CircuitBreakerHalfOpenMaxCalls int           `yaml:"circuit_breaker_half_open_max_calls"`
}

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	Enabled             bool          `yaml:"enabled"`
	ReadTimeout         time.Duration `yaml:"read_timeout"`
	WriteTimeout        time.Duration `yaml:"write_timeout"`
	PingPeriod          time.Duration `yaml:"ping_period"`
	MaxMessageSize      int64         `yaml:"max_message_size"`
	BroadcastBufferSize int           `yaml:"broadcast_buffer_size"`
	EnableMetrics       bool          `yaml:"enable_metrics"`
}

// MetricsConfig holds metrics collection configuration
type MetricsConfig struct {
	Enabled             bool          `yaml:"enabled"`
	CollectionInterval  time.Duration `yaml:"collection_interval"`
	EnableHealthCheck   bool          `yaml:"enable_health_check"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	BroadcastMetrics    bool          `yaml:"broadcast_metrics"`
}

// DashboardConfig holds dashboard configuration
type DashboardConfig struct {
	Enabled                bool          `yaml:"enabled"`
	RealTimeUpdateInterval time.Duration `yaml:"real_time_update_interval"`
	MaxConnections         int           `yaml:"max_connections"`
	EnableMetrics          bool          `yaml:"enable_metrics"`
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Path:        getEnv("DB_PATH", "data/monik.db"),
			MaxOpenConn: getEnvAsInt("DB_MAX_OPEN_CONN", 25),
			MaxIdleConn: getEnvAsInt("DB_MAX_IDLE_CONN", 5),
		},
		Router: RouterConfig{
			IP:       getEnv("ROUTER_IP", "192.168.88.1"),
			Port:     getEnvAsInt("ROUTER_PORT", 8728),
			Username: getEnv("ROUTER_USERNAME", "admin"),
			Password: getEnv("ROUTER_PASSWORD", ""),
			Timeout:  getEnvAsDuration("ROUTER_TIMEOUT", 30*time.Second),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		WAN: WANDetectionConfig{
			Enabled:          getEnvAsBool("WAN_ENABLED", true),
			DetectionMethod:  getEnv("WAN_DETECTION_METHOD", "auto"),
			ManualInterface:  getEnv("WAN_MANUAL_INTERFACE", ""),
			CacheDuration:    getEnvAsDuration("WAN_CACHE_DURATION", 5*time.Minute),
			TrafficThreshold: getEnvAsUint64("WAN_TRAFFIC_THRESHOLD", 1024*1024), // 1MB per minute
		},
		Worker: WorkerPoolConfig{
			MaxWorkers:                     getEnvAsInt("WORKER_MAX_WORKERS", 4),
			QueueSize:                      getEnvAsInt("WORKER_QUEUE_SIZE", 100),
			WorkerTimeout:                  getEnvAsDuration("WORKER_TIMEOUT", 30*time.Second),
			LoadThreshold:                  getEnvAsFloat64("WORKER_LOAD_THRESHOLD", 0.8),
			RebalanceEvery:                 getEnvAsDuration("WORKER_REBALANCE_EVERY", 5*time.Minute),
			LoadBalancingStrategy:          getEnv("WORKER_LOAD_BALANCING_STRATEGY", "round_robin"),
			CircuitBreakerEnabled:          getEnvAsBool("WORKER_CIRCUIT_BREAKER_ENABLED", true),
			CircuitBreakerFailureThreshold: getEnvAsInt("WORKER_CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
			CircuitBreakerRecoveryTimeout:  getEnvAsDuration("WORKER_CIRCUIT_BREAKER_RECOVERY_TIMEOUT", 60*time.Second),
			CircuitBreakerHalfOpenMaxCalls: getEnvAsInt("WORKER_CIRCUIT_BREAKER_HALF_OPEN_MAX_CALLS", 3),
		},
		WebSocket: WebSocketConfig{
			Enabled:             getEnvAsBool("WEBSOCKET_ENABLED", true),
			ReadTimeout:         getEnvAsDuration("WEBSOCKET_READ_TIMEOUT", 60*time.Second),
			WriteTimeout:        getEnvAsDuration("WEBSOCKET_WRITE_TIMEOUT", 10*time.Second),
			PingPeriod:          getEnvAsDuration("WEBSOCKET_PING_PERIOD", 54*time.Second),
			MaxMessageSize:      getEnvAsInt64("WEBSOCKET_MAX_MESSAGE_SIZE", 512),
			BroadcastBufferSize: getEnvAsInt("WEBSOCKET_BROADCAST_BUFFER_SIZE", 10000),
			EnableMetrics:       getEnvAsBool("WEBSOCKET_ENABLE_METRICS", true),
		},
		Metrics: MetricsConfig{
			Enabled:             getEnvAsBool("METRICS_ENABLED", true),
			CollectionInterval:  getEnvAsDuration("METRICS_COLLECTION_INTERVAL", 30*time.Second),
			EnableHealthCheck:   getEnvAsBool("METRICS_ENABLE_HEALTH_CHECK", true),
			HealthCheckInterval: getEnvAsDuration("METRICS_HEALTH_CHECK_INTERVAL", 60*time.Second),
			BroadcastMetrics:    getEnvAsBool("METRICS_BROADCAST_METRICS", true),
		},
		Dashboard: DashboardConfig{
			Enabled:                getEnvAsBool("DASHBOARD_ENABLED", true),
			RealTimeUpdateInterval: getEnvAsDuration("DASHBOARD_REAL_TIME_UPDATE_INTERVAL", 1*time.Second),
			MaxConnections:         getEnvAsInt("DASHBOARD_MAX_CONNECTIONS", 100),
			EnableMetrics:          getEnvAsBool("DASHBOARD_ENABLE_METRICS", true),
		},
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsDuration gets an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as bool or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsUint64 gets an environment variable as uint64 or returns a default value
func getEnvAsUint64(key string, defaultValue uint64) uint64 {
	if value := os.Getenv(key); value != "" {
		if uintValue, err := strconv.ParseUint(value, 10, 64); err == nil {
			return uintValue
		}
	}
	return defaultValue
}

// getEnvAsFloat64 gets an environment variable as float64 or returns a default value
func getEnvAsFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvAsInt64 gets an environment variable as int64 or returns a default value
func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
