package main

import (
	"errors"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

// Config defines the structure of the configuration file.
type Config struct {
	GitCommit    string        `yaml:"git_commit", envconfig:"DRAP_GIT_COMMIT"`
	GitTag       string        `yaml:"git_tag", envconfig:"DRAP_GIT_TAG"`
	BuildTime    string        `yaml:"build_time", envconfig:"DRAP_BUILD_TIME"`
	IsProduction bool          `yaml:"is_production", envconfig:"DRAP_IS_PRODUCTION"`
	LogLevel     zapcore.Level `yaml:"log_level", envconfig:"DRAP_LOG_LEVEL"`
	LogFile      string        `yaml:"log_file", envconfig:"DRAP_LOG_FILE"`

	Server struct {
		Host            string        `yaml:"host", envconfig:"DRAP_SERVER_HOST"`
		Port            string        `yaml:"port", envconfig:"DRAP_SERVER_PORT"`
		CertsFile       string        `yaml:"certs_file", envconfig:"DRAP_SERVER_CERTS_FILE"`
		KeyFile         string        `yaml:"key_file", envconfig:"DRAP_SERVER_KEY_FILE"`
		ReadTimeout     time.Duration `yaml:"read_timeout", envconfig:"DRAP_SERVER_READ_TIMEOUT"`
		WriteTimeout    time.Duration `yaml:"write_timeout", envconfig:"DRAP_SERVER_WRITE_TIMEOUT"`
		RequestTimeout  time.Duration `yaml:"request_timeout", envconfig:"DRAP_SERVER_REQUEST_TIMEOUT"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout", envconfig:"DRAP_SERVER_SHUTDOWN_TIMEOUT"`
	} `yaml:"server"`

	Redis struct {
		Host          string `yaml:"host", envconfig:"DRAP_REDIS_HOST"`
		Port          string `yaml:"port", envconfig:"DRAP_REDIS_PORT"`
		DialTimeout   int    `yaml:"dial_timeout", envconfig:"DRAP_REDIS_DIAL_TIMEOUT"`
		ReadTimeout   int    `yaml:"read_timeout", envconfig:"DRAP_REDIS_READ_TIMEOUT"`
		WriteTimeout  int    `yaml:"write_timeout", envconfig:"DRAP_REDIS_WRITE_TIMEOUT"`
		PoolSize      int    `yaml:"pool_size", envconfig:"DRAP_REDIS_POOL_SIZE"`
		PoolTimeout   int    `yaml:"pool_timeout", envconfig:"DRAP_REDIS_POOL_TIMEOUT"`
		Username      string `yaml:"username", envconfig:"DRAP_REDIS_USERNAME"`
		Password      string `yaml:"password", envconfig:"DRAP_REDIS_PASSWORD"`
		DatabaseIndex int    `yaml:"db_index", envconfig:"DRAP_REDIS_DATABASE_INDEX"`
	} `yaml:"redis"`
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
