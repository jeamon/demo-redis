package main

import (
	"sync"

	"go.uber.org/zap"
)

// APIHandler defines the API handler.
type APIHandler struct {
	logger      *zap.Logger
	config      *Config
	stats       *Statistics
	mode        *Maintenance
	bookService BookServiceProvider
}

// NewAPIHandler provides a new instance of APIHandler.
func NewAPIHandler(logger *zap.Logger, config *Config, stats *Statistics, bs BookServiceProvider) *APIHandler {
	m := &Maintenance{}
	m.enabled.Store(false)
	stats.status = make(map[int]uint64)
	stats.mu = &sync.RWMutex{}
	return &APIHandler{logger: logger, config: config, stats: stats, mode: m, bookService: bs}
}
