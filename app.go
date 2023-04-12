package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AppProvider interface {
	Run() error
	Serve() func() error
	Stop(context.Context, context.Context) func() error
}

type App struct {
	logger      *zap.Logger
	config      *Config
	server      *http.Server
	redisClient *redis.Client
	cleanups    []func()
}

// NewApp provides an instance
func NewApp() (AppProvider, error) {
	var err error
	var app *App

	// Setup the configuration module.
	config, err := LoadConfigFile("./config.yml")
	if err != nil {
		return app, fmt.Errorf("failed to load configurations from file: %s", err)
	}

	// Use environment variables with prefix `DRAP`.
	err = LoadConfigEnvs("DRAP", config)
	if err != nil {
		return app, fmt.Errorf("failed to load configurations from environment: %s", err)
	}

	err = InitConfig(config, GitCommit, GitTag, BuildTime)
	if err != nil {
		return app, fmt.Errorf("failed to initialize configurations: %s", err)
	}

	// Ensure the logs folder exists and Setup the logging module.
	err = os.MkdirAll(filepath.Dir(config.LogFile), 0o700)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging folder: %s", err)
	}
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging file: %s", err)
	}
	closer := func() {
		if cerr := logFile.Close(); cerr != nil {
			fmt.Println("error during closing of log file: ", cerr)
		}
	}
	logger, flusher := SetupLogging(config, logFile)

	// Setup the connection to redis server.
	redisClient, err := GetRedisClient(config)
	if err != nil {
		return app, fmt.Errorf("failed to connect to redis server: %s", err)
	}

	// Setup the repository and api services and routing..
	redisBookStorage := NewRedisBookStorage(logger, redisClient)
	bookService := NewBookService(logger, config, redisBookStorage)
	apiService := NewAPIHandler(
		logger,
		config,
		&Statistics{version: config.GitTag, started: time.Now()},
		bookService,
	)

	// Build the stack of middlewares.
	middlewares := Middlewares{
		apiService.PanicRecoveryMiddleware,
		apiService.RequestsCounterMiddleware,
		apiService.RequestIDMiddleware,
		CORSMiddleware,
		apiService.CoreMiddleware,
	}

	// Configure the endpoints with their handlers and middlewares.
	router := apiService.SetupRoutes(httprouter.New(), &middlewares)

	// Start the api server.
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port),
		Handler: router,
	}

	return &App{
		logger:      logger,
		config:      config,
		server:      srv,
		redisClient: redisClient,
		cleanups: []func(){
			flusher,
			closer,
		},
	}, nil
}

// Run starts the api web server and a goroutine which is responsible to stop it.
func (app *App) Run() error {
	defer app.Clean()
	nCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(nCtx)

	g.Go(app.Serve())
	g.Go(app.Stop(nCtx, gCtx))

	err := g.Wait()
	app.logger.Info("api server stopped",
		zap.String("host", app.config.Server.Host),
		zap.String("port", app.config.Server.Port),
		zap.Error(err),
	)
	return err
}

// Clean calls all registered cleanups functions.
func (app *App) Clean() {
	for _, f := range app.cleanups {
		f()
	}
}

// Serve starts the api web server. It returned error
// will be caught by the errorgroup.
func (app *App) Serve() func() error {
	return func() error {
		app.logger.Info("api server starting",
			zap.String("host", app.config.Server.Host),
			zap.String("port", app.config.Server.Port),
		)
		err := app.server.ListenAndServe()
		if err == http.ErrServerClosed {
			err = nil
		}
		return err
	}
}

// Stop listens for the group context and triggers the server graceful shutdown.
// It states the reason of its call. We proceed with a brutal shutdown if the
// the graceful did not complete successfully. We explicitly return `nil` to
// allow the errorgroup catches only the `Serve` method result.
func (app *App) Stop(nCtx, gCtx context.Context) func() error {
	return func() error {
		<-gCtx.Done()

		if nCtx.Err() != nil {
			app.logger.Info("api server stopping. reason: requested to stop")
		} else {
			app.logger.Info("api server stopping. reason: errored at running")
		}

		sCtx, cancel := context.WithTimeout(context.Background(), time.Duration(app.config.Server.ShutdownTimeout)*time.Second)
		defer cancel()
		err := app.server.Shutdown(sCtx)
		switch err {
		case nil, http.ErrServerClosed:
			app.logger.Info("api server graceful shutdown succeeded")
		case context.DeadlineExceeded:
			app.logger.Info("api server graceful shutdown timed out")
		default:
			app.logger.Info("api server graceful shutdown failed", zap.Error(err))
		}

		if err != nil && err != http.ErrServerClosed {
			app.logger.Info("api server going to force shutdown", zap.Error(app.server.Close()))
		}
		return nil
	}
}
