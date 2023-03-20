package main

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetupLogging is a helper function that initialize the logging module.
// In production all logs are saved to the defined file. In development
// the same logs are printed to standard output as well. It only adds
// stacktrace to debug level logs. All logs come with commit & tag value.
func SetupLogging(config *Config, logFile *os.File) (*zap.Logger, func()) {
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
		logger = zap.New(zapCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
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
		logger = zap.New(zapCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	}
	logger = logger.With(zap.String("commit", config.GitCommit), zap.String("tag", config.GitTag))

	flusher := func() {
		if err := logger.Sync(); err != nil {
			log.Println("error during flushing any buffered log entries:", err)
		}
	}

	return logger, flusher
}
