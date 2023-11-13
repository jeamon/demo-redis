package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LoggerContextKey ContextKey = "request.logger"
)

// RSyncWrite is a rotable and concurent safe file-based logs writer.
// It is aimed to be used by Zap core routine. So it must implements
// the zap.WriteSyncer interface. The rotation happens based on the
// file size, once it reaches the max defined value.
type RSyncWrite struct {
	clock Clocker
	sync.Mutex
	file   *os.File
	folder string
	max    int
	size   int64
	isProd bool
}

func NewRSyncWriter(config *Config, clock Clocker) *RSyncWrite {
	return &RSyncWrite{
		clock:  clock,
		folder: config.LogFolder,
		max:    config.LogMaxSize,
		isProd: config.IsProduction,
	}
}

// Close closes the current log file.
func (rsw *RSyncWrite) Close() error {
	rsw.Lock()
	defer rsw.Unlock()
	if rsw.file == nil {
		return nil
	}
	return rsw.file.Close()
}

func (rsw *RSyncWrite) Sync() error {
	return rsw.file.Sync()
}

// Write implements the io.Writer interface with dynamic file rotation capability on max size.
func (rsw *RSyncWrite) Write(p []byte) (n int, err error) {
	rsw.Lock()
	defer rsw.Unlock()
	pLen := len(p)
	if pLen > rsw.max*1048576 {
		return 0, fmt.Errorf("logging: log size %d exceeds max file size %d", pLen, rsw.max)
	}
	if int64(pLen)+rsw.size > int64(rsw.max)*1048576 || rsw.file == nil {
		if rsw.file != nil {
			if err := rsw.file.Close(); err != nil {
				return 0, err
			}
		}

		path := CreateLogFilePath(rsw.folder, rsw.isProd, rsw.clock.Now())
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return 0, err
		}
		rsw.file = file
		rsw.size = 0
	}
	n, err = rsw.file.Write(p)
	rsw.size += int64(pLen)
	return n, err
}

// SyncWrite implements zap.SyncWriter. This is a small hack to avoid usual
// `Handle is invalid` error when calling Sync() on logger using os.stdout.
type SyncWrite struct {
	out *os.File
}

func (sw *SyncWrite) Sync() error {
	return nil
}

func (sw *SyncWrite) Write(p []byte) (n int, err error) {
	return sw.out.Write(p)
}

// SetupLogging is a helper function that initializes the logging module.
// In production all logs are saved to the defined file. In development
// the same logs are printed to standard output as well. It only adds
// stacktrace to fatal level logs. All logs come with commit & tag value.
// The custom clock provides timestamp in UTC for production environment
// and timestamp in Local timezone in development setup.
func SetupLogging(config *Config, w *RSyncWrite, clock TickerClocker) (*zap.Logger, func() error) {
	var logger *zap.Logger
	if config.IsProduction {
		zapConfig := zap.NewProductionEncoderConfig()
		zapConfig.TimeKey = "ts"
		zapConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.LevelKey = "lvl"
		zapConfig.NameKey = "name"
		zapConfig.MessageKey = "msg"
		zapConfig.CallerKey = "caller"
		zapConfig.StacktraceKey = "skt"
		fileEncoder := zapcore.NewJSONEncoder(zapConfig)
		zapCore := zapcore.NewTee(zapcore.NewCore(fileEncoder, w, config.LogLevel))
		logger = zap.New(zapCore, zap.AddCaller(), zap.AddStacktrace(zapcore.FatalLevel))
		logger = logger.WithOptions(zap.WithClock(clock))
	} else {
		zapConfig := zap.NewDevelopmentEncoderConfig()
		zapConfig.TimeKey = "ts"
		zapConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.LevelKey = "lvl"
		zapConfig.NameKey = "name"
		zapConfig.MessageKey = "msg"
		zapConfig.CallerKey = "caller"
		zapConfig.StacktraceKey = "skt"
		fileEncoder := zapcore.NewJSONEncoder(zapConfig)
		consoleEncoder := zapcore.NewConsoleEncoder(zapConfig)
		zapCore := zapcore.NewTee(
			zapcore.NewCore(fileEncoder, w, config.LogLevel),
			zapcore.NewCore(consoleEncoder, zapcore.Lock(&SyncWrite{os.Stdout}), config.LogLevel))
		logger = zap.New(zapCore, zap.AddCaller(), zap.AddStacktrace(zapcore.FatalLevel))
		logger = logger.WithOptions(zap.WithClock(clock))
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
	value := ctx.Value(LoggerContextKey)
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
