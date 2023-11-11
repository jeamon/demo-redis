package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ContextRequestLogger ContextKey = "request.logger"
)

// SetupLogging is a helper function that initialize the logging module.
// In production all logs are saved to the defined file. In development
// the same logs are printed to standard output as well. It only adds
// stacktrace to debug level logs. All logs come with commit & tag value.
func SetupLogging(config *Config, logFile *os.File) (*zap.Logger, func() error) {
	var logger *zap.Logger
	if config.IsProduction {
		zapConfig := zap.NewProductionEncoderConfig()
		zapConfig.TimeKey = "timestamp"
		zapConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.LevelKey = "level"
		zapConfig.NameKey = "name"
		zapConfig.MessageKey = "msg"
		zapConfig.CallerKey = "caller"
		zapConfig.StacktraceKey = "stacktrace"
		fileEncoder := zapcore.NewJSONEncoder(zapConfig)
		zapCore := zapcore.NewTee(zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), config.LogLevel))
		logger = zap.New(zapCore, zap.AddCaller(), zap.AddStacktrace(zapcore.FatalLevel))
	} else {
		zapConfig := zap.NewDevelopmentEncoderConfig()
		zapConfig.TimeKey = "timestamp"
		zapConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.LevelKey = "level"
		zapConfig.NameKey = "name"
		zapConfig.MessageKey = "msg"
		zapConfig.CallerKey = "caller"
		zapConfig.StacktraceKey = "stacktrace"
		fileEncoder := zapcore.NewJSONEncoder(zapConfig)
		consoleEncoder := zapcore.NewConsoleEncoder(zapConfig)
		zapCore := zapcore.NewTee(
			zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), config.LogLevel),
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), config.LogLevel))
		logger = zap.New(zapCore, zap.AddCaller(), zap.AddStacktrace(zapcore.FatalLevel))
	}
	logger = logger.With(zap.String("app.commit", config.GitCommit), zap.String("app.tag", config.GitTag), zap.String("app.built", config.BuildTime))

	flusher := func() error {
		if err := logger.Sync(); err != nil {
			return fmt.Errorf("[flush logs]: %w", err)
		}
		return nil
	}

	return logger, flusher
}

// GetLoggerFromCtx retrieves previously set logger from the context and returns it.
// If the logger can't be retrieved it will return the initial logger of the App.
func (api *APIHandler) GetLoggerFromContext(ctx context.Context) *zap.Logger {
	value := ctx.Value(ContextRequestLogger)
	if value != nil {
		return value.(*zap.Logger)
	}
	return api.logger
}

// CreateLogFilePath returns the absolute path of the initial log file.
func CreateLogFilePath(folder string, isProd bool, t time.Time) string {
	var envKey string
	if isProd {
		envKey = "prod"
	} else {
		envKey = "dev"
	}
	suffix := fmt.Sprintf("%02d%02d%02d.%02d%02d%02d.%s.log", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), envKey)
	return filepath.Join(folder, suffix)
}
