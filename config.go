package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

// Config defines the structure of the configuration file.
type Config struct {
	GitCommit      string        `yaml:"git_commit" envconfig:"DRAP_GIT_COMMIT"`
	GitTag         string        `yaml:"git_tag" envconfig:"DRAP_GIT_TAG"`
	BuildTime      string        `yaml:"build_time" envconfig:"DRAP_BUILD_TIME"`
	IsProduction   bool          `yaml:"is_production" envconfig:"DRAP_IS_PRODUCTION"`
	LogLevel       zapcore.Level `yaml:"log_level" envconfig:"DRAP_LOG_LEVEL"`
	LogFile        string        `yaml:"log_file" envconfig:"DRAP_LOG_FILE"`
	ProfilerEnable bool          `yaml:"profiler_enable" envconfig:"DRAP_PROFILER_ENABLE"`
	Server         ServerConfig  `yaml:"server"`
	Redis          RedisConfig   `yaml:"redis"`
	BoltDB         BoltDBConfig  `yaml:"boltdb"`
}

type ServerConfig struct {
	Host            string        `yaml:"host" envconfig:"DRAP_SERVER_HOST"`
	Port            string        `yaml:"port" envconfig:"DRAP_SERVER_PORT"`
	CertsFile       string        `yaml:"certs_file" envconfig:"DRAP_SERVER_CERTS_FILE"`
	KeyFile         string        `yaml:"key_file" envconfig:"DRAP_SERVER_KEY_FILE"`
	ReadTimeout     time.Duration `yaml:"read_timeout" envconfig:"DRAP_SERVER_READ_TIMEOUT"`
	WriteTimeout    time.Duration `yaml:"write_timeout" envconfig:"DRAP_SERVER_WRITE_TIMEOUT"`
	RequestTimeout  time.Duration `yaml:"request_timeout" envconfig:"DRAP_SERVER_REQUEST_TIMEOUT"` // Time to wait for a request to finish
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" envconfig:"DRAP_SERVER_SHUTDOWN_TIMEOUT"`
}

type RedisConfig struct {
	Host          string        `yaml:"host" envconfig:"DRAP_REDIS_HOST"`
	Port          string        `yaml:"port" envconfig:"DRAP_REDIS_PORT"`
	DialTimeout   time.Duration `yaml:"dial_timeout" envconfig:"DRAP_REDIS_DIAL_TIMEOUT"`
	ReadTimeout   time.Duration `yaml:"read_timeout" envconfig:"DRAP_REDIS_READ_TIMEOUT"`
	WriteTimeout  time.Duration `yaml:"write_timeout" envconfig:"DRAP_REDIS_WRITE_TIMEOUT"`
	PoolSize      int           `yaml:"pool_size" envconfig:"DRAP_REDIS_POOL_SIZE"`
	PoolTimeout   time.Duration `yaml:"pool_timeout" envconfig:"DRAP_REDIS_POOL_TIMEOUT"`
	Username      string        `yaml:"username" envconfig:"DRAP_REDIS_USERNAME"`
	Password      string        `yaml:"password" envconfig:"DRAP_REDIS_PASSWORD"`
	DatabaseIndex int           `yaml:"db_index" envconfig:"DRAP_REDIS_DATABASE_INDEX"`
}

type BoltDBConfig struct {
	FilePath   string        `yaml:"filepath" envconfig:"DRAP_BOLTDB_FILE_PATH"`
	Timeout    time.Duration `yaml:"timeout" envconfig:"DRAP_BOLTDB_TIMEOUT"`
	BucketName string        `yaml:"bucket_name" envconfig:"DRAP_BOLTDB_BUCKET_NAME"`
}

// LoadConfigFile provides an instance of config structure for the all application.
func LoadConfigFile(configFile string) (*Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cfg := &Config{}
	yd := yaml.NewDecoder(file)
	err = yd.Decode(cfg)

	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadConfigEnv reads the environments variables and provides an instance of the App config.
func LoadConfigEnvs(prefix string, config *Config) error {
	return envconfig.Process(prefix, config)
}

// InitConfig setup defaults values for non provided parameters
// and configures build tags values to be used if provided.
func InitConfig(config *Config, gitCommit, gitTag, buildTime string) error {
	if len(gitCommit) != 0 {
		config.GitCommit = gitCommit
	}

	if len(gitTag) != 0 {
		config.GitTag = gitTag
	}

	if len(buildTime) != 0 {
		config.BuildTime = buildTime
	}

	if len(config.Server.Host) == 0 || len(config.Server.Port) == 0 {
		return errors.New("make sure to set valid server address and port in configuration file")
	}

	if len(config.Redis.Host) == 0 || len(config.Redis.Port) == 0 {
		return errors.New("make sure to set valid redis address and port in configuration file")
	}

	return nil
}

// LoadAndInitConfigs loads in order the configs from various predefined sources
// then build the App configuration data.
func LoadAndInitConfigs(gitCommit, gitTag, buildTime string) (*Config, error) {
	// Setup the yaml configuration from file.
	config, err := LoadConfigFile("./config.yml")
	if err != nil {
		return config, fmt.Errorf("failed to load configurations from file: %s", err)
	}

	// Set the environment configuration.
	err = godotenv.Load("./config.env")
	if err != nil {
		return config, fmt.Errorf("failed to set environment configurations: %s", err)
	}

	// Use environment variables with prefix `DRAP`.
	err = LoadConfigEnvs("DRAP", config)
	if err != nil {
		return config, fmt.Errorf("failed to load configurations from environment: %s", err)
	}

	err = InitConfig(config, gitCommit, gitTag, buildTime)
	if err != nil {
		return config, fmt.Errorf("failed to initialize configurations: %s", err)
	}
	return config, nil
}
