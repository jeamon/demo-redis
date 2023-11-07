package main

import (
	"context"

	"go.uber.org/zap"
)

type Consumer interface {
	Consume(ctx context.Context, qids ...string) error
}

type boltDBConsumer struct {
	logger *zap.Logger
	queue  Queuer
	repo   BookStorage
}

func NewBoltDBConsumer(logger *zap.Logger, q Queuer, repo BookStorage) Consumer {
	return &boltDBConsumer{logger, q, repo}
}

func (bc *boltDBConsumer) Consume(ctx context.Context, qids ...string) error {
	var book Book
	var err error
	var qid string
	for {
		qid, book, err = bc.queue.Pop(ctx, qids...)
		if err != nil && ctx.Err() != nil {
			bc.logger.Info("consumer: queue pop call: context is done: exit", zap.String("reason", ctx.Err().Error()))
			return nil
		}

		if err != nil {
			bc.logger.Error("consumer: error on queue pop call", zap.Error(err))
			continue
		}

		switch qid {
		case CreateQueue:
			if err = bc.repo.Add(ctx, book.ID, book); err != nil {
				bc.logger.Error("consumer: failed to create", zap.Any("book", book), zap.Error(err))
			}
		case UpdateQueue:
			if _, err = bc.repo.Update(ctx, book.ID, book); err != nil {
				bc.logger.Error("consumer: failed to update", zap.Any("book", book), zap.Error(err))
			}
		case DeleteQueue:
			if err = bc.repo.Delete(ctx, book.ID); err != nil {
				bc.logger.Error("consumer: failed to delete", zap.String("id", book.ID), zap.Error(err))
			}
		default:
			bc.logger.Warn("consumer: received book on unknow queue id", zap.String("qid", qid), zap.Any("book", book))
		}
	}
}
