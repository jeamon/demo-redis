package main

import (
	"errors"
	"os"

	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

// Config defines the structure of the configuration file.
type Config struct {
	GitCommit    string        `yaml:"git_commit"`
	GitTag       string        `yaml:"git_tag"`
	IsProduction bool          `yaml:"is_production"`
	LogLevel     zapcore.Level `yaml:"log_level"`
	LogFileName  string        `yaml:"log_file_name"`

	Server struct {
		Host            string `yaml:"host"`
		Port            string `yaml:"port"`
		CertsFile       string `yaml:"certs_file"`
		KeyFile         string `yaml:"key_file"`
		RequestTimeout  int    `yaml:"request_timeout"`
		ShutdownTimeout int    `yaml:"shutdown_timeout"`
	} `yaml:"server"`

	Redis struct {
		Host          string `yaml:"host"`
		Port          string `yaml:"port"`
		DialTimeout   int    `yaml:"dial_timeout"`
		ReadTimeout   int    `yaml:"read_timeout"`
		WriteTimeout  int    `yaml:"write_timeout"`
		PoolSize      int    `yaml:"pool_size"`
		PoolTimeout   int    `yaml:"pool_timeout"`
		Username      string `yaml:"username"`
		Password      string `yaml:"password"`
		DatabaseIndex int    `yaml:"db_index"`
	} `yaml:"redis"`
}

// LoadConfig provides an instance of config structure for the all application.
func LoadConfig(configFile string) (*Config, error) {
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

// InitConfig setup defaults values for non provided parameters
// and configures build tags values to be used if provided.
func InitConfig(config *Config, gitCommit, gitTag string) error {
	if len(gitCommit) != 0 {
		config.GitCommit = gitCommit
	}

	if len(gitTag) != 0 {
		config.GitTag = gitTag
	}

	if len(config.Server.Host) == 0 || len(config.Server.Port) == 0 {
		return errors.New("make sure to set valid server address and port in configuration file")
	}

	if len(config.Redis.Host) == 0 || len(config.Redis.Port) == 0 {
		return errors.New("make sure to set valid redis address and port in configuration file")
	}

	return nil
}
