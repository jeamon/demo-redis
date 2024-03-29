package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/julienschmidt/httprouter"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AppProvider interface {
	Run() error
	Serve() func() error
	Stop(context.Context, context.Context) func() error
}

type App struct {
	logger         *zap.Logger
	config         *Config
	server         *http.Server
	redisClient    *redis.Client
	cleanups       []func() error
	queueConsumers []func(context.Context) error
}

// NewApp provides an instance of App.
func NewApp() (AppProvider, error) {
	var app *App
	config, err := LoadAndInitConfigs(GitCommit, GitTag, BuildTime)
	if err != nil {
		return nil, fmt.Errorf("failed to setup app configuration: %s", err)
	}

	// ensure the logs folder exists and Setup the logging module.
	err = os.MkdirAll(config.LogFolder, 0o700)
	if err != nil {
		return nil, fmt.Errorf("logging: failed to create folder: %v", err)
	}
	clock := NewClock(config.IsProduction)
	rswriter := NewRSyncWriter(config, clock)
	logger, logsFlusher := SetupLogging(config, rswriter, NewTickClock(clock))

	// Setup the connection to redis and boltDB servers.
	redisClient, err := NewRedisClient(config)
	if err != nil {
		return app, fmt.Errorf("failed to connect to redis server: %s", err)
	}

	boltDBClient, err := GetBoltDBClient(config)
	if err != nil {
		return app, fmt.Errorf("failed to connect to boltDB server: %s", err)
	}
	boltBookStorage := NewBoltBookStorage(logger, &config.BoltDB, boltDBClient)

	// Setup the repository and api services and routing.
	redisBookStorage := NewRedisBookStorage(logger, redisClient)
	redisQueue := NewRedisQueue(redisClient)
	boltDBConsumer := NewBoltDBConsumer(logger, redisQueue, boltBookStorage)

	bookService := NewBookService(logger, config, clock, redisBookStorage, boltBookStorage, redisQueue)
	stats := NewStatistics(config.GitTag, config.GitCommit, runtime.Version(), runtime.GOOS+"/"+runtime.GOARCH, IsAppRunningInDocker(), clock.Now())
	apiService := NewAPIHandler(logger, config, stats, clock, NewIDsHandler(), bookService)

	// Build the map of middlewares stacks.
	middlewaresPublic, middlewaresOps := apiService.MiddlewaresStacks()

	// Configure the endpoints with their handlers and middlewares.
	router := apiService.SetupRoutes(httprouter.New(),
		&MiddlewareMap{
			public: middlewaresPublic.Chain,
			ops:    middlewaresOps.Chain,
		},
	)

	// Build the api server definition.
	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port),
		Handler:        router,
		ReadTimeout:    config.Server.ReadTimeout,
		WriteTimeout:   config.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20,           // Max headers size : 1MB
		ConnContext:    SaveConnInContext, // add underlying connection into the request context
	}

	boltDBConsume := func(ctx context.Context) error {
		return boltDBConsumer.Consume(ctx, CreateQueue, UpdateQueue, DeleteQueue)
	}
	return &App{
		logger:      logger,
		config:      config,
		server:      srv,
		redisClient: redisClient,
		cleanups: []func() error{
			logsFlusher,
			rswriter.Close,
		},
		queueConsumers: []func(ctx context.Context) error{boltDBConsume},
	}, nil
}

// Run starts the api web server and a goroutine which is responsible to stop it.
func (app *App) Run() error {
	nCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(nCtx)

	g.Go(app.ConsumeQueues(gCtx, g))
	g.Go(app.Serve())
	g.Go(app.Stop(nCtx, gCtx))

	err := g.Wait()
	app.logger.Info("api server stopped",
		zap.String("app.host", app.config.Server.Host),
		zap.String("app.port", app.config.Server.Port),
		zap.Error(err),
	)
	errs := app.Clean()
	return errors.Join(err, errs)
}

// Clean calls all registered cleanups functions and returned aggregated errors.
func (app *App) Clean() error {
	var errs error
	for _, f := range app.cleanups {
		ferr := f()
		errs = errors.Join(errs, ferr)
	}
	return errs
}

// Serve starts the api web server. It returned error
// will be caught by the errorgroup.
func (app *App) Serve() func() error {
	return func() error {
		app.logger.Info("api server starting",
			zap.String("app.host", app.config.Server.Host),
			zap.String("app.port", app.config.Server.Port),
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

		sCtx, cancel := context.WithTimeout(context.Background(), app.config.Server.ShutdownTimeout)
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

		if err := app.redisClient.Close(); err != nil {
			app.logger.Info("error closing redis client", zap.Error(err))
		}
		return nil
	}
}

// ConsumeQueues runs all queue consumers into separate controlled goroutines.
func (app *App) ConsumeQueues(gCtx context.Context, g *errgroup.Group) func() error {
	return func() error {
		for _, consume := range app.queueConsumers {
			f := func() error {
				return consume(gCtx)
			}
			g.Go(f)
		}
		return nil
	}
}
