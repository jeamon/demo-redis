package main

import (
	"context"

	"go.uber.org/zap"
)

type BookServiceProvider interface {
	Add(ctx context.Context, id string, book Book) error
	GetOne(ctx context.Context, id string) (Book, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, id string, book Book) (Book, error)
	GetAll(ctx context.Context) ([]Book, error)
}

type BookService struct {
	logger  *zap.Logger
	config  *Config
	clock   Clocker
	storage BookStorage
	queue   Queuer
}

func NewBookService(logger *zap.Logger, config *Config, clock Clocker, storage BookStorage, queue Queuer) BookServiceProvider {
	return &BookService{
		logger:  logger,
		config:  config,
		clock:   clock,
		storage: storage,
		queue:   queue,
	}
}

func (bs *BookService) Add(ctx context.Context, id string, book Book) error {
	if err := bs.queue.Push(ctx, CreateQueue, book); err != nil {
		bs.logger.Error("service: failed to push book to queue", zap.String("qid", CreateQueue), zap.Error(err))
	}
	return bs.storage.Add(ctx, id, book)
}

func (bs *BookService) GetOne(ctx context.Context, id string) (Book, error) {
	book, err := bs.storage.GetOne(ctx, id)
	return book, err
}

func (bs *BookService) Delete(ctx context.Context, id string) error {
	if err := bs.queue.Push(ctx, DeleteQueue, Book{ID: id}); err != nil {
		bs.logger.Error("service: failed to push to queue", zap.String("qid", DeleteQueue), zap.Error(err))
	}
	return bs.storage.Delete(ctx, id)
}

func (bs *BookService) Update(ctx context.Context, id string, book Book) (Book, error) {
	if err := bs.queue.Push(ctx, UpdateQueue, book); err != nil {
		bs.logger.Error("service: failed to push to queue", zap.String("qid", UpdateQueue), zap.Error(err))
	}

	book.UpdatedAt = bs.clock.Now().UTC().String()
	return bs.storage.Update(ctx, id, book)
}

func (bs *BookService) GetAll(ctx context.Context) ([]Book, error) {
	books, err := bs.storage.GetAll(ctx)
	return books, err
}
