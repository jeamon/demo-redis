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
	logger   *zap.Logger
	config   *Config
	clock    Clocker
	pstorage BookStorage // primary storage
	bstorage BookStorage // backup storage
	queue    Queuer
}

func NewBookService(logger *zap.Logger, config *Config, clock Clocker, pstorage BookStorage, bstorage BookStorage, queue Queuer) BookServiceProvider {
	return &BookService{
		logger:   logger,
		config:   config,
		clock:    clock,
		pstorage: pstorage,
		bstorage: bstorage,
		queue:    queue,
	}
}

func (bs *BookService) Add(ctx context.Context, id string, book Book) error {
	err := bs.pstorage.Add(ctx, id, book)
	if err != nil {
		return err
	}
	if perr := bs.queue.Push(ctx, CreateQueue, book); perr != nil {
		bs.logger.Error("service: failed to push book to queue", zap.String("qid", CreateQueue), zap.Error(perr))
	}
	return err
}

func (bs *BookService) GetOne(ctx context.Context, id string) (Book, error) {
	book, err := bs.pstorage.GetOne(ctx, id)
	if err == nil {
		return book, err
	}

	book, err = bs.bstorage.GetOne(ctx, id)
	if err != nil {
		return book, err
	}

	if perr := bs.pstorage.Add(ctx, id, book); perr != nil {
		bs.logger.Error("service: failed to cache book into pstorage", zap.String("id", id), zap.Error(perr))
	}
	return book, err
}

func (bs *BookService) Delete(ctx context.Context, id string) error {
	err := bs.pstorage.Delete(ctx, id)
	if err != nil {
		return err
	}
	if perr := bs.queue.Push(ctx, DeleteQueue, Book{ID: id}); perr != nil {
		bs.logger.Error("service: failed to push to queue", zap.String("qid", DeleteQueue), zap.Error(perr))
	}
	return err
}

func (bs *BookService) Update(ctx context.Context, id string, book Book) (Book, error) {
	book.UpdatedAt = bs.clock.Now().UTC().String()
	b, err := bs.pstorage.Update(ctx, id, book)
	if err != nil {
		return b, err
	}
	if perr := bs.queue.Push(ctx, UpdateQueue, book); perr != nil {
		bs.logger.Error("service: failed to push to queue", zap.String("qid", UpdateQueue), zap.Error(perr))
	}
	return b, err
}

// GetAll fetches all books from backup storage. In case there is nothing
// or an error occurred, it fallback to primary storage results.
func (bs *BookService) GetAll(ctx context.Context) ([]Book, error) {
	bbooks, berr := bs.bstorage.GetAll(ctx)
	if berr != nil || len(bbooks) == 0 {
		return bs.pstorage.GetAll(ctx)
	}
	return bbooks, berr
}
